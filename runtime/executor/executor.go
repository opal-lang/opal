package executor

import (
	"bytes"
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
	Debug              DebugLevel     // Debug tracing (development only)
	Telemetry          TelemetryLevel // Telemetry collection (production-safe)
	LockdownStdStreams bool           // Lock down stdout/stderr (security)
	Scrubber           io.Writer      // Output writer with secret scrubbing (optional)
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
	Output      string              // Captured output (scrubbed)
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

// SecretScrubber wraps an io.Writer and redacts registered secrets
type SecretScrubber struct {
	writer  io.Writer
	secrets map[string]string // secret value -> placeholder
	mu      sync.RWMutex
}

// NewSecretScrubber creates a new secret scrubber
func NewSecretScrubber(w io.Writer) *SecretScrubber {
	invariant.NotNil(w, "writer")
	return &SecretScrubber{
		writer:  w,
		secrets: make(map[string]string),
	}
}

// RegisterSecret marks a value for redaction with its placeholder
func (s *SecretScrubber) RegisterSecret(value, placeholder string) {
	invariant.Precondition(value != "", "secret value cannot be empty")
	invariant.Precondition(placeholder != "", "placeholder cannot be empty")

	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[value] = placeholder
}

// Write implements io.Writer, redacting secrets before output
func (s *SecretScrubber) Write(p []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	output := string(p)
	for secret, placeholder := range s.secrets {
		output = strings.ReplaceAll(output, secret, placeholder)
	}

	n, err := s.writer.Write([]byte(output))
	// Return original length to satisfy io.Writer contract
	if err == nil {
		return len(p), nil
	}
	return n, err
}

// LockdownConfig configures stdout/stderr lockdown
type LockdownConfig struct {
	Scrubber io.Writer // Writer to redirect output to (must not be nil)
}

// LockDownStdStreams redirects stdout/stderr to a controlled writer
// Returns a restore function that must be called to restore original streams
func LockDownStdStreams(config *LockdownConfig) (restore func()) {
	// INPUT CONTRACT
	invariant.NotNil(config, "config")
	invariant.NotNil(config.Scrubber, "config.Scrubber")

	// Save original streams
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create pipes that redirect to scrubber
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	// Copy from pipes to scrubber in background
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(config.Scrubber, rOut)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(config.Scrubber, rErr)
	}()

	// Return restore function
	return func() {
		// Close write ends to signal EOF to copy goroutines
		_ = wOut.Close()
		_ = wErr.Close()

		// Wait for copy goroutines to finish
		wg.Wait()

		// Close read ends
		_ = rOut.Close()
		_ = rErr.Close()

		// Restore original streams
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}
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
// Secret scrubbing is handled by the CLI before calling Execute.
func Execute(steps []sdk.Step, config Config) (*ExecutionResult, error) {
	// INPUT CONTRACT (preconditions)
	invariant.NotNil(steps, "steps")

	// Set up stdout/stderr lockdown if enabled
	var outputBuf bytes.Buffer
	var restore func()
	var needsRestore bool

	if config.LockdownStdStreams {
		// Create scrubber
		var scrubber *SecretScrubber
		if config.Scrubber != nil {
			// Use provided scrubber
			scrubber = NewSecretScrubber(config.Scrubber)
		} else {
			// Default: capture to buffer
			scrubber = NewSecretScrubber(&outputBuf)
		}

		// Note: Secret registration is handled by CLI before calling Execute
		// The executor has no knowledge of secrets - that's a CLI concern

		// Lock down stdout/stderr
		restore = LockDownStdStreams(&LockdownConfig{
			Scrubber: scrubber,
		})
		needsRestore = true
		// Note: We call restore() explicitly before reading outputBuf to avoid race
	}

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

	// Restore stdout/stderr BEFORE reading outputBuf to avoid race condition
	// The goroutines in LockDownStdStreams must finish writing before we read
	if needsRestore {
		restore()
	}

	// OUTPUT CONTRACT (postconditions)
	invariant.InRange(e.exitCode, 0, 255, "exit code")
	invariant.Postcondition(e.stepsRun >= 0, "steps run must be non-negative")
	invariant.Postcondition(e.stepsRun <= len(steps), "steps run cannot exceed total steps")

	return &ExecutionResult{
		ExitCode:    e.exitCode,
		Duration:    duration,
		StepsRun:    e.stepsRun,
		Output:      outputBuf.String(), // Captured and scrubbed output (safe after restore())
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
