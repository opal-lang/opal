package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
	"github.com/opal-lang/opal/core/types"
	"github.com/opal-lang/opal/runtime/vault"
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
	vault  *vault.Vault // For DisplayID resolution (nil if no secrets)

	// Execution state
	stepsRun         int
	exitCode         int
	currentTransport string // Current transport context (e.g., "local", "transport:abc123")

	// Observability
	debugEvents []DebugEvent
	telemetry   *ExecutionTelemetry
	startTime   time.Time
}

// Execute runs SDK steps and returns the result.
//
// The executor resolves DisplayIDs to actual values during execution when vault is provided.
// Without vault, commands execute as-is (useful for testing or non-secret workflows).
//
// Context enables cancellation and timeout propagation through the execution chain.
// The CLI handles secret scrubbing by redirecting stdout/stderr through the scrubber.
func Execute(ctx context.Context, steps []sdk.Step, config Config, vlt *vault.Vault) (*ExecutionResult, error) {
	// INPUT CONTRACT (preconditions)
	invariant.NotNil(ctx, "ctx")
	invariant.NotNil(steps, "steps")

	e := &executor{
		config:           config,
		vault:            vlt,
		currentTransport: "local", // Default to local transport
		startTime:        time.Now(),
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

	// Create root ExecutionContext with current environment and workdir
	// This is the entry point - all nested decorators will inherit from this
	rootExecCtx := newExecutionContext(make(map[string]interface{}), e, ctx)

	// Execute all steps sequentially
	for _, step := range steps {
		stepStart := time.Now()

		if config.Debug >= DebugDetailed {
			e.recordDebugEvent("step_start", step.ID, "executing tree")
		}

		exitCode := e.executeStep(rootExecCtx, step)
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
	// Exit code must be valid: -1 (canceled), 0 (success), or 1-255 (failure)
	invariant.Postcondition(
		e.exitCode == decorator.ExitCanceled || (e.exitCode >= 0 && e.exitCode <= 255),
		"exit code must be -1 (canceled) or in range [0, 255], got %d", e.exitCode)
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

// executeStep executes a single step by executing its tree.
//
// Transport context: The executor tracks the current transport (e.g., "local", "transport:abc123")
// and passes it to Vault for transport boundary checks. Site-based authorization is handled
// by contract verification (plan hash), not runtime checks.
func (e *executor) executeStep(execCtx sdk.ExecutionContext, step sdk.Step) int {
	// INPUT CONTRACT
	invariant.NotNil(execCtx, "execCtx")
	invariant.Precondition(step.Tree != nil, "step must have a tree")

	return e.executeTree(execCtx, step.Tree)
}

// executeTree executes a tree node and returns the exit code
func (e *executor) executeTree(execCtx sdk.ExecutionContext, node sdk.TreeNode) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommand(execCtx, n)

	case *sdk.PipelineNode:
		return e.executePipeline(execCtx, n)

	case *sdk.AndNode:
		// Execute left, then right only if left succeeded
		leftExit := e.executeTree(execCtx, n.Left)
		if leftExit != 0 {
			return leftExit // Short-circuit on failure
		}
		return e.executeTree(execCtx, n.Right)

	case *sdk.OrNode:
		// Execute left, then right only if left failed
		leftExit := e.executeTree(execCtx, n.Left)
		if leftExit == 0 {
			return leftExit // Short-circuit on success
		}
		return e.executeTree(execCtx, n.Right)

	case *sdk.SequenceNode:
		// Execute all nodes, return last exit code
		var lastExit int
		for _, child := range n.Nodes {
			lastExit = e.executeTree(execCtx, child)
		}
		return lastExit

	case *sdk.RedirectNode:
		return e.executeRedirect(execCtx, n)

	default:
		invariant.Invariant(false, "unknown TreeNode type: %T", node)
		return 1 // Unreachable
	}
}

// executePipeline executes a pipeline of commands with stdout→stdin streaming
// Uses io.Pipe() for streaming (bash-compatible: concurrent execution, not buffered)
// Returns exit code of last command (bash semantics)
func (e *executor) executePipeline(execCtx sdk.ExecutionContext, pipeline *sdk.PipelineNode) int {
	invariant.NotNil(execCtx, "execCtx")
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")

	// Single command - no piping needed
	if numCommands == 1 {
		return e.executeTreeNode(execCtx, pipeline.Commands[0], nil, nil)
	}

	// Create OS pipes between commands (kernel handles SIGPIPE)
	// For N commands, we need N-1 pipes
	// Using os.Pipe() instead of io.Pipe() allows direct process-to-process pipes
	// without copy goroutines, ensuring proper EPIPE/SIGPIPE semantics
	pipeReaders := make([]*os.File, numCommands-1)
	pipeWriters := make([]*os.File, numCommands-1)
	for i := 0; i < numCommands-1; i++ {
		pr, pw, err := os.Pipe()
		if err != nil {
			// Clean up any pipes created so far
			for j := 0; j < i; j++ {
				_ = pipeReaders[j].Close()
				_ = pipeWriters[j].Close()
			}
			return 1 // Pipe creation failed
		}
		pipeReaders[i] = pr
		pipeWriters[i] = pw
	}

	// Track exit codes for all commands (PIPESTATUS)
	exitCodes := make([]int, numCommands)

	// Track pipe close operations to ensure each pipe is closed only once
	// Using sync.Once prevents "Bad file descriptor" errors from double-close
	pipeReaderCloseOnce := make([]sync.Once, numCommands-1)
	pipeWriterCloseOnce := make([]sync.Once, numCommands-1)

	// Execute all commands concurrently (bash behavior)
	var wg sync.WaitGroup
	wg.Add(numCommands)

	// CRITICAL: Context cancellation flows through to LocalSession.Run
	// which kills the entire process group. We don't need to close pipes
	// in a separate goroutine - the defer statements below handle cleanup
	// after processes exit. Closing pipes early creates a race condition
	// where pipes are closed while processes are still starting.

	for i := 0; i < numCommands; i++ {
		cmdIndex := i
		node := pipeline.Commands[i]

		go func() {
			defer wg.Done()

			// Determine stdin for this command
			var stdin io.Reader
			if cmdIndex > 0 {
				stdin = pipeReaders[cmdIndex-1]
				// Close the read end after this command completes
				// This ensures the pipe is fully closed when both ends are done
				// Use sync.Once to prevent double-close errors
				defer func() {
					idx := cmdIndex - 1
					pipeReaderCloseOnce[idx].Do(func() {
						_ = pipeReaders[idx].Close()
					})
				}()
			}

			// Determine stdout for this command
			var stdout io.Writer
			if cmdIndex < numCommands-1 {
				stdout = pipeWriters[cmdIndex]
				// CRITICAL: Close write end immediately after command completes
				// This sends EOF to downstream command, allowing it to exit
				// and triggering EPIPE for upstream if it's still writing
				// Use sync.Once to prevent double-close errors
				defer func() {
					idx := cmdIndex
					pipeWriterCloseOnce[idx].Do(func() {
						_ = pipeWriters[idx].Close()
					})
				}()
			} else {
				// Last command writes to terminal (scrubbed by CLI)
				stdout = os.Stdout
			}

			// Execute tree node (CommandNode or RedirectNode) with pipes
			exitCodes[cmdIndex] = e.executeTreeNode(execCtx, node, stdin, stdout)
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
func (e *executor) executeTreeNode(execCtx sdk.ExecutionContext, node sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommandWithPipes(execCtx, n, stdin, stdout)
	case *sdk.RedirectNode:
		// For redirect in pipeline, we need to handle it specially
		// The redirect's source gets the piped stdin, and its output goes to the sink
		// The piped stdout (if any) is ignored because redirect captures output
		// NOTE: We don't close the stdout pipe here - it will be closed by the
		// defer in the goroutine that created it. Closing it here would cause
		// "Bad file descriptor" errors if the upstream command is still writing.

		// Execute redirect with piped stdin
		return e.executeRedirectWithStdin(execCtx, n, stdin)
	default:
		invariant.Invariant(false, "invalid pipeline element type %T", node)
		return 1
	}
}

// executeCommand executes a single command node
func (e *executor) executeCommand(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode) int {
	invariant.NotNil(execCtx, "execCtx")
	return e.executeCommandWithPipes(execCtx, cmd, nil, nil)
}

// executeCommandWithPipes executes a command with optional piped stdin/stdout
// execCtx: execution context with environment, workdir, and cancellation
// stdin: piped input (nil if not piped)
// stdout: piped output (nil if not piped)
func (e *executor) executeCommandWithPipes(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	// Strip @ prefix from decorator name for registry lookup
	decoratorName := strings.TrimPrefix(cmd.Name, "@")

	// Try new decorator registry first
	if entry, exists := decorator.Global().Lookup(decoratorName); exists {
		// Check if it's an Exec decorator
		if execDec, ok := entry.Impl.(decorator.Exec); ok {
			return e.executeNewDecorator(execCtx, cmd, execDec, stdin, stdout)
		}
	}

	// Fall back to old SDK registry for decorators not yet migrated
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
		if execDecorator, ok := handler.(ExecutionDecorator); ok {
			sdkHandler = execDecorator.Execute
		} else {
			invariant.Invariant(false, "invalid handler type for %s", cmd.Name)
		}
	}

	// Clone execution context with command args and pipes if needed
	// This preserves parent context's environ/workdir while adding command-specific args
	var cmdExecCtx sdk.ExecutionContext
	if stdin != nil || stdout != nil {
		cmdExecCtx = execCtx.Clone(cmd.Args, stdin, stdout)
	} else {
		cmdExecCtx = execCtx.Clone(cmd.Args, nil, nil)
	}

	// Call handler with SDK block using cloned context
	exitCode, err := sdkHandler(cmdExecCtx, cmd.Block)
	if err != nil {
		// Log error but return exit code
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return exitCode
}

// resolveDisplayIDs scans params for DisplayID strings and resolves them to actual values.
//
// DisplayIDs are content-addressed placeholders (format: opal:<base64url-hash>) that appear
// in the plan instead of actual secret values. This enables:
//   - Plan contracts to be stored/transmitted without exposing secrets
//   - Contract verification without re-resolving secrets
//   - Deterministic hashing for contract stability
//
// During execution, we resolve DisplayIDs back to actual values just before passing
// params to decorators. The vault enforces transport boundary checks to prevent
// secrets from leaking across host boundaries. Site-based authorization is handled
// by contract verification (plan hash), not runtime checks.
func (e *executor) resolveDisplayIDs(params map[string]any, decoratorName string) (map[string]any, error) {
	displayIDPattern := regexp.MustCompile(`opal:[A-Za-z0-9_-]{22}`)
	resolved := make(map[string]any)

	for key, val := range params {
		strVal, ok := val.(string)
		if !ok {
			// Not a string, keep as-is
			resolved[key] = val
			continue
		}

		// Find all DisplayIDs in the string
		matches := displayIDPattern.FindAllString(strVal, -1)
		if len(matches) == 0 {
			// No DisplayIDs, keep as-is
			resolved[key] = val
			continue
		}

		// Resolve each DisplayID
		result := strVal
		for _, displayID := range matches {
			// Resolve DisplayID with transport boundary check
			actualValue, err := e.vault.ResolveDisplayIDWithTransport(displayID, e.currentTransport)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %s in %s.%s: %w", displayID, decoratorName, key, err)
			}

			// Replace DisplayID with actual value
			result = strings.ReplaceAll(result, displayID, fmt.Sprint(actualValue))
		}

		resolved[key] = result
	}

	return resolved, nil
}

// executeNewDecorator executes a decorator from the new registry.
// Converts ExecutionContext to decorator ExecContext and executes via Exec interface.
func (e *executor) executeNewDecorator(
	execCtx sdk.ExecutionContext,
	cmd *sdk.CommandNode,
	execDec decorator.Exec,
	stdin io.Reader,
	stdout io.Writer,
) int {
	// Convert SDK command args to decorator params
	params := make(map[string]any)
	for k, v := range cmd.Args {
		params[k] = v
	}

	// Resolve DisplayIDs to actual values if vault is available
	if e.vault != nil {
		var err error
		params, err = e.resolveDisplayIDs(params, cmd.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving secrets: %v\n", err)
			return 1
		}
	}

	// Create execution node
	node := execDec.Wrap(nil, params)

	// CRITICAL: Create session from ExecutionContext to respect decorator hierarchy
	// This ensures @env/@workdir decorators work correctly
	// DO NOT use os.Getwd()/os.Environ() - that discards parent context!
	session := decorator.NewLocalSession().
		WithWorkdir(execCtx.Workdir()).
		WithEnv(execCtx.Environ())
	defer func() {
		_ = session.Close() // Ignore close errors in defer
	}()

	// Default stdout to terminal if not provided
	// This ensures output is visible for non-piped commands
	if stdout == nil {
		stdout = os.Stdout
	}

	// Create ExecContext with parent context for cancellation
	decoratorExecCtx := decorator.ExecContext{
		Context: execCtx.Context(), // Extract Go context for cancellation/deadlines
		Session: session,
		Stdin:   stdin, // Pass io.Reader directly (was: io.ReadAll + []byte)
		Stdout:  stdout,
		Stderr:  os.Stderr, // NEW: Forward stderr to terminal
		Trace:   nil,
	}

	// Execute - the shellNode will pass ctx to Session.Run() for cancellation
	result, err := node.Execute(decoratorExecCtx)
	if err != nil {
		// Log error but return exit code (matches SDK behavior)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return result.ExitCode
}

// executeTreeWithStdout executes a tree node with stdout redirected to a custom writer.
// This is used by redirect and pipe operators to wire stdout between commands.
// Supports all tree node types: CommandNode, PipelineNode, AndNode, OrNode, SequenceNode.
func (e *executor) executeTreeWithStdout(execCtx sdk.ExecutionContext, tree sdk.TreeNode, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := tree.(type) {
	case *sdk.CommandNode:
		// Simple command - redirect stdout directly
		return e.executeCommandWithPipes(execCtx, n, nil, stdout)

	case *sdk.PipelineNode:
		// Pipeline - redirect final command's stdout
		return e.executePipelineWithStdout(execCtx, n, stdout)

	case *sdk.AndNode:
		// AND operator: execute left, then right only if left succeeded
		// Both sides have stdout redirected (bash subshell semantics)
		leftExit := e.executeTreeWithStdout(execCtx, n.Left, stdout)
		if leftExit != 0 {
			return leftExit // Short-circuit on failure
		}
		return e.executeTreeWithStdout(execCtx, n.Right, stdout)

	case *sdk.OrNode:
		// OR operator: execute left, then right only if left failed
		// Both sides have stdout redirected (bash subshell semantics)
		leftExit := e.executeTreeWithStdout(execCtx, n.Left, stdout)
		if leftExit == 0 {
			return leftExit // Short-circuit on success
		}
		return e.executeTreeWithStdout(execCtx, n.Right, stdout)

	case *sdk.SequenceNode:
		// Sequence operator: execute all nodes with stdout redirected
		// All commands write to the same sink (bash subshell semantics)
		var lastExit int
		for _, node := range n.Nodes {
			lastExit = e.executeTreeWithStdout(execCtx, node, stdout)
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
func (e *executor) executeTreeWithStdinStdout(execCtx sdk.ExecutionContext, tree sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := tree.(type) {
	case *sdk.CommandNode:
		// Simple command - redirect both stdin and stdout
		return e.executeCommandWithPipes(execCtx, n, stdin, stdout)

	case *sdk.PipelineNode:
		// Pipeline - first command gets stdin, last command's stdout is redirected
		// This is complex - for now, just execute with stdout redirect
		// TODO: Handle stdin properly for pipelines
		return e.executePipelineWithStdout(execCtx, n, stdout)

	case *sdk.AndNode:
		// AND operator: only right side gets stdin (bash semantics)
		leftExit := e.executeTreeWithStdout(execCtx, n.Left, stdout)
		if leftExit != 0 {
			return leftExit
		}
		return e.executeTreeWithStdinStdout(execCtx, n.Right, stdin, stdout)

	case *sdk.OrNode:
		// OR operator: only right side gets stdin (bash semantics)
		leftExit := e.executeTreeWithStdout(execCtx, n.Left, stdout)
		if leftExit == 0 {
			return leftExit
		}
		return e.executeTreeWithStdinStdout(execCtx, n.Right, stdin, stdout)

	case *sdk.SequenceNode:
		// Sequence: only last command gets stdin (bash semantics)
		var lastExit int
		for i, node := range n.Nodes {
			if i == len(n.Nodes)-1 {
				lastExit = e.executeTreeWithStdinStdout(execCtx, node, stdin, stdout)
			} else {
				lastExit = e.executeTreeWithStdout(execCtx, node, stdout)
			}
		}
		return lastExit

	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported tree node type: %T\n", tree)
		return 127
	}
}

// executePipelineWithStdout executes a pipeline with the final command's stdout redirected.
// This is used by redirect operators to capture pipeline output.
func (e *executor) executePipelineWithStdout(execCtx sdk.ExecutionContext, pipeline *sdk.PipelineNode, finalStdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")

	// Single command - redirect stdout directly
	if numCommands == 1 {
		return e.executeTreeNode(execCtx, pipeline.Commands[0], nil, finalStdout)
	}

	// CRITICAL: Use os.Pipe() instead of io.Pipe() for proper SIGPIPE semantics
	// This ensures (yes | head -n1) > out.txt completes quickly instead of hanging
	// The kernel sends SIGPIPE to writers when readers close, unlike io.Pipe()
	pipeReaders := make([]*os.File, numCommands-1)
	pipeWriters := make([]*os.File, numCommands-1)
	for i := 0; i < numCommands-1; i++ {
		pr, pw, err := os.Pipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating pipe: %v\n", err)
			return 1
		}
		pipeReaders[i] = pr
		pipeWriters[i] = pw
	}

	// Track exit codes for all commands
	exitCodes := make([]int, numCommands)

	// Track pipe close operations to ensure each pipe is closed only once
	// Using sync.Once prevents "Bad file descriptor" errors from double-close
	pipeReaderCloseOnce := make([]sync.Once, numCommands-1)
	pipeWriterCloseOnce := make([]sync.Once, numCommands-1)

	// Execute all commands concurrently
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
				stdin = pipeReaders[cmdIndex-1]
				// Close the read end after this command completes
				defer func() {
					idx := cmdIndex - 1
					pipeReaderCloseOnce[idx].Do(func() {
						_ = pipeReaders[idx].Close()
					})
				}()
			}

			// Determine stdout for this command
			var stdout io.Writer
			if cmdIndex < numCommands-1 {
				stdout = pipeWriters[cmdIndex]
				// CRITICAL: Close write end immediately after command completes
				// This sends EOF to downstream command and SIGPIPE to upstream
				defer func() {
					idx := cmdIndex
					pipeWriterCloseOnce[idx].Do(func() {
						_ = pipeWriters[idx].Close()
					})
				}()
			} else {
				// Last command writes to finalStdout (the redirect sink)
				stdout = finalStdout
			}

			// Execute tree node (CommandNode or RedirectNode) with pipes
			exitCodes[cmdIndex] = e.executeTreeNode(execCtx, node, stdin, stdout)
		}()
	}

	// Wait for all commands to complete
	wg.Wait()

	// Return exit code of last command (bash semantics)
	return exitCodes[numCommands-1]
}

// executeRedirect executes a redirect operation (> or >>)
// Opens the sink and redirects source's stdout to it
func (e *executor) executeRedirect(execCtx sdk.ExecutionContext, redirect *sdk.RedirectNode) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")

	// Use the provided execution context for opening the sink
	// This context provides the transport (local/SSH/Docker) and respects parent environ/workdir

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
	writer, err := redirect.Sink.Open(execCtx, redirect.Mode, nil)
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
	return e.executeTreeWithStdout(execCtx, redirect.Source, writer)
}

// executeRedirectWithStdin executes a redirect with piped stdin (for use in pipelines)
// This is like executeRedirect but also handles stdin from previous pipeline command
func (e *executor) executeRedirectWithStdin(execCtx sdk.ExecutionContext, redirect *sdk.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")

	// Use the provided execution context for opening the sink

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
	writer, err := redirect.Sink.Open(execCtx, redirect.Mode, nil)
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
	return e.executeTreeWithStdinStdout(execCtx, redirect.Source, stdin, writer)
}

// recordDebugEvent records a debug event (only if debug enabled)
func (e *executor) recordDebugEvent(event string, stepID uint64, contextInfo string) {
	if e.config.Debug == DebugOff {
		return
	}

	e.debugEvents = append(e.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		StepID:    stepID,
		Context:   contextInfo,
	})
}
