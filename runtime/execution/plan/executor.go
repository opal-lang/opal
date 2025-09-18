package plan

import (
	"fmt"
	"io"
	"strings"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/ir"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// PlanExecutor handles dry-run/plan execution mode
// This shows what would be executed without actually running commands
type PlanExecutor struct {
	registry *decorators.Registry
	output   io.Writer
}

// NewPlanExecutor creates a new plan executor
func NewPlanExecutor(registry *decorators.Registry, output io.Writer) *PlanExecutor {
	return &PlanExecutor{
		registry: registry,
		output:   output,
	}
}

// ExecuteNode executes an IR node in plan mode (dry-run)
func (e *PlanExecutor) ExecuteNode(ctx *context.Ctx, node ir.Node) context.CommandResult {
	if ctx.Debug {
		_, _ = fmt.Fprintf(e.output, "[DEBUG PlanExecutor] ExecuteNode called with node type: %T\n", node)
	}

	switch n := node.(type) {
	case ir.CommandSeq:
		if ctx.Debug {
			_, _ = fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Processing CommandSeq with %d steps\n", len(n.Steps))
		}
		return e.executeCommandSeq(ctx, n)
	case ir.Wrapper:
		if ctx.Debug {
			_, _ = fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Processing Wrapper: %s\n", n.Kind)
		}
		return e.executeWrapper(ctx, n)
	case ir.Pattern:
		if ctx.Debug {
			_, _ = fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Processing Pattern: %s\n", n.Kind)
		}
		return e.executePattern(ctx, n)
	default:
		if ctx.Debug {
			_, _ = fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Unknown node type: %T\n", node)
		}
		return context.CommandResult{
			Stderr:   fmt.Sprintf("Unknown node type: %T", node),
			ExitCode: 1,
		}
	}
}

// executeCommandSeq shows a sequence of command steps
func (e *PlanExecutor) executeCommandSeq(ctx *context.Ctx, seq ir.CommandSeq) context.CommandResult {
	_, _ = fmt.Fprintf(e.output, "📋 Plan: Executing %d command steps:\n", len(seq.Steps))

	for i, step := range seq.Steps {
		_, _ = fmt.Fprintf(e.output, "  Step %d: ", i+1)
		e.showStep(ctx, step)
	}

	return context.CommandResult{ExitCode: 0}
}

// executeWrapper shows a block decorator plan
func (e *PlanExecutor) executeWrapper(ctx *context.Ctx, wrapper ir.Wrapper) context.CommandResult {
	_, exists := e.registry.GetBlock(wrapper.Kind)
	if !exists {
		_, _ = fmt.Fprintf(e.output, "❌ Unknown block decorator: @%s\n", wrapper.Kind)
		return context.CommandResult{
			Stderr:   fmt.Sprintf("Block decorator @%s not found", wrapper.Kind),
			ExitCode: 1,
		}
	}

	// Show the wrapper plan
	_, _ = fmt.Fprintf(e.output, "🔄 @%s", wrapper.Kind)
	if len(wrapper.Params) > 0 {
		_, _ = fmt.Fprintf(e.output, "(%s)", e.formatParams(wrapper.Params))
	}
	_, _ = fmt.Fprintf(e.output, " {\n")

	// Show inner content with indentation
	innerResult := e.executeCommandSeq(ctx, wrapper.Inner)

	_, _ = fmt.Fprintf(e.output, "}\n")

	return innerResult
}

// executePattern shows a pattern decorator plan
func (e *PlanExecutor) executePattern(ctx *context.Ctx, pattern ir.Pattern) context.CommandResult {
	_, exists := e.registry.GetPattern(pattern.Kind)
	if !exists {
		_, _ = fmt.Fprintf(e.output, "❌ Unknown pattern decorator: @%s\n", pattern.Kind)
		return context.CommandResult{
			Stderr:   fmt.Sprintf("Pattern decorator @%s not found", pattern.Kind),
			ExitCode: 1,
		}
	}

	_, _ = fmt.Fprintf(e.output, "🔀 @%s", pattern.Kind)
	if len(pattern.Params) > 0 {
		_, _ = fmt.Fprintf(e.output, "(%s)", e.formatParams(pattern.Params))
	}
	_, _ = fmt.Fprintf(e.output, " {\n")

	for branchName, branchSeq := range pattern.Branches {
		_, _ = fmt.Fprintf(e.output, "  Branch '%s':\n", branchName)
		e.executeCommandSeq(ctx, branchSeq)
	}

	_, _ = fmt.Fprintf(e.output, "}\n")

	return context.CommandResult{ExitCode: 0}
}

// showStep displays a command step in plan format
func (e *PlanExecutor) showStep(ctx *context.Ctx, step ir.CommandStep) {
	if len(step.Chain) == 0 {
		_, _ = fmt.Fprintf(e.output, "(empty)\n")
		return
	}

	var parts []string
	for i, element := range step.Chain {
		part := e.showElement(ctx, element)

		// Add operator if not the last element
		if i < len(step.Chain)-1 {
			switch element.OpNext {
			case ir.ChainOpAnd:
				part += " &&"
			case ir.ChainOpOr:
				part += " ||"
			case ir.ChainOpPipe:
				part += " |"
			case ir.ChainOpAppend:
				if element.Target != "" {
					part += fmt.Sprintf(" >> %s", element.Target)
				}
			}
		} else if element.OpNext == ir.ChainOpAppend && element.Target != "" {
			part += fmt.Sprintf(" >> %s", element.Target)
		}

		parts = append(parts, part)
	}

	_, _ = fmt.Fprintf(e.output, "%s\n", strings.Join(parts, " "))
}

// showElement displays a single chain element
func (e *PlanExecutor) showElement(ctx *context.Ctx, element ir.ChainElement) string {
	switch element.Kind {
	case ir.ElementKindShell:
		return e.showShellElement(ctx, element)
	case ir.ElementKindAction:
		return fmt.Sprintf("@%s(%s)", element.Name, e.formatDecoratorArgs(element.Args))
	case ir.ElementKindBlock:
		return fmt.Sprintf("@%s(...) { %d inner steps }", element.Name, len(element.InnerSteps))
	default:
		return fmt.Sprintf("<%s>", element.Kind)
	}
}

// showShellElement displays a shell element with structured content
func (e *PlanExecutor) showShellElement(ctx *context.Ctx, element ir.ChainElement) string {
	if element.Content == nil {
		return "<missing content>"
	}

	// Convert execution context to decorator context for content resolution

	// Create a simple description based on element type
	if element.Content != nil {
		// For shell elements with structured content, show resolved content
		var result strings.Builder
		for _, part := range element.Content.Parts {
			switch part.Kind {
			case ir.PartKindLiteral:
				result.WriteString(part.Text)
			case ir.PartKindDecorator:
				result.WriteString("@" + part.DecoratorName + "(...)")
			}
		}
		return result.String()
	}

	// For action/block elements, show name with args
	if element.Name != "" {
		return fmt.Sprintf("@%s(...)", element.Name)
	}

	return "shell command"
}

// formatParams formats wrapper parameters for display
func (e *PlanExecutor) formatParams(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for name, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%v", name, value))
	}
	return strings.Join(parts, ", ")
}

// formatDecoratorArgs formats decorator arguments for display
func (e *PlanExecutor) formatDecoratorArgs(args []decorators.Param) string {
	if len(args) == 0 {
		return ""
	}

	var parts []string
	for _, arg := range args {
		if arg.GetName() != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", arg.GetName(), arg.GetValue()))
		} else {
			parts = append(parts, fmt.Sprintf("%v", arg.GetValue()))
		}
	}
	return strings.Join(parts, ", ")
}

// toDecoratorContext converts execution context to decorator context
//
//nolint:unused // Helper function for future context conversion needs
func (e *PlanExecutor) toDecoratorContext(ctx *context.Ctx) *context.Ctx {
	var ui *context.UIConfig
	if ctx.UIConfig != nil {
		ui = &context.UIConfig{
			NoColor:     ctx.UIConfig.NoColor,
			AutoConfirm: ctx.UIConfig.AutoConfirm,
		}
	}

	return &context.Ctx{
		Env:     ctx.Env,
		Vars:    ctx.Vars,
		WorkDir: ctx.WorkDir,
		Stdout:  ctx.Stdout,
		Stderr:  ctx.Stderr,
		Stdin:   ctx.Stdin,
		DryRun:  ctx.DryRun,
		Debug:   ctx.Debug,
		UI:      ui,
	}
}
