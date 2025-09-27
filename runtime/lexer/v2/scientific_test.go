package v2

import (
	"testing"
)

// TestBasicScientificNotation tests valid scientific notation patterns
func TestBasicScientificNotation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple_e_notation",
			input: "1e6",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e6", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "decimal_e_notation",
			input: "2.5e-3",
			expected: []tokenExpectation{
				{SCIENTIFIC, "2.5e-3", 1, 1},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "uppercase_E_notation",
			input: "1.23E+4",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1.23E+4", 1, 1},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "positive_exponent",
			input: "4e+2",
			expected: []tokenExpectation{
				{SCIENTIFIC, "4e+2", 1, 1},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "negative_exponent",
			input: "4e-2",
			expected: []tokenExpectation{
				{SCIENTIFIC, "4e-2", 1, 1},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "capital_E_no_sign",
			input: "1E10",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1E10", 1, 1},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "zero_exponent",
			input: "3.14e0",
			expected: []tokenExpectation{
				{SCIENTIFIC, "3.14e0", 1, 1},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "decimal_start_scientific",
			input: ".5e3",
			expected: []tokenExpectation{
				{SCIENTIFIC, ".5e3", 1, 1},
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

// TestScientificEdgeCases tests Go-compatible error handling
func TestScientificEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "incomplete_exponent",
			input: "1e",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e", 1, 1}, // Go lexer allows this
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "double_e_stops_at_first",
			input: "1ee6",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e", 1, 1}, // Stop at first invalid char
				{IDENTIFIER, "e6", 1, 3}, // Rest becomes identifier
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "e_with_sign_but_no_digits",
			input: "1e+",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e+", 1, 1}, // Go lexer allows this
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "e_with_minus_but_no_digits",
			input: "1e-",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e-", 1, 1}, // Go lexer allows this
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "multiple_dots_stops_at_second",
			input: "1.2.3e4",
			expected: []tokenExpectation{
				{FLOAT, "1.2", 1, 1},       // First number
				{SCIENTIFIC, ".3e4", 1, 4}, // Second number IS scientific notation
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

// TestScientificInContext tests scientific notation in realistic usage
func TestScientificInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "scientific_in_assignment",
			input: "var timeout = 1e6",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "timeout", 1, 5},
				{EQUALS, "", 1, 13},
				{SCIENTIFIC, "1e6", 1, 15},
				{EOF, "", 1, 18},
			},
		},
		{
			name:  "scientific_in_arithmetic",
			input: "total = base + 2.5e-3",
			expected: []tokenExpectation{
				{IDENTIFIER, "total", 1, 1},
				{EQUALS, "", 1, 7},
				{IDENTIFIER, "base", 1, 9},
				{PLUS, "", 1, 14},
				{SCIENTIFIC, "2.5e-3", 1, 16},
				{EOF, "", 1, 22},
			},
		},
		{
			name:  "negative_scientific",
			input: "var small = -1.5e-10",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "small", 1, 5},
				{EQUALS, "", 1, 11},
				{MINUS, "", 1, 13},
				{SCIENTIFIC, "1.5e-10", 1, 14},
				{EOF, "", 1, 21},
			},
		},
		{
			name:  "scientific_with_spaces",
			input: "1e6 + 2.5e-3",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e6", 1, 1},
				{PLUS, "", 1, 5},
				{SCIENTIFIC, "2.5e-3", 1, 7},
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

// TestScientificBoundaries tests that scientific notation stops at correct boundaries
func TestScientificBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "followed_by_identifier",
			input: "1e6microseconds",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e6", 1, 1},
				{IDENTIFIER, "microseconds", 1, 4},
				{EOF, "", 1, 16},
			},
		},
		{
			name:  "followed_by_punctuation",
			input: "1e6,",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e6", 1, 1},
				{COMMA, "", 1, 4},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "followed_by_operator",
			input: "1e6+2",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e6", 1, 1},
				{PLUS, "", 1, 4},
				{INTEGER, "2", 1, 5},
				{EOF, "", 1, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
