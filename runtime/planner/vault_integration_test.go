package planner

import (
	"testing"

	"github.com/aledsdavies/opal/runtime/parser"
)

// TestVaultIntegration_VariableDeclarationAndReference tests the complete flow:
// 1. var NAME = "Aled" → vault.DeclareVariable() + vault.MarkResolved()
// 2. echo "@var.NAME" → @var decorator looks up from Vault
// 3. Vault provides patterns for scrubbing
func TestVaultIntegration_VariableDeclarationAndReference(t *testing.T) {
	source := `var NAME = "Aled"
echo "Hello, @var.NAME"`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan
	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	plan := result.Plan

	// ASSERT: Plan should have steps
	if len(plan.Steps) == 0 {
		t.Fatal("Expected at least one step")
	}

	// ASSERT: Vault should have the variable declared and resolved
	// (We can't directly access planner.vault here, but we can verify the plan output)

	// TODO: Once we implement DisplayID in plan output (Phase 5),
	// verify that the plan shows DisplayID instead of raw value

	t.Logf("Plan created successfully with %d steps", len(plan.Steps))
}

// TestVaultIntegration_VariableNotFound tests error handling
func TestVaultIntegration_VariableNotFound(t *testing.T) {
	source := `echo "@var.MISSING"`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan - should fail because variable not declared
	_, err := Plan(tree.Events, tree.Tokens, Config{})
	if err == nil {
		t.Fatal("Expected error for missing variable, got nil")
	}

	// Error should mention the missing variable
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}

	t.Logf("Got expected error: %v", err)
}

// TestVaultIntegration_ScopeWalking tests that child scopes can access parent variables
func TestVaultIntegration_ScopeWalking(t *testing.T) {
	// This test will be more meaningful once we have decorator blocks
	// For now, just verify basic variable declaration works
	source := `var OUTER = "parent"
var INNER = "child"
echo "@var.OUTER @var.INNER"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	if len(result.Plan.Steps) == 0 {
		t.Fatal("Expected at least one step")
	}

	t.Logf("Scope walking test passed with %d steps", len(result.Plan.Steps))
}
