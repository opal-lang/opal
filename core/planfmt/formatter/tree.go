package formatter

import (
	"fmt"
	"io"
	"strings"

	"github.com/opal-lang/opal/core/planfmt"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Colorize wraps text in ANSI color codes if color is enabled
func Colorize(text, color string, useColor bool) string {
	if !useColor {
		return text
	}
	return color + text + ColorReset
}

// FormatTree renders a plan as a tree structure to the given writer.
// This is used for --dry-run output to show the execution plan visually.
func FormatTree(w io.Writer, plan *planfmt.Plan, useColor bool) {
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
		renderTreeStep(w, step, isLast, useColor)
	}
}

// renderTreeStep renders a single step with tree characters
func renderTreeStep(w io.Writer, step planfmt.Step, isLast, useColor bool) {
	// Choose tree character
	var prefix string
	if isLast {
		prefix = "└─ "
	} else {
		prefix = "├─ "
	}

	// Handle TryNode specially to render its nested blocks
	if try, ok := step.Tree.(*planfmt.TryNode); ok {
		renderTryBlock(w, try, prefix, "", isLast, useColor)
		return
	}

	// Render the execution tree
	treeStr := renderExecutionNode(step.Tree, useColor)
	_, _ = fmt.Fprintf(w, "%s%s\n", prefix, treeStr)

	// Render nested blocks if this is a CommandNode with a Block
	if cmd, ok := step.Tree.(*planfmt.CommandNode); ok && len(cmd.Block) > 0 {
		renderNestedBlock(w, cmd.Block, "   ", useColor)
	}
}

// renderTryBlock renders a try/catch/finally block with proper indentation
func renderTryBlock(w io.Writer, try *planfmt.TryNode, prefix, indent string, isLast, useColor bool) {
	// Render try {
	tryPrefix := prefix
	if indent != "" {
		if isLast {
			tryPrefix = indent + "└─ "
		} else {
			tryPrefix = indent + "├─ "
		}
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", tryPrefix, Colorize("try {", ColorYellow, useColor))

	// Render try block contents
	tryIndent := indent + "│  "
	if isLast {
		tryIndent = indent + "   "
	}
	renderNestedBlock(w, try.TryBlock, tryIndent, useColor)

	// Render try closing brace
	_, _ = fmt.Fprintf(w, "%s%s\n", tryIndent, Colorize("}", ColorYellow, useColor))

	// Render catch block if present
	if len(try.CatchBlock) > 0 {
		_, _ = fmt.Fprintf(w, "%s%s\n", tryIndent, Colorize("catch {", ColorYellow, useColor))
		renderNestedBlock(w, try.CatchBlock, tryIndent, useColor)
		_, _ = fmt.Fprintf(w, "%s%s\n", tryIndent, Colorize("}", ColorYellow, useColor))
	}

	// Render finally block if present
	if len(try.FinallyBlock) > 0 {
		_, _ = fmt.Fprintf(w, "%s%s\n", tryIndent, Colorize("finally {", ColorYellow, useColor))
		renderNestedBlock(w, try.FinallyBlock, tryIndent, useColor)
		_, _ = fmt.Fprintf(w, "%s%s\n", tryIndent, Colorize("}", ColorYellow, useColor))
	}
}

// renderNestedBlock renders nested steps with proper indentation
func renderNestedBlock(w io.Writer, steps []planfmt.Step, indent string, useColor bool) {
	for i, step := range steps {
		isLast := i == len(steps)-1
		var prefix string
		if isLast {
			prefix = indent + "└─ "
		} else {
			prefix = indent + "├─ "
		}

		treeStr := renderExecutionNode(step.Tree, useColor)
		_, _ = fmt.Fprintf(w, "%s%s\n", prefix, treeStr)

		// Recursively render nested blocks
		if cmd, ok := step.Tree.(*planfmt.CommandNode); ok && len(cmd.Block) > 0 {
			newIndent := indent + "   "
			renderNestedBlock(w, cmd.Block, newIndent, useColor)
		}
	}
}

// renderExecutionNode renders an execution node to a string
func renderExecutionNode(node planfmt.ExecutionNode, useColor bool) string {
	switch n := node.(type) {
	case *planfmt.CommandNode:
		return renderCommandNode(n, useColor)
	case *planfmt.PipelineNode:
		return renderPipelineNode(n, useColor)
	case *planfmt.AndNode:
		return renderAndNode(n, useColor)
	case *planfmt.OrNode:
		return renderOrNode(n, useColor)
	case *planfmt.SequenceNode:
		return renderSequenceNode(n, useColor)
	case *planfmt.TryNode:
		return renderTryNode(n, useColor)
	default:
		return fmt.Sprintf("(unknown node type: %T)", node)
	}
}

// renderTryNode renders a try/catch/finally node for inline display
func renderTryNode(try *planfmt.TryNode, useColor bool) string {
	return Colorize("try { ... }", ColorYellow, useColor)
}

// renderCommandNode renders a single command
func renderCommandNode(cmd *planfmt.CommandNode, useColor bool) string {
	decorator := Colorize(cmd.Decorator, ColorBlue, useColor)
	commandStr := getCommandString(cmd)
	return fmt.Sprintf("%s %s", decorator, commandStr)
}

// renderPipelineNode renders a pipeline (cmd1 | cmd2 | cmd3)
func renderPipelineNode(pipe *planfmt.PipelineNode, useColor bool) string {
	var parts []string
	for _, elem := range pipe.Commands {
		// Pipeline elements can be CommandNode or RedirectNode
		parts = append(parts, renderExecutionNode(elem, useColor))
	}
	return strings.Join(parts, " | ")
}

// renderAndNode renders an AND node (left && right)
func renderAndNode(and *planfmt.AndNode, useColor bool) string {
	left := renderExecutionNode(and.Left, useColor)
	right := renderExecutionNode(and.Right, useColor)
	return fmt.Sprintf("%s && %s", left, right)
}

// renderOrNode renders an OR node (left || right)
func renderOrNode(or *planfmt.OrNode, useColor bool) string {
	left := renderExecutionNode(or.Left, useColor)
	right := renderExecutionNode(or.Right, useColor)
	return fmt.Sprintf("%s || %s", left, right)
}

// renderSequenceNode renders a sequence node (node1 ; node2 ; node3)
func renderSequenceNode(seq *planfmt.SequenceNode, useColor bool) string {
	var parts []string
	for _, node := range seq.Nodes {
		parts = append(parts, renderExecutionNode(node, useColor))
	}
	return strings.Join(parts, " ; ")
}

// getCommandString extracts the command string from a CommandNode for display
func getCommandString(cmd *planfmt.CommandNode) string {
	// For @shell decorator, look for "command" arg
	for _, arg := range cmd.Args {
		if arg.Key == "command" && arg.Val.Kind == planfmt.ValueString {
			return arg.Val.Str
		}
	}
	// Fallback: show all args with proper value formatting
	var parts []string
	for _, arg := range cmd.Args {
		parts = append(parts, fmt.Sprintf("%s=%s", arg.Key, formatValue(&arg.Val)))
	}
	return strings.Join(parts, " ")
}
