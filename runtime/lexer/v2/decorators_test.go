package v2

import "testing"

// TestBasicDecorators tests @ symbol and basic decorator syntax
func TestBasicDecorators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "at symbol only",
			input: "@",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{EOF, "", 1, 2},
			},
		},
		{
			name:  "var decorator",
			input: "@var",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{VAR, "var", 1, 2},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "env decorator",
			input: "@env",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "env", 1, 2},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "retry decorator",
			input: "@retry",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "retry", 1, 2},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "timeout decorator",
			input: "@timeout",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "timeout", 1, 2},
				{EOF, "", 1, 9},
			},
		},
		{
			name:  "parallel decorator",
			input: "@parallel",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "parallel", 1, 2},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "shell decorator",
			input: "@shell",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "shell", 1, 2},
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

// TestComplexDecorators tests decorators with dot notation and namespaces
func TestComplexDecorators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "aws secret",
			input: "@aws.secret",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "aws", 1, 2},
				{DOT, "", 1, 5},
				{IDENTIFIER, "secret", 1, 6},
				{EOF, "", 1, 12},
			},
		},
		{
			name:  "http get",
			input: "@http.get",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "http", 1, 2},
				{DOT, "", 1, 6},
				{IDENTIFIER, "get", 1, 7},
				{EOF, "", 1, 10},
			},
		},
		{
			name:  "random password",
			input: "@random.password",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "random", 1, 2},
				{DOT, "", 1, 8},
				{IDENTIFIER, "password", 1, 9},
				{EOF, "", 1, 17},
			},
		},
		{
			name:  "crypto generate key",
			input: "@crypto.generate_key",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "crypto", 1, 2},
				{DOT, "", 1, 8},
				{IDENTIFIER, "generate_key", 1, 9},
				{EOF, "", 1, 21},
			},
		},
		{
			name:  "aws ec2 deploy",
			input: "@aws.ec2.deploy",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "aws", 1, 2},
				{DOT, "", 1, 5},
				{IDENTIFIER, "ec2", 1, 6},
				{DOT, "", 1, 9},
				{IDENTIFIER, "deploy", 1, 10},
				{EOF, "", 1, 16},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestDecoratorWithArguments tests decorators with parentheses and arguments
func TestDecoratorWithArguments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "var with argument",
			input: "@var(PORT)",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{VAR, "var", 1, 2},
				{LPAREN, "", 1, 5},
				{IDENTIFIER, "PORT", 1, 6},
				{RPAREN, "", 1, 10},
				{EOF, "", 1, 11},
			},
		},
		{
			name:  "env with string argument",
			input: `@env("DATABASE_URL")`,
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "env", 1, 2},
				{LPAREN, "", 1, 5},
				{STRING, `"DATABASE_URL"`, 1, 6},
				{RPAREN, "", 1, 20},
				{EOF, "", 1, 21},
			},
		},
		{
			name:  "env with default",
			input: `@env("PORT", default=3000)`,
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "env", 1, 2},
				{LPAREN, "", 1, 5},
				{STRING, `"PORT"`, 1, 6},
				{COMMA, "", 1, 12},
				{IDENTIFIER, "default", 1, 14},
				{EQUALS, "", 1, 21},
				{INTEGER, "3000", 1, 22},
				{RPAREN, "", 1, 26},
				{EOF, "", 1, 27},
			},
		},
		{
			name:  "retry with duration",
			input: "@retry(attempts=3, delay=10s)",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "retry", 1, 2},
				{LPAREN, "", 1, 7},
				{IDENTIFIER, "attempts", 1, 8},
				{EQUALS, "", 1, 16},
				{INTEGER, "3", 1, 17},
				{COMMA, "", 1, 18},
				{IDENTIFIER, "delay", 1, 20},
				{EQUALS, "", 1, 25},
				{DURATION, "10s", 1, 26},
				{RPAREN, "", 1, 29},
				{EOF, "", 1, 30},
			},
		},
		{
			name:  "timeout with duration",
			input: "@timeout(30m)",
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "timeout", 1, 2},
				{LPAREN, "", 1, 9},
				{DURATION, "30m", 1, 10},
				{RPAREN, "", 1, 13},
				{EOF, "", 1, 14},
			},
		},
		{
			name:  "aws secret with key",
			input: `@aws.secret("api-token")`,
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "aws", 1, 2},
				{DOT, "", 1, 5},
				{IDENTIFIER, "secret", 1, 6},
				{LPAREN, "", 1, 12},
				{STRING, `"api-token"`, 1, 13},
				{RPAREN, "", 1, 24},
				{EOF, "", 1, 25},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestDecoratorInContext tests decorators within larger opal expressions
func TestDecoratorInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable assignment with env decorator",
			input: `var PORT = @env("PORT", default=3000)`,
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "PORT", 1, 5},
				{EQUALS, "", 1, 10},
				{AT, "", 1, 12},
				{IDENTIFIER, "env", 1, 13},
				{LPAREN, "", 1, 16},
				{STRING, `"PORT"`, 1, 17},
				{COMMA, "", 1, 23},
				{IDENTIFIER, "default", 1, 25},
				{EQUALS, "", 1, 32},
				{INTEGER, "3000", 1, 33},
				{RPAREN, "", 1, 37},
				{EOF, "", 1, 38},
			},
		},
		{
			name:  "shell command with var interpolation",
			input: `kubectl scale --replicas=@var(REPLICAS)`,
			expected: []tokenExpectation{
				{IDENTIFIER, "kubectl", 1, 1},
				{IDENTIFIER, "scale", 1, 9},
				{DECREMENT, "", 1, 15},
				{IDENTIFIER, "replicas", 1, 17},
				{EQUALS, "", 1, 25},
				{AT, "", 1, 26},
				{VAR, "var", 1, 27},
				{LPAREN, "", 1, 30},
				{IDENTIFIER, "REPLICAS", 1, 31},
				{RPAREN, "", 1, 39},
				{EOF, "", 1, 40},
			},
		},
		{
			name:  "decorator in conditional",
			input: `if @var(ENV) == "production" { }`,
			expected: []tokenExpectation{
				{IF, "if", 1, 1},
				{AT, "", 1, 4},
				{VAR, "var", 1, 5},
				{LPAREN, "", 1, 8},
				{IDENTIFIER, "ENV", 1, 9},
				{RPAREN, "", 1, 12},
				{EQ_EQ, "", 1, 14},
				{STRING, `"production"`, 1, 17},
				{LBRACE, "", 1, 30},
				{RBRACE, "", 1, 32},
				{EOF, "", 1, 33},
			},
		},
		{
			name:  "shell decorator call",
			input: `@shell("echo hello")`,
			expected: []tokenExpectation{
				{AT, "", 1, 1},
				{IDENTIFIER, "shell", 1, 2},
				{LPAREN, "", 1, 7},
				{STRING, `"echo hello"`, 1, 8},
				{RPAREN, "", 1, 20},
				{EOF, "", 1, 21},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
