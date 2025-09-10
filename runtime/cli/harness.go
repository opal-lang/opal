package cli

import (
	"fmt"
	"os"

	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/spf13/cobra"
)

// ================================================================================================
// STATIC CLI HARNESS - Minimal Cobra setup that works with generated commands
// ================================================================================================

// CommandFn represents a generated command function
type CommandFn func(*Ctx) CommandResult

// GeneratedCommand represents a command from the generated code
type GeneratedCommand struct {
	Name  string    `json:"name"`
	Short string    `json:"short"`
	Run   CommandFn `json:"-"`
}

// CLIHarness provides the static Cobra CLI framework
type CLIHarness struct {
	rootCmd   *cobra.Command
	globalCtx *Ctx
	dryRun    bool
	noColor   bool
}

// NewCLIHarness creates a new CLI harness
func NewCLIHarness(name, version string) *CLIHarness {
	ctx := NewCtx()

	rootCmd := &cobra.Command{
		Use:     name,
		Short:   "Generated CLI from devcmd",
		Version: version,
	}

	harness := &CLIHarness{
		rootCmd:   rootCmd,
		globalCtx: ctx,
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&harness.dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&harness.noColor, "no-color", false, "Disable colored output in dry-run mode")

	return harness
}

// RegisterCommands registers generated commands with the CLI
func (h *CLIHarness) RegisterCommands(commands []GeneratedCommand) {
	for _, cmd := range commands {
		h.addCommand(cmd)
	}
}

// addCommand adds a single command to the CLI
func (h *CLIHarness) addCommand(genCmd GeneratedCommand) {
	cmd := &cobra.Command{
		Use:   genCmd.Name,
		Short: genCmd.Short,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Update context with global flags
			ctx := h.globalCtx.Clone()
			ctx.DryRun = h.dryRun

			if ctx.DryRun {
				// In dry-run mode, we would call a plan function instead
				// For now, just show that we're in dry-run mode
				planOutput := fmt.Sprintf("Would execute: %s", genCmd.Name)
				if h.noColor {
					fmt.Print(planOutput + "\n")
				} else {
					fmt.Printf("\x1b[1m\x1b[34m%s\x1b[0m\n", planOutput)
				}
				return nil
			}

			// Execute the command
			result := genCmd.Run(ctx)

			// Handle the result
			if result.Stdout != "" {
				fmt.Print(result.Stdout)
			}
			if result.Stderr != "" {
				fmt.Fprint(os.Stderr, result.Stderr)
			}

			if result.Failed() {
				return fmt.Errorf("exit %d", result.ExitCode)
			}

			return nil
		},
	}

	h.rootCmd.AddCommand(cmd)
}

// Execute runs the CLI
func (h *CLIHarness) Execute() error {
	return h.rootCmd.Execute()
}

// GetRootCommand returns the root cobra command for customization
func (h *CLIHarness) GetRootCommand() *cobra.Command {
	return h.rootCmd
}

// ================================================================================================
// PLAN EXECUTION SUPPORT
// ================================================================================================

// PlanFn represents a generated plan function
type PlanFn func(*Ctx) *plan.ExecutionPlan

// GeneratedCommandWithPlan represents a command with both execution and planning
type GeneratedCommandWithPlan struct {
	Name  string    `json:"name"`
	Short string    `json:"short"`
	Run   CommandFn `json:"-"`
	Plan  PlanFn    `json:"-"`
}

// RegisterCommandsWithPlan registers commands that support both execution and planning
func (h *CLIHarness) RegisterCommandsWithPlan(commands []GeneratedCommandWithPlan) {
	for _, cmd := range commands {
		h.addCommandWithPlan(cmd)
	}
}

// addCommandWithPlan adds a command with plan support
func (h *CLIHarness) addCommandWithPlan(genCmd GeneratedCommandWithPlan) {
	cmd := &cobra.Command{
		Use:   genCmd.Name,
		Short: genCmd.Short,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Update context with global flags
			ctx := h.globalCtx.Clone()
			ctx.DryRun = h.dryRun

			if ctx.DryRun && genCmd.Plan != nil {
				// Execute plan mode
				executionPlan := genCmd.Plan(ctx)
				executionPlan.Context["command_name"] = genCmd.Name

				if h.noColor {
					fmt.Print(executionPlan.StringNoColor())
				} else {
					fmt.Print(executionPlan.String())
				}
				return nil
			}

			// Execute the command
			result := genCmd.Run(ctx)

			// Handle the result
			if result.Stdout != "" {
				fmt.Print(result.Stdout)
			}
			if result.Stderr != "" {
				fmt.Fprint(os.Stderr, result.Stderr)
			}

			if result.Failed() {
				return fmt.Errorf("exit %d", result.ExitCode)
			}

			return nil
		},
	}

	h.rootCmd.AddCommand(cmd)
}

// ================================================================================================
// HELPER FUNCTIONS FOR GENERATED CODE
// ================================================================================================

// ChainCommands executes commands in sequence with proper operator semantics
func ChainCommands(ctx *Ctx, elements ...ChainElement) CommandResult {
	if len(elements) == 0 {
		return CommandResult{ExitCode: 0}
	}

	var prev *CommandResult

	for i, element := range elements {
		// Determine if we should execute this element
		shouldExecute := true
		if i > 0 && prev != nil {
			prevElement := elements[i-1]
			switch prevElement.Operator {
			case "&&":
				shouldExecute = prev.Success()
			case "||":
				shouldExecute = prev.Failed()
			case "|":
				shouldExecute = true // Always execute for pipes
			}
		}

		if !shouldExecute {
			continue
		}

		// Execute the element
		var result CommandResult
		switch element.Type {
		case "shell":
			result = ExecShell(ctx, element.Command)
		case "action":
			// This would call the specific action decorator
			result = CommandResult{
				Stderr:   fmt.Sprintf("Action execution not implemented: %s", element.Action),
				ExitCode: 1,
			}
		default:
			result = CommandResult{
				Stderr:   fmt.Sprintf("Unknown element type: %s", element.Type),
				ExitCode: 1,
			}
		}

		prev = &result

		// Stop on failure unless next operator is OR
		if result.Failed() && i < len(elements)-1 && element.Operator != "||" {
			break
		}
	}

	if prev != nil {
		return *prev
	}
	return CommandResult{ExitCode: 0}
}

// ChainElement represents one element in a command chain
type ChainElement struct {
	Type     string // "shell" | "action"
	Command  string // Shell command text
	Action   string // Action decorator name
	Operator string // "&&" | "||" | "|" | ">>"
}

// Shell creates a shell command element
func Shell(cmd string) ChainElement {
	return ChainElement{Type: "shell", Command: cmd}
}

// Action creates an action decorator element
func Action(name string) ChainElement {
	return ChainElement{Type: "action", Action: name}
}

// And sets the && operator
func (e ChainElement) And() ChainElement {
	e.Operator = "&&"
	return e
}

// Or sets the || operator
func (e ChainElement) Or() ChainElement {
	e.Operator = "||"
	return e
}

// Pipe sets the | operator
func (e ChainElement) Pipe() ChainElement {
	e.Operator = "|"
	return e
}
