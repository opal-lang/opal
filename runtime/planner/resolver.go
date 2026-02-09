package planner

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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
	sessionStack   []decorator.Session
	config         ResolveConfig
	telemetry      *PlanTelemetry
	telemetryLevel TelemetryLevel
	scopes         *ScopeStack
	activeFunction *FunctionIR

	// Resolution state
	decoratorExprIDs map[string]string // decorator key (e.g., "env.HOME") → exprID
	pendingCalls     []decoratorCall   // Decorator calls to batch resolve
	errors           []error           // Collected errors

	// @env allowance context (non-idempotent transports forbid @env)
	envAllowed      bool
	envBlockedBy    string
	envContextStack []envContext
}

type envContext struct {
	allowed   bool
	decorator string
}

// ResolveConfig configures the resolution process.
type ResolveConfig struct {
	TargetFunction string          // Empty = script mode, non-empty = command mode
	Context        context.Context // Execution context
	PlanHash       []byte          // Deterministic plan hash/salt for value resolution context
	StepPath       string          // Step path prefix for value resolution provenance
	Telemetry      *PlanTelemetry  // Optional telemetry sink
	TelemetryLevel TelemetryLevel  // Telemetry level
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
	if config.Context == nil {
		config.Context = context.Background()
	}

	r := &Resolver{
		graph:            graph,
		vault:            v,
		session:          session,
		config:           config,
		telemetry:        config.Telemetry,
		telemetryLevel:   config.TelemetryLevel,
		decoratorExprIDs: make(map[string]string),
		envAllowed:       true,
	}

	if v != nil {
		v.EnterTransport(localTransportID(v.GetPlanKey()))
	}

	return r.resolve()
}

func (r *Resolver) checkContext() error {
	if r.config.Context == nil {
		return nil
	}
	if err := r.config.Context.Err(); err != nil {
		return fmt.Errorf("resolution canceled: %w", err)
	}
	return nil
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
	if stmts == nil && len(r.errors) > 0 {
		// Error already recorded
		return nil, r.buildError()
	}
	// stmts may be empty (nil or len==0) which is valid - produces empty plan

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
	var pending []*StatementIR

	for i := 0; i < len(stmts); i++ {
		if err := r.checkContext(); err != nil {
			return nil, err
		}
		stmt := stmts[i]

		switch stmt.Kind {
		case StmtVarDecl:
			r.collectVarDecl(stmt.VarDecl)
			if err := r.buildError(); err != nil {
				return nil, err
			}
			pending = append(pending, stmt)
			result = append(result, stmt)

		case StmtCommand:
			r.collectCommand(stmt.Command)
			if err := r.buildError(); err != nil {
				return nil, err
			}
			pending = append(pending, stmt)
			result = append(result, stmt)

		case StmtBlocker:
			r.collectBlockerInputs(stmt.Blocker)
			if err := r.buildError(); err != nil {
				return nil, err
			}

			if err := r.batchResolve(); err != nil {
				return nil, err
			}
			if err := r.finalizePendingStatements(pending); err != nil {
				return nil, err
			}
			pending = nil

			resolvedBlocker, err := r.resolveBlockerWithResolvedInputs(stmt)
			if err != nil {
				return nil, err
			}
			result = append(result, resolvedBlocker)

		case StmtTry:
			if err := r.batchResolve(); err != nil {
				return nil, err
			}
			if err := r.finalizePendingStatements(pending); err != nil {
				return nil, err
			}
			pending = nil

			// Try/catch is a runtime construct - resolve all branches
			resolvedTry, err := r.resolveTry(stmt)
			if err != nil {
				return nil, err
			}
			result = append(result, resolvedTry)
		}
	}

	if err := r.batchResolve(); err != nil {
		return nil, err
	}
	if err := r.finalizePendingStatements(pending); err != nil {
		return nil, err
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

func (r *Resolver) resolveVarDeclStatement(stmt *StatementIR) error {
	if stmt == nil {
		return nil
	}

	r.collectVarDecl(stmt.VarDecl)
	if err := r.buildError(); err != nil {
		return err
	}
	if err := r.batchResolve(); err != nil {
		return err
	}
	if stmt.VarDecl != nil {
		if err := r.checkTransportBoundaryExpr(stmt.VarDecl.Value); err != nil {
			return err
		}
	}

	return nil
}

func (r *Resolver) finalizePendingStatements(stmts []*StatementIR) error {
	for _, stmt := range stmts {
		if stmt == nil {
			continue
		}

		switch stmt.Kind {
		case StmtVarDecl:
			if stmt.VarDecl != nil {
				if err := r.checkTransportBoundaryExpr(stmt.VarDecl.Value); err != nil {
					return err
				}
			}
		case StmtCommand:
			if err := r.checkTransportBoundaryCommand(stmt.Command); err != nil {
				return err
			}
			if err := r.resolveCommandBlock(stmt.Command); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Resolver) collectBlockerInputs(blocker *BlockerIR) {
	if blocker == nil {
		return
	}

	// Collect blocker condition/collection
	r.collectExpr(blocker.Condition, "")
	if blocker.Kind == BlockerFor && blocker.Collection != nil {
		r.collectExpr(blocker.Collection, "")
	}
}

func (r *Resolver) resolveBlockerInputs(blocker *BlockerIR) error {
	r.collectBlockerInputs(blocker)

	if err := r.buildError(); err != nil {
		return err
	}

	return r.batchResolve()
}

func (r *Resolver) evaluateCondition(expr *ExprIR, description string, checkTransport bool) (any, error) {
	if checkTransport {
		if err := r.checkTransportBoundaryExpr(expr); err != nil {
			return nil, err
		}
	}

	result, err := EvaluateExpr(expr, r.getValue)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate %s: %w", description, err)
	}

	return result, nil
}

func (r *Resolver) evaluateBlockerCollection(blocker *BlockerIR, checkTransport bool) ([]any, error) {
	if checkTransport {
		if err := r.checkTransportBoundaryExpr(blocker.Collection); err != nil {
			return nil, err
		}
	}

	collection, err := r.evaluateCollection(blocker.Collection)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate for collection: %w", err)
	}

	return collection, nil
}

func (r *Resolver) bindLoopVar(loopVar string, item any) {
	loopVarRaw := fmt.Sprintf("literal:%v", item)
	loopVarExprID := r.vault.DeclareVariable(loopVar, loopVarRaw)

	r.vault.StoreUnresolvedValue(loopVarExprID, item)
	r.vault.MarkTouched(loopVarExprID)
	if r.scopes != nil {
		r.scopes.Define(loopVar, loopVarExprID)
	}
}

func (r *Resolver) resolveBlockerWithResolvedInputs(stmt *StatementIR) (*StatementIR, error) {
	blocker := stmt.Blocker

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

	result, err := r.evaluateCondition(blocker.Condition, "if condition", true)
	if err != nil {
		return nil, err
	}

	taken := IsTruthy(result)
	blocker.Taken = &taken

	// Resolve taken branch, prune untaken branch
	if taken {
		if r.scopes != nil {
			r.scopes.Push()
			defer r.scopes.Pop()
		}
		resolved, err := r.resolveStatements(blocker.ThenBranch)
		if err != nil {
			return nil, err
		}
		blocker.ThenBranch = resolved
		blocker.ElseBranch = nil // Prune untaken branch
	} else {
		if blocker.ElseBranch != nil {
			if r.scopes != nil {
				r.scopes.Push()
				defer r.scopes.Pop()
			}
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

	collection, err := r.evaluateBlockerCollection(blocker, true)
	if err != nil {
		return nil, err
	}

	// Build iterations with deep-copied bodies
	blocker.Iterations = make([]LoopIteration, len(collection))
	for i, item := range collection {
		if err := r.checkContext(); err != nil {
			return nil, err
		}

		var resolvedBody []*StatementIR
		err := func() error {
			if r.scopes != nil {
				r.scopes.Push()
				defer r.scopes.Pop()
			}

			r.bindLoopVar(blocker.LoopVar, item)

			// Deep-copy the body for this iteration
			bodyCopy := DeepCopyStatements(blocker.ThenBranch)

			// Resolve the copied body (may contain nested blockers)
			var err error
			resolvedBody, err = r.resolveStatements(bodyCopy)
			if err != nil {
				return fmt.Errorf("failed to resolve for-loop iteration %d: %w", i, err)
			}

			return nil
		}()
		if err != nil {
			return nil, err
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

	value, err := r.evaluateCondition(blocker.Condition, "when condition", true)
	if err != nil {
		return nil, err
	}

	// Find first matching arm
	blocker.MatchedArm = -1
	for i, arm := range blocker.Arms {
		if err := r.checkTransportBoundaryExpr(arm.Pattern); err != nil {
			return nil, err
		}
		if matchPattern(arm.Pattern, value, r.getValue) {
			blocker.MatchedArm = i

			// Resolve the matched arm's body
			var resolved []*StatementIR
			err := r.withScope(func() error {
				var err error
				resolved, err = r.resolveStatements(arm.Body)
				return err
			})
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

	if r.scopes != nil {
		r.scopes.Push()
		defer r.scopes.Pop()
	}
	try.TryBlock, err = r.resolveStatements(try.TryBlock)
	if err != nil {
		return nil, err
	}

	if r.scopes != nil {
		r.scopes.Push()
		defer r.scopes.Pop()
	}
	try.CatchBlock, err = r.resolveStatements(try.CatchBlock)
	if err != nil {
		return nil, err
	}

	if r.scopes != nil {
		r.scopes.Push()
		defer r.scopes.Pop()
	}
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

	var restoreTransport string
	if transportDec, desc, ok := lookupTransportDecorator(cmd.Decorator); ok {
		params, err := evaluateArgs(cmd.Args, r.getValue)
		if err != nil {
			return err
		}

		parentTransport := r.vault.CurrentTransport()
		transportID, err := deriveTransportID(r.vault.GetPlanKey(), cmd.Decorator, params, parentTransport)
		if err != nil {
			return err
		}

		restoreTransport = parentTransport
		r.vault.EnterTransport(transportID)
		defer func() {
			r.vault.EnterTransport(restoreTransport)
		}()

		if desc.Capabilities.Idempotent {
			session, err := transportDec.Open(r.session, params)
			if err != nil {
				return fmt.Errorf("failed to open transport %q: %w", cmd.Decorator, err)
			}
			if delta := extractEnvDelta(params); len(delta) > 0 {
				session = session.WithEnv(delta)
			}
			r.pushSession(session)
			defer r.popSession()
			r.pushEnvContext(true, "")
			defer r.popEnvContext()
		}
		if !desc.Capabilities.Idempotent {
			r.pushEnvContext(false, cmd.Decorator)
			defer r.popEnvContext()
		}
	}

	if r.scopes != nil {
		r.scopes.Push()
		defer r.scopes.Pop()
	}
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
		if err := r.checkContext(); err != nil {
			return err
		}
		if stmt.Span.Start >= fn.Span.Start {
			continue
		}
		if err := r.resolvePreludeStatement(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (r *Resolver) resolvePreludeStatements(stmts []*StatementIR) error {
	for _, stmt := range stmts {
		if err := r.checkContext(); err != nil {
			return err
		}
		if err := r.resolvePreludeStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resolver) resolvePreludeStatement(stmt *StatementIR) error {
	if stmt == nil {
		return nil
	}

	switch stmt.Kind {
	case StmtVarDecl:
		return r.resolveVarDeclStatement(stmt)
	case StmtBlocker:
		return r.resolvePreludeBlocker(stmt)
	case StmtCommand, StmtTry:
		return nil
	default:
		return nil
	}
}

func (r *Resolver) resolvePreludeBlocker(stmt *StatementIR) error {
	blocker := stmt.Blocker
	if blocker == nil {
		return nil
	}

	if err := r.resolveBlockerInputs(blocker); err != nil {
		return err
	}

	switch blocker.Kind {
	case BlockerIf:
		return r.resolvePreludeIf(blocker)
	case BlockerFor:
		return r.resolvePreludeFor(blocker)
	case BlockerWhen:
		return r.resolvePreludeWhen(blocker)
	default:
		return fmt.Errorf("unknown blocker kind: %d", blocker.Kind)
	}
}

func (r *Resolver) resolvePreludeIf(blocker *BlockerIR) error {
	result, err := r.evaluateCondition(blocker.Condition, "if condition", false)
	if err != nil {
		return err
	}

	if IsTruthy(result) {
		if r.scopes != nil {
			r.scopes.Push()
			defer r.scopes.Pop()
		}
		return r.resolvePreludeStatements(blocker.ThenBranch)
	}

	if blocker.ElseBranch != nil {
		if r.scopes != nil {
			r.scopes.Push()
			defer r.scopes.Pop()
		}
		return r.resolvePreludeStatements(blocker.ElseBranch)
	}

	return nil
}

func (r *Resolver) pushSession(session decorator.Session) {
	r.sessionStack = append(r.sessionStack, r.session)
	r.session = session
}

func (r *Resolver) popSession() {
	if len(r.sessionStack) == 0 {
		return
	}
	if r.session != nil {
		_ = r.session.Close()
	}
	prev := r.sessionStack[len(r.sessionStack)-1]
	r.sessionStack = r.sessionStack[:len(r.sessionStack)-1]
	r.session = prev
}

func (r *Resolver) pushEnvContext(allowed bool, decoratorName string) {
	r.envContextStack = append(r.envContextStack, envContext{
		allowed:   r.envAllowed,
		decorator: r.envBlockedBy,
	})
	if !r.envAllowed {
		allowed = false
		if r.envBlockedBy != "" {
			decoratorName = r.envBlockedBy
		}
	}
	r.envAllowed = allowed
	r.envBlockedBy = decoratorName
}

func (r *Resolver) popEnvContext() {
	if len(r.envContextStack) == 0 {
		return
	}
	prev := r.envContextStack[len(r.envContextStack)-1]
	r.envContextStack = r.envContextStack[:len(r.envContextStack)-1]
	r.envAllowed = prev.allowed
	r.envBlockedBy = prev.decorator
}

func extractEnvDelta(params map[string]any) map[string]string {
	if params == nil {
		return nil
	}
	value, ok := params["env"]
	if !ok {
		return nil
	}
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]string:
		return v
	case map[string]any:
		delta := make(map[string]string, len(v))
		for key, raw := range v {
			str, ok := raw.(string)
			if !ok {
				continue
			}
			delta[key] = str
		}
		return delta
	default:
		return nil
	}
}

func (r *Resolver) resolvePreludeFor(blocker *BlockerIR) error {
	collection, err := r.evaluateBlockerCollection(blocker, false)
	if err != nil {
		return err
	}

	for i, item := range collection {
		if err := r.checkContext(); err != nil {
			return err
		}
		err := func() error {
			if r.scopes != nil {
				r.scopes.Push()
				defer r.scopes.Pop()
			}

			r.bindLoopVar(blocker.LoopVar, item)

			if err := r.resolvePreludeStatements(blocker.ThenBranch); err != nil {
				return fmt.Errorf("failed to resolve for-loop iteration %d: %w", i, err)
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Resolver) resolvePreludeWhen(blocker *BlockerIR) error {
	value, err := r.evaluateCondition(blocker.Condition, "when condition", false)
	if err != nil {
		return err
	}

	for _, arm := range blocker.Arms {
		if matchPattern(arm.Pattern, value, r.getValue) {
			return r.withScope(func() error {
				return r.resolvePreludeStatements(arm.Body)
			})
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
		// Check if value is transport-sensitive and use appropriate declaration method
		if r.isExprTransportSensitive(decl.Value) {
			exprID = r.vault.DeclareVariableTransportSensitive(decl.Name, raw)
		} else {
			exprID = r.vault.DeclareVariable(decl.Name, raw)
		}
		decl.ExprID = exprID // Update the IR with the generated ExprID
	}

	// Update scopes so subsequent ExprVarRef lookups can find this variable.
	// This is critical for:
	// 1. Variables declared in taken branches (mutations leak per spec)
	// 2. Loop variables injected during for-loop unrolling
	if r.scopes != nil {
		r.scopes.Define(decl.Name, exprID)
	}

	// Record @var resolution for telemetry (declaration)
	r.recordVarResolution()

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
			if strings.HasPrefix(exprID, "placeholder:") {
				return "", false
			}
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
		if !r.checkEnvAllowed(expr) {
			return
		}
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
		// Record @var resolution for telemetry (reference)
		r.recordVarResolution()
		// Mark as touched (in execution path)
		r.vault.MarkTouched(varExprID)

		// Note: The value should already be in r.values[expr.VarName] from
		// when the variable was declared. If not, it will fail during evaluation.

	case ExprDecoratorRef:
		if !r.checkEnvAllowed(expr) {
			return
		}
		// Track decorator call for batch resolution
		raw := buildDecoratorRaw(expr.Decorator)

		// Generate or use provided exprID
		if exprID == "" {
			if r.isDecoratorTransportSensitive(expr.Decorator.Name) {
				exprID = r.vault.TrackExpressionTransportSensitive(raw)
			} else {
				exprID = r.vault.TrackExpression(raw)
			}
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

func (r *Resolver) checkEnvAllowed(expr *ExprIR) bool {
	if expr == nil || expr.Decorator == nil {
		return true
	}
	if expr.Decorator.Name != "env" {
		return true
	}
	if r.envAllowed {
		return true
	}
	decoratorName := r.envBlockedBy
	if decoratorName == "" {
		decoratorName = "non-idempotent transport"
	}
	r.errors = append(r.errors, &EvalError{
		Message: fmt.Sprintf("@env cannot be used inside %s", decoratorName),
		Span:    expr.Span,
	})
	return false
}

func (r *Resolver) checkTransportBoundaryCommand(cmd *CommandStmtIR) error {
	if cmd == nil {
		return nil
	}

	if cmd.Command != nil {
		for _, part := range cmd.Command.Parts {
			if err := r.checkTransportBoundaryExpr(part); err != nil {
				return err
			}
		}
	}

	for _, arg := range cmd.Args {
		if err := r.checkTransportBoundaryExpr(arg.Value); err != nil {
			return err
		}
	}

	if cmd.RedirectTarget != nil {
		for _, part := range cmd.RedirectTarget.Parts {
			if err := r.checkTransportBoundaryExpr(part); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Resolver) checkTransportBoundaryExpr(expr *ExprIR) error {
	if expr == nil {
		return nil
	}

	switch expr.Kind {
	case ExprLiteral:
		if arr, ok := expr.Value.([]*ExprIR); ok {
			for _, item := range arr {
				if err := r.checkTransportBoundaryExpr(item); err != nil {
					return err
				}
			}
		}
		if obj, ok := expr.Value.(map[string]*ExprIR); ok {
			for _, item := range obj {
				if err := r.checkTransportBoundaryExpr(item); err != nil {
					return err
				}
			}
		}
		return nil

	case ExprVarRef:
		exprID, ok := r.lookupScopeExprID(expr.VarName)
		if !ok {
			return nil
		}
		return r.vault.CheckTransportBoundary(exprID)

	case ExprDecoratorRef:
		key := decoratorKey(expr.Decorator)
		exprID, ok := r.decoratorExprIDs[key]
		if !ok {
			return nil
		}
		return r.vault.CheckTransportBoundary(exprID)

	case ExprBinaryOp:
		if err := r.checkTransportBoundaryExpr(expr.Left); err != nil {
			return err
		}
		return r.checkTransportBoundaryExpr(expr.Right)

	default:
		return nil
	}
}

// isExprTransportSensitive checks if an expression is transport-sensitive.
// Returns true for:
// - Decorator refs where the decorator has TransportSensitive capability
// - Var refs where the referenced variable is transport-sensitive
// - Binary ops where either operand is transport-sensitive
func (r *Resolver) isExprTransportSensitive(expr *ExprIR) bool {
	if expr == nil {
		return false
	}

	switch expr.Kind {
	case ExprLiteral:
		return false

	case ExprVarRef:
		// Check if the referenced variable is transport-sensitive
		refExprID, ok := r.lookupScopeExprID(expr.VarName)
		if !ok {
			return false
		}
		return r.vault.IsExpressionTransportSensitive(refExprID)

	case ExprDecoratorRef:
		// Check if the decorator has TransportSensitive capability
		return r.isDecoratorTransportSensitive(expr.Decorator.Name)

	case ExprBinaryOp:
		// Check if either operand is transport-sensitive
		return r.isExprTransportSensitive(expr.Left) || r.isExprTransportSensitive(expr.Right)

	default:
		return false
	}
}

// isDecoratorTransportSensitive checks if a decorator has the TransportSensitive capability.
func (r *Resolver) isDecoratorTransportSensitive(name string) bool {
	trimmed := strings.TrimPrefix(name, "@")
	if trimmed == "" {
		return false
	}
	entry, ok := decorator.Global().Lookup(trimmed)
	if !ok {
		return false
	}
	return entry.Impl.Descriptor().Capabilities.TransportSensitive
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

	decoratorNames := make([]string, 0, len(groups))
	for decoratorName := range groups {
		decoratorNames = append(decoratorNames, decoratorName)
	}
	sort.Strings(decoratorNames)

	// Resolve each group in batch
	for _, decoratorName := range decoratorNames {
		calls := groups[decoratorName]
		var duration time.Duration
		if r.telemetry != nil && r.telemetryLevel >= TelemetryTiming {
			start := time.Now()
			if err := r.resolveBatch(decoratorName, calls); err != nil {
				return err
			}
			duration = time.Since(start)
		} else {
			if err := r.resolveBatch(decoratorName, calls); err != nil {
				return err
			}
		}
		r.recordBatchResolution(decoratorName, len(calls), duration)
	}

	// Clear pending calls for next wave
	r.pendingCalls = nil

	// Generate DisplayIDs for all touched expressions
	r.vault.ResolveAllTouched()

	return nil
}

func (r *Resolver) recordVarResolution() {
	r.recordDecoratorResolution("@var", 1)
}

func (r *Resolver) recordDecoratorResolution(decoratorName string, count int) {
	if r.telemetry == nil {
		return
	}
	metrics := r.getOrCreateMetrics(decoratorName)
	if metrics == nil {
		return
	}
	metrics.TotalCalls += count
}

func (r *Resolver) recordBatchResolution(decoratorName string, batchSize int, duration time.Duration) {
	if r.telemetry == nil {
		return
	}
	metrics := r.getOrCreateMetrics(decoratorName)
	if metrics == nil {
		return
	}
	metrics.TotalCalls += batchSize
	metrics.BatchCalls++
	metrics.BatchSizes = append(metrics.BatchSizes, batchSize)
	if r.telemetryLevel >= TelemetryTiming {
		metrics.TotalTime += duration
	}
}

func (r *Resolver) getOrCreateMetrics(decoratorName string) *DecoratorResolutionMetrics {
	if r.telemetry == nil {
		return nil
	}
	if r.telemetry.DecoratorResolutions == nil {
		r.telemetry.DecoratorResolutions = make(map[string]*DecoratorResolutionMetrics)
	}
	metrics := r.telemetry.DecoratorResolutions[decoratorName]
	if metrics == nil {
		metrics = &DecoratorResolutionMetrics{BatchSizes: []int{}}
		r.telemetry.DecoratorResolutions[decoratorName] = metrics
	}
	return metrics
}

// resolveBatch resolves a batch of calls for a single decorator type.
func (r *Resolver) resolveBatch(decoratorName string, calls []decoratorCall) error {
	// Build ValueCall slice for batch resolution
	valueCalls := make([]decorator.ValueCall, len(calls))
	for i, call := range calls {
		valueCall, err := buildValueCall(call.decorator, r.getValue)
		if err != nil {
			return err
		}
		valueCalls[i] = valueCall
	}

	// Build evaluation context
	stepPath := r.config.StepPath
	if stepPath == "" {
		stepPath = "planner.resolve"
	}
	if decoratorName != "" {
		stepPath = stepPath + "." + decoratorName
	}

	planHash := r.config.PlanHash
	if len(planHash) == 0 {
		planHash = r.vault.GetPlanKey()
	}

	ctx := decorator.ValueEvalContext{
		Session:     r.session,
		LookupValue: r.getValue,
		PlanHash:    planHash,
		StepPath:    stepPath,
	}

	// Get current transport scope
	currentScope := r.session.TransportScope()

	// Call decorator's batch Resolve
	// This is where the magic happens - multiple @aws.secret calls → one API request
	results, err := decorator.Global().ResolveValues(ctx, currentScope, valueCalls...)
	if err != nil {
		return fmt.Errorf("failed to resolve @%s: %w (cannot plan if cannot resolve)", decoratorName, err)
	}
	if len(results) != len(calls) {
		return fmt.Errorf("internal error: resolver received %d results for %d @%s calls", len(results), len(calls), decoratorName)
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
func buildValueCall(d *DecoratorRef, getValue ValueLookup) (decorator.ValueCall, error) {
	if d == nil {
		return decorator.ValueCall{Params: map[string]any{}}, nil
	}

	call := decorator.ValueCall{
		Path:   d.Name,
		Params: make(map[string]any),
	}

	// If there's a selector, use the first element as Primary
	if len(d.Selector) > 0 {
		primary := d.Selector[0]
		call.Primary = &primary
	}

	for i, arg := range d.Args {
		if arg == nil {
			continue
		}
		value, err := EvaluateExpr(arg, getValue)
		if err != nil {
			return decorator.ValueCall{}, fmt.Errorf("failed to evaluate decorator arg %d for @%s: %w", i+1, call.Path, err)
		}

		paramName := fmt.Sprintf("arg%d", i+1)
		if i < len(d.ArgNames) {
			candidate := d.ArgNames[i]
			if candidate != "" {
				if _, positional := parsePositionalArgKey(candidate); !positional {
					paramName = candidate
				}
			}
		}

		if _, exists := call.Params[paramName]; exists {
			return decorator.ValueCall{}, fmt.Errorf("duplicate decorator argument %q for @%s", paramName, call.Path)
		}
		call.Params[paramName] = value
	}

	return call, nil
}

func parsePositionalArgKey(key string) (int, bool) {
	if !strings.HasPrefix(key, "arg") {
		return 0, false
	}
	if len(key) <= 3 {
		return 0, false
	}

	index, err := strconv.Atoi(key[3:])
	if err != nil || index <= 0 {
		return 0, false
	}

	return index, true
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

func (r *Resolver) withScope(run func() error) error {
	if r.scopes == nil {
		return run()
	}

	r.scopes.Push()
	defer r.scopes.Pop()
	return run()
}

// transportStringToScope converts a transport string to TransportScope.
