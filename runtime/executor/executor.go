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
)

// DisplayIDResolver resolves display IDs in an execution transport context.
type DisplayIDResolver interface {
	ResolveDisplayIDWithTransport(displayID, currentTransportID string) (any, error)
}

// Config configures the executor
type Config struct {
	Debug          DebugLevel     // Debug tracing (development only)
	Telemetry      TelemetryLevel // Telemetry collection (production-safe)
	sessionFactory sessionFactory
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
	config   Config
	vault    DisplayIDResolver // For DisplayID resolution (nil if no secrets)
	sessions *sessionRuntime

	// Execution state
	stepsRun int
	exitCode int

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
func Execute(ctx context.Context, steps []sdk.Step, config Config, vlt DisplayIDResolver) (*ExecutionResult, error) {
	// INPUT CONTRACT (preconditions)
	invariant.NotNil(ctx, "ctx")
	invariant.NotNil(steps, "steps")

	e := &executor{
		config:    config,
		vault:     vlt,
		sessions:  newSessionRuntime(config.sessionFactory),
		startTime: time.Now(),
	}
	defer e.sessions.Close()

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
// Transport context is explicit per command via TransportID. DisplayID resolution
// uses that TransportID for boundary checks. Site-based authorization is handled
// by contract verification (plan hash), not runtime checks.
func (e *executor) executeStep(execCtx sdk.ExecutionContext, step sdk.Step) int {
	// INPUT CONTRACT
	invariant.NotNil(execCtx, "execCtx")
	invariant.Precondition(step.Tree != nil, "step must have a tree")

	var stdin io.Reader
	var stdout io.Writer
	if ec, ok := execCtx.(*executionContext); ok {
		stdin = ec.stdin
		stdout = ec.stdoutPipe
	}

	return e.executeTreeIO(execCtx, step.Tree, stdin, stdout)
}

// executeTreeIO executes a tree node with optional stdin/stdout overrides.
// stdin is only applied where input is meaningful (commands, pipelines, right side of &&/||,
// and last element of sequence), matching shell behavior.
func (e *executor) executeTreeIO(execCtx sdk.ExecutionContext, node sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommandWithPipes(execCtx, n, stdin, stdout)

	case *sdk.PipelineNode:
		return e.executePipelineIO(execCtx, n, stdin, stdout)

	case *sdk.AndNode:
		leftExit := e.executeTreeIO(execCtx, n.Left, nil, stdout)
		if leftExit != 0 {
			return leftExit
		}
		return e.executeTreeIO(execCtx, n.Right, stdin, stdout)

	case *sdk.OrNode:
		leftExit := e.executeTreeIO(execCtx, n.Left, nil, stdout)
		if leftExit == 0 {
			return leftExit
		}
		return e.executeTreeIO(execCtx, n.Right, stdin, stdout)

	case *sdk.SequenceNode:
		var lastExit int
		for i, child := range n.Nodes {
			childStdin := io.Reader(nil)
			if i == len(n.Nodes)-1 {
				childStdin = stdin
			}
			lastExit = e.executeTreeIO(execCtx, child, childStdin, stdout)
		}
		return lastExit

	case *sdk.RedirectNode:
		if stdin != nil {
			return e.executeRedirectWithStdin(execCtx, n, stdin)
		}
		return e.executeRedirect(execCtx, n)

	default:
		invariant.Invariant(false, "unknown TreeNode type: %T", node)
		return 1 // Unreachable
	}
}

// executePipelineIO executes a pipeline with optional stdin for the first command
// and optional stdout override for the last command.
func (e *executor) executePipelineIO(execCtx sdk.ExecutionContext, pipeline *sdk.PipelineNode, initialStdin io.Reader, finalStdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")

	// Single command - no piping needed
	if numCommands == 1 {
		return e.executeTreeNode(execCtx, pipeline.Commands[0], initialStdin, finalStdout)
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
			if cmdIndex == 0 {
				stdin = initialStdin
			} else {
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
				if finalStdout != nil {
					stdout = finalStdout
				} else {
					// Last command writes to terminal (scrubbed by CLI)
					stdout = os.Stdout
				}
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

// executeCommandWithPipes executes a command with optional piped stdin/stdout
// execCtx: execution context with environment, workdir, and cancellation
// stdin: piped input (nil if not piped)
// stdout: piped output (nil if not piped)
func (e *executor) executeCommandWithPipes(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	commandExecCtx := withExecutionTransport(execCtx, cmd.TransportID)

	// Strip @ prefix from decorator name for registry lookup
	decoratorName := strings.TrimPrefix(cmd.Name, "@")

	// Lookup decorator in registry
	entry, exists := decorator.Global().Lookup(decoratorName)
	invariant.Invariant(exists, "unknown decorator: %s", cmd.Name)

	// Check if it's an Exec decorator
	execDec, ok := entry.Impl.(decorator.Exec)
	invariant.Invariant(ok, "%s is not an execution decorator", cmd.Name)

	return e.executeDecorator(commandExecCtx, cmd, execDec, stdin, stdout)
}

func withExecutionTransport(execCtx sdk.ExecutionContext, transportID string) sdk.ExecutionContext {
	ec, ok := execCtx.(*executionContext)
	if !ok {
		return execCtx
	}

	return ec.withTransportID(normalizedTransportID(transportID))
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
func (e *executor) resolveDisplayIDs(params map[string]any, decoratorName, transportID string) (map[string]any, error) {
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
			actualValue, err := e.vault.ResolveDisplayIDWithTransport(displayID, normalizedTransportID(transportID))
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

// executeDecorator executes a decorator via the Exec interface.
// Converts ExecutionContext to decorator ExecContext and resolves DisplayIDs before execution.
func (e *executor) executeDecorator(
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
		params, err = e.resolveDisplayIDs(params, cmd.Name, cmd.TransportID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving secrets: %v\n", err)
			return 1
		}
	}

	var next decorator.ExecNode
	if len(cmd.Block) > 0 {
		next = &blockNode{execCtx: execCtx, steps: cmd.Block}
	}

	// Create execution node
	node := execDec.Wrap(next, params)
	if node == nil {
		if next == nil {
			return 0
		}
		node = next
	}

	baseSession, sessionErr := e.sessions.SessionFor(cmd.TransportID)
	if sessionErr != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", sessionErr)
		return 1
	}
	session := sessionForExecutionContext(baseSession, execCtx)

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

func sessionForExecutionContext(base decorator.Session, execCtx sdk.ExecutionContext) decorator.Session {
	session := base

	if workdir := execCtx.Workdir(); workdir != "" && workdir != session.Cwd() {
		session = session.WithWorkdir(workdir)
	}

	if delta := envDelta(session.Env(), execCtx.Environ()); len(delta) > 0 {
		session = session.WithEnv(delta)
	}

	return session
}

func envDelta(base, target map[string]string) map[string]string {
	delta := make(map[string]string)
	for key, targetValue := range target {
		if baseValue, ok := base[key]; !ok || baseValue != targetValue {
			delta[key] = targetValue
		}
	}
	return delta
}

// executeRedirect executes a redirect operation (> or >>)
// Opens the sink and redirects source's stdout to it
func (e *executor) executeRedirect(execCtx sdk.ExecutionContext, redirect *sdk.RedirectNode) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")
	redirectExecCtx := withExecutionTransport(execCtx, sourceTransportID(redirect.Source))

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
	writer, err := redirect.Sink.Open(redirectExecCtx, redirect.Mode, nil)
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
	return e.executeTreeIO(redirectExecCtx, redirect.Source, nil, writer)
}

// executeRedirectWithStdin executes a redirect with piped stdin (for use in pipelines)
// This is like executeRedirect but also handles stdin from previous pipeline command
func (e *executor) executeRedirectWithStdin(execCtx sdk.ExecutionContext, redirect *sdk.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")
	redirectExecCtx := withExecutionTransport(execCtx, sourceTransportID(redirect.Source))

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
	writer, err := redirect.Sink.Open(redirectExecCtx, redirect.Mode, nil)
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
	return e.executeTreeIO(redirectExecCtx, redirect.Source, stdin, writer)
}

func sourceTransportID(node sdk.TreeNode) string {
	if node == nil {
		return "local"
	}

	switch n := node.(type) {
	case *sdk.CommandNode:
		return normalizedTransportID(n.TransportID)
	case *sdk.PipelineNode:
		if len(n.Commands) == 0 {
			return "local"
		}
		return sourceTransportID(n.Commands[0])
	case *sdk.RedirectNode:
		return sourceTransportID(n.Source)
	case *sdk.AndNode:
		return sourceTransportID(n.Left)
	case *sdk.OrNode:
		return sourceTransportID(n.Left)
	case *sdk.SequenceNode:
		if len(n.Nodes) == 0 {
			return "local"
		}
		return sourceTransportID(n.Nodes[0])
	default:
		return "local"
	}
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
