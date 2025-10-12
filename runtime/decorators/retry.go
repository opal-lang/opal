package decorators

import (
	"github.com/aledsdavies/opal/core/types"
)

func init() {
	schema := types.NewSchema("retry", types.KindExecution).
		Description("Retry failed operations with exponential backoff").
		Param("times", types.TypeInt).
		Description("Number of retry attempts").
		Default(3).
		Done().
		Param("delay", types.TypeDuration).
		Description("Initial delay between retries").
		Default("1s").
		Examples("1s", "5s", "30s").
		Done().
		Param("backoff", types.TypeString).
		Description("Backoff strategy").
		Default("exponential").
		Examples("exponential", "linear", "constant").
		Done().
		WithBlock(types.BlockOptional).
		Build()

	handler := func(ctx types.Context, args types.Args) error {
		// Implementation would go here
		return nil
	}

	if err := types.Global().RegisterExecutionWithSchema(schema, handler); err != nil {
		panic(err)
	}
}
