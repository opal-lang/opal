package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommand tests basic command execution
func TestCommand(t *testing.T) {
	var stdout bytes.Buffer

	cmd := Command("echo", "hello")
	cmd.SetStdout(&stdout)

	exitCode, err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "hello\n", stdout.String())
}

// TestCommandContext tests command with context timeout
func TestCommandContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	var stdout bytes.Buffer

	// This command will be killed by context timeout (sleep 10 seconds)
	cmd := CommandContext(ctx, "sleep", "10")
	cmd.SetStdout(&stdout)

	exitCode, _ := cmd.Run()

	// Should return 124 (conventional timeout exit code)
	assert.Equal(t, 124, exitCode, "Context timeout should return exit code 124")
}

// TestBash tests bash script execution
func TestBash(t *testing.T) {
	var stdout bytes.Buffer

	cmd := Bash("echo 'hello from bash'")
	cmd.SetStdout(&stdout)

	exitCode, err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "hello from bash\n", stdout.String())
}

// TestBashWithOperators tests bash with operators
func TestBashWithOperators(t *testing.T) {
	var stdout bytes.Buffer

	cmd := Bash("echo 'first' && echo 'second'")
	cmd.SetStdout(&stdout)

	exitCode, err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout.String(), "first")
	assert.Contains(t, stdout.String(), "second")
}

// TestBashFailure tests bash command failure
func TestBashFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := Bash("exit 42")
	cmd.SetStdout(&stdout)
	cmd.SetStderr(&stderr)

	exitCode, err := cmd.Run()

	require.NoError(t, err) // No error, just non-zero exit
	assert.Equal(t, 42, exitCode)
}

// TestSetDir tests working directory setting
func TestSetDir(t *testing.T) {
	var stdout bytes.Buffer

	cmd := Bash("pwd")
	cmd.SetDir("/tmp")
	cmd.SetStdout(&stdout)

	exitCode, err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.True(t, strings.HasPrefix(stdout.String(), "/tmp"))
}

// TestStartWait tests Start/Wait pattern
func TestStartWait(t *testing.T) {
	var stdout bytes.Buffer

	cmd := Command("echo", "async")
	cmd.SetStdout(&stdout)

	err := cmd.Start()
	require.NoError(t, err)

	exitCode, err := cmd.Wait()
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "async\n", stdout.String())
}

// TestCommandNotFound tests handling of missing commands
func TestCommandNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := Command("nonexistent-command-xyz")
	cmd.SetStdout(&stdout)
	cmd.SetStderr(&stderr)

	exitCode, err := cmd.Run()

	assert.Error(t, err)
	assert.Equal(t, 127, exitCode)
}

// TestInvariantEmptyCommandName tests panic on empty command name
func TestInvariantEmptyCommandName(t *testing.T) {
	assert.Panics(t, func() {
		Command("")
	})
}

// TestInvariantEmptyBashScript tests panic on empty bash script
func TestInvariantEmptyBashScript(t *testing.T) {
	assert.Panics(t, func() {
		Bash("")
	})
}

// TestInvariantNilContext tests panic on nil context
func TestInvariantNilContext(t *testing.T) {
	assert.Panics(t, func() {
		//nolint:staticcheck // Testing nil context handling
		CommandContext(nil, "echo", "test")
	})
}

// TestInvariantNilStdout tests panic on nil stdout
func TestInvariantNilStdout(t *testing.T) {
	cmd := Command("echo", "test")
	assert.Panics(t, func() {
		cmd.SetStdout(nil)
	})
}

// TestInvariantNilStderr tests panic on nil stderr
func TestInvariantNilStderr(t *testing.T) {
	cmd := Command("echo", "test")
	assert.Panics(t, func() {
		cmd.SetStderr(nil)
	})
}

// TestInvariantEmptyDir tests panic on empty directory
func TestInvariantEmptyDir(t *testing.T) {
	cmd := Command("echo", "test")
	assert.Panics(t, func() {
		cmd.SetDir("")
	})
}

// TestAppendEnv tests adding environment variables while preserving PATH
func TestAppendEnv(t *testing.T) {
	var stdout bytes.Buffer

	cmd := Bash("echo $MY_VAR:$PATH")
	cmd.AppendEnv(map[string]string{"MY_VAR": "test-value"})
	cmd.SetStdout(&stdout)

	exitCode, err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	output := stdout.String()
	assert.Contains(t, output, "test-value")
	// PATH should still be present
	assert.Contains(t, output, "/")
}

// TestSetStdin tests feeding input to command
func TestSetStdin(t *testing.T) {
	var stdout bytes.Buffer
	stdin := strings.NewReader("hello from stdin")

	cmd := Command("cat")
	cmd.SetStdin(stdin)
	cmd.SetStdout(&stdout)

	exitCode, err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "hello from stdin", stdout.String())
}

// TestInvariantNilStdin tests panic on nil stdin
func TestInvariantNilStdin(t *testing.T) {
	cmd := Command("cat")
	assert.Panics(t, func() {
		cmd.SetStdin(nil)
	})
}

// TestInvariantNilAppendEnv tests panic on nil env map
func TestInvariantNilAppendEnv(t *testing.T) {
	cmd := Command("echo", "test")
	assert.Panics(t, func() {
		cmd.AppendEnv(nil)
	})
}

// TestInvariantEmptyEnvKey tests panic on empty env key
func TestInvariantEmptyEnvKey(t *testing.T) {
	cmd := Command("echo", "test")
	assert.Panics(t, func() {
		cmd.AppendEnv(map[string]string{"": "value"})
	})
}

// TestOutput tests capturing stdout
func TestOutput(t *testing.T) {
	cmd := Command("echo", "test output")

	output, exitCode, err := cmd.Output()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "test output\n", string(output))
}

// TestCombinedOutput tests capturing stdout+stderr
func TestCombinedOutput(t *testing.T) {
	cmd := Bash("echo 'stdout' && echo 'stderr' >&2")

	output, exitCode, err := cmd.CombinedOutput()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, string(output), "stdout")
	assert.Contains(t, string(output), "stderr")
}

// TestStartWaitCancellation tests that Start+Wait normalizes timeout to 124
func TestStartWaitCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	var stdout bytes.Buffer

	cmd := CommandContext(ctx, "sleep", "10")
	cmd.SetStdout(&stdout)

	err := cmd.Start()
	require.NoError(t, err)

	exitCode, _ := cmd.Wait()

	// Should return 124 (same as Run() with timeout)
	assert.Equal(t, 124, exitCode, "Start+Wait should normalize timeout to 124")
}
