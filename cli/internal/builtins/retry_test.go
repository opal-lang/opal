package decorators

import (
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestRetryDecorator_Basic(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test basic retry functionality
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'attempt'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "3"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorExecutesCorrectly().
		GeneratorCodeContains("for attempt := 1; attempt <= 3", "1 * time.Second", "func() error").
		PlanSucceeds().
		PlanReturnsElement("decorator").
		CompletesWithin("1s").
		SupportsDevcmdChaining().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_WithDelay(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test retry with custom delay
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'retry with delay'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
			{Name: "delay", Value: &ast.DurationLiteral{Value: "500ms"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("for attempt := 1; attempt <= 2", "500 * time.Millisecond").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator with delay test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_DefaultDelay(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test default delay (1s)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'default delay test'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("1 * time.Second"). // Should use default 1s delay
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator default delay test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_SingleAttempt(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test single attempt (edge case)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'single attempt'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "1"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("for attempt := 1; attempt <= 1").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator single attempt test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_FailingCommand(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test retry behavior with failing command
	content := []ast.CommandContent{
		decoratortesting.Shell("false"), // This will always fail
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "3"}},
			{Name: "delay", Value: &ast.DurationLiteral{Value: "10ms"}}, // Short delay for testing
		}, content)

	// Note: Interpreter will fail after all retries, but generator and plan should work
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("command failed after %d attempts").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator failing command test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_MultipleCommands(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test retry with multiple commands
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'command 1'"),
		decoratortesting.Shell("echo 'command 2'"),
		decoratortesting.Shell("echo 'command 3'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator multiple commands test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_NestedDecorators(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test with nested decorators
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "timeout",
			Args: []ast.NamedParameter{
				{Name: "duration", Value: &ast.DurationLiteral{Value: "5s"}},
			},
			Content: []ast.CommandContent{
				decoratortesting.Shell("echo 'nested in timeout'"),
			},
		},
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator nested decorators test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_InvalidParameters(t *testing.T) {
	decorator := &RetryDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test missing attempts parameter
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterFails("requires at least 1 parameter").
		GeneratorFails("requires at least 1 parameter").
		PlanFails("requires at least 1 parameter").
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator missing attempts test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_ZeroAttempts(t *testing.T) {
	decorator := &RetryDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test zero attempts (invalid)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "0"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterFails("must be positive").
		GeneratorFails("must be positive").
		PlanFails("must be positive").
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator zero attempts test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_NegativeAttempts(t *testing.T) {
	decorator := &RetryDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test negative attempts (invalid)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "-1"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterFails("must be positive").
		GeneratorFails("must be positive").
		PlanFails("must be positive").
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator negative attempts test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_InvalidDelay(t *testing.T) {
	decorator := &RetryDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test invalid delay format - the decorator might use defaults, so check for that behavior
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
			{Name: "delay", Value: &ast.StringLiteral{Value: "invalid"}},
		}, content)

	// Invalid delay should cause validation failure
	errors := decoratortesting.Assert(result).
		InterpreterFails("must be of type duration").
		GeneratorFails("must be of type duration").
		PlanFails("must be of type duration").
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator invalid delay test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_EmptyContent(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test with no commands
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
		}, []ast.CommandContent{})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator empty content test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_LongDelay(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test with longer delay to ensure proper duration handling
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'long delay test'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "2"}},
			{Name: "delay", Value: &ast.DurationLiteral{Value: "2m30s"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("2m30s").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator long delay test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_HighAttemptCount(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test with high attempt count
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'high attempts'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "10"}},
			{Name: "delay", Value: &ast.DurationLiteral{Value: "1ms"}}, // Very short delay for testing
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("for attempt := 1; attempt <= 10").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator high attempt count test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_PerformanceCharacteristics(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test that the decorator itself doesn't add significant overhead
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'performance test'"),
	}

	start := time.Now()
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "3"}},
			{Name: "delay", Value: &ast.DurationLiteral{Value: "1ms"}},
		}, content)
	generatorDuration := time.Since(start)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		CompletesWithin("100ms"). // Should be very fast for generation
		Validate()

	// Additional check that generation is fast
	if generatorDuration > 100*time.Millisecond {
		errors = append(errors, "Retry decorator generation is too slow")
	}

	if len(errors) > 0 {
		t.Errorf("RetryDecorator performance test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestRetryDecorator_ErrorRecoveryScenario(t *testing.T) {
	decorator := &RetryDecorator{}

	// Test typical error recovery scenario with mixed commands
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'setup'"),
		decoratortesting.Shell("echo 'main operation'"),
		decoratortesting.Shell("echo 'cleanup'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "attempts", Value: &ast.NumberLiteral{Value: "3"}},
			{Name: "delay", Value: &ast.DurationLiteral{Value: "100ms"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("for attempt := 1; attempt <= 3").
		GeneratorCodeContains("100 * time.Millisecond", "time.Sleep").
		PlanSucceeds().
		SupportsDevcmdChaining().
		Validate()

	if len(errors) > 0 {
		t.Errorf("RetryDecorator error recovery scenario test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
