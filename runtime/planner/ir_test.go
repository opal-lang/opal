package planner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// ========== ScopeStack Tests ==========

func TestScopeStack_NewScopeStack(t *testing.T) {
	s := NewScopeStack()
	if s.Depth() != 1 {
		t.Errorf("Depth() = %d, want 1", s.Depth())
	}
}

func TestScopeStack_Define_Lookup(t *testing.T) {
	s := NewScopeStack()
	s.Define("X", "expr-1")

	exprID, ok := s.Lookup("X")
	if !ok {
		t.Fatal("Lookup(X) returned false, want true")
	}
	if exprID != "expr-1" {
		t.Errorf("Lookup(X) = %q, want %q", exprID, "expr-1")
	}
}

func TestScopeStack_Lookup_NotFound(t *testing.T) {
	s := NewScopeStack()

	_, ok := s.Lookup("MISSING")
	if ok {
		t.Error("Lookup(MISSING) returned true, want false")
	}
}

func TestScopeStack_Push_Pop(t *testing.T) {
	s := NewScopeStack()
	s.Define("X", "expr-1")

	s.Push()
	if s.Depth() != 2 {
		t.Errorf("Depth() after Push = %d, want 2", s.Depth())
	}

	// Can still see outer scope
	exprID, ok := s.Lookup("X")
	if !ok || exprID != "expr-1" {
		t.Errorf("Lookup(X) in inner scope = %q, %v, want %q, true", exprID, ok, "expr-1")
	}

	s.Pop()
	if s.Depth() != 1 {
		t.Errorf("Depth() after Pop = %d, want 1", s.Depth())
	}
}

func TestScopeStack_Shadowing(t *testing.T) {
	s := NewScopeStack()
	s.Define("X", "expr-outer")

	s.Push()
	s.Define("X", "expr-inner")

	// Inner scope shadows outer
	exprID, ok := s.Lookup("X")
	if !ok || exprID != "expr-inner" {
		t.Errorf("Lookup(X) in inner scope = %q, %v, want %q, true", exprID, ok, "expr-inner")
	}

	s.Pop()

	// After pop, outer is visible again
	exprID, ok = s.Lookup("X")
	if !ok || exprID != "expr-outer" {
		t.Errorf("Lookup(X) after Pop = %q, %v, want %q, true", exprID, ok, "expr-outer")
	}
}

func TestScopeStack_PopRootScope(t *testing.T) {
	s := NewScopeStack()
	s.Define("X", "expr-1")

	// Pop on root scope should be a no-op
	s.Pop()
	if s.Depth() != 1 {
		t.Errorf("Depth() after Pop on root = %d, want 1", s.Depth())
	}

	// Variable should still be accessible
	exprID, ok := s.Lookup("X")
	if !ok || exprID != "expr-1" {
		t.Errorf("Lookup(X) after Pop on root = %q, %v, want %q, true", exprID, ok, "expr-1")
	}
}

// ========== StatementIR Tests ==========

func TestStatementIR_Command(t *testing.T) {
	stmt := &StatementIR{
		Kind: StmtCommand,
		Command: &CommandStmtIR{
			Decorator: "@shell",
			Command: &CommandExpr{
				Parts: []*ExprIR{
					{Kind: ExprLiteral, Value: "echo hello"},
				},
			},
		},
	}

	if stmt.Kind != StmtCommand {
		t.Errorf("Kind = %v, want StmtCommand", stmt.Kind)
	}
	if stmt.Command.Decorator != "@shell" {
		t.Errorf("Command.Decorator = %q, want %q", stmt.Command.Decorator, "@shell")
	}
}

func TestStatementIR_VarDecl(t *testing.T) {
	stmt := &StatementIR{
		Kind: StmtVarDecl,
		VarDecl: &VarDeclIR{
			Name:   "ENV",
			Value:  &ExprIR{Kind: ExprLiteral, Value: "production"},
			ExprID: "expr-123",
		},
	}

	if stmt.Kind != StmtVarDecl {
		t.Errorf("Kind = %v, want StmtVarDecl", stmt.Kind)
	}
	if stmt.VarDecl.Name != "ENV" {
		t.Errorf("VarDecl.Name = %q, want %q", stmt.VarDecl.Name, "ENV")
	}
}

func TestStatementIR_Blocker_If(t *testing.T) {
	stmt := &StatementIR{
		Kind: StmtBlocker,
		Blocker: &BlockerIR{
			Kind:  BlockerIf,
			Depth: 1,
			Condition: &ExprIR{
				Kind:  ExprBinaryOp,
				Op:    "==",
				Left:  &ExprIR{Kind: ExprVarRef, VarName: "ENV"},
				Right: &ExprIR{Kind: ExprLiteral, Value: "prod"},
			},
			ThenBranch: []*StatementIR{
				{Kind: StmtCommand, Command: &CommandStmtIR{Decorator: "@shell"}},
			},
			ElseBranch: []*StatementIR{
				{Kind: StmtCommand, Command: &CommandStmtIR{Decorator: "@shell"}},
			},
		},
	}

	if stmt.Kind != StmtBlocker {
		t.Errorf("Kind = %v, want StmtBlocker", stmt.Kind)
	}
	if stmt.Blocker.Kind != BlockerIf {
		t.Errorf("Blocker.Kind = %v, want BlockerIf", stmt.Blocker.Kind)
	}
	if len(stmt.Blocker.ThenBranch) != 1 {
		t.Errorf("len(ThenBranch) = %d, want 1", len(stmt.Blocker.ThenBranch))
	}
	if len(stmt.Blocker.ElseBranch) != 1 {
		t.Errorf("len(ElseBranch) = %d, want 1", len(stmt.Blocker.ElseBranch))
	}
}

// ========== ExecutionGraph Tests ==========

func TestExecutionGraph_ScriptMode(t *testing.T) {
	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{Kind: StmtVarDecl, VarDecl: &VarDeclIR{Name: "X", ExprID: "expr-1"}},
			{Kind: StmtCommand, Command: &CommandStmtIR{Decorator: "@shell"}},
		},
		Scopes: NewScopeStack(),
	}

	if len(graph.Statements) != 2 {
		t.Errorf("len(Statements) = %d, want 2", len(graph.Statements))
	}
	if graph.Functions != nil {
		t.Error("Functions should be nil in script mode")
	}
}

func TestExecutionGraph_CommandMode(t *testing.T) {
	graph := &ExecutionGraph{
		Functions: map[string]*FunctionIR{
			"deploy": {
				Name: "deploy",
				Body: []*StatementIR{
					{Kind: StmtCommand, Command: &CommandStmtIR{Decorator: "@shell"}},
				},
			},
		},
		Scopes: NewScopeStack(),
	}

	if graph.Functions == nil {
		t.Fatal("Functions should not be nil in command mode")
	}
	fn, ok := graph.Functions["deploy"]
	if !ok {
		t.Fatal("Functions[deploy] not found")
	}
	if fn.Name != "deploy" {
		t.Errorf("fn.Name = %q, want %q", fn.Name, "deploy")
	}
}

// ========== BlockerIR Tests ==========

func TestBlockerIR_Unresolved(t *testing.T) {
	blocker := &BlockerIR{
		Kind:      BlockerIf,
		Depth:     1,
		Condition: &ExprIR{Kind: ExprVarRef, VarName: "FLAG"},
		Taken:     nil, // Unresolved
	}

	if blocker.Taken != nil {
		t.Error("Taken should be nil for unresolved blocker")
	}
}

func TestBlockerIR_Resolved_True(t *testing.T) {
	taken := true
	blocker := &BlockerIR{
		Kind:      BlockerIf,
		Depth:     1,
		Condition: &ExprIR{Kind: ExprVarRef, VarName: "FLAG"},
		Taken:     &taken,
	}

	if blocker.Taken == nil || *blocker.Taken != true {
		t.Error("Taken should be true for resolved blocker")
	}
}

func TestBlockerIR_ForLoop(t *testing.T) {
	blocker := &BlockerIR{
		Kind:    BlockerFor,
		Depth:   1,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind: ExprDecoratorRef,
			Decorator: &DecoratorRef{
				Name:     "var",
				Selector: []string{"ITEMS"},
			},
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtCommand, Command: &CommandStmtIR{Decorator: "@shell"}},
		},
	}

	if blocker.Kind != BlockerFor {
		t.Errorf("Kind = %v, want BlockerFor", blocker.Kind)
	}
	if blocker.LoopVar != "item" {
		t.Errorf("LoopVar = %q, want %q", blocker.LoopVar, "item")
	}
}

// ========== VarDeclIR Tests ==========

func TestVarDeclIR_Literal(t *testing.T) {
	decl := &VarDeclIR{
		Name:   "ENV",
		Value:  &ExprIR{Kind: ExprLiteral, Value: "production"},
		ExprID: "expr-abc123",
	}

	if decl.Name != "ENV" {
		t.Errorf("Name = %q, want %q", decl.Name, "ENV")
	}
	if decl.Value.Kind != ExprLiteral {
		t.Errorf("Value.Kind = %v, want ExprLiteral", decl.Value.Kind)
	}
	if decl.ExprID != "expr-abc123" {
		t.Errorf("ExprID = %q, want %q", decl.ExprID, "expr-abc123")
	}
}

func TestVarDeclIR_DecoratorRef(t *testing.T) {
	decl := &VarDeclIR{
		Name: "HOME",
		Value: &ExprIR{
			Kind: ExprDecoratorRef,
			Decorator: &DecoratorRef{
				Name:     "env",
				Selector: []string{"HOME"},
			},
		},
		ExprID: "expr-def456",
	}

	if decl.Value.Kind != ExprDecoratorRef {
		t.Errorf("Value.Kind = %v, want ExprDecoratorRef", decl.Value.Kind)
	}
	if diff := cmp.Diff([]string{"HOME"}, decl.Value.Decorator.Selector); diff != "" {
		t.Errorf("Selector mismatch (-want +got):\n%s", diff)
	}
}

// ========== CommandStmtIR Tests ==========

func TestCommandStmtIR_Simple(t *testing.T) {
	cmd := &CommandStmtIR{
		Decorator: "@shell",
		Command: &CommandExpr{
			Parts: []*ExprIR{
				{Kind: ExprLiteral, Value: "echo hello"},
			},
		},
	}

	if cmd.Decorator != "@shell" {
		t.Errorf("Decorator = %q, want %q", cmd.Decorator, "@shell")
	}
	if len(cmd.Command.Parts) != 1 {
		t.Errorf("len(Command.Parts) = %d, want 1", len(cmd.Command.Parts))
	}
}

func TestCommandStmtIR_WithOperator(t *testing.T) {
	cmd := &CommandStmtIR{
		Decorator: "@shell",
		Command: &CommandExpr{
			Parts: []*ExprIR{
				{Kind: ExprLiteral, Value: "echo hello"},
			},
		},
		Operator: "&&",
	}

	if cmd.Operator != "&&" {
		t.Errorf("Operator = %q, want %q", cmd.Operator, "&&")
	}
}

func TestCommandStmtIR_DecoratorBlock(t *testing.T) {
	cmd := &CommandStmtIR{
		Decorator: "@retry",
		Args: []*ExprIR{
			{Kind: ExprLiteral, Value: 3},
		},
		Block: []*StatementIR{
			{Kind: StmtCommand, Command: &CommandStmtIR{Decorator: "@shell"}},
		},
	}

	if cmd.Decorator != "@retry" {
		t.Errorf("Decorator = %q, want %q", cmd.Decorator, "@retry")
	}
	if len(cmd.Args) != 1 {
		t.Errorf("len(Args) = %d, want 1", len(cmd.Args))
	}
	if len(cmd.Block) != 1 {
		t.Errorf("len(Block) = %d, want 1", len(cmd.Block))
	}
}
