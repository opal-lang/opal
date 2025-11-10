package streamscrub

import (
	"bytes"
	"strings"
	"testing"
)

// TestOpalPlaceholderFormat verifies the opal:s:hash format
func TestOpalPlaceholderFormat(t *testing.T) {
	key := make([]byte, 32)
	gen, err := NewOpalPlaceholderGeneratorWithKey(key)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	secret := []byte("MY_SECRET_TOKEN")
	placeholder := gen.Generate(secret)

	// Should have format: opal:s:hash
	if !strings.HasPrefix(placeholder, "opal:s:") {
		t.Errorf("placeholder should start with 'opal:s:', got: %q", placeholder)
	}

	// Should be fixed length
	parts := strings.Split(placeholder, ":")
	if len(parts) != 3 {
		t.Errorf("placeholder should have 3 parts (opal:s:hash), got %d: %q", len(parts), placeholder)
	}

	if parts[0] != "opal" {
		t.Errorf("first part should be 'opal', got: %q", parts[0])
	}
	if parts[1] != "s" {
		t.Errorf("second part should be 's', got: %q", parts[1])
	}
	if len(parts[2]) == 0 {
		t.Errorf("hash part should not be empty")
	}
}

// TestOpalPlaceholderDeterminism verifies same secret → same placeholder
func TestOpalPlaceholderDeterminism(t *testing.T) {
	key := make([]byte, 32)
	gen, _ := NewOpalPlaceholderGeneratorWithKey(key)

	secret := []byte("TEST_SECRET")
	ph1 := gen.Generate(secret)
	ph2 := gen.Generate(secret)
	ph3 := gen.Generate(secret)

	if ph1 != ph2 || ph2 != ph3 {
		t.Errorf("non-deterministic placeholders: %q, %q, %q", ph1, ph2, ph3)
	}
}

// TestOpalPlaceholderDifferentKeys verifies different keys → different placeholders
func TestOpalPlaceholderDifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 1)
	}

	gen1, _ := NewOpalPlaceholderGeneratorWithKey(key1)
	gen2, _ := NewOpalPlaceholderGeneratorWithKey(key2)

	secret := []byte("SAME_SECRET")
	ph1 := gen1.Generate(secret)
	ph2 := gen2.Generate(secret)

	if ph1 == ph2 {
		t.Errorf("same placeholder for different keys: %q", ph1)
	}

	// Both should still have opal:s: prefix
	if !strings.HasPrefix(ph1, "opal:s:") || !strings.HasPrefix(ph2, "opal:s:") {
		t.Errorf("placeholders should have opal:s: prefix: %q, %q", ph1, ph2)
	}
}

// TestOpalPlaceholderIntegration tests with scrubber
func TestOpalPlaceholderIntegration(t *testing.T) {
	var buf bytes.Buffer
	gen, _ := NewOpalPlaceholderGenerator()

	secret := []byte("API_KEY_12345")
	placeholder := gen.Generate(secret)

	provider := NewPatternProvider(func() []Pattern {
		return []Pattern{
			{Value: secret, Placeholder: []byte(placeholder)},
		}
	})

	s := New(&buf, WithPlaceholderFunc(gen.PlaceholderFunc()), WithSecretProvider(provider))

	input := []byte("The key is: API_KEY_12345\n")
	s.Write(input)
	s.Flush()

	got := buf.String()

	// Secret should be redacted
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("secret leaked: %q", got)
	}

	// Should contain opal:s: placeholder
	if !strings.Contains(got, "opal:s:") {
		t.Errorf("opal:s: placeholder not found: %q", got)
	}
}
