package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"sync"

	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	coreruntime "github.com/builtwithtofu/sigil/core/runtime"
	"github.com/builtwithtofu/sigil/core/types"
)

type executorSessionTestPlugin struct{}

type pluginSessionIDCheckCapability struct{}

type pluginSessionBoundaryCapability struct{}

type pluginCaptureSinkCapability struct{}

type pluginChdirCapability struct{}

type pluginPlanRedirectErrorSinkCapability struct{}

type pluginPlanRedirectReadonlySinkCapability struct{}

type pluginPoolProbeTransportCapability struct{}

type pluginSessionIDCheckNode struct {
	expect string
}

type pluginSessionBoundaryNode struct {
	next coreplugin.ExecNode
	id   string
}

type pluginChdirNode struct {
	next coreplugin.ExecNode
	dir  string
}

type pluginSessionOverrideParent struct {
	base    coreplugin.ParentTransport
	session coreruntime.Session
}

type pluginSinkCaptureRecord struct {
	openCount  int
	sessionIDs []string
	output     bytes.Buffer
}

type pluginSinkCaptureStore struct {
	mu      sync.Mutex
	records map[string]*pluginSinkCaptureRecord
}

type pluginCaptureSinkWriter struct {
	id string
}

var (
	registerExecutorSessionTestPluginOnce sync.Once
	pluginTestSinkStore                   = &pluginSinkCaptureStore{records: map[string]*pluginSinkCaptureRecord{}}
	pluginPoolProbeOpenMu                 sync.Mutex
	pluginPoolProbeOpenCount              int
)

func (p *executorSessionTestPlugin) Identity() coreplugin.PluginIdentity {
	return coreplugin.PluginIdentity{Name: "executor-session-test", Version: "1.0.0", APIVersion: 1}
}

func (p *executorSessionTestPlugin) Capabilities() []coreplugin.Capability {
	return []coreplugin.Capability{pluginSessionIDCheckCapability{}, pluginSessionBoundaryCapability{}, pluginCaptureSinkCapability{}, pluginChdirCapability{}, pluginPlanRedirectErrorSinkCapability{}, pluginPlanRedirectReadonlySinkCapability{}, pluginPoolProbeTransportCapability{}}
}

func (c pluginSessionIDCheckCapability) Path() string { return "test.sessionid.check" }

func (c pluginSessionIDCheckCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "expect", Type: types.TypeString, Required: true}}}
}

func (c pluginSessionIDCheckCapability) Wrap(next coreplugin.ExecNode, args coreplugin.ResolvedArgs) coreplugin.ExecNode {
	_ = next
	return pluginSessionIDCheckNode{expect: args.GetString("expect")}
}

func (n pluginSessionIDCheckNode) Execute(ctx coreplugin.ExecContext) (coreplugin.Result, error) {
	session, ok := decoratorSessionFromParent(ctx.Session())
	if !ok {
		return coreplugin.Result{ExitCode: coreplugin.ExitFailure}, fmt.Errorf("expected runtime session in parent transport")
	}
	if session.ID() != n.expect {
		return coreplugin.Result{ExitCode: 99}, fmt.Errorf("session mismatch: want %q got %q", n.expect, session.ID())
	}
	return coreplugin.Result{ExitCode: coreplugin.ExitSuccess}, nil
}

func (c pluginSessionBoundaryCapability) Path() string { return "test.session.boundary" }

func (c pluginSessionBoundaryCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "id", Type: types.TypeString, Required: true}}, Block: coreplugin.BlockRequired}
}

func (c pluginSessionBoundaryCapability) Wrap(next coreplugin.ExecNode, args coreplugin.ResolvedArgs) coreplugin.ExecNode {
	return pluginSessionBoundaryNode{next: next, id: args.GetString("id")}
}

func (n pluginSessionBoundaryNode) Execute(ctx coreplugin.ExecContext) (coreplugin.Result, error) {
	if n.next == nil {
		return coreplugin.Result{ExitCode: coreplugin.ExitSuccess}, nil
	}
	base, ok := decoratorSessionFromParent(ctx.Session())
	if !ok {
		return coreplugin.Result{ExitCode: coreplugin.ExitFailure}, fmt.Errorf("expected runtime session in parent transport")
	}
	override := &transportScopedSession{id: n.id, session: base}
	child := pluginExecContext{ctx: ctx.Context(), session: pluginSessionOverrideParent{base: ctx.Session(), session: override}, stdin: ctx.Stdin(), stdout: ctx.Stdout(), stderr: ctx.Stderr()}
	return n.next.Execute(child)
}

func (c pluginCaptureSinkCapability) Path() string { return "test.capture.sink" }

func (c pluginCaptureSinkCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "command", Type: types.TypeString, Required: true}}}
}

func (c pluginCaptureSinkCapability) RedirectCaps() coreplugin.RedirectCaps {
	return coreplugin.RedirectCaps{Write: true, Append: true}
}

func (c pluginCaptureSinkCapability) OpenForRead(ctx coreplugin.ExecContext, args coreplugin.ResolvedArgs) (io.ReadCloser, error) {
	_ = ctx
	_ = args
	return nil, fmt.Errorf("read unsupported")
}

func (c pluginCaptureSinkCapability) OpenForWrite(ctx coreplugin.ExecContext, args coreplugin.ResolvedArgs, appendMode bool) (io.WriteCloser, error) {
	id := args.GetString("command")
	pluginTestSinkStore.withRecord(id, func(record *pluginSinkCaptureRecord) {
		record.openCount++
		if session, ok := decoratorSessionFromParent(ctx.Session()); ok {
			record.sessionIDs = append(record.sessionIDs, session.ID())
		}
		if !appendMode {
			record.output.Reset()
		}
	})
	return &pluginCaptureSinkWriter{id: id}, nil
}

func (c pluginChdirCapability) Path() string { return "test.chdir" }

func (c pluginChdirCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "dir", Type: types.TypeString, Required: true}}, Block: coreplugin.BlockRequired}
}

func (c pluginChdirCapability) Wrap(next coreplugin.ExecNode, args coreplugin.ResolvedArgs) coreplugin.ExecNode {
	return pluginChdirNode{next: next, dir: args.GetString("dir")}
}

func (n pluginChdirNode) Execute(ctx coreplugin.ExecContext) (coreplugin.Result, error) {
	if n.next == nil {
		return coreplugin.Result{ExitCode: coreplugin.ExitSuccess}, nil
	}
	base, ok := decoratorSessionFromParent(ctx.Session())
	if !ok {
		return coreplugin.Result{ExitCode: coreplugin.ExitFailure}, fmt.Errorf("expected runtime session in parent transport")
	}
	override := base.WithWorkdir(n.dir)
	child := pluginExecContext{ctx: ctx.Context(), session: pluginSessionOverrideParent{base: ctx.Session(), session: override}, stdin: ctx.Stdin(), stdout: ctx.Stdout(), stderr: ctx.Stderr()}
	return n.next.Execute(child)
}

func (p pluginSessionOverrideParent) decoratorSession() coreruntime.Session { return p.session }

func (p pluginSessionOverrideParent) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	return p.base.Run(ctx, argv, opts)
}

func (p pluginSessionOverrideParent) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return p.base.Put(ctx, data, path, mode)
}

func (p pluginSessionOverrideParent) Get(ctx context.Context, path string) ([]byte, error) {
	return p.base.Get(ctx, path)
}

func (p pluginSessionOverrideParent) Snapshot() coreplugin.SessionSnapshot {
	return coreplugin.SessionSnapshot{Env: p.session.Env(), Workdir: p.session.Cwd(), Platform: p.session.Platform()}
}

func (p pluginSessionOverrideParent) Close() error { return nil }

func (p pluginSessionOverrideParent) NetworkDialer() coreplugin.NetworkDialer {
	provider, ok := p.base.(coreplugin.NetworkDialerProvider)
	if !ok {
		return nil
	}
	return provider.NetworkDialer()
}

func (c pluginPlanRedirectErrorSinkCapability) Path() string { return "test.plan.redirect.sink" }

func (c pluginPlanRedirectErrorSinkCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "command", Type: types.TypeString, Required: true}, {Name: "read", Type: types.TypeBool}, {Name: "write", Type: types.TypeBool}, {Name: "append", Type: types.TypeBool}, {Name: "fail_open", Type: types.TypeString}, {Name: "fail_close", Type: types.TypeString}}}
}

func (c pluginPlanRedirectErrorSinkCapability) RedirectCaps() coreplugin.RedirectCaps {
	return coreplugin.RedirectCaps{Read: true, Write: true, Append: true}
}

func (c pluginPlanRedirectErrorSinkCapability) OpenForRead(ctx coreplugin.ExecContext, args coreplugin.ResolvedArgs) (io.ReadCloser, error) {
	_ = ctx
	readClose := error(nil)
	if args.GetStringOptional("fail_close") == "read" {
		readClose = fmt.Errorf("read close failed")
	}
	if args.GetStringOptional("fail_open") == "open" {
		return nil, fmt.Errorf("open failed")
	}
	return &planCloseControlledReader{reader: bytes.NewBufferString("alpha\n"), closeErr: readClose}, nil
}

func (c pluginPlanRedirectErrorSinkCapability) OpenForWrite(ctx coreplugin.ExecContext, args coreplugin.ResolvedArgs, appendMode bool) (io.WriteCloser, error) {
	_ = ctx
	_ = appendMode
	if args.GetStringOptional("fail_open") == "open" {
		return nil, fmt.Errorf("open failed")
	}
	writeClose := error(nil)
	if args.GetStringOptional("fail_close") == "write" {
		writeClose = fmt.Errorf("write close failed")
	}
	return &planCloseControlledWriter{closeErr: writeClose}, nil
}

func (c pluginPlanRedirectReadonlySinkCapability) Path() string { return "test.plan.redirect.readonly" }

func (c pluginPlanRedirectReadonlySinkCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "command", Type: types.TypeString, Required: true}}}
}

func (c pluginPlanRedirectReadonlySinkCapability) RedirectCaps() coreplugin.RedirectCaps {
	return coreplugin.RedirectCaps{Read: true}
}

func (c pluginPlanRedirectReadonlySinkCapability) OpenForRead(ctx coreplugin.ExecContext, args coreplugin.ResolvedArgs) (io.ReadCloser, error) {
	_ = ctx
	_ = args
	return &planCloseControlledReader{reader: bytes.NewBufferString("alpha\n")}, nil
}

func (c pluginPlanRedirectReadonlySinkCapability) OpenForWrite(ctx coreplugin.ExecContext, args coreplugin.ResolvedArgs, appendMode bool) (io.WriteCloser, error) {
	_ = ctx
	_ = args
	_ = appendMode
	return nil, fmt.Errorf("write unsupported")
}

func (c pluginPoolProbeTransportCapability) Path() string { return "test.transport.poolprobe" }

func (c pluginPoolProbeTransportCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "host", Type: types.TypeString}, {Name: "port", Type: types.TypeInt}, {Name: "user", Type: types.TypeString}, {Name: "key", Type: types.TypeString}, {Name: "password", Type: types.TypeString}, {Name: "strict_host_key", Type: types.TypeBool}}}
}

func (c pluginPoolProbeTransportCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	_ = ctx
	_ = args
	pluginPoolProbeOpenMu.Lock()
	pluginPoolProbeOpenCount++
	pluginPoolProbeOpenMu.Unlock()
	snapshot := coreplugin.SessionSnapshot{Env: map[string]string{}, Platform: "linux"}
	if parent != nil {
		snapshot = parent.Snapshot()
	}
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	snapshot.Env["SIGIL_TRANSPORT_SUFFIX"] = "/poolprobe"
	return &noDialerOpenedTransport{id: "poolprobe", parent: parent, snapshot: snapshot}, nil
}

func (s *pluginSinkCaptureStore) reset(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = &pluginSinkCaptureRecord{}
}

func (s *pluginSinkCaptureStore) withRecord(id string, f func(*pluginSinkCaptureRecord)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	if !ok {
		record = &pluginSinkCaptureRecord{}
		s.records[id] = record
	}
	f(record)
}

func (s *pluginSinkCaptureStore) snapshot(id string) pluginSinkCaptureRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	if !ok {
		return pluginSinkCaptureRecord{}
	}
	copyRecord := pluginSinkCaptureRecord{openCount: record.openCount, sessionIDs: append([]string(nil), record.sessionIDs...)}
	copyRecord.output.Write(record.output.Bytes())
	return copyRecord
}

func (w *pluginCaptureSinkWriter) Write(p []byte) (int, error) {
	pluginTestSinkStore.withRecord(w.id, func(record *pluginSinkCaptureRecord) {
		_, _ = record.output.Write(p)
	})
	return len(p), nil
}

func (w *pluginCaptureSinkWriter) Close() error { return nil }

func registerExecutorSessionTestPlugin() {
	registerExecutorSessionTestPluginOnce.Do(func() {
		_ = coreplugin.Global().Register(&executorSessionTestPlugin{})
	})
}

func resetPluginPoolProbeOpenCount() {
	pluginPoolProbeOpenMu.Lock()
	defer pluginPoolProbeOpenMu.Unlock()
	pluginPoolProbeOpenCount = 0
}

func pluginPoolProbeOpenCountValue() int {
	pluginPoolProbeOpenMu.Lock()
	defer pluginPoolProbeOpenMu.Unlock()
	return pluginPoolProbeOpenCount
}
