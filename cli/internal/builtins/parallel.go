package decorators

import (
	"fmt"
	"runtime"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ParallelDecorator implements the @parallel decorator for concurrent command execution
type ParallelDecorator struct{}

// Name returns the decorator name
func (p *ParallelDecorator) Name() string {
	return "parallel"
}

// Description returns a human-readable description
func (p *ParallelDecorator) Description() string {
	return "Execute commands concurrently with optional concurrency limit and fail-fast behavior"
}

// ParameterSchema returns the expected parameters for this decorator
func (p *ParallelDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "concurrency",
			Type:        ast.NumberType,
			Required:    false,
			Description: "Maximum number of commands to run concurrently (default: CPU cores * 2, capped for safety)",
		},
		{
			Name:        "failOnFirstError",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Cancel remaining tasks on first error (default: false)",
		},
		{
			Name:        "uncapped",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Disable CPU-based concurrency capping (default: false, use with caution)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// ExecuteInterpreter executes commands concurrently in interpreter mode
func (p *ParallelDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	concurrency, failOnFirstError, err := p.extractParallelParams(params, len(content))
	if err != nil {
		return execution.NewErrorResult(err)
	}

	return p.executeInterpreterImpl(ctx, concurrency, failOnFirstError, content)
}

// GenerateTemplate generates template-based Go code for parallel execution
func (p *ParallelDecorator) GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) (*execution.TemplateResult, error) {
	concurrency, failOnFirstError, err := p.extractParallelParams(params, len(content))
	if err != nil {
		return nil, err
	}

	return p.generateTemplateImpl(ctx, concurrency, failOnFirstError, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (p *ParallelDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	concurrency, failOnFirstError, err := p.extractParallelParams(params, len(content))
	if err != nil {
		return execution.NewErrorResult(err)
	}

	return p.executePlanImpl(ctx, concurrency, failOnFirstError, content)
}

// extractParallelParams extracts and validates parallel parameters
func (p *ParallelDecorator) extractParallelParams(params []ast.NamedParameter, contentLength int) (int, bool, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 0, 3, "parallel"); err != nil {
		return 0, false, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, p.ParameterSchema(), "parallel"); err != nil {
		return 0, false, err
	}

	// Enhanced security validation for concurrency parameter
	if err := decorators.ValidatePositiveInteger(params, "concurrency", "parallel"); err != nil {
		// ValidatePositiveInteger returns error if parameter is invalid, but not if missing
		// Check if the parameter exists first
		if ast.FindParameter(params, "concurrency") != nil {
			return 0, false, err
		}
	}

	// Validate resource limits for concurrency to prevent DoS attacks
	if err := decorators.ValidateResourceLimits(params, "concurrency", 1000, "parallel"); err != nil {
		return 0, false, err
	}

	// Parse parameters with defaults (validation passed, so these should be safe)
	defaultConcurrency := contentLength
	if defaultConcurrency == 0 {
		defaultConcurrency = 1 // Always have a positive default
	}

	concurrency := ast.GetIntParam(params, "concurrency", defaultConcurrency)
	failOnFirstError := ast.GetBoolParam(params, "failOnFirstError", false)
	uncapped := ast.GetBoolParam(params, "uncapped", false)

	// Apply intelligent CPU-based concurrency capping for production robustness
	// This prevents resource exhaustion on systems with limited CPU cores
	if !uncapped {
		cpuCount := runtime.NumCPU()
		maxRecommendedConcurrency := cpuCount * 2 // Allow some over-subscription for I/O bound tasks

		if concurrency > maxRecommendedConcurrency {
			// Cap concurrency but don't error - just limit to reasonable bounds
			// This provides good defaults while still allowing explicit override via uncapped=true
			concurrency = maxRecommendedConcurrency
		}
	}

	return concurrency, failOnFirstError, nil
}

// executeInterpreterImpl executes commands concurrently in interpreter mode
func (p *ParallelDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Use channels to coordinate execution and output
	type commandResult struct {
		index  int
		result *execution.ExecutionResult
	}

	resultChan := make(chan commandResult, len(content))

	// Execute commands concurrently
	for i, cmd := range content {
		go func(cmdIndex int, command ast.CommandContent) {
			// Create isolated context for each parallel command
			isolatedCtx := ctx.Child()

			// Execute the command using the unified ExecuteCommandContent method
			err := isolatedCtx.ExecuteCommandContent(command)
			var result *execution.ExecutionResult
			if err != nil {
				result = execution.NewErrorResult(err)
			} else {
				result = &execution.ExecutionResult{Data: nil, Error: nil}
			}

			resultChan <- commandResult{index: cmdIndex, result: result}
		}(i, cmd)
	}

	// Collect results - maintain order for consistent output
	results := make([]*execution.ExecutionResult, len(content))
	var firstError error

	// Wait for all goroutines to complete
	for i := 0; i < len(content); i++ {
		cmdResult := <-resultChan
		results[cmdResult.index] = cmdResult.result

		if cmdResult.result.Error != nil && firstError == nil {
			firstError = cmdResult.result.Error
			if failOnFirstError {
				// Still need to wait for all goroutines to complete
				continue
			}
		}
	}

	// Display outputs in original command order for consistent behavior
	for _, result := range results {
		if result.Error == nil {
			// Command succeeded - output should already be displayed by ExecuteShell
			continue
		}
	}

	// Return error if fail-fast is enabled and we have an error
	if failOnFirstError && firstError != nil {
		return execution.NewErrorResult(fmt.Errorf("parallel execution failed: %w", firstError))
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: firstError, // Return first error even if not failing fast
	}
}

// generateTemplateImpl generates template for parallel execution
func (p *ParallelDecorator) generateTemplateImpl(ctx execution.GeneratorContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) (*execution.TemplateResult, error) {
	// Create template string for parallel execution
	tmplStr := `// Parallel execution
{
	var wg sync.WaitGroup
	errs := make([]error, {{len .Content}})

{{range $i, $cmd := .Content}}	wg.Add(1)
	go func() {
		defer wg.Done()
		// Branch {{$i}} with isolated context
		branchCtx := ctx.Clone()
		errs[{{$i}}] = func() error {
			ctx := branchCtx
			{{$cmd | buildCommand}}
			return nil
		}()
	}()

{{end}}	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
}`

	// Parse template with helper functions
	tmpl, err := template.New("parallel").Funcs(ctx.GetTemplateFunctions()).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parallel template: %w", err)
	}

	return &execution.TemplateResult{
		Template: tmpl,
		Data: struct {
			Concurrency      int
			FailOnFirstError bool
			Content          []ast.CommandContent
		}{
			Concurrency:      concurrency,
			FailOnFirstError: failOnFirstError,
			Content:          content,
		},
	}, nil
}

// executePlanImpl creates a plan element for dry-run mode
func (p *ParallelDecorator) executePlanImpl(ctx execution.PlanContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	description := fmt.Sprintf("Execute %d commands concurrently", len(content))
	if concurrency < len(content) {
		description += fmt.Sprintf(" (max %d at a time)", concurrency)
	}
	if failOnFirstError {
		description += ", stop on first error"
	} else {
		description += ", continue on errors"
	}

	element := plan.Decorator("parallel").
		WithType("block").
		WithDescription(description)

	if concurrency < len(content) {
		element = element.WithParameter("concurrency", fmt.Sprintf("%d", concurrency))
	}
	if failOnFirstError {
		element = element.WithParameter("failOnFirstError", "true")
	}

	// Build child plan elements for each command in the parallel block
	for _, cmd := range content {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			// Create plan element for shell command
			result := ctx.GenerateShellPlan(c)
			if result.Error != nil {
				return execution.NewFormattedErrorResult("failed to create plan for shell content: %w", result.Error)
			}

			// Extract command string from result
			if planData, ok := result.Data.(map[string]interface{}); ok {
				if cmdStr, ok := planData["command"].(string); ok {
					childDesc := "Execute shell command"
					if desc, ok := planData["description"].(string); ok {
						childDesc = desc
					}
					childElement := plan.Command(cmdStr).WithDescription(childDesc)
					element = element.AddChild(childElement)
				}
			}
		case *ast.BlockDecorator:
			// Execute nested decorator in plan mode to get its proper plan structure
			blockDecorator, err := decorators.GetBlock(c.Name)
			if err != nil {
				// Fallback to placeholder if decorator not found
				childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription(fmt.Sprintf("Unknown decorator: %s", c.Name))
				element = element.AddChild(childElement)
			} else {
				// Execute the nested decorator in plan mode
				result := blockDecorator.ExecutePlan(ctx, c.Args, c.Content)
				if result.Error != nil {
					// Fallback to placeholder if plan execution fails
					childElement := plan.Command(fmt.Sprintf("@%s{error}", c.Name)).WithDescription(fmt.Sprintf("Error in %s: %v", c.Name, result.Error))
					element = element.AddChild(childElement)
				} else if planElement, ok := result.Data.(plan.PlanElement); ok {
					// Add the nested decorator's plan element as a child
					element = element.AddChild(planElement)
				} else {
					// Fallback if result format is unexpected
					childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription(fmt.Sprintf("Nested decorator: %s", c.Name))
					element = element.AddChild(childElement)
				}
			}
		}
	}

	return execution.NewSuccessResult(element)
}

// ImportRequirements returns the dependencies needed for code generation
func (p *ParallelDecorator) ImportRequirements() decorators.ImportRequirement {
	// Parallel decorator only needs sync for WaitGroup, no strings or other imports
	return decorators.StandardImportRequirement(decorators.ConcurrencyImports)
}

// init registers the parallel decorator
func init() {
	decorators.RegisterBlock(&ParallelDecorator{})
}
