package executor

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/sdk"
)

// executionContext implements sdk.ExecutionContext
// All fields are immutable - modifications create new contexts
type executionContext struct {
	executor   *executor
	args       map[string]interface{} // Decorator arguments (from sdk.Command)
	ctx        context.Context
	environ    map[string]string // Immutable snapshot
	workdir    string            // Immutable snapshot
	stdin      io.Reader         // Piped input (nil if not piped)
	stdoutPipe io.Writer         // Piped output (nil if not piped)
}

// newExecutionContext creates a new execution context for a decorator
// Captures current environment and working directory as immutable snapshots
func newExecutionContext(args map[string]interface{}, exec *executor, ctx context.Context) sdk.ExecutionContext {
	invariant.NotNil(ctx, "context")
	invariant.NotNil(args, "args")

	// Capture current working directory at context creation time
	// This ensures isolation - changes to os.Getwd() won't affect this context
	wd, err := os.Getwd()
	if err != nil {
		panic("failed to get working directory: " + err.Error())
	}

	return &executionContext{
		executor:   exec,
		args:       args,
		ctx:        ctx,
		environ:    captureEnviron(), // Immutable snapshot
		workdir:    wd,               // Immutable snapshot
		stdin:      nil,              // Root context has no piped input
		stdoutPipe: nil,              // Root context has no piped output
	}
}

// ExecuteBlock executes nested steps (callback to executor)
// Works with sdk.Step natively - no conversion needed
func (e *executionContext) ExecuteBlock(steps []sdk.Step) (int, error) {
	// Execute steps using executor logic (now works with sdk.Step natively)
	for _, step := range steps {
		exitCode := e.executor.executeStep(step)
		if exitCode != 0 {
			return exitCode, nil
		}
	}
	return 0, nil
}

// Context returns the Go context for cancellation and deadlines
func (e *executionContext) Context() context.Context {
	return e.ctx
}

// ArgString retrieves a string argument
func (e *executionContext) ArgString(key string) string {
	if val, ok := e.args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// ArgInt retrieves an integer argument
func (e *executionContext) ArgInt(key string) int64 {
	if val, ok := e.args[key]; ok {
		if i, ok := val.(int64); ok {
			return i
		}
		if i, ok := val.(int); ok {
			return int64(i)
		}
	}
	return 0
}

// ArgBool retrieves a boolean argument
func (e *executionContext) ArgBool(key string) bool {
	if val, ok := e.args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// ArgDuration retrieves a duration argument
// TODO: Implement when SDK supports Duration type
func (e *executionContext) ArgDuration(key string) time.Duration {
	// For now, durations are stored as strings and parsed
	// This will be updated when SDK adds Duration support
	_ = key
	return 0
}

// Args returns a snapshot of all arguments for logging
func (e *executionContext) Args() map[string]interface{} {
	// Return a copy to prevent external mutation
	copy := make(map[string]interface{}, len(e.args))
	for k, v := range e.args {
		copy[k] = v
	}
	return copy
}

// Environ returns the environment variables (immutable snapshot)
func (e *executionContext) Environ() map[string]string {
	return e.environ
}

// Workdir returns the working directory (immutable snapshot)
func (e *executionContext) Workdir() string {
	return e.workdir
}

// WithContext returns a new context with the specified Go context
// Original context is unchanged (immutable)
func (e *executionContext) WithContext(ctx context.Context) sdk.ExecutionContext {
	invariant.NotNil(ctx, "context")

	return &executionContext{
		executor:   e.executor,
		args:       e.args,
		ctx:        ctx,
		environ:    e.environ,    // Share immutable snapshot
		workdir:    e.workdir,    // Share immutable snapshot
		stdin:      e.stdin,      // Preserve pipes
		stdoutPipe: e.stdoutPipe, // Preserve pipes
	}
}

// WithEnviron returns a new context with the specified environment
// Original context is unchanged (immutable)
func (e *executionContext) WithEnviron(env map[string]string) sdk.ExecutionContext {
	invariant.NotNil(env, "environment")

	// Deep copy to ensure immutability
	envCopy := make(map[string]string, len(env))
	for k, v := range env {
		envCopy[k] = v
	}

	return &executionContext{
		executor:   e.executor,
		args:       e.args,
		ctx:        e.ctx,
		environ:    envCopy,
		workdir:    e.workdir,
		stdin:      e.stdin,      // Preserve pipes
		stdoutPipe: e.stdoutPipe, // Preserve pipes
	}
}

// WithWorkdir returns a new context with the specified working directory
// Original context is unchanged (immutable)
//
// Accepts both absolute and relative paths:
// - Absolute paths (e.g., "/tmp") are used as-is
// - Relative paths (e.g., "subdir", "../other", "./foo") are resolved against current context workdir
//
// Examples:
//
//	ctx.WithWorkdir("/tmp")           // → /tmp
//	ctx.WithWorkdir("subdir")         // → /current/subdir
//	ctx.WithWorkdir("../other")       // → /other (if current is /current)
//	ctx.WithWorkdir("foo/bar")        // → /current/foo/bar
//
// Chaining works intuitively:
//
//	ctx.WithWorkdir("foo").WithWorkdir("bar")  // → /current/foo/bar
//	ctx.WithWorkdir("foo").WithWorkdir("..")   // → /current
func (e *executionContext) WithWorkdir(dir string) sdk.ExecutionContext {
	invariant.Precondition(dir != "", "working directory cannot be empty")

	var resolved string
	if filepath.IsAbs(dir) {
		// Absolute path - use as-is
		resolved = dir
	} else {
		// Relative path - resolve against current context workdir
		resolved = filepath.Join(e.workdir, dir)
	}

	// Clean the path (remove . and .. components, collapse multiple slashes)
	resolved = filepath.Clean(resolved)

	return &executionContext{
		executor:   e.executor,
		args:       e.args,
		ctx:        e.ctx,
		environ:    e.environ,
		workdir:    resolved,
		stdin:      e.stdin,      // Preserve pipes
		stdoutPipe: e.stdoutPipe, // Preserve pipes
	}
}

// Stdin returns piped input (nil if not piped)
func (e *executionContext) Stdin() io.Reader {
	return e.stdin
}

// StdoutPipe returns piped output (nil if not piped)
func (e *executionContext) StdoutPipe() io.Writer {
	return e.stdoutPipe
}

// Clone creates a new context for a child command
// Inherits: Go context, environment, workdir
// Replaces: args, stdin, stdoutPipe
func (e *executionContext) Clone(
	args map[string]interface{},
	stdin io.Reader,
	stdoutPipe io.Writer,
) sdk.ExecutionContext {
	invariant.NotNil(args, "args")

	return &executionContext{
		executor:   e.executor,
		args:       args,
		ctx:        e.ctx,      // INHERIT parent context
		environ:    e.environ,  // INHERIT environment
		workdir:    e.workdir,  // INHERIT workdir
		stdin:      stdin,      // NEW (may be nil)
		stdoutPipe: stdoutPipe, // NEW (may be nil)
	}
}

// captureEnviron captures current environment as immutable snapshot
// Returns a new map that won't be affected by future os.Setenv() calls
func captureEnviron() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		// Split on first '=' only
		if idx := strings.IndexByte(e, '='); idx > 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return env
}
