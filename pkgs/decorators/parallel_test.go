package decorators

import (
	"context"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

func TestParallelDecorator_Execute(t *testing.T) {
	// Create test program with variables
	program := ast.NewProgram()

	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test content with shell commands
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo test1")),
		ast.Shell(ast.Text("echo test2")),
	}

	decorator := &ParallelDecorator{}

	// Test interpreter mode
	t.Run("InterpreterMode", func(t *testing.T) {
		interpreterCtx := ctx.WithMode(execution.InterpreterMode)
		result := decorator.Execute(interpreterCtx, nil, content)

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
		result := decorator.Execute(generatorCtx, nil, content)

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
		}
	})

	// Test plan mode
	t.Run("PlanMode", func(t *testing.T) {
		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, nil, content)

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

	// Test with parameters
	t.Run("WithParameters", func(t *testing.T) {
		params := []ast.NamedParameter{
			{Name: "concurrency", Value: ast.Num(2)},
			{Name: "failOnFirstError", Value: ast.Bool(true)},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, content)

		if result.Error != nil {
			t.Errorf("Unexpected error with parameters: %v", result.Error)
		}
	})

	// Test invalid mode
	t.Run("InvalidMode", func(t *testing.T) {
		invalidCtx := ctx.WithMode(execution.ExecutionMode(999))
		result := decorator.Execute(invalidCtx, nil, content)

		if result.Error == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}
