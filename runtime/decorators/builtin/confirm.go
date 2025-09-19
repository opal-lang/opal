package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// Register the @confirm decorator on package import
func init() {
	decorator := NewConfirmDecorator()
	decorators.RegisterBlock(decorator)
	decorators.RegisterExecutionDecorator(decorator)
}

// ConfirmDecorator implements the @confirm decorator using the core decorator interfaces
type ConfirmDecorator struct{}

// NewConfirmDecorator creates a new confirm decorator
func NewConfirmDecorator() *ConfirmDecorator {
	return &ConfirmDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (c *ConfirmDecorator) Name() string {
	return "confirm"
}

// Description returns a human-readable description
func (c *ConfirmDecorator) Description() string {
	return "Prompt user for confirmation before executing commands"
}

// ParameterSchema returns the expected parameters for this decorator
func (c *ConfirmDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "message",
			Type:        decorators.ArgTypeString,
			Required:    false,
			Description: "Custom confirmation message (default: 'Continue?')",
		},
		{
			Name:        "defaultYes",
			Type:        decorators.ArgTypeBool,
			Required:    false,
			Description: "Default to 'yes' if user just presses enter (default: false)",
		},
	}
}

// Examples returns usage examples
func (c *ConfirmDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `deploy: @confirm("Deploy to production?") {
    kubectl apply -f prod.yaml
}`,
			Description: "Confirm before production deployment",
		},
		{
			Code: `clean: @confirm(defaultYes=true) {
    rm -rf build/
}`,
			Description: "Confirm with default yes for cleanup",
		},
		{
			Code: `reset: @confirm("This will reset all data. Are you sure?") {
    docker-compose down -v
    docker system prune -af
}`,
			Description: "Confirm before destructive operations",
		},
	}
}

// Note: ImportRequirements removed - will be added back when code generation is implemented

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap prompts for user confirmation before executing inner commands
func (c *ConfirmDecorator) WrapCommands(ctx decorators.Context, args []decorators.Param, inner interface{}) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "runtime execution not implemented yet - use plan mode",
		ExitCode: 1,
	}
}

// Describe returns description for dry-run display
func (c *ConfirmDecorator) Describe(ctx decorators.Context, args []decorators.Param, inner plan.ExecutionStep) plan.ExecutionStep {
	message, defaultYes, err := c.extractParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepSequence,
			Description: fmt.Sprintf("@confirm(<error: %v>)", err),
			Command:     "",
		}
	}

	description := fmt.Sprintf("@confirm(%q)", message)
	if defaultYes {
		description += " [default: yes]"
	}

	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: description,
		Command:     fmt.Sprintf("prompt: %s", message),
		Children:    []plan.ExecutionStep{inner}, // Nested execution
		Metadata: map[string]string{
			"decorator":  "confirm",
			"message":    message,
			"defaultYes": fmt.Sprintf("%t", defaultYes),
		},
	}
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractParameters extracts and validates confirm parameters
func (c *ConfirmDecorator) extractParameters(params []decorators.Param) (message string, defaultYes bool, err error) {
	// Set defaults
	message = "Continue?"
	defaultYes = false

	// Extract optional parameters
	for _, param := range params {
		switch param.GetName() {
		case "":
			// Positional parameter - assume it's the message
			if val, ok := param.GetValue().(string); ok {
				message = val
			} else {
				return "", false, fmt.Errorf("@confirm message must be a string, got %T", param.GetValue())
			}
		case "message":
			if val, ok := param.GetValue().(string); ok {
				message = val
			} else {
				return "", false, fmt.Errorf("@confirm message parameter must be a string, got %T", param.GetValue())
			}
		case "defaultYes":
			if val, ok := param.GetValue().(bool); ok {
				defaultYes = val
			} else {
				return "", false, fmt.Errorf("@confirm defaultYes parameter must be a boolean, got %T", param.GetValue())
			}
		default:
			return "", false, fmt.Errorf("@confirm unknown parameter: %s", param.GetName())
		}
	}

	if message == "" {
		return "", false, fmt.Errorf("@confirm message cannot be empty")
	}

	return message, defaultYes, nil
}

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (target interface)
// ================================================================================================

// Plan generates an execution plan for the confirm operation
func (c *ConfirmDecorator) Plan(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	message, defaultYes, err := c.extractParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@confirm(<error: %v>)", err),
			Command:     "",
			Metadata: map[string]string{
				"decorator": "confirm",
				"error":     err.Error(),
			},
		}
	}

	description := fmt.Sprintf("@confirm(%q)", message)
	if defaultYes {
		description += " [default: yes]"
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: description,
		Command:     fmt.Sprintf("prompt: %s", message),
		Children:    []plan.ExecutionStep{}, // Will be populated by plan generator
		Metadata: map[string]string{
			"decorator":      "confirm",
			"message":        message,
			"defaultYes":     fmt.Sprintf("%t", defaultYes),
			"execution_mode": "interactive",
			"color":          plan.ColorYellow,
		},
	}
}

// Execute performs the confirm operation
func (c *ConfirmDecorator) Execute(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return &simpleCommandResult{
		stdout:   "",
		stderr:   "confirm execution not implemented yet - use plan mode",
		exitCode: 1,
	}
}

// RequiresBlock returns the block requirements for @confirm
func (c *ConfirmDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
