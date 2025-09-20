package builtin

import (
	"fmt"
	"strconv"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// Register the @parallel decorator on package import
func init() {
	decorator := NewParallelDecorator()
	decorators.RegisterBlock(decorator)
	decorators.RegisterExecutionDecorator(decorator)
}

// ParallelDecorator implements the @parallel decorator using the core decorator interfaces
type ParallelDecorator struct{}

// ParallelParams represents validated parameters for @parallel decorator
type ParallelParams struct {
	Mode        string `json:"mode"`        // Failure mode: 'fail-fast', 'immediate', or 'all'
	Concurrency int    `json:"concurrency"` // Maximum concurrent jobs
}

// NewParallelDecorator creates a new parallel decorator
func NewParallelDecorator() *ParallelDecorator {
	return &ParallelDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (p *ParallelDecorator) Name() string {
	return "parallel"
}

// Description returns a human-readable description
func (p *ParallelDecorator) Description() string {
	return "Execute commands concurrently with configurable failure modes and concurrency limits"
}

// ParameterSchema returns the expected parameters for this decorator
func (p *ParallelDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "mode",
			Type:        decorators.ArgTypeString,
			Required:    false,
			Description: "Failure mode: 'fail-fast' (default), 'immediate', or 'all'",
		},
		{
			Name:        "concurrency",
			Type:        decorators.ArgTypeInt,
			Required:    false,
			Description: "Maximum concurrent jobs (default: CPU cores * 2)",
		},
	}
}

// Examples returns usage examples
func (p *ParallelDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `build: @parallel {
    go build ./cmd/server
    go build ./cmd/cli
    go build ./cmd/worker
}`,
			Description: "Build multiple binaries in parallel (fail-fast mode)",
		},
		{
			Code: `test: @parallel(mode="all") {
    go test ./pkg/...
    npm test
    python -m pytest
}`,
			Description: "Run all tests regardless of failures",
		},
		{
			Code: `deploy: @parallel(concurrency=2) {
    kubectl apply -f service.yaml
    kubectl apply -f deployment.yaml
    kubectl apply -f ingress.yaml
}`,
			Description: "Deploy with limited concurrency",
		},
	}
}

// Note: ImportRequirements removed - will be added back when code generation is implemented

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// WrapCommands executes multiple steps in parallel using the documented API
func (p *ParallelDecorator) WrapCommands(ctx decorators.Context, args []decorators.Param, commands interface{}) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "runtime execution not implemented yet - use plan mode",
		ExitCode: 1,
	}
}

// Describe returns description for dry-run display
func (p *ParallelDecorator) Describe(ctx decorators.Context, args []decorators.Param, inner plan.ExecutionStep) plan.ExecutionStep {
	// Calculate default concurrency based on CPU cores
	defaultConcurrency := ctx.SystemInfo().GetNumCPU() * 2
	maxConcurrency := 50
	if defaultConcurrency > maxConcurrency {
		defaultConcurrency = maxConcurrency
	}

	mode, userConcurrency, err := p.extractParameters(args, defaultConcurrency)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@parallel(<error: %v>)", err),
			Command:     "",
		}
	}

	// Count the number of parallel tasks using the plan core helper
	taskCount := plan.CountParallelTasks(inner)

	// Calculate effective concurrency: min(user_setting, task_count, cpu_cores)
	cpuCores := ctx.SystemInfo().GetNumCPU()
	effectiveConcurrency := plan.MinInt(userConcurrency, taskCount, cpuCores)

	// Format description
	description := fmt.Sprintf("@parallel {%d concurrent}", effectiveConcurrency)

	// Create step with inner commands
	innerStep := plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: "Inner commands",
		Children:    []plan.ExecutionStep{inner},
	}
	if inner.Children != nil {
		innerStep.Children = inner.Children
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: description,
		Children:    []plan.ExecutionStep{innerStep},
		Timing: &plan.TimingInfo{
			ConcurrencyLimit: effectiveConcurrency,
		},
		Metadata: map[string]string{
			"decorator":             "parallel",
			"mode":                  mode,
			"user_concurrency":      fmt.Sprintf("%d", userConcurrency),
			"effective_concurrency": fmt.Sprintf("%d", effectiveConcurrency),
			"task_count":            fmt.Sprintf("%d", taskCount),
			"cpu_cores":             fmt.Sprintf("%d", cpuCores),
		},
	}
}

// ================================================================================================
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateBlockHint provides code generation hint for parallel execution

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractParameters extracts and validates parallel execution parameters
func (p *ParallelDecorator) extractParameters(params []decorators.Param, defaultConcurrency int) (mode string, concurrency int, err error) {
	// Set defaults
	mode = "fail-fast"
	concurrency = defaultConcurrency

	// Extract optional parameters
	for _, param := range params {
		switch param.GetName() {
		case "mode":
			if val, ok := param.GetValue().(string); ok {
				mode = val
			} else {
				return "", 0, fmt.Errorf("@parallel mode parameter must be a string, got %T", param.GetValue())
			}
		case "concurrency":
			if val, ok := param.GetValue().(int); ok {
				concurrency = val
			} else if val, ok := param.GetValue().(float64); ok {
				concurrency = int(val)
			} else if val, ok := param.GetValue().(string); ok {
				// Try to parse string as integer
				if parsed, err := strconv.Atoi(val); err == nil {
					concurrency = parsed
				} else {
					return "", 0, fmt.Errorf("@parallel concurrency parameter must be a number, got string %q", val)
				}
			} else {
				return "", 0, fmt.Errorf("@parallel concurrency parameter must be a number, got %T", param.GetValue())
			}
		default:
			return "", 0, fmt.Errorf("@parallel unknown parameter: %s", param.GetName())
		}
	}

	// Cap concurrency for safety
	maxConcurrency := 50
	if concurrency > maxConcurrency {
		return "", 0, fmt.Errorf("@parallel concurrency cannot exceed %d, got %d", maxConcurrency, concurrency)
	}

	// Validate mode
	validModes := map[string]bool{
		"fail-fast": true, // Stop scheduling on first failure, wait for running tasks
		"immediate": true, // Stop scheduling and cancel running tasks immediately
		"all":       true, // Run all tasks to completion, aggregate errors
	}
	if !validModes[mode] {
		return "", 0, fmt.Errorf("invalid @parallel mode %q, must be one of: fail-fast, immediate, all", mode)
	}

	// Validate concurrency
	if concurrency <= 0 {
		return "", 0, fmt.Errorf("@parallel concurrency must be positive, got %d", concurrency)
	}
	if concurrency > maxConcurrency {
		return "", 0, fmt.Errorf("@parallel concurrency %d exceeds maximum allowed %d", concurrency, maxConcurrency)
	}

	return mode, concurrency, nil
}

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (generic interface)
// ================================================================================================

// Validate parses and validates parameters, returning typed ParallelParams as any
func (p *ParallelDecorator) Validate(params []decorators.Param) (any, error) {
	// Set defaults
	result := ParallelParams{
		Mode:        "fail-fast",
		Concurrency: 4, // Default concurrency
	}

	// Extract optional parameters
	for _, param := range params {
		switch param.GetName() {
		case "mode":
			if val, ok := param.GetValue().(string); ok {
				result.Mode = val
			} else {
				return nil, fmt.Errorf("@parallel mode parameter must be a string, got %T", param.GetValue())
			}
		case "concurrency":
			if val, ok := param.GetValue().(int); ok {
				result.Concurrency = val
			} else if val, ok := param.GetValue().(float64); ok {
				result.Concurrency = int(val)
			} else if val, ok := param.GetValue().(string); ok {
				// Try to parse string as integer
				if parsed, err := strconv.Atoi(val); err == nil {
					result.Concurrency = parsed
				} else {
					return nil, fmt.Errorf("@parallel concurrency parameter must be a number, got string %q", val)
				}
			} else {
				return nil, fmt.Errorf("@parallel concurrency parameter must be a number, got %T", param.GetValue())
			}
		default:
			return nil, fmt.Errorf("@parallel unknown parameter: %s", param.GetName())
		}
	}

	// Cap concurrency for safety
	maxConcurrency := 50
	if result.Concurrency > maxConcurrency {
		return nil, fmt.Errorf("@parallel concurrency cannot exceed %d, got %d", maxConcurrency, result.Concurrency)
	}

	// Validate mode
	validModes := map[string]bool{
		"fail-fast": true, // Stop scheduling on first failure, wait for running tasks
		"immediate": true, // Stop scheduling and cancel running tasks immediately
		"all":       true, // Run all tasks to completion, aggregate errors
	}
	if !validModes[result.Mode] {
		return nil, fmt.Errorf("invalid @parallel mode %q, must be one of: fail-fast, immediate, all", result.Mode)
	}

	// Validate concurrency
	if result.Concurrency <= 0 {
		return nil, fmt.Errorf("@parallel concurrency must be positive, got %d", result.Concurrency)
	}
	if result.Concurrency > maxConcurrency {
		return nil, fmt.Errorf("@parallel concurrency %d exceeds maximum allowed %d", result.Concurrency, maxConcurrency)
	}

	return result, nil
}

// Plan generates an execution plan for the parallel operation
func (p *ParallelDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(ParallelParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: "@parallel(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "parallel",
				"error":     "invalid_params",
			},
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@parallel(mode=%s, concurrency=%d)", params.Mode, params.Concurrency),
		Command:     fmt.Sprintf("# Execute %d commands in parallel", params.Concurrency),
		Children:    []plan.ExecutionStep{}, // Will be populated by plan generator
		Timing: &plan.TimingInfo{
			ConcurrencyLimit: params.Concurrency,
		},
		Metadata: map[string]string{
			"decorator":      "parallel",
			"mode":           params.Mode,
			"concurrency":    fmt.Sprintf("%d", params.Concurrency),
			"execution_mode": "concurrency",
			"color":          plan.ColorBlue,
		},
	}
}

// Execute performs the parallel operation
func (p *ParallelDecorator) Execute(ctx decorators.Context, validated any) (decorators.CommandResult, error) {
	params, ok := validated.(ParallelParams)
	if !ok {
		return nil, fmt.Errorf("@parallel: invalid parameters")
	}

	// TODO: Runtime execution - implement when interpreter is rebuilt
	// Will use params.Mode and params.Concurrency for execution control
	_ = params // Prevent unused variable warning
	return &simpleCommandResult{
		stdout:   "",
		stderr:   "parallel execution not implemented yet - use plan mode",
		exitCode: 1,
	}, nil
}

// RequiresBlock returns the block requirements for @parallel
func (p *ParallelDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
