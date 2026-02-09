package executor

import (
	"context"
	"testing"
	"time"

	"github.com/opal-lang/opal/core/sdk"
	_ "github.com/opal-lang/opal/runtime/decorators"
	"github.com/stretchr/testify/assert"
)

// TestContextCancellationStopsExecution verifies that cancelling the context
// stops execution immediately without waiting for commands to complete.
func TestContextCancellationStopsExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a long-running command (sleep 10 seconds)
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "@shell",
			Args: map[string]interface{}{
				"command": "sleep 10",
			},
		},
	}}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should stop quickly (< 1s), not wait for full 10s
	assert.Less(t, duration, 1*time.Second, "execution should stop quickly after cancellation")

	// Should return non-zero exit code (command was interrupted)
	assert.NotEqual(t, 0, result.ExitCode, "cancelled execution should return non-zero exit code")

	// Error should be nil (cancellation is not an error from Execute's perspective)
	assert.NoError(t, err, "Execute should not return error on cancellation")
}

// TestTimeoutPropagatesThroughRedirects verifies that context timeout
// is respected even when executing redirects.
func TestTimeoutPropagatesThroughRedirects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	outPath := t.TempDir() + "/test-redirect-timeout.txt"

	// Redirect with infinite output
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.RedirectNode{
			Source: &sdk.CommandNode{
				Name: "@shell",
				Args: map[string]interface{}{
					"command": "yes", // Infinite output
				},
			},
			Sink: &sdk.FsPathSink{Path: outPath},
			Mode: sdk.RedirectOverwrite,
		},
	}}

	start := time.Now()
	result, err := Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should timeout after ~100ms
	assert.Less(t, duration, 200*time.Millisecond, "should timeout quickly")
	assert.NotEqual(t, 0, result.ExitCode, "timed out execution should return non-zero")
	assert.NoError(t, err)
}

// TestPipelineCancellationStopsAllCommands verifies that cancelling a pipeline
// stops all commands in the pipeline, not just the first one.
func TestPipelineCancellationStopsAllCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Pipeline with multiple long-running commands
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.PipelineNode{
			Commands: []sdk.TreeNode{
				&sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
				&sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
				&sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
			},
		},
	}}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// All commands should stop quickly (< 1s total, not 30s)
	assert.Less(t, duration, 1*time.Second, "all pipeline commands should stop after cancellation")
}

// TestNestedDecoratorCancellation verifies that cancellation propagates
// through nested decorators (e.g., @retry { @timeout { @shell } }).
func TestNestedDecoratorCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Nested decorators: @retry { @shell { sleep 10 } }
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "@retry",
			Args: map[string]interface{}{
				"times": int64(5),
			},
			Block: []sdk.Step{{
				ID: 2,
				Tree: &sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
			}},
		},
	}}

	start := time.Now()
	result, err := Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should respect outer timeout, not retry 5 times
	assert.Less(t, duration, 200*time.Millisecond, "should respect outer timeout")
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NoError(t, err)
}

// TestCancellationDuringExecuteBlock verifies that cancellation works
// when decorators call ExecuteBlock (nested execution).
func TestCancellationDuringExecuteBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Decorator with block that contains long-running command
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "@shell",
			Args: map[string]interface{}{
				"command": "echo 'before'",
			},
			Block: []sdk.Step{{
				ID: 2,
				Tree: &sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
			}},
		},
	}}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should stop quickly
	assert.Less(t, duration, 1*time.Second, "ExecuteBlock should respect cancellation")
}

// TestMultipleCancellations verifies that calling cancel multiple times
// is safe and idempotent.
func TestMultipleCancellations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "@shell",
			Args: map[string]interface{}{
				"command": "sleep 5",
			},
		},
	}}

	// Cancel multiple times
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
		cancel() // Second call should be safe
		cancel() // Third call should be safe
	}()

	start := time.Now()
	result, err := Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	assert.Less(t, duration, 1*time.Second)
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NoError(t, err)
}

// TestCancellationWithSequenceOperator verifies that cancellation works
// with sequence operators (;).
func TestCancellationWithSequenceOperator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Sequence: cmd1 ; cmd2 ; cmd3 (all long-running)
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.SequenceNode{
			Nodes: []sdk.TreeNode{
				&sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
				&sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
				&sdk.CommandNode{
					Name: "@shell",
					Args: map[string]interface{}{
						"command": "sleep 10",
					},
				},
			},
		},
	}}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should stop quickly, not run all 3 commands
	assert.Less(t, duration, 1*time.Second, "sequence should stop after cancellation")
}

// TestCancellationWithAndOperator verifies that cancellation works
// with AND operators (&&).
func TestCancellationWithAndOperator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// AND: cmd1 && cmd2 (both long-running)
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.AndNode{
			Left: &sdk.CommandNode{
				Name: "@shell",
				Args: map[string]interface{}{
					"command": "sleep 10",
				},
			},
			Right: &sdk.CommandNode{
				Name: "@shell",
				Args: map[string]interface{}{
					"command": "sleep 10",
				},
			},
		},
	}}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should stop quickly
	assert.Less(t, duration, 1*time.Second, "AND operator should respect cancellation")
}

// TestCancellationWithOrOperator verifies that cancellation works
// with OR operators (||).
func TestCancellationWithOrOperator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// OR: cmd1 || cmd2 (both long-running)
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.OrNode{
			Left: &sdk.CommandNode{
				Name: "@shell",
				Args: map[string]interface{}{
					"command": "sleep 10",
				},
			},
			Right: &sdk.CommandNode{
				Name: "@shell",
				Args: map[string]interface{}{
					"command": "sleep 10",
				},
			},
		},
	}}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should stop quickly
	assert.Less(t, duration, 1*time.Second, "OR operator should respect cancellation")
}

// TestDeadlineExceeded verifies that context deadline is properly detected.
func TestDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond))
	defer cancel()

	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "@shell",
			Args: map[string]interface{}{
				"command": "sleep 10",
			},
		},
	}}

	start := time.Now()
	result, err := Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should stop after deadline
	assert.Less(t, duration, 500*time.Millisecond, "should stop after deadline")
	assert.NotEqual(t, 0, result.ExitCode)
	assert.NoError(t, err)

	// Context should report deadline exceeded
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

// TestAlreadyCancelledContext verifies that passing an already-cancelled
// context doesn't execute anything.
func TestAlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.CommandNode{
			Name: "@shell",
			Args: map[string]interface{}{
				"command": "echo 'should not run'",
			},
		},
	}}

	start := time.Now()
	result, err := Execute(ctx, steps, Config{}, testVault())
	duration := time.Since(start)

	// Should return immediately
	assert.Less(t, duration, 100*time.Millisecond, "should return immediately with cancelled context")
	assert.NotEqual(t, 0, result.ExitCode, "should return non-zero for cancelled context")
	assert.NoError(t, err)
}
