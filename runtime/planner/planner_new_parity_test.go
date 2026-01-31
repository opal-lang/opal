package planner

import (
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
)

// getPlanDecorator extracts the decorator name from a CommandNode
func getPlanDecorator(tree planfmt.ExecutionNode) string {
	if tree == nil {
		return ""
	}
	cmd, ok := tree.(*planfmt.CommandNode)
	if !ok {
		return ""
	}
	return cmd.Decorator
}

// parseAndPlan is a helper that parses source and runs the new planner
func parseAndPlan(t *testing.T, source, target string) (*planfmt.Plan, error) {
	t.Helper()

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	return PlanNew(tree.Events, tree.Tokens, Config{
		Target: target,
	})
}

// =============================================================================
// Shell Command Tests
// =============================================================================

// TestParity_SimpleShellCommand tests basic echo command planning
func TestParity_SimpleShellCommand(t *testing.T) {
	source := `echo "Hello, World!"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify plan structure
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if step.Tree == nil {
		t.Fatal("Expected tree, got nil")
	}

	// Tree should be a CommandNode with @shell decorator
	if getPlanDecorator(step.Tree) != "@shell" {
		t.Errorf("Expected @shell decorator, got %q", getPlanDecorator(step.Tree))
	}

	// Check command argument
	expectedCmd := `echo "Hello, World!"`
	if getCommandArg(step.Tree, "command") != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, getCommandArg(step.Tree, "command"))
	}
}

// TestParity_ShellCommandWithDashes tests commands with flag arguments
func TestParity_ShellCommandWithDashes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wc with -l flag",
			input:    `wc -l`,
			expected: `wc -l`,
		},
		{
			name:     "echo with -n flag",
			input:    `echo -n "hello"`,
			expected: `echo -n "hello"`,
		},
		{
			name:     "ls with -la flags",
			input:    `ls -la`,
			expected: `ls -la`,
		},
		{
			name:     "kubectl with -f flag",
			input:    `kubectl apply -f deployment.yaml`,
			expected: `kubectl apply -f deployment.yaml`,
		},
		{
			name:     "grep with -v flag",
			input:    `grep -v "pattern"`,
			expected: `grep -v "pattern"`,
		},
		{
			name:     "double dash --",
			input:    `echo -- "end of options"`,
			expected: `echo -- "end of options"`,
		},
		{
			name:     "long flag --file",
			input:    `command --file config.yaml`,
			expected: `command --file config.yaml`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := parseAndPlan(t, tt.input, "")
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			if len(plan.Steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
			}

			step := plan.Steps[0]
			if getPlanDecorator(step.Tree) != "@shell" {
				t.Errorf("Expected @shell decorator, got %q", getPlanDecorator(step.Tree))
			}

			actual := getCommandArg(step.Tree, "command")
			if actual != tt.expected {
				t.Errorf("Expected command %q, got %q", tt.expected, actual)
			}
		})
	}
}

// TestParity_MultipleShellCommands tests planning multiple commands
func TestParity_MultipleShellCommands(t *testing.T) {
	source := `echo "First"
echo "Second"
echo "Third"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 3 steps
	if len(plan.Steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(plan.Steps))
	}

	// Verify each step has correct command
	expectedCommands := []string{`echo "First"`, `echo "Second"`, `echo "Third"`}
	for i, step := range plan.Steps {
		if getPlanDecorator(step.Tree) != "@shell" {
			t.Errorf("Step %d: Expected @shell decorator, got %q", i, getPlanDecorator(step.Tree))
		}

		actual := getCommandArg(step.Tree, "command")
		if actual != expectedCommands[i] {
			t.Errorf("Step %d: Expected command %q, got %q", i, expectedCommands[i], actual)
		}
	}
}

// =============================================================================
// Function Mode Tests
// =============================================================================

// TestParity_FunctionDefinition tests planning function definitions
func TestParity_FunctionDefinition(t *testing.T) {
	source := `fun hello = echo "Hello"
fun world = echo "World"`

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan in script mode with only functions - should produce empty plan
	// (functions are only executed when targeted in command mode)
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// In script mode with only functions, plan should be empty (valid but no steps)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps in script mode with only functions, got %d", len(plan.Steps))
	}
}

// TestParity_CommandModeTargetSelection tests targeting a specific function
func TestParity_CommandModeTargetSelection(t *testing.T) {
	source := `fun hello = echo "Hello"
fun world = echo "World"
fun deploy = kubectl apply -f k8s/`

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan targeting "hello" function
	plan, err := PlanNew(tree.Events, tree.Tokens, Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should target hello function
	if plan.Target != "hello" {
		t.Errorf("Expected target 'hello', got %q", plan.Target)
	}

	// Should have 1 step (the echo from hello)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step for hello function, got %d", len(plan.Steps))
	}

	// Verify it's the right command
	cmd := getCommandArg(plan.Steps[0].Tree, "command")
	expectedCmd := `echo "Hello"`
	if cmd != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, cmd)
	}
}

// TestParity_TargetNotFound tests error when target doesn't exist
func TestParity_TargetNotFound(t *testing.T) {
	source := `fun hello = echo "Hello"`

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	_, err := PlanNew(tree.Events, tree.Tokens, Config{
		Target: "nonexistent",
	})

	if err == nil {
		t.Fatal("Expected error for non-existent target")
	}
}

// =============================================================================
// Plan Structure Tests
// =============================================================================

// TestParity_EmptyPlan tests planning empty source
func TestParity_EmptyPlan(t *testing.T) {
	source := ``

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps for empty plan, got %d", len(plan.Steps))
	}
}

// TestParity_StepIDUniqueness tests that step IDs are unique
func TestParity_StepIDUniqueness(t *testing.T) {
	source := `echo "First"
echo "Second"
echo "Third"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(plan.Steps))
	}

	// Verify IDs are sequential starting from 1
	for i, step := range plan.Steps {
		expectedID := uint64(i + 1)
		if step.ID != expectedID {
			t.Errorf("Step %d: Expected ID %d, got %d", i, expectedID, step.ID)
		}
	}
}

// =============================================================================
// Operator Tests
// =============================================================================

// TestParity_ShellCommandWithOperators tests && and || operators
func TestParity_ShellCommandWithOperators(t *testing.T) {
	source := `echo "first" && echo "second"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Operators chain commands into a single step with AndNode
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step for chained commands, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]

	// Check if it's an AndNode (representing &&)
	andNode, ok := step.Tree.(*planfmt.AndNode)
	if !ok {
		t.Fatalf("Expected AndNode for && chain, got %T", step.Tree)
	}

	// Verify both sides are CommandNodes
	if andNode.Left == nil || andNode.Right == nil {
		t.Error("AndNode should have both left and right children")
	}

	// Verify left and right are @shell commands
	_, leftOk := andNode.Left.(*planfmt.CommandNode)
	_, rightOk := andNode.Right.(*planfmt.CommandNode)
	if !leftOk || !rightOk {
		t.Errorf("AndNode children should be CommandNodes: left=%T, right=%T", andNode.Left, andNode.Right)
	}
}

// TestParity_MultipleSteps tests multiple independent steps
func TestParity_MultipleSteps(t *testing.T) {
	source := `echo "step1"
echo "step2"
echo "step3"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(plan.Steps))
	}

	// Each step should be a CommandNode
	for i, step := range plan.Steps {
		if getPlanDecorator(step.Tree) != "@shell" {
			t.Errorf("Step %d: Expected @shell decorator, got %q", i, getPlanDecorator(step.Tree))
		}
	}
}

// =============================================================================
// Contract/Plan Salt Tests
// =============================================================================

// TestParity_PlanSalt tests that plan has salt for contract verification
func TestParity_PlanSalt(t *testing.T) {
	source := `echo "test"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify plan has salt
	if len(plan.PlanSalt) == 0 {
		t.Error("Expected plan salt to be set")
	}
}

// =============================================================================
// Blocked Tests (Skipped)
// =============================================================================

// TestParity_RedirectOperators tests > and >> operators
// BLOCKED: Redirect operators not implemented in IR builder
func TestParity_RedirectOperators(t *testing.T) {
	t.Skip("TODO: blocked by redirect operators not implemented in IR builder")

	source := `echo "output" > file.txt`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have redirect information
	t.Logf("Plan steps: %+v", plan.Steps)
}

// TestParity_RedirectWithChaining tests redirects with operators
// BLOCKED: Redirect operators not implemented in IR builder
func TestParity_RedirectWithChaining(t *testing.T) {
	t.Skip("TODO: blocked by redirect operators not implemented in IR builder")

	source := `echo "first" > file.txt && echo "second"`

	plan, err := parseAndPlan(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	t.Logf("Plan steps: %+v", plan.Steps)
}
