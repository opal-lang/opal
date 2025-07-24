package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/engine"
	"github.com/aledsdavies/devcmd/pkgs/errors"
	"github.com/aledsdavies/devcmd/pkgs/parser"
	"github.com/spf13/cobra"
)

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
	noColor      bool
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
	// Check if stdin has data (is being piped to)
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		// Data is being piped to stdin
		return os.Stdin, func() error { return nil }, nil
	}

	// Fall back to reading from file
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
	// Get input reader (file or stdin)
	reader, closeFunc, err := getInputReader()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeFunc(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close input: %v\n", closeErr)
		}
	}()

	// Parse the command definitions
	program, err := parser.Parse(reader)
	if err != nil {
		return fmt.Errorf("error parsing commands: %w", err)
	}

	// Generate Go output using the engine
	eng := engine.New(program)
	genResult, err := eng.GenerateCode(program)
	if err != nil {
		return fmt.Errorf("error generating Go output: %w", err)
	}

	// If output directory specified, write files there
	if outputDir != "" {
		moduleName := strings.ReplaceAll(binaryName, "-", "_")
		if err := eng.WriteFiles(genResult, outputDir, moduleName); err != nil {
			return fmt.Errorf("error writing files: %w", err)
		}

		if debug {
			fmt.Fprintf(os.Stderr, "✅ Generated main.go and go.mod in %s\n", outputDir)
		}
	} else {
		// Default behavior: output main.go to stdout
		fmt.Print(genResult.String())
	}

	return nil
}

func buildCommand(cmd *cobra.Command, args []string) error {
	// Get input reader (file or stdin)
	reader, closeFunc, err := getInputReader()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeFunc(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close input: %v\n", closeErr)
		}
	}()

	program, err := parser.Parse(reader)
	if err != nil {
		return fmt.Errorf("error parsing commands: %w", err)
	}

	// Generate Go source code using the engine
	eng := engine.New(program)
	genResult, err := eng.GenerateCode(program)
	if err != nil {
		return fmt.Errorf("error generating Go source: %w", err)
	}

	// Determine output path
	outputPath := output
	if outputPath == "" {
		outputPath = "./" + binaryName
	}

	// Make output path absolute
	if !filepath.IsAbs(outputPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting working directory: %w", err)
		}
		outputPath = filepath.Join(wd, outputPath)
	}

	// Create temporary directory for build
	tempDir, err := os.MkdirTemp("", "devcmd-build-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove temp directory: %v\n", removeErr)
		}
	}()

	// Handle generate-only mode
	if generateOnly {
		if outputDir != "" {
			// Write files to specified directory
			moduleName := strings.ReplaceAll(binaryName, "-", "_")
			if err := eng.WriteFiles(genResult, outputDir, moduleName); err != nil {
				return fmt.Errorf("error writing source files: %w", err)
			}
			if debug {
				fmt.Fprintf(os.Stderr, "✅ Generated files written to: %s\n", outputDir)
			}
		} else {
			// Output main.go to stdout
			fmt.Print(genResult.Code.String())
		}
		return nil
	}

	// Use engine to write both files to temp directory
	moduleName := strings.ReplaceAll(binaryName, "-", "_")
	if err := eng.WriteFiles(genResult, tempDir, moduleName); err != nil {
		return fmt.Errorf("error writing source files: %w", err)
	}

	// Run go mod tidy to generate go.sum and download dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Stderr = os.Stderr
	if debug {
		fmt.Fprintf(os.Stderr, "Running go mod tidy...\n")
		tidyCmd.Stdout = os.Stderr
	}
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("error running go mod tidy: %w", err)
	}

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", outputPath, ".")
	buildCmd.Dir = tempDir
	buildCmd.Stderr = os.Stderr

	if debug {
		fmt.Fprintf(os.Stderr, "Building binary: %s\n", outputPath)
		buildCmd.Stdout = os.Stderr
	}

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("error building binary: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "✅ Successfully built: %s\n", outputPath)
	}

	return nil
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

	// Use the engine to execute the specific command
	eng := engine.New(program)

	if dryRun {
		// Execute in plan mode to show execution plan
		plan, err := eng.ExecuteCommandPlan(targetCommand)
		if err != nil {
			return errors.NewCommandExecutionError(commandName, err)
		}

		// Print the plan using the plan DSL's beautiful ASCII tree visualization
		if noColor {
			fmt.Print(plan.StringNoColor())
		} else {
			fmt.Print(plan.String())
		}
		return nil
	}

	// Execute the specific command normally
	cmdResult, err := eng.ExecuteCommand(targetCommand)
	if err != nil {
		return errors.NewCommandExecutionError(commandName, err)
	}

	if cmdResult.Status == "failed" {
		return errors.New(errors.ErrCommandExecution, fmt.Sprintf("Command '%s' failed: %s", commandName, cmdResult.Error)).
			WithContext("command", commandName).
			WithContext("error_details", cmdResult.Error)
	}

	return nil
}
