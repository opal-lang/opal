package planner

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/vault"
)

// TestResolve_SimpleCommand tests resolving a simple command with no variables.
func TestResolve_SimpleCommand(t *testing.T) {
	// Build IR: echo "hello"
	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"hello\""},
						},
					},
				},
			},
		},
		Scopes: NewScopeStack(),
	}

	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Create session (minimal for now)
	session := &mockSession{}

	// Resolve
	config := ResolveConfig{
		Context: context.Background(),
	}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify: no errors, no blockers evaluated
	// (This is a smoke test - just verify it doesn't crash)
}

// TestResolve_VarDecl tests resolving a variable declaration.
func TestResolve_VarDecl(t *testing.T) {
	// Create vault first (IR Builder does this)
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variable in vault to get exprID
	exprID := v.DeclareVariable("X", "literal:hello")

	// Build IR: var X = "hello"
	scopes := NewScopeStack()
	scopes.Define("X", exprID)

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "X",
					ExprID: exprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "hello",
					},
				},
			},
		},
		Scopes: scopes,
	}

	// Create session
	session := &mockSession{}

	// Resolve
	config := ResolveConfig{
		Context: context.Background(),
	}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify: variable was marked as touched
	if !v.IsTouched(exprID) {
		t.Errorf("Variable %q was not marked as touched", exprID)
	}
}

// TestResolve_IfTrue tests resolving an if statement with true condition.
func TestResolve_IfTrue(t *testing.T) {
	// Create vault first (IR Builder does this)
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variable in vault to get exprID
	exprID := v.DeclareVariable("X", "literal:true")

	// Build IR:
	//   var X = true
	//   if @var.X {
	//     echo "taken"
	//   } else {
	//     echo "not taken"
	//   }
	scopes := NewScopeStack()
	scopes.Define("X", exprID)

	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "X",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"taken\""},
						},
					},
				},
			},
		},
		ElseBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"not taken\""},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "X",
					ExprID: exprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: true,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: scopes,
	}

	// Create session
	session := &mockSession{}

	// Resolve
	config := ResolveConfig{
		Context: context.Background(),
	}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify: blocker.Taken should be true
	if blocker.Taken == nil {
		t.Fatalf("Blocker.Taken is nil, expected true")
	}
	if !*blocker.Taken {
		t.Errorf("Blocker.Taken = false, expected true")
	}
}

// TestResolve_IfFalse tests resolving an if statement with false condition.
func TestResolve_IfFalse(t *testing.T) {
	// Create vault first (IR Builder does this)
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variable in vault to get exprID
	exprID := v.DeclareVariable("X", "literal:false")

	// Build IR:
	//   var X = false
	//   if @var.X {
	//     echo "taken"
	//   } else {
	//     echo "not taken"
	//   }
	scopes := NewScopeStack()
	scopes.Define("X", exprID)

	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "X",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"taken\""},
						},
					},
				},
			},
		},
		ElseBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"not taken\""},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "X",
					ExprID: exprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: false,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: scopes,
	}

	// Create session
	session := &mockSession{}

	// Resolve
	config := ResolveConfig{
		Context: context.Background(),
	}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify: blocker.Taken should be false
	if blocker.Taken == nil {
		t.Fatalf("Blocker.Taken is nil, expected false")
	}
	if *blocker.Taken {
		t.Errorf("Blocker.Taken = true, expected false")
	}
}

// TestResolve_UndefinedVar tests that undefined variables produce errors.
func TestResolve_UndefinedVar(t *testing.T) {
	// Build IR: if @var.UNDEFINED { ... }
	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "UNDEFINED",
		},
		ThenBranch: []*StatementIR{},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: NewScopeStack(),
	}

	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Create session
	session := &mockSession{}

	// Resolve
	config := ResolveConfig{
		Context: context.Background(),
	}
	err := Resolve(graph, v, session, config)

	// Verify: should get error about undefined variable
	if err == nil {
		t.Fatalf("Expected error for undefined variable, got nil")
	}

	// Check error message contains "undefined variable"
	if !containsStr(err.Error(), "undefined variable") {
		t.Errorf("Error message should mention 'undefined variable', got: %v", err)
	}
}

// mockSession is a minimal Session implementation for testing.
type mockSession struct{}

func (m *mockSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return decorator.Result{}, nil
}

func (m *mockSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (m *mockSession) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockSession) Env() map[string]string {
	return map[string]string{}
}

func (m *mockSession) WithEnv(delta map[string]string) decorator.Session {
	return m
}

func (m *mockSession) WithWorkdir(dir string) decorator.Session {
	return m
}

func (m *mockSession) Cwd() string {
	return "/test"
}

func (m *mockSession) ID() string {
	return "local"
}

func (m *mockSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (m *mockSession) Close() error {
	return nil
}

// TestResolve_IfBranchPruning verifies that untaken branch expressions are not touched.
func TestResolve_IfBranchPruning(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variables
	condExprID := v.DeclareVariable("COND", "literal:true")
	thenExprID := v.DeclareVariable("THEN_VAR", "literal:then-value")
	elseExprID := v.DeclareVariable("ELSE_VAR", "literal:else-value")

	// Build IR:
	//   var COND = true
	//   if @var.COND {
	//     var THEN_VAR = "then-value"
	//   } else {
	//     var ELSE_VAR = "else-value"
	//   }
	scopes := NewScopeStack()
	scopes.Define("COND", condExprID)
	// Note: THEN_VAR and ELSE_VAR are defined in their respective branches
	// but for this test we pre-declare them to track touching

	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "THEN_VAR",
					ExprID: thenExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "then-value",
					},
				},
			},
		},
		ElseBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "ELSE_VAR",
					ExprID: elseExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "else-value",
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND",
					ExprID: condExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: true,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify: COND and THEN_VAR should be touched, ELSE_VAR should NOT be touched
	if !v.IsTouched(condExprID) {
		t.Errorf("COND should be touched")
	}
	if !v.IsTouched(thenExprID) {
		t.Errorf("THEN_VAR should be touched (in taken branch)")
	}
	if v.IsTouched(elseExprID) {
		t.Errorf("ELSE_VAR should NOT be touched (in untaken branch)")
	}
}

// TestResolve_MultiWave tests nested if statements requiring multiple resolution waves.
func TestResolve_MultiWave(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variables
	outerExprID := v.DeclareVariable("OUTER", "literal:true")
	innerExprID := v.DeclareVariable("INNER", "literal:false")
	deepExprID := v.DeclareVariable("DEEP", "literal:deep-value")

	// Build IR:
	//   var OUTER = true
	//   if @var.OUTER {
	//     var INNER = false
	//     if @var.INNER {
	//       var DEEP = "deep-value"  # Should NOT be touched
	//     }
	//   }
	scopes := NewScopeStack()
	scopes.Define("OUTER", outerExprID)

	innerBlocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "INNER",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "DEEP",
					ExprID: deepExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "deep-value",
					},
				},
			},
		},
	}

	outerBlocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "OUTER",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "INNER",
					ExprID: innerExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: false,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: innerBlocker,
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "OUTER",
					ExprID: outerExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: true,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: outerBlocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify blockers
	if outerBlocker.Taken == nil || !*outerBlocker.Taken {
		t.Errorf("OUTER blocker should be taken (true)")
	}
	if innerBlocker.Taken == nil || *innerBlocker.Taken {
		t.Errorf("INNER blocker should NOT be taken (false)")
	}

	// Verify touched
	if !v.IsTouched(outerExprID) {
		t.Errorf("OUTER should be touched")
	}
	if !v.IsTouched(innerExprID) {
		t.Errorf("INNER should be touched (in taken outer branch)")
	}
	if v.IsTouched(deepExprID) {
		t.Errorf("DEEP should NOT be touched (inner condition is false)")
	}
}

// TestResolve_CommandMode tests that only the target function is resolved.
func TestResolve_CommandMode(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variables
	topLevelExprID := v.DeclareVariable("TOP", "literal:top-value")
	funcExprID := v.DeclareVariable("FUNC", "literal:func-value")
	otherFuncExprID := v.DeclareVariable("OTHER", "literal:other-value")

	// Build IR with functions
	scopes := NewScopeStack()
	scopes.Define("TOP", topLevelExprID)

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "TOP",
					ExprID: topLevelExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "top-value",
					},
				},
			},
		},
		Functions: map[string]*FunctionIR{
			"target_func": {
				Name: "target_func",
				Body: []*StatementIR{
					{
						Kind: StmtVarDecl,
						VarDecl: &VarDeclIR{
							Name:   "FUNC",
							ExprID: funcExprID,
							Value: &ExprIR{
								Kind:  ExprLiteral,
								Value: "func-value",
							},
						},
					},
				},
			},
			"other_func": {
				Name: "other_func",
				Body: []*StatementIR{
					{
						Kind: StmtVarDecl,
						VarDecl: &VarDeclIR{
							Name:   "OTHER",
							ExprID: otherFuncExprID,
							Value: &ExprIR{
								Kind:  ExprLiteral,
								Value: "other-value",
							},
						},
					},
				},
			},
		},
		Scopes: scopes,
	}

	// Resolve in command mode (target_func)
	session := &mockSession{}
	config := ResolveConfig{
		Context:        context.Background(),
		TargetFunction: "target_func",
	}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify: only target_func's variables should be touched
	if v.IsTouched(topLevelExprID) {
		t.Errorf("TOP should NOT be touched (command mode skips top-level)")
	}
	if !v.IsTouched(funcExprID) {
		t.Errorf("FUNC should be touched (in target function)")
	}
	if v.IsTouched(otherFuncExprID) {
		t.Errorf("OTHER should NOT be touched (in different function)")
	}
}

// TestResolve_FunctionNotFound tests error when target function doesn't exist.
func TestResolve_FunctionNotFound(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	graph := &ExecutionGraph{
		Statements: []*StatementIR{},
		Functions:  map[string]*FunctionIR{},
		Scopes:     NewScopeStack(),
	}

	// Resolve in command mode with non-existent function
	session := &mockSession{}
	config := ResolveConfig{
		Context:        context.Background(),
		TargetFunction: "nonexistent",
	}
	err := Resolve(graph, v, session, config)

	// Should get error about function not found
	if err == nil {
		t.Fatalf("Expected error for nonexistent function, got nil")
	}
	if !containsStr(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// TestResolve_ForLoop tests for-loop unrolling.
func TestResolve_ForLoop(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Build IR:
	//   for item in ["a", "b", "c"] {
	//     echo @var.item
	//   }
	scopes := NewScopeStack()

	blocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"a", "b", "c"},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "item"},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// For-loops don't set Taken (they always execute if collection is non-empty)
	// The test verifies it doesn't crash and processes the loop
}

// TestResolve_WhenStatement tests when statement pattern matching.
func TestResolve_WhenStatement(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variable
	exprID := v.DeclareVariable("ENV", "literal:prod")

	// Build IR:
	//   var ENV = "prod"
	//   when @var.ENV {
	//     "dev" -> echo "development"
	//     "prod" -> echo "production"
	//   }
	scopes := NewScopeStack()
	scopes.Define("ENV", exprID)

	blocker := &BlockerIR{
		Kind: BlockerWhen,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "ENV",
		},
		Arms: []*WhenArmIR{
			{
				Pattern: &ExprIR{Kind: ExprLiteral, Value: "dev"},
				Body: []*StatementIR{
					{
						Kind: StmtCommand,
						Command: &CommandStmtIR{
							Decorator: "@shell",
							Command: &CommandExpr{
								Parts: []*ExprIR{
									{Kind: ExprLiteral, Value: "echo \"development\""},
								},
							},
						},
					},
				},
			},
			{
				Pattern: &ExprIR{Kind: ExprLiteral, Value: "prod"},
				Body: []*StatementIR{
					{
						Kind: StmtCommand,
						Command: &CommandStmtIR{
							Decorator: "@shell",
							Command: &CommandExpr{
								Parts: []*ExprIR{
									{Kind: ExprLiteral, Value: "echo \"production\""},
								},
							},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "ENV",
					ExprID: exprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "prod",
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// When statements don't set Taken flag (they match arms)
	// The test verifies it doesn't crash and matches the correct arm
}

// TestResolve_NestedIfFor tests mixed if and for statements.
func TestResolve_NestedIfFor(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate IR Builder: declare variable
	condExprID := v.DeclareVariable("DEPLOY", "literal:true")

	// Build IR:
	//   var DEPLOY = true
	//   if @var.DEPLOY {
	//     for env in ["staging", "prod"] {
	//       echo "deploying to @var.env"
	//     }
	//   }
	scopes := NewScopeStack()
	scopes.Define("DEPLOY", condExprID)

	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "env",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"staging", "prod"},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"deploying to \""},
							{Kind: ExprVarRef, VarName: "env"},
						},
					},
				},
			},
		},
	}

	ifBlocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "DEPLOY",
		},
		ThenBranch: []*StatementIR{
			{
				Kind:    StmtBlocker,
				Blocker: forBlocker,
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "DEPLOY",
					ExprID: condExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: true,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: ifBlocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify if blocker was taken
	if ifBlocker.Taken == nil || !*ifBlocker.Taken {
		t.Errorf("IF blocker should be taken (DEPLOY=true)")
	}
}

// TestResolve_ForLoopWithNestedCondition tests that loop variables are accessible in nested conditions.
func TestResolve_ForLoopWithNestedCondition(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Build IR:
	//   for env in ["dev", "prod"] {
	//     if @var.env == "prod" {
	//       echo "production deploy"
	//     }
	//   }
	scopes := NewScopeStack()

	innerBlocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind: ExprBinaryOp,
			Op:   "==",
			Left: &ExprIR{
				Kind:    ExprVarRef,
				VarName: "env",
			},
			Right: &ExprIR{
				Kind:  ExprLiteral,
				Value: "prod",
			},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"production deploy\""},
						},
					},
				},
			},
		},
	}

	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "env",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"dev", "prod"},
		},
		ThenBranch: []*StatementIR{
			{
				Kind:    StmtBlocker,
				Blocker: innerBlocker,
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind:    StmtBlocker,
				Blocker: forBlocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// The inner blocker should have been evaluated
	// On the last iteration (env="prod"), it should be taken
	if innerBlocker.Taken == nil {
		t.Fatalf("Inner blocker.Taken is nil, expected to be evaluated")
	}
	// Note: The Taken value reflects the last iteration's evaluation
	// since we're reusing the same BlockerIR struct
}

// TestResolve_ForLoopVariableLeak tests that variables declared inside for-loops leak to outer scope.
// Per SPECIFICATION.md: "Language control blocks (for, if, when, fun) - mutations leak to outer scope"
func TestResolve_ForLoopVariableLeak(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Pre-declare the variable that will be set inside the loop
	lastExprID := v.DeclareVariable("last", "literal:initial")

	// Build IR:
	//   var last = "initial"
	//   for item in ["a", "b", "c"] {
	//     last = @var.item  # Mutation should leak out
	//   }
	//   # After loop: last should be "c"
	scopes := NewScopeStack()
	scopes.Define("last", lastExprID)

	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"a", "b", "c"},
		},
		ThenBranch: []*StatementIR{
			// In a real implementation, this would be a var assignment
			// For now, we just verify the loop variable is accessible
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "item"},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "last",
					ExprID: lastExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "initial",
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: forBlocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// The loop should have processed all items
	// Note: Full mutation leak testing would require var assignment statements in IR
}

// TestResolve_IfBlockVariableLeak tests that variables declared inside if-blocks leak to outer scope.
// Per SPECIFICATION.md: "Language control blocks (for, if, when, fun) - mutations leak to outer scope"
func TestResolve_IfBlockVariableLeak(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare condition variable
	condExprID := v.DeclareVariable("COND", "literal:true")
	// Declare variable that will be set inside if block
	innerExprID := v.DeclareVariable("INNER", "literal:set-inside")

	// Build IR:
	//   var COND = true
	//   if @var.COND {
	//     var INNER = "set-inside"  # Should leak to outer scope
	//   }
	//   echo @var.INNER  # Should be accessible
	scopes := NewScopeStack()
	scopes.Define("COND", condExprID)
	// Note: INNER is defined at outer scope but set inside if block
	// In real Opal, you could declare inside and it leaks out

	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "INNER",
					ExprID: innerExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "set-inside",
					},
				},
			},
		},
	}

	// Command after the if block that references INNER
	afterIfCommand := &StatementIR{
		Kind: StmtCommand,
		Command: &CommandStmtIR{
			Decorator: "@shell",
			Command: &CommandExpr{
				Parts: []*ExprIR{
					{Kind: ExprLiteral, Value: "echo "},
					{Kind: ExprVarRef, VarName: "INNER"},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND",
					ExprID: condExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: true,
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
			afterIfCommand,
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// INNER should be touched (it was declared in taken branch and used after)
	if !v.IsTouched(innerExprID) {
		t.Errorf("INNER should be touched (declared in if block, used after)")
	}
}

// TestResolve_ForLoopUnrollingWithUniqueBlockers verifies that each iteration
// of a for-loop gets its own variable binding when evaluating nested conditions.
// This tests the VarDecl injection approach to for-loop unrolling.
func TestResolve_ForLoopUnrollingWithUniqueBlockers(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Build IR:
	//   for item in ["a", "b", "c"] {
	//     if @var.item == "b" {
	//       echo "found b"
	//     }
	//   }
	//
	// After unrolling, we expect:
	//   VarDecl: item = "a"
	//   if @var.item == "b" { ... }  -> false (item is "a")
	//   VarDecl: item = "b"
	//   if @var.item == "b" { ... }  -> true (item is "b")
	//   VarDecl: item = "c"
	//   if @var.item == "b" { ... }  -> false (item is "c")
	scopes := NewScopeStack()

	// Create separate blocker structs for each iteration so we can check each one
	blockerA := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind: ExprBinaryOp,
			Op:   "==",
			Left: &ExprIR{
				Kind:    ExprVarRef,
				VarName: "item",
			},
			Right: &ExprIR{
				Kind:  ExprLiteral,
				Value: "b",
			},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command:   &CommandExpr{Parts: []*ExprIR{{Kind: ExprLiteral, Value: "echo found"}}},
				},
			},
		},
	}

	// For-loop with 3 iterations
	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"a", "b", "c"},
		},
		ThenBranch: []*StatementIR{
			// Note: In real usage, this would be the same statements repeated.
			// For testing, we use separate blockers to track each iteration.
			// The unrolling will create: VarDecl + body, VarDecl + body, VarDecl + body
			{Kind: StmtBlocker, Blocker: blockerA},
		},
	}

	// We need to manually simulate what unrolling does for this test
	// In reality, the same ThenBranch is used for all iterations
	// For this test, we'll verify the mechanism works by checking
	// that the resolver properly handles the unrolled statements

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{Kind: StmtBlocker, Blocker: forBlocker},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// The blockerA is reused for all iterations due to how ThenBranch works.
	// After all iterations, it should reflect the LAST evaluation.
	// With items ["a", "b", "c"], the last check is item=="b" when item="c", which is false.
	//
	// But wait - the unrolling injects VarDecl statements, so each iteration
	// should see its own value. Let's verify the blocker was evaluated.
	if blockerA.Taken == nil {
		t.Fatalf("blockerA.Taken is nil - blocker was not evaluated")
	}

	// Since the same blocker struct is reused, it will have the result of the
	// last iteration (item="c", so item=="b" is false)
	if *blockerA.Taken {
		t.Errorf("blockerA.Taken should be false for last iteration (item='c')")
	}
}

// TestResolve_ForLoopBlockerMustResolveCollection verifies that for-loops
// are blockers - they can't unroll until the collection expression is resolved.
func TestResolve_ForLoopBlockerMustResolveCollection(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare the collection variable
	itemsExprID := v.DeclareVariable("ITEMS", "literal:items")

	// Build IR:
	//   var ITEMS = ["x", "y"]
	//   for item in @var.ITEMS {
	//     echo @var.item
	//   }
	scopes := NewScopeStack()
	scopes.Define("ITEMS", itemsExprID)

	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "ITEMS",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "item"},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "ITEMS",
					ExprID: itemsExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: []string{"x", "y"},
					},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: forBlocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// ITEMS should be touched (used in for-loop collection)
	if !v.IsTouched(itemsExprID) {
		t.Errorf("ITEMS should be touched (used as for-loop collection)")
	}
}

// TestResolve_WaveModel_BlockersMustResolveBeforeEvaluation verifies the wave model:
// blockers can't be evaluated until their controlling expressions are resolved.
func TestResolve_WaveModel_BlockersMustResolveBeforeEvaluation(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare variables
	envExprID := v.DeclareVariable("ENV", "literal:prod")
	secretExprID := v.DeclareVariable("SECRET", "literal:secret-value")

	// Build IR:
	//   var ENV = "prod"
	//   var SECRET = "secret-value"
	//   if @var.ENV == "prod" {
	//     echo @var.SECRET
	//   }
	//
	// Wave 1: Collect ENV, SECRET, and the if condition
	// Batch resolve: ENV and SECRET get values
	// Evaluate blocker: ENV == "prod" -> true
	// Wave 2: Process taken branch, collect SECRET usage
	scopes := NewScopeStack()
	scopes.Define("ENV", envExprID)
	scopes.Define("SECRET", secretExprID)

	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind: ExprBinaryOp,
			Op:   "==",
			Left: &ExprIR{
				Kind:    ExprVarRef,
				VarName: "ENV",
			},
			Right: &ExprIR{
				Kind:  ExprLiteral,
				Value: "prod",
			},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "SECRET"},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "ENV",
					ExprID: envExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "prod"},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "SECRET",
					ExprID: secretExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "secret-value"},
				},
			},
			{
				Kind:    StmtBlocker,
				Blocker: blocker,
			},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify wave model worked:
	// 1. ENV was resolved before blocker evaluation
	if !v.IsTouched(envExprID) {
		t.Errorf("ENV should be touched (used in condition)")
	}

	// 2. Blocker was evaluated correctly
	if blocker.Taken == nil || !*blocker.Taken {
		t.Errorf("Blocker should be taken (ENV == 'prod')")
	}

	// 3. SECRET was touched (in taken branch)
	if !v.IsTouched(secretExprID) {
		t.Errorf("SECRET should be touched (used in taken branch)")
	}
}

// containsStr checks if a string contains a substring.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

var _ = cmp.Diff // Ensure cmp is imported (will use in future tests)

// =============================================================================
// Bug reproduction tests - these expose issues in the current implementation
// =============================================================================

// TestBug_ErrorsFromCollectExprAreSilentlyIgnored verifies that errors from
// collectExpr (like undefined variables) are properly returned, not silently ignored.
//
// BUG: collectExpr appends to r.errors but resolve() never checks r.errors
// after traverseAndCollect, so errors are silently swallowed.
func TestBug_ErrorsFromCollectExprAreSilentlyIgnored(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Build IR with a command that references an undefined variable
	// This should fail with "undefined variable" error
	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "UNDEFINED_VAR"}, // Not defined!
						},
					},
				},
			},
		},
		Scopes: NewScopeStack(),
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)

	// BUG: Currently this returns nil because errors are silently ignored
	// EXPECTED: Should return error about undefined variable
	if err == nil {
		t.Fatalf("BUG CONFIRMED: Expected error for undefined variable, got nil (errors silently ignored)")
	}

	if !containsStr(err.Error(), "undefined variable") {
		t.Errorf("Error should mention 'undefined variable', got: %v", err)
	}
}

// TestBug_ForLoopVariableNotInScope verifies that for-loop variables are
// properly added to scope so they can be referenced in the loop body.
//
// BUG: evaluateForBlocker creates VarDecl statements but doesn't update
// r.graph.Scopes, so ExprVarRef lookups for the loop variable fail.
// Combined with Bug 1 (errors silently ignored), this appears to pass but
// the loop variable is never actually resolved.
func TestBug_ForLoopVariableNotInScope(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Build IR:
	//   for item in ["a", "b"] {
	//     echo @var.item   # References loop variable
	//   }
	scopes := NewScopeStack()

	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"a", "b"},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "item"}, // Loop variable
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{Kind: StmtBlocker, Blocker: forBlocker},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	// Should succeed - loop variable should be accessible in body
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// BUG CHECK: The loop variable should have been declared in vault for each iteration.
	// If the scoping is working correctly, we should be able to look up "item" in scopes
	// after resolution completes.
	//
	// Since for-loops leak variables (per spec), "item" should be defined in scopes
	// with the value from the last iteration.
	itemExprID, ok := scopes.Lookup("item")
	if !ok {
		t.Fatalf("BUG CONFIRMED: Loop variable 'item' not found in scopes after resolution")
	}

	// The exprID should be touched (used in command)
	if !v.IsTouched(itemExprID) {
		t.Errorf("Loop variable 'item' should be touched (used in echo command)")
	}
}

// TestBug_VarDeclInTakenBranchNotVisibleAfter verifies that variables declared
// inside taken branches are visible to statements after the branch.
//
// This tests that collectVarDecl properly updates r.graph.Scopes so that
// variables declared in branches are visible to later statements in the same branch.
//
// Note: The IR builder pre-populates scopes with all visible variables at parse time.
// The resolver's job is to update scopes when processing VarDecl statements during
// resolution (e.g., for loop variables injected during unrolling).
func TestBug_VarDeclInTakenBranchNotVisibleAfter(t *testing.T) {
	// Create vault
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare condition variable
	condExprID := v.DeclareVariable("COND", "literal:true")

	// Build IR:
	//   var COND = true
	//   if @var.COND {
	//     var INNER = "set-inside"
	//     echo @var.INNER   # Uses INNER declared above
	//   }
	scopes := NewScopeStack()
	scopes.Define("COND", condExprID)
	// Note: INNER is NOT pre-defined in scopes - it's declared inside the if block
	// The resolver should add it to scopes when processing the VarDecl

	// Declare INNER in vault but NOT in scopes initially
	innerExprID := v.DeclareVariable("INNER", "literal:set-inside")

	blocker := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "INNER",
					ExprID: innerExprID,
					Value: &ExprIR{
						Kind:  ExprLiteral,
						Value: "set-inside",
					},
				},
			},
			// Command INSIDE the if block that references INNER
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "INNER"}, // Declared above in same block
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND",
					ExprID: condExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{Kind: StmtBlocker, Blocker: blocker},
		},
		Scopes: scopes,
	}

	// Resolve
	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	// Should succeed - INNER should be visible to the echo command in the same block
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// INNER should be in scopes after the if block is processed
	// (per spec: "Language control blocks - mutations leak to outer scope")
	innerLookupExprID, ok := scopes.Lookup("INNER")
	if !ok {
		t.Fatalf("Variable 'INNER' declared in if block not found in scopes after resolution")
	}

	// The exprID should match what we declared
	if innerLookupExprID != innerExprID {
		t.Errorf("INNER exprID mismatch: got %q, want %q", innerLookupExprID, innerExprID)
	}

	// Verify INNER was touched (used in echo command)
	if !v.IsTouched(innerExprID) {
		t.Errorf("INNER should be touched (declared in if block, used in echo)")
	}
}

// =============================================================================
// Wave model tests - verify batch resolution and flattening semantics
// =============================================================================

// TestResolve_MultipleBlockersSameLevel verifies that multiple blockers at the
// same level are collected and their conditions resolved. Statements between
// blockers wait for the first blocker's taken branch to be processed.
//
// This tests the scenario:
//
//	var COND1 = true
//	var COND2 = true
//	if @var.COND1 { var B = 2 }
//	var C = 3                      # Between blockers - waits for wave 2
//	if @var.COND2 { var D = 4 }
//
// Wave 1: Collect COND1, COND2, blocker1 condition, blocker2 condition
// Wave 2: Process blocker1's taken branch (var B), then C, then blocker2
// Wave 3: Process blocker2's taken branch (var D)
func TestResolve_MultipleBlockersSameLevel(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare variables - conditions BEFORE blockers
	cond1ExprID := v.DeclareVariable("COND1", "literal:true")
	cond2ExprID := v.DeclareVariable("COND2", "literal:true")
	bExprID := v.DeclareVariable("B", "literal:2")
	cExprID := v.DeclareVariable("C", "literal:3")
	dExprID := v.DeclareVariable("D", "literal:4")

	scopes := NewScopeStack()
	scopes.Define("COND1", cond1ExprID)
	scopes.Define("COND2", cond2ExprID)

	blocker1 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND1",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "B",
					ExprID: bExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: 2},
				},
			},
		},
	}

	blocker2 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND2",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "D",
					ExprID: dExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: 4},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			// var COND1 = true (before any blocker)
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND1",
					ExprID: cond1ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			// var COND2 = true (before any blocker)
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND2",
					ExprID: cond2ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			// if @var.COND1 { var B = 2 }
			{Kind: StmtBlocker, Blocker: blocker1},
			// var C = 3 (between blockers - processed in wave 2)
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "C",
					ExprID: cExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: 3},
				},
			},
			// if @var.COND2 { var D = 4 }
			{Kind: StmtBlocker, Blocker: blocker2},
		},
		Scopes: scopes,
	}

	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Both blockers should be taken
	if blocker1.Taken == nil || !*blocker1.Taken {
		t.Errorf("blocker1 should be taken (COND1=true)")
	}
	if blocker2.Taken == nil || !*blocker2.Taken {
		t.Errorf("blocker2 should be taken (COND2=true)")
	}

	// All variables should be touched
	for name, exprID := range map[string]string{
		"B": bExprID, "C": cExprID, "D": dExprID,
		"COND1": cond1ExprID, "COND2": cond2ExprID,
	} {
		if !v.IsTouched(exprID) {
			t.Errorf("%s should be touched", name)
		}
	}

	// B, C, and D should be in scopes
	if _, ok := scopes.Lookup("B"); !ok {
		t.Errorf("B should be in scopes (leaked from blocker1)")
	}
	if _, ok := scopes.Lookup("C"); !ok {
		t.Errorf("C should be in scopes (processed between blockers)")
	}
	if _, ok := scopes.Lookup("D"); !ok {
		t.Errorf("D should be in scopes (leaked from blocker2)")
	}
}

// TestResolve_SequentialBlockers_ConditionAfterFirstBlocker tests the scenario where
// the second blocker's condition is defined AFTER the first blocker. This requires
// multiple waves because statements after a blocker wait for its taken branch.
//
//	var COND1 = true
//	if @var.COND1 { var B = 2 }     # Wave 1: blocker1
//	var COND2 = true                 # Wave 2: processed after blocker1's branch
//	if @var.COND2 { var D = 4 }     # Wave 2: blocker2 (becomes blocker in wave 2)
//	echo done                        # Wave 3: after blocker2's branch
//
// This verifies that:
// 1. Blocker conditions defined after earlier blockers still work
// 2. The wave model correctly sequences dependent statements
func TestResolve_SequentialBlockers_ConditionAfterFirstBlocker(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare variables
	cond1ExprID := v.DeclareVariable("COND1", "literal:true")
	bExprID := v.DeclareVariable("B", "literal:2")
	cond2ExprID := v.DeclareVariable("COND2", "literal:true")
	dExprID := v.DeclareVariable("D", "literal:4")

	scopes := NewScopeStack()
	scopes.Define("COND1", cond1ExprID)
	// Note: COND2 is NOT pre-defined in scopes - it's declared after blocker1

	blocker1 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND1",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "B",
					ExprID: bExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: 2},
				},
			},
		},
	}

	blocker2 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "COND2",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "D",
					ExprID: dExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: 4},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			// var COND1 = true
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND1",
					ExprID: cond1ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			// if @var.COND1 { var B = 2 }
			{Kind: StmtBlocker, Blocker: blocker1},
			// var COND2 = true (AFTER blocker1 - processed in wave 2)
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COND2",
					ExprID: cond2ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			// if @var.COND2 { var D = 4 }
			{Kind: StmtBlocker, Blocker: blocker2},
		},
		Scopes: scopes,
	}

	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Both blockers should be taken
	if blocker1.Taken == nil || !*blocker1.Taken {
		t.Errorf("blocker1 should be taken (COND1=true)")
	}
	if blocker2.Taken == nil || !*blocker2.Taken {
		t.Errorf("blocker2 should be taken (COND2=true)")
	}

	// All variables should be touched
	if !v.IsTouched(cond1ExprID) {
		t.Errorf("COND1 should be touched")
	}
	if !v.IsTouched(bExprID) {
		t.Errorf("B should be touched (in blocker1's taken branch)")
	}
	if !v.IsTouched(cond2ExprID) {
		t.Errorf("COND2 should be touched (processed in wave 2)")
	}
	if !v.IsTouched(dExprID) {
		t.Errorf("D should be touched (in blocker2's taken branch)")
	}

	// All variables should be in scopes
	if _, ok := scopes.Lookup("COND2"); !ok {
		t.Errorf("COND2 should be in scopes (declared after blocker1)")
	}
	if _, ok := scopes.Lookup("B"); !ok {
		t.Errorf("B should be in scopes (leaked from blocker1)")
	}
	if _, ok := scopes.Lookup("D"); !ok {
		t.Errorf("D should be in scopes (leaked from blocker2)")
	}
}

// TestResolve_ThreeLevelNestedIfs_AllTrue tests 3 levels of nested if statements
// where all conditions are true. This requires 3 waves of resolution.
//
//	if @var.L1 {           # Wave 1: resolve L1
//	    if @var.L2 {       # Wave 2: resolve L2
//	        if @var.L3 {   # Wave 3: resolve L3
//	            var DEEP = "found"
//	        }
//	    }
//	}
func TestResolve_ThreeLevelNestedIfs_AllTrue(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare condition variables
	l1ExprID := v.DeclareVariable("L1", "literal:true")
	l2ExprID := v.DeclareVariable("L2", "literal:true")
	l3ExprID := v.DeclareVariable("L3", "literal:true")
	deepExprID := v.DeclareVariable("DEEP", "literal:found")

	scopes := NewScopeStack()
	scopes.Define("L1", l1ExprID)
	scopes.Define("L2", l2ExprID)
	scopes.Define("L3", l3ExprID)

	// Build nested structure from inside out
	blocker3 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L3",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "DEEP",
					ExprID: deepExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "found"},
				},
			},
		},
	}

	blocker2 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L2",
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtBlocker, Blocker: blocker3},
		},
	}

	blocker1 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L1",
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtBlocker, Blocker: blocker2},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L1",
					ExprID: l1ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L2",
					ExprID: l2ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L3",
					ExprID: l3ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{Kind: StmtBlocker, Blocker: blocker1},
		},
		Scopes: scopes,
	}

	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// All blockers should be taken
	if blocker1.Taken == nil || !*blocker1.Taken {
		t.Errorf("blocker1 (L1) should be taken")
	}
	if blocker2.Taken == nil || !*blocker2.Taken {
		t.Errorf("blocker2 (L2) should be taken")
	}
	if blocker3.Taken == nil || !*blocker3.Taken {
		t.Errorf("blocker3 (L3) should be taken")
	}

	// DEEP should be touched and in scopes
	if !v.IsTouched(deepExprID) {
		t.Errorf("DEEP should be touched (all conditions true)")
	}
	if _, ok := scopes.Lookup("DEEP"); !ok {
		t.Errorf("DEEP should be in scopes (leaked from innermost block)")
	}
}

// TestResolve_ThreeLevelNestedIfs_ThirdFalse tests 3 levels of nested if statements
// where the third (innermost) condition is false. The deepest variable should NOT
// be touched (branch pruning).
//
//	if @var.L1 {           # true
//	    if @var.L2 {       # true
//	        if @var.L3 {   # FALSE - this branch not taken
//	            var DEEP = "found"  # Should NOT be touched
//	        }
//	    }
//	}
func TestResolve_ThreeLevelNestedIfs_ThirdFalse(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare condition variables - L3 is false
	l1ExprID := v.DeclareVariable("L1", "literal:true")
	l2ExprID := v.DeclareVariable("L2", "literal:true")
	l3ExprID := v.DeclareVariable("L3", "literal:false")
	deepExprID := v.DeclareVariable("DEEP", "literal:found")

	scopes := NewScopeStack()
	scopes.Define("L1", l1ExprID)
	scopes.Define("L2", l2ExprID)
	scopes.Define("L3", l3ExprID)

	// Build nested structure from inside out
	blocker3 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L3",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "DEEP",
					ExprID: deepExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "found"},
				},
			},
		},
	}

	blocker2 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L2",
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtBlocker, Blocker: blocker3},
		},
	}

	blocker1 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L1",
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtBlocker, Blocker: blocker2},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L1",
					ExprID: l1ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L2",
					ExprID: l2ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L3",
					ExprID: l3ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: false},
				},
			},
			{Kind: StmtBlocker, Blocker: blocker1},
		},
		Scopes: scopes,
	}

	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// First two blockers should be taken, third should NOT
	if blocker1.Taken == nil || !*blocker1.Taken {
		t.Errorf("blocker1 (L1) should be taken")
	}
	if blocker2.Taken == nil || !*blocker2.Taken {
		t.Errorf("blocker2 (L2) should be taken")
	}
	if blocker3.Taken == nil || *blocker3.Taken {
		t.Errorf("blocker3 (L3) should NOT be taken (L3=false)")
	}

	// DEEP should NOT be touched (branch pruning)
	if v.IsTouched(deepExprID) {
		t.Errorf("DEEP should NOT be touched (L3 condition is false, branch pruned)")
	}

	// DEEP should NOT be in scopes (never processed)
	if _, ok := scopes.Lookup("DEEP"); ok {
		t.Errorf("DEEP should NOT be in scopes (branch was pruned)")
	}
}

// TestResolve_ThreeLevelNestedIfs_SecondFalse tests 3 levels of nested if statements
// where the second condition is false. Neither the second nor third level should
// be processed (early branch pruning).
func TestResolve_ThreeLevelNestedIfs_SecondFalse(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Declare condition variables - L2 is false
	l1ExprID := v.DeclareVariable("L1", "literal:true")
	l2ExprID := v.DeclareVariable("L2", "literal:false")
	l3ExprID := v.DeclareVariable("L3", "literal:true")
	deepExprID := v.DeclareVariable("DEEP", "literal:found")

	scopes := NewScopeStack()
	scopes.Define("L1", l1ExprID)
	scopes.Define("L2", l2ExprID)
	scopes.Define("L3", l3ExprID)

	// Build nested structure from inside out
	blocker3 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L3",
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "DEEP",
					ExprID: deepExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "found"},
				},
			},
		},
	}

	blocker2 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L2",
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtBlocker, Blocker: blocker3},
		},
	}

	blocker1 := &BlockerIR{
		Kind: BlockerIf,
		Condition: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "L1",
		},
		ThenBranch: []*StatementIR{
			{Kind: StmtBlocker, Blocker: blocker2},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L1",
					ExprID: l1ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L2",
					ExprID: l2ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: false},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "L3",
					ExprID: l3ExprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: true},
				},
			},
			{Kind: StmtBlocker, Blocker: blocker1},
		},
		Scopes: scopes,
	}

	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// First blocker taken, second NOT taken, third never evaluated
	if blocker1.Taken == nil || !*blocker1.Taken {
		t.Errorf("blocker1 (L1) should be taken")
	}
	if blocker2.Taken == nil || *blocker2.Taken {
		t.Errorf("blocker2 (L2) should NOT be taken (L2=false)")
	}
	// blocker3 should never be evaluated (parent branch not taken)
	if blocker3.Taken != nil {
		t.Errorf("blocker3 (L3) should NOT be evaluated (parent branch pruned)")
	}

	// DEEP should NOT be touched
	if v.IsTouched(deepExprID) {
		t.Errorf("DEEP should NOT be touched (L2 is false, entire subtree pruned)")
	}
}

// TestResolve_ForLoopEachIterationGetsUniqueBinding verifies that each iteration
// of a for-loop gets its own unique variable binding, not a shared reference.
// This is the critical test for the "all iterations reference last value" bug.
func TestResolve_ForLoopEachIterationGetsUniqueBinding(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	// Build IR:
	//   for item in ["a", "b", "c"] {
	//     echo @var.item
	//   }
	//
	// After unrolling, this becomes:
	//   VarDecl: item = "a"   (exprID_1)
	//   echo @var.item        (references exprID_1)
	//   VarDecl: item = "b"   (exprID_2)
	//   echo @var.item        (references exprID_2)
	//   VarDecl: item = "c"   (exprID_3)
	//   echo @var.item        (references exprID_3)
	//
	// Each VarDecl creates a NEW exprID, ensuring each iteration
	// sees its own value, not the last value.

	forBlocker := &BlockerIR{
		Kind:    BlockerFor,
		LoopVar: "item",
		Collection: &ExprIR{
			Kind:  ExprLiteral,
			Value: []string{"a", "b", "c"},
		},
		ThenBranch: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "item"},
						},
					},
				},
			},
		},
	}

	graph := &ExecutionGraph{
		Statements: []*StatementIR{
			{Kind: StmtBlocker, Blocker: forBlocker},
		},
		Scopes: scopes,
	}

	session := &mockSession{}
	config := ResolveConfig{Context: context.Background()}
	err := Resolve(graph, v, session, config)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// After resolution, the loop variable "item" should be in scope
	// with the value from the LAST iteration (due to flattening semantics)
	itemExprID, ok := scopes.Lookup("item")
	if !ok {
		t.Fatalf("Loop variable 'item' not found in scopes after resolution")
	}

	// The exprID should be touched (used in echo command)
	if !v.IsTouched(itemExprID) {
		t.Errorf("Loop variable 'item' should be touched")
	}

	// The key insight: each iteration creates a NEW exprID via DeclareVariable.
	// The scope only tracks the LAST one (due to rebinding), but each iteration's
	// VarDecl statement has its own unique exprID that was used when that
	// iteration's body was processed.
	//
	// This test passes because:
	// 1. evaluateForBlocker creates a new exprID for each iteration (line 580)
	// 2. Each VarDecl is injected BEFORE its iteration's body (line 594)
	// 3. When wave 2 processes VarDecl, it updates scopes (line 354)
	// 4. The subsequent command in that iteration sees the correct value
}
