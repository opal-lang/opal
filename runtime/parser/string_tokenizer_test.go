package parser

import (
	"testing"

	"github.com/opal-lang/opal/core/types"
)

func init() {
	// Register a test execution decorator to verify it doesn't interpolate
	dummyHandler := func(ctx types.Context, args types.Args) error {
		return nil
	}
	types.Global().RegisterExecution("test_exec", dummyHandler)
}

func TestTokenizeString(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		quoteType byte
		expected  []StringPart
	}{
		{
			name:      "single quote no interpolation",
			content:   "Hello @var.name",
			quoteType: '\'',
			expected: []StringPart{
				{Start: 0, End: 15, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},
			},
		},
		{
			name:      "double quote with var interpolation",
			content:   "Hello @var.name",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 6, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},   // "Hello "
				{Start: 7, End: 10, IsLiteral: false, PropertyStart: 11, PropertyEnd: 15}, // @var.name
			},
		},
		{
			name:      "double quote with env interpolation",
			content:   "Path: @env.HOME/config",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 6, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},   // "Path: "
				{Start: 7, End: 10, IsLiteral: false, PropertyStart: 11, PropertyEnd: 15}, // @env.HOME
				{Start: 15, End: 22, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}, // "/config"
			},
		},
		{
			name:      "unregistered decorator stays literal",
			content:   "Email: user@example.com",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 23, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},
			},
		},
		{
			name:      "multiple interpolations",
			content:   "@var.first and @var.second",
			quoteType: '"',
			expected: []StringPart{
				{Start: 1, End: 4, IsLiteral: false, PropertyStart: 5, PropertyEnd: 10},    // @var.first
				{Start: 10, End: 15, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},  // " and "
				{Start: 16, End: 19, IsLiteral: false, PropertyStart: 20, PropertyEnd: 26}, // @var.second
			},
		},
		{
			name:      "decorator without property",
			content:   "Value: @var",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 7, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},   // "Value: "
				{Start: 8, End: 11, IsLiteral: false, PropertyStart: -1, PropertyEnd: -1}, // @var
			},
		},
		{
			name:      "@ at end of string",
			content:   "Email @",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 7, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}, // "Email @" (merged)
			},
		},
		{
			name:      "@ with no identifier",
			content:   "Symbol @ here",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 13, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}, // "Symbol @ here" (merged)
			},
		},
		{
			name:      "backtick with interpolation",
			content:   "Deploy to @env.ENVIRONMENT",
			quoteType: '`',
			expected: []StringPart{
				{Start: 0, End: 10, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},   // "Deploy to "
				{Start: 11, End: 14, IsLiteral: false, PropertyStart: 15, PropertyEnd: 26}, // @env.ENVIRONMENT
			},
		},
		{
			name:      "empty string",
			content:   "",
			quoteType: '"',
			expected:  nil,
		},
		{
			name:      "only literal",
			content:   "no decorators here",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 18, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},
			},
		},
		{
			name:      "decorator at start",
			content:   "@var.name is here",
			quoteType: '"',
			expected: []StringPart{
				{Start: 1, End: 4, IsLiteral: false, PropertyStart: 5, PropertyEnd: 9},   // @var.name
				{Start: 9, End: 17, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}, // " is here"
			},
		},
		{
			name:      "execution decorator stays literal",
			content:   "Running @test_exec.cmd('ls')",
			quoteType: '"',
			expected: []StringPart{
				{Start: 0, End: 28, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}, // Entire string is literal
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := TokenizeString([]byte(tt.content), tt.quoteType)

			if len(parts) != len(tt.expected) {
				t.Fatalf("expected %d parts, got %d\nExpected: %+v\nGot: %+v",
					len(tt.expected), len(parts), tt.expected, parts)
			}

			for i, part := range parts {
				exp := tt.expected[i]
				if part.Start != exp.Start {
					t.Errorf("part %d: expected Start=%d, got %d", i, exp.Start, part.Start)
				}
				if part.End != exp.End {
					t.Errorf("part %d: expected End=%d, got %d", i, exp.End, part.End)
				}
				if part.IsLiteral != exp.IsLiteral {
					t.Errorf("part %d: expected IsLiteral=%v, got %v", i, exp.IsLiteral, part.IsLiteral)
				}
				if part.PropertyStart != exp.PropertyStart {
					t.Errorf("part %d: expected PropertyStart=%d, got %d", i, exp.PropertyStart, part.PropertyStart)
				}
				if part.PropertyEnd != exp.PropertyEnd {
					t.Errorf("part %d: expected PropertyEnd=%d, got %d", i, exp.PropertyEnd, part.PropertyEnd)
				}

				// Verify the actual content matches
				if part.IsLiteral {
					actualText := string([]byte(tt.content)[part.Start:part.End])
					t.Logf("part %d (literal): %q", i, actualText)
				} else {
					decoratorName := string([]byte(tt.content)[part.Start:part.End])
					t.Logf("part %d (decorator): %q", i, decoratorName)
					if part.PropertyStart != -1 {
						propertyName := string([]byte(tt.content)[part.PropertyStart:part.PropertyEnd])
						t.Logf("  property: %q", propertyName)
					}
				}
			}
		})
	}
}
