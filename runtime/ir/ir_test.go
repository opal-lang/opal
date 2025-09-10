package ir_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test context with environment
func testCtx(env map[string]string) *ir.Ctx {
	// Create realistic frozen environment using NewEnvSnapshot
	snapshot, err := ir.NewEnvSnapshot(ir.EnvOptions{})
	if err != nil {
		panic(fmt.Sprintf("Failed to create environment snapshot: %v", err))
	}

	// Override with custom test values
	if len(env) > 0 {
		for k, v := range env {
			snapshot.Values[k] = v
		}
	}

	return &ir.Ctx{
		Env:  snapshot,
		Vars: make(map[string]string),
	}
}

// Helper to create context with extra settings
func testCtxWithOpts(env map[string]string, workDir string, dryRun bool) *ir.Ctx {
	ctx := testCtx(env)
	ctx.WorkDir = workDir
	ctx.DryRun = dryRun
	return ctx
}

// TestExecShell tests the ExecShell primitive
func TestExecShell(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *ir.Ctx
		cmd      string
		wantExit int
		wantOut  string
		wantErr  string
		contains bool // if true, use Contains instead of Equal for output
	}{
		{
			name:     "simple echo command",
			ctx:      testCtx(map[string]string{}),
			cmd:      "echo hello",
			wantExit: 0,
			wantOut:  "hello\n",
		},
		{
			name:     "command with exit code",
			ctx:      testCtx(map[string]string{}),
			cmd:      "exit 42",
			wantExit: 42,
		},
		{
			name:     "command with stderr",
			ctx:      testCtx(map[string]string{}),
			cmd:      "echo error >&2",
			wantExit: 0,
			wantErr:  "error\n",
		},
		{
			name: "command with environment variable",
			ctx: testCtx(map[string]string{
				"TEST_VAR": "test_value",
			}),
			cmd:      "echo $TEST_VAR",
			wantExit: 0,
			wantOut:  "test_value\n",
		},
		{
			name:     "command with working directory",
			ctx:      testCtxWithOpts(map[string]string{}, "/tmp", false),
			cmd:      "pwd",
			wantExit: 0,
			wantOut:  "/tmp\n",
		},
		{
			name:     "dry run mode",
			ctx:      testCtxWithOpts(map[string]string{}, "", true),
			cmd:      "rm -rf /",
			wantExit: 0,
			wantOut:  "[DRY-RUN] rm -rf /",
		},
		{
			name: "command with multiple environment variables",
			ctx: testCtx(map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			}),
			cmd:      "echo $VAR1-$VAR2-$VAR3",
			wantExit: 0,
			wantOut:  "value1-value2-value3\n",
		},
		{
			name:     "command not found",
			ctx:      testCtx(map[string]string{}),
			cmd:      "nonexistent_command_12345",
			wantExit: 127,
			contains: true,
			wantErr:  "not found",
		},
		{
			name:     "complex shell command with pipes",
			ctx:      testCtx(map[string]string{}),
			cmd:      "echo 'hello world' | grep world",
			wantExit: 0,
			wantOut:  "hello world\n",
		},
		{
			name:     "command with && operator",
			ctx:      testCtx(map[string]string{}),
			cmd:      "echo first && echo second",
			wantExit: 0,
			wantOut:  "first\nsecond\n",
		},
		{
			name:     "command with || operator",
			ctx:      testCtx(map[string]string{}),
			cmd:      "false || echo fallback",
			wantExit: 0,
			wantOut:  "fallback\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ir.ExecShell(tt.ctx, tt.cmd)

			assert.Equal(t, tt.wantExit, result.ExitCode, "exit code mismatch")

			if tt.contains {
				if tt.wantOut != "" {
					assert.Contains(t, result.Stdout, tt.wantOut, "stdout mismatch")
				}
				if tt.wantErr != "" {
					assert.Contains(t, result.Stderr, tt.wantErr, "stderr mismatch")
				}
			} else {
				if tt.wantOut != "" {
					assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch")
				}
				if tt.wantErr != "" {
					assert.Equal(t, tt.wantErr, result.Stderr, "stderr mismatch")
				}
			}
		})
	}
}

// TestExecShellWithInput tests the ExecShellWithInput primitive
func TestExecShellWithInput(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *ir.Ctx
		cmd      string
		input    string
		wantExit int
		wantOut  string
		wantErr  string
	}{
		{
			name:     "cat with input",
			ctx:      testCtx(map[string]string{}),
			cmd:      "cat",
			input:    "hello from stdin",
			wantExit: 0,
			wantOut:  "hello from stdin",
		},
		{
			name:     "grep with input",
			ctx:      testCtx(map[string]string{}),
			cmd:      "grep world",
			input:    "hello world\ngoodbye moon\nworld peace",
			wantExit: 0,
			wantOut:  "hello world\nworld peace\n",
		},
		{
			name:     "wc with input",
			ctx:      testCtx(map[string]string{}),
			cmd:      "wc -l",
			input:    "line1\nline2\nline3\n",
			wantExit: 0,
			wantOut:  "3\n",
		},
		{
			name:     "sed with input",
			ctx:      testCtx(map[string]string{}),
			cmd:      "sed 's/old/new/g'",
			input:    "old text with old words",
			wantExit: 0,
			wantOut:  "new text with new words",
		},
		{
			name:     "dry run with input",
			ctx:      testCtxWithOpts(map[string]string{}, "", true),
			cmd:      "dangerous_command",
			input:    "some input",
			wantExit: 0,
			wantOut:  "[DRY-RUN] dangerous_command (piped input)",
		},
		{
			name:     "multiline input processing",
			ctx:      testCtx(map[string]string{}),
			cmd:      "sort",
			input:    "charlie\nalpha\nbravo\n",
			wantExit: 0,
			wantOut:  "alpha\nbravo\ncharlie\n",
		},
		{
			name: "command with env and input",
			ctx: testCtx(map[string]string{
				"PREFIX": ">>",
			}),
			cmd:      "sed \"s/^/$PREFIX /\"",
			input:    "line1\nline2",
			wantExit: 0,
			wantOut:  ">> line1\n>> line2",
		},
		{
			name:     "empty input",
			ctx:      testCtx(map[string]string{}),
			cmd:      "cat",
			input:    "",
			wantExit: 0,
			wantOut:  "",
		},
		{
			name:     "large input handling",
			ctx:      testCtx(map[string]string{}),
			cmd:      "wc -c",
			input:    strings.Repeat("x", 10000),
			wantExit: 0,
			wantOut:  "10000\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ir.ExecShellWithInput(tt.ctx, tt.cmd, tt.input)

			assert.Equal(t, tt.wantExit, result.ExitCode, "exit code mismatch")
			assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch")
			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, result.Stderr, "stderr mismatch")
			}
		})
	}
}

// TestAppendToFile tests the AppendToFile primitive
func TestAppendToFile(t *testing.T) {
	tests := []struct {
		name        string
		initialData string
		appendData  string
		wantContent string
		wantError   bool
	}{
		{
			name:        "append to new file",
			initialData: "",
			appendData:  "first line\n",
			wantContent: "first line\n",
		},
		{
			name:        "append to existing file",
			initialData: "existing content\n",
			appendData:  "new line\n",
			wantContent: "existing content\nnew line\n",
		},
		{
			name:        "append multiple times",
			initialData: "line 1\n",
			appendData:  "line 2\nline 3\n",
			wantContent: "line 1\nline 2\nline 3\n",
		},
		{
			name:        "append empty string",
			initialData: "content",
			appendData:  "",
			wantContent: "content",
		},
		{
			name:        "append with special characters",
			initialData: "start\n",
			appendData:  "special: !@#$%^&*()_+\n",
			wantContent: "start\nspecial: !@#$%^&*()_+\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")

			// Write initial data if any
			if tt.initialData != "" {
				err := os.WriteFile(tmpFile, []byte(tt.initialData), 0o644)
				require.NoError(t, err)
			}

			// Test append
			err := ir.AppendToFile(tmpFile, tt.appendData)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Read and verify content
			content, err := os.ReadFile(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, string(content))
		})
	}
}

// TestCommandResultMethods tests CommandResult helper methods
func TestCommandResultMethods(t *testing.T) {
	t.Run("Success method", func(t *testing.T) {
		result := ir.CommandResult{ExitCode: 0}
		assert.True(t, result.Success())
		assert.False(t, result.Failed())

		result.ExitCode = 1
		assert.False(t, result.Success())
		assert.True(t, result.Failed())
	})

	t.Run("Error method", func(t *testing.T) {
		result := ir.CommandResult{ExitCode: 0}
		assert.NoError(t, result.Error())

		result = ir.CommandResult{
			ExitCode: 1,
			Stderr:   "command failed",
		}
		err := result.Error()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exit code 1")
		assert.Contains(t, err.Error(), "command failed")

		result = ir.CommandResult{
			ExitCode: 127,
			Stderr:   "",
		}
		err = result.Error()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exit code 127")
	})
}

// TestCtxMethods tests Ctx helper methods
func TestCtxMethods(t *testing.T) {
	t.Run("Clone method", func(t *testing.T) {
		env := &ir.EnvSnapshot{
			Values: map[string]string{
				"PATH": "/usr/bin",
				"USER": "test",
			},
			Fingerprint: "abc123",
		}

		ctx := &ir.Ctx{
			Env: env,
			Vars: map[string]string{
				"BUILD_DIR": "./build",
				"PORT":      "8080",
			},
			WorkDir: "/tmp",
			DryRun:  true,
			Debug:   true,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Stdin:   os.Stdin,
		}

		cloned := ctx.Clone()

		// Verify all fields are copied
		assert.Equal(t, ctx.WorkDir, cloned.WorkDir)
		assert.Equal(t, ctx.DryRun, cloned.DryRun)
		assert.Equal(t, ctx.Debug, cloned.Debug)
		assert.Equal(t, ctx.Stdout, cloned.Stdout)
		assert.Equal(t, ctx.Stderr, cloned.Stderr)
		assert.Equal(t, ctx.Stdin, cloned.Stdin)

		// EnvSnapshot is shared (immutable)
		assert.Same(t, ctx.Env, cloned.Env)

		// Vars map is deep copied
		assert.Equal(t, ctx.Vars, cloned.Vars)
		// Maps are reference types, so we test by modification
		originalBuildDir := cloned.Vars["BUILD_DIR"]

		// Modify original and verify clone is unaffected
		ctx.Vars["BUILD_DIR"] = "modified"
		assert.Equal(t, originalBuildDir, cloned.Vars["BUILD_DIR"])
		assert.NotEqual(t, ctx.Vars["BUILD_DIR"], cloned.Vars["BUILD_DIR"])
	})

	t.Run("WithWorkDir method", func(t *testing.T) {
		ctx := testCtx(map[string]string{"VAR": "value"})
		ctx.WorkDir = "/original"

		newCtx := ctx.WithWorkDir("/new")

		// Original unchanged
		assert.Equal(t, "/original", ctx.WorkDir)

		// New context has updated workdir
		assert.Equal(t, "/new", newCtx.WorkDir)

		// Other fields preserved
		assert.Equal(t, ctx.Env, newCtx.Env)
		assert.Equal(t, ctx.Vars, newCtx.Vars)
	})
}

// TestExecShellWithStdin tests that ctx.Stdin is properly connected
func TestExecShellWithStdin(t *testing.T) {
	input := "test input from ctx.Stdin"
	ctx := testCtx(map[string]string{})
	ctx.Stdin = strings.NewReader(input)

	result := ir.ExecShell(ctx, "cat")

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, input, result.Stdout)
}

// TestExecShellDebugMode tests debug output
func TestExecShellDebugMode(t *testing.T) {
	var debugOutput strings.Builder

	ctx := testCtx(map[string]string{})
	ctx.Debug = true
	ctx.Stderr = &debugOutput

	result := ir.ExecShell(ctx, "echo test")

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "test\n", result.Stdout)

	// Check debug output
	debugStr := debugOutput.String()
	assert.Contains(t, debugStr, "[DEBUG]")
	assert.Contains(t, debugStr, "echo test")
	assert.Contains(t, debugStr, "Exit Code: 0")
}
