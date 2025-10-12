package decorators

import (
	"github.com/aledsdavies/opal/core/types"
)

func init() {
	schema := types.NewSchema("parallel", types.KindExecution).
		Description("Execute tasks in parallel").
		Param("maxConcurrency", types.TypeInt).
		Description("Maximum number of concurrent tasks").
		Default(0). // 0 = unlimited
		Done().
		RequiresBlock().
		Build()

	handler := func(ctx types.Context, args types.Args) error {
		// Implementation would go here
		return nil
	}

	if err := types.Global().RegisterExecutionWithSchema(schema, handler); err != nil {
		panic(err)
	}
}
