package decorator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/ssh"
)

var sshServer *SSHTestServer

// TestMain sets up and tears down the SSH test server for all tests.
func TestMain(m *testing.M) {
	// Create SSH server once for all tests
	// We use a fake testing.T just to satisfy the API
	fakeT := &testing.T{}
	sshServer = StartSSHTestServer(fakeT)

	// Run all tests
	code := m.Run()

	// Cleanup
	if sshServer != nil {
		sshServer.Stop()
	}

	os.Exit(code)
}

// getSSHTestServer returns the shared SSH test server.
func getSSHTestServer(t *testing.T) *SSHTestServer {
	t.Helper()
	return sshServer
}

func TestTransportCaps(t *testing.T) {
	transport := &SSHTransport{}
	caps := transport.Capabilities()

	if !caps.Has(TransportCapNetwork) {
		t.Fatalf("expected SSH transport to have network capability")
	}

	if !caps.Has(TransportCapEnvironment) {
		t.Fatalf("expected SSH transport to have environment capability")
	}

	if caps.Has(TransportCapIsolation) {
		t.Fatalf("did not expect SSH transport isolation capability")
	}

	if caps.Has(TransportCapFilesystem) {
		t.Fatalf("did not expect SSH transport filesystem capability")
	}
}

// TestSSHSessionRun tests real SSH connection to pure Go server
func TestSSHSessionRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	// Create SSH session using pure Go server
	session, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false, // Pass Signer directly
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Test Run()
	result, err := session.Run(context.Background(), []string{"echo", "hello"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "hello" {
		t.Errorf("Output: got %q, want %q", output, "hello")
	}
}

func TestNewSSHSession_AcceptsInt64Port(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	session, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": int64(server.Port),
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session with int64 port: %v", err)
	}
	defer session.Close()

	result, err := session.Run(context.Background(), []string{"echo", "int64-port"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "int64-port" {
		t.Errorf("Output: got %q, want %q", output, "int64-port")
	}
}

// TestSSHSessionEnv tests reading remote environment
func TestSSHSessionEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	session, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Get remote environment
	env := session.Env()

	// Should have common env vars
	if env["HOME"] == "" {
		t.Error("Remote HOME is empty")
	}

	if env["USER"] == "" {
		t.Error("Remote USER is empty")
	}

	// Verify it matches what we expect
	expectedUser := os.Getenv("USER")
	if env["USER"] != expectedUser {
		t.Errorf("Remote USER: got %q, want %q", env["USER"], expectedUser)
	}
}

// TestSSHSessionIsolation tests that SSH sessions are isolated from local
func TestSSHSessionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	// Create local session
	local := NewLocalSession()
	localModified := local.WithEnv(map[string]string{
		"OPAL_TEST_VAR": "local_value",
	})

	// Create SSH session
	ssh, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer ssh.Close()

	// Verify local session has the var
	localEnv := localModified.Env()
	if localEnv["OPAL_TEST_VAR"] != "local_value" {
		t.Errorf("Local OPAL_TEST_VAR: got %q, want %q", localEnv["OPAL_TEST_VAR"], "local_value")
	}

	// Verify SSH session does NOT have the var
	sshEnv := ssh.Env()
	if _, ok := sshEnv["OPAL_TEST_VAR"]; ok {
		t.Error("SSH session has OPAL_TEST_VAR from local session (leaked!)")
	}
}

// TestSSHSessionPooling tests that sessions are reused
func TestSSHSessionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	pool := NewSessionPool()
	defer pool.CloseAll()

	transport := NewMonitoredTransport(&SSHTransport{})
	parent := NewLocalSession()
	params := map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	}

	// First call creates session
	session1, err := pool.GetOrCreate(transport, parent, params)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Second call reuses session
	session2, err := pool.GetOrCreate(transport, parent, params)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Verify only one Open call
	if transport.OpenCalls != 1 {
		t.Errorf("OpenCalls: got %d, want 1 (should reuse)", transport.OpenCalls)
	}

	// Verify same session instance
	if session1 != session2 {
		t.Error("Expected same session instance for same params")
	}
}

// TestSSHSessionEnvironmentDifferentFromLocal verifies SSH session has different environment
func TestSSHSessionEnvironmentDifferentFromLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	// Get local session environment
	local := NewLocalSession()
	localEnv := local.Env()

	// Get SSH session environment
	ssh, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer ssh.Close()

	sshEnv := ssh.Env()

	// Both should have USER and HOME
	if localEnv["USER"] == "" {
		t.Error("Local USER is empty")
	}
	if sshEnv["USER"] == "" {
		t.Error("SSH USER is empty")
	}
	if localEnv["HOME"] == "" {
		t.Error("Local HOME is empty")
	}
	if sshEnv["HOME"] == "" {
		t.Error("SSH HOME is empty")
	}

	// USER should be the same (same user on both)
	if localEnv["USER"] != sshEnv["USER"] {
		t.Errorf("USER mismatch: local=%q ssh=%q", localEnv["USER"], sshEnv["USER"])
	}

	// HOME should be the same (same user, same machine)
	if localEnv["HOME"] != sshEnv["HOME"] {
		t.Errorf("HOME mismatch: local=%q ssh=%q", localEnv["HOME"], sshEnv["HOME"])
	}

	// PWD might be different (SSH starts in HOME, local might be elsewhere)
	localPwd := localEnv["PWD"]
	sshPwd := sshEnv["PWD"]
	t.Logf("Local PWD: %q, SSH PWD: %q", localPwd, sshPwd)
}

// TestSSHSessionCommandExecutionDifferentFromLocal verifies commands run in SSH context
func TestSSHSessionCommandExecutionDifferentFromLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	// Create SSH session
	ssh, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer ssh.Close()

	// Run command via SSH
	result, err := ssh.Run(context.Background(), []string{"pwd"}, RunOpts{})
	if err != nil {
		t.Fatalf("SSH Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("SSH command failed with exit code %d", result.ExitCode)
	}

	sshPwd := strings.TrimSpace(string(result.Stdout))
	if sshPwd == "" {
		t.Error("SSH pwd returned empty string")
	}

	// Verify it's an absolute path
	if !strings.HasPrefix(sshPwd, "/") {
		t.Errorf("SSH pwd not absolute: %q", sshPwd)
	}

	t.Logf("SSH working directory: %q", sshPwd)
}

// TestSSHSessionWithEnvModifiesRemoteEnvironment verifies WithEnv works over SSH
func TestSSHSessionWithEnvModifiesRemoteEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	server := getSSHTestServer(t)
	if server == nil {
		t.Skip("SSH test server not available")
	}

	// Create SSH session
	ssh, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": server.Port,
		"user": os.Getenv("USER"),
		"key":  server.ClientKey, "strict_host_key": false,
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer ssh.Close()

	// Create modified session with custom env
	sshWithEnv := ssh.WithEnv(map[string]string{
		"OPAL_SSH_TEST": "ssh_value",
	})

	// Run command that uses the env var
	result, err := sshWithEnv.Run(context.Background(), []string{"sh", "-c", "echo $OPAL_SSH_TEST"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "ssh_value" {
		t.Errorf("Output: got %q, want %q", output, "ssh_value")
	}

	// Verify original session doesn't have the var
	origEnv := ssh.Env()
	if _, ok := origEnv["OPAL_SSH_TEST"]; ok {
		t.Error("Original SSH session has OPAL_SSH_TEST (leaked!)")
	}

	// Verify modified session has the var
	modEnv := sshWithEnv.Env()
	if modEnv["OPAL_SSH_TEST"] != "ssh_value" {
		t.Errorf("Modified SSH session OPAL_SSH_TEST: got %q, want %q", modEnv["OPAL_SSH_TEST"], "ssh_value")
	}
}

// TestNewSSHSession_EmptyHostValidation tests that empty host fails validation
func TestNewSSHSession_EmptyHostValidation(t *testing.T) {
	// Test empty string host
	_, err := NewSSHSession(map[string]any{
		"host": "",
		"port": 22,
	})

	if err == nil {
		t.Fatal("Expected error for empty host, got nil")
	}

	transportErr, ok := err.(TransportError)
	if !ok {
		t.Fatalf("Expected TransportError, got %T: %v", err, err)
	}

	if transportErr.Code != TransportErrorCodeValidationFailed {
		t.Errorf("Error code: got %q, want %q", transportErr.Code, TransportErrorCodeValidationFailed)
	}

	if transportErr.Message != "SSH host cannot be empty" {
		t.Errorf("Error message: got %q, want %q", transportErr.Message, "SSH host cannot be empty")
	}

	// Test whitespace-only host
	_, err = NewSSHSession(map[string]any{
		"host": "   ",
		"port": 22,
	})

	if err == nil {
		t.Fatal("Expected error for whitespace-only host, got nil")
	}
}

// TestNewSSHSession_InvalidPortValidation tests that invalid ports fail validation
func TestNewSSHSession_InvalidPortValidation(t *testing.T) {
	testCases := []struct {
		name string
		port int
	}{
		{"port 0", 0},
		{"port -1", -1},
		{"port 70000", 70000},
		{"port 65536", 65536},
		{"port -100", -100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSSHSession(map[string]any{
				"host": "127.0.0.1",
				"port": tc.port,
			})

			if err == nil {
				t.Fatalf("Expected error for port %d, got nil", tc.port)
			}

			transportErr, ok := err.(TransportError)
			if !ok {
				t.Fatalf("Expected TransportError, got %T: %v", err, err)
			}

			if transportErr.Code != TransportErrorCodeValidationFailed {
				t.Errorf("Error code: got %q, want %q", transportErr.Code, TransportErrorCodeValidationFailed)
			}

			expectedMsg := fmt.Sprintf("SSH port must be between 1 and 65535, got %d", tc.port)
			if transportErr.Message != expectedMsg {
				t.Errorf("Error message: got %q, want %q", transportErr.Message, expectedMsg)
			}
		})
	}
}

// TestNewSSHSession_InvalidKeyFileValidation tests that non-existent key file fails validation
func TestNewSSHSession_InvalidKeyFileValidation(t *testing.T) {
	_, err := NewSSHSession(map[string]any{
		"host": "127.0.0.1",
		"port": 22,
		"key":  "/nonexistent/path/to/key",
	})

	if err == nil {
		t.Fatal("Expected error for non-existent key file, got nil")
	}

	transportErr, ok := err.(TransportError)
	if !ok {
		t.Fatalf("Expected TransportError, got %T: %v", err, err)
	}

	if transportErr.Code != TransportErrorCodeValidationFailed {
		t.Errorf("Error code: got %q, want %q", transportErr.Code, TransportErrorCodeValidationFailed)
	}

	if !strings.Contains(transportErr.Message, "SSH key file not accessible") {
		t.Errorf("Error message should contain 'SSH key file not accessible', got: %q", transportErr.Message)
	}

	if transportErr.Cause == nil {
		t.Error("Expected Cause to be set for key file validation error")
	}
}

func TestSSHSessionDialContextNotConnected(t *testing.T) {
	session := &SSHSession{}

	conn, err := session.DialContext(context.Background(), "tcp", "127.0.0.1:22")
	if conn != nil {
		t.Fatalf("DialContext connection: got non-nil, want nil")
	}

	if err == nil {
		t.Fatal("DialContext error: got nil, want non-nil")
	}

	if diff := cmp.Diff("ssh session not connected", err.Error()); diff != "" {
		t.Fatalf("DialContext error mismatch (-want +got):\n%s", diff)
	}
}

func TestSSHSessionDialContextDelegatesToClient(t *testing.T) {
	originalDialContext := sshClientDialContext
	t.Cleanup(func() {
		sshClientDialContext = originalDialContext
	})

	expectedErr := errors.New("delegated dial error")
	called := false

	var gotClient *ssh.Client
	var gotCtx context.Context
	var gotNetwork string
	var gotAddr string

	sshClientDialContext = func(client *ssh.Client, ctx context.Context, network, addr string) (net.Conn, error) {
		called = true
		gotClient = client
		gotCtx = ctx
		gotNetwork = network
		gotAddr = addr
		return nil, expectedErr
	}

	client := &ssh.Client{}
	session := &SSHSession{client: client}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := session.DialContext(ctx, "tcp", "example.internal:443")
	if conn != nil {
		t.Fatalf("DialContext connection: got non-nil, want nil")
	}

	if !called {
		t.Fatal("DialContext did not delegate to ssh client")
	}

	if gotClient != client {
		t.Fatal("DialContext delegated with unexpected client")
	}

	if gotCtx != ctx {
		t.Fatal("DialContext delegated with unexpected context")
	}

	if diff := cmp.Diff("tcp", gotNetwork); diff != "" {
		t.Fatalf("DialContext network mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("example.internal:443", gotAddr); diff != "" {
		t.Fatalf("DialContext addr mismatch (-want +got):\n%s", diff)
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("DialContext error: got %v, want %v", err, expectedErr)
	}
}

func TestSSHSessionDialContextRequiresDeadline(t *testing.T) {
	originalDialContext := sshClientDialContext
	t.Cleanup(func() {
		sshClientDialContext = originalDialContext
	})

	called := false
	sshClientDialContext = func(client *ssh.Client, ctx context.Context, network, addr string) (net.Conn, error) {
		called = true
		return nil, nil
	}

	session := &SSHSession{client: &ssh.Client{}}

	conn, err := session.DialContext(context.Background(), "tcp", "example.internal:443")
	if conn != nil {
		t.Fatalf("DialContext connection: got non-nil, want nil")
	}
	if err == nil {
		t.Fatal("DialContext error: got nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "requires context deadline") {
		t.Fatalf("DialContext error: got %q, want actionable deadline guidance", err.Error())
	}
	if called {
		t.Fatal("DialContext delegated without required deadline")
	}
}

func TestSSHSessionDialContextTimeoutIsActionable(t *testing.T) {
	originalDialContext := sshClientDialContext
	t.Cleanup(func() {
		sshClientDialContext = originalDialContext
	})

	sshClientDialContext = func(client *ssh.Client, ctx context.Context, network, addr string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	session := &SSHSession{client: &ssh.Client{}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	conn, err := session.DialContext(ctx, "tcp", "example.internal:443")
	if conn != nil {
		t.Fatalf("DialContext connection: got non-nil, want nil")
	}
	if err == nil {
		t.Fatal("DialContext error: got nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("DialContext error: got %q, want timeout guidance", err.Error())
	}
}

func TestDialSSHClientHandshakeTimeoutUsesConfigurableTimeout(t *testing.T) {
	originalNewClientConn := sshNewClientConn
	t.Cleanup(func() {
		sshNewClientConn = originalNewClientConn
	})

	sshNewClientConn = func(conn net.Conn, addr string, config *ssh.ClientConfig) (ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
		time.Sleep(100 * time.Millisecond)
		return nil, nil, nil, errors.New("should not complete before timeout")
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		clientConn, serverConn := net.Pipe()
		go func() {
			<-ctx.Done()
			_ = serverConn.Close()
		}()
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _, err := dialSSHClient(ctx, dialContext, map[string]any{
		"host":              "example.internal",
		"port":              22,
		"user":              "tester",
		"strict_host_key":   false,
		"handshake_timeout": "10ms",
	})
	if err == nil {
		t.Fatal("dialSSHClient error: got nil, want non-nil")
	}

	var transportErr TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("dialSSHClient error type: got %T, want TransportError", err)
	}
	if transportErr.Cause == nil {
		t.Fatal("dialSSHClient error cause: got nil, want handshake timeout cause")
	}
	if !strings.Contains(transportErr.Cause.Error(), "handshake timed out") {
		t.Fatalf("dialSSHClient error cause: got %q, want handshake timeout guidance", transportErr.Cause.Error())
	}
}

func TestSSHDialPoolHasCapAndReturnsActionableTimeout(t *testing.T) {
	for i := 0; i < maxSSHDialPoolSize; i++ {
		if err := acquireSSHDialSlot(context.Background()); err != nil {
			t.Fatalf("acquireSSHDialSlot setup failed at slot %d: %v", i, err)
		}
	}
	t.Cleanup(func() {
		for i := 0; i < maxSSHDialPoolSize; i++ {
			releaseSSHDialSlot()
		}
	})

	if diff := cmp.Diff(maxSSHDialPoolSize, cap(sshDialPool)); diff != "" {
		t.Fatalf("ssh dial pool cap mismatch (-want +got):\n%s", diff)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := acquireSSHDialSlot(ctx)
	if err == nil {
		t.Fatal("acquireSSHDialSlot error: got nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "pool exhausted") {
		t.Fatalf("acquireSSHDialSlot error: got %q, want actionable pool timeout", err.Error())
	}
}

func TestSSHTransportOpenFailsWithoutParentNetworkDialer(t *testing.T) {
	transport := &SSHTransport{}

	session, err := transport.Open(&sshTransportParentWithoutDialer{}, map[string]any{"host": "example.internal", "port": 2222})
	if session != nil {
		t.Fatalf("Open session: got non-nil, want nil")
	}
	if err == nil {
		t.Fatal("Open error: got nil, want non-nil")
	}

	if diff := cmp.Diff("session does not provide a sealed network dialer", err.Error()); diff != "" {
		t.Fatalf("Open error mismatch (-want +got):\n%s", diff)
	}
}

func TestSSHTransportNodeExecuteFailsWithoutParentNetworkDialer(t *testing.T) {
	transport := &SSHTransport{}
	node := transport.Wrap(nil, map[string]any{
		"host":    "example.internal",
		"port":    2222,
		"command": "echo hi",
	})

	result, err := node.Execute(ExecContext{Session: &sshTransportParentWithoutDialer{}})
	if err == nil {
		t.Fatal("Execute error: got nil, want non-nil")
	}

	if diff := cmp.Diff(ExitFailure, result.ExitCode); diff != "" {
		t.Fatalf("Execute exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("session does not provide a sealed network dialer", err.Error()); diff != "" {
		t.Fatalf("Execute error mismatch (-want +got):\n%s", diff)
	}
}

func TestSSHTransportOpenRejectsUnsealedNetworkDialer(t *testing.T) {
	transport := &SSHTransport{}

	session, err := transport.Open(&sshTransportParentUnsealedDialer{}, map[string]any{"host": "example.internal", "port": 2222})
	if session != nil {
		t.Fatalf("Open session: got non-nil, want nil")
	}
	if err == nil {
		t.Fatal("Open error: got nil, want non-nil")
	}

	if diff := cmp.Diff("session does not provide a sealed network dialer", err.Error()); diff != "" {
		t.Fatalf("Open error mismatch (-want +got):\n%s", diff)
	}
}

func TestSSHTransportNodeExecuteReusesScopedSSHSessionWithoutDial(t *testing.T) {
	transport := &SSHTransport{}
	node := transport.Wrap(nil, map[string]any{"host": "example.internal", "port": 2222})

	result, err := node.Execute(ExecContext{Session: &sshTransportScopedWithoutDialer{}})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if diff := cmp.Diff(ExitSuccess, result.ExitCode); diff != "" {
		t.Fatalf("Execute exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestSSHTransportOpenUsesPortFromParams(t *testing.T) {
	originalNewClientConn := sshNewClientConn
	t.Cleanup(func() {
		sshNewClientConn = originalNewClientConn
	})

	parent := &sshTransportParentWithDialer{}
	transport := &SSHTransport{}

	sshNewClientConn = func(conn net.Conn, addr string, config *ssh.ClientConfig) (ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
		_ = conn.Close()
		return nil, nil, nil, errors.New("forced handshake error")
	}

	_, err := transport.Open(parent, map[string]any{
		"host":            "example.internal",
		"port":            2201,
		"user":            "tester",
		"strict_host_key": false,
	})
	if err == nil {
		t.Fatal("Open error: got nil, want non-nil")
	}

	if diff := cmp.Diff("tcp", parent.gotNetwork); diff != "" {
		t.Fatalf("Open network mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("example.internal:2201", parent.gotAddr); diff != "" {
		t.Fatalf("Open addr mismatch (-want +got):\n%s", diff)
	}
}

type sshTransportParentWithoutDialer struct{}

type sshTransportScopedWithoutDialer struct {
	sshTransportParentWithoutDialer
}

func (s *sshTransportParentWithoutDialer) Run(ctx context.Context, argv []string, opts RunOpts) (Result, error) {
	return Result{}, nil
}

func (s *sshTransportParentWithoutDialer) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (s *sshTransportParentWithoutDialer) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (s *sshTransportParentWithoutDialer) Env() map[string]string {
	return map[string]string{}
}

func (s *sshTransportParentWithoutDialer) WithEnv(delta map[string]string) Session {
	return s
}

func (s *sshTransportParentWithoutDialer) WithWorkdir(dir string) Session {
	return s
}

func (s *sshTransportParentWithoutDialer) Cwd() string {
	return ""
}

func (s *sshTransportParentWithoutDialer) ID() string {
	return "test-parent"
}

func (s *sshTransportParentWithoutDialer) TransportScope() TransportScope {
	return TransportScopeLocal
}

func (s *sshTransportParentWithoutDialer) Platform() string {
	return ""
}

func (s *sshTransportParentWithoutDialer) Close() error {
	return nil
}

func (s *sshTransportScopedWithoutDialer) ID() string {
	return "transport:ssh"
}

func (s *sshTransportScopedWithoutDialer) TransportScope() TransportScope {
	return TransportScopeSSH
}

type sshTransportParentWithDialer struct {
	sshTransportParentWithoutDialer
	gotNetwork string
	gotAddr    string
}

type sshTransportParentUnsealedDialer struct {
	sshTransportParentWithoutDialer
}

func (s *sshTransportParentWithDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	s.gotNetwork = network
	s.gotAddr = addr
	clientConn, serverConn := net.Pipe()
	_ = serverConn.Close()
	return clientConn, nil
}

func (s *sshTransportParentWithDialer) sealNetworkDialer() NetworkDialer {
	return s
}

func (s *sshTransportParentUnsealedDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return nil, errors.New("unsealed dialer should be rejected")
}
