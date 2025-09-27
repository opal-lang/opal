package v2

import (
	"testing"
)

// TestLogicalOperators tests basic logical operators following TDD approach
func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// Basic logical operators
		{
			name:  "logical and",
			input: "&&",
			expected: []tokenExpectation{
				{
					Type: AND_AND, Text: "",
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
			name:  "logical or",
			input: "||",
			expected: []tokenExpectation{
				{
					Type: OR_OR, Text: "",
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
			name:  "logical not",
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

		// Simple logical expressions
		{
			name:  "simple and expression",
			input: "true && false",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "true",
					Line:   1,
					Column: 1,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 6,
				},
				{
					Type: IDENTIFIER, Text: "false",
					Line:   1,
					Column: 9,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 14,
				},
			},
		},
		{
			name:  "simple or expression",
			input: "enabled || debug",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "enabled",
					Line:   1,
					Column: 1,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 9,
				},
				{
					Type: IDENTIFIER, Text: "debug",
					Line:   1,
					Column: 12,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 17,
				},
			},
		},
		{
			name:  "negation expression",
			input: "!active",
			expected: []tokenExpectation{
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "active",
					Line:   1,
					Column: 2,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 8,
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

// TestLogicalChaining tests complex logical operator chaining scenarios
func TestLogicalChaining(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// AND chaining
		{
			name:  "double and chain",
			input: "a && b && c",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "a",
					Line:   1,
					Column: 1,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "b",
					Line:   1,
					Column: 6,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "c",
					Line:   1,
					Column: 11,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 12,
				},
			},
		},
		{
			name:  "triple and chain",
			input: "x && y && z && w",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 6,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "z",
					Line:   1,
					Column: 11,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 13,
				},
				{
					Type: IDENTIFIER, Text: "w",
					Line:   1,
					Column: 16,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 17,
				},
			},
		},

		// OR chaining
		{
			name:  "double or chain",
			input: "dev || test || prod",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "dev",
					Line:   1,
					Column: 1,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 5,
				},
				{
					Type: IDENTIFIER, Text: "test",
					Line:   1,
					Column: 8,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 13,
				},
				{
					Type: IDENTIFIER, Text: "prod",
					Line:   1,
					Column: 16,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 20,
				},
			},
		},

		// Mixed AND/OR chaining (precedence handled by parser)
		{
			name:  "mixed and or chain",
			input: "a && b || c && d",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "a",
					Line:   1,
					Column: 1,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "b",
					Line:   1,
					Column: 6,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "c",
					Line:   1,
					Column: 11,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 13,
				},
				{
					Type: IDENTIFIER, Text: "d",
					Line:   1,
					Column: 16,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 17,
				},
			},
		},

		// NOT chaining and combinations
		{
			name:  "not with and",
			input: "!debug && enabled",
			expected: []tokenExpectation{
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "debug",
					Line:   1,
					Column: 2,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "enabled",
					Line:   1,
					Column: 11,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 18,
				},
			},
		},
		{
			name:  "multiple not operators",
			input: "!(!active)",
			expected: []tokenExpectation{
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: LPAREN, Text: "",
					Line:   1,
					Column: 2,
				},
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "active",
					Line:   1,
					Column: 4,
				},
				{
					Type: RPAREN, Text: "",
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
			name:  "not with or chain",
			input: "!maintenance || !debug || force",
			expected: []tokenExpectation{
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "maintenance",
					Line:   1,
					Column: 2,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 14,
				},
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 17,
				},
				{
					Type: IDENTIFIER, Text: "debug",
					Line:   1,
					Column: 18,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 24,
				},
				{
					Type: IDENTIFIER, Text: "force",
					Line:   1,
					Column: 27,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 32,
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

// TestComparisonLogicalChaining tests complex chains mixing comparison and logical operators
func TestComparisonLogicalChaining(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// Comparison with logical operators
		{
			name:  "equals and greater than",
			input: "x == 5 && y > 10",
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
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 11,
				},
				{
					Type: GT, Text: "",
					Line:   1,
					Column: 13,
				},
				{
					Type: INTEGER, Text: "10",
					Line:   1,
					Column: 15,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 17,
				},
			},
		},
		{
			name:  "not equals or less than equal",
			input: "status != \"error\" || retries <= 3",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "status",
					Line:   1,
					Column: 1,
				},
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 8,
				},
				{Type: STRING, Text: "\"error\"", Line: 1, Column: 11},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 19,
				},
				{
					Type: IDENTIFIER, Text: "retries",
					Line:   1,
					Column: 22,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 30,
				},
				{
					Type: INTEGER, Text: "3",
					Line:   1,
					Column: 33,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 34,
				},
			},
		},

		// Complex chaining scenarios
		{
			name:  "triple comparison with and",
			input: "a < b && b <= c && c > 0",
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
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 7,
				},
				{
					Type: IDENTIFIER, Text: "b",
					Line:   1,
					Column: 10,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 12,
				},
				{
					Type: IDENTIFIER, Text: "c",
					Line:   1,
					Column: 15,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 17,
				},
				{
					Type: IDENTIFIER, Text: "c",
					Line:   1,
					Column: 20,
				},
				{
					Type: GT, Text: "",
					Line:   1,
					Column: 22,
				},
				{
					Type: INTEGER, Text: "0",
					Line:   1,
					Column: 24,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 25,
				},
			},
		},
		{
			name:  "mixed comparison and logical with not",
			input: "!debug && (x >= 5 || y != null)",
			expected: []tokenExpectation{
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "debug",
					Line:   1,
					Column: 2,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: LPAREN, Text: "",
					Line:   1,
					Column: 11,
				},
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 12,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 14,
				},
				{
					Type: INTEGER, Text: "5",
					Line:   1,
					Column: 17,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 19,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 22,
				},
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 24,
				},
				{
					Type: IDENTIFIER, Text: "null",
					Line:   1,
					Column: 27,
				},
				{
					Type: RPAREN, Text: "",
					Line:   1,
					Column: 31,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 32,
				},
			},
		},

		// Arithmetic with comparison and logical
		{
			name:  "arithmetic comparison logical chain",
			input: "count + 1 >= max && timeout * 2 <= limit || force",
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
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 18,
				},
				{
					Type: IDENTIFIER, Text: "timeout",
					Line:   1,
					Column: 21,
				},
				{
					Type: MULTIPLY, Text: "",
					Line:   1,
					Column: 29,
				},
				{
					Type: INTEGER, Text: "2",
					Line:   1,
					Column: 31,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 33,
				},
				{
					Type: IDENTIFIER, Text: "limit",
					Line:   1,
					Column: 36,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 42,
				},
				{
					Type: IDENTIFIER, Text: "force",
					Line:   1,
					Column: 45,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 50,
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

// TestLogicalOperatorEdgeCases tests boundary conditions and operator disambiguation
func TestLogicalOperatorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// Single ampersand/pipe (future: bitwise operators or error)
		{
			name:  "single ampersand",
			input: "&",
			expected: []tokenExpectation{
				{
					Type: ILLEGAL, Text: "",
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
			name:  "single pipe",
			input: "|",
			expected: []tokenExpectation{
				{
					Type: ILLEGAL, Text: "",
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

		// Mixed with assignment and comparison
		{
			name:  "not equals vs logical not assignment",
			input: "x != y && z = !w",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 6,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "z",
					Line:   1,
					Column: 11,
				},
				{
					Type: EQUALS, Text: "",
					Line:   1,
					Column: 13,
				},
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 15,
				},
				{
					Type: IDENTIFIER, Text: "w",
					Line:   1,
					Column: 16,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 17,
				},
			},
		},

		// No spaces between operators
		{
			name:  "operators without spaces",
			input: "x&&y||z!=w",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 2,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   1,
					Column: 4,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 5,
				},
				{
					Type: IDENTIFIER, Text: "z",
					Line:   1,
					Column: 7,
				},
				{
					Type: NOT_EQ, Text: "",
					Line:   1,
					Column: 8,
				},
				{
					Type: IDENTIFIER, Text: "w",
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

		// Triple operators (should be two separate operators)
		{
			name:  "triple ampersand",
			input: "&&&",
			expected: []tokenExpectation{
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: ILLEGAL, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 4,
				},
			},
		},
		{
			name:  "triple pipe",
			input: "|||",
			expected: []tokenExpectation{
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: ILLEGAL, Text: "",
					Line:   1,
					Column: 3,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 4,
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

// TestLogicalInDevcmdContext tests logical operators in realistic devcmd scenarios
func TestLogicalInDevcmdContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// Environment-based conditionals
		{
			name:  "environment check with and",
			input: "if env == \"production\" && replicas >= 3 {",
			expected: []tokenExpectation{
				{
					Type: IF, Text: "if",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "env",
					Line:   1,
					Column: 4,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 8,
				},
				{Type: STRING, Text: "\"production\"", Line: 1, Column: 11},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 24,
				},
				{
					Type: IDENTIFIER, Text: "replicas",
					Line:   1,
					Column: 27,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 36,
				},
				{
					Type: INTEGER, Text: "3",
					Line:   1,
					Column: 39,
				},
				{
					Type: LBRACE, Text: "",
					Line:   1,
					Column: 41,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 42,
				},
			},
		},

		// Deployment conditions
		{
			name:  "deployment readiness check",
			input: "if !maintenance && (health == \"ok\" || override) {",
			expected: []tokenExpectation{
				{
					Type: IF, Text: "if",
					Line:   1,
					Column: 1,
				},
				{
					Type: NOT, Text: "",
					Line:   1,
					Column: 4,
				},
				{
					Type: IDENTIFIER, Text: "maintenance",
					Line:   1,
					Column: 5,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 17,
				},
				{
					Type: LPAREN, Text: "",
					Line:   1,
					Column: 20,
				},
				{
					Type: IDENTIFIER, Text: "health",
					Line:   1,
					Column: 21,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 28,
				},
				{Type: STRING, Text: "\"ok\"", Line: 1, Column: 31},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 36,
				},
				{
					Type: IDENTIFIER, Text: "override",
					Line:   1,
					Column: 39,
				},
				{
					Type: RPAREN, Text: "",
					Line:   1,
					Column: 47,
				},
				{
					Type: LBRACE, Text: "",
					Line:   1,
					Column: 49,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 50,
				},
			},
		},

		// Resource validation
		{
			name:  "resource validation chain",
			input: "cpu <= maxCpu && memory <= maxMemory && replicas > 0",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "cpu",
					Line:   1,
					Column: 1,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 5,
				},
				{
					Type: IDENTIFIER, Text: "maxCpu",
					Line:   1,
					Column: 8,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 15,
				},
				{
					Type: IDENTIFIER, Text: "memory",
					Line:   1,
					Column: 18,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 25,
				},
				{
					Type: IDENTIFIER, Text: "maxMemory",
					Line:   1,
					Column: 28,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 38,
				},
				{
					Type: IDENTIFIER, Text: "replicas",
					Line:   1,
					Column: 41,
				},
				{
					Type: GT, Text: "",
					Line:   1,
					Column: 50,
				},
				{
					Type: INTEGER, Text: "0",
					Line:   1,
					Column: 52,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 53,
				},
			},
		},

		// Timeout and retry logic
		{
			name:  "timeout with duration and retries",
			input: "timeout >= 30s && retries <= 5 || force",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "timeout",
					Line:   1,
					Column: 1,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   1,
					Column: 9,
				},
				{
					Type: DURATION, Text: "30s",
					Line:   1,
					Column: 12,
				},
				{
					Type: AND_AND, Text: "",
					Line:   1,
					Column: 16,
				},
				{
					Type: IDENTIFIER, Text: "retries",
					Line:   1,
					Column: 19,
				},
				{
					Type: LT_EQ, Text: "",
					Line:   1,
					Column: 27,
				},
				{
					Type: INTEGER, Text: "5",
					Line:   1,
					Column: 30,
				},
				{
					Type: OR_OR, Text: "",
					Line:   1,
					Column: 32,
				},
				{
					Type: IDENTIFIER, Text: "force",
					Line:   1,
					Column: 35,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 40,
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
