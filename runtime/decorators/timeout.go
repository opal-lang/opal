package decorators

import (
	"github.com/aledsdavies/opal/core/types"
)

func init() {
	schema := types.NewSchema("timeout", types.KindExecution).
		Description("Execute block with timeout constraint").
		Param("duration", types.TypeDuration).
		Description("Maximum execution time").
		Required().
		Examples("30s", "5m", "1h").
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
