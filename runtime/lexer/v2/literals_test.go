package v2

import (
	"testing"
)

// ===== INTEGER TESTS =====

// TestBasicIntegers tests simple integer tokenization
func TestBasicIntegers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "single_digit",
			input: "5",
			expected: []tokenExpectation{
				{INTEGER, "5", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "multiple_digits",
			input: "123",
			expected: []tokenExpectation{
				{INTEGER, "123", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "zero",
			input: "0",
			expected: []tokenExpectation{
				{INTEGER, "0", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "large_number",
			input: "1234567890",
			expected: []tokenExpectation{
				{INTEGER, "1234567890", 1, 1},
				{EOF, "", 1, 11},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestNegativeIntegers tests negative integer handling (MINUS + INTEGER tokens)
func TestNegativeIntegers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "negative_single",
			input: "-5",
			expected: []tokenExpectation{
				{MINUS, "", 1, 1},
				{INTEGER, "5", 1, 2},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "negative_multiple",
			input: "-123",
			expected: []tokenExpectation{
				{MINUS, "", 1, 1},
				{INTEGER, "123", 1, 2},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "negative_zero",
			input: "-0",
			expected: []tokenExpectation{
				{MINUS, "", 1, 1},
				{INTEGER, "0", 1, 2},
				{EOF, "", 1, 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestIntegerBoundaries tests integer separation and boundaries
func TestIntegerBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "space_separated",
			input: "123 456",
			expected: []tokenExpectation{
				{INTEGER, "123", 1, 1},
				{INTEGER, "456", 1, 5},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "punctuation_separated",
			input: "123,456",
			expected: []tokenExpectation{
				{INTEGER, "123", 1, 1},
				{COMMA, "", 1, 4},
				{INTEGER, "456", 1, 5},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "newline_separated",
			input: "123\n456",
			expected: []tokenExpectation{
				{INTEGER, "123", 1, 1},
				{NEWLINE, "", 1, 4},
				{INTEGER, "456", 2, 1},
				{EOF, "", 2, 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestIntegerInContext tests integers in realistic usage
func TestIntegerInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "port_assignment",
			input: "var PORT = 3000",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "PORT", 1, 5},
				{EQUALS, "", 1, 10},
				{INTEGER, "3000", 1, 12},
				{EOF, "", 1, 16},
			},
		},
		{
			name:  "array_access",
			input: "services[0]",
			expected: []tokenExpectation{
				{IDENTIFIER, "services", 1, 1},
				{LSQUARE, "", 1, 9},
				{INTEGER, "0", 1, 10},
				{RSQUARE, "", 1, 11},
				{EOF, "", 1, 12},
			},
		},
		{
			name:  "negative_in_assignment",
			input: "offset = -100",
			expected: []tokenExpectation{
				{IDENTIFIER, "offset", 1, 1},
				{EQUALS, "", 1, 8},
				{MINUS, "", 1, 10},
				{INTEGER, "100", 1, 11},
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

// ===== FLOAT TESTS =====

// TestBasicFloats tests simple float tokenization
func TestBasicFloats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple_decimal",
			input: "3.14",
			expected: []tokenExpectation{
				{FLOAT, "3.14", 1, 1},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "zero_decimal",
			input: "0.0",
			expected: []tokenExpectation{
				{FLOAT, "0.0", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "leading_zero",
			input: "0.5",
			expected: []tokenExpectation{
				{FLOAT, "0.5", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "trailing_zero",
			input: "5.0",
			expected: []tokenExpectation{
				{FLOAT, "5.0", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "integer_with_decimal",
			input: "123.0",
			expected: []tokenExpectation{
				{FLOAT, "123.0", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "many_decimals",
			input: "3.14159",
			expected: []tokenExpectation{
				{FLOAT, "3.14159", 1, 1},
				{EOF, "", 1, 8},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestNegativeFloats tests negative float handling
func TestNegativeFloats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "negative_decimal",
			input: "-3.14",
			expected: []tokenExpectation{
				{MINUS, "", 1, 1},
				{FLOAT, "3.14", 1, 2},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "negative_zero",
			input: "-0.0",
			expected: []tokenExpectation{
				{MINUS, "", 1, 1},
				{FLOAT, "0.0", 1, 2},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "negative_small",
			input: "-0.5",
			expected: []tokenExpectation{
				{MINUS, "", 1, 1},
				{FLOAT, "0.5", 1, 2},
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

// TestFloatBoundaries tests float separation and boundaries
func TestFloatBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "space_separated",
			input: "3.14 2.71",
			expected: []tokenExpectation{
				{FLOAT, "3.14", 1, 1},
				{FLOAT, "2.71", 1, 6},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "punctuation_separated",
			input: "3.14,2.71",
			expected: []tokenExpectation{
				{FLOAT, "3.14", 1, 1},
				{COMMA, "", 1, 5},
				{FLOAT, "2.71", 1, 6},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "mixed_with_integers",
			input: "123 3.14 456",
			expected: []tokenExpectation{
				{INTEGER, "123", 1, 1},
				{FLOAT, "3.14", 1, 5},
				{INTEGER, "456", 1, 10},
				{EOF, "", 1, 13},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestFloatInContext tests floats in realistic usage
func TestFloatInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "version_number",
			input: "var VERSION = 1.5",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "VERSION", 1, 5},
				{EQUALS, "", 1, 13},
				{FLOAT, "1.5", 1, 15},
				{EOF, "", 1, 18},
			},
		},
		{
			name:  "timeout_value",
			input: "timeout = 2.5",
			expected: []tokenExpectation{
				{IDENTIFIER, "timeout", 1, 1},
				{EQUALS, "", 1, 9},
				{FLOAT, "2.5", 1, 11},
				{EOF, "", 1, 14},
			},
		},
		{
			name:  "array_of_floats",
			input: "[1.0, 2.5, 3.14]",
			expected: []tokenExpectation{
				{LSQUARE, "", 1, 1},
				{FLOAT, "1.0", 1, 2},
				{COMMA, "", 1, 5},
				{FLOAT, "2.5", 1, 7},
				{COMMA, "", 1, 10},
				{FLOAT, "3.14", 1, 12},
				{RSQUARE, "", 1, 16},
				{EOF, "", 1, 17},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestFloatEdgeCases tests edge cases following Go's lexer behavior
func TestFloatEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "multiple_decimals_go_style",
			input: "3.14.15",
			expected: []tokenExpectation{
				{FLOAT, "3.14", 1, 1},
				{FLOAT, ".15", 1, 5}, // Go treats .15 as valid float
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "double_decimal_go_style",
			input: "3..5",
			expected: []tokenExpectation{
				{FLOAT, "3.", 1, 1}, // Go treats 3. as valid float
				{FLOAT, ".5", 1, 3}, // Go treats .5 as valid float
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "leading_decimal",
			input: ".5",
			expected: []tokenExpectation{
				{FLOAT, ".5", 1, 1}, // Go supports leading decimal
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "trailing_decimal",
			input: "5.",
			expected: []tokenExpectation{
				{FLOAT, "5.", 1, 1}, // Go supports trailing decimal
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "just_decimal",
			input: ".",
			expected: []tokenExpectation{
				// Standalone dot is DOT token for decorator syntax like @aws.secret
				{DOT, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "decimal_in_identifier_context",
			input: "var3.14name",
			expected: []tokenExpectation{
				{IDENTIFIER, "var3", 1, 1},
				{FLOAT, ".14", 1, 5}, // Go handles leading decimal in float
				{IDENTIFIER, "name", 1, 8},
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

// ===== SCIENTIFIC NOTATION TESTS =====
// TODO: Add scientific notation tests

// ===== DURATION TESTS =====
// TODO: Add duration tests
