package executor

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockTransport_BasicExec tests basic command execution
func TestMockTransport_BasicExec(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("echo hello", ExitSuccess, "hello\n", "")

	var stdout bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
	}

	exitCode, err := mock.Exec(context.Background(), []string{"echo", "hello"}, opts)

	require.NoError(t, err)
	assert.Equal(t, ExitSuccess, exitCode)
	assert.Equal(t, "hello\n", stdout.String())

	// Verify call was recorded
	calls := mock.ExecCalls()
	assert.Equal(t, 1, len(calls))
	assert.Equal(t, []string{"echo", "hello"}, calls[0].Argv)
}

// TestMockTransport_EnvironmentIsolation tests that MockTransport uses in-memory env
func TestMockTransport_EnvironmentIsolation(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	// Set custom base environment (simulates remote environment)
	mock.SetBaseEnv(map[string]string{
		"PATH":   "/remote/bin",
		"HOME":   "/home/remoteuser",
		"CUSTOM": "remote_value",
	})

	mock.SetExecResponse("env", ExitSuccess, "", "")

	opts := ExecOpts{
		Env: map[string]string{
			"DECORATOR_VAR": "decorator_value",
		},
	}

	_, err := mock.Exec(context.Background(), []string{"env"}, opts)
	require.NoError(t, err)

	// Verify merged environment
	calls := mock.ExecCalls()
	require.Equal(t, 1, len(calls))

	env := calls[0].Env
	assert.Equal(t, "/remote/bin", env["PATH"])
	assert.Equal(t, "/home/remoteuser", env["HOME"])
	assert.Equal(t, "remote_value", env["CUSTOM"])
	assert.Equal(t, "decorator_value", env["DECORATOR_VAR"])

	// Verify it's NOT using os.Environ()
	// (os.Environ() would have different PATH)
	assert.NotContains(t, env["PATH"], "/nix/store")
}

// TestMockTransport_EnvironmentOverride tests decorator vars override base env
func TestMockTransport_EnvironmentOverride(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetBaseEnv(map[string]string{
		"VAR": "base_value",
	})

	mock.SetExecResponse("test", ExitSuccess, "", "")

	opts := ExecOpts{
		Env: map[string]string{
			"VAR": "overridden_value",
		},
	}

	_, err := mock.Exec(context.Background(), []string{"test"}, opts)
	require.NoError(t, err)

	calls := mock.ExecCalls()
	assert.Equal(t, "overridden_value", calls[0].Env["VAR"])
}

// TestMockTransport_MultipleCommands tests multiple command responses
func TestMockTransport_MultipleCommands(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("echo hello", ExitSuccess, "hello\n", "")
	mock.SetExecResponse("echo world", ExitSuccess, "world\n", "")

	var stdout1, stdout2 bytes.Buffer

	// First command
	exitCode1, err1 := mock.Exec(context.Background(), []string{"echo", "hello"}, ExecOpts{Stdout: &stdout1})
	require.NoError(t, err1)
	assert.Equal(t, ExitSuccess, exitCode1)
	assert.Equal(t, "hello\n", stdout1.String())

	// Second command
	exitCode2, err2 := mock.Exec(context.Background(), []string{"echo", "world"}, ExecOpts{Stdout: &stdout2})
	require.NoError(t, err2)
	assert.Equal(t, ExitSuccess, exitCode2)
	assert.Equal(t, "world\n", stdout2.String())

	// Verify both calls recorded
	calls := mock.ExecCalls()
	assert.Equal(t, 2, len(calls))
}

// TestMockTransport_DefaultResponse tests default response for unconfigured commands
func TestMockTransport_DefaultResponse(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetDefaultResponse(ExitSuccess, "default output\n", "")

	var stdout bytes.Buffer
	exitCode, err := mock.Exec(context.Background(), []string{"unknown", "command"}, ExecOpts{Stdout: &stdout})

	require.NoError(t, err)
	assert.Equal(t, ExitSuccess, exitCode)
	assert.Equal(t, "default output\n", stdout.String())
}

// TestMockTransport_ErrorResponse tests error responses
func TestMockTransport_ErrorResponse(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("fail", ExitCommandFailed, "", "error message\n")

	var stderr bytes.Buffer
	exitCode, err := mock.Exec(context.Background(), []string{"fail"}, ExecOpts{Stderr: &stderr})

	require.NoError(t, err)
	assert.Equal(t, ExitCommandFailed, exitCode)
	assert.Equal(t, "error message\n", stderr.String())
}

// TestMockTransport_StdinCapture tests stdin capture
func TestMockTransport_StdinCapture(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("cat", ExitSuccess, "echoed input\n", "")

	stdin := strings.NewReader("test input\n")
	_, err := mock.Exec(context.Background(), []string{"cat"}, ExecOpts{Stdin: stdin})
	require.NoError(t, err)

	// Verify stdin was captured
	assert.Equal(t, "test input\n", mock.CaptureStdin())
}

// TestMockTransport_Put tests file upload
func TestMockTransport_Put(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	content := []byte("file content\n")
	src := bytes.NewReader(content)

	err := mock.Put(context.Background(), src, "/remote/path/file.txt", 0o644)
	require.NoError(t, err)

	// Verify file was stored
	storedContent, mode, exists := mock.GetFile("/remote/path/file.txt")
	assert.True(t, exists)
	assert.Equal(t, content, storedContent)
	assert.Equal(t, fs.FileMode(0o644), mode)

	// Verify call was recorded
	calls := mock.PutCalls()
	assert.Equal(t, 1, len(calls))
	assert.Equal(t, "/remote/path/file.txt", calls[0].Dst)
	assert.Equal(t, content, calls[0].Content)
}

// TestMockTransport_Get tests file download
func TestMockTransport_Get(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	// Put a file first
	content := []byte("remote file content\n")
	mock.Put(context.Background(), bytes.NewReader(content), "/remote/file.txt", 0o644)

	// Get the file
	var dst bytes.Buffer
	err := mock.Get(context.Background(), "/remote/file.txt", &dst)
	require.NoError(t, err)
	assert.Equal(t, content, dst.Bytes())

	// Verify call was recorded
	calls := mock.GetCalls()
	assert.Equal(t, 1, len(calls))
	assert.Equal(t, "/remote/file.txt", calls[0].Src)
	assert.Equal(t, content, calls[0].Content)
}

// TestMockTransport_GetNonexistent tests getting a file that doesn't exist
func TestMockTransport_GetNonexistent(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	var dst bytes.Buffer
	err := mock.Get(context.Background(), "/nonexistent.txt", &dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

// TestMockTransport_Close tests Close tracking
func TestMockTransport_Close(t *testing.T) {
	mock := NewMockTransport()

	assert.False(t, mock.WasClosed())

	err := mock.Close()
	assert.NoError(t, err)
	assert.True(t, mock.WasClosed())

	// Multiple closes should be safe
	err = mock.Close()
	assert.NoError(t, err)
}

// TestMockTransport_Reset tests resetting state
func TestMockTransport_Reset(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	// Execute some commands
	mock.SetExecResponse("test", ExitSuccess, "output\n", "")
	mock.Exec(context.Background(), []string{"test"}, ExecOpts{})
	mock.Put(context.Background(), bytes.NewReader([]byte("data")), "/file.txt", 0o644)

	assert.Equal(t, 1, len(mock.ExecCalls()))
	assert.Equal(t, 1, len(mock.PutCalls()))

	// Reset
	mock.Reset()

	assert.Equal(t, 0, len(mock.ExecCalls()))
	assert.Equal(t, 0, len(mock.PutCalls()))
	assert.False(t, mock.WasClosed())
}

// TestMockTransport_AssertHelpers tests assertion helpers
func TestMockTransport_AssertHelpers(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("echo test", ExitSuccess, "test\n", "")

	opts := ExecOpts{
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	mock.Exec(context.Background(), []string{"echo", "test"}, opts)

	// Test AssertExecCalled
	mock.AssertExecCalled(t, []string{"echo", "test"})

	// Test AssertEnvSet
	mock.AssertEnvSet(t, "TEST_VAR", "test_value")
}

// TestMockTransport_ConcurrentAccess tests thread safety
func TestMockTransport_ConcurrentAccess(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("test", ExitSuccess, "output\n", "")

	// Execute commands concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			mock.Exec(context.Background(), []string{"test"}, ExecOpts{})
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 calls
	calls := mock.ExecCalls()
	assert.Equal(t, 10, len(calls))
}

// TestMockTransport_WorkingDirectory tests Dir option
func TestMockTransport_WorkingDirectory(t *testing.T) {
	mock := NewMockTransport()
	defer mock.Close()

	mock.SetExecResponse("pwd", ExitSuccess, "/custom/dir\n", "")

	opts := ExecOpts{
		Dir: "/custom/dir",
	}

	mock.Exec(context.Background(), []string{"pwd"}, opts)

	calls := mock.ExecCalls()
	assert.Equal(t, "/custom/dir", calls[0].Dir)
}

// TestMockTransport_DemonstratesRemotePattern demonstrates how MockTransport
// simulates remote transports (SSH, Docker, SSM)
func TestMockTransport_DemonstratesRemotePattern(t *testing.T) {
	// This test demonstrates the key difference between LocalTransport and remote transports

	// LocalTransport uses os.Environ() as base
	// MockTransport (like SSH/Docker/SSM) uses custom base environment

	mock := NewMockTransport()
	defer mock.Close()

	// Simulate SSH connection to remote server
	mock.SetBaseEnv(map[string]string{
		"PATH":         "/usr/bin:/bin",        // Remote server's PATH
		"HOME":         "/home/deploy",         // Remote user's home
		"USER":         "deploy",               // Remote username
		"HOSTNAME":     "production-server-01", // Remote hostname
		"AWS_REGION":   "us-east-1",            // Remote server's AWS config
		"DATABASE_URL": "postgres://remote/db", // Remote database
	})

	mock.SetExecResponse("env", ExitSuccess, "", "")

	// Decorator adds variables
	opts := ExecOpts{
		Env: map[string]string{
			"DEPLOYMENT_ID": "deploy-123",
		},
	}

	mock.Exec(context.Background(), []string{"env"}, opts)

	calls := mock.ExecCalls()
	env := calls[0].Env

	// Verify remote environment is used (NOT local os.Environ())
	assert.Equal(t, "/home/deploy", env["HOME"])
	assert.Equal(t, "deploy", env["USER"])
	assert.Equal(t, "production-server-01", env["HOSTNAME"])

	// Verify decorator variable was added
	assert.Equal(t, "deploy-123", env["DEPLOYMENT_ID"])

	// Verify local environment is NOT present
	// (If this were LocalTransport, we'd see /nix/store in PATH)
	assert.NotContains(t, env["PATH"], "/nix/store")
}
