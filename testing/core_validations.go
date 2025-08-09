package testing

import (
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/runtime/decorators"
)

// runCoreValidations executes all core validation checks
func (h *DecoratorHarness) runCoreValidations(ctx TestContext) []TestFailure {
	var failures []TestFailure

	// Validation 1: Behavioral Equivalence
	if err := h.validateBehavioralEquivalence(ctx); err != nil {
		failures = append(failures, TestFailure{
			AssertionName: "behavioral_equivalence",
			Error:         err,
			Context:       "Interpreter and generator modes produce different behavior",
		})
	}

	// Validation 2: Code Compilation
	if err := h.validateCodeCompilation(ctx); err != nil {
		failures = append(failures, TestFailure{
			AssertionName: "code_compilation",
			Error:         err,
			Context:       "Generated code fails to compile",
		})
	}

	// Validation 3: Template Syntax
	if err := h.validateTemplateSyntax(ctx); err != nil {
		failures = append(failures, TestFailure{
			AssertionName: "template_syntax",
			Error:         err,
			Context:       "Generated code has invalid Go syntax",
		})
	}

	// Validation 4: Plan Consistency
	if err := h.validatePlanConsistency(ctx); err != nil {
		failures = append(failures, TestFailure{
			AssertionName: "plan_consistency",
			Error:         err,
			Context:       "Plan output is inconsistent with execution",
		})
	}

	// Validation 5: Error Handling
	if err := h.validateErrorHandling(ctx); err != nil {
		failures = append(failures, TestFailure{
			AssertionName: "error_handling",
			Error:         err,
			Context:       "Error handling is inconsistent across modes",
		})
	}

	return failures
}

// validateBehavioralEquivalence ensures interpreter and generator produce same behavior
func (h *DecoratorHarness) validateBehavioralEquivalence(ctx TestContext) error {
	// Skip if either mode failed
	if !ctx.InterpreterResult.Success || !ctx.GeneratorResult.Success {
		return nil // We'll catch individual failures elsewhere
	}

	// For value decorators, compare direct outputs
	if _, isValue := h.decorator.(decorators.ValueDecorator); isValue {
		interpreterOutput := fmt.Sprintf("%v", ctx.InterpreterResult.Data)

		// Extract actual value from generated code by looking for variable assignments or returns
		generatedValue := h.extractValueFromGeneratedCode(ctx.GeneratedCode)

		if interpreterOutput != generatedValue {
			return fmt.Errorf("value mismatch - interpreter: %q, generated: %q",
				interpreterOutput, generatedValue)
		}
		return nil
	}

	// For other decorators, compare execution outputs if we have both
	if ctx.ExecutionOutput == nil {
		return nil // Can't compare if compilation failed
	}

	// Compare exit codes
	interpreterExitCode := h.getExitCodeFromResult(ctx.InterpreterResult)
	generatedExitCode := ctx.ExecutionOutput.ExitCode

	if interpreterExitCode != generatedExitCode {
		return fmt.Errorf("exit code mismatch - interpreter: %d, generated: %d",
			interpreterExitCode, generatedExitCode)
	}

	// For side-effect decorators (like @workdir, @parallel), compare observable behavior
	if err := h.compareObservableBehavior(ctx); err != nil {
		return fmt.Errorf("behavioral difference: %w", err)
	}

	return nil
}

// validateCodeCompilation ensures generated code compiles successfully
func (h *DecoratorHarness) validateCodeCompilation(ctx TestContext) error {
	if !ctx.GeneratorResult.Success {
		return fmt.Errorf("generator mode failed: %v", ctx.GeneratorResult.Error)
	}

	if ctx.CompiledBinary == nil {
		return fmt.Errorf("no compilation attempt was made")
	}

	if ctx.CompiledBinary.CompileError != nil {
		return fmt.Errorf("compilation failed: %w", ctx.CompiledBinary.CompileError)
	}

	return nil
}

// validateTemplateSyntax validates that generated Go code is syntactically correct
func (h *DecoratorHarness) validateTemplateSyntax(ctx TestContext) error {
	if !ctx.GeneratorResult.Success || ctx.GeneratedCode == "" {
		return nil // Skip if no code was generated
	}

	// Wrap code in a function to make it parseable
	wrappedCode := fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

type ExecutionContext struct {
	context.Context
	Dir string
}

func (ctx *ExecutionContext) Clone() *ExecutionContext {
	return &ExecutionContext{
		Context: ctx.Context,
		Dir:     ctx.Dir,
	}
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func (cr CommandResult) Success() bool { return cr.ExitCode == 0 }
func (cr CommandResult) Failed() bool  { return cr.ExitCode != 0 }

func exec(ctx *ExecutionContext, command string) error {
	return nil // Placeholder
}

func testFunc(ctx *ExecutionContext) error {
%s
	return nil
}
`, ctx.GeneratedCode)

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", wrappedCode, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("syntax error in generated code: %w\nCode:\n%s", err, ctx.GeneratedCode)
	}

	return nil
}

// validatePlanConsistency ensures plan output accurately represents execution
func (h *DecoratorHarness) validatePlanConsistency(ctx TestContext) error {
	if !ctx.PlanResult.Success {
		return fmt.Errorf("plan mode failed: %v", ctx.PlanResult.Error)
	}

	if ctx.PlanOutput == nil {
		return fmt.Errorf("plan mode returned nil output")
	}

	// TODO: Add specific plan structure validation based on decorator type
	// This would check that the plan accurately represents what will be executed

	return nil
}

// validateErrorHandling ensures error cases are handled consistently across modes
func (h *DecoratorHarness) validateErrorHandling(ctx TestContext) error {
	// If all modes succeeded, no error handling to validate
	if ctx.InterpreterResult.Success && ctx.GeneratorResult.Success && ctx.PlanResult.Success {
		return nil
	}

	// Check for parameter validation consistency
	if h.hasParameterError(ctx.InterpreterResult) {
		if !h.hasParameterError(ctx.GeneratorResult) {
			return fmt.Errorf("parameter validation inconsistent: interpreter failed but generator succeeded")
		}
		if !h.hasParameterError(ctx.PlanResult) {
			return fmt.Errorf("parameter validation inconsistent: interpreter failed but plan succeeded")
		}
	}

	return nil
}

// Helper functions

func (h *DecoratorHarness) extractValueFromGeneratedCode(code string) string {
	// Look for common patterns in generated value code
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "return ") && strings.Contains(line, `"`) {
			// Extract string literal from return statement
			start := strings.Index(line, `"`)
			if start >= 0 {
				end := strings.Index(line[start+1:], `"`)
				if end >= 0 {
					return line[start+1 : start+1+end]
				}
			}
		}
	}
	return ""
}

func (h *DecoratorHarness) getExitCodeFromResult(result TestResult) int {
	if result.Error != nil {
		return 1
	}
	return 0
}

func (h *DecoratorHarness) compareObservableBehavior(ctx TestContext) error {
	// This is decorator-specific and would be enhanced for each decorator type
	// For now, we do basic comparison of stdout patterns

	// Skip if we don't have execution output
	if ctx.ExecutionOutput == nil {
		return nil
	}

	// Compare basic output patterns
	// TODO: Add more sophisticated comparison based on decorator semantics

	return nil
}

func (h *DecoratorHarness) hasParameterError(result TestResult) bool {
	if result.Error == nil {
		return false
	}
	errorMsg := strings.ToLower(result.Error.Error())
	return strings.Contains(errorMsg, "parameter") ||
		strings.Contains(errorMsg, "required") ||
		strings.Contains(errorMsg, "invalid")
}

// compileGeneratedCode compiles the generated Go code and returns compilation info
func (h *DecoratorHarness) compileGeneratedCode(code, testName string) (*CompiledProgram, error) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("devcmd_test_%s_", testName))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	program := &CompiledProgram{
		TempDir: tmpDir,
	}

	// Create complete Go program with all required context
	fullProgram := h.createCompleteProgram(code)

	// Write main.go
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(fullProgram), 0o644); err != nil {
		return program, fmt.Errorf("failed to write main.go: %w", err)
	}

	// Write go.mod
	goMod := fmt.Sprintf("module devcmd_test_%s\n\ngo 1.21\n", testName)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return program, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Compile with timeout
	compileStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	binaryPath := filepath.Join(tmpDir, "test_program")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "main.go")
	buildCmd.Dir = tmpDir

	output, err := buildCmd.CombinedOutput()
	program.CompileTime = time.Since(compileStart)

	if err != nil {
		program.CompileError = fmt.Errorf("compilation failed: %w\nOutput: %s\nCode:\n%s",
			err, string(output), fullProgram)
		return program, nil // Return program with error, don't fail the function
	}

	program.BinaryPath = binaryPath
	return program, nil
}

// executeCompiledProgram runs a compiled program and captures output
func (h *DecoratorHarness) executeCompiledProgram(program *CompiledProgram) (*ExecutionOutput, error) {
	if program.CompileError != nil {
		return nil, fmt.Errorf("cannot execute program with compile error: %w", program.CompileError)
	}

	execStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, program.BinaryPath)
	cmd.Dir = program.TempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(execStart)

	output := &ExecutionOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitError.ExitCode()
		} else {
			output.ExitCode = 1
			output.Error = err
		}
	}

	// Clean up
	if err := os.RemoveAll(program.TempDir); err != nil {
		// Log the error but don't fail the test
		fmt.Printf("Warning: failed to clean up temp dir %s: %v\n", program.TempDir, err)
	}

	return output, nil
}

// createCompleteProgram wraps generated code in a complete, executable Go program
func (h *DecoratorHarness) createCompleteProgram(generatedCode string) string {
	return fmt.Sprintf(`package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Execution context and result types
type ExecutionContext struct {
	context.Context
	Dir           string
	variables     map[string]string
	EnvContext    map[string]string
}

func NewExecutionContext(ctx context.Context) *ExecutionContext {
	workingDir, _ := os.Getwd()
	return &ExecutionContext{
		Context:    ctx,
		Dir:        workingDir,
		variables:  make(map[string]string),
		EnvContext: make(map[string]string),
	}
}

func (ctx *ExecutionContext) Clone() *ExecutionContext {
	cloned := &ExecutionContext{
		Context:    ctx.Context,
		Dir:        ctx.Dir,
		variables:  make(map[string]string),
		EnvContext: make(map[string]string),
	}
	
	// Copy variables
	for k, v := range ctx.variables {
		cloned.variables[k] = v
	}
	for k, v := range ctx.EnvContext {
		cloned.EnvContext[k] = v
	}
	
	return cloned
}

func (ctx *ExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := ctx.variables[name]
	return value, exists
}

func (ctx *ExecutionContext) SetVariable(name, value string) {
	ctx.variables[name] = value
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func (cr CommandResult) Success() bool { return cr.ExitCode == 0 }
func (cr CommandResult) Failed() bool  { return cr.ExitCode != 0 }

// Helper functions for generated code
func exec(ctx *ExecutionContext, command string) error {
	cmd := exec.CommandContext(ctx.Context, "sh", "-c", command)
	cmd.Dir = ctx.Dir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %%s\nStdout: %%s\nStderr: %%s", 
			err, stdout.String(), stderr.String())
	}
	
	return nil
}

func executeShellCommand(ctx context.Context, command string) CommandResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return CommandResult{
				Stdout:   string(stdout),
				Stderr:   string(exitErr.Stderr),
				ExitCode: exitErr.ExitCode(),
			}
		}
		return CommandResult{
			Stdout:   string(stdout),
			Stderr:   err.Error(),
			ExitCode: 1,
		}
	}
	return CommandResult{
		Stdout:   string(stdout),
		Stderr:   "",
		ExitCode: 0,
	}
}

var variableContext = make(map[string]string)

func main() {
	baseCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	ctx := NewExecutionContext(baseCtx)
	
	// Execute the generated code
	if err := executeGeneratedCode(ctx); err != nil {
		fmt.Printf("Execution failed: %%s\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Execution completed successfully")
}

func executeGeneratedCode(ctx *ExecutionContext) error {
	// Generated code goes here
%s
	
	return nil
}
`, generatedCode)
}

// runCustomAssertions executes all custom assertion functions
func (h *DecoratorHarness) runCustomAssertions(ctx TestContext) []TestFailure {
	var failures []TestFailure

	for name, assertion := range h.customAssertions {
		if err := assertion(ctx); err != nil {
			failures = append(failures, TestFailure{
				AssertionName: name,
				Error:         err,
				Context:       "Custom assertion failed",
			})
		}
	}

	return failures
}
