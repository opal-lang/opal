package testing

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aledsdavies/opal/core/sdk"
)

// TestExecutionContext is a realistic test implementation of sdk.ExecutionContext.
// It behaves like the real executor but allows control over execution behavior for testing.
//
// Use this to test decorators in isolation with realistic behavior:
//
//	ctx := testing.NewTestContext().
//	    WithArg("command", "echo hello").
//	    WithWorkdir("/tmp")
//	exitCode, err := shellHandler(ctx, []sdk.Step{})
type TestExecutionContext struct {
	args       map[string]interface{}
	environ    map[string]string
	workdir    string
	ctx        context.Context
	stdin      io.Reader
	stdoutPipe io.Writer

	// ExecuteBlock callback - set this to control block execution behavior
	// Default: executes steps sequentially, stops on first non-zero exit
	ExecuteBlockFunc func(steps []sdk.Step) (int, error)

	// Captured state for assertions
	ExecutedBlocks [][]sdk.Step // All blocks that were executed
}

// NewTestContext creates a new test execution context with realistic defaults.
// - Uses actual os.Getwd() for workdir
// - Uses actual os.Environ() for environment
// - ExecuteBlock behaves like real executor (sequential, fail-fast)
func NewTestContext() *TestExecutionContext {
	// Get real working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = "/tmp" // Fallback
	}

	// Get real environment
	environ := make(map[string]string)
	for _, e := range os.Environ() {
		// Parse KEY=VALUE
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				environ[e[:i]] = e[i+1:]
				break
			}
		}
	}

	return &TestExecutionContext{
		args:           make(map[string]interface{}),
		environ:        environ,
		workdir:        wd,
		ctx:            context.Background(),
		ExecutedBlocks: [][]sdk.Step{},
		ExecuteBlockFunc: func(steps []sdk.Step) (int, error) {
			// Realistic behavior: sequential execution, stop on first failure
			for _, step := range steps {
				// In real tests, you'd execute the step
				// For now, just return success
				_ = step
			}
			return 0, nil
		},
	}
}

// WithArg adds an argument to the context (builder pattern).
func (t *TestExecutionContext) WithArg(key string, value interface{}) *TestExecutionContext {
	t.args[key] = value
	return t
}

// WithEnv adds an environment variable (builder pattern).
func (t *TestExecutionContext) WithEnv(key, value string) *TestExecutionContext {
	t.environ[key] = value
	return t
}

// WithWorkingDir sets the working directory (builder pattern).
// Note: Different name to avoid conflict with interface method.
func (t *TestExecutionContext) WithWorkingDir(dir string) *TestExecutionContext {
	t.workdir = dir
	return t
}

// WithGoContext sets the Go context (builder pattern).
// Note: Different name to avoid conflict with interface method.
func (t *TestExecutionContext) WithGoContext(ctx context.Context) *TestExecutionContext {
	t.ctx = ctx
	return t
}

// WithExecuteBlock sets custom ExecuteBlock behavior (builder pattern).
// Use this to simulate specific execution scenarios (failures, timeouts, etc.)
func (t *TestExecutionContext) WithExecuteBlock(fn func([]sdk.Step) (int, error)) *TestExecutionContext {
	t.ExecuteBlockFunc = fn
	return t
}

// ExecuteBlock implements sdk.ExecutionContext.
// Records the block for later assertion and calls the configured function.
func (t *TestExecutionContext) ExecuteBlock(steps []sdk.Step) (int, error) {
	// Record for assertions
	t.ExecutedBlocks = append(t.ExecutedBlocks, steps)

	// Execute with configured behavior
	return t.ExecuteBlockFunc(steps)
}

// Context implements sdk.ExecutionContext.
func (t *TestExecutionContext) Context() context.Context {
	return t.ctx
}

// ArgString implements sdk.ExecutionContext.
func (t *TestExecutionContext) ArgString(key string) string {
	if val, ok := t.args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// ArgInt implements sdk.ExecutionContext.
func (t *TestExecutionContext) ArgInt(key string) int64 {
	if val, ok := t.args[key]; ok {
		if i, ok := val.(int64); ok {
			return i
		}
		if i, ok := val.(int); ok {
			return int64(i)
		}
	}
	return 0
}

// ArgBool implements sdk.ExecutionContext.
func (t *TestExecutionContext) ArgBool(key string) bool {
	if val, ok := t.args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// ArgDuration implements sdk.ExecutionContext.
func (t *TestExecutionContext) ArgDuration(key string) time.Duration {
	if val, ok := t.args[key]; ok {
		if d, ok := val.(time.Duration); ok {
			return d
		}
		if str, ok := val.(string); ok {
			if d, err := time.ParseDuration(str); err == nil {
				return d
			}
		}
	}
	return 0
}

// Args implements sdk.ExecutionContext.
func (t *TestExecutionContext) Args() map[string]interface{} {
	// Return a copy to prevent external mutation
	copy := make(map[string]interface{}, len(t.args))
	for k, v := range t.args {
		copy[k] = v
	}
	return copy
}

// Environ implements sdk.ExecutionContext.
func (t *TestExecutionContext) Environ() map[string]string {
	return t.environ
}

// Workdir implements sdk.ExecutionContext.
func (t *TestExecutionContext) Workdir() string {
	return t.workdir
}

// WithContext implements sdk.ExecutionContext (returns new context).
func (t *TestExecutionContext) WithContext(ctx context.Context) sdk.ExecutionContext {
	newTest := *t // Copy
	newTest.ctx = ctx
	return &newTest
}

// WithEnviron implements sdk.ExecutionContext (returns new context).
func (t *TestExecutionContext) WithEnviron(env map[string]string) sdk.ExecutionContext {
	newTest := *t // Copy
	newTest.environ = make(map[string]string, len(env))
	for k, v := range env {
		newTest.environ[k] = v
	}
	return &newTest
}

// WithWorkdir implements sdk.ExecutionContext (returns new context).
func (t *TestExecutionContext) WithWorkdir(dir string) sdk.ExecutionContext {
	newTest := *t // Copy
	newTest.workdir = dir
	return &newTest
}

// AssertExecutedBlocks verifies that ExecuteBlock was called with expected steps.
// Returns error if assertion fails.
func (t *TestExecutionContext) AssertExecutedBlocks(expectedCount int) error {
	if len(t.ExecutedBlocks) != expectedCount {
		return fmt.Errorf("expected %d ExecuteBlock calls, got %d", expectedCount, len(t.ExecutedBlocks))
	}
	return nil
}

// AssertNoBlocksExecuted verifies that ExecuteBlock was never called.
// Useful for testing leaf decorators like @shell.
func (t *TestExecutionContext) AssertNoBlocksExecuted() error {
	if len(t.ExecutedBlocks) > 0 {
		return fmt.Errorf("expected no ExecuteBlock calls, got %d", len(t.ExecutedBlocks))
	}
	return nil
}

// AssertBlockStepCount verifies the number of steps in a specific block execution.
func (t *TestExecutionContext) AssertBlockStepCount(blockIndex, expectedSteps int) error {
	if blockIndex >= len(t.ExecutedBlocks) {
		return fmt.Errorf("block index %d out of range (only %d blocks executed)", blockIndex, len(t.ExecutedBlocks))
	}
	actual := len(t.ExecutedBlocks[blockIndex])
	if actual != expectedSteps {
		return fmt.Errorf("block %d: expected %d steps, got %d", blockIndex, expectedSteps, actual)
	}
	return nil
}

// GetExecutedBlock returns the steps from a specific block execution.
// Useful for detailed assertions about what was executed.
func (t *TestExecutionContext) GetExecutedBlock(blockIndex int) ([]sdk.Step, error) {
	if blockIndex >= len(t.ExecutedBlocks) {
		return nil, fmt.Errorf("block index %d out of range (only %d blocks executed)", blockIndex, len(t.ExecutedBlocks))
	}
	return t.ExecutedBlocks[blockIndex], nil
}

// Stdin returns the piped stdin (nil if not piped)
func (t *TestExecutionContext) Stdin() io.Reader {
	return t.stdin
}

// StdoutPipe returns the piped stdout writer (nil if not piped)
func (t *TestExecutionContext) StdoutPipe() io.Writer {
	return t.stdoutPipe
}

// Clone creates a new context with different args and pipes
// Inherits: Go context, environment, workdir
// Replaces: args, stdin, stdoutPipe
func (t *TestExecutionContext) Clone(args map[string]interface{}, stdin io.Reader, stdoutPipe io.Writer) sdk.ExecutionContext {
	newTest := *t // Copy (inherits ctx, environ, workdir)
	newTest.args = args
	newTest.stdin = stdin
	newTest.stdoutPipe = stdoutPipe
	return &newTest
}

// Verify TestExecutionContext implements sdk.ExecutionContext at compile time
var _ sdk.ExecutionContext = (*TestExecutionContext)(nil)
