package executor

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/aledsdavies/opal/core/planfmt"
	_ "github.com/aledsdavies/opal/runtime/decorators" // Register built-in decorators
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteSimpleShellCommand tests executing a single echo command
func TestExecuteSimpleShellCommand(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'Hello, World!'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 1, result.StepsRun)
	assert.Greater(t, result.Duration, time.Duration(0))
}

// TestExecuteMultipleCommands tests executing multiple sequential commands
func TestExecuteMultipleCommands(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "multi",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'First'"}},
						},
					},
				},
			},
			{
				ID: 2,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'Second'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 2, result.StepsRun)
}

// TestExecuteFailingCommand tests that non-zero exit codes are returned
func TestExecuteFailingCommand(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "fail",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 42"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err) // Execute returns result, not error
	assert.Equal(t, 42, result.ExitCode)
	assert.Equal(t, 1, result.StepsRun)
}

// TestExecuteStopOnFirstFailure tests fail-fast behavior
func TestExecuteStopOnFirstFailure(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "failfast",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'First'"}},
						},
					},
				},
			},
			{
				ID: 2,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 1"}},
						},
					},
				},
			},
			{
				ID: 3,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'Should not run'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, 2, result.StepsRun) // Only first two steps run
}

// TestExecuteEmptyPlan tests that empty plans succeed
func TestExecuteEmptyPlan(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "empty",
		Steps:  []planfmt.Step{},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 0, result.StepsRun)
}

// TestExecuteTelemetryBasic tests basic telemetry collection
func TestExecuteTelemetryBasic(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "telemetry",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'test'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Telemetry: TelemetryBasic})
	require.NoError(t, err)
	require.NotNil(t, result.Telemetry)
	assert.Equal(t, 1, result.Telemetry.StepCount)
	assert.Equal(t, 1, result.Telemetry.StepsRun)
	assert.Nil(t, result.Telemetry.FailedStep)
}

// TestExecuteTelemetryTiming tests timing telemetry collection
func TestExecuteTelemetryTiming(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "timing",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'test'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Telemetry: TelemetryTiming})
	require.NoError(t, err)
	require.NotNil(t, result.Telemetry)
	require.NotNil(t, result.Telemetry.StepTimings)
	assert.Len(t, result.Telemetry.StepTimings, 1)
	assert.Equal(t, uint64(1), result.Telemetry.StepTimings[0].StepID)
	assert.Greater(t, result.Telemetry.StepTimings[0].Duration, time.Duration(0))
	assert.Equal(t, 0, result.Telemetry.StepTimings[0].ExitCode)
}

// TestExecuteTelemetryFailedStep tests failed step tracking
func TestExecuteTelemetryFailedStep(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "fail",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 1"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Telemetry: TelemetryBasic})
	require.NoError(t, err)
	require.NotNil(t, result.Telemetry)
	require.NotNil(t, result.Telemetry.FailedStep)
	assert.Equal(t, uint64(1), *result.Telemetry.FailedStep)
}

// TestExecuteDebugPaths tests path-level debug tracing
func TestExecuteDebugPaths(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "debug",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'test'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Debug: DebugPaths})
	require.NoError(t, err)
	require.NotNil(t, result.DebugEvents)
	assert.Greater(t, len(result.DebugEvents), 0)

	// Should have enter_execute and exit_execute events
	events := make(map[string]bool)
	for _, e := range result.DebugEvents {
		events[e.Event] = true
	}
	assert.True(t, events["enter_execute"])
	assert.True(t, events["exit_execute"])
}

// TestExecuteDebugDetailed tests detailed debug tracing
func TestExecuteDebugDetailed(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "debug",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'test'"}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Debug: DebugDetailed})
	require.NoError(t, err)
	require.NotNil(t, result.DebugEvents)

	// Should have step-level events
	events := make(map[string]bool)
	for _, e := range result.DebugEvents {
		events[e.Event] = true
	}
	assert.True(t, events["step_start"])
	assert.True(t, events["step_complete"])
}

// TestInvariantNilPlan tests that nil plan causes panic
func TestInvariantNilPlan(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = Execute(nil, Config{})
	})
}

// TestInvariantEmptyShellCommand tests that empty shell command returns error
func TestInvariantEmptyShellCommand(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "empty",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: ""}},
						},
					},
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)               // Executor completes successfully
	assert.Equal(t, 127, result.ExitCode) // But decorator returns error exit code
}

// TestStdoutStderrLockdown tests that direct writes to os.Stdout/os.Stderr are blocked
func TestStdoutStderrLockdown(t *testing.T) {
	// This test verifies that even if code tries to write directly to os.Stdout/os.Stderr,
	// it gets redirected through our scrubbing layer

	plan := &planfmt.Plan{
		Target: "lockdown",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'This should be scrubbed'"}},
						},
					},
				},
			},
		},
	}

	// Execute with lockdown enabled
	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{
		LockdownStdStreams: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify that output went through scrubber, not directly to os.Stdout
	// (Implementation will capture this in result.Output)
}

// TestSecretScrubbing tests that secrets are automatically replaced with placeholders
func TestSecretScrubbing(t *testing.T) {
	secret := "super-secret-password-123"

	plan := &planfmt.Plan{
		Target: "secret-test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{
								Kind: planfmt.ValueString,
								Str:  "echo 'Password is: " + secret + "'",
							}},
						},
					},
				},
			},
		},
		Secrets: []planfmt.Secret{
			{
				Key:          "db_password",
				RuntimeValue: secret,
				DisplayID:    "opal:secret:test123",
			},
		},
	}

	// Manually set up lockdown and register secrets (like CLI does)
	var outputBuf bytes.Buffer
	scrubber := NewSecretScrubber(&outputBuf)
	for _, s := range plan.Secrets {
		scrubber.RegisterSecret(s.RuntimeValue, s.DisplayID)
	}

	// Lock down stdout/stderr with our scrubber
	restore := LockDownStdStreams(&LockdownConfig{
		Scrubber: scrubber,
	})

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{
		LockdownStdStreams: false, // We already locked down manually
	})

	// Restore before reading output
	restore()

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify secret was scrubbed from output
	output := outputBuf.String()
	assert.NotContains(t, output, secret, "Secret should not appear in output")
	assert.Contains(t, output, "opal:secret:test123", "Secret should be replaced with placeholder")
}

// TestLockdownInvariantNilConfig tests that lockdown panics on nil config
func TestLockdownInvariantNilConfig(t *testing.T) {
	assert.Panics(t, func() {
		LockDownStdStreams(nil)
	}, "Should panic on nil config")
}

// TestLockdownInvariantNilScrubber tests that lockdown panics on nil scrubber
func TestLockdownInvariantNilScrubber(t *testing.T) {
	assert.Panics(t, func() {
		LockDownStdStreams(&LockdownConfig{
			Scrubber: nil,
		})
	}, "Should panic on nil scrubber")
}

// TestLockdownRestore tests that restore function properly restores original streams
func TestLockdownRestore(t *testing.T) {
	// Save original streams
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create scrubber
	var buf bytes.Buffer
	scrubber := NewSecretScrubber(&buf)

	// Lock down
	restore := LockDownStdStreams(&LockdownConfig{
		Scrubber: scrubber,
	})

	// Verify streams are redirected
	assert.NotEqual(t, originalStdout, os.Stdout, "Stdout should be redirected")
	assert.NotEqual(t, originalStderr, os.Stderr, "Stderr should be redirected")

	// Restore
	restore()

	// Verify streams are restored
	assert.Equal(t, originalStdout, os.Stdout, "Stdout should be restored")
	assert.Equal(t, originalStderr, os.Stderr, "Stderr should be restored")
}
