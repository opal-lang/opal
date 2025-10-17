package formatter_test

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/planfmt/formatter"
)

// TestFormatCommand verifies command formatting
func TestFormatCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      planfmt.Command
		expected string
	}{
		{
			name: "simple shell command",
			cmd: planfmt.Command{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Hello"`}},
				},
			},
			expected: `@shell echo "Hello"`,
		},
		{
			name: "command with operator (operator handled by FormatStep)",
			cmd: planfmt.Command{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}},
				},
				Operator: "&&",
			},
			expected: `@shell echo "A"`, // Operator not included in command format
		},
		{
			name: "retry decorator",
			cmd: planfmt.Command{
				Decorator: "@retry",
				Args: []planfmt.Arg{
					{Key: "attempts", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 3}},
				},
			},
			expected: `@retry(attempts=3)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatCommand(&tt.cmd)
			if result != tt.expected {
				t.Errorf("FormatCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFormatStep verifies step formatting
func TestFormatStep(t *testing.T) {
	tests := []struct {
		name     string
		step     planfmt.Step
		expected string
	}{
		{
			name: "single command",
			step: planfmt.Step{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Hello"`}},
						},
					},
				},
			},
			expected: `@shell echo "Hello"`,
		},
		{
			name: "chained commands",
			step: planfmt.Step{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}},
						},
						Operator: "&&",
					},
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "B"`}},
						},
					},
				},
			},
			expected: `@shell echo "A" && @shell echo "B"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatStep(&tt.step)
			if result != tt.expected {
				t.Errorf("FormatStep() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFormat verifies full plan formatting
func TestFormat(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Hello"`}},
						},
					},
				},
			},
			{
				ID: 2,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "World"`}},
						},
					},
				},
			},
		},
	}

	expected := `target: hello
step 1: @shell echo "Hello"
step 2: @shell echo "World"
`

	result := formatter.Format(plan)
	if result != expected {
		t.Errorf("Format() mismatch:\nGot:\n%s\nWant:\n%s", result, expected)
	}
}
