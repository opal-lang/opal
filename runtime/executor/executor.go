package executor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/sdk"
	"github.com/aledsdavies/opal/core/types"
)

// Config configures the executor
type Config struct {
	Debug     DebugLevel     // Debug tracing (development only)
	Telemetry TelemetryLevel // Telemetry collection (production-safe)
}

// DebugLevel controls debug tracing (development only)
type DebugLevel int

const (
	DebugOff      DebugLevel = iota // No debug info (default)
	DebugPaths                      // Step entry/exit tracing
	DebugDetailed                   // Command output, timing details
)

// TelemetryLevel controls telemetry collection (production-safe)
type TelemetryLevel int

const (
	TelemetryOff    TelemetryLevel = iota // Zero overhead (default)
	TelemetryBasic                        // Step counts only
	TelemetryTiming                       // Counts + timing per step
)

// ExecutionResult holds the result of plan execution
type ExecutionResult struct {
	ExitCode    int                 // Final exit code (0 = success)
	Duration    time.Duration       // Total execution time
	StepsRun    int                 // Number of steps executed
	Telemetry   *ExecutionTelemetry // Additional metrics (nil if TelemetryOff)
	DebugEvents []DebugEvent        // Debug events (nil if DebugOff)
}

// ExecutionTelemetry holds additional execution metrics (optional, production-safe)
type ExecutionTelemetry struct {
	StepCount   int          // Total steps in plan
	StepsRun    int          // Steps actually executed
	StepTimings []StepTiming // Per-step timing (if TelemetryTiming)
	FailedStep  *uint64      // Step ID that failed (if any)
}

// StepTiming holds timing information for a single step
type StepTiming struct {
	StepID   uint64
	Duration time.Duration
	ExitCode int
}

// DebugEvent represents a debug trace event
type DebugEvent struct {
	Timestamp time.Time
	Event     string // "enter_execute", "step_start", "step_complete", etc.
	StepID    uint64 // Current step ID (0 if not step-specific)
	Context   string // Additional context
}

// executor holds execution state
type executor struct {
	config Config

	// Execution state
	stepsRun int
	exitCode int

	// Observability
	debugEvents []DebugEvent
	telemetry   *ExecutionTelemetry
	startTime   time.Time
}

// Execute runs SDK steps and returns the result.
// The executor only sees SDK types - it has no knowledge of planfmt.
// Secret scrubbing is handled by the CLI (stdout/stderr already locked down).
func Execute(steps []sdk.Step, config Config) (*ExecutionResult, error) {
	// INPUT CONTRACT (preconditions)
	invariant.NotNil(steps, "steps")

	e := &executor{
		config:    config,
		startTime: time.Now(),
	}

	// Initialize telemetry if enabled
	if config.Telemetry != TelemetryOff {
		e.telemetry = &ExecutionTelemetry{
			StepCount: len(steps),
		}
		if config.Telemetry == TelemetryTiming {
			e.telemetry.StepTimings = make([]StepTiming, 0, len(steps))
		}
	}

	// Record debug event: enter_execute
	if config.Debug >= DebugPaths {
		e.recordDebugEvent("enter_execute", 0, fmt.Sprintf("steps=%d", len(steps)))
	}

	// Execute all steps sequentially
	for _, step := range steps {
		stepStart := time.Now()

		if config.Debug >= DebugDetailed {
			e.recordDebugEvent("step_start", step.ID, fmt.Sprintf("commands=%d", len(step.Commands)))
		}

		exitCode := e.executeStep(step)
		e.stepsRun++

		stepDuration := time.Since(stepStart)

		// Record timing if enabled
		if config.Telemetry == TelemetryTiming {
			e.telemetry.StepTimings = append(e.telemetry.StepTimings, StepTiming{
				StepID:   step.ID,
				Duration: stepDuration,
				ExitCode: exitCode,
			})
		}

		if config.Debug >= DebugDetailed {
			e.recordDebugEvent("step_complete", step.ID, fmt.Sprintf("exit=%d, duration=%v", exitCode, stepDuration))
		}

		// Fail-fast: stop on first failure
		if exitCode != 0 {
			e.exitCode = exitCode
			if e.telemetry != nil {
				stepID := step.ID
				e.telemetry.FailedStep = &stepID
			}
			break
		}
	}

	// Update telemetry
	if e.telemetry != nil {
		e.telemetry.StepsRun = e.stepsRun
	}

	// Record debug event: exit_execute
	duration := time.Since(e.startTime)
	if config.Debug >= DebugPaths {
		e.recordDebugEvent("exit_execute", 0, fmt.Sprintf("steps_run=%d, exit=%d, duration=%v", e.stepsRun, e.exitCode, duration))
	}

	// OUTPUT CONTRACT (postconditions)
	invariant.InRange(e.exitCode, 0, 255, "exit code")
	invariant.Postcondition(e.stepsRun >= 0, "steps run must be non-negative")
	invariant.Postcondition(e.stepsRun <= len(steps), "steps run cannot exceed total steps")

	return &ExecutionResult{
		ExitCode:    e.exitCode,
		Duration:    duration,
		StepsRun:    e.stepsRun,
		Telemetry:   e.telemetry,
		DebugEvents: e.debugEvents,
	}, nil
}

// executeStep executes a single step using the decorator registry
func (e *executor) executeStep(step sdk.Step) int {
	// INPUT CONTRACT
	invariant.Precondition(len(step.Commands) > 0, "step must have at least one command")

	// For MVP: Only support single command per step (no operators yet)
	if len(step.Commands) > 1 {
		panic("operator chaining not yet implemented in Phase 3E")
	}

	cmd := step.Commands[0]

	// Strip @ prefix from decorator name for registry lookup
	decoratorName := strings.TrimPrefix(cmd.Name, "@")

	// Look up handler from registry
	handler, kind, exists := types.Global().GetSDKHandler(decoratorName)
	if !exists {
		panic(fmt.Sprintf("unknown decorator: %s", cmd.Name))
	}

	// Verify it's an execution decorator
	if kind != types.DecoratorKindExecution {
		panic(fmt.Sprintf("%s is not an execution decorator", cmd.Name))
	}

	// Type assert to SDK handler
	sdkHandler, ok := handler.(func(sdk.ExecutionContext, []sdk.Step) (int, error))
	if !ok {
		panic(fmt.Sprintf("invalid handler type for %s", cmd.Name))
	}

	// Create execution context with SDK args directly
	ctx := newExecutionContext(cmd.Args, e, context.Background())

	// Call handler with SDK block
	exitCode, err := sdkHandler(ctx, cmd.Block)
	if err != nil {
		// Log error but return exit code
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return exitCode
}

// recordDebugEvent records a debug event (only if debug enabled)
func (e *executor) recordDebugEvent(event string, stepID uint64, context string) {
	if e.config.Debug == DebugOff {
		return
	}

	e.debugEvents = append(e.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		StepID:    stepID,
		Context:   context,
	})
}
