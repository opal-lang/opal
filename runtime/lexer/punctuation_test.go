package lexer

import "testing"

// TestBasicPunctuation tests simple single-character punctuation
func TestBasicPunctuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "equals sign",
			input: "=",
			expected: []tokenExpectation{
				{EQUALS, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "colon",
			input: ":",
			expected: []tokenExpectation{
				{COLON, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "left brace",
			input: "{",
			expected: []tokenExpectation{
				{LBRACE, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "right brace",
			input: "}",
			expected: []tokenExpectation{
				{RBRACE, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "left paren",
			input: "(",
			expected: []tokenExpectation{
				{LPAREN, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "right paren",
			input: ")",
			expected: []tokenExpectation{
				{RPAREN, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "at symbol",
			input: "@",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestPunctuationWithWhitespace tests punctuation with surrounding whitespace
func TestPunctuationWithWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "equals with spaces",
			input: " = ",
			expected: []tokenExpectation{
				{EQUALS, "", 1, 2},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "braces with tabs",
			input: "\t{\t}",
			expected: []tokenExpectation{
				{LBRACE, "", 1, 2}, // Tab = 1 byte
				{RBRACE, "", 1, 4}, // Tab after { = 1 more byte
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "colon with newline",
			input: ":\n",
			expected: []tokenExpectation{
				{COLON, "", 1, 1},
				{NEWLINE, "", 1, 2}, // Newline is now meaningful
				{EOF, "", 2, 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestPunctuationInContext tests punctuation in realistic scenarios
func TestPunctuationInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable assignment",
			input: "var name = value",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "name", 1, 5},
				{EQUALS, "", 1, 10},
				{IDENTIFIER, "value", 1, 12},
				{EOF, "", 1, 17},
			},
		},
		{
			name:  "command definition",
			input: "deploy: {",
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy", 1, 1},
				{COLON, "", 1, 7},
				{LBRACE, "", 1, 9},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "function call",
			input: "func(arg)",
			expected: []tokenExpectation{
				{IDENTIFIER, "func", 1, 1},
				{LPAREN, "", 1, 5},
				{IDENTIFIER, "arg", 1, 6},
				{RPAREN, "", 1, 9},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "nested braces",
			input: "{ { } }",
			expected: []tokenExpectation{
				{LBRACE, "", 1, 1},
				{LBRACE, "", 1, 3},
				{RBRACE, "", 1, 5},
				{RBRACE, "", 1, 7},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "decorator syntax",
			input: "@env(NAME)",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "env", 1, 2},
				{LPAREN, "", 1, 5},
				{IDENTIFIER, "NAME", 1, 6},
				{RPAREN, "", 1, 10},
				{EOF, "", 1, 11},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestSquareBrackets tests array/index syntax
func TestSquareBrackets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "left square bracket",
			input: "[",
			expected: []tokenExpectation{
				{LSQUARE, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "right square bracket",
			input: "]",
			expected: []tokenExpectation{
				{RSQUARE, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "array access",
			input: "items[index]", // Use identifier instead of number for now
			expected: []tokenExpectation{
				{IDENTIFIER, "items", 1, 1},
				{LSQUARE, "", 1, 6},
				{IDENTIFIER, "index", 1, 7},
				{RSQUARE, "", 1, 12},
				{EOF, "", 1, 13},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestCommaAndSemicolon tests separators
func TestCommaAndSemicolon(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "comma",
			input: ",",
			expected: []tokenExpectation{
				{COMMA, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "semicolon",
			input: ";",
			expected: []tokenExpectation{
				{SEMICOLON, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "array literal",
			input: "[a, b, c]",
			expected: []tokenExpectation{
				{LSQUARE, "", 1, 1},
				{IDENTIFIER, "a", 1, 2},
				{COMMA, "", 1, 3},
				{IDENTIFIER, "b", 1, 5},
				{COMMA, "", 1, 6},
				{IDENTIFIER, "c", 1, 8},
				{RSQUARE, "", 1, 9},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "sequential commands",
			input: "cmd1; cmd2",
			expected: []tokenExpectation{
				{IDENTIFIER, "cmd1", 1, 1},
				{SEMICOLON, "", 1, 5},
				{IDENTIFIER, "cmd2", 1, 7},
				{EOF, "", 1, 11},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestPunctuationPositioning tests accurate position tracking
func TestPunctuationPositioning(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		tokenType    TokenType
		expectedLine int
		expectedCol  int
	}{
		{name: "equals at start", input: "=", tokenType: EQUALS, expectedLine: 1, expectedCol: 1},
		{name: "colon after spaces", input: "   :", tokenType: COLON, expectedLine: 1, expectedCol: 4},
		{name: "brace after tab", input: "\t{", tokenType: LBRACE, expectedLine: 1, expectedCol: 2},
		{name: "comma second line", input: "x\n,", tokenType: COMMA, expectedLine: 2, expectedCol: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newTestLexer(tt.input)

			// Find the punctuation token
			for {
				token := lexer.NextToken()
				if token.Type == tt.tokenType {
					if token.Position.Line != tt.expectedLine {
						t.Errorf("Expected line %d, got %d", tt.expectedLine, token.Position.Line)
					}
					if token.Position.Column != tt.expectedCol {
						t.Errorf("Expected column %d, got %d", tt.expectedCol, token.Position.Column)
					}
					return
				}
				if token.Type == EOF {
					t.Errorf("Never found %v token in input %q", tt.tokenType, tt.input)
					return
				}
			}
		})
	}
}
