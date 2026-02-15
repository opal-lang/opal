package planner

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators" // Register decorators
	"github.com/opal-lang/opal/runtime/vault"
)

// TestEmit_SimpleCommand tests emitting a simple command with no variables.
// Input IR: echo "hello"
// Expected: Plan with 1 Step containing CommandNode with @shell decorator
func TestEmit_SimpleCommand(t *testing.T) {
	// Build resolved IR: echo "hello"
	result := &ResolveResult{
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
	}

	// Create vault (no secrets in this test)
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Create emitter
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	// Emit plan
	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Verify plan structure
	if plan == nil {
		t.Fatal("Emit() returned nil plan")
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if step.ID != 1 {
		t.Errorf("Step.ID = %d, want 1", step.ID)
	}

	// Verify tree is a CommandNode
	cmdNode, ok := step.Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.CommandNode", step.Tree)
	}

	if cmdNode.Decorator != "@shell" {
		t.Errorf("CommandNode.Decorator = %q, want %q", cmdNode.Decorator, "@shell")
	}

	// Verify command argument
	if len(cmdNode.Args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(cmdNode.Args))
	}

	expectedArg := planfmt.Arg{
		Key: "command",
		Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo \"hello\""},
	}
	if diff := cmp.Diff(expectedArg, cmdNode.Args[0]); diff != "" {
		t.Errorf("Arg mismatch (-want +got):\n%s", diff)
	}
}

func TestEmit_NilResolveResult(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(nil, v, NewScopeStack(), "")

	_, err := emitter.Emit()
	if err == nil {
		t.Fatal("Expected error for nil resolve result")
	}

	expected := "cannot emit: no resolve result"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("Error mismatch (-want +got):\n%s", diff)
	}
}

func TestEmit_NilCommand(t *testing.T) {
	result := &ResolveResult{
		Statements: []*StatementIR{{
			Kind: StmtCommand,
		}},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	_, err := emitter.Emit()
	if err == nil {
		t.Fatal("Expected error for nil command")
	}

	expected := "nil command in StmtCommand at index 0"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("Error mismatch (-want +got):\n%s", diff)
	}
}

func TestEmitterGetValue_EnumMemberPrecedesDecoratorKey(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	exprID := v.TrackExpression("@env.HOME")
	v.StoreUnresolvedValue(exprID, "/tmp/decorator")

	result := &ResolveResult{
		DecoratorExprIDs: map[string]string{"env.HOME": exprID},
		EnumMemberValues: map[string]string{"env.HOME": "enum-home"},
	}

	emitter := NewEmitter(result, v, NewScopeStack(), "")
	value, ok := emitter.getValue("env.HOME")
	if !ok {
		t.Fatal("expected value")
	}

	if diff := cmp.Diff("enum-home", value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

// TestEmit_MultipleCommands tests emitting multiple sequential commands.
func TestEmit_MultipleCommands(t *testing.T) {
	// Build resolved IR: echo "a"; echo "b"
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"a\""},
						},
					},
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"b\""},
						},
					},
				},
			},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify step IDs are sequential
	if plan.Steps[0].ID != 1 {
		t.Errorf("Step[0].ID = %d, want 1", plan.Steps[0].ID)
	}
	if plan.Steps[1].ID != 2 {
		t.Errorf("Step[1].ID = %d, want 2", plan.Steps[1].ID)
	}

	// Verify commands
	cmd0 := plan.Steps[0].Tree.(*planfmt.CommandNode)
	cmd1 := plan.Steps[1].Tree.(*planfmt.CommandNode)

	if cmd0.Args[0].Val.Str != "echo \"a\"" {
		t.Errorf("Step[0] command = %q, want %q", cmd0.Args[0].Val.Str, "echo \"a\"")
	}
	if cmd1.Args[0].Val.Str != "echo \"b\"" {
		t.Errorf("Step[1] command = %q, want %q", cmd1.Args[0].Val.Str, "echo \"b\"")
	}
}

func TestEmit_CommandChainDoesNotMutateOperators(t *testing.T) {
	cmd1 := &CommandStmtIR{
		Decorator: "@shell",
		Operator:  ";",
		Command: &CommandExpr{
			Parts: []*ExprIR{{Kind: ExprLiteral, Value: "echo \"a\""}},
		},
	}
	cmd2 := &CommandStmtIR{
		Decorator: "@shell",
		Command: &CommandExpr{
			Parts: []*ExprIR{{Kind: ExprLiteral, Value: "echo \"b\""}},
		},
	}

	result := &ResolveResult{
		Statements: []*StatementIR{
			{Kind: StmtCommand, Command: cmd1},
			{Kind: StmtCommand, Command: cmd2},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	_, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if diff := cmp.Diff(";", cmd1.Operator); diff != "" {
		t.Errorf("Operator mismatch (-want +got):\n%s", diff)
	}
}

// TestEmit_EmptyResult tests emitting an empty result.
func TestEmit_EmptyResult(t *testing.T) {
	result := &ResolveResult{
		Statements: nil,
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if plan == nil {
		t.Fatal("Emit() returned nil plan")
	}

	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(plan.Steps))
	}
}

// TestEmit_Target tests that target is set correctly.
func TestEmit_Target(t *testing.T) {
	result := &ResolveResult{
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
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "deploy")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if plan.Target != "deploy" {
		t.Errorf("Plan.Target = %q, want %q", plan.Target, "deploy")
	}
}

func TestEmit_CallTraceStatement(t *testing.T) {
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtCallTrace,
				CallTrace: &CallTraceStmtIR{
					Label: "deploy(prod, retries=5)",
					Block: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command:   &CommandExpr{Parts: []*ExprIR{{Kind: ExprLiteral, Value: "echo \"hello\""}}},
							},
						},
					},
				},
			},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "deploy")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	logic, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.LogicNode", plan.Steps[0].Tree)
	}
	if diff := cmp.Diff("call", logic.Kind); diff != "" {
		t.Fatalf("logic kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("deploy(prod, retries=5)", logic.Condition); diff != "" {
		t.Fatalf("call label mismatch (-want +got):\n%s", diff)
	}
	if len(logic.Block) != 1 {
		t.Fatalf("len(logic block) = %d, want 1", len(logic.Block))
	}
	if _, ok := logic.Block[0].Tree.(*planfmt.CommandNode); !ok {
		t.Fatalf("logic block[0] tree is %T, want *planfmt.CommandNode", logic.Block[0].Tree)
	}
}

// TestEmit_PlanSalt tests that PlanSalt is set.
func TestEmit_PlanSalt(t *testing.T) {
	result := &ResolveResult{
		Statements: nil,
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.PlanSalt) != 32 {
		t.Errorf("Plan.PlanSalt length = %d, want 32", len(plan.PlanSalt))
	}

	// Verify salt is not all zeros
	allZero := true
	for _, b := range plan.PlanSalt {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Plan.PlanSalt is all zeros, should be random")
	}
}

// TestEmit_VarRef_DisplayIDSubstitution tests that variable references are
// replaced with DisplayID placeholders in the emitted command.
//
// Input IR: var NAME = "world"; echo "Hello @var.NAME"
// Expected: Command string contains DisplayID placeholder (opal:...)
func TestEmit_VarRef_DisplayIDSubstitution(t *testing.T) {
	// Setup vault with a resolved variable
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Simulate what IR Builder + Resolver do:
	// 1. Declare variable in vault
	exprID := v.DeclareVariable("NAME", "literal:world")
	// 2. Store the value
	v.StoreUnresolvedValue(exprID, "world")
	// 3. Mark as touched (in execution path)
	v.MarkTouched(exprID)
	// 4. Resolve to generate DisplayID
	v.ResolveAllTouched()

	// Get the DisplayID that was generated
	displayID := v.GetDisplayID(exprID)
	if displayID == "" {
		t.Fatal("DisplayID not generated for variable")
	}

	// Setup scopes with the variable
	scopes := NewScopeStack()
	scopes.Define("NAME", exprID)

	// Build resolved IR: echo "Hello @var.NAME"
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "NAME",
					ExprID: exprID,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "world"},
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"Hello "},
							{Kind: ExprVarRef, VarName: "NAME"},
							{Kind: ExprLiteral, Value: "\""},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmdNode := plan.Steps[0].Tree.(*planfmt.CommandNode)
	commandStr := cmdNode.Args[0].Val.Str

	// The command should contain the DisplayID, not the variable name
	expectedCommand := "echo \"Hello " + displayID + "\""
	if commandStr != expectedCommand {
		t.Errorf("Command string mismatch:\n  got:  %q\n  want: %q", commandStr, expectedCommand)
	}

	// Verify DisplayID has correct format
	if len(displayID) < 5 || displayID[:5] != "opal:" {
		t.Errorf("DisplayID should have 'opal:' prefix, got: %q", displayID)
	}
}

// TestEmit_VarRef_SecretUsesTracking tests that SecretUses are recorded
// when variables are used in commands.
func TestEmit_VarRef_SecretUsesTracking(t *testing.T) {
	// Setup vault with a resolved variable
	v := vault.NewWithPlanKey([]byte("test-key"))

	exprID := v.DeclareVariable("SECRET", "literal:secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	displayID := v.GetDisplayID(exprID)

	scopes := NewScopeStack()
	scopes.Define("SECRET", exprID)

	// Build resolved IR: echo @var.SECRET
	result := &ResolveResult{
		Statements: []*StatementIR{
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

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Verify SecretUses contains the variable usage
	if len(plan.SecretUses) != 1 {
		t.Fatalf("Expected 1 SecretUse, got %d", len(plan.SecretUses))
	}

	use := plan.SecretUses[0]

	// Verify DisplayID matches
	if use.DisplayID != displayID {
		t.Errorf("SecretUse.DisplayID = %q, want %q", use.DisplayID, displayID)
	}

	// Verify Site is set and contains expected path components
	if use.Site == "" {
		t.Error("SecretUse.Site should not be empty")
	}
	if !containsStr(use.Site, "root") {
		t.Errorf("SecretUse.Site should contain 'root', got: %q", use.Site)
	}
	if !containsStr(use.Site, "params") {
		t.Errorf("SecretUse.Site should contain 'params', got: %q", use.Site)
	}

	// Verify SiteID is set (HMAC-based)
	if use.SiteID == "" {
		t.Error("SecretUse.SiteID should not be empty")
	}
}

// TestEmit_MultipleVarRefs tests multiple variable references in one command.
func TestEmit_MultipleVarRefs(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	// Setup two variables
	exprID1 := v.DeclareVariable("FIRST", "literal:first")
	v.StoreUnresolvedValue(exprID1, "first")
	v.MarkTouched(exprID1)

	exprID2 := v.DeclareVariable("SECOND", "literal:second")
	v.StoreUnresolvedValue(exprID2, "second")
	v.MarkTouched(exprID2)

	v.ResolveAllTouched()

	displayID1 := v.GetDisplayID(exprID1)
	displayID2 := v.GetDisplayID(exprID2)

	scopes := NewScopeStack()
	scopes.Define("FIRST", exprID1)
	scopes.Define("SECOND", exprID2)

	// Build resolved IR: echo @var.FIRST @var.SECOND
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "FIRST"},
							{Kind: ExprLiteral, Value: " "},
							{Kind: ExprVarRef, VarName: "SECOND"},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	cmdNode := plan.Steps[0].Tree.(*planfmt.CommandNode)
	commandStr := cmdNode.Args[0].Val.Str

	expectedCommand := "echo " + displayID1 + " " + displayID2
	if commandStr != expectedCommand {
		t.Errorf("Command string mismatch:\n  got:  %q\n  want: %q", commandStr, expectedCommand)
	}

	// Should have 2 SecretUses (one for each variable)
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses, got %d", len(plan.SecretUses))
	}
}

func TestEmit_ScopeOrderUsesLatestBinding(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	exprID1 := v.DeclareVariable("COUNT", "literal:1")
	v.StoreUnresolvedValue(exprID1, "1")
	v.MarkTouched(exprID1)

	exprID2 := v.DeclareVariable("COUNT", "literal:2")
	v.StoreUnresolvedValue(exprID2, "2")
	v.MarkTouched(exprID2)

	v.ResolveAllTouched()

	displayID1 := v.GetDisplayID(exprID1)
	displayID2 := v.GetDisplayID(exprID2)

	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COUNT",
					ExprID: exprID1,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "1"},
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "COUNT"},
						},
					},
				},
			},
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COUNT",
					ExprID: exprID2,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "2"},
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "COUNT"},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	command1 := plan.Steps[0].Tree.(*planfmt.CommandNode).Args[0].Val.Str
	command2 := plan.Steps[1].Tree.(*planfmt.CommandNode).Args[0].Val.Str

	if !strings.Contains(command1, displayID1) {
		t.Errorf("First command should use displayID1, got %q", command1)
	}
	if !strings.Contains(command2, displayID2) {
		t.Errorf("Second command should use displayID2, got %q", command2)
	}
}

func TestEmit_BlockScopeIsolation(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	exprIDOuter := v.DeclareVariable("COUNT", "literal:1")
	v.StoreUnresolvedValue(exprIDOuter, "1")
	v.MarkTouched(exprIDOuter)

	exprIDInner := v.DeclareVariable("COUNT", "literal:2")
	v.StoreUnresolvedValue(exprIDInner, "2")
	v.MarkTouched(exprIDInner)

	v.ResolveAllTouched()

	displayIDOuter := v.GetDisplayID(exprIDOuter)
	displayIDInner := v.GetDisplayID(exprIDInner)

	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtVarDecl,
				VarDecl: &VarDeclIR{
					Name:   "COUNT",
					ExprID: exprIDOuter,
					Value:  &ExprIR{Kind: ExprLiteral, Value: "1"},
				},
			},
			{
				Kind:         StmtCommand,
				CreatesScope: true,
				Command: &CommandStmtIR{
					Decorator: "@retry",
					Args: []ArgIR{
						{Name: "times", Value: &ExprIR{Kind: ExprLiteral, Value: 3}},
					},
					Block: []*StatementIR{
						{
							Kind: StmtVarDecl,
							VarDecl: &VarDeclIR{
								Name:   "COUNT",
								ExprID: exprIDInner,
								Value:  &ExprIR{Kind: ExprLiteral, Value: "2"},
							},
						},
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo "},
										{Kind: ExprVarRef, VarName: "COUNT"},
									},
								},
							},
						},
					},
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprVarRef, VarName: "COUNT"},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	retryNode := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if len(retryNode.Block) != 1 {
		t.Fatalf("Expected retry block to have 1 step, got %d", len(retryNode.Block))
	}

	innerCmd := retryNode.Block[0].Tree.(*planfmt.CommandNode).Args[0].Val.Str
	outerCmd := plan.Steps[1].Tree.(*planfmt.CommandNode).Args[0].Val.Str

	if !strings.Contains(innerCmd, displayIDInner) {
		t.Errorf("Inner command should use displayIDInner, got %q", innerCmd)
	}
	if !strings.Contains(outerCmd, displayIDOuter) {
		t.Errorf("Outer command should use displayIDOuter, got %q", outerCmd)
	}
}

func TestEmit_DecoratorRefDisplayID(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))

	ref := &DecoratorRef{Name: "env", Selector: []string{"HOME"}}
	exprID := v.TrackExpression("@env.HOME")
	v.StoreUnresolvedValue(exprID, "/home/test")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	displayID := v.GetDisplayID(exprID)

	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo "},
							{Kind: ExprDecoratorRef, Decorator: ref},
						},
					},
				},
			},
		},
		DecoratorExprIDs: map[string]string{
			decoratorKey(ref): exprID,
		},
	}

	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	command := plan.Steps[0].Tree.(*planfmt.CommandNode).Args[0].Val.Str
	if !strings.Contains(command, displayID) {
		t.Errorf("Expected decorator DisplayID in command, got %q", command)
	}
}

// TestEmit_IfStatement_ThenBranch tests that if statements emit as LogicNode with structure.
//
// Input IR: if @var.ENV == "prod" { echo "then" } else { echo "else" }
// Expected: Single LogicNode step with condition, result="true", and nested command
func TestEmit_IfStatement_ThenBranch(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	// Build resolved IR with Taken=true (then branch taken)
	taken := true
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind: BlockerIf,
					Condition: &ExprIR{
						Kind:  ExprBinaryOp,
						Op:    "==",
						Left:  &ExprIR{Kind: ExprVarRef, VarName: "ENV"},
						Right: &ExprIR{Kind: ExprLiteral, Value: "prod"},
					},
					Taken: &taken,
					ThenBranch: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"then\""},
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
										{Kind: ExprLiteral, Value: "echo \"else\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Should have exactly 1 step (LogicNode wrapping the if)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Verify it's a LogicNode
	logicNode, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.LogicNode", plan.Steps[0].Tree)
	}

	// Verify LogicNode fields
	if logicNode.Kind != "if" {
		t.Errorf("LogicNode.Kind = %q, want %q", logicNode.Kind, "if")
	}
	if logicNode.Result != "true" {
		t.Errorf("LogicNode.Result = %q, want %q", logicNode.Result, "true")
	}

	// Verify nested block has the then branch command
	if len(logicNode.Block) != 1 {
		t.Fatalf("LogicNode.Block has %d steps, want 1", len(logicNode.Block))
	}

	cmdNode, ok := logicNode.Block[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Block[0].Tree is %T, want *planfmt.CommandNode", logicNode.Block[0].Tree)
	}
	if cmdNode.Args[0].Val.Str != "echo \"then\"" {
		t.Errorf("Expected then branch command, got: %q", cmdNode.Args[0].Val.Str)
	}
}

// TestEmit_IfStatement_ElseBranch tests that else branch is emitted when condition is false.
//
// Input IR: if @var.ENV == "prod" { echo "then" } else { echo "else" }
// Expected: LogicNode with result="false" and else branch command
func TestEmit_IfStatement_ElseBranch(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	// Build resolved IR with Taken=false (else branch taken)
	taken := false
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind: BlockerIf,
					Condition: &ExprIR{
						Kind:  ExprBinaryOp,
						Op:    "==",
						Left:  &ExprIR{Kind: ExprVarRef, VarName: "ENV"},
						Right: &ExprIR{Kind: ExprLiteral, Value: "prod"},
					},
					Taken: &taken,
					ThenBranch: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"then\""},
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
										{Kind: ExprLiteral, Value: "echo \"else\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Should have exactly 1 step (LogicNode)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	logicNode := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if logicNode.Result != "false" {
		t.Errorf("LogicNode.Result = %q, want %q", logicNode.Result, "false")
	}

	// Verify nested block has the else branch command
	if len(logicNode.Block) != 1 {
		t.Fatalf("LogicNode.Block has %d steps, want 1", len(logicNode.Block))
	}

	cmdNode := logicNode.Block[0].Tree.(*planfmt.CommandNode)
	if cmdNode.Args[0].Val.Str != "echo \"else\"" {
		t.Errorf("Expected else branch command, got: %q", cmdNode.Args[0].Val.Str)
	}
}

// TestEmit_IfStatement_NoElse tests if statement with no else branch when condition is false.
//
// Input IR: if false { echo "then" }
// Expected: No steps emitted (nothing to show)
func TestEmit_IfStatement_NoElse(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	// Build resolved IR with Taken=false and no else branch
	taken := false
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind: BlockerIf,
					Condition: &ExprIR{
						Kind:  ExprLiteral,
						Value: false,
					},
					Taken: &taken,
					ThenBranch: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"then\""},
									},
								},
							},
						},
					},
					ElseBranch: nil, // No else branch
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Should have no steps (condition false, no else branch)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(plan.Steps))
	}
}

// TestEmit_IfStatement_NestedCommands tests if statement with multiple commands in branch.
//
// Input IR: if true { echo "a"; echo "b" }
// Expected: LogicNode with 2 nested command steps
func TestEmit_IfStatement_NestedCommands(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	taken := true
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind: BlockerIf,
					Condition: &ExprIR{
						Kind:  ExprLiteral,
						Value: true,
					},
					Taken: &taken,
					ThenBranch: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"a\""},
									},
								},
							},
						},
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"b\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Should have 1 LogicNode step
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	logicNode := plan.Steps[0].Tree.(*planfmt.LogicNode)

	// LogicNode should have 2 nested steps
	if len(logicNode.Block) != 2 {
		t.Fatalf("LogicNode.Block has %d steps, want 2", len(logicNode.Block))
	}

	cmd0 := logicNode.Block[0].Tree.(*planfmt.CommandNode)
	cmd1 := logicNode.Block[1].Tree.(*planfmt.CommandNode)

	if cmd0.Args[0].Val.Str != "echo \"a\"" {
		t.Errorf("Block[0] command = %q, want %q", cmd0.Args[0].Val.Str, "echo \"a\"")
	}
	if cmd1.Args[0].Val.Str != "echo \"b\"" {
		t.Errorf("Block[1] command = %q, want %q", cmd1.Args[0].Val.Str, "echo \"b\"")
	}
}

// TestEmit_ForLoop_Iterations tests that for-loops emit one LogicNode per iteration.
//
// Input IR: for region in ["us-east", "eu-west"] { echo "Deploying to @var.region" }
// Expected: 2 LogicNode steps, one per iteration, each showing the loop variable value
func TestEmit_ForLoop_Iterations(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	// Build resolved IR with 2 iterations (Resolver populates Iterations)
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind:    BlockerFor,
					LoopVar: "region",
					Collection: &ExprIR{
						Kind:  ExprLiteral,
						Value: []string{"us-east", "eu-west"},
					},
					// Iterations populated by Resolver
					Iterations: []LoopIteration{
						{
							Value: "us-east",
							Body: []*StatementIR{
								{
									Kind: StmtCommand,
									Command: &CommandStmtIR{
										Decorator: "@shell",
										Command: &CommandExpr{
											Parts: []*ExprIR{
												{Kind: ExprLiteral, Value: "echo \"Deploying to us-east\""},
											},
										},
									},
								},
							},
						},
						{
							Value: "eu-west",
							Body: []*StatementIR{
								{
									Kind: StmtCommand,
									Command: &CommandStmtIR{
										Decorator: "@shell",
										Command: &CommandExpr{
											Parts: []*ExprIR{
												{Kind: ExprLiteral, Value: "echo \"Deploying to eu-west\""},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Should have 2 steps (one LogicNode per iteration)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify first iteration
	logic0 := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if logic0.Kind != "for" {
		t.Errorf("Step[0].Kind = %q, want %q", logic0.Kind, "for")
	}
	if !containsStr(logic0.Result, "us-east") {
		t.Errorf("Step[0].Result should contain 'us-east', got: %q", logic0.Result)
	}
	if !containsStr(logic0.Result, "iteration 1") {
		t.Errorf("Step[0].Result should contain 'iteration 1', got: %q", logic0.Result)
	}
	if len(logic0.Block) != 1 {
		t.Fatalf("Step[0].Block has %d steps, want 1", len(logic0.Block))
	}
	cmd0 := logic0.Block[0].Tree.(*planfmt.CommandNode)
	if cmd0.Args[0].Val.Str != "echo \"Deploying to us-east\"" {
		t.Errorf("Step[0] command = %q, want %q", cmd0.Args[0].Val.Str, "echo \"Deploying to us-east\"")
	}

	// Verify second iteration
	logic1 := plan.Steps[1].Tree.(*planfmt.LogicNode)
	if !containsStr(logic1.Result, "eu-west") {
		t.Errorf("Step[1].Result should contain 'eu-west', got: %q", logic1.Result)
	}
	if !containsStr(logic1.Result, "iteration 2") {
		t.Errorf("Step[1].Result should contain 'iteration 2', got: %q", logic1.Result)
	}
	cmd1 := logic1.Block[0].Tree.(*planfmt.CommandNode)
	if cmd1.Args[0].Val.Str != "echo \"Deploying to eu-west\"" {
		t.Errorf("Step[1] command = %q, want %q", cmd1.Args[0].Val.Str, "echo \"Deploying to eu-west\"")
	}
}

// TestEmit_ForLoop_EmptyIterations tests for-loop with no iterations.
//
// Input IR: for x in [] { echo "never" }
// Expected: No steps emitted
func TestEmit_ForLoop_EmptyIterations(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind:       BlockerFor,
					LoopVar:    "x",
					Collection: &ExprIR{Kind: ExprLiteral, Value: []string{}},
					Iterations: nil, // Empty - no iterations
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps for empty loop, got %d", len(plan.Steps))
	}
}

// TestEmit_ForLoop_MultipleCommandsPerIteration tests for-loop with multiple commands per iteration.
//
// Input IR: for x in ["a"] { echo "start"; echo "end" }
// Expected: 1 LogicNode with 2 nested command steps
func TestEmit_ForLoop_MultipleCommandsPerIteration(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtBlocker,
				Blocker: &BlockerIR{
					Kind:       BlockerFor,
					LoopVar:    "x",
					Collection: &ExprIR{Kind: ExprLiteral, Value: []string{"a"}},
					Iterations: []LoopIteration{
						{
							Value: "a",
							Body: []*StatementIR{
								{
									Kind: StmtCommand,
									Command: &CommandStmtIR{
										Decorator: "@shell",
										Command: &CommandExpr{
											Parts: []*ExprIR{
												{Kind: ExprLiteral, Value: "echo \"start\""},
											},
										},
									},
								},
								{
									Kind: StmtCommand,
									Command: &CommandStmtIR{
										Decorator: "@shell",
										Command: &CommandExpr{
											Parts: []*ExprIR{
												{Kind: ExprLiteral, Value: "echo \"end\""},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	logicNode := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if len(logicNode.Block) != 2 {
		t.Fatalf("LogicNode.Block has %d steps, want 2", len(logicNode.Block))
	}

	cmd0 := logicNode.Block[0].Tree.(*planfmt.CommandNode)
	cmd1 := logicNode.Block[1].Tree.(*planfmt.CommandNode)

	if cmd0.Args[0].Val.Str != "echo \"start\"" {
		t.Errorf("Block[0] command = %q, want %q", cmd0.Args[0].Val.Str, "echo \"start\"")
	}
	if cmd1.Args[0].Val.Str != "echo \"end\"" {
		t.Errorf("Block[1] command = %q, want %q", cmd1.Args[0].Val.Str, "echo \"end\"")
	}
}

// TestEmit_AndOperator tests that && chains create AndNode tree.
//
// Input IR: echo "a" && echo "b"
// Expected: Single step with AndNode tree
func TestEmit_AndOperator(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	// Two commands chained with &&
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"a\""},
						},
					},
					Operator: "&&", // Chain to next command
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"b\""},
						},
					},
					// No operator - end of chain
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Should have 1 step (both commands in one AndNode)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Tree should be AndNode
	andNode, ok := plan.Steps[0].Tree.(*planfmt.AndNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.AndNode", plan.Steps[0].Tree)
	}

	// Left should be first command
	leftCmd, ok := andNode.Left.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("AndNode.Left is %T, want *planfmt.CommandNode", andNode.Left)
	}
	if leftCmd.Args[0].Val.Str != "echo \"a\"" {
		t.Errorf("Left command = %q, want %q", leftCmd.Args[0].Val.Str, "echo \"a\"")
	}

	// Right should be second command
	rightCmd, ok := andNode.Right.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("AndNode.Right is %T, want *planfmt.CommandNode", andNode.Right)
	}
	if rightCmd.Args[0].Val.Str != "echo \"b\"" {
		t.Errorf("Right command = %q, want %q", rightCmd.Args[0].Val.Str, "echo \"b\"")
	}
}

// TestEmit_PipeOperator tests that | chains create PipelineNode tree.
//
// Input IR: echo "hello" | grep "hello"
// Expected: Single step with PipelineNode tree
func TestEmit_PipeOperator(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	result := &ResolveResult{
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
					Operator: "|",
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "grep \"hello\""},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	pipeNode, ok := plan.Steps[0].Tree.(*planfmt.PipelineNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.PipelineNode", plan.Steps[0].Tree)
	}

	if len(pipeNode.Commands) != 2 {
		t.Fatalf("PipelineNode has %d commands, want 2", len(pipeNode.Commands))
	}

	cmd0 := pipeNode.Commands[0].(*planfmt.CommandNode)
	cmd1 := pipeNode.Commands[1].(*planfmt.CommandNode)

	if cmd0.Args[0].Val.Str != "echo \"hello\"" {
		t.Errorf("Command[0] = %q, want %q", cmd0.Args[0].Val.Str, "echo \"hello\"")
	}
	if cmd1.Args[0].Val.Str != "grep \"hello\"" {
		t.Errorf("Command[1] = %q, want %q", cmd1.Args[0].Val.Str, "grep \"hello\"")
	}
}

// TestEmit_OrOperator tests that || chains create OrNode tree.
//
// Input IR: false || echo "fallback"
// Expected: Single step with OrNode tree
func TestEmit_OrOperator(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("test-key"))
	scopes := NewScopeStack()

	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "false"},
						},
					},
					Operator: "||",
				},
			},
			{
				Kind: StmtCommand,
				Command: &CommandStmtIR{
					Decorator: "@shell",
					Command: &CommandExpr{
						Parts: []*ExprIR{
							{Kind: ExprLiteral, Value: "echo \"fallback\""},
						},
					},
				},
			},
		},
	}

	emitter := NewEmitter(result, v, scopes, "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	orNode, ok := plan.Steps[0].Tree.(*planfmt.OrNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.OrNode", plan.Steps[0].Tree)
	}

	leftCmd := orNode.Left.(*planfmt.CommandNode)
	rightCmd := orNode.Right.(*planfmt.CommandNode)

	if leftCmd.Args[0].Val.Str != "false" {
		t.Errorf("Left command = %q, want %q", leftCmd.Args[0].Val.Str, "false")
	}
	if rightCmd.Args[0].Val.Str != "echo \"fallback\"" {
		t.Errorf("Right command = %q, want %q", rightCmd.Args[0].Val.Str, "echo \"fallback\"")
	}
}

// TestEmit_TryCatch_Basic tests basic try/catch emission.
// Input IR: try { echo "risky" } catch { echo "error" }
// Expected: Single step with TryNode containing both blocks
func TestEmit_TryCatch_Basic(t *testing.T) {
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtTry,
				Try: &TryIR{
					TryBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"risky\""},
									},
								},
							},
						},
					},
					CatchBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"error\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	tryNode, ok := plan.Steps[0].Tree.(*planfmt.TryNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.TryNode", plan.Steps[0].Tree)
	}

	if len(tryNode.TryBlock) != 1 {
		t.Errorf("TryBlock has %d steps, want 1", len(tryNode.TryBlock))
	}

	if len(tryNode.CatchBlock) != 1 {
		t.Errorf("CatchBlock has %d steps, want 1", len(tryNode.CatchBlock))
	}

	if len(tryNode.FinallyBlock) != 0 {
		t.Errorf("FinallyBlock has %d steps, want 0", len(tryNode.FinallyBlock))
	}
}

// TestEmit_TryCatchFinally tests try/catch/finally emission.
// Input IR: try { echo "risky" } catch { echo "error" } finally { echo "cleanup" }
// Expected: TryNode with all three blocks
func TestEmit_TryCatchFinally(t *testing.T) {
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtTry,
				Try: &TryIR{
					TryBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"risky\""},
									},
								},
							},
						},
					},
					CatchBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"error\""},
									},
								},
							},
						},
					},
					FinallyBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"cleanup\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	tryNode, ok := plan.Steps[0].Tree.(*planfmt.TryNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.TryNode", plan.Steps[0].Tree)
	}

	if len(tryNode.TryBlock) != 1 {
		t.Errorf("TryBlock has %d steps, want 1", len(tryNode.TryBlock))
	}

	if len(tryNode.CatchBlock) != 1 {
		t.Errorf("CatchBlock has %d steps, want 1", len(tryNode.CatchBlock))
	}

	if len(tryNode.FinallyBlock) != 1 {
		t.Errorf("FinallyBlock has %d steps, want 1", len(tryNode.FinallyBlock))
	}
}

// TestEmit_TryOnly tests try block without catch or finally.
// Input IR: try { echo "risky" }
// Expected: TryNode with only TryBlock populated
func TestEmit_TryOnly(t *testing.T) {
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtTry,
				Try: &TryIR{
					TryBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"risky\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	tryNode, ok := plan.Steps[0].Tree.(*planfmt.TryNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.TryNode", plan.Steps[0].Tree)
	}

	if len(tryNode.TryBlock) != 1 {
		t.Errorf("TryBlock has %d steps, want 1", len(tryNode.TryBlock))
	}

	if len(tryNode.CatchBlock) != 0 {
		t.Errorf("CatchBlock has %d steps, want 0", len(tryNode.CatchBlock))
	}

	if len(tryNode.FinallyBlock) != 0 {
		t.Errorf("FinallyBlock has %d steps, want 0", len(tryNode.FinallyBlock))
	}
}

// TestEmit_TryFinally tests try/finally without catch.
// Input IR: try { echo "risky" } finally { echo "cleanup" }
// Expected: TryNode with TryBlock and FinallyBlock, empty CatchBlock
func TestEmit_TryFinally(t *testing.T) {
	result := &ResolveResult{
		Statements: []*StatementIR{
			{
				Kind: StmtTry,
				Try: &TryIR{
					TryBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"risky\""},
									},
								},
							},
						},
					},
					FinallyBlock: []*StatementIR{
						{
							Kind: StmtCommand,
							Command: &CommandStmtIR{
								Decorator: "@shell",
								Command: &CommandExpr{
									Parts: []*ExprIR{
										{Kind: ExprLiteral, Value: "echo \"cleanup\""},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v := vault.NewWithPlanKey([]byte("test-key"))
	emitter := NewEmitter(result, v, NewScopeStack(), "")

	plan, err := emitter.Emit()
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	tryNode, ok := plan.Steps[0].Tree.(*planfmt.TryNode)
	if !ok {
		t.Fatalf("Step.Tree is %T, want *planfmt.TryNode", plan.Steps[0].Tree)
	}

	if len(tryNode.TryBlock) != 1 {
		t.Errorf("TryBlock has %d steps, want 1", len(tryNode.TryBlock))
	}

	if len(tryNode.CatchBlock) != 0 {
		t.Errorf("CatchBlock has %d steps, want 0", len(tryNode.CatchBlock))
	}

	if len(tryNode.FinallyBlock) != 1 {
		t.Errorf("FinallyBlock has %d steps, want 1", len(tryNode.FinallyBlock))
	}
}
