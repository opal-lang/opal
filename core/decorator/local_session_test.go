package decorator

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestLocalSessionRun verifies command execution
func TestLocalSessionRun(t *testing.T) {
	session := NewLocalSession()

	// Run simple echo command
	result, err := session.Run(context.Background(), []string{"echo", "hello world"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "hello world" {
		t.Errorf("Stdout: got %q, want %q", output, "hello world")
	}
}

// TestLocalSessionRunWithStdin verifies stdin handling
func TestLocalSessionRunWithStdin(t *testing.T) {
	session := NewLocalSession()

	// Run cat command with stdin
	result, err := session.Run(context.Background(), []string{"cat"}, RunOpts{
		Stdin: bytes.NewReader([]byte("test input")), // Wrap in bytes.NewReader
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}

	output := string(result.Stdout)
	if output != "test input" {
		t.Errorf("Stdout: got %q, want %q", output, "test input")
	}
}

// TestLocalSessionRunFailure verifies non-zero exit codes
func TestLocalSessionRunFailure(t *testing.T) {
	session := NewLocalSession()

	// Run command that fails
	result, err := session.Run(context.Background(), []string{"sh", "-c", "exit 42"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("ExitCode: got %d, want 42", result.ExitCode)
	}
}

// TestLocalSessionEnv verifies environment handling
func TestLocalSessionEnv(t *testing.T) {
	session := NewLocalSession()

	// Get environment
	env := session.Env()

	// Should have PATH (from os.Environ)
	if _, ok := env["PATH"]; !ok {
		t.Error("Environment missing PATH")
	}

	// Verify it's a copy (immutable)
	env["TEST_VAR"] = "modified"
	env2 := session.Env()
	if _, ok := env2["TEST_VAR"]; ok {
		t.Error("Environment was mutated (should be immutable)")
	}
}

// TestLocalSessionWithEnv verifies copy-on-write environment
func TestLocalSessionWithEnv(t *testing.T) {
	session := NewLocalSession()

	// Create new session with additional env var
	session2 := session.WithEnv(map[string]string{
		"MY_VAR": "my_value",
	})

	// Original session unchanged
	if _, ok := session.Env()["MY_VAR"]; ok {
		t.Error("Original session was mutated")
	}

	// New session has the variable
	env2 := session2.Env()
	if env2["MY_VAR"] != "my_value" {
		t.Errorf("MY_VAR: got %q, want %q", env2["MY_VAR"], "my_value")
	}

	// New session still has original env vars
	if _, ok := env2["PATH"]; !ok {
		t.Error("New session missing PATH from parent")
	}
}

// TestLocalSessionWithEnvInCommand verifies env vars are passed to commands
func TestLocalSessionWithEnvInCommand(t *testing.T) {
	session := NewLocalSession()

	// Create session with custom env var
	session2 := session.WithEnv(map[string]string{
		"TEST_VAR": "test_value",
	})

	// Run command that reads the env var
	result, err := session2.Run(context.Background(), []string{"sh", "-c", "echo $TEST_VAR"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "test_value" {
		t.Errorf("Output: got %q, want %q", output, "test_value")
	}
}

// TestLocalSessionPut verifies file writing
func TestLocalSessionPut(t *testing.T) {
	session := NewLocalSession()

	// Create temp directory
	tmpDir := t.TempDir()

	// Write file
	path := filepath.Join(tmpDir, "test.txt")
	err := session.Put(context.Background(), []byte("hello world"), path, 0o644)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file exists and has correct content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(content) != "hello world" {
		t.Errorf("Content: got %q, want %q", string(content), "hello world")
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Mode().Perm() != 0o644 {
		t.Errorf("Permissions: got %o, want %o", info.Mode().Perm(), 0o644)
	}
}

// TestLocalSessionPutCreatesDirectories verifies parent directories are created
func TestLocalSessionPutCreatesDirectories(t *testing.T) {
	session := NewLocalSession()

	// Create temp directory
	tmpDir := t.TempDir()

	// Write file in nested directory
	path := filepath.Join(tmpDir, "a", "b", "c", "test.txt")
	err := session.Put(context.Background(), []byte("nested"), path, 0o644)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(content) != "nested" {
		t.Errorf("Content: got %q, want %q", string(content), "nested")
	}
}

// TestLocalSessionGet verifies file reading
func TestLocalSessionGet(t *testing.T) {
	session := NewLocalSession()

	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("file content"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file via session
	content, err := session.Get(context.Background(), path)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(content) != "file content" {
		t.Errorf("Content: got %q, want %q", string(content), "file content")
	}
}

// TestLocalSessionCwd verifies working directory
func TestLocalSessionCwd(t *testing.T) {
	session := NewLocalSession()

	cwd := session.Cwd()
	if cwd == "" {
		t.Error("Cwd returned empty string")
	}

	// Should be an absolute path
	if !filepath.IsAbs(cwd) {
		t.Errorf("Cwd not absolute: %q", cwd)
	}
}

// TestLocalSessionClose verifies cleanup
func TestLocalSessionClose(t *testing.T) {
	session := NewLocalSession()

	// Close should succeed (no-op for local session)
	err := session.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Should be safe to call multiple times
	err = session.Close()
	if err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

// TestLocalSessionRelativePaths verifies relative path handling
func TestLocalSessionRelativePaths(t *testing.T) {
	session := NewLocalSession()
	tmpDir := t.TempDir()

	// Create session with custom cwd (simulated)
	// Note: LocalSession doesn't have SetCwd yet, so we test with absolute paths
	// This test demonstrates the intended behavior

	// Write file with relative path
	err := session.Put(context.Background(), []byte("test"), filepath.Join(tmpDir, "relative.txt"), 0o644)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Read file with relative path
	content, err := session.Get(context.Background(), filepath.Join(tmpDir, "relative.txt"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(content) != "test" {
		t.Errorf("Content: got %q, want %q", string(content), "test")
	}
}

// Example: Using LocalSession to run a command
func ExampleLocalSession_Run() {
	session := NewLocalSession()

	// Run echo command
	result, err := session.Run(context.Background(), []string{"echo", "Hello, World!"}, RunOpts{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Exit code:", result.ExitCode)
	fmt.Println("Output:", strings.TrimSpace(string(result.Stdout)))
	// Output:
	// Exit code: 0
	// Output: Hello, World!
}

// Example: Using LocalSession with environment variables
func ExampleLocalSession_WithEnv() {
	session := NewLocalSession()

	// Create new session with custom env var
	session2 := session.WithEnv(map[string]string{
		"MY_VAR": "my_value",
	})

	// Run command that uses the env var
	result, _ := session2.Run(context.Background(), []string{"sh", "-c", "echo $MY_VAR"}, RunOpts{})
	fmt.Println(strings.TrimSpace(string(result.Stdout)))
	// Output:
	// my_value
}

// Example: Using LocalSession to write and read files
func ExampleLocalSession_Put() {
	session := NewLocalSession()

	// Write file
	_ = session.Put(context.Background(), []byte("Hello, File!"), "/tmp/test.txt", 0o644)

	// Read file back
	content, _ := session.Get(context.Background(), "/tmp/test.txt")
	_ = content
}

// TestLocalSessionStreamsStdin verifies that stdin streams data (not buffered)
func TestLocalSessionStreamsStdin(t *testing.T) {
	session := NewLocalSession()

	// Create pipe for streaming
	pr, pw := io.Pipe()

	var stdout bytes.Buffer
	opts := RunOpts{
		Stdin:  pr,
		Stdout: &stdout,
	}

	// Write data in goroutine (simulates streaming)
	go func() {
		pw.Write([]byte("line1\n"))
		time.Sleep(10 * time.Millisecond)
		pw.Write([]byte("line2\n"))
		pw.Close()
	}()

	// Run cat (reads stdin, writes to stdout)
	result, err := session.Run(context.Background(), []string{"cat"}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}
	if stdout.String() != "line1\nline2\n" {
		t.Errorf("Stdout: got %q, want %q", stdout.String(), "line1\nline2\n")
	}
}

// TestLocalSessionForwardsStderr verifies stderr forwarding
func TestLocalSessionForwardsStderr(t *testing.T) {
	session := NewLocalSession()

	var stdout, stderr bytes.Buffer
	opts := RunOpts{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	// Run command that writes to stderr
	result, err := session.Run(context.Background(),
		[]string{"sh", "-c", "echo out; echo err >&2"}, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}
	if stdout.String() != "out\n" {
		t.Errorf("Stdout: got %q, want %q", stdout.String(), "out\n")
	}
	if stderr.String() != "err\n" {
		t.Errorf("Stderr: got %q, want %q", stderr.String(), "err\n")
	}
}

// TestLocalSessionKillsProcessGroupOnCancel verifies process group cancellation
func TestLocalSessionKillsProcessGroupOnCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Process groups not supported on Windows")
	}

	session := NewLocalSession()

	ctx, cancel := context.WithCancel(context.Background())

	var stdout bytes.Buffer
	opts := RunOpts{
		Stdout: &stdout,
	}

	// Start long-running command
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := session.Run(ctx, []string{"sleep", "10"}, opts)
	duration := time.Since(start)

	// Should error due to context cancellation
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// Should return ExitCanceled (-1)
	if result.ExitCode != ExitCanceled {
		t.Errorf("ExitCode: got %d, want %d (ExitCanceled)", result.ExitCode, ExitCanceled)
	}

	// Should complete quickly (not wait for full 10 seconds)
	if duration > 1*time.Second {
		t.Errorf("Duration: got %v, want < 1s (process should be killed quickly)", duration)
	}

	// Note: Verifying no zombie processes is platform-specific and hard to test reliably
	// The Setpgid=true ensures the entire process group is killed
}
