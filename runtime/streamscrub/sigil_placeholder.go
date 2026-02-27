package streamscrub

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/blake2b"
)

// SigilPlaceholderGenerator creates placeholders in Sigil format: sigil:s:hash
type SigilPlaceholderGenerator struct {
	key []byte
}

// NewSigilPlaceholderGenerator creates a generator with random per-run key.
// Placeholders use format: sigil:s:hash
func NewSigilPlaceholderGenerator() (*SigilPlaceholderGenerator, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate placeholder key: %w", err)
	}
	return &SigilPlaceholderGenerator{key: key}, nil
}

// NewSigilPlaceholderGeneratorWithKey creates a generator with a specific key.
func NewSigilPlaceholderGeneratorWithKey(key []byte) (*SigilPlaceholderGenerator, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be exactly 32 bytes, got %d", len(key))
	}
	keyCopy := make([]byte, 32)
	copy(keyCopy, key)
	return &SigilPlaceholderGenerator{key: keyCopy}, nil
}

// Generate creates a placeholder in Sigil format: sigil:s:hash
func (g *SigilPlaceholderGenerator) Generate(secret []byte) string {
	hash, err := blake2b.New256(g.key)
	if err != nil {
		panic(fmt.Sprintf("blake2b.New256 failed: %v", err))
	}

	hash.Write(secret)
	digest := hash.Sum(nil)

	// Use first 8 bytes (64 bits) for compact representation
	encoded := base64.RawURLEncoding.EncodeToString(digest[:8])

	// Format: sigil:s:hash (Sigil secret with keyed hash)
	return fmt.Sprintf("sigil:s:%s", encoded)
}

// PlaceholderFunc returns a function compatible with WithPlaceholderFunc.
func (g *SigilPlaceholderGenerator) PlaceholderFunc() PlaceholderFunc {
	return func(secret []byte) string {
		return g.Generate(secret)
	}
}
