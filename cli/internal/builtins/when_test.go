package decorators

import (
	"os"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestWhenDecorator_BasicMatching(t *testing.T) {
	decorator := &WhenDecorator{}

	// Set environment for testing
	if err := os.Setenv("NODE_ENV", "production"); err != nil {
		t.Fatalf("Failed to set test env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("NODE_ENV"); err != nil {
			t.Logf("Warning: Failed to unset test env var: %v", err)
		}
	}()

	patterns := []ast.PatternBranch{
		decoratortesting.PatternBranch("production", "echo 'prod mode'", "echo 'building for production'"),
		decoratortesting.PatternBranch("development", "echo 'dev mode'", "echo 'starting dev server'"),
		decoratortesting.PatternBranch("*", "echo 'unknown mode'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "NODE_ENV"),
		}, patterns)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("NODE_ENV", "production").
		PlanSucceeds().
		PlanReturnsElement("conditional").
		CompletesWithin("100ms").
		ModesAreConsistent().
		SupportsDevcmdChaining().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator basic matching test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_DefaultWildcard(t *testing.T) {
	decorator := &WhenDecorator{}

	// Set environment to something not explicitly matched
	if err := os.Setenv("DEPLOY_ENV", "staging"); err != nil {
		t.Fatalf("Failed to set test env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("DEPLOY_ENV"); err != nil {
			t.Logf("Warning: Failed to unset test env var: %v", err)
		}
	}()

	patterns := []ast.PatternBranch{
		decoratortesting.PatternBranch("production", "echo 'prod'"),
		decoratortesting.PatternBranch("development", "echo 'dev'"),
		decoratortesting.PatternBranch("*", "echo 'default case'", "echo 'fallback'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "DEPLOY_ENV"),
		}, patterns)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("DEPLOY_ENV", "default").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator default wildcard test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_UndefinedVariable(t *testing.T) {
	decorator := &WhenDecorator{}

	patterns := []ast.PatternBranch{
		decoratortesting.PatternBranch("production", "echo 'prod'"),
		decoratortesting.PatternBranch("*", "echo 'default'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "UNDEFINED_VAR"),
		}, patterns)

	// Should fall back to wildcard pattern for undefined variables
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator undefined variable test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_NoWildcard(t *testing.T) {
	decorator := &WhenDecorator{}

	// Set environment that doesn't match any pattern
	if err := os.Setenv("TEST_ENV", "unknown"); err != nil {
		t.Fatalf("Failed to set test env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_ENV"); err != nil {
			t.Logf("Warning: Failed to unset test env var: %v", err)
		}
	}()

	patterns := []ast.PatternBranch{
		decoratortesting.PatternBranch("production", "echo 'prod'"),
		decoratortesting.PatternBranch("development", "echo 'dev'"),
		// No wildcard pattern
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "TEST_ENV"),
		}, patterns)

	// Should handle gracefully when no pattern matches
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator no wildcard test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_EmptyPatterns(t *testing.T) {
	decorator := &WhenDecorator{}

	// Test with no patterns
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "TEST_VAR"),
		}, []ast.PatternBranch{})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds(). // Should handle gracefully
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator empty patterns test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_NestedDecorators(t *testing.T) {
	decorator := &WhenDecorator{}

	if err := os.Setenv("BUILD_TYPE", "release"); err != nil {
		t.Fatalf("Failed to set test env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("BUILD_TYPE"); err != nil {
			t.Logf("Warning: Failed to unset test env var: %v", err)
		}
	}()

	// Test with nested decorators in pattern branches
	patterns := []ast.PatternBranch{
		{
			Pattern: &ast.IdentifierPattern{Name: "release"},
			Commands: []ast.CommandContent{
				&ast.BlockDecorator{
					Name: "timeout",
					Args: []ast.NamedParameter{
						{Name: "duration", Value: &ast.DurationLiteral{Value: "10m"}},
					},
					Content: []ast.CommandContent{
						decoratortesting.Shell("go build -ldflags '-s -w' ./..."),
					},
				},
			},
		},
		{
			Pattern: &ast.IdentifierPattern{Name: "debug"},
			Commands: []ast.CommandContent{
				decoratortesting.Shell("go build -race ./..."),
			},
		},
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "BUILD_TYPE"),
		}, patterns)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("BUILD_TYPE", "release").
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator nested decorators test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_ParameterValidation(t *testing.T) {
	decorator := &WhenDecorator{}

	// Test missing variable parameter
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{}, []ast.PatternBranch{
			decoratortesting.PatternBranch("test", "echo 'test'"),
		})

	errors := decoratortesting.Assert(result).
		InterpreterFails("requires exactly 1 parameter").
		GeneratorFails("requires exactly 1 parameter").
		PlanFails("requires exactly 1 parameter").
		Validate()

	if len(errors) > 0 {
		t.Errorf("WhenDecorator parameter validation test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWhenDecorator_GeneratorVariableResolution(t *testing.T) {
	decorator := &WhenDecorator{}

	// Critical test: Generator mode should NOT resolve variables at generation time
	// It should generate code that resolves them at runtime

	patterns := []ast.PatternBranch{
		decoratortesting.PatternBranch("ci", "echo 'CI build'"),
		decoratortesting.PatternBranch("local", "echo 'Local build'"),
		decoratortesting.PatternBranch("*", "echo 'Unknown build'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestPatternDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("variable", "CI_ENVIRONMENT"),
		}, patterns)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		// Generated code should contain runtime variable resolution, not hardcoded values
		GeneratorCodeContains("CI_ENVIRONMENT").
		Validate()

	// Additional check: generated code should NOT contain evidence of generation-time resolution
	if result.GeneratorResult.Success {
		if code, ok := result.GeneratorResult.Data.(string); ok {
			if decoratortesting.ContainsExecutionEvidence(code) {
				errors = append(errors, "CRITICAL: Generator resolved variables at generation time!")
			}
		}
	}

	if len(errors) > 0 {
		t.Errorf("WhenDecorator generator variable resolution test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
