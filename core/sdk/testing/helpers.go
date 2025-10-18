package testing

import (
	"context"
	"time"

	"github.com/aledsdavies/opal/core/sdk"
)

// SimulateBlockFailure returns an ExecuteBlock function that fails with given exit code.
// Use this to test how decorators handle block execution failures.
//
// Example:
//
//	ctx := NewTestContext().
//	    WithExecuteBlock(SimulateBlockFailure(42))
//	exitCode, _ := retryHandler(ctx, block)
//	// retryHandler should retry on failure
func SimulateBlockFailure(exitCode int) func([]sdk.Step) (int, error) {
	return func(steps []sdk.Step) (int, error) {
		return exitCode, nil
	}
}

// SimulateBlockSuccess returns an ExecuteBlock function that always succeeds.
// This is the default behavior, but useful for explicit test clarity.
func SimulateBlockSuccess() func([]sdk.Step) (int, error) {
	return func(steps []sdk.Step) (int, error) {
		return 0, nil
	}
}

// SimulateBlockTimeout returns an ExecuteBlock function that times out.
// Use this to test timeout handling in decorators.
//
// Example:
//
//	ctx := NewTestContext().
//	    WithExecuteBlock(SimulateBlockTimeout(5 * time.Second))
//	exitCode, _ := timeoutHandler(ctx, block)
//	// Should return timeout exit code (124)
func SimulateBlockTimeout(duration time.Duration) func([]sdk.Step) (int, error) {
	return func(steps []sdk.Step) (int, error) {
		time.Sleep(duration)
		return 124, context.DeadlineExceeded
	}
}

// SimulateBlockRetries returns an ExecuteBlock function that fails N times then succeeds.
// Use this to test retry logic.
//
// Example:
//
//	ctx := NewTestContext().
//	    WithExecuteBlock(SimulateBlockRetries(2, 1)) // Fail twice, then succeed
//	exitCode, _ := retryHandler(ctx, block)
//	// Should retry and eventually succeed
func SimulateBlockRetries(failCount int, failExitCode int) func([]sdk.Step) (int, error) {
	attempts := 0
	return func(steps []sdk.Step) (int, error) {
		attempts++
		if attempts <= failCount {
			return failExitCode, nil
		}
		return 0, nil
	}
}

// CountingExecuteBlock wraps an ExecuteBlock function and counts invocations.
// Use this to verify retry counts, parallel execution, etc.
//
// Example:
//
//	counter := 0
//	ctx := NewTestContext().
//	    WithExecuteBlock(CountingExecuteBlock(&counter, SimulateBlockSuccess()))
//	retryHandler(ctx, block)
//	// Assert counter == expectedRetries
func CountingExecuteBlock(counter *int, fn func([]sdk.Step) (int, error)) func([]sdk.Step) (int, error) {
	return func(steps []sdk.Step) (int, error) {
		*counter++
		return fn(steps)
	}
}

// NewTestStep creates a test sdk.Step with given ID and commands.
// Helper for building test blocks.
func NewTestStep(id uint64, commands ...sdk.Command) sdk.Step {
	return sdk.Step{
		ID:       id,
		Commands: commands,
	}
}

// NewTestCommand creates a test sdk.Command with given name and args.
// Helper for building test steps.
func NewTestCommand(name string, args map[string]interface{}) sdk.Command {
	if args == nil {
		args = make(map[string]interface{})
	}
	return sdk.Command{
		Name: name,
		Args: args,
	}
}

// NewTestCommandWithBlock creates a test sdk.Command with a block.
// Helper for building nested test structures.
func NewTestCommandWithBlock(name string, args map[string]interface{}, block []sdk.Step) sdk.Command {
	if args == nil {
		args = make(map[string]interface{})
	}
	return sdk.Command{
		Name:  name,
		Args:  args,
		Block: block,
	}
}
