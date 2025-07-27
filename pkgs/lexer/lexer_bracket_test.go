package lexer

import (
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/types"
)

func TestShellBracketStructures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple parameter expansion",
			input: `test: echo ${VAR}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${VAR}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "parameter expansion with default",
			input: `test: echo ${VAR:-default}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${VAR:-default}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "nested parameter expansion",
			input: `test: echo ${VAR:-${DEFAULT}}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${VAR:-${DEFAULT}}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "command substitution with braces",
			input: `test: echo $(find . -name "*.go" -exec ls {} +)`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo $(find . -name \"*.go\" -exec ls {} +)"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "mixed parameter expansion and command substitution",
			input: `test: echo ${VAR:-$(date +%Y)}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${VAR:-$(date +%Y)}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "parameter expansion with @var function decorator",
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
			name:  "parameter expansion with TWO @var decorators (parser failing case)",
			input: `test: echo ${@var(VAR):-@var(DEFAULT)}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "VAR"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, ":-"},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "DEFAULT"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, "}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "array syntax",
			input: `test: echo ${ARRAY[0]}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo ${ARRAY[0]}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "brace expansion",
			input: `test: echo {a,b,c}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo {a,b,c}"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "find with -exec and braces",
			input: `test: find . -name "*.txt" -exec rm {} \;`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "find . -name \"*.txt\" -exec rm {} \\;"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name:  "complex shell with multiple bracket types",
			input: `test: for f in $(find . -name "*.go"); do echo ${f%.go}.bin; done`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "for f in $(find . -name \"*.go\"); do echo ${f%.go}.bin; done"},
				{types.SHELL_END, ""},
				{types.EOF, ""},
			},
		},
		{
			name: "block command with shell brackets inside",
			input: `test: {
    echo ${VAR:-default}
    find . -exec ls {} +
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo ${VAR:-default}"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "find . -exec ls {} +"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name: "decorator with shell brackets in block",
			input: `test: @timeout(30s) {
    rsync -av ${SRC}/ ${DEST}/
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "rsync -av ${SRC}/ ${DEST}/"},
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
