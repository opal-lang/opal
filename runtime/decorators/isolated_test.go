package decorators

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/runtime/isolation"
	"github.com/google/go-cmp/cmp"
)

func TestIsolatedDecoratorRegistered(t *testing.T) {
	if decorator.Global().IsRegistered("isolated") {
		t.Error("decorator 'isolated' should NOT be registered - it's a namespace now")
	}
}

func TestIsolatedTransportCapabilities(t *testing.T) {
	dec := &IsolatedTransportDecorator{}
	got := dec.Capabilities()

	if !got.Has(decorator.TransportCapIsolation) {
		t.Fatal("expected isolation capability")
	}

	if !got.Has(decorator.TransportCapNetwork) {
		t.Fatal("expected network capability")
	}
}

func TestIsolatedIsolationContextUsesFactoryType(t *testing.T) {
	dec := &IsolatedTransportDecorator{}
	ctx := dec.IsolationContext()
	if ctx == nil {
		t.Fatal("expected non-nil isolation context")
	}

	if diff := cmp.Diff(fmt.Sprintf("%T", isolation.NewIsolator()), fmt.Sprintf("%T", ctx)); diff != "" {
		t.Fatalf("isolation context type mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedOpenLevelNoneCreatesIsolatedSession(t *testing.T) {
	dec := &IsolatedTransportDecorator{}
	parent := decorator.NewLocalSession()

	session, err := dec.Open(parent, map[string]any{"level": "none"})
	if err != nil {
		t.Fatalf("open isolated session: %v", err)
	}

	if diff := cmp.Diff(decorator.TransportScopeIsolated, session.TransportScope()); diff != "" {
		t.Fatalf("transport scope mismatch (-want +got):\n%s", diff)
	}

	if !strings.HasSuffix(session.ID(), "/isolated") {
		t.Fatalf("expected isolated session id suffix, got %q", session.ID())
	}

	result, runErr := session.Run(context.Background(), []string{"sh", "-c", "printf isolated-ok"}, decorator.RunOpts{})
	if runErr != nil {
		t.Fatalf("run through isolated session: %v", runErr)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("isolated-ok", string(result.Stdout)); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedOpenStandardAppliesIsolation(t *testing.T) {
	if os.Getenv("OPAL_ISOLATED_HELPER") == "standard" {
		dec := &IsolatedTransportDecorator{}
		parent := decorator.NewLocalSession()

		_, err := dec.Open(parent, map[string]any{"level": "standard", "network": "allow"})
		if err != nil {
			if canSkipIsolatedError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			fmt.Printf("ERR:%v\n", err)
			os.Exit(1)
		}

		fmt.Print("OK:isolated-standard-applied\n")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestIsolatedOpenStandardAppliesIsolation$")
	cmd.Env = append(os.Environ(), "OPAL_ISOLATED_HELPER=standard")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "SKIP:") {
			t.Skip(strings.TrimSpace(string(out)))
		}
		t.Fatalf("helper failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	if strings.Contains(output, "SKIP:") {
		t.Skip(strings.TrimSpace(output))
	}
	if !strings.Contains(output, "OK:isolated-standard-applied") {
		t.Fatalf("expected helper success output, got:\n%s", output)
	}
}

func canSkipIsolatedError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.ENOSYS) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "operation not permitted") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "read-only file system")
}

func TestIsolatedSessionDialContextLoopbackPolicyBlocksExternalAddress(t *testing.T) {
	originalLookup := lookupIP
	lookupIP = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	t.Cleanup(func() {
		lookupIP = originalLookup
	})

	parent := &recordingDialerSession{}
	session := &isolatedSession{
		parent:        parent,
		networkPolicy: decorator.NetworkPolicyLoopbackOnly,
	}

	_, err := session.DialContext(context.Background(), "tcp", "example.com:443")
	if err == nil {
		t.Fatal("expected external network dial to be blocked")
	}

	if diff := cmp.Diff("network access denied by isolation policy: example.com:443 resolves to non-loopback IP 93.184.216.34", err.Error()); diff != "" {
		t.Fatalf("dial error mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(0, parent.dialCalls); diff != "" {
		t.Fatalf("dial call count mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedSessionDialContextDenyPolicyReturnsError(t *testing.T) {
	parent := &recordingDialerSession{}
	session := &isolatedSession{
		parent:        parent,
		networkPolicy: decorator.NetworkPolicyDeny,
	}

	_, err := session.DialContext(context.Background(), "tcp", "127.0.0.1:8080")
	if err == nil {
		t.Fatal("expected deny policy to reject network dial")
	}

	if diff := cmp.Diff("network access denied by isolation policy", err.Error()); diff != "" {
		t.Fatalf("dial error mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(0, parent.dialCalls); diff != "" {
		t.Fatalf("dial call count mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedSessionDialContextLoopbackPolicyAllowsLoopbackIP(t *testing.T) {
	originalLookup := lookupIP
	lookupIP = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	t.Cleanup(func() {
		lookupIP = originalLookup
	})

	conn := &stubConn{}
	parent := &recordingDialerSession{dialConn: conn}
	session := &isolatedSession{
		parent:        parent,
		networkPolicy: decorator.NetworkPolicyLoopbackOnly,
	}

	gotConn, err := session.DialContext(context.Background(), "tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("dial through isolated session: %v", err)
	}

	if diff := cmp.Diff(conn, gotConn); diff != "" {
		t.Fatalf("dialed connection mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, parent.dialCalls); diff != "" {
		t.Fatalf("dial call count mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedSessionDialContextLoopbackPolicyRejectsRebindingCandidates(t *testing.T) {
	originalLookup := lookupIP
	lookupIP = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("8.8.8.8")}, nil
	}
	t.Cleanup(func() {
		lookupIP = originalLookup
	})

	parent := &recordingDialerSession{}
	session := &isolatedSession{
		parent:        parent,
		networkPolicy: decorator.NetworkPolicyLoopbackOnly,
	}

	_, err := session.DialContext(context.Background(), "tcp", "localhost:443")
	if err == nil {
		t.Fatal("expected mixed loopback/external resolution to be blocked")
	}

	if diff := cmp.Diff("network access denied by isolation policy: localhost:443 resolves to non-loopback IP 8.8.8.8", err.Error()); diff != "" {
		t.Fatalf("dial error mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(0, parent.dialCalls); diff != "" {
		t.Fatalf("dial call count mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedSessionDialContextAllowPolicyDelegatesToParentDialer(t *testing.T) {
	conn := &stubConn{}
	parent := &recordingDialerSession{dialConn: conn}
	session := &isolatedSession{
		parent:        parent,
		networkPolicy: decorator.NetworkPolicyAllow,
	}

	gotConn, err := session.DialContext(context.Background(), "tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("dial through isolated session: %v", err)
	}

	if diff := cmp.Diff(conn, gotConn); diff != "" {
		t.Fatalf("dialed connection mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, parent.dialCalls); diff != "" {
		t.Fatalf("dial call count mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("tcp", parent.lastNetwork); diff != "" {
		t.Fatalf("dial network mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("127.0.0.1:8080", parent.lastAddr); diff != "" {
		t.Fatalf("dial address mismatch (-want +got):\n%s", diff)
	}
}

type recordingDialerSession struct {
	dialCalls   int
	lastNetwork string
	lastAddr    string
	dialConn    net.Conn
	dialErr     error
}

func (s *recordingDialerSession) Run(context.Context, []string, decorator.RunOpts) (decorator.Result, error) {
	return decorator.Result{}, nil
}

func (s *recordingDialerSession) Put(context.Context, []byte, string, fs.FileMode) error {
	return nil
}

func (s *recordingDialerSession) Get(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (s *recordingDialerSession) Env() map[string]string {
	return map[string]string{}
}

func (s *recordingDialerSession) WithEnv(map[string]string) decorator.Session {
	return s
}

func (s *recordingDialerSession) WithWorkdir(string) decorator.Session {
	return s
}

func (s *recordingDialerSession) Cwd() string {
	return ""
}

func (s *recordingDialerSession) ID() string {
	return "recording"
}

func (s *recordingDialerSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (s *recordingDialerSession) Platform() string {
	return "linux"
}

func (s *recordingDialerSession) Close() error {
	return nil
}

func (s *recordingDialerSession) DialContext(_ context.Context, network, addr string) (net.Conn, error) {
	s.dialCalls++
	s.lastNetwork = network
	s.lastAddr = addr
	if s.dialErr != nil {
		return nil, s.dialErr
	}
	if s.dialConn == nil {
		return &stubConn{}, nil
	}
	return s.dialConn, nil
}

func (s *recordingDialerSession) NetworkDialer() decorator.NetworkDialer {
	return s
}

type stubConn struct{}

func (c *stubConn) Read([]byte) (int, error)         { return 0, nil }
func (c *stubConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c *stubConn) Close() error                     { return nil }
func (c *stubConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (c *stubConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (c *stubConn) SetDeadline(time.Time) error      { return nil }
func (c *stubConn) SetReadDeadline(time.Time) error  { return nil }
func (c *stubConn) SetWriteDeadline(time.Time) error { return nil }
