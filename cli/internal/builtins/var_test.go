package decorators

import (
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestVarDecorator_Basic(t *testing.T) {
	decorator := &VarDecorator{}

	// Test basic variable resolution
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithVariable("MY_VAR", "hello_world").
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "MY_VAR"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns("hello_world").
		GeneratorSucceeds().
		GeneratorCodeContains("MY_VAR").
		PlanSucceeds().
		CompletesWithin("50ms").
		ModesAreConsistent().
		Validate()

	if len(errors) > 0 {
		t.Errorf("VarDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestVarDecorator_UndefinedVariable(t *testing.T) {
	decorator := &VarDecorator{}

	// Test undefined variable handling
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "UNDEFINED_VAR"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterFails("not defined").
		GeneratorSucceeds(). // Generator should work for planning
		PlanSucceeds().      // Plan should show undefined variables
		Validate()

	if len(errors) > 0 {
		t.Errorf("VarDecorator undefined variable test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestVarDecorator_EmptyValue(t *testing.T) {
	decorator := &VarDecorator{}

	// Test empty variable value
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithVariable("EMPTY_VAR", "").
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "EMPTY_VAR"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns("").
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("VarDecorator empty value test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestVarDecorator_SpecialCharacters(t *testing.T) {
	decorator := &VarDecorator{}

	// Test variable with special characters
	result := decoratortesting.NewDecoratorTest(t, decorator).
		WithVariable("SPECIAL_VAR", "hello world!@#$%^&*()").
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.IdentifierParam("", "SPECIAL_VAR"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns("hello world!@#$%^&*()").
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("VarDecorator special characters test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestVarDecorator_NoParameter(t *testing.T) {
	decorator := &VarDecorator{}

	// Test missing parameter
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{})

	errors := decoratortesting.Assert(result).
		InterpreterFails("requires exactly 1 parameter").
		GeneratorFails("requires exactly 1 parameter").
		PlanFails("requires exactly 1 parameter").
		Validate()

	if len(errors) > 0 {
		t.Errorf("VarDecorator no parameter test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
