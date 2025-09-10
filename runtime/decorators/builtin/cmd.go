package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/codegen"
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
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@cmd: execution delegate not available"),
			ExitCode: 1,
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
		Type:        plan.StepDecorator,               // Changed to StepDecorator so formatter uses info metadata
		Description: fmt.Sprintf("@cmd(%s)", cmdName), // Set complete description directly
		Command:     "",
		Children:    []plan.ExecutionStep{}, // Generator will populate based on hints
		Metadata: map[string]string{
			"decorator":      "cmd",
			"expansion_type": "command_reference",
			"command_name":   cmdName,
		},
	}
}

// ================================================================================================
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateHint provides code generation hint for command execution
// TODO: Update to use []decorators.DecoratorParam instead of []ast.NamedParameter
func (c *CmdDecorator) GenerateHint(ops codegen.GenOps, args []decorators.DecoratorParam) codegen.TempResult {
	cmdName, err := c.extractCommandName(args)
	if err != nil {
		return ops.Literal(fmt.Sprintf("<error: %v>", err))
	}

	// Generate a call to the command function
	// The actual function name will be sanitized (e.g., "test-all" -> "testAll")
	functionName := c.sanitizeCommandName(cmdName)
	return ops.Literal(fmt.Sprintf("execute%s(ctx)", functionName))
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

// extractCommandName extracts the command name from AST parameters (legacy for codegen)
func (c *CmdDecorator) extractCommandName(params []decorators.DecoratorParam) (string, error) {
	if len(params) == 0 {
		return "", fmt.Errorf("@cmd requires a command name")
	}

	var cmdName string
	if params[0].Name == "" {
		// Positional parameter
		if val, ok := params[0].Value.(string); ok {
			cmdName = val
		} else {
			return "", fmt.Errorf("@cmd expects a string, got %T", params[0].Value)
		}
	} else if params[0].Name == "name" {
		// Named parameter
		if val, ok := params[0].Value.(string); ok {
			cmdName = val
		} else {
			return "", fmt.Errorf("@cmd name parameter must be a string")
		}
	} else {
		return "", fmt.Errorf("@cmd first parameter must be the command name")
	}

	if cmdName == "" {
		return "", fmt.Errorf("@cmd requires a non-empty command name")
	}

	return cmdName, nil
}

// sanitizeCommandName converts a command name to a valid Go function name
func (c *CmdDecorator) sanitizeCommandName(name string) string {
	// Convert kebab-case and snake_case to CamelCase
	// e.g., "test-all" -> "TestAll", "build_docker" -> "BuildDocker"

	result := make([]rune, 0, len(name))
	capitalizeNext := true

	for _, r := range name {
		if r == '-' || r == '_' || r == ' ' {
			capitalizeNext = true
		} else if capitalizeNext {
			if r >= 'a' && r <= 'z' {
				result = append(result, r-'a'+'A')
			} else {
				result = append(result, r)
			}
			capitalizeNext = false
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}
