package builtin

import (
	"fmt"
	"strconv"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @parallel decorator on package import
func init() {
	decorators.RegisterBlock(NewParallelDecorator())
}

// ParallelDecorator implements the @parallel decorator using the core decorator interfaces
type ParallelDecorator struct{}

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

// ImportRequirements returns the dependencies needed for code generation
func (p *ParallelDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"sync", "runtime"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// WrapCommands executes multiple steps in parallel using the documented API
func (p *ParallelDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, commands decorators.CommandSeq) decorators.CommandResult {
	// Calculate default concurrency based on CPU cores
	defaultConcurrency := ctx.NumCPU * 2
	maxConcurrency := 50
	if defaultConcurrency > maxConcurrency {
		defaultConcurrency = maxConcurrency
	}

	mode, _, err := p.extractParameters(args, defaultConcurrency)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@parallel parameter error: %v", err),
			ExitCode: 1,
		}
	}

	// Convert mode string to ParallelMode enum
	var parallelMode decorators.ParallelMode
	switch mode {
	case "fail-fast":
		parallelMode = decorators.ParallelModeFailFast
	case "immediate":
		parallelMode = decorators.ParallelModeFailImmediate
	case "all":
		parallelMode = decorators.ParallelModeAll
	default:
		parallelMode = decorators.ParallelModeFailFast
	}

	// Use the documented context helper method for parallel execution
	result := ctx.ExecParallel(commands.Steps, parallelMode)

	// Add parallel execution marker for debugging
	if result.Stdout != "" {
		result.Stdout = fmt.Sprintf("[parallel] %s", result.Stdout)
	}

	return result
}

// Describe returns description for dry-run display
func (p *ParallelDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	// Calculate default concurrency based on CPU cores
	defaultConcurrency := ctx.NumCPU * 2
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
	cpuCores := ctx.NumCPU
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
func (p *ParallelDecorator) extractParameters(params []decorators.DecoratorParam, defaultConcurrency int) (mode string, concurrency int, err error) {
	// Set defaults
	mode = "fail-fast"
	concurrency = defaultConcurrency

	// Extract optional parameters
	for _, param := range params {
		switch param.Name {
		case "mode":
			if val, ok := param.Value.(string); ok {
				mode = val
			} else {
				return "", 0, fmt.Errorf("@parallel mode parameter must be a string, got %T", param.Value)
			}
		case "concurrency":
			if val, ok := param.Value.(int); ok {
				concurrency = val
			} else if val, ok := param.Value.(float64); ok {
				concurrency = int(val)
			} else if val, ok := param.Value.(string); ok {
				// Try to parse string as integer
				if parsed, err := strconv.Atoi(val); err == nil {
					concurrency = parsed
				} else {
					return "", 0, fmt.Errorf("@parallel concurrency parameter must be a number, got string %q", val)
				}
			} else {
				return "", 0, fmt.Errorf("@parallel concurrency parameter must be a number, got %T", param.Value)
			}
		default:
			return "", 0, fmt.Errorf("@parallel unknown parameter: %s", param.Name)
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
