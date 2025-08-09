package execution

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ExecutionResult represents the result of executing shell content
type ExecutionResult struct {
	// Data contains the result:
	// - InterpreterContext: nil (execution happens directly)
	// - GeneratorContext: string (Go code)
	// - PlanContext: plan.PlanElement (plan element)
	Data interface{}

	// Error contains any execution error
	Error error
}

// CommandResult represents the structured output from command execution
// Used by ActionDecorators to enable proper piping and chaining
type CommandResult struct {
	Stdout   string // Standard output as string
	Stderr   string // Standard error as string
	ExitCode int    // Exit code (0 = success)
}

// Success returns true if the command executed successfully (exit code 0)
func (r CommandResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (non-zero exit code)
func (r CommandResult) Failed() bool {
	return r.ExitCode != 0
}

// Error returns an error representation when the command failed
func (r CommandResult) Error() error {
	if r.Success() {
		return nil
	}
	if r.Stderr != "" {
		return fmt.Errorf("exit code %d: %s", r.ExitCode, r.Stderr)
	}
	return fmt.Errorf("exit code %d", r.ExitCode)
}

// ================================================================================================
// MODE-SPECIFIC CONTEXT INTERFACES
// ================================================================================================

// BaseContext provides common functionality shared across all execution modes
type BaseContext interface {
	context.Context

	// Variable management
	GetVariable(name string) (string, bool)
	SetVariable(name, value string)
	GetEnv(name string) (string, bool)
	InitializeVariables() error

	// Program access
	GetProgram() *ast.Program
	GetWorkingDir() string
	IsDebug() bool
	IsDryRun() bool
}

// InterpreterContext provides functionality for direct command execution
type InterpreterContext interface {
	BaseContext

	// Direct execution - commands are run immediately
	ExecuteShell(content *ast.ShellContent) *ExecutionResult
	ExecuteCommandContent(content ast.CommandContent) error
	ExecuteCommand(commandName string) error

	// Decorator lookups (needed for interpreter mode decorator processing)
	GetValueDecoratorLookup() func(name string) (interface{}, bool)
	GetBlockDecoratorLookup() func(name string) (interface{}, bool)

	// Environment variable tracking for runtime consistency
	TrackEnvironmentVariable(key, defaultValue string)
	GetTrackedEnvironmentVariables() map[string]string

	// Typed context management
	Child() InterpreterContext
	WithTimeout(timeout time.Duration) (InterpreterContext, context.CancelFunc)
	WithCancel() (InterpreterContext, context.CancelFunc)
	WithWorkingDir(workingDir string) InterpreterContext
	WithCurrentCommand(commandName string) InterpreterContext
}

// TemplateResult contains a parsed template and its data
type TemplateResult struct {
	Template *template.Template
	Data     interface{}
}

// GeneratorContext provides functionality for Go code generation
type GeneratorContext interface {
	BaseContext

	// Template-based code generation
	GetTemplateFunctions() template.FuncMap
	BuildCommandContent(commands []ast.CommandContent) (*TemplateResult, error)
	ExecuteTemplate(result *TemplateResult) (string, error)

	// Decorator registry access
	GetBlockDecoratorLookup() func(name string) (interface{}, bool)
	GetPatternDecoratorLookup() func(name string) (interface{}, bool)
	GetValueDecoratorLookup() func(name string) (interface{}, bool)

	// Context info for generation
	GetCurrentCommand() string

	// Environment variable tracking for generated code
	TrackEnvironmentVariableReference(key, defaultValue string)
	GetTrackedEnvironmentVariableReferences() map[string]string

	// Simple child context for nested generation
	Child() GeneratorContext
}

// PlanContext provides functionality for execution planning/dry-run
type PlanContext interface {
	BaseContext

	// Plan generation - commands produce plan elements for visualization
	GenerateShellPlan(content *ast.ShellContent) *ExecutionResult
	GenerateCommandPlan(commandName string) (*ExecutionResult, error)

	// Environment variable tracking for planning interpreter behavior
	TrackEnvironmentVariable(key, defaultValue string)
	GetTrackedEnvironmentVariables() map[string]string

	// Typed context management
	Child() PlanContext
	WithTimeout(timeout time.Duration) (PlanContext, context.CancelFunc)
	WithCancel() (PlanContext, context.CancelFunc)
	WithWorkingDir(workingDir string) PlanContext
	WithCurrentCommand(commandName string) PlanContext
}

// ================================================================================================
// TYPE-SAFE CONTEXT FACTORY FUNCTIONS
// ================================================================================================

// NewInterpreterContext creates a new interpreter execution context
func NewInterpreterContext(ctx context.Context, program *ast.Program) InterpreterContext {
	return &InterpreterExecutionContext{
		BaseExecutionContext: newBaseContext(ctx, program),
		trackedEnvVars:       make(map[string]string),
	}
}

// NewGeneratorContext creates a new generator execution context
func NewGeneratorContext(ctx context.Context, program *ast.Program) GeneratorContext {
	return &GeneratorExecutionContext{
		BaseExecutionContext: newBaseContext(ctx, program),
		trackedEnvVars:       make(map[string]string),
	}
}

// NewPlanContext creates a new plan execution context
func NewPlanContext(ctx context.Context, program *ast.Program) PlanContext {
	return &PlanExecutionContext{
		BaseExecutionContext: newBaseContext(ctx, program),
		trackedEnvVars:       make(map[string]string),
	}
}

// newBaseContext creates a new base context with captured environment variables
func newBaseContext(ctx context.Context, program *ast.Program) *BaseExecutionContext {
	// Capture environment variables for deterministic behavior
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	workingDir := "."
	if wd, err := os.Getwd(); err == nil {
		workingDir = wd
	}

	return &BaseExecutionContext{
		Context:    ctx,
		Program:    program,
		Variables:  make(map[string]string),
		env:        envMap,
		WorkingDir: workingDir,
		Debug:      false,
		DryRun:     false,
	}
}

// Helper functions for standardized ExecutionResult creation

// NewSuccessResult creates a successful ExecutionResult with the given data
func NewSuccessResult(data interface{}) *ExecutionResult {
	return &ExecutionResult{
		Data:  data,
		Error: nil,
	}
}

// NewErrorResult creates a failed ExecutionResult with the given error
func NewErrorResult(err error) *ExecutionResult {
	return &ExecutionResult{
		Data:  nil,
		Error: err,
	}
}

// NewFormattedErrorResult creates a failed ExecutionResult with a formatted error message
func NewFormattedErrorResult(format string, args ...interface{}) *ExecutionResult {
	return &ExecutionResult{
		Data:  nil,
		Error: fmt.Errorf(format, args...),
	}
}
