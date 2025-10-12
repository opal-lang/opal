package lexer

import "testing"

// TestBasicStrings tests simple string literal recognition
func TestBasicStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "double quoted string",
			input: `"hello world"`,
			expected: []tokenExpectation{
				{STRING, `"hello world"`, 1, 1},
				{EOF, "", 1, 14},
			},
		},
		{
			name:  "single quoted string",
			input: `'hello world'`,
			expected: []tokenExpectation{
				{STRING, `'hello world'`, 1, 1},
				{EOF, "", 1, 14},
			},
		},
		{
			name:  "backtick string",
			input: "`hello world`",
			expected: []tokenExpectation{
				{STRING, "`hello world`", 1, 1},
				{EOF, "", 1, 14},
			},
		},
		{
			name:  "empty double quoted string",
			input: `""`,
			expected: []tokenExpectation{
				{STRING, `""`, 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "empty single quoted string",
			input: `''`,
			expected: []tokenExpectation{
				{STRING, `''`, 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "empty backtick string",
			input: "``",
			expected: []tokenExpectation{
				{STRING, "``", 1, 1},
				{EOF, "", 1, 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestMultilineStrings tests backtick multiline strings
func TestMultilineStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "backtick with newline",
			input: "`line1\nline2`",
			expected: []tokenExpectation{
				{STRING, "`line1\nline2`", 1, 1},
				{EOF, "", 2, 7}, // Ends on line 2
			},
		},
		{
			name:  "backtick multiple newlines",
			input: "`first\nsecond\nthird`",
			expected: []tokenExpectation{
				{STRING, "`first\nsecond\nthird`", 1, 1},
				{EOF, "", 3, 7}, // Ends on line 3
			},
		},
		{
			name:  "backtick with leading/trailing newlines",
			input: "`\nhello\n`",
			expected: []tokenExpectation{
				{STRING, "`\nhello\n`", 1, 1},
				{EOF, "", 3, 2}, // Ends on line 3, column 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestStringWithInterpolation tests @var() and @env() in strings
func TestStringWithInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "double quoted with @var",
			input: `"Deploy @var(SERVICE) to production"`,
			expected: []tokenExpectation{
				{STRING, `"Deploy @var(SERVICE) to production"`, 1, 1},
				{EOF, "", 1, 37},
			},
		},
		{
			name:  "backtick with @env",
			input: "`Server: @env(HOSTNAME)\nPort: @env(PORT)`",
			expected: []tokenExpectation{
				{STRING, "`Server: @env(HOSTNAME)\nPort: @env(PORT)`", 1, 1},
				{EOF, "", 2, 18},
			},
		},
		{
			name:  "single quoted no interpolation",
			input: `'Raw string with @var(SERVICE) literal'`,
			expected: []tokenExpectation{
				{STRING, `'Raw string with @var(SERVICE) literal'`, 1, 1},
				{EOF, "", 1, 40},
			},
		},
		{
			name:  "multiple interpolations",
			input: `"@var(ENV): @var(SERVICE) v@var(VERSION)"`,
			expected: []tokenExpectation{
				{STRING, `"@var(ENV): @var(SERVICE) v@var(VERSION)"`, 1, 1},
				{EOF, "", 1, 42},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestStringWithWhitespace tests strings with surrounding whitespace
func TestStringWithWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "string with leading space",
			input: ` "hello"`,
			expected: []tokenExpectation{
				{STRING, `"hello"`, 1, 2},
				{EOF, "", 1, 9},
			},
		},
		{
			name:  "string with trailing space",
			input: `"hello" `,
			expected: []tokenExpectation{
				{STRING, `"hello"`, 1, 1},
				{EOF, "", 1, 9},
			},
		},
		{
			name:  "string with tabs",
			input: "\t`multiline`\t",
			expected: []tokenExpectation{
				{STRING, "`multiline`", 1, 2}, // Tab = 1 byte
				{EOF, "", 1, 14},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestStringInContext tests strings in realistic scenarios
func TestStringInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable assignment with string",
			input: `var MESSAGE = "Hello World"`,
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "MESSAGE", 1, 5},
				{EQUALS, "", 1, 13},
				{STRING, `"Hello World"`, 1, 15},
				{EOF, "", 1, 28},
			},
		},
		{
			name:  "command with string argument",
			input: `echo "Starting deployment"`,
			expected: []tokenExpectation{
				{IDENTIFIER, "echo", 1, 1},
				{STRING, `"Starting deployment"`, 1, 6},
				{EOF, "", 1, 27},
			},
		},
		{
			name:  "array of strings",
			input: `["api", "worker", "ui"]`,
			expected: []tokenExpectation{
				{LSQUARE, "", 1, 1},
				{STRING, `"api"`, 1, 2},
				{COMMA, "", 1, 7},
				{STRING, `"worker"`, 1, 9},
				{COMMA, "", 1, 17},
				{STRING, `"ui"`, 1, 19},
				{RSQUARE, "", 1, 23},
				{EOF, "", 1, 24},
			},
		},
		{
			name:  "multiline command block",
			input: "deploy: {\n    echo `Starting deployment\n    of @var(SERVICE)`\n}",
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy", 1, 1},
				{COLON, "", 1, 7},
				{LBRACE, "", 1, 9},
				{NEWLINE, "", 1, 10},
				{IDENTIFIER, "echo", 2, 5}, // NEWLINE tokens now meaningful
				{STRING, "`Starting deployment\n    of @var(SERVICE)`", 2, 10},
				{NEWLINE, "", 3, 22},
				{RBRACE, "", 4, 1}, // NEWLINE tokens now meaningful
				{EOF, "", 4, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestStringEscaping tests escape sequences in strings
func TestStringEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "escaped quotes in double string",
			input: `"He said \"hello\""`,
			expected: []tokenExpectation{
				{STRING, `"He said \"hello\""`, 1, 1},
				{EOF, "", 1, 20}, // String length 19, EOF at 20
			},
		},
		{
			name:  "escaped quotes in single string",
			input: `'It\'s working'`,
			expected: []tokenExpectation{
				{STRING, `'It\'s working'`, 1, 1},
				{EOF, "", 1, 16}, // String length 15, EOF at 16
			},
		},
		{
			name:  "escaped newline in double string",
			input: `"line1\nline2"`,
			expected: []tokenExpectation{
				{STRING, `"line1\nline2"`, 1, 1},
				{EOF, "", 1, 15},
			},
		},
		{
			name:  "escaped backtick in backtick string",
			input: "`code: \\`echo hello\\``",
			expected: []tokenExpectation{
				{STRING, "`code: \\`echo hello\\``", 1, 1},
				{EOF, "", 1, 23}, // String length 22, EOF at 23
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestStringPositioning tests accurate position tracking for strings
func TestStringPositioning(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedLine int
		expectedCol  int
	}{
		{name: "string at start", input: `"test"`, expectedLine: 1, expectedCol: 1},
		{name: "string after spaces", input: `   "test"`, expectedLine: 1, expectedCol: 4},
		{name: "string after tab", input: "\t\"test\"", expectedLine: 1, expectedCol: 2},
		{name: "string second line", input: "x\n\"test\"", expectedLine: 2, expectedCol: 1},
		{name: "multiline string", input: "x\n`multi\nline`", expectedLine: 2, expectedCol: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newTestLexer(tt.input)

			// Find the string token
			for {
				token := lexer.NextToken()
				if token.Type == STRING {
					if token.Position.Line != tt.expectedLine {
						t.Errorf("Expected line %d, got %d", tt.expectedLine, token.Position.Line)
					}
					if token.Position.Column != tt.expectedCol {
						t.Errorf("Expected column %d, got %d", tt.expectedCol, token.Position.Column)
					}
					return
				}
				if token.Type == EOF {
					t.Errorf("Never found STRING token in input %q", tt.input)
					return
				}
			}
		})
	}
}
