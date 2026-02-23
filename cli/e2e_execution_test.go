package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildE2EBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	opalBin := filepath.Join(tmpDir, "opal")

	cmd := exec.Command("go", "build", "-o", opalBin, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build opal: %v\nOutput: %s", err, output)
	}

	return opalBin
}

func createE2ETestFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "test.opl")
	err := os.WriteFile(tmpFile, []byte(strings.TrimSpace(content)), 0o644)
	require.NoError(t, err)

	return tmpFile
}

func runE2E(t *testing.T, opalBin string, args ...string) string {
	t.Helper()

	cmd := exec.Command(opalBin, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("opal failed: %v\nStderr: %s\nStdout: %s", err, exitErr.Stderr, output)
		}
		t.Fatalf("opal failed: %v", err)
	}

	return string(output)
}

func runE2EWithStderr(t *testing.T, opalBin string, args ...string) (stdout, stderr string) {
	t.Helper()

	cmd := exec.Command(opalBin, args...)
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			_ = exitErr
		} else {
			t.Fatalf("opal failed to run: %v", err)
		}
	}

	return stdoutBuf.String(), stderrBuf.String()
}

func runE2EExpectError(t *testing.T, opalBin string, args ...string) (stderr string) {
	t.Helper()

	cmd := exec.Command(opalBin, args...)
	output, err := cmd.Output()
	if err == nil {
		t.Fatalf("expected opal to fail, but it succeeded with output: %s", output)
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(exitErr.Stderr)
	}

	t.Fatalf("opal failed unexpectedly: %v", err)
	return ""
}

func loadE2EFixture(t *testing.T, name string) string {
	t.Helper()

	fixturePath := filepath.Join("..", "testdata", "e2e", name)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}

	return string(content)
}

func TestE2EPlaceholder(t *testing.T) {
	opalBin := buildE2EBinary(t)
	assert.FileExists(t, opalBin)
}
