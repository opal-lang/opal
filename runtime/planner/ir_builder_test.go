package planner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/runtime/lexer"
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

func TestBuildIR_CommandWithSpaces(t *testing.T) {
	graph := buildIR(t, `echo hello world`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Command == nil || stmt.Command.Command == nil {
		t.Fatal("Command is nil")
	}

	parts := stmt.Command.Command.Parts

	// Should have: echo, space, hello, space, world
	if len(parts) != 5 {
		t.Fatalf("len(Parts) = %d, want 5", len(parts))
	}

	// Check parts
	if parts[0].Kind != ExprLiteral || parts[0].Value != "echo" {
		t.Errorf("parts[0] = %v %q, want literal 'echo'", parts[0].Kind, parts[0].Value)
	}
	if parts[1].Kind != ExprLiteral || parts[1].Value != " " {
		t.Errorf("parts[1] = %v %q, want literal ' '", parts[1].Kind, parts[1].Value)
	}
	if parts[2].Kind != ExprLiteral || parts[2].Value != "hello" {
		t.Errorf("parts[2] = %v %q, want literal 'hello'", parts[2].Kind, parts[2].Value)
	}
	if parts[3].Kind != ExprLiteral || parts[3].Value != " " {
		t.Errorf("parts[3] = %v %q, want literal ' '", parts[3].Kind, parts[3].Value)
	}
	if parts[4].Kind != ExprLiteral || parts[4].Value != "world" {
		t.Errorf("parts[4] = %v %q, want literal 'world'", parts[4].Kind, parts[4].Value)
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
	// String value - lexer may include or exclude quotes
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

func TestBuildIR_DecoratorArg_IdentifierValue(t *testing.T) {
	graph := buildIR(t, `@env.HOME(default=HOME)`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Command == nil {
		t.Fatal("Command is nil")
	}

	args := stmt.Command.Args
	if len(args) != 1 {
		t.Fatalf("len(Args) = %d, want 1", len(args))
	}

	arg := args[0]
	if arg.Name != "default" {
		t.Errorf("Arg.Name = %q, want %q", arg.Name, "default")
	}
	if arg.Value == nil {
		t.Fatal("Arg.Value is nil")
	}
	if arg.Value.Kind != ExprLiteral {
		t.Errorf("Arg.Value.Kind = %v, want ExprLiteral", arg.Value.Kind)
	}
	if diff := cmp.Diff("HOME", arg.Value.Value); diff != "" {
		t.Errorf("Arg.Value mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildIR_DecoratorArg_PositionalIdentifier(t *testing.T) {
	graph := buildIR(t, `@env(HOME)`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Command == nil {
		t.Fatal("Command is nil")
	}

	args := stmt.Command.Args
	if len(args) != 1 {
		t.Fatalf("len(Args) = %d, want 1", len(args))
	}

	arg := args[0]
	if arg.Name != "arg1" {
		t.Errorf("Arg.Name = %q, want %q", arg.Name, "arg1")
	}
	if arg.Value == nil {
		t.Fatal("Arg.Value is nil")
	}
	if arg.Value.Kind != ExprLiteral {
		t.Errorf("Arg.Value.Kind = %v, want ExprLiteral", arg.Value.Kind)
	}
	if diff := cmp.Diff("HOME", arg.Value.Value); diff != "" {
		t.Errorf("Arg.Value mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildIR_VarDecl_DecoratorRef_MixedArgNamesPreserved(t *testing.T) {
	graph := buildIR(t, `var X = @retry(delay=2s, 3, backoff="constant")`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.VarDecl == nil || stmt.VarDecl.Value == nil || stmt.VarDecl.Value.Decorator == nil {
		t.Fatal("decorator expression is nil")
	}

	dec := stmt.VarDecl.Value.Decorator
	if diff := cmp.Diff([]string{"delay", "arg2", "backoff"}, dec.ArgNames); diff != "" {
		t.Errorf("Decorator.ArgNames mismatch (-want +got):\n%s", diff)
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

	// Check command has interpolated parts
	cmd := graph.Statements[1].Command
	if cmd == nil || cmd.Command == nil {
		t.Fatal("Command is nil")
	}

	parts := cmd.Command.Parts

	// Find var ref part
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

func TestBuildIR_CommandWithVarCallForm(t *testing.T) {
	graph := buildIR(t, `var NAME = "world"
echo @var("NAME")`)

	if len(graph.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want 2", len(graph.Statements))
	}

	cmd := graph.Statements[1].Command
	if cmd == nil || cmd.Command == nil {
		t.Fatal("Command is nil")
	}

	parts := cmd.Command.Parts
	hasVarRef := false
	for _, part := range parts {
		if part.Kind == ExprVarRef && part.VarName == "NAME" {
			hasVarRef = true
			break
		}
	}

	if !hasVarRef {
		t.Error("@var(\"NAME\") should normalize to VarRef(NAME)")
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

// ========== For Loop Tests ==========

func TestBuildIR_ForLoop(t *testing.T) {
	graph := buildIR(t, `for item in items { echo item }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}
	if stmt.Blocker == nil {
		t.Fatal("Blocker is nil")
	}
	if stmt.Blocker.Kind != BlockerFor {
		t.Errorf("Blocker.Kind = %v, want BlockerFor", stmt.Blocker.Kind)
	}

	// Check loop variable
	if stmt.Blocker.LoopVar != "item" {
		t.Errorf("LoopVar = %q, want %q", stmt.Blocker.LoopVar, "item")
	}

	// Check collection
	if stmt.Blocker.Collection == nil {
		t.Fatal("Collection is nil")
	}
	if stmt.Blocker.Collection.Kind != ExprVarRef {
		t.Errorf("Collection.Kind = %v, want ExprVarRef", stmt.Blocker.Collection.Kind)
	}
	if stmt.Blocker.Collection.VarName != "items" {
		t.Errorf("Collection.VarName = %q, want %q", stmt.Blocker.Collection.VarName, "items")
	}

	// Check body
	if len(stmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(ThenBranch) = %d, want 1", len(stmt.Blocker.ThenBranch))
	}
}

func TestBuildIR_ForLoopWithDecorator(t *testing.T) {
	graph := buildIR(t, `for item in @var.LIST { echo item }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Blocker.Collection == nil {
		t.Fatal("Collection is nil")
	}
	if stmt.Blocker.Collection.Kind != ExprVarRef {
		t.Errorf("Collection.Kind = %v, want ExprVarRef", stmt.Blocker.Collection.Kind)
	}
	if stmt.Blocker.Collection.VarName != "LIST" {
		t.Errorf("Collection.VarName = %q, want %q", stmt.Blocker.Collection.VarName, "LIST")
	}
}

// ========== Try/Catch Tests ==========

func TestBuildIR_TryCatch(t *testing.T) {
	graph := buildIR(t, `try { echo "risky" } catch { echo "error" }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtTry {
		t.Errorf("stmt.Kind = %v, want StmtTry", stmt.Kind)
	}
	if stmt.Try == nil {
		t.Fatal("Try is nil")
	}

	// Check try block
	if len(stmt.Try.TryBlock) != 1 {
		t.Errorf("len(TryBlock) = %d, want 1", len(stmt.Try.TryBlock))
	}

	// Check catch block
	if len(stmt.Try.CatchBlock) != 1 {
		t.Errorf("len(CatchBlock) = %d, want 1", len(stmt.Try.CatchBlock))
	}
}

func TestBuildIR_TryCatchFinally(t *testing.T) {
	graph := buildIR(t, `try { echo "risky" } catch { echo "error" } finally { echo "cleanup" }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Try == nil {
		t.Fatal("Try is nil")
	}

	// Check all blocks
	if len(stmt.Try.TryBlock) != 1 {
		t.Errorf("len(TryBlock) = %d, want 1", len(stmt.Try.TryBlock))
	}
	if len(stmt.Try.CatchBlock) != 1 {
		t.Errorf("len(CatchBlock) = %d, want 1", len(stmt.Try.CatchBlock))
	}
	if len(stmt.Try.FinallyBlock) != 1 {
		t.Errorf("len(FinallyBlock) = %d, want 1", len(stmt.Try.FinallyBlock))
	}
}

// ========== When Statement Tests ==========

func TestBuildIR_When(t *testing.T) {
	graph := buildIR(t, `when @var.X { "a" -> echo "got a" }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}
	if stmt.Blocker == nil {
		t.Fatal("Blocker is nil")
	}
	if stmt.Blocker.Kind != BlockerWhen {
		t.Errorf("Blocker.Kind = %v, want BlockerWhen", stmt.Blocker.Kind)
	}

	// Check condition
	if stmt.Blocker.Condition == nil {
		t.Fatal("Condition is nil")
	}
	if stmt.Blocker.Condition.Kind != ExprVarRef {
		t.Errorf("Condition.Kind = %v, want ExprVarRef", stmt.Blocker.Condition.Kind)
	}

	// Check arms
	if len(stmt.Blocker.Arms) != 1 {
		t.Fatalf("len(Arms) = %d, want 1", len(stmt.Blocker.Arms))
	}

	arm := stmt.Blocker.Arms[0]
	if arm.Pattern == nil {
		t.Fatal("Pattern is nil")
	}
	if arm.Pattern.Kind != ExprLiteral {
		t.Errorf("Pattern.Kind = %v, want ExprLiteral", arm.Pattern.Kind)
	}
	if len(arm.Body) != 1 {
		t.Errorf("len(Body) = %d, want 1", len(arm.Body))
	}
}

func TestBuildIR_WhenMultipleArms(t *testing.T) {
	graph := buildIR(t, `when @var.X { 
		"a" -> echo "got a"
		"b" -> echo "got b"
	}`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Blocker == nil {
		t.Fatal("Blocker is nil")
	}

	// Check arms
	if len(stmt.Blocker.Arms) != 2 {
		t.Fatalf("len(Arms) = %d, want 2", len(stmt.Blocker.Arms))
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

// ========== String Literal Validation Tests ==========

func TestBuildIR_MalformedStringLiterals(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantStmts int
	}{
		{
			name:      "empty string with quotes",
			source:    `echo ""`,
			wantStmts: 1,
		},
		{
			name:      "simple string",
			source:    `echo "hello"`,
			wantStmts: 1,
		},
		{
			name:      "string with spaces",
			source:    `echo "hello world"`,
			wantStmts: 1,
		},
		{
			name:      "interpolated string with var",
			source:    `var NAME = "test"; echo "hello @var.NAME"`,
			wantStmts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := buildIR(t, tt.source)

			if len(graph.Statements) != tt.wantStmts {
				t.Errorf("len(Statements) = %d, want %d", len(graph.Statements), tt.wantStmts)
			}
		})
	}
}

func TestBuildInterpolatedString_MalformedLiteral(t *testing.T) {
	builder := &irBuilder{
		events: []parser.Event{
			{Kind: parser.EventOpen, Data: uint32(parser.NodeInterpolatedString)},
			{Kind: parser.EventToken, Data: 0},
			{Kind: parser.EventClose, Data: uint32(parser.NodeInterpolatedString)},
		},
		tokens: []lexer.Token{{Type: lexer.STRING, Text: []byte("\"")}},
	}

	_, err := builder.buildInterpolatedString()
	if err == nil {
		t.Fatal("expected error for malformed interpolated string")
	}

	expected := "malformed interpolated string literal at position 0"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildIR_EmptyStringPreservesQuotes(t *testing.T) {
	graph := buildIR(t, `echo ""`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	cmd := graph.Statements[0].Command
	if cmd == nil || cmd.Command == nil {
		t.Fatal("Command is nil")
	}

	expected := `echo ""`
	if diff := cmp.Diff(expected, RenderCommand(cmd.Command, nil)); diff != "" {
		t.Errorf("RenderCommand mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildIR_ForLoopWithArrayLiteral(t *testing.T) {
	graph := buildIR(t, `for item in ["a", "b", "c"] { echo @var.item }`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtBlocker {
		t.Errorf("stmt.Kind = %v, want StmtBlocker", stmt.Kind)
	}
	if stmt.Blocker == nil {
		t.Fatal("Blocker is nil")
	}
	if stmt.Blocker.Kind != BlockerFor {
		t.Errorf("Blocker.Kind = %v, want BlockerFor", stmt.Blocker.Kind)
	}

	// Check collection is a literal array
	if stmt.Blocker.Collection == nil {
		t.Fatal("Collection is nil")
	}
	if stmt.Blocker.Collection.Kind != ExprLiteral {
		t.Errorf("Collection.Kind = %v, want ExprLiteral", stmt.Blocker.Collection.Kind)
	}

	// Check the array value - elements are stored as []*ExprIR
	elements, ok := stmt.Blocker.Collection.Value.([]*ExprIR)
	if !ok {
		t.Fatalf("Collection.Value = %T, want []*ExprIR", stmt.Blocker.Collection.Value)
	}
	if len(elements) != 3 {
		t.Errorf("len(elements) = %d, want 3", len(elements))
	}

	// Verify elements are proper expressions
	for i, elem := range elements {
		if elem.Kind != ExprLiteral {
			t.Errorf("element %d Kind = %v, want ExprLiteral", i, elem.Kind)
		}
	}
}

func TestBuildIR_VarDeclWithArrayLiteral(t *testing.T) {
	graph := buildIR(t, `var items = ["web1", "web2"]`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtVarDecl {
		t.Errorf("stmt.Kind = %v, want StmtVarDecl", stmt.Kind)
	}
	if stmt.VarDecl == nil {
		t.Fatal("VarDecl is nil")
	}

	// Check value is a literal array
	if stmt.VarDecl.Value == nil {
		t.Fatal("VarDecl.Value is nil")
	}
	if stmt.VarDecl.Value.Kind != ExprLiteral {
		t.Errorf("VarDecl.Value.Kind = %v, want ExprLiteral", stmt.VarDecl.Value.Kind)
	}

	// Check the array value - elements are stored as []*ExprIR
	elements, ok := stmt.VarDecl.Value.Value.([]*ExprIR)
	if !ok {
		t.Fatalf("VarDecl.Value.Value = %T, want []*ExprIR", stmt.VarDecl.Value.Value)
	}
	if len(elements) != 2 {
		t.Errorf("len(elements) = %d, want 2", len(elements))
	}

	// Verify elements are proper expressions
	for i, elem := range elements {
		if elem.Kind != ExprLiteral {
			t.Errorf("element %d Kind = %v, want ExprLiteral", i, elem.Kind)
		}
	}
}

func TestBuildIR_ArrayLiteralWithObjectElements(t *testing.T) {
	graph := buildIR(t, `var items = [{name: "a"}, {name: "b"}]`)

	if len(graph.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(graph.Statements))
	}

	stmt := graph.Statements[0]
	if stmt.Kind != StmtVarDecl {
		t.Errorf("stmt.Kind = %v, want StmtVarDecl", stmt.Kind)
	}

	// Check the array value - elements are stored as []*ExprIR
	elements, ok := stmt.VarDecl.Value.Value.([]*ExprIR)
	if !ok {
		t.Fatalf("VarDecl.Value.Value = %T, want []*ExprIR", stmt.VarDecl.Value.Value)
	}
	if len(elements) != 2 {
		t.Errorf("len(elements) = %d, want 2", len(elements))
	}

	// Check first element is an *ExprIR (object literal expression)
	if elements[0].Kind != ExprLiteral {
		t.Errorf("elements[0].Kind = %v, want ExprLiteral", elements[0].Kind)
	}

	// Check the object value inside the expression - fields are map[string]*ExprIR
	objFields, ok := elements[0].Value.(map[string]*ExprIR)
	if !ok {
		t.Fatalf("elements[0].Value = %T, want map[string]*ExprIR", elements[0].Value)
	}

	// Check the "name" field
	nameField, ok := objFields["name"]
	if !ok {
		t.Fatal("object missing 'name' field")
	}
	if nameField.Kind != ExprLiteral || nameField.Value != "a" {
		t.Errorf("name field = %v, want literal 'a'", nameField.Value)
	}
}

func TestTokenToValue_StringQuoteStripping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"double quotes", `"hello"`, "hello"},
		{"single quotes", `'world'`, "world"},
		{"minimal double quote", `"a"`, "a"},
		{"minimal single quote", `'b'`, "b"},
		{"no quotes matching chars", `aba`, "aba"},   // Should NOT strip - not quotes
		{"no quotes matching chars 2", `xyx`, "xyx"}, // Should NOT strip - not quotes
		{"single char no quote", `x`, "x"},
		{"empty", ``, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := lexer.Token{
				Type: lexer.STRING,
				Text: []byte(tt.input),
			}
			result := tokenToValue(tok)
			if result != tt.expected {
				t.Errorf("tokenToValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeepCopyStatements_PreservesRedirectMetadata(t *testing.T) {
	original := []*StatementIR{
		{
			Kind: StmtCommand,
			Command: &CommandStmtIR{
				Decorator:    "@shell",
				Operator:     "&&",
				RedirectMode: ">>",
				Command: &CommandExpr{Parts: []*ExprIR{
					{Kind: ExprLiteral, Value: "echo hello"},
				}},
				RedirectTarget: &CommandExpr{Parts: []*ExprIR{
					{Kind: ExprLiteral, Value: "output.log"},
				}},
			},
		},
	}

	copied := DeepCopyStatements(original)
	if len(copied) != 1 {
		t.Fatalf("len(copied) = %d, want 1", len(copied))
	}

	copyCmd := copied[0].Command
	if copyCmd == nil {
		t.Fatal("copied command is nil")
	}

	if diff := cmp.Diff(">>", copyCmd.RedirectMode); diff != "" {
		t.Errorf("RedirectMode mismatch (-want +got):\n%s", diff)
	}

	if copyCmd.RedirectTarget == nil || len(copyCmd.RedirectTarget.Parts) != 1 {
		t.Fatalf("RedirectTarget missing in copied statement: %#v", copyCmd.RedirectTarget)
	}

	if diff := cmp.Diff("output.log", copyCmd.RedirectTarget.Parts[0].Value); diff != "" {
		t.Errorf("RedirectTarget value mismatch (-want +got):\n%s", diff)
	}

	if copyCmd == original[0].Command {
		t.Fatal("copied command shares pointer with original")
	}
	if copyCmd.RedirectTarget == original[0].Command.RedirectTarget {
		t.Fatal("copied redirect target shares pointer with original")
	}
}

func TestDeepCopyStatements_RedirectTargetMutationIsolation(t *testing.T) {
	original := []*StatementIR{
		{
			Kind: StmtCommand,
			Command: &CommandStmtIR{
				Decorator:    "@shell",
				RedirectMode: ">",
				Command: &CommandExpr{Parts: []*ExprIR{
					{Kind: ExprLiteral, Value: "echo hello"},
				}},
				RedirectTarget: &CommandExpr{Parts: []*ExprIR{
					{Kind: ExprLiteral, Value: "out.txt"},
				}},
			},
		},
	}

	copyStmts := DeepCopyStatements(original)
	copyStmts[0].Command.RedirectTarget.Parts[0].Value = "changed.txt"

	if diff := cmp.Diff("out.txt", original[0].Command.RedirectTarget.Parts[0].Value); diff != "" {
		t.Errorf("original redirect target mutated by copy (-want +got):\n%s", diff)
	}
}
