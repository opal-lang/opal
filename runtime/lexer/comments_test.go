package lexer

import (
	"testing"
)

// TestLineComments tests // style comments
func TestLineComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple line comment",
			input: "// hello world",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " hello world",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 15,
				},
			},
		},
		{
			name:  "line comment without space",
			input: "//comment",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: "comment",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 10,
				},
			},
		},
		{
			name:  "empty line comment",
			input: "//",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: "",
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
			name:  "line comment with code before",
			input: "var x = 5 // set initial value",
			expected: []tokenExpectation{
				{
					Type: VAR, Text: "var",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 5,
				},
				{
					Type: EQUALS, Text: "",
					Line:   1,
					Column: 7,
				},
				{
					Type: INTEGER, Text: "5",
					Line:   1,
					Column: 9,
				},
				{
					Type: COMMENT, Text: " set initial value",
					Line:   1,
					Column: 11,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 31,
				},
			},
		},
		{
			name:  "line comment with newline",
			input: "// first line\nvar x = 1",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " first line",
					Line:   1,
					Column: 1,
				},
				{
					Type: NEWLINE, Text: "",
					Line:   1,
					Column: 14,
				},
				{
					Type: VAR, Text: "var",
					Line:   2,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "x",
					Line:   2,
					Column: 5,
				},
				{
					Type: EQUALS, Text: "",
					Line:   2,
					Column: 7,
				},
				{
					Type: INTEGER, Text: "1",
					Line:   2,
					Column: 9,
				},
				{
					Type: EOF, Text: "",
					Line:   2,
					Column: 10,
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

// TestBlockComments tests /* */ style comments with content extraction
func TestBlockComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "simple block comment",
			input: "/* hello world */",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " hello world ",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 18,
				},
			},
		},
		{
			name:  "block comment without spaces",
			input: "/*comment*/",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: "comment",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 12,
				},
			},
		},
		{
			name:  "empty block comment",
			input: "/**/",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: "",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 5,
				},
			},
		},
		{
			name:  "multiline block comment",
			input: "/* line 1\nline 2\nline 3 */",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " line 1\nline 2\nline 3 ",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   3,
					Column: 10,
				},
			},
		},
		{
			name:  "block comment with code before and after",
			input: "var x = /* initial */ 5",
			expected: []tokenExpectation{
				{
					Type: VAR, Text: "var",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 5,
				},
				{
					Type: EQUALS, Text: "",
					Line:   1,
					Column: 7,
				},
				{
					Type: COMMENT, Text: " initial ",
					Line:   1,
					Column: 9,
				},
				{
					Type: INTEGER, Text: "5",
					Line:   1,
					Column: 23,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 24,
				},
			},
		},
		{
			name:  "nested-looking comment content",
			input: "/* outer /* inner */ content */",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " outer /* inner ",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "content",
					Line:   1,
					Column: 22,
				},
				{
					Type: MULTIPLY, Text: "",
					Line:   1,
					Column: 30,
				},
				{
					Type: DIVIDE, Text: "",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestCommentEdgeCases tests edge cases and error conditions
func TestCommentEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "single slash not comment",
			input: "/",
			expected: []tokenExpectation{
				{
					Type: DIVIDE, Text: "",
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
			name:  "single star not comment",
			input: "*",
			expected: []tokenExpectation{
				{
					Type: MULTIPLY, Text: "",
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
			name:  "unterminated block comment",
			input: "/* unterminated",
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " unterminated",
					Line:   1,
					Column: 1,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 16,
				},
			},
		},
		{
			name:  "block comment at EOF",
			input: "var x /*end*/",
			expected: []tokenExpectation{
				{
					Type: VAR, Text: "var",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 5,
				},
				{
					Type: COMMENT, Text: "end",
					Line:   1,
					Column: 7,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 14,
				},
			},
		},
		{
			name:  "multiple comments same line",
			input: "x // comment 1\ny /* comment 2 */ z",
			expected: []tokenExpectation{
				{
					Type: IDENTIFIER, Text: "x",
					Line:   1,
					Column: 1,
				},
				{
					Type: COMMENT, Text: " comment 1",
					Line:   1,
					Column: 3,
				},
				{
					Type: NEWLINE, Text: "",
					Line:   1,
					Column: 15,
				},
				{
					Type: IDENTIFIER, Text: "y",
					Line:   2,
					Column: 1,
				},
				{
					Type: COMMENT, Text: " comment 2 ",
					Line:   2,
					Column: 3,
				},
				{
					Type: IDENTIFIER, Text: "z",
					Line:   2,
					Column: 19,
				},
				{
					Type: EOF, Text: "",
					Line:   2,
					Column: 20,
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

// TestCommentsInDevcmdContext tests comments in realistic opal scenarios
func TestCommentsInDevcmdContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable with comment",
			input: "var replicas = 3 // default replica count",
			expected: []tokenExpectation{
				{
					Type: VAR, Text: "var",
					Line:   1,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "replicas",
					Line:   1,
					Column: 5,
				},
				{
					Type: EQUALS, Text: "",
					Line:   1,
					Column: 14,
				},
				{
					Type: INTEGER, Text: "3",
					Line:   1,
					Column: 16,
				},
				{
					Type: COMMENT, Text: " default replica count",
					Line:   1,
					Column: 18,
				},
				{
					Type: EOF, Text: "",
					Line:   1,
					Column: 42,
				},
			},
		},
		{
			name:  "conditional with block comment",
			input: "if /* production check */ env == \"prod\" {",
			expected: []tokenExpectation{
				{
					Type: IF, Text: "if",
					Line:   1,
					Column: 1,
				},
				{
					Type: COMMENT, Text: " production check ",
					Line:   1,
					Column: 4,
				},
				{
					Type: IDENTIFIER, Text: "env",
					Line:   1,
					Column: 27,
				},
				{
					Type: EQ_EQ, Text: "",
					Line:   1,
					Column: 31,
				},
				{Type: STRING, Text: "\"prod\"", Line: 1, Column: 34},
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
		{
			name: "multiline script with comments",
			input: `// Deploy script
var timeout = 30s /* deployment timeout */
if replicas >= 3 { // production ready`,
			expected: []tokenExpectation{
				{
					Type: COMMENT, Text: " Deploy script",
					Line:   1,
					Column: 1,
				},
				{
					Type: NEWLINE, Text: "",
					Line:   1,
					Column: 17,
				},
				{
					Type: VAR, Text: "var",
					Line:   2,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "timeout",
					Line:   2,
					Column: 5,
				},
				{
					Type: EQUALS, Text: "",
					Line:   2,
					Column: 13,
				},
				{
					Type: DURATION, Text: "30s",
					Line:   2,
					Column: 15,
				},
				{
					Type: COMMENT, Text: " deployment timeout ",
					Line:   2,
					Column: 19,
				},
				{
					Type: NEWLINE, Text: "",
					Line:   2,
					Column: 43,
				},
				{
					Type: IF, Text: "if",
					Line:   3,
					Column: 1,
				},
				{
					Type: IDENTIFIER, Text: "replicas",
					Line:   3,
					Column: 4,
				},
				{
					Type: GT_EQ, Text: "",
					Line:   3,
					Column: 13,
				},
				{
					Type: INTEGER, Text: "3",
					Line:   3,
					Column: 16,
				},
				{
					Type: LBRACE, Text: "",
					Line:   3,
					Column: 18,
				},
				{
					Type: COMMENT, Text: " production ready",
					Line:   3,
					Column: 20,
				},
				{
					Type: EOF, Text: "",
					Line:   3,
					Column: 39,
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
