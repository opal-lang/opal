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
	StmtCommand StatementKind = iota // Shell command or decorator invocation
	StmtVarDecl                      // Variable declaration
	StmtBlocker                      // Control flow (if/when/for)
	StmtTry                          // Try/catch/finally error handling
)

// StatementIR represents a statement in the execution graph.
type StatementIR struct {
	Kind         StatementKind
	Span         SourceSpan
	CreatesScope bool // True for decorator blocks, try/catch

	// Exactly one of these is set based on Kind
	Command *CommandStmtIR // For StmtCommand
	VarDecl *VarDeclIR     // For StmtVarDecl
	Blocker *BlockerIR     // For StmtBlocker
	Try     *TryIR         // For StmtTry
}

// CommandStmtIR represents a command statement.
// Note: Named CommandStmtIR to avoid conflict with existing CommandIR in planner.go.
// Will be renamed when old planner is removed.
type CommandStmtIR struct {
	Decorator string         // "@shell", "@retry", etc.
	Command   *CommandExpr   // The command with interpolated expressions
	Args      []*ExprIR      // Decorator arguments
	Block     []*StatementIR // Nested statements (for decorator blocks)
	Operator  string         // "&&", "||", "|", ";" - chain to next command
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

	// When-specific (pattern matching)
	Arms []*WhenArmIR // Pattern arms (for "when expr { pattern -> ... }")
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
	ErrorVar     string         // Variable name for caught error (optional)
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
