package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// Register the @var decorator on package import
func init() {
	decorator := NewVarDecorator()
	// Register with legacy interface (Phase 4: remove this)
	decorators.Register(decorator)
	// Register with new interface
	decorators.RegisterValueDecorator(decorator)
}

// VarDecorator implements the @var decorator using the core decorator interfaces
type VarDecorator struct{}

// NewVarDecorator creates a new var decorator
func NewVarDecorator() *VarDecorator {
	return &VarDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (v *VarDecorator) Name() string {
	return "var"
}

// Description returns a human-readable description
func (v *VarDecorator) Description() string {
	return "Reference variables defined in the CLI file"
}

// ParameterSchema returns the expected parameters for this decorator
func (v *VarDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "name",
			Type:        decorators.ArgTypeIdentifier,
			Required:    true,
			Description: "Variable name to reference",
		},
	}
}

// Examples returns usage examples
func (v *VarDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code:        "@var(BUILD_DIR)",
			Description: "Reference the BUILD_DIR variable",
		},
		{
			Code:        "cd @var(SRC) && make",
			Description: "Use variable in shell command",
		},
		{
			Code:        "docker build -t app:@var(VERSION) .",
			Description: "Use variable in command arguments",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (v *VarDecorator) ImportRequirements() execution.ImportRequirement {
	return execution.ImportRequirement{
		Packages: []string{},
		Binaries: []string{},
		Env:      map[string]string{},
	}
}

// ================================================================================================
// LEGACY VALUE DECORATOR METHODS (will be removed in Phase 4)
// ================================================================================================

// Render expands the variable value in the current context
func (v *VarDecorator) Render(ctx decorators.Context, args []decorators.Param) (string, error) {
	varName, err := v.extractDecoratorVariableName(args)
	if err != nil {
		return "", fmt.Errorf("@var parameter error: %w", err)
	}

	// Look up variable in context
	if value, exists := ctx.GetVar(varName); exists {
		return value, nil
	}

	// Variable not found
	return "", fmt.Errorf("undefined variable: %s", varName)
}

// Describe returns description for dry-run display
func (v *VarDecorator) Describe(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	varName, err := v.extractDecoratorVariableName(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@var(<error: %v>)", err),
			Command:     "",
		}
	}

	// Resolve the variable value during plan generation
	value, exists := ctx.GetVar(varName)
	if exists && value != "" {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@var(%s)¹ → %q", varName, value),
			Command:     value,
			Metadata: map[string]string{
				"decorator":   "var",
				"variable":    varName,
				"value":       value,
				"resolved_at": "plan",
				"source":      "cli_variable",
			},
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: fmt.Sprintf("@var(%s)¹ → <undefined>", varName),
		Command:     "",
		Metadata: map[string]string{
			"decorator": "var",
			"variable":  varName,
			"error":     "undefined_variable",
		},
	}
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractDecoratorVariableName extracts the variable name from decorator parameters
func (v *VarDecorator) extractDecoratorVariableName(params []decorators.Param) (string, error) {
	// Extract variable name (first positional parameter or named "name")
	varName, err := decorators.ExtractPositionalString(params, 0, "")
	if err != nil || varName == "" {
		// Try named parameter
		varName, err = decorators.ExtractString(params, "name", "")
		if err != nil || varName == "" {
			return "", fmt.Errorf("@var requires a variable name")
		}
	}

	if varName == "" {
		return "", fmt.Errorf("@var requires a non-empty variable name")
	}

	return varName, nil
}

// ================================================================================================
// NEW VALUE DECORATOR METHODS (target interface)
// ================================================================================================

// Plan generates an execution plan showing how the variable will be resolved
func (v *VarDecorator) Plan(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	varName, err := v.extractDecoratorVariableName(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@var(<error: %v>)", err),
			Command:     "",
			Metadata: map[string]string{
				"decorator": "var",
				"error":     err.Error(),
			},
		}
	}

	// Resolve the variable value during plan generation
	value, exists := ctx.GetVar(varName)
	if exists && value != "" {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@var(%s) → %q", varName, value),
			Command:     value,
			Metadata: map[string]string{
				"decorator":   "var",
				"variable":    varName,
				"value":       value,
				"resolved_at": "plan",
				"source":      "cli_variable",
			},
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@var(%s) → <undefined>", varName),
		Command:     "",
		Metadata: map[string]string{
			"decorator": "var",
			"variable":  varName,
			"error":     "undefined_variable",
		},
	}
}

// Resolve gets the actual variable value during execution
func (v *VarDecorator) Resolve(ctx decorators.Context, args []decorators.Param) (string, error) {
	varName, err := v.extractDecoratorVariableName(args)
	if err != nil {
		return "", fmt.Errorf("@var parameter error: %w", err)
	}

	// Look up variable in context
	if value, exists := ctx.GetVar(varName); exists {
		return value, nil
	}

	// Variable not found
	return "", fmt.Errorf("undefined variable: %s", varName)
}

// IsExpensive returns false as variable lookups are fast
func (v *VarDecorator) IsExpensive() bool {
	return false // Variable lookups are very fast
}

// extractVariableName extracts the variable name from AST parameters
// Legacy extractVariableName method disabled - used AST types
// TODO: Remove when GenerateHint is properly updated
