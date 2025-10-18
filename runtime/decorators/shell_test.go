package decorators

import (
	"testing"

	"github.com/aledsdavies/opal/core/sdk"
	sdktesting "github.com/aledsdavies/opal/core/sdk/testing"
	"github.com/aledsdavies/opal/core/types"
)

// TestShellDecorator_SimpleCommand tests basic command execution
func TestShellDecorator_SimpleCommand(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithArg("command", "echo hello")

	exitCode, err := shellHandler(ctx, []sdk.Step{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", exitCode)
	}

	// Verify no blocks were executed (leaf decorator)
	if err := ctx.AssertNoBlocksExecuted(); err != nil {
		t.Error(err)
	}
}

// TestShellDecorator_FailingCommand tests non-zero exit codes
func TestShellDecorator_FailingCommand(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithArg("command", "exit 42")

	exitCode, err := shellHandler(ctx, []sdk.Step{})
	// Exit code should be 42, no error
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got: %d", exitCode)
	}
}

// TestShellDecorator_UsesContextWorkdir tests that context workdir is used
func TestShellDecorator_UsesContextWorkdir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	ctx := sdktesting.NewTestContext().
		WithArg("command", "pwd").
		WithWorkingDir(tmpDir)

	exitCode, err := shellHandler(ctx, []sdk.Step{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", exitCode)
	}

	// Command should have run in tmpDir
	// (Output verification would require capturing stdout)
}

// TestShellDecorator_UsesContextEnviron tests that context environ is used
func TestShellDecorator_UsesContextEnviron(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithArg("command", "echo $TEST_SHELL_VAR").
		WithEnv("TEST_SHELL_VAR", "from_context")

	exitCode, err := shellHandler(ctx, []sdk.Step{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", exitCode)
	}

	// Command should see TEST_SHELL_VAR=from_context
	// (Output verification would require capturing stdout)
}

// TestShellDecorator_MissingCommandArg tests error when command arg is missing
func TestShellDecorator_MissingCommandArg(t *testing.T) {
	ctx := sdktesting.NewTestContext()
	// No "command" arg

	exitCode, err := shellHandler(ctx, []sdk.Step{})

	// Should return error
	if err == nil {
		t.Error("expected error for missing command arg, got nil")
	}
	if exitCode != 127 {
		t.Errorf("expected exit code 127 for missing command, got: %d", exitCode)
	}
}

// TestShellDecorator_RejectsBlock tests that @shell rejects blocks
func TestShellDecorator_RejectsBlock(t *testing.T) {
	ctx := sdktesting.NewTestContext().
		WithArg("command", "echo hello")

	// Pass a non-empty block (should be rejected)
	block := []sdk.Step{
		sdktesting.NewTestStep(1, sdktesting.NewTestCommand("shell", nil)),
	}

	exitCode, err := shellHandler(ctx, block)

	// Should return error
	if err == nil {
		t.Error("expected error for block, got nil")
	}
	if exitCode != 127 {
		t.Errorf("expected exit code 127, got: %d", exitCode)
	}
}

// TestShellDecorator_Registered tests that @shell is registered in global registry
func TestShellDecorator_Registered(t *testing.T) {
	// Verify @shell is registered
	handler, kind, exists := types.Global().GetSDKHandler("shell")
	if !exists {
		t.Fatal("@shell should be registered")
	}

	// Verify it's an execution decorator
	if kind != types.DecoratorKindExecution {
		t.Errorf("expected execution decorator, got kind %v", kind)
	}

	// Verify handler is correct type
	if handler == nil {
		t.Error("handler should not be nil")
	}
}
