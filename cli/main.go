package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aledsdavies/devcmd/cli/internal/parser"
	"github.com/aledsdavies/devcmd/cli/internal/validation"
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/errors"
	"github.com/aledsdavies/devcmd/runtime/ast"
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin" // Import for decorator registration
	"github.com/aledsdavies/devcmd/runtime/execution"
	"github.com/aledsdavies/devcmd/runtime/execution/plan"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/spf13/cobra"
)

// mapCommandResolver implements BaseCommandResolver interface for plan generation
type mapCommandResolver struct {
	commands map[string]ir.Node
}

func (r *mapCommandResolver) GetCommand(name string) (ir.Node, error) {
	if command, exists := r.commands[name]; exists {
		return command, nil
	}
	return nil, fmt.Errorf("command %q not found", name)
}

// Build-time variables - can be set via ldflags
var (
	Version   string = "dev"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
)

// Global flags
var (
	commandsFile string
	templateFile string
	binaryName   string
	output       string
	debug        bool
	outputDir    string
	generateOnly bool
	dryRun       bool
	resolve      bool
	noColor      bool

	// Standardized UI control flags
	colorMode   string // auto, always, never
	quiet       bool   // minimal output (errors only)
	verbose     bool   // extra debugging output
	interactive string // auto, always, never
	autoConfirm bool   // auto-confirm all prompts (--yes)
	ci          bool   // CI mode (implies --no-interactive --no-color)
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		formatAndPrintError(err)
		os.Exit(1)
	}
}

// formatAndPrintError formats and prints errors in a user-friendly way
func formatAndPrintError(err error) {
	if devErr, ok := err.(*errors.DevCmdError); ok {
		// Handle structured DevCmd errors
		switch devErr.GetType() {
		case errors.ErrCommandNotFound:
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			if commands, exists := devErr.GetContext("available_commands"); exists {
				if cmdList, ok := commands.([]string); ok && len(cmdList) > 0 {
					fmt.Fprintf(os.Stderr, "💡 Available commands: %v\n", cmdList)
				}
			}
		case errors.ErrNoCommandsDefined:
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			fmt.Fprintf(os.Stderr, "💡 Create a commands.cli file or pipe command definitions to stdin\n")
		case errors.ErrCommandExecution:
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			if details, exists := devErr.GetContext("error_details"); exists {
				fmt.Fprintf(os.Stderr, "   Details: %v\n", details)
			}
		case errors.ErrVariableNotFound:
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			if varName, exists := devErr.GetContext("variable"); exists {
				fmt.Fprintf(os.Stderr, "💡 Make sure the variable '%s' is defined before using it\n", varName)
			}
		case errors.ErrInputRead:
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			if devErr.Cause != nil {
				fmt.Fprintf(os.Stderr, "   Cause: %v\n", devErr.Cause)
			}
		case errors.ErrFileParse:
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			if devErr.Cause != nil {
				fmt.Fprintf(os.Stderr, "   Parse error: %v\n", devErr.Cause)
			}
		default:
			// Generic structured error
			fmt.Fprintf(os.Stderr, "❌ %s\n", devErr.Message)
			if devErr.Cause != nil {
				fmt.Fprintf(os.Stderr, "   Cause: %v\n", devErr.Cause)
			}
		}
	} else {
		// Handle regular errors
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
	}
}

// getInputReader returns a reader for the command definitions, supporting both files and stdin
func getInputReader() (io.Reader, func() error, error) {
	// Only use stdin if explicitly requested (like with "-f -") or if data is being piped AND no file was explicitly set
	if commandsFile == "-" {
		// Explicitly requested stdin
		return os.Stdin, func() error { return nil }, nil
	}

	// Check if data is being piped to stdin AND we're using the default file
	// (This means user ran: echo "commands" | devcmd run cmd)
	if commandsFile == "commands.cli" { // default value - check if stdin has piped data
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 && stat.Size() > 0 {
			// Data is being piped to stdin and there's actual content
			return os.Stdin, func() error { return nil }, nil
		}
	}

	// Read from specified file (including default "commands.cli")
	file, err := os.Open(commandsFile)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file %s: %w", commandsFile, err)
	}

	closeFunc := func() error {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", closeErr)
		}
		return nil
	}

	return file, closeFunc, nil
}

var rootCmd = &cobra.Command{
	Use:   "devcmd [flags]",
	Short: "Generate Go CLI applications from command definitions",
	Long: `devcmd generates standalone Go CLI executables from simple command definition files.
It reads .cli files containing command definitions and outputs Go source code or compiled binaries.
By default, it looks for commands.cli in the current directory.`,
	Args:          cobra.NoArgs,
	RunE:          generateCommand,
	SilenceUsage:  true, // Don't show usage on execution errors
	SilenceErrors: true, // Don't let Cobra print errors, we'll handle them
}

var buildCmd = &cobra.Command{
	Use:   "build [flags]",
	Short: "Build CLI binary from command definitions",
	Long: `Build a compiled Go CLI binary from command definitions.
This generates the Go source code and compiles it into an executable binary.
By default, it looks for commands.cli in the current directory.`,
	Args:         cobra.NoArgs,
	RunE:         buildCommand,
	SilenceUsage: true, // Don't show usage on execution errors
}

var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run a command directly from command definitions",
	Long: `Execute a command directly from the CLI file without compilation.
This interprets and runs the command immediately, useful for development and testing.
By default, it looks for commands.cli in the current directory.`,
	Args:         cobra.MinimumNArgs(1),
	RunE:         runCommand,
	SilenceUsage: true, // Don't show usage on execution errors
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display version, build time, and git commit information for devcmd.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devcmd %s\n", Version)
		fmt.Printf("Built: %s\n", BuildTime)
		fmt.Printf("Commit: %s\n", GitCommit)
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&commandsFile, "file", "f", "commands.cli", "Path to commands file")
	rootCmd.PersistentFlags().StringVar(&templateFile, "template", "", "Custom template file for generation")
	rootCmd.PersistentFlags().StringVar(&binaryName, "binary", "dev", "Binary name for the generated CLI")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVar(&outputDir, "output-dir", "", "Directory to write generated files (default: stdout for main.go only)")

	// Standardized UI control flags (persistent across all commands)
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Control colored output: auto, always, never")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output (shorthand for --color=never)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output (errors only)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Extra debugging output")
	rootCmd.PersistentFlags().StringVar(&interactive, "interactive", "auto", "Control interactive prompts: auto, always, never")
	rootCmd.PersistentFlags().BoolVar(&autoConfirm, "yes", false, "Auto-confirm all prompts")
	rootCmd.PersistentFlags().BoolVar(&ci, "ci", false, "CI mode (optimized for CI environments)")

	// Add version flag support
	var showVersion bool
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Show version information")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Printf("devcmd %s\n", Version)
			fmt.Printf("Built: %s\n", BuildTime)
			fmt.Printf("Commit: %s\n", GitCommit)
			os.Exit(0)
		}

		// Process flag precedence and validation
		processStandardizedFlags()
	}

	// Build command specific flags
	buildCmd.Flags().StringVarP(&output, "output", "o", "", "Output binary path (default: ./<binary-name>)")
	buildCmd.Flags().BoolVar(&generateOnly, "generate-only", false, "Generate code only without building binary")

	// Run command specific flags
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running commands")
	runCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output in dry-run mode")

	// Add subcommands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

func generateCommand(cmd *cobra.Command, args []string) error {
	// TODO: Implement IR-based code generation
	return fmt.Errorf("code generation not yet implemented with new IR-based engine")
}

func buildCommand(cmd *cobra.Command, args []string) error {
	// TODO: Implement IR-based build command
	return fmt.Errorf("build command not yet implemented with new IR-based engine")
}

func runCommand(cmd *cobra.Command, args []string) error {
	commandName := args[0]

	// Get input reader (file or stdin)
	reader, closeFunc, err := getInputReader()
	if err != nil {
		return errors.NewInputError("Failed to read command definitions", err)
	}
	defer func() {
		if closeErr := closeFunc(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close input: %v\n", closeErr)
		}
	}()

	program, err := parser.Parse(reader)
	if err != nil {
		return errors.NewParseError("Failed to parse command definitions", err)
	}

	// Validate program for recursion before processing
	if err := validation.ValidateNoRecursion(program); err != nil {
		return errors.NewParseError("Command validation failed", err)
	}

	// Find the command to execute
	var targetCommand *ast.CommandDecl
	for i := range program.Commands {
		if program.Commands[i].Name == commandName {
			targetCommand = &program.Commands[i]
			break
		}
	}

	if targetCommand == nil {
		// List available commands
		var availableCommands []string
		for _, command := range program.Commands {
			availableCommands = append(availableCommands, command.Name)
		}
		if len(availableCommands) == 0 {
			return errors.New(errors.ErrNoCommandsDefined, fmt.Sprintf("Command '%s' not found: no commands are defined in the file", commandName)).
				WithContext("command", commandName)
		}
		return errors.NewCommandNotFoundError(commandName, availableCommands)
	}

	// Create evaluator with decorator registry
	registry := decorators.GlobalRegistry()
	evaluator := execution.NewNodeEvaluator(registry)

	// Transform all commands to IR for @cmd decorator support
	commands := make(map[string]ir.Node)
	for _, command := range program.Commands {
		irNode, err := ir.TransformCommand(&command)
		if err != nil {
			return fmt.Errorf("failed to transform command '%s' to IR: %w", command.Name, err)
		}
		commands[command.Name] = irNode
	}

	// Transform AST to IR
	irNode, err := ir.TransformCommand(targetCommand)
	if err != nil {
		return fmt.Errorf("failed to transform command to IR: %w", err)
	}

	// Create execution context with proper constructor
	ctx, err := ir.NewCtx(ir.CtxOptions{
		EnvOptions: ir.EnvOptions{
			BlockList: []string{"PWD", "OLDPWD", "SHLVL", "RANDOM", "PS*", "TERM"},
		},
		Vars:     extractVariables(program), // Extract variables from program AST
		DryRun:   dryRun,
		Debug:    debug,
		Commands: commands, // Add the commands map to context
		UIConfig: &ir.UIConfig{
			ColorMode:   colorMode,
			Quiet:       quiet,
			Verbose:     verbose,
			Interactive: interactive,
			AutoConfirm: autoConfirm,
			CI:          ci,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create execution context: %w", err)
	}

	// Handle dry-run mode with plan generation
	if dryRun {
		// Create a command resolver from the commands map
		commandResolver := &mapCommandResolver{commands: ctx.Commands}

		// Create plan generator and generate execution plan
		generator := plan.NewGeneratorWithResolver(registry, commandResolver)
		executionPlan := generator.GenerateFromIR(ctx, irNode, commandName)

		// Format and display the plan based on UI preferences
		var planOutput string
		if ctx.UIConfig.ColorMode == "never" || noColor {
			planOutput = executionPlan.StringNoColor()
		} else {
			planOutput = executionPlan.String()
		}

		// Use os.Stdout.WriteString for more reliable console output
		if _, err := os.Stdout.WriteString(planOutput); err != nil {
			return fmt.Errorf("failed to write plan output: %w", err)
		}

		// Ensure there's a newline at the end
		if len(planOutput) > 0 && !strings.HasSuffix(planOutput, "\n") {
			os.Stdout.WriteString("\n")
		}

		if debug {
			fmt.Printf("\n[DEBUG] Plan contains %d steps\n", len(executionPlan.Steps))
			fmt.Printf("[DEBUG] Plan context: %+v\n", executionPlan.Context)
		}

		return nil
	}

	// Execute the IR node in normal mode
	result := evaluator.EvaluateNode(ctx, irNode)

	if debug {
		fmt.Printf("[DEBUG] Result stdout: %q\n", result.Stdout)
		fmt.Printf("[DEBUG] Result stderr: %q\n", result.Stderr)
		fmt.Printf("[DEBUG] Result exit code: %d\n", result.ExitCode)
	}

	// Handle the result
	// Normal execution: output is already streamed live
	// Only print captured output if we're in quiet mode and there was an error
	if quiet && result.ExitCode != 0 {
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}

	return nil
}

// extractVariables extracts variables from the program AST and returns them as a string map
func extractVariables(program *ast.Program) map[string]string {
	vars := make(map[string]string)
	for _, variable := range program.Variables {
		if stringLit, ok := variable.Value.(*ast.StringLiteral); ok {
			vars[variable.Name] = stringLit.String()
		}
		// TODO: Handle other expression types (numbers, booleans, etc.) as needed
	}
	return vars
}

// processStandardizedFlags handles flag validation and precedence for UI control flags
func processStandardizedFlags() {
	// Handle --no-color shorthand
	if noColor {
		colorMode = "never"
	}

	// Validate color mode
	switch colorMode {
	case "auto", "always", "never":
		// Valid values
	default:
		fmt.Fprintf(os.Stderr, "❌ Invalid --color value: %s (must be: auto, always, never)\n", colorMode)
		os.Exit(1)
	}

	// Validate interactive mode
	switch interactive {
	case "auto", "always", "never":
		// Valid values
	default:
		fmt.Fprintf(os.Stderr, "❌ Invalid --interactive value: %s (must be: auto, always, never)\n", interactive)
		os.Exit(1)
	}

	// Handle --ci mode implications
	if ci {
		// CI mode implies non-interactive and no color
		interactive = "never"
		colorMode = "never"
		quiet = true // CI environments typically want minimal output
	}

	// Handle --yes flag implications
	if autoConfirm {
		// Auto-confirm implies we can be non-interactive for prompts
		// but don't override explicit --interactive settings
		if interactive == "auto" {
			interactive = "never"
		}
	}

	// Handle conflicting verbosity flags
	if quiet && verbose {
		fmt.Fprintf(os.Stderr, "❌ Cannot use both --quiet and --verbose flags\n")
		os.Exit(1)
	}

	// Set debug mode from verbose flag if debug wasn't explicitly set
	if verbose && !debug {
		debug = true
	}
}
