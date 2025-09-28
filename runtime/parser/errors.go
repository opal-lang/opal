package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/opal/core/types"
)

// ParseError represents a parsing error with location and context information
type ParseError struct {
	Type        ErrorType
	Message     string
	Token       types.Token
	Input       string
	Context     string
	OpenedAt    *types.Token // For bracket mismatch errors
	Suggestions []string     // Possible fixes
}

// BracketTracker tracks opening brackets and their context for better error reporting
type BracketTracker struct {
	stack []BracketInfo
}

type BracketInfo struct {
	Type    types.TokenType // LBRACE, LPAREN
	Token   types.Token     // Opening token with position
	Context string          // "command block", "decorator args", etc.
}

// Push adds an opening bracket to the tracker
func (bt *BracketTracker) Push(tokenType types.TokenType, token types.Token, context string) {
	bt.stack = append(bt.stack, BracketInfo{
		Type:    tokenType,
		Token:   token,
		Context: context,
	})
}

// Pop removes the last opening bracket, returns error if mismatch
func (bt *BracketTracker) Pop(expected types.TokenType, closingToken types.Token) error {
	if len(bt.stack) == 0 {
		return fmt.Errorf("unexpected '%s' at %d:%d - no matching opening bracket",
			closingToken.Value, closingToken.Line, closingToken.Column)
	}

	top := bt.stack[len(bt.stack)-1]
	bt.stack = bt.stack[:len(bt.stack)-1]

	if !isMatchingBracket(top.Type, expected) {
		return fmt.Errorf("mismatched brackets: '%s' opened at %d:%d but '%s' found at %d:%d",
			top.Token.Value, top.Token.Line, top.Token.Column,
			closingToken.Value, closingToken.Line, closingToken.Column)
	}

	return nil
}

// GetUnclosedBrackets returns all unclosed brackets for error reporting
func (bt *BracketTracker) GetUnclosedBrackets() []BracketInfo {
	return bt.stack
}

// IsEmpty returns true if all brackets are closed
func (bt *BracketTracker) IsEmpty() bool {
	return len(bt.stack) == 0
}

// Helper function to check if brackets match
func isMatchingBracket(opening, closing types.TokenType) bool {
	switch opening {
	case types.LBRACE:
		return closing == types.RBRACE
	case types.LPAREN:
		return closing == types.RPAREN
	default:
		return false
	}
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
