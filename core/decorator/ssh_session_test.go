package decorator

import (
	"context"
	"os"
	"strings"
	"testing"
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
