package formatter_test

import (
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/core/planfmt/formatter"
)

// TestDiff verifies diff comparison logic
func TestDiff(t *testing.T) {
	tests := []struct {
		name             string
		expected         *planfmt.Plan
		actual           *planfmt.Plan
		wantAdded        int
		wantRemoved      int
		wantModified     int
		wantTargetChange bool
	}{
		{
			name: "identical plans",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
				},
			},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name: "step modified",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "B"`}}}}},
				},
			},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 1,
		},
		{
			name: "step added",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
					{ID: 2, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "B"`}}}}},
				},
			},
			wantAdded:    1,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name: "step removed",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
					{ID: 2, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "B"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "A"`}}}}},
				},
			},
			wantAdded:    0,
			wantRemoved:  1,
			wantModified: 0,
		},
		{
			name: "target changed",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps:  []planfmt.Step{},
			},
			actual: &planfmt.Plan{
				Target: "goodbye",
				Steps:  []planfmt.Step{},
			},
			wantAdded:        0,
			wantRemoved:      0,
			wantModified:     0,
			wantTargetChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Diff(tt.expected, tt.actual)

			if len(result.Added) != tt.wantAdded {
				t.Errorf("Added count = %d, want %d", len(result.Added), tt.wantAdded)
			}
			if len(result.Removed) != tt.wantRemoved {
				t.Errorf("Removed count = %d, want %d", len(result.Removed), tt.wantRemoved)
			}
			if len(result.Modified) != tt.wantModified {
				t.Errorf("Modified count = %d, want %d", len(result.Modified), tt.wantModified)
			}
			if (result.TargetChanged != "") != tt.wantTargetChange {
				t.Errorf("TargetChanged = %q, want change=%v", result.TargetChanged, tt.wantTargetChange)
			}
		})
	}
}

// TestFormatDiff verifies diff display formatting
func TestFormatDiff(t *testing.T) {
	tests := []struct {
		name     string
		expected *planfmt.Plan
		actual   *planfmt.Plan
		want     string
	}{
		{
			name: "modified step",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Old"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "New"`}}}}},
				},
			},
			want: `Modified steps:
  step 1:
    - @shell echo "Old"
    + @shell echo "New"

`,
		},
		{
			name: "added step",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps:  []planfmt.Step{},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "New"`}}}}},
				},
			},
			want: `Added steps:
  + step 1: @shell echo "New"

`,
		},
		{
			name: "removed step",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Old"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps:  []planfmt.Step{},
			},
			want: `Removed steps:
  - step 1: @shell echo "Old"

`,
		},
		{
			name: "no differences",
			expected: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Same"`}}}}},
				},
			},
			actual: &planfmt.Plan{
				Target: "hello",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Same"`}}}}},
				},
			},
			want: `No differences found.
`,
		},
		{
			name: "call trace modified",
			expected: &planfmt.Plan{
				Target: "deploy",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.LogicNode{Kind: "call", Condition: "deploy(prod)"}},
				},
			},
			actual: &planfmt.Plan{
				Target: "deploy",
				Steps: []planfmt.Step{
					{ID: 1, Tree: &planfmt.LogicNode{Kind: "call", Condition: "deploy(prod, token=opal:abc123)"}},
				},
			},
			want: `Modified steps:
  step 1:
    - deploy(prod)
    + deploy(prod, token=opal:abc123)

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := formatter.Diff(tt.expected, tt.actual)
			got := formatter.FormatDiff(diff, false) // no color

			if got != tt.want {
				t.Errorf("FormatDiff() mismatch:\nGot:\n%s\nWant:\n%s", got, tt.want)
			}
		})
	}
}
