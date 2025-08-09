package decorators

import (
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestTimeoutDecorator_Basic(t *testing.T) {
	decorator := &TimeoutDecorator{}

	// Test basic timeout functionality
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'quick command'"),
		decoratortesting.Shell("sleep 0.1"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "duration", Value: &ast.DurationLiteral{Value: "5s"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorExecutesCorrectly().
		GeneratorCodeContains("context.WithTimeout(ctx, 5 * time.Second)").
		GeneratorCodeContainsWithContext("context cancellation", "context.WithTimeout").
		GeneratorCodeContainsWithContext("timeout handling", "select {", "case <-ctx.Done():").
		PlanSucceeds().
		PlanReturnsElement("decorator").
		CompletesWithin("1s").
		SupportsDevcmdChaining().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTimeoutDecorator_DefaultTimeout(t *testing.T) {
	decorator := &TimeoutDecorator{}

	// Test default timeout (30s)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("30s"). // Should use default
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator default timeout test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTimeoutDecorator_ShortTimeout(t *testing.T) {
	decorator := &TimeoutDecorator{}

	// Test timeout with command that might exceed it
	content := []ast.CommandContent{
		decoratortesting.Shell("sleep 10"), // This should timeout
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "duration", Value: &ast.DurationLiteral{Value: "100ms"}},
		}, content)

	// Note: Interpreter might fail due to timeout, but generator and plan should work
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("100 * time.Millisecond").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator short timeout test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTimeoutDecorator_InvalidDuration(t *testing.T) {
	decorator := &TimeoutDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test invalid duration format
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "duration", Value: &ast.StringLiteral{Value: "invalid"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterFails("duration").
		GeneratorFails("duration").
		PlanFails("duration").
		Validate()

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator invalid duration test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTimeoutDecorator_EmptyContent(t *testing.T) {
	decorator := &TimeoutDecorator{}

	// Test with no commands
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "duration", Value: &ast.DurationLiteral{Value: "1s"}},
		}, []ast.CommandContent{})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator empty content test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTimeoutDecorator_NestedDecorators(t *testing.T) {
	decorator := &TimeoutDecorator{}

	// Test with nested decorators
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "retry",
			Args: []ast.NamedParameter{
				{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
			},
			Content: []ast.CommandContent{
				decoratortesting.Shell("echo 'nested test'"),
			},
		},
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "duration", Value: &ast.DurationLiteral{Value: "2s"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator nested decorators test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTimeoutDecorator_PerformanceCharacteristics(t *testing.T) {
	decorator := &TimeoutDecorator{}

	// Test that the decorator itself doesn't add significant overhead
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'performance test'"),
	}

	start := time.Now()
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "duration", Value: &ast.DurationLiteral{Value: "10s"}},
		}, content)
	generatorDuration := time.Since(start)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		CompletesWithin("100ms"). // Should be very fast for generation
		Validate()

	// Additional check that generation is fast
	if generatorDuration > 100*time.Millisecond {
		errors = append(errors, "Timeout decorator generation is too slow")
	}

	if len(errors) > 0 {
		t.Errorf("TimeoutDecorator performance test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
