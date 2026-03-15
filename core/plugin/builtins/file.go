package builtins

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// FilePlugin exposes redirect file endpoint capability.
type FilePlugin struct{}

func (p *FilePlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "file", Version: "1.0.0", APIVersion: 1}
}

func (p *FilePlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{FileRedirectCapability{}}
}

// FileRedirectCapability provides file redirect read/write support.
type FileRedirectCapability struct{}

func (c FileRedirectCapability) Path() string { return "file" }

func (c FileRedirectCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "path", Type: types.TypeString, Required: true},
			{Name: "perm", Type: types.TypeInt, Default: 0o644},
		},
		Block: plugin.BlockForbidden,
	}
}

func (c FileRedirectCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return fileLeafNode{}
}

func (c FileRedirectCapability) RedirectCaps() plugin.RedirectCaps {
	return plugin.RedirectCaps{Read: true, Write: true, Append: true, Atomic: true}
}

func (c FileRedirectCapability) OpenForRead(ctx plugin.ExecContext, args plugin.ResolvedArgs) (io.ReadCloser, error) {
	path, err := resolveFilePath(ctx, args)
	if err != nil {
		return nil, err
	}
	data, err := ctx.Session().Get(readCtx(ctx), path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (c FileRedirectCapability) OpenForWrite(ctx plugin.ExecContext, args plugin.ResolvedArgs, appendMode bool) (io.WriteCloser, error) {
	path, err := resolveFilePath(ctx, args)
	if err != nil {
		return nil, err
	}
	perm := fs.FileMode(args.GetInt("perm"))
	if perm == 0 {
		perm = 0o644
	}
	writer := &fileRedirectWriter{ctx: ctx, session: ctx.Session(), path: path, perm: perm, appendMode: appendMode}
	return writer, nil
}

type fileLeafNode struct{}

func (n fileLeafNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	return plugin.Result{ExitCode: plugin.ExitFailure}, fmt.Errorf("@file is an I/O endpoint, not an executable decorator")
}

type fileRedirectWriter struct {
	ctx        plugin.ExecContext
	session    plugin.ParentTransport
	path       string
	perm       fs.FileMode
	appendMode bool
	buf        bytes.Buffer
	closed     bool
}

func (w *fileRedirectWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("writer is closed")
	}
	if err := readCtx(w.ctx).Err(); err != nil {
		return 0, err
	}
	return w.buf.Write(p)
}

func (w *fileRedirectWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if !w.appendMode {
		if err := readCtx(w.ctx).Err(); err != nil {
			return err
		}
	}
	data := w.buf.Bytes()
	flushCtx := readCtx(w.ctx)
	if w.appendMode {
		flushCtx = context.WithoutCancel(flushCtx)
		if deadline, ok := readCtx(w.ctx).Deadline(); ok {
			var cancel context.CancelFunc
			flushCtx, cancel = context.WithDeadline(flushCtx, deadline)
			defer cancel()
		}
		existing, err := w.session.Get(flushCtx, w.path)
		if err == nil {
			data = append(append([]byte(nil), existing...), data...)
		} else if isMissingFileError(err) {
			data = append([]byte(nil), data...)
		} else {
			return fmt.Errorf("failed to read file %q for append: %w", w.path, err)
		}
	}
	if err := w.session.Put(flushCtx, data, w.path, w.perm); err != nil {
		return fmt.Errorf("failed to write file %q: %w", w.path, err)
	}
	return nil
}

func isMissingFileError(err error) bool {
	return err != nil && (os.IsNotExist(err) || strings.Contains(err.Error(), "not found"))
}

func resolveFilePath(ctx plugin.ExecContext, args plugin.ResolvedArgs) (string, error) {
	path := args.GetString("path")
	if path == "" {
		return "", fmt.Errorf("@file requires path parameter")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	workdir := ctx.Session().Snapshot().Workdir
	if workdir == "" {
		return path, nil
	}
	return filepath.Join(workdir, path), nil
}

func readCtx(ctx plugin.ExecContext) context.Context {
	if ctx.Context() != nil {
		return ctx.Context()
	}
	return context.Background()
}
