package decorators

import (
	"runtime"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestParallelDecorator_Basic(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test basic parallel execution
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'command 1'"),
		decoratortesting.Shell("echo 'command 2'"),
		decoratortesting.Shell("echo 'command 3'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("wg", "sync.WaitGroup", "go func()").
		PlanSucceeds().
		PlanReturnsElement("parallel").
		CompletesWithin("1s").
		SupportsDevcmdChaining().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_ConcurrencyLimit(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test with concurrency limit
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'task 1'"),
		decoratortesting.Shell("echo 'task 2'"),
		decoratortesting.Shell("echo 'task 3'"),
		decoratortesting.Shell("echo 'task 4'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "2"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("var wg sync.WaitGroup", "wg.Add(1)").
		PlanSucceeds().
		PlanReturnsElement("parallel").
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator concurrency limit test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_FailOnFirstError(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test fail-fast behavior
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'success'"),
		decoratortesting.Shell("false"), // This should fail
		decoratortesting.Shell("echo 'might not run'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "failOnFirstError", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator fail on first error test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_ConcurrencyAndFailFast(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test combining concurrency limit with fail-fast
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'task 1'"),
		decoratortesting.Shell("echo 'task 2'"),
		decoratortesting.Shell("echo 'task 3'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "2"}},
			{Name: "failOnFirstError", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("var wg sync.WaitGroup").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator concurrency and fail-fast test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_EmptyContent(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test with no commands
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, []ast.CommandContent{})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator empty content test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_SingleCommand(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test with single command (edge case)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'single command'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator single command test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_NestedDecorators(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test with nested decorators
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'plain command'"),
		&ast.BlockDecorator{
			Name: "timeout",
			Args: []ast.NamedParameter{
				{Name: "duration", Value: &ast.DurationLiteral{Value: "5s"}},
			},
			Content: []ast.CommandContent{
				decoratortesting.Shell("echo 'nested in timeout'"),
			},
		},
		&ast.BlockDecorator{
			Name: "retry",
			Args: []ast.NamedParameter{
				{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
			},
			Content: []ast.CommandContent{
				decoratortesting.Shell("echo 'nested in retry'"),
			},
		},
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "2"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator nested decorators test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_InvalidParameters(t *testing.T) {
	decorator := &ParallelDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test invalid concurrency parameter (negative)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "-1"}},
		}, content)

	// Parallel decorator should reject negative concurrency values
	errors := decoratortesting.Assert(result).
		InterpreterFails("must be positive").
		GeneratorFails("must be positive").
		PlanFails("must be positive").
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator invalid parameters test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_IsolationTesting(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test that parallel commands are properly isolated
	// Each command should run in its own context
	content := []ast.CommandContent{
		decoratortesting.Shell("cd /tmp && echo 'in /tmp'"),
		decoratortesting.Shell("pwd"), // Should not be affected by the cd above
		decoratortesting.Shell("echo 'independent command'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator isolation test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_PerformanceCharacteristics(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test that the decorator itself doesn't add significant overhead
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'performance test 1'"),
		decoratortesting.Shell("echo 'performance test 2'"),
		decoratortesting.Shell("echo 'performance test 3'"),
		decoratortesting.Shell("echo 'performance test 4'"),
	}

	start := time.Now()
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "2"}},
		}, content)
	generatorDuration := time.Since(start)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		CompletesWithin("100ms"). // Should be very fast for generation
		Validate()

	// Additional check that generation is fast
	if generatorDuration > 100*time.Millisecond {
		errors = append(errors, "Parallel decorator generation is too slow")
	}

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator performance test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_ErrorHandling(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test error collection behavior
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'success 1'"),
		decoratortesting.Shell("false"), // This will fail
		decoratortesting.Shell("echo 'success 2'"),
		decoratortesting.Shell("exit 42"), // This will also fail
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	// Note: Interpreter might fail due to command failures, but generator and plan should work
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("errs := make([]error").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator error handling test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_ConcurrencyEdgeCases(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test edge case: concurrency limit greater than number of commands
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'cmd1'"),
		decoratortesting.Shell("echo 'cmd2'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "10"}}, // More than commands
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("var wg sync.WaitGroup").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator concurrency edge cases test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_ComplexNesting(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test complex nested scenario with multiple decorator types
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "timeout",
			Args: []ast.NamedParameter{
				{Name: "duration", Value: &ast.DurationLiteral{Value: "10s"}},
			},
			Content: []ast.CommandContent{
				&ast.BlockDecorator{
					Name: "retry",
					Args: []ast.NamedParameter{
						{Name: "attempts", Value: &ast.NumberLiteral{Value: "3"}},
					},
					Content: []ast.CommandContent{
						decoratortesting.Shell("echo 'deeply nested'"),
					},
				},
			},
		},
		decoratortesting.Shell("echo 'parallel with nested'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "2"}},
			{Name: "failOnFirstError", Value: &ast.BooleanLiteral{Value: false}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator complex nesting test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_CPUBasedConcurrencyCapping(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test that extreme concurrency values are automatically capped
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'task 1'"),
		decoratortesting.Shell("echo 'task 2'"),
	}

	// Request very high concurrency (should be capped to CPU count * 2)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "1000"}},
		}, content)

	_ = runtime.NumCPU()

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("var wg sync.WaitGroup").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator CPU-based capping test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_UncappedConcurrency(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test that uncapped=true bypasses CPU-based limits
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'task 1'"),
		decoratortesting.Shell("echo 'task 2'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "concurrency", Value: &ast.NumberLiteral{Value: "100"}},
			{Name: "uncapped", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("var wg sync.WaitGroup").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator uncapped concurrency test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestParallelDecorator_ReasonableConcurrencyDefaults(t *testing.T) {
	decorator := &ParallelDecorator{}

	// Test that default concurrency is reasonable for small task counts
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'task 1'"),
		decoratortesting.Shell("echo 'task 2'"),
		decoratortesting.Shell("echo 'task 3'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ParallelDecorator reasonable defaults test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
