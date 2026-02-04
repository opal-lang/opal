package planfmt

import (
	"io"

	"github.com/opal-lang/opal/core/decorator"
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
	sdkSteps := make([]sdk.Step, 0, len(planSteps))
	for _, planStep := range planSteps {
		appendSDKSteps(&sdkSteps, planStep, registry)
	}
	return sdkSteps
}

// ToSDKStep converts a single planfmt.Step to sdk.Step.
func ToSDKStep(planStep Step) sdk.Step {
	steps := ToSDKStepsWithRegistry([]Step{planStep}, types.Global())
	invariant.Precondition(len(steps) == 1, "expected 1 sdk step, got %d", len(steps))
	return steps[0]
}

func toSDKStepWithRegistry(planStep Step, registry *types.Registry) sdk.Step {
	return sdk.Step{
		ID:   planStep.ID,
		Tree: toSDKTreeWithRegistry(planStep.Tree, registry),
	}
}

func appendSDKSteps(dst *[]sdk.Step, planStep Step, registry *types.Registry) {
	if logic, ok := planStep.Tree.(*LogicNode); ok {
		for _, blockStep := range logic.Block {
			appendSDKSteps(dst, blockStep, registry)
		}
		return
	}
	*dst = append(*dst, toSDKStepWithRegistry(planStep, registry))
}

// toSDKTreeWithRegistry converts planfmt.ExecutionNode to sdk.TreeNode.
// This recursively converts the entire tree structure.
func toSDKTreeWithRegistry(node ExecutionNode, registry *types.Registry) sdk.TreeNode {
	switch n := node.(type) {
	case *CommandNode:
		return &sdk.CommandNode{
			Name:        n.Decorator,
			TransportID: n.TransportID,
			Args:        ToSDKArgs(n.Args),
			Block:       ToSDKStepsWithRegistry(n.Block, registry), // Recursive for nested steps
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
	case *TryNode:
		return &sdk.TryNode{
			TryBlock:     ToSDKStepsWithRegistry(n.TryBlock, registry),
			CatchBlock:   ToSDKStepsWithRegistry(n.CatchBlock, registry),
			FinallyBlock: ToSDKStepsWithRegistry(n.FinallyBlock, registry),
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
		args[arg.Key] = toSDKValue(arg.Val)
	}
	return args
}

func toSDKValue(val Value) interface{} {
	switch val.Kind {
	case ValueString:
		return val.Str
	case ValueInt:
		return val.Int
	case ValueBool:
		return val.Bool
	case ValuePlaceholder:
		return val.Ref
	case ValueFloat:
		return val.Float
	case ValueDuration:
		return val.Duration
	case ValueArray:
		items := make([]interface{}, len(val.Array))
		for i, item := range val.Array {
			items[i] = toSDKValue(item)
		}
		return items
	case ValueMap:
		mapped := make(map[string]interface{}, len(val.Map))
		for key, item := range val.Map {
			mapped[key] = toSDKValue(item)
		}
		return mapped
	default:
		return nil
	}
}

// commandNodeToSink converts a CommandNode (redirect target) to a Sink.
// Looks up the decorator in the registry and wraps it as a Sink if it implements IO.
func commandNodeToSink(target *CommandNode, _ *types.Registry) sdk.Sink {
	// Strip @ prefix from decorator name for registry lookup
	decoratorName := target.Decorator
	if decoratorName != "" && decoratorName[0] == '@' {
		decoratorName = decoratorName[1:]
	}

	// Get decorator from new registry
	entry, exists := decorator.Global().Lookup(decoratorName)
	invariant.Invariant(exists, "decorator %s not registered (parser should have rejected this)", target.Decorator)

	// Check if decorator implements IO
	ioDecorator, ok := entry.Impl.(decorator.IO)
	invariant.Invariant(ok, "decorator %s does not implement IO (parser should have rejected this)", target.Decorator)

	args := ToSDKArgs(target.Args)

	// If decorator implements IOFactory, create a new instance with params
	if factory, ok := ioDecorator.(decorator.IOFactory); ok {
		ioDecorator = factory.WithParams(args)
	}

	// Return an adapter that wraps the IO decorator as a Sink
	return &ioSinkAdapter{
		io:   ioDecorator,
		args: args,
	}
}

// ioSinkAdapter wraps a decorator.IO as an sdk.Sink.
// This bridges the new decorator IO interface to the SDK sink interface.
type ioSinkAdapter struct {
	io   decorator.IO
	args map[string]interface{}
}

func (a *ioSinkAdapter) Caps() sdk.SinkCaps {
	caps := a.io.IOCaps()
	return sdk.SinkCaps{
		Overwrite:      caps.Write,
		Append:         caps.Append,
		Atomic:         caps.Atomic,
		ConcurrentSafe: false,
	}
}

func (a *ioSinkAdapter) Open(ctx sdk.ExecutionContext, mode sdk.RedirectMode, meta map[string]any) (io.WriteCloser, error) {
	// Create minimal ExecContext for the IO decorator
	execCtx := decorator.ExecContext{
		Context: ctx.Context(),
	}

	// Determine append mode from redirect mode
	appendMode := mode == sdk.RedirectAppend

	// Open for writing
	return a.io.OpenWrite(execCtx, appendMode)
}

func (a *ioSinkAdapter) Identity() (string, string) {
	// Get path from args if available
	if path, ok := a.args["command"].(string); ok {
		return "io", path
	}
	return "io", ""
}
