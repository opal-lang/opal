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

// timeoutExecutionTemplate generates Go code for timeout logic
const timeoutExecutionTemplate = `func() error {
	timeout, err := time.ParseDuration("{{.Duration}}")
	if err != nil {
		return fmt.Errorf("invalid timeout duration '{{.Duration}}': %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during execution: %v", r)
			}
		}()

		{{range $i, $cmd := .Commands}}
		// Check for cancellation before command {{$i}}
		select {
		case <-timeoutCtx.Done():
			done <- timeoutCtx.Err()
			return
		default:
		}

		// Execute command {{$i}}
		if err := func() error {
			{{generateShellCode $cmd}}
			return nil
		}(); err != nil {
			done <- err
			return
		}
		{{end}}

		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("command execution failed: %w", err)
		}
		return nil
	case <-timeoutCtx.Done():
		return fmt.Errorf("command execution timed out after {{.Duration}}")
	}
}()`

// TimeoutTemplateData holds data for the timeout template
type TimeoutTemplateData struct {
	Duration string
	Commands []ast.CommandContent
}

// TimeoutDecorator implements the @timeout decorator for command execution with time limits
type TimeoutDecorator struct{}

// Name returns the decorator name
func (t *TimeoutDecorator) Name() string {
	return "timeout"
}

// Description returns a human-readable description
func (t *TimeoutDecorator) Description() string {
	return "Execute commands with a time limit, cancelling on timeout"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TimeoutDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "duration",
			Type:        ast.DurationType,
			Required:    true,
			Description: "Maximum execution time (e.g., '30s', '5m', '1h')",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (t *TimeoutDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Validate parameters first

	// Parse timeout duration
	var timeout time.Duration
	durationParam := ast.FindParameter(params, "duration")
	if durationParam == nil && len(params) > 0 {
		durationParam = &params[0]
	}
	if durationParam != nil {
		if durLit, ok := durationParam.Value.(*ast.DurationLiteral); ok {
			var err error
			timeout, err = time.ParseDuration(durLit.Value)
			if err != nil {
				return &execution.ExecutionResult{
					Mode:  ctx.Mode(),
					Data:  nil,
					Error: fmt.Errorf("invalid duration '%s': %w", durLit.Value, err),
				}
			}
		} else {
			return &execution.ExecutionResult{
				Mode:  ctx.Mode(),
				Data:  nil,
				Error: fmt.Errorf("duration parameter must be a duration literal"),
			}
		}
	} else {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("timeout decorator requires a duration parameter"),
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return t.executeInterpreter(ctx, timeout, content)
	case execution.GeneratorMode:
		return t.executeGenerator(ctx, timeout, content)
	case execution.PlanMode:
		return t.executePlan(ctx, timeout, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes commands with timeout in interpreter mode
func (t *TimeoutDecorator) executeInterpreter(ctx *execution.ExecutionContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Create context with timeout
	timeoutCtx, cancel := ctx.WithTimeout(timeout)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Execute commands in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during execution: %v", r)
			}
		}()

		// Execute commands using the unified execution engine
		for _, cmd := range content {
			// Check for cancellation before each command
			select {
			case <-timeoutCtx.Done():
				done <- timeoutCtx.Err()
				return
			default:
			}

			// Execute the command using the engine's content executor
			if err := timeoutCtx.ExecuteCommandContent(cmd); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err != nil {
			return &execution.ExecutionResult{
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: fmt.Errorf("command execution failed: %w", err),
			}
		}
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: nil,
		}
	case <-timeoutCtx.Done():
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("command execution timed out after %s", timeout),
		}
	}
}

// executeGenerator generates Go code for timeout logic using templates
func (t *TimeoutDecorator) executeGenerator(ctx *execution.ExecutionContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Prepare template data
	templateData := TimeoutTemplateData{
		Duration: timeout.String(),
		Commands: content,
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("timeout").Funcs(ctx.GetTemplateFunctions()).Parse(timeoutExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse timeout template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute timeout template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (t *TimeoutDecorator) executePlan(ctx *execution.ExecutionContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	durationStr := timeout.String()
	description := fmt.Sprintf("Execute %d commands with %s timeout (cancel if exceeded)", len(content), durationStr)

	element := plan.Decorator("timeout").
		WithType("block").
		WithTimeout(timeout).
		WithParameter("duration", durationStr).
		WithDescription(description)

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (t *TimeoutDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"time", "context", "fmt"}, // Timeout needs time and context packages
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the timeout decorator
func init() {
	RegisterBlock(&TimeoutDecorator{})
}
