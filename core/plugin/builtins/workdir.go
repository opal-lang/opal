package builtins

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// FSPlugin exposes filesystem wrapper capabilities.
type FSPlugin struct{}

func (p *FSPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "fs", Version: "1.0.0", APIVersion: 1}
}

func (p *FSPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{WorkdirWrapperCapability{}}
}

// WorkdirWrapperCapability executes nested nodes with a different working directory.
type WorkdirWrapperCapability struct{}

func (c WorkdirWrapperCapability) Path() string { return "fs.workdir" }

func (c WorkdirWrapperCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{{Name: "path", Type: types.TypeString, Required: true}},
		Block:  plugin.BlockRequired,
	}
}

func (c WorkdirWrapperCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return workdirNode{next: next, path: args.GetString("path")}
}

type workdirNode struct {
	next plugin.ExecNode
	path string
}

func (n workdirNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	if n.next == nil {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}
	if n.path == "" {
		return plugin.Result{ExitCode: plugin.ExitFailure}, fmt.Errorf("@fs.workdir path must not be empty")
	}

	baseSession := ctx.Session()
	if baseSession == nil {
		return plugin.Result{ExitCode: plugin.ExitFailure}, fmt.Errorf("@fs.workdir requires session")
	}

	childSession := workdirParentTransport{inner: baseSession, path: n.path}
	return n.next.Execute(workdirExecContext{ctx: ctx.Context(), session: childSession, stdin: ctx.Stdin(), stdout: ctx.Stdout(), stderr: ctx.Stderr()})
}

type workdirParentTransport struct {
	inner plugin.ParentTransport
	path  string
}

func (s workdirParentTransport) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	if opts.Dir == "" {
		opts.Dir = s.path
	}
	return s.inner.Run(ctx, argv, opts)
}

func (s workdirParentTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.inner.Put(ctx, data, path, mode)
}

func (s workdirParentTransport) Get(ctx context.Context, path string) ([]byte, error) {
	return s.inner.Get(ctx, path)
}

func (s workdirParentTransport) Snapshot() plugin.SessionSnapshot {
	snapshot := s.inner.Snapshot()
	snapshot.Workdir = s.path
	return snapshot
}

func (s workdirParentTransport) UnwrapParentTransport() plugin.ParentTransport {
	return s.inner
}

func (s workdirParentTransport) Close() error { return nil }

type workdirExecContext struct {
	ctx     context.Context
	session plugin.ParentTransport
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (c workdirExecContext) Context() context.Context        { return c.ctx }
func (c workdirExecContext) Session() plugin.ParentTransport { return c.session }
func (c workdirExecContext) Stdin() io.Reader                { return c.stdin }
func (c workdirExecContext) Stdout() io.Writer               { return c.stdout }
func (c workdirExecContext) Stderr() io.Writer               { return c.stderr }
