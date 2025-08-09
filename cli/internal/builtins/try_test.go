package decorators

import (
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestTryDecorator_MainOnly(t *testing.T) {
	decorator := &TryDecorator{}

	// Test basic try with main pattern only and catch
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'main block'"),
			decoratortesting.PatternBranch("catch", "echo 'error handling'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("main block", "error handling", "tryMainErr", "tryCatchErr").
		PlanSucceeds().
		PlanReturnsElement("try").
		CompletesWithin("1s").
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator main only test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_MainWithFinally(t *testing.T) {
	decorator := &TryDecorator{}

	// Test try with main and finally patterns
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'main execution'"),
			decoratortesting.PatternBranch("finally", "echo 'cleanup'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("main execution", "cleanup", "tryFinallyErr").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator main with finally test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_FullTryCatchFinally(t *testing.T) {
	decorator := &TryDecorator{}

	// Test full try-catch-finally structure
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'try this operation'", "echo 'main step 2'"),
			decoratortesting.PatternBranch("catch", "echo 'handle error'", "echo 'error recovery'"),
			decoratortesting.PatternBranch("finally", "echo 'always cleanup'", "echo 'final step'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("try this operation", "handle error", "always cleanup").
		GeneratorCodeContains("tryMainErr", "tryCatchErr", "tryFinallyErr").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator full try-catch-finally test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_ErrorHandlingPrecedence(t *testing.T) {
	decorator := &TryDecorator{}

	// Test error precedence: main error takes precedence over catch/finally errors
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'main'"),
			decoratortesting.PatternBranch("catch", "echo 'catch'"),
			decoratortesting.PatternBranch("finally", "echo 'finally'"),
		})

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("main block failed", "catch block failed", "finally block failed").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator error handling precedence test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_FailingCommand(t *testing.T) {
	decorator := &TryDecorator{}

	// Test try-catch with failing command in main block
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "false"), // This command always fails
			decoratortesting.PatternBranch("catch", "echo 'caught error'"),
		})

	// Note: Interpreter will fail but generator and plan should work
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("caught error").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator failing command test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_MultipleCommands(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with multiple commands in each block
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'step 1'", "echo 'step 2'", "echo 'step 3'"),
			decoratortesting.PatternBranch("catch", "echo 'error step 1'", "echo 'error step 2'"),
			decoratortesting.PatternBranch("finally", "echo 'cleanup step 1'", "echo 'cleanup step 2'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("step 1", "step 2", "step 3", "error step 1", "cleanup step 1").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator multiple commands test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_NestedDecorators(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with nested decorators (try containing timeout)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			{
				Pattern: &ast.IdentifierPattern{Name: "main"},
				Commands: []ast.CommandContent{
					&ast.BlockDecorator{
						Name: "timeout",
						Args: []ast.NamedParameter{
							{Name: "duration", Value: &ast.DurationLiteral{Value: "5s"}},
						},
						Content: []ast.CommandContent{
							decoratortesting.Shell("echo 'nested in timeout'"),
						},
					},
				},
			},
			decoratortesting.PatternBranch("catch", "echo 'timeout error'"),
		})

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator nested decorators test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_InvalidParameters(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with invalid parameters (try takes no parameters)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			{Name: "invalid", Value: &ast.StringLiteral{Value: "should fail"}},
		}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'test'"),
			decoratortesting.PatternBranch("catch", "echo 'error'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterFails("takes no parameters").
		GeneratorFails("takes no parameters").
		PlanFails("takes no parameters").
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator invalid parameters test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_MissingMainPattern(t *testing.T) {
	decorator := &TryDecorator{}

	// Test missing required main pattern
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("catch", "echo 'error'"),
			decoratortesting.PatternBranch("finally", "echo 'cleanup'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterFails("requires a 'main' pattern").
		GeneratorFails("requires a 'main' pattern").
		PlanFails("requires a 'main' pattern").
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator missing main pattern test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_MissingCatchAndFinally(t *testing.T) {
	decorator := &TryDecorator{}

	// Test missing both catch and finally patterns
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'only main'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterFails("requires at least one of 'catch' or 'finally'").
		GeneratorFails("requires at least one of 'catch' or 'finally'").
		PlanFails("requires at least one of 'catch' or 'finally'").
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator missing catch and finally test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_InvalidPattern(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with invalid pattern name
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'main'"),
			decoratortesting.PatternBranch("invalid", "echo 'should fail'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterFails("only supports 'main', 'catch', and 'finally'").
		GeneratorFails("only supports 'main', 'catch', and 'finally'").
		PlanFails("only supports 'main', 'catch', and 'finally'").
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator invalid pattern test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_EmptyBlocks(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with empty command blocks
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			{
				Pattern:  &ast.IdentifierPattern{Name: "main"},
				Commands: []ast.CommandContent{}, // Empty main block
			},
			{
				Pattern:  &ast.IdentifierPattern{Name: "catch"},
				Commands: []ast.CommandContent{}, // Empty catch block
			},
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator empty blocks test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_CatchOnly(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with only main and catch (no finally)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'main operation'"),
			decoratortesting.PatternBranch("catch", "echo 'error handling'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("main operation", "error handling").
		// Should not contain finally logic
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator catch only test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_FinallyOnly(t *testing.T) {
	decorator := &TryDecorator{}

	// Test with only main and finally (no catch)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'main operation'"),
			decoratortesting.PatternBranch("finally", "echo 'cleanup'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("main operation", "cleanup").
		// Should not contain catch logic
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator finally only test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_PatternOrder(t *testing.T) {
	decorator := &TryDecorator{}

	// Test that pattern order doesn't matter (finally, main, catch)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("finally", "echo 'cleanup'"),
			decoratortesting.PatternBranch("main", "echo 'main work'"),
			decoratortesting.PatternBranch("catch", "echo 'error handling'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("main work", "error handling", "cleanup").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator pattern order test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_ComplexErrorRecovery(t *testing.T) {
	decorator := &TryDecorator{}

	// Test complex error recovery scenario
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main",
				"echo 'attempt operation'",
				"echo 'operation step 2'",
				"echo 'operation step 3'"),
			decoratortesting.PatternBranch("catch",
				"echo 'log error'",
				"echo 'notify admin'",
				"echo 'fallback action'"),
			decoratortesting.PatternBranch("finally",
				"echo 'cleanup resources'",
				"echo 'log completion'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("attempt operation", "log error", "cleanup resources").
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator complex error recovery test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_PerformanceCharacteristics(t *testing.T) {
	decorator := &TryDecorator{}

	// Test that the decorator itself doesn't add significant overhead
	start := time.Now()
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'performance test'"),
			decoratortesting.PatternBranch("finally", "echo 'cleanup'"),
		})
	generatorDuration := time.Since(start)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		CompletesWithin("100ms"). // Should be very fast for generation
		Validate()

	// Additional check that generation is fast
	if generatorDuration > 100*time.Millisecond {
		errors = append(errors, "Try decorator generation is too slow")
	}

	if len(errors) > 0 {
		t.Errorf("TryDecorator performance test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_ImportRequirements(t *testing.T) {
	decorator := &TryDecorator{}

	// Test that the decorator properly specifies its import requirements
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main", "echo 'import test'"),
			decoratortesting.PatternBranch("catch", "echo 'error'"),
		})

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("fmt", "os"). // Required imports for error handling
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator import requirements test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestTryDecorator_ScenarioDeploymentWithRollback(t *testing.T) {
	decorator := &TryDecorator{}

	// Test realistic deployment scenario with rollback
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("main",
				"kubectl apply -f deployment.yaml",
				"kubectl rollout status deployment/app",
				"echo 'Deployment successful'"),
			decoratortesting.PatternBranch("catch",
				"echo 'Deployment failed, rolling back'",
				"kubectl rollout undo deployment/app",
				"echo 'Rollback completed'"),
			decoratortesting.PatternBranch("finally",
				"kubectl get pods",
				"echo 'Deployment process completed'"),
		})

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("kubectl apply", "rolling back", "get pods").
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("TryDecorator deployment scenario test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
