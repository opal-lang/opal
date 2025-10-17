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
				Steps:  nil,
			},
			wantErr: false,
		},
		{
			name: "single step is valid",
			plan: &planfmt.Plan{
				Target: "deploy",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Commands: []planfmt.Command{
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate step IDs are invalid",
			plan: &planfmt.Plan{
				Target: "deploy",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Commands: []planfmt.Command{
							{
								Decorator: "@retry",
								Block: []planfmt.Step{
									{
										ID: 2,
										Commands: []planfmt.Command{
											{Decorator: "@shell"},
										},
									},
									{
										ID: 2, // Duplicate!
										Commands: []planfmt.Command{
											{Decorator: "@shell"},
										},
									},
								},
							},
						},
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
				Steps: []planfmt.Step{
					{
						ID: 1,
						Commands: []planfmt.Command{
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo"}}, // Out of order!
								},
							},
						},
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

// TestPlanDigest tests plan integrity hashing
func TestPlanDigest(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	digest1, err := plan.Digest()
	if err != nil {
		t.Fatalf("Digest() error = %v", err)
	}
	if digest1 == "" {
		t.Error("Digest() returned empty string")
	}
	if len(digest1) < 10 {
		t.Errorf("Digest() too short: %s", digest1)
	}

	// Same plan should produce same digest
	digest2, err := plan.Digest()
	if err != nil {
		t.Fatalf("Digest() error = %v", err)
	}
	if digest1 != digest2 {
		t.Errorf("Same plan produced different digests: %s != %s", digest1, digest2)
	}
}

// TestPlanDigestDifferentPlans tests that different plans produce different digests
func TestPlanDigestDifferentPlans(t *testing.T) {
	plan1 := &planfmt.Plan{
		Target: "test1",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	plan2 := &planfmt.Plan{
		Target: "test2", // Different target
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	digest1, err := plan1.Digest()
	if err != nil {
		t.Fatalf("plan1.Digest() error = %v", err)
	}

	digest2, err := plan2.Digest()
	if err != nil {
		t.Fatalf("plan2.Digest() error = %v", err)
	}

	if digest1 == digest2 {
		t.Errorf("Different plans produced same digest: %s", digest1)
	}
}
