package planfmt

import (
	"context"
	"io"
	"time"

	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
	"github.com/opal-lang/opal/core/types"
)

// ToSDKSteps converts planfmt.Step slice to sdk.Step slice.
// This is the boundary between binary format (planfmt) and execution model (sdk).
//
// The executor only sees SDK types - it has no knowledge of planfmt.
// All value interpolation is already done by the planner, so steps contain
// actual values (not placeholders), except for value decorator results which
// use DisplayID placeholders that get scrubbed during output.
func ToSDKSteps(planSteps []Step) []sdk.Step {
	return ToSDKStepsWithRegistry(planSteps, types.Global())
}

// ToSDKStepsWithRegistry converts planfmt.Step slice to sdk.Step slice using the given registry.
func ToSDKStepsWithRegistry(planSteps []Step, registry *types.Registry) []sdk.Step {
	sdkSteps := make([]sdk.Step, len(planSteps))
	for i, planStep := range planSteps {
		sdkSteps[i] = toSDKStepWithRegistry(planStep, registry)
	}
	return sdkSteps
}

// ToSDKStep converts a single planfmt.Step to sdk.Step.
func ToSDKStep(planStep Step) sdk.Step {
	return toSDKStepWithRegistry(planStep, types.Global())
}

func toSDKStepWithRegistry(planStep Step, registry *types.Registry) sdk.Step {
	return sdk.Step{
		ID:   planStep.ID,
		Tree: toSDKTreeWithRegistry(planStep.Tree, registry),
	}
}

// toSDKTreeWithRegistry converts planfmt.ExecutionNode to sdk.TreeNode.
// This recursively converts the entire tree structure.
func toSDKTreeWithRegistry(node ExecutionNode, registry *types.Registry) sdk.TreeNode {
	switch n := node.(type) {
	case *CommandNode:
		return &sdk.CommandNode{
			Name:  n.Decorator,
			Args:  ToSDKArgs(n.Args),
			Block: ToSDKStepsWithRegistry(n.Block, registry), // Recursive for nested steps
		}
	case *PipelineNode:
		commands := make([]sdk.TreeNode, len(n.Commands))
		for i, elem := range n.Commands {
			// Invariant: Pipeline elements must be CommandNode or RedirectNode
			// (bash allows: cmd1 | cmd2 > file, but not: cmd1 | (cmd2 && cmd3))
			switch elem.(type) {
			case *CommandNode, *RedirectNode:
				// Recursively convert to SDK TreeNode
				commands[i] = toSDKTreeWithRegistry(elem, registry)
			default:
				invariant.Invariant(false, "invalid pipeline element type %T (only CommandNode and RedirectNode allowed)", elem)
			}
		}
		return &sdk.PipelineNode{Commands: commands}
	case *AndNode:
		return &sdk.AndNode{
			Left:  toSDKTreeWithRegistry(n.Left, registry),
			Right: toSDKTreeWithRegistry(n.Right, registry),
		}
	case *OrNode:
		return &sdk.OrNode{
			Left:  toSDKTreeWithRegistry(n.Left, registry),
			Right: toSDKTreeWithRegistry(n.Right, registry),
		}
	case *SequenceNode:
		nodes := make([]sdk.TreeNode, len(n.Nodes))
		for i, child := range n.Nodes {
			nodes[i] = toSDKTreeWithRegistry(child, registry)
		}
		return &sdk.SequenceNode{Nodes: nodes}
	case *RedirectNode:
		// Convert Target CommandNode to Sink by evaluating the decorator
		sink := commandNodeToSink(&n.Target, registry)

		return &sdk.RedirectNode{
			Source: toSDKTreeWithRegistry(n.Source, registry),
			Sink:   sink,
			Mode:   sdk.RedirectMode(n.Mode),
		}
	default:
		invariant.Invariant(false, "unknown ExecutionNode type: %T", node)
		return nil // unreachable
	}
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

// commandNodeToSink converts a CommandNode (redirect target) to a Sink.
// Looks up the decorator in the registry and calls AsSink() if it implements SinkProvider.
func commandNodeToSink(target *CommandNode, registry *types.Registry) sdk.Sink {
	// Strip @ prefix from decorator name for registry lookup
	decoratorName := target.Decorator
	if decoratorName != "" && decoratorName[0] == '@' {
		decoratorName = decoratorName[1:]
	}

	// Get decorator handler from registry
	handler, _, exists := registry.GetSDKHandler(decoratorName)
	invariant.Invariant(exists, "decorator %s not registered (parser should have rejected this)", target.Decorator)

	// Check if decorator implements SinkProvider
	sinkProvider, ok := handler.(sdk.SinkProvider)
	invariant.Invariant(ok, "decorator %s does not implement SinkProvider (parser should have rejected this)", target.Decorator)

	// Create minimal execution context with args
	ctx := &minimalContext{args: ToSDKArgs(target.Args)}

	// Call AsSink() on the decorator instance
	return sinkProvider.AsSink(ctx)
}

// minimalContext is a minimal ExecutionContext for evaluating redirect targets.
// Redirect targets only need args - no stdin/stdout/environ/etc.
type minimalContext struct {
	args map[string]interface{}
}

func (m *minimalContext) ExecuteBlock(steps []sdk.Step) (int, error) {
	invariant.Invariant(false, "redirect target tried to execute block")
	return 0, nil // unreachable
}
func (m *minimalContext) Context() context.Context { return context.Background() }
func (m *minimalContext) ArgString(key string) string {
	if v, ok := m.args[key].(string); ok {
		return v
	}
	return ""
}

func (m *minimalContext) ArgInt(key string) int64 {
	if v, ok := m.args[key].(int64); ok {
		return v
	}
	return 0
}

func (m *minimalContext) ArgBool(key string) bool {
	if v, ok := m.args[key].(bool); ok {
		return v
	}
	return false
}
func (m *minimalContext) ArgDuration(key string) time.Duration { return 0 }
func (m *minimalContext) Args() map[string]interface{}         { return m.args }
func (m *minimalContext) Environ() map[string]string           { return nil }
func (m *minimalContext) Workdir() string                      { return "" }
func (m *minimalContext) WithContext(ctx context.Context) sdk.ExecutionContext {
	return m
}

func (m *minimalContext) WithEnviron(env map[string]string) sdk.ExecutionContext {
	return m
}
func (m *minimalContext) WithWorkdir(dir string) sdk.ExecutionContext { return m }
func (m *minimalContext) Stdin() io.Reader                            { return nil }
func (m *minimalContext) StdoutPipe() io.Writer                       { return nil }
func (m *minimalContext) Clone(args map[string]interface{}, stdin io.Reader, stdoutPipe io.Writer) sdk.ExecutionContext {
	return &minimalContext{args: args}
}
func (m *minimalContext) Transport() interface{} { return nil }
