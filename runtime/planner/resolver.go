package planner

import (
	"context"
	"fmt"
	"strings"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/vault"
)

// Resolver processes an ExecutionGraph, resolving all expressions and determining
// which branches are taken. Uses a wave-based resolution model.
//
// # Blockers
//
// A "blocker" is a control flow construct (if/when/for) that cannot be evaluated
// until its condition or collection expression is resolved. We can't know which
// branch to take (if/when) or how many iterations to unroll (for) until the
// controlling expression has a concrete value.
//
// Example blockers:
//   - `if @var.ENV == "prod"` - blocked until ENV is resolved
//   - `for item in @var.ITEMS` - blocked until ITEMS collection is resolved
//   - `when @env.REGION` - blocked until REGION is resolved
//
// # Wave-Based Resolution
//
// Resolution proceeds in waves. Each wave:
//  1. Traverse the IR graph, collecting expressions that need resolution
//  2. Identify blockers (control flow with unresolved conditions)
//  3. Mark touched all expressions up to the blockers
//  4. Batch resolve all touched expressions (grouped by decorator type for efficiency)
//  5. Evaluate blockers with resolved values, set Taken flags
//  6. Queue taken branches for next wave (branch pruning: untaken branches ignored)
//  7. Repeat until no more blockers
//
// # For-Loop Unrolling
//
// For-loops are blockers because we can't unroll until the collection is resolved.
// Once resolved, unrolling produces flat statements with VarDecl injections:
//
//	for item in ["a", "b"] { echo @var.item }
//
// Unrolls to (in nextWaveStmts):
//
//	VarDecl: item = "a"
//	echo @var.item
//	VarDecl: item = "b"
//	echo @var.item
//
// Each VarDecl rebinds the loop variable before its iteration's body statements.
// This ensures nested blockers (like `if @var.item == "b"`) see the correct value.
//
// # Key Principles
//
//   - Batch-first: Collect ALL expressions up to blockers, THEN batch resolve
//   - Wave-based: Multiple passes through the graph as branches open
//   - Branch pruning: Untaken branches are never traversed or resolved (security + efficiency)
type Resolver struct {
	graph   *ExecutionGraph
	vault   *vault.Vault
	session decorator.Session
	config  ResolveConfig

	// Resolution state
	values        map[string]any  // exprID → resolved value (for condition evaluation)
	pendingCalls  []decoratorCall // Decorator calls to batch resolve
	nextWaveStmts []*StatementIR  // Statements for next wave (taken branches)
	errors        []error         // Collected errors
}

// ResolveConfig configures the resolution process.
type ResolveConfig struct {
	TargetFunction string          // Empty = script mode, non-empty = command mode
	Context        context.Context // Execution context
}

// decoratorCall represents a decorator call to be batch resolved.
type decoratorCall struct {
	expr      *ExprIR       // The expression being resolved
	decorator *DecoratorRef // Structured decorator reference
	raw       string        // Raw decorator string (for Vault tracking)
	exprID    string        // Expression ID in Vault
	varName   string        // Variable name if this is a var decl (for storing by name)
}

// Resolve processes the execution graph and resolves all expressions.
// Returns error if resolution fails (undefined variables, decorator failures, etc.).
func Resolve(graph *ExecutionGraph, v *vault.Vault, session decorator.Session, config ResolveConfig) error {
	r := &Resolver{
		graph:   graph,
		vault:   v,
		session: session,
		config:  config,
		values:  make(map[string]any),
	}

	return r.resolve()
}

// resolve is the main wave loop.
func (r *Resolver) resolve() error {
	// Select statements based on mode (script vs command)
	stmts := r.selectStatements()
	if stmts == nil {
		// Error already recorded
		return r.buildError()
	}

	// Wave loop
	for {
		// Phase 1: Traverse and collect
		blockers := r.traverseAndCollect(stmts)

		// Phase 2: Batch resolve all touched expressions
		if err := r.batchResolve(); err != nil {
			return err
		}

		// Phase 3: Evaluate blockers
		for _, blocker := range blockers {
			if err := r.evaluateBlocker(blocker); err != nil {
				return err
			}
		}

		// Phase 4: Check for more blockers
		if len(blockers) == 0 {
			break // No more blockers, done
		}

		// Continue with next wave (statements from taken branches)
		stmts = r.nextWaveStmts
		r.nextWaveStmts = nil
	}

	return nil
}

// selectStatements chooses which statements to process based on mode.
func (r *Resolver) selectStatements() []*StatementIR {
	if r.config.TargetFunction != "" {
		// Command mode: only the target function
		fn, ok := r.graph.Functions[r.config.TargetFunction]
		if !ok {
			r.errors = append(r.errors, fmt.Errorf("function %q not found", r.config.TargetFunction))
			return nil
		}
		return fn.Body
	}

	// Script mode: all top-level statements
	return r.graph.Statements
}

// traverseAndCollect walks the IR graph, collecting expressions and identifying blockers.
func (r *Resolver) traverseAndCollect(stmts []*StatementIR) []*BlockerIR {
	var blockers []*BlockerIR

	for _, stmt := range stmts {
		switch stmt.Kind {
		case StmtVarDecl:
			// Handle variable declaration specially
			r.collectVarDecl(stmt.VarDecl)

		case StmtCommand:
			// Collect all expressions in command parts
			for _, part := range stmt.Command.Command.Parts {
				r.collectExpr(part, "")
			}

		case StmtBlocker:
			// Collect condition expression (for if/when)
			r.collectExpr(stmt.Blocker.Condition, "")
			// Collect collection expression (for for-loops)
			if stmt.Blocker.Kind == BlockerFor && stmt.Blocker.Collection != nil {
				r.collectExpr(stmt.Blocker.Collection, "")
			}
			// Add to blockers list (will evaluate after resolution)
			blockers = append(blockers, stmt.Blocker)
			// DON'T traverse into branches yet - wait for condition resolution

		case StmtTry:
			// Try/catch is special - both branches are in the plan
			// (exception is runtime, not plan-time)
			blockers = append(blockers, r.traverseAndCollect(stmt.Try.TryBlock)...)
			blockers = append(blockers, r.traverseAndCollect(stmt.Try.CatchBlock)...)
			blockers = append(blockers, r.traverseAndCollect(stmt.Try.FinallyBlock)...)
		}
	}

	return blockers
}

// collectVarDecl handles variable declaration - stores value by variable name for condition evaluation.
func (r *Resolver) collectVarDecl(decl *VarDeclIR) {
	exprID := decl.ExprID

	// Collect the value expression (may add pending decorator calls)
	r.collectExprForVar(decl.Value, exprID, decl.Name)

	// For literals, we can store the value immediately for condition evaluation
	if decl.Value.Kind == ExprLiteral {
		r.vault.StoreUnresolvedValue(exprID, decl.Value.Value)
		r.vault.MarkTouched(exprID)
		// Store by variable name for EvaluateExpr
		r.values[decl.Name] = decl.Value.Value
	}
	// For decorator refs, the value will be stored after batch resolution
}

// collectExprForVar collects an expression that's part of a variable declaration.
// varName is used to store the resolved value by variable name for condition evaluation.
func (r *Resolver) collectExprForVar(expr *ExprIR, exprID, varName string) {
	if expr == nil {
		return
	}

	if expr.Kind == ExprDecoratorRef {
		// Track decorator call for batch resolution, with variable name
		raw := buildDecoratorRaw(expr.Decorator)

		if exprID == "" {
			exprID = r.vault.TrackExpression(raw)
		}

		r.pendingCalls = append(r.pendingCalls, decoratorCall{
			expr:      expr,
			decorator: expr.Decorator,
			raw:       raw,
			exprID:    exprID,
			varName:   varName, // Track variable name for storing by name later
		})
	} else {
		// For non-decorator expressions, use regular collectExpr
		r.collectExpr(expr, exprID)
	}
}

// collectExpr collects an expression for resolution.
// exprID is the pre-assigned expression ID (for var decls), or empty to generate one.
func (r *Resolver) collectExpr(expr *ExprIR, exprID string) {
	if expr == nil {
		return
	}

	switch expr.Kind {
	case ExprLiteral:
		// Literals don't need resolution - store directly if we have an exprID
		if exprID != "" {
			r.vault.StoreUnresolvedValue(exprID, expr.Value)
			r.vault.MarkTouched(exprID)
		}
		// Note: For literals in var decls, the value is stored by collectVarDecl

	case ExprVarRef:
		// Look up exprID from scope
		varExprID, ok := r.graph.Scopes.Lookup(expr.VarName)
		if !ok {
			r.errors = append(r.errors, &EvalError{
				Message: "undefined variable (no hoisting allowed)",
				VarName: expr.VarName,
				Span:    expr.Span,
			})
			return
		}
		// Mark as touched (in execution path)
		r.vault.MarkTouched(varExprID)

		// Note: The value should already be in r.values[expr.VarName] from
		// when the variable was declared. If not, it will fail during evaluation.

	case ExprDecoratorRef:
		// Track decorator call for batch resolution
		raw := buildDecoratorRaw(expr.Decorator)

		// Generate or use provided exprID
		if exprID == "" {
			exprID = r.vault.TrackExpression(raw)
		}

		r.pendingCalls = append(r.pendingCalls, decoratorCall{
			expr:      expr,
			decorator: expr.Decorator,
			raw:       raw,
			exprID:    exprID,
		})

	case ExprBinaryOp:
		// Recursively collect operands
		r.collectExpr(expr.Left, "")
		r.collectExpr(expr.Right, "")
	}
}

// batchResolve resolves all pending decorator calls in batches (grouped by decorator type).
func (r *Resolver) batchResolve() error {
	if len(r.pendingCalls) == 0 {
		return nil // No decorators to resolve
	}

	// Group pending calls by decorator name
	groups := make(map[string][]decoratorCall)
	for _, call := range r.pendingCalls {
		groups[call.decorator.Name] = append(groups[call.decorator.Name], call)
	}

	// Resolve each group in batch
	for decoratorName, calls := range groups {
		if err := r.resolveBatch(decoratorName, calls); err != nil {
			return err
		}
	}

	// Clear pending calls for next wave
	r.pendingCalls = nil

	// Generate DisplayIDs for all touched expressions
	r.vault.ResolveAllTouched()

	return nil
}

// resolveBatch resolves a batch of calls for a single decorator type.
func (r *Resolver) resolveBatch(decoratorName string, calls []decoratorCall) error {
	// Build ValueCall slice for batch resolution
	valueCalls := make([]decorator.ValueCall, len(calls))
	for i, call := range calls {
		valueCalls[i] = buildValueCall(call.decorator)
	}

	// Build evaluation context
	ctx := decorator.ValueEvalContext{
		Session:  r.session,
		Vault:    r.vault,
		PlanHash: nil, // TODO: Get from config
		StepPath: "",  // TODO: Track current step path
	}

	// Get current transport scope
	currentScope := transportStringToScope(r.vault.CurrentTransport())

	// Call decorator's batch Resolve
	// This is where the magic happens - multiple @aws.secret calls → one API request
	results, err := decorator.Global().ResolveValues(ctx, currentScope, valueCalls...)
	if err != nil {
		return fmt.Errorf("failed to resolve @%s: %w (cannot plan if cannot resolve)", decoratorName, err)
	}

	// Store results in Vault
	for i, result := range results {
		call := calls[i]
		exprID := call.exprID

		// Store value in Vault
		r.vault.StoreUnresolvedValue(exprID, result.Value)
		r.vault.MarkTouched(exprID)

		// Store in values map for condition evaluation
		// Store by decorator key (e.g., "env.HOME") for direct decorator refs in conditions
		decKey := decoratorKey(call.decorator)
		r.values[decKey] = result.Value

		// If this decorator is part of a var decl, also store by variable name
		if call.varName != "" {
			r.values[call.varName] = result.Value
		}
	}

	return nil
}

// evaluateBlocker evaluates a blocker and sets its Taken flag.
func (r *Resolver) evaluateBlocker(blocker *BlockerIR) error {
	switch blocker.Kind {
	case BlockerIf:
		return r.evaluateIfBlocker(blocker)
	case BlockerFor:
		return r.evaluateForBlocker(blocker)
	case BlockerWhen:
		return r.evaluateWhenBlocker(blocker)
	default:
		return fmt.Errorf("unknown blocker kind: %d", blocker.Kind)
	}
}

// evaluateIfBlocker evaluates an if statement and queues the taken branch.
func (r *Resolver) evaluateIfBlocker(blocker *BlockerIR) error {
	// Evaluate condition using resolved values
	result, err := EvaluateExpr(blocker.Condition, r.values)
	if err != nil {
		return fmt.Errorf("failed to evaluate if condition: %w", err)
	}

	taken := IsTruthy(result)
	blocker.Taken = &taken

	// Queue taken branch for next wave
	if taken {
		r.nextWaveStmts = append(r.nextWaveStmts, blocker.ThenBranch...)
	} else if blocker.ElseBranch != nil {
		r.nextWaveStmts = append(r.nextWaveStmts, blocker.ElseBranch...)
	}
	// Untaken branch is NEVER added to nextWaveStmts → branch pruning

	return nil
}

// evaluateForBlocker evaluates a for-loop and unrolls it.
// Unrolling injects VarDecl statements before each iteration's body so the
// loop variable is properly bound when those statements are processed in wave 2.
func (r *Resolver) evaluateForBlocker(blocker *BlockerIR) error {
	// Evaluate collection
	collection, err := r.evaluateCollection(blocker.Collection)
	if err != nil {
		return fmt.Errorf("failed to evaluate for collection: %w", err)
	}

	// Unroll loop - inject VarDecl + body statements for each iteration
	for _, item := range collection {
		// Create exprID for this iteration's loop variable
		loopVarRaw := fmt.Sprintf("literal:%v", item)
		loopVarExprID := r.vault.DeclareVariable(blocker.LoopVar, loopVarRaw)

		// Inject a VarDecl statement that binds the loop variable for this iteration
		// This ensures the variable is properly bound when wave 2 processes these statements
		iterVarDecl := &StatementIR{
			Kind: StmtVarDecl,
			VarDecl: &VarDeclIR{
				Name:   blocker.LoopVar,
				ExprID: loopVarExprID,
				Value:  &ExprIR{Kind: ExprLiteral, Value: item},
			},
		}

		// Add: VarDecl (rebinds loop var) + body statements
		r.nextWaveStmts = append(r.nextWaveStmts, iterVarDecl)
		r.nextWaveStmts = append(r.nextWaveStmts, blocker.ThenBranch...)
	}

	return nil
}

// evaluateWhenBlocker evaluates a when statement and queues the matching arm.
func (r *Resolver) evaluateWhenBlocker(blocker *BlockerIR) error {
	// Evaluate condition
	value, err := EvaluateExpr(blocker.Condition, r.values)
	if err != nil {
		return fmt.Errorf("failed to evaluate when condition: %w", err)
	}

	// Find first matching arm
	for _, arm := range blocker.Arms {
		if matchPattern(arm.Pattern, value, r.values) {
			r.nextWaveStmts = append(r.nextWaveStmts, arm.Body...)
			return nil
		}
	}

	// No matching arm (when statements don't require exhaustive patterns)
	return nil
}

// evaluateCollection evaluates a collection expression for a for-loop.
func (r *Resolver) evaluateCollection(expr *ExprIR) ([]any, error) {
	value, err := EvaluateExpr(expr, r.values)
	if err != nil {
		return nil, err
	}

	// Convert to slice
	switch v := value.(type) {
	case []any:
		return v, nil
	case []string:
		result := make([]any, len(v))
		for i, s := range v {
			result[i] = s
		}
		return result, nil
	case []int:
		result := make([]any, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	default:
		return nil, fmt.Errorf("for-loop collection must be a list, got %T", value)
	}
}

// matchPattern checks if a value matches a pattern.
func matchPattern(pattern *ExprIR, value any, values map[string]any) bool {
	// Evaluate pattern expression
	patternValue, err := EvaluateExpr(pattern, values)
	if err != nil {
		return false
	}

	// Simple equality check for now
	// TODO: Support regex, ranges, etc.
	return compareEqual(patternValue, value)
}

// buildValueCall converts a DecoratorRef to a decorator.ValueCall.
func buildValueCall(d *DecoratorRef) decorator.ValueCall {
	call := decorator.ValueCall{
		Path:   d.Name,
		Params: make(map[string]any),
	}

	// If there's a selector, use the first element as Primary
	if len(d.Selector) > 0 {
		primary := d.Selector[0]
		call.Primary = &primary
	}

	// TODO: Handle Args (parameterized decorators)

	return call
}

// buildDecoratorRaw builds a raw decorator string from a DecoratorRef.
// e.g., DecoratorRef{Name: "env", Selector: ["HOME"]} → "@env.HOME"
func buildDecoratorRaw(d *DecoratorRef) string {
	if d == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("@")
	sb.WriteString(d.Name)
	for _, s := range d.Selector {
		sb.WriteString(".")
		sb.WriteString(s)
	}

	return sb.String()
}

// buildError builds an error from collected errors.
func (r *Resolver) buildError() error {
	if len(r.errors) == 0 {
		return nil
	}
	if len(r.errors) == 1 {
		return r.errors[0]
	}

	// Multiple errors - combine them
	var sb strings.Builder
	sb.WriteString("multiple resolution errors:\n")
	for i, err := range r.errors {
		sb.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
	}
	return fmt.Errorf("%s", sb.String())
}

// transportStringToScope converts a transport string to TransportScope.
func transportStringToScope(transport string) decorator.TransportScope {
	switch transport {
	case "":
		return decorator.TransportScopeLocal
	case "local":
		return decorator.TransportScopeLocal
	case "ssh":
		return decorator.TransportScopeSSH
	default:
		// Unknown transport - treat as remote
		return decorator.TransportScopeRemote
	}
}
