// Package formatter provides human-readable formatting for plans.
// This includes text output, diffs, and tree displays.
package formatter

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/opal/core/planfmt"
)

// Format returns a human-readable text representation of the plan.
// Used for contract files and inspection.
//
// Format:
//
//	target: <name>
//	step 1: <formatted step>
//	step 2: <formatted step>
func Format(plan *planfmt.Plan) string {
	var b strings.Builder

	// Write target
	fmt.Fprintf(&b, "target: %s\n", plan.Target)

	// Write each step
	for i, step := range plan.Steps {
		fmt.Fprintf(&b, "step %d: %s\n", i+1, FormatStep(&step))
	}

	return b.String()
}

// FormatStep returns a single step as text by formatting its execution tree.
func FormatStep(step *planfmt.Step) string {
	return formatExecutionNode(step.Tree)
}

// formatExecutionNode formats an execution node to text
func formatExecutionNode(node planfmt.ExecutionNode) string {
	switch n := node.(type) {
	case *planfmt.CommandNode:
		return formatCommandNode(n)
	case *planfmt.PipelineNode:
		var parts []string
		for _, cmd := range n.Commands {
			parts = append(parts, formatCommandNode(&cmd))
		}
		return strings.Join(parts, " | ")
	case *planfmt.AndNode:
		return fmt.Sprintf("%s && %s", formatExecutionNode(n.Left), formatExecutionNode(n.Right))
	case *planfmt.OrNode:
		return fmt.Sprintf("%s || %s", formatExecutionNode(n.Left), formatExecutionNode(n.Right))
	case *planfmt.SequenceNode:
		var parts []string
		for _, child := range n.Nodes {
			parts = append(parts, formatExecutionNode(child))
		}
		return strings.Join(parts, " ; ")
	default:
		return fmt.Sprintf("(unknown: %T)", node)
	}
}

// formatCommandNode formats a single command node
func formatCommandNode(cmd *planfmt.CommandNode) string {
	// Special case: @shell with single "command" arg - show command directly
	if cmd.Decorator == "@shell" && len(cmd.Args) == 1 && cmd.Args[0].Key == "command" {
		return fmt.Sprintf("@shell %s", cmd.Args[0].Val.Str)
	}

	// General case: decorator with args
	if len(cmd.Args) == 0 {
		return cmd.Decorator
	}

	// Format args as key=value pairs
	var argParts []string
	for _, arg := range cmd.Args {
		argParts = append(argParts, fmt.Sprintf("%s=%s", arg.Key, formatValue(&arg.Val)))
	}

	return fmt.Sprintf("%s(%s)", cmd.Decorator, strings.Join(argParts, ", "))
}

// formatValue formats a value based on its kind
func formatValue(val *planfmt.Value) string {
	switch val.Kind {
	case planfmt.ValueString:
		return val.Str
	case planfmt.ValueInt:
		return fmt.Sprintf("%d", val.Int)
	case planfmt.ValueBool:
		return fmt.Sprintf("%t", val.Bool)
	case planfmt.ValuePlaceholder:
		return fmt.Sprintf("$%d", val.Ref)
	default:
		return "?"
	}
}
