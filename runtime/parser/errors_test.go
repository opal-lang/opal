package parser

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/runtime/lexer"
)

func TestErrorFormatterCompact(t *testing.T) {
	source := []byte("fun greet() {}\nfun deploy(env) {}\nfun greet(name {\n")

	err := ParseError{
		Filename:   "commands.opl",
		Position:   lexer.Position{Line: 3, Column: 15, Offset: 50},
		Message:    "missing closing parenthesis",
		Context:    "parameter list",
		Expected:   []lexer.TokenType{lexer.RPAREN},
		Got:        lexer.LBRACE,
		Suggestion: "Add ')' after the last parameter",
	}

	formatter := ErrorFormatter{
		Source:   source,
		Filename: "commands.opl",
		Compact:  true,
		Color:    false,
	}

	output := formatter.Format(err)

	// Expected format:
	// commands.opl:3:15: missing closing parenthesis in parameter list
	//  3 | fun greet(name {
	//    |               ^ expected ')'
	//    Add ')' after the last parameter

	expectedLines := []string{
		"commands.opl:3:15: missing closing parenthesis in parameter list",
		" 3 | fun greet(name {",
		"   |               ^ expected ')'",
		"   Add ')' after the last parameter",
	}

	outputLines := strings.Split(strings.TrimSpace(output), "\n")

	if len(outputLines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d\nOutput:\n%s", len(expectedLines), len(outputLines), output)
	}

	for i, expected := range expectedLines {
		if i >= len(outputLines) {
			t.Errorf("Missing line %d: %q", i, expected)
			continue
		}
		if outputLines[i] != expected {
			t.Errorf("Line %d mismatch:\nExpected: %q\nGot:      %q", i, expected, outputLines[i])
		}
	}
}

func TestErrorFormatterDetailed(t *testing.T) {
	source := []byte("fun greet() {}\nfun deploy(env) {}\nfun greet(name {\n")

	err := ParseError{
		Filename:   "commands.opl",
		Position:   lexer.Position{Line: 3, Column: 15, Offset: 50},
		Message:    "missing closing parenthesis",
		Context:    "parameter list",
		Expected:   []lexer.TokenType{lexer.RPAREN},
		Got:        lexer.LBRACE,
		Suggestion: "Add ')' after the last parameter",
		Example:    "fun greet(name) {}",
		Note:       "Parameter lists must be enclosed in parentheses",
	}

	formatter := ErrorFormatter{
		Source:   source,
		Filename: "commands.opl",
		Compact:  false,
		Color:    false,
	}

	output := formatter.Format(err)

	// Expected format:
	// Error: missing closing parenthesis
	//   --> commands.opl:3:15
	//    |
	//  3 | fun greet(name {
	//    |               ^ expected ')'
	//    |
	//    = Suggestion: Add ')' after the last parameter
	//    = Example: fun greet(name) {}
	//    = Note: Parameter lists must be enclosed in parentheses

	expectedLines := []string{
		"Error: missing closing parenthesis",
		"  --> commands.opl:3:15",
		"   |",
		" 3 | fun greet(name {",
		"   |               ^ expected ')'",
		"   |",
		"   = Suggestion: Add ')' after the last parameter",
		"   = Example: fun greet(name) {}",
		"   = Note: Parameter lists must be enclosed in parentheses",
	}

	outputLines := strings.Split(strings.TrimSpace(output), "\n")

	if len(outputLines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d\nOutput:\n%s", len(expectedLines), len(outputLines), output)
	}

	for i, expected := range expectedLines {
		if i >= len(outputLines) {
			t.Errorf("Missing line %d: %q", i, expected)
			continue
		}
		if outputLines[i] != expected {
			t.Errorf("Line %d mismatch:\nExpected: %q\nGot:      %q", i, expected, outputLines[i])
		}
	}
}

func TestErrorFormatterMultipleExpected(t *testing.T) {
	err := ParseError{
		Position: lexer.Position{Line: 1, Column: 5, Offset: 4},
		Message:  "unexpected token",
		Expected: []lexer.TokenType{lexer.IDENTIFIER, lexer.LPAREN, lexer.LBRACE},
		Got:      lexer.SEMICOLON,
	}

	formatter := ErrorFormatter{
		Source:  []byte("fun ;"),
		Compact: true,
		Color:   false,
	}

	output := formatter.Format(err)

	// Should format as: "expected identifier, '(', or '{'"
	if !strings.Contains(output, "identifier, '(', or '{'") {
		t.Errorf("Expected formatted token list, got:\n%s", output)
	}
}

func TestTokenName(t *testing.T) {
	tests := []struct {
		token lexer.TokenType
		want  string
	}{
		{lexer.LPAREN, "'('"},
		{lexer.RPAREN, "')'"},
		{lexer.LBRACE, "'{'"},
		{lexer.RBRACE, "'}'"},
		{lexer.IDENTIFIER, "identifier"},
		{lexer.STRING, "string"},
		{lexer.INTEGER, "number"},
		{lexer.EOF, "end of file"},
	}

	for _, tt := range tests {
		got := tokenName(tt.token)
		if got != tt.want {
			t.Errorf("tokenName(%v) = %q, want %q", tt.token, got, tt.want)
		}
	}
}
