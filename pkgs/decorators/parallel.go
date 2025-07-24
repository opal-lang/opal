package decorators

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// ParallelDecorator implements the @parallel decorator for concurrent command execution
type ParallelDecorator struct{}

// Template for parallel execution code generation
const parallelExecutionTemplate = `func() error {
	{{if .FailOnFirstError}}ctx, cancel := context.WithCancel(context.Background())
	defer cancel(){{end}}

	semaphore := make(chan struct{}, {{.Concurrency}})
	var wg sync.WaitGroup
	errChan := make(chan error, {{.CommandCount}})

	{{range $i, $cmd := .Commands}}
	// Command {{$i}}
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Acquire semaphore
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		{{if $.FailOnFirstError}}// Check cancellation
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}{{end}}

		// Execute command using template function
		if err := func() error {
			{{generateShellCode $cmd}}
		}(); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()
	{{end}}

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
			{{if .FailOnFirstError}}cancel() // Cancel remaining tasks
			break{{end}}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution failed: %s", strings.Join(errors, "; "))
	}
	return nil
}()`

// TemplateData holds data for template execution
type ParallelTemplateData struct {
	Concurrency      int
	FailOnFirstError bool
	CommandCount     int
	Commands         []ast.CommandContent
}

// Name returns the decorator name
func (p *ParallelDecorator) Name() string {
	return "parallel"
}

// Description returns a human-readable description
func (p *ParallelDecorator) Description() string {
	return "Execute commands concurrently with optional concurrency limit and fail-fast behavior"
}

// ParameterSchema returns the expected parameters for this decorator
func (p *ParallelDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "concurrency",
			Type:        ast.NumberType,
			Required:    false,
			Description: "Maximum number of commands to run concurrently (default: unlimited)",
		},
		{
			Name:        "failOnFirstError",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Cancel remaining tasks on first error (default: false)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (p *ParallelDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Parse parameters with defaults
	concurrency := len(content) // Default: no limit (run all at once)
	failOnFirstError := false   // Default: continue on errors

	concurrency = ast.GetIntParam(params, "concurrency", concurrency)
	failOnFirstError = ast.GetBoolParam(params, "failOnFirstError", failOnFirstError)

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return p.executeInterpreter(ctx, concurrency, failOnFirstError, content)
	case execution.GeneratorMode:
		return p.executeGenerator(ctx, concurrency, failOnFirstError, content)
	case execution.PlanMode:
		return p.executePlan(ctx, concurrency, failOnFirstError, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes commands concurrently in interpreter mode
func (p *ParallelDecorator) executeInterpreter(ctx *execution.ExecutionContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Create context for cancellation if failOnFirstError is true
	execCtx := ctx
	var cancel context.CancelFunc
	if failOnFirstError {
		execCtx, cancel = ctx.WithCancel()
		defer cancel()
	}

	// Use semaphore to limit concurrency
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(content))

	// Execute each command with concurrency control
	for i, cmd := range content {
		// Check if context is cancelled
		select {
		case <-execCtx.Done():
			return &execution.ExecutionResult{
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: execCtx.Err(),
			}
		default:
		}

		wg.Add(1)
		go func(commandIndex int, command ast.CommandContent) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check cancellation again before executing
			select {
			case <-execCtx.Done():
				errChan <- execCtx.Err()
				return
			default:
			}

			// Execute the actual command content
			err := execCtx.ExecuteCommandContent(command)
			errChan <- err
		}(i, cmd)
	}

	// Wait for all commands to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors and handle fail-fast behavior
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
			if failOnFirstError && cancel != nil {
				cancel() // Cancel remaining tasks
				break
			}
		}
	}

	var finalError error
	if len(errors) > 0 {
		finalError = fmt.Errorf("parallel execution failed: %s", strings.Join(errors, "; "))
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: finalError,
	}
}

// executeGenerator generates Go code for parallel execution
func (p *ParallelDecorator) executeGenerator(ctx *execution.ExecutionContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Prepare template data
	templateData := ParallelTemplateData{
		Concurrency:      concurrency,
		FailOnFirstError: failOnFirstError,
		CommandCount:     len(content),
		Commands:         content, // Pass raw AST content
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("parallel").Funcs(ctx.GetTemplateFunctions()).Parse(parallelExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse parallel template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute parallel template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (p *ParallelDecorator) executePlan(ctx *execution.ExecutionContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
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

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (p *ParallelDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"context", "sync", "fmt", "strings"}, // Parallel needs sync, context, etc.
		ThirdParty:      []string{},                                    // Plan import removed - only needed for dry-run which isn't implemented in generated binaries yet
		GoModules:       map[string]string{},
	}
}

// init registers the parallel decorator
func init() {
	RegisterBlock(&ParallelDecorator{})
}
