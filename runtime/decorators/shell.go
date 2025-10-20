package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/sdk"
	"github.com/aledsdavies/opal/core/sdk/executor"
	"github.com/aledsdavies/opal/core/types"
)

func init() {
	// Register the @shell decorator with schema
	schema := types.NewSchema("shell", types.KindExecution).
		Description("Execute shell commands").
		Param("command", types.TypeString).
		Description("Shell command to execute").
		Required().
		Done().
		WithIO(types.AcceptsStdin, types.ProducesStdout). // No ScrubByDefault = bash-compatible
		Build()

	if err := types.Global().RegisterSDKHandlerWithSchema(schema, shellHandler); err != nil {
		panic(fmt.Sprintf("failed to register @shell decorator: %v", err))
	}
}

// shellHandler implements the @shell decorator using SDK types.
// This is a leaf decorator - it doesn't use the block parameter.
//
// CRITICAL: Uses context workdir and environ, NOT os globals.
// This ensures isolation for @parallel, @ssh, @docker, etc.
func shellHandler(ctx sdk.ExecutionContext, block []sdk.Step) (int, error) {
	// Leaf decorator - block should be empty
	if len(block) > 0 {
		return 127, fmt.Errorf("@shell does not accept a block")
	}

	// Get command string from context args
	cmdStr := ctx.ArgString("command")
	if cmdStr == "" {
		return 127, fmt.Errorf("@shell requires command argument")
	}

	// Create command using SDK (automatically routes through scrubber)
	cmd := executor.BashContext(ctx.Context(), cmdStr)

	// CRITICAL: Use context state, not os state
	// This ensures isolation for @parallel, @ssh, @docker, etc.
	cmd.SetDir(ctx.Workdir())

	// For environment, we need to convert map to slice
	// Use AppendEnv for now (adds to existing env)
	// TODO: Consider if we need full replacement instead
	cmd.AppendEnv(ctx.Environ())

	// Execute and return exit code
	return cmd.Run()
}
