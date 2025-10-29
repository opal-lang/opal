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
		PrimaryParam("name", types.TypeString, "Variable name to retrieve", "deployEnv", "version", "region").
		Returns(types.TypeString, "Value of the variable").
		TransportScope(decorator.TransportScopeAny).
		Pure().
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

// Resolve implements the Value interface.
func (d *VarDecorator) Resolve(ctx decorator.ValueEvalContext, call decorator.ValueCall) (any, error) {
	invariant.NotNil(ctx.Vars, "ctx.Vars")

	// Get variable name from primary parameter
	if call.Primary == nil {
		return nil, fmt.Errorf("@var requires a variable name")
	}

	varName := *call.Primary

	// Look up variable in context
	value, exists := ctx.Vars[varName]
	if !exists {
		return nil, fmt.Errorf("variable %q not found", varName)
	}

	return value, nil
}

// Register @var decorator with the global registry
func init() {
	if err := decorator.Register("var", &VarDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @var decorator: %v", err))
	}
}
