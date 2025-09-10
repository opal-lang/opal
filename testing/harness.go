package testing

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ================================================================================================
// GENERIC TEST UTILITIES - Reusable utilities for any testing scenario
// ================================================================================================

// CommandResult represents the result of executing a command
type CommandResult struct {
	Exit     int           `json:"exit"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}

// Success returns true if the command succeeded (exit code 0)
func (r CommandResult) Success() bool {
	return r.Exit == 0
}

// Failed returns true if the command failed (non-zero exit code)
func (r CommandResult) Failed() bool {
	return r.Exit != 0
}

// ================================================================================================
// UTILITY FUNCTIONS
// ================================================================================================

// toKeyValList converts environment map to KEY=VALUE slice for exec.Cmd
func toKeyValList(env map[string]string) []string {
	var result []string
	for key, value := range env {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}

// runAndCapture runs a command and captures stdout, stderr, and exit code
func runAndCapture(cmd *exec.Cmd) (stdout, stderr string, exitCode int) {
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	} else {
		exitCode = 0
	}

	return stdout, stderr, exitCode
}

// extractEnvFingerprint extracts environment fingerprint from plan output
func extractEnvFingerprint(planOutput string) string {
	// TODO: Parse JSON and extract env_fingerprint field
	// For now, return empty string
	return ""
}

// ================================================================================================
// TEST CONTEXT HELPERS
// ================================================================================================

// TestEnvironment represents environment variables for testing
type TestEnvironment map[string]string

// NewTestEnvironment creates a test environment with common variables
func NewTestEnvironment() TestEnvironment {
	return TestEnvironment{
		"USER": "testuser",
		"HOME": "/tmp/testhome",
		"PATH": "/usr/bin:/bin",
	}
}

// WithVars adds variables to the test environment
func (e TestEnvironment) WithVars(vars map[string]string) TestEnvironment {
	result := make(TestEnvironment)
	for k, v := range e {
		result[k] = v
	}
	for k, v := range vars {
		result[k] = v
	}
	return result
}

// WriteTestCLI writes a test CLI file and returns its path
func WriteTestCLI(t TestingT, content string) string {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-*.cli")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Write content
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write test CLI: %v", err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}

// CleanupTestCLI removes a test CLI file
func CleanupTestCLI(path string) {
	os.Remove(path)
}

// TestingT is a minimal interface for testing frameworks
type TestingT interface {
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Helper()
}

// ================================================================================================
// FILE UTILITIES
// ================================================================================================

// CreateTempDir creates a temporary directory for tests
func CreateTempDir(t TestingT, pattern string) string {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// CleanupTempDir removes a temporary directory
func CleanupTempDir(dir string) {
	os.RemoveAll(dir)
}

// WriteFile writes content to a file
func WriteFile(t TestingT, path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// ReadFile reads content from a file
func ReadFile(t TestingT, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CopyFile copies a file from src to dst
func CopyFile(t TestingT, src, dst string) {
	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("Failed to open source file %s: %v", src, err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Failed to create destination file %s: %v", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}
}
