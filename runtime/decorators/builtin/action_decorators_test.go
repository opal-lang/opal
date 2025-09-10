package builtin

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create test context for decorator tests
func createTestCtx() *decorators.Ctx {
	return &decorators.Ctx{
		Env:     &ir.EnvSnapshot{Values: map[string]string{}},
		Vars:    map[string]string{},
		WorkDir: "",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		DryRun:  false,
		Debug:   false,
		NumCPU:  4, // Deterministic value for tests
	}
}

// TestLogDecorator tests the @log action decorator
func TestLogDecorator(t *testing.T) {
	tests := []struct {
		name       string
		params     []decorators.DecoratorParam
		wantOut    string
		wantErr    string
		wantExit   int
		checkColor bool
	}{
		{
			name: "simple log message",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "Hello world"},
			},
			wantOut:  "Hello world\n",
			wantExit: 0,
		},
		{
			name: "log message with level",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "Error occurred"},
				{Name: "level", Value: "error"},
			},
			wantOut:  "Error occurred\n",
			wantExit: 0,
		},
		{
			name: "log message with color formatting",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "{green}Success!{/green}"},
			},
			wantOut:    "\033[32mSuccess!\033[0m\n",
			wantExit:   0,
			checkColor: true,
		},
		{
			name: "log message with multiple colors",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "{red}Error:{/red} {yellow}Warning message{/yellow}"},
			},
			wantOut:    "\033[31mError:\033[0m \033[33mWarning message\033[0m\n",
			wantExit:   0,
			checkColor: true,
		},
		{
			name: "empty message error",
			params: []decorators.DecoratorParam{
				{Name: "", Value: ""},
			},
			wantErr:  "@log parameter error: @log requires a message",
			wantExit: 1,
		},
		{
			name: "invalid log level",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "Test message"},
				{Name: "level", Value: "invalid"},
			},
			wantErr:  "@log parameter error: invalid log level \"invalid\", must be one of: debug, info, warn, error",
			wantExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewLogDecorator()
			ctx := createTestCtx()

			result := log.Run(ctx, tt.params)

			assert.Equal(t, tt.wantExit, result.ExitCode, "unexpected exit code")

			if tt.checkColor {
				// When checking color output, use exact match
				assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch with color codes")
			} else {
				// For normal output, check content
				if tt.wantOut != "" {
					assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch")
				}
			}

			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, result.Stderr, "stderr mismatch")
			}
		})
	}
}

// TestLogDecoratorDescribe tests the plan/describe functionality
func TestLogDecoratorDescribe(t *testing.T) {
	tests := []struct {
		name        string
		params      []decorators.DecoratorParam
		wantDesc    string
		wantCommand string
	}{
		{
			name: "simple message describe",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "Building project"},
			},
			wantDesc:    "Log: [INFO] Building project",
			wantCommand: "Log: [INFO] Building project",
		},
		{
			name: "error level describe",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "Build failed"},
				{Name: "level", Value: "error"},
			},
			wantDesc:    "Log: [ERROR] Build failed",
			wantCommand: "Log: [ERROR] Build failed",
		},
		{
			name: "long message truncation",
			params: []decorators.DecoratorParam{
				{Name: "", Value: strings.Repeat("a", 70)},
			},
			wantDesc:    "Log: [INFO] " + strings.Repeat("a", 65) + "...",
			wantCommand: "Log: [INFO] " + strings.Repeat("a", 65) + "...",
		},
		{
			name: "message with color templates removed",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "{green}Success message{/green}"},
			},
			wantDesc:    "Log: [INFO] Success message",
			wantCommand: "Log: [INFO] Success message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewLogDecorator()
			ctx := createTestCtx()

			step := log.Describe(ctx, tt.params)

			assert.Equal(t, tt.wantDesc, step.Description, "description mismatch")
			assert.Equal(t, tt.wantCommand, step.Command, "command mismatch")
			assert.Equal(t, "log", step.Metadata["decorator"], "decorator metadata mismatch")
		})
	}
}

// TestCmdDecorator tests the @cmd action decorator
func TestCmdDecorator(t *testing.T) {
	tests := []struct {
		name     string
		params   []decorators.DecoratorParam
		wantOut  string
		wantExit int
	}{
		{
			name: "simple command reference",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "build"},
			},
			wantOut:  "[TODO: Execute command 'build']",
			wantExit: 0,
		},
		{
			name: "named parameter",
			params: []decorators.DecoratorParam{
				{Name: "name", Value: "test"},
			},
			wantOut:  "[TODO: Execute command 'test']",
			wantExit: 0,
		},
		{
			name: "empty command name error",
			params: []decorators.DecoratorParam{
				{Name: "", Value: ""},
			},
			wantExit: 1,
		},
		{
			name:     "no parameters error",
			params:   []decorators.DecoratorParam{},
			wantExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdDecorator()
			ctx := createTestCtx()

			result := cmd.Run(ctx, tt.params)

			assert.Equal(t, tt.wantExit, result.ExitCode, "unexpected exit code")

			if tt.wantOut != "" {
				assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch")
			}
		})
	}
}

// TestCmdDecoratorDescribe tests the plan/describe functionality
func TestCmdDecoratorDescribe(t *testing.T) {
	tests := []struct {
		name        string
		params      []decorators.DecoratorParam
		wantDesc    string
		wantCommand string
	}{
		{
			name: "simple command describe",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "build"},
			},
			wantDesc:    "@cmd(build)",
			wantCommand: "# Execute command: build",
		},
		{
			name: "named parameter describe",
			params: []decorators.DecoratorParam{
				{Name: "name", Value: "test-all"},
			},
			wantDesc:    "@cmd(test-all)",
			wantCommand: "# Execute command: test-all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdDecorator()
			ctx := createTestCtx()

			step := cmd.Describe(ctx, tt.params)

			assert.Equal(t, tt.wantDesc, step.Description, "description mismatch")
			assert.Equal(t, tt.wantCommand, step.Command, "command mismatch")
			assert.Equal(t, "cmd", step.Metadata["decorator"], "decorator metadata mismatch")
		})
	}
}

// TestActionDecoratorsParameterSchema tests parameter schemas are correct
func TestActionDecoratorsParameterSchema(t *testing.T) {
	t.Run("LogDecorator schema", func(t *testing.T) {
		log := NewLogDecorator()
		schema := log.ParameterSchema()

		require.Len(t, schema, 3, "expected 3 parameters")

		// Check message parameter
		assert.Equal(t, "message", schema[0].Name)
		assert.Equal(t, decorators.ArgTypeString, schema[0].Type)
		assert.True(t, schema[0].Required)

		// Check level parameter
		assert.Equal(t, "level", schema[1].Name)
		assert.Equal(t, decorators.ArgTypeString, schema[1].Type)
		assert.False(t, schema[1].Required)

		// Check plain parameter
		assert.Equal(t, "plain", schema[2].Name)
		assert.Equal(t, decorators.ArgTypeBool, schema[2].Type)
		assert.False(t, schema[2].Required)
	})

	t.Run("CmdDecorator schema", func(t *testing.T) {
		cmd := NewCmdDecorator()
		schema := cmd.ParameterSchema()

		require.Len(t, schema, 1, "expected 1 parameter")

		// Check name parameter
		assert.Equal(t, "name", schema[0].Name)
		assert.Equal(t, decorators.ArgTypeIdentifier, schema[0].Type)
		assert.True(t, schema[0].Required)
	})
}

// TestActionDecoratorsRegistration tests that decorators are registered correctly
func TestActionDecoratorsRegistration(t *testing.T) {
	registry := decorators.GlobalRegistry()

	t.Run("log decorator registered", func(t *testing.T) {
		logDecorator, found := registry.GetAction("log")
		require.True(t, found, "log decorator should be registered")
		assert.Equal(t, "log", logDecorator.Name())
	})

	t.Run("cmd decorator registered", func(t *testing.T) {
		cmdDecorator, found := registry.GetAction("cmd")
		require.True(t, found, "cmd decorator should be registered")
		assert.Equal(t, "cmd", cmdDecorator.Name())
	})
}

// TestActionDecoratorsInShellChain tests action decorators work in shell command chains
func TestActionDecoratorsInShellChain(t *testing.T) {
	t.Run("log in chain with shell commands", func(t *testing.T) {
		// This test verifies that @log can be used in shell chains like:
		// @log("Starting") && echo "test" && @log("Done")

		log := NewLogDecorator()
		ctx := createTestCtx()

		// Test first log in chain
		params1 := []decorators.DecoratorParam{{Name: "", Value: "Starting process"}}
		result1 := log.Run(ctx, params1)
		assert.Equal(t, 0, result1.ExitCode, "first log should succeed")
		assert.Equal(t, "Starting process\n", result1.Stdout, "first log output")

		// Test second log in chain
		params2 := []decorators.DecoratorParam{{Name: "", Value: "Process complete"}}
		result2 := log.Run(ctx, params2)
		assert.Equal(t, 0, result2.ExitCode, "second log should succeed")
		assert.Equal(t, "Process complete\n", result2.Stdout, "second log output")
	})

	t.Run("cmd in chain with shell commands", func(t *testing.T) {
		// This test verifies that @cmd can be used in shell chains like:
		// @cmd(build) && echo "deployed"

		cmd := NewCmdDecorator()
		ctx := createTestCtx()

		// Test cmd in chain (placeholder implementation)
		params := []decorators.DecoratorParam{{Name: "", Value: "build"}}
		result := cmd.Run(ctx, params)
		assert.Equal(t, 0, result.ExitCode, "cmd should succeed")
		assert.Contains(t, result.Stdout, "build", "cmd output should mention command")
	})
}

// TestLogDecoratorQuietMode tests that log respects quiet flag
func TestLogDecoratorQuietMode(t *testing.T) {
	log := NewLogDecorator()

	t.Run("quiet mode suppresses info logs", func(t *testing.T) {
		ctx := createTestCtx()
		ctx.UI = &decorators.UIConfig{Quiet: true}

		params := []decorators.DecoratorParam{
			{Name: "", Value: "Info message"},
			{Name: "level", Value: "info"},
		}

		result := log.Run(ctx, params)
		assert.Equal(t, 0, result.ExitCode, "should succeed")
		assert.Equal(t, "", result.Stdout, "quiet mode should suppress info logs")
		assert.Equal(t, "", result.Stderr, "no stderr expected")
	})

	t.Run("quiet mode shows error logs", func(t *testing.T) {
		ctx := createTestCtx()
		ctx.UI = &decorators.UIConfig{Quiet: true}

		params := []decorators.DecoratorParam{
			{Name: "", Value: "Error message"},
			{Name: "level", Value: "error"},
			{Name: "plain", Value: false}, // Set plain=false so errors go to stderr
		}

		result := log.Run(ctx, params)
		assert.Equal(t, 0, result.ExitCode, "should succeed")
		assert.Equal(t, "Error message\n", result.Stdout, "quiet mode should show error logs")
		assert.Equal(t, "", result.Stderr, "error goes to stdout")
	})
}

// TestLogDecoratorMultilineAndMultipleLogs tests multiline strings and multiple log calls
func TestLogDecoratorMultilineAndMultipleLogs(t *testing.T) {
	log := NewLogDecorator()

	t.Run("multiline string with backticks", func(t *testing.T) {
		ctx := createTestCtx()

		multilineMessage := "Line 1\nLine 2\nLine 3"
		params := []decorators.DecoratorParam{
			{Name: "", Value: multilineMessage},
		}

		result := log.Run(ctx, params)
		assert.Equal(t, 0, result.ExitCode, "should succeed")
		assert.Equal(t, "Line 1\nLine 2\nLine 3\n", result.Stdout, "multiline output should preserve newlines")
	})

	t.Run("multiple log calls in sequence", func(t *testing.T) {
		ctx := createTestCtx()

		// First log
		params1 := []decorators.DecoratorParam{{Name: "", Value: "First message"}}
		result1 := log.Run(ctx, params1)
		assert.Equal(t, 0, result1.ExitCode, "first log should succeed")
		assert.Equal(t, "First message\n", result1.Stdout, "first log output")

		// Second log
		params2 := []decorators.DecoratorParam{{Name: "", Value: "Second message"}}
		result2 := log.Run(ctx, params2)
		assert.Equal(t, 0, result2.ExitCode, "second log should succeed")
		assert.Equal(t, "Second message\n", result2.Stdout, "second log output")
	})

	t.Run("multiline with different levels", func(t *testing.T) {
		ctx := createTestCtx()

		// Info level multiline
		infoMultiline := "Info:\nMultiple\nLines"
		paramsInfo := []decorators.DecoratorParam{
			{Name: "", Value: infoMultiline},
			{Name: "level", Value: "info"},
		}

		resultInfo := log.Run(ctx, paramsInfo)
		assert.Equal(t, 0, resultInfo.ExitCode, "info multiline should succeed")
		assert.Equal(t, "Info:\nMultiple\nLines\n", resultInfo.Stdout, "info multiline output")

		// Error level multiline
		errorMultiline := "Error:\nSomething\nWent wrong"
		paramsError := []decorators.DecoratorParam{
			{Name: "", Value: errorMultiline},
			{Name: "level", Value: "error"},
		}

		resultError := log.Run(ctx, paramsError)
		assert.Equal(t, 0, resultError.ExitCode, "error multiline should succeed")
		assert.Equal(t, "Error:\nSomething\nWent wrong\n", resultError.Stdout, "error multiline goes to stdout")
		assert.Equal(t, "", resultError.Stderr, "error doesn't go to stderr")
	})
}

// TestLogDecoratorDescribeMultiline tests describe method with multiline messages
func TestLogDecoratorDescribeMultiline(t *testing.T) {
	log := NewLogDecorator()

	t.Run("multiline message gets truncated in describe", func(t *testing.T) {
		ctx := createTestCtx()

		multilineMessage := "First line\nSecond line\nThird line"
		params := []decorators.DecoratorParam{
			{Name: "", Value: multilineMessage},
		}

		step := log.Describe(ctx, params)

		// Should show first line with "..." indicating more content
		expected := "Log: [INFO] First line ..."
		assert.Equal(t, expected, step.Description, "multiline should be truncated in describe")

		// Metadata should indicate multiple lines
		assert.Equal(t, "3", step.Metadata["lines"], "metadata should show line count")
	})

	t.Run("multiline error message describe", func(t *testing.T) {
		ctx := createTestCtx()

		multilineMessage := "Error occurred:\nStack trace line 1\nStack trace line 2"
		params := []decorators.DecoratorParam{
			{Name: "", Value: multilineMessage},
			{Name: "level", Value: "error"},
		}

		step := log.Describe(ctx, params)

		// Should show first line with proper error level
		expected := "Log: [ERROR] Error occurred: ..."
		assert.Equal(t, expected, step.Description, "error multiline should be truncated in describe")

		// Metadata should indicate error level and multiple lines
		assert.Equal(t, "error", step.Metadata["level"], "metadata should show error level")
		assert.Equal(t, "3", step.Metadata["lines"], "metadata should show line count")
	})
}
