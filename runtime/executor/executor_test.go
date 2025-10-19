package executor

import (
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

// TestOperatorSemicolon tests semicolon operator (always execute next)
func TestOperatorSemicolon(t *testing.T) {
	tests := []struct {
		name     string
		commands []planfmt.Command
		wantExit int
	}{
		{
			name: "all succeed",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'a'"}}},
					Operator:  ";",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'b'"}}},
					Operator:  ";",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'c'"}}},
					Operator:  "",
				},
			},
			wantExit: 0,
		},
		{
			name: "first fails, rest run",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 1"}}},
					Operator:  ";",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'still runs'"}}},
					Operator:  ";",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 0"}}},
					Operator:  "",
				},
			},
			wantExit: 0, // Last command succeeds
		},
		{
			name: "last fails",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'a'"}}},
					Operator:  ";",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 42"}}},
					Operator:  "",
				},
			},
			wantExit: 42, // Last command's exit code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID:       1,
						Commands: tt.commands,
					},
				},
			}

			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := Execute(steps, Config{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantExit, result.ExitCode)
			assert.Equal(t, 1, result.StepsRun)
		})
	}
}

// TestOperatorAND tests AND operator (execute next only if previous succeeded)
func TestOperatorAND(t *testing.T) {
	tests := []struct {
		name     string
		commands []planfmt.Command
		wantExit int
	}{
		{
			name: "both succeed",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'a'"}}},
					Operator:  "&&",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'b'"}}},
					Operator:  "",
				},
			},
			wantExit: 0,
		},
		{
			name: "first fails, second skipped",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 1"}}},
					Operator:  "&&",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'should not run'"}}},
					Operator:  "",
				},
			},
			wantExit: 1, // First command's exit code
		},
		{
			name: "chain of ANDs all succeed",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'a'"}}},
					Operator:  "&&",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'b'"}}},
					Operator:  "&&",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'c'"}}},
					Operator:  "",
				},
			},
			wantExit: 0,
		},
		{
			name: "chain of ANDs, middle fails",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'a'"}}},
					Operator:  "&&",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 42"}}},
					Operator:  "&&",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'should not run'"}}},
					Operator:  "",
				},
			},
			wantExit: 42, // Middle command's exit code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID:       1,
						Commands: tt.commands,
					},
				},
			}

			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := Execute(steps, Config{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantExit, result.ExitCode)
			assert.Equal(t, 1, result.StepsRun)
		})
	}
}

// TestOperatorOR tests OR operator (execute next only if previous failed)
func TestOperatorOR(t *testing.T) {
	tests := []struct {
		name     string
		commands []planfmt.Command
		wantExit int
	}{
		{
			name: "first succeeds, second skipped",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'success'"}}},
					Operator:  "||",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'should not run'"}}},
					Operator:  "",
				},
			},
			wantExit: 0,
		},
		{
			name: "first fails, second runs and succeeds",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 1"}}},
					Operator:  "||",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'fallback'"}}},
					Operator:  "",
				},
			},
			wantExit: 0, // Fallback succeeds
		},
		{
			name: "first fails, second fails too",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 1"}}},
					Operator:  "||",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 2"}}},
					Operator:  "",
				},
			},
			wantExit: 2, // Second command's exit code
		},
		{
			name: "chain of ORs, first succeeds",
			commands: []planfmt.Command{
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'success'"}}},
					Operator:  "||",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'fallback1'"}}},
					Operator:  "||",
				},
				{
					Decorator: "@shell",
					Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'fallback2'"}}},
					Operator:  "",
				},
			},
			wantExit: 0, // First succeeds, rest skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID:       1,
						Commands: tt.commands,
					},
				},
			}

			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := Execute(steps, Config{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantExit, result.ExitCode)
			assert.Equal(t, 1, result.StepsRun)
		})
	}
}
