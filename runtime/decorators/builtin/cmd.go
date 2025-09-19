package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// ErrorResult implements decorators.CommandResult for error cases
type ErrorResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func (r *ErrorResult) GetStdout() string { return r.stdout }
func (r *ErrorResult) GetStderr() string { return r.stderr }
func (r *ErrorResult) GetExitCode() int  { return r.exitCode }
func (r *ErrorResult) IsSuccess() bool   { return r.exitCode == 0 }

// Register the @cmd decorator on package import
func init() {
	decorator := NewCmdDecorator()
	// Register with legacy interface (Phase 4: remove this)
	decorators.RegisterAction(decorator)
	// Register with new interface
	decorators.RegisterExecutionDecorator(decorator)
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

// ================================================================================================
// LEGACY ACTION DECORATOR METHODS (will be removed in Phase 4)
// ================================================================================================

// Run executes the referenced command using core interfaces
func (c *CmdDecorator) Run(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	cmdName, err := c.extractDecoratorCommandName(args)
	if err != nil {
		// Create a simple error result that implements the interface
		return &ErrorResult{
			stderr:   fmt.Sprintf("@cmd parameter error: %v", err),
			exitCode: 1,
		}
	}

	// Use the interface method
	return ctx.ExecShell(cmdName)
}

// Describe returns description for dry-run display with expansion hints
func (c *CmdDecorator) Describe(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
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
func (c *CmdDecorator) extractDecoratorCommandName(params []decorators.Param) (string, error) {
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

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (target interface)
// ================================================================================================

// Plan generates an execution plan for the command reference
func (c *CmdDecorator) Plan(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	cmdName, err := c.extractDecoratorCommandName(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@cmd(<error: %v>)", err),
			Command:     "",
			Metadata: map[string]string{
				"decorator": "cmd",
				"error":     err.Error(),
			},
		}
	}

	// Return a command reference step with expansion hints for the plan generator
	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@cmd(%s)", cmdName),
		Command:     fmt.Sprintf("# Execute command: %s", cmdName),
		Children:    []plan.ExecutionStep{},
		Metadata: map[string]string{
			"decorator":      "cmd",
			"expansion_type": "command_reference",
			"command_name":   cmdName,
		},
	}
}

// Execute runs the referenced command
func (c *CmdDecorator) Execute(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	cmdName, err := c.extractDecoratorCommandName(args)
	if err != nil {
		return &ErrorResult{
			stderr:   fmt.Sprintf("@cmd parameter error: %v", err),
			exitCode: 1,
		}
	}

	// Execute the referenced command
	return ctx.ExecShell(cmdName)
}

// RequiresBlock returns the block requirements for @cmd
func (c *CmdDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockNone,
		Required: false,
	}
}
