package streamscrub

import (
	"bytes"
	"sort"
	"testing"
)

// mockProvider is a simple SecretProvider for testing
type mockProvider struct {
	secrets map[string]string // secret → placeholder
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		secrets: make(map[string]string),
	}
}

func (m *mockProvider) AddSecret(secret, placeholder string) {
	m.secrets[secret] = placeholder
}

func (m *mockProvider) HandleChunk(chunk []byte) ([]byte, error) {
	result := chunk

	// Build sorted list (longest first)
	type entry struct {
		secret      []byte
		placeholder []byte
	}
	var entries []entry
	for secret, placeholder := range m.secrets {
		entries = append(entries, entry{
			secret:      []byte(secret),
			placeholder: []byte(placeholder),
		})
	}

	// Sort by descending length
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].secret) > len(entries[j].secret)
	})

	// Replace all secrets (longest first)
	for _, e := range entries {
		result = bytes.ReplaceAll(result, e.secret, e.placeholder)
	}

	return result, nil
}

func (m *mockProvider) MaxSecretLength() int {
	maxLen := 0
	for secret := range m.secrets {
		if len(secret) > maxLen {
			maxLen = len(secret)
		}
	}
	return maxLen
}

// TestSecretProvider_NilProvider tests scrubber with no provider (pass-through)
func TestSecretProvider_NilProvider(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf, WithSecretProvider(nil))

	input := []byte("secret data here")
	s.Write(input)
	s.Flush()

	// Should pass through unchanged (no provider)
	if got := buf.String(); got != "secret data here" {
		t.Errorf("got %q, want %q", got, "secret data here")
	}
}

// TestSecretProvider_MockProvider tests scrubber with mock provider
func TestSecretProvider_MockProvider(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("secret123", "opal:abc")

	s := New(&buf, WithSecretProvider(provider))

	input := []byte("The value is: secret123")
	s.Write(input)
	s.Flush()

	want := "The value is: opal:abc"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestSecretProvider_MultipleSecrets tests multiple secrets in output
func TestSecretProvider_MultipleSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("secret1", "opal:aaa")
	provider.AddSecret("secret2", "opal:bbb")

	s := New(&buf, WithSecretProvider(provider))

	input := []byte("First: secret1, Second: secret2")
	s.Write(input)
	s.Flush()

	got := buf.String()
	// Both secrets should be replaced
	if bytes.Contains([]byte(got), []byte("secret1")) {
		t.Errorf("secret1 not scrubbed: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("secret2")) {
		t.Errorf("secret2 not scrubbed: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:aaa")) {
		t.Errorf("placeholder opal:aaa not found: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:bbb")) {
		t.Errorf("placeholder opal:bbb not found: %q", got)
	}
}

// TestSecretProvider_LongestMatch tests that longest secret wins
func TestSecretProvider_LongestMatch(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("SECRET", "opal:short")
	provider.AddSecret("SECRET_EXTENDED", "opal:long")

	s := New(&buf, WithSecretProvider(provider))

	input := []byte("Value: SECRET_EXTENDED")
	s.Write(input)
	s.Flush()

	got := buf.String()
	// Should use longest match
	if !bytes.Contains([]byte(got), []byte("opal:long")) {
		t.Errorf("longest match not used: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("opal:short")) {
		t.Errorf("short match incorrectly used: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("SECRET")) {
		t.Errorf("secret not scrubbed: %q", got)
	}
}

// TestSecretProvider_NoSecrets tests provider that finds no secrets
func TestSecretProvider_NoSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("secret123", "opal:abc")

	s := New(&buf, WithSecretProvider(provider))

	input := []byte("No secrets here")
	s.Write(input)
	s.Flush()

	// Should pass through unchanged (no secrets found)
	if got := buf.String(); got != "No secrets here" {
		t.Errorf("got %q, want %q", got, "No secrets here")
	}
}

// TestSecretProvider_ChunkBoundary tests secret split across writes
func TestSecretProvider_ChunkBoundary(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("SECRET_TOKEN", "opal:xyz")

	s := New(&buf, WithSecretProvider(provider))

	// Split secret across two writes
	s.Write([]byte("Value: SECRET_"))
	s.Write([]byte("TOKEN here"))
	s.Flush()

	got := buf.String()
	// Secret should be scrubbed even though split
	if bytes.Contains([]byte(got), []byte("SECRET_TOKEN")) {
		t.Errorf("secret leaked across boundary: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:xyz")) {
		t.Errorf("placeholder not found: %q", got)
	}
}

// ========== NewPatternProvider SDK Helper Tests ==========

// TestNewPatternProvider_EmptyPatterns tests provider with no patterns
func TestNewPatternProvider_EmptyPatterns(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{}
	}

	provider := NewPatternProvider(source)

	chunk := []byte("some data here")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Should pass through unchanged
	if !bytes.Equal(result, chunk) {
		t.Errorf("got %q, want %q", result, chunk)
	}
}

// TestNewPatternProvider_SinglePattern tests provider with one pattern
func TestNewPatternProvider_SinglePattern(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("secret123"), Placeholder: []byte("opal:abc")},
		}
	}

	provider := NewPatternProvider(source)

	chunk := []byte("The value is: secret123")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	want := "The value is: opal:abc"
	if string(result) != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

// TestNewPatternProvider_MultiplePatterns tests provider with multiple patterns
func TestNewPatternProvider_MultiplePatterns(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("secret1"), Placeholder: []byte("opal:aaa")},
			{Value: []byte("secret2"), Placeholder: []byte("opal:bbb")},
		}
	}

	provider := NewPatternProvider(source)

	chunk := []byte("First: secret1, Second: secret2")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	got := string(result)
	// Both secrets should be replaced
	if bytes.Contains([]byte(got), []byte("secret1")) {
		t.Errorf("secret1 not scrubbed: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("secret2")) {
		t.Errorf("secret2 not scrubbed: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:aaa")) {
		t.Errorf("placeholder opal:aaa not found: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:bbb")) {
		t.Errorf("placeholder opal:bbb not found: %q", got)
	}
}

// TestNewPatternProvider_LongestFirst tests longest-first matching
func TestNewPatternProvider_LongestFirst(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("SECRET"), Placeholder: []byte("opal:short")},
			{Value: []byte("SECRET_EXTENDED"), Placeholder: []byte("opal:long")},
		}
	}

	provider := NewPatternProvider(source)

	chunk := []byte("Value: SECRET_EXTENDED")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	got := string(result)
	// Should use longest match
	want := "Value: opal:long"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Verify no partial replacement
	if bytes.Contains([]byte(got), []byte("SECRET")) {
		t.Errorf("secret not fully scrubbed: %q", got)
	}
}

// TestNewPatternProvider_DynamicPatterns tests that patterns can change between calls
func TestNewPatternProvider_DynamicPatterns(t *testing.T) {
	patterns := []Pattern{
		{Value: []byte("secret1"), Placeholder: []byte("opal:aaa")},
	}

	source := func() []Pattern {
		return patterns
	}

	provider := NewPatternProvider(source)

	// First call with secret1
	chunk1 := []byte("Value: secret1")
	result1, err := provider.HandleChunk(chunk1)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	want1 := "Value: opal:aaa"
	if string(result1) != want1 {
		t.Errorf("got %q, want %q", result1, want1)
	}

	// Update patterns
	patterns = []Pattern{
		{Value: []byte("secret1"), Placeholder: []byte("opal:aaa")},
		{Value: []byte("secret2"), Placeholder: []byte("opal:bbb")},
	}

	// Second call with secret2 (should now be replaced)
	chunk2 := []byte("Value: secret2")
	result2, err := provider.HandleChunk(chunk2)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	want2 := "Value: opal:bbb"
	if string(result2) != want2 {
		t.Errorf("got %q, want %q", result2, want2)
	}
}

// TestNewPatternProvider_EmptyValue tests pattern with empty value (should be skipped)
func TestNewPatternProvider_EmptyValue(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte(""), Placeholder: []byte("opal:empty")},
			{Value: []byte("secret"), Placeholder: []byte("opal:abc")},
		}
	}

	provider := NewPatternProvider(source)

	chunk := []byte("Value: secret")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	want := "Value: opal:abc"
	if string(result) != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

// TestNewPatternProvider_NoMatch tests chunk with no matching patterns
func TestNewPatternProvider_NoMatch(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("secret123"), Placeholder: []byte("opal:abc")},
		}
	}

	provider := NewPatternProvider(source)

	chunk := []byte("No secrets here")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Should pass through unchanged
	if !bytes.Equal(result, chunk) {
		t.Errorf("got %q, want %q", result, chunk)
	}
}

// TestNewPatternProvider_MaxSecretLength tests MaxSecretLength method
func TestNewPatternProvider_MaxSecretLength(t *testing.T) {
	tests := []struct {
		name     string
		patterns []Pattern
		wantMax  int
	}{
		{
			name:     "empty patterns",
			patterns: []Pattern{},
			wantMax:  0,
		},
		{
			name: "single pattern",
			patterns: []Pattern{
				{Value: []byte("secret"), Placeholder: []byte("opal:abc")},
			},
			wantMax: 6, // len("secret")
		},
		{
			name: "multiple patterns",
			patterns: []Pattern{
				{Value: []byte("short"), Placeholder: []byte("opal:aaa")},
				{Value: []byte("SECRET_EXTENDED_TOKEN"), Placeholder: []byte("opal:bbb")},
				{Value: []byte("medium"), Placeholder: []byte("opal:ccc")},
			},
			wantMax: 21, // len("SECRET_EXTENDED_TOKEN")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := func() []Pattern {
				return tt.patterns
			}

			provider := NewPatternProvider(source)
			got := provider.MaxSecretLength()

			if got != tt.wantMax {
				t.Errorf("MaxSecretLength() = %d, want %d", got, tt.wantMax)
			}
		})
	}
}

// ========== Integration Tests: NewPatternProvider + Scrubber ==========

// TestIntegration_PatternProviderWithScrubber tests complete flow
func TestIntegration_PatternProviderWithScrubber(t *testing.T) {
	// Create pattern source
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("ghp_abc123xyz"), Placeholder: []byte("opal:token1")},
			{Value: []byte("sk_test_456"), Placeholder: []byte("opal:token2")},
		}
	}

	// Create provider using SDK helper
	provider := NewPatternProvider(source)

	// Create scrubber with provider
	var buf bytes.Buffer
	s := New(&buf, WithSecretProvider(provider))

	// Write data with secrets
	input := []byte("API Key: ghp_abc123xyz, Stripe: sk_test_456")
	s.Write(input)
	s.Flush()

	got := buf.String()

	// Verify secrets replaced
	want := "API Key: opal:token1, Stripe: opal:token2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Verify no secrets leaked
	if bytes.Contains([]byte(got), []byte("ghp_abc123xyz")) {
		t.Errorf("secret ghp_abc123xyz leaked: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("sk_test_456")) {
		t.Errorf("secret sk_test_456 leaked: %q", got)
	}
}

// TestIntegration_DynamicPatternsWithScrubber tests patterns changing between writes
func TestIntegration_DynamicPatternsWithScrubber(t *testing.T) {
	// Mutable pattern list
	patterns := []Pattern{
		{Value: []byte("secret1"), Placeholder: []byte("opal:aaa")},
	}

	source := func() []Pattern {
		return patterns
	}

	provider := NewPatternProvider(source)

	var buf bytes.Buffer
	s := New(&buf, WithSecretProvider(provider))

	// First write with secret1
	s.Write([]byte("Value: secret1\n"))
	s.Flush()

	// Add secret2 to patterns
	patterns = append(patterns, Pattern{
		Value:       []byte("secret2"),
		Placeholder: []byte("opal:bbb"),
	})

	// Second write with secret2 (should now be replaced)
	s.Write([]byte("Value: secret2\n"))
	s.Flush()

	got := buf.String()
	want := "Value: opal:aaa\nValue: opal:bbb\n"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestNewPatternProviderWithVariants tests encoding variant generation
func TestNewPatternProviderWithVariants(t *testing.T) {
	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("test"), Placeholder: []byte("REDACTED")},
		}
	}

	provider := NewPatternProviderWithVariants(source)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "raw secret",
			input: "Value: test",
			want:  "Value: REDACTED",
		},
		{
			name:  "hex lowercase",
			input: "Value: 74657374",
			want:  "Value: REDACTED",
		},
		{
			name:  "hex uppercase",
			input: "Value: 74657374",
			want:  "Value: REDACTED",
		},
		{
			name:  "base64 standard",
			input: "Value: dGVzdA==",
			want:  "Value: REDACTED",
		},
		{
			name:  "base64 raw",
			input: "Value: dGVzdA",
			want:  "Value: REDACTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.HandleChunk([]byte(tt.input))
			if err != nil {
				t.Fatalf("HandleChunk failed: %v", err)
			}

			got := string(result)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
