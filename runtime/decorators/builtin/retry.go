package builtin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/core/plan"
	"github.com/aledsdavies/opal/runtime/execution/context"
)

// simpleCommandResult implements CommandResult for error cases
type simpleCommandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func (r *simpleCommandResult) GetStdout() string { return r.stdout }
func (r *simpleCommandResult) GetStderr() string { return r.stderr }
func (r *simpleCommandResult) GetExitCode() int  { return r.exitCode }
func (r *simpleCommandResult) IsSuccess() bool   { return r.exitCode == 0 }

// Register the @retry decorator on package import
func init() {
	decorator := NewRetryDecorator()
	// Register with legacy interface (Phase 4: remove this)
	decorators.RegisterBlock(decorator)
	// Register with new interface
	decorators.RegisterExecutionDecorator(decorator)
}

// RetryDecorator implements the @retry decorator using the core decorator interfaces
type RetryDecorator struct{}

// RetryParams represents validated parameters for @retry decorator
type RetryParams struct {
	Attempts int           `json:"attempts"` // Maximum number of attempts (default: 3)
	Delay    time.Duration `json:"delay"`    // Delay between attempts (default: 1s)
	// TODO: Add ExecutableBlock when DAG resolution is implemented
	// Block    decorators.ExecutableBlock `json:"block"` // Commands to retry
}

// NewRetryDecorator creates a new retry decorator
func NewRetryDecorator() *RetryDecorator {
	return &RetryDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (r *RetryDecorator) Name() string {
	return "retry"
}

// Description returns a human-readable description
func (r *RetryDecorator) Description() string {
	return "Retry command execution on failure with configurable attempts and delay"
}

// ParameterSchema returns the expected parameters for this decorator
func (r *RetryDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "attempts",
			Type:        decorators.ArgTypeInt,
			Required:    false,
			Description: "Maximum number of attempts (default: 3)",
		},
		{
			Name:        "delay",
			Type:        decorators.ArgTypeDuration,
			Required:    false,
			Description: "Delay between attempts (e.g., '1s', '5s', '30s', default: 1s)",
		},
		{
			Name:        "exponentialBackoff",
			Type:        decorators.ArgTypeBool,
			Required:    false,
			Description: "Use exponential backoff for delays (default: false)",
		},
	}
}

// Examples returns usage examples
func (r *RetryDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `deploy: @retry(attempts=3, delay="5s") {
    kubectl apply -f k8s/
}`,
			Description: "Retry deployment up to 3 times with 5 second delays",
		},
		{
			Code: `test: @retry(attempts=5, exponentialBackoff=true) {
    npm test
}`,
			Description: "Retry tests with exponential backoff",
		},
		{
			Code: `download: @retry {
    curl -f https://example.com/file.zip
}`,
			Description: "Retry download with default settings (3 attempts, 1s delay)",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
// Note: ImportRequirements removed - will be added back when code generation is implemented

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap executes the inner commands with retry logic
func (r *RetryDecorator) WrapCommands(ctx decorators.Context, args []decorators.Param, inner interface{}) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "runtime execution not implemented yet - use plan mode",
		ExitCode: 1,
	}
}

// Describe returns description for dry-run display
func (r *RetryDecorator) Describe(ctx decorators.Context, args []decorators.Param, inner plan.ExecutionStep) plan.ExecutionStep {
	attempts, delay, exponentialBackoff, err := r.extractParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@retry(<error: %v>)", err),
			Command:     "",
		}
	}

	return plan.ExecutionStep{
		Type:     plan.StepDecorator,
		Children: []plan.ExecutionStep{inner},
		Timing: &plan.TimingInfo{
			RetryAttempts: attempts,
			RetryDelay:    &delay,
		},
		Metadata: map[string]string{
			"decorator":          "retry",
			"info":               fmt.Sprintf("{%d attempts, %s delay}", attempts, delay),
			"attempts":           fmt.Sprintf("%d", attempts),
			"delay":              delay.String(),
			"exponentialBackoff": fmt.Sprintf("%t", exponentialBackoff),
		},
	}
}

// ================================================================================================
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateBlockHint provides code generation hint for retry execution

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractParameters extracts and validates retry parameters
func (r *RetryDecorator) extractParameters(params []decorators.Param) (attempts int, delay time.Duration, exponentialBackoff bool, err error) {
	// Extract attempts (first positional parameter or named "attempts")
	attempts, err = decorators.ExtractInt(params, "attempts", 3)
	if err != nil {
		// Try positional parameter
		attemptsStr, posErr := decorators.ExtractPositionalString(params, 0, "3")
		if posErr == nil {
			if parsed, parseErr := strconv.Atoi(attemptsStr); parseErr == nil {
				attempts = parsed
			} else {
				return 0, 0, false, fmt.Errorf("@retry first parameter must be a number, got %q", attemptsStr)
			}
		} else {
			return 0, 0, false, fmt.Errorf("@retry attempts parameter error: %v", err)
		}
	}

	// Extract optional parameters with defaults
	delayStr, err := decorators.ExtractString(params, "delay", "1s")
	if err != nil {
		return 0, 0, false, fmt.Errorf("@retry delay parameter error: %v", err)
	}
	delay, err = time.ParseDuration(delayStr)
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid delay format %q: %w (use format like '1s', '5s', '30s')", delayStr, err)
	}

	exponentialBackoff, err = decorators.ExtractBool(params, "exponentialBackoff", false)
	if err != nil {
		return 0, 0, false, fmt.Errorf("@retry exponentialBackoff parameter error: %v", err)
	}

	// Validate parameters
	if attempts <= 0 {
		return 0, 0, false, fmt.Errorf("@retry attempts must be positive, got %d", attempts)
	}
	if attempts > 20 {
		return 0, 0, false, fmt.Errorf("@retry attempts %d exceeds maximum allowed 20", attempts)
	}
	if delay < 0 {
		return 0, 0, false, fmt.Errorf("@retry delay must be non-negative, got %v", delay)
	}
	if delay > 10*time.Minute {
		return 0, 0, false, fmt.Errorf("@retry delay %v exceeds maximum allowed 10m", delay)
	}

	return attempts, delay, exponentialBackoff, nil
}

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (target interface)
// ================================================================================================

// ================================================================================================
// NEW GENERIC INTERFACE METHODS (ExecutionDecorator[any])
// ================================================================================================

// Validate validates parameters and returns RetryParams
func (r *RetryDecorator) Validate(args []decorators.Param) (any, error) {
	// Extract attempts (first positional parameter or named "attempts")
	attempts, err := decorators.ExtractInt(args, "attempts", 3) // default 3 attempts
	if err != nil {
		return nil, fmt.Errorf("@retry attempts parameter error: %w", err)
	}

	// Extract delay (second positional parameter or named "delay")
	delay, err := decorators.ExtractDuration(args, "delay", 1*time.Second) // default 1s delay
	if err != nil {
		return nil, fmt.Errorf("@retry delay parameter error: %w", err)
	}

	if attempts < 1 {
		return nil, fmt.Errorf("@retry attempts must be at least 1")
	}

	return RetryParams{
		Attempts: attempts,
		Delay:    delay,
	}, nil
}

// Plan generates an execution plan using validated parameters
func (r *RetryDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(RetryParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: "@retry(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "retry",
				"error":     "invalid_params",
			},
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@retry(attempts=%d, delay=%v) { ... }", params.Attempts, params.Delay),
		Command:     "",
		Metadata: map[string]string{
			"decorator": "retry",
			"attempts":  fmt.Sprintf("%d", params.Attempts),
			"delay":     params.Delay.String(),
			"status":    "awaiting_executable_block_implementation",
		},
	}
}

// Execute performs the actual retry logic using validated parameters
func (r *RetryDecorator) Execute(ctx decorators.Context, validated any) (decorators.CommandResult, error) {
	_, ok := validated.(RetryParams)
	if !ok {
		return nil, fmt.Errorf("@retry: invalid parameters")
	}

	// TODO: When ExecutableBlock is implemented, this will become:
	// for attempt := 1; attempt <= params.Attempts; attempt++ {
	//     for _, stmt := range params.Block {
	//         result, err := stmt.Execute(ctx)
	//         if err != nil && attempt < params.Attempts {
	//             time.Sleep(params.Delay)
	//             break // retry the block
	//         }
	//     }
	// }

	return nil, fmt.Errorf("@retry: ExecutableBlock not yet implemented - use legacy interface for now")
}

// RequiresBlock returns the block requirements for @retry
func (r *RetryDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
