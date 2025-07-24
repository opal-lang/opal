package decorators

import (
	"context"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

func TestTryDecorator_Execute(t *testing.T) {
	// Create test program
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test patterns with main, error, and finally branches
	patterns := []ast.PatternBranch{
		{
			Pattern: &ast.IdentifierPattern{Name: "main"},
			Commands: []ast.CommandContent{
				ast.Shell(ast.Text("echo main command")),
			},
		},
		{
			Pattern: &ast.IdentifierPattern{Name: "error"},
			Commands: []ast.CommandContent{
				ast.Shell(ast.Text("echo error handler")),
			},
		},
		{
			Pattern: &ast.IdentifierPattern{Name: "finally"},
			Commands: []ast.CommandContent{
				ast.Shell(ast.Text("echo finally block")),
			},
		},
	}

	decorator := &TryDecorator{}

	// Test no parameters (should be valid)
	params := []ast.NamedParameter{}

	// Test interpreter mode
	t.Run("InterpreterMode", func(t *testing.T) {
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

			// Should contain try-catch logic
			if !containsAll(code, []string{"mainErr", "errorErr", "finallyErr"}) {
				t.Errorf("Generated code missing expected try-catch logic")
			}
		}
	})

	// Test plan mode
	t.Run("PlanMode", func(t *testing.T) {
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

	// Test missing main pattern
	t.Run("MissingMainPattern", func(t *testing.T) {
		invalidPatterns := []ast.PatternBranch{
			{
				Pattern: &ast.IdentifierPattern{Name: "error"},
				Commands: []ast.CommandContent{
					ast.Shell(ast.Text("echo error handler")),
				},
			},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, invalidPatterns)

		if result.Error == nil {
			t.Errorf("Expected error for missing main pattern")
		}
	})

	// Test missing error and finally patterns
	t.Run("MissingErrorAndFinallyPatterns", func(t *testing.T) {
		invalidPatterns := []ast.PatternBranch{
			{
				Pattern: &ast.IdentifierPattern{Name: "main"},
				Commands: []ast.CommandContent{
					ast.Shell(ast.Text("echo main command")),
				},
			},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, invalidPatterns)

		if result.Error == nil {
			t.Errorf("Expected error for missing both error and finally patterns")
		}
	})

	// Test invalid pattern
	t.Run("InvalidPattern", func(t *testing.T) {
		invalidPatterns := []ast.PatternBranch{
			{
				Pattern: &ast.IdentifierPattern{Name: "main"},
				Commands: []ast.CommandContent{
					ast.Shell(ast.Text("echo main command")),
				},
			},
			{
				Pattern: &ast.IdentifierPattern{Name: "invalid"},
				Commands: []ast.CommandContent{
					ast.Shell(ast.Text("echo invalid")),
				},
			},
			{
				Pattern: &ast.IdentifierPattern{Name: "error"},
				Commands: []ast.CommandContent{
					ast.Shell(ast.Text("echo error handler")),
				},
			},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, invalidPatterns)

		if result.Error == nil {
			t.Errorf("Expected error for invalid pattern")
		}
	})

	// Test with parameters (should fail)
	t.Run("WithParameters", func(t *testing.T) {
		invalidParams := []ast.NamedParameter{
			{Name: "invalid", Value: ast.Str("test")},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, invalidParams, patterns)

		if result.Error == nil {
			t.Errorf("Expected error for parameters (try takes no parameters)")
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
