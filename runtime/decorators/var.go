package decorators

import (
	"fmt"
	"reflect"

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
	invariant.NotNil(ctx.Vault, "ctx.Vault")

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

		// Use reflection to call Vault methods (avoids circular import)
		vaultValue := reflect.ValueOf(ctx.Vault)

		// Call LookupVariable(varName) -> (string, error)
		lookupMethod := vaultValue.MethodByName("LookupVariable")
		if !lookupMethod.IsValid() {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("Vault.LookupVariable method not found"),
			}
			continue
		}

		lookupResults := lookupMethod.Call([]reflect.Value{reflect.ValueOf(varName)})
		if len(lookupResults) != 2 {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("unexpected LookupVariable return values"),
			}
			continue
		}

		// Check for error
		if !lookupResults[1].IsNil() {
			err := lookupResults[1].Interface().(error)
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  err,
			}
			continue
		}

		exprID := lookupResults[0].String()

		// Call GetExpression(exprID) -> *Expression
		getExprMethod := vaultValue.MethodByName("GetExpression")
		if !getExprMethod.IsValid() {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("Vault.GetExpression method not found"),
			}
			continue
		}

		exprResults := getExprMethod.Call([]reflect.Value{reflect.ValueOf(exprID)})
		if len(exprResults) != 1 || exprResults[0].IsNil() {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("variable %q expression not found", varName),
			}
			continue
		}

		// Access Expression.Value field via reflection
		exprValue := exprResults[0].Elem()
		valueField := exprValue.FieldByName("Value")
		if !valueField.IsValid() {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("Expression.Value field not found"),
			}
			continue
		}

		// Return value directly (preserves original type: string, int, bool, map, slice)
		results[i] = decorator.ResolveResult{
			Value:  valueField.Interface(),
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
