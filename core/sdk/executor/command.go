package executor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/aledsdavies/opal/core/invariant"
)

// Cmd represents a safe command execution wrapper
// All output is automatically routed through the scrubber
type Cmd struct {
	cmd    *exec.Cmd
	stdout io.Writer
	stderr io.Writer
}

// CommandContext creates a new command with context
// This is the ONLY safe way for decorators to execute commands
// Output is automatically routed through os.Stdout/os.Stderr which are locked down
func CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	invariant.Precondition(name != "", "command name cannot be empty")
	invariant.NotNil(ctx, "context")

	cmd := exec.CommandContext(ctx, name, args...)
	return &Cmd{
		cmd:    cmd,
		stdout: os.Stdout, // Locked down by CLI, routes through scrubber
		stderr: os.Stderr, // Locked down by CLI, routes through scrubber
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

// SetEnv replaces the entire environment for the command
// AppendEnv adds environment variables to the command
// Preserves existing environment (including PATH) and adds new variables
// This is the recommended way to set environment variables
func (c *Cmd) AppendEnv(kv map[string]string) *Cmd {
	invariant.NotNil(kv, "environment map")
	env := os.Environ()
	for k, v := range kv {
		invariant.Precondition(k != "", "environment variable key cannot be empty")
		env = append(env, k+"="+v)
	}
	c.cmd.Env = env
	return c
}

// SetStdin sets the stdin for the command
// This allows feeding input to commands like cat, grep, etc.
func (c *Cmd) SetStdin(r io.Reader) *Cmd {
	invariant.NotNil(r, "stdin reader")
	c.cmd.Stdin = r
	return c
}

// SetDir sets the working directory for the command
func (c *Cmd) SetDir(dir string) *Cmd {
	invariant.Precondition(dir != "", "directory cannot be empty")
	c.cmd.Dir = dir
	return c
}

// Run executes the command and waits for it to complete
// Returns exit code (0 = success, non-zero = failure)
// Context cancellation returns exit code 124 (conventional timeout code)
func (c *Cmd) Run() (int, error) {
	// Wire up stdout/stderr
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr

	// Execute
	if err := c.cmd.Run(); err != nil {
		// Context cancellation/timeout
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return 124, nil // Conventional timeout exit code
		}
		// Process exited with non-zero code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Check if process was killed by signal (exit code -1)
			// This happens when context is cancelled
			if exitErr.ExitCode() == -1 {
				return 124, nil // Normalize to timeout code
			}
			return exitErr.ExitCode(), nil
		}
		// Other errors (e.g., command not found) return 127
		return 127, err
	}

	return 0, nil
}

// Start starts the command but does not wait for it to complete
// Use Wait() to wait for completion
func (c *Cmd) Start() error {
	// Wire up stdout/stderr
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr

	return c.cmd.Start()
}

// Wait waits for the command to complete
// Must be called after Start()
// Context cancellation returns exit code 124 (same as Run())
func (c *Cmd) Wait() (int, error) {
	if err := c.cmd.Wait(); err != nil {
		// Context cancellation/timeout
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return 124, nil // Conventional timeout exit code
		}
		// Process exited with non-zero code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Check if process was killed by signal (exit code -1)
			// This happens when context is cancelled
			if exitErr.ExitCode() == -1 {
				return 124, nil // Normalize to timeout code
			}
			return exitErr.ExitCode(), nil
		}
		// Other errors (e.g., command not found) return 127
		return 127, err
	}
	return 0, nil
}

// Process returns the underlying os.Process (for advanced use cases)
func (c *Cmd) Process() *os.Process {
	return c.cmd.Process
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
