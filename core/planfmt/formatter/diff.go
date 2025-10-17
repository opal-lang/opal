package formatter

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/opal/core/planfmt"
)

// DiffResult represents the differences between two plans.
type DiffResult struct {
	TargetChanged string     // Non-empty if target changed (format: "old -> new")
	Added         []StepDiff // Steps added in actual
	Removed       []StepDiff // Steps removed from expected
	Modified      []StepDiff // Steps that changed
}

// StepDiff represents a difference in a single step.
type StepDiff struct {
	StepNum  int    // Step number (1-indexed)
	Expected string // Formatted expected step (empty for added steps)
	Actual   string // Formatted actual step (empty for removed steps)
}

// Diff compares two plans and returns structured differences.
// Compares step-by-step to identify added, removed, and modified steps.
func Diff(expected, actual *planfmt.Plan) *DiffResult {
	result := &DiffResult{}

	// Check target change
	if expected.Target != actual.Target {
		result.TargetChanged = fmt.Sprintf("%s -> %s", expected.Target, actual.Target)
	}

	// Compare steps
	maxSteps := len(expected.Steps)
	if len(actual.Steps) > maxSteps {
		maxSteps = len(actual.Steps)
	}

	for i := 0; i < maxSteps; i++ {
		stepNum := i + 1

		// Step removed
		if i >= len(actual.Steps) {
			result.Removed = append(result.Removed, StepDiff{
				StepNum:  stepNum,
				Expected: FormatStep(&expected.Steps[i]),
				Actual:   "",
			})
			continue
		}

		// Step added
		if i >= len(expected.Steps) {
			result.Added = append(result.Added, StepDiff{
				StepNum:  stepNum,
				Expected: "",
				Actual:   FormatStep(&actual.Steps[i]),
			})
			continue
		}

		// Step potentially modified
		expectedStr := FormatStep(&expected.Steps[i])
		actualStr := FormatStep(&actual.Steps[i])

		if expectedStr != actualStr {
			result.Modified = append(result.Modified, StepDiff{
				StepNum:  stepNum,
				Expected: expectedStr,
				Actual:   actualStr,
			})
		}
	}

	return result
}

// FormatDiff returns a human-readable diff display.
// Shows added, removed, and modified steps with optional color coding.
func FormatDiff(result *DiffResult, useColor bool) string {
	var b strings.Builder

	// Color codes (ANSI)
	red := ""
	green := ""
	yellow := ""
	reset := ""
	if useColor {
		red = "\033[31m"
		green = "\033[32m"
		yellow = "\033[33m"
		reset = "\033[0m"
	}

	// Target change
	if result.TargetChanged != "" {
		fmt.Fprintf(&b, "%sTarget changed: %s%s\n\n", yellow, result.TargetChanged, reset)
	}

	// Modified steps
	if len(result.Modified) > 0 {
		fmt.Fprintf(&b, "%sModified steps:%s\n", yellow, reset)
		for _, diff := range result.Modified {
			fmt.Fprintf(&b, "  step %d:\n", diff.StepNum)
			fmt.Fprintf(&b, "    %s- %s%s\n", red, diff.Expected, reset)
			fmt.Fprintf(&b, "    %s+ %s%s\n", green, diff.Actual, reset)
		}
		fmt.Fprintln(&b)
	}

	// Added steps
	if len(result.Added) > 0 {
		fmt.Fprintf(&b, "%sAdded steps:%s\n", green, reset)
		for _, diff := range result.Added {
			fmt.Fprintf(&b, "  %s+ step %d: %s%s\n", green, diff.StepNum, diff.Actual, reset)
		}
		fmt.Fprintln(&b)
	}

	// Removed steps
	if len(result.Removed) > 0 {
		fmt.Fprintf(&b, "%sRemoved steps:%s\n", red, reset)
		for _, diff := range result.Removed {
			fmt.Fprintf(&b, "  %s- step %d: %s%s\n", red, diff.StepNum, diff.Expected, reset)
		}
		fmt.Fprintln(&b)
	}

	// Summary
	if len(result.Modified) == 0 && len(result.Added) == 0 && len(result.Removed) == 0 && result.TargetChanged == "" {
		fmt.Fprintln(&b, "No differences found.")
	}

	return b.String()
}
