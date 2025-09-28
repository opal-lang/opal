package v2

import (
	"testing"
)

// TestComparisonOperators tests all comparison operators following TDD approach
func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// Single character comparison operators
		{
			name:  "less than",
			input: "<",
			expected: []tokenExpectation{
				{
					Type: LT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 2,
				},
			},
		},
		{
			name:  "greater than",
			input: ">",
			expected: []tokenExpectation{
				{
					Type: GT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 2,
				},
			},
		},

		// Double character comparison operators
		{
			name:  "equals",
			input: "==",
			expected: []tokenExpectation{
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 3,
				},
			},
		},
		{
			name:  "not equals",
			input: "!=",
			expected: []tokenExpectation{
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 3,
				},
			},
		},
		{
			name:  "less than or equal",
			input: "<=",
			expected: []tokenExpectation{
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 3,
				},
			},
		},
		{
			name:  "greater than or equal",
			input: ">=",
			expected: []tokenExpectation{
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 3,
				},
			},
		},

		// Comparison operators in expressions
		{
			name:  "variable comparison",
			input: "x == 5",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: INTEGER, Text: "5",
					Line:   1,
					Column: 6,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 7,
				},
			},
		},
		{
			name:  "environment variable comparison",
			input: "@env(DEBUG) != \"false\"",
			expected: []tokenExpectation{
				{
					Type: AT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "env",
					Line:   1,
					Column: 2,
				},
				{
					Type: LPAREN, Text: "",
					Line:   1,
					Column: 5,
				},
				{
					Type: IDENTIFIER, Text: "DEBUG",
					Line:   1,
					Column: 6,
				},
				{
					Type: RPAREN, Text: "",
					Line:   1,
					Column: 11,
				},
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 13,
				},
				{Type: STRING, Text: "\"false\"", Line: 1, Column: 16},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 23,
				},
			},
		},
		{
			name:  "numeric comparisons",
			input: "replicas >= 3 && timeout <= 30s",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "replicas",
					Line:   1,
					Column: 1,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 10,
				},
				{
					Type: INTEGER, Text: "3",
					Line:   1,
					Column: 13,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 15,
				},
				{
					Type: IDENTIFIER, Text: "timeout",
					Line:   1,
					Column: 18,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 26,
				},
				{
					Type: DURATION, Text: "30s",
					Line:   1,
					Column: 29,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 32,
				},
			},
		},

		// Edge cases
		{
			name:  "assignment vs equality",
			input: "x = 5 == y",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: EQUALS, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: INTEGER, Text: "5",
					Line:   1,
					Column: 5,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 7,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 10,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 11,
				},
			},
		},
		{
			name:  "chained comparisons",
			input: "a < b <= c > d >= e",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "a",
					Line:   1,
					Column: 1,
				},
				{
					Type: LT, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "b",
					Line:   1,
					Column: 5,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 7,
				},
				{
					Type: IDENTIFIER, Text: "c",
					Line:   1,
					Column: 10,
				},
				{
					Type: GT, Text: "",
					Line:   1,
					Column: 12,
				},
				{
					Type: IDENTIFIER, Text: "d",
					Line:   1,
					Column: 14,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 16,
				},
				{
					Type: IDENTIFIER, Text: "e",
					Line:   1,
					Column: 19,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 20,
				},
			},
		},

		// Mixed with other operators
		{
			name:  "comparison with arithmetic",
			input: "count + 1 >= max * 2",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "count",
					Line:   1,
					Column: 1,
				},
				{
					Type: PLUS, Text: "",
					Line:   1,
					Column: 7,
				},
				{
					Type: INTEGER, Text: "1",
					Line:   1,
					Column: 9,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 11,
				},
				{
					Type: IDENTIFIER, Text: "max",
					Line:   1,
					Column: 14,
				},
				{
					Type: MULTIPLY, Text: "",
					Line:   1,
					Column: 18,
				},
				{
					Type: INTEGER, Text: "2",
					Line:   1,
					Column: 20,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 21,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestComparisonOperatorEdgeCases tests boundary conditions and error cases
func TestComparisonOperatorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "single equals (assignment)",
			input: "=",
			expected: []tokenExpectation{
				{
					Type: EQUALS, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 2,
				},
			},
		},
		{
			name:  "single exclamation (logical not)",
			input: "!",
			expected: []tokenExpectation{
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 2,
				},
			},
		},
		{
			name:  "mixed operators without spaces",
			input: "x>=y<=z==w!=v",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 2,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 4,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 5,
				},
				{
					Type: IDENTIFIER, Text: "z",
					Line:   1,
					Column: 7,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "w",
					Line:   1,
					Column: 10,
				},
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 11,
				},
				{
					Type: IDENTIFIER, Text: "v",
					Line:   1,
					Column: 13,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 14,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestComparisonInControlFlow tests comparison operators in realistic opal scenarios
func TestComparisonInControlFlow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "if statement with comparison",
			input: "if replicas > 0 {",
			expected: []tokenExpectation{
				{
					Type: IF, Text: "if",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "replicas",
					Line:   1,
					Column: 4,
				},
				{
					Type: GT, Text: "",
					Line:   1,
					Column: 13,
				},
				{
					Type: INTEGER, Text: "0",
					Line:   1,
					Column: 15,
				},
				{
					Type: LBRACE, Text: "",
					Line:   1,
					Column: 17,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 18,
				},
			},
		},
		{
			name:  "environment comparison",
			input: "if @env(STAGE) == \"production\" {",
			expected: []tokenExpectation{
				{
					Type: IF, Text: "if",
					Line:   1,
					Column: 1,
				},
				{
					Type: AT, Text: "",
					Line:   1,
					Column: 4,
				},
				{
					Type: IDENTIFIER, Text: "env",
					Line:   1,
					Column: 5,
				},
				{
					Type: LPAREN, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "STAGE",
					Line:   1,
					Column: 9,
				},
				{
					Type: RPAREN, Text: "",
					Line:   1,
					Column: 14,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 16,
				},
				{Type: STRING, Text: "\"production\"", Line: 1, Column: 19},
				{
					Type: LBRACE, Text: "",
					Line:   1,
					Column: 32,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 33,
				},
			},
		},
		{
			name:  "duration comparison",
			input: "if timeout <= 5m {",
			expected: []tokenExpectation{
				{
					Type: IF, Text: "if",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "timeout",
					Line:   1,
					Column: 4,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 12,
				},
				{
					Type: DURATION, Text: "5m",
					Line:   1,
					Column: 15,
				},
				{
					Type: LBRACE, Text: "",
					Line:   1,
					Column: 18,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 19,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
