package decorators

import (
	"fmt"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
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
func (v *VarDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "name",
			Type:        ast.IdentifierType,
			Required:    true,
			Description: "Variable name to reference",
		},
	}
}

// ExpandInterpreter returns the actual variable value for interpreter mode
func (v *VarDecorator) ExpandInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter) *execution.ExecutionResult {
	varName, err := v.extractVariableName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("var parameter error: %w", err),
		}
	}

	// Look up the variable value from the .cli file variables
	if value, exists := ctx.GetVariable(varName); exists {
		return &execution.ExecutionResult{
			Data:  value, // Return the actual string value
			Error: nil,
		}
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: fmt.Errorf("variable '%s' not defined in .cli file", varName),
	}
}

// GenerateTemplate returns template for Go code that resolves the variable for generator mode
func (v *VarDecorator) GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter) (*execution.TemplateResult, error) {
	varName, err := v.extractVariableName(params)
	if err != nil {
		return nil, fmt.Errorf("var parameter error: %w", err)
	}

	// Create simple template that outputs the variable name
	// Variables are defined at the top of generated functions
	tmplStr := `{{.VarName}}`

	// Parse template
	tmpl, err := template.New("var").Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse var template: %w", err)
	}

	return &execution.TemplateResult{
		Template: tmpl,
		Data: struct {
			VarName string
		}{
			VarName: varName,
		},
	}, nil
}

// ExpandPlan returns description for dry-run display in plan mode
func (v *VarDecorator) ExpandPlan(ctx execution.PlanContext, params []ast.NamedParameter) *execution.ExecutionResult {
	varName, err := v.extractVariableName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("var parameter error: %w", err),
		}
	}

	// Look up the variable value for display
	if value, exists := ctx.GetVariable(varName); exists {
		return &execution.ExecutionResult{
			Data:  fmt.Sprintf("@var(%s) → %q", varName, value),
			Error: nil,
		}
	}

	return &execution.ExecutionResult{
		Data:  fmt.Sprintf("@var(%s) → <undefined>", varName),
		Error: nil,
	}
}

// extractVariableName extracts the variable name from decorator parameters
func (v *VarDecorator) extractVariableName(params []ast.NamedParameter) (string, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 1, 1, "var"); err != nil {
		return "", err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, v.ParameterSchema(), "var"); err != nil {
		return "", err
	}

	// Parse parameters (validation passed, so these should be safe)
	// Try to get the "name" parameter first
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		// Fallback to first parameter if no "name" parameter
		nameParam = &params[0]
	}

	if nameParam != nil {
		if ident, ok := nameParam.Value.(*ast.Identifier); ok {
			return ident.Name, nil
		}
	}

	return "", fmt.Errorf("@var decorator requires a valid identifier parameter")
}

// ImportRequirements returns the dependencies needed for code generation
func (v *VarDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{}, // No additional imports needed
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the var decorator
func init() {
	decorators.RegisterValue(&VarDecorator{})
}
