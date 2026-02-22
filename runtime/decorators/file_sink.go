package decorators

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/opal-lang/opal/core/decorator"
)

// FileSinkDecorator implements the @file decorator for file I/O operations.
// It serves as a sink for output redirection (> and >>) and a source for
// input redirection (<).
//
// Usage:
//
//	echo "hello" > @file("output.txt")    # Overwrite (atomic)
//	echo "world" >> @file("output.txt")   # Append
//	cat < @file("input.txt")              # Read
type FileSinkDecorator struct {
	params map[string]any
}

// fileConfig holds the parsed parameters for @file decorator.
type fileConfig struct {
	Path string `decorator:"path"`
	Perm int    `decorator:"perm"` // File permissions (default 0644)
}

// Descriptor returns the decorator metadata.
func (d *FileSinkDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("file").
		Summary("File sink/source for I/O redirection").
		ParamString("path", "File path for I/O operations").
		Required().
		Examples("/var/log/app.log", "./output.txt").
		Done().
		ParamInt("perm", "File permissions (octal, default 0644)").
		Default(0o644).
		Done().
		Block(decorator.BlockForbidden).
		TransportScope(decorator.TransportScopeAny).
		Roles(decorator.RoleEndpoint).
		Build()
}

// fileNode wraps file I/O operations (not used for execution, only I/O).
type fileNode struct {
	params map[string]any
}

// Execute is a no-op for @file decorator.
// @file is only used as an I/O endpoint, not as an executable command.
// This method exists to satisfy the ExecNode interface but should never
// be called in normal operation.
func (n *fileNode) Execute(_ decorator.ExecContext) (decorator.Result, error) {
	return decorator.Result{ExitCode: decorator.ExitFailure},
		fmt.Errorf("@file is an I/O endpoint, not an executable decorator")
}

// Wrap implements the Exec interface.
// @file is a leaf decorator that only provides I/O capabilities.
func (d *FileSinkDecorator) Wrap(_ decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &fileNode{params: params}
}

// IOCaps returns the I/O capabilities for the @file decorator.
// Supports read (<), write (>), and append (>>) with atomic writes for overwrite mode.
func (d *FileSinkDecorator) IOCaps() decorator.IOCaps {
	return decorator.IOCaps{
		Read:   true, // < @file(path)
		Write:  true, // > @file(path)
		Append: true, // >> @file(path)
		Atomic: true, // Overwrite uses temp file + rename
	}
}

// parseConfig extracts configuration from params with defaults.
func (d *FileSinkDecorator) parseConfig() fileConfig {
	cfg := fileConfig{Perm: 0o644} // Default permissions

	if path, ok := d.params["path"].(string); ok {
		cfg.Path = path
	}
	if perm, ok := d.params["perm"].(int); ok && perm > 0 {
		cfg.Perm = perm
	}

	return cfg
}

// OpenRead opens the file for reading (< source).
// Uses the session's current working directory for relative paths.
func (d *FileSinkDecorator) OpenRead(ctx decorator.ExecContext, _ ...decorator.IOOpts) (io.ReadCloser, error) {
	cfg := d.parseConfig()
	if cfg.Path == "" {
		return nil, fmt.Errorf("@file requires path parameter")
	}
	session := ctx.Session
	if session == nil {
		return nil, fmt.Errorf("execution context missing session")
	}

	execCtx := ctx.Context
	if execCtx == nil {
		execCtx = context.Background()
	}

	// Resolve path against session's working directory
	path := resolvePath(cfg.Path, session)

	reader, writer := io.Pipe()

	go func() {
		res, err := session.Run(execCtx, []string{"sh", "-c", "cat " + shellQuote(path)}, decorator.RunOpts{Stdout: writer})
		if err != nil {
			_ = writer.CloseWithError(fmt.Errorf("failed to read file %q: %w", path, err))
			return
		}
		if res.ExitCode != decorator.ExitSuccess {
			_ = writer.CloseWithError(fmt.Errorf("failed to read file %q: %s", path, strings.TrimSpace(string(res.Stderr))))
			return
		}
		_ = writer.Close()
	}()

	return reader, nil
}

// OpenWrite opens the file for writing (> or >> sink).
// - append=false: Overwrite mode with atomic writes (temp file + rename)
// - append=true: Append mode (direct append)
func (d *FileSinkDecorator) OpenWrite(ctx decorator.ExecContext, appendMode bool, _ ...decorator.IOOpts) (io.WriteCloser, error) {
	cfg := d.parseConfig()
	if cfg.Path == "" {
		return nil, fmt.Errorf("@file requires path parameter")
	}
	session := ctx.Session
	if session == nil {
		return nil, fmt.Errorf("execution context missing session")
	}

	execCtx := ctx.Context
	if execCtx == nil {
		execCtx = context.Background()
	}

	// Resolve path against session's working directory
	path := resolvePath(cfg.Path, session)
	perm := fs.FileMode(cfg.Perm)

	return &fileSinkWriter{
		session:    session,
		context:    execCtx,
		path:       path,
		perm:       perm,
		appendMode: appendMode,
		commandErr: make(chan error, 1),
	}, nil
}

type fileSinkWriter struct {
	session    decorator.Session
	context    context.Context
	path       string
	perm       fs.FileMode
	appendMode bool
	stdinW     *io.PipeWriter
	once       sync.Once
	commandErr chan error
	closed     bool
}

func (w *fileSinkWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("writer is closed")
	}

	select {
	case <-w.context.Done():
		return 0, w.context.Err()
	default:
	}

	if err := w.startCommand(); err != nil {
		return 0, err
	}

	return w.stdinW.Write(p)
}

func (w *fileSinkWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	select {
	case <-w.context.Done():
		return w.context.Err()
	default:
	}

	if err := w.startCommand(); err != nil {
		return err
	}

	if err := w.stdinW.Close(); err != nil {
		return err
	}

	return <-w.commandErr
}

func (w *fileSinkWriter) startCommand() error {
	w.once.Do(func() {
		stdinR, stdinW := io.Pipe()
		w.stdinW = stdinW

		mode := ">"
		if w.appendMode {
			mode = ">>"
		}

		dir := filepath.Dir(w.path)
		command := "mkdir -p " + shellQuote(dir) + " && cat " + mode + " " + shellQuote(w.path)

		go func() {
			res, err := w.session.Run(w.context, []string{"sh", "-c", command}, decorator.RunOpts{Stdin: stdinR})
			if err != nil {
				w.commandErr <- fmt.Errorf("failed to write file %q: %w", w.path, err)
				return
			}
			if res.ExitCode != decorator.ExitSuccess {
				msg := strings.TrimSpace(string(res.Stderr))
				if msg == "" {
					msg = fmt.Sprintf("exit code %d", res.ExitCode)
				}
				w.commandErr <- fmt.Errorf("failed to write file %q: %s", w.path, msg)
				return
			}
			w.commandErr <- nil
		}()
	})

	return nil
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

// resolvePath resolves a path against the session's working directory.
// Absolute paths are returned unchanged.
func resolvePath(path string, session decorator.Session) string {
	if filepath.IsAbs(path) {
		return path
	}

	// Resolve against session's current working directory
	if session != nil {
		return filepath.Join(session.Cwd(), path)
	}

	return path
}

// WithParams implements decorator.IOFactory.
// Creates a new FileSinkDecorator instance with the given parameters.
func (d *FileSinkDecorator) WithParams(params map[string]any) decorator.IO {
	return &FileSinkDecorator{params: params}
}

// Register @file decorator
func init() {
	if err := decorator.Register("file", &FileSinkDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @file decorator: %v", err))
	}
}
