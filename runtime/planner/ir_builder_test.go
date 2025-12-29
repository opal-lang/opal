package planner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/runtime/parser"

	// Import decorators to populate the registry for @var detection
	_ "github.com/opal-lang/opal/runtime/decorators"
)

// Helper to parse source and build IR
func buildIR(t *testing.T, source string) *ExecutionGraph {
	t.Helper()

	tree := parser.Parse([]byte(source))

	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	graph, err := BuildIR(tree.Events, tree.Tokens)
	if err != nil {
		t.Fatalf("BuildIR() error = %v", err)
	}

	return graph
}

// ========== Basic IR Building Tests ==========

func TestBuildIR_EmptySource(t *testing.T) {
	graph := buildIR(t, "")

	if graph == nil {
		t.Fatal("BuildIR() returned nil")
	}
	if len(graph.Statements) != 0 {
		t.Errorf("len(Statements) = %d, want 0", len(graph.Statements))
	}
}

func TestBuildIR_SimpleCommand(t *testing.T) {
	graph := buildIR(t, `echo "hello"`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtCommand {
		t.Errorf("stmt.Kind = %v, want StmtCommand", stmt.Kind)
	}
	if stmt.Command == nil {
		t.Fatal("stmt.Command is nil")
	}
	if stmt.Command.Decorator != "@shell" {
		t.Errorf("Decorator = %q, want %q", stmt.Command.Decorator, "@shell")
	}
}

func TestBuildIR_VarDecl_Literal(t *testing.T) {
	graph := buildIR(t, `var ENV = "production"`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtVarDecl {
		t.Errorf("stmt.Kind = %v, want StmtVarDecl", stmt.Kind)
	}
	if stmt.VarDecl == nil {
		t.Fatal("stmt.VarDecl is nil")
	}
	if stmt.VarDecl.Name != "ENV" {
		t.Errorf("Name = %q, want %q", stmt.VarDecl.Name, "ENV")
	}
	if stmt.VarDecl.Value == nil {
		t.Fatal("Value is nil")
	}
	if stmt.VarDecl.Value.Kind != ExprLiteral {
		t.Errorf("Value.Kind = %v, want ExprLiteral", stmt.VarDecl.Value.Kind)
	}
	// String value - the lexer may include or exclude quotes
	val, ok := stmt.VarDecl.Value.Value.(string)
	if !ok {
		t.Fatalf("Value.Value is not string, got %T", stmt.VarDecl.Value.Value)
	}
	if val != "production" && val != `"production"` {
		t.Errorf("Value.Value = %q, want %q", val, "production")
	}
}

func TestBuildIR_VarDecl_Int(t *testing.T) {
	graph := buildIR(t, `var COUNT = 42`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.VarDecl.Value.Kind != ExprLiteral {
		t.Errorf("Value.Kind = %v, want ExprLiteral", stmt.VarDecl.Value.Kind)
	}
	if stmt.VarDecl.Value.Value != int64(42) {
		t.Errorf("Value.Value = %v (%T), want 42", stmt.VarDecl.Value.Value, stmt.VarDecl.Value.Value)
	}
}

func TestBuildIR_VarDecl_DecoratorRef(t *testing.T) {
	graph := buildIR(t, `var HOME = @env.HOME`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.VarDecl.Value.Kind != ExprDecoratorRef {
		t.Errorf("Value.Kind = %v, want ExprDecoratorRef", stmt.VarDecl.Value.Kind)
	}
	if stmt.VarDecl.Value.Decorator == nil {
		t.Fatal("Decorator is nil")
	}
	if stmt.VarDecl.Value.Decorator.Name != "env" {
		t.Errorf("Decorator.Name = %q, want %q", stmt.VarDecl.Value.Decorator.Name, "env")
	}
	if diff := cmp.Diff([]string{"HOME"}, stmt.VarDecl.Value.Decorator.Selector); diff != "" {
		t.Errorf("Decorator.Selector mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildIR_MultipleStatements(t *testing.T) {
	graph := buildIR(t, `var X = 1
echo "hello"
var Y = 2`)

	if len(graph.Statements) != 3 {
		t.Fatalf("len(Statements) = %d, want 3", len(graph.Statements))
	}

	if graph.Statements[0].Kind != StmtVarDecl {
		t.Errorf("Statements[0].Kind = %v, want StmtVarDecl", graph.Statements[0].Kind)
	}
	if graph.Statements[1].Kind != StmtCommand {
		t.Errorf("Statements[1].Kind = %v, want StmtCommand", graph.Statements[1].Kind)
	}
	if graph.Statements[2].Kind != StmtVarDecl {
		t.Errorf("Statements[2].Kind = %v, want StmtVarDecl", graph.Statements[2].Kind)
	}
}

func TestBuildIR_CommandWithVarRef(t *testing.T) {
	// Test direct @var.NAME usage (not inside string)
	graph := buildIR(t, `var NAME = "world"
echo @var.NAME`)

	if len(graph.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want 2", len(graph.Statements))
	}

	// Check the command has interpolated parts
	cmd := graph.Statements[1].Command
	if cmd == nil || cmd.Command == nil {
		t.Fatal("Command is nil")
	}

	parts := cmd.Command.Parts

	// Find the var ref part
	hasVarRef := false
	for _, part := range parts {
		if part.Kind == ExprVarRef && part.VarName == "NAME" {
			hasVarRef = true
			break
		}
	}
	if !hasVarRef {
		t.Error("Command should contain VarRef to NAME")
	}
}

// ========== Scope Tests ==========

func TestBuildIR_ScopeTracking(t *testing.T) {
	graph := buildIR(t, `var X = 1`)

	if graph.Scopes == nil {
		t.Fatal("Scopes is nil")
	}

	// Variable should be in scope
	exprID, ok := graph.Scopes.Lookup("X")
	if !ok {
		t.Error("Variable X not found in scope")
	}
	if exprID == "" {
		t.Error("ExprID for X is empty")
	}
}

// ========== If Statement Tests ==========

func TestBuildIR_IfWithLiteralCondition(t *testing.T) {
	graph := buildIR(t, `if true { echo "yes" }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}
	if stmt.Blocker == nil {
		t.Fatal("stmt.Blocker is nil")
	}
	if stmt.Blocker.Kind != BlockerIf {
		t.Errorf("Blocker.Kind = %v, want BlockerIf", stmt.Blocker.Kind)
	}

	// Check condition
	if stmt.Blocker.Condition == nil {
		t.Fatal("Condition is nil")
	}
	if stmt.Blocker.Condition.Kind != ExprLiteral {
		t.Errorf("Condition.Kind = %v, want ExprLiteral", stmt.Blocker.Condition.Kind)
	}
	if stmt.Blocker.Condition.Value != true {
		t.Errorf("Condition.Value = %v, want true", stmt.Blocker.Condition.Value)
	}

	// Check then branch
	if len(stmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(ThenBranch) = %d, want 1", len(stmt.Blocker.ThenBranch))
	}
	if stmt.Blocker.ThenBranch[0].Kind != StmtCommand {
		t.Errorf("ThenBranch[0].Kind = %v, want StmtCommand", stmt.Blocker.ThenBranch[0].Kind)
	}
}

func TestBuildIR_IfWithVarCondition(t *testing.T) {
	graph := buildIR(t, `var ENV = "prod"
if @var.ENV == "prod" { echo "production" }`)

	if len(graph.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want 2", len(graph.Statements))
	}

	stmt := graph.Statements[1]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}

	// Check condition is a binary expression
	if stmt.Blocker.Condition == nil {
		t.Fatal("Condition is nil")
	}
	if stmt.Blocker.Condition.Kind != ExprBinaryOp {
		t.Errorf("Condition.Kind = %v, want ExprBinaryOp", stmt.Blocker.Condition.Kind)
	}
	if stmt.Blocker.Condition.Op != "==" {
		t.Errorf("Condition.Op = %q, want %q", stmt.Blocker.Condition.Op, "==")
	}

	// Check left side is @var.ENV
	if stmt.Blocker.Condition.Left == nil {
		t.Fatal("Condition.Left is nil")
	}
	if stmt.Blocker.Condition.Left.Kind != ExprVarRef {
		t.Errorf("Condition.Left.Kind = %v, want ExprVarRef", stmt.Blocker.Condition.Left.Kind)
	}
	if stmt.Blocker.Condition.Left.VarName != "ENV" {
		t.Errorf("Condition.Left.VarName = %q, want %q", stmt.Blocker.Condition.Left.VarName, "ENV")
	}

	// Check right side is "prod"
	if stmt.Blocker.Condition.Right == nil {
		t.Fatal("Condition.Right is nil")
	}
	if stmt.Blocker.Condition.Right.Kind != ExprLiteral {
		t.Errorf("Condition.Right.Kind = %v, want ExprLiteral", stmt.Blocker.Condition.Right.Kind)
	}
}

func TestBuildIR_IfElse(t *testing.T) {
	graph := buildIR(t, `if false { echo "yes" } else { echo "no" }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Blocker == nil {
		t.Fatal("Blocker is nil")
	}

	// Check then branch
	if len(stmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(ThenBranch) = %d, want 1", len(stmt.Blocker.ThenBranch))
	}

	// Check else branch
	if len(stmt.Blocker.ElseBranch) != 1 {
		t.Errorf("len(ElseBranch) = %d, want 1", len(stmt.Blocker.ElseBranch))
	}
	if stmt.Blocker.ElseBranch[0].Kind != StmtCommand {
		t.Errorf("ElseBranch[0].Kind = %v, want StmtCommand", stmt.Blocker.ElseBranch[0].Kind)
	}
}

func TestBuildIR_IfWithIdentifierCondition(t *testing.T) {
	graph := buildIR(t, `var isReady = true
if isReady { echo "ready" }`)

	if len(graph.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want 2", len(graph.Statements))
	}

	stmt := graph.Statements[1]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}

	// Check condition is an identifier (VarRef)
	if stmt.Blocker.Condition == nil {
		t.Fatal("Condition is nil")
	}
	if stmt.Blocker.Condition.Kind != ExprVarRef {
		t.Errorf("Condition.Kind = %v, want ExprVarRef", stmt.Blocker.Condition.Kind)
	}
	if stmt.Blocker.Condition.VarName != "isReady" {
		t.Errorf("Condition.VarName = %q, want %q", stmt.Blocker.Condition.VarName, "isReady")
	}

	// Check then branch
	if len(stmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(ThenBranch) = %d, want 1", len(stmt.Blocker.ThenBranch))
	}
}

func TestBuildIR_ElseIf(t *testing.T) {
	graph := buildIR(t, `var X = 2
if @var.X == 1 { 
    echo "one" 
} else if @var.X == 2 { 
    echo "two" 
} else { 
    echo "other" 
}`)

	if len(graph.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want 2", len(graph.Statements))
	}

	stmt := graph.Statements[1]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}

	// Check first if
	if stmt.Blocker.Condition == nil {
		t.Fatal("Condition is nil")
	}
	if len(stmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(ThenBranch) = %d, want 1", len(stmt.Blocker.ThenBranch))
	}

	// Check else-if (should be nested if in else branch)
	if len(stmt.Blocker.ElseBranch) != 1 {
		t.Fatalf("len(ElseBranch) = %d, want 1", len(stmt.Blocker.ElseBranch))
	}

	elseIfStmt := stmt.Blocker.ElseBranch[0]
	if elseIfStmt.Kind != StmtBlocker {
		t.Errorf("elseIfStmt.Kind = %v, want StmtBlocker", elseIfStmt.Kind)
	}
	if elseIfStmt.Blocker == nil {
		t.Fatal("elseIfStmt.Blocker is nil")
	}

	// Check else-if condition
	if elseIfStmt.Blocker.Condition == nil {
		t.Fatal("elseIfStmt.Condition is nil")
	}
	if elseIfStmt.Blocker.Condition.Kind != ExprBinaryOp {
		t.Errorf("elseIfStmt.Condition.Kind = %v, want ExprBinaryOp", elseIfStmt.Blocker.Condition.Kind)
	}

	// Check else-if then branch
	if len(elseIfStmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(elseIfStmt.ThenBranch) = %d, want 1", len(elseIfStmt.Blocker.ThenBranch))
	}

	// Check final else branch
	if len(elseIfStmt.Blocker.ElseBranch) != 1 {
		t.Errorf("len(elseIfStmt.ElseBranch) = %d, want 1", len(elseIfStmt.Blocker.ElseBranch))
	}
}
