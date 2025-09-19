package builtin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
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

// Plan generates an execution plan for the retry operation
func (r *RetryDecorator) Plan(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	attempts, delay, exponentialBackoff, err := r.extractParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@retry(<error: %v>)", err),
			Command:     "",
			Metadata: map[string]string{
				"decorator": "retry",
				"error":     err.Error(),
			},
		}
	}

	// Create timing info for the plan
	var desc string
	if exponentialBackoff {
		desc = fmt.Sprintf("@retry(attempts=%d, delay=%v, exponentialBackoff=true)", attempts, delay)
	} else {
		desc = fmt.Sprintf("@retry(attempts=%d, delay=%v)", attempts, delay)
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: desc,
		Command:     fmt.Sprintf("# Retry with %d attempts", attempts),
		Children:    []plan.ExecutionStep{}, // Will be populated by the plan generator
		Timing: &plan.TimingInfo{
			RetryAttempts: attempts,
			RetryDelay:    &delay,
		},
		Metadata: map[string]string{
			"decorator":          "retry",
			"attempts":           fmt.Sprintf("%d", attempts),
			"delay":              delay.String(),
			"exponentialBackoff": fmt.Sprintf("%t", exponentialBackoff),
			"execution_mode":     "error_handling",
			"color":              plan.ColorCyan,
		},
	}
}

// Execute performs the retry operation
func (r *RetryDecorator) Execute(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	// For now, return a placeholder result
	return &simpleCommandResult{
		stdout:   "",
		stderr:   "retry execution not implemented yet - use plan mode",
		exitCode: 1,
	}
}

// RequiresBlock returns the block requirements for @retry
func (r *RetryDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
