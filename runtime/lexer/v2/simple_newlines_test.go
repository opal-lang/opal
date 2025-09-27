package v2

import (
	"testing"
)

func TestSimpleNewlines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "single newline",
			input: "hello\nworld",
			expected: []tokenExpectation{
				{IDENTIFIER, "hello", 1, 1},
				{NEWLINE, "", 1, 6},
				{IDENTIFIER, "world", 2, 1},
				{EOF, "", 2, 6},
			},
		},
		{
			name:  "consecutive newlines - only first emitted",
			input: "hello\n\n\nworld",
			expected: []tokenExpectation{
				{IDENTIFIER, "hello", 1, 1},
				{NEWLINE, "", 1, 6},
				{IDENTIFIER, "world", 4, 1},
				{EOF, "", 4, 6},
			},
		},
		{
			name:  "leading newlines",
			input: "\n\nhello",
			expected: []tokenExpectation{
				{NEWLINE, "", 1, 1},
				{IDENTIFIER, "hello", 3, 1},
				{EOF, "", 3, 6},
			},
		},
		{
			name:  "only newlines",
			input: "\n\n\n",
			expected: []tokenExpectation{
				{NEWLINE, "", 1, 1},
				{EOF, "", 4, 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
