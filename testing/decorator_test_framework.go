package testing

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// DecoratorTestSuite provides a comprehensive testing framework for decorators
// that is completely independent of the engine and focuses on decorator validation
type DecoratorTestSuite struct {
	t         *testing.T
	decorator decorators.Decorator
	program   *ast.Program
	variables map[string]string
	env       map[string]string
}

// NewDecoratorTest creates a new independent decorator test suite
func NewDecoratorTest(t *testing.T, decorator decorators.Decorator) *DecoratorTestSuite {
	return &DecoratorTestSuite{
		t:         t,
		decorator: decorator,
		program:   ast.NewProgram(),
		variables: make(map[string]string),
		env:       make(map[string]string),
	}
}

// WithVariable adds a variable to the test environment
func (d *DecoratorTestSuite) WithVariable(name, value string) *DecoratorTestSuite {
	d.variables[name] = value
	return d
}

// WithEnv adds an environment variable to the test environment
func (d *DecoratorTestSuite) WithEnv(name, value string) *DecoratorTestSuite {
	d.env[name] = value
	return d
}

// WithCommand adds a command definition to the test program
func (d *DecoratorTestSuite) WithCommand(name string, content ...string) *DecoratorTestSuite {
	// Create shell content for each line
	var commandContent []ast.CommandContent
	for _, line := range content {
		commandContent = append(commandContent, &ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{
					Text: line,
				},
			},
		})
	}

	// Add command to program
	d.program.Commands = append(d.program.Commands, ast.CommandDecl{
		Name: name,
		Body: ast.CommandBody{
			Content: commandContent,
		},
	})

	return d
}

// TestResult contains the results from testing a decorator in a specific mode
type TestResult struct {
	Mode     string
	Success  bool
	Data     interface{}
	Error    error
	Duration time.Duration
}

// ValidationResult contains comprehensive validation results across all modes
type ValidationResult struct {
	InterpreterResult TestResult
	GeneratorResult   TestResult
	PlanResult        TestResult
	StructuralValid   bool
	ValidationErrors  []string
}

// === VALUE DECORATOR TESTING ===

// TestValueDecorator tests a ValueDecorator across all modes with comprehensive validation
func (d *DecoratorTestSuite) TestValueDecorator(params []ast.NamedParameter) ValidationResult {
	valueDecorator, ok := d.decorator.(decorators.ValueDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not a ValueDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := valueDecorator.ExpandInterpreter(ctx, params)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		templateResult, err := valueDecorator.GenerateTemplate(ctx, params)
		duration := time.Since(start)

		// Execute template to get actual generated code
		var generatedCode string
		var templateExecError error
		if err == nil && templateResult != nil && templateResult.Template != nil {
			var buf strings.Builder
			if templateExecError = templateResult.Template.Execute(&buf, templateResult.Data); templateExecError == nil {
				generatedCode = strings.TrimSpace(buf.String())
			}
		}

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  err == nil && templateExecError == nil,
			Data:     generatedCode, // Store executed template code, not TemplateResult
			Error:    err,
			Duration: duration,
		}

		// Store template exec error if it occurred
		if templateExecError != nil {
			result.GeneratorResult.Error = templateExecError
		}

		d.validateGeneratorTemplateResult(templateResult, err, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := valueDecorator.ExpandPlan(ctx, params)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode structural validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === ACTION DECORATOR TESTING ===

// TestActionDecorator tests an ActionDecorator across all modes
func (d *DecoratorTestSuite) TestActionDecorator(params []ast.NamedParameter) ValidationResult {
	actionDecorator, ok := d.decorator.(decorators.ActionDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not an ActionDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := actionDecorator.ExpandInterpreter(ctx, params)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		templateResult, err := actionDecorator.GenerateTemplate(ctx, params)
		duration := time.Since(start)

		// Execute template to get actual generated code
		var generatedCode string
		var templateExecError error
		if err == nil && templateResult != nil && templateResult.Template != nil {
			var buf strings.Builder
			if templateExecError = templateResult.Template.Execute(&buf, templateResult.Data); templateExecError == nil {
				generatedCode = strings.TrimSpace(buf.String())
			}
		}

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  err == nil && templateExecError == nil,
			Data:     generatedCode, // Store executed template code, not TemplateResult
			Error:    err,
			Duration: duration,
		}

		// Store template exec error if it occurred
		if templateExecError != nil {
			result.GeneratorResult.Error = templateExecError
		}

		d.validateGeneratorTemplateResult(templateResult, err, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := actionDecorator.ExpandPlan(ctx, params)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === BLOCK DECORATOR TESTING ===

// TestBlockDecorator tests a BlockDecorator across all modes
func (d *DecoratorTestSuite) TestBlockDecorator(params []ast.NamedParameter, content []ast.CommandContent) ValidationResult {
	blockDecorator, ok := d.decorator.(decorators.BlockDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not a BlockDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := blockDecorator.ExecuteInterpreter(ctx, params, content)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		templateResult, err := blockDecorator.GenerateTemplate(ctx, params, content)
		duration := time.Since(start)

		// Execute template to get actual generated code
		var generatedCode string
		var templateExecError error
		if err == nil && templateResult != nil && templateResult.Template != nil {
			var buf strings.Builder
			if templateExecError = templateResult.Template.Execute(&buf, templateResult.Data); templateExecError == nil {
				generatedCode = strings.TrimSpace(buf.String())
			}
		}

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  err == nil && templateExecError == nil,
			Data:     generatedCode, // Store executed template code, not TemplateResult
			Error:    err,
			Duration: duration,
		}

		// Store template exec error if it occurred
		if templateExecError != nil {
			result.GeneratorResult.Error = templateExecError
		}

		d.validateGeneratorTemplateResult(templateResult, err, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := blockDecorator.ExecutePlan(ctx, params, content)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === PATTERN DECORATOR TESTING ===

// TestPatternDecorator tests a PatternDecorator across all modes
func (d *DecoratorTestSuite) TestPatternDecorator(params []ast.NamedParameter, patterns []ast.PatternBranch) ValidationResult {
	patternDecorator, ok := d.decorator.(decorators.PatternDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not a PatternDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := patternDecorator.ExecuteInterpreter(ctx, params, patterns)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		templateResult, err := patternDecorator.GenerateTemplate(ctx, params, patterns)
		duration := time.Since(start)

		// Execute template to get actual generated code
		var generatedCode string
		var templateExecError error
		if err == nil && templateResult != nil && templateResult.Template != nil {
			var buf strings.Builder
			if templateExecError = templateResult.Template.Execute(&buf, templateResult.Data); templateExecError == nil {
				generatedCode = strings.TrimSpace(buf.String())
			}
		}

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  err == nil && templateExecError == nil,
			Data:     generatedCode, // Store executed template code, not TemplateResult
			Error:    err,
			Duration: duration,
		}

		// Store template exec error if it occurred
		if templateExecError != nil {
			result.GeneratorResult.Error = templateExecError
		}

		d.validateGeneratorTemplateResult(templateResult, err, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := patternDecorator.ExecutePlan(ctx, params, patterns)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === CONTEXT CREATION ===

func (d *DecoratorTestSuite) createInterpreterContext() execution.InterpreterContext {
	ctx := execution.NewInterpreterContext(context.Background(), d.program)

	// CRITICAL: Set up decorator lookup functions FIRST, before any other operations
	// This ensures decorators are available during interpreter execution
	d.setupInterpreterDecoratorLookups(ctx)

	// Set up variables
	for name, value := range d.variables {
		ctx.SetVariable(name, value)
	}

	if err := ctx.InitializeVariables(); err != nil {
		d.t.Fatalf("Failed to initialize interpreter context: %v", err)
	}

	return ctx
}

func (d *DecoratorTestSuite) createGeneratorContext() execution.GeneratorContext {
	ctx := execution.NewGeneratorContext(context.Background(), d.program)

	// CRITICAL: Set up decorator lookup functions FIRST, before any other operations
	// This ensures they're available when template functions are created
	d.setupDecoratorLookups(ctx)

	// Set up variables
	for name, value := range d.variables {
		ctx.SetVariable(name, value)
	}

	// Set up environment variables for tracking
	for name, value := range d.env {
		// In a real scenario, we'd set the actual env var too
		// but for testing we just track it
		// TODO: Add proper env var tracking when method is available
		_ = name
		_ = value
	}

	if err := ctx.InitializeVariables(); err != nil {
		d.t.Fatalf("Failed to initialize generator context: %v", err)
	}

	return ctx
}

func (d *DecoratorTestSuite) createPlanContext() execution.PlanContext {
	ctx := execution.NewPlanContext(context.Background(), d.program)

	// Set up variables
	for name, value := range d.variables {
		ctx.SetVariable(name, value)
	}

	if err := ctx.InitializeVariables(); err != nil {
		d.t.Fatalf("Failed to initialize plan context: %v", err)
	}

	return ctx
}

// === MODE-SPECIFIC VALIDATION ===

func (d *DecoratorTestSuite) validateInterpreterResult(execResult *execution.ExecutionResult, result *ValidationResult) {
	// Interpreter mode should either succeed or fail gracefully
	// Data can be anything or nil
	// Error should be descriptive if present

	if execResult.Error != nil && strings.TrimSpace(execResult.Error.Error()) == "" {
		result.ValidationErrors = append(result.ValidationErrors,
			"Interpreter mode returned empty error message")
	}
}

func (d *DecoratorTestSuite) validateGeneratorTemplateResult(templateResult *execution.TemplateResult, err error, result *ValidationResult) {
	// Generator mode should return valid template result
	if err == nil {
		if templateResult == nil {
			result.ValidationErrors = append(result.ValidationErrors,
				"Generator mode returned nil template result")
		} else {
			// Validate template can be executed
			if templateResult.Template == nil {
				result.ValidationErrors = append(result.ValidationErrors,
					"Template result has nil template")
			}
			if templateResult.Data == nil {
				result.ValidationErrors = append(result.ValidationErrors,
					"Template result has nil data")
			}

			// Try to execute template to validate syntax
			if templateResult.Template != nil && templateResult.Data != nil {
				var buf strings.Builder
				if err := templateResult.Template.Execute(&buf, templateResult.Data); err != nil {
					result.ValidationErrors = append(result.ValidationErrors,
						fmt.Sprintf("Template execution failed: %v", err))
				} else {
					code := strings.TrimSpace(buf.String())
					if code == "" {
						result.ValidationErrors = append(result.ValidationErrors,
							"Template executed to empty code")
					}
				}
			}
		}
	}
}

func (d *DecoratorTestSuite) validatePlanResult(execResult *execution.ExecutionResult, result *ValidationResult) {
	// Plan mode should return plan data structure
	if execResult.Error == nil {
		if execResult.Data == nil {
			result.ValidationErrors = append(result.ValidationErrors,
				"Plan mode returned nil data - expected plan element")
		}
		// TODO: Add plan structure validation here
	}
}

func (d *DecoratorTestSuite) validateCrossModeConsistency(result *ValidationResult) {
	// Check that modes are consistent with each other
	// For example, if interpreter fails, generator might still work
	// but they should fail for similar reasons

	result.StructuralValid = len(result.ValidationErrors) == 0

	// Add more cross-mode validation logic here
	// - Parameter handling consistency
	// - Error condition consistency
	// - Data type consistency where applicable
}

// setupInterpreterDecoratorLookups configures decorator registry access for interpreter context
func (d *DecoratorTestSuite) setupInterpreterDecoratorLookups(ctx execution.InterpreterContext) {
	// Cast to the concrete type to access the setup methods
	if interpreterCtx, ok := ctx.(*execution.InterpreterExecutionContext); ok {
		// Set up action decorator lookup function using the decorator registry
		interpreterCtx.SetActionDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, exists := decorators.GetActionDecorator(name)
			return decorator, exists
		})

		// Set up value decorator lookup function using the decorator registry
		interpreterCtx.SetValueDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetValue(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})

		// Set up block decorator lookup function using the decorator registry
		interpreterCtx.SetBlockDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetBlock(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})
	}
}

// setupDecoratorLookups configures decorator registry access for testing nested decorators
func (d *DecoratorTestSuite) setupDecoratorLookups(ctx execution.GeneratorContext) {
	// Cast to the concrete type to access the setup methods
	if generatorCtx, ok := ctx.(*execution.GeneratorExecutionContext); ok {
		// Set up block decorator lookup function using the decorator registry
		generatorCtx.SetBlockDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetBlock(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})

		// Set up pattern decorator lookup function using the decorator registry
		generatorCtx.SetPatternDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetPattern(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})

		// Set up action decorator lookup function using the decorator registry
		generatorCtx.SetActionDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, exists := decorators.GetActionDecorator(name)
			return decorator, exists
		})

		// Set up value decorator lookup function using the decorator registry
		generatorCtx.SetValueDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetValue(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})
	}
}
