package decorators

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/opal-lang/opal/core/decorator"
)

// ShellDecorator implements the @shell decorator using the new decorator architecture.
// It executes shell commands via Session.Run() with bash -c wrapper.
// It also implements IO for file I/O operations (redirect sources and sinks).
type ShellDecorator struct {
	params map[string]any // Parameters for I/O mode
}

// Descriptor returns the decorator metadata.
func (d *ShellDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("shell").
		Summary("Execute shell commands or file I/O").
		ParamString("command", "Shell command or file path").
		Required().
		Examples("echo hello", "npm run build", "/path/to/file.txt").
		Done().
		Block(decorator.BlockForbidden).                      // Leaf decorator - no blocks
		TransportScope(decorator.TransportScopeAny).          // Works in any session
		Roles(decorator.RoleWrapper, decorator.RoleEndpoint). // Executes work AND provides I/O
		Build()
}

// Wrap implements the Exec interface.
// @shell is a leaf decorator - it ignores the 'next' parameter and executes directly.
func (d *ShellDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &shellNode{params: params}
}

// shellNode wraps shell command execution.
type shellNode struct {
	params map[string]any
}

// Execute implements the ExecNode interface.
// Executes the shell command via Session.Run() with bash -c wrapper.
func (n *shellNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	// Extract command from params
	command, ok := n.params["command"].(string)
	if !ok || command == "" {
		return decorator.Result{ExitCode: 127}, fmt.Errorf("@shell requires command parameter")
	}

	// INVARIANT: Command must not contain unresolved DisplayIDs
	// DisplayIDs should be resolved to actual values before execution
	// Format: opal:<base64url> where base64url is 22 chars [A-Za-z0-9_-]
	// Use regex to avoid false positives (e.g., "Documentation for opal: see docs/")
	displayIDPattern := regexp.MustCompile(`opal:[A-Za-z0-9_-]{22}`)
	if displayIDPattern.MatchString(command) {
		panic(fmt.Sprintf("INVARIANT VIOLATION: Command contains unresolved DisplayID: %s\n"+
			"DisplayIDs must be resolved to actual values before execution.\n"+
			"Format: opal:<base64url-hash> (22 chars)\n"+
			"This indicates the executor is not resolving secrets from the plan.", command))
	}

	// Use parent context for cancellation and deadlines
	// If no context provided, use background
	execCtx := ctx.Context
	if execCtx == nil {
		execCtx = context.Background()
	}

	// Execute command through session with bash -c wrapper
	argv := []string{"bash", "-c", command}

	// Configure I/O from ExecContext
	opts := decorator.RunOpts{
		Stdin:  ctx.Stdin,  // Piped input (nil if not piped)
		Stdout: ctx.Stdout, // Piped output (nil if not piped)
		Stderr: ctx.Stderr, // Forward stderr
	}

	result, err := ctx.Session.Run(execCtx, argv, opts)

	return result, err
}

// IOCaps implements decorator.IO.
// Returns the I/O capabilities for file operations.
func (d *ShellDecorator) IOCaps() decorator.IOCaps {
	return decorator.IOCaps{
		Read:   true,  // < file.txt
		Write:  true,  // > file.txt
		Append: true,  // >> file.txt
		Atomic: false, // TODO: implement atomic writes via temp + rename
	}
}

// OpenRead implements decorator.IO.
// Opens a file for reading (< source).
func (d *ShellDecorator) OpenRead(ctx decorator.ExecContext, opts ...decorator.IOOpts) (io.ReadCloser, error) {
	filePath, ok := d.params["command"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("@shell I/O requires command parameter (file path)")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for reading: %w", err)
	}
	return file, nil
}

// OpenWrite implements decorator.IO.
// Opens a file for writing (> or >> sink).
func (d *ShellDecorator) OpenWrite(ctx decorator.ExecContext, appendMode bool, opts ...decorator.IOOpts) (io.WriteCloser, error) {
	filePath, ok := d.params["command"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("@shell I/O requires command parameter (file path)")
	}

	if appendMode {
		// >> append mode
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for append: %w", err)
		}
		return file, nil
	}

	// > overwrite mode
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for writing: %w", err)
	}
	return file, nil
}

// WithParams implements decorator.IOFactory.
// Creates a new ShellDecorator instance with the given parameters.
// This is used for redirect targets where @shell("file.txt") needs params.
func (d *ShellDecorator) WithParams(params map[string]any) decorator.IO {
	return &ShellDecorator{params: params}
}

// Register @shell decorator
func init() {
	// Register with decorator registry
	if err := decorator.Register("shell", &ShellDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @shell decorator: %v", err))
	}
}
