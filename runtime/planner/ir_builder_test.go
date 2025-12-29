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
