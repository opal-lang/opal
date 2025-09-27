package v2

import (
	"testing"
)

// TestBasicArithmetic tests simple arithmetic expressions
func TestBasicArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple_addition",
			input: "2 + 3",
			expected: []tokenExpectation{
				{INTEGER, "2", 1, 1},
				{PLUS, "", 1, 3},
				{INTEGER, "3", 1, 5},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "simple_subtraction",
			input: "5 - 2",
			expected: []tokenExpectation{
				{INTEGER, "5", 1, 1},
				{MINUS, "", 1, 3},
				{INTEGER, "2", 1, 5},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "simple_multiplication",
			input: "4 * 6",
			expected: []tokenExpectation{
				{INTEGER, "4", 1, 1},
				{MULTIPLY, "", 1, 3},
				{INTEGER, "6", 1, 5},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "simple_division",
			input: "8 / 2",
			expected: []tokenExpectation{
				{INTEGER, "8", 1, 1},
				{DIVIDE, "", 1, 3},
				{INTEGER, "2", 1, 5},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "simple_modulo",
			input: "10 % 3",
			expected: []tokenExpectation{
				{INTEGER, "10", 1, 1},
				{MODULO, "", 1, 4},
				{INTEGER, "3", 1, 6},
				{EOF, "", 1, 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestArithmeticPrecedence tests operator precedence cases
func TestArithmeticPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "multiplication_before_addition",
			input: "2 + 3 * 4",
			expected: []tokenExpectation{
				{INTEGER, "2", 1, 1},
				{PLUS, "", 1, 3},
				{INTEGER, "3", 1, 5},
				{MULTIPLY, "", 1, 7},
				{INTEGER, "4", 1, 9},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "division_before_subtraction",
			input: "10 - 8 / 2",
			expected: []tokenExpectation{
				{INTEGER, "10", 1, 1},
				{MINUS, "", 1, 4},
				{INTEGER, "8", 1, 6},
				{DIVIDE, "", 1, 8},
				{INTEGER, "2", 1, 10},
				{EOF, "", 1, 11},
			},
		},
		{
			name:  "modulo_before_addition",
			input: "7 + 5 % 3",
			expected: []tokenExpectation{
				{INTEGER, "7", 1, 1},
				{PLUS, "", 1, 3},
				{INTEGER, "5", 1, 5},
				{MODULO, "", 1, 7},
				{INTEGER, "3", 1, 9},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "parentheses_override",
			input: "(2 + 3) * 4",
			expected: []tokenExpectation{
				{LPAREN, "", 1, 1},
				{INTEGER, "2", 1, 2},
				{PLUS, "", 1, 4},
				{INTEGER, "3", 1, 6},
				{RPAREN, "", 1, 7},
				{MULTIPLY, "", 1, 9},
				{INTEGER, "4", 1, 11},
				{EOF, "", 1, 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestArithmeticInVariables tests arithmetic in variable assignments
func TestArithmeticInVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable_assignment_arithmetic",
			input: "var total = base + offset",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "total", 1, 5},
				{EQUALS, "", 1, 11},
				{IDENTIFIER, "base", 1, 13},
				{PLUS, "", 1, 18},
				{IDENTIFIER, "offset", 1, 20},
				{EOF, "", 1, 26},
			},
		},
		{
			name:  "complex_calculation",
			input: "var result = (a * b) + (c / d)",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "result", 1, 5},
				{EQUALS, "", 1, 12},
				{LPAREN, "", 1, 14},
				{IDENTIFIER, "a", 1, 15},
				{MULTIPLY, "", 1, 17},
				{IDENTIFIER, "b", 1, 19},
				{RPAREN, "", 1, 20},
				{PLUS, "", 1, 22},
				{LPAREN, "", 1, 24},
				{IDENTIFIER, "c", 1, 25},
				{DIVIDE, "", 1, 27},
				{IDENTIFIER, "d", 1, 29},
				{RPAREN, "", 1, 30},
				{EOF, "", 1, 31},
			},
		},
		{
			name:  "negative_numbers",
			input: "var diff = -5 + 3",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "diff", 1, 5},
				{EQUALS, "", 1, 10},
				{MINUS, "", 1, 12},
				{INTEGER, "5", 1, 13},
				{PLUS, "", 1, 15},
				{INTEGER, "3", 1, 17},
				{EOF, "", 1, 18},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestAssignmentOperators tests arithmetic assignment operators
func TestAssignmentOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "plus_equals",
			input: "counter += 1",
			expected: []tokenExpectation{
				{IDENTIFIER, "counter", 1, 1},
				{PLUS_ASSIGN, "", 1, 9},
				{INTEGER, "1", 1, 12},
				{EOF, "", 1, 13},
			},
		},
		{
			name:  "minus_equals",
			input: "total -= cost",
			expected: []tokenExpectation{
				{IDENTIFIER, "total", 1, 1},
				{MINUS_ASSIGN, "", 1, 7},
				{IDENTIFIER, "cost", 1, 10},
				{EOF, "", 1, 14},
			},
		},
		{
			name:  "multiply_equals",
			input: "value *= 2",
			expected: []tokenExpectation{
				{IDENTIFIER, "value", 1, 1},
				{MULTIPLY_ASSIGN, "", 1, 7},
				{INTEGER, "2", 1, 10},
				{EOF, "", 1, 11},
			},
		},
		{
			name:  "divide_equals",
			input: "average /= count",
			expected: []tokenExpectation{
				{IDENTIFIER, "average", 1, 1},
				{DIVIDE_ASSIGN, "", 1, 9},
				{IDENTIFIER, "count", 1, 12},
				{EOF, "", 1, 17},
			},
		},
		{
			name:  "modulo_equals",
			input: "index %= size",
			expected: []tokenExpectation{
				{IDENTIFIER, "index", 1, 1},
				{MODULO_ASSIGN, "", 1, 7},
				{IDENTIFIER, "size", 1, 10},
				{EOF, "", 1, 14},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestIncrementDecrement tests increment and decrement operators
func TestIncrementDecrement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "increment",
			input: "counter++",
			expected: []tokenExpectation{
				{IDENTIFIER, "counter", 1, 1},
				{INCREMENT, "", 1, 8},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "decrement",
			input: "attempts--",
			expected: []tokenExpectation{
				{IDENTIFIER, "attempts", 1, 1},
				{DECREMENT, "", 1, 9},
				{EOF, "", 1, 11},
			},
		},
		{
			name:  "increment_in_loop",
			input: "for i in range { counter++ }",
			expected: []tokenExpectation{
				{FOR, "for", 1, 1},
				{IDENTIFIER, "i", 1, 5},
				{IN, "in", 1, 7},
				{IDENTIFIER, "range", 1, 10},
				{LBRACE, "", 1, 16},
				{IDENTIFIER, "counter", 1, 18},
				{INCREMENT, "", 1, 25},
				{RBRACE, "", 1, 28},
				{EOF, "", 1, 29},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestArithmeticInDevcmdContext tests arithmetic in realistic devcmd scenarios
func TestArithmeticInDevcmdContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "replica_calculation",
			input: "var replicas = base_replicas * environments",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "replicas", 1, 5},
				{EQUALS, "", 1, 14},
				{IDENTIFIER, "base_replicas", 1, 16},
				{MULTIPLY, "", 1, 30},
				{IDENTIFIER, "environments", 1, 32},
				{EOF, "", 1, 44},
			},
		},
		{
			name:  "timeout_calculation",
			input: "var timeout = base_timeout + (retry_attempt * backoff)",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "timeout", 1, 5},
				{EQUALS, "", 1, 13},
				{IDENTIFIER, "base_timeout", 1, 15},
				{PLUS, "", 1, 28},
				{LPAREN, "", 1, 30},
				{IDENTIFIER, "retry_attempt", 1, 31},
				{MULTIPLY, "", 1, 45},
				{IDENTIFIER, "backoff", 1, 47},
				{RPAREN, "", 1, 54},
				{EOF, "", 1, 55},
			},
		},
		{
			name:  "batch_processing",
			input: "remaining -= batch_size",
			expected: []tokenExpectation{
				{IDENTIFIER, "remaining", 1, 1},
				{MINUS_ASSIGN, "", 1, 11},
				{IDENTIFIER, "batch_size", 1, 14},
				{EOF, "", 1, 24},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
