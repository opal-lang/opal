package decorator

import (
	"context"
	"io"
	"io/fs"
)

// Session represents an execution context (local, SSH, Docker, K8s, etc.).
// All execution happens within a Session.
// Sessions are context-aware for cancellation and timeout propagation.
type Session interface {
	// Run executes a command with arguments
	// Context controls cancellation and timeouts
	Run(ctx context.Context, argv []string, opts RunOpts) (Result, error)

	// Put writes data to a file on the session's filesystem
	// Context controls cancellation and timeouts
	Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error

	// Get reads data from a file on the session's filesystem
	// Context controls cancellation and timeouts
	Get(ctx context.Context, path string) ([]byte, error)

	// Env returns an immutable snapshot of environment variables
	Env() map[string]string

	// WithEnv returns a new Session with environment delta applied (copy-on-write)
	// The delta only applies to executions from the returned Session
	WithEnv(delta map[string]string) Session

	// WithWorkdir returns a new Session with working directory set (copy-on-write)
	// The workdir only applies to executions from the returned Session
	WithWorkdir(dir string) Session

	// Cwd returns the current working directory
	Cwd() string

	// ID returns a unique identifier for this session
	// Format: "local", "ssh:hostname", "docker:container-id", "k8s:pod-name"
	// Used for session-scoped variable tracking and cross-session leakage prevention
	ID() string

	// TransportScope returns the transport scope of this session
	// Used to enforce decorator transport-scope guards
	TransportScope() TransportScope

	// Close cleans up the session
	Close() error
}

// RunOpts configures command execution.
type RunOpts struct {
	Stdin  io.Reader // Changed from []byte to io.Reader to enable streaming
	Stdout io.Writer // Optional: if nil, captured in Result.Stdout
	Stderr io.Writer // Optional: if nil, captured in Result.Stderr
	Dir    string    // Optional: working directory for this execution only
}

// Result is the outcome of command execution.
type Result struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

// Standard exit codes for command execution.
const (
	// ExitSuccess indicates successful command execution
	ExitSuccess = 0

	// ExitCanceled indicates command was canceled by context
	// This is the canonical signal for "aborted by policy" (timeout, user cancel, etc.)
	ExitCanceled = -1

	// ExitFailure indicates generic command failure
	ExitFailure = 1
)
