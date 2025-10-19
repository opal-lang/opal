package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk/secret"
	_ "github.com/aledsdavies/opal/runtime/decorators" // Register built-in decorators
	"github.com/aledsdavies/opal/runtime/executor"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
	"github.com/aledsdavies/opal/runtime/streamscrub"
	"github.com/spf13/cobra"
)

func main() {
	// CRITICAL: Lock down stdout/stderr at CLI entry point
	// This ensures even lexer/parser/planner cannot leak secrets
	var outputBuf bytes.Buffer

	// Create Opal-specific placeholder generator (format: opal:s:hash)
	opalGen, err := streamscrub.NewOpalPlaceholderGenerator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: failed to create placeholder generator: %v\n", err)
		os.Exit(1)
	}

	// Create scrubber with Opal placeholders
	scrubber := streamscrub.New(&outputBuf, streamscrub.WithPlaceholderFunc(opalGen.PlaceholderFunc()))

	// Redirect all stdout/stderr through scrubber
	restore := scrubber.LockdownStreams()

	var (
		file     string
		planFile string
		dryRun   bool
		resolve  bool
		debug    bool
		noColor  bool
		timing   bool
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
			exitCode, err := runCommand(cmd, args, file, dryRun, resolve, debug, noColor, timing, scrubber, &outputBuf)
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
	rootCmd.PersistentFlags().BoolVar(&timing, "timing", false, "Show pipeline timing breakdown")

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

func runCommand(cmd *cobra.Command, args []string, file string, dryRun, resolve, debug, noColor, timing bool, scrubber *streamscrub.Scrubber, outputBuf *bytes.Buffer) (int, error) {
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

	// Parse with telemetry if timing enabled
	var tree *parser.ParseTree
	var pipelineTiming struct {
		ParseTime   time.Duration
		PlanTime    time.Duration
		ExecuteTime time.Duration
	}

	if timing {
		tree = parser.Parse(source, parser.WithTelemetryTiming())
		if tree.Telemetry != nil {
			pipelineTiming.ParseTime = tree.Telemetry.TotalTime
		}
	} else {
		tree = parser.Parse(source)
	}
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

	// Plan with telemetry if timing enabled
	var plan *planfmt.Plan
	if timing {
		planResult, err := planner.PlanWithObservability(tree.Events, tokens, planner.Config{
			Target:    commandName,
			IDFactory: idFactory,
			Debug:     debugLevel,
			Telemetry: planner.TelemetryTiming,
		})
		if err != nil {
			return 1, fmt.Errorf("planning failed: %w", err)
		}
		plan = planResult.Plan
		pipelineTiming.PlanTime = planResult.PlanTime
	} else {
		var err error
		plan, err = planner.Plan(tree.Events, tokens, planner.Config{
			Target:    commandName,
			IDFactory: idFactory,
			Debug:     debugLevel,
		})
		if err != nil {
			return 1, fmt.Errorf("planning failed: %w", err)
		}
	}

	// Register all secrets with scrubber (ALL value decorator results are secrets)
	for _, secret := range plan.Secrets {
		// Use DisplayID as placeholder (e.g., "opal:secret:3J98t56A")
		scrubber.RegisterSecret([]byte(secret.RuntimeValue), []byte(secret.DisplayID))
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

	// Execute with telemetry level based on timing flag
	telemetryLevel := executor.TelemetryBasic
	if timing {
		telemetryLevel = executor.TelemetryTiming
	}

	result, err := executor.Execute(steps, executor.Config{
		Debug:     execDebug,
		Telemetry: telemetryLevel,
	})
	if err != nil {
		return 1, fmt.Errorf("execution failed: %w", err)
	}

	pipelineTiming.ExecuteTime = result.Duration

	// Print timing breakdown if timing flag enabled
	if timing {
		displayPipelineTiming(pipelineTiming, result)
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
func runFromPlan(planFile, sourceFile string, debug, noColor bool, scrubber *streamscrub.Scrubber, outputBuf *bytes.Buffer) (int, error) {
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
		scrubber.RegisterSecret([]byte(secret.RuntimeValue), []byte(secret.DisplayID))
	}

	// Step 4: Execute the verified plan
	execDebug := executor.DebugOff
	if debug {
		execDebug = executor.DebugDetailed
	}

	// Convert plan to SDK steps at the boundary
	steps := planfmt.ToSDKSteps(freshPlan.Steps)

	result, err := executor.Execute(steps, executor.Config{
		Debug:     execDebug,
		Telemetry: executor.TelemetryBasic,
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

// displayPipelineTiming shows a breakdown of pipeline timing
func displayPipelineTiming(timing struct {
	ParseTime   time.Duration
	PlanTime    time.Duration
	ExecuteTime time.Duration
}, result *executor.ExecutionResult,
) {
	totalTime := timing.ParseTime + timing.PlanTime + timing.ExecuteTime

	fmt.Fprintf(os.Stderr, "\nPipeline Timing:\n")
	fmt.Fprintf(os.Stderr, "  Parse:   %v\n", timing.ParseTime)
	fmt.Fprintf(os.Stderr, "  Plan:    %v\n", timing.PlanTime)

	if result != nil && result.Telemetry != nil {
		fmt.Fprintf(os.Stderr, "  Execute: %v (%d steps)\n", timing.ExecuteTime, result.StepsRun)

		// Show per-step timing if available
		if len(result.Telemetry.StepTimings) > 0 {
			for _, st := range result.Telemetry.StepTimings {
				fmt.Fprintf(os.Stderr, "    Step %d: %v (exit %d)\n", st.StepID, st.Duration, st.ExitCode)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "  Execute: %v\n", timing.ExecuteTime)
	}

	fmt.Fprintf(os.Stderr, "  Total:   %v\n", totalTime)
}
