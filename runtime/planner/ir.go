package planner

// ExecutionGraph is the complete execution graph built from parser events.
// Contains all possible execution paths - the Resolver determines which are taken.
type ExecutionGraph struct {
	Statements []*StatementIR         // Top-level statements (script mode)
	Functions  map[string]*FunctionIR // Function definitions (command mode)
	Scopes     *ScopeStack            // Variable scopes (name → exprID)
}

// FunctionIR represents a function definition.
type FunctionIR struct {
	Name   string
	Params []ParamIR // Function parameters
	Body   []*StatementIR
	Span   SourceSpan
	Scopes *ScopeStack // Scope snapshot for command mode prelude
}

// ParamIR represents a function parameter.
type ParamIR struct {
	Name    string
	Type    string  // Type annotation (optional)
	Default *ExprIR // Default value (optional)
}

// StatementKind identifies the type of statement.
type StatementKind int

const (
	StmtCommand      StatementKind = iota // Shell command or decorator invocation
	StmtVarDecl                           // Variable declaration
	StmtBlocker                           // Control flow (if/when/for)
	StmtTry                               // Try/catch/finally error handling
	StmtFunctionCall                      // Function call statement
	StmtCallTrace                         // Call provenance wrapper (display-only)
)

// StatementIR represents a statement in the execution graph.
type StatementIR struct {
	Kind         StatementKind
	Span         SourceSpan
	CreatesScope bool // True for decorator blocks, try/catch

	// Exactly one of these is set based on Kind
	Command      *CommandStmtIR      // For StmtCommand
	VarDecl      *VarDeclIR          // For StmtVarDecl
	Blocker      *BlockerIR          // For StmtBlocker
	Try          *TryIR              // For StmtTry
	FunctionCall *FunctionCallStmtIR // For StmtFunctionCall
	CallTrace    *CallTraceStmtIR    // For StmtCallTrace
}

// CommandStmtIR represents a command statement.
type CommandStmtIR struct {
	Decorator      string         // "@shell", "@retry", etc.
	Command        *CommandExpr   // The command with interpolated expressions
	Args           []ArgIR        // Decorator arguments
	Block          []*StatementIR // Nested statements (for decorator blocks)
	Operator       string         // "&&", "||", "|", ";" - chain to next command
	RedirectMode   string         // ">", ">>" - redirect mode (empty if no redirect)
	RedirectTarget *CommandExpr   // For redirect operators, the target path (nil otherwise)
}

// FunctionCallStmtIR represents a function call statement.
type FunctionCallStmtIR struct {
	Name string  // Function name
	Args []ArgIR // Call arguments (positional and named)
}

// CallTraceStmtIR wraps expanded statements with function-call provenance.
// This is display-only metadata and does not affect execution semantics.
type CallTraceStmtIR struct {
	Label string         // e.g. deploy(token=opal:abc123) in rendered output
	Block []*StatementIR // Fully resolved expanded statements
}

// ArgIR represents a decorator argument.
type ArgIR struct {
	Name  string // Parameter name
	Value *ExprIR
}

// VarDeclIR represents a variable declaration.
type VarDeclIR struct {
	Name   string  // Variable name (without @var. prefix)
	Value  *ExprIR // Value expression
	ExprID string  // Unique expression ID (set during IR building)
}

// BlockerKind identifies the type of control flow blocker.
type BlockerKind int

const (
	BlockerIf   BlockerKind = iota // if condition { ... } else { ... }
	BlockerWhen                    // when expr { pattern -> ... }
	BlockerFor                     // for item in collection { ... }
)

// BlockerIR represents control flow that blocks execution until resolved.
// The Resolver evaluates conditions and sets the Taken flag.
//
// After resolution:
//   - if/when: Taken is set, untaken branch is nil (pruned)
//   - for: Iterations contains resolved loop iterations with deep-copied body
type BlockerIR struct {
	Kind       BlockerKind
	Depth      int            // Nesting level (for wave-based resolution)
	Condition  *ExprIR        // Condition expression (for if/when)
	ThenBranch []*StatementIR // Statements if condition is true
	ElseBranch []*StatementIR // Statements if condition is false (optional)
	Taken      *bool          // Set by Resolver: true=then, false=else, nil=unresolved

	// For-loop specific
	LoopVar    string  // Loop variable name (for "for x in ...")
	Collection *ExprIR // Collection expression (for "for x in collection")

	// Set by Resolver for for-loops: resolved iterations with deep-copied body
	Iterations []LoopIteration

	// When-specific (pattern matching)
	Arms []*WhenArmIR // Pattern arms (for "when expr { pattern -> ... }")

	// Set by Resolver for when: index of the matched arm (-1 if none)
	MatchedArm int
}

// LoopIteration represents one iteration of a resolved for-loop.
// Each iteration has its own deep-copied body to allow independent resolution
// of nested blockers (e.g., if statements inside the loop).
type LoopIteration struct {
	Value any            // Loop variable value for this iteration
	Body  []*StatementIR // Deep-copied statements for this iteration
}

// WhenArmIR represents a single arm in a when statement.
type WhenArmIR struct {
	Pattern *ExprIR        // Pattern to match (literal, regex, range, else)
	Body    []*StatementIR // Statements to execute if pattern matches
}

// TryIR represents try/catch/finally error handling.
type TryIR struct {
	TryBlock     []*StatementIR // Statements in try block
	CatchBlock   []*StatementIR // Statements in catch block (optional)
	FinallyBlock []*StatementIR // Statements in finally block (optional)
}

// ScopeStack tracks variable scopes during IR building.
// Each scope maps variable names to their expression IDs.
type ScopeStack struct {
	scopes []map[string]string // Stack of name → exprID maps
}

// NewScopeStack creates a new scope stack with a root scope.
func NewScopeStack() *ScopeStack {
	return &ScopeStack{
		scopes: []map[string]string{make(map[string]string)},
	}
}

// Push creates a new scope.
func (s *ScopeStack) Push() {
	s.scopes = append(s.scopes, make(map[string]string))
}

// Pop removes the current scope.
func (s *ScopeStack) Pop() {
	if len(s.scopes) > 1 {
		s.scopes = s.scopes[:len(s.scopes)-1]
	}
}

// Define adds a variable to the current scope.
func (s *ScopeStack) Define(name, exprID string) {
	if len(s.scopes) > 0 {
		s.scopes[len(s.scopes)-1][name] = exprID
	}
}

// Lookup finds a variable in the scope stack (innermost first).
// Returns the exprID and true if found, empty string and false otherwise.
func (s *ScopeStack) Lookup(name string) (string, bool) {
	// Search from innermost to outermost scope
	for i := len(s.scopes) - 1; i >= 0; i-- {
		if exprID, ok := s.scopes[i][name]; ok {
			return exprID, true
		}
	}
	return "", false
}

// Depth returns the current scope depth.
func (s *ScopeStack) Depth() int {
	return len(s.scopes)
}

// Clone returns a deep copy of the scope stack.
func (s *ScopeStack) Clone() *ScopeStack {
	cloned := make([]map[string]string, len(s.scopes))
	for i, scope := range s.scopes {
		scopeCopy := make(map[string]string, len(scope))
		for key, value := range scope {
			scopeCopy[key] = value
		}
		cloned[i] = scopeCopy
	}
	return &ScopeStack{scopes: cloned}
}

// DeepCopyStatements creates a deep copy of a statement slice.
// Used by the Resolver to create independent copies for each loop iteration,
// allowing nested blockers to be resolved independently.
func DeepCopyStatements(stmts []*StatementIR) []*StatementIR {
	if stmts == nil {
		return nil
	}
	result := make([]*StatementIR, len(stmts))
	for i, stmt := range stmts {
		result[i] = DeepCopyStatement(stmt)
	}
	return result
}

// DeepCopyStatement creates a deep copy of a single statement.
func DeepCopyStatement(stmt *StatementIR) *StatementIR {
	if stmt == nil {
		return nil
	}
	result := &StatementIR{
		Kind:         stmt.Kind,
		Span:         stmt.Span,
		CreatesScope: stmt.CreatesScope,
	}
	switch stmt.Kind {
	case StmtCommand:
		result.Command = deepCopyCommandStmt(stmt.Command)
	case StmtVarDecl:
		result.VarDecl = deepCopyVarDecl(stmt.VarDecl)
	case StmtBlocker:
		result.Blocker = deepCopyBlocker(stmt.Blocker)
	case StmtTry:
		result.Try = deepCopyTry(stmt.Try)
	case StmtFunctionCall:
		result.FunctionCall = deepCopyFunctionCallStmt(stmt.FunctionCall)
	case StmtCallTrace:
		result.CallTrace = deepCopyCallTraceStmt(stmt.CallTrace)
	}
	return result
}

func deepCopyCallTraceStmt(trace *CallTraceStmtIR) *CallTraceStmtIR {
	if trace == nil {
		return nil
	}
	return &CallTraceStmtIR{
		Label: trace.Label,
		Block: DeepCopyStatements(trace.Block),
	}
}

func deepCopyFunctionCallStmt(call *FunctionCallStmtIR) *FunctionCallStmtIR {
	if call == nil {
		return nil
	}
	return &FunctionCallStmtIR{
		Name: call.Name,
		Args: deepCopyArgs(call.Args),
	}
}

func deepCopyCommandStmt(cmd *CommandStmtIR) *CommandStmtIR {
	if cmd == nil {
		return nil
	}
	return &CommandStmtIR{
		Decorator:      cmd.Decorator,
		Command:        deepCopyCommandExpr(cmd.Command),
		Args:           deepCopyArgs(cmd.Args),
		Block:          DeepCopyStatements(cmd.Block),
		Operator:       cmd.Operator,
		RedirectMode:   cmd.RedirectMode,
		RedirectTarget: deepCopyCommandExpr(cmd.RedirectTarget),
	}
}

func deepCopyArgs(args []ArgIR) []ArgIR {
	if args == nil {
		return nil
	}
	result := make([]ArgIR, len(args))
	for i, arg := range args {
		result[i] = ArgIR{
			Name:  arg.Name,
			Value: deepCopyExpr(arg.Value),
		}
	}
	return result
}

func deepCopyVarDecl(decl *VarDeclIR) *VarDeclIR {
	if decl == nil {
		return nil
	}
	return &VarDeclIR{
		Name:   decl.Name,
		Value:  deepCopyExpr(decl.Value),
		ExprID: decl.ExprID,
	}
}

func deepCopyBlocker(blocker *BlockerIR) *BlockerIR {
	if blocker == nil {
		return nil
	}
	result := &BlockerIR{
		Kind:       blocker.Kind,
		Depth:      blocker.Depth,
		Condition:  deepCopyExpr(blocker.Condition),
		ThenBranch: DeepCopyStatements(blocker.ThenBranch),
		ElseBranch: DeepCopyStatements(blocker.ElseBranch),
		LoopVar:    blocker.LoopVar,
		Collection: deepCopyExpr(blocker.Collection),
		MatchedArm: blocker.MatchedArm,
	}
	// Copy Taken pointer
	if blocker.Taken != nil {
		taken := *blocker.Taken
		result.Taken = &taken
	}
	// Copy Iterations
	if blocker.Iterations != nil {
		result.Iterations = make([]LoopIteration, len(blocker.Iterations))
		for i, iter := range blocker.Iterations {
			result.Iterations[i] = LoopIteration{
				Value: iter.Value,
				Body:  DeepCopyStatements(iter.Body),
			}
		}
	}
	// Copy Arms
	if blocker.Arms != nil {
		result.Arms = make([]*WhenArmIR, len(blocker.Arms))
		for i, arm := range blocker.Arms {
			result.Arms[i] = &WhenArmIR{
				Pattern: deepCopyExpr(arm.Pattern),
				Body:    DeepCopyStatements(arm.Body),
			}
		}
	}
	return result
}

func deepCopyTry(try *TryIR) *TryIR {
	if try == nil {
		return nil
	}
	return &TryIR{
		TryBlock:     DeepCopyStatements(try.TryBlock),
		CatchBlock:   DeepCopyStatements(try.CatchBlock),
		FinallyBlock: DeepCopyStatements(try.FinallyBlock),
	}
}

func deepCopyCommandExpr(cmd *CommandExpr) *CommandExpr {
	if cmd == nil {
		return nil
	}
	return &CommandExpr{
		Parts: deepCopyExprs(cmd.Parts),
	}
}

func deepCopyExprs(exprs []*ExprIR) []*ExprIR {
	if exprs == nil {
		return nil
	}
	result := make([]*ExprIR, len(exprs))
	for i, expr := range exprs {
		result[i] = deepCopyExpr(expr)
	}
	return result
}

func deepCopyExpr(expr *ExprIR) *ExprIR {
	if expr == nil {
		return nil
	}
	result := &ExprIR{
		Kind:    expr.Kind,
		Span:    expr.Span,
		Value:   expr.Value,
		VarName: expr.VarName,
		Op:      expr.Op,
		Left:    deepCopyExpr(expr.Left),
		Right:   deepCopyExpr(expr.Right),
	}
	if expr.Decorator != nil {
		result.Decorator = &DecoratorRef{
			Name:     expr.Decorator.Name,
			Selector: append([]string(nil), expr.Decorator.Selector...),
			Args:     deepCopyExprs(expr.Decorator.Args),
			ArgNames: append([]string(nil), expr.Decorator.ArgNames...),
		}
	}
	return result
}
