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
			err, ok := lookupResults[1].Interface().(error)
			if !ok {
				results[i] = decorator.ResolveResult{
					Value:  nil,
					Origin: fmt.Sprintf("var.%s", varName),
					Error:  fmt.Errorf("lookupVariable returned unexpected error type: %T", lookupResults[1].Interface()),
				}
				continue
			}
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  err,
			}
			continue
		}

		// Extract exprID with type assertion
		exprID, ok := lookupResults[0].Interface().(string)
		if !ok {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("lookupVariable returned non-string exprID: %T", lookupResults[0].Interface()),
			}
			continue
		}

		// Record reference to authorize this site before accessing
		// Use a default parameter name for decorator resolution
		paramName := "value"

		recordRefMethod := vaultValue.MethodByName("RecordReference")
		if !recordRefMethod.IsValid() {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("Vault.RecordReference method not found"),
			}
			continue
		}

		recordResults := recordRefMethod.Call([]reflect.Value{
			reflect.ValueOf(exprID),
			reflect.ValueOf(paramName),
		})

		// Check return signature first
		if len(recordResults) != 1 {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("unexpected RecordReference return values: got %d, want 1", len(recordResults)),
			}
			continue
		}

		// Check if RecordReference returned an error
		if !recordResults[0].IsNil() {
			err, ok := recordResults[0].Interface().(error)
			if !ok {
				results[i] = decorator.ResolveResult{
					Value:  nil,
					Origin: fmt.Sprintf("var.%s", varName),
					Error:  fmt.Errorf("recordReference returned unexpected type: %T", recordResults[0].Interface()),
				}
				continue
			}
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  err,
			}
			continue
		}

		// Call Access(exprID, paramName) -> (any, error)
		accessMethod := vaultValue.MethodByName("Access")
		if !accessMethod.IsValid() {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("Vault.Access method not found"),
			}
			continue
		}

		accessResults := accessMethod.Call([]reflect.Value{
			reflect.ValueOf(exprID),
			reflect.ValueOf(paramName),
		})
		if len(accessResults) != 2 {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  fmt.Errorf("unexpected Access return values"),
			}
			continue
		}

		// Check for error
		if !accessResults[1].IsNil() {
			err, ok := accessResults[1].Interface().(error)
			if !ok {
				results[i] = decorator.ResolveResult{
					Value:  nil,
					Origin: fmt.Sprintf("var.%s", varName),
					Error:  fmt.Errorf("access returned unexpected error type: %T", accessResults[1].Interface()),
				}
				continue
			}
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("var.%s", varName),
				Error:  err,
			}
			continue
		}

		// Return value directly (preserves original type: string, int, bool, map, slice)
		results[i] = decorator.ResolveResult{
			Value:  accessResults[0].Interface(),
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
