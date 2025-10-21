package executor

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLocalTransportExec_SimpleCommand tests basic command execution
func TestLocalTransportExec_SimpleCommand(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout, stderr bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	exitCode, err := transport.Exec(context.Background(), []string{"echo", "hello"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "hello\n", stdout.String())
	assert.Empty(t, stderr.String())
}

// TestLocalTransportExec_WithStdin tests command with stdin
func TestLocalTransportExec_WithStdin(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout bytes.Buffer
	stdin := strings.NewReader("test input\n")
	opts := ExecOpts{
		Stdin:  stdin,
		Stdout: &stdout,
	}

	exitCode, err := transport.Exec(context.Background(), []string{"cat"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "test input\n", stdout.String())
}

// TestLocalTransportExec_WithEnvironment tests command with custom environment
func TestLocalTransportExec_WithEnvironment(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	exitCode, err := transport.Exec(context.Background(), []string{"sh", "-c", "echo $TEST_VAR"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "test_value\n", stdout.String())
}

// TestLocalTransportExec_WithWorkdir tests command with custom working directory
func TestLocalTransportExec_WithWorkdir(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	// Create temp directory
	tmpDir := t.TempDir()

	var stdout bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Dir:    tmpDir,
	}

	exitCode, err := transport.Exec(context.Background(), []string{"pwd"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	// pwd output should match tmpDir (with newline)
	assert.Equal(t, tmpDir+"\n", stdout.String())
}

// TestLocalTransportExec_NonZeroExit tests command that fails
func TestLocalTransportExec_NonZeroExit(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout, stderr bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	exitCode, err := transport.Exec(context.Background(), []string{"sh", "-c", "exit 42"}, opts)

	require.NoError(t, err) // No error, just non-zero exit
	assert.Equal(t, 42, exitCode)
}

// TestLocalTransportExec_CommandNotFound tests command that doesn't exist
func TestLocalTransportExec_CommandNotFound(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout, stderr bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	exitCode, err := transport.Exec(context.Background(), []string{"nonexistent-command-xyz"}, opts)

	assert.Error(t, err)
	assert.Equal(t, 127, exitCode) // Convention: 127 for command not found
}

// TestLocalTransportExec_ContextCancellation tests context cancellation
func TestLocalTransportExec_ContextCancellation(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var stdout bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
	}

	exitCode, err := transport.Exec(ctx, []string{"sleep", "10"}, opts)

	require.NoError(t, err)        // Context cancellation returns exit code, not error
	assert.Equal(t, 124, exitCode) // Convention: 124 for timeout/cancellation
}

// TestLocalTransportPut tests file upload (local copy)
func TestLocalTransportPut(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	// Create temp directory for destination
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "test.txt")

	// Source content
	content := "test file content\n"
	src := strings.NewReader(content)

	// Put file
	err := transport.Put(context.Background(), src, dstPath, 0o644)
	require.NoError(t, err)

	// Verify file exists and has correct content
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Verify file permissions
	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o644), info.Mode().Perm())
}

// TestLocalTransportPut_CreateParentDirs tests Put creates parent directories
func TestLocalTransportPut_CreateParentDirs(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	// Create temp directory
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "subdir", "nested", "test.txt")

	// Source content
	content := "nested file\n"
	src := strings.NewReader(content)

	// Put file (should create parent dirs)
	err := transport.Put(context.Background(), src, dstPath, 0o644)
	require.NoError(t, err)

	// Verify file exists
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

// TestLocalTransportGet tests file download (local read)
func TestLocalTransportGet(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	// Create temp file
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := "source file content\n"
	err := os.WriteFile(srcPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Get file
	var dst bytes.Buffer
	err = transport.Get(context.Background(), srcPath, &dst)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, content, dst.String())
}

// TestLocalTransportGet_FileNotFound tests Get with missing file
func TestLocalTransportGet_FileNotFound(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var dst bytes.Buffer
	err := transport.Get(context.Background(), "/nonexistent/file.txt", &dst)

	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

// TestLocalTransportClose tests Close is safe to call
func TestLocalTransportClose(t *testing.T) {
	transport := &LocalTransport{}

	// Close should not error
	err := transport.Close()
	assert.NoError(t, err)

	// Multiple closes should be safe
	err = transport.Close()
	assert.NoError(t, err)
}

// TestLocalTransportExec_StderrSeparate tests stdout and stderr are separate
func TestLocalTransportExec_StderrSeparate(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout, stderr bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	// Command that writes to both stdout and stderr
	exitCode, err := transport.Exec(context.Background(), []string{"sh", "-c", "echo stdout; echo stderr >&2"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "stdout\n", stdout.String())
	assert.Equal(t, "stderr\n", stderr.String())
}

// TestLocalTransportExec_NilWriters tests nil stdout/stderr are handled
func TestLocalTransportExec_NilWriters(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	opts := ExecOpts{
		Stdout: nil, // Should use os.Stdout
		Stderr: nil, // Should use os.Stderr
	}

	// Should not panic with nil writers
	exitCode, err := transport.Exec(context.Background(), []string{"echo", "test"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

// TestLocalTransportExec_EmptyEnv tests empty environment map
func TestLocalTransportExec_EmptyEnv(t *testing.T) {
	transport := &LocalTransport{}
	defer transport.Close()

	var stdout bytes.Buffer
	opts := ExecOpts{
		Stdout: &stdout,
		Env:    map[string]string{}, // Empty env
	}

	// Should inherit parent environment
	exitCode, err := transport.Exec(context.Background(), []string{"sh", "-c", "echo $PATH"}, opts)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	// PATH should be inherited from parent
	assert.NotEmpty(t, stdout.String())
}
