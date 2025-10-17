package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/aledsdavies/opal/core/planfmt"
)

// DisplayPlan renders a plan as a tree structure
func DisplayPlan(w io.Writer, plan *planfmt.Plan, useColor bool) {
	// Print target name
	_, _ = fmt.Fprintf(w, "%s:\n", plan.Target)

	// Handle empty plan
	if len(plan.Steps) == 0 {
		_, _ = fmt.Fprintf(w, "(no steps)\n")
		return
	}

	// Render each step
	for i, step := range plan.Steps {
		isLast := i == len(plan.Steps)-1
		renderStep(w, step, isLast, useColor)
	}
}

// renderStep renders a single step with tree characters
func renderStep(w io.Writer, step planfmt.Step, isLast bool, useColor bool) {
	// Choose tree character
	var prefix string
	if isLast {
		prefix = "└─ "
	} else {
		prefix = "├─ "
	}

	// For steps with operators, show decorator once then commands with operators
	if len(step.Commands) > 1 {
		// Get decorator from first command (all should be same in a step)
		decorator := Colorize(step.Commands[0].Decorator, ColorBlue, useColor)

		// Build command parts with operators
		var cmdParts []string
		for i, cmd := range step.Commands {
			cmdParts = append(cmdParts, getCommandString(cmd))
			if i < len(step.Commands)-1 {
				cmdParts = append(cmdParts, cmd.Operator)
			}
		}

		fullCommand := decorator + " " + strings.Join(cmdParts, " ")
		_, _ = fmt.Fprintf(w, "%s%s\n", prefix, fullCommand)
		return
	}

	// Single command: show decorator + command
	cmd := step.Commands[0]
	decorator := Colorize(cmd.Decorator, ColorBlue, useColor)
	commandStr := getCommandString(cmd)
	_, _ = fmt.Fprintf(w, "%s%s %s\n", prefix, decorator, commandStr)
}

// getCommandString extracts the command string from a Command
func getCommandString(cmd planfmt.Command) string {
	// For @shell decorator, look for "command" arg
	for _, arg := range cmd.Args {
		if arg.Key == "command" && arg.Val.Kind == planfmt.ValueString {
			return arg.Val.Str
		}
	}
	// Fallback: show all args
	var parts []string
	for _, arg := range cmd.Args {
		parts = append(parts, fmt.Sprintf("%s=%v", arg.Key, arg.Val.Str))
	}
	return strings.Join(parts, " ")
}
