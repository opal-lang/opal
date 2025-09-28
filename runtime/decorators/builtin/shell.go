package builtin

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/core/plan"
)

// ShellDecorator implements the @shell decorator for direct shell command execution.
// This is the foundational decorator that enables the "shell syntax as decorator sugar" concept.
// All shell commands like "npm install" become "@shell('npm install')" internally.
type ShellDecorator struct{}

// ShellParams represents validated parameters for @shell decorator
type ShellParams struct {
	Command string `json:"command"` // Shell command to execute
}

// NewShellDecorator creates a new shell decorator
func NewShellDecorator() *ShellDecorator {
	return &ShellDecorator{}
}

// Name returns the decorator name
func (s *ShellDecorator) Name() string {
	return "shell"
}

// Description returns a human-readable description
func (s *ShellDecorator) Description() string {
	return "Execute shell commands directly"
}

// ParameterSchema returns the parameter schema for the shell decorator
func (s *ShellDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "command",
			Type:        decorators.ArgTypeString,
			Required:    true,
			Description: "Shell command to execute",
		},
	}
}

// Examples returns usage examples
func (s *ShellDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code:        `npm install`,
			Description: "Shell command (automatically becomes @shell internally)",
		},
		{
			Code:        `kubectl apply -f k8s/`,
			Description: "Complex shell command with arguments",
		},
		{
			Code:        `echo "Building" && npm run build`,
			Description: "Shell operators (each part becomes separate @shell decorator)",
		},
	}
}

// ================================================================================================
// NEW GENERIC INTERFACE METHODS (ExecutionDecorator[any])
// ================================================================================================

// Validate validates parameters and returns ShellParams
func (s *ShellDecorator) Validate(args []decorators.Param) (any, error) {
	// Extract shell command (first positional parameter or named "command")
	command, err := decorators.ExtractPositionalString(args, 0, "")
	if err != nil || command == "" {
		// Try named parameter "command"
		command, err = decorators.ExtractString(args, "command", "")
		if err != nil || command == "" {
			return nil, fmt.Errorf("@shell requires a command")
		}
	}

	if command == "" {
		return nil, fmt.Errorf("@shell requires a non-empty command")
	}

	return ShellParams{Command: command}, nil
}

// Plan generates an execution plan using validated parameters
func (s *ShellDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(ShellParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: "@shell(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "shell",
				"error":     "invalid_params",
			},
		}
	}

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: fmt.Sprintf("@shell(%q)", params.Command),
		Command:     params.Command,
		Metadata: map[string]string{
			"decorator": "shell",
			"command":   params.Command,
		},
	}
}

// Execute performs the actual shell command execution using validated parameters
func (s *ShellDecorator) Execute(ctx decorators.Context, validated any) (decorators.CommandResult, error) {
	params, ok := validated.(ShellParams)
	if !ok {
		return nil, fmt.Errorf("@shell: invalid parameters")
	}

	if params.Command == "" {
		return nil, fmt.Errorf("@shell: empty command")
	}

	result := ctx.ExecShell(params.Command)
	return result, nil
}

// RequiresBlock returns the block requirements for shell decorator
func (s *ShellDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockNone,
		Required: false,
	}
}

// Register the shell decorator globally
func init() {
	decorators.RegisterExecutionDecorator(NewShellDecorator())
}
