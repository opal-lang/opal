package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/planfmt/formatter"
	"github.com/aledsdavies/opal/runtime/planner"
)

// CLIError represents a formatted CLI error with context
type CLIError struct {
	Type    string // "parse", "plan", "contract", "execution"
	Message string
	Details string // Additional context
	Hint    string // How to fix it
}

// Error implements the error interface
func (e *CLIError) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if e.Details != "" {
		b.WriteString("\n")
		b.WriteString(e.Details)
	}
	if e.Hint != "" {
		b.WriteString("\n")
		b.WriteString(e.Hint)
	}
	return b.String()
}

// FormatError formats an error for CLI output with colors
func FormatError(w io.Writer, err error, useColor bool) {
	if err == nil {
		return
	}

	// Check for specific error types
	switch e := err.(type) {
	case *planner.PlanError:
		formatPlanError(w, e, useColor)
	case *CLIError:
		formatCLIError(w, e, useColor)
	default:
		// Generic error
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Colorize("Error: ", ColorRed, useColor), err.Error(), ColorReset)
	}
}

// formatPlanError formats planner errors with suggestions
func formatPlanError(w io.Writer, err *planner.PlanError, useColor bool) {
	_, _ = fmt.Fprintf(w, "%s%s%s\n", Colorize("Error: ", ColorRed, useColor), err.Message, ColorReset)

	if err.Context != "" {
		_, _ = fmt.Fprintf(w, "%sContext: %s%s\n", Colorize("  ", ColorGray, useColor), err.Context, ColorReset)
	}

	if err.Suggestion != "" {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Colorize("  ", ColorYellow, useColor), err.Suggestion, ColorReset)
	}

	if err.Example != "" {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Colorize("  ", ColorGray, useColor), err.Example, ColorReset)
	}
}

// formatCLIError formats CLI errors
func formatCLIError(w io.Writer, err *CLIError, useColor bool) {
	_, _ = fmt.Fprintf(w, "%s%s%s\n", Colorize("Error: ", ColorRed, useColor), err.Message, ColorReset)

	if err.Details != "" {
		_, _ = fmt.Fprintf(w, "\n%s\n", err.Details)
	}

	if err.Hint != "" {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", Colorize("Hint: ", ColorYellow, useColor), err.Hint, ColorReset)
	}
}

// FormatContractVerificationError formats contract verification failures with diff
func FormatContractVerificationError(w io.Writer, contractPlan, freshPlan *planfmt.Plan, useColor bool) {
	_, _ = fmt.Fprintf(w, "%sCONTRACT VERIFICATION FAILED%s\n\n", Colorize("", ColorRed, useColor), ColorReset)

	// Show detailed diff of what changed
	diff := formatter.Diff(contractPlan, freshPlan)
	diffOutput := formatter.FormatDiff(diff, useColor)
	_, _ = fmt.Fprint(w, diffOutput)
}
