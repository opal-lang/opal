package executor

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

type sessionIDCheckDecorator struct{}

type sessionBoundaryDecorator struct{}

type transportSessionCheckDecorator struct{}

type testSSHTransportDecorator struct{}

func (d *sessionIDCheckDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.sessionid.check").
		Summary("Checks active session identifier").
		Roles(decorator.RoleWrapper).
		Build()
}

func (d *sessionBoundaryDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.session.boundary").
		Summary("Runs a block with an overridden session transport ID").
		Roles(decorator.RoleWrapper).
		ParamString("id", "Session identifier to use in the block").
		Required().
		Done().
		Block(decorator.BlockRequired).
		Build()
}

func (d *transportSessionCheckDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.transport.session.check").
		Summary("Checks that execution uses a non-local transport session").
		Roles(decorator.RoleWrapper).
		Build()
}

func (d *testSSHTransportDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.transport.sshprobe").
		Summary("Provides a deterministic SSH-scoped transport session for tests").
		Roles(decorator.RoleBoundary).
		Build()
}

func (d *testSSHTransportDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork | decorator.TransportCapEnvironment
}

func (d *sessionIDCheckDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &sessionIDCheckNode{params: params}
}

func (d *sessionBoundaryDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &sessionBoundaryNode{next: next, params: params}
}

func (d *transportSessionCheckDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &transportSessionCheckNode{}
}

func (d *testSSHTransportDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	return &testSSHSession{parent: parent}, nil
}

func (d *testSSHTransportDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return next
}

type sessionIDCheckNode struct {
	params map[string]any
}

type sessionBoundaryNode struct {
	next   decorator.ExecNode
	params map[string]any
}

type transportSessionCheckNode struct{}

type testSSHSession struct {
	parent decorator.Session
}

func (n *sessionIDCheckNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	want, _ := n.params["expect"].(string)
	got := ctx.Session.ID()
	if diff := cmp.Diff(want, got); diff != "" {
		return decorator.Result{ExitCode: 99}, fmt.Errorf("session mismatch (-want +got):\n%s", diff)
	}
	return decorator.Result{ExitCode: 0}, nil
}

func (n *sessionBoundaryNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if n.next == nil {
		return decorator.Result{ExitCode: 0}, nil
	}

	id, _ := n.params["id"].(string)
	if id == "" {
		return decorator.Result{ExitCode: 1}, fmt.Errorf("missing session id")
	}

	child := ctx.WithSession(&transportScopedSession{id: id, session: ctx.Session})
	return n.next.Execute(child)
}

func (n *transportSessionCheckNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	wrapped, ok := ctx.Session.(*transportScopedSession)
	if !ok {
		return decorator.Result{ExitCode: 98}, fmt.Errorf("expected transportScopedSession, got %T", ctx.Session)
	}

	if _, isLocal := wrapped.session.(*decorator.LocalSession); isLocal {
		return decorator.Result{ExitCode: 98}, fmt.Errorf("expected transport session, got local session")
	}

	if diff := cmp.Diff(decorator.TransportScopeSSH, wrapped.TransportScope()); diff != "" {
		return decorator.Result{ExitCode: 98}, fmt.Errorf("transport scope mismatch (-want +got):\n%s", diff)
	}

	return decorator.Result{ExitCode: 0}, nil
}

func (s *testSSHSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *testSSHSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *testSSHSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}

func (s *testSSHSession) Env() map[string]string {
	return s.parent.Env()
}

func (s *testSSHSession) WithEnv(delta map[string]string) decorator.Session {
	return &testSSHSession{parent: s.parent.WithEnv(delta)}
}

func (s *testSSHSession) WithWorkdir(dir string) decorator.Session {
	return &testSSHSession{parent: s.parent.WithWorkdir(dir)}
}

func (s *testSSHSession) Cwd() string {
	return s.parent.Cwd()
}

func (s *testSSHSession) ID() string {
	return "ssh:probe"
}

func (s *testSSHSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeSSH
}

func (s *testSSHSession) Close() error {
	return nil
}

var (
	registerSessionIDCheckDecoratorOnce  sync.Once
	registerSessionBoundaryDecoratorOnce sync.Once
	registerTransportSessionCheckOnce    sync.Once
	registerTestSSHTransportOnce         sync.Once
)

func registerSessionIDCheckDecorator(t *testing.T) {
	t.Helper()
	var registerErr error
	registerSessionIDCheckDecoratorOnce.Do(func() {
		registerErr = decorator.Register("test.sessionid.check", &sessionIDCheckDecorator{})
	})
	if registerErr != nil {
		t.Fatalf("register test.sessionid.check: %v", registerErr)
	}
}

func registerSessionBoundaryDecorator(t *testing.T) {
	t.Helper()
	var registerErr error
	registerSessionBoundaryDecoratorOnce.Do(func() {
		registerErr = decorator.Register("test.session.boundary", &sessionBoundaryDecorator{})
	})
	if registerErr != nil {
		t.Fatalf("register test.session.boundary: %v", registerErr)
	}
}

func registerTransportSessionCheckDecorator(t *testing.T) {
	t.Helper()
	var registerErr error
	registerTransportSessionCheckOnce.Do(func() {
		registerErr = decorator.Register("test.transport.session.check", &transportSessionCheckDecorator{})
	})
	if registerErr != nil {
		t.Fatalf("register test.transport.session.check: %v", registerErr)
	}
}

func registerTestSSHTransportDecorator(t *testing.T) {
	t.Helper()
	var registerErr error
	registerTestSSHTransportOnce.Do(func() {
		registerErr = decorator.Register("test.transport.sshprobe", &testSSHTransportDecorator{})
	})
	if registerErr != nil {
		t.Fatalf("register test.transport.sshprobe: %v", registerErr)
	}
}

func TestExecuteRoutesSessionByTransportID(t *testing.T) {
	registerSessionIDCheckDecorator(t)

	plan := &planfmt.Plan{Target: "route-by-transport", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "local"}}, ""),
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:A"}}, "transport:A"),
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:A"}}, "transport:A"),
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:B"}}, "transport:B"),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestExecuteBlockInheritsWrapperSessionTransportID(t *testing.T) {
	registerSessionIDCheckDecorator(t)
	registerSessionBoundaryDecorator(t)

	plan := &planfmt.Plan{Target: "wrapper-session-inherit", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.CommandNode{
			Decorator: "@test.session.boundary",
			Args:      []planfmt.Arg{{Key: "id", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "transport:boundary"}}},
			Block: []planfmt.Step{{
				ID:   2,
				Tree: planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:boundary"}}, ""),
			}},
		},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestExecutePlan_TransportIDUsesTransportSession(t *testing.T) {
	registerTransportSessionCheckDecorator(t)
	registerTestSSHTransportDecorator(t)

	const (
		localTransportID = "local"
		sshTransportID   = "transport:ssh"
	)

	plan := &planfmt.Plan{
		Target: "transport-session-resolution",
		Transports: []planfmt.Transport{
			{ID: localTransportID, Decorator: "local", ParentID: ""},
			{ID: sshTransportID, Decorator: "@test.transport.sshprobe", ParentID: localTransportID},
		},
		Steps: []planfmt.Step{{
			ID: 1,
			Tree: &planfmt.CommandNode{
				Decorator:   "@test.transport.session.check",
				TransportID: sshTransportID,
			},
		}},
	}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func planExec(name string, args map[string]planfmt.Value, transportID string) *planfmt.CommandNode {
	planArgs := make([]planfmt.Arg, 0, len(args))
	for key, value := range args {
		planArgs = append(planArgs, planfmt.Arg{Key: key, Val: value})
	}

	return &planfmt.CommandNode{
		Decorator:   name,
		TransportID: transportID,
		Args:        planArgs,
	}
}

func TestSessionRuntimeReusesAndClosesSessions(t *testing.T) {
	factoryCalls := map[string]int{}
	stats := map[string]*decorator.SessionStats{}

	runtime := newSessionRuntime(func(transportID string) (decorator.Session, error) {
		factoryCalls[transportID]++
		wrapped := &transportScopedSession{id: transportID, session: decorator.NewLocalSession()}
		monitored := decorator.NewMonitoredSession(wrapped)
		stats[transportID] = monitored.Stats()
		return monitored, nil
	})

	sessionA1, err := runtime.SessionFor("transport:A")
	if err != nil {
		t.Fatalf("create A session: %v", err)
	}
	sessionA2, err := runtime.SessionFor("transport:A")
	if err != nil {
		t.Fatalf("reuse A session: %v", err)
	}
	_, err = runtime.SessionFor("transport:B")
	if err != nil {
		t.Fatalf("create B session: %v", err)
	}

	if sessionA1 != sessionA2 {
		t.Fatalf("session reuse mismatch: expected identical session instances")
	}

	wantCalls := map[string]int{"transport:A": 1, "transport:B": 1}
	if diff := cmp.Diff(wantCalls, factoryCalls); diff != "" {
		t.Fatalf("factory calls mismatch (-want +got):\n%s", diff)
	}

	runtime.Close()
	if diff := cmp.Diff(1, stats["transport:A"].CloseCalls); diff != "" {
		t.Fatalf("transport:A close mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, stats["transport:B"].CloseCalls); diff != "" {
		t.Fatalf("transport:B close mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionRuntimeFreezesEnvPerTransportOnFirstUse(t *testing.T) {
	t.Setenv("OPAL_FREEZE_PER_TRANSPORT", "first")

	runtime := newSessionRuntime(nil)
	defer runtime.Close()

	sessionA1, err := runtime.SessionFor("transport:A")
	if err != nil {
		t.Fatalf("create transport:A session: %v", err)
	}

	wantFirst := "first"
	if diff := cmp.Diff(wantFirst, sessionA1.Env()["OPAL_FREEZE_PER_TRANSPORT"]); diff != "" {
		t.Fatalf("transport:A initial env mismatch (-want +got):\n%s", diff)
	}

	t.Setenv("OPAL_FREEZE_PER_TRANSPORT", "second")

	sessionA2, err := runtime.SessionFor("transport:A")
	if err != nil {
		t.Fatalf("reuse transport:A session: %v", err)
	}
	if sessionA1 != sessionA2 {
		t.Fatalf("transport:A should reuse frozen session instance")
	}
	if diff := cmp.Diff(wantFirst, sessionA2.Env()["OPAL_FREEZE_PER_TRANSPORT"]); diff != "" {
		t.Fatalf("transport:A should keep frozen env (-want +got):\n%s", diff)
	}

	sessionB, err := runtime.SessionFor("transport:B")
	if err != nil {
		t.Fatalf("create transport:B session: %v", err)
	}
	wantSecond := "second"
	if diff := cmp.Diff(wantSecond, sessionB.Env()["OPAL_FREEZE_PER_TRANSPORT"]); diff != "" {
		t.Fatalf("transport:B env mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionRuntimeFreezesWorkdirPerTransportOnFirstUse(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd)
	})

	dirA := t.TempDir()
	dirB := t.TempDir()

	if err := os.Chdir(dirA); err != nil {
		t.Fatalf("chdir dirA: %v", err)
	}

	runtime := newSessionRuntime(nil)
	defer runtime.Close()

	sessionA1, err := runtime.SessionFor("transport:A")
	if err != nil {
		t.Fatalf("create transport:A session: %v", err)
	}

	if err := os.Chdir(dirB); err != nil {
		t.Fatalf("chdir dirB: %v", err)
	}

	sessionA2, err := runtime.SessionFor("transport:A")
	if err != nil {
		t.Fatalf("reuse transport:A session: %v", err)
	}
	sessionB, err := runtime.SessionFor("transport:B")
	if err != nil {
		t.Fatalf("create transport:B session: %v", err)
	}

	if sessionA1 != sessionA2 {
		t.Fatalf("transport:A should reuse same session instance")
	}
	if diff := cmp.Diff(dirA, sessionA1.Cwd()); diff != "" {
		t.Fatalf("transport:A cwd mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(dirA, sessionA2.Cwd()); diff != "" {
		t.Fatalf("transport:A reused cwd mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(dirB, sessionB.Cwd()); diff != "" {
		t.Fatalf("transport:B cwd mismatch (-want +got):\n%s", diff)
	}
}
