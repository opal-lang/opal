package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @timeout decorator on package import
func init() {
	decorators.RegisterBlock(NewTimeoutDecorator())
}

// TimeoutDecorator implements the @timeout decorator using the core decorator interfaces
type TimeoutDecorator struct{}

// NewTimeoutDecorator creates a new timeout decorator
func NewTimeoutDecorator() *TimeoutDecorator {
	return &TimeoutDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (t *TimeoutDecorator) Name() string {
	return "timeout"
}

// Description returns a human-readable description
func (t *TimeoutDecorator) Description() string {
	return "Execute commands with a time limit, cancelling execution on timeout"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TimeoutDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "duration",
			Type:        decorators.ArgTypeDuration, // Duration literal like 30s, 5m, 1h
			Required:    false,
			Description: "Maximum execution time (e.g., 30s, 5m, 1h), defaults to 30s",
		},
	}
}

// Examples returns usage examples
func (t *TimeoutDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `test: @timeout("5m") {
    go test -race ./...
}`,
			Description: "Run tests with 5 minute timeout",
		},
		{
			Code: `build: @timeout("30s") {
    npm run build
}`,
			Description: "Build with 30 second timeout",
		},
		{
			Code: `deploy: @timeout("10m") {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
}`,
			Description: "Deploy with 10 minute timeout for rollout",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (t *TimeoutDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"context", "time"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap executes the inner commands with timeout
func (t *TimeoutDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	duration, err := t.extractDuration(args)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@timeout parameter error: %v", err),
			ExitCode: 1,
		}
	}

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Channel to receive the result from inner execution
	resultChan := make(chan decorators.CommandResult, 1)

	// Execute inner commands in a goroutine
	go func() {
		defer func() {
			// Recover from panics to prevent the timeout goroutine from crashing
			if r := recover(); r != nil {
				resultChan <- decorators.CommandResult{
					Stderr:   fmt.Sprintf("panic during execution: %v", r),
					ExitCode: 1,
				}
			}
		}()

		result := ctx.ExecSequential(inner.Steps)
		resultChan <- result
	}()

	// Wait for either completion or timeout
	select {
	case result := <-resultChan:
		// Execution completed successfully
		return result

	case <-timeoutCtx.Done():
		// Timeout occurred
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("command timed out after %v", duration),
			ExitCode: 124, // Standard timeout exit code
		}
	}
}

// Describe returns description for dry-run display
func (t *TimeoutDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	duration, err := t.extractDuration(args)
	if err != nil {
		return plan.NewErrorStep("timeout", err)
	}

	// Create timeout step using core helpers
	step := plan.NewDecoratorStep("timeout", plan.StepDecorator)
	step.Description = fmt.Sprintf("@timeout(%v)", duration)

	// Add metadata for display formatting (matches test expectations)
	plan.AddMetadata(&step, "info", fmt.Sprintf("{%s timeout}", duration))
	plan.AddMetadata(&step, "duration", duration.String())
	plan.AddMetadata(&step, "seconds", fmt.Sprintf("%.0f", duration.Seconds()))

	// Add timing information
	step.Timing = &plan.TimingInfo{
		Timeout: &duration,
	}

	// Set children
	plan.SetChildren(&step, []plan.ExecutionStep{inner})

	return step
}

// ================================================================================================
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateBlockHint provides code generation hint for timeout execution

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractDuration extracts and validates the timeout duration
func (t *TimeoutDecorator) extractDuration(params []decorators.DecoratorParam) (time.Duration, error) {
	// Default timeout
	defaultDuration := 30 * time.Second

	if len(params) == 0 {
		return defaultDuration, nil
	}

	var duration time.Duration
	var err error

	// Extract duration parameter
	if params[0].Name == "" {
		// Positional parameter
		if val, ok := params[0].Value.(time.Duration); ok {
			duration = val
		} else if val, ok := params[0].Value.(string); ok {
			// Fallback for string values
			duration, err = time.ParseDuration(val)
			if err != nil {
				return 0, fmt.Errorf("invalid duration format %q: %w (use format like '30s', '5m', '1h')", val, err)
			}
		} else {
			return 0, fmt.Errorf("@timeout duration must be a duration or string, got %T", params[0].Value)
		}
	} else if params[0].Name == "duration" {
		// Named parameter
		if val, ok := params[0].Value.(time.Duration); ok {
			duration = val
		} else if val, ok := params[0].Value.(string); ok {
			// Fallback for string values
			duration, err = time.ParseDuration(val)
			if err != nil {
				return 0, fmt.Errorf("invalid duration format %q: %w (use format like '30s', '5m', '1h')", val, err)
			}
		} else {
			return 0, fmt.Errorf("@timeout duration parameter must be a duration or string, got %T", params[0].Value)
		}
	} else {
		return 0, fmt.Errorf("@timeout unknown parameter: %s", params[0].Name)
	}

	// Use default if zero value
	if duration == 0 {
		return defaultDuration, nil
	}

	// Validate duration bounds
	if duration <= 0 {
		return 0, fmt.Errorf("timeout duration must be positive, got %v", duration)
	}

	// Prevent excessively long timeouts (more than 24 hours)
	maxDuration := 24 * time.Hour
	if duration > maxDuration {
		return 0, fmt.Errorf("timeout duration %v exceeds maximum allowed %v", duration, maxDuration)
	}

	return duration, nil
}
