package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// tokenExpectation represents expected token with type and value
type tokenExpectation struct {
	Type  TokenType
	Value string
}

// assertTokens compares actual tokens with expected, providing clear error messages
func assertTokens(t *testing.T, name string, input string, expected []tokenExpectation) {
	t.Helper()

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// Convert tokens to comparable format (excluding positions)
	actualComp := tokensToComparableNoPos(tokens)
	expectedComp := expectationsToComparableNoPos(expected)

	// Use cmp.Diff for clean output
	if diff := cmp.Diff(expectedComp, actualComp); diff != "" {
		t.Errorf("\n%s: token mismatch (-want +got):\n%s", name, diff)

		// Only show analysis for specific known issues
		if len(tokens) != len(expected) {
			t.Logf("\nToken count: expected %d, got %d", len(expected), len(tokens))
		}

		// Show input for context
		if strings.Contains(diff, "SHELL_TEXT") || strings.Contains(diff, "IDENTIFIER") {
			t.Logf("\nInput: %q", input)

			// Brief analysis of the likely issue
			if strings.Contains(input, "@") && strings.Contains(diff, "npm") {
				t.Logf("Issue: Shell text after decorator { is being parsed as IDENTIFIERs")
			} else if strings.Contains(diff, "build:") {
				t.Logf("Issue: Colon in 'build:prod' treated as structural token")
			}
		}
		return
	}

	// Position validation (only if tokens match)
	for i, tok := range tokens {
		if tok.Line <= 0 || (tok.Column <= 0) {
			// NEWLINE tokens no longer exist
			t.Errorf("%s: token[%d] %s has invalid position: %d:%d",
				name, i, tok.Type, tok.Line, tok.Column)
		}
	}
}

// Helper to convert tokens to comparable format without positions
func tokensToComparableNoPos(tokens []Token) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tokens))
	for i, tok := range tokens {
		result[i] = map[string]interface{}{
			"type":  tok.Type.String(),
			"value": tok.Value,
		}
	}
	return result
}

// Helper to convert expectations to comparable format without positions
func expectationsToComparableNoPos(expected []tokenExpectation) []map[string]interface{} {
	result := make([]map[string]interface{}, len(expected))
	for i, exp := range expected {
		result[i] = map[string]interface{}{
			"type":  exp.Type.String(),
			"value": exp.Value,
		}
	}
	return result
}

func TestCoreStructure(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable declaration",
			input: `var PORT = 8080`,
			expected: []tokenExpectation{
				{VAR, "var"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"},
				{EOF, ""},
			},
		},
		{
			name:  "simple command",
			input: `build: echo hello`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello"},
				{EOF, ""},
			},
		},
		{
			name:  "watch command",
			input: `watch server: node app.js`,
			expected: []tokenExpectation{
				{WATCH, "watch"},
				{IDENTIFIER, "server"},
				{COLON, ":"},
				{SHELL_TEXT, "node app.js"},
				{EOF, ""},
			},
		},
		{
			name:  "stop command",
			input: `stop server: pkill node`,
			expected: []tokenExpectation{
				{STOP, "stop"},
				{IDENTIFIER, "server"},
				{COLON, ":"},
				{SHELL_TEXT, "pkill node"},
				{EOF, ""},
			},
		},
		{
			name:  "command with block",
			input: `deploy: { npm run build; npm run deploy }`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build; npm run deploy"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name:  "decorator with arguments",
			input: `@timeout(30s)`,
			expected: []tokenExpectation{
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "30s"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "grouped variables",
			input: "var (\n  PORT = 8080\n  HOST = \"localhost\"\n)",
			expected: []tokenExpectation{
				{VAR, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"},
				{IDENTIFIER, "HOST"},
				{EQUALS, "="},
				{STRING, "localhost"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestLiteralTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "string types",
			input: `"double" 'single' ` + "`backtick`",
			expected: []tokenExpectation{
				{STRING, "double"},
				{STRING, "single"},
				{STRING, "backtick"},
				{EOF, ""},
			},
		},
		{
			name:  "number types",
			input: `42 3.14 -100 0.5`,
			expected: []tokenExpectation{
				{NUMBER, "42"},
				{NUMBER, "3.14"},
				{NUMBER, "-100"},
				{NUMBER, "0.5"},
				{EOF, ""},
			},
		},
		{
			name:  "duration types",
			input: `30s 5m 1h 500ms 2.5s`,
			expected: []tokenExpectation{
				{DURATION, "30s"},
				{DURATION, "5m"},
				{DURATION, "1h"},
				{DURATION, "500ms"},
				{DURATION, "2.5s"},
				{EOF, ""},
			},
		},
		{
			name:  "boolean types",
			input: `true false`,
			expected: []tokenExpectation{
				{BOOLEAN, "true"},
				{BOOLEAN, "false"},
				{EOF, ""},
			},
		},
		{
			name:  "boolean vs identifier",
			input: `var truename = true`,
			expected: []tokenExpectation{
				{VAR, "var"},
				{IDENTIFIER, "truename"},
				{EQUALS, "="},
				{BOOLEAN, "true"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestShellContentHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "shell with semicolons",
			input: `build: echo hello; echo world`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello; echo world"},
				{EOF, ""},
			},
		},
		{
			name:  "shell with pipes",
			input: `process: cat file | grep pattern | sort`,
			expected: []tokenExpectation{
				{IDENTIFIER, "process"},
				{COLON, ":"},
				{SHELL_TEXT, "cat file | grep pattern | sort"},
				{EOF, ""},
			},
		},
		{
			name:  "shell with logical operators",
			input: `deploy: npm build && npm test || exit 1`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{SHELL_TEXT, "npm build && npm test || exit 1"},
				{EOF, ""},
			},
		},
		{
			name:  "shell with redirections",
			input: `log: tail -f app.log > output.txt 2>&1`,
			expected: []tokenExpectation{
				{IDENTIFIER, "log"},
				{COLON, ":"},
				{SHELL_TEXT, "tail -f app.log > output.txt 2>&1"},
				{EOF, ""},
			},
		},
		{
			name: "multi-line shell in block",
			input: `test: {
    echo "line1"
    echo "line2"
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo \"line1\""},
				{SHELL_TEXT, "echo \"line2\""},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestLineContinuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name: "basic line continuation",
			input: `build: echo hello \
world`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello world"}, // Continuation merged with space
				{EOF, ""},
			},
		},
		{
			name: "multiple continuations",
			input: `build: echo hello \
beautiful \
world`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello beautiful world"},
				{EOF, ""},
			},
		},
		{
			name: "continuation in block",
			input: `build: {
    echo hello \
    world
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo hello world"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "continuation in quoted string",
			input: `build: echo 'hello \
world'`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo 'hello \\\nworld'"}, // Preserved in quotes
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestPatternDecorators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name: "simple when pattern with newlines",
			input: `deploy: @when(ENV) {
  prod: echo production
  dev: echo development
  *: echo default
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{WHEN, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{SHELL_TEXT, "echo production"},
				{IDENTIFIER, "dev"},
				{COLON, ":"},
				{SHELL_TEXT, "echo development"},
				{ASTERISK, "*"},
				{COLON, ":"},
				{SHELL_TEXT, "echo default"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "when pattern with explicit blocks",
			input: `deploy: @when(ENV) {
  prod: { npm run build && npm run deploy }
  dev: npm run dev-deploy
  *: { echo "Unknown env: $ENV"; exit 1 }
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{WHEN, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build && npm run deploy"},
				{RBRACE, "}"},
				{IDENTIFIER, "dev"},
				{COLON, ":"},
				{SHELL_TEXT, "npm run dev-deploy"},
				{ASTERISK, "*"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo \"Unknown env: $ENV\"; exit 1"},
				{RBRACE, "}"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "try pattern",
			input: `test: @try {
  main: npm test
  error: echo "failed"
  finally: echo "done"
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{AT, "@"},
				{TRY, "try"},
				{LBRACE, "{"},
				{IDENTIFIER, "main"},
				{COLON, ":"},
				{SHELL_TEXT, "npm test"},
				{IDENTIFIER, "error"},
				{COLON, ":"},
				{SHELL_TEXT, "echo \"failed\""},
				{IDENTIFIER, "finally"},
				{COLON, ":"},
				{SHELL_TEXT, "echo \"done\""},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "nested decorators in patterns",
			input: `deploy: @when(ENV) {
  prod: @timeout(60s) { deploy prod }
  dev: @timeout(30s) { deploy dev }
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{WHEN, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "60s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "deploy prod"},
				{RBRACE, "}"},
				{IDENTIFIER, "dev"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "30s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "deploy dev"},
				{RBRACE, "}"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "pattern with multiple decorators per branch",
			input: `build: @when(STAGE) {
  prod: @timeout(60s) { npm run build:prod }
  dev: @retry(3) { npm run build:dev }
  test: @parallel {
    npm run build:test
    npm run lint
  }
  *: npm run build
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{AT, "@"},
				{WHEN, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "STAGE"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "60s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build:prod"},
				{RBRACE, "}"},
				{IDENTIFIER, "dev"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "retry"},
				{LPAREN, "("},
				{NUMBER, "3"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build:dev"},
				{RBRACE, "}"},
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "parallel"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build:test"},
				{SHELL_TEXT, "npm run lint"},
				{RBRACE, "}"},
				{ASTERISK, "*"},
				{COLON, ":"},
				{SHELL_TEXT, "npm run build"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "when pattern with multi-line blocks",
			input: `server: @when(NODE_ENV) {
  production: {
    npm run build:prod
    npm run deploy
  }
  development: {
    npm run build:dev
    npm start
  }
  *: echo "Unknown environment"
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "server"},
				{COLON, ":"},
				{AT, "@"},
				{WHEN, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "NODE_ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "production"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build:prod"},
				{SHELL_TEXT, "npm run deploy"},
				{RBRACE, "}"},
				{IDENTIFIER, "development"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build:dev"},
				{SHELL_TEXT, "npm start"},
				{RBRACE, "}"},
				{ASTERISK, "*"},
				{COLON, ":"},
				{SHELL_TEXT, "echo \"Unknown environment\""},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "try pattern with nested decorators",
			input: `test: @try {
  main: @timeout(30s) { npm run build }
  error: @retry(3) { echo "Retrying..." }
  finally: echo "Done"
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{AT, "@"},
				{TRY, "try"},
				{LBRACE, "{"},
				{IDENTIFIER, "main"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "30s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build"},
				{RBRACE, "}"},
				{IDENTIFIER, "error"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "retry"},
				{LPAREN, "("},
				{NUMBER, "3"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo \"Retrying...\""},
				{RBRACE, "}"},
				{IDENTIFIER, "finally"},
				{COLON, ":"},
				{SHELL_TEXT, "echo \"Done\""},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "pattern blocks work correctly without NEWLINE tokens",
			input: `deploy: @when(ENV) {
  prod: echo prod
  dev: echo dev
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{WHEN, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{SHELL_TEXT, "echo prod"},
				{IDENTIFIER, "dev"},
				{COLON, ":"},
				{SHELL_TEXT, "echo dev"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First run the standard assertion
			assertTokens(t, tt.name, tt.input, tt.expected)

			// NEWLINE tokens no longer exist - validation not needed
			// Only check tests that actually have pattern decorators
			if strings.Contains(tt.input, "@when") || strings.Contains(tt.input, "@try") {
				lexer := New(tt.input)
				tokens := lexer.TokenizeToSlice()

				inPatternBlock := false
				braceDepth := 0

				for i, tok := range tokens {
					// Track when we enter a pattern block
					if i > 0 && (tokens[i-1].Type == WHEN || tokens[i-1].Type == TRY) && tok.Type == LBRACE {
						inPatternBlock = true
						braceDepth = 1
					} else if inPatternBlock {
						// Track brace nesting
						switch tok.Type {
						case LBRACE:
							braceDepth++
						case RBRACE:
							braceDepth--
							if braceDepth == 0 {
								inPatternBlock = false
							}
						}
					}

					// NEWLINE tokens no longer exist - no need to check
				}
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "empty input",
			input: "",
			expected: []tokenExpectation{
				{EOF, ""},
			},
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
			expected: []tokenExpectation{
				{EOF, ""},
			},
		},
		{
			name:  "comment only",
			input: "# comment",
			expected: []tokenExpectation{
				{COMMENT, "# comment"},
				{EOF, ""},
			},
		},
		{
			name:  "empty command",
			input: "empty:",
			expected: []tokenExpectation{
				{IDENTIFIER, "empty"},
				{COLON, ":"},
				{EOF, ""},
			},
		},
		{
			name:  "empty block",
			input: "empty: { }",
			expected: []tokenExpectation{
				{IDENTIFIER, "empty"},
				{COLON, ":"},
				{LBRACE, "{"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use modified assertTokens that handles position edge cases
			lexer := New(tt.input)
			tokens := lexer.TokenizeToSlice()

			// Check token count
			if len(tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			// Check each token
			for i, exp := range tt.expected {
				actual := tokens[i]

				if actual.Type != exp.Type {
					t.Errorf("Token[%d]: expected type %s, got %s", i, exp.Type, actual.Type)
				}

				if actual.Value != exp.Value {
					t.Errorf("Token[%d]: expected value %q, got %q", i, exp.Value, actual.Value)
				}

				// Special position handling for edge cases
				if actual.Line <= 0 || (actual.Column <= 0 && actual.Type != EOF) {
					t.Errorf("Token[%d] has invalid position: %d:%d",
						i, actual.Line, actual.Column)
				}
			}
		})
	}
}

func TestModeTransitions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "language to command mode",
			input: "build: echo hello",
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},      // LanguageMode
				{COLON, ":"},               // LanguageMode → CommandMode
				{SHELL_TEXT, "echo hello"}, // CommandMode
				{EOF, ""},                  // LanguageMode
			},
		},
		{
			name:  "decorator in command mode",
			input: "build: @timeout(30s) { echo hello }",
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},      // LanguageMode
				{COLON, ":"},               // LanguageMode
				{AT, "@"},                  // CommandMode → LanguageMode
				{IDENTIFIER, "timeout"},    // LanguageMode
				{LPAREN, "("},              // LanguageMode
				{DURATION, "30s"},          // LanguageMode
				{RPAREN, ")"},              // LanguageMode
				{LBRACE, "{"},              // LanguageMode → CommandMode
				{SHELL_TEXT, "echo hello"}, // CommandMode
				{RBRACE, "}"},              // CommandMode → LanguageMode
				{EOF, ""},                  // LanguageMode
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "single line comment",
			input: "# This is a comment",
			expected: []tokenExpectation{
				{COMMENT, "# This is a comment"},
				{EOF, ""},
			},
		},
		{
			name:  "multi-line comment",
			input: "/* This is\na multi-line\ncomment */",
			expected: []tokenExpectation{
				{MULTILINE_COMMENT, "/* This is\na multi-line\ncomment */"},
				{EOF, ""},
			},
		},
		{
			name:  "comment after command",
			input: "build: echo hello # comment",
			expected: []tokenExpectation{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello # comment"}, // Comment is part of shell
				{EOF, ""},
			},
		},
		{
			name:  "comment at start followed by var declaration",
			input: "# Data science project development\nvar PYTHON = \"python3\"",
			expected: []tokenExpectation{
				{COMMENT, "# Data science project development"},
				{VAR, "var"},
				{IDENTIFIER, "PYTHON"},
				{EQUALS, "="},
				{STRING, "python3"},
				{EOF, ""},
			},
		},
		{
			name:  "shell command with hash in URL",
			input: "fetch: curl https://example.com#anchor",
			expected: []tokenExpectation{
				{IDENTIFIER, "fetch"},
				{COLON, ":"},
				{SHELL_TEXT, "curl https://example.com#anchor"}, // # should be part of shell text
				{EOF, ""},
			},
		},
		{
			name:  "shell command with git issue reference",
			input: "commit: git commit -m \"Fix issue #123\"",
			expected: []tokenExpectation{
				{IDENTIFIER, "commit"},
				{COLON, ":"},
				{SHELL_TEXT, "git commit -m \"Fix issue #123\""}, // # should be part of shell text
				{EOF, ""},
			},
		},
		{
			name:  "reproduce failing example pattern",
			input: "var PYTHON = \"python3\"\n# Data science project development\nsetup: echo \"Setting up...\"",
			expected: []tokenExpectation{
				{VAR, "var"},
				{IDENTIFIER, "PYTHON"},
				{EQUALS, "="},
				{STRING, "python3"},
				{COMMENT, "# Data science project development"},
				{IDENTIFIER, "setup"},
				{COLON, ":"},
				{SHELL_TEXT, "echo \"Setting up...\""},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestTokenPositions verifies accurate position tracking for each token
func TestTokenPositions(t *testing.T) {
	input := `var PORT = 8080
build: echo hello`

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// Expected positions (1-based indexing)
	expectedPositions := []struct {
		tokenType TokenType
		line      int
		column    int
		value     string
	}{
		{VAR, 1, 1, "var"},         // 'var' starts at column 1
		{IDENTIFIER, 1, 5, "PORT"}, // 'PORT' starts at column 5
		{EQUALS, 1, 10, "="},       // '=' at column 10
		{NUMBER, 1, 12, "8080"},    // '8080' starts at column 12
		// NEWLINE token removed from position tests
		{IDENTIFIER, 2, 1, "build"},      // 'build' starts at line 2, column 1
		{COLON, 2, 6, ":"},               // ':' at column 6
		{SHELL_TEXT, 2, 8, "echo hello"}, // Shell text starts at column 8
		{EOF, 2, 18, ""},                 // EOF position (CORRECTED from 19)
	}

	// Convert to comparable format
	actualComp := make([]map[string]interface{}, len(tokens))
	expectedComp := make([]map[string]interface{}, len(expectedPositions))

	for i, tok := range tokens {
		actualComp[i] = map[string]interface{}{
			"type":   tok.Type.String(),
			"value":  tok.Value,
			"line":   tok.Line,
			"column": tok.Column,
		}
	}

	for i, exp := range expectedPositions {
		// NEWLINE special case removed
		column := exp.column

		expectedComp[i] = map[string]interface{}{
			"type":   exp.tokenType.String(),
			"value":  exp.value,
			"line":   exp.line,
			"column": column,
		}
	}

	if diff := cmp.Diff(expectedComp, actualComp); diff != "" {
		t.Errorf("Token positions mismatch (-want +got):\n%s", diff)

		// Note: NEWLINE token issues no longer exist
	}
}

func TestVarInShellText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple @var in shell text",
			input: `test: cd @var(DIR)`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "cd @var(DIR)"},
				{EOF, ""},
			},
		},
		{
			name:  "@var with surrounding spaces",
			input: `test: cd @var(DIR) && pwd`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "cd @var(DIR) && pwd"},
				{EOF, ""},
			},
		},
		{
			name:  "multiple @var in single command",
			input: `test: echo @var(FIRST) @var(SECOND)`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "echo @var(FIRST) @var(SECOND)"},
				{EOF, ""},
			},
		},
		{
			name:  "@var in quoted string",
			input: `test: echo "Hello @var(NAME)"`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, `echo "Hello @var(NAME)"`},
				{EOF, ""},
			},
		},
		{
			name:  "@var with shell operators",
			input: `test: cat @var(FILE) | grep pattern`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "cat @var(FILE) | grep pattern"},
				{EOF, ""},
			},
		},
		{
			name:  "@var in block command",
			input: `test: { cd @var(DIR); make }`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "cd @var(DIR); make"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name: "@var with line continuation",
			input: `test: cd @var(DIR) \
&& make`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "cd @var(DIR) && make"},
				{EOF, ""},
			},
		},
		{
			name:  "@env in shell text",
			input: `test: echo @env("NODE_ENV")`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, `echo @env("NODE_ENV")`},
				{EOF, ""},
			},
		},
		{
			name:  "mixed @var and @env",
			input: `test: @var(CMD) --env=@env("ENV_VAR")`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, `@var(CMD) --env=@env("ENV_VAR")`},
				{EOF, ""},
			},
		},
		{
			name:  "@var in shell parameter expansion",
			input: `test: echo ${@var(VAR):-default}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "echo ${@var(VAR):-default}"},
				{EOF, ""},
			},
		},
		{
			name:  "@var in command substitution",
			input: `test: echo $(ls @var(DIR) | wc -l)`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "echo $(ls @var(DIR) | wc -l)"},
				{EOF, ""},
			},
		},
		{
			name:  "@var at start of shell text",
			input: `test: @var(CMD) arg1 arg2`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "@var(CMD) arg1 arg2"},
				{EOF, ""},
			},
		},
		{
			name:  "@var at end of shell text",
			input: `test: cd /path/to/@var(DIR)`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "cd /path/to/@var(DIR)"},
				{EOF, ""},
			},
		},
		{
			name:  "@ not followed by var should be literal",
			input: `test: echo user@host.com`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "echo user@host.com"},
				{EOF, ""},
			},
		},
		{
			name:  "@var without parentheses is literal",
			input: `test: echo @var is not a decorator`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{SHELL_TEXT, "echo @var is not a decorator"},
				{EOF, ""},
			},
		},
		{
			name:  "complex shell with multiple @var",
			input: `deploy: cd @var(SRC) && npm run build:@var(ENV) && cp -r dist/* @var(DEST)/`,
			expected: []tokenExpectation{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{SHELL_TEXT, "cd @var(SRC) && npm run build:@var(ENV) && cp -r dist/* @var(DEST)/"},
				{EOF, ""},
			},
		},
		{
			name: "multi-line block with @var",
			input: `test: {
    echo "Building @var(APP)"
    cd @var(DIR)
    make @var(TARGET)
}`,
			expected: []tokenExpectation{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, `echo "Building @var(APP)"`},
				{SHELL_TEXT, "cd @var(DIR)"},
				{SHELL_TEXT, "make @var(TARGET)"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name:  "@var with double @ in shell text",
			input: `connect: ssh -p @var(PORT) user@@var(HOST)`,
			expected: []tokenExpectation{
				{IDENTIFIER, "connect"},
				{COLON, ":"},
				{SHELL_TEXT, "ssh -p @var(PORT) user@@var(HOST)"},
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// Benchmark for performance validation
func BenchmarkLexer(b *testing.B) {
	input := generateLargeInput(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(input)
		tokens := lexer.TokenizeToSlice()
		_ = tokens
	}
}

// generateLargeInput creates a realistic Devcmd file for performance testing
func generateLargeInput(lines int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		switch i % 4 {
		case 0:
			fmt.Fprintf(&sb, "var VAR%d = \"value%d\"\n", i, i)
		case 1:
			fmt.Fprintf(&sb, "cmd%d: echo hello %d\n", i, i)
		case 2:
			fmt.Fprintf(&sb, "build%d: @timeout(30s) { npm run build:%d }\n", i, i)
		default:
			fmt.Fprintf(&sb, "test%d: @when(ENV) { prod: echo prod %d; dev: echo dev %d }\n", i, i, i)
		}
	}
	return sb.String()
}

// TestRealWorldFormatCommand tests lexing of the failing format command from commands.cli
func TestRealWorldFormatCommand(t *testing.T) {
	input := `# Format all code
format: {
    echo "📝 Formatting all code..."
    echo "Formatting Go code..."
    @parallel {
        if command -v gofumpt >/dev/null 2>&1; then gofumpt -w .; else go fmt ./...; fi
        if command -v nixpkgs-fmt >/dev/null 2>&1; then find . -name '*.nix' -exec nixpkgs-fmt {} +; else echo "⚠️  nixpkgs-fmt not available"; fi
    }
    echo "✅ Code formatted!"
}`

	expected := []tokenExpectation{
		{COMMENT, "# Format all code"},
		{IDENTIFIER, "format"},
		{COLON, ":"},
		{LBRACE, "{"},
		{SHELL_TEXT, "echo \"📝 Formatting all code...\""},
		{SHELL_TEXT, "echo \"Formatting Go code...\""},
		{AT, "@"},
		{IDENTIFIER, "parallel"},
		{LBRACE, "{"},
		{SHELL_TEXT, "if command -v gofumpt >/dev/null 2>&1; then gofumpt -w .; else go fmt ./...; fi"},
		{SHELL_TEXT, "if command -v nixpkgs-fmt >/dev/null 2>&1; then find . -name '*.nix' -exec nixpkgs-fmt {} +; else echo \"⚠️  nixpkgs-fmt not available\"; fi"},
		{RBRACE, "}"},
		{SHELL_TEXT, "echo \"✅ Code formatted!\""},
		{RBRACE, "}"},
		{EOF, ""},
	}

	assertTokens(t, "Real world format command", input, expected)
}
