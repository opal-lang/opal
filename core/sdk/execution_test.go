package sdk_test

import (
	"testing"

	"github.com/opal-lang/opal/core/sdk"
)

// TestStep_Construction verifies Step can be constructed with a tree
func TestStep_Construction(t *testing.T) {
	step := sdk.Step{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "shell",
			Args: map[string]interface{}{
				"command": "echo hello",
			},
		},
	}

	if step.ID != 1 {
		t.Errorf("expected ID 1, got %d", step.ID)
	}
	if step.Tree == nil {
		t.Fatal("expected tree to be non-nil")
	}
	cmd, ok := step.Tree.(*sdk.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", step.Tree)
	}
	if cmd.Name != "shell" {
		t.Errorf("expected command name 'shell', got %q", cmd.Name)
	}
}

// TestCommandNode_WithBlock verifies CommandNode can have nested steps
func TestCommandNode_WithBlock(t *testing.T) {
	cmd := &sdk.CommandNode{
		Name: "retry",
		Args: map[string]interface{}{
			"times": int64(3),
		},
		Block: []sdk.Step{
			{
				ID: 2,
				Tree: &sdk.CommandNode{
					Name: "shell",
					Args: map[string]interface{}{"command": "echo nested"},
				},
			},
		},
	}

	if cmd.Name != "retry" {
		t.Errorf("expected name 'retry', got %q", cmd.Name)
	}
	if len(cmd.Block) != 1 {
		t.Errorf("expected 1 block step, got %d", len(cmd.Block))
	}
	if cmd.Block[0].ID != 2 {
		t.Errorf("expected block step ID 2, got %d", cmd.Block[0].ID)
	}
}

// TestCommandNode_NoBlock verifies CommandNode can have empty block (leaf decorator)
func TestCommandNode_NoBlock(t *testing.T) {
	cmd := &sdk.CommandNode{
		Name: "shell",
		Args: map[string]interface{}{
			"command": "echo hello",
		},
		Block: []sdk.Step{}, // Empty block for leaf decorator
	}

	if len(cmd.Block) != 0 {
		t.Errorf("expected empty block, got %d steps", len(cmd.Block))
	}
}

// TestAndNode_Structure verifies AndNode structure
func TestAndNode_Structure(t *testing.T) {
	node := &sdk.AndNode{
		Left: &sdk.CommandNode{
			Name: "shell",
			Args: map[string]interface{}{"command": "echo first"},
		},
		Right: &sdk.CommandNode{
			Name: "shell",
			Args: map[string]interface{}{"command": "echo second"},
		},
	}

	if node.Left == nil {
		t.Error("expected Left to be non-nil")
	}
	if node.Right == nil {
		t.Error("expected Right to be non-nil")
	}
}

// TestPipelineNode_Structure verifies PipelineNode structure
func TestPipelineNode_Structure(t *testing.T) {
	node := &sdk.PipelineNode{
		Commands: []sdk.TreeNode{
			&sdk.CommandNode{
				Name: "shell",
				Args: map[string]interface{}{"command": "echo hello"},
			},
			&sdk.CommandNode{
				Name: "shell",
				Args: map[string]interface{}{"command": "grep hello"},
			},
		},
	}

	if len(node.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(node.Commands))
	}
}

// TestExecutionContext_Interface verifies ExecutionContext is an interface
func TestExecutionContext_Interface(t *testing.T) {
	// This test just verifies the interface compiles
	// Actual implementation will be in runtime/executor
	var _ sdk.ExecutionContext = nil // nil is valid for interface type
}

// TestExecutionHandler_Signature verifies ExecutionHandler signature
func TestExecutionHandler_Signature(t *testing.T) {
	// Verify we can create a handler with the correct signature
	handler := func(ctx sdk.ExecutionContext, block []sdk.Step) (int, error) {
		// Leaf decorator - ignores block
		if len(block) > 0 {
			t.Error("leaf decorator should receive empty block")
		}
		return 0, nil
	}

	// Call with empty block (leaf decorator)
	exitCode, err := handler(nil, []sdk.Step{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// TestValueHandler_Signature verifies ValueHandler signature
func TestValueHandler_Signature(t *testing.T) {
	// Verify we can create a value handler with the correct signature
	handler := func(ctx sdk.ExecutionContext) (interface{}, error) {
		return "resolved_value", nil
	}

	// Call handler
	value, err := handler(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "resolved_value" {
		t.Errorf("expected 'resolved_value', got %v", value)
	}
}

// TestStep_NestedBlocks verifies deep nesting works
func TestStep_NestedBlocks(t *testing.T) {
	// @retry(3) { @timeout(30s) { @shell("echo nested") } }
	step := sdk.Step{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "retry",
			Args: map[string]interface{}{"times": int64(3)},
			Block: []sdk.Step{
				{
					ID: 2,
					Tree: &sdk.CommandNode{
						Name: "timeout",
						Args: map[string]interface{}{"duration": "30s"},
						Block: []sdk.Step{
							{
								ID: 3,
								Tree: &sdk.CommandNode{
									Name: "shell",
									Args: map[string]interface{}{"command": "echo nested"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Verify structure
	retryCmd, ok := step.Tree.(*sdk.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", step.Tree)
	}
	if retryCmd.Name != "retry" {
		t.Errorf("expected 'retry', got %q", retryCmd.Name)
	}
	if len(retryCmd.Block) != 1 {
		t.Fatalf("expected 1 block step in retry, got %d", len(retryCmd.Block))
	}

	timeoutStep := retryCmd.Block[0]
	timeoutCmd, ok := timeoutStep.Tree.(*sdk.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", timeoutStep.Tree)
	}
	if timeoutCmd.Name != "timeout" {
		t.Errorf("expected 'timeout', got %q", timeoutCmd.Name)
	}
	if len(timeoutCmd.Block) != 1 {
		t.Fatalf("expected 1 block step in timeout, got %d", len(timeoutCmd.Block))
	}

	shellStep := timeoutCmd.Block[0]
	shellCmd, ok := shellStep.Tree.(*sdk.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", shellStep.Tree)
	}
	if shellCmd.Name != "shell" {
		t.Errorf("expected 'shell', got %q", shellCmd.Name)
	}
}
