package builtin

import (
	"fmt"
	"time"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @timeout decorator on package import
func init() {
	decorator := NewTimeoutDecorator()
	// Register with legacy interface (Phase 4: remove this)
	decorators.RegisterBlock(decorator)
	// Register with new interface
	decorators.RegisterExecutionDecorator(decorator)
}

// TimeoutDecorator implements the @timeout decorator using the core decorator interfaces
type TimeoutDecorator struct{}

// TimeoutParams represents validated parameters for @timeout decorator
type TimeoutParams struct {
	Duration time.Duration `json:"duration"` // Maximum execution time (default: 30s)
}

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

// Describe generates a plan step showing timeout configuration
func (t *TimeoutDecorator) Describe(ctx decorators.Context, args []decorators.Param, inner plan.ExecutionStep) plan.ExecutionStep {
	duration, err := t.extractDuration(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        "error",
			Description: fmt.Sprintf("@timeout parameter error: %v", err),
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@timeout {%s timeout}", duration),
		Children:    []plan.ExecutionStep{inner},
		Timing: &plan.TimingInfo{
			Timeout: &duration,
		},
		Metadata: map[string]string{
			"decorator": "timeout",
			"type":      "block",
			"timeout":   duration.String(),
		},
	}
}

// ================================================================================================
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateBlockHint provides code generation hint for timeout execution

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractDuration extracts and validates the timeout duration
func (t *TimeoutDecorator) extractDuration(params []decorators.Param) (time.Duration, error) {
	// Default timeout
	defaultDuration := 30 * time.Second

	if len(params) == 0 {
		return defaultDuration, nil
	}

	var duration time.Duration
	var err error

	// Extract duration parameter
	switch params[0].GetName() {
	case "":
		// Positional parameter
		if val, ok := params[0].GetValue().(time.Duration); ok {
			duration = val
		} else if val, ok := params[0].GetValue().(string); ok {
			// Fallback for string values
			duration, err = time.ParseDuration(val)
			if err != nil {
				return 0, fmt.Errorf("invalid duration format %q: %w (use format like '30s', '5m', '1h')", val, err)
			}
		} else {
			return 0, fmt.Errorf("@timeout duration must be a duration or string, got %T", params[0].GetValue())
		}
	case "duration":
		// Named parameter
		if val, ok := params[0].GetValue().(time.Duration); ok {
			duration = val
		} else if val, ok := params[0].GetValue().(string); ok {
			// Fallback for string values
			duration, err = time.ParseDuration(val)
			if err != nil {
				return 0, fmt.Errorf("invalid duration format %q: %w (use format like '30s', '5m', '1h')", val, err)
			}
		} else {
			return 0, fmt.Errorf("@timeout duration parameter must be a duration or string, got %T", params[0].GetValue())
		}
	default:
		return 0, fmt.Errorf("@timeout unknown parameter: %s", params[0].GetName())
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

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (target interface)
// ================================================================================================

// Plan generates an execution plan for the timeout operation

// ================================================================================================
// NEW GENERIC INTERFACE METHODS (ExecutionDecorator[any])
// ================================================================================================

// Validate validates parameters and returns TimeoutParams
func (t *TimeoutDecorator) Validate(args []decorators.Param) (any, error) {
	// Extract duration (first positional parameter or named "duration")
	duration, err := decorators.ExtractDuration(args, "duration", 30*time.Second) // default 30s
	if err != nil {
		return nil, fmt.Errorf("@timeout duration parameter error: %w", err)
	}

	if duration <= 0 {
		return nil, fmt.Errorf("@timeout duration must be positive")
	}

	return TimeoutParams{
		Duration: duration,
	}, nil
}

// Plan generates an execution plan using validated parameters
func (t *TimeoutDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(TimeoutParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: "@timeout(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "timeout",
				"error":     "invalid_params",
			},
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@timeout(%v) { ... }", params.Duration),
		Command:     "",
		Metadata: map[string]string{
			"decorator": "timeout",
			"duration":  params.Duration.String(),
			"status":    "awaiting_executable_block_implementation",
		},
	}
}

// Execute performs the actual timeout logic using validated parameters
func (t *TimeoutDecorator) Execute(ctx decorators.Context, validated any) (decorators.CommandResult, error) {
	_, ok := validated.(TimeoutParams)
	if !ok {
		return nil, fmt.Errorf("@timeout: invalid parameters")
	}

	// TODO: When ExecutableBlock is implemented, this will become:
	// ctx, cancel := context.WithTimeout(ctx, params.Duration)
	// defer cancel()
	// for _, stmt := range params.Block {
	//     result, err := stmt.Execute(ctx)
	//     if err != nil {
	//         if errors.Is(err, context.DeadlineExceeded) {
	//             return nil, fmt.Errorf("@timeout: execution exceeded %v", params.Duration)
	//         }
	//         return result, err
	//     }
	// }

	return nil, fmt.Errorf("@timeout: ExecutableBlock not yet implemented - use legacy interface for now")
}

// RequiresBlock returns the block requirements for @timeout
func (t *TimeoutDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
