package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test 1: Resolved plan determinism - same plan twice → identical DisplayIDs
func TestResolvedPlanDeterminism(t *testing.T) {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}

	factory1 := NewIDFactory(ModePlan, planKey)
	factory2 := NewIDFactory(ModePlan, planKey)

	ctx := IDContext{
		PlanHash:  []byte("plan-hash-abc123"),
		StepPath:  "deploy.step[0]",
		Decorator: "@aws.secret",
		KeyName:   "DB_PASS",
		Kind:      "s",
	}

	value := []byte("my-secret-password")

	id1 := factory1.Make(ctx, value)
	id2 := factory2.Make(ctx, value)

	assert.Equal(t, id1, id2, "Same plan context and value should produce identical DisplayIDs")
	assert.Contains(t, id1, "opal:s:", "DisplayID should have correct format")
}

// Test 2: Plan boundary unlinkability - same value in different plans → different IDs
func TestPlanBoundaryUnlinkability(t *testing.T) {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}

	factory := NewIDFactory(ModePlan, planKey)

	ctx1 := IDContext{
		PlanHash:  []byte("plan-hash-abc123"),
		StepPath:  "deploy.step[0]",
		Decorator: "@aws.secret",
		KeyName:   "DB_PASS",
		Kind:      "s",
	}

	ctx2 := IDContext{
		PlanHash:  []byte("plan-hash-xyz789"), // Different plan hash
		StepPath:  "deploy.step[0]",
		Decorator: "@aws.secret",
		KeyName:   "DB_PASS",
		Kind:      "s",
	}

	value := []byte("my-secret-password")

	id1 := factory.Make(ctx1, value)
	id2 := factory.Make(ctx2, value)

	assert.NotEqual(t, id1, id2, "Same value in different plans should produce different DisplayIDs")
}

// Test 3: Direct exec randomness - same script twice → different DisplayIDs
func TestDirectExecRandomness(t *testing.T) {
	// Simulate two direct execution runs with different run keys
	runKey1 := make([]byte, 32)
	runKey2 := make([]byte, 32)
	for i := range runKey1 {
		runKey1[i] = byte(i)
		runKey2[i] = byte(i + 1) // Different key
	}

	factory1 := NewIDFactory(ModeRun, runKey1)
	factory2 := NewIDFactory(ModeRun, runKey2)

	ctx := IDContext{
		PlanHash:  []byte("plan-hash-abc123"),
		StepPath:  "deploy.step[0]",
		Decorator: "@aws.secret",
		KeyName:   "DB_PASS",
		Kind:      "s",
	}

	value := []byte("my-secret-password")

	id1 := factory1.Make(ctx, value)
	id2 := factory2.Make(ctx, value)

	assert.NotEqual(t, id1, id2, "Different run keys should produce different DisplayIDs")
}

// Test 4: Context sensitivity - change step_path/key_name → ID changes
func TestContextSensitivity(t *testing.T) {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}

	factory := NewIDFactory(ModePlan, planKey)

	baseCtx := IDContext{
		PlanHash:  []byte("plan-hash-abc123"),
		StepPath:  "deploy.step[0]",
		Decorator: "@aws.secret",
		KeyName:   "DB_PASS",
		Kind:      "s",
	}

	value := []byte("my-secret-password")

	baseID := factory.Make(baseCtx, value)

	// Change StepPath
	ctxStepPath := baseCtx
	ctxStepPath.StepPath = "deploy.step[1]"
	idStepPath := factory.Make(ctxStepPath, value)
	assert.NotEqual(t, baseID, idStepPath, "Different StepPath should produce different DisplayID")

	// Change KeyName
	ctxKeyName := baseCtx
	ctxKeyName.KeyName = "API_KEY"
	idKeyName := factory.Make(ctxKeyName, value)
	assert.NotEqual(t, baseID, idKeyName, "Different KeyName should produce different DisplayID")

	// Change Decorator
	ctxDecorator := baseCtx
	ctxDecorator.Decorator = "@env"
	idDecorator := factory.Make(ctxDecorator, value)
	assert.NotEqual(t, baseID, idDecorator, "Different Decorator should produce different DisplayID")

	// Change Kind
	ctxKind := baseCtx
	ctxKind.Kind = "v"
	idKind := factory.Make(ctxKind, value)
	assert.NotEqual(t, baseID, idKind, "Different Kind should produce different DisplayID")
}

// Test 5: No length leak - different value lengths → no correlation
func TestNoLengthLeak(t *testing.T) {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}

	factory := NewIDFactory(ModePlan, planKey)

	ctx := IDContext{
		PlanHash:  []byte("plan-hash-abc123"),
		StepPath:  "deploy.step[0]",
		Decorator: "@aws.secret",
		KeyName:   "SECRET",
		Kind:      "s",
	}

	shortValue := []byte("abc")
	longValue := []byte("this-is-a-very-long-secret-value-with-many-characters")

	idShort := factory.Make(ctx, shortValue)
	idLong := factory.Make(ctx, longValue)

	// IDs should have same length (no length leak)
	assert.Equal(t, len(idShort), len(idLong), "DisplayIDs should have same length regardless of value length")

	// IDs should be different (different values)
	assert.NotEqual(t, idShort, idLong, "Different values should produce different DisplayIDs")

	// Both should have correct format
	assert.Contains(t, idShort, "opal:s:", "Short value DisplayID should have correct format")
	assert.Contains(t, idLong, "opal:s:", "Long value DisplayID should have correct format")
}

// Test 6: Format validation
func TestDisplayIDFormat(t *testing.T) {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}

	factory := NewIDFactory(ModePlan, planKey)

	tests := []struct {
		name string
		kind string
		want string
	}{
		{"secret", "s", "opal:s:"},
		{"value", "v", "opal:"},
		{"step", "st", "opal:st:"},
		{"plan", "pl", "opal:pl:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := IDContext{
				PlanHash:  []byte("plan-hash"),
				StepPath:  "test",
				Decorator: "@test",
				KeyName:   "TEST",
				Kind:      tt.kind,
			}

			id := factory.Make(ctx, []byte("test-value"))
			assert.Contains(t, id, tt.want, "DisplayID should contain correct kind prefix")
		})
	}
}
