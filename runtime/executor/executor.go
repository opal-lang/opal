package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
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
			e.recordDebugEvent("step_start", step.ID, "executing tree")
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

// executeStep executes a single step by executing its tree
func (e *executor) executeStep(step sdk.Step) int {
	// INPUT CONTRACT
	invariant.Precondition(step.Tree != nil, "step must have a tree")

	return e.executeTree(step.Tree)
}

// executeTree executes a tree node and returns the exit code
func (e *executor) executeTree(node sdk.TreeNode) int {
	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommand(n)

	case *sdk.PipelineNode:
		return e.executePipeline(n)

	case *sdk.AndNode:
		// Execute left, then right only if left succeeded
		leftExit := e.executeTree(n.Left)
		if leftExit != 0 {
			return leftExit // Short-circuit on failure
		}
		return e.executeTree(n.Right)

	case *sdk.OrNode:
		// Execute left, then right only if left failed
		leftExit := e.executeTree(n.Left)
		if leftExit == 0 {
			return leftExit // Short-circuit on success
		}
		return e.executeTree(n.Right)

	case *sdk.SequenceNode:
		// Execute all nodes, return last exit code
		var lastExit int
		for _, child := range n.Nodes {
			lastExit = e.executeTree(child)
		}
		return lastExit

	default:
		invariant.Invariant(false, "unknown TreeNode type: %T", node)
		return 1 // Unreachable
	}
}

// executePipeline executes a pipeline of commands with stdout→stdin streaming
// Uses io.Pipe() for streaming (bash-compatible: concurrent execution, not buffered)
// Returns exit code of last command (bash semantics)
func (e *executor) executePipeline(pipeline *sdk.PipelineNode) int {
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")

	// Single command - no piping needed
	if numCommands == 1 {
		return e.executeCommand(&pipeline.Commands[0])
	}

	// Create pipes between commands
	// For N commands, we need N-1 pipes
	pipes := make([]*io.PipeReader, numCommands-1)
	pipeWriters := make([]*io.PipeWriter, numCommands-1)
	for i := 0; i < numCommands-1; i++ {
		pipes[i], pipeWriters[i] = io.Pipe()
	}

	// Track exit codes for all commands (PIPESTATUS)
	exitCodes := make([]int, numCommands)

	// Execute all commands concurrently (bash behavior)
	var wg sync.WaitGroup
	wg.Add(numCommands)

	for i := 0; i < numCommands; i++ {
		cmdIndex := i
		cmd := &pipeline.Commands[i]

		go func() {
			defer wg.Done()

			// Determine stdin for this command
			var stdin io.Reader
			if cmdIndex > 0 {
				stdin = pipes[cmdIndex-1]
			}

			// Determine stdout for this command
			var stdout io.Writer
			if cmdIndex < numCommands-1 {
				stdout = pipeWriters[cmdIndex]
				defer func() {
					_ = pipeWriters[cmdIndex].Close() // Signal EOF to next command
				}()
			}

			// Execute command with pipes
			exitCodes[cmdIndex] = e.executeCommandWithPipes(cmd, stdin, stdout)
		}()
	}

	// Wait for all commands to complete
	wg.Wait()

	// Return last command's exit code (bash semantics)
	// TODO: Store PIPESTATUS in telemetry for debugging
	return exitCodes[numCommands-1]
}

// executeCommand executes a single command node
func (e *executor) executeCommand(cmd *sdk.CommandNode) int {
	return e.executeCommandWithPipes(cmd, nil, nil)
}

// executeCommandWithPipes executes a command with optional piped stdin/stdout
// stdin: piped input (nil if not piped)
// stdout: piped output (nil if not piped)
func (e *executor) executeCommandWithPipes(cmd *sdk.CommandNode, stdin io.Reader, stdout io.Writer) int {
	// Strip @ prefix from decorator name for registry lookup
	decoratorName := strings.TrimPrefix(cmd.Name, "@")

	// Look up handler from registry
	handler, kind, exists := types.Global().GetSDKHandler(decoratorName)
	invariant.Invariant(exists, "unknown decorator: %s", cmd.Name)

	// Verify it's an execution decorator
	invariant.Invariant(kind == types.DecoratorKindExecution, "%s is not an execution decorator", cmd.Name)

	// Type assert to SDK handler
	sdkHandler, ok := handler.(func(sdk.ExecutionContext, []sdk.Step) (int, error))
	invariant.Invariant(ok, "invalid handler type for %s", cmd.Name)

	// Create base execution context
	baseCtx := newExecutionContext(cmd.Args, e, context.Background())

	// Clone with pipes if needed
	var ctx sdk.ExecutionContext
	if stdin != nil || stdout != nil {
		ctx = baseCtx.Clone(cmd.Args, stdin, stdout)
	} else {
		ctx = baseCtx
	}

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
