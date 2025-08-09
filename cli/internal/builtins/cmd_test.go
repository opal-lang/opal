package decorators

import (
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestCmdDecorator_Basic(t *testing.T) {
	decorator := &CmdDecorator{}

	// Test basic functionality
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithCommand("test_cmd", "echo 'hello from test_cmd'").
		TestActionDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "test_cmd"),
		})

	// Validate results across all modes
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("execute").
		PlanSucceeds().
		PlanReturnsElement("command").
		CompletesWithin("100ms").
		ModesAreConsistent().
		SupportsDevcmdChaining().
		Validate()

	if len(errors) > 0 {
		t.Errorf("CmdDecorator basic test failed with %d errors:\n%s", len(errors), decoratortesting.JoinErrors(errors))
	}
}

func TestCmdDecorator_InvalidCommand(t *testing.T) {
	decorator := &CmdDecorator{}

	// Test with non-existent command
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestActionDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "NON_EXISTENT_COMMAND"),
		})

	// All modes should fail for invalid command
	errors := decoratortesting.Assert(result).
		InterpreterFails("not found").
		GeneratorSucceeds(). // Generator can still produce code
		GeneratorProducesValidGo().
		PlanFails("not found"). // Plan should also fail for invalid command
		Validate()

	if len(errors) > 0 {
		t.Errorf("CmdDecorator invalid command test failed with %d errors:\n%s", len(errors), decoratortesting.JoinErrors(errors))
	}
}

func TestCmdDecorator_ParameterValidation(t *testing.T) {
	decorator := &CmdDecorator{}

	// Test with missing required parameter
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestActionDecorator([]ast.NamedParameter{})

	// All modes should fail due to missing parameter
	errors := decoratortesting.Assert(result).
		InterpreterFails("command name parameter").
		GeneratorFails("command name parameter").
		PlanFails("command name parameter").
		ValidatesParameters(decorator, []ast.NamedParameter{}).
		Validate()

	if len(errors) > 0 {
		t.Errorf("CmdDecorator parameter validation test failed with %d errors:\n%s", len(errors), decoratortesting.JoinErrors(errors))
	}
}

func TestCmdDecorator_ChainedExecution(t *testing.T) {
	decorator := &CmdDecorator{}

	// Test that cmd decorator can be used in chained scenarios
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithCommand("chain_cmd", "echo 'chain test'", "echo 'second line'").
		TestActionDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "chain_cmd"),
		})

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		SupportsDevcmdChaining().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("CmdDecorator chaining test failed with %d errors:\n%s", len(errors), decoratortesting.JoinErrors(errors))
	}
}

func TestCmdDecorator_ModeIsolation(t *testing.T) {
	decorator := &CmdDecorator{}

	// Critical test: ensure generator mode NEVER executes commands
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithCommand("dangerous_cmd", "echo 'EXECUTION_DETECTED: This should never run in generator mode'").
		TestActionDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "dangerous_cmd"),
		})

	// Generator mode should produce code but never execute
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		Validate()

	// Additional check: generated code should contain function call, not execution
	if result.GeneratorResult.Success {
		if code, ok := result.GeneratorResult.Data.(string); ok {
			if decoratortesting.ContainsExecutionEvidence(code) {
				errors = append(errors, "CRITICAL: Generator mode contains evidence of command execution!")
			}
		}
	}

	if len(errors) > 0 {
		t.Errorf("CmdDecorator mode isolation test failed with %d errors:\n%s", len(errors), decoratortesting.JoinErrors(errors))
	}
}

func TestCmdDecorator_ComprehensiveValidation(t *testing.T) {
	decorator := &CmdDecorator{}

	// Comprehensive test covering all aspects
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithCommand("comprehensive_cmd", "echo 'comprehensive test'", "pwd", "echo 'done'").
		WithVariable("ARG", "test").
		WithEnv("PATH", "/usr/bin").
		TestActionDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "comprehensive_cmd"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("execute", "Comprehensive").
		PlanSucceeds().
		PlanReturnsElement("command").
		CompletesWithin("500ms").
		ModesAreConsistent().
		ValidatesParameters(decorator, nil).
		SupportsDevcmdChaining().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("CmdDecorator comprehensive validation failed with %d errors:\n%s", len(errors), decoratortesting.JoinErrors(errors))
	}
}
