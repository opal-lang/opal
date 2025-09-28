package builtin

import (
	"fmt"
	"os"

	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/core/plan"
	"github.com/aledsdavies/opal/runtime/execution"
)

// Register the @env decorator on package import
func init() {
	decorator := NewEnvDecorator()
	// Register with legacy interface (Phase 4: remove this)
	decorators.Register(decorator)
	// Register with new interface
	decorators.RegisterValueDecorator(decorator)
}

// EnvDecorator implements the @env decorator using the core decorator interfaces
type EnvDecorator struct{}

// EnvParams represents validated parameters for @env decorator
type EnvParams struct {
	Name    string `json:"name"`    // Environment variable name to reference
	Default string `json:"default"` // Default value if environment variable is not set or empty
}

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
func (e *EnvDecorator) ImportRequirements() execution.ImportRequirement {
	return execution.ImportRequirement{
		Packages: []string{},
		Binaries: []string{},
		Env:      map[string]string{},
	}
}

// ================================================================================================
// LEGACY VALUE DECORATOR METHODS (will be removed in Phase 4)
// ================================================================================================

// Render expands the environment variable value from frozen context
func (e *EnvDecorator) Render(ctx decorators.Context, args []decorators.Param) (string, error) {
	key, defaultValue, allowEmpty, err := e.extractDecoratorParameters(args)
	if err != nil {
		return "", fmt.Errorf("@env parameter error: %w", err)
	}

	// ✅ CORRECT: Read from frozen environment (deterministic)
	value, exists := ctx.GetEnv(key)

	// Use default if not exists or empty (unless allowEmpty=true)
	if !exists || (!allowEmpty && value == "") {
		return defaultValue, nil
	}

	return value, nil
}

// Describe returns description for dry-run display
func (e *EnvDecorator) Describe(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	key, defaultValue, allowEmpty, err := e.extractDecoratorParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@env(<error: %v>)", err),
			Command:     "",
		}
	}

	// Get the environment variable value from frozen environment
	value, exists := ctx.GetEnv(key)
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
// HELPER METHODS
// ================================================================================================

// extractDecoratorParameters extracts the environment variable key, default value, and allowEmpty flag
func (e *EnvDecorator) extractDecoratorParameters(params []decorators.Param) (key string, defaultValue string, allowEmpty bool, err error) {
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

// ================================================================================================
// NEW VALUE DECORATOR METHODS (target interface)
// ================================================================================================

// IsExpensive returns false as environment variable lookups are fast
func (e *EnvDecorator) IsExpensive() bool {
	return false // Environment variable lookups are very fast
}

// ================================================================================================
// NEW GENERIC INTERFACE METHODS (ValueDecorator[any])
// ================================================================================================

// resolveValue performs the actual environment variable resolution
func (e *EnvDecorator) resolveValue(params EnvParams) (string, string) {
	// Get the environment variable value directly from OS
	value := os.Getenv(params.Name)
	actualValue := value
	source := "environment"

	// Use default if not set or empty
	if value == "" {
		actualValue = params.Default
		source = "default"
	}

	return actualValue, source
}

// Validate validates parameters and returns EnvParams
func (e *EnvDecorator) Validate(args []decorators.Param) (any, error) {
	// Extract environment variable name (first positional parameter or named "name")
	envName, err := decorators.ExtractPositionalString(args, 0, "")
	if err != nil || envName == "" {
		// Try named parameter "name"
		envName, err = decorators.ExtractString(args, "name", "")
		if err != nil || envName == "" {
			return nil, fmt.Errorf("@env requires an environment variable name")
		}
	}

	if envName == "" {
		return nil, fmt.Errorf("@env requires a non-empty environment variable name")
	}

	// Extract optional default value (second positional parameter or named "default")
	defaultValue, _ := decorators.ExtractPositionalString(args, 1, "")
	if defaultValue == "" {
		// Try named parameter "default"
		defaultValue, _ = decorators.ExtractString(args, "default", "")
	}

	return EnvParams{
		Name:    envName,
		Default: defaultValue,
	}, nil
}

// Plan generates an execution plan using validated parameters
func (e *EnvDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(EnvParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: "@env(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "env",
				"error":     "invalid_params",
			},
		}
	}

	actualValue, source := e.resolveValue(params)
	description := fmt.Sprintf("@env(%s) → %q (%s)", params.Name, actualValue, source)

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: description,
		Command:     actualValue,
		Metadata: map[string]string{
			"decorator": "env",
			"name":      params.Name,
			"value":     actualValue,
			"source":    source,
			"default":   params.Default,
		},
	}
}

// Resolve gets the actual environment variable value using validated parameters
func (e *EnvDecorator) Resolve(ctx decorators.Context, validated any) (decorators.Resolved, error) {
	params, ok := validated.(EnvParams)
	if !ok {
		return nil, fmt.Errorf("@env: invalid parameters")
	}

	actualValue, _ := e.resolveValue(params)

	// Return resolved value
	return decorators.NewResolved(
		actualValue,
		decorators.ResolvedString,
		fmt.Sprintf("env:%s:%s", params.Name, actualValue),
	), nil
}

// extractParameters extracts the environment variable key, default value, and allowEmpty flag
// Legacy extractParameters method disabled - used AST types
// TODO: Remove when GenerateHint is properly updated
