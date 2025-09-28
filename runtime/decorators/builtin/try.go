package builtin

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/core/ir"
	"github.com/aledsdavies/opal/core/plan"
	"github.com/aledsdavies/opal/runtime/execution/context"
)

// Register the @try decorator on package import
func init() {
	decorator := NewTryDecorator()
	decorators.RegisterPattern(decorator)
	// Note: @try stays on legacy interface, not migrating to generic interface
}

// TryDecorator implements the @try decorator using the core decorator interfaces
type TryDecorator struct{}

// NewTryDecorator creates a new try decorator
func NewTryDecorator() *TryDecorator {
	return &TryDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (t *TryDecorator) Name() string {
	return "try"
}

// Description returns a human-readable description
func (t *TryDecorator) Description() string {
	return "Execute commands with error handling using try/catch/finally pattern"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TryDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		// @try doesn't take parameters - it uses pattern branches
	}
}

// Validate checks if the provided patterns are valid for @try decorator
func (t *TryDecorator) Validate(patternNames []string) []error {
	var errors []error

	// Check for required "main" pattern
	hasMain := false
	for _, name := range patternNames {
		if name == "main" {
			hasMain = true
			break
		}
	}
	if !hasMain {
		errors = append(errors, fmt.Errorf("@try requires a 'main' pattern"))
	}

	// Check that all patterns are allowed
	allowed := map[string]bool{"main": true, "catch": true, "finally": true}
	for _, name := range patternNames {
		if !allowed[name] {
			errors = append(errors, fmt.Errorf("@try does not allow pattern '%s' (allowed: main, catch, finally)", name))
		}
	}

	return errors
}

// Examples returns usage examples
func (t *TryDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `deploy: @try {
    main: kubectl apply -f k8s/
    catch: kubectl rollback deployment/app
    finally: kubectl get pods
}`,
			Description: "Deploy with rollback on failure and status check",
		},
		{
			Code: `backup: @try {
    main: {
        mysqldump mydb > backup.sql
        aws s3 cp backup.sql s3://backups/
    }
    catch: {
        echo "Backup failed, cleaning up..."
        rm -f backup.sql
    }
    finally: echo "Backup operation completed"
}`,
			Description: "Database backup with cleanup on failure",
		},
		{
			Code: `test: @try {
    main: npm test
    catch: echo "Tests failed, but continuing..."
}`,
			Description: "Run tests with graceful failure handling",
		},
	}
}

// Note: ImportRequirements removed - will be added back when code generation is implemented

// ================================================================================================
// PATTERN DECORATOR METHODS
// ================================================================================================

// SelectBranch executes try/catch/finally logic
func (t *TryDecorator) SelectBranch(ctx *context.Ctx, args []decorators.Param, branches map[string]ir.CommandSeq) context.CommandResult {
	// Validate that we have required branches
	_, hasMain := branches["main"]
	if !hasMain {
		return context.CommandResult{
			Stderr:   "@try requires a 'main' branch",
			ExitCode: 1,
		}
	}

	// TODO: Runtime execution - implement when interpreter is rebuilt
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "interpreter not implemented",
		ExitCode: 1,
	}
}

// Describe returns description for dry-run display
func (t *TryDecorator) Describe(ctx decorators.Context, args []decorators.Param, branches map[string]plan.ExecutionStep) plan.ExecutionStep {
	description := "@try"

	var children []plan.ExecutionStep
	branchOrder := []string{"main", "catch", "finally"}

	// Add branches in logical order
	for _, branchName := range branchOrder {
		if branchStep, exists := branches[branchName]; exists {
			// Mark required vs optional branches
			if branchName == "main" {
				branchStep.Description = fmt.Sprintf("%s: %s", branchName, branchStep.Description)
			} else {
				branchStep.Description = fmt.Sprintf("%s (optional): %s", branchName, branchStep.Description)
			}
			children = append(children, branchStep)
		}
	}

	// Count available branches for metadata
	branchCount := 0
	hasCatch := false
	hasFinally := false

	if _, exists := branches["main"]; exists {
		branchCount++
	}
	if _, exists := branches["catch"]; exists {
		branchCount++
		hasCatch = true
	}
	if _, exists := branches["finally"]; exists {
		branchCount++
		hasFinally = true
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: description,
		Command:     "try/catch/finally error handling",
		Children:    children,
		Metadata: map[string]string{
			"decorator":      "try",
			"execution_mode": "error_handling",
			"color":          plan.ColorCyan,
			"info":           "", // @try doesn't need extra info
			"branchCount":    fmt.Sprintf("%d", branchCount),
			"hasCatch":       fmt.Sprintf("%t", hasCatch),
			"hasFinally":     fmt.Sprintf("%t", hasFinally),
		},
	}
}

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (target interface)
// ================================================================================================

// Plan generates an execution plan for the try operation
func (t *TryDecorator) Plan(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	// @try is a complex pattern decorator that handles conditional execution
	// For now, create a simplified plan representation
	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: "@try { ... } catch { ... } finally { ... }",
		Command:     "# Try-catch-finally pattern execution",
		Children:    []plan.ExecutionStep{}, // Will be populated by plan generator with branches
		Metadata: map[string]string{
			"decorator":      "try",
			"execution_mode": "conditional",
			"pattern_type":   "try_catch_finally",
			"color":          plan.ColorYellow,
		},
	}
}

// Execute performs the try operation
func (t *TryDecorator) Execute(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return &simpleCommandResult{
		stdout:   "",
		stderr:   "try execution not implemented yet - use plan mode",
		exitCode: 1,
	}
}

// RequiresBlock returns the block requirements for @try
func (t *TryDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockPattern,
		Required: true,
	}
}
