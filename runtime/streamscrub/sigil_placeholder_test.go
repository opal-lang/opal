package streamscrub

import (
	"bytes"
	"strings"
	"testing"
)

// TestSigilPlaceholderFormat verifies the sigil:s:hash format
func TestSigilPlaceholderFormat(t *testing.T) {
	key := make([]byte, 32)
	gen, err := NewSigilPlaceholderGeneratorWithKey(key)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	secret := []byte("MY_SECRET_TOKEN")
	placeholder := gen.Generate(secret)

	// Should have format: sigil:s:hash
	if !strings.HasPrefix(placeholder, "sigil:s:") {
		t.Errorf("placeholder should start with 'sigil:s:', got: %q", placeholder)
	}

	// Should be fixed length
	parts := strings.Split(placeholder, ":")
	if len(parts) != 3 {
		t.Errorf("placeholder should have 3 parts (sigil:s:hash), got %d: %q", len(parts), placeholder)
	}

	if parts[0] != "sigil" {
		t.Errorf("first part should be 'sigil', got: %q", parts[0])
	}
	if parts[1] != "s" {
		t.Errorf("second part should be 's', got: %q", parts[1])
	}
	if len(parts[2]) == 0 {
		t.Errorf("hash part should not be empty")
	}
}

// TestSigilPlaceholderDeterminism verifies same secret → same placeholder
func TestSigilPlaceholderDeterminism(t *testing.T) {
	key := make([]byte, 32)
	gen, _ := NewSigilPlaceholderGeneratorWithKey(key)

	secret := []byte("TEST_SECRET")
	ph1 := gen.Generate(secret)
	ph2 := gen.Generate(secret)
	ph3 := gen.Generate(secret)

	if ph1 != ph2 || ph2 != ph3 {
		t.Errorf("non-deterministic placeholders: %q, %q, %q", ph1, ph2, ph3)
	}
}

// TestSigilPlaceholderDifferentKeys verifies different keys → different placeholders
func TestSigilPlaceholderDifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 1)
	}

	gen1, _ := NewSigilPlaceholderGeneratorWithKey(key1)
	gen2, _ := NewSigilPlaceholderGeneratorWithKey(key2)

	secret := []byte("SAME_SECRET")
	ph1 := gen1.Generate(secret)
	ph2 := gen2.Generate(secret)

	if ph1 == ph2 {
		t.Errorf("same placeholder for different keys: %q", ph1)
	}

	// Both should still have sigil:s: prefix
	if !strings.HasPrefix(ph1, "sigil:s:") || !strings.HasPrefix(ph2, "sigil:s:") {
		t.Errorf("placeholders should have sigil:s: prefix: %q, %q", ph1, ph2)
	}
}

// TestSigilPlaceholderIntegration tests with scrubber
func TestSigilPlaceholderIntegration(t *testing.T) {
	var buf bytes.Buffer
	gen, _ := NewSigilPlaceholderGenerator()

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

	// Should contain sigil:s: placeholder
	if !strings.Contains(got, "sigil:s:") {
		t.Errorf("sigil:s: placeholder not found: %q", got)
	}
}
