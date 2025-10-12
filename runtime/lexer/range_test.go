package lexer

import "testing"

func TestRangeOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "three dots",
			input: "...",
			expected: []tokenExpectation{
				{DOTDOTDOT, "...", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "range in expression",
			input: "1...10",
			expected: []tokenExpectation{
				{INTEGER, "1", 1, 1},
				{DOTDOTDOT, "...", 1, 2},
				{INTEGER, "10", 1, 5},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "range with spaces",
			input: "1 ... 10",
			expected: []tokenExpectation{
				{INTEGER, "1", 1, 1},
				{DOTDOTDOT, "...", 1, 3},
				{INTEGER, "10", 1, 7},
				{EOF, "", 1, 9},
			},
		},
		{
			name:  "two dots not three - Go float behavior",
			input: "1..10",
			expected: []tokenExpectation{
				{FLOAT, "1.", 1, 1},
				{FLOAT, ".10", 1, 3},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "four dots",
			input: "....",
			expected: []tokenExpectation{
				{DOTDOTDOT, "...", 1, 1},
				{DOT, "", 1, 4},
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
