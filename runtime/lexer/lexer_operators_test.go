package lexer

import (
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/types"
)

func TestLexer_ShellOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []types.TokenType
		values   []string
	}{
		{
			name:     "simple_and_operator",
			input:    `test: echo "first" && echo "second"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.AND, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "first", "\"", "&&", " echo ", "\"", "second", "\"", "", ""},
		},
		{
			name:     "simple_or_operator",
			input:    `test: echo "first" || echo "second"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.OR, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "first", "\"", "||", " echo ", "\"", "second", "\"", "", ""},
		},
		{
			name:     "simple_pipe_operator",
			input:    `test: echo "hello" | grep hello`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.PIPE, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "hello", "\"", "|", " grep hello", "", ""},
		},
		{
			name:     "simple_append_operator",
			input:    `test: echo "data" >> file.txt`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.APPEND, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "data", "\"", ">>", " file.txt", "", ""},
		},
		{
			name:     "multiple_operators",
			input:    `test: build && test || echo "failed" | tee log.txt`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.AND, types.SHELL_TEXT, types.OR, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.PIPE, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "build", "&&", "test", "||", " echo ", "\"", "failed", "\"", "|", " tee log.txt", "", ""},
		},
		{
			name:     "operators_with_spacing",
			input:    `test: cmd1&&cmd2 ||   cmd3  |  cmd4>>file`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.AND, types.SHELL_TEXT, types.OR, types.SHELL_TEXT, types.PIPE, types.SHELL_TEXT, types.APPEND, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "cmd1", "&&", "cmd2", "||", "cmd3", "|", "cmd4", ">>", "file", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(strings.NewReader(tt.input))
			tokens := tokenizeAll(l)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("wrong number of tokens. expected=%d, got=%d\nTokens: %v",
					len(tt.expected), len(tokens), tokensToStrings(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("token[%d] wrong type. expected=%v, got=%v",
						i, expectedType, tokens[i].Type)
				}
				if i < len(tt.values) && tokens[i].Value != tt.values[i] {
					t.Errorf("token[%d] wrong value. expected=%q, got=%q",
						i, tt.values[i], tokens[i].Value)
				}
			}
		})
	}
}

func TestLexer_OperatorsInStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []types.TokenType
		values   []string
	}{
		{
			name:     "and_in_double_quotes",
			input:    `test: echo "this && that"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "this && that", "\"", "", ""},
		},
		{
			name:     "pipe_in_single_quotes",
			input:    `test: echo 'cmd1 | cmd2'`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "'", "cmd1 | cmd2", "'", "", ""},
		},
		{
			name:     "append_in_backticks",
			input:    "test: echo `data >> file`",
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "`", "data >> file", "`", "", ""},
		},
		{
			name:     "mixed_quotes_with_operators",
			input:    `test: echo "first && second" | grep 'third || fourth' >> "file >> name"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.PIPE, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.APPEND, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "first && second", "\"", "|", " grep ", "'", "third || fourth", "'", ">>", " ", "\"", "file >> name", "\"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(strings.NewReader(tt.input))
			tokens := tokenizeAll(l)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("wrong number of tokens. expected=%d, got=%d\nTokens: %v",
					len(tt.expected), len(tokens), tokensToStrings(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("token[%d] wrong type. expected=%v, got=%v",
						i, expectedType, tokens[i].Type)
				}
				if i < len(tt.values) && tokens[i].Value != tt.values[i] {
					t.Errorf("token[%d] wrong value. expected=%q, got=%q",
						i, tt.values[i], tokens[i].Value)
				}
			}
		})
	}
}

func TestLexer_EscapedOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []types.TokenType
		values   []string
	}{
		{
			name:     "escaped_and",
			input:    `test: echo first \&\& echo second`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", `echo first \&\& echo second`, "", ""},
		},
		{
			name:     "escaped_pipe",
			input:    `test: echo "data \| more"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "data \\| more", "\"", "", ""},
		},
		{
			name:     "partial_escape",
			input:    `test: echo \&& valid || echo "done"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.AND, types.SHELL_TEXT, types.OR, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo \\", "&&", "valid", "||", " echo ", "\"", "done", "\"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(strings.NewReader(tt.input))
			tokens := tokenizeAll(l)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("wrong number of tokens. expected=%d, got=%d\nTokens: %v",
					len(tt.expected), len(tokens), tokensToStrings(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("token[%d] wrong type. expected=%v, got=%v",
						i, expectedType, tokens[i].Type)
				}
				if i < len(tt.values) && tokens[i].Value != tt.values[i] {
					t.Errorf("token[%d] wrong value. expected=%q, got=%q",
						i, tt.values[i], tokens[i].Value)
				}
			}
		})
	}
}

func TestLexer_OperatorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []types.TokenType
		values   []string
	}{
		{
			name:     "single_ampersand",
			input:    `test: cmd1 & cmd2`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", `cmd1 & cmd2`, "", ""},
		},
		{
			name:     "single_pipe_vs_or",
			input:    `test: cmd1 | cmd2 || cmd3`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.PIPE, types.SHELL_TEXT, types.OR, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "cmd1", "|", "cmd2", "||", " cmd3", "", ""},
		},
		{
			name:     "single_gt_vs_append",
			input:    `test: cmd1 > file && cmd2 >> file`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.AND, types.SHELL_TEXT, types.APPEND, types.SHELL_TEXT, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", `cmd1 > file`, "&&", "cmd2", ">>", " file", "", ""},
		},
		{
			name:     "operators_at_start_end",
			input:    `test: && echo "start" ||`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.AND, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.OR, types.EOF},
			values:   []string{"test", ":", "&&", " echo ", "\"", "start", "\"", "||", ""},
		},
		{
			name:     "empty_between_operators",
			input:    `test: echo "first" &&  && echo "second"`,
			expected: []types.TokenType{types.IDENTIFIER, types.COLON, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.AND, types.AND, types.SHELL_TEXT, types.STRING_START, types.STRING_TEXT, types.STRING_END, types.SHELL_END, types.EOF},
			values:   []string{"test", ":", "echo ", "\"", "first", "\"", "&&", "&&", " echo ", "\"", "second", "\"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(strings.NewReader(tt.input))
			tokens := tokenizeAll(l)

			if len(tokens) != len(tt.expected) {
				t.Fatalf("wrong number of tokens. expected=%d, got=%d\nTokens: %v",
					len(tt.expected), len(tokens), tokensToStrings(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("token[%d] wrong type. expected=%v, got=%v",
						i, expectedType, tokens[i].Type)
				}
				if i < len(tt.values) && tokens[i].Value != tt.values[i] {
					t.Errorf("token[%d] wrong value. expected=%q, got=%q",
						i, tt.values[i], tokens[i].Value)
				}
			}
		})
	}
}

// Helper functions for testing
func tokenizeAll(l *Lexer) []types.Token {
	return l.TokenizeToSlice()
}

func tokensToStrings(tokens []types.Token) []string {
	var result []string
	for _, token := range tokens {
		result = append(result, token.Type.String()+"="+token.Value)
	}
	return result
}
