package decorators

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// RetryDecorator implements the @retry decorator for retrying failed command execution
type RetryDecorator struct{}

// Template for retry execution code generation
const retryExecutionTemplate = `return func() error {
	maxAttempts := {{.MaxAttempts}}
	delay, err := time.ParseDuration({{printf "%q" .Delay}})
	if err != nil {
		return fmt.Errorf("invalid retry delay '{{.Delay}}': %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("Retry attempt %d/%d\n", attempt, maxAttempts)

		// Execute commands
		execErr := func() error {
			{{range $i, $cmd := .Commands}}
			// Execute command {{$i}}
			if err := func() error {
				{{generateShellCode $cmd}}
			}(); err != nil {
				return err
			}
			{{end}}
			return nil
		}()

		if execErr == nil {
			fmt.Printf("Commands succeeded on attempt %d\n", attempt)
			return nil
		}

		lastErr = execErr
		fmt.Printf("Attempt %d failed: %v\n", attempt, execErr)

		// Don't delay after the last attempt
		if attempt < maxAttempts {
			fmt.Printf("Waiting %s before next attempt...\n", delay)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all %d retry attempts failed, last error: %w", maxAttempts, lastErr)
}()`

// RetryTemplateData holds data for template execution
type RetryTemplateData struct {
	MaxAttempts int
	Delay       string
	Commands    []ast.CommandContent
}

// Name returns the decorator name
func (r *RetryDecorator) Name() string {
	return "retry"
}

// Description returns a human-readable description
func (r *RetryDecorator) Description() string {
	return "Retry command execution on failure with configurable attempts and delay"
}

// ParameterSchema returns the expected parameters for this decorator
func (r *RetryDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "attempts",
			Type:        ast.NumberType,
			Required:    true,
			Description: "Maximum number of retry attempts",
		},
		{
			Name:        "delay",
			Type:        ast.DurationType,
			Required:    false,
			Description: "Delay between retry attempts (default: 1s)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (r *RetryDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Validate parameters first

	// Check parameter count (retry supports 1-2 parameters: attempts, and optionally delay)
	if len(params) == 0 {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("retry decorator requires an 'attempts' parameter"),
		}
	}
	if len(params) > 2 {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("retry decorator accepts at most 2 parameters (attempts, delay), got %d", len(params)),
		}
	}

	// Check that attempts parameter exists
	attemptsParam := ast.FindParameter(params, "attempts")
	if attemptsParam == nil {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("retry decorator requires an 'attempts' parameter"),
		}
	}

	// Parse parameters
	maxAttempts := ast.GetIntParam(params, "attempts", 3)
	delay := ast.GetDurationParam(params, "delay", 1*time.Second)

	// Validate attempts is positive
	if maxAttempts <= 0 {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("retry attempts must be positive, got %d", maxAttempts),
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return r.executeInterpreter(ctx, maxAttempts, delay, content)
	case execution.GeneratorMode:
		return r.executeGenerator(ctx, maxAttempts, delay, content)
	case execution.PlanMode:
		return r.executePlan(ctx, maxAttempts, delay, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes retry logic in interpreter mode
func (r *RetryDecorator) executeInterpreter(ctx *execution.ExecutionContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Execute commands using the unified execution engine
		execErr := r.executeCommands(ctx, content)

		if execErr == nil {
			return &execution.ExecutionResult{
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: nil, // Success!
			}
		}

		lastErr = execErr

		// Don't delay after the last attempt
		if attempt < maxAttempts {
			time.Sleep(delay)
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: fmt.Errorf("all %d retry attempts failed, last error: %w", maxAttempts, lastErr),
	}
}

// executeGenerator generates Go code for retry logic
func (r *RetryDecorator) executeGenerator(ctx *execution.ExecutionContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Parse delay for code generation
	defaultDelay := delay.String()

	// Prepare template data
	templateData := RetryTemplateData{
		MaxAttempts: maxAttempts,
		Delay:       defaultDelay,
		Commands:    content,
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("retry").Funcs(ctx.GetTemplateFunctions()).Parse(retryExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse retry template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute retry template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (r *RetryDecorator) executePlan(ctx *execution.ExecutionContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	delayStr := delay.String()

	description := fmt.Sprintf("Execute %d commands with up to %d attempts", len(content), maxAttempts)
	if delayStr != "" && delayStr != "0s" {
		description += fmt.Sprintf(", %s delay between retries", delayStr)
	}

	element := plan.Decorator("retry").
		WithType("block").
		WithParameter("attempts", fmt.Sprintf("%d", maxAttempts)).
		WithDescription(description)

	if delayStr != "" && delayStr != "0s" {
		element = element.WithParameter("delay", delayStr)
	}

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// executeCommands executes commands using the unified execution engine
func (r *RetryDecorator) executeCommands(ctx *execution.ExecutionContext, content []ast.CommandContent) error {
	for _, cmd := range content {
		if err := ctx.ExecuteCommandContent(cmd); err != nil {
			return err
		}
	}
	return nil
}

// ImportRequirements returns the dependencies needed for code generation
func (r *RetryDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"time", "fmt"}, // Retry needs time for delays and fmt for errors
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the retry decorator
func init() {
	RegisterBlock(&RetryDecorator{})
}
