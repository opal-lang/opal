package builtin

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aledsdavies/devcmd/codegen"
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @confirm decorator on package import
func init() {
	decorators.RegisterBlock(NewConfirmDecorator())
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

// ImportRequirements returns the dependencies needed for code generation
func (c *ConfirmDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"bufio", "os", "strings"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap prompts for user confirmation before executing inner commands
func (c *ConfirmDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	message, defaultYes, err := c.extractParameters(args)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@confirm parameter error: %v", err),
			ExitCode: 1,
		}
	}

	// In dry-run mode, don't prompt - just show what would be prompted
	if ctx.DryRun {
		return decorators.CommandResult{
			Stdout:   fmt.Sprintf("[DRY-RUN] Would prompt: %s", message),
			ExitCode: 0,
		}
	}

	// Display confirmation prompt
	prompt := message
	if defaultYes {
		prompt += " [Y/n]: "
	} else {
		prompt += " [y/N]: "
	}

	fmt.Fprint(ctx.Stderr, prompt)

	// Read user response
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return decorators.CommandResult{
			Stderr:   "failed to read user input",
			ExitCode: 1,
		}
	}

	response := strings.TrimSpace(strings.ToLower(scanner.Text()))

	// Determine if user confirmed
	confirmed := false
	if response == "" {
		// User just pressed enter - use default
		confirmed = defaultYes
	} else if response == "y" || response == "yes" {
		confirmed = true
	} else if response == "n" || response == "no" {
		confirmed = false
	} else {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("invalid response %q, please answer 'y' or 'n'", response),
			ExitCode: 1,
		}
	}

	if !confirmed {
		return decorators.CommandResult{
			Stdout:   "Operation cancelled by user",
			ExitCode: 130, // Standard "interrupted by user" exit code
		}
	}

	// User confirmed - execute inner commands
	return ctx.ExecSequential(inner.Steps)
}

// Describe returns description for dry-run display
func (c *ConfirmDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
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
// OPTIONAL CODE GENERATION HINT
// ================================================================================================

// GenerateBlockHint provides code generation hint for confirmation prompt
func (c *ConfirmDecorator) GenerateBlockHint(ops codegen.GenOps, args []decorators.DecoratorParam, body func(codegen.GenOps) codegen.TempResult) codegen.TempResult {
	message, defaultYes, err := c.extractParameters(args)
	if err != nil {
		return ops.Literal(fmt.Sprintf("<error: %v>", err))
	}

	innerResult := body(ops)

	// Generate confirmation prompt code
	confirmCode := fmt.Sprintf(`func() CommandResult {
		// Display confirmation prompt
		prompt := %q
		if %t {
			prompt += " [Y/n]: "
		} else {
			prompt += " [y/N]: "
		}
		
		fmt.Fprint(os.Stderr, prompt)
		
		// Read user response
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return CommandResult{Stderr: "failed to read user input", ExitCode: 1}
		}
		
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		
		// Determine if user confirmed
		confirmed := false
		if response == "" {
			confirmed = %t // defaultYes
		} else if response == "y" || response == "yes" {
			confirmed = true
		} else if response == "n" || response == "no" {
			confirmed = false
		} else {
			return CommandResult{
				Stderr: fmt.Sprintf("invalid response %%q, please answer 'y' or 'n'", response),
				ExitCode: 1,
			}
		}
		
		if !confirmed {
			return CommandResult{Stdout: "Operation cancelled by user", ExitCode: 130}
		}
		
		// User confirmed - execute commands
		return %s
	}()`, message, defaultYes, defaultYes, innerResult.String())

	return ops.Literal(confirmCode)
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractParameters extracts and validates confirm parameters
func (c *ConfirmDecorator) extractParameters(params []decorators.DecoratorParam) (message string, defaultYes bool, err error) {
	// Set defaults
	message = "Continue?"
	defaultYes = false

	// Extract optional parameters
	for _, param := range params {
		switch param.Name {
		case "":
			// Positional parameter - assume it's the message
			if val, ok := param.Value.(string); ok {
				message = val
			} else {
				return "", false, fmt.Errorf("@confirm message must be a string, got %T", param.Value)
			}
		case "message":
			if val, ok := param.Value.(string); ok {
				message = val
			} else {
				return "", false, fmt.Errorf("@confirm message parameter must be a string, got %T", param.Value)
			}
		case "defaultYes":
			if val, ok := param.Value.(bool); ok {
				defaultYes = val
			} else {
				return "", false, fmt.Errorf("@confirm defaultYes parameter must be a boolean, got %T", param.Value)
			}
		default:
			return "", false, fmt.Errorf("@confirm unknown parameter: %s", param.Name)
		}
	}

	if message == "" {
		return "", false, fmt.Errorf("@confirm message cannot be empty")
	}

	return message, defaultYes, nil
}
