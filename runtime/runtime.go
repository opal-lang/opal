package runtime

import (
	"fmt"
	"io"
	"strings"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/errors"
	"github.com/aledsdavies/devcmd/core/ir"
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin" // Import for decorator registration
	"github.com/aledsdavies/devcmd/runtime/execution"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
	"github.com/aledsdavies/devcmd/runtime/execution/plan"
	"github.com/aledsdavies/devcmd/runtime/parser"
	"github.com/aledsdavies/devcmd/runtime/validation"
)

// ExecutionOptions configures how commands are executed
type ExecutionOptions struct {
	Command     string    // Command name to execute
	DryRun      bool      // Generate execution plan instead of running
	Debug       bool      // Enable debug output
	NoColor     bool      // Disable colored output
	AutoConfirm bool      // Auto-confirm prompts
	Stdout      io.Writer // Where to write stdout
	Stderr      io.Writer // Where to write stderr
}

// Execute parses command definitions from source and executes the specified command
func Execute(source io.Reader, opts ExecutionOptions) error {
	// Parse the command definitions
	program, err := parser.Parse(source)
	if err != nil {
		return errors.NewParseError("Failed to parse command definitions", err)
	}

	// Validate program for recursion
	if err := validation.ValidateNoRecursion(program); err != nil {
		return errors.NewParseError("Command validation failed", err)
	}

	return ExecuteWithProgram(program, opts)
}

// ExecuteWithProgram executes a command with a pre-parsed AST program
func ExecuteWithProgram(program *ast.Program, opts ExecutionOptions) error {
	// Find the target command
	targetCommand := findCommand(program, opts.Command)
	if targetCommand == nil {
		return createCommandNotFoundError(opts.Command, program.Commands)
	}

	// Transform all commands to IR for @cmd decorator support
	commands, err := transformAllCommands(program.Commands)
	if err != nil {
		return fmt.Errorf("failed to transform commands to IR: %w", err)
	}

	// Transform target command to IR
	irNode, err := execution.TransformCommand(targetCommand)
	if err != nil {
		return fmt.Errorf("failed to transform command to IR: %w", err)
	}

	// Extract variables from program
	vars := extractVariables(program)

	// Create execution context
	ctx, err := execution.NewCtx(execution.CtxOptions{
		EnvOptions: execution.EnvOptions{
			BlockList: []string{"PWD", "OLDPWD", "SHLVL", "RANDOM", "PS*", "TERM"},
		},
		Vars:     vars,
		DryRun:   opts.DryRun,
		Debug:    opts.Debug,
		Commands: commands,
		UIConfig: &context.UIConfig{
			NoColor:     opts.NoColor,
			AutoConfirm: opts.AutoConfirm,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create execution context: %w", err)
	}

	// Handle dry-run mode with plan generation
	if opts.DryRun {
		return generateAndDisplayPlan(ctx, irNode, opts)
	}

	// Execute in normal mode
	return executeCommand(ctx, irNode, opts)
}

// findCommand locates a command by name in the program
func findCommand(program *ast.Program, commandName string) *ast.CommandDecl {
	for i := range program.Commands {
		if program.Commands[i].Name == commandName {
			return &program.Commands[i]
		}
	}
	return nil
}

// createCommandNotFoundError creates a detailed error for missing commands
func createCommandNotFoundError(commandName string, commands []ast.CommandDecl) error {
	var availableCommands []string
	for _, command := range commands {
		availableCommands = append(availableCommands, command.Name)
	}

	if len(availableCommands) == 0 {
		return errors.New(errors.ErrNoCommandsDefined,
			fmt.Sprintf("Command '%s' not found: no commands are defined in the file", commandName)).
			WithContext("command", commandName)
	}

	return errors.NewCommandNotFoundError(commandName, availableCommands)
}

// transformAllCommands transforms all commands to IR for cross-command references
func transformAllCommands(commands []ast.CommandDecl) (map[string]ir.Node, error) {
	commandMap := make(map[string]ir.Node)
	for _, command := range commands {
		irNode, err := execution.TransformCommand(&command)
		if err != nil {
			return nil, fmt.Errorf("failed to transform command '%s': %w", command.Name, err)
		}
		commandMap[command.Name] = irNode
	}
	return commandMap, nil
}

// extractVariables extracts variables from the program AST safely
func extractVariables(program *ast.Program) map[string]string {
	vars := make(map[string]string)
	for _, variable := range program.Variables {
		if variable.Value != nil {
			if stringLit, ok := variable.Value.(*ast.StringLiteral); ok && stringLit != nil {
				// Check if Parts is properly initialized before calling String()
				if stringLit.Parts != nil {
					safeString := extractStringLiteralSafely(stringLit)
					vars[variable.Name] = safeString
				} else {
					// Fallback to Raw field if Parts is nil
					vars[variable.Name] = stringLit.Raw
				}
			}
		}
	}
	return vars
}

// extractStringLiteralSafely safely extracts string content from StringLiteral
func extractStringLiteralSafely(s *ast.StringLiteral) string {
	if s == nil || s.Parts == nil {
		return s.Raw
	}

	var result strings.Builder
	for _, part := range s.Parts {
		if part != nil {
			result.WriteString(part.String())
		}
	}
	return result.String()
}

// generateAndDisplayPlan generates and displays an execution plan for dry-run mode
func generateAndDisplayPlan(ctx *context.Ctx, irNode ir.Node, opts ExecutionOptions) error {
	// Create plan generator
	registry := decorators.GlobalRegistry()
	generator := plan.NewGenerator(registry)

	// Generate execution plan
	executionPlan := generator.GenerateFromIR(ctx, irNode, opts.Command)

	// Display the plan using the built-in formatter
	if opts.NoColor {
		_, _ = fmt.Fprint(opts.Stdout, executionPlan.StringNoColor())
	} else {
		_, _ = fmt.Fprint(opts.Stdout, executionPlan.String())
	}
	_, _ = fmt.Fprint(opts.Stdout, "\n") // Add trailing newline

	if opts.Debug {
		_, _ = fmt.Fprintf(opts.Stderr, "[DEBUG] Generated plan with %d steps\n", len(executionPlan.Steps))
	}

	return nil
}

// executeCommand executes the command in normal mode
func executeCommand(ctx *context.Ctx, irNode ir.Node, opts ExecutionOptions) error {
	// Create evaluator with decorator registry
	registry := decorators.GlobalRegistry()
	evaluator := execution.NewNodeEvaluator(registry)

	// Execute the IR node
	result := evaluator.ExecuteNode(ctx, irNode)

	// Write outputs to provided writers
	if result.Stdout != "" {
		_, _ = fmt.Fprint(opts.Stdout, result.Stdout)
	}
	if result.Stderr != "" {
		_, _ = fmt.Fprint(opts.Stderr, result.Stderr)
	}

	if opts.Debug {
		_, _ = fmt.Fprintf(opts.Stderr, "[DEBUG] Command completed with exit code: %d\n", result.ExitCode)
	}

	// Return error if command failed
	if result.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}

	return nil
}
