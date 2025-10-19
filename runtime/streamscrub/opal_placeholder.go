package streamscrub

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/blake2b"
)

// OpalPlaceholderGenerator creates placeholders in Opal format: opal:s:hash
type OpalPlaceholderGenerator struct {
	key []byte
}

// NewOpalPlaceholderGenerator creates a generator with random per-run key.
// Placeholders use format: opal:s:hash
func NewOpalPlaceholderGenerator() (*OpalPlaceholderGenerator, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate placeholder key: %w", err)
	}
	return &OpalPlaceholderGenerator{key: key}, nil
}

// NewOpalPlaceholderGeneratorWithKey creates a generator with a specific key.
func NewOpalPlaceholderGeneratorWithKey(key []byte) (*OpalPlaceholderGenerator, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be exactly 32 bytes, got %d", len(key))
	}
	keyCopy := make([]byte, 32)
	copy(keyCopy, key)
	return &OpalPlaceholderGenerator{key: keyCopy}, nil
}

// Generate creates a placeholder in Opal format: opal:s:hash
func (g *OpalPlaceholderGenerator) Generate(secret []byte) string {
	hash, err := blake2b.New256(g.key)
	if err != nil {
		panic(fmt.Sprintf("blake2b.New256 failed: %v", err))
	}

	hash.Write(secret)
	digest := hash.Sum(nil)

	// Use first 8 bytes (64 bits) for compact representation
	encoded := base64.RawURLEncoding.EncodeToString(digest[:8])

	// Format: opal:s:hash (Opal secret with keyed hash)
	return fmt.Sprintf("opal:s:%s", encoded)
}

// PlaceholderFunc returns a function compatible with WithPlaceholderFunc.
func (g *OpalPlaceholderGenerator) PlaceholderFunc() PlaceholderFunc {
	return func(secret []byte) string {
		return g.Generate(secret)
	}
}
