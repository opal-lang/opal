package planner

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestBuildStepTree tests tree building from flat command lists
// Following TDD: test complete exact tree structure, not partial matches
func TestBuildStepTree(t *testing.T) {
	tests := []struct {
		name     string
		commands []Command
		want     planfmt.ExecutionNode
	}{
		{
			name: "single command",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.CommandNode{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
				},
			},
		},
		{
			name: "two commands with AND",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo first"}},
					},
					Operator: "&&",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo second"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.AndNode{
				Left: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo first"}},
					},
				},
				Right: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo second"}},
					},
				},
			},
		},
		{
			name: "two commands with OR",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo first"}},
					},
					Operator: "||",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo second"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.OrNode{
				Left: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo first"}},
					},
				},
				Right: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo second"}},
					},
				},
			},
		},
		{
			name: "two commands with pipe",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep test"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.PipelineNode{
				Commands: []planfmt.CommandNode{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
						},
					},
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep test"}},
						},
					},
				},
			},
		},
		{
			name: "three commands with chained pipes",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep test"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "wc -l"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.PipelineNode{
				Commands: []planfmt.CommandNode{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
						},
					},
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep test"}},
						},
					},
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "wc -l"}},
						},
					},
				},
			},
		},
		{
			name: "operator precedence: pipe > AND",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
					},
					Operator: "&&",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.AndNode{
				Left: &planfmt.PipelineNode{
					Commands: []planfmt.CommandNode{
						{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
							},
						},
						{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
							},
						},
					},
				},
				Right: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
				},
			},
		},
		{
			name: "operator precedence: AND > OR",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: "&&",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: "||",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.OrNode{
				Left: &planfmt.AndNode{
					Left: &planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
						},
					},
					Right: &planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
						},
					},
				},
				Right: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
					},
				},
			},
		},
		{
			name: "complex precedence: pipe > AND > OR",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
					},
					Operator: "&&",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep b"}},
					},
					Operator: "||",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo fallback"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.OrNode{
				Left: &planfmt.AndNode{
					Left: &planfmt.PipelineNode{
						Commands: []planfmt.CommandNode{
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
								},
							},
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
								},
							},
						},
					},
					Right: &planfmt.PipelineNode{
						Commands: []planfmt.CommandNode{
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
								},
							},
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep b"}},
								},
							},
						},
					},
				},
				Right: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo fallback"}},
					},
				},
			},
		},
		{
			name: "semicolon operator (lowest precedence)",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: ";",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: "&&",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					&planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
						},
					},
					&planfmt.AndNode{
						Left: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
							},
						},
						Right: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
							},
						},
					},
				},
			},
		},
		{
			name: "BUG: semicolon with AND causes infinite loop",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: "&&",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: ";",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					&planfmt.AndNode{
						Left: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
							},
						},
						Right: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
							},
						},
					},
					&planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
						},
					},
				},
			},
		},
		{
			name: "BUG: semicolon with OR causes infinite loop",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: "||",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: ";",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					&planfmt.OrNode{
						Left: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
							},
						},
						Right: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
							},
						},
					},
					&planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
						},
					},
				},
			},
		},
		{
			name: "BUG: semicolon with pipe causes infinite loop",
			commands: []Command{
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
					},
					Operator: "|",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
					},
					Operator: ";",
				},
				{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
					},
					Operator: "",
				},
			},
			want: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					&planfmt.PipelineNode{
						Commands: []planfmt.CommandNode{
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
								},
							},
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
								},
							},
						},
					},
					&planfmt.CommandNode{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStepTree(tt.commands)

			// Use cmp.Diff for complete exact tree comparison
			// Allow unexported fields comparison for ExecutionNode interface types
			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(planfmt.CommandNode{}),
				cmpopts.IgnoreUnexported(planfmt.PipelineNode{}),
				cmpopts.IgnoreUnexported(planfmt.AndNode{}),
				cmpopts.IgnoreUnexported(planfmt.OrNode{}),
				cmpopts.IgnoreUnexported(planfmt.SequenceNode{}),
			}

			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("buildStepTree() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
