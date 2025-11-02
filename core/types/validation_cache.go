package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// validatorCache caches compiled JSON Schema validators
type validatorCache struct {
	mu      sync.RWMutex
	cache   map[string]*jsonschema.Schema
	maxSize int
}

// newValidatorCache creates a new validator cache
func newValidatorCache(maxSize int) *validatorCache {
	return &validatorCache{
		cache:   make(map[string]*jsonschema.Schema),
		maxSize: maxSize,
	}
}

// get retrieves cached validator by schema hash
func (c *validatorCache) get(schemaHash string) (*jsonschema.Schema, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.cache[schemaHash]
	return v, ok
}

// put stores validator in cache
func (c *validatorCache) put(schemaHash string, validator *jsonschema.Schema) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: if cache full, clear it
	// (LRU would be better but adds complexity)
	if len(c.cache) >= c.maxSize {
		c.cache = make(map[string]*jsonschema.Schema)
	}

	c.cache[schemaHash] = validator
}

// hashSchema computes SHA-256 hash of JSON Schema
func hashSchema(schema JSONSchema) (string, error) {
	b, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}
