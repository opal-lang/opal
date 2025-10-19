package streamscrub

import (
	"bytes"
	"strings"
	"testing"
)

// TestPlaceholderGeneratorDeterminism - same secret + same key = same placeholder
func TestPlaceholderGeneratorDeterminism(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	gen, err := NewPlaceholderGeneratorWithKey(key)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	secret := []byte("MY_SECRET_TOKEN")

	// Generate placeholder multiple times
	ph1 := gen.Generate(secret)
	ph2 := gen.Generate(secret)
	ph3 := gen.Generate(secret)

	// All should be identical
	if ph1 != ph2 || ph2 != ph3 {
		t.Errorf("non-deterministic placeholders: %q, %q, %q", ph1, ph2, ph3)
	}

	// Should have expected format
	if !strings.HasPrefix(ph1, "<REDACTED:") || !strings.HasSuffix(ph1, ">") {
		t.Errorf("unexpected format: %q", ph1)
	}
}

// TestPlaceholderGeneratorDifferentKeys - different keys = different placeholders
func TestPlaceholderGeneratorDifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 1) // Different from key1
	}

	gen1, _ := NewPlaceholderGeneratorWithKey(key1)
	gen2, _ := NewPlaceholderGeneratorWithKey(key2)

	secret := []byte("SAME_SECRET")

	ph1 := gen1.Generate(secret)
	ph2 := gen2.Generate(secret)

	// Different keys should produce different placeholders
	if ph1 == ph2 {
		t.Errorf("same placeholder for different keys: %q", ph1)
	}
}

// TestPlaceholderGeneratorDifferentSecrets - different secrets = different placeholders
func TestPlaceholderGeneratorDifferentSecrets(t *testing.T) {
	gen, _ := NewPlaceholderGenerator()

	secret1 := []byte("SECRET_ONE")
	secret2 := []byte("SECRET_TWO")

	ph1 := gen.Generate(secret1)
	ph2 := gen.Generate(secret2)

	// Different secrets should produce different placeholders
	if ph1 == ph2 {
		t.Errorf("same placeholder for different secrets: %q", ph1)
	}
}

// TestPlaceholderGeneratorFixedLength - all placeholders have same length
func TestPlaceholderGeneratorFixedLength(t *testing.T) {
	gen, _ := NewPlaceholderGenerator()

	secrets := [][]byte{
		[]byte("a"),
		[]byte("short"),
		[]byte("medium_length_secret"),
		[]byte("very_long_secret_that_is_much_longer_than_the_others"),
	}

	var lengths []int
	for _, secret := range secrets {
		ph := gen.Generate(secret)
		lengths = append(lengths, len(ph))
	}

	// All placeholders should have the same length (no length leakage)
	firstLen := lengths[0]
	for i, l := range lengths {
		if l != firstLen {
			t.Errorf("placeholder %d has different length: %d vs %d", i, l, firstLen)
		}
	}
}

// TestPlaceholderGeneratorIntegration - use with scrubber
func TestPlaceholderGeneratorIntegration(t *testing.T) {
	key := make([]byte, 32)
	gen, _ := NewPlaceholderGeneratorWithKey(key)

	var buf bytes.Buffer
	s := New(&buf, WithPlaceholderFunc(gen.PlaceholderFunc()))

	secret := []byte("API_KEY_12345")
	s.RegisterSecret(secret, []byte(gen.Generate(secret)))

	input := []byte("The key is: API_KEY_12345\n")
	s.Write(input)
	s.Flush()

	got := buf.String()

	// Secret should be redacted
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("secret leaked: %q", got)
	}

	// Should contain keyed placeholder
	if !strings.Contains(got, "<REDACTED:") {
		t.Errorf("keyed placeholder not found: %q", got)
	}
}

// TestPlaceholderGeneratorInvalidKey - reject invalid key sizes
func TestPlaceholderGeneratorInvalidKey(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
	}{
		{"too-short", 16},
		{"too-long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewPlaceholderGeneratorWithKey(key)
			if err == nil {
				t.Errorf("expected error for key length %d", tt.keyLen)
			}
		})
	}
}
