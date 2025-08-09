package decorators

import (
	"fmt"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// EnvDecorator implements the @env decorator for environment variable access
type EnvDecorator struct{}

// Name returns the decorator name
func (e *EnvDecorator) Name() string {
	return "env"
}

// Description returns a human-readable description
func (e *EnvDecorator) Description() string {
	return "Access environment variables with optional defaults"
}

// ParameterSchema returns the expected parameters for this decorator
func (e *EnvDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
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
			Description: "Default value if environment variable is not set or empty",
		},
		{
			Name:        "allowEmpty",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "If true, empty string values are preserved instead of using default",
		},
	}
}

// ExpandInterpreter returns the captured environment variable value for interpreter mode
func (e *EnvDecorator) ExpandInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter) *execution.ExecutionResult {
	key, defaultValue, allowEmpty, err := e.extractParameters(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("env parameter error: %w", err),
		}
	}

	// Get the environment variable value from captured environment (deterministic)
	value, exists := ctx.GetEnv(key)

	// Use captured value or default based on allowEmpty flag
	if !exists || (!allowEmpty && value == "") {
		value = defaultValue
	}

	return &execution.ExecutionResult{
		Data:  value, // Return the captured/default value
		Error: nil,
	}
}

// GenerateTemplate returns template for Go code that references captured environment for generator mode
func (e *EnvDecorator) GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter) (*execution.TemplateResult, error) {
	key, defaultValue, allowEmpty, err := e.extractParameters(params)
	if err != nil {
		return nil, fmt.Errorf("env parameter error: %w", err)
	}

	// Track this environment variable for global capture generation
	ctx.TrackEnvironmentVariableReference(key, defaultValue)

	// Create template for environment variable access
	var tmplStr string
	if defaultValue != "" {
		if allowEmpty {
			// If allowEmpty=true, only use default if not exists
			tmplStr = `func() string { if val, exists := ctx.Env[{{printf "%q" .Key}}]; exists { return val }; return {{printf "%q" .DefaultValue}} }()`
		} else {
			// Default behavior: use default if not exists or empty
			tmplStr = `func() string { if val, exists := ctx.Env[{{printf "%q" .Key}}]; exists && val != "" { return val }; return {{printf "%q" .DefaultValue}} }()`
		}
	} else {
		// No default, just use captured value
		tmplStr = `ctx.Env[{{printf "%q" .Key}}]`
	}

	// Parse template
	tmpl, err := template.New("env").Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env template: %w", err)
	}

	return &execution.TemplateResult{
		Template: tmpl,
		Data: struct {
			Key          string
			DefaultValue string
			AllowEmpty   bool
		}{
			Key:          key,
			DefaultValue: defaultValue,
			AllowEmpty:   allowEmpty,
		},
	}, nil
}

// ExpandPlan returns description showing the captured environment value for plan mode
func (e *EnvDecorator) ExpandPlan(ctx execution.PlanContext, params []ast.NamedParameter) *execution.ExecutionResult {
	key, defaultValue, allowEmpty, err := e.extractParameters(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("env parameter error: %w", err),
		}
	}

	// Get the environment variable value from captured environment (deterministic)
	value, exists := ctx.GetEnv(key)

	var displayValue string
	// Apply same logic as interpreter mode for consistency
	if !exists || (!allowEmpty && value == "") {
		if defaultValue != "" {
			displayValue = fmt.Sprintf("@env(%s) → %q (default)", key, defaultValue)
		} else {
			displayValue = fmt.Sprintf("@env(%s) → <unset>", key)
		}
	} else {
		displayValue = fmt.Sprintf("@env(%s) → %q (captured)", key, value)
	}

	return &execution.ExecutionResult{
		Data:  displayValue,
		Error: nil,
	}
}

// extractParameters extracts the environment variable key and default value from decorator parameters
func (e *EnvDecorator) extractParameters(params []ast.NamedParameter) (key string, defaultValue string, allowEmpty bool, err error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 1, 3, "env"); err != nil {
		return "", "", false, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, e.ParameterSchema(), "env"); err != nil {
		return "", "", false, err
	}

	// Validate environment variable name if present
	if err := decorators.ValidateEnvironmentVariableName(params, "key", "env"); err != nil {
		return "", "", false, err
	}

	// Parse parameters (validation passed, so these should be safe)
	key = ast.GetStringParam(params, "key", "")
	if key == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		switch v := params[0].Value.(type) {
		case *ast.StringLiteral:
			key = v.Value
		case *ast.Identifier:
			// Allow identifiers for convenience (e.g., @env(HOME) instead of @env("HOME"))
			key = v.Name
		}
	}

	// Additional check for empty key (shouldn't happen after validation)
	if key == "" {
		return "", "", false, fmt.Errorf("@env decorator requires a valid environment variable name")
	}

	// Get default value if provided
	defaultValue = ast.GetStringParam(params, "default", "")

	// Get allowEmpty flag (defaults to false for backward compatibility)
	allowEmpty = ast.GetBoolParam(params, "allowEmpty", false)

	return key, defaultValue, allowEmpty, nil
}

// ImportRequirements returns the dependencies needed for code generation
func (e *EnvDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{}, // No imports needed - generates string literals
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the env decorator
func init() {
	decorators.RegisterValue(&EnvDecorator{})
}
