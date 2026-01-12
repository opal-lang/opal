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
// # Scope Semantics: Metaprogramming vs Execution Blocks
//
// Opal has two kinds of blocks with different scoping rules:
//
// **Language control blocks (if, for, when, fun)** - These are METAPROGRAMMING
// constructs that are evaluated at plan-time and "disappear" in the final plan.
// Variables declared inside these blocks LEAK to the outer scope because the
// block structure is flattened away.
//
//	var OUTER = "before"
//	if @var.COND {
//	    var INNER = "inside"   // Leaks to outer scope!
//	}
//	echo @var.INNER            // Works if COND was true
//
// After planning (if COND is true), this becomes:
//
//	var OUTER = "before"
//	var INNER = "inside"       // Flattened - no if block anymore
//	echo @var.INNER
//
// **Execution blocks (try/catch, @retry, @timeout, etc.)** - These are RUNTIME
// constructs that remain in the plan. Variables declared inside these blocks
// are ISOLATED and do NOT leak to the outer scope.
//
//	var counter = 0
//	try {
//	    var counter = 5        // Local copy, doesn't affect outer
//	}
//	echo @var.counter          // Still 0
//
// This distinction exists because:
//   - Metaprogramming blocks resolve deterministically at plan-time
//   - Execution blocks have runtime behavior (retries, exceptions) that's non-deterministic
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
// # Sequential Processing with Flattening
//
// Because metaprogramming blocks leak variables, statements AFTER a blocker must
// wait until the blocker is evaluated and its taken branch is processed. This
// ensures variables declared in taken branches are visible to subsequent statements.
//
// Processing order for: `var A; if COND { var B }; echo @var.B`
//  1. Process `var A` - add to scope
//  2. Hit blocker `if COND` - STOP, resolve COND
//  3. Evaluate blocker - condition is true
//  4. Queue: taken branch statements + remaining statements after blocker
//  5. Process `var B` from taken branch - add to scope
//  6. Process `echo @var.B` - B is now visible!
//
// # Wave-Based Resolution
//
// Resolution proceeds in waves, processing statements sequentially:
//  1. Collect expressions until we hit a blocker
//  2. Batch resolve all collected expressions (grouped by decorator type)
//  3. Evaluate the blocker, determine taken branch
//  4. Queue: taken branch + remaining statements after blocker
//  5. Repeat until no more statements
//
// # For-Loop Unrolling
//
// For-loops are blockers because we can't unroll until the collection is resolved.
// Once resolved, unrolling produces flat statements with VarDecl injections:
//
//	for item in ["a", "b"] { echo @var.item }
//
// Unrolls to:
//
//	VarDecl: item = "a"
//	echo @var.item
//	VarDecl: item = "b"
//	echo @var.item
//
// Each VarDecl rebinds the loop variable before its iteration's body statements.
// This ensures nested blockers (like `if @var.item == "b"`) see the correct value.
//
// # Branch Pruning
//
// Untaken branches are never traversed or resolved. This provides:
//   - Security: Secrets in untaken branches are never resolved or exposed
//   - Efficiency: No wasted API calls for unused decorator values
//
// # Key Principles
//
//   - Sequential processing: Statements after blockers wait for taken branch
//   - Flattening semantics: Metaprogramming blocks disappear, variables leak
//   - Scope isolation: Execution blocks (try/catch, decorators) don't leak
//   - Batch-first: Collect expressions up to blocker, THEN batch resolve
//   - Branch pruning: Untaken branches are never resolved
type Resolver struct {
	graph          *ExecutionGraph
	vault          *vault.Vault
	session        decorator.Session
	config         ResolveConfig
	scopes         *ScopeStack
	activeFunction *FunctionIR

	// Resolution state
	decoratorExprIDs map[string]string // decorator key (e.g., "env.HOME") → exprID
	pendingCalls     []decoratorCall   // Decorator calls to batch resolve
	errors           []error           // Collected errors
}

// ResolveConfig configures the resolution process.
type ResolveConfig struct {
	TargetFunction string          // Empty = script mode, non-empty = command mode
	Context        context.Context // Execution context
}

// ResolveResult contains the resolved execution tree.
// The tree preserves structure for rich dry-run output:
//   - Blockers remain as nodes with Taken set and untaken branches pruned
//   - For-loops have Iterations populated with resolved values and deep-copied bodies
//   - Try/catch blocks are preserved as-is (runtime constructs)
type ResolveResult struct {
	Statements       []*StatementIR    // Pruned tree (only taken branches, nested blockers resolved)
	DecoratorExprIDs map[string]string // Decorator key → exprID for display ID lookup
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
// Returns a ResolveResult containing the pruned execution tree, or an error
// if resolution fails (undefined variables, decorator failures, etc.).
//
// The returned tree preserves structure for rich dry-run output:
//   - Blockers remain as nodes with Taken set and untaken branches pruned (set to nil)
//   - For-loops have Iterations populated with resolved values and deep-copied bodies
//   - Try/catch blocks are preserved as-is (runtime constructs)
func Resolve(graph *ExecutionGraph, v *vault.Vault, session decorator.Session, config ResolveConfig) (*ResolveResult, error) {
	r := &Resolver{
		graph:            graph,
		vault:            v,
		session:          session,
		config:           config,
		decoratorExprIDs: make(map[string]string),
	}

	return r.resolve()
}

// getValue looks up a value by name from Vault.
// This is the single source of truth for values - no duplicate storage.
//
// Lookup order:
//  1. Variable name → exprID via ScopeStack → value via Vault
//  2. Decorator key → exprID via decoratorExprIDs → value via Vault
func (r *Resolver) getValue(name string) (any, bool) {
	// Try variable lookup first (via scope)
	if r.scopes != nil {
		if exprID, ok := r.scopes.Lookup(name); ok {
			return r.vault.GetUnresolvedValue(exprID)
		}
	}
	// Try decorator key lookup
	if exprID, ok := r.decoratorExprIDs[name]; ok {
		return r.vault.GetUnresolvedValue(exprID)
	}
	return nil, false
}

// resolve is the main resolution entry point.
//
// The resolver processes statements in waves, preserving tree structure:
//  1. Collect expressions from statements until we hit a blocker
//  2. Batch resolve all collected expressions (grouped by decorator type)
//  3. Evaluate the blocker condition
//  4. Prune untaken branch, recursively resolve taken branch
//  5. Continue with remaining statements
//
// The result is a pruned tree where:
//   - Blockers remain as nodes with Taken set
//   - Untaken branches are set to nil
//   - For-loops have Iterations populated with deep-copied bodies
//   - All nested blockers are recursively resolved
func (r *Resolver) resolve() (*ResolveResult, error) {
	// Select statements based on mode (script vs command)
	stmts := r.selectStatements()
	if stmts == nil {
		// Error already recorded
		return nil, r.buildError()
	}

	if r.activeFunction != nil {
		if err := r.resolvePrelude(r.activeFunction); err != nil {
			return nil, err
		}
	}

	// Resolve the statement list, returning the pruned tree
	resolved, err := r.resolveStatements(stmts)
	if err != nil {
		return nil, err
	}

	decoratorExprIDs := make(map[string]string, len(r.decoratorExprIDs))
	for key, exprID := range r.decoratorExprIDs {
		decoratorExprIDs[key] = exprID
	}

	return &ResolveResult{
		Statements:       resolved,
		DecoratorExprIDs: decoratorExprIDs,
	}, nil
}

// resolveStatements resolves a list of statements, returning the pruned tree.
// This is the core recursive resolution function.
func (r *Resolver) resolveStatements(stmts []*StatementIR) ([]*StatementIR, error) {
	if len(stmts) == 0 {
		return nil, nil
	}

	var result []*StatementIR

	for i := 0; i < len(stmts); i++ {
		stmt := stmts[i]

		switch stmt.Kind {
		case StmtVarDecl:
			// Collect and resolve variable declaration
			r.collectVarDecl(stmt.VarDecl)
			if err := r.buildError(); err != nil {
				return nil, err
			}
			if err := r.batchResolve(); err != nil {
				return nil, err
			}
			result = append(result, stmt)

		case StmtCommand:
			// Collect expressions in command
			r.collectCommand(stmt.Command)
			if err := r.buildError(); err != nil {
				return nil, err
			}
			if err := r.batchResolve(); err != nil {
				return nil, err
			}
			if err := r.resolveCommandBlock(stmt.Command); err != nil {
				return nil, err
			}
			result = append(result, stmt)

		case StmtBlocker:
			// Collect blocker condition, resolve, evaluate, then recurse into taken branch
			resolvedBlocker, err := r.resolveBlocker(stmt)
			if err != nil {
				return nil, err
			}
			result = append(result, resolvedBlocker)

		case StmtTry:
			// Try/catch is a runtime construct - resolve all branches
			resolvedTry, err := r.resolveTry(stmt)
			if err != nil {
				return nil, err
			}
			result = append(result, resolvedTry)
		}
	}

	return result, nil
}

// collectCommand collects all expressions in a command for resolution.
func (r *Resolver) collectCommand(cmd *CommandStmtIR) {
	if cmd == nil || cmd.Command == nil {
		return
	}
	for _, part := range cmd.Command.Parts {
		r.collectExpr(part, "")
	}
	// Also collect expressions in decorator args
	for _, arg := range cmd.Args {
		r.collectExpr(arg.Value, "")
	}
}

// resolveBlocker resolves a blocker statement and returns the pruned result.
// The blocker node is preserved with Taken set and untaken branch pruned.
func (r *Resolver) resolveBlocker(stmt *StatementIR) (*StatementIR, error) {
	blocker := stmt.Blocker

	// Collect blocker condition/collection
	r.collectExpr(blocker.Condition, "")
	if blocker.Kind == BlockerFor && blocker.Collection != nil {
		r.collectExpr(blocker.Collection, "")
	}

	// Check for errors from collection
	if err := r.buildError(); err != nil {
		return nil, err
	}

	// Batch resolve expressions
	if err := r.batchResolve(); err != nil {
		return nil, err
	}

	// Evaluate the blocker
	switch blocker.Kind {
	case BlockerIf:
		return r.resolveIfBlocker(stmt)
	case BlockerFor:
		return r.resolveForBlocker(stmt)
	case BlockerWhen:
		return r.resolveWhenBlocker(stmt)
	default:
		return nil, fmt.Errorf("unknown blocker kind: %d", blocker.Kind)
	}
}

// resolveIfBlocker evaluates an if statement and returns the pruned result.
func (r *Resolver) resolveIfBlocker(stmt *StatementIR) (*StatementIR, error) {
	blocker := stmt.Blocker

	// Evaluate condition
	result, err := EvaluateExpr(blocker.Condition, r.getValue)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate if condition: %w", err)
	}

	taken := IsTruthy(result)
	blocker.Taken = &taken

	// Resolve taken branch, prune untaken branch
	if taken {
		resolved, err := r.resolveStatements(blocker.ThenBranch)
		if err != nil {
			return nil, err
		}
		blocker.ThenBranch = resolved
		blocker.ElseBranch = nil // Prune untaken branch
	} else {
		if blocker.ElseBranch != nil {
			resolved, err := r.resolveStatements(blocker.ElseBranch)
			if err != nil {
				return nil, err
			}
			blocker.ElseBranch = resolved
		}
		blocker.ThenBranch = nil // Prune untaken branch
	}

	return stmt, nil
}

// resolveForBlocker evaluates a for-loop and populates Iterations.
func (r *Resolver) resolveForBlocker(stmt *StatementIR) (*StatementIR, error) {
	blocker := stmt.Blocker

	// Evaluate collection
	collection, err := r.evaluateCollection(blocker.Collection)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate for collection: %w", err)
	}

	// Build iterations with deep-copied bodies
	blocker.Iterations = make([]LoopIteration, len(collection))
	for i, item := range collection {
		// Create exprID for this iteration's loop variable
		loopVarRaw := fmt.Sprintf("literal:%v", item)
		loopVarExprID := r.vault.DeclareVariable(blocker.LoopVar, loopVarRaw)

		// Store the value and update scope
		r.vault.StoreUnresolvedValue(loopVarExprID, item)
		r.vault.MarkTouched(loopVarExprID)
		if r.scopes != nil {
			r.scopes.Define(blocker.LoopVar, loopVarExprID)
		}

		// Deep-copy the body for this iteration
		bodyCopy := DeepCopyStatements(blocker.ThenBranch)

		// Resolve the copied body (may contain nested blockers)
		resolvedBody, err := r.resolveStatements(bodyCopy)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve for-loop iteration %d: %w", i, err)
		}

		blocker.Iterations[i] = LoopIteration{
			Value: item,
			Body:  resolvedBody,
		}
	}

	// Clear ThenBranch since we've moved content to Iterations
	// (ThenBranch was the template, Iterations are the resolved copies)
	blocker.ThenBranch = nil

	return stmt, nil
}

// resolveWhenBlocker evaluates a when statement and returns the pruned result.
func (r *Resolver) resolveWhenBlocker(stmt *StatementIR) (*StatementIR, error) {
	blocker := stmt.Blocker

	// Evaluate condition
	value, err := EvaluateExpr(blocker.Condition, r.getValue)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate when condition: %w", err)
	}

	// Find first matching arm
	blocker.MatchedArm = -1
	for i, arm := range blocker.Arms {
		if matchPattern(arm.Pattern, value, r.getValue) {
			blocker.MatchedArm = i

			// Resolve the matched arm's body
			resolved, err := r.resolveStatements(arm.Body)
			if err != nil {
				return nil, err
			}
			arm.Body = resolved

			// Clear other arms' bodies (pruned)
			for j, otherArm := range blocker.Arms {
				if j != i {
					otherArm.Body = nil
				}
			}
			break
		}
	}

	return stmt, nil
}

// resolveTry resolves a try/catch/finally statement.
// Both try and catch branches are preserved (runtime determines which executes).
func (r *Resolver) resolveTry(stmt *StatementIR) (*StatementIR, error) {
	try := stmt.Try

	// Resolve all branches - they all need to be in the plan
	var err error

	r.scopes.Push()
	defer r.scopes.Pop()
	try.TryBlock, err = r.resolveStatements(try.TryBlock)
	if err != nil {
		return nil, err
	}

	r.scopes.Push()
	defer r.scopes.Pop()
	try.CatchBlock, err = r.resolveStatements(try.CatchBlock)
	if err != nil {
		return nil, err
	}

	r.scopes.Push()
	defer r.scopes.Pop()
	try.FinallyBlock, err = r.resolveStatements(try.FinallyBlock)
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

func (r *Resolver) resolveCommandBlock(cmd *CommandStmtIR) error {
	if cmd == nil || len(cmd.Block) == 0 {
		return nil
	}

	r.scopes.Push()
	defer r.scopes.Pop()
	resolved, err := r.resolveStatements(cmd.Block)
	if err != nil {
		return err
	}

	cmd.Block = resolved
	return nil
}

func (r *Resolver) resolvePrelude(fn *FunctionIR) error {
	if fn == nil {
		return nil
	}

	for _, stmt := range r.graph.Statements {
		if stmt.Kind != StmtVarDecl {
			continue
		}
		if stmt.Span.Start >= fn.Span.Start {
			continue
		}

		r.collectVarDecl(stmt.VarDecl)
		if err := r.buildError(); err != nil {
			return err
		}
		if err := r.batchResolve(); err != nil {
			return err
		}
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
		r.activeFunction = fn
		if fn.Scopes != nil {
			r.scopes = fn.Scopes
		} else {
			r.scopes = r.graph.Scopes
		}
		return fn.Body
	}

	// Script mode: all top-level statements
	r.activeFunction = nil
	r.scopes = r.graph.Scopes
	return r.graph.Statements
}

// collectVarDecl handles variable declaration - stores value by variable name for condition evaluation.
func (r *Resolver) collectVarDecl(decl *VarDeclIR) {
	exprID := decl.ExprID

	// Generate ExprID if not already set.
	// ExprID is intentionally NOT set during IR building - it's generated here
	// during resolution based on the actual scope context (loop iteration, transport).
	// This ensures each loop iteration gets a unique ExprID for variables declared
	// in the loop body.
	if exprID == "" {
		raw := r.buildVarDeclRaw(decl)
		exprID = r.vault.DeclareVariable(decl.Name, raw)
		decl.ExprID = exprID // Update the IR with the generated ExprID
	}

	// Update scopes so subsequent ExprVarRef lookups can find this variable.
	// This is critical for:
	// 1. Variables declared in taken branches (mutations leak per spec)
	// 2. Loop variables injected during for-loop unrolling
	if r.scopes != nil {
		r.scopes.Define(decl.Name, exprID)
	}

	// Collect the value expression (may add pending decorator calls)
	r.collectExprForVar(decl.Value, exprID, decl.Name)

	// Handle different expression types
	switch decl.Value.Kind {
	case ExprLiteral:
		// For literals, store the value immediately in Vault
		r.vault.StoreUnresolvedValue(exprID, decl.Value.Value)
		r.vault.MarkTouched(exprID)

	case ExprVarRef:
		// For var refs, look up the referenced value and store it
		// Also mark this VarDecl's ExprID as touched
		refExprID, ok := r.lookupScopeExprID(decl.Value.VarName)
		if ok {
			// Get the value from the referenced variable
			if val, exists := r.vault.GetUnresolvedValue(refExprID); exists {
				r.vault.StoreUnresolvedValue(exprID, val)
			}
			r.vault.MarkTouched(exprID)
		}
		// Note: If the variable isn't found, collectExprForVar already added an error

	case ExprDecoratorRef:
		// For decorator refs, the value will be stored after batch resolution
		// Mark as touched now so it's included in resolution
		r.vault.MarkTouched(exprID)
	}
}

// lookupScopeExprID finds an exprID using the active scope stack.
func (r *Resolver) lookupScopeExprID(name string) (string, bool) {
	if r.scopes != nil {
		if exprID, ok := r.scopes.Lookup(name); ok {
			return exprID, true
		}
	}
	return "", false
}

// buildVarDeclRaw builds a raw string for a variable declaration.
// The raw string is used to generate a deterministic ExprID.
//
// The raw includes:
// - For literals: "literal:<value>"
// - For decorator refs: "@decorator.selector"
// - For var refs: "varref:<name>:<referenced_exprID>" (includes dependency)
//
// Including the referenced ExprID for var refs ensures that:
//
//	for item in ["a", "b"] { var X = @var.item }
//
// produces unique ExprIDs for X in each iteration, because item has
// different ExprIDs per iteration.
func (r *Resolver) buildVarDeclRaw(decl *VarDeclIR) string {
	if decl.Value == nil {
		return fmt.Sprintf("var:%s:nil", decl.Name)
	}

	switch decl.Value.Kind {
	case ExprLiteral:
		return fmt.Sprintf("literal:%v", decl.Value.Value)

	case ExprDecoratorRef:
		return buildDecoratorRaw(decl.Value.Decorator)

	case ExprVarRef:
		// Include the referenced variable's ExprID to ensure uniqueness
		// when the same var ref appears in different scopes (e.g., loop iterations)
		refExprID, ok := r.lookupScopeExprID(decl.Value.VarName)
		if !ok {
			// Variable not found - will be caught later as an error
			return fmt.Sprintf("varref:%s:undefined", decl.Value.VarName)
		}
		return fmt.Sprintf("varref:%s:%s", decl.Value.VarName, refExprID)

	default:
		return fmt.Sprintf("expr:%s:%d", decl.Name, decl.Value.Kind)
	}
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
		varExprID, ok := r.lookupScopeExprID(expr.VarName)
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
		// Still generate DisplayIDs for literals/vars touched in this wave
		r.vault.ResolveAllTouched()
		return nil
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

		// Store value in Vault (single source of truth)
		r.vault.StoreUnresolvedValue(exprID, result.Value)
		r.vault.MarkTouched(exprID)

		// Track decorator key → exprID for getValue() lookups
		// This allows direct decorator refs in conditions (e.g., if @env.HOME == "/root")
		decKey := decoratorKey(call.decorator)
		r.decoratorExprIDs[decKey] = exprID

		// Note: Variable name → exprID is already tracked in ScopeStack
		// via collectVarDecl, so no need to duplicate here
	}

	return nil
}

// evaluateCollection evaluates a collection expression for a for-loop.
func (r *Resolver) evaluateCollection(expr *ExprIR) ([]any, error) {
	value, err := EvaluateExpr(expr, r.getValue)
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
func matchPattern(pattern *ExprIR, value any, getValue ValueLookup) bool {
	// Evaluate pattern expression
	patternValue, err := EvaluateExpr(pattern, getValue)
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
