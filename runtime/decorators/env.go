package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/types"
)

// EnvDecorator implements the @env value decorator.
// @env is transport-aware - it reads from the session's environment.
type EnvDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *EnvDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("env").
		Summary("Access environment variables from the current session").
		Roles(decorator.RoleProvider).
		PrimaryParam("property", types.TypeString, "Environment variable name", "HOME", "PATH", "USER").
		Param("default", types.TypeString, "Default value if environment variable is not set", "", "/home/user", "us-east-1").
		Returns(types.TypeString, "Value of the environment variable").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

// Resolve implements the Value interface.
func (d *EnvDecorator) Resolve(ctx decorator.ValueEvalContext, call decorator.ValueCall) (any, error) {
	invariant.NotNil(ctx.Session, "ctx.Session")

	// Get environment variable name from primary parameter
	if call.Primary == nil {
		return nil, fmt.Errorf("@env requires an environment variable name")
	}

	envVar := *call.Primary

	// Read from session environment (transport-aware)
	env := ctx.Session.Env()
	value, exists := env[envVar]

	if !exists {
		// Check for default parameter
		if defaultVal, hasDefault := call.Params["default"]; hasDefault {
			return defaultVal, nil
		}
		return nil, fmt.Errorf("environment variable %q not found", envVar)
	}

	return value, nil
}

// Register @env decorator with the global registry
func init() {
	if err := decorator.Register("env", &EnvDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @env decorator: %v", err))
	}
}
