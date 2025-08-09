package decorators

import (
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestConfirmDecorator_Basic(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test basic confirm functionality with default message
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'confirmed'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("Do you want to continue?", "bufio.NewReader", "confirmed").
		PlanSucceeds().
		PlanReturnsElement("confirm").
		CompletesWithin("1s").
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_CustomMessage(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with custom confirmation message
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'custom message confirmed'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Deploy to production?"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("Deploy to production?").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator custom message test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_DefaultYes(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with defaultYes=true parameter
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'default yes'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Continue with default yes?"}},
			{Name: "defaultYes", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("[Y/n]", "confirmed = true").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator default yes test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_DefaultNo(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with defaultYes=false parameter (explicit)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'default no'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Continue with default no?"}},
			{Name: "defaultYes", Value: &ast.BooleanLiteral{Value: false}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("[y/N]", "confirmed = false").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator default no test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_AbortOnNo(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with abortOnNo=true parameter (default behavior)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'should abort if no'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Abort on no?"}},
			{Name: "abortOnNo", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("user cancelled execution").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator abort on no test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_SkipOnNo(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with abortOnNo=false parameter (skip execution)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'should skip if no'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Skip on no?"}},
			{Name: "abortOnNo", Value: &ast.BooleanLiteral{Value: false}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("return nil"). // Should return nil instead of error
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator skip on no test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_CaseSensitive(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with caseSensitive=true parameter
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'case sensitive'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Case sensitive confirmation?"}},
			{Name: "caseSensitive", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("response == \"y\" || response == \"Y\""). // Case sensitive matching
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator case sensitive test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_CaseInsensitive(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with caseSensitive=false parameter (default)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'case insensitive'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Case insensitive confirmation?"}},
			{Name: "caseSensitive", Value: &ast.BooleanLiteral{Value: false}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("strings.ToLower(response)"). // Case insensitive matching
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator case insensitive test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_CIDetection(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test CI detection functionality (default: skip in CI)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'CI auto-confirmed'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Deploy in CI?"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("CI environment detected", "envContext", "CI", "GITHUB_ACTIONS").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator CI detection test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_DisableCISkip(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with ci=false parameter (disable CI skip)
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'always prompt'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Always prompt?"}},
			{Name: "ci", Value: &ast.BooleanLiteral{Value: false}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		// Should not contain CI detection logic
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator disable CI skip test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_AllParameters(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with all parameters specified
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'all parameters'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Full parameter test?"}},
			{Name: "defaultYes", Value: &ast.BooleanLiteral{Value: true}},
			{Name: "abortOnNo", Value: &ast.BooleanLiteral{Value: false}},
			{Name: "caseSensitive", Value: &ast.BooleanLiteral{Value: true}},
			{Name: "ci", Value: &ast.BooleanLiteral{Value: false}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("Full parameter test?", "[Y/n]").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator all parameters test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_MultipleCommands(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with multiple commands to execute after confirmation
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'first command'"),
		decoratortesting.Shell("echo 'second command'"),
		decoratortesting.Shell("echo 'third command'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Execute multiple commands?"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("first command", "second command", "third command").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator multiple commands test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_NestedDecorators(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with nested decorators (confirm containing timeout)
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "timeout",
			Args: []ast.NamedParameter{
				{Name: "duration", Value: &ast.DurationLiteral{Value: "5s"}},
			},
			Content: []ast.CommandContent{
				decoratortesting.Shell("echo 'nested timeout'"),
			},
		},
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Confirm with nested timeout?"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator nested decorators test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_EmptyContent(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with no commands (edge case)
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Confirm with no commands?"}},
		}, []ast.CommandContent{})

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator empty content test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_SpecialCharactersInMessage(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with special characters in message
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'special chars'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Confirm with \"quotes\" and 'apostrophes'?"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		// Message should be properly escaped in generated code
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator special characters test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_LongMessage(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test with very long confirmation message
	longMessage := "This is a very long confirmation message that tests how the decorator handles extended text. " +
		"It includes multiple sentences and should be properly handled in all execution modes. " +
		"The message continues to test edge cases with lengthy user prompts."

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'long message confirmed'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: longMessage}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator long message test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_PerformanceCharacteristics(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test that the decorator itself doesn't add significant overhead
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'performance test'"),
	}

	start := time.Now()
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Performance test?"}},
		}, content)
	generatorDuration := time.Since(start)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		CompletesWithin("100ms"). // Should be very fast for generation
		Validate()

	// Additional check that generation is fast
	if generatorDuration > 100*time.Millisecond {
		errors = append(errors, "Confirm decorator generation is too slow")
	}

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator performance test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_ImportRequirements(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test that the decorator properly specifies its import requirements
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'import test'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "Import test?"}},
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("bufio", "fmt", "os", "strings"). // Required imports
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator import requirements test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestConfirmDecorator_ScenarioDeploymentConfirmation(t *testing.T) {
	decorator := &ConfirmDecorator{}

	// Test realistic deployment confirmation scenario
	content := []ast.CommandContent{
		decoratortesting.Shell("kubectl apply -f production.yaml"),
		decoratortesting.Shell("kubectl rollout status deployment/app"),
		decoratortesting.Shell("echo 'Deployment complete'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "message", Value: &ast.StringLiteral{Value: "🚀 Deploy to production environment?"}},
			{Name: "defaultYes", Value: &ast.BooleanLiteral{Value: false}}, // Default to no for safety
			{Name: "abortOnNo", Value: &ast.BooleanLiteral{Value: true}},   // Abort if declined
		}, content)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("Deploy to production", "kubectl apply", "user cancelled execution").
		PlanSucceeds().
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("ConfirmDecorator deployment scenario test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
