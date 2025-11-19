package decorators

import (
	"fmt"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/types"
)

// EnvDecorator implements the @env value decorator.
// @env is transport-aware - it reads from the session's environment.
type EnvDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *EnvDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("env").
		Summary("Access environment variables from the current session").
		Roles(decorator.RoleProvider).
		PrimaryParamString("property", "Environment variable name").
		Examples("HOME", "PATH", "USER").
		Done().
		ParamString("default", "Default value if environment variable is not set").
		Examples("", "/home/user", "us-east-1").
		Done().
		Returns(types.TypeString, "Value of the environment variable").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

// Resolve implements the Value interface with batch support.
// @env batches all env var lookups into a single Session.Env() call.
func (d *EnvDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	invariant.NotNil(ctx.Session, "ctx.Session")

	// Batch optimization: Get environment once for all calls
	env := ctx.Session.Env()

	results := make([]decorator.ResolveResult, len(calls))

	for i, call := range calls {
		// Get environment variable name from primary parameter
		if call.Primary == nil {
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: "@env.<unknown>",
				Error:  fmt.Errorf("@env requires an environment variable name"),
			}
			continue
		}

		envVar := *call.Primary

		// Look up in batched environment
		value, exists := env[envVar]

		if !exists {
			// Check for default parameter
			if defaultVal, hasDefault := call.Params["default"]; hasDefault {
				results[i] = decorator.ResolveResult{
					Value:  defaultVal,
					Origin: fmt.Sprintf("@env.%s", envVar),
					Error:  nil,
				}
				continue
			}
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: fmt.Sprintf("@env.%s", envVar),
				Error:  fmt.Errorf("environment variable %q not found", envVar),
			}
			continue
		}

		results[i] = decorator.ResolveResult{
			Value:  value,
			Origin: fmt.Sprintf("@env.%s", envVar),
			Error:  nil,
		}
	}

	return results, nil
}

// Register @env decorator with the global registry
func init() {
	if err := decorator.Register("env", &EnvDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @env decorator: %v", err))
	}
}
