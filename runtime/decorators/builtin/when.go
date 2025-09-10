package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @when decorator on package import
func init() {
	decorators.RegisterPattern(NewWhenDecorator())
}

// WhenDecorator implements the @when decorator using the core decorator interfaces
type WhenDecorator struct{}

// NewWhenDecorator creates a new when decorator
func NewWhenDecorator() *WhenDecorator {
	return &WhenDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (w *WhenDecorator) Name() string {
	return "when"
}

// Description returns a human-readable description
func (w *WhenDecorator) Description() string {
	return "Conditionally execute commands based on environment variable pattern matching"
}

// ParameterSchema returns the expected parameters for this decorator
func (w *WhenDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "env",
			Type:        decorators.ArgTypeString,
			Required:    true,
			Description: "Environment variable name to match against",
		},
	}
}

// PatternSchema defines what patterns @when accepts
func (w *WhenDecorator) PatternSchema() decorators.PatternSchema {
	return decorators.PatternSchema{
		AllowedPatterns:     []string{}, // Any pattern is allowed
		RequiredPatterns:    []string{}, // No required patterns
		AllowsWildcard:      true,       // "default" wildcard pattern is allowed
		AllowsAnyIdentifier: true,       // Any identifier can be a pattern
		Description:         "Pattern branches that match the environment variable value",
	}
}

// Examples returns usage examples
func (w *WhenDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `deploy: @when(ENV) {
    prod: kubectl apply -f prod.yaml
    staging: kubectl apply -f staging.yaml
    dev: kubectl apply -f dev.yaml
    default: echo "Unknown environment: $ENV"
}`,
			Description: "Deploy based on ENV environment variable",
		},
		{
			Code: `build: @when(TARGET_OS) {
    linux: GOOS=linux go build
    windows: GOOS=windows go build  
    darwin: GOOS=darwin go build
    default: go build
}`,
			Description: "Conditional build based on TARGET_OS environment variable",
		},
		{
			Code: `test: @when(CI) {
    true: go test -race -coverprofile=coverage.out ./...
    default: go test ./...
}`,
			Description: "Different test behavior in CI vs local development",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (w *WhenDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// PATTERN DECORATOR METHODS
// ================================================================================================

// SelectBranch chooses and executes the appropriate branch based on environment variable matching
func (w *WhenDecorator) SelectBranch(ctx *decorators.Ctx, args []decorators.DecoratorParam, branches map[string]decorators.CommandSeq) decorators.CommandResult {
	envVar, err := w.extractEnvVar(args)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@when parameter error: %v", err),
			ExitCode: 1,
		}
	}

	// Get environment variable value from frozen environment
	value, exists := ctx.Env.Get(envVar)
	if !exists {
		value = "" // Treat missing env var as empty string
	}

	// Find matching branch
	selectedBranch := w.selectBranch(value, branches)
	if selectedBranch == "__NO_MATCH__" {
		// No matching branch - return error
		return decorators.CommandResult{
			Stdout:   "",
			Stderr:   fmt.Sprintf("no matching branch for %s=\"%s\"", envVar, value),
			ExitCode: 1,
		}
	}

	// Execute the selected branch
	branchCommands, exists := branches[selectedBranch]
	if !exists {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("internal error: branch %q not found", selectedBranch),
			ExitCode: 1,
		}
	}

	return ctx.ExecSequential(branchCommands.Steps)
}

// Describe returns description for dry-run display
func (w *WhenDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep {
	envVar, err := w.extractEnvVar(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: fmt.Sprintf("@when(<error: %v>)", err),
			Command:     "",
			Metadata: map[string]string{
				"decorator":      "when",
				"execution_mode": string(plan.ModeConditional),
				"color":          plan.ColorCyan,
				"error":          err.Error(),
			},
		}
	}

	// Get current environment variable value
	value, exists := ctx.Env.Get(envVar)
	if !exists {
		value = "" // Missing env var
	}

	// Determine which branch would be selected
	selectedBranch := w.selectBranchForPlan(value, branches)

	description := fmt.Sprintf("@when(%s)", envVar)
	if exists {
		description += fmt.Sprintf(" → %s=%q (selected: %s)", envVar, value, selectedBranch)
	} else {
		description += fmt.Sprintf(" → %s=<unset> (selected: %s)", envVar, selectedBranch)
	}

	// Add all branches as children, marking the selected one
	var children []plan.ExecutionStep
	for branchName, branchStep := range branches {
		if branchName == selectedBranch {
			branchStep.Description = "→ " + branchStep.Description // Mark as selected
		}
		children = append(children, branchStep)
	}

	// Build info string for display
	infoStr := fmt.Sprintf("{%s = %s → %s}", envVar, value, selectedBranch)

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: description,
		Command:     fmt.Sprintf("match $%s against patterns", envVar),
		Children:    children,
		Metadata: map[string]string{
			"decorator":      "when",
			"execution_mode": string(plan.ModeConditional),
			"color":          plan.ColorCyan,
			"info":           infoStr,
			"envVar":         envVar,
			"currentValue":   value,
			"exists":         fmt.Sprintf("%t", exists),
			"selectedBranch": selectedBranch,
			"branchCount":    fmt.Sprintf("%d", len(branches)),
		},
	}
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractEnvVar extracts the environment variable name parameter
func (w *WhenDecorator) extractEnvVar(params []decorators.DecoratorParam) (string, error) {
	if len(params) == 0 {
		return "", fmt.Errorf("@when requires an environment variable name")
	}

	var envVar string
	switch params[0].Name {
	case "":
		// Positional parameter
		if val, ok := params[0].Value.(string); ok {
			envVar = val
		} else {
			return "", fmt.Errorf("@when environment variable must be a string, got %T", params[0].Value)
		}
	case "env":
		// Named parameter
		if val, ok := params[0].Value.(string); ok {
			envVar = val
		} else {
			return "", fmt.Errorf("@when env parameter must be a string")
		}
	default:
		return "", fmt.Errorf("@when unknown parameter: %s", params[0].Name)
	}

	if envVar == "" {
		return "", fmt.Errorf("@when environment variable name cannot be empty")
	}

	return envVar, nil
}

// selectBranch finds the matching branch for an environment variable value
// Returns the branch name, or a special sentinel value if no match is found
func (w *WhenDecorator) selectBranch(value string, branches map[string]decorators.CommandSeq) string {
	// First, look for exact match
	if _, exists := branches[value]; exists {
		return value
	}

	// If no exact match, look for default
	if _, exists := branches["default"]; exists {
		return "default"
	}

	return "__NO_MATCH__" // No match found - use sentinel value
}

// selectBranchForPlan finds the matching branch for plan generation (doesn't need CommandSeq)
func (w *WhenDecorator) selectBranchForPlan(value string, branches map[string]plan.ExecutionStep) string {
	// First, look for exact match
	if _, exists := branches[value]; exists {
		return value
	}

	// If no exact match, look for wildcard
	if _, exists := branches["default"]; exists {
		return "default"
	}

	return "__NO_MATCH__" // No match found - use sentinel value
}
