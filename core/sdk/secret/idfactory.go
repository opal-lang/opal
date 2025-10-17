package secret

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
)

// DisplayIDMode determines how DisplayIDs are generated
type DisplayIDMode int

const (
	// ModePlan generates deterministic IDs for resolved plans (contract verification)
	ModePlan DisplayIDMode = iota
	// ModeRun generates random-looking IDs for direct execution (security)
	ModeRun
)

// IDContext provides context for DisplayID generation
type IDContext struct {
	PlanHash  []byte // Canonical plan digest
	StepPath  string // e.g., "deploy.step[0]"
	Decorator string // e.g., "@aws.secret"
	KeyName   string // e.g., "DB_PASS", "HOME"
	Kind      string // "s" (secret), "v" (value), "st" (step), "pl" (plan)
}

// IDFactory generates DisplayIDs using a keyed PRF
type IDFactory interface {
	Make(ctx IDContext, value []byte) string
}

// keyedIDFactory implements IDFactory using BLAKE2s-128 keyed hash
type keyedIDFactory struct {
	mode DisplayIDMode
	key  []byte // 32-byte key for BLAKE2s
}

// NewIDFactory creates a new IDFactory with the given mode and key
//
// For ModePlan: key should be derived from PSE seed (deterministic)
// For ModeRun: key should be a fresh random nonce (random-looking)
func NewIDFactory(mode DisplayIDMode, key []byte) IDFactory {
	if len(key) != 32 {
		panic(fmt.Sprintf("IDFactory key must be 32 bytes, got %d", len(key)))
	}

	keyCopy := make([]byte, 32)
	copy(keyCopy, key)

	return &keyedIDFactory{
		mode: mode,
		key:  keyCopy,
	}
}

// Make generates a DisplayID using keyed BLAKE2s-128 PRF
//
// PRF: BLAKE2s-128(key, planhash || context || BLAKE2b-256(value))
// - Key: plan_key (ModePlan) or run_key (ModeRun)
// - Input: planhash || step_path || decorator || key_name || kind || hash(value)
// - Output: base58(digest) → "opal:s:3J98t56A" (22 chars typical)
//
// Using hash(value) prevents length leaks while keeping the ID deterministic
// for the same value within the same context.
func (f *keyedIDFactory) Make(ctx IDContext, value []byte) string {
	// Build PRF input: planhash || context || hash(value)
	var input bytes.Buffer

	// PlanHash
	input.Write(ctx.PlanHash)

	// Context fields (order matters for determinism)
	input.WriteString(ctx.StepPath)
	input.WriteString("\x00") // Null separator
	input.WriteString(ctx.Decorator)
	input.WriteString("\x00")
	input.WriteString(ctx.KeyName)
	input.WriteString("\x00")
	input.WriteString(ctx.Kind)
	input.WriteString("\x00")

	// Hash of value (prevents length leak)
	valueHash := blake2b.Sum256(value)
	input.Write(valueHash[:])

	// Compute keyed BLAKE2s-128 digest
	digest, err := blake2s.New128(f.key)
	if err != nil {
		panic(fmt.Sprintf("failed to create BLAKE2s hasher: %v", err))
	}
	digest.Write(input.Bytes())
	hash := digest.Sum(nil)

	// Encode first 8 bytes to base58 (64 bits is sufficient for collision resistance)
	encoded := EncodeBase58(hash[:8])

	// Format: opal:kind:encoded
	return fmt.Sprintf("opal:%s:%s", ctx.Kind, encoded)
}
