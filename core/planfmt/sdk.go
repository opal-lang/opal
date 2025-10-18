package planfmt

import (
	"github.com/aledsdavies/opal/core/sdk"
)

// ToSDKSteps converts planfmt.Step slice to sdk.Step slice.
// This is the boundary between binary format (planfmt) and execution model (sdk).
//
// The executor only sees SDK types - it has no knowledge of planfmt.
// All value interpolation is already done by the planner, so steps contain
// actual values (not placeholders), except for value decorator results which
// use DisplayID placeholders that get scrubbed during output.
func ToSDKSteps(planSteps []Step) []sdk.Step {
	sdkSteps := make([]sdk.Step, len(planSteps))
	for i, planStep := range planSteps {
		sdkSteps[i] = ToSDKStep(planStep)
	}
	return sdkSteps
}

// ToSDKStep converts a single planfmt.Step to sdk.Step.
func ToSDKStep(planStep Step) sdk.Step {
	return sdk.Step{
		ID:       planStep.ID,
		Commands: ToSDKCommands(planStep.Commands),
	}
}

// ToSDKCommands converts planfmt.Command slice to sdk.Command slice.
func ToSDKCommands(planCmds []Command) []sdk.Command {
	sdkCmds := make([]sdk.Command, len(planCmds))
	for i, planCmd := range planCmds {
		sdkCmds[i] = sdk.Command{
			Name:     planCmd.Decorator,
			Args:     ToSDKArgs(planCmd.Args),
			Block:    ToSDKSteps(planCmd.Block), // Recursive
			Operator: planCmd.Operator,
		}
	}
	return sdkCmds
}

// ToSDKArgs converts []planfmt.Arg to map[string]interface{}.
// This provides a cleaner interface for decorators to access arguments.
func ToSDKArgs(planArgs []Arg) map[string]interface{} {
	args := make(map[string]interface{})
	for _, arg := range planArgs {
		switch arg.Val.Kind {
		case ValueString:
			args[arg.Key] = arg.Val.Str
		case ValueInt:
			args[arg.Key] = arg.Val.Int
		case ValueBool:
			args[arg.Key] = arg.Val.Bool
			// TODO: Handle other value types (float, duration, etc.) as needed
		}
	}
	return args
}
