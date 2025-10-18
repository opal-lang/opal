package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk/secret"
	"github.com/aledsdavies/opal/runtime/executor"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
	"github.com/spf13/cobra"
)

func main() {
	// CRITICAL: Lock down stdout/stderr at CLI entry point
	// This ensures even lexer/parser/planner cannot leak secrets
	var outputBuf bytes.Buffer
	scrubber := executor.NewSecretScrubber(&outputBuf)

	// Redirect all stdout/stderr through scrubber
	restore := executor.LockDownStdStreams(&executor.LockdownConfig{
		Scrubber: scrubber,
	})

	var (
		file     string
		planFile string
		dryRun   bool
		resolve  bool
		debug    bool
		noColor  bool
	)

	rootCmd := &cobra.Command{
		Use:           "opal [command]",
		Short:         "Execute commands defined in opal files",
		Args:          cobra.MaximumNArgs(1), // 0 args if --plan, 1 arg otherwise
		SilenceErrors: true,                  // We handle error printing ourselves
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mode 4: Execute from plan file (contract verification)
			if planFile != "" {
				if len(args) > 0 {
					return fmt.Errorf("cannot specify command name with --plan flag")
				}
				exitCode, err := runFromPlan(planFile, file, debug, noColor, scrubber, &outputBuf)
				if err != nil {
					cmd.SilenceUsage = true // We've already printed detailed error
					return err
				}
				if exitCode != 0 {
					return fmt.Errorf("command failed with exit code %d", exitCode)
				}
				return nil
			}

			// Modes 1-3: Execute from source
			if len(args) != 1 {
				return fmt.Errorf("command name required (or use --plan)")
			}
			exitCode, err := runCommand(cmd, args, file, dryRun, resolve, debug, noColor, scrubber, &outputBuf)
			if err != nil {
				cmd.SilenceUsage = true // We've already printed detailed error
				return err
			}
			if exitCode != 0 {
				// Store exit code for later (can't os.Exit here - skips defers)
				return fmt.Errorf("command failed with exit code %d", exitCode)
			}
			return nil
		},
	}

	// Add flags
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", "commands.opl", "Path to command definitions file")
	rootCmd.PersistentFlags().StringVar(&planFile, "plan", "", "Execute from pre-generated plan file (Mode 4)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&resolve, "resolve", false, "Resolve all values in plan (use with --dry-run)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Execute command and capture exit code
	exitCode := 0
	if err := rootCmd.Execute(); err != nil {
		// Error messages go through scrubber
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitCode = 1
	}

	// CRITICAL: Restore streams BEFORE writing to real stdout
	restore()

	// Now write captured (and scrubbed) output to real stdout
	_, _ = os.Stdout.Write(outputBuf.Bytes())

	// Exit with proper code (after all cleanup)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runCommand(cmd *cobra.Command, args []string, file string, dryRun, resolve, debug, noColor bool, scrubber *executor.SecretScrubber, outputBuf *bytes.Buffer) (int, error) {
	commandName := args[0]

	// Get input reader based on file options
	reader, closeFunc, err := getInputReader(file)
	if err != nil {
		return 1, err
	}
	defer func() { _ = closeFunc() }()

	// Read source
	source, err := io.ReadAll(reader)
	if err != nil {
		return 1, fmt.Errorf("error reading input: %w", err)
	}

	// Lex
	l := lexer.NewLexer()
	l.Init(source)
	tokens := l.GetTokens()

	// Parse
	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		// Use parser's error formatter for nice output
		formatter := &parser.ErrorFormatter{
			Source:   source,
			Filename: file,
			Compact:  false, // Use detailed format
			Color:    !noColor,
		}
		for _, parseErr := range tree.Errors {
			fmt.Fprint(os.Stderr, formatter.Format(parseErr))
		}
		return 1, fmt.Errorf("parse errors encountered")
	}

	// Plan
	debugLevel := planner.DebugOff
	if debug {
		debugLevel = planner.DebugDetailed
	}

	// Create IDFactory based on mode
	// - resolve mode (contract generation): use ModePlan for deterministic IDs
	// - direct execution: use ModeRun for random IDs (security)
	// Note: For contract generation, we need to plan twice:
	//   1. First plan to get the plan hash
	//   2. Second plan with deterministic IDFactory derived from plan hash
	// For MVP (no value decorators yet), we can skip this and just use nil
	var idFactory secret.IDFactory
	if !resolve {
		// Mode 1: Direct execution - use random IDs for security
		var err error
		idFactory, err = planfmt.NewRunIDFactory()
		if err != nil {
			return 1, fmt.Errorf("failed to create ID factory: %w", err)
		}
	}
	// For resolve mode, leave idFactory as nil for now (MVP has no value decorators)
	// When we add value decorators, we'll need to plan twice to get deterministic IDs

	plan, err := planner.Plan(tree.Events, tokens, planner.Config{
		Target:    commandName,
		IDFactory: idFactory,
		Debug:     debugLevel,
	})
	if err != nil {
		return 1, fmt.Errorf("planning failed: %w", err)
	}

	// Register all secrets with scrubber (ALL value decorator results are secrets)
	for _, secret := range plan.Secrets {
		// Use DisplayID as placeholder (e.g., "opal:secret:3J98t56A")
		scrubber.RegisterSecret(secret.RuntimeValue, secret.DisplayID)
	}

	// Dry-run mode: show plan or generate contract
	if dryRun {
		if resolve {
			// Mode 3: Resolved Plan (Contract Generation)
			// Generate plan hash and write minimal contract file
			// Note: In MVP, we don't actually resolve values yet (no value decorators)
			// but the infrastructure is ready for when we add them

			// Compute plan hash by serializing to buffer
			var planBuf bytes.Buffer
			planHash, err := planfmt.Write(&planBuf, plan)
			if err != nil {
				return 1, fmt.Errorf("failed to compute plan hash: %w", err)
			}

			// Write contract to stdout (target + hash + full plan)
			// Note: Don't write messages to stderr here - they go through lockdown
			// and end up in the output buffer along with the contract
			if err := planfmt.WriteContract(os.Stdout, commandName, planHash, plan); err != nil {
				return 1, fmt.Errorf("failed to write contract: %w", err)
			}

			// Debug output would also go to the contract file, so skip it
			// Users can use --dry-run without --resolve to see plan details
		} else {
			// Mode 2: Quick Plan (Dry-Run)
			// Display plan as tree
			DisplayPlan(os.Stdout, plan, !noColor)
		}
		return 0, nil
	}

	// Execute (lockdown already active from main())
	execDebug := executor.DebugOff
	if debug {
		execDebug = executor.DebugDetailed
	}

	// Convert plan to SDK steps at the boundary
	// The executor only sees SDK types - it has no knowledge of planfmt
	steps := planfmt.ToSDKSteps(plan.Steps)

	result, err := executor.Execute(steps, executor.Config{
		Debug:              execDebug,
		Telemetry:          executor.TelemetryBasic,
		LockdownStdStreams: false, // Already locked down at CLI level
	})
	if err != nil {
		return 1, fmt.Errorf("execution failed: %w", err)
	}

	// Print execution summary if debug enabled
	if debug {
		fmt.Fprintf(os.Stderr, "\nExecution summary:\n")
		fmt.Fprintf(os.Stderr, "  Steps run: %d/%d\n", result.StepsRun, len(steps))
		fmt.Fprintf(os.Stderr, "  Duration: %v\n", result.Duration)
		fmt.Fprintf(os.Stderr, "  Exit code: %d\n", result.ExitCode)
	}

	// Return exit code to main (don't call os.Exit - skips defers!)
	return result.ExitCode, nil
}

// getInputReader handles the 3 modes of input:
// 1. Explicit stdin with -f -
// 2. Piped input (auto-detected when using default file)
// 3. File input (specific file or default commands.opl)
func getInputReader(file string) (io.Reader, func() error, error) {
	// Mode 1: Explicit stdin
	if file == "-" {
		return os.Stdin, func() error { return nil }, nil
	}

	// Mode 2: Check for piped input when using default file
	if file == "commands.opl" {
		if hasPipedInput() {
			return os.Stdin, func() error { return nil }, nil
		}
	}

	// Mode 3: File input
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file %s: %w", file, err)
	}

	closeFunc := func() error {
		return f.Close()
	}

	return f, closeFunc, nil
}

// hasPipedInput detects if there's data piped to stdin
func hasPipedInput() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is not a character device (i.e., it's piped)
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// runFromPlan executes with contract verification (Mode 4: Contract Execution)
// Flow: Load contract → Replan fresh → Compare hashes → Execute if match
func runFromPlan(planFile, sourceFile string, debug, noColor bool, scrubber *executor.SecretScrubber, outputBuf *bytes.Buffer) (int, error) {
	// Step 1: Load contract from plan file
	f, err := os.Open(planFile)
	if err != nil {
		return 1, fmt.Errorf("failed to open plan file: %w", err)
	}
	defer func() { _ = f.Close() }()

	target, contractHash, contractPlan, err := planfmt.ReadContract(f)
	if err != nil {
		return 1, fmt.Errorf("failed to read contract: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Loaded contract from %s\n", planFile)
		fmt.Fprintf(os.Stderr, "Contract hash: %x\n", contractHash)
		fmt.Fprintf(os.Stderr, "Target: %s\n", target)
		fmt.Fprintf(os.Stderr, "Contract plan steps: %d\n", len(contractPlan.Steps))
	}

	// Step 2: Replan from current source
	reader, closeFunc, err := getInputReader(sourceFile)
	if err != nil {
		return 1, err
	}
	defer func() { _ = closeFunc() }()

	source, err := io.ReadAll(reader)
	if err != nil {
		return 1, fmt.Errorf("error reading source: %w", err)
	}

	// Lex
	l := lexer.NewLexer()
	l.Init(source)
	tokens := l.GetTokens()

	// Parse
	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		// Use parser's error formatter for nice output
		formatter := &parser.ErrorFormatter{
			Source:   source,
			Filename: sourceFile,
			Compact:  false, // Use detailed format
			Color:    !noColor,
		}
		for _, parseErr := range tree.Errors {
			fmt.Fprint(os.Stderr, formatter.Format(parseErr))
		}
		return 1, fmt.Errorf("parse errors in source (contract verification failed)")
	}

	// Plan (use same target as contract)
	debugLevel := planner.DebugOff
	if debug {
		debugLevel = planner.DebugDetailed
	}

	// For contract verification, we want deterministic IDs
	// But we need the plan hash first to create the IDFactory
	// For MVP (no value decorators), we can use nil
	// When we add value decorators, we'll need to plan twice
	freshPlan, err := planner.Plan(tree.Events, tokens, planner.Config{
		Target:    target,
		IDFactory: nil, // MVP: no value decorators yet
		Debug:     debugLevel,
	})
	if err != nil {
		return 1, fmt.Errorf("planning failed: %w", err)
	}

	// Step 3: Compare hashes (contract verification)
	var freshHashBuf bytes.Buffer
	freshHash, err := planfmt.Write(&freshHashBuf, freshPlan)
	if err != nil {
		return 1, fmt.Errorf("failed to hash fresh plan: %w", err)
	}

	if freshHash != contractHash {
		// Use error formatter for consistent output
		FormatContractVerificationError(os.Stderr, contractPlan, freshPlan, !noColor)

		// Show hashes for debugging
		if debug {
			fmt.Fprintf(os.Stderr, "\n%s\n", Colorize("Debug info:", ColorCyan, !noColor))
			fmt.Fprintf(os.Stderr, "  Contract hash: %x\n", contractHash)
			fmt.Fprintf(os.Stderr, "  Fresh hash:    %x\n", freshHash)
		}

		return 1, fmt.Errorf("contract verification failed")
	}

	if debug {
		fmt.Fprintf(os.Stderr, "✓ Contract verified (hash matches)\n")
		fmt.Fprintf(os.Stderr, "Steps: %d\n", len(freshPlan.Steps))
	}

	// Register all secrets with scrubber
	for _, secret := range freshPlan.Secrets {
		scrubber.RegisterSecret(secret.RuntimeValue, secret.DisplayID)
	}

	// Step 4: Execute the verified plan
	execDebug := executor.DebugOff
	if debug {
		execDebug = executor.DebugDetailed
	}

	// Convert plan to SDK steps at the boundary
	steps := planfmt.ToSDKSteps(freshPlan.Steps)

	result, err := executor.Execute(steps, executor.Config{
		Debug:              execDebug,
		Telemetry:          executor.TelemetryBasic,
		LockdownStdStreams: false, // Already locked down at CLI level
	})
	if err != nil {
		return 1, fmt.Errorf("execution failed: %w", err)
	}

	// Print execution summary if debug enabled
	if debug {
		fmt.Fprintf(os.Stderr, "\nExecution summary:\n")
		fmt.Fprintf(os.Stderr, "  Steps run: %d/%d\n", result.StepsRun, len(steps))
		fmt.Fprintf(os.Stderr, "  Duration: %v\n", result.Duration)
		fmt.Fprintf(os.Stderr, "  Exit code: %d\n", result.ExitCode)
	}

	return result.ExitCode, nil
}
