package planfmt

import (
	"crypto/rand"
	"fmt"

	"github.com/aledsdavies/opal/core/sdk/secret"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

// NewPlanIDFactory creates an IDFactory for deterministic DisplayIDs in resolved plans
// Uses HKDF to derive a plan-specific key from the plan digest
//
// This ensures:
// - Same plan → same DisplayIDs (contract verification works)
// - Different plans → different DisplayIDs (unlinkability)
func NewPlanIDFactory(plan *Plan) (secret.IDFactory, error) {
	// Get plan digest
	digest, err := plan.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to compute plan digest: %w", err)
	}

	// Derive plan_key using HKDF from plan digest
	// This makes DisplayIDs deterministic within a plan but unlinkable across plans
	info := []byte("opal/displayid/plan/v1")
	kdf := hkdf.New(sha3.New256, []byte(digest), nil, info)

	planKey := make([]byte, 32)
	if _, err := kdf.Read(planKey); err != nil {
		return nil, fmt.Errorf("failed to derive plan key: %w", err)
	}

	return secret.NewIDFactory(secret.ModePlan, planKey), nil
}

// NewRunIDFactory creates an IDFactory for random DisplayIDs in direct execution
// Uses a fresh random key for each run
//
// This ensures:
// - Different runs → different DisplayIDs (security)
// - No correlation across runs (prevents tracking)
func NewRunIDFactory() (secret.IDFactory, error) {
	// Generate fresh random key for this run
	runKey := make([]byte, 32)
	if _, err := rand.Read(runKey); err != nil {
		return nil, fmt.Errorf("failed to generate run key: %w", err)
	}

	return secret.NewIDFactory(secret.ModeRun, runKey), nil
}
