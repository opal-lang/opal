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

// ConfirmParams represents validated parameters for @confirm decorator
type ConfirmParams struct {
	Message    string `json:"message"`     // Confirmation message (default: "Continue?")
	DefaultYes bool   `json:"default_yes"` // Default to yes if user just presses enter
}

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

// ================================================================================================
// NEW GENERIC INTERFACE METHODS (ExecutionDecorator[any])
// ================================================================================================

// Validate validates parameters and returns ConfirmParams
func (c *ConfirmDecorator) Validate(args []decorators.Param) (any, error) {
	// Extract message (first positional parameter or named "message")
	message, err := decorators.ExtractPositionalString(args, 0, "Continue?")
	if err != nil {
		// Try named parameter "message"
		message, err = decorators.ExtractString(args, "message", "Continue?")
		if err != nil {
			return nil, fmt.Errorf("@confirm message parameter error: %w", err)
		}
	}

	// Extract defaultYes flag (optional, defaults to false)
	defaultYes, err := decorators.ExtractBool(args, "defaultYes", false)
	if err != nil {
		return nil, fmt.Errorf("@confirm defaultYes parameter error: %w", err)
	}

	return ConfirmParams{
		Message:    message,
		DefaultYes: defaultYes,
	}, nil
}

// Plan generates an execution plan using validated parameters
func (c *ConfirmDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(ConfirmParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: "@confirm(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "confirm",
				"error":     "invalid_params",
			},
		}
	}

	defaultText := ""
	if params.DefaultYes {
		defaultText = " [Y/n]"
	} else {
		defaultText = " [y/N]"
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: fmt.Sprintf("@confirm(%q%s) { ... }", params.Message, defaultText),
		Command:     "",
		Metadata: map[string]string{
			"decorator":   "confirm",
			"message":     params.Message,
			"default_yes": fmt.Sprintf("%t", params.DefaultYes),
			"status":      "awaiting_executable_block_implementation",
		},
	}
}

// Execute performs the actual confirmation using validated parameters
func (c *ConfirmDecorator) Execute(ctx decorators.Context, validated any) (decorators.CommandResult, error) {
	_, ok := validated.(ConfirmParams)
	if !ok {
		return nil, fmt.Errorf("@confirm: invalid parameters")
	}

	// TODO: When ExecutableBlock is implemented, this will become:
	// response, err := ctx.Confirm(params.Message)
	// if err != nil {
	//     return nil, err
	// }
	// if response {
	//     for _, stmt := range params.Block {
	//         result, err := stmt.Execute(ctx)
	//         if err != nil {
	//             return result, err
	//         }
	//     }
	// }

	return nil, fmt.Errorf("@confirm: ExecutableBlock not yet implemented - use legacy interface for now")
}

// RequiresBlock returns the block requirements for @confirm
func (c *ConfirmDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
