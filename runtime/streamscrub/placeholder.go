package streamscrub

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"

	"golang.org/x/crypto/blake2b"
)

// PlaceholderGenerator creates deterministic placeholders using keyed BLAKE2b.
type PlaceholderGenerator struct {
	mu  sync.Mutex
	key []byte // Per-run key for BLAKE2b
}

// NewPlaceholderGenerator creates a new generator with a random per-run key.
func NewPlaceholderGenerator() (*PlaceholderGenerator, error) {
	// Generate a random 32-byte key for this run
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate placeholder key: %w", err)
	}

	return &PlaceholderGenerator{key: key}, nil
}

// NewPlaceholderGeneratorWithKey creates a generator with a specific key.
// Use this when you need deterministic placeholders across runs (e.g., for testing).
func NewPlaceholderGeneratorWithKey(key []byte) (*PlaceholderGenerator, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be exactly 32 bytes, got %d", len(key))
	}

	keyCopy := make([]byte, 32)
	copy(keyCopy, key)

	return &PlaceholderGenerator{key: keyCopy}, nil
}

// Generate creates a deterministic placeholder for a secret.
// Same secret + same key → same placeholder.
// Different key → different placeholder (prevents correlation across runs).
func (g *PlaceholderGenerator) Generate(secret []byte) string {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Use BLAKE2b-256 with the per-run key
	hash, err := blake2b.New256(g.key)
	if err != nil {
		// Should never happen with valid key
		panic(fmt.Sprintf("blake2b.New256 failed: %v", err))
	}

	hash.Write(secret)
	digest := hash.Sum(nil)

	// Encode as base64 (URL-safe, no padding) for compact representation
	encoded := base64.RawURLEncoding.EncodeToString(digest[:8]) // Use first 8 bytes (64 bits)

	// Format: <REDACTED:hash>
	// Fixed length, no information about secret length
	return fmt.Sprintf("<REDACTED:%s>", encoded)
}

// PlaceholderFunc returns a function compatible with WithPlaceholderFunc.
func (g *PlaceholderGenerator) PlaceholderFunc() PlaceholderFunc {
	return func(secret []byte) string {
		return g.Generate(secret)
	}
}
