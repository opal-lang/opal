package main

import (
	"fmt"
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

func TestE2ERedirect_Overwrite(t *testing.T) {
	opalBin := buildE2EBinary(t)
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.txt")
	testFile := createE2ETestFile(t, fmt.Sprintf(`
fun write = echo "Hello" > %s
`, outFile))
	runE2E(t, opalBin, "-f", testFile, "write")
	data, _ := os.ReadFile(outFile)
	assert.Equal(t, "Hello\n", string(data))
}

func TestE2ERedirect_Append(t *testing.T) {
	opalBin := buildE2EBinary(t)
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.txt")
	testFile := createE2ETestFile(t, fmt.Sprintf(`
fun write = echo "Hello" > %s; echo "World" >> %s
`, outFile, outFile))
	runE2E(t, opalBin, "-f", testFile, "write")
	data, _ := os.ReadFile(outFile)
	assert.Equal(t, "Hello\nWorld\n", string(data))
}

func TestE2ERedirect_Pipe(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
fun pipe = echo "hello world" | grep hello
`)
	output := runE2E(t, opalBin, "-f", testFile, "pipe")
	assert.Contains(t, output, "hello world")
}

func TestE2EMeta_ConditionalPrunesUntakenBranch(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
var env = "prod"
fun deploy {
	if @var.env == "prod" { echo "PROD-BRANCH" } else { echo "DEV-BRANCH" }
}
`)
	output := runE2E(t, opalBin, "-f", testFile, "deploy")
	assert.Contains(t, output, "PROD-BRANCH")
	assert.NotContains(t, output, "DEV-BRANCH")
}

func TestE2EMeta_ConditionalWithVariable(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
var target = "staging"
fun deploy {
	if @var.target == "production" { echo "PRODUCTION-BRANCH" } else { echo "STAGING-BRANCH" }
}
`)
	output := runE2E(t, opalBin, "-f", testFile, "deploy")
	assert.Contains(t, output, "STAGING-BRANCH")
	assert.NotContains(t, output, "PRODUCTION-BRANCH")
}

func TestE2EVariable_Expansion(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
var name = "World"
fun greet = echo "Hello @var.name!"
`)
	output := runE2E(t, opalBin, "-f", testFile, "greet")
	assert.Contains(t, output, "Hello ")
	assert.Contains(t, output, "!")
	assert.NotContains(t, output, "@var.name", "Decorator should be expanded, not literal")
}

func TestE2EVariable_EnvAccess(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
fun check = echo "Home: @env.HOME"
`)
	output := runE2E(t, opalBin, "-f", testFile, "check")
	assert.Contains(t, output, "Home: ")
	assert.NotContains(t, output, "@env.HOME", "Decorator should be expanded, not literal")
}

func TestE2EVariable_Interpolation(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
var name = "Alice"
fun test = echo "Hello @var.name!"
`)
	output := runE2E(t, opalBin, "-f", testFile, "test")
	assert.NotContains(t, output, "@var.name", "Decorator should be expanded")
	assert.Contains(t, output, "Hello ")
	assert.Contains(t, output, "!")
}

func TestE2EDecorator_Retry(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
	fun flaky = @retry(times=3, delay=100ms) { echo "attempt"; false; }
  echo "attempt"
  exit 1
}
`)
	stderr := runE2EExpectError(t, opalBin, "-f", testFile, "flaky")
	assert.Contains(t, stderr, "exit code 1")
}

func TestE2EDecorator_Timeout(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
	fun slow = @timeout(duration="1s") { sleep 10; }
	  sleep 10
	}
`)
	stderr := runE2EExpectError(t, opalBin, "-f", testFile, "slow")
	assert.Contains(t, stderr, "timeout")
}

func TestE2EDecorator_Parallel(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
fun concurrent = @parallel { echo "Task A"; echo "Task B"; echo "Task C"; }
	  echo "Task A"
	  echo "Task B"
	  echo "Task C"
	}
`)
	output := runE2E(t, opalBin, "-f", testFile, "concurrent")
	assert.Contains(t, output, "Task A")
	assert.Contains(t, output, "Task B")
	assert.Contains(t, output, "Task C")
}

func TestE2EWorkflow_BuildTestDeploy(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
fun build = echo "Building..."
fun test = echo "Testing..."
fun deploy = echo "Deploying..."

fun release = { build; test; deploy }
`)
	output := runE2E(t, opalBin, "-f", testFile, "release")
	assert.Contains(t, output, "Building...")
	assert.Contains(t, output, "Testing...")
	assert.Contains(t, output, "Deploying...")
}

func TestE2EWorkflow_WithVariables(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
var appName = "myapp"
fun deploy = echo "Deploying @var.appName"
`)
	output := runE2E(t, opalBin, "-f", testFile, "deploy")
	t.Logf("Actual output: %q", output)
	assert.Contains(t, output, "Deploying myapp")
}

func TestE2EMeta_ForRange(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
fun loop {
	for i in [1, 2, 3] { echo "item @var.i" }
}
`)
	output := runE2E(t, opalBin, "-f", testFile, "loop")
	assert.Contains(t, output, "item 1")
	assert.Contains(t, output, "item 2")
	assert.Contains(t, output, "item 3")
}

func TestE2EMeta_ForVariable(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
var items = ["apple", "banana", "cherry"]
fun loop {
	for fruit in @var.items { echo "@var.fruit" }
}
`)
	output := runE2E(t, opalBin, "-f", testFile, "loop")
	assert.Contains(t, output, "apple")
	assert.Contains(t, output, "banana")
	assert.Contains(t, output, "cherry")
}

func TestE2EMeta_RetryTimeout(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
@timeout("5s")
@retry(2)
fun flaky = test -f /nonexistent || echo "success"
`)
	output := runE2E(t, opalBin, "-f", testFile, "flaky")
	assert.Contains(t, output, "success")
}

func TestE2EMeta_ParallelWithRetry(t *testing.T) {
	opalBin := buildE2EBinary(t)
	testFile := createE2ETestFile(t, `
@parallel
fun multi = (
	@retry(2) { echo "A" }
	@retry(2) { echo "B" }
)
`)
	output := runE2E(t, opalBin, "-f", testFile, "multi")
	assert.Contains(t, output, "A")
	assert.Contains(t, output, "B")
}
