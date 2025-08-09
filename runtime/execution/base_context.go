package execution

import (
	"context"
	"fmt"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ChainElement represents an element in an ActionDecorator command chain
type ChainElement struct {
	Type         string               // "action", "operator", "text"
	ActionName   string               // For ActionDecorator
	ActionArgs   []ast.NamedParameter // For ActionDecorator
	Operator     string               // "&&", "||", "|", ">>"
	Text         string               // For text parts
	VariableName string               // Generated variable name
	IsPipeTarget bool                 // True if this element receives piped input
	IsFileTarget bool                 // True if this element is a file for >> operation
}

// ChainOperator represents the type of chaining operator
type ChainOperator string

const (
	AndOperator    ChainOperator = "&&" // Execute next if current succeeds
	OrOperator     ChainOperator = "||" // Execute next if current fails
	PipeOperator   ChainOperator = "|"  // Pipe stdout to next command
	AppendOperator ChainOperator = ">>" // Append stdout to file
)

// BaseExecutionContext provides the common implementation for all execution contexts
type BaseExecutionContext struct {
	context.Context

	// Core data
	Program   *ast.Program
	Variables map[string]string // Resolved variable values
	env       map[string]string // Immutable environment variables captured at command start

	// Execution state
	WorkingDir string
	Debug      bool
	DryRun     bool

	// Current command name for generating meaningful variable names
	currentCommand string

	// Decorator lookup functions (set by engine during initialization)
	valueDecoratorLookup  func(name string) (interface{}, bool)
	actionDecoratorLookup func(name string) (interface{}, bool)
	blockDecoratorLookup  func(name string) (interface{}, bool)

	// Shell execution counter for unique variable naming
	shellCounter int

	// Child context counter for unique variable naming across parallel contexts
	childCounter int
}

// SetValueDecoratorLookup sets the value decorator lookup function (called by engine during setup)
func (c *BaseExecutionContext) SetValueDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.valueDecoratorLookup = lookup
}

// SetActionDecoratorLookup sets the action decorator lookup function (called by engine during setup)
func (c *BaseExecutionContext) SetActionDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.actionDecoratorLookup = lookup
}

// SetBlockDecoratorLookup sets the block decorator lookup function (called by engine during setup)
func (c *BaseExecutionContext) SetBlockDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.blockDecoratorLookup = lookup
}

// newBaseExecutionContext creates a new base execution context

// GetVariable retrieves a variable value
func (c *BaseExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := c.Variables[name]
	return value, exists
}

// SetVariable sets a variable value
func (c *BaseExecutionContext) SetVariable(name, value string) {
	c.Variables[name] = value
}

// GetEnv retrieves an environment variable from the immutable captured environment
func (c *BaseExecutionContext) GetEnv(name string) (string, bool) {
	value, exists := c.env[name]
	return value, exists
}

// GetProgram returns the AST program
func (c *BaseExecutionContext) GetProgram() *ast.Program {
	return c.Program
}

// GetWorkingDir returns the current working directory
func (c *BaseExecutionContext) GetWorkingDir() string {
	return c.WorkingDir
}

// IsDebug returns whether debug mode is enabled
func (c *BaseExecutionContext) IsDebug() bool {
	return c.Debug
}

// IsDryRun returns whether dry run mode is enabled
func (c *BaseExecutionContext) IsDryRun() bool {
	return c.DryRun
}

// InitializeVariables processes and sets all variables from the program
func (c *BaseExecutionContext) InitializeVariables() error {
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

// resolveVariableValue converts an AST expression to its string value
func (c *BaseExecutionContext) resolveVariableValue(expr ast.Expression) (string, error) {
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

// ================================================================================================
// SHARED UTILITY METHODS
// ================================================================================================
