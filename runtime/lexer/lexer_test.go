package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/types"
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin" // Import for decorator registration

	"github.com/google/go-cmp/cmp"
)

// tokenExpectation represents expected token with type and value
type tokenExpectation struct {
	Type  types.TokenType
	Value string
}

// assertTokens compares actual tokens with expected, providing clear error messages
func assertTokens(t *testing.T, name string, input string, expected []tokenExpectation) {
	t.Helper()

	lexer := New(strings.NewReader(input))
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
		if strings.Contains(diff, "types.SHELL_TEXT") || strings.Contains(diff, "types.IDENTIFIER") {
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
func tokensToComparableNoPos(tokens []types.Token) []map[string]interface{} {
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
				{types.VAR, "var"},
				{types.IDENTIFIER, "PORT"},
				{types.EQUALS, "="},
				{types.NUMBER, "8080"},
				{types.EOF, ""},
			},
		},
		{
			name:  "simple command",
			input: `build: echo hello`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo hello"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "watch command",
			input: `watch server: node app.js`,
			expected: []tokenExpectation{
				{types.WATCH, "watch"},
				{types.IDENTIFIER, "server"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "node app.js"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "stop command",
			input: `stop server: pkill node`,
			expected: []tokenExpectation{
				{types.STOP, "stop"},
				{types.IDENTIFIER, "server"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "pkill node"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "command with block",
			input: `deploy: { npm run build; npm run deploy }`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build; npm run deploy"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:  "decorator with arguments",
			input: `@timeout(30s)`,
			expected: []tokenExpectation{
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.EOF, ""},
			},
		},
		{
			name:  "grouped variables",
			input: "var (\n  PORT = 8080\n  HOST = \"localhost\"\n)",
			expected: []tokenExpectation{
				{types.VAR, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PORT"},
				{types.EQUALS, "="},
				{types.NUMBER, "8080"},
				{types.IDENTIFIER, "HOST"},
				{types.EQUALS, "="},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "localhost"},
				{types.STRING_END, "\""},
				{types.RPAREN, ")"},
				{types.EOF, ""},
			},
		},
		// Real-world decorator use cases
		{
			name:  "simple @workdir block decorator",
			input: `test: @workdir("core") { pwd }`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "workdir"},
				{types.LPAREN, "("},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "core"},
				{types.STRING_END, "\""},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "pwd"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:  "simple @cmd function decorator",
			input: `test: @cmd(base)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "cmd"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "base"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "nested @parallel with @workdir",
			input: `test: @parallel {
    @workdir("core") { go test }
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "workdir"},
				{types.LPAREN, "("},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "core"},
				{types.STRING_END, "\""},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "go test"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:  "@cmd with shell chaining",
			input: `test: @cmd(test-core) && @cmd(test-runtime)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "cmd"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "test-core"},
				{types.RPAREN, ")"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " "},
				{types.AT, "@"},
				{types.IDENTIFIER, "cmd"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "test-runtime"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
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
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "double"},
				{types.STRING_END, "\""},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "single"},
				{types.STRING_END, "'"},
				{types.STRING_START, "`"},
				{types.STRING_TEXT, "backtick"},
				{types.STRING_END, "`"},
				{types.EOF, ""},
			},
		},
		{
			name:  "number types",
			input: `42 3.14 -100 0.5`,
			expected: []tokenExpectation{
				{types.NUMBER, "42"},
				{types.NUMBER, "3.14"},
				{types.NUMBER, "-100"},
				{types.NUMBER, "0.5"},
				{types.EOF, ""},
			},
		},
		{
			name:  "duration types",
			input: `30s 5m 1h 500ms 2.5s`,
			expected: []tokenExpectation{
				{types.DURATION, "30s"},
				{types.DURATION, "5m"},
				{types.DURATION, "1h"},
				{types.DURATION, "500ms"},
				{types.DURATION, "2.5s"},
				{types.EOF, ""},
			},
		},
		{
			name:  "boolean types",
			input: `true false`,
			expected: []tokenExpectation{
				{types.BOOLEAN, "true"},
				{types.BOOLEAN, "false"},
				{types.EOF, ""},
			},
		},
		{
			name:  "boolean vs identifier",
			input: `var truename = true`,
			expected: []tokenExpectation{
				{types.VAR, "var"},
				{types.IDENTIFIER, "truename"},
				{types.EQUALS, "="},
				{types.BOOLEAN, "true"},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestStringQuoteTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "double quoted string",
			input: `"hello world"`,
			expected: []tokenExpectation{
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "hello world"},
				{types.STRING_END, "\""},
				{types.EOF, ""},
			},
		},
		{
			name:  "single quoted string",
			input: `'hello world'`,
			expected: []tokenExpectation{
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "hello world"},
				{types.STRING_END, "'"},
				{types.EOF, ""},
			},
		},
		{
			name:  "backtick string",
			input: "`hello world`",
			expected: []tokenExpectation{
				{types.STRING_START, "`"},
				{types.STRING_TEXT, "hello world"},
				{types.STRING_END, "`"},
				{types.EOF, ""},
			},
		},
		{
			name:  "double quotes with @var should tokenize properly",
			input: `"Building @var(PROJECT)"`,
			expected: []tokenExpectation{
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Building "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PROJECT"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.EOF, ""},
			},
		},
		{
			name:  "single quotes with @var should remain literal",
			input: `'Building @var(PROJECT)'`,
			expected: []tokenExpectation{
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "Building @var(PROJECT)"},
				{types.STRING_END, "'"},
				{types.EOF, ""},
			},
		},
		{
			name:  "backticks with @var should tokenize properly",
			input: "`Building @var(PROJECT)`",
			expected: []tokenExpectation{
				{types.STRING_START, "`"},
				{types.STRING_TEXT, "Building "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PROJECT"},
				{types.RPAREN, ")"},
				{types.STRING_END, "`"},
				{types.EOF, ""},
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
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo hello; echo world"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "shell with pipes",
			input: `process: cat file | grep pattern | sort`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "process"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cat file"},
				{types.PIPE, "|"},
				{types.SHELL_TEXT, "grep pattern"},
				{types.PIPE, "|"},
				{types.SHELL_TEXT, " sort"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "shell with logical operators",
			input: `deploy: npm build && npm test || exit 1`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "npm build"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, "npm test"},
				{types.OR, "||"},
				{types.SHELL_TEXT, " exit 1"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "shell with redirections",
			input: `log: tail -f app.log > output.txt 2>&1`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "log"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "tail -f app.log > output.txt 2>&1"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "multi-line shell in block",
			input: `test: {
    echo "line1"
    echo "line2"
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "line1"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "line2"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
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
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo hello world"}, // Continuation merged with space
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "multiple continuations",
			input: `build: echo hello \
beautiful \
world`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo hello beautiful world"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "continuation in block",
			input: `build: {
    echo hello \
    world
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo hello world"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "continuation in single quoted string (preserved)",
			input: `build: echo 'hello \
world'`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "hello \\\nworld"}, // Preserved in single quotes
				{types.STRING_END, "'"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "continuation in double quoted string (processed)",
			input: `build: echo "hello \
world"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "hello \\\nworld"}, // Raw content preserved in STRING_TEXT
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "continuation in backtick string (processed)",
			input: "build: echo `hello \\\nworld`",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "`"},
				{types.STRING_TEXT, "hello \\\nworld"}, // Raw content preserved in STRING_TEXT
				{types.STRING_END, "`"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
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
  default: echo default
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
				{types.IDENTIFIER, "default"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo default"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "when pattern with explicit blocks",
			input: `deploy: @when(ENV) {
  prod: { npm run build && npm run deploy }
  dev: npm run dev-deploy
  default: { echo "Unknown env: $ENV"; exit 1 }
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
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " npm run deploy"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "npm run dev-deploy"},
				{types.SHELL_END, ""},
				{types.IDENTIFIER, "default"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Unknown env: $ENV"},
				{types.STRING_END, "\""},
				{types.SHELL_TEXT, "; exit 1"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
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
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "try"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "main"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "npm test"},
				{types.SHELL_END, ""},
				{types.IDENTIFIER, "error"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "failed"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.IDENTIFIER, "finally"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "done"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "nested decorators in patterns",
			input: `deploy: @when(ENV) {
  prod: @timeout(60s) { deploy prod }
  dev: @timeout(30s) { deploy dev }
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
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "60s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "deploy prod"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "deploy dev"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
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
  default: npm run build
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "STAGE"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "prod"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "60s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build:prod"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.NUMBER, "3"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build:dev"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build:test"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run lint"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "default"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "npm run build"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
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
  default: echo "Unknown environment"
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "server"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NODE_ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "production"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build:prod"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run deploy"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "development"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build:dev"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm start"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "default"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Unknown environment"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
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
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "try"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "main"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "error"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.NUMBER, "3"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Retrying..."},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.IDENTIFIER, "finally"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Done"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "pattern blocks work correctly without NEWLINE tokens",
			input: `deploy: @when(ENV) {
  prod: echo prod
  dev: echo dev
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
				{types.SHELL_TEXT, "echo prod"},
				{types.SHELL_END, ""},
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo dev"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
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
				lexer := New(strings.NewReader(tt.input))
				tokens := lexer.TokenizeToSlice()

				inPatternBlock := false
				braceDepth := 0

				for i, tok := range tokens {
					// Track when we enter a pattern block
					if i > 0 && tokens[i-1].Type == types.IDENTIFIER && tok.Type == types.LBRACE {
						inPatternBlock = true
						braceDepth = 1
					} else if inPatternBlock {
						// Track brace nesting
						switch tok.Type {
						case types.LBRACE:
							braceDepth++
						case types.RBRACE:
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
				{types.EOF, ""},
			},
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
			expected: []tokenExpectation{
				{types.EOF, ""},
			},
		},
		{
			name:  "comment only",
			input: "# comment",
			expected: []tokenExpectation{
				{types.COMMENT, "# comment"},
				{types.EOF, ""},
			},
		},
		{
			name:  "empty command",
			input: "empty:",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "empty"},
				{types.COLON, ":"},
				{types.EOF, ""},
			},
		},
		{
			name:  "empty block",
			input: "empty: { }",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "empty"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use modified assertTokens that handles position edge cases
			lexer := New(strings.NewReader(tt.input))
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
					t.Errorf("types.Token[%d]: expected type %s, got %s", i, exp.Type, actual.Type)
				}

				if actual.Value != exp.Value {
					t.Errorf("types.Token[%d]: expected value %q, got %q", i, exp.Value, actual.Value)
				}

				// Special position handling for edge cases
				if actual.Line <= 0 || (actual.Column <= 0 && actual.Type != types.EOF) {
					t.Errorf("types.Token[%d] has invalid position: %d:%d",
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
				{types.IDENTIFIER, "build"},      // LanguageMode
				{types.COLON, ":"},               // LanguageMode → CommandMode
				{types.SHELL_TEXT, "echo hello"}, // CommandMode
				{types.SHELL_END, ""},            // End of shell command
				{types.EOF, ""},                  // LanguageMode
			},
		},
		{
			name:  "decorator in command mode",
			input: "build: @timeout(30s) { echo hello }",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},      // LanguageMode
				{types.COLON, ":"},               // LanguageMode
				{types.AT, "@"},                  // CommandMode → LanguageMode
				{types.IDENTIFIER, "timeout"},    // LanguageMode
				{types.LPAREN, "("},              // LanguageMode
				{types.DURATION, "30s"},          // LanguageMode
				{types.RPAREN, ")"},              // LanguageMode
				{types.LBRACE, "{"},              // LanguageMode → CommandMode
				{types.SHELL_TEXT, "echo hello"}, // CommandMode
				{types.SHELL_END, ""},            // End of shell command
				{types.RBRACE, "}"},              // CommandMode → LanguageMode
				{types.EOF, ""},                  // LanguageMode
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
				{types.COMMENT, "# This is a comment"},
				{types.EOF, ""},
			},
		},
		{
			name:  "multi-line comment",
			input: "/* This is\na multi-line\ncomment */",
			expected: []tokenExpectation{
				{types.MULTILINE_COMMENT, "/* This is\na multi-line\ncomment */"},
				{types.EOF, ""},
			},
		},
		{
			name:  "comment after command",
			input: "build: echo hello # comment",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo hello # comment"}, // Comment is part of shell
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "comment at start followed by var declaration",
			input: "# Data science project development\nvar PYTHON = \"python3\"",
			expected: []tokenExpectation{
				{types.COMMENT, "# Data science project development"},
				{types.VAR, "var"},
				{types.IDENTIFIER, "PYTHON"},
				{types.EQUALS, "="},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "python3"},
				{types.STRING_END, "\""},
				{types.EOF, ""},
			},
		},
		{
			name:  "shell command with hash in URL",
			input: "fetch: curl https://example.com#anchor",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "fetch"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "curl https://example.com#anchor"}, // # should be part of shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "shell command with git issue reference",
			input: "commit: git commit -m \"Fix issue #123\"",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "commit"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "git commit -m "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Fix issue #123"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "reproduce failing example pattern",
			input: "var PYTHON = \"python3\"\n# Data science project development\nsetup: echo \"Setting up...\"",
			expected: []tokenExpectation{
				{types.VAR, "var"},
				{types.IDENTIFIER, "PYTHON"},
				{types.EQUALS, "="},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "python3"},
				{types.STRING_END, "\""},
				{types.COMMENT, "# Data science project development"},
				{types.IDENTIFIER, "setup"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Setting up..."},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
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

	lexer := New(strings.NewReader(input))
	tokens := lexer.TokenizeToSlice()

	// Expected positions (1-based indexing)
	expectedPositions := []struct {
		tokenType types.TokenType
		line      int
		column    int
		value     string
	}{
		{types.VAR, 1, 1, "var"},         // 'var' starts at column 1
		{types.IDENTIFIER, 1, 5, "PORT"}, // 'PORT' starts at column 5
		{types.EQUALS, 1, 10, "="},       // '=' at column 10
		{types.NUMBER, 1, 12, "8080"},    // '8080' starts at column 12
		// NEWLINE token removed from position tests
		{types.IDENTIFIER, 2, 1, "build"},      // 'build' starts at line 2, column 1
		{types.COLON, 2, 6, ":"},               // ':' at column 6
		{types.SHELL_TEXT, 2, 8, "echo hello"}, // Shell text starts at column 8
		{types.SHELL_END, 2, 18, ""},           // SHELL_END marks end of shell command
		{types.EOF, 2, 18, ""},                 // types.EOF position (CORRECTED from 19)
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
		t.Errorf("types.Token positions mismatch (-want +got):\n%s", diff)

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
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cd "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var with surrounding spaces",
			input: `test: cd @var(DIR) && pwd`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cd "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " pwd"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "multiple @var in single command",
			input: `test: echo @var(FIRST) @var(SECOND)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "FIRST"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, " "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "SECOND"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var in quoted string",
			input: `test: echo "Hello @var(NAME)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Hello "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NAME"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var with shell operators",
			input: `test: cat @var(FILE) | grep pattern`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cat "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "FILE"},
				{types.RPAREN, ")"},
				{types.PIPE, "|"},
				{types.SHELL_TEXT, " grep pattern"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var in block command",
			input: `test: { cd @var(DIR); make }`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "cd "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, "; make"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "@var with line continuation",
			input: `test: cd @var(DIR) \
&& make`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cd "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " make"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@env in shell text",
			input: `test: echo @env("NODE_ENV")`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.AT, "@"},
				{types.IDENTIFIER, "env"},
				{types.LPAREN, "("},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "NODE_ENV"},
				{types.STRING_END, "\""},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "mixed @var and @env",
			input: `test: @var(CMD) --env=@env("ENV_VAR")`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "CMD"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, " --env="},
				{types.AT, "@"},
				{types.IDENTIFIER, "env"},
				{types.LPAREN, "("},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "ENV_VAR"},
				{types.STRING_END, "\""},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var in shell parameter expansion",
			input: `test: echo ${@var(VAR):-default}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "VAR"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, ":-default}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var in command substitution",
			input: `test: echo $(ls @var(DIR) | wc -l)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo $(ls "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.PIPE, "|"},
				{types.SHELL_TEXT, " wc -l)"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var at start of shell text",
			input: `test: @var(CMD) arg1 arg2`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "CMD"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, " arg1 arg2"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var at end of shell text",
			input: `test: cd /path/to/@var(DIR)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cd /path/to/"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@ not followed by var should be literal",
			input: `test: echo user@host.com`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo user@host.com"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var without parentheses is literal",
			input: `test: echo @var is not a decorator`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo @var is not a decorator"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "complex shell with multiple @var",
			input: `deploy: cd @var(SRC) && npm run build:@var(ENV) && cp -r dist/* @var(DEST)/`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "cd "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "SRC"},
				{types.RPAREN, ")"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " npm run build:"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " cp -r dist/* "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DEST"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, "/"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
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
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Building "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "APP"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "cd "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DIR"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "make "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "TARGET"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var with double @ in shell text",
			input: `connect: ssh -p @var(PORT) user@@var(HOST)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "connect"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "ssh -p "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PORT"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, " user@"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "HOST"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var should be tokenized properly (including in quotes)",
			input: `test: echo "@var(PORT)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PORT"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@env should be tokenized properly (same as @var)",
			input: `test: echo "@env(HOME)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.AT, "@"},
				{types.IDENTIFIER, "env"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "HOME"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestStringInShellCommands tests the correct tokenization of strings within shell commands
// Strings should be treated as atomic units - operators inside strings should NOT be tokenized
func TestStringInShellCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "string with pipe operator should be atomic",
			input: `process: echo "hello | world" | grep hello`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "process"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},          // Shell text before string
				{types.STRING_START, "\""},           // STRING_START token
				{types.STRING_TEXT, "hello | world"}, // STRING_TEXT - | inside is literal
				{types.STRING_END, "\""},             // STRING_END token
				{types.PIPE, "|"},                    // Pipe operator outside string is tokenized
				{types.SHELL_TEXT, " grep hello"},    // Remaining shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "string with && operator should be atomic",
			input: `test: echo "build && test" && echo done`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},           // STRING_START token
				{types.STRING_TEXT, "build && test"}, // STRING_TEXT - && inside is literal
				{types.STRING_END, "\""},             // STRING_END token
				{types.AND, "&&"},                    // The actual && operator outside string
				{types.SHELL_TEXT, " echo done"},     // Remaining shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "string with multiple operators should be atomic",
			input: `complex: echo "cmd1 && cmd2 | grep pattern" > output.txt`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "complex"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},                         // STRING_START token
				{types.STRING_TEXT, "cmd1 && cmd2 | grep pattern"}, // STRING_TEXT - operators inside are literal
				{types.STRING_END, "\""},                           // STRING_END token
				{types.SHELL_TEXT, " > output.txt"},                // Shell redirection outside string
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "single quoted string with operators should be atomic",
			input: `process: echo 'hello && world' && echo done`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "process"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "hello && world"}, // STRING token - && inside should not be tokenized
				{types.STRING_END, "'"},
				{types.AND, "&&"},                // The actual && operator outside string
				{types.SHELL_TEXT, " echo done"}, // Remaining shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "backtick string with operators should be atomic",
			input: "process: echo `hello | world` | grep hello",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "process"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "`"},            // STRING_START token
				{types.STRING_TEXT, "hello | world"}, // STRING_TEXT - | inside is literal
				{types.STRING_END, "`"},              // STRING_END token
				{types.PIPE, "|"},                    // The actual pipe operator outside string
				{types.SHELL_TEXT, " grep hello"},    // Remaining shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "string with @var should be atomic but @var should be tokenized separately",
			input: `test: echo "Value: @var(NAME)" | grep pattern`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},          // STRING_START token
				{types.STRING_TEXT, "Value: "},      // STRING_TEXT before decorator
				{types.AT, "@"},                     // @ token
				{types.IDENTIFIER, "var"},           // var identifier
				{types.LPAREN, "("},                 // ( token
				{types.IDENTIFIER, "NAME"},          // NAME identifier
				{types.RPAREN, ")"},                 // ) token
				{types.STRING_END, "\""},            // STRING_END token
				{types.PIPE, "|"},                   // Shell operator outside string
				{types.SHELL_TEXT, " grep pattern"}, // Remaining shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "multiple strings with operators between",
			input: `test: echo "first | string" && echo "second && string"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},              // First STRING_START
				{types.STRING_TEXT, "first | string"},   // First STRING_TEXT - | inside is literal
				{types.STRING_END, "\""},                // First STRING_END
				{types.AND, "&&"},                       // The actual && operator between strings
				{types.SHELL_TEXT, " echo "},            // Shell text between operator and string
				{types.STRING_START, "\""},              // Second STRING_START
				{types.STRING_TEXT, "second && string"}, // Second STRING_TEXT - && inside is literal
				{types.STRING_END, "\""},                // Second STRING_END
				{types.SHELL_END, ""},                   // Missing SHELL_END token
				{types.EOF, ""},
			},
		},
		{
			name:  "string in command substitution",
			input: `test: echo $(cat "file | with | pipes") | grep pattern`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo $(cat "},
				{types.STRING_START, "\""},                 // STRING_START token
				{types.STRING_TEXT, "file | with | pipes"}, // STRING_TEXT - pipes inside are literal
				{types.STRING_END, "\""},                   // STRING_END token
				{types.SHELL_TEXT, ")"},                    // Closing paren
				{types.PIPE, "|"},                          // Shell operator outside string
				{types.SHELL_TEXT, " grep pattern"},        // Remaining shell text
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "complex real-world example",
			input: `deploy: echo "Building app && running tests" && npm run build:prod | tee "build && test.log"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},                           // First STRING_START
				{types.STRING_TEXT, "Building app && running tests"}, // First STRING_TEXT - operators inside are literal
				{types.STRING_END, "\""},                             // First STRING_END
				{types.AND, "&&"},                                    // First shell operator
				{types.SHELL_TEXT, "npm run build:prod"},             // Shell text (no leading space)
				{types.PIPE, "|"},                                    // Second shell operator
				{types.SHELL_TEXT, " tee "},                          // Shell text before final string
				{types.STRING_START, "\""},                           // Second STRING_START
				{types.STRING_TEXT, "build && test.log"},             // Second STRING_TEXT - operators inside are literal
				{types.STRING_END, "\""},                             // Second STRING_END
				{types.SHELL_END, ""},                                // Missing SHELL_END token
				{types.EOF, ""},
			},
		},
		{
			name:  "nested quotes with decorators - single quotes inside double quotes",
			input: `test: echo "Current context: @env('KUBE_CONTEXT'), Project: @var(KUBE_CONTEXT)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Current context: "},
				{types.AT, "@"},
				{types.IDENTIFIER, "env"},
				{types.LPAREN, "("},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "KUBE_CONTEXT"}, // Single quoted string inside double quotes
				{types.STRING_END, "'"},
				{types.RPAREN, ")"},
				{types.STRING_TEXT, ", Project: "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "KUBE_CONTEXT"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestValueDecoratorsInStrings tests that value decorators are properly tokenized within interpolated strings
func TestValueDecoratorsInStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple @var in double quoted string",
			input: `test: echo "Hello @var(NAME)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},    // Start of string
				{types.STRING_TEXT, "Hello "}, // Text part
				{types.AT, "@"},               // Value decorator
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NAME"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""}, // End of string
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var at start of string",
			input: `test: echo "@var(NAME) says hello"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NAME"},
				{types.RPAREN, ")"},
				{types.STRING_TEXT, " says hello"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var at end of string",
			input: `test: echo "Hello @var(NAME)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Hello "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NAME"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "multiple @var in string",
			input: `test: echo "@var(GREETING) @var(NAME)!"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "GREETING"},
				{types.RPAREN, ")"},
				{types.STRING_TEXT, " "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NAME"},
				{types.RPAREN, ")"},
				{types.STRING_TEXT, "!"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@env in double quoted string",
			input: `test: echo "Path: @env(PATH)"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Path: "},
				{types.AT, "@"},
				{types.IDENTIFIER, "env"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PATH"},
				{types.RPAREN, ")"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "@var should NOT be expanded in single quoted string",
			input: `test: echo 'Hello @var(NAME)'`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "Hello @var(NAME)"}, // Single quoted - should be atomic
				{types.STRING_END, "'"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "value decorators should be expanded in backtick multi-line string",
			input: `test: echo ` + "`" + `Hello @var(NAME)
on port @var(PORT)` + "`",
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "`"},
				{types.STRING_TEXT, "Hello "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "NAME"},
				{types.RPAREN, ")"},
				{types.STRING_TEXT, "\non port "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PORT"},
				{types.RPAREN, ")"},
				{types.STRING_END, "`"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "string with both @var and operators (operators should be literal)",
			input: `test: echo "Command: @var(CMD) | grep pattern"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Command: "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "CMD"},
				{types.RPAREN, ")"},
				{types.STRING_TEXT, " | grep pattern"}, // Operators inside string are literal text
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
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
		lexer := New(strings.NewReader(input))
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
			fmt.Fprintf(&sb, "var types.VAR%d = \"value%d\"\n", i, i)
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

// TestBlockDecoratorShellContent tests that shell content inside block decorators is properly lexed as SHELL_TEXT
func TestBlockDecoratorShellContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "parallel decorator with simple shell content",
			input: `services: @parallel { server; client }`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "services"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "server; client"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:  "timeout decorator with shell content",
			input: `build: @timeout(30s) { npm run build }`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:  "retry decorator with complex shell content",
			input: `test: @retry(3) { npm test && echo "success" }`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.NUMBER, "3"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm test"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "success"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "nested parallel decorator inside command block",
			input: `format: {
    @parallel {
        echo test
    }
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "format"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo test"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestFailingSetupCommand tests the specific case that's failing in commands.cli
func TestFailingSetupCommand(t *testing.T) {
	input := `setup: {
    echo "Setting up development environment..."
    echo "Downloading Go dependencies for all modules..."
    @parallel {
        cd core && go mod download
        cd runtime && go mod download
        cd testing && go mod download
        cd cli && go mod download
    }
    echo "Setup complete! Run 'dev test' to verify."
}`

	lexer := New(strings.NewReader(input))
	tokens := lexer.TokenizeToSlice()

	// Print tokens for debugging
	for i, token := range tokens {
		t.Logf("Token %d: %s = %q", i, token.Type, token.Value)
	}

	// Check that @parallel is correctly tokenized as AT + IDENTIFIER, not SHELL_TEXT
	found := false
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type == types.AT && tokens[i+1].Type == types.IDENTIFIER && tokens[i+1].Value == "parallel" {
			found = true
			break
		}
	}

	if !found {
		t.Error("@parallel should be tokenized as AT + IDENTIFIER, but was tokenized differently")
	}
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
		{types.COMMENT, "# Format all code"},
		{types.IDENTIFIER, "format"},
		{types.COLON, ":"},
		{types.LBRACE, "{"},
		{types.SHELL_TEXT, "echo "},
		{types.STRING_START, "\""},
		{types.STRING_TEXT, "📝 Formatting all code..."},
		{types.STRING_END, "\""},
		{types.SHELL_END, ""},
		{types.SHELL_TEXT, "echo "},
		{types.STRING_START, "\""},
		{types.STRING_TEXT, "Formatting Go code..."},
		{types.STRING_END, "\""},
		{types.SHELL_END, ""},
		{types.AT, "@"},
		{types.IDENTIFIER, "parallel"},
		{types.LBRACE, "{"},
		{types.SHELL_TEXT, "if command -v gofumpt >/dev/null 2>&1; then gofumpt -w .; else go fmt ./...; fi"},
		{types.SHELL_END, ""},
		{types.SHELL_TEXT, "if command -v nixpkgs-fmt >/dev/null 2>&1; then find . -name "},
		{types.STRING_START, "'"},
		{types.STRING_TEXT, "*.nix"},
		{types.STRING_END, "'"},
		{types.SHELL_TEXT, " -exec nixpkgs-fmt {} +; else echo "},
		{types.STRING_START, "\""},
		{types.STRING_TEXT, "⚠️  nixpkgs-fmt not available"},
		{types.STRING_END, "\""},
		{types.SHELL_TEXT, "; fi"},
		{types.SHELL_END, ""},
		{types.RBRACE, "}"},
		{types.SHELL_TEXT, "echo "},
		{types.STRING_START, "\""},
		{types.STRING_TEXT, "✅ Code formatted!"},
		{types.STRING_END, "\""},
		{types.SHELL_END, ""},
		{types.RBRACE, "}"},
		{types.EOF, ""},
	}

	assertTokens(t, "Real world format command", input, expected)
}

func TestWatchStopMultipleCommands(t *testing.T) {
	input := "watch server: npm start\nstop server: pkill node"

	expected := []tokenExpectation{
		{types.WATCH, "watch"},
		{types.IDENTIFIER, "server"},
		{types.COLON, ":"},
		{types.SHELL_TEXT, "npm start"},
		{types.SHELL_END, ""},
		{types.STOP, "stop"},
		{types.IDENTIFIER, "server"},
		{types.COLON, ":"},
		{types.SHELL_TEXT, "pkill node"},
		{types.SHELL_END, ""},
		{types.EOF, ""},
	}

	assertTokens(t, "watch and stop commands on separate lines", input, expected)
}

// TestSpecialCharacters tests lexing of commands with special characters
// This test uses the same inputs as the failing error handling test to diagnose the issue
func TestSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "valid command without special chars",
			input: `valid: echo "This works"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "valid"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "This works"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "command with special characters",
			input: `special-chars: echo "Special: !#\$%^&*()"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "special-chars"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Special: !#\\$%^&*()"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "unicode command",
			input: `unicode: echo "Hello 世界"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "unicode"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Hello 世界"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "special chars in single quotes",
			input: `single-quote: echo 'Special: !#$%^&*()'`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "single-quote"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "Special: !#$%^&*()"},
				{types.STRING_END, "'"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "mixed quotes with special chars",
			input: `mixed: echo "Before" && echo 'Special: !@#$%' && echo "After"`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "mixed"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Before"},
				{types.STRING_END, "\""},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " echo "},
				{types.STRING_START, "'"},
				{types.STRING_TEXT, "Special: !@#$%"},
				{types.STRING_END, "'"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "After"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestNamedParameters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "retry with named parameter",
			input: `test: @retry(attempts=3) { echo "task" }`,
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
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "task"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "complex nested decorators with named parameters",
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
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Dev environment"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Always execute"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
