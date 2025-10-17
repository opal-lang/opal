package planner_test

import (
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
)

// TestSimpleShellCommand tests converting a simple shell command to @shell decorator
func TestSimpleShellCommand(t *testing.T) {
	source := []byte(`echo "Hello, World!"`)

	// Parse
	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan (script mode - no target)
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify plan structure
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if len(step.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(step.Commands))
	}

	cmd := step.Commands[0]
	if cmd.Decorator != "@shell" {
		t.Errorf("Expected @shell decorator, got %q", cmd.Decorator)
	}

	// Check command argument
	if len(cmd.Args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(cmd.Args))
	}

	arg := cmd.Args[0]
	if arg.Key != "command" {
		t.Errorf("Expected arg key 'command', got %q", arg.Key)
	}

	if arg.Val.Kind != planfmt.ValueString {
		t.Errorf("Expected ValueString, got %v", arg.Val.Kind)
	}

	expectedCmd := `echo "Hello, World!"`
	if arg.Val.Str != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, arg.Val.Str)
	}
}

// TestMultipleShellCommands tests multiple newline-separated commands
func TestMultipleShellCommands(t *testing.T) {
	source := []byte(`echo "First"
echo "Second"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 2 steps (newline-separated)
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify first step
	if len(plan.Steps[0].Commands) != 1 {
		t.Errorf("Step 0: expected 1 command, got %d", len(plan.Steps[0].Commands))
	}
	if plan.Steps[0].Commands[0].Args[0].Val.Str != `echo "First"` {
		t.Errorf("Step 0: wrong command: %q", plan.Steps[0].Commands[0].Args[0].Val.Str)
	}

	// Verify second step
	if len(plan.Steps[1].Commands) != 1 {
		t.Errorf("Step 1: expected 1 command, got %d", len(plan.Steps[1].Commands))
	}
	if plan.Steps[1].Commands[0].Args[0].Val.Str != `echo "Second"` {
		t.Errorf("Step 1: wrong command: %q", plan.Steps[1].Commands[0].Args[0].Val.Str)
	}
}

// TestFunctionDefinition tests parsing a function definition
func TestFunctionDefinition(t *testing.T) {
	source := []byte(`fun hello = echo "Hello, World!"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Command mode - target "hello"
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (the function body)
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Verify it's the shell command from the function body
	if len(plan.Steps[0].Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(plan.Steps[0].Commands))
	}

	cmd := plan.Steps[0].Commands[0]
	if cmd.Decorator != "@shell" {
		t.Errorf("Expected @shell, got %q", cmd.Decorator)
	}

	if cmd.Args[0].Val.Str != `echo "Hello, World!"` {
		t.Errorf("Wrong command: %q", cmd.Args[0].Val.Str)
	}
}

// TestCommandModeTargetSelection tests finding a specific function
func TestCommandModeTargetSelection(t *testing.T) {
	source := []byte(`fun hello = echo "Hello"
fun goodbye = echo "Goodbye"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Target "goodbye" - should only plan that function
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "goodbye",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (only goodbye function body)
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Verify it's the goodbye command
	if plan.Steps[0].Commands[0].Args[0].Val.Str != `echo "Goodbye"` {
		t.Errorf("Wrong command: %q", plan.Steps[0].Commands[0].Args[0].Val.Str)
	}
}

// TestScriptModeFullExecution tests executing entire file (no target)
func TestScriptModeFullExecution(t *testing.T) {
	source := []byte(`echo "Top level"
fun hello = echo "In function"
echo "Another top level"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Script mode - no target, execute all top-level commands
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 2 steps (two top-level echo commands, function is skipped)
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify first command
	if plan.Steps[0].Commands[0].Args[0].Val.Str != `echo "Top level"` {
		t.Errorf("Step 0: wrong command: %q", plan.Steps[0].Commands[0].Args[0].Val.Str)
	}

	// Verify second command
	if plan.Steps[1].Commands[0].Args[0].Val.Str != `echo "Another top level"` {
		t.Errorf("Step 1: wrong command: %q", plan.Steps[1].Commands[0].Args[0].Val.Str)
	}
}

// TestEmptyPlan tests empty input
func TestEmptyPlan(t *testing.T) {
	source := []byte(``)

	tree := parser.Parse(source)

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Empty plan is valid
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(plan.Steps))
	}
}

// TestStepIDUniqueness tests that all step IDs are unique
func TestStepIDUniqueness(t *testing.T) {
	source := []byte(`echo "First"
echo "Second"
echo "Third"`)

	tree := parser.Parse(source)

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Collect all step IDs
	seen := make(map[uint64]bool)
	for _, step := range plan.Steps {
		if seen[step.ID] {
			t.Errorf("Duplicate step ID: %d", step.ID)
		}
		seen[step.ID] = true
	}

	// All IDs should be unique
	if len(seen) != len(plan.Steps) {
		t.Errorf("Expected %d unique IDs, got %d", len(plan.Steps), len(seen))
	}
}

// TestArgSorting tests that args are sorted by key
func TestArgSorting(t *testing.T) {
	source := []byte(`echo "Hello"`)

	tree := parser.Parse(source)

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify args are sorted (for determinism)
	cmd := plan.Steps[0].Commands[0]
	for i := 1; i < len(cmd.Args); i++ {
		if cmd.Args[i-1].Key >= cmd.Args[i].Key {
			t.Errorf("Args not sorted: %q >= %q", cmd.Args[i-1].Key, cmd.Args[i].Key)
		}
	}
}

// TestTargetNotFound tests error when target function doesn't exist
func TestTargetNotFound(t *testing.T) {
	source := []byte(`fun hello = echo "Hello"`)

	tree := parser.Parse(source)

	// Target a function that doesn't exist
	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "nonexistent",
	})

	if err == nil {
		t.Fatal("Expected error for nonexistent target, got nil")
	}

	// Error message should contain key information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "command not found: nonexistent") {
		t.Errorf("Expected 'command not found: nonexistent', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Define the function") {
		t.Errorf("Expected suggestion in error message, got: %s", errMsg)
	}
}

// TestTargetNotFoundWithSuggestion tests "did you mean" suggestions
func TestTargetNotFoundWithSuggestion(t *testing.T) {
	source := []byte(`fun hello = echo "Hello"
fun deploy = echo "Deploying"`)

	tree := parser.Parse(source)

	// Target a function with a typo
	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "helo", // Missing 'l'
	})

	if err == nil {
		t.Fatal("Expected error for nonexistent target, got nil")
	}

	// Should suggest "hello"
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Did you mean 'hello'?") {
		t.Errorf("Expected 'Did you mean' suggestion, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Available commands:") {
		t.Errorf("Expected available commands list, got: %s", errMsg)
	}
}

func TestPlanShellCommandWithOperators(t *testing.T) {
	// Test that commands with operators are grouped into a single step
	source := []byte(`fun test = echo "A" && echo "B"`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "test",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have ONE step with TWO commands
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	if len(plan.Steps[0].Commands) != 2 {
		t.Errorf("Expected 2 commands in step, got %d", len(plan.Steps[0].Commands))
	}

	// First command should have && operator
	if plan.Steps[0].Commands[0].Operator != "&&" {
		t.Errorf("Expected first command to have && operator, got %q", plan.Steps[0].Commands[0].Operator)
	}

	// Second command should have empty operator (last in step)
	if plan.Steps[0].Commands[1].Operator != "" {
		t.Errorf("Expected second command to have empty operator, got %q", plan.Steps[0].Commands[1].Operator)
	}
}

func TestPlanMultipleSteps(t *testing.T) {
	// Test that newline-separated commands in SCRIPT MODE create separate steps
	source := []byte("echo \"A\"\necho \"B\"")

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have TWO steps, each with ONE command
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	if len(plan.Steps[0].Commands) != 1 {
		t.Errorf("Expected first step to have 1 command, got %d", len(plan.Steps[0].Commands))
	}

	if len(plan.Steps[1].Commands) != 1 {
		t.Errorf("Expected second step to have 1 command, got %d", len(plan.Steps[1].Commands))
	}
}

func TestPlanFunctionWithOperatorsAndNewline(t *testing.T) {
	// Test: fun hello = echo "A" && echo "B" || echo "C"\necho "D"
	// In SCRIPT MODE (no target), should produce: 2 steps, 4 commands total
	// Step 1: 3 commands (A && B || C) - top-level operators in one step
	// Step 2: 1 command (D) - newline creates new step
	source := []byte(`echo "A" && echo "B" || echo "C"
echo "D"`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have TWO steps
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Step 1: THREE commands with operators
	if len(plan.Steps[0].Commands) != 3 {
		t.Errorf("Expected step 0 to have 3 commands, got %d", len(plan.Steps[0].Commands))
	}

	// Verify step 1 commands and operators
	if plan.Steps[0].Commands[0].Args[0].Val.Str != `echo "A"` {
		t.Errorf("Step 0, cmd 0: expected 'echo \"A\"', got %q", plan.Steps[0].Commands[0].Args[0].Val.Str)
	}
	if plan.Steps[0].Commands[0].Operator != "&&" {
		t.Errorf("Step 0, cmd 0: expected && operator, got %q", plan.Steps[0].Commands[0].Operator)
	}

	if plan.Steps[0].Commands[1].Args[0].Val.Str != `echo "B"` {
		t.Errorf("Step 0, cmd 1: expected 'echo \"B\"', got %q", plan.Steps[0].Commands[1].Args[0].Val.Str)
	}
	if plan.Steps[0].Commands[1].Operator != "||" {
		t.Errorf("Step 0, cmd 1: expected || operator, got %q", plan.Steps[0].Commands[1].Operator)
	}

	if plan.Steps[0].Commands[2].Args[0].Val.Str != `echo "C"` {
		t.Errorf("Step 0, cmd 2: expected 'echo \"C\"', got %q", plan.Steps[0].Commands[2].Args[0].Val.Str)
	}
	if plan.Steps[0].Commands[2].Operator != "" {
		t.Errorf("Step 0, cmd 2: expected empty operator (last in step), got %q", plan.Steps[0].Commands[2].Operator)
	}

	// Step 2: ONE command (top-level echo "D")
	if len(plan.Steps[1].Commands) != 1 {
		t.Errorf("Expected step 1 to have 1 command, got %d", len(plan.Steps[1].Commands))
	}

	if plan.Steps[1].Commands[0].Args[0].Val.Str != `echo "D"` {
		t.Errorf("Step 1, cmd 0: expected 'echo \"D\"', got %q", plan.Steps[1].Commands[0].Args[0].Val.Str)
	}
	if plan.Steps[1].Commands[0].Operator != "" {
		t.Errorf("Step 1, cmd 0: expected empty operator, got %q", plan.Steps[1].Commands[0].Operator)
	}
}
