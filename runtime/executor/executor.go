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

	case *sdk.RedirectNode:
		return e.executeRedirect(n)

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
		return e.executeTreeNode(pipeline.Commands[0], nil, nil)
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
		node := pipeline.Commands[i]

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

			// Execute tree node (CommandNode or RedirectNode) with pipes
			exitCodes[cmdIndex] = e.executeTreeNode(node, stdin, stdout)
		}()
	}

	// Wait for all commands to complete
	wg.Wait()

	// Return last command's exit code (bash semantics)
	// TODO: Store PIPESTATUS in telemetry for debugging
	return exitCodes[numCommands-1]
}

// executeTreeNode executes a tree node (CommandNode or RedirectNode) with optional pipes
// This is used by executePipeline to handle both commands and redirects in pipelines
func (e *executor) executeTreeNode(node sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommandWithPipes(n, stdin, stdout)
	case *sdk.RedirectNode:
		// For redirect in pipeline, we need to handle it specially
		// The redirect's source gets the piped stdin, and its output goes to the sink
		// The piped stdout (if any) is ignored because redirect captures output
		if stdout != nil {
			// If there's a piped stdout, we need to close it to signal EOF
			// But we can't write to it because output goes to redirect sink
			if closer, ok := stdout.(io.Closer); ok {
				defer func() {
					_ = closer.Close() // Ignore close error - nothing we can do
				}()
			}
		}
		// Execute redirect with piped stdin
		return e.executeRedirectWithStdin(n, stdin)
	default:
		invariant.Invariant(false, "invalid pipeline element type %T", node)
		return 1
	}
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

	// Type assert to SDK handler (function or struct with Execute method)
	var sdkHandler func(sdk.ExecutionContext, []sdk.Step) (int, error)

	// Try function first
	if fn, ok := handler.(func(sdk.ExecutionContext, []sdk.Step) (int, error)); ok {
		sdkHandler = fn
	} else {
		// Try struct with Execute method
		type ExecutionDecorator interface {
			Execute(sdk.ExecutionContext, []sdk.Step) (int, error)
		}
		if decorator, ok := handler.(ExecutionDecorator); ok {
			sdkHandler = decorator.Execute
		} else {
			invariant.Invariant(false, "invalid handler type for %s", cmd.Name)
		}
	}

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

// executeTreeWithStdout executes a tree node with stdout redirected to a custom writer.
// This is used by redirect and pipe operators to wire stdout between commands.
// Supports all tree node types: CommandNode, PipelineNode, AndNode, OrNode, SequenceNode.
func (e *executor) executeTreeWithStdout(tree sdk.TreeNode, stdout io.Writer) int {
	switch n := tree.(type) {
	case *sdk.CommandNode:
		// Simple command - redirect stdout directly
		return e.executeCommandWithPipes(n, nil, stdout)

	case *sdk.PipelineNode:
		// Pipeline - redirect final command's stdout
		return e.executePipelineWithStdout(n, stdout)

	case *sdk.AndNode:
		// AND operator: execute left, then right only if left succeeded
		// Both sides have stdout redirected (bash subshell semantics)
		leftExit := e.executeTreeWithStdout(n.Left, stdout)
		if leftExit != 0 {
			return leftExit // Short-circuit on failure
		}
		return e.executeTreeWithStdout(n.Right, stdout)

	case *sdk.OrNode:
		// OR operator: execute left, then right only if left failed
		// Both sides have stdout redirected (bash subshell semantics)
		leftExit := e.executeTreeWithStdout(n.Left, stdout)
		if leftExit == 0 {
			return leftExit // Short-circuit on success
		}
		return e.executeTreeWithStdout(n.Right, stdout)

	case *sdk.SequenceNode:
		// Sequence operator: execute all nodes with stdout redirected
		// All commands write to the same sink (bash subshell semantics)
		var lastExit int
		for _, node := range n.Nodes {
			lastExit = e.executeTreeWithStdout(node, stdout)
		}
		return lastExit

	case *sdk.RedirectNode:
		// Nested redirect - not supported (would need to chain sinks)
		fmt.Fprintf(os.Stderr, "Error: nested redirects not supported\n")
		return 127

	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported tree node type for redirect: %T\n", tree)
		return 127
	}
}

// executeTreeWithStdinStdout executes a tree node with both stdin and stdout redirected
// This is used for redirects inside pipelines where the source needs piped stdin
func (e *executor) executeTreeWithStdinStdout(tree sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	switch n := tree.(type) {
	case *sdk.CommandNode:
		// Simple command - redirect both stdin and stdout
		return e.executeCommandWithPipes(n, stdin, stdout)

	case *sdk.PipelineNode:
		// Pipeline - first command gets stdin, last command's stdout is redirected
		// This is complex - for now, just execute with stdout redirect
		// TODO: Handle stdin properly for pipelines
		return e.executePipelineWithStdout(n, stdout)

	case *sdk.AndNode:
		// AND operator: only right side gets stdin (bash semantics)
		leftExit := e.executeTreeWithStdout(n.Left, stdout)
		if leftExit != 0 {
			return leftExit
		}
		return e.executeTreeWithStdinStdout(n.Right, stdin, stdout)

	case *sdk.OrNode:
		// OR operator: only right side gets stdin (bash semantics)
		leftExit := e.executeTreeWithStdout(n.Left, stdout)
		if leftExit == 0 {
			return leftExit
		}
		return e.executeTreeWithStdinStdout(n.Right, stdin, stdout)

	case *sdk.SequenceNode:
		// Sequence: only last command gets stdin (bash semantics)
		var lastExit int
		for i, node := range n.Nodes {
			if i == len(n.Nodes)-1 {
				lastExit = e.executeTreeWithStdinStdout(node, stdin, stdout)
			} else {
				lastExit = e.executeTreeWithStdout(node, stdout)
			}
		}
		return lastExit

	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported tree node type: %T\n", tree)
		return 127
	}
}

// executePipelineWithStdout executes a pipeline with the final command's stdout redirected.
// This is similar to executePipeline but allows overriding the final stdout.
func (e *executor) executePipelineWithStdout(pipeline *sdk.PipelineNode, finalStdout io.Writer) int {
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")

	// Single command - redirect stdout directly
	if numCommands == 1 {
		return e.executeTreeNode(pipeline.Commands[0], nil, finalStdout)
	}

	// Create pipes between commands (N-1 pipes for N commands)
	pipes := make([]*io.PipeReader, numCommands-1)
	pipeWriters := make([]*io.PipeWriter, numCommands-1)
	for i := 0; i < numCommands-1; i++ {
		pipes[i], pipeWriters[i] = io.Pipe()
	}

	// Track exit codes for all commands
	exitCodes := make([]int, numCommands)

	// Execute all commands concurrently
	var wg sync.WaitGroup
	wg.Add(numCommands)

	// First command: no stdin, stdout to pipe[0]
	go func() {
		defer wg.Done()
		defer func() { _ = pipeWriters[0].Close() }()
		exitCodes[0] = e.executeTreeNode(pipeline.Commands[0], nil, pipeWriters[0])
	}()

	// Middle commands: stdin from pipe[i-1], stdout to pipe[i]
	for i := 1; i < numCommands-1; i++ {
		i := i // Capture loop variable
		go func() {
			defer wg.Done()
			defer func() { _ = pipeWriters[i].Close() }()
			exitCodes[i] = e.executeTreeNode(pipeline.Commands[i], pipes[i-1], pipeWriters[i])
		}()
	}

	// Last command: stdin from pipe[N-2], stdout to finalStdout
	go func() {
		defer wg.Done()
		exitCodes[numCommands-1] = e.executeTreeNode(pipeline.Commands[numCommands-1], pipes[numCommands-2], finalStdout)
	}()

	// Wait for all commands to complete
	wg.Wait()

	// Return exit code of last command (bash semantics)
	return exitCodes[numCommands-1]
}

// executeRedirect executes a redirect operation (> or >>)
// Opens the sink and redirects source's stdout to it
func (e *executor) executeRedirect(redirect *sdk.RedirectNode) int {
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")

	// Create execution context for opening the sink
	// This context provides the transport (local/SSH/Docker)
	ctx := newExecutionContext(make(map[string]interface{}), e, context.Background())

	// Check sink capabilities before opening
	caps := redirect.Sink.Caps()
	if redirect.Mode == sdk.RedirectOverwrite && !caps.Overwrite {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: sink %s (%s) does not support overwrite (>)\n", kind, path)
		return 1
	}
	if redirect.Mode == sdk.RedirectAppend && !caps.Append {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: sink %s (%s) does not support append (>>)\n", kind, path)
		return 1
	}

	// Open the sink for writing
	writer, err := redirect.Sink.Open(ctx, redirect.Mode, nil)
	if err != nil {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: failed to open sink %s (%s): %v\n", kind, path, err)
		return 1
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			kind, path := redirect.Sink.Identity()
			fmt.Fprintf(os.Stderr, "Error: failed to close sink %s (%s): %v\n", kind, path, closeErr)
		}
	}()

	// Execute source with stdout redirected to sink
	// We need to temporarily override stdout for the source execution
	return e.executeTreeWithStdout(redirect.Source, writer)
}

// executeRedirectWithStdin executes a redirect with piped stdin (for use in pipelines)
// This is like executeRedirect but also handles stdin from previous pipeline command
func (e *executor) executeRedirectWithStdin(redirect *sdk.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")

	// Create execution context for opening the sink
	ctx := newExecutionContext(make(map[string]interface{}), e, context.Background())

	// Check sink capabilities before opening
	caps := redirect.Sink.Caps()
	if redirect.Mode == sdk.RedirectOverwrite && !caps.Overwrite {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: sink %s (%s) does not support overwrite (>)\n", kind, path)
		return 1
	}
	if redirect.Mode == sdk.RedirectAppend && !caps.Append {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: sink %s (%s) does not support append (>>)\n", kind, path)
		return 1
	}

	// Open the sink for writing
	writer, err := redirect.Sink.Open(ctx, redirect.Mode, nil)
	if err != nil {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: failed to open sink %s (%s): %v\n", kind, path, err)
		return 1
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			kind, path := redirect.Sink.Identity()
			fmt.Fprintf(os.Stderr, "Error: failed to close sink %s (%s): %v\n", kind, path, closeErr)
		}
	}()

	// Execute source with stdin from pipe and stdout redirected to sink
	return e.executeTreeWithStdinStdout(redirect.Source, stdin, writer)
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
