package execution

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// PlanExecutionContext implements PlanContext for execution planning/dry-run
type PlanExecutionContext struct {
	*BaseExecutionContext

	// Environment variable tracking for planning interpreter behavior
	trackedEnvVars map[string]string
}

// ================================================================================================
// PLAN-SPECIFIC FUNCTIONALITY
// ================================================================================================

// GenerateShellPlan creates a plan element for shell execution
func (c *PlanExecutionContext) GenerateShellPlan(content *ast.ShellContent) *ExecutionResult {
	// CRITICAL FIX: Don't use InterpreterMode for plan generation as it executes ActionDecorators
	// Instead, create a plan-safe command string that doesn't execute anything
	cmdStr, err := c.composeShellCommandForPlan(content)
	if err != nil {
		return &ExecutionResult{
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
		Data:  planData,
		Error: nil,
	}
}

// GenerateCommandPlan generates a plan for a command by name (used by decorators like @cmd)
func (c *PlanExecutionContext) GenerateCommandPlan(commandName string) (*ExecutionResult, error) {
	// Command plan generation is now handled directly by the engine
	return &ExecutionResult{
		Data:  nil,
		Error: fmt.Errorf("GenerateCommandPlan is deprecated - use engine plan generation directly"),
	}, nil
}

// TrackEnvironmentVariable tracks an environment variable for plan consistency
func (c *PlanExecutionContext) TrackEnvironmentVariable(key, defaultValue string) {
	if c.trackedEnvVars == nil {
		c.trackedEnvVars = make(map[string]string)
	}
	c.trackedEnvVars[key] = defaultValue
}

// GetTrackedEnvironmentVariables returns all tracked environment variables
func (c *PlanExecutionContext) GetTrackedEnvironmentVariables() map[string]string {
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

// ================================================================================================
// CONTEXT MANAGEMENT WITH TYPE SAFETY
// ================================================================================================

// Child creates a child plan context that inherits from the parent but can be modified independently
func (c *PlanExecutionContext) Child() PlanContext {
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

		// Initialize unique counter space for this child to avoid variable name conflicts
		// Each child gets a unique counter space based on parent's counter and child ID
		shellCounter: c.shellCounter + (childID * 1000), // Give each child 1000 numbers of space
		childCounter: 0,                                 // Reset child counter for this context's children
	}

	// Copy variables (child gets its own copy)
	for name, value := range c.Variables {
		childBase.Variables[name] = value
	}

	return &PlanExecutionContext{
		BaseExecutionContext: childBase,
		trackedEnvVars:       make(map[string]string),
	}
}

// WithTimeout creates a new plan context with timeout
func (c *PlanExecutionContext) WithTimeout(timeout time.Duration) (PlanContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &PlanExecutionContext{BaseExecutionContext: &newBase}, cancel
}

// WithCancel creates a new plan context with cancellation
func (c *PlanExecutionContext) WithCancel() (PlanContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &PlanExecutionContext{BaseExecutionContext: &newBase}, cancel
}

// WithWorkingDir creates a new plan context with the specified working directory
func (c *PlanExecutionContext) WithWorkingDir(workingDir string) PlanContext {
	newBase := *c.BaseExecutionContext
	newBase.WorkingDir = workingDir
	return &PlanExecutionContext{BaseExecutionContext: &newBase}
}

// WithCurrentCommand creates a new plan context with the specified current command name
func (c *PlanExecutionContext) WithCurrentCommand(commandName string) PlanContext {
	newBase := *c.BaseExecutionContext
	newBase.currentCommand = commandName
	return &PlanExecutionContext{BaseExecutionContext: &newBase}
}

// ================================================================================================
// PLAN-SAFE SHELL COMMAND COMPOSITION
// ================================================================================================

// composeShellCommandForPlan composes shell command for plan display without executing ActionDecorators
func (c *PlanExecutionContext) composeShellCommandForPlan(content *ast.ShellContent) (string, error) {
	var parts []string

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			parts = append(parts, p.Text)
		case *ast.ValueDecorator:
			// For plan mode, resolve value decorators to show actual values
			// Special handling for @var decorator which just needs variable lookup
			if p.Name == "var" && len(p.Args) > 0 {
				// Extract variable name from decorator arguments
				if len(p.Args) > 0 {
					var varName string
					// Try to get named parameter "name" first
					for _, arg := range p.Args {
						if arg.Name == "name" {
							if ident, ok := arg.Value.(*ast.Identifier); ok {
								varName = ident.Name
								break
							}
						}
					}
					// Fallback to first parameter if no named "name" parameter
					if varName == "" && len(p.Args) > 0 {
						if ident, ok := p.Args[0].Value.(*ast.Identifier); ok {
							varName = ident.Name
						}
					}

					// Look up the variable value
					if varName != "" {
						if value, exists := c.GetVariable(varName); exists {
							parts = append(parts, value)
						} else {
							parts = append(parts, fmt.Sprintf("@var(%s)", varName))
						}
					} else {
						parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
					}
				} else {
					parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
				}
			} else {
				// For other value decorators, show decorator syntax
				parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
			}
		case *ast.ActionDecorator:
			// For plan mode, just show the decorator syntax without executing
			parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
		default:
			return "", fmt.Errorf("unsupported shell part type for plan: %T", part)
		}
	}

	return strings.Join(parts, ""), nil
}
