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

// FormatStep returns a single step as text.
// Chains multiple commands with their operators.
func FormatStep(step *planfmt.Step) string {
	var parts []string

	for i := range step.Commands {
		cmd := &step.Commands[i]
		formatted := FormatCommand(cmd)

		// Add operator if present (not last command)
		if cmd.Operator != "" {
			formatted += " " + cmd.Operator
		}

		parts = append(parts, formatted)
	}

	return strings.Join(parts, " ")
}

// FormatCommand returns a single command as text.
// Handles different decorator types and argument formatting.
func FormatCommand(cmd *planfmt.Command) string {
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
