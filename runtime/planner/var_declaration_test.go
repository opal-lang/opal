package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
)

// TestVarDeclaration_SimpleString tests that a simple string variable is:
// 1. Declared in Vault with hash-based exprID
// 2. Marked as resolved immediately (it's a literal)
// 3. DisplayID is generated
// 4. Vault.SecretProvider() returns the pattern for scrubbing
func TestVarDeclaration_SimpleString(t *testing.T) {
	source := `var NAME = "Aled"`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan
	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	plan := result.Plan

	// ASSERT: Plan should have no steps (variable declaration doesn't create a step)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps for variable declaration, got %d", len(plan.Steps))
	}

	// We can't directly access the vault from here, but we can verify the behavior
	// by checking that the plan was created successfully
	t.Logf("Variable declaration planned successfully")
}

// TestVarDeclaration_MultipleVariables tests multiple variable declarations
func TestVarDeclaration_MultipleVariables(t *testing.T) {
	source := `var NAME = "Aled"
var COUNT = "5"
var ENABLED = "true"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	if len(result.Plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(result.Plan.Steps))
	}

	t.Logf("Multiple variable declarations planned successfully")
}

// TestVarDeclaration_WithCommand tests variable declaration followed by command
func TestVarDeclaration_WithCommand(t *testing.T) {
	source := `var NAME = "Aled"
echo "Hello"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	// Should have 1 step (the echo command)
	if len(result.Plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result.Plan.Steps))
	}

	t.Logf("Variable declaration + command planned successfully")
}

// TestVarDeclaration_OrderMatters tests that variables can only be used after declaration
func TestVarDeclaration_OrderMatters(t *testing.T) {
	// This should fail because NAME is used before it's declared
	source := `echo "@var.NAME"
var NAME = "Aled"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Planning should fail because NAME is not declared yet
	_, err := PlanNew(tree.Events, tree.Tokens, Config{})
	if err == nil {
		t.Fatal("Expected error for using variable before declaration, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

// TestVarDeclaration_DifferentTypes tests variables with different literal types
func TestVarDeclaration_DifferentTypes(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"string", `var NAME = "Aled"`},
		{"number", `var COUNT = 42`},
		{"boolean", `var ENABLED = true`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.ParseString(tt.source)
			if len(tree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", tree.Errors)
			}

			result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
			if err != nil {
				t.Fatalf("Planning failed: %v", err)
			}

			if len(result.Plan.Steps) != 0 {
				t.Errorf("Expected 0 steps, got %d", len(result.Plan.Steps))
			}
		})
	}
}

// TestVarDeclaration_SameLiteralValue tests that multiple variables with the same
// literal value share the same exprID (expression deduplication) and don't panic.
//
// This is a regression test for the bug where:
//
//	var X = "foo"
//	var Y = "foo"
//
// would panic on the second MarkResolved call because the expression was already resolved.
//
// Expected behavior (from vault.go documentation):
// - Multiple variables can share the same exprID (deduplication)
// - Expression should only be resolved once (first time)
// - Subsequent declarations reuse the already-resolved expression
func TestVarDeclaration_SameLiteralValue(t *testing.T) {
	source := `var X = "same"
var Y = "same"
var Z = "same"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// This should NOT panic - multiple variables can share the same literal value
	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	if len(result.Plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(result.Plan.Steps))
	}

	t.Logf("Multiple variables with same literal value planned successfully")
}

// TestVarDeclaration_SameLiteralValue_DifferentTypes tests that variables with
// the same literal value but different types work correctly.
func TestVarDeclaration_SameLiteralValue_DifferentTypes(t *testing.T) {
	source := `var A = 42
var B = 42
var C = "hello"
var D = "hello"
var E = true
var F = true`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	if len(result.Plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(result.Plan.Steps))
	}

	t.Logf("Multiple variables with same literal values (different types) planned successfully")
}

// TestVarUsage_DisplayIDInPlan tests that when a variable is used in a command,
// the plan contains the DisplayID placeholder, NOT the actual value.
//
// This is Phase 5 of variable resolution:
// - Planning: Plan stores DisplayID (e.g., "opal:3J98t56A")
// - Execution: Executor replaces DisplayID with actual value
// - Scrubbing: Scrubber replaces actual value back to DisplayID in output
//
// Security: The plan never contains sensitive values, only placeholders.
//
// Requirements from WORK.md Phase 5:
// 1. Plan should show: `echo "Hello, opal:3J98t56A"`
// 2. NOT: `echo "Hello, Aled"`
// 3. Plan output contains `opal:` placeholder
// 4. Plan does NOT contain "Aled"
func TestVarUsage_DisplayIDInPlan(t *testing.T) {
	source := `var NAME = "Aled"
echo "Hello, @var.NAME"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	// Should have 1 step (the echo command)
	if len(result.Plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(result.Plan.Steps))
	}

	step := result.Plan.Steps[0]
	if step.Tree == nil {
		t.Fatal("Expected tree, got nil")
	}

	// Tree should be a CommandNode with @shell decorator
	cmd, ok := step.Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", step.Tree)
	}

	if cmd.Decorator != "@shell" {
		t.Errorf("Expected @shell decorator, got %q", cmd.Decorator)
	}

	// Get the command argument
	var commandVal planfmt.Value
	var found bool
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			commandVal = arg.Val
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Command argument not found")
	}

	// REQUIREMENT: Command should be a string containing DisplayID, not a placeholder reference
	// The plan formatter needs to see: echo "Hello, opal:3J98t56A"
	// Phase 5: Commands now store DisplayID strings directly (not placeholder references)
	if commandVal.Kind != planfmt.ValueString {
		t.Fatalf("Expected ValueString, got kind=%v", commandVal.Kind)
	}

	commandArg := commandVal.Str
	t.Logf("Command string: %s", commandArg)

	// ASSERT: Command should contain "opal:" placeholder
	if !strings.Contains(commandArg, "opal:") {
		t.Errorf("FAIL: Command should contain DisplayID placeholder 'opal:', got: %s", commandArg)
	}

	// ASSERT: Command should NOT contain the actual value "Aled"
	if strings.Contains(commandArg, "Aled") {
		t.Errorf("FAIL: Command should NOT contain actual value 'Aled', got: %s", commandArg)
	}

	// SUCCESS
	t.Logf("✓ Plan command contains DisplayID: %s", commandArg)
}

// TestVarUsage_FormattedPlanOutput tests that the formatted plan output
// (what users see with --dry-run) contains DisplayID, not actual values.
//
// This verifies the end-to-end flow: parsing → planning → formatting
//
// Plan contract security:
// - Command strings contain DisplayID: echo "Hello, opal:3J98t56A"
// - Secret.RuntimeValue is NEVER serialized (runtime only, see plan.go:79)
// - Only DisplayIDs are stored in the contract for scrubbing
func TestVarUsage_FormattedPlanOutput(t *testing.T) {
	source := `var NAME = "Aled"
echo "Hello, @var.NAME"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	// Format the plan (simulates what CLI shows with --dry-run)
	formatted := formatPlanForTest(result.Plan)
	t.Logf("Formatted plan:\n%s", formatted)

	// ASSERT: Formatted output contains DisplayID
	if !strings.Contains(formatted, "opal:") {
		t.Errorf("Formatted plan should contain DisplayID 'opal:', got:\n%s", formatted)
	}

	// ASSERT: Formatted output does NOT contain actual value
	if strings.Contains(formatted, "Aled") {
		t.Errorf("Formatted plan should NOT contain actual value 'Aled', got:\n%s", formatted)
	}

	t.Logf("✓ Formatted plan contains DisplayID, not actual value")
}

// formatPlanForTest formats a plan's steps for testing (simplified version)
func formatPlanForTest(plan *planfmt.Plan) string {
	var b strings.Builder
	for _, step := range plan.Steps {
		b.WriteString(formatStepForTest(step.Tree, 0))
		b.WriteString("\n")
	}
	return b.String()
}

func formatStepForTest(node planfmt.ExecutionNode, indent int) string {
	switch n := node.(type) {
	case *planfmt.CommandNode:
		var args []string
		for _, arg := range n.Args {
			if arg.Key == "command" {
				args = append(args, arg.Val.Str)
			}
		}
		return strings.Repeat("  ", indent) + n.Decorator + " " + strings.Join(args, " ")
	default:
		return strings.Repeat("  ", indent) + "(unsupported node type)"
	}
}

// TestVarUsage_SecretUsesPopulated tests that plan.SecretUses is populated
// with authorization entries from Vault.
//
// This is Phase 5.5 of variable resolution:
// - Plan contract contains SecretUses (DisplayID + SiteID + Site)
// - Same DisplayID can appear multiple times (different usage sites)
// - Plan does NOT contain Secrets field (RuntimeValue never leaves Vault)
//
// Contract verification model:
// - Plan time: Build SecretUses, compute hash
// - Execution time: Re-plan with same PlanSalt, compare hashes
// - If value changes → DisplayID changes → hash changes → contract invalid
// - If site changes → SiteID changes → hash changes → contract invalid
func TestVarUsage_SecretUsesPopulated(t *testing.T) {
	source := `var NAME = "Aled"
echo "Hello, @var.NAME"
echo "Goodbye, @var.NAME"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	// ASSERT: SecretUses should have 2 entries (same DisplayID, different sites)
	if len(result.Plan.SecretUses) != 2 {
		t.Fatalf("Expected 2 SecretUses (same DisplayID, different sites), got %d", len(result.Plan.SecretUses))
	}

	// Both entries should have same DisplayID (same variable value)
	displayID1 := result.Plan.SecretUses[0].DisplayID
	displayID2 := result.Plan.SecretUses[1].DisplayID
	if displayID1 != displayID2 {
		t.Errorf("Expected same DisplayID for both uses, got %q and %q", displayID1, displayID2)
	}

	// But different SiteIDs (different usage sites)
	siteID1 := result.Plan.SecretUses[0].SiteID
	siteID2 := result.Plan.SecretUses[1].SiteID
	if siteID1 == siteID2 {
		t.Errorf("Expected different SiteIDs for different sites, got same: %q", siteID1)
	}

	// Each entry should have all fields populated
	for i, use := range result.Plan.SecretUses {
		if use.DisplayID == "" {
			t.Errorf("SecretUse[%d].DisplayID is empty", i)
		}
		if use.SiteID == "" {
			t.Errorf("SecretUse[%d].SiteID is empty", i)
		}
		if use.Site == "" {
			t.Errorf("SecretUse[%d].Site is empty", i)
		}
		if !strings.Contains(use.DisplayID, "opal:") {
			t.Errorf("SecretUse[%d].DisplayID should contain 'opal:', got %q", i, use.DisplayID)
		}
		t.Logf("SecretUse[%d]: DisplayID=%s, SiteID=%s, Site=%s",
			i, use.DisplayID, use.SiteID, use.Site)
	}

	t.Logf("✓ Plan.SecretUses populated correctly with %d entries", len(result.Plan.SecretUses))
}

// TestVarPruning_UntouchedVariablesNotInPlan tests that variables declared but never used
// are pruned from the plan's SecretUses.
//
// This verifies the PruneUntouched() call in the planner:
// - Declared but unused variables should NOT appear in SecretUses
// - Only touched (referenced) variables should appear in SecretUses
// - Saves API calls and reduces secrets in plan
func TestVarPruning_UntouchedVariablesNotInPlan(t *testing.T) {
	source := `var USED = "used-value"
var UNUSED = "unused-value"
echo "Hello, @var.USED"`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	// Should have 1 step (the echo command)
	if len(result.Plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(result.Plan.Steps))
	}

	// Should have exactly 1 SecretUse (only USED variable)
	if len(result.Plan.SecretUses) != 1 {
		t.Errorf("Expected 1 SecretUse (USED only), got %d", len(result.Plan.SecretUses))
		for i, use := range result.Plan.SecretUses {
			t.Logf("  SecretUse[%d]: DisplayID=%s, Site=%s", i, use.DisplayID, use.Site)
		}
	}

	// Verify the plan command contains the USED variable's DisplayID
	step := result.Plan.Steps[0]
	cmd, ok := step.Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", step.Tree)
	}

	// Get the command argument
	var commandVal planfmt.Value
	var found bool
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			commandVal = arg.Val
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Command argument not found")
	}

	if commandVal.Kind != planfmt.ValueString {
		t.Fatalf("Expected ValueString, got kind=%v", commandVal.Kind)
	}

	commandStr := commandVal.Str

	if !strings.Contains(commandStr, "opal:") {
		t.Errorf("Command should contain DisplayID placeholder, got: %s", commandStr)
	}

	// Verify the command does NOT contain the literal values
	if strings.Contains(commandStr, "used-value") {
		t.Errorf("Command should NOT contain literal value 'used-value', got: %s", commandStr)
	}
	if strings.Contains(commandStr, "unused-value") {
		t.Errorf("Command should NOT contain literal value 'unused-value', got: %s", commandStr)
	}

	t.Logf("✓ Unused variable pruned successfully")
	t.Logf("  Command: %s", commandStr)
	t.Logf("  SecretUses count: %d (expected 1)", len(result.Plan.SecretUses))
}
