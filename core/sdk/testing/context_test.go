package testing_test

import (
	"context"
	"testing"
	"time"

	"github.com/aledsdavies/opal/core/sdk"
	sdktesting "github.com/aledsdavies/opal/core/sdk/testing"
)

// TestNewTestContext verifies realistic defaults
func TestNewTestContext(t *testing.T) {
	ctx := sdktesting.NewTestContext()

	// Should have real workdir
	if ctx.Workdir() == "" {
		t.Error("workdir should not be empty")
	}

	// Should have real environment
	environ := ctx.Environ()
	if len(environ) == 0 {
		t.Error("environ should not be empty (should have real env vars)")
	}

	// Should have PATH (common env var)
	if _, hasPath := environ["PATH"]; !hasPath {
		t.Error("environ should include PATH from real environment")
	}
}

// TestWithArg verifies argument setting
func TestWithArg(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithArg("command", "echo hello").
		WithArg("times", int64(3)).
		WithArg("enabled", true)

	if ctx.ArgString("command") != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", ctx.ArgString("command"))
	}
	if ctx.ArgInt("times") != 3 {
		t.Errorf("expected 3, got %d", ctx.ArgInt("times"))
	}
	if !ctx.ArgBool("enabled") {
		t.Error("expected true")
	}
}

// TestWithEnv verifies environment modification
func TestWithEnv(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithEnv("TEST_VAR", "test_value")

	if ctx.Environ()["TEST_VAR"] != "test_value" {
		t.Errorf("expected 'test_value', got %q", ctx.Environ()["TEST_VAR"])
	}
}

// TestWithWorkingDir verifies workdir modification
func TestWithWorkingDir(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithWorkingDir("/custom/path")

	if ctx.Workdir() != "/custom/path" {
		t.Errorf("expected '/custom/path', got %q", ctx.Workdir())
	}
}

// TestExecuteBlockRecording verifies block execution is recorded
func TestExecuteBlockRecording(t *testing.T) {
	ctx := sdktesting.NewTestContext()

	// Execute a block
	steps := []sdk.Step{
		{ID: 1, Commands: []sdk.Command{{Name: "shell"}}},
	}
	_, _ = ctx.ExecuteBlock(steps)

	// Verify it was recorded
	if err := ctx.AssertExecutedBlocks(1); err != nil {
		t.Error(err)
	}

	// Execute another block
	_, _ = ctx.ExecuteBlock(steps)

	// Verify both recorded
	if err := ctx.AssertExecutedBlocks(2); err != nil {
		t.Error(err)
	}
}

// TestAssertNoBlocksExecuted verifies leaf decorator testing
func TestAssertNoBlocksExecuted(t *testing.T) {
	ctx := sdktesting.NewTestContext()

	// Should pass initially
	if err := ctx.AssertNoBlocksExecuted(); err != nil {
		t.Error(err)
	}

	// Execute a block
	ctx.ExecuteBlock([]sdk.Step{})

	// Should fail now
	if err := ctx.AssertNoBlocksExecuted(); err == nil {
		t.Error("expected error after ExecuteBlock call")
	}
}

// TestSimulateBlockFailure verifies failure simulation
func TestSimulateBlockFailure(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithExecuteBlock(sdktesting.SimulateBlockFailure(42))

	exitCode, err := ctx.ExecuteBlock([]sdk.Step{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

// TestSimulateBlockRetries verifies retry simulation
func TestSimulateBlockRetries(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithExecuteBlock(sdktesting.SimulateBlockRetries(2, 1))

	// First call - should fail
	exitCode, _ := ctx.ExecuteBlock([]sdk.Step{})
	if exitCode != 1 {
		t.Errorf("first call: expected exit code 1, got %d", exitCode)
	}

	// Second call - should fail
	exitCode, _ = ctx.ExecuteBlock([]sdk.Step{})
	if exitCode != 1 {
		t.Errorf("second call: expected exit code 1, got %d", exitCode)
	}

	// Third call - should succeed
	exitCode, _ = ctx.ExecuteBlock([]sdk.Step{})
	if exitCode != 0 {
		t.Errorf("third call: expected exit code 0, got %d", exitCode)
	}
}

// TestCountingExecuteBlock verifies invocation counting
func TestCountingExecuteBlock(t *testing.T) {
	counter := 0
	ctx := sdktesting.NewTestContext().
		WithExecuteBlock(sdktesting.CountingExecuteBlock(&counter, sdktesting.SimulateBlockSuccess()))

	// Call multiple times
	ctx.ExecuteBlock([]sdk.Step{})
	ctx.ExecuteBlock([]sdk.Step{})
	ctx.ExecuteBlock([]sdk.Step{})

	if counter != 3 {
		t.Errorf("expected 3 invocations, got %d", counter)
	}
}

// TestWithContextImmutability verifies context wrapping creates new instances
func TestWithContextImmutability(t *testing.T) {
	original := sdktesting.NewTestContext().WithWorkingDir("/original")

	// Wrap with new workdir
	wrapped := original.WithWorkdir("/wrapped")

	// Original should be unchanged
	if original.Workdir() != "/original" {
		t.Errorf("original workdir changed to %q", original.Workdir())
	}

	// Wrapped should have new workdir
	if wrapped.Workdir() != "/wrapped" {
		t.Errorf("wrapped workdir is %q, expected '/wrapped'", wrapped.Workdir())
	}
}

// TestWithEnvironImmutability verifies environment wrapping creates new instances
func TestWithEnvironImmutability(t *testing.T) {
	original := sdktesting.NewTestContext().WithEnv("ORIGINAL", "value")

	// Wrap with new environment
	newEnv := map[string]string{"NEW": "value"}
	wrapped := original.WithEnviron(newEnv)

	// Original should be unchanged
	if _, hasOriginal := original.Environ()["ORIGINAL"]; !hasOriginal {
		t.Error("original environment lost ORIGINAL key")
	}

	// Wrapped should have new environment
	if _, hasNew := wrapped.Environ()["NEW"]; !hasNew {
		t.Error("wrapped environment missing NEW key")
	}
	if _, hasOriginal := wrapped.Environ()["ORIGINAL"]; hasOriginal {
		t.Error("wrapped environment should not have ORIGINAL key")
	}
}

// TestArgDuration verifies duration argument handling
func TestArgDuration(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithArg("timeout", 5*time.Second).
		WithArg("delay", "10s")

	// Duration type
	if ctx.ArgDuration("timeout") != 5*time.Second {
		t.Errorf("expected 5s, got %v", ctx.ArgDuration("timeout"))
	}

	// String parsed as duration
	if ctx.ArgDuration("delay") != 10*time.Second {
		t.Errorf("expected 10s, got %v", ctx.ArgDuration("delay"))
	}

	// Missing arg
	if ctx.ArgDuration("missing") != 0 {
		t.Error("missing arg should return 0")
	}
}

// TestGetExecutedBlock verifies block retrieval
func TestGetExecutedBlock(t *testing.T) {
	ctx := sdktesting.NewTestContext()

	step1 := sdk.Step{ID: 1}
	step2 := sdk.Step{ID: 2}

	ctx.ExecuteBlock([]sdk.Step{step1})
	ctx.ExecuteBlock([]sdk.Step{step2})

	// Get first block
	block, err := ctx.GetExecutedBlock(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(block) != 1 || block[0].ID != 1 {
		t.Error("first block should contain step with ID 1")
	}

	// Get second block
	block, err = ctx.GetExecutedBlock(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(block) != 1 || block[0].ID != 2 {
		t.Error("second block should contain step with ID 2")
	}

	// Out of range
	_, err = ctx.GetExecutedBlock(2)
	if err == nil {
		t.Error("expected error for out of range index")
	}
}

// TestHelperFunctions verifies test helper functions
func TestHelperFunctions(t *testing.T) {
	// NewTestStep
	step := sdktesting.NewTestStep(1,
		sdktesting.NewTestCommand("shell", map[string]interface{}{"command": "echo hi"}),
	)
	if step.ID != 1 {
		t.Errorf("expected ID 1, got %d", step.ID)
	}
	if len(step.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(step.Commands))
	}

	// NewTestCommandWithBlock
	cmd := sdktesting.NewTestCommandWithBlock("retry", map[string]interface{}{"times": 3}, []sdk.Step{step})
	if cmd.Name != "retry" {
		t.Errorf("expected name 'retry', got %q", cmd.Name)
	}
	if len(cmd.Block) != 1 {
		t.Errorf("expected 1 block step, got %d", len(cmd.Block))
	}
}

// TestGoContextPropagation verifies context.Context handling
func TestGoContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCtx := sdktesting.NewTestContext().WithGoContext(ctx)

	if testCtx.Context() != ctx {
		t.Error("context should be the same instance")
	}

	// Test wrapping
	newCtx, newCancel := context.WithTimeout(context.Background(), time.Second)
	defer newCancel()

	wrapped := testCtx.WithContext(newCtx)
	if wrapped.Context() != newCtx {
		t.Error("wrapped context should have new context")
	}
}
