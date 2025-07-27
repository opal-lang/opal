package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// ExecutionContext provides execution context for decorators and implements context.Context
type ExecutionContext struct {
	context.Context

	// Core data
	Program   *ast.Program
	Variables map[string]string // Resolved variable values
	Env       map[string]string // Environment variables

	// Execution state
	WorkingDir string
	Debug      bool
	DryRun     bool

	// Execution mode for the unified pattern
	mode ExecutionMode

	// Template functions for code generation (populated by engine)
	templateFunctions template.FuncMap

	// Command content executor for nested command execution (populated by engine)
	contentExecutor func(ast.CommandContent) error

	// Function decorator lookup (populated by engine to avoid circular imports)
	functionDecoratorLookup func(name string) (FunctionDecorator, bool)
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(parent context.Context, program *ast.Program) *ExecutionContext {
	if parent == nil {
		parent = context.Background()
	}

	return &ExecutionContext{
		Context:           parent,
		Program:           program,
		Variables:         make(map[string]string),
		Env:               make(map[string]string),
		Debug:             false,
		DryRun:            false,
		mode:              InterpreterMode, // Default mode
		templateFunctions: make(template.FuncMap),
	}
}

// WithTimeout creates a new context with timeout
func (c *ExecutionContext) WithTimeout(timeout time.Duration) (*ExecutionContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newCtx := *c
	newCtx.Context = ctx
	return &newCtx, cancel
}

// WithCancel creates a new context with cancellation
func (c *ExecutionContext) WithCancel() (*ExecutionContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newCtx := *c
	newCtx.Context = ctx
	return &newCtx, cancel
}

// WithMode creates a new context with the specified execution mode
func (c *ExecutionContext) WithMode(mode ExecutionMode) *ExecutionContext {
	newCtx := *c
	newCtx.mode = mode
	return &newCtx
}

// Mode returns the current execution mode
func (c *ExecutionContext) Mode() ExecutionMode {
	return c.mode
}

// ExecuteShell executes shell content in the current mode
func (c *ExecutionContext) ExecuteShell(content *ast.ShellContent) *ExecutionResult {
	switch c.mode {
	case InterpreterMode:
		return c.executeShellInterpreter(content)
	case GeneratorMode:
		return c.executeShellGenerator(content)
	case PlanMode:
		return c.executeShellPlan(content)
	default:
		return &ExecutionResult{
			Mode:  c.mode,
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", c.mode),
		}
	}
}

// executeShellInterpreter executes shell content directly
func (c *ExecutionContext) executeShellInterpreter(content *ast.ShellContent) *ExecutionResult {
	// Compose the command string from parts
	cmdStr, err := c.composeShellCommand(content)
	if err != nil {
		return &ExecutionResult{
			Mode:  InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("failed to compose shell command: %w", err),
		}
	}

	// Execute the command
	cmd := exec.CommandContext(c.Context, "sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if c.WorkingDir != "" {
		cmd.Dir = c.WorkingDir
	}

	err = cmd.Run()
	return &ExecutionResult{
		Mode:  InterpreterMode,
		Data:  nil,
		Error: err,
	}
}

// executeShellGenerator generates Go code for shell execution
func (c *ExecutionContext) executeShellGenerator(content *ast.ShellContent) *ExecutionResult {
	// Build Go expression parts for the command
	var goExprParts []string

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			// Plain text - add as quoted string
			goExprParts = append(goExprParts, strconv.Quote(p.Text))

		case *ast.FunctionDecorator:
			// Check if function decorator lookup is available
			if c.functionDecoratorLookup == nil {
				return &ExecutionResult{
					Mode:  GeneratorMode,
					Data:  "",
					Error: fmt.Errorf("function decorator lookup not available (engine not properly initialized)"),
				}
			}

			// Look up the function decorator in the registry
			funcDecorator, exists := c.functionDecoratorLookup(p.Name)
			if !exists {
				return &ExecutionResult{
					Mode:  GeneratorMode,
					Data:  "",
					Error: fmt.Errorf("function decorator @%s not found in registry", p.Name),
				}
			}

			// Execute the decorator in generator mode to get Go code
			generatorCtx := c.WithMode(GeneratorMode)
			result := funcDecorator.Expand(generatorCtx, p.Args)
			if result.Error != nil {
				return &ExecutionResult{
					Mode:  GeneratorMode,
					Data:  "",
					Error: fmt.Errorf("@%s decorator code generation failed: %w", p.Name, result.Error),
				}
			}

			// Extract the generated Go code
			if code, ok := result.Data.(string); ok {
				goExprParts = append(goExprParts, code)
			} else {
				return &ExecutionResult{
					Mode:  GeneratorMode,
					Data:  "",
					Error: fmt.Errorf("@%s decorator returned non-string code: %T", p.Name, result.Data),
				}
			}

		default:
			return &ExecutionResult{
				Mode:  GeneratorMode,
				Data:  "",
				Error: fmt.Errorf("unsupported shell part type: %T", part),
			}
		}
	}

	// Build the Go expression for the command
	var cmdExpr string
	if len(goExprParts) == 1 {
		cmdExpr = goExprParts[0]
	} else {
		cmdExpr = strings.Join(goExprParts, " + ")
	}

	// Generate the Go code using the dedicated template for CLI generation
	tmpl, err := template.New("shellCommandCLI").Parse(shellCommandCLITemplate)
	if err != nil {
		return &ExecutionResult{
			Mode:  GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse shell command CLI template: %w", err),
		}
	}

	templateData := struct {
		CommandExpression string
	}{
		CommandExpression: cmdExpr,
	}

	var goCode strings.Builder
	if err := tmpl.Execute(&goCode, templateData); err != nil {
		return &ExecutionResult{
			Mode:  GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute shell command CLI template: %w", err),
		}
	}

	return &ExecutionResult{
		Mode:  GeneratorMode,
		Data:  goCode.String(),
		Error: nil,
	}
}

// executeShellPlan creates a plan element for shell execution
func (c *ExecutionContext) executeShellPlan(content *ast.ShellContent) *ExecutionResult {
	// Compose the command string for display using interpreter-style resolution
	// This ensures variables are resolved to their actual values for the plan display
	interpreterCtx := c.WithMode(InterpreterMode)
	cmdStr, err := interpreterCtx.composeShellCommand(content)
	if err != nil {
		return &ExecutionResult{
			Mode:  PlanMode,
			Data:  nil,
			Error: fmt.Errorf("failed to compose shell command for plan: %w", err),
		}
	}

	// For now, return a simple plan representation
	// TODO: Replace with proper plan.PlanElement when we move plan package
	planData := map[string]interface{}{
		"type":        "shell",
		"command":     cmdStr,
		"description": "Execute shell command: " + cmdStr,
	}

	return &ExecutionResult{
		Mode:  PlanMode,
		Data:  planData,
		Error: nil,
	}
}

// composeShellCommand composes the shell command string from AST parts
func (c *ExecutionContext) composeShellCommand(content *ast.ShellContent) (string, error) {
	var parts []string

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			parts = append(parts, p.Text)

		case *ast.FunctionDecorator:
			expanded, err := c.processFunctionDecorator(p)
			if err != nil {
				return "", err
			}
			parts = append(parts, expanded)

		default:
			return "", fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	return strings.Join(parts, ""), nil
}

// processFunctionDecorator processes function decorators using the unified Execute pattern
func (c *ExecutionContext) processFunctionDecorator(decorator *ast.FunctionDecorator) (string, error) {
	// Check if function decorator lookup is available
	if c.functionDecoratorLookup == nil {
		return "", fmt.Errorf("function decorator lookup not available (engine not properly initialized)")
	}

	// Look up the function decorator in the registry
	funcDecorator, exists := c.functionDecoratorLookup(decorator.Name)
	if !exists {
		return "", fmt.Errorf("function decorator @%s not found in registry", decorator.Name)
	}

	// Execute the decorator using the unified Execute pattern
	result := funcDecorator.Expand(c, decorator.Args)
	if result.Error != nil {
		return "", fmt.Errorf("@%s decorator execution failed: %w", decorator.Name, result.Error)
	}

	// Extract the string result for substitution
	if value, ok := result.Data.(string); ok {
		return value, nil
	}

	return "", fmt.Errorf("@%s decorator returned non-string result: %T", decorator.Name, result.Data)
}

// GetVariable retrieves a variable value
func (c *ExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := c.Variables[name]
	return value, exists
}

// SetVariable sets a variable value
func (c *ExecutionContext) SetVariable(name, value string) {
	c.Variables[name] = value
}

// GetEnv retrieves an environment variable
func (c *ExecutionContext) GetEnv(name string) (string, bool) {
	value, exists := c.Env[name]
	return value, exists
}

// SetEnv sets an environment variable
func (c *ExecutionContext) SetEnv(name, value string) {
	c.Env[name] = value
}

// InitializeVariables processes and sets all variables from the program
func (c *ExecutionContext) InitializeVariables() error {
	if c.Program == nil {
		return nil
	}

	// Process individual variables
	for _, variable := range c.Program.Variables {
		value, err := c.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		c.SetVariable(variable.Name, value)
	}

	// Process variable groups
	for _, group := range c.Program.VarGroups {
		for _, variable := range group.Variables {
			value, err := c.resolveVariableValue(variable.Value)
			if err != nil {
				return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
			}
			c.SetVariable(variable.Name, value)
		}
	}

	return nil
}

// GetTemplateFunctions returns the template function map for code generation
func (c *ExecutionContext) GetTemplateFunctions() template.FuncMap {
	return c.templateFunctions
}

// SetTemplateFunctions sets the template function map (used by engine)
func (c *ExecutionContext) SetTemplateFunctions(funcs template.FuncMap) {
	c.templateFunctions = funcs
}

// ExecuteCommandContent executes command content using the engine's executor (used by decorators)
func (c *ExecutionContext) ExecuteCommandContent(content ast.CommandContent) error {
	if c.contentExecutor == nil {
		return fmt.Errorf("command content executor not available (engine not properly initialized)")
	}
	return c.contentExecutor(content)
}

// SetContentExecutor sets the command content executor (used by engine)
func (c *ExecutionContext) SetContentExecutor(executor func(ast.CommandContent) error) {
	c.contentExecutor = executor
}

// SetFunctionDecoratorLookup sets the function decorator lookup (used by engine)
func (c *ExecutionContext) SetFunctionDecoratorLookup(lookup func(name string) (FunctionDecorator, bool)) {
	c.functionDecoratorLookup = lookup
}

// Shell command execution template for use within error-returning functions
const shellCommandTemplate = `			cmdStr := {{.CommandExpression}}
			execCmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin
			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("command failed: %w", err)
			}
			return nil`

// Template for shell command generation in main CLI (calls os.Exit on failure)
const shellCommandCLITemplate = `		func() {
			cmdStr := {{.CommandExpression}}
			execCmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin
			if err := execCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
				os.Exit(1)
			}
		}()`

// GenerateShellCodeForTemplate generates clean Go code for shell command execution
// This method is designed to be used by decorator templates to generate proper Go code
// that returns an error instead of calling os.Exit
func (c *ExecutionContext) GenerateShellCodeForTemplate(content *ast.ShellContent) (string, error) {
	// Build Go expression parts for the command (similar to executeShellGenerator)
	var goExprParts []string

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			// Plain text - add as quoted string
			goExprParts = append(goExprParts, strconv.Quote(p.Text))

		case *ast.FunctionDecorator:
			// Check if function decorator lookup is available
			if c.functionDecoratorLookup == nil {
				return "", fmt.Errorf("function decorator lookup not available (engine not properly initialized)")
			}

			// Look up the function decorator in the registry
			funcDecorator, exists := c.functionDecoratorLookup(p.Name)
			if !exists {
				return "", fmt.Errorf("function decorator @%s not found in registry", p.Name)
			}

			// Execute the decorator in generator mode to get Go code
			generatorCtx := c.WithMode(GeneratorMode)
			result := funcDecorator.Expand(generatorCtx, p.Args)
			if result.Error != nil {
				return "", fmt.Errorf("@%s decorator code generation failed: %w", p.Name, result.Error)
			}

			// Extract the generated Go code
			if code, ok := result.Data.(string); ok {
				goExprParts = append(goExprParts, code)
			} else {
				return "", fmt.Errorf("@%s decorator returned non-string code: %T", p.Name, result.Data)
			}

		default:
			return "", fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	// Build the Go expression for the command
	var cmdExpr string
	if len(goExprParts) == 1 {
		cmdExpr = goExprParts[0]
	} else {
		cmdExpr = strings.Join(goExprParts, " + ")
	}

	// Use template to generate the code
	tmpl, err := template.New("shellCommand").Parse(shellCommandTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse shell command template: %w", err)
	}

	templateData := struct {
		CommandExpression string
	}{
		CommandExpression: cmdExpr,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute shell command template: %w", err)
	}

	return result.String(), nil
}

// resolveVariableValue converts an AST expression to its string value
func (c *ExecutionContext) resolveVariableValue(expr ast.Expression) (string, error) {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value, nil
	case *ast.NumberLiteral:
		return v.Value, nil
	case *ast.BooleanLiteral:
		if v.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.DurationLiteral:
		return v.Value, nil
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}
