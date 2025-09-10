package builtin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @retry decorator on package import
func init() {
	decorators.RegisterBlock(NewRetryDecorator())
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
func (r *RetryDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"time"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap executes the inner commands with retry logic
func (r *RetryDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	attempts, delay, exponentialBackoff, err := r.extractParameters(args)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@retry parameter error: %v", err),
			ExitCode: 1,
		}
	}

	var lastResult decorators.CommandResult
	currentDelay := delay

	for attempt := 1; attempt <= attempts; attempt++ {
		// Execute the inner commands
		result := ctx.ExecSequential(inner.Steps)

		// If successful, return immediately
		if result.Success() {
			if attempt > 1 && ctx.Debug {
				fmt.Fprintf(ctx.Stderr, "[DEBUG] @retry succeeded on attempt %d/%d\n", attempt, attempts)
			}
			return result
		}

		// Store the result (in case this is the last attempt)
		lastResult = result

		// If this was the last attempt, return the failure
		if attempt == attempts {
			if ctx.Debug {
				fmt.Fprintf(ctx.Stderr, "[DEBUG] @retry failed after %d attempts\n", attempts)
			}
			break
		}

		// Wait before retrying (except on the last attempt)
		if ctx.Debug {
			fmt.Fprintf(ctx.Stderr, "[DEBUG] @retry attempt %d/%d failed (exit code %d), retrying in %v\n",
				attempt, attempts, result.ExitCode, currentDelay)
		}

		time.Sleep(currentDelay)

		// Update delay for exponential backoff
		if exponentialBackoff {
			currentDelay *= 2
		}
	}

	return lastResult
}

// Describe returns description for dry-run display
func (r *RetryDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
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
func (r *RetryDecorator) extractParameters(params []decorators.DecoratorParam) (attempts int, delay time.Duration, exponentialBackoff bool, err error) {
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
