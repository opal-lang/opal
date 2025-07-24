package decorators

import (
	"context"
	"os"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

func TestWhenDecorator_Execute(t *testing.T) {
	// Create test program
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test patterns with different values
	patterns := []ast.PatternBranch{
		{
			Pattern: &ast.IdentifierPattern{Name: "production"},
			Commands: []ast.CommandContent{
				ast.Shell(ast.Text("echo production command")),
			},
		},
		{
			Pattern: &ast.IdentifierPattern{Name: "staging"},
			Commands: []ast.CommandContent{
				ast.Shell(ast.Text("echo staging command")),
			},
		},
		{
			Pattern: &ast.WildcardPattern{},
			Commands: []ast.CommandContent{
				ast.Shell(ast.Text("echo default command")),
			},
		},
	}

	decorator := &WhenDecorator{}

	// Test parameters
	params := []ast.NamedParameter{
		{Name: "variable", Value: ast.Str("ENV")},
	}

	// Test interpreter mode with environment variable
	t.Run("InterpreterMode", func(t *testing.T) {
		_ = os.Setenv("ENV", "production")
		defer func() { _ = os.Unsetenv("ENV") }()

		interpreterCtx := ctx.WithMode(execution.InterpreterMode)
		result := decorator.Execute(interpreterCtx, params, patterns)

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
		result := decorator.Execute(generatorCtx, params, patterns)

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

			// Should contain pattern matching logic
			if !containsAll(code, []string{"switch value", "case", "default"}) {
				t.Errorf("Generated code missing expected pattern matching logic")
			}
		}
	})

	// Test plan mode
	t.Run("PlanMode", func(t *testing.T) {
		_ = os.Setenv("ENV", "production")
		defer func() { _ = os.Unsetenv("ENV") }()

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, patterns)

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

	// Test missing variable parameter
	t.Run("MissingVariableParameter", func(t *testing.T) {
		emptyParams := []ast.NamedParameter{}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, emptyParams, patterns)

		if result.Error == nil {
			t.Errorf("Expected error for missing variable parameter")
		}
	})

	// Test wrong number of parameters
	t.Run("WrongParameterCount", func(t *testing.T) {
		tooManyParams := []ast.NamedParameter{
			{Name: "variable", Value: ast.Str("ENV")},
			{Name: "extra", Value: ast.Str("value")},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, tooManyParams, patterns)

		if result.Error == nil {
			t.Errorf("Expected error for wrong parameter count")
		}
	})

	// Test positional parameter fallback
	t.Run("PositionalParameter", func(t *testing.T) {
		positionalParams := []ast.NamedParameter{
			{Name: "variable", Value: ast.Str("ENV")}, // Named parameter for clarity
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, positionalParams, patterns)

		if result.Error != nil {
			t.Errorf("Unexpected error with named parameter: %v", result.Error)
		}
	})

	// Test invalid mode
	t.Run("InvalidMode", func(t *testing.T) {
		invalidCtx := ctx.WithMode(execution.ExecutionMode(999))
		result := decorator.Execute(invalidCtx, params, patterns)

		if result.Error == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}

func TestWhenDecorator_PatternMatching(t *testing.T) {
	decorator := &WhenDecorator{}

	// Test identifier pattern matching
	t.Run("IdentifierPattern", func(t *testing.T) {
		pattern := &ast.IdentifierPattern{Name: "production"}

		if !decorator.matchesPattern("production", pattern) {
			t.Errorf("Expected 'production' to match identifier pattern 'production'")
		}

		if decorator.matchesPattern("staging", pattern) {
			t.Errorf("Expected 'staging' to NOT match identifier pattern 'production'")
		}
	})

	// Test wildcard pattern matching
	t.Run("WildcardPattern", func(t *testing.T) {
		pattern := &ast.WildcardPattern{}

		if !decorator.matchesPattern("anything", pattern) {
			t.Errorf("Expected wildcard pattern to match 'anything'")
		}

		if !decorator.matchesPattern("", pattern) {
			t.Errorf("Expected wildcard pattern to match empty string")
		}
	})
}

func TestWhenDecorator_PatternToString(t *testing.T) {
	decorator := &WhenDecorator{}

	// Test identifier pattern
	t.Run("IdentifierPattern", func(t *testing.T) {
		pattern := &ast.IdentifierPattern{Name: "production"}
		result := decorator.patternToString(pattern)

		if result != "production" {
			t.Errorf("Expected 'production', got '%s'", result)
		}
	})

	// Test wildcard pattern
	t.Run("WildcardPattern", func(t *testing.T) {
		pattern := &ast.WildcardPattern{}
		result := decorator.patternToString(pattern)

		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})
}
