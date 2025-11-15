package decorators

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/sdk"
	"github.com/aledsdavies/opal/core/types"
)

// ShellDecorator implements the @shell decorator using the new decorator architecture.
// It executes shell commands via Session.Run() with bash -c wrapper.
// It also implements Endpoint for file I/O operations.
type ShellDecorator struct {
	params map[string]any // Parameters for endpoint mode
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
		Stderr: ctx.Stderr, // NEW: Forward stderr
	}

	result, err := ctx.Session.Run(execCtx, argv, opts)

	return result, err
}

// Open implements the Endpoint interface for file I/O.
// When used as redirect target, @shell("file.txt") opens the file for reading or writing.
func (d *ShellDecorator) Open(ctx decorator.ExecContext, mode decorator.IOType) (io.ReadWriteCloser, error) {
	// Extract file path from params
	filePath, ok := d.params["command"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("@shell endpoint requires command parameter (file path)")
	}

	// Open file based on mode
	switch mode {
	case decorator.IORead:
		// Open for reading
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for reading: %w", err)
		}
		return file, nil

	case decorator.IOWrite:
		// Open for writing (create or truncate)
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for writing: %w", err)
		}
		return file, nil

	case decorator.IODuplex:
		// Open for read/write
		file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for duplex: %w", err)
		}
		return file, nil

	default:
		return nil, fmt.Errorf("unsupported I/O mode: %s", mode)
	}
}

// shellSDKAdapter adapts ShellDecorator to old SDK interfaces (SinkProvider)
// This is a temporary bridge during migration to support redirect targets.
type shellSDKAdapter struct{}

// AsSink implements sdk.SinkProvider for redirect targets
func (a *shellSDKAdapter) AsSink(ctx sdk.ExecutionContext) sdk.Sink {
	// Extract file path from args
	filePath, ok := ctx.Args()["command"].(string)
	if !ok || filePath == "" {
		panic("@shell sink requires command parameter (file path)")
	}

	return &shellFileSink{path: filePath}
}

// shellFileSink implements sdk.Sink for file I/O
type shellFileSink struct {
	path string
}

func (s *shellFileSink) Caps() sdk.SinkCaps {
	return sdk.SinkCaps{
		Overwrite:      true,
		Append:         true,
		Atomic:         false,
		ConcurrentSafe: false,
	}
}

func (s *shellFileSink) Open(ctx sdk.ExecutionContext, mode sdk.RedirectMode, meta map[string]any) (io.WriteCloser, error) {
	switch mode {
	case sdk.RedirectOverwrite:
		return os.Create(s.path)
	case sdk.RedirectAppend:
		return os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	default:
		return nil, fmt.Errorf("unsupported redirect mode: %v", mode)
	}
}

func (s *shellFileSink) Identity() (string, string) {
	return "fs.file", s.path
}

// Register @shell decorator with both registries (dual registration during migration)
func init() {
	// Register with new decorator registry
	if err := decorator.Register("shell", &ShellDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @shell decorator: %v", err))
	}

	// Also register with old SDK registry for backward compatibility (redirect targets)
	// This allows @shell("file.txt") to work as redirect target during migration
	types.Global().RegisterSDKHandler("shell", types.DecoratorKindExecution, &shellSDKAdapter{})
}
