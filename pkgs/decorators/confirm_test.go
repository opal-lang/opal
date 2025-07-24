package decorators

import (
	"context"
	"os"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

func TestConfirmDecorator_Execute(t *testing.T) {
	// Create test program
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test content with shell commands
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo test1")),
		ast.Shell(ast.Text("echo test2")),
	}

	decorator := &ConfirmDecorator{}

	// Test default parameters
	params := []ast.NamedParameter{}

	// Test interpreter mode - we can't actually test interactive input in unit tests
	// so we'll focus on CI mode behavior
	t.Run("InterpreterMode_CI", func(t *testing.T) {
		// Set CI environment variable
		_ = os.Setenv("CI", "true")
		defer func() { _ = os.Unsetenv("CI") }()

		interpreterCtx := ctx.WithMode(execution.InterpreterMode)
		result := decorator.Execute(interpreterCtx, params, content)

		if result.Mode != execution.InterpreterMode {
			t.Errorf("Expected InterpreterMode, got %v", result.Mode)
		}

		// Data should be nil for interpreter mode
		if result.Data != nil {
			t.Errorf("Expected nil data for interpreter mode, got %v", result.Data)
		}

		// Error might be non-nil due to missing command executor, that's okay for this test
	})

	// Test generator mode
	t.Run("GeneratorMode", func(t *testing.T) {
		generatorCtx := ctx.WithMode(execution.GeneratorMode)
		result := decorator.Execute(generatorCtx, params, content)

		if result.Mode != execution.GeneratorMode {
			t.Errorf("Expected GeneratorMode, got %v", result.Mode)
		}

		// Data should be a string containing Go code
		if result.Error == nil {
			code, ok := result.Data.(string)
			if !ok {
				t.Errorf("Expected string data for generator mode, got %T", result.Data)
			}
			if code == "" {
				t.Errorf("Expected non-empty generated code")
			}

			// Should contain confirmation logic
			if !containsAll(code, []string{"func() error", "return nil"}) {
				t.Errorf("Generated code missing expected confirmation logic")
			}
		}
	})

	// Test plan mode
	t.Run("PlanMode", func(t *testing.T) {
		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, content)

		if result.Mode != execution.PlanMode {
			t.Errorf("Expected PlanMode, got %v", result.Mode)
		}

		if result.Error != nil {
			t.Errorf("Unexpected error in plan mode: %v", result.Error)
		}

		// Data should be a plan element
		if result.Data == nil {
			t.Errorf("Expected plan element data for plan mode, got nil")
		}
	})

	// Test plan mode in CI environment
	t.Run("PlanMode_CI", func(t *testing.T) {
		// Set CI environment variable
		_ = os.Setenv("CI", "true")
		defer func() { _ = os.Unsetenv("CI") }()

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, content)

		if result.Mode != execution.PlanMode {
			t.Errorf("Expected PlanMode, got %v", result.Mode)
		}

		if result.Error != nil {
			t.Errorf("Unexpected error in plan mode: %v", result.Error)
		}

		// Data should be a plan element
		if result.Data == nil {
			t.Errorf("Expected plan element data for plan mode, got nil")
		}
	})

	// Test custom parameters
	t.Run("CustomParameters", func(t *testing.T) {
		customParams := []ast.NamedParameter{
			{Name: "message", Value: ast.Str("Are you sure?")},
			{Name: "defaultYes", Value: ast.Bool(true)},
			{Name: "abortOnNo", Value: ast.Bool(false)},
			{Name: "caseSensitive", Value: ast.Bool(true)},
			{Name: "ci", Value: ast.Bool(false)},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, customParams, content)

		if result.Error != nil {
			t.Errorf("Unexpected error with custom parameters: %v", result.Error)
		}

		// Data should be a plan element
		if result.Data == nil {
			t.Errorf("Expected plan element data for plan mode, got nil")
		}
	})

	// Validation is now handled at parse-time via ParameterSchema

	// Test invalid mode
	t.Run("InvalidMode", func(t *testing.T) {
		invalidCtx := ctx.WithMode(execution.ExecutionMode(999))
		result := decorator.Execute(invalidCtx, params, content)

		if result.Error == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}

func TestIsCI(t *testing.T) {
	// Test CI detection
	t.Run("NoCIEnv", func(t *testing.T) {
		// Clear all CI environment variables
		ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "TRAVIS", "CIRCLECI", "JENKINS_URL", "GITLAB_CI", "BUILDKITE", "BUILD_NUMBER"}
		for _, envVar := range ciVars {
			_ = os.Unsetenv(envVar)
		}

		if isCI() {
			t.Errorf("Expected isCI() to return false when no CI env vars are set")
		}
	})

	t.Run("WithCIEnv", func(t *testing.T) {
		_ = os.Setenv("CI", "true")
		defer func() { _ = os.Unsetenv("CI") }()

		if !isCI() {
			t.Errorf("Expected isCI() to return true when CI env var is set")
		}
	})

	t.Run("WithGitHubActions", func(t *testing.T) {
		_ = os.Setenv("GITHUB_ACTIONS", "true")
		defer func() { _ = os.Unsetenv("GITHUB_ACTIONS") }()

		if !isCI() {
			t.Errorf("Expected isCI() to return true when GITHUB_ACTIONS env var is set")
		}
	})
}
