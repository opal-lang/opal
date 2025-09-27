package v2

import "testing"

// TestHasSpaceBefore tests that tokens correctly track if whitespace preceded them
func TestHasSpaceBefore(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectationWithSpace
	}{
		{
			name:  "no initial whitespace",
			input: "deploy",
			expected: []tokenExpectationWithSpace{
				{IDENTIFIER, "deploy", 1, 1, false}, // No space before first token
				{EOF, "", 1, 7, false},
			},
		},
		{
			name:  "whitespace between tokens",
			input: "deploy kubectl",
			expected: []tokenExpectationWithSpace{
				{IDENTIFIER, "deploy", 1, 1, false}, // No space before first token
				{IDENTIFIER, "kubectl", 1, 8, true}, // Space before second token
				{EOF, "", 1, 15, false},
			},
		},
		{
			name:  "shell flag without space",
			input: "kubectl --replicas",
			expected: []tokenExpectationWithSpace{
				{IDENTIFIER, "kubectl", 1, 1, false},   // No space before first token
				{DECREMENT, "", 1, 9, true},            // Space before -- (no text for punctuation)
				{IDENTIFIER, "replicas", 1, 11, false}, // No space after --
				{EOF, "", 1, 19, false},
			},
		},
		{
			name:  "command with colon",
			input: "deploy: kubectl",
			expected: []tokenExpectationWithSpace{
				{IDENTIFIER, "deploy", 1, 1, false}, // No space before first token
				{COLON, "", 1, 7, false},            // No space before colon (no text for punctuation)
				{IDENTIFIER, "kubectl", 1, 9, true}, // Space before kubectl
				{EOF, "", 1, 16, false},
			},
		},
		{
			name:  "multiple spaces treated as one",
			input: "deploy   kubectl",
			expected: []tokenExpectationWithSpace{
				{IDENTIFIER, "deploy", 1, 1, false},  // No space before first token
				{IDENTIFIER, "kubectl", 1, 10, true}, // Multiple spaces = true
				{EOF, "", 1, 17, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokensWithWhitespace(t, tt.name, tt.input, tt.expected)
		})
	}
}

// tokenExpectationWithSpace includes hasSpaceBefore expectation
type tokenExpectationWithSpace struct {
	tokenType      TokenType
	text           string
	line           int
	column         int
	hasSpaceBefore bool
}

// assertTokensWithWhitespace validates tokens including HasSpaceBefore flag
func assertTokensWithWhitespace(t *testing.T, testName, input string, expected []tokenExpectationWithSpace) {
	lexer := NewLexer(input)
	tokens := lexer.GetTokens()

	if len(tokens) != len(expected) {
		t.Fatalf("%s: expected %d tokens, got %d", testName, len(expected), len(tokens))
	}

	for i, exp := range expected {
		token := tokens[i]

		if token.Type != exp.tokenType {
			t.Errorf("%s[%d]: expected type %v, got %v", testName, i, exp.tokenType, token.Type)
		}

		if exp.text != "" && string(token.Text) != exp.text {
			t.Errorf("%s[%d]: expected text %q, got %q", testName, i, exp.text, string(token.Text))
		}

		if token.Position.Line != exp.line {
			t.Errorf("%s[%d]: expected line %d, got %d", testName, i, exp.line, token.Position.Line)
		}

		if token.Position.Column != exp.column {
			t.Errorf("%s[%d]: expected column %d, got %d", testName, i, exp.column, token.Position.Column)
		}

		// Check HasSpaceBefore flag - this is the new test
		if token.HasSpaceBefore != exp.hasSpaceBefore {
			t.Errorf("%s[%d]: expected HasSpaceBefore %v, got %v for token %v",
				testName, i, exp.hasSpaceBefore, token.HasSpaceBefore, token.Type)
		}
	}
}

// TestShellCommandReconstruction tests reconstructing shell commands from tokens
func TestShellCommandReconstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "shell flag without space",
			input:    "kubectl --replicas 3",
			expected: "kubectl --replicas 3",
		},
		{
			name:     "shell flag with extra spaces",
			input:    "kubectl   --replicas    3",
			expected: "kubectl --replicas 3", // Normalized spacing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := lexer.GetTokens()

			// Remove EOF token for reconstruction
			if len(tokens) > 0 && tokens[len(tokens)-1].Type == EOF {
				tokens = tokens[:len(tokens)-1]
			}

			reconstructed := reconstructShellCommand(tokens)
			if reconstructed != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, reconstructed)
			}
		})
	}
}

// reconstructShellCommand demonstrates how to reconstruct shell commands using HasSpaceBefore
func reconstructShellCommand(tokens []Token) string {
	var result string
	for i, token := range tokens {
		if i > 0 && token.HasSpaceBefore {
			result += " "
		}

		// Get token text - handle punctuation tokens that have Text: nil
		tokenText := getTokenText(token)
		result += tokenText
	}
	return result
}

// getTokenText returns the string representation of a token
func getTokenText(token Token) string {
	if token.Text != nil {
		return string(token.Text)
	}

	// Handle punctuation tokens that have Text: nil
	switch token.Type {
	case DECREMENT:
		return "--"
	case INCREMENT:
		return "++"
	case PLUS:
		return "+"
	case MINUS:
		return "-"
	case MULTIPLY:
		return "*"
	case DIVIDE:
		return "/"
	case MODULO:
		return "%"
	case EQUALS:
		return "="
	case EQ_EQ:
		return "=="
	case NOT_EQ:
		return "!="
	case LT:
		return "<"
	case LT_EQ:
		return "<="
	case GT:
		return ">"
	case GT_EQ:
		return ">="
	case AND_AND:
		return "&&"
	case OR_OR:
		return "||"
	case NOT:
		return "!"
	case COLON:
		return ":"
	case COMMA:
		return ","
	case SEMICOLON:
		return ";"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case LSQUARE:
		return "["
	case RSQUARE:
		return "]"
	case AT:
		return "@"
	case DOT:
		return "."
	default:
		return ""
	}
}
