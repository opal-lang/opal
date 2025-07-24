package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/types"
)

// ParseError represents a parsing error with location and context information
type ParseError struct {
	Type    ErrorType
	Message string
	Token   types.Token
	Input   string
	Context string
}

// ErrorType represents different categories of parsing errors
type ErrorType int

const (
	ErrorSyntax ErrorType = iota
	ErrorType_
	ErrorUnexpected
	ErrorMissing
	ErrorInvalid
)

func (e ErrorType) String() string {
	switch e {
	case ErrorSyntax:
		return "syntax error"
	case ErrorType_:
		return "type mismatch"
	case ErrorUnexpected:
		return "unexpected token"
	case ErrorMissing:
		return "missing"
	case ErrorInvalid:
		return "invalid"
	default:
		return "error"
	}
}

// Error returns the formatted error message with line/column and code snippet
func (e ParseError) Error() string {
	snippet := e.createCodeSnippet()
	return fmt.Sprintf("%s: %s\n%s",
		e.Type.String(),
		e.Message,
		snippet)
}

// createCodeSnippet creates a code snippet showing the error location
func (e ParseError) createCodeSnippet() string {
	if e.Input == "" || e.Token.Line == 0 {
		return ""
	}

	lines := strings.Split(e.Input, "\n")
	if e.Token.Line > len(lines) {
		return ""
	}

	lineContent := lines[e.Token.Line-1]

	// Create the snippet in Rust/Clang style
	var snippet strings.Builder
	// Location pointer like " --> src/file.rs:5:13"
	snippet.WriteString(fmt.Sprintf("  --> %d:%d\n", e.Token.Line, e.Token.Column))
	// Line separator
	snippet.WriteString("   |\n")
	// Source line with line number
	snippet.WriteString(fmt.Sprintf("%2d | %s\n", e.Token.Line, lineContent))
	// Caret pointer line
	snippet.WriteString("   | ")
	if e.Token.Column > 0 && e.Token.Column <= len(lineContent)+1 {
		snippet.WriteString(strings.Repeat(" ", e.Token.Column-1) + "^")
	}

	return snippet.String()
}

// Helper functions for creating standard error types

// NewSyntaxError creates a syntax error with location information
func (p *Parser) NewSyntaxError(message string) error {
	return ParseError{
		Type:    ErrorSyntax,
		Message: message,
		Token:   p.current(),
		Input:   p.input,
	}
}

// NewTypeError creates a type mismatch error for parameter validation
func (p *Parser) NewTypeError(paramName string, expectedType types.ExpressionType, gotToken types.Token) error {
	message := fmt.Sprintf("parameter '%s' expects %s, got %s",
		paramName, expectedType.String(), gotToken.Type.String())

	return ParseError{
		Type:    ErrorType_,
		Message: message,
		Token:   gotToken,
		Input:   p.input,
	}
}

// NewUnexpectedTokenError creates an error for unexpected tokens
func (p *Parser) NewUnexpectedTokenError(expected string, got types.Token) error {
	message := fmt.Sprintf("expected %s, got %s", expected, got.Type.String())

	return ParseError{
		Type:    ErrorUnexpected,
		Message: message,
		Token:   got,
		Input:   p.input,
	}
}

// NewMissingTokenError creates an error for missing expected tokens
func (p *Parser) NewMissingTokenError(expected string) error {
	message := fmt.Sprintf("expected %s", expected)

	return ParseError{
		Type:    ErrorMissing,
		Message: message,
		Token:   p.current(),
		Input:   p.input,
	}
}

// NewInvalidError creates a generic invalid error
func (p *Parser) NewInvalidError(message string) error {
	return ParseError{
		Type:    ErrorInvalid,
		Message: message,
		Token:   p.current(),
		Input:   p.input,
	}
}

// NewGenericError creates a generic error (for backwards compatibility)
func (p *Parser) NewGenericError(message string) error {
	return ParseError{
		Type:    ErrorSyntax,
		Message: message,
		Token:   p.current(),
		Input:   p.input,
	}
}
