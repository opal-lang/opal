package decorators

import (
	"fmt"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/types"
)

// VarDecorator implements the @var value decorator for accessing plan-time variables.
//
// # Trait Inheritance
//
// @var inherits transport sensitivity from its source declaration:
//
//   - var X = "literal"      → @var.X is transport-agnostic (literals have no transport context)
//   - var X = @env.HOME      → @var.X is transport-sensitive (inherits from @env)
//   - var X = @aws.secret(k) → @var.X inherits from @aws.secret's sensitivity
//
// The @var decorator itself has no fixed sensitivity - it's a passthrough that
// inherits traits from whatever value was assigned to the variable.
//
// # Transport Boundary Enforcement
//
// When @var.X is used across a transport boundary (e.g., inside @ssh block),
// the vault checks the underlying expression's TransportSensitive flag:
//
//   - If the source was transport-sensitive (@env), access is blocked
//   - If the source was transport-agnostic (literal), access is allowed
//
// This ensures that environment variables resolved locally cannot leak into
// remote sessions, even when accessed indirectly through @var.
//
// # Example
//
//	var LOCAL_HOME = @env.HOME     // HOME inherits transport-sensitivity from @env
//	var VERSION = "1.0.0"          // VERSION is transport-agnostic (literal)
//
//	@ssh("server") {
//	    echo @var.VERSION          // OK: VERSION is transport-agnostic
//	    echo @var.LOCAL_HOME       // ERROR: LOCAL_HOME is transport-sensitive
//	}
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
	results := make([]decorator.ResolveResult, len(calls))

	for i, call := range calls {
		var varName string
		if call.Primary != nil {
			varName = *call.Primary
		} else if raw, ok := call.Params["arg1"]; ok {
			name, ok := raw.(string)
			if !ok {
				results[i] = decorator.ResolveResult{
					Value:  nil,
					Origin: "var.<unknown>",
					Error:  fmt.Errorf("@var arg1 must be a string"),
				}
				continue
			}
			varName = name
		}

		if varName == "" {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: "var.<unknown>",
				Error:  fmt.Errorf("@var requires a variable name"),
			}
			continue
		}

		if ctx.LookupValue == nil {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("@var lookup function is not available"),
			}
			continue
		}

		value, ok := ctx.LookupValue(varName)
		if !ok {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("variable %q not found in any scope", varName),
			}
			continue
		}

		// Return value directly (preserves original type: string, int, bool, map, slice)
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
