package testing

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// DecoratorHarness provides an extensible testing framework with custom assertions
type DecoratorHarness struct {
	t                *testing.T
	decorator        decorators.Decorator
	testCases        map[string]TestCase
	customAssertions map[string]CustomAssertion
	program          *ast.Program
	variables        map[string]string
	env              map[string]string
}

// TestCase represents a single test scenario
type TestCase struct {
	Name        string
	Description string
	Params      []ast.NamedParameter
	Content     []ast.CommandContent
	Patterns    []ast.PatternBranch // For pattern decorators
}

// CustomAssertion is a function that validates decorator-specific behavior
type CustomAssertion func(ctx TestContext) error

// TestContext provides all execution results and artifacts to custom assertions
type TestContext struct {
	TestCase          TestCase
	InterpreterResult TestResult
	GeneratorResult   TestResult
	PlanResult        TestResult
	GeneratedCode     string
	CompiledBinary    *CompiledProgram
	ExecutionOutput   *ExecutionOutput
	PlanOutput        interface{}
}

// CompiledProgram represents a compiled test program
type CompiledProgram struct {
	BinaryPath   string
	TempDir      string
	CompileError error
	CompileTime  time.Duration
}

// ExecutionOutput captures the output of running a compiled program
type ExecutionOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// TestReport contains comprehensive results from running all validations
type TestReport struct {
	Decorator      string
	TotalTestCases int
	CorePassed     int
	CustomPassed   int
	Failed         []TestFailure

	// Performance metrics
	TotalDuration        time.Duration
	AverageCompileTime   time.Duration
	AverageExecutionTime time.Duration

	// Debugging artifacts
	GeneratedArtifacts map[string]string
}

// TestFailure represents a failed test assertion
type TestFailure struct {
	TestCase      string
	AssertionName string
	Error         error
	Context       string
}

// NewDecoratorHarness creates a new extensible decorator test harness
func NewDecoratorHarness(t *testing.T, decorator decorators.Decorator) *DecoratorHarness {
	return &DecoratorHarness{
		t:                t,
		decorator:        decorator,
		testCases:        make(map[string]TestCase),
		customAssertions: make(map[string]CustomAssertion),
		program:          ast.NewProgram(),
		variables:        make(map[string]string),
		env:              make(map[string]string),
	}
}

// WithTestCase adds a test case to the harness
func (h *DecoratorHarness) WithTestCase(name string, params []ast.NamedParameter, content []ast.CommandContent) *DecoratorHarness {
	h.testCases[name] = TestCase{
		Name:    name,
		Params:  params,
		Content: content,
	}
	return h
}

// WithPatternTestCase adds a pattern decorator test case
func (h *DecoratorHarness) WithPatternTestCase(name string, params []ast.NamedParameter, patterns []ast.PatternBranch) *DecoratorHarness {
	h.testCases[name] = TestCase{
		Name:     name,
		Params:   params,
		Patterns: patterns,
	}
	return h
}

// WithVariable adds a variable to the test environment
func (h *DecoratorHarness) WithVariable(name, value string) *DecoratorHarness {
	h.variables[name] = value
	return h
}

// WithEnv adds an environment variable to the test environment
func (h *DecoratorHarness) WithEnv(name, value string) *DecoratorHarness {
	h.env[name] = value
	return h
}

// WithCustomAssertion adds a custom assertion function
func (h *DecoratorHarness) WithCustomAssertion(name string, assertion CustomAssertion) *DecoratorHarness {
	h.customAssertions[name] = assertion
	return h
}

// Execute runs all core validations and custom assertions
func (h *DecoratorHarness) Execute() TestReport {
	startTime := time.Now()
	report := TestReport{
		Decorator:          h.decorator.Name(),
		TotalTestCases:     len(h.testCases),
		Failed:             []TestFailure{},
		GeneratedArtifacts: make(map[string]string),
	}

	var totalCompileTime, totalExecutionTime time.Duration
	compileCount := 0

	for testName, testCase := range h.testCases {
		h.t.Run(testName, func(t *testing.T) {
			// Execute test case in all modes
			ctx, err := h.executeTestCase(testCase)
			if err != nil {
				report.Failed = append(report.Failed, TestFailure{
					TestCase:      testName,
					AssertionName: "execution",
					Error:         err,
					Context:       "Failed to execute test case in all modes",
				})
				return
			}

			// Store generated artifacts for debugging
			report.GeneratedArtifacts[fmt.Sprintf("%s_generated_code", testName)] = ctx.GeneratedCode

			// Track performance
			if ctx.CompiledBinary != nil {
				totalCompileTime += ctx.CompiledBinary.CompileTime
				compileCount++
			}
			if ctx.ExecutionOutput != nil {
				totalExecutionTime += ctx.ExecutionOutput.Duration
			}

			// Run core validations
			coreResults := h.runCoreValidations(ctx)
			if len(coreResults) == 0 {
				report.CorePassed++
			} else {
				for _, failure := range coreResults {
					failure.TestCase = testName
					report.Failed = append(report.Failed, failure)
				}
			}

			// Run custom assertions
			customResults := h.runCustomAssertions(ctx)
			if len(customResults) == 0 {
				report.CustomPassed++
			} else {
				for _, failure := range customResults {
					failure.TestCase = testName
					report.Failed = append(report.Failed, failure)
				}
			}
		})
	}

	// Calculate performance metrics
	report.TotalDuration = time.Since(startTime)
	if compileCount > 0 {
		report.AverageCompileTime = totalCompileTime / time.Duration(compileCount)
		report.AverageExecutionTime = totalExecutionTime / time.Duration(compileCount)
	}

	// Print summary
	h.printTestSummary(report)

	return report
}

// executeTestCase runs a test case in all three modes and returns comprehensive context
func (h *DecoratorHarness) executeTestCase(testCase TestCase) (TestContext, error) {
	ctx := TestContext{TestCase: testCase}

	// Create execution contexts
	interpreterCtx := h.createInterpreterContext()
	generatorCtx := h.createGeneratorContext()
	planCtx := h.createPlanContext()

	// Execute in interpreter mode
	ctx.InterpreterResult = h.executeInterpreter(interpreterCtx, testCase)

	// Execute in generator mode
	ctx.GeneratorResult = h.executeGenerator(generatorCtx, testCase)
	if ctx.GeneratorResult.Success {
		if code, ok := ctx.GeneratorResult.Data.(string); ok {
			ctx.GeneratedCode = code
		} else if templateResult, ok := ctx.GeneratorResult.Data.(*execution.TemplateResult); ok {
			// Execute template to get generated code
			var buf bytes.Buffer
			if err := templateResult.Template.Execute(&buf, templateResult.Data); err != nil {
				return ctx, fmt.Errorf("failed to execute template: %w", err)
			}
			ctx.GeneratedCode = buf.String()
		}

		// Compile and execute generated code
		compiledProgram, err := h.compileGeneratedCode(ctx.GeneratedCode, testCase.Name)
		if err != nil {
			return ctx, fmt.Errorf("failed to compile generated code: %w", err)
		}
		ctx.CompiledBinary = compiledProgram

		if compiledProgram.CompileError == nil {
			execOutput, err := h.executeCompiledProgram(compiledProgram)
			if err != nil {
				return ctx, fmt.Errorf("failed to execute compiled program: %w", err)
			}
			ctx.ExecutionOutput = execOutput
		}
	}

	// Execute in plan mode
	ctx.PlanResult = h.executePlan(planCtx, testCase)
	ctx.PlanOutput = ctx.PlanResult.Data

	return ctx, nil
}

// executeInterpreter runs the test case in interpreter mode
func (h *DecoratorHarness) executeInterpreter(ctx execution.InterpreterContext, testCase TestCase) TestResult {
	start := time.Now()

	var result *execution.ExecutionResult

	// Use GetDecoratorType to avoid interface collision issues
	decoratorType := decorators.GetDecoratorType(h.decorator)
	switch decoratorType {
	case decorators.PatternType:
		if decorator, ok := h.decorator.(decorators.PatternDecorator); ok {
			result = decorator.ExecuteInterpreter(ctx, testCase.Params, testCase.Patterns)
		}
	case decorators.BlockType:
		if decorator, ok := h.decorator.(decorators.BlockDecorator); ok {
			result = decorator.ExecuteInterpreter(ctx, testCase.Params, testCase.Content)
		}
	case decorators.ActionType:
		if decorator, ok := h.decorator.(decorators.ActionDecorator); ok {
			result = decorator.ExpandInterpreter(ctx, testCase.Params)
		}
	case decorators.ValueType:
		if decorator, ok := h.decorator.(decorators.ValueDecorator); ok {
			result = decorator.ExpandInterpreter(ctx, testCase.Params)
		}
	default:
		return TestResult{
			Mode:     "interpreter",
			Success:  false,
			Error:    fmt.Errorf("unsupported decorator type: %T", h.decorator),
			Duration: time.Since(start),
		}
	}

	return TestResult{
		Mode:     "interpreter",
		Success:  result.Error == nil,
		Data:     result.Data,
		Error:    result.Error,
		Duration: time.Since(start),
	}
}

// executeGenerator runs the test case in generator mode
func (h *DecoratorHarness) executeGenerator(ctx execution.GeneratorContext, testCase TestCase) TestResult {
	start := time.Now()

	// Use GetDecoratorType to avoid interface collision issues
	decoratorType := decorators.GetDecoratorType(h.decorator)
	switch decoratorType {
	case decorators.PatternType:
		if decorator, ok := h.decorator.(decorators.PatternDecorator); ok {
			templateResult, err := decorator.GenerateTemplate(ctx, testCase.Params, testCase.Patterns)
			return TestResult{
				Mode:     "generator",
				Success:  err == nil,
				Data:     templateResult,
				Error:    err,
				Duration: time.Since(start),
			}
		}
	case decorators.BlockType:
		if decorator, ok := h.decorator.(decorators.BlockDecorator); ok {
			templateResult, err := decorator.GenerateTemplate(ctx, testCase.Params, testCase.Content)
			return TestResult{
				Mode:     "generator",
				Success:  err == nil,
				Data:     templateResult,
				Error:    err,
				Duration: time.Since(start),
			}
		}
	case decorators.ActionType:
		if decorator, ok := h.decorator.(decorators.ActionDecorator); ok {
			templateResult, err := decorator.GenerateTemplate(ctx, testCase.Params)
			return TestResult{
				Mode:     "generator",
				Success:  err == nil,
				Data:     templateResult,
				Error:    err,
				Duration: time.Since(start),
			}
		}
	case decorators.ValueType:
		if decorator, ok := h.decorator.(decorators.ValueDecorator); ok {
			templateResult, err := decorator.GenerateTemplate(ctx, testCase.Params)
			return TestResult{
				Mode:     "generator",
				Success:  err == nil,
				Data:     templateResult,
				Error:    err,
				Duration: time.Since(start),
			}
		}
	default:
		return TestResult{
			Mode:     "generator",
			Success:  false,
			Error:    fmt.Errorf("unsupported decorator type: %T", h.decorator),
			Duration: time.Since(start),
		}
	}

	// Fallback if type assertion failed
	return TestResult{
		Mode:     "generator",
		Success:  false,
		Error:    fmt.Errorf("failed to execute decorator of type %T", h.decorator),
		Duration: time.Since(start),
	}
}

// executePlan runs the test case in plan mode
func (h *DecoratorHarness) executePlan(ctx execution.PlanContext, testCase TestCase) TestResult {
	start := time.Now()

	var result *execution.ExecutionResult

	// Use GetDecoratorType to avoid interface collision issues
	decoratorType := decorators.GetDecoratorType(h.decorator)
	switch decoratorType {
	case decorators.PatternType:
		if decorator, ok := h.decorator.(decorators.PatternDecorator); ok {
			result = decorator.ExecutePlan(ctx, testCase.Params, testCase.Patterns)
		}
	case decorators.BlockType:
		if decorator, ok := h.decorator.(decorators.BlockDecorator); ok {
			result = decorator.ExecutePlan(ctx, testCase.Params, testCase.Content)
		}
	case decorators.ActionType:
		if decorator, ok := h.decorator.(decorators.ActionDecorator); ok {
			result = decorator.ExpandPlan(ctx, testCase.Params)
		}
	case decorators.ValueType:
		if decorator, ok := h.decorator.(decorators.ValueDecorator); ok {
			result = decorator.ExpandPlan(ctx, testCase.Params)
		}
	default:
		return TestResult{
			Mode:     "plan",
			Success:  false,
			Error:    fmt.Errorf("unsupported decorator type: %T", h.decorator),
			Duration: time.Since(start),
		}
	}

	return TestResult{
		Mode:     "plan",
		Success:  result.Error == nil,
		Data:     result.Data,
		Error:    result.Error,
		Duration: time.Since(start),
	}
}

// Context creation methods
func (h *DecoratorHarness) createInterpreterContext() execution.InterpreterContext {
	ctx := execution.NewInterpreterContext(context.Background(), h.program)

	// CRITICAL: Set up decorator lookup functions FIRST, before any other operations
	// This ensures decorators are available during interpreter execution
	h.setupInterpreterDecoratorLookups(ctx)

	for name, value := range h.variables {
		ctx.SetVariable(name, value)
	}

	if err := ctx.InitializeVariables(); err != nil {
		fmt.Printf("Warning: failed to initialize variables: %v\n", err)
	}
	return ctx
}

func (h *DecoratorHarness) createGeneratorContext() execution.GeneratorContext {
	ctx := execution.NewGeneratorContext(context.Background(), h.program)

	for name, value := range h.variables {
		ctx.SetVariable(name, value)
	}

	// Set up decorator lookups for template functions
	h.setupDecoratorLookups(ctx)

	if err := ctx.InitializeVariables(); err != nil {
		fmt.Printf("Warning: failed to initialize variables: %v\n", err)
	}
	return ctx
}

func (h *DecoratorHarness) createPlanContext() execution.PlanContext {
	ctx := execution.NewPlanContext(context.Background(), h.program)

	for name, value := range h.variables {
		ctx.SetVariable(name, value)
	}

	if err := ctx.InitializeVariables(); err != nil {
		fmt.Printf("Warning: failed to initialize variables: %v\n", err)
	}
	return ctx
}

// setupInterpreterDecoratorLookups configures decorator registry access for interpreter context
func (h *DecoratorHarness) setupInterpreterDecoratorLookups(ctx execution.InterpreterContext) {
	if interpreterCtx, ok := ctx.(*execution.InterpreterExecutionContext); ok {
		// Set up action decorator lookup function using the decorator registry
		interpreterCtx.SetActionDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, exists := decorators.GetActionDecorator(name)
			return decorator, exists
		})

		// Set up value decorator lookup function using the decorator registry
		interpreterCtx.SetValueDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetValue(name)
			return decorator, err == nil
		})

		// Set up block decorator lookup function using the decorator registry
		interpreterCtx.SetBlockDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetBlock(name)
			return decorator, err == nil
		})
	}
}

// setupDecoratorLookups configures decorator registry access for template functions
func (h *DecoratorHarness) setupDecoratorLookups(ctx execution.GeneratorContext) {
	if generatorCtx, ok := ctx.(*execution.GeneratorExecutionContext); ok {
		// Set up action decorator lookup function using the decorator registry
		generatorCtx.SetActionDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, exists := decorators.GetActionDecorator(name)
			return decorator, exists
		})

		generatorCtx.SetBlockDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetBlock(name)
			return decorator, err == nil
		})

		generatorCtx.SetPatternDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetPattern(name)
			return decorator, err == nil
		})

		generatorCtx.SetValueDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetValue(name)
			return decorator, err == nil
		})
	}
}

// printTestSummary prints a summary of test results
func (h *DecoratorHarness) printTestSummary(report TestReport) {
	h.t.Logf("=== Decorator Test Summary: %s ===", report.Decorator)
	h.t.Logf("Total Test Cases: %d", report.TotalTestCases)
	h.t.Logf("Core Validations Passed: %d/%d", report.CorePassed, report.TotalTestCases)
	h.t.Logf("Custom Assertions Passed: %d/%d", report.CustomPassed, report.TotalTestCases)
	h.t.Logf("Total Failures: %d", len(report.Failed))
	h.t.Logf("Total Duration: %v", report.TotalDuration)

	if report.AverageCompileTime > 0 {
		h.t.Logf("Average Compile Time: %v", report.AverageCompileTime)
		h.t.Logf("Average Execution Time: %v", report.AverageExecutionTime)
	}

	if len(report.Failed) > 0 {
		h.t.Logf("\n=== Failures ===")
		for _, failure := range report.Failed {
			h.t.Logf("❌ %s::%s - %v", failure.TestCase, failure.AssertionName, failure.Error)
		}
	}
	h.t.Logf("=====================================")
}
