package lexer

import "testing"

// TestVarKeyword tests the "var" keyword recognition
func TestVarKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple var",
			input: "var",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "var with whitespace",
			input: "  var  ",
			expected: []tokenExpectation{
				{VAR, "var", 1, 3},
				{EOF, "", 1, 8}, // EOF after skipping trailing whitespace
			},
		},
		{
			name:  "var declaration context",
			input: "var API_KEY",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "API_KEY", 1, 5},
				{EOF, "", 1, 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestFunKeyword tests the "fun" keyword recognition
func TestFunKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple fun",
			input: "fun",
			expected: []tokenExpectation{
				{FUN, "fun", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "fun with whitespace",
			input: "  fun  ",
			expected: []tokenExpectation{
				{FUN, "fun", 1, 3},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "fun assignment syntax",
			input: "fun deploy =",
			expected: []tokenExpectation{
				{FUN, "fun", 1, 1},
				{IDENTIFIER, "deploy", 1, 5},
				{EQUALS, "", 1, 12},
				{EOF, "", 1, 13},
			},
		},
		{
			name:  "fun with parameters",
			input: "fun greet(name) =",
			expected: []tokenExpectation{
				{FUN, "fun", 1, 1},
				{IDENTIFIER, "greet", 1, 5},
				{LPAREN, "", 1, 10},
				{IDENTIFIER, "name", 1, 11},
				{RPAREN, "", 1, 15},
				{EQUALS, "", 1, 17},
				{EOF, "", 1, 18},
			},
		},
		{
			name:  "fun block syntax",
			input: "fun test {",
			expected: []tokenExpectation{
				{FUN, "fun", 1, 1},
				{IDENTIFIER, "test", 1, 5},
				{LBRACE, "", 1, 10},
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

// TestForKeyword tests the "for" keyword recognition
func TestForKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple for",
			input: "for",
			expected: []tokenExpectation{
				{FOR, "for", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "for loop context",
			input: "for service in",
			expected: []tokenExpectation{
				{FOR, "for", 1, 1},
				{IDENTIFIER, "service", 1, 5},
				{IN, "in", 1, 13},
				{EOF, "", 1, 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestIfKeyword tests the "if" keyword recognition
func TestIfKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple if",
			input: "if",
			expected: []tokenExpectation{
				{IF, "if", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "if condition context",
			input: "if ENV",
			expected: []tokenExpectation{
				{IF, "if", 1, 1},
				{IDENTIFIER, "ENV", 1, 4},
				{EOF, "", 1, 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestWhenKeyword tests the "when" keyword recognition
func TestWhenKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple when",
			input: "when",
			expected: []tokenExpectation{
				{WHEN, "when", 1, 1},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "when pattern context",
			input: "when ENV",
			expected: []tokenExpectation{
				{WHEN, "when", 1, 1},
				{IDENTIFIER, "ENV", 1, 6},
				{EOF, "", 1, 9},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestTryKeyword tests the "try" keyword recognition
func TestTryKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple try",
			input: "try",
			expected: []tokenExpectation{
				{TRY, "try", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "try catch context",
			input: "try catch",
			expected: []tokenExpectation{
				{TRY, "try", 1, 1},
				{CATCH, "catch", 1, 5},
				{EOF, "", 1, 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestElseKeyword tests the "else" keyword recognition
func TestElseKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple else",
			input: "else",
			expected: []tokenExpectation{
				{ELSE, "else", 1, 1},
				{EOF, "", 1, 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestKeywordVsIdentifier tests distinguishing keywords from similar identifiers
func TestKeywordVsIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TokenType
		text     string
	}{
		// Keywords should be recognized as keywords
		{name: "var keyword", input: "var", expected: VAR, text: "var"},
		{name: "struct keyword", input: "struct", expected: STRUCT, text: "struct"},
		{name: "for keyword", input: "for", expected: FOR, text: "for"},
		{name: "if keyword", input: "if", expected: IF, text: "if"},
		{name: "when keyword", input: "when", expected: WHEN, text: "when"},
		{name: "try keyword", input: "try", expected: TRY, text: "try"},
		{name: "else keyword", input: "else", expected: ELSE, text: "else"},
		{name: "in keyword", input: "in", expected: IN, text: "in"},
		{name: "catch keyword", input: "catch", expected: CATCH, text: "catch"},
		{name: "finally keyword", input: "finally", expected: FINALLY, text: "finally"},

		// Identifiers with keywords as prefix should be IDENTIFIER
		{name: "variable identifier", input: "variable", expected: IDENTIFIER, text: "variable"},
		{name: "var_name identifier", input: "var_name", expected: IDENTIFIER, text: "var_name"},
		{name: "forEach identifier", input: "forEach", expected: IDENTIFIER, text: "forEach"},
		{name: "for_each identifier", input: "for_each", expected: IDENTIFIER, text: "for_each"},
		{name: "ifTrue identifier", input: "ifTrue", expected: IDENTIFIER, text: "ifTrue"},
		{name: "ifCondition identifier", input: "ifCondition", expected: IDENTIFIER, text: "ifCondition"},
		{name: "tryAgain identifier", input: "tryAgain", expected: IDENTIFIER, text: "tryAgain"},
		{name: "try_catch identifier", input: "try_catch", expected: IDENTIFIER, text: "try_catch"},
		{name: "when_clause identifier", input: "when_clause", expected: IDENTIFIER, text: "when_clause"},
		{name: "elsewhere identifier", input: "elsewhere", expected: IDENTIFIER, text: "elsewhere"},

		// Identifiers with keywords as suffix should be IDENTIFIER
		{name: "myvar identifier", input: "myvar", expected: IDENTIFIER, text: "myvar"},
		{name: "config_var identifier", input: "config_var", expected: IDENTIFIER, text: "config_var"},
		{name: "what_if identifier", input: "what_if", expected: IDENTIFIER, text: "what_if"},
		{name: "until_when identifier", input: "until_when", expected: IDENTIFIER, text: "until_when"},
		{name: "did_try identifier", input: "did_try", expected: IDENTIFIER, text: "did_try"},

		// Identifiers with keywords in middle should be IDENTIFIER
		{name: "config_var_name identifier", input: "config_var_name", expected: IDENTIFIER, text: "config_var_name"},
		{name: "do_for_loop identifier", input: "do_for_loop", expected: IDENTIFIER, text: "do_for_loop"},
		{name: "check_if_true identifier", input: "check_if_true", expected: IDENTIFIER, text: "check_if_true"},

		// Mixed case variations should be IDENTIFIER (keywords are case-sensitive)
		{name: "Var identifier", input: "Var", expected: IDENTIFIER, text: "Var"},
		{name: "VAR identifier", input: "VAR", expected: IDENTIFIER, text: "VAR"},
		{name: "For identifier", input: "For", expected: IDENTIFIER, text: "For"},
		{name: "IF identifier", input: "IF", expected: IDENTIFIER, text: "IF"},
		{name: "TRY identifier", input: "TRY", expected: IDENTIFIER, text: "TRY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newTestLexer(tt.input)
			token := lexer.NextToken()

			if token.Type != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, token.Type)
			}
			if token.String() != tt.text {
				t.Errorf("Expected text %q, got %q", tt.text, token.String())
			}
		})
	}
}

// TestKeywordBoundaries tests that keywords must be complete words
func TestKeywordBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "var followed by underscore",
			input: "var_API_KEY",
			expected: []tokenExpectation{
				{IDENTIFIER, "var_API_KEY", 1, 1}, // Should be one identifier, not var + _API_KEY
				{EOF, "", 1, 12},
			},
		},
		{
			name:  "for followed by number",
			input: "for2",
			expected: []tokenExpectation{
				{IDENTIFIER, "for2", 1, 1}, // Should be one identifier, not for + 2
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "if followed by hyphen",
			input: "if-statement",
			expected: []tokenExpectation{
				{IF, "if", 1, 1},                // keyword 'if'
				{MINUS, "", 1, 3},               // minus operator
				{IDENTIFIER, "statement", 1, 4}, // identifier 'statement'
				{EOF, "", 1, 13},
			},
		},
		{
			name:  "multiple keywords in identifiers",
			input: "for_var_if_when",
			expected: []tokenExpectation{
				{IDENTIFIER, "for_var_if_when", 1, 1}, // All one identifier
				{EOF, "", 1, 16},
			},
		},
		{
			name:  "keyword-like with different casing",
			input: "Variable ForEach IfElse",
			expected: []tokenExpectation{
				{IDENTIFIER, "Variable", 1, 1},
				{IDENTIFIER, "ForEach", 1, 10},
				{IDENTIFIER, "IfElse", 1, 18},
				{EOF, "", 1, 24},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestKeywordPositioning tests accurate position tracking for keywords
func TestKeywordPositioning(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		keyword      TokenType
		expectedLine int
		expectedCol  int
	}{
		{name: "var at start", input: "var", keyword: VAR, expectedLine: 1, expectedCol: 1},
		{name: "var after spaces", input: "   var", keyword: VAR, expectedLine: 1, expectedCol: 4},
		{name: "for after tab", input: "\tfor", keyword: FOR, expectedLine: 1, expectedCol: 2},
		{name: "if second line", input: "deploy:\nif", keyword: IF, expectedLine: 2, expectedCol: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newTestLexer(tt.input)

			// Find the keyword token
			for {
				token := lexer.NextToken()
				if token.Type == tt.keyword {
					if token.Position.Line != tt.expectedLine {
						t.Errorf("Expected line %d, got %d", tt.expectedLine, token.Position.Line)
					}
					if token.Position.Column != tt.expectedCol {
						t.Errorf("Expected column %d, got %d", tt.expectedCol, token.Position.Column)
					}
					return
				}
				if token.Type == EOF {
					t.Errorf("Never found %v token in input %q", tt.keyword, tt.input)
					return
				}
			}
		})
	}
}

func TestQuestionToken(t *testing.T) {
	assertTokens(t, "question token", "Type?", []tokenExpectation{
		{IDENTIFIER, "Type", 1, 1},
		{QUESTION, "", 1, 5},
		{EOF, "", 1, 6},
	})
}
