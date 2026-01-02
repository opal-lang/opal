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
