package sdk_test

import (
	"testing"

	"github.com/aledsdavies/opal/core/sdk"
)

// TestStep_Construction verifies Step can be constructed
func TestStep_Construction(t *testing.T) {
	step := sdk.Step{
		ID: 1,
		Commands: []sdk.Command{
			{
				Name: "shell",
				Args: map[string]interface{}{
					"command": "echo hello",
				},
			},
		},
	}

	if step.ID != 1 {
		t.Errorf("expected ID 1, got %d", step.ID)
	}
	if len(step.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(step.Commands))
	}
	if step.Commands[0].Name != "shell" {
		t.Errorf("expected command name 'shell', got %q", step.Commands[0].Name)
	}
}

// TestCommand_WithBlock verifies Command can have nested steps
func TestCommand_WithBlock(t *testing.T) {
	cmd := sdk.Command{
		Name: "retry",
		Args: map[string]interface{}{
			"times": int64(3),
		},
		Block: []sdk.Step{
			{
				ID: 2,
				Commands: []sdk.Command{
					{Name: "shell", Args: map[string]interface{}{"command": "echo nested"}},
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

// TestCommand_NoBlock verifies Command can have empty block (leaf decorator)
func TestCommand_NoBlock(t *testing.T) {
	cmd := sdk.Command{
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

// TestCommand_WithOperator verifies Command can have operator for chaining
func TestCommand_WithOperator(t *testing.T) {
	cmd := sdk.Command{
		Name:     "shell",
		Args:     map[string]interface{}{"command": "echo first"},
		Operator: "&&",
	}

	if cmd.Operator != "&&" {
		t.Errorf("expected operator '&&', got %q", cmd.Operator)
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
		Commands: []sdk.Command{
			{
				Name: "retry",
				Args: map[string]interface{}{"times": int64(3)},
				Block: []sdk.Step{
					{
						ID: 2,
						Commands: []sdk.Command{
							{
								Name: "timeout",
								Args: map[string]interface{}{"duration": "30s"},
								Block: []sdk.Step{
									{
										ID: 3,
										Commands: []sdk.Command{
											{
												Name: "shell",
												Args: map[string]interface{}{"command": "echo nested"},
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

	// Verify structure
	if len(step.Commands) != 1 {
		t.Fatalf("expected 1 top-level command, got %d", len(step.Commands))
	}

	retryCmd := step.Commands[0]
	if retryCmd.Name != "retry" {
		t.Errorf("expected 'retry', got %q", retryCmd.Name)
	}
	if len(retryCmd.Block) != 1 {
		t.Fatalf("expected 1 block step in retry, got %d", len(retryCmd.Block))
	}

	timeoutStep := retryCmd.Block[0]
	if len(timeoutStep.Commands) != 1 {
		t.Fatalf("expected 1 command in timeout step, got %d", len(timeoutStep.Commands))
	}

	timeoutCmd := timeoutStep.Commands[0]
	if timeoutCmd.Name != "timeout" {
		t.Errorf("expected 'timeout', got %q", timeoutCmd.Name)
	}
	if len(timeoutCmd.Block) != 1 {
		t.Fatalf("expected 1 block step in timeout, got %d", len(timeoutCmd.Block))
	}

	shellStep := timeoutCmd.Block[0]
	if len(shellStep.Commands) != 1 {
		t.Fatalf("expected 1 command in shell step, got %d", len(shellStep.Commands))
	}

	shellCmd := shellStep.Commands[0]
	if shellCmd.Name != "shell" {
		t.Errorf("expected 'shell', got %q", shellCmd.Name)
	}
}
