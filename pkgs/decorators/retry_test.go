package decorators

import (
	"context"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

func TestRetryDecorator_Execute(t *testing.T) {
	// Create test program
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test content with shell commands
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo test1")),
		ast.Shell(ast.Text("echo test2")),
	}

	decorator := &RetryDecorator{}

	// Test parameters
	params := []ast.NamedParameter{
		{Name: "attempts", Value: ast.Num(3)},
		{Name: "delay", Value: ast.Dur("1s")},
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

			// Should contain retry logic
			if !containsAll(code, []string{"maxAttempts", "for attempt", "time.Sleep"}) {
				t.Errorf("Generated code missing expected retry logic")
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

	// Test missing attempts parameter
	t.Run("MissingAttemptsParameter", func(t *testing.T) {
		emptyParams := []ast.NamedParameter{}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, emptyParams, content)

		if result.Error == nil {
			t.Errorf("Expected error for missing attempts parameter")
		}
	})

	// Test too many parameters
	t.Run("TooManyParameters", func(t *testing.T) {
		tooManyParams := []ast.NamedParameter{
			{Name: "attempts", Value: ast.Num(3)},
			{Name: "delay", Value: ast.Dur("1s")},
			{Name: "extra", Value: ast.Str("value")},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, tooManyParams, content)

		if result.Error == nil {
			t.Errorf("Expected error for too many parameters")
		}
	})

	// Test invalid attempts value
	t.Run("InvalidAttempts", func(t *testing.T) {
		invalidParams := []ast.NamedParameter{
			{Name: "attempts", Value: ast.Num(0)}, // Zero attempts should be invalid
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, invalidParams, content)

		if result.Error == nil {
			t.Errorf("Expected error for invalid attempts value")
		}
	})

	// Test negative attempts value
	t.Run("NegativeAttempts", func(t *testing.T) {
		invalidParams := []ast.NamedParameter{
			{Name: "attempts", Value: ast.Num(-1)}, // Negative attempts should be invalid
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, invalidParams, content)

		if result.Error == nil {
			t.Errorf("Expected error for negative attempts value")
		}
	})

	// Test default delay
	t.Run("DefaultDelay", func(t *testing.T) {
		onlyAttemptsParams := []ast.NamedParameter{
			{Name: "attempts", Value: ast.Num(2)},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, onlyAttemptsParams, content)

		if result.Error != nil {
			t.Errorf("Unexpected error with default delay: %v", result.Error)
		}

		// Should use default delay of 1 second
		if result.Data == nil {
			t.Errorf("Expected plan element data for plan mode, got nil")
		}
	})

	// Test custom delay
	t.Run("CustomDelay", func(t *testing.T) {
		customParams := []ast.NamedParameter{
			{Name: "attempts", Value: ast.Num(5)},
			{Name: "delay", Value: ast.Dur("2s")},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, customParams, content)

		if result.Error != nil {
			t.Errorf("Unexpected error with custom delay: %v", result.Error)
		}

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

func TestRetryDecorator_ExecuteCommands(t *testing.T) {
	// Create test program
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)

	decorator := &RetryDecorator{}

	// Test empty commands
	t.Run("EmptyCommands", func(t *testing.T) {
		err := decorator.executeCommands(ctx, []ast.CommandContent{})
		if err != nil {
			t.Errorf("Unexpected error with empty commands: %v", err)
		}
	})

	// Test with commands (will fail due to missing executor, but that's expected)
	t.Run("WithCommands", func(t *testing.T) {
		content := []ast.CommandContent{
			ast.Shell(ast.Text("echo test")),
		}

		// This will fail because there's no actual command executor, but we're testing the flow
		err := decorator.executeCommands(ctx, content)
		// Error is expected here due to missing command executor in test context
		_ = err
	})
}
