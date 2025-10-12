package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/types"
)

func init() {
	// Register the @var decorator with schema
	schema := types.NewSchema("var", types.KindValue).
		Description("Access script variables").
		PrimaryParam("name", types.TypeString, "Variable name").
		Returns(types.TypeString, "Value of the variable").
		Build()

	if err := types.Global().RegisterValueWithSchema(schema, varHandler); err != nil {
		panic(fmt.Sprintf("failed to register @var decorator: %v", err))
	}
}

// varHandler implements the @var decorator
// Accesses variables from the context
func varHandler(ctx types.Context, args types.Args) (types.Value, error) {
	// @var requires a primary property (the variable name)
	if args.Primary == nil {
		return nil, fmt.Errorf("@var requires a variable name")
	}

	varName := (*args.Primary).(string)

	// Look up the variable in the context
	value, exists := ctx.Variables[varName]
	if !exists {
		return nil, fmt.Errorf("variable %q not found", varName)
	}

	return value, nil
}
