package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/types"
)

// envDecorator implements the @env value decorator
type envDecorator struct{}

// Handle implements the value decorator handler
func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error) {
	// @env requires a primary property (the env var name)
	if args.Primary == nil {
		return nil, fmt.Errorf("@env requires an environment variable name")
	}

	envVar := (*args.Primary).(string)

	// Look up the environment variable
	value, exists := ctx.Env[envVar]
	if !exists {
		// Check for default parameter
		if args.Params != nil {
			if defaultVal, hasDefault := args.Params["default"]; hasDefault {
				return defaultVal, nil
			}
		}
		return nil, fmt.Errorf("environment variable %q not found", envVar)
	}

	return value, nil
}

// TransportScope implements ValueScopeProvider
// @env is root-only because it reads from the local environment at plan-time
func (e *envDecorator) TransportScope() types.TransportScope {
	return types.ScopeRootOnly
}

func init() {
	// Register the @env decorator with schema
	schema := types.NewSchema("env", types.KindValue).
		Description("Access environment variables").
		PrimaryParam("property", types.TypeString, "Environment variable name").
		Param("default", types.TypeString).
		Description("Default value if environment variable is not set").
		Optional().
		Examples("", "/home/user", "us-east-1").
		Done().
		Returns(types.TypeString, "Value of the environment variable").
		Build()

	// Add examples to primary parameter
	if propParam, ok := schema.Parameters["property"]; ok {
		propParam.Examples = []string{"HOME", "PATH", "USER"}
		schema.Parameters["property"] = propParam
	}

	// Register with the decorator instance (not just the function)
	// This allows the registry to check for ValueScopeProvider interface
	decorator := &envDecorator{}
	if err := types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle); err != nil {
		panic(fmt.Sprintf("failed to register @env decorator: %v", err))
	}
}
