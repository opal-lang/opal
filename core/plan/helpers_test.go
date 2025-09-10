package plan

import (
	"fmt"
	"testing"
)

func TestNewShellStep(t *testing.T) {
	cmd := "echo hello world"
	step := NewShellStep(cmd)

	if step.Type != StepShell {
		t.Errorf("Expected StepShell, got %v", step.Type)
	}

	if step.Description != cmd {
		t.Errorf("Expected description %q, got %q", cmd, step.Description)
	}

	if step.Command != cmd {
		t.Errorf("Expected command %q, got %q", cmd, step.Command)
	}

	if step.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
}

func TestNewDecoratorStep(t *testing.T) {
	name := "timeout"
	stepType := StepDecorator
	step := NewDecoratorStep(name, stepType)

	if step.Type != stepType {
		t.Errorf("Expected %v, got %v", stepType, step.Type)
	}

	if step.Description != "@timeout" {
		t.Errorf("Expected '@timeout', got %q", step.Description)
	}

	if step.Metadata["decorator"] != name {
		t.Errorf("Expected decorator metadata %q, got %q", name, step.Metadata["decorator"])
	}
}

func TestNewErrorStep(t *testing.T) {
	decorator := "parallel"
	err := fmt.Errorf("invalid parameter")
	step := NewErrorStep(decorator, err)

	if step.Type != StepShell {
		t.Errorf("Expected StepShell, got %v", step.Type)
	}

	expectedDesc := "@parallel(<error: invalid parameter>)"
	if step.Description != expectedDesc {
		t.Errorf("Expected %q, got %q", expectedDesc, step.Description)
	}

	if step.Command != "" {
		t.Errorf("Expected empty command, got %q", step.Command)
	}

	if step.Metadata["decorator"] != decorator {
		t.Errorf("Expected decorator metadata %q, got %q", decorator, step.Metadata["decorator"])
	}

	if step.Metadata["error"] != err.Error() {
		t.Errorf("Expected error metadata %q, got %q", err.Error(), step.Metadata["error"])
	}
}

func TestAddMetadata(t *testing.T) {
	step := NewShellStep("echo test")

	// Test adding metadata to step with existing metadata
	AddMetadata(&step, "key1", "value1")
	AddMetadata(&step, "key2", "value2")

	if step.Metadata["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %q", step.Metadata["key1"])
	}

	if step.Metadata["key2"] != "value2" {
		t.Errorf("Expected key2=value2, got %q", step.Metadata["key2"])
	}

	// Test adding metadata to step with nil metadata
	emptyStep := ExecutionStep{}
	AddMetadata(&emptyStep, "test", "value")

	if emptyStep.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}

	if emptyStep.Metadata["test"] != "value" {
		t.Errorf("Expected test=value, got %q", emptyStep.Metadata["test"])
	}
}

func TestSetChildren(t *testing.T) {
	parent := NewDecoratorStep("timeout", StepDecorator)
	child1 := NewShellStep("echo 1")
	child2 := NewShellStep("echo 2")

	children := []ExecutionStep{child1, child2}
	SetChildren(&parent, children)

	if len(parent.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(parent.Children))
	}

	if parent.Children[0].Command != "echo 1" {
		t.Errorf("Expected first child 'echo 1', got %q", parent.Children[0].Command)
	}

	if parent.Children[1].Command != "echo 2" {
		t.Errorf("Expected second child 'echo 2', got %q", parent.Children[1].Command)
	}
}

func TestTruncateCommand(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long command", 10, "this is..."},
		{"exactly ten!", 10, "exactly..."},
		{"a", 3, "a"},
		{"abcd", 3, "..."},
		{"toolong", 5, "to..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := TruncateCommand(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("TruncateCommand(%q, %d) = %q, expected %q",
				tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestTruncateCommand_EdgeCases(t *testing.T) {
	// Test maxLen <= 3
	result := TruncateCommand("toolong", 1)
	if result != "..." {
		t.Errorf("Expected '...' for maxLen=1, got %q", result)
	}

	result = TruncateCommand("toolong", 0)
	if result != "..." {
		t.Errorf("Expected '...' for maxLen=0, got %q", result)
	}

	// Test exactly at boundary
	result = TruncateCommand("abc", 6)
	if result != "abc" {
		t.Errorf("Expected 'abc' (no truncation), got %q", result)
	}
}

func TestHelpersIntegration(t *testing.T) {
	// Test using helpers together to build a complex step
	step := NewDecoratorStep("parallel", StepDecorator)

	// Add metadata
	AddMetadata(&step, "concurrency", "3")
	AddMetadata(&step, "mode", "fail-fast")

	// Create children
	child1 := NewShellStep("task 1")
	child2 := NewShellStep("task 2 with a very long command that should be truncated")
	child2.Command = TruncateCommand(child2.Command, 20)

	children := []ExecutionStep{child1, child2}
	SetChildren(&step, children)

	// Verify the result
	if step.Type != StepDecorator {
		t.Error("Expected StepDecorator")
	}

	if step.Metadata["concurrency"] != "3" {
		t.Error("Expected concurrency metadata")
	}

	if len(step.Children) != 2 {
		t.Error("Expected 2 children")
	}

	if len(step.Children[1].Command) > 20 {
		t.Error("Expected second child command to be truncated")
	}

	if !contains(step.Children[1].Command, "...") {
		t.Error("Expected truncated command to contain ellipsis")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
