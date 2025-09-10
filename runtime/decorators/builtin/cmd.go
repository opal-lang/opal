package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @cmd decorator on package import
func init() {
	decorators.RegisterAction(NewCmdDecorator())
}

// CmdDecorator implements the @cmd decorator using the core decorator interfaces
type CmdDecorator struct{}

// NewCmdDecorator creates a new cmd decorator
func NewCmdDecorator() *CmdDecorator {
	return &CmdDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (c *CmdDecorator) Name() string {
	return "cmd"
}

// Description returns a human-readable description
func (c *CmdDecorator) Description() string {
	return "Execute another defined command by name for reuse and composition"
}

// ParameterSchema returns the expected parameters for this decorator
func (c *CmdDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "name",
			Type:        decorators.ArgTypeIdentifier,
			Required:    true,
			Description: "Name of the command to execute",
		},
	}
}

// Examples returns usage examples
func (c *CmdDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code:        "@cmd(build)",
			Description: "Execute the build command",
		},
		{
			Code:        "deploy: @cmd(build) && kubectl apply -f k8s/",
			Description: "Run build command then deploy",
		},
		{
			Code:        "@cmd(test) || echo \"Tests failed\"",
			Description: "Run tests with fallback on failure",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (c *CmdDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// ACTION DECORATOR METHODS
// ================================================================================================

// Run executes the referenced command
func (c *CmdDecorator) Run(ctx *decorators.Ctx, args []decorators.DecoratorParam) decorators.CommandResult {
	cmdName, err := c.extractDecoratorCommandName(args)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@cmd parameter error: %v", err),
			ExitCode: 1,
		}
	}

	// Use the execution delegate to execute the command by name
	if ctx.Executor == nil {
		// Stub behavior for testing - output TODO message
		return decorators.CommandResult{
			Stdout:   fmt.Sprintf("[TODO: Execute command '%s']", cmdName),
			Stderr:   "",
			ExitCode: 0,
		}
	}

	// Execute the command through the delegate (which has access to the command registry)
	return ctx.Executor.ExecuteCommand(ctx, cmdName)
}

// Describe returns description for dry-run display with expansion hints
func (c *CmdDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam) plan.ExecutionStep {
	cmdName, err := c.extractDecoratorCommandName(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@cmd(<error: %v>)", err),
			Command:     "",
		}
	}

	// Return a command reference step with expansion hints for the plan generator
	// The generator will see "expansion_type": "command_reference" and use its
	// CommandResolver to look up and expand the command structure
	return plan.ExecutionStep{
		Type:        plan.StepDecorator,                            // Changed to StepDecorator so formatter uses info metadata
		Description: fmt.Sprintf("@cmd(%s)", cmdName),              // Set complete description directly
		Command:     fmt.Sprintf("# Execute command: %s", cmdName), // Tests expect this format
		Children:    []plan.ExecutionStep{},                        // Generator will populate based on hints
		Metadata: map[string]string{
			"decorator":      "cmd",
			"expansion_type": "command_reference",
			"command_name":   cmdName,
		},
	}
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractDecoratorCommandName extracts the command name from decorator parameters
func (c *CmdDecorator) extractDecoratorCommandName(params []decorators.DecoratorParam) (string, error) {
	// Extract command name (first positional parameter or named "name")
	cmdName, err := decorators.ExtractPositionalString(params, 0, "")
	if err != nil || cmdName == "" {
		// Try named parameter
		cmdName, err = decorators.ExtractString(params, "name", "")
		if err != nil || cmdName == "" {
			return "", fmt.Errorf("@cmd requires a command name")
		}
	}

	if cmdName == "" {
		return "", fmt.Errorf("@cmd requires a non-empty command name")
	}

	return cmdName, nil
}
