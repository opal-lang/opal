package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var (
		file    string
		dryRun  bool
		debug   bool
		noColor bool
	)

	rootCmd := &cobra.Command{
		Use:   "opal [command]",
		Short: "Execute commands defined in opal files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, file, dryRun, debug, noColor)
		},
	}

	// Add flags
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", "commands.cli", "Path to command definitions file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(cmd *cobra.Command, args []string, file string, dryRun, debug, noColor bool) error {
	commandName := args[0]

	// Get input reader based on file options
	reader, closeFunc, err := getInputReader(file)
	if err != nil {
		return err
	}
	defer func() { _ = closeFunc() }()

	// TODO: Execute using new runtime implementation
	_ = reader // Ignore unused reader for now
	fmt.Fprintf(os.Stderr, "Runtime execution not yet implemented - AST redesign in progress\n")
	fmt.Fprintf(os.Stderr, "Command: %s, File: %s, DryRun: %t\n", commandName, file, dryRun)
	fmt.Fprintf(os.Stderr, "This is expected during the transition to new AST implementation\n")
	return nil // Return success during transition period
}

// getInputReader handles the 3 modes of input:
// 1. Explicit stdin with -f -
// 2. Piped input (auto-detected when using default file)
// 3. File input (specific file or default commands.cli)
func getInputReader(file string) (io.Reader, func() error, error) {
	// Mode 1: Explicit stdin
	if file == "-" {
		return os.Stdin, func() error { return nil }, nil
	}

	// Mode 2: Check for piped input when using default file
	if file == "commands.cli" {
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
	// Note: We don't check Size() > 0 because pipes may not report size correctly
	return (stat.Mode() & os.ModeCharDevice) == 0
}
