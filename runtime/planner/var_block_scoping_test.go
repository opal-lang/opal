package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"

	_ "github.com/opal-lang/opal/runtime/decorators" // Register decorators for parser
)

// Block Scoping Tests
//
// These tests verify that execution decorator blocks (@retry, @timeout, @parallel, etc.)
// create isolated scopes where:
// 1. Variables declared inside blocks are scoped to that block
// 2. Mutations inside blocks do NOT leak to outer scope
// 3. Parent variables can be READ inside child blocks (flow in)
// 4. Child variables CANNOT escape to parent scope (don't flow out)
//
// This is DIFFERENT from language control blocks (for, if, when) which DO mutate outer scope.
// See SPECIFICATION.md "Scope semantics" for the complete model.

// ========== Basic Isolated Scope Tests ==========

// TestVarBlockScoping_ExecutionDecorator_IsolatedScope tests that
// execution decorator blocks (@retry, @timeout, etc.) create isolated scopes.
func TestVarBlockScoping_ExecutionDecorator_IsolatedScope(t *testing.T) {
	source := `
var COUNT = "5"
@retry {
    var COUNT = "3"
    echo "@var.COUNT"
}
echo "@var.COUNT"
`

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

	// ASSERT: Should have 2 steps (@retry decorator + outer echo)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// ASSERT: First echo (inside @retry) should use COUNT=3
	// ASSERT: Second echo (outside @retry) should use COUNT=5
	// They should have DIFFERENT DisplayIDs

	// First step is @retry decorator with block
	retryStep := plan.Steps[0]
	retryCmd, ok := retryStep.Tree.(*planfmt.CommandNode)
	if !ok || retryCmd.Decorator != "@retry" {
		t.Fatalf("Expected first step to be @retry, got %T with decorator %v", retryStep.Tree, retryCmd)
	}
	if len(retryCmd.Block) != 1 {
		t.Fatalf("Expected @retry block to have 1 step, got %d", len(retryCmd.Block))
	}

	firstCommand := getCommandString(retryCmd.Block[0])
	secondCommand := getCommandString(plan.Steps[1])

	// Extract DisplayIDs from commands
	firstDisplayID := extractDisplayID(firstCommand)
	secondDisplayID := extractDisplayID(secondCommand)

	if firstDisplayID == "" {
		t.Errorf("First command missing DisplayID: %s", firstCommand)
	}
	if secondDisplayID == "" {
		t.Errorf("Second command missing DisplayID: %s", secondCommand)
	}

	// CRITICAL: DisplayIDs should be DIFFERENT (different values)
	if firstDisplayID == secondDisplayID {
		t.Errorf("DisplayIDs should be different (isolated scopes), but both are: %s", firstDisplayID)
		t.Errorf("First command:  %s", firstCommand)
		t.Errorf("Second command: %s", secondCommand)
	}

	// ASSERT: Both values should be in SecretUses (both touched)
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses (both COUNT values touched), got %d", len(plan.SecretUses))
	}

	t.Logf("✓ First echo (inside @retry):  %s", firstCommand)
	t.Logf("✓ Second echo (outside @retry): %s", secondCommand)
	t.Logf("✓ Different DisplayIDs confirm isolated scopes")
}

// TestVarBlockScoping_ExecutionDecorator_NoLeakage tests that variables
// declared inside execution decorator blocks don't leak to outer scope.
func TestVarBlockScoping_ExecutionDecorator_NoLeakage(t *testing.T) {
	source := `
@retry {
    var SECRET = "inside"
}
echo "@var.SECRET"
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan
	_, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})

	// ASSERT: Should fail - SECRET not found in outer scope
	if err == nil {
		t.Fatal("Expected error for variable not found, got nil")
	}

	// ASSERT: Error should mention SECRET
	if !strings.Contains(err.Error(), "SECRET") {
		t.Errorf("Error should mention 'SECRET', got: %v", err)
	}

	// ASSERT: Error should mention undefined variable
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Errorf("Error should mention 'undefined variable', got: %v", err)
	}

	t.Logf("✓ Variable declared in @retry block correctly doesn't leak: %v", err)
}

// TestVarBlockScoping_ExecutionDecorator_ParentReadable tests that
// parent scope variables are accessible inside execution decorator blocks.
func TestVarBlockScoping_ExecutionDecorator_ParentReadable(t *testing.T) {
	source := `
var API_KEY = "parent-key"
@retry {
    echo "@var.API_KEY"
}
`

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

	// ASSERT: Should have 1 step (echo command)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	// ASSERT: Echo inside @retry should use parent's API_KEY
	command := getCommandString(plan.Steps[0])
	if !strings.Contains(command, "opal:") {
		t.Errorf("Command should contain DisplayID, got: %s", command)
	}

	// ASSERT: SecretUses should contain API_KEY
	if len(plan.SecretUses) != 1 {
		t.Errorf("Expected 1 SecretUse (API_KEY), got %d", len(plan.SecretUses))
	}

	t.Logf("✓ Parent variable accessible in @retry block: %s", command)
}

// ========== Nested Blocks Tests ==========

// TestVarBlockScoping_NestedExecutionDecorators tests deeply nested
// execution decorator blocks.
func TestVarBlockScoping_NestedExecutionDecorators(t *testing.T) {
	source := `
var COUNT = "5"
@retry {
    var COUNT = "3"
    @timeout(duration=5s) {
        var COUNT = "1"
        echo "@var.COUNT"
    }
    echo "@var.COUNT"
}
echo "@var.COUNT"
`

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

	// ASSERT: Should have 2 steps (@retry decorator + outer echo)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Navigate the tree structure:
	// Step 0: @retry { @timeout { echo COUNT=1 }; echo COUNT=3 }
	// Step 1: echo COUNT=5

	retryCmd := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if retryCmd.Decorator != "@retry" {
		t.Fatalf("Expected @retry decorator, got %s", retryCmd.Decorator)
	}

	// Inside @retry: @timeout block and echo COUNT=3
	if len(retryCmd.Block) != 2 {
		t.Fatalf("Expected @retry to have 2 steps, got %d", len(retryCmd.Block))
	}

	timeoutCmd := retryCmd.Block[0].Tree.(*planfmt.CommandNode)
	if timeoutCmd.Decorator != "@timeout" {
		t.Fatalf("Expected @timeout decorator, got %s", timeoutCmd.Decorator)
	}

	// Inside @timeout: echo COUNT=1
	if len(timeoutCmd.Block) != 1 {
		t.Fatalf("Expected @timeout to have 1 step, got %d", len(timeoutCmd.Block))
	}

	// ASSERT: Innermost echo uses COUNT=1
	// ASSERT: Middle echo uses COUNT=3
	// ASSERT: Outermost echo uses COUNT=5
	// Three different DisplayIDs

	innermostCommand := getCommandString(timeoutCmd.Block[0]) // echo COUNT=1
	middleCommand := getCommandString(retryCmd.Block[1])      // echo COUNT=3
	outermostCommand := getCommandString(plan.Steps[1])       // echo COUNT=5

	innermostDisplayID := extractDisplayID(innermostCommand)
	middleDisplayID := extractDisplayID(middleCommand)
	outermostDisplayID := extractDisplayID(outermostCommand)

	// ASSERT: All three DisplayIDs should be different
	if innermostDisplayID == middleDisplayID || middleDisplayID == outermostDisplayID || innermostDisplayID == outermostDisplayID {
		t.Errorf("All three DisplayIDs should be different (nested isolated scopes)")
		t.Errorf("Innermost:  %s (COUNT=1)", innermostCommand)
		t.Errorf("Middle:     %s (COUNT=3)", middleCommand)
		t.Errorf("Outermost:  %s (COUNT=5)", outermostCommand)
	}

	// ASSERT: All three values in SecretUses
	if len(plan.SecretUses) != 3 {
		t.Errorf("Expected 3 SecretUses (all COUNT values touched), got %d", len(plan.SecretUses))
	}

	t.Logf("✓ Innermost echo (COUNT=1): %s", innermostCommand)
	t.Logf("✓ Middle echo (COUNT=3):    %s", middleCommand)
	t.Logf("✓ Outermost echo (COUNT=5): %s", outermostCommand)
}

// TestVarBlockScoping_DifferentDecoratorTypes tests that different
// execution decorator types all create isolated scopes.
func TestVarBlockScoping_DifferentDecoratorTypes(t *testing.T) {
	source := `
@retry {
    var A = "retry-scope"
}
@timeout(duration=5s) {
    var B = "timeout-scope"
}
@parallel {
    var C = "parallel-scope"
}
echo "@var.A @var.B @var.C"
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan
	_, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})

	// ASSERT: Should fail - none of A, B, C accessible outside blocks
	if err == nil {
		t.Fatal("Expected error for variables not found, got nil")
	}

	// ASSERT: Error should mention at least one of the variables
	errStr := err.Error()
	if !strings.Contains(errStr, "A") && !strings.Contains(errStr, "B") && !strings.Contains(errStr, "C") {
		t.Errorf("Error should mention one of the variables (A, B, or C), got: %v", err)
	}

	t.Logf("✓ All execution decorator types create isolated scopes: %v", err)
}

// TestVarBlockScoping_SiblingExecutionDecorators tests that sibling
// execution decorator blocks don't interfere.
func TestVarBlockScoping_SiblingExecutionDecorators(t *testing.T) {
	source := `
@retry {
    var COUNT = "3"
    echo "@var.COUNT"
}
@retry {
    var COUNT = "7"
    echo "@var.COUNT"
}
`

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

	// ASSERT: Should have 2 steps (2 @retry decorators)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// ASSERT: First echo uses COUNT=3
	// ASSERT: Second echo uses COUNT=7
	// Different DisplayIDs (independent scopes)

	// Both steps are @retry blocks, navigate into them
	firstCommand := getCommandString(plan.Steps[0])
	secondCommand := getCommandString(plan.Steps[1])

	firstDisplayID := extractDisplayID(firstCommand)
	secondDisplayID := extractDisplayID(secondCommand)

	// ASSERT: DisplayIDs should be different
	if firstDisplayID == secondDisplayID {
		t.Errorf("DisplayIDs should be different (independent sibling scopes)")
		t.Errorf("First:  %s", firstCommand)
		t.Errorf("Second: %s", secondCommand)
	}

	t.Logf("✓ First @retry block (COUNT=3):  %s", firstCommand)
	t.Logf("✓ Second @retry block (COUNT=7): %s", secondCommand)
}

// ========== Edge Cases ==========

// TestVarBlockScoping_NoHoisting_InExecutionDecorator tests that
// no-hoisting rule applies inside execution decorator blocks.
func TestVarBlockScoping_NoHoisting_InExecutionDecorator(t *testing.T) {
	source := `
@retry {
    echo "@var.SECRET"
    var SECRET = "value"
}
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan
	_, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{})

	// ASSERT: Should fail - use before declare
	if err == nil {
		t.Fatal("Expected error for use before declare, got nil")
	}

	// ASSERT: Error should mention SECRET
	if !strings.Contains(err.Error(), "SECRET") {
		t.Errorf("Error should mention 'SECRET', got: %v", err)
	}

	t.Logf("✓ No-hoisting rule applies in @retry block: %v", err)
}

// TestVarBlockScoping_EmptyExecutionDecorator tests that empty
// execution decorator blocks don't cause errors.
func TestVarBlockScoping_EmptyExecutionDecorator(t *testing.T) {
	source := `
var COUNT = "5"
@retry {
}
echo "@var.COUNT"
`

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

	// ASSERT: Should have 2 steps (empty @retry + echo command)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// ASSERT: First step is empty @retry block
	retryCmd := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if retryCmd.Decorator != "@retry" {
		t.Fatalf("Expected @retry decorator, got %s", retryCmd.Decorator)
	}
	if len(retryCmd.Block) != 0 {
		t.Errorf("Expected empty @retry block, got %d steps", len(retryCmd.Block))
	}

	// ASSERT: Second step is echo using COUNT=5 (parent scope)
	command := getCommandString(plan.Steps[1])
	if !strings.Contains(command, "opal:") {
		t.Errorf("Command should contain DisplayID, got: %s", command)
	}

	t.Logf("✓ Empty @retry block doesn't affect parent scope: %s", command)
}

// TestVarBlockScoping_MultipleVariables_InExecutionDecorator tests
// multiple variables in execution decorator block.
func TestVarBlockScoping_MultipleVariables_InExecutionDecorator(t *testing.T) {
	source := `
var A = "outer-a"
var B = "outer-b"
@retry {
    var A = "inner-a"
    var C = "inner-c"
    echo "@var.A @var.B @var.C"
}
echo "@var.A @var.B"
`

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

	// ASSERT: Should have 2 steps (@retry decorator + outer echo)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// ASSERT: First echo: A=inner-a, B=outer-b (inherited), C=inner-c
	// ASSERT: Second echo: A=outer-a (restored), B=outer-b
	// C not accessible outside block (would need third echo to test, but we can verify via SecretUses)

	// First step is @retry block, navigate into it
	firstCommand := getCommandString(plan.Steps[0])
	secondCommand := getCommandString(plan.Steps[1])

	// First command should have 3 DisplayIDs (A, B, C)
	firstDisplayIDs := extractAllDisplayIDs(firstCommand)
	if len(firstDisplayIDs) != 3 {
		t.Errorf("First command should have 3 DisplayIDs (A, B, C), got %d: %s", len(firstDisplayIDs), firstCommand)
	}

	// Second command should have 2 DisplayIDs (A, B)
	secondDisplayIDs := extractAllDisplayIDs(secondCommand)
	if len(secondDisplayIDs) != 2 {
		t.Errorf("Second command should have 2 DisplayIDs (A, B), got %d: %s", len(secondDisplayIDs), secondCommand)
	}

	// A should have different DisplayID in first vs second command (shadowed then restored)
	// B should have same DisplayID in both commands (inherited)

	t.Logf("✓ First echo (inside @retry):  %s", firstCommand)
	t.Logf("✓ Second echo (outside @retry): %s", secondCommand)
}

// ========== Stress Tests ==========

// TestVarBlockScoping_InFunction tests that decorator blocks work in functions.
func TestVarBlockScoping_InFunction(t *testing.T) {
	source := `
fun test {
    var COUNT = "5"
    @retry {
        var COUNT = "3"
        echo "@var.COUNT"
    }
    echo "@var.COUNT"
}
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan
	result, err := PlanNewWithObservability(tree.Events, tree.Tokens, Config{Target: "test"})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	plan := result.Plan

	// ASSERT: Should have 2 steps (@retry decorator + outer echo)
	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// ASSERT: First echo (inside @retry) should use COUNT=3
	// ASSERT: Second echo (outside @retry) should use COUNT=5
	// They should have DIFFERENT DisplayIDs

	// First step is @retry block, navigate into it
	firstCommand := getCommandString(plan.Steps[0])
	secondCommand := getCommandString(plan.Steps[1])

	firstDisplayID := extractDisplayID(firstCommand)
	secondDisplayID := extractDisplayID(secondCommand)

	if firstDisplayID == "" {
		t.Errorf("First command missing DisplayID: %s", firstCommand)
	}
	if secondDisplayID == "" {
		t.Errorf("Second command missing DisplayID: %s", secondCommand)
	}

	// CRITICAL: DisplayIDs should be DIFFERENT (different values)
	if firstDisplayID == secondDisplayID {
		t.Errorf("DisplayIDs should be different (isolated scopes), but both are: %s", firstDisplayID)
		t.Errorf("First command:  %s", firstCommand)
		t.Errorf("Second command: %s", secondCommand)
	}

	// ASSERT: Both values should be in SecretUses (both touched)
	if len(plan.SecretUses) != 2 {
		t.Errorf("Expected 2 SecretUses (both COUNT values touched), got %d", len(plan.SecretUses))
	}

	t.Logf("✓ First echo (inside @retry):  %s", firstCommand)
	t.Logf("✓ Second echo (outside @retry): %s", secondCommand)
	t.Logf("✓ Function decorator blocks work correctly")
}

// TestVarBlockScoping_DeeplyNested tests that deeply nested
// blocks work correctly (5 levels).
func TestVarBlockScoping_DeeplyNested(t *testing.T) {
	source := `
var COUNT = "5"
@retry {
    @parallel {
        @timeout(duration=5s) {
            @retry {
                @timeout(duration=5s) {
                    echo "@var.COUNT"
                }
            }
        }
    }
}
`

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

	// ASSERT: Should have 1 step (@retry decorator at root)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	// ASSERT: Echo at depth 5 can access COUNT from root
	// getCommandString will recursively navigate into blocks
	command := getCommandString(plan.Steps[0])
	if !strings.Contains(command, "opal:") {
		t.Errorf("Command should contain DisplayID, got: %s", command)
	}

	t.Logf("✓ Deeply nested (5 levels) can access root variable: %s", command)
}

// ========== Helper Functions ==========

// getCommandString extracts the command string from a step's execution tree.
// Handles both direct shell commands and decorator blocks (navigates into .Block[]).
func getCommandString(step planfmt.Step) string {
	if step.Tree == nil {
		return ""
	}

	cmd, ok := step.Tree.(*planfmt.CommandNode)
	if !ok {
		return ""
	}

	// If this is a shell command, return the command arg
	if cmd.Decorator == "@shell" {
		return getCommandArg(step.Tree, "command")
	}

	// If this is a decorator block, navigate into the first step in the block
	if len(cmd.Block) > 0 {
		return getCommandString(cmd.Block[0])
	}

	return ""
}

// extractDisplayID extracts the first DisplayID from a command string
func extractDisplayID(command string) string {
	// DisplayID format: opal:XXXXXXXXXXXXXXXXXXXX (22 chars base64url)
	start := strings.Index(command, "opal:")
	if start == -1 {
		return ""
	}

	// Extract the DisplayID (opal: + 22 chars)
	end := start + 5 + 22 // "opal:" (5) + base64url (22)
	if end > len(command) {
		end = len(command)
	}

	return command[start:end]
}

// extractAllDisplayIDs extracts all DisplayIDs from a command string
func extractAllDisplayIDs(command string) []string {
	var displayIDs []string
	remaining := command

	for {
		start := strings.Index(remaining, "opal:")
		if start == -1 {
			break
		}

		// Extract the DisplayID
		end := start + 5 + 22 // "opal:" (5) + base64url (22)
		if end > len(remaining) {
			end = len(remaining)
		}

		displayID := remaining[start:end]
		displayIDs = append(displayIDs, displayID)

		// Continue searching after this DisplayID
		remaining = remaining[end:]
	}

	return displayIDs
}
