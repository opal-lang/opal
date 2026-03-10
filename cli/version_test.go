package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func buildVersionBinary(t *testing.T, version string) string {
	t.Helper()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "sigil")

	cmd := exec.Command("go", "build", "-ldflags", fmt.Sprintf("-X main.Version=%s", version), "-o", binPath, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build sigil: %v\nOutput: %s", err, output)
	}

	return binPath
}

func runVersionCommand(t *testing.T, binPath string, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err == nil {
		return stdoutBuf.String(), stderrBuf.String(), 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("failed to run sigil: %v", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode()
}

func TestVersionCommand(t *testing.T) {
	binPath := buildVersionBinary(t, "0.2.0")

	stdout, stderr, exitCode := runVersionCommand(t, binPath, "version")

	if diff := cmp.Diff(0, exitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("sigil 0.2.0\n", stdout); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", stderr); diff != "" {
		t.Fatalf("stderr mismatch (-want +got):\n%s", diff)
	}
}

func TestVersionCommandJSON(t *testing.T) {
	binPath := buildVersionBinary(t, "0.2.0")

	stdout, stderr, exitCode := runVersionCommand(t, binPath, "version", "--json")

	if diff := cmp.Diff(0, exitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("{\"version\":\"0.2.0\"}\n", stdout); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", stderr); diff != "" {
		t.Fatalf("stderr mismatch (-want +got):\n%s", diff)
	}
}
