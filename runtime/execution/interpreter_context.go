package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// InterpreterExecutionContext implements InterpreterContext for direct command execution
type InterpreterExecutionContext struct {
	*BaseExecutionContext

	// Environment variable tracking for interpreter execution
	// Maps env var name -> default value (empty string if no default)
	// These are captured at command start to prevent changes during execution
	trackedEnvVars map[string]string
}

// ================================================================================================
// INTERPRETER-SPECIFIC FUNCTIONALITY
// ================================================================================================

// ExecuteShell executes shell content directly
func (c *InterpreterExecutionContext) ExecuteShell(content *ast.ShellContent) *ExecutionResult {
	// Compose the command string from parts
	cmdStr, err := c.composeShellCommand(content)
	if err != nil {
		return &ExecutionResult{
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
		Data:  nil,
		Error: err,
	}
}

// ExecuteCommandContent executes any command content type in interpreter mode
func (c *InterpreterExecutionContext) ExecuteCommandContent(content ast.CommandContent) error {
	switch cmd := content.(type) {
	case *ast.ShellContent:
		result := c.ExecuteShell(cmd)
		return result.Error

	case *ast.BlockDecorator:
		// Handle block decorators by looking them up in the registry
		return c.executeBlockDecorator(cmd)

	case *ast.PatternDecorator:
		// Handle pattern decorators like @when
		return c.executePatternDecorator(cmd)

	case *ast.PatternContent:
		// Handle pattern content (branches within @when)
		return fmt.Errorf("pattern content cannot be executed directly - should be part of pattern decorator")

	case *ast.ActionDecorator:
		// Action decorators as standalone commands (like in @parallel { @cmd(...) })
		return c.executeActionDecorator(cmd)

	default:
		return fmt.Errorf("unsupported command content type: %T", content)
	}
}

// executeBlockDecorator handles block decorator execution in interpreter mode
func (c *InterpreterExecutionContext) executeBlockDecorator(blockDec *ast.BlockDecorator) error {
	// Import the decorators package to access the registry
	// Note: This creates a circular dependency, so we'll use the lookup function approach
	blockDecoratorLookup := c.GetBlockDecoratorLookup()
	if blockDecoratorLookup == nil {
		return fmt.Errorf("block decorator lookup not available (engine not properly initialized)")
	}

	decoratorInterface, exists := blockDecoratorLookup(blockDec.Name)
	if !exists {
		return fmt.Errorf("block decorator @%s not found", blockDec.Name)
	}

	// Cast to the expected interface type for interpreter mode
	blockDecorator, ok := decoratorInterface.(interface {
		ExecuteInterpreter(ctx InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *ExecutionResult
	})
	if !ok {
		return fmt.Errorf("block decorator @%s does not implement expected ExecuteInterpreter method", blockDec.Name)
	}

	// Execute the block decorator
	result := blockDecorator.ExecuteInterpreter(c, blockDec.Args, blockDec.Content)
	return result.Error
}

// executePatternDecorator handles pattern decorator execution in interpreter mode
func (c *InterpreterExecutionContext) executePatternDecorator(patternDec *ast.PatternDecorator) error {
	// Get the pattern decorator from the registry
	blockDecoratorLookup := c.GetBlockDecoratorLookup()
	if blockDecoratorLookup == nil {
		return fmt.Errorf("pattern decorator lookup not available (engine not properly initialized)")
	}

	decoratorInterface, exists := blockDecoratorLookup(patternDec.Name)
	if !exists {
		return fmt.Errorf("pattern decorator @%s not found", patternDec.Name)
	}

	// Cast to the expected interface type for interpreter mode
	patternDecorator, ok := decoratorInterface.(interface {
		ExecuteInterpreter(ctx InterpreterContext, params []ast.NamedParameter, branches []ast.PatternBranch) *ExecutionResult
	})
	if !ok {
		return fmt.Errorf("pattern decorator @%s does not implement expected ExecuteInterpreter method", patternDec.Name)
	}

	// Execute the pattern decorator
	result := patternDecorator.ExecuteInterpreter(c, patternDec.Args, patternDec.Patterns)
	return result.Error
}

// executeActionDecorator handles action decorator execution in interpreter mode
func (c *InterpreterExecutionContext) executeActionDecorator(actionDec *ast.ActionDecorator) error {
	// Action decorators as standalone commands need special handling
	// They should be processed as part of shell content, not as standalone commands
	return fmt.Errorf("action decorator @%s cannot be executed as standalone command - should be part of shell content", actionDec.Name)
}

// ExecuteCommand executes a full command by name (used by decorators like @cmd)
func (c *InterpreterExecutionContext) ExecuteCommand(commandName string) error {
	// This method is no longer needed with the new architecture
	// Commands should be executed directly through the engine patterns
	return fmt.Errorf("ExecuteCommand is deprecated - use engine command execution patterns")
}

// ================================================================================================
// CONTEXT MANAGEMENT WITH TYPE SAFETY
// ================================================================================================

// Child creates a child interpreter context that inherits from the parent but can be modified independently
func (c *InterpreterExecutionContext) Child() InterpreterContext {
	// Increment child counter to ensure unique variable naming across parallel contexts
	c.childCounter++
	childID := c.childCounter

	childBase := &BaseExecutionContext{
		Context:   c.Context,
		Program:   c.Program,
		Variables: make(map[string]string),
		env:       c.env, // Share the same immutable environment reference

		// Copy execution state
		WorkingDir:     c.WorkingDir,
		Debug:          c.Debug,
		DryRun:         c.DryRun,
		currentCommand: c.currentCommand,

		// Copy decorator lookups from parent (critical for nested decorator execution)
		valueDecoratorLookup:  c.BaseExecutionContext.valueDecoratorLookup,
		actionDecoratorLookup: c.BaseExecutionContext.actionDecoratorLookup,
		blockDecoratorLookup:  c.BaseExecutionContext.blockDecoratorLookup,

		// Initialize unique counter space for this child to avoid variable name conflicts
		// Each child gets a unique counter space based on parent's counter and child ID
		shellCounter: c.shellCounter + (childID * 1000), // Give each child 1000 numbers of space
		childCounter: 0,                                 // Reset child counter for this context's children
	}

	// Copy variables (child gets its own copy)
	for name, value := range c.Variables {
		childBase.Variables[name] = value
	}

	return &InterpreterExecutionContext{
		BaseExecutionContext: childBase,
		trackedEnvVars:       make(map[string]string),
	}
}

// WithTimeout creates a new interpreter context with timeout
func (c *InterpreterExecutionContext) WithTimeout(timeout time.Duration) (InterpreterContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}, cancel
}

// WithCancel creates a new interpreter context with cancellation
func (c *InterpreterExecutionContext) WithCancel() (InterpreterContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}, cancel
}

// WithWorkingDir creates a new interpreter context with the specified working directory
func (c *InterpreterExecutionContext) WithWorkingDir(workingDir string) InterpreterContext {
	newBase := *c.BaseExecutionContext
	newBase.WorkingDir = workingDir
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}
}

// WithCurrentCommand creates a new interpreter context with the specified current command name
func (c *InterpreterExecutionContext) WithCurrentCommand(commandName string) InterpreterContext {
	newBase := *c.BaseExecutionContext
	newBase.currentCommand = commandName
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}
}

// ================================================================================================
// SHELL COMMAND COMPOSITION
// ================================================================================================

// composeShellCommand composes the shell command string from AST parts
func (c *InterpreterExecutionContext) composeShellCommand(content *ast.ShellContent) (string, error) {
	var parts []string

	for _, part := range content.Parts {
		result, err := c.processShellPart(part)
		if err != nil {
			return "", err
		}

		if value, ok := result.(string); ok {
			parts = append(parts, value)
		} else {
			return "", fmt.Errorf("shell part returned non-string result: %T", result)
		}
	}

	return strings.Join(parts, ""), nil
}

// processShellPart processes any shell part (text, value decorator, action decorator) for interpreter mode
func (c *InterpreterExecutionContext) processShellPart(part ast.ShellPart) (interface{}, error) {
	switch p := part.(type) {
	case *ast.TextPart:
		return p.Text, nil

	case *ast.ValueDecorator:
		return c.processValueDecorator(p)

	case *ast.ActionDecorator:
		return c.processActionDecorator(p)

	default:
		return nil, fmt.Errorf("unsupported shell part type: %T", part)
	}
}

// processValueDecorator handles value decorators in interpreter mode
func (c *InterpreterExecutionContext) processValueDecorator(decorator *ast.ValueDecorator) (interface{}, error) {
	// Use the value decorator lookup to get the decorator from the registry
	lookupFunc := c.GetValueDecoratorLookup()
	if lookupFunc == nil {
		return nil, fmt.Errorf("value decorator lookup not available (engine not properly initialized)")
	}

	decoratorInterface, exists := lookupFunc(decorator.Name)
	if !exists {
		return nil, fmt.Errorf("value decorator @%s not found", decorator.Name)
	}

	// Cast to the expected interface type for interpreter mode
	valueDecorator, ok := decoratorInterface.(interface {
		ExpandInterpreter(ctx InterpreterContext, params []ast.NamedParameter) *ExecutionResult
	})
	if !ok {
		return nil, fmt.Errorf("value decorator @%s does not implement expected ExpandInterpreter method", decorator.Name)
	}

	// Call ExpandInterpreter to get the expanded value
	result := valueDecorator.ExpandInterpreter(c, decorator.Args)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to expand value decorator @%s: %w", decorator.Name, result.Error)
	}

	// Return the expanded value
	return result.Data, nil
}

// GetValueDecoratorLookup returns the value decorator lookup function for interpreter mode
func (c *InterpreterExecutionContext) GetValueDecoratorLookup() func(name string) (interface{}, bool) {
	// Value decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.valueDecoratorLookup
}

// GetActionDecoratorLookup returns the action decorator lookup function for interpreter mode
func (c *InterpreterExecutionContext) GetActionDecoratorLookup() func(name string) (interface{}, bool) {
	// Action decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.BaseExecutionContext.actionDecoratorLookup
}

// GetBlockDecoratorLookup returns the block decorator lookup function for interpreter mode
func (c *InterpreterExecutionContext) GetBlockDecoratorLookup() func(name string) (interface{}, bool) {
	// Block decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.BaseExecutionContext.blockDecoratorLookup
}

// SetValueDecoratorLookup sets the value decorator lookup function (called by engine during setup)
func (c *InterpreterExecutionContext) SetValueDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.BaseExecutionContext.SetValueDecoratorLookup(lookup)
}

// SetActionDecoratorLookup sets the action decorator lookup function (called by engine during setup)
func (c *InterpreterExecutionContext) SetActionDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.BaseExecutionContext.SetActionDecoratorLookup(lookup)
}

// SetBlockDecoratorLookup sets the block decorator lookup function (called by engine during setup)
func (c *InterpreterExecutionContext) SetBlockDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.BaseExecutionContext.SetBlockDecoratorLookup(lookup)
}

// TrackEnvironmentVariable tracks an environment variable for consistent access during execution
func (c *InterpreterExecutionContext) TrackEnvironmentVariable(key, defaultValue string) {
	if c.trackedEnvVars == nil {
		c.trackedEnvVars = make(map[string]string)
	}
	c.trackedEnvVars[key] = defaultValue
}

// GetTrackedEnvironmentVariables returns all tracked environment variables
func (c *InterpreterExecutionContext) GetTrackedEnvironmentVariables() map[string]string {
	if c.trackedEnvVars == nil {
		return make(map[string]string)
	}
	// Return a copy to prevent external modifications
	result := make(map[string]string)
	for k, v := range c.trackedEnvVars {
		result[k] = v
	}
	return result
}

// processActionDecorator handles action decorators in interpreter mode
func (c *InterpreterExecutionContext) processActionDecorator(decorator *ast.ActionDecorator) (interface{}, error) {
	// Use the action decorator lookup to get the decorator from the registry
	lookupFunc := c.GetActionDecoratorLookup()
	if lookupFunc == nil {
		return nil, fmt.Errorf("action decorator lookup not available (engine not properly initialized)")
	}

	decoratorInterface, exists := lookupFunc(decorator.Name)
	if !exists {
		return nil, fmt.Errorf("action decorator @%s not found", decorator.Name)
	}

	// Cast to the expected interface type for interpreter mode
	actionDecorator, ok := decoratorInterface.(interface {
		ExpandInterpreter(ctx InterpreterContext, params []ast.NamedParameter) *ExecutionResult
	})
	if !ok {
		return nil, fmt.Errorf("action decorator @%s does not implement expected ExpandInterpreter method", decorator.Name)
	}

	// Call ExpandInterpreter to get the expanded result for shell chaining
	result := actionDecorator.ExpandInterpreter(c, decorator.Args)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to expand action decorator @%s: %w", decorator.Name, result.Error)
	}

	// Return the expanded value for shell composition
	return result.Data, nil
}
