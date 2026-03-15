package executor

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/planfmt"
	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/google/go-cmp/cmp"
)

type multiHopRootSession struct{}

var (
	multiHopDialMu    sync.Mutex
	multiHopDialCalls []string
)

type closeOrderSession struct {
	id        string
	orderMu   *sync.Mutex
	order     *[]string
	closeErr  error
	closeCall int
}

func (s *multiHopRootSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return decorator.Result{ExitCode: 0}, nil
}

func (s *multiHopRootSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (s *multiHopRootSession) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (s *multiHopRootSession) Env() map[string]string {
	return map[string]string{}
}

func (s *multiHopRootSession) WithEnv(delta map[string]string) decorator.Session {
	return s
}

func (s *multiHopRootSession) WithWorkdir(dir string) decorator.Session {
	return s
}

func (s *multiHopRootSession) Cwd() string {
	return ""
}

func (s *multiHopRootSession) Platform() string {
	return ""
}

func (s *multiHopRootSession) ID() string {
	return "local"
}

func (s *multiHopRootSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (s *multiHopRootSession) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	recordMultiHopDial("local", addr)
	clientConn, serverConn := net.Pipe()
	_ = serverConn.Close()
	return clientConn, nil
}

func (s *multiHopRootSession) Close() error {
	return nil
}

func (s *closeOrderSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return decorator.Result{ExitCode: 0}, nil
}

func (s *closeOrderSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (s *closeOrderSession) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (s *closeOrderSession) Env() map[string]string {
	return nil
}

func (s *closeOrderSession) WithEnv(delta map[string]string) decorator.Session {
	return s
}

func (s *closeOrderSession) WithWorkdir(dir string) decorator.Session {
	return s
}

func (s *closeOrderSession) Cwd() string {
	return ""
}

func (s *closeOrderSession) Platform() string {
	return ""
}

func (s *closeOrderSession) ID() string {
	return s.id
}

func (s *closeOrderSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (s *closeOrderSession) Close() error {
	s.orderMu.Lock()
	defer s.orderMu.Unlock()
	*s.order = append(*s.order, s.id)
	s.closeCall++
	return s.closeErr
}

func multiHopRootSessionFactory(transportID string) (decorator.Session, error) {
	_ = transportID
	return &multiHopRootSession{}, nil
}

func resetMultiHopDialCalls() {
	multiHopDialMu.Lock()
	defer multiHopDialMu.Unlock()
	multiHopDialCalls = nil
}

func recordMultiHopDial(sourceID, addr string) {
	multiHopDialMu.Lock()
	defer multiHopDialMu.Unlock()
	multiHopDialCalls = append(multiHopDialCalls, sourceID+"->"+addr)
}

func multiHopDialCallsValue() []string {
	multiHopDialMu.Lock()
	defer multiHopDialMu.Unlock()
	return append([]string(nil), multiHopDialCalls...)
}

func TestNestedTransportUsesParentNetworkDialer(t *testing.T) {
	registerParentDialerTestPlugin()

	t.Run("happy path delegates nested dial through parent", func(t *testing.T) {
		resetMultiHopDialCalls()

		runtime := newSessionRuntime(multiHopRootSessionFactory)
		defer runtime.Close()
		runtime.registerPlanTransports([]planfmt.Transport{
			{ID: "local", Decorator: "local", ParentID: ""},
			{ID: "transport:bastion", Decorator: "@test.transport.multihop", ParentID: "local", Args: []planfmt.Arg{{Key: "addr", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "bastion.internal:22"}}, {Key: "id", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "session:bastion"}}}},
			{ID: "transport:internal", Decorator: "@test.transport.multihop", ParentID: "transport:bastion", Args: []planfmt.Arg{{Key: "addr", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "internal.internal:22"}}, {Key: "id", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "session:internal"}}}},
		})

		bastion, err := runtime.SessionFor("transport:bastion")
		if err != nil {
			t.Fatalf("session for transport:bastion: %v", err)
		}
		if diff := cmp.Diff("transport:bastion", bastion.ID()); diff != "" {
			t.Fatalf("bastion session id mismatch (-want +got):\n%s", diff)
		}

		internal, err := runtime.SessionFor("transport:internal")
		if err != nil {
			t.Fatalf("session for transport:internal: %v", err)
		}
		if diff := cmp.Diff("transport:internal", internal.ID()); diff != "" {
			t.Fatalf("internal session id mismatch (-want +got):\n%s", diff)
		}

		wantDials := []string{
			"local->bastion.internal:22",
			"session:bastion->internal.internal:22",
		}
		if diff := cmp.Diff(wantDials, multiHopDialCallsValue()); diff != "" {
			t.Fatalf("dial delegation mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("error when parent session has no NetworkDialer", func(t *testing.T) {
		registerParentDialerTestPlugin()
		runtime := newSessionRuntime(multiHopRootSessionFactory)
		defer runtime.Close()

		runtime.registerPlanTransports([]planfmt.Transport{
			{ID: "local", Decorator: "local", ParentID: ""},
			{ID: "transport:no-dialer", Decorator: "@test.transport.nodialer", ParentID: "local", Args: []planfmt.Arg{{Key: "id", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "session:no-dialer"}}}},
			{ID: "transport:child", Decorator: "@test.transport.multihop", ParentID: "transport:no-dialer", Args: []planfmt.Arg{{Key: "addr", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "internal.internal:22"}}, {Key: "id", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "session:child"}}}},
		})

		_, err := runtime.SessionFor("transport:no-dialer")
		if err != nil {
			t.Fatalf("session for transport:no-dialer: %v", err)
		}

		_, err = runtime.SessionFor("transport:child")
		if err == nil {
			t.Fatal("expected session creation error")
		}

		wantErr := "failed to create session for transport \"transport:child\": open plugin transport \"@test.transport.multihop\": parent transport executor.pluginParentSession does not implement NetworkDialer"
		if diff := cmp.Diff(wantErr, err.Error()); diff != "" {
			t.Fatalf("error mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExecuteRoutesSessionByTransportID(t *testing.T) {
	registerExecutorSessionTestPlugin()

	plan := &planfmt.Plan{Target: "route-by-transport", Transports: localTestTransports("transport:A", "transport:B"), Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "local"}}, ""),
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:A"}}, "transport:A"),
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:A"}}, "transport:A"),
			planExec("@test.sessionid.check", map[string]planfmt.Value{"expect": {Kind: planfmt.ValueString, Str: "transport:B"}}, "transport:B"),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: scopedLocalSessionFactory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestExecuteBlockInheritsWrapperSessionTransportID(t *testing.T) {
	registerExecutorSessionTestPlugin()

	plan := &planfmt.Plan{Target: "wrapper-session-inherit", Transports: localTestTransports("transport:boundary"), Steps: []planfmt.Step{{
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

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: scopedLocalSessionFactory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestExecutePlan_TransportIDUsesTransportSession(t *testing.T) {
	registerExecutorSessionTestPlugin()

	const (
		localTransportID = "local"
		sshTransportID   = "transport:ssh"
	)

	plan := &planfmt.Plan{
		Target: "transport-session-resolution",
		Transports: []planfmt.Transport{
			{ID: localTransportID, Decorator: "local", ParentID: ""},
			{ID: sshTransportID, Decorator: "@test.transport", ParentID: localTransportID},
		},
		Steps: []planfmt.Step{{
			ID: 1,
			Tree: &planfmt.CommandNode{
				Decorator:   "@test.sessionid.check",
				TransportID: sshTransportID,
				Args:        []planfmt.Arg{{Key: "expect", Val: planfmt.Value{Kind: planfmt.ValueString, Str: sshTransportID}}},
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

func TestExecutePlan_ReusesPooledTransportSessionPerTarget(t *testing.T) {
	registerExecutorSessionTestPlugin()
	resetPluginPoolProbeOpenCount()

	const (
		localTransportID = "local"
		childA1          = "transport:ssh:A1"
		childA2          = "transport:ssh:A2"
	)

	transportArgs := []planfmt.Arg{
		{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}},
		{Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}},
	}

	plan := &planfmt.Plan{
		Target: "transport-pool-integration",
		Transports: []planfmt.Transport{
			{ID: localTransportID, Decorator: "local", ParentID: ""},
			{ID: childA1, Decorator: "@test.transport.poolprobe", ParentID: localTransportID, Args: transportArgs},
			{ID: childA2, Decorator: "@test.transport.poolprobe", ParentID: localTransportID, Args: transportArgs},
		},
		Steps: []planfmt.Step{{
			ID: 1,
			Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
				&planfmt.CommandNode{Decorator: "@test.transport.poolprobe", TransportID: localTransportID, Args: transportArgs, Block: []planfmt.Step{{ID: 2, Tree: &planfmt.CommandNode{Decorator: "@test.sessionid.check", TransportID: childA1, Args: []planfmt.Arg{{Key: "expect", Val: planfmt.Value{Kind: planfmt.ValueString, Str: childA1}}}}}}},
				&planfmt.CommandNode{Decorator: "@test.transport.poolprobe", TransportID: localTransportID, Args: transportArgs, Block: []planfmt.Step{{ID: 3, Tree: &planfmt.CommandNode{Decorator: "@test.sessionid.check", TransportID: childA2, Args: []planfmt.Arg{{Key: "expect", Val: planfmt.Value{Kind: planfmt.ValueString, Str: childA2}}}}}}},
			}},
		}},
	}

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: scopedLocalSessionFactory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, pluginPoolProbeOpenCountValue()); diff != "" {
		t.Fatalf("pool open count mismatch (-want +got):\n%s", diff)
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
	runtime.registerPlanTransports(localTestTransports("transport:A", "transport:B"))

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

	runtime := newSessionRuntime(scopedLocalSessionFactory)
	defer runtime.Close()
	runtime.registerPlanTransports(localTestTransports("transport:A", "transport:B"))

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

	runtime := newSessionRuntime(scopedLocalSessionFactory)
	defer runtime.Close()
	runtime.registerPlanTransports(localTestTransports("transport:A", "transport:B"))

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

func TestSessionRuntimeReturnsErrorForUnknownTransportID(t *testing.T) {
	runtime := newSessionRuntime(scopedLocalSessionFactory)
	defer runtime.Close()

	_, err := runtime.SessionFor("transport:missing")
	if err == nil {
		t.Fatalf("expected unknown transport error")
	}

	if diff := cmp.Diff(true, strings.Contains(err.Error(), "unknown transport \"transport:missing\": transport not registered")); diff != "" {
		t.Fatalf("error mismatch (-want +got):\n%s\nerr: %q", diff, err.Error())
	}
}

func TestSessionRuntimePoolsByParentAndParams(t *testing.T) {
	registerExecutorSessionTestPlugin()
	resetPluginPoolProbeOpenCount()

	runtime := newSessionRuntime(scopedLocalSessionFactory)
	defer runtime.Close()
	runtime.registerPlanTransports([]planfmt.Transport{
		{ID: "parent:A", Decorator: "local", ParentID: ""},
		{ID: "parent:B", Decorator: "local", ParentID: ""},
		{ID: "child:A1", Decorator: "@test.transport.poolprobe", ParentID: "parent:A", Args: []planfmt.Arg{{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}}, {Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}}}},
		{ID: "child:A2", Decorator: "@test.transport.poolprobe", ParentID: "parent:A", Args: []planfmt.Arg{{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}}, {Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}}}},
		{ID: "child:B1", Decorator: "@test.transport.poolprobe", ParentID: "parent:B", Args: []planfmt.Arg{{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}}, {Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}}}},
	})

	first, err := runtime.SessionFor("child:A1")
	if err != nil {
		t.Fatalf("session for child:A1: %v", err)
	}
	second, err := runtime.SessionFor("child:A2")
	if err != nil {
		t.Fatalf("session for child:A2: %v", err)
	}
	third, err := runtime.SessionFor("child:B1")
	if err != nil {
		t.Fatalf("session for child:B1: %v", err)
	}

	firstScoped, ok := first.(*transportScopedSession)
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Fatalf("child:A1 wrapper type mismatch (-want +got):\n%s", diff)
	}
	secondScoped, ok := second.(*transportScopedSession)
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Fatalf("child:A2 wrapper type mismatch (-want +got):\n%s", diff)
	}
	thirdScoped, ok := third.(*transportScopedSession)
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Fatalf("child:B1 wrapper type mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(true, firstScoped.session == secondScoped.session); diff != "" {
		t.Fatalf("same parent+params should reuse base session (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, firstScoped.session == thirdScoped.session); diff != "" {
		t.Fatalf("different parent should not reuse base session (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, pluginPoolProbeOpenCountValue()); diff != "" {
		t.Fatalf("pool open count mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionPoolKeyIncludesAuthFingerprint(t *testing.T) {
	parent := &transportScopedSession{id: "parent:A", session: decorator.NewLocalSession()}

	baseParams := map[string]any{
		"host":            "example.internal",
		"port":            22,
		"user":            "alice",
		"key":             "/home/alice/.ssh/id_ed25519",
		"strict_host_key": true,
	}

	baseKey := SessionPoolKey(parent, baseParams)
	if diff := cmp.Diff(true, strings.HasPrefix(baseKey, "parent:A:")); diff != "" {
		t.Fatalf("pool key parent prefix mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, strings.Contains(baseKey, "/home/alice/.ssh/id_ed25519")); diff != "" {
		t.Fatalf("pool key should not expose key path (-want +got):\n%s\nkey: %q", diff, baseKey)
	}

	passwordParams := map[string]any{
		"host":            "example.internal",
		"port":            22,
		"user":            "alice",
		"password":        "super-secret-password",
		"strict_host_key": true,
	}
	passwordKey := SessionPoolKey(parent, passwordParams)
	if diff := cmp.Diff(false, strings.Contains(passwordKey, "super-secret-password")); diff != "" {
		t.Fatalf("pool key should not expose password (-want +got):\n%s\nkey: %q", diff, passwordKey)
	}

	differentPasswordParams := map[string]any{
		"host":            "example.internal",
		"port":            22,
		"user":            "alice",
		"password":        "different-secret-password",
		"strict_host_key": true,
	}
	differentPasswordKey := SessionPoolKey(parent, differentPasswordParams)
	if diff := cmp.Diff(false, passwordKey == differentPasswordKey); diff != "" {
		t.Fatalf("different passwords should change key (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, strings.Contains(differentPasswordKey, "different-secret-password")); diff != "" {
		t.Fatalf("pool key should not expose password (-want +got):\n%s\nkey: %q", diff, differentPasswordKey)
	}

	userChanged := map[string]any{
		"host":            "example.internal",
		"port":            22,
		"user":            "bob",
		"key":             "/home/alice/.ssh/id_ed25519",
		"strict_host_key": true,
	}
	if diff := cmp.Diff(false, baseKey == SessionPoolKey(parent, userChanged)); diff != "" {
		t.Fatalf("different auth user should change key (-want +got):\n%s", diff)
	}

	hostPolicyChanged := map[string]any{
		"host":            "example.internal",
		"port":            22,
		"user":            "alice",
		"key":             "/home/alice/.ssh/id_ed25519",
		"strict_host_key": false,
	}
	if diff := cmp.Diff(false, baseKey == SessionPoolKey(parent, hostPolicyChanged)); diff != "" {
		t.Fatalf("different host key policy should change key (-want +got):\n%s", diff)
	}

	methodChanged := map[string]any{
		"host":            "example.internal",
		"port":            22,
		"user":            "alice",
		"password":        "another-secret",
		"strict_host_key": true,
	}
	if diff := cmp.Diff(false, baseKey == SessionPoolKey(parent, methodChanged)); diff != "" {
		t.Fatalf("different auth method should change key (-want +got):\n%s", diff)
	}
}

func TestSessionRuntimePoolsByParentAndAuthFingerprint(t *testing.T) {
	registerExecutorSessionTestPlugin()
	resetPluginPoolProbeOpenCount()

	runtime := newSessionRuntime(scopedLocalSessionFactory)
	defer runtime.Close()
	runtime.registerPlanTransports([]planfmt.Transport{
		{ID: "parent:A", Decorator: "local", ParentID: ""},
		{ID: "child:key:A1", Decorator: "@test.transport.poolprobe", ParentID: "parent:A", Args: []planfmt.Arg{{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}}, {Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}}, {Key: "user", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "alice"}}, {Key: "key", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "/home/alice/.ssh/id_ed25519"}}, {Key: "strict_host_key", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}}}},
		{ID: "child:key:A2", Decorator: "@test.transport.poolprobe", ParentID: "parent:A", Args: []planfmt.Arg{{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}}, {Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}}, {Key: "user", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "alice"}}, {Key: "key", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "/home/alice/.ssh/id_ed25519"}}, {Key: "strict_host_key", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}}}},
		{ID: "child:password:B1", Decorator: "@test.transport.poolprobe", ParentID: "parent:A", Args: []planfmt.Arg{{Key: "host", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "example.internal"}}, {Key: "port", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 22}}, {Key: "user", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "alice"}}, {Key: "password", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "super-secret-password"}}, {Key: "strict_host_key", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}}}},
	})

	first, err := runtime.SessionFor("child:key:A1")
	if err != nil {
		t.Fatalf("session for child:key:A1: %v", err)
	}
	second, err := runtime.SessionFor("child:key:A2")
	if err != nil {
		t.Fatalf("session for child:key:A2: %v", err)
	}
	third, err := runtime.SessionFor("child:password:B1")
	if err != nil {
		t.Fatalf("session for child:password:B1: %v", err)
	}

	firstScoped, ok := first.(*transportScopedSession)
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Fatalf("child:key:A1 wrapper type mismatch (-want +got):\n%s", diff)
	}
	secondScoped, ok := second.(*transportScopedSession)
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Fatalf("child:key:A2 wrapper type mismatch (-want +got):\n%s", diff)
	}
	thirdScoped, ok := third.(*transportScopedSession)
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Fatalf("child:password:B1 wrapper type mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(true, firstScoped.session == secondScoped.session); diff != "" {
		t.Fatalf("same parent+auth should reuse base session (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, firstScoped.session == thirdScoped.session); diff != "" {
		t.Fatalf("different auth should use different base session (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, pluginPoolProbeOpenCountValue()); diff != "" {
		t.Fatalf("pool open count mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionRuntimeCloseUsesPostorderTraversal(t *testing.T) {
	runtime := newSessionRuntime(nil)
	runtime.registerPlanTransports([]planfmt.Transport{
		{ID: "local", Decorator: "local", ParentID: ""},
		{ID: "transport:parent", Decorator: "local", ParentID: "local"},
		{ID: "transport:child", Decorator: "@test.transport.poolprobe", ParentID: "transport:parent"},
	})

	order := []string{}
	orderMu := &sync.Mutex{}

	parentSession := &closeOrderSession{id: "transport:parent", orderMu: orderMu, order: &order}
	childBaseSession := &closeOrderSession{id: "transport:child(base)", orderMu: orderMu, order: &order}

	runtime.direct["transport:parent"] = parentSession
	runtime.sessions["transport:parent"] = parentSession
	runtime.pooled["pool:child"] = childBaseSession
	runtime.sessions["transport:child"] = &transportScopedSession{id: "transport:child", session: childBaseSession}

	runtime.Close()

	wantOrder := []string{"transport:child(base)", "transport:parent"}
	if diff := cmp.Diff(wantOrder, order); diff != "" {
		t.Fatalf("close order mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, parentSession.closeCall); diff != "" {
		t.Fatalf("parent close call count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, childBaseSession.closeCall); diff != "" {
		t.Fatalf("child close call count mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionRuntimeCloseContinuesAfterCloseError(t *testing.T) {
	runtime := newSessionRuntime(nil)
	runtime.registerPlanTransports([]planfmt.Transport{
		{ID: "local", Decorator: "local", ParentID: ""},
		{ID: "transport:parent", Decorator: "local", ParentID: "local"},
		{ID: "transport:child", Decorator: "@test.transport.poolprobe", ParentID: "transport:parent"},
	})

	order := []string{}
	orderMu := &sync.Mutex{}

	parentSession := &closeOrderSession{id: "transport:parent", orderMu: orderMu, order: &order}
	childBaseSession := &closeOrderSession{id: "transport:child(base)", orderMu: orderMu, order: &order, closeErr: errors.New("child close failed")}

	runtime.direct["transport:parent"] = parentSession
	runtime.sessions["transport:parent"] = parentSession
	runtime.pooled["pool:child"] = childBaseSession
	runtime.sessions["transport:child"] = &transportScopedSession{id: "transport:child", session: childBaseSession}

	var logBuf bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
	})

	runtime.Close()

	if diff := cmp.Diff(1, parentSession.closeCall); diff != "" {
		t.Fatalf("parent close call count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, childBaseSession.closeCall); diff != "" {
		t.Fatalf("child close call count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, strings.Contains(logBuf.String(), "session runtime: close transport \"transport:child\": child close failed")); diff != "" {
		t.Fatalf("close error log mismatch (-want +got):\n%s\nlog: %q", diff, logBuf.String())
	}
}
