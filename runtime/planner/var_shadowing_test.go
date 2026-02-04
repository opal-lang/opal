package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"
)

// ========== Same-Level Shadowing Tests ==========
//
// Variables can be redeclared at the same scope level.
// Later declaration shadows (overrides) earlier declaration.
// This allows updating variable values in sequential code.

// Helper: Get command argument from tree
func getCommandArg(tree planfmt.ExecutionNode, key string) string {
	if tree == nil {
		return ""
	}
	cmd, ok := tree.(*planfmt.CommandNode)
	if !ok {
		return ""
	}
	for _, arg := range cmd.Args {
		if arg.Key == key {
			return arg.Val.Str
		}
	}
	return ""
}

func TestVarShadowing_SameLevelOverride_Works(t *testing.T) {
	// Later declaration at same scope level should override earlier one

	source := []byte(`
var COUNT = "5"
var COUNT = "10"
echo "@var.COUNT"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error on same-level override: %v", err)
	}

	// Verify plan was created
	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Verify we have steps
	if len(plan.Steps) == 0 {
		t.Fatal("Plan should have steps")
	}

	// The echo command should use the SECOND declaration (COUNT = "10")
	// We can verify this by checking that the DisplayID in the command
	// corresponds to the value "10", not "5"

	// Get the command from the tree
	step := plan.Steps[0]
	if step.Tree == nil {
		t.Fatal("Step tree should not be nil")
	}

	command := getCommandArg(step.Tree, "command")
	if !containsDisplayID(command) {
		t.Errorf("Command should contain DisplayID, got: %s", command)
	}

	// Verify the vault resolved the correct value
	// We need to check that the second declaration's exprID is what's used
	// This is tricky to verify directly, but we can check SecretUses
	if len(plan.SecretUses) == 0 {
		t.Fatal("Plan should have SecretUses")
	}

	// There should be exactly ONE SecretUse (for the second declaration)
	// Both declarations might create expressions, but only the second is referenced
	if len(plan.SecretUses) != 1 {
		t.Errorf("Expected 1 SecretUse (second declaration), got %d", len(plan.SecretUses))
	}
}

func TestVarShadowing_MultipleOverrides_UsesLatest(t *testing.T) {
	// Multiple overrides should use the latest declaration

	source := []byte(`
var COUNT = "5"
var COUNT = "10"
var COUNT = "15"
echo "@var.COUNT"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error on multiple overrides: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Should have exactly ONE SecretUse (for the third/latest declaration)
	if len(plan.SecretUses) != 1 {
		t.Errorf("Expected 1 SecretUse (latest declaration), got %d", len(plan.SecretUses))
	}
}

func TestVarShadowing_BothValuesStored_ForDeduplication(t *testing.T) {
	// Both declarations should create expressions (for deduplication)
	// but only the latest is accessible via variable name

	source := []byte(`
var COUNT = "5"
var COUNT = "10"
var ANOTHER = "5"
echo "@var.COUNT"
echo "@var.ANOTHER"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// We should have 2 SecretUses:
	// 1. COUNT = "10" (second declaration)
	// 2. ANOTHER = "5" (which should share exprID with first COUNT declaration)
	//
	// The first COUNT = "5" should be deduplicated with ANOTHER = "5"
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses (COUNT='10' and ANOTHER='5'), got %d", len(plan.SecretUses))
	}

	// Verify we have steps
	if len(plan.Steps) == 0 {
		t.Fatal("Plan should have steps")
	}

	step := plan.Steps[0]
	if step.Tree == nil {
		t.Fatal("Step tree should not be nil")
	}

	// We don't need to verify the exact tree structure here
	// The SecretUses count is sufficient to verify deduplication works
}

func TestVarShadowing_DifferentSteps_Works(t *testing.T) {
	// Shadowing should work across different steps (steps are not scopes)

	source := []byte(`
var COUNT = "5"
echo "@var.COUNT"

var COUNT = "10"
echo "@var.COUNT"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Should have 2 steps
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Should have 2 SecretUses (one for each value)
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses, got %d", len(plan.SecretUses))
	}

	// Both commands should have DisplayIDs
	for i, step := range plan.Steps {
		if step.Tree == nil {
			t.Fatalf("Step %d tree should not be nil", i)
		}
		cmd := getCommandArg(step.Tree, "command")
		if !containsDisplayID(cmd) {
			t.Errorf("Step %d command should contain DisplayID, got: %s", i, cmd)
		}
	}
}

func TestVarShadowing_SameValue_SharesExprID(t *testing.T) {
	// Redeclaring with same value should share exprID (deduplication)

	source := []byte(`
var COUNT = "5"
var COUNT = "5"
echo "@var.COUNT"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Should have exactly ONE SecretUse (both declarations share same exprID)
	if len(plan.SecretUses) != 1 {
		t.Errorf("Expected 1 SecretUse (same value deduplicated), got %d", len(plan.SecretUses))
	}
}

func TestVarShadowing_IntermediateUse_UsesCorrectValue(t *testing.T) {
	// Using variable between declarations should use the value at that point

	source := []byte(`
var COUNT = "5"
echo "@var.COUNT"
var COUNT = "10"
echo "@var.COUNT"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Should have 2 steps (each command is a separate step)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Should have 2 SecretUses (one for "5", one for "10")
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses (both values used), got %d", len(plan.SecretUses))
	}

	// Verify both steps have DisplayIDs in their commands
	for i, step := range plan.Steps {
		if step.Tree == nil {
			t.Fatalf("Step %d tree should not be nil", i)
		}
		cmd := getCommandArg(step.Tree, "command")
		if !containsDisplayID(cmd) {
			t.Errorf("Step %d command should contain DisplayID, got: %s", i, cmd)
		}
	}
}

func TestVarShadowing_ExecutionOrder_Preserved(t *testing.T) {
	// Verify that the execution order is preserved correctly
	// First echo uses first value, second echo uses second value

	source := []byte(`
var NAME = "Alice"
echo "@var.NAME"
var NAME = "Bob"
echo "@var.NAME"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))

	// Mark both values as resolved so we can verify DisplayIDs
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Verify we have the right structure (2 steps)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify we have 2 SecretUses (one for each value)
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses, got %d", len(plan.SecretUses))
	}

	// The DisplayIDs should be DIFFERENT (different values)
	// We can verify this by checking that we have 2 distinct DisplayIDs in SecretUses
	displayIDs := make(map[string]bool)
	for _, use := range plan.SecretUses {
		displayIDs[use.DisplayID] = true
	}

	if len(displayIDs) != 2 {
		t.Errorf("Expected 2 distinct DisplayIDs, got %d", len(displayIDs))
	}

	// Verify both steps have DisplayIDs in their commands
	for i, step := range plan.Steps {
		if step.Tree == nil {
			t.Fatalf("Step %d tree should not be nil", i)
		}
		cmd := getCommandArg(step.Tree, "command")
		if !containsDisplayID(cmd) {
			t.Errorf("Step %d command should contain DisplayID, got: %s", i, cmd)
		}
	}
}

func TestVarShadowing_CommandsHaveDifferentDisplayIDs(t *testing.T) {
	// CRITICAL TEST: Verify that commands actually use different DisplayIDs
	// This test will FAIL if Pass 3 does a fresh LookupVariable() instead of using captured exprIDs

	source := []byte(`
var COUNT = "5"
echo "@var.COUNT"
var COUNT = "10"
echo "@var.COUNT"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Should have 2 steps
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Extract the actual command strings
	cmd1 := getCommandArg(plan.Steps[0].Tree, "command")
	cmd2 := getCommandArg(plan.Steps[1].Tree, "command")

	t.Logf("First command:  %s", cmd1)
	t.Logf("Second command: %s", cmd2)

	// The commands should be DIFFERENT (different DisplayIDs)
	if cmd1 == cmd2 {
		t.Errorf("❌ BUG: Both commands have the same DisplayID!")
		t.Errorf("   This means Pass 3 is doing a fresh LookupVariable() instead of using captured exprIDs")
		t.Errorf("   Both commands: %s", cmd1)
		t.Errorf("   Expected: First should use DisplayID for '5', second should use DisplayID for '10'")
	}

	// Verify both contain DisplayIDs
	if !containsDisplayID(cmd1) {
		t.Errorf("First command should contain DisplayID, got: %s", cmd1)
	}
	if !containsDisplayID(cmd2) {
		t.Errorf("Second command should contain DisplayID, got: %s", cmd2)
	}
}

// ========== Helper Functions ==========

func containsDisplayID(s string) bool {
	// DisplayID format: opal:<base64>
	return strings.Contains(s, "opal:")
}
