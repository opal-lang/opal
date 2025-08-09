package lexer

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/types"
)

// TestPatternDecoratorEdgeCases focuses on the specific edge cases that are failing
// This will be our TDD test suite for rebuilding the pattern decorator logic
func TestPatternDecoratorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
		failing  bool // Mark which tests are currently failing
	}{
		{
			name: "FAILING: complex nested decorators - mode transition issue",
			input: `test: @retry(attempts=3) {
		@when(ENV) {
			development: echo "Dev environment"
		}
		echo "Always execute"
	}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "attempts"},
				{types.EQUALS, "="},
				{types.NUMBER, "3"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "development"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo \"Dev environment\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.SHELL_TEXT, "echo \"Always execute\""}, // This should be SHELL_TEXT, not broken tokens
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
			failing: true,
		},
		{
			name: "WORKING: simple when pattern",
			input: `deploy: @when(ENV) {
  prod: echo production
  dev: echo development
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "prod"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo production"},
				{types.SHELL_END, ""},
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo development"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
			failing: false,
		},
		{
			name: "TEST CASE: simple shell after pattern - isolated",
			input: `test: {
		@when(ENV) {
			dev: echo hello  
		}
		echo after
	}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo hello"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.SHELL_TEXT, "echo after"},
				{types.SHELL_END, ""}, // This is the key problem area
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
			failing: true, // Assuming this will fail based on similar pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.failing {
				t.Logf("This test is marked as failing - running to document current behavior")
			}

			lexer := New(strings.NewReader(tt.input))
			tokens := lexer.TokenizeToSlice()

			// Show detailed token analysis for failing tests
			if tt.failing {
				t.Logf("Input:\n%s", tt.input)
				t.Logf("Actual tokens:")
				for i, tok := range tokens {
					t.Logf("  [%d] %s: %q", i, tok.Type, tok.Value)
				}
				t.Logf("Expected tokens:")
				for i, exp := range tt.expected {
					t.Logf("  [%d] %s: %q", i, exp.Type, exp.Value)
				}
			}

			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestModeTransitionIssues specifically tests the mode transition logic
func TestModeTransitionIssues(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "pattern_exit_to_command_mode",
			input: `test: {
	@when(ENV) { dev: echo hello }
	echo after_pattern
}`,
			description: "After exiting @when pattern, should return to CommandMode for 'echo after_pattern'",
		},
		{
			name: "nested_pattern_in_block_decorator",
			input: `test: @retry(3) {
	@when(ENV) { dev: echo hello }
	echo after
}`,
			description: "Pattern decorator nested inside block decorator - complex mode transitions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Description: %s", tt.description)
			t.Logf("Input:\n%s", tt.input)

			lexer := New(strings.NewReader(tt.input))
			tokens := lexer.TokenizeToSlice()

			t.Logf("Actual tokens:")
			for i, tok := range tokens {
				t.Logf("  [%d] %s: %q (line %d, col %d)", i, tok.Type, tok.Value, tok.Line, tok.Column)
			}

			// Look for the problematic "echo after" token
			foundShellText := false
			for _, tok := range tokens {
				if tok.Value == "echo after_pattern" || tok.Value == "echo after" {
					if tok.Type == types.SHELL_TEXT {
						foundShellText = true
						t.Logf("✅ Found correct SHELL_TEXT token: %q", tok.Value)
					} else {
						t.Logf("❌ Found token with wrong type %s: %q", tok.Type, tok.Value)
					}
				}
			}

			if !foundShellText {
				t.Logf("❌ Did not find expected shell text token")
			}
		})
	}
}
