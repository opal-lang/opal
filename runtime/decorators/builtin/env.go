package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @env decorator on package import
func init() {
	decorators.Register(NewEnvDecorator())
}

// EnvDecorator implements the @env decorator using the core decorator interfaces
type EnvDecorator struct{}

// NewEnvDecorator creates a new env decorator
func NewEnvDecorator() *EnvDecorator {
	return &EnvDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (e *EnvDecorator) Name() string {
	return "env"
}

// Description returns a human-readable description
func (e *EnvDecorator) Description() string {
	return "Access environment variables with optional defaults from frozen environment"
}

// ParameterSchema returns the expected parameters for this decorator
func (e *EnvDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "key",
			Type:        decorators.ArgTypeString,
			Required:    true,
			Description: "Environment variable name",
		},
		{
			Name:        "default",
			Type:        decorators.ArgTypeString,
			Required:    false,
			Description: "Default value if environment variable is not set",
		},
		{
			Name:        "allowEmpty",
			Type:        decorators.ArgTypeBool,
			Required:    false,
			Description: "If true, empty string values are preserved instead of using default",
		},
	}
}

// Examples returns usage examples
func (e *EnvDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code:        "@env(HOME)",
			Description: "Reference the HOME environment variable",
		},
		{
			Code:        "@env(PORT, \"8080\")",
			Description: "Use PORT with default value of 8080",
		},
		{
			Code:        "kubectl --context @env(KUBE_CONTEXT) apply",
			Description: "Use environment variable in shell command",
		},
		{
			Code:        "@env(DEBUG, \"false\", allowEmpty=true)",
			Description: "Allow empty DEBUG variable, use default only if unset",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (e *EnvDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// VALUE DECORATOR METHODS
// ================================================================================================

// Render expands the environment variable value from frozen context
func (e *EnvDecorator) Render(ctx *decorators.Ctx, args []decorators.DecoratorParam) (string, error) {
	key, defaultValue, allowEmpty, err := e.extractDecoratorParameters(args)
	if err != nil {
		return "", fmt.Errorf("@env parameter error: %w", err)
	}

	// ✅ CORRECT: Read from frozen environment (deterministic)
	value, exists := ctx.Env.Get(key)

	// Use default if not exists or empty (unless allowEmpty=true)
	if !exists || (!allowEmpty && value == "") {
		return defaultValue, nil
	}

	return value, nil
}

// Describe returns description for dry-run display
func (e *EnvDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam) plan.ExecutionStep {
	key, defaultValue, allowEmpty, err := e.extractDecoratorParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@env(<error: %v>)", err),
			Command:     "",
		}
	}

	// Get the environment variable value from frozen environment
	value, exists := ctx.Env.Get(key)
	actualValue := value
	source := "captured"

	// Apply same logic as Render for consistency
	if !exists || (!allowEmpty && value == "") {
		actualValue = defaultValue
		source = "default"
	}

	description := fmt.Sprintf("@env(%s) → %q (%s)", key, actualValue, source)

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: description,
		Command:     actualValue,
		Metadata: map[string]string{
			"decorator":  "env",
			"key":        key,
			"value":      actualValue,
			"source":     source,
			"allowEmpty": fmt.Sprintf("%t", allowEmpty),
		},
	}
}

// ================================================================================================
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateHint would provide code generation hint - @env is ALWAYS runtime
// TODO: Implement GenerateHint with []decorators.DecoratorParam when codegen is needed

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractDecoratorParameters extracts the environment variable key, default value, and allowEmpty flag
func (e *EnvDecorator) extractDecoratorParameters(params []decorators.DecoratorParam) (key string, defaultValue string, allowEmpty bool, err error) {
	// Extract key (first positional parameter or named "key")
	key, err = decorators.ExtractPositionalString(params, 0, "")
	if err != nil || key == "" {
		// Try named parameter
		key, err = decorators.ExtractString(params, "key", "")
		if err != nil || key == "" {
			return "", "", false, fmt.Errorf("@env requires an environment variable key")
		}
	}

	// Extract optional parameters with defaults
	defaultValue, err = decorators.ExtractString(params, "default", "")
	if err != nil {
		return "", "", false, fmt.Errorf("@env default parameter error: %v", err)
	}

	// Try positional default value if not found as named parameter
	if defaultValue == "" {
		defaultValue, _ = decorators.ExtractPositionalString(params, 1, "")
	}

	allowEmpty, err = decorators.ExtractBool(params, "allowEmpty", false)
	if err != nil {
		return "", "", false, fmt.Errorf("@env allowEmpty parameter error: %v", err)
	}

	return key, defaultValue, allowEmpty, nil
}

// extractParameters extracts the environment variable key, default value, and allowEmpty flag (legacy for codegen)
// Legacy extractParameters method disabled - used AST types
// TODO: Remove when GenerateHint is properly updated
