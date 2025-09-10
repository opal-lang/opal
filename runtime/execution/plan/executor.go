package plan

import (
	"fmt"
	"io"
	"strings"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/runtime/ir"
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
func (e *PlanExecutor) ExecuteNode(ctx *ir.Ctx, node ir.Node) ir.CommandResult {
	if ctx.Debug {
		fmt.Fprintf(e.output, "[DEBUG PlanExecutor] ExecuteNode called with node type: %T\n", node)
	}

	switch n := node.(type) {
	case ir.CommandSeq:
		if ctx.Debug {
			fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Processing CommandSeq with %d steps\n", len(n.Steps))
		}
		return e.executeCommandSeq(ctx, n)
	case ir.Wrapper:
		if ctx.Debug {
			fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Processing Wrapper: %s\n", n.Kind)
		}
		return e.executeWrapper(ctx, n)
	case ir.Pattern:
		if ctx.Debug {
			fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Processing Pattern: %s\n", n.Kind)
		}
		return e.executePattern(ctx, n)
	default:
		if ctx.Debug {
			fmt.Fprintf(e.output, "[DEBUG PlanExecutor] Unknown node type: %T\n", node)
		}
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Unknown node type: %T", node),
			ExitCode: 1,
		}
	}
}

// executeCommandSeq shows a sequence of command steps
func (e *PlanExecutor) executeCommandSeq(ctx *ir.Ctx, seq ir.CommandSeq) ir.CommandResult {
	fmt.Fprintf(e.output, "📋 Plan: Executing %d command steps:\n", len(seq.Steps))

	for i, step := range seq.Steps {
		fmt.Fprintf(e.output, "  Step %d: ", i+1)
		e.showStep(ctx, step)
	}

	return ir.CommandResult{ExitCode: 0}
}

// executeWrapper shows a block decorator plan
func (e *PlanExecutor) executeWrapper(ctx *ir.Ctx, wrapper ir.Wrapper) ir.CommandResult {
	_, exists := e.registry.GetBlock(wrapper.Kind)
	if !exists {
		fmt.Fprintf(e.output, "❌ Unknown block decorator: @%s\n", wrapper.Kind)
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Block decorator @%s not found", wrapper.Kind),
			ExitCode: 1,
		}
	}

	// Show the wrapper plan
	fmt.Fprintf(e.output, "🔄 @%s", wrapper.Kind)
	if len(wrapper.Params) > 0 {
		fmt.Fprintf(e.output, "(%s)", e.formatParams(wrapper.Params))
	}
	fmt.Fprintf(e.output, " {\n")

	// Show inner content with indentation
	innerResult := e.executeCommandSeq(ctx, wrapper.Inner)

	fmt.Fprintf(e.output, "}\n")

	return innerResult
}

// executePattern shows a pattern decorator plan
func (e *PlanExecutor) executePattern(ctx *ir.Ctx, pattern ir.Pattern) ir.CommandResult {
	_, exists := e.registry.GetPattern(pattern.Kind)
	if !exists {
		fmt.Fprintf(e.output, "❌ Unknown pattern decorator: @%s\n", pattern.Kind)
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Pattern decorator @%s not found", pattern.Kind),
			ExitCode: 1,
		}
	}

	fmt.Fprintf(e.output, "🔀 @%s", pattern.Kind)
	if len(pattern.Params) > 0 {
		fmt.Fprintf(e.output, "(%s)", e.formatParams(pattern.Params))
	}
	fmt.Fprintf(e.output, " {\n")

	for branchName, branchSeq := range pattern.Branches {
		fmt.Fprintf(e.output, "  Branch '%s':\n", branchName)
		e.executeCommandSeq(ctx, branchSeq)
	}

	fmt.Fprintf(e.output, "}\n")

	return ir.CommandResult{ExitCode: 0}
}

// showStep displays a command step in plan format
func (e *PlanExecutor) showStep(ctx *ir.Ctx, step ir.CommandStep) {
	if len(step.Chain) == 0 {
		fmt.Fprintf(e.output, "(empty)\n")
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

	fmt.Fprintf(e.output, "%s\n", strings.Join(parts, " "))
}

// showElement displays a single chain element
func (e *PlanExecutor) showElement(ctx *ir.Ctx, element ir.ChainElement) string {
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
func (e *PlanExecutor) showShellElement(ctx *ir.Ctx, element ir.ChainElement) string {
	if element.Content == nil {
		return "<missing content>"
	}

	// Convert execution context to decorator context for content resolution
	decorCtx := e.toDecoratorContext(ctx)

	// Show plan description with value decorator resolution using ChainElement method
	return element.PlanDescription(decorCtx, e.registry)
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
func (e *PlanExecutor) formatDecoratorArgs(args []decorators.DecoratorParam) string {
	if len(args) == 0 {
		return ""
	}

	var parts []string
	for _, arg := range args {
		if arg.Name != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", arg.Name, arg.Value))
		} else {
			parts = append(parts, fmt.Sprintf("%v", arg.Value))
		}
	}
	return strings.Join(parts, ", ")
}

// toDecoratorContext converts execution context to decorator context
func (e *PlanExecutor) toDecoratorContext(ctx *ir.Ctx) *decorators.Ctx {
	var ui *decorators.UIConfig
	if ctx.UIConfig != nil {
		ui = &decorators.UIConfig{
			ColorMode:   ctx.UIConfig.ColorMode,
			Quiet:       ctx.UIConfig.Quiet,
			Verbose:     ctx.UIConfig.Verbose,
			Interactive: ctx.UIConfig.Interactive,
			AutoConfirm: ctx.UIConfig.AutoConfirm,
			CI:          ctx.UIConfig.CI,
		}
	}

	return &decorators.Ctx{
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
