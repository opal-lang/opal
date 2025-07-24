package decorators

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// EnvDecorator implements the @env decorator for environment variable access
type EnvDecorator struct{}

// Template for environment variable access with default value
const envWithDefaultTemplate = `func() string {
	if value := os.Getenv({{.Key}}); value != "" {
		return value
	}
	return {{.Default}}
}()`

// Name returns the decorator name
func (e *EnvDecorator) Name() string {
	return "env"
}

// Description returns a human-readable description
func (e *EnvDecorator) Description() string {
	return "Access environment variables with optional defaults"
}

// ParameterSchema returns the expected parameters for this decorator
func (e *EnvDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "key",
			Type:        ast.StringType,
			Required:    true,
			Description: "Environment variable name",
		},
		{
			Name:        "default",
			Type:        ast.StringType,
			Required:    false,
			Description: "Default value if environment variable is not set",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (e *EnvDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter) *execution.ExecutionResult {
	// Validate parameters first

	// Get the environment variable key using helper
	key := ast.GetStringParam(params, "key", "")
	if key == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if keyLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			key = keyLiteral.Value
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return e.executeInterpreter(ctx, key, params)
	case execution.GeneratorMode:
		return e.executeGenerator(ctx, key, params)
	case execution.PlanMode:
		return e.executePlan(ctx, key, params)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter gets environment variable value in interpreter mode
func (e *EnvDecorator) executeInterpreter(ctx *execution.ExecutionContext, key string, params []ast.NamedParameter) *execution.ExecutionResult {
	// Get the environment variable value
	value := os.Getenv(key)

	// If not found and default provided, use default
	if value == "" {
		value = ast.GetStringParam(params, "default", "")
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  value,
		Error: nil,
	}
}

// executeGenerator generates Go code for environment variable access
func (e *EnvDecorator) executeGenerator(ctx *execution.ExecutionContext, key string, params []ast.NamedParameter) *execution.ExecutionResult {
	// Get default value if provided
	defaultValue := ast.GetStringParam(params, "default", "")

	var code string
	// Generate Go code based on whether default is provided
	if defaultValue == "" {
		// No default value - simple os.Getenv call
		code = fmt.Sprintf(`os.Getenv(%q)`, key)
	} else {
		// With default value - use template
		tmpl, err := template.New("envWithDefault").Parse(envWithDefaultTemplate)
		if err != nil {
			return &execution.ExecutionResult{
				Mode:  execution.GeneratorMode,
				Data:  "",
				Error: fmt.Errorf("failed to parse env template: %w", err),
			}
		}

		templateData := struct {
			Key     string
			Default string
		}{
			Key:     fmt.Sprintf("%q", key),
			Default: fmt.Sprintf("%q", defaultValue),
		}

		var result strings.Builder
		if err := tmpl.Execute(&result, templateData); err != nil {
			return &execution.ExecutionResult{
				Mode:  execution.GeneratorMode,
				Data:  "",
				Error: fmt.Errorf("failed to execute env template: %w", err),
			}
		}
		code = result.String()
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  code,
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (e *EnvDecorator) executePlan(ctx *execution.ExecutionContext, key string, params []ast.NamedParameter) *execution.ExecutionResult {
	// Get default value if provided
	defaultValue := ast.GetStringParam(params, "default", "")

	// Get the actual environment value (in dry run, we still check the env)
	var description string
	actualValue := os.Getenv(key)
	if actualValue != "" {
		description = fmt.Sprintf("Environment variable: $%s → %q", key, actualValue)
	} else if defaultValue != "" {
		description = fmt.Sprintf("Environment variable: $%s → %q (default)", key, defaultValue)
	} else {
		description = fmt.Sprintf("Environment variable: $%s → <unset>", key)
	}

	element := plan.Decorator("env").
		WithType("function").
		WithParameter("key", key).
		WithDescription(description)

	if defaultValue != "" {
		element = element.WithParameter("default", defaultValue)
	}

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (e *EnvDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"os"}, // Env decorator needs os package
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the env decorator
func init() {
	RegisterFunction(&EnvDecorator{})
}
