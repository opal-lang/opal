package decorator

import (
	"context"
	"io"
)

// Exec is the interface for decorators that wrap execution.
// Exec decorators use the middleware pattern to compose behavior.
// Examples: @retry, @timeout, @parallel
type Exec interface {
	Decorator
	Wrap(next ExecNode, params map[string]any) ExecNode
}

// ExecNode represents an executable node in the execution tree.
type ExecNode interface {
	Execute(ctx ExecContext) (Result, error)
}

// BranchExecutor is an optional extension for block-aware execution nodes.
// It allows wrapper decorators (like @parallel) to execute top-level block
// branches independently while keeping executor internals generic.
type BranchExecutor interface {
	ExecNode
	BranchCount() int
	ExecuteBranch(index int, ctx ExecContext) (Result, error)
}

// ExecContext provides the execution context for command execution.
type ExecContext struct {
	// Context is the parent context for cancellation and deadlines
	// Decorators should pass this to Session.Run() and other operations
	Context context.Context

	// Session is the ambient execution context (env, cwd, transport)
	Session Session

	// Stdin provides input data for piped commands (nil if not piped)
	// Used for pipe operators: cmd1 | cmd2
	// Changed from []byte to io.Reader to enable streaming
	Stdin io.Reader

	// Stdout captures output for piped commands (nil if not piped)
	// Used for pipe operators: cmd1 | cmd2
	Stdout io.Writer

	// Stderr captures error output (nil defaults to os.Stderr)
	// Stderr NEVER pipes in POSIX - always goes to terminal
	Stderr io.Writer

	// Trace is the telemetry span for observability
	// Opal runtime creates parent span automatically
	// Decorators can create child spans for internal tracking
	Trace Span
}

// WithContext returns a copy with a new cancellation/deadline context.
func (c ExecContext) WithContext(ctx context.Context) ExecContext {
	c.Context = ctx
	return c
}

// WithSession returns a copy with a new ambient session.
func (c ExecContext) WithSession(session Session) ExecContext {
	c.Session = session
	return c
}

// WithIO returns a copy with overridden stdio streams.
func (c ExecContext) WithIO(stdin io.Reader, stdout, stderr io.Writer) ExecContext {
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	return c
}
