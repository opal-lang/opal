package executor

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/aledsdavies/opal/core/invariant"
)

// Cmd represents a safe command execution wrapper
// All output is automatically routed through the scrubber
//
// Internally uses Transport abstraction (LocalTransport by default).
// Future: Can be extended to support remote execution via custom transports.
type Cmd struct {
	transport Transport         // Transport for execution (LocalTransport by default)
	ctx       context.Context   // Execution context
	argv      []string          // Command and arguments
	stdin     io.Reader         // Command stdin (may be nil)
	stdout    io.Writer         // Command stdout (scrubbed)
	stderr    io.Writer         // Command stderr (scrubbed)
	env       map[string]string // Environment variables
	dir       string            // Working directory
}

// CommandContext creates a new command with context
// This is the ONLY safe way for decorators to execute commands
// Output is automatically routed through os.Stdout/os.Stderr which are locked down
func CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	invariant.Precondition(name != "", "command name cannot be empty")
	invariant.NotNil(ctx, "context")

	// Build argv
	argv := make([]string, 0, len(args)+1)
	argv = append(argv, name)
	argv = append(argv, args...)

	return &Cmd{
		transport: &LocalTransport{}, // Default to local execution
		ctx:       ctx,
		argv:      argv,
		stdout:    os.Stdout, // Locked down by CLI, routes through scrubber
		stderr:    os.Stderr, // Locked down by CLI, routes through scrubber
		env:       make(map[string]string),
	}
}

// CommandWithTransport creates a new command with a custom transport
// This is for future use by decorators like @ssh.connect, @docker.exec
// NOT part of public API yet - internal use only
func CommandWithTransport(ctx context.Context, transport Transport, name string, args ...string) *Cmd {
	invariant.Precondition(name != "", "command name cannot be empty")
	invariant.NotNil(ctx, "context")
	invariant.NotNil(transport, "transport")

	// Build argv
	argv := make([]string, 0, len(args)+1)
	argv = append(argv, name)
	argv = append(argv, args...)

	return &Cmd{
		transport: transport,
		ctx:       ctx,
		argv:      argv,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		env:       make(map[string]string),
	}
}

// Command creates a new command without context
// This is the ONLY safe way for decorators to execute commands
// Output is automatically routed through os.Stdout/os.Stderr which are locked down
func Command(name string, args ...string) *Cmd {
	return CommandContext(context.Background(), name, args...)
}

// Bash creates a command that executes a bash script
// This is the safe way to run shell commands from decorators
func Bash(script string) *Cmd {
	invariant.Precondition(script != "", "bash script cannot be empty")
	return Command("bash", "-c", script)
}

// BashContext creates a command that executes a bash script with context
func BashContext(ctx context.Context, script string) *Cmd {
	invariant.Precondition(script != "", "bash script cannot be empty")
	return CommandContext(ctx, "bash", "-c", script)
}

// SetStdout overrides stdout (for testing or capturing output)
// WARNING: Only use this if you know what you're doing
// The default stdout is already locked down and scrubbed
func (c *Cmd) SetStdout(w io.Writer) *Cmd {
	invariant.NotNil(w, "stdout writer")
	c.stdout = w
	return c
}

// SetStderr overrides stderr (for testing or capturing output)
// WARNING: Only use this if you know what you're doing
// The default stderr is already locked down and scrubbed
func (c *Cmd) SetStderr(w io.Writer) *Cmd {
	invariant.NotNil(w, "stderr writer")
	c.stderr = w
	return c
}

// AppendEnv adds environment variables to the command
// Preserves existing environment (including PATH) and adds new variables
// This is the recommended way to set environment variables
func (c *Cmd) AppendEnv(kv map[string]string) *Cmd {
	invariant.NotNil(kv, "environment map")
	for k, v := range kv {
		invariant.Precondition(k != "", "environment variable key cannot be empty")
		c.env[k] = v
	}
	return c
}

// SetStdin sets the stdin for the command
// This allows feeding input to commands like cat, grep, etc.
func (c *Cmd) SetStdin(r io.Reader) *Cmd {
	invariant.NotNil(r, "stdin reader")
	c.stdin = r
	return c
}

// SetDir sets the working directory for the command
func (c *Cmd) SetDir(dir string) *Cmd {
	invariant.Precondition(dir != "", "directory cannot be empty")
	c.dir = dir
	return c
}

// Run executes the command and waits for it to complete
// Returns exit code (0 = success, non-zero = failure)
// Context cancellation returns exit code 124 (conventional timeout code)
func (c *Cmd) Run() (int, error) {
	opts := ExecOpts{
		Stdin:  c.stdin,
		Stdout: c.stdout,
		Stderr: c.stderr,
		Env:    c.env,
		Dir:    c.dir,
	}

	return c.transport.Exec(c.ctx, c.argv, opts)
}

// Output runs the command and returns its stdout output
// Stderr is still routed through the locked-down stderr
// WARNING: Captured buffers are not scrubbed; print responsibly
// Returns (output, exitCode, error)
func (c *Cmd) Output() ([]byte, int, error) {
	var buf bytes.Buffer
	c.SetStdout(&buf)

	exitCode, err := c.Run()
	return buf.Bytes(), exitCode, err
}

// CombinedOutput runs the command and returns combined stdout+stderr output
// WARNING: Captured buffers are not scrubbed; print responsibly
// Returns (output, exitCode, error)
func (c *Cmd) CombinedOutput() ([]byte, int, error) {
	var buf bytes.Buffer
	c.SetStdout(&buf)
	c.SetStderr(&buf)

	exitCode, err := c.Run()
	return buf.Bytes(), exitCode, err
}
