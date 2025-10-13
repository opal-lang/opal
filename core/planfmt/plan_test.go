package planfmt_test

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestPlanValidation verifies plan invariants are checked
func TestPlanValidation(t *testing.T) {
	tests := []struct {
		name    string
		plan    *planfmt.Plan
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty plan is valid",
			plan: &planfmt.Plan{
				Target: "deploy",
				Root:   nil,
			},
			wantErr: false,
		},
		{
			name: "single step is valid",
			plan: &planfmt.Plan{
				Target: "deploy",
				Root: &planfmt.Step{
					ID:   1,
					Kind: planfmt.KindDecorator,
					Op:   "shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate step IDs are invalid",
			plan: &planfmt.Plan{
				Target: "deploy",
				Root: &planfmt.Step{
					ID:   1,
					Kind: planfmt.KindDecorator,
					Op:   "retry",
					Children: []*planfmt.Step{
						{ID: 2, Kind: planfmt.KindDecorator, Op: "shell"},
						{ID: 2, Kind: planfmt.KindDecorator, Op: "shell"}, // Duplicate!
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate step ID: 2",
		},
		{
			name: "unsorted args are invalid",
			plan: &planfmt.Plan{
				Target: "deploy",
				Root: &planfmt.Step{
					ID:   1,
					Kind: planfmt.KindDecorator,
					Op:   "shell",
					Args: []planfmt.Arg{
						{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo"}}, // Out of order!
					},
				},
			},
			wantErr: true,
			errMsg:  "args not sorted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
