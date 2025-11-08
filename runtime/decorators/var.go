package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/types"
)

// VarDecorator implements the @var value decorator.
// @var is transport-agnostic - it reads from the plan-time variable store.
type VarDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *VarDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("var").
		Summary("Access plan-time variables").
		Roles(decorator.RoleProvider).
		PrimaryParamString("name", "Variable name to retrieve").
		Examples("deployEnv", "version", "region").
		Done().
		Returns(types.TypeString, "Value of the variable").
		TransportScope(decorator.TransportScopeAny).
		Pure().
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

// Resolve implements the Value interface with batch support.
// @var just loops internally since there are no external calls.
func (d *VarDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	invariant.NotNil(ctx.Vars, "ctx.Vars")

	results := make([]decorator.ResolveResult, len(calls))

	for i, call := range calls {
		// Get variable name from primary parameter
		if call.Primary == nil {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: "var.<unknown>",
				Error:  fmt.Errorf("@var requires a variable name"),
			}
			continue
		}

		varName := *call.Primary

		// Look up variable in context
		value, exists := ctx.Vars[varName]
		if !exists {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("variable %q not found", varName),
			}
			continue
		}

		results[i] = decorator.ResolveResult{
			Value:  value,
			Origin: fmt.Sprintf("var.%s", varName),
			Error:  nil,
		}
	}

	return results, nil
}

// Register @var decorator with the global registry
func init() {
	if err := decorator.Register("var", &VarDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @var decorator: %v", err))
	}
}
