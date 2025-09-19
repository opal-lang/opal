package builtin

import (
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// ShellDecorator implements the @shell decorator for direct shell command execution.
// This is the foundational decorator that enables the "shell syntax as decorator sugar" concept.
// All shell commands like "npm install" become "@shell('npm install')" internally.
type ShellDecorator struct{}

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

// Plan generates an execution plan for the shell command
func (s *ShellDecorator) Plan(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	command, _ := decorators.ExtractPositionalString(args, 0, "")

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: "@shell",
		Command:     command,
		Metadata: map[string]string{
			"decorator":      "shell",
			"execution_mode": "direct",
			"color":          plan.ColorGreen,
		},
	}
}

// Execute performs the actual shell command execution
func (s *ShellDecorator) Execute(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	command, _ := decorators.ExtractPositionalString(args, 0, "")

	if command == "" {
		// Return a simple error result - we'll need to implement this
		return &simpleCommandResult{
			stdout:   "",
			stderr:   "shell decorator: no command provided",
			exitCode: 1,
		}
	}

	return ctx.ExecShell(command)
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
