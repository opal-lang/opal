package planfmt

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"

	"github.com/aledsdavies/opal/core/sdk/secret"
)

// NewPlanIDFactory creates an IDFactory for deterministic DisplayIDs in resolved plans.
// Uses canonical plan hash (structure only, no DisplayIDs) to break circular dependency.
// DisplayIDs need plan_hash, but plan_hash can't include DisplayIDs that don't exist yet.
//
// This ensures:
// - Same plan structure → same DisplayIDs (contract verification works)
// - Different plans → different DisplayIDs (unlinkability)
func NewPlanIDFactory(plan *Plan) (secret.IDFactory, error) {
	// Get canonical plan hash (structure only, NO DisplayIDs)
	// This breaks the circular dependency: DisplayIDs need plan_hash, but plan_hash can't include DisplayIDs
	canonical, err := plan.Canonicalize()
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize plan: %w", err)
	}

	planHash, err := canonical.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to compute canonical hash: %w", err)
	}

	// Derive plan_key using HKDF from canonical hash
	// This makes DisplayIDs deterministic within a plan but unlinkable across plans
	info := []byte("opal/displayid/plan/v1")
	kdf := hkdf.New(sha3.New256, planHash[:], nil, info)

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
