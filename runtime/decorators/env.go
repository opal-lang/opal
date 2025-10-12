package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/types"
)

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

	if err := types.Global().RegisterValueWithSchema(schema, envHandler); err != nil {
		panic(fmt.Sprintf("failed to register @env decorator: %v", err))
	}
}

// envHandler implements the @env decorator
// Accesses environment variables from the context
func envHandler(ctx types.Context, args types.Args) (types.Value, error) {
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
