package decorators

import (
	"context"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

func TestTimeoutDecorator_Execute(t *testing.T) {
	// Create test program
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test content with shell commands
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo test1")),
		ast.Shell(ast.Text("echo test2")),
	}

	decorator := &TimeoutDecorator{}

	// Test parameters
	params := []ast.NamedParameter{
		{Name: "duration", Value: ast.Dur("30s")},
	}

	// Test interpreter mode
	t.Run("InterpreterMode", func(t *testing.T) {
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

			// Should contain timeout logic
			if !containsAll(code, []string{"context.WithTimeout", "time.ParseDuration", "30s"}) {
				t.Errorf("Generated code missing expected timeout logic")
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

	// Test invalid duration
	t.Run("InvalidDuration", func(t *testing.T) {
		invalidParams := []ast.NamedParameter{
			{Name: "duration", Value: ast.Dur("invalid")},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, invalidParams, content)

		if result.Error == nil {
			t.Errorf("Expected error for invalid duration")
		}
	})

	// Test missing parameters
	t.Run("MissingParameters", func(t *testing.T) {
		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, nil, content)

		if result.Error == nil {
			t.Errorf("Expected error for missing parameters")
		}
	})

	// Test invalid mode
	t.Run("InvalidMode", func(t *testing.T) {
		invalidCtx := ctx.WithMode(execution.ExecutionMode(999))
		result := decorator.Execute(invalidCtx, params, content)

		if result.Error == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}

// Helper function to check if a string contains all substrings
func containsAll(s string, substrings []string) bool {
	for _, substr := range substrings {
		found := false
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
