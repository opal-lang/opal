package builtin

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @try decorator on package import
func init() {
	decorators.RegisterPattern(NewTryDecorator())
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

// PatternSchema defines what patterns @try accepts
func (t *TryDecorator) PatternSchema() decorators.PatternSchema {
	return decorators.PatternSchema{
		AllowedPatterns:     []string{"main", "catch", "finally"},
		RequiredPatterns:    []string{"main"}, // main block is required
		AllowsWildcard:      false,            // No wildcard patterns
		AllowsAnyIdentifier: false,            // Only specific patterns allowed
		Description:         "Error handling with main/catch/finally blocks",
	}
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

// ImportRequirements returns the dependencies needed for code generation
func (t *TryDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// PATTERN DECORATOR METHODS
// ================================================================================================

// SelectBranch executes try/catch/finally logic
func (t *TryDecorator) SelectBranch(ctx *decorators.Ctx, args []decorators.DecoratorParam, branches map[string]decorators.CommandSeq) decorators.CommandResult {
	// Validate that we have required branches
	mainBranch, hasMain := branches["main"]
	if !hasMain {
		return decorators.CommandResult{
			Stderr:   "@try requires a 'main' branch",
			ExitCode: 1,
		}
	}

	catchBranch, hasCatch := branches["catch"]
	finallyBranch, hasFinally := branches["finally"]

	var mainResult decorators.CommandResult
	var finalResult decorators.CommandResult

	// Execute main branch
	mainResult = ctx.ExecSequential(mainBranch.Steps)

	// If main failed and we have a catch branch, execute it
	if mainResult.Failed() && hasCatch {
		if ctx.Debug {
			_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] @try main failed (exit code %d), executing catch block\n", mainResult.ExitCode)
		}

		catchResult := ctx.ExecSequential(catchBranch.Steps)

		// Catch result determines the overall success/failure
		// This allows catch blocks to either:
		// 1. Return success (exit code 0) to "handle" the error
		// 2. Return failure to propagate the error
		finalResult = catchResult

		// Combine output from both main and catch
		finalResult.Stdout = mainResult.Stdout + catchResult.Stdout
		finalResult.Stderr = mainResult.Stderr + catchResult.Stderr
	} else {
		// Main succeeded or no catch block - use main result
		finalResult = mainResult
	}

	// Always execute finally block if present
	if hasFinally {
		if ctx.Debug {
			_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] @try executing finally block\n")
		}

		finallyResult := ctx.ExecSequential(finallyBranch.Steps)

		// Finally block output is always included
		finalResult.Stdout += finallyResult.Stdout
		finalResult.Stderr += finallyResult.Stderr

		// Finally block failure overrides the overall result
		// This ensures cleanup failures are not ignored
		if finallyResult.Failed() {
			finalResult.ExitCode = finallyResult.ExitCode
		}
	}

	return finalResult
}

// Describe returns description for dry-run display
func (t *TryDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep {
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
			"execution_mode": string(plan.ModeErrorHandling),
			"color":          plan.ColorCyan,
			"info":           "", // @try doesn't need extra info
			"branchCount":    fmt.Sprintf("%d", branchCount),
			"hasCatch":       fmt.Sprintf("%t", hasCatch),
			"hasFinally":     fmt.Sprintf("%t", hasFinally),
		},
	}
}
