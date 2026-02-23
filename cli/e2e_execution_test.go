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

func TestE2EOperator_AndBothRun(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun test = echo "A" && echo "B"`)
	output := runE2E(t, opalBin, "-f", testFile, "test")
	assert.Equal(t, "A\nB\n", output)
}

func TestE2EOperator_AndShortCircuit(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun test = exit 1 && echo "B"`)
	output, _ := runE2EWithStderr(t, opalBin, "-f", testFile, "test")
	assert.Equal(t, "", output)
}

func TestE2EOperator_OrShortCircuit(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun test = echo "A" || echo "B"`)
	output := runE2E(t, opalBin, "-f", testFile, "test")
	assert.Equal(t, "A\n", output)
}

func TestE2EOperator_OrFallback(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun test = exit 1 || echo "B"`)
	output := runE2E(t, opalBin, "-f", testFile, "test")
	assert.Equal(t, "B\n", output)
}

func TestE2EOperator_Sequence(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun test = echo "A" ; echo "B"`)
	output := runE2E(t, opalBin, "-f", testFile, "test")
	assert.Equal(t, "A\nB\n", output)
}

func TestE2EOperator_Complex(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun test = echo "A" && echo "B" || echo "C"`)
	output := runE2E(t, opalBin, "-f", testFile, "test")
	assert.Equal(t, "A\nB\n", output)
}

func TestE2ECore_SimpleEcho(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun hello = echo "Hello"`)
	output := runE2E(t, opalBin, "-f", testFile, "hello")
	assert.Equal(t, "Hello\n", output)
}

func TestE2ECore_MultiLine(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun multiline = echo "Line1"; echo "Line2"; echo "Line3"`)
	output := runE2E(t, opalBin, "-f", testFile, "multiline")
	assert.Equal(t, "Line1\nLine2\nLine3\n", output)
}

func TestE2ECore_CommandSucceeds(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun success = echo "Success"`)
	output := runE2E(t, opalBin, "-f", testFile, "success")
	assert.Equal(t, "Success\n", output)
}

func TestE2ECore_CommandFails(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun fail = exit 1`)
	stderr := runE2EExpectError(t, opalBin, "-f", testFile, "fail")
	assert.Contains(t, stderr, "command failed with exit code 1")
}

func TestE2ECore_Stderr(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun err = sh -c "echo error >&2"`)
	output := runE2E(t, opalBin, "-f", testFile, "err")
	assert.Equal(t, "error\n", output)
}

func TestE2EScript_TopLevelOnly(t *testing.T) {
	opalBin := buildE2EBinary(t)
	scriptFile := createE2ETestFile(t, `
echo "Line 1"
echo "Line 2"
`)
	output := runE2E(t, opalBin, "-f", scriptFile)
	assert.Equal(t, "Line 1\nLine 2\n", output)
}

func TestE2EScript_ExecutesInOrder(t *testing.T) {
	opalBin := buildE2EBinary(t)
	scriptFile := createE2ETestFile(t, `
echo "First"
echo "Second"
echo "Third"
`)
	output := runE2E(t, opalBin, "-f", scriptFile)
	assert.Equal(t, "First\nSecond\nThird\n", output)
}

func TestE2EScript_MixedFunctionsAndCommands(t *testing.T) {
	opalBin := buildE2EBinary(t)
	scriptFile := createE2ETestFile(t, `
fun deploy = echo "deploying"
fun test = echo "testing"

echo "Top level 1"
echo "Top level 2"
`)
	output := runE2E(t, opalBin, "-f", scriptFile)
	assert.Equal(t, "Top level 1\nTop level 2\n", output)
	assert.NotContains(t, output, "deploying", "Functions should not execute in script mode")
	assert.NotContains(t, output, "testing", "Functions should not execute in script mode")
}

func TestE2EScript_OutputConsistency(t *testing.T) {
	opalBin := buildE2EBinary(t)

	scriptFile := createE2ETestFile(t, `
echo "Line 1"
echo "Line 2"
`)

	functionFile := createE2ETestFile(t, `fun multiline = echo "Line 1"; echo "Line 2"`)

	scriptOutput := runE2E(t, opalBin, "-f", scriptFile)
	functionOutput := runE2E(t, opalBin, "-f", functionFile, "multiline")

	assert.Equal(t, scriptOutput, functionOutput)
}

func TestE2EConsistency_DirectVsContract(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun hello = echo "Hello"`)

	directOutput := runE2E(t, opalBin, "-f", testFile, "hello")

	planFile := filepath.Join(t.TempDir(), "test.plan")
	cmd := exec.Command(opalBin, "-f", testFile, "hello", "--dry-run", "--resolve")
	planData, err := cmd.Output()
	require.NoError(t, err)
	os.WriteFile(planFile, planData, 0o644)

	contractOutput := runE2E(t, opalBin, "--plan", planFile, "-f", testFile)

	assert.Equal(t, directOutput, contractOutput)
}

func TestE2EConsistency_DryRunShowsPlan(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun hello = echo "Hello"`)

	stdout, stderr := runE2EWithStderr(t, opalBin, "-f", testFile, "hello", "--dry-run")

	assert.Empty(t, stderr)
	assert.Contains(t, stdout, "hello:")
	assert.Contains(t, stdout, "└─")
	assert.NotContains(t, stdout, "Hello\n")
}

func TestE2EConsistency_ResolvedPlan(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `fun hello = echo "Hello"`)

	planFile := filepath.Join(t.TempDir(), "test.plan")
	cmd := exec.Command(opalBin, "-f", testFile, "hello", "--dry-run", "--resolve")
	planData, err := cmd.Output()
	require.NoError(t, err)

	assert.Greater(t, len(planData), 4, "Plan should have magic bytes")
	assert.Equal(t, "OPAL", string(planData[0:4]), "Plan should start with OPAL magic")

	os.WriteFile(planFile, planData, 0o644)

	output := runE2E(t, opalBin, "--plan", planFile, "-f", testFile)
	assert.Equal(t, "Hello\n", output)
}
