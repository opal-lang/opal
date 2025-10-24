package formatter_test

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/planfmt/formatter"
)

// TestFormatStep verifies step formatting with trees
func TestFormatStep(t *testing.T) {
	tests := []struct {
		name     string
		step     planfmt.Step
		expected string
	}{
		{
			name: "simple shell command",
			step: planfmt.Step{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Hello"`}},
					},
				},
			},
			expected: `@shell echo "Hello"`,
		},
		{
			name: "command with AND operator",
			step: planfmt.Step{
				ID: 1,
				Tree: &planfmt.AndNode{
					Left: &planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}},
						},
					},
					Right: &planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "B"`}},
						},
					},
				},
			},
			expected: `@shell echo "A" && @shell echo "B"`,
		},
		{
			name: "pipeline",
			step: planfmt.Step{
				ID: 1,
				Tree: &planfmt.PipelineNode{
					Commands: []planfmt.ExecutionNode{
						&planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
							},
						},
						&planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep hello"}},
							},
						},
					},
				},
			},
			expected: `@shell echo hello | @shell grep hello`,
		},
		{
			name: "retry decorator",
			step: planfmt.Step{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@retry",
					Args: []planfmt.Arg{
						{Key: "attempts", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 3}},
					},
				},
			},
			expected: `@retry(attempts=3)`,
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
		Target: "deploy",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "npm build"}},
					},
				},
			},
			{
				ID: 2,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "docker build"}},
					},
				},
			},
		},
	}

	result := formatter.Format(plan)
	expected := "target: deploy\nstep 1: @shell npm build\nstep 2: @shell docker build\n"

	if result != expected {
		t.Errorf("Format() mismatch\nGot:\n%s\nWant:\n%s", result, expected)
	}
}
