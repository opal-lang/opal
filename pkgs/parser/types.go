package parser

import (
	"fmt"
	"strings"
)

// CommandElement represents any element that can appear in command text
// This supports a proper AST structure for nested decorators
type CommandElement interface {
	String() string
	IsDecorator() bool
}

// TextElement represents literal text in commands
type TextElement struct {
	Text string
}

func (t *TextElement) String() string {
	return t.Text
}

func (t *TextElement) IsDecorator() bool {
	return false
}

// DecoratorElement represents a decorator like @var(SRC) or @sh(...)
type DecoratorElement struct {
	Name    string           // "var", "sh", "parallel", etc.
	Type    string           // "function", "simple", "block"
	Args    []CommandElement // For function decorators: contents of @name(...)
	Block   []BlockStatement // For block decorators: @name: { ... }
	Command []CommandElement // For simple decorators: @name: command
}

func (d *DecoratorElement) String() string {
	switch d.Type {
	case "function":
		var argStrs []string
		for _, arg := range d.Args {
			argStrs = append(argStrs, arg.String())
		}
		return fmt.Sprintf("@%s(%s)", d.Name, strings.Join(argStrs, ""))
	case "simple":
		var cmdStrs []string
		for _, cmd := range d.Command {
			cmdStrs = append(cmdStrs, cmd.String())
		}
		return fmt.Sprintf("@%s: %s", d.Name, strings.Join(cmdStrs, ""))
	case "block":
		// Block representation would be more complex
		return fmt.Sprintf("@%s: { ... }", d.Name)
	default:
		return fmt.Sprintf("@%s", d.Name)
	}
}

func (d *DecoratorElement) IsDecorator() bool {
	return true
}

// BlockStatement represents a statement within a block command
// Enhanced to support nested decorator structures
type BlockStatement struct {
	// New AST-based approach
	Elements []CommandElement // Command broken into elements (text + decorators)

	// Legacy fields for backward compatibility
	Command        string           // Flattened command text (for compatibility)
	IsDecorated    bool             // Whether this is a decorated command
	Decorator      string           // The decorator name
	DecoratorType  string           // "function", "simple", or "block"
	DecoratedBlock []BlockStatement // For block-type decorators
}

// Helper methods for BlockStatement (updated for new structure)
func (bs *BlockStatement) IsFunction() bool {
	return bs.IsDecorated && bs.DecoratorType == "function"
}

func (bs *BlockStatement) IsSimpleDecorator() bool {
	return bs.IsDecorated && bs.DecoratorType == "simple"
}

func (bs *BlockStatement) IsBlockDecorator() bool {
	return bs.IsDecorated && bs.DecoratorType == "block"
}

func (bs *BlockStatement) GetCommand() string {
	if bs.Command != "" {
		return bs.Command // Use legacy field if available
	}

	// Generate from elements
	var parts []string
	for _, elem := range bs.Elements {
		parts = append(parts, elem.String())
	}
	return strings.Join(parts, "")
}

func (bs *BlockStatement) GetDecorator() string {
	return bs.Decorator
}

func (bs *BlockStatement) GetNestedBlock() []BlockStatement {
	return bs.DecoratedBlock
}

// GetParsedElements returns the structured command elements
// This is the new API for accessing the parsed structure
func (bs *BlockStatement) GetParsedElements() []CommandElement {
	return bs.Elements
}

// HasNestedDecorators checks if this statement contains nested decorators
func (bs *BlockStatement) HasNestedDecorators() bool {
	for _, elem := range bs.Elements {
		if elem.IsDecorator() {
			return true
		}
	}
	return false
}

// GetDecorators returns all decorator elements in this statement
func (bs *BlockStatement) GetDecorators() []*DecoratorElement {
	var decorators []*DecoratorElement
	for _, elem := range bs.Elements {
		if decorator, ok := elem.(*DecoratorElement); ok {
			decorators = append(decorators, decorator)
		}
	}
	return decorators
}

// Definition represents a variable definition in the command file
type Definition struct {
	Name  string // The variable name
	Value string // The variable value
	Line  int    // The line number in the source file
}

// Command represents a command definition in the command file
// Enhanced to support the new AST structure
type Command struct {
	Name    string           // The command name
	Command string           // The command text for simple commands (legacy)
	Line    int              // The line number in the source file
	IsWatch bool             // Whether this is a watch command
	IsStop  bool             // Whether this is a stop command
	IsBlock bool             // Whether this is a block command
	Block   []BlockStatement // The statements for block commands

	// New structured representation
	Elements []CommandElement // For simple commands broken into elements
}

// GetParsedElements returns the structured command elements for simple commands
func (c *Command) GetParsedElements() []CommandElement {
	return c.Elements
}

// HasNestedDecorators checks if this command contains nested decorators
func (c *Command) HasNestedDecorators() bool {
	if c.IsBlock {
		for _, stmt := range c.Block {
			if stmt.HasNestedDecorators() {
				return true
			}
		}
		return false
	}

	for _, elem := range c.Elements {
		if elem.IsDecorator() {
			return true
		}
	}
	return false
}

// CommandFile represents the parsed command file
type CommandFile struct {
	Definitions []Definition // All variable definitions
	Commands    []Command    // All command definitions
	Lines       []string     // Original file lines for error reporting
}
