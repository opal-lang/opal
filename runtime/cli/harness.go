package cli

import (
	"github.com/aledsdavies/devcmd/runtime/execution/context"
	"github.com/spf13/cobra"
)

// ================================================================================================
// THIN CLI HARNESS - Just argument parsing and runtime delegation
// ================================================================================================

// CLIHarness provides minimal Cobra CLI framework
type CLIHarness struct {
	rootCmd *cobra.Command
	dryRun  bool
	noColor bool
}

// NewCLIHarness creates a new minimal CLI harness
func NewCLIHarness(name, version string) *CLIHarness {
	rootCmd := &cobra.Command{
		Use:     name,
		Short:   "CLI from devcmd",
		Version: version,
	}

	harness := &CLIHarness{
		rootCmd: rootCmd,
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&harness.dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&harness.noColor, "no-color", false, "Disable colored output in dry-run mode")

	return harness
}

// Execute runs the CLI
func (h *CLIHarness) Execute() error {
	return h.rootCmd.Execute()
}

// GetRootCommand returns the root cobra command for customization
func (h *CLIHarness) GetRootCommand() *cobra.Command {
	return h.rootCmd
}

// CreateContext creates a runtime context from CLI flags
func (h *CLIHarness) CreateContext() (*context.Ctx, error) {
	opts := context.CtxOptions{
		DryRun: h.dryRun,
		// Other options can be added here as needed
	}

	return context.NewCtx(opts)
}
