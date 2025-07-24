package ast

import (
	"fmt"
	"strconv"
)

// NewProgram creates a program AST node
func NewProgram(items ...interface{}) *Program {
	var variables []VariableDecl
	var commands []CommandDecl

	for _, item := range items {
		switch v := item.(type) {
		case VariableDecl:
			variables = append(variables, v)
		case CommandDecl:
			commands = append(commands, v)
		case *VariableDecl:
			variables = append(variables, *v)
		case *CommandDecl:
			commands = append(commands, *v)
		}
	}

	return &Program{
		Variables: variables,
		Commands:  commands,
	}
}

// Var creates a variable declaration: var NAME = VALUE
func Var(name string, value Expression) VariableDecl {
	return VariableDecl{
		Name:  name,
		Value: value,
	}
}

// Cmd creates a simple command: NAME: BODY
func Cmd(name string, body CommandContent) CommandDecl {
	return CommandDecl{
		Name: name,
		Type: Command,
		Body: CommandBody{
			Content: []CommandContent{body},
		},
	}
}

// Shell creates a shell content node
func Shell(parts ...ShellPart) *ShellContent {
	return &ShellContent{
		Parts: parts,
	}
}

// Text creates a text part
func Text(text string) *TextPart {
	return &TextPart{
		Text: text,
	}
}

// At creates a function decorator within shell content: @var(NAME)
func At(name string, args ...NamedParameter) *FunctionDecorator {
	return &FunctionDecorator{
		Name: name,
		Args: args,
	}
}

// Id creates an identifier expression
func Id(name string) *Identifier {
	return &Identifier{
		Name: name,
	}
}

// Str creates a string literal expression
func Str(value string) *StringLiteral {
	return &StringLiteral{
		Value: value,
	}
}

// Num creates a number literal expression
func Num(value interface{}) *NumberLiteral {
	switch v := value.(type) {
	case int:
		return &NumberLiteral{
			Value: strconv.Itoa(v),
		}
	case int64:
		return &NumberLiteral{
			Value: strconv.FormatInt(v, 10),
		}
	case float64:
		return &NumberLiteral{
			Value: strconv.FormatFloat(v, 'g', -1, 64),
		}
	case float32:
		return &NumberLiteral{
			Value: strconv.FormatFloat(float64(v), 'g', -1, 32),
		}
	case string:
		return &NumberLiteral{
			Value: v,
		}
	default:
		return &NumberLiteral{
			Value: fmt.Sprintf("%v", v),
		}
	}
}

// Bool creates a boolean literal expression
func Bool(value bool) *BooleanLiteral {
	return &BooleanLiteral{
		Value: value,
	}
}

// Dur creates a duration literal expression
func Dur(value string) *DurationLiteral {
	return &DurationLiteral{
		Value: value,
	}
}

// Param creates a named parameter for decorators
func Param(name string, value Expression) NamedParameter {
	return NamedParameter{
		Name:  name,
		Value: value,
	}
}

// UnnamedParam creates an unnamed parameter (positional)
func UnnamedParam(value Expression) NamedParameter {
	return NamedParameter{
		Value: value,
	}
}

// NewBlockDecorator creates a block decorator node
func NewBlockDecorator(name string, args []NamedParameter, content []CommandContent) *BlockDecorator {
	return &BlockDecorator{
		Name:    name,
		Args:    args,
		Content: content,
	}
}

// NewPatternDecorator creates a pattern decorator node
func NewPatternDecorator(name string, args []NamedParameter, patterns []PatternBranch) *PatternDecorator {
	return &PatternDecorator{
		Name:     name,
		Args:     args,
		Patterns: patterns,
	}
}

// NewPatternBranch creates a pattern branch
func NewPatternBranch(pattern Pattern, commands []CommandContent) PatternBranch {
	return PatternBranch{
		Pattern:  pattern,
		Commands: commands,
	}
}

// NewIdentifierPattern creates an identifier pattern
func NewIdentifierPattern(name string) *IdentifierPattern {
	return &IdentifierPattern{
		Name: name,
	}
}
