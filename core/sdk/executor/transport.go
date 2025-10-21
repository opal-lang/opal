package executor

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aledsdavies/opal/core/invariant"
)

// Transport abstracts command execution and file transfer for different environments.
// Implementations: LocalTransport (os/exec), SSHTransport (future), DockerTransport (future)
//
// Design principle: Transport is an implementation detail of decorators, not a first-class
// ExecutionContext member. Decorators like @ssh.connect wrap ExecutionContext and use
// Transport internally to redirect @shell commands to remote systems.
//
// Security: All I/O flows through provided readers/writers which are scrubber-compatible.
// Transport cannot bypass the scrubber - it only receives pre-scrubbed writers.
type Transport interface {
	// Exec executes a command and returns exit code.
	// All I/O flows through provided readers/writers (scrubber-compatible).
	//
	// Exit codes:
	//   0   - Success
	//   1-126 - Command-specific failure
	//   127 - Command not found
	//   124 - Timeout/cancellation (context cancelled)
	//
	// Returns (exitCode, nil) for normal execution (even if command fails).
	// Returns (127, err) only for system errors (command not found, etc).
	Exec(ctx context.Context, argv []string, opts ExecOpts) (exitCode int, err error)

	// Put transfers a file/directory to the destination.
	// src: local file reader, dst: destination path, mode: file permissions
	//
	// For LocalTransport: copies to local filesystem
	// For SSHTransport: uploads via SFTP
	// For DockerTransport: uses docker cp
	Put(ctx context.Context, src io.Reader, dst string, mode fs.FileMode) error

	// Get retrieves a file/directory from the source.
	// src: source path, dst: local file writer
	//
	// For LocalTransport: reads from local filesystem
	// For SSHTransport: downloads via SFTP
	// For DockerTransport: uses docker cp
	Get(ctx context.Context, src string, dst io.Writer) error

	// Close cleans up transport resources (connections, sessions).
	// Safe to call multiple times.
	Close() error
}

// ExecOpts contains options for command execution.
type ExecOpts struct {
	Stdin  io.Reader         // Command stdin (may be nil)
	Stdout io.Writer         // Command stdout (scrubbed)
	Stderr io.Writer         // Command stderr (scrubbed)
	Env    map[string]string // Environment variables (empty = inherit parent)
	Dir    string            // Working directory (empty = current dir)
}

// LocalTransport implements Transport for local command execution using os/exec.
// This is the default transport used by executor.Command().
type LocalTransport struct {
	// No state needed for local execution
}

// Exec executes a command locally using os/exec.
func (t *LocalTransport) Exec(ctx context.Context, argv []string, opts ExecOpts) (int, error) {
	invariant.Precondition(len(argv) > 0, "argv cannot be empty")
	invariant.NotNil(ctx, "context")

	// Create command
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	// Set working directory
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Set environment
	if len(opts.Env) > 0 {
		// Convert map to slice
		env := os.Environ() // Start with parent environment
		for k, v := range opts.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	// Wire up I/O
	cmd.Stdin = opts.Stdin
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = os.Stdout // Default to os.Stdout (locked down by CLI)
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = os.Stderr // Default to os.Stderr (locked down by CLI)
	}

	// Execute
	if err := cmd.Run(); err != nil {
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

// Put copies a file to the local filesystem.
// Creates parent directories if they don't exist.
func (t *LocalTransport) Put(ctx context.Context, src io.Reader, dst string, mode fs.FileMode) error {
	invariant.NotNil(ctx, "context")
	invariant.NotNil(src, "source reader")
	invariant.Precondition(dst != "", "destination path cannot be empty")

	// Create parent directories
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Create destination file
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Copy content
	if _, err := io.Copy(f, src); err != nil {
		return err
	}

	return nil
}

// Get reads a file from the local filesystem.
func (t *LocalTransport) Get(ctx context.Context, src string, dst io.Writer) error {
	invariant.NotNil(ctx, "context")
	invariant.Precondition(src != "", "source path cannot be empty")
	invariant.NotNil(dst, "destination writer")

	// Open source file
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Copy content
	if _, err := io.Copy(dst, f); err != nil {
		return err
	}

	return nil
}

// Close is a no-op for LocalTransport (no resources to clean up).
// Safe to call multiple times.
func (t *LocalTransport) Close() error {
	return nil
}
