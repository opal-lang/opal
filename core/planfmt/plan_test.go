package planfmt_test

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/google/go-cmp/cmp"
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
						Tree: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
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
						Tree: &planfmt.CommandNode{
							Decorator: "@retry",
							Block: []planfmt.Step{
								{
									ID:   2,
									Tree: &planfmt.CommandNode{Decorator: "@shell"},
								},
								{
									ID:   2, // Duplicate!
									Tree: &planfmt.CommandNode{Decorator: "@shell"},
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
						Tree: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo"}}, // Out of order!
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
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
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
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
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
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
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

// ============================================================================
// Phase 1: Site-Based Secret Authority - Plan Data Structures
// ============================================================================

// TestPlanSecretUses_RecordsUseSites verifies that SecretUses can be recorded
func TestPlanSecretUses_RecordsUseSites(t *testing.T) {
	plan := &planfmt.Plan{}

	use := planfmt.SecretUse{
		DisplayID: "opal:v:3J98t56A",
		SiteID:    "Xj9K...",
		Site:      "root/retry[0]/params/apiKey",
	}

	err := plan.AddSecretUse(use)
	if err != nil {
		t.Fatalf("AddSecretUse() error = %v", err)
	}

	if len(plan.SecretUses) != 1 {
		t.Errorf("SecretUses length = %d, want 1", len(plan.SecretUses))
	}

	got := plan.SecretUses[0]
	if diff := cmp.Diff(use, got); diff != "" {
		t.Errorf("SecretUse mismatch (-want +got):\n%s", diff)
	}
}

// TestPlanHash_IncludesSecretUses verifies plan hash includes SecretUses
func TestPlanHash_IncludesSecretUses(t *testing.T) {
	plan1 := planfmt.NewPlan()
	plan1.Target = "test"
	plan1.AddSecretUse(planfmt.SecretUse{
		DisplayID: "opal:v:AAA",
		SiteID:    "site-X",
		Site:      "root/retry[0]/params/apiKey",
	})
	plan1.Freeze()

	plan2 := planfmt.NewPlan()
	plan2.Target = "test"
	plan2.AddSecretUse(planfmt.SecretUse{
		DisplayID: "opal:v:BBB",
		SiteID:    "site-Y",
		Site:      "root/timeout[0]/params/duration",
	})
	plan2.Freeze()

	if plan1.Hash == plan2.Hash {
		t.Errorf("Different SecretUses produced same hash: %s", plan1.Hash)
	}

	if plan1.Hash == "" {
		t.Error("plan1.Hash is empty after Freeze()")
	}
	if plan2.Hash == "" {
		t.Error("plan2.Hash is empty after Freeze()")
	}
}

// TestPlanFreeze_PreventsMutation verifies frozen plans reject mutations
func TestPlanFreeze_PreventsMutation(t *testing.T) {
	plan := planfmt.NewPlan()
	plan.Freeze()

	err := plan.AddSecretUse(planfmt.SecretUse{
		DisplayID: "opal:v:ATTACK",
		SiteID:    "malicious",
		Site:      "root/evil[0]/params/stolen",
	})

	if err == nil {
		t.Error("AddSecretUse() on frozen plan should return error, got nil")
	}

	if !contains(err.Error(), "frozen") {
		t.Errorf("AddSecretUse() error = %v, want error containing 'frozen'", err)
	}
}

// TestPlanHash_DetectsTampering verifies hash detects tampering
func TestPlanHash_DetectsTampering(t *testing.T) {
	plan := planfmt.NewPlan()
	plan.Target = "test"
	plan.AddSecretUse(planfmt.SecretUse{
		DisplayID: "opal:v:LEGIT",
		SiteID:    "site-1",
		Site:      "root/retry[0]/params/apiKey",
	})
	plan.Freeze()

	originalHash := plan.Hash

	// Simulate attacker tampering with frozen plan (bypassing AddSecretUse)
	plan.SecretUses = append(plan.SecretUses, planfmt.SecretUse{
		DisplayID: "opal:v:ATTACK",
		SiteID:    "malicious-site",
		Site:      "root/evil[0]/params/stolen",
	})

	// Recompute hash to detect tampering
	currentHash := plan.ComputeHash()

	if originalHash == currentHash {
		t.Error("Hash did not change after tampering with SecretUses")
	}
}

// TestPlanSalt_IsRandom verifies each plan gets unique random salt
func TestPlanSalt_IsRandom(t *testing.T) {
	plan1 := planfmt.NewPlan()
	plan2 := planfmt.NewPlan()

	if len(plan1.PlanSalt) != 32 {
		t.Errorf("plan1.PlanSalt length = %d, want 32 bytes", len(plan1.PlanSalt))
	}

	if len(plan2.PlanSalt) != 32 {
		t.Errorf("plan2.PlanSalt length = %d, want 32 bytes", len(plan2.PlanSalt))
	}

	if cmp.Equal(plan1.PlanSalt, plan2.PlanSalt) {
		t.Error("Two plans have identical PlanSalt (should be random)")
	}
}

// TestPlanHash_ChangesWhenSecretUsesChange verifies hash is sensitive to SecretUses
func TestPlanHash_ChangesWhenSecretUsesChange(t *testing.T) {
	plan := planfmt.NewPlan()
	plan.Target = "test"

	// Hash with no SecretUses
	hash1 := plan.ComputeHash()

	// Add one SecretUse
	plan.AddSecretUse(planfmt.SecretUse{
		DisplayID: "opal:v:AAA",
		SiteID:    "site-1",
		Site:      "root/retry[0]/params/apiKey",
	})
	hash2 := plan.ComputeHash()

	// Add another SecretUse
	plan.AddSecretUse(planfmt.SecretUse{
		DisplayID: "opal:v:BBB",
		SiteID:    "site-2",
		Site:      "root/timeout[0]/params/duration",
	})
	hash3 := plan.ComputeHash()

	if hash1 == hash2 {
		t.Error("Hash did not change after adding first SecretUse")
	}

	if hash2 == hash3 {
		t.Error("Hash did not change after adding second SecretUse")
	}

	if hash1 == hash3 {
		t.Error("Hash with 0 and 2 SecretUses should differ")
	}
}
