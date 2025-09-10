package plan

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/ir"
)

// PlanExpansionResolver implements decorators.ExpansionResolver for plan generation
// This provides resolution capabilities specific to the plan generator
type PlanExpansionResolver struct {
	baseCommandResolver BaseCommandResolver
}

// BaseCommandResolver provides access to command definitions for expansion
type BaseCommandResolver interface {
	GetCommand(name string) (ir.Node, error)
}

// NewPlanExpansionResolver creates a new expansion resolver for plan generation
func NewPlanExpansionResolver(commandResolver BaseCommandResolver) *PlanExpansionResolver {
	return &PlanExpansionResolver{
		baseCommandResolver: commandResolver,
	}
}

// ResolveCommand implements decorators.CommandResolver interface for command lookup
func (r *PlanExpansionResolver) ResolveCommand(name string) (interface{}, error) {
	if r.baseCommandResolver == nil {
		return nil, fmt.Errorf("no command resolver available")
	}
	return r.baseCommandResolver.GetCommand(name)
}

// CommandResolver is the old interface name - kept for backward compatibility
// Use BaseCommandResolver for new code
type CommandResolver = BaseCommandResolver

// Generator handles plan generation from IR nodes
// This converts IR structures to ExecutionPlan for visualization
type Generator struct {
	registry        *decorators.Registry
	commandResolver CommandResolver
}

// NewGenerator creates a new plan generator
func NewGenerator(registry *decorators.Registry) *Generator {
	return &Generator{
		registry:        registry,
		commandResolver: nil, // Can be set later with SetCommandResolver
	}
}

// NewGeneratorWithResolver creates a new plan generator with command resolution capability
func NewGeneratorWithResolver(registry *decorators.Registry, resolver CommandResolver) *Generator {
	return &Generator{
		registry:        registry,
		commandResolver: resolver,
	}
}

// SetCommandResolver sets the command resolver for this generator
func (g *Generator) SetCommandResolver(resolver CommandResolver) {
	g.commandResolver = resolver
}

// GenerateFromIR generates an ExecutionPlan from an IR node
// This is the main entry point called from the engine
func (g *Generator) GenerateFromIR(ctx *ir.Ctx, node ir.Node, commandName string) *plan.ExecutionPlan {
	if ctx.Debug {
		fmt.Printf("[DEBUG PlanGenerator] GenerateFromIR called for command: %s, node type: %T\n", commandName, node)
	}

	executionPlan := &plan.ExecutionPlan{
		Steps:   []plan.ExecutionStep{},
		Edges:   []plan.PlanEdge{},
		Context: make(map[string]interface{}),
		Summary: plan.PlanSummary{},
	}

	// Add context information
	executionPlan.Context["command_name"] = commandName
	if ctx.DryRun {
		executionPlan.Context["mode"] = "dry_run"
	}

	if ctx.Debug {
		fmt.Printf("[DEBUG PlanGenerator] Context set, generating main step from node\n")
	}

	// Generate the main execution step from the IR node
	mainStep := g.generateStep(ctx, node)
	mainStep.ID = "0"
	executionPlan.Steps = append(executionPlan.Steps, mainStep)

	if ctx.Debug {
		fmt.Printf("[DEBUG PlanGenerator] Main step generated: Type=%s, Description=%s, Children=%d\n",
			mainStep.Type, mainStep.Description, len(mainStep.Children))
	}

	// Assign stable IDs to all steps
	executionPlan.AssignStableIDs()

	// Expand command references if resolver is available
	if g.commandResolver != nil {
		if ctx.Debug {
			fmt.Printf("[DEBUG PlanGenerator] Command resolver available, expanding references\n")
		}
		g.expandCommandReferences(ctx, executionPlan)
	} else if ctx.Debug {
		fmt.Printf("[DEBUG PlanGenerator] No command resolver available\n")
	}

	// Generate summary
	executionPlan.Summary = g.generateSummary(executionPlan)

	if ctx.Debug {
		fmt.Printf("[DEBUG PlanGenerator] Plan generation complete: %d steps, %d decorators\n",
			executionPlan.Summary.TotalSteps, len(executionPlan.Summary.DecoratorsUsed))
	}

	return executionPlan
}

// generateStep converts an IR node to an ExecutionStep
func (g *Generator) generateStep(ctx *ir.Ctx, node ir.Node) plan.ExecutionStep {
	switch n := node.(type) {
	case ir.CommandSeq:
		return g.generateCommandSeq(ctx, n)
	case ir.Wrapper:
		return g.generateWrapper(ctx, n)
	case ir.Pattern:
		return g.generatePattern(ctx, n)
	default:
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("Unknown node type: %T", node),
		}
	}
}

// generateCommandSeq generates a sequence of command steps
func (g *Generator) generateCommandSeq(ctx *ir.Ctx, seq ir.CommandSeq) plan.ExecutionStep {
	step := plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: fmt.Sprintf("Execute %d command steps", len(seq.Steps)),
		Children:    make([]plan.ExecutionStep, 0, len(seq.Steps)),
	}

	for i, cmdStep := range seq.Steps {
		childStep := g.generateCommandStep(ctx, cmdStep)
		childStep.ID = fmt.Sprintf("%d", i)
		step.Children = append(step.Children, childStep)
	}

	return step
}

// generateCommandStep generates a single command step (chain of elements)
func (g *Generator) generateCommandStep(ctx *ir.Ctx, cmdStep ir.CommandStep) plan.ExecutionStep {
	if len(cmdStep.Chain) == 0 {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: "(empty)",
		}
	}

	if len(cmdStep.Chain) == 1 {
		element := cmdStep.Chain[0]

		// Check if single element has operators (like >> output.txt)
		if element.OpNext != "" {
			// Treat single element with operators as a chain too
			decorCtx := g.toDecoratorContext(ctx)

			var elementText string
			if element.Kind == ir.ElementKindShell && element.Content != nil {
				resolvedText, err := element.Content.Resolve(decorCtx, g.registry)
				if err != nil {
					elementText = fmt.Sprintf("<error: %v>", err)
				} else {
					elementText = resolvedText
				}
			} else if element.Kind == ir.ElementKindAction {
				elementText = fmt.Sprintf("@%s", element.Name)
				if len(element.Args) > 0 {
					elementText += "("
					var argParts []string
					for _, arg := range element.Args {
						if arg.Name == "" {
							argParts = append(argParts, fmt.Sprintf("%v", arg.Value))
						} else {
							argParts = append(argParts, fmt.Sprintf("%s=%v", arg.Name, arg.Value))
						}
					}
					elementText += strings.Join(argParts, ", ") + ")"
				}
			} else {
				elementText = "<unknown>"
			}

			// Add operator
			var parts []string
			parts = append(parts, elementText)
			switch element.OpNext {
			case ir.ChainOpAppend:
				if element.Target != "" {
					parts = append(parts, ">>", element.Target)
				} else {
					parts = append(parts, ">>")
				}
			case ir.ChainOpAnd:
				parts = append(parts, "&&")
			case ir.ChainOpOr:
				parts = append(parts, "||")
			case ir.ChainOpPipe:
				parts = append(parts, "|")
			}

			chainCommand := strings.Join(parts, " ")

			// Build metadata with operator information
			metadata := map[string]string{
				"kind": "shell_chain",
			}

			// Add operator metadata for tests
			switch element.OpNext {
			case ir.ChainOpAppend:
				metadata["op_next"] = ">>"
			case ir.ChainOpAnd:
				metadata["op_next"] = "&&"
			case ir.ChainOpOr:
				metadata["op_next"] = "||"
			case ir.ChainOpPipe:
				metadata["op_next"] = "|"
			}

			return plan.ExecutionStep{
				Type:        plan.StepShell,
				Description: chainCommand,
				Command:     chainCommand,
				Metadata:    metadata,
			}
		}

		// Single element without operators - convert directly
		return g.generateChainElement(ctx, element)
	}

	// Multiple elements - treat as a single shell chain command per spec
	// Shell operators like && || | should be one command, not separate steps
	decorCtx := g.toDecoratorContext(ctx)

	// Build the complete chain command string
	var parts []string
	for i, element := range cmdStep.Chain {
		var elementText string

		// Get the text for this element
		if element.Kind == ir.ElementKindShell && element.Content != nil {
			resolvedText, err := element.Content.Resolve(decorCtx, g.registry)
			if err != nil {
				elementText = fmt.Sprintf("<error: %v>", err)
			} else {
				elementText = resolvedText
			}
		} else if element.Kind == ir.ElementKindAction {
			// For action decorators in chains, show the decorator syntax
			elementText = fmt.Sprintf("@%s", element.Name)
			if len(element.Args) > 0 {
				elementText += "("
				var argParts []string
				for _, arg := range element.Args {
					if arg.Name == "" {
						argParts = append(argParts, fmt.Sprintf("%v", arg.Value))
					} else {
						argParts = append(argParts, fmt.Sprintf("%s=%v", arg.Name, arg.Value))
					}
				}
				elementText += strings.Join(argParts, ", ") + ")"
			}
		} else {
			elementText = "<unknown>"
		}

		// Add the element text
		parts = append(parts, elementText)

		// Add operator if not the last element
		if i < len(cmdStep.Chain)-1 {
			switch element.OpNext {
			case ir.ChainOpAnd:
				parts = append(parts, "&&")
			case ir.ChainOpOr:
				parts = append(parts, "||")
			case ir.ChainOpPipe:
				parts = append(parts, "|")
			case ir.ChainOpAppend:
				if element.Target != "" {
					parts = append(parts, ">>", element.Target)
				} else {
					parts = append(parts, ">>")
				}
			}
		}
	}

	// Create a single shell step for the entire chain
	chainCommand := strings.Join(parts, " ")

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: chainCommand,
		Command:     chainCommand,
		Metadata: map[string]string{
			"kind": "shell_chain",
		},
	}
}

// generateChainElement generates a single chain element
func (g *Generator) generateChainElement(ctx *ir.Ctx, element ir.ChainElement) plan.ExecutionStep {
	switch element.Kind {
	case ir.ElementKindShell:
		return g.generateShellElement(ctx, element)
	case ir.ElementKindAction:
		return g.generateActionElement(ctx, element)
	case ir.ElementKindBlock:
		return g.generateBlockElement(ctx, element)
	default:
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("Unknown element kind: %s", element.Kind),
		}
	}
}

// generateShellElement generates a shell command element
func (g *Generator) generateShellElement(ctx *ir.Ctx, element ir.ChainElement) plan.ExecutionStep {
	// Convert execution context to decorator context
	decorCtx := g.toDecoratorContext(ctx)

	// Get resolved command text using structured content
	var command string
	var description string

	if element.Content != nil {
		// Use the new structured content system
		resolvedText, err := element.Content.Resolve(decorCtx, g.registry)
		if err != nil {
			command = fmt.Sprintf("<error resolving content: %v>", err)
			description = command
		} else {
			command = resolvedText
			description = element.PlanDescription(decorCtx, g.registry)
		}
	} else {
		// Fallback for old-style elements (should not happen in new code)
		command = "<missing structured content>"
		description = command
	}

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: description,
		Command:     command,
		Metadata: map[string]string{
			"kind": "shell",
		},
	}
}

// generateActionElement generates an action decorator element
func (g *Generator) generateActionElement(ctx *ir.Ctx, element ir.ChainElement) plan.ExecutionStep {
	decorator, exists := g.registry.GetAction(element.Name)
	if !exists {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("Unknown action: @%s", element.Name),
			Metadata: map[string]string{
				"error": "decorator_not_found",
			},
		}
	}

	// Convert execution context to decorator context
	decorCtx := g.toDecoratorContext(ctx)

	// Use decorator's Describe method for plan generation
	return decorator.Describe(decorCtx, element.Args)
}

// generateBlockElement generates a block decorator element
func (g *Generator) generateBlockElement(ctx *ir.Ctx, element ir.ChainElement) plan.ExecutionStep {
	decorator, exists := g.registry.GetBlock(element.Name)
	if !exists {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("Unknown block decorator: @%s", element.Name),
			Metadata: map[string]string{
				"error": "decorator_not_found",
			},
		}
	}

	// Convert inner steps to ExecutionStep
	innerStep := plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: "Inner commands",
		Children:    make([]plan.ExecutionStep, 0, len(element.InnerSteps)),
	}

	for i, innerCmdStep := range element.InnerSteps {
		childStep := g.generateCommandStep(ctx, innerCmdStep)
		childStep.ID = fmt.Sprintf("%d", i)
		innerStep.Children = append(innerStep.Children, childStep)
	}

	// Convert execution context to decorator context
	decorCtx := g.toDecoratorContext(ctx)

	// Use decorator's Describe method for plan generation
	return decorator.Describe(decorCtx, element.Args, innerStep)
}

// generateWrapper generates a wrapper (block decorator) step
func (g *Generator) generateWrapper(ctx *ir.Ctx, wrapper ir.Wrapper) plan.ExecutionStep {
	decorator, exists := g.registry.GetBlock(wrapper.Kind)
	if !exists {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("Unknown wrapper: @%s", wrapper.Kind),
			Metadata: map[string]string{
				"error": "decorator_not_found",
			},
		}
	}

	// Generate inner step
	innerStep := g.generateStep(ctx, wrapper.Inner)

	// Add metadata about original command step count for decorators like @parallel
	if innerStep.Metadata == nil {
		innerStep.Metadata = make(map[string]string)
	}
	innerStep.Metadata["original_step_count"] = fmt.Sprintf("%d", len(wrapper.Inner.Steps))

	// Convert parameters to decorator format
	var args []decorators.DecoratorParam
	for name, value := range wrapper.Params {
		args = append(args, decorators.DecoratorParam{
			Name:  name,
			Value: value,
		})
	}

	// Convert execution context to decorator context
	decorCtx := g.toDecoratorContext(ctx)

	// Use decorator's Describe method for plan generation
	return decorator.Describe(decorCtx, args, innerStep)
}

// generatePattern generates a pattern decorator step
func (g *Generator) generatePattern(ctx *ir.Ctx, pattern ir.Pattern) plan.ExecutionStep {
	decorator, exists := g.registry.GetPattern(pattern.Kind)
	if !exists {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("Unknown pattern: @%s", pattern.Kind),
			Metadata: map[string]string{
				"decorator":      pattern.Kind,
				"execution_mode": string(plan.ModeConditional),
				"error":          "decorator_not_found",
			},
		}
	}

	// Generate branches
	branches := make(map[string]plan.ExecutionStep)
	for branchName, branchSeq := range pattern.Branches {
		branches[branchName] = g.generateStep(ctx, branchSeq)
	}

	// Convert parameters to decorator format
	var args []decorators.DecoratorParam
	for name, value := range pattern.Params {
		args = append(args, decorators.DecoratorParam{
			Name:  name,
			Value: value,
		})
	}

	// Convert execution context to decorator context
	decorCtx := g.toDecoratorContext(ctx)

	// Use decorator's Describe method for plan generation
	return decorator.Describe(decorCtx, args, branches)
}

// generateSummary generates plan summary information
func (g *Generator) generateSummary(executionPlan *plan.ExecutionPlan) plan.PlanSummary {
	summary := plan.PlanSummary{
		DecoratorsUsed:  []string{},
		RequiredImports: []string{},
	}

	// Count different types of steps
	g.countSteps(executionPlan.Steps, &summary)

	return summary
}

// countSteps recursively counts steps by type
func (g *Generator) countSteps(steps []plan.ExecutionStep, summary *plan.PlanSummary) {
	for _, step := range steps {
		summary.TotalSteps++

		switch step.Type {
		case plan.StepShell:
			summary.ShellCommands++

		case plan.StepDecorator:
			// Fully plugin-friendly approach using execution modes!
			if decoratorName := step.Metadata["decorator"]; decoratorName != "" {
				g.addDecoratorToSummary(decoratorName, summary)
			}

			// Use execution mode for categorization - works with any decorator
			if modeStr := step.Metadata["execution_mode"]; modeStr != "" {
				mode := plan.ExecutionMode(modeStr)

				// Legacy field updates for backward compatibility
				switch mode {
				case plan.ModeParallel:
					summary.ParallelSections++
				case plan.ModeConditional:
					summary.ConditionalBranches++
				case plan.ModeErrorHandling:
					summary.HasErrorHandling = true
				}
			}

		case plan.StepSequence:
			// No special counting needed for sequences
		}

		// Recursively count children
		g.countSteps(step.Children, summary)
	}
}

// addDecoratorToSummary adds a decorator to the summary if not already present
func (g *Generator) addDecoratorToSummary(decoratorName string, summary *plan.PlanSummary) {
	for _, existing := range summary.DecoratorsUsed {
		if existing == decoratorName {
			return // Already added
		}
	}
	summary.DecoratorsUsed = append(summary.DecoratorsUsed, decoratorName)
}

// expandCommandReferences recursively expands steps based on expansion hints in metadata
func (g *Generator) expandCommandReferences(ctx *ir.Ctx, executionPlan *plan.ExecutionPlan) {
	// Recursively expand all steps based on their expansion hints
	g.expandStepsRecursive(ctx, executionPlan.Steps)
}

// expandStepsRecursive recursively processes steps to expand based on expansion hints
func (g *Generator) expandStepsRecursive(ctx *ir.Ctx, steps []plan.ExecutionStep) {
	for i := range steps {
		step := &steps[i]

		// Check if this step has expansion hints
		if step.Metadata != nil {
			expansionType := step.Metadata["expansion_type"]

			switch expansionType {
			case "command_reference":
				g.expandCommandReference(ctx, step)
			case "template_include":
				g.expandTemplateInclude(ctx, step)
			case "module_import":
				g.expandModuleImport(ctx, step)
			case "file_include":
				g.expandFileInclude(ctx, step)
			}
		}

		// Recursively process children
		g.expandStepsRecursive(ctx, step.Children)
	}
}

// expandCommandReference expands a command reference using the command resolver
func (g *Generator) expandCommandReference(ctx *ir.Ctx, step *plan.ExecutionStep) {
	cmdName := step.Metadata["command_name"]
	if cmdName == "" {
		step.Description = fmt.Sprintf("%s <error: no command_name in metadata>", step.Description)
		return
	}

	if g.commandResolver == nil {
		step.Description = fmt.Sprintf("%s <error: no command resolver>", step.Description)
		return
	}

	// Look up the command definition
	commandNode, err := g.commandResolver.GetCommand(cmdName)
	if err != nil {
		// Command not found - update description to show error
		step.Description = fmt.Sprintf("%s <error: %v>", step.Description, err)
		return
	}

	// Generate plan for the referenced command and set as children
	expandedSteps := g.generateStep(ctx, commandNode)
	if expandedSteps.Type == plan.StepSequence && len(expandedSteps.Children) > 0 {
		// If it's a sequence with children, use the children directly
		step.Children = expandedSteps.Children
	} else {
		// Otherwise, add the expanded step as a single child
		step.Children = []plan.ExecutionStep{expandedSteps}
	}
}

// expandTemplateInclude expands a template include (placeholder for future implementation)
func (g *Generator) expandTemplateInclude(ctx *ir.Ctx, step *plan.ExecutionStep) {
	templatePath := step.Metadata["template_path"]
	step.Description = fmt.Sprintf("%s <template expansion not yet implemented: %s>", step.Description, templatePath)
}

// expandModuleImport expands a module import (placeholder for future implementation)
func (g *Generator) expandModuleImport(ctx *ir.Ctx, step *plan.ExecutionStep) {
	moduleName := step.Metadata["module_name"]
	step.Description = fmt.Sprintf("%s <module expansion not yet implemented: %s>", step.Description, moduleName)
}

// expandFileInclude expands a file include (placeholder for future implementation)
func (g *Generator) expandFileInclude(ctx *ir.Ctx, step *plan.ExecutionStep) {
	includePath := step.Metadata["include_path"]
	step.Description = fmt.Sprintf("%s <file expansion not yet implemented: %s>", step.Description, includePath)
}

// generateMixedChain handles chains that contain action decorators
// These are broken down into individual steps for better readability
func (g *Generator) generateMixedChain(ctx *ir.Ctx, chain []ir.ChainElement) plan.ExecutionStep {
	// Create a parent "Command chain" step with individual children
	var children []plan.ExecutionStep

	for _, element := range chain {
		childStep := g.generateChainElement(ctx, element)
		children = append(children, childStep)
	}

	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: "Command chain",
		Command:     "Command chain",
		Children:    children,
		Metadata: map[string]string{
			"kind": "mixed_chain",
		},
	}
}

// toDecoratorContext converts execution context to decorator context
func (g *Generator) toDecoratorContext(ctx *ir.Ctx) *decorators.Ctx {
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
		NumCPU:  ctx.NumCPU,
	}
}
