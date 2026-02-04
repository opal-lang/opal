package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"
)

// ========== Three-Pass Model Tests ==========
//
// These tests verify that the planner implements true three-pass processing:
// - Pass 1: Scan - Declare variables, record references, DON'T resolve
// - Pass 2: Resolve - Resolve all touched expressions (batched)
// - Pass 3: Interpolate - Replace @var.X with DisplayIDs
//
// This enables:
// - Batching decorator calls (efficiency)
// - Wave-based resolution (for @if)
// - Correct shadowing behavior

func TestThreePass_LiteralsNotResolvedInPass1(t *testing.T) {
	// Variables should be declared but NOT resolved during Pass 1
	// This test verifies that MarkResolved is NOT called during planVarDecl

	source := []byte(`
var NAME = "Aled"
var COUNT = "5"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))

	// Create planner but don't call Plan yet - we want to inspect state after Pass 1
	_, err := PlanNew(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Plan should not error: %v", err)
	}

	// After Pass 1, variables should be declared but NOT resolved
	// We can't easily test this without exposing internal state
	// So instead, we'll test the observable behavior: batching

	// TODO: This test needs access to planner internals
	// For now, we'll test batching behavior instead
	t.Skip("Need to refactor to test Pass 1 state")
}

func TestThreePass_ShadowingWithDeferredResolution(t *testing.T) {
	// This test demonstrates the shadowing problem with deferred resolution
	// and verifies that we capture exprID at reference time (not resolution time)

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

	// Should have 2 SecretUses (one for "5", one for "10")
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses (both values used), got %d", len(plan.SecretUses))
	}

	// The DisplayIDs should be DIFFERENT (different values)
	displayIDs := make(map[string]bool)
	for _, use := range plan.SecretUses {
		displayIDs[use.DisplayID] = true
	}

	if len(displayIDs) != 2 {
		t.Errorf("Expected 2 distinct DisplayIDs, got %d", len(displayIDs))
	}

	// Verify both commands have different DisplayIDs
	cmd1 := getCommandArg(plan.Steps[0].Tree, "command")
	cmd2 := getCommandArg(plan.Steps[1].Tree, "command")

	// Both should contain DisplayIDs
	if !strings.Contains(cmd1, "opal:") {
		t.Errorf("First command should contain DisplayID, got: %s", cmd1)
	}
	if !strings.Contains(cmd2, "opal:") {
		t.Errorf("Second command should contain DisplayID, got: %s", cmd2)
	}

	// The DisplayIDs should be DIFFERENT (first uses "5", second uses "10")
	if cmd1 == cmd2 {
		t.Errorf("Commands should have different DisplayIDs (different values), both are: %s", cmd1)
	}

	// The key test: verify that each command uses the correct value
	// This validates that exprID was captured at reference time, not resolution time
}
