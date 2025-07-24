package decorators

import (
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// VarDecorator implements the @var decorator for variable references
type VarDecorator struct{}

// Name returns the decorator name
func (v *VarDecorator) Name() string {
	return "var"
}

// Description returns a human-readable description
func (v *VarDecorator) Description() string {
	return "Reference variables defined in the CLI file"
}

// ParameterSchema returns the expected parameters for this decorator
func (v *VarDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "name",
			Type:        ast.IdentifierType,
			Required:    true,
			Description: "Variable name to reference",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (v *VarDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter) *execution.ExecutionResult {
	// Validate parameters first

	// Get the variable name
	var varName string
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		nameParam = &params[0]
	}
	if ident, ok := nameParam.Value.(*ast.Identifier); ok {
		varName = ident.Name
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return v.executeInterpreter(ctx, varName)
	case execution.GeneratorMode:
		return v.executeGenerator(ctx, varName)
	case execution.PlanMode:
		return v.executePlan(ctx, varName)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter gets variable value in interpreter mode
func (v *VarDecorator) executeInterpreter(ctx *execution.ExecutionContext, varName string) *execution.ExecutionResult {
	// Look up the variable in the execution context
	if value, exists := ctx.GetVariable(varName); exists {
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  value,
			Error: nil,
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: fmt.Errorf("variable '%s' not defined", varName),
	}
}

// executeGenerator generates Go code for variable access
func (v *VarDecorator) executeGenerator(ctx *execution.ExecutionContext, varName string) *execution.ExecutionResult {
	// Return the variable name to reference the generated variable
	if _, exists := ctx.GetVariable(varName); exists {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  varName,
			Error: nil,
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  nil,
		Error: fmt.Errorf("variable '%s' not defined", varName),
	}
}

// executePlan creates a plan element for dry-run mode
func (v *VarDecorator) executePlan(ctx *execution.ExecutionContext, varName string) *execution.ExecutionResult {
	// Look up the variable in the execution context
	var description string
	if value, exists := ctx.GetVariable(varName); exists {
		description = fmt.Sprintf("Variable resolution: ${%s} → %q", varName, value)
	} else {
		description = fmt.Sprintf("Variable resolution: ${%s} → <undefined>", varName)
	}

	element := plan.Decorator("var").
		WithType("function").
		WithParameter("name", varName).
		WithDescription(description)

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (v *VarDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{}, // No additional imports needed
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the var decorator
func init() {
	RegisterFunction(&VarDecorator{})
}
