package decorators

import (
	"fmt"
	"reflect"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
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
