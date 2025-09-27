package v2

import (
	"testing"
)

// TestBasicIdentifiers tests simple identifier tokenization
func TestBasicIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple_identifier",
			input: "myVar",
			expected: []tokenExpectation{
				{IDENTIFIER, "myVar", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "underscore_identifier",
			input: "my_var",
			expected: []tokenExpectation{
				{IDENTIFIER, "my_var", 1, 1},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "number_suffix",
			input: "var123",
			expected: []tokenExpectation{
				{IDENTIFIER, "var123", 1, 1},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "underscore_start",
			input: "_private",
			expected: []tokenExpectation{
				{IDENTIFIER, "_private", 1, 1},
				{EOF, "", 1, 9},
			},
		},
		{
			name:  "mixed_case",
			input: "deployToProduction",
			expected: []tokenExpectation{
				{IDENTIFIER, "deployToProduction", 1, 1},
				{EOF, "", 1, 19},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestIdentifierStyles tests different naming conventions
func TestIdentifierStyles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "camelCase",
			input: "apiUrl",
			expected: []tokenExpectation{
				{IDENTIFIER, "apiUrl", 1, 1},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "snake_case",
			input: "api_url",
			expected: []tokenExpectation{
				{IDENTIFIER, "api_url", 1, 1},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "PascalCase",
			input: "ApiUrl",
			expected: []tokenExpectation{
				{IDENTIFIER, "ApiUrl", 1, 1},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "SCREAMING_SNAKE",
			input: "API_URL",
			expected: []tokenExpectation{
				{IDENTIFIER, "API_URL", 1, 1},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "numbers_mixed",
			input: "service2_url",
			expected: []tokenExpectation{
				{IDENTIFIER, "service2_url", 1, 1},
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

// TestIdentifierBoundaries tests identifier boundaries and separation
func TestIdentifierBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "space_separated",
			input: "var1 var2",
			expected: []tokenExpectation{
				{IDENTIFIER, "var1", 1, 1},
				{IDENTIFIER, "var2", 1, 6},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "punctuation_separated",
			input: "var1=var2",
			expected: []tokenExpectation{
				{IDENTIFIER, "var1", 1, 1},
				{EQUALS, "", 1, 5},
				{IDENTIFIER, "var2", 1, 6},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "newline_separated",
			input: "var1\nvar2",
			expected: []tokenExpectation{
				{IDENTIFIER, "var1", 1, 1},
				{NEWLINE, "", 1, 5},
				{IDENTIFIER, "var2", 2, 1},
				{EOF, "", 2, 5},
			},
		},
		{
			name:  "multiple_underscores",
			input: "var__name",
			expected: []tokenExpectation{
				{IDENTIFIER, "var__name", 1, 1},
				{EOF, "", 1, 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestIdentifierVsKeyword tests that identifiers containing keywords are handled correctly
func TestIdentifierVsKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "var_prefix",
			input: "variable",
			expected: []tokenExpectation{
				{IDENTIFIER, "variable", 1, 1},
				{EOF, "", 1, 9},
			},
		},
		{
			name:  "var_suffix",
			input: "myvar",
			expected: []tokenExpectation{
				{IDENTIFIER, "myvar", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "var_middle",
			input: "service_var_name",
			expected: []tokenExpectation{
				{IDENTIFIER, "service_var_name", 1, 1},
				{EOF, "", 1, 17},
			},
		},
		{
			name:  "multiple_keywords",
			input: "forwardIfTry",
			expected: []tokenExpectation{
				{IDENTIFIER, "forwardIfTry", 1, 1},
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

// TestIdentifierInContext tests identifiers in realistic usage
func TestIdentifierInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable_declaration",
			input: "var myService = value",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "myService", 1, 5},
				{EQUALS, "", 1, 15},
				{IDENTIFIER, "value", 1, 17},
				{EOF, "", 1, 22},
			},
		},
		{
			name:  "command_definition",
			input: "deployToProduction:",
			expected: []tokenExpectation{
				{IDENTIFIER, "deployToProduction", 1, 1},
				{COLON, "", 1, 19},
				{EOF, "", 1, 20},
			},
		},
		{
			name:  "function_call_style",
			input: "checkHealth()",
			expected: []tokenExpectation{
				{IDENTIFIER, "checkHealth", 1, 1},
				{LPAREN, "", 1, 12},
				{RPAREN, "", 1, 13},
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
