package scrubber

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicScrubbing tests simple secret replacement
func TestBasicScrubbing(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "my-secret-password"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	input := "The password is: my-secret-password\n"
	n, err := s.Write([]byte(input))

	require.NoError(t, err)
	assert.Equal(t, len(input), n)

	// Flush to write remaining carry
	_ = s.Flush()

	assert.Equal(t, "The password is: <secret:1>\n", buf.String())
	assert.NotContains(t, buf.String(), secret)
}

// TestChunkBoundarySplit tests secrets split across Write() calls
func TestChunkBoundarySplit(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "secret-value-123"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Split secret across three writes
	_, _ = s.Write([]byte("prefix secret-"))
	_, _ = s.Write([]byte("value-"))
	_, _ = s.Write([]byte("123 suffix"))

	// Flush remaining carry
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, secret, "Secret should be scrubbed even when split")
	assert.Contains(t, output, placeholder, "Placeholder should appear")
}

// TestMultipleSecrets tests multiple secrets with different lengths
func TestMultipleSecrets(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register secrets in random order
	RegisterSecret(s, "short", "<s1>")
	RegisterSecret(s, "this-is-a-longer-secret", "<s2>")
	RegisterSecret(s, "medium-secret", "<s3>")

	input := "short this-is-a-longer-secret medium-secret"
	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, "short")
	assert.NotContains(t, output, "this-is-a-longer-secret")
	assert.NotContains(t, output, "medium-secret")
	assert.Contains(t, output, "<s1>")
	assert.Contains(t, output, "<s2>")
	assert.Contains(t, output, "<s3>")
}

// TestSubstringSecrets tests when one secret is substring of another
func TestSubstringSecrets(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Longer secret should be matched first
	RegisterSecret(s, "password", "<short>")
	RegisterSecret(s, "password123", "<long>")

	input := "password password123"
	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	// Longer secret should match first, leaving shorter one
	assert.Contains(t, output, "<long>")
	assert.Contains(t, output, "<short>")
	assert.NotContains(t, output, "password123")
}

// TestBase64EncodedSecret tests base64-encoded secrets are caught
func TestBase64EncodedSecret(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "my-secret-value"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Secret appears base64-encoded
	encoded := base64.StdEncoding.EncodeToString([]byte(secret))
	input := "Encoded: " + encoded + "\n"

	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, encoded, "Base64-encoded secret should be scrubbed")
	assert.Contains(t, output, placeholder)
}

// TestHexEncodedSecret tests hex-encoded secrets are caught
func TestHexEncodedSecret(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "secret"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Secret appears hex-encoded
	encoded := hex.EncodeToString([]byte(secret))
	input := "Hex: " + encoded + "\n"

	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, encoded, "Hex-encoded secret should be scrubbed")
	assert.Contains(t, output, placeholder)
}

// TestURLEncodedSecret tests URL-encoded secrets are caught
func TestURLEncodedSecret(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "my secret!"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Secret appears URL-encoded
	encoded := url.QueryEscape(secret)
	input := "URL: " + encoded + "\n"

	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, encoded, "URL-encoded secret should be scrubbed")
	assert.Contains(t, output, placeholder)
}

// TestReversedSecret tests reversed secrets are caught
func TestReversedSecret(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "password123"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Secret appears reversed
	reversed := reverseString(secret)
	input := "Reversed: " + reversed + "\n"

	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, reversed, "Reversed secret should be scrubbed")
	assert.Contains(t, output, placeholder)
}

// TestSeparatorObfuscation tests secrets with separators inserted
func TestSeparatorObfuscation(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "password"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Secret with separators: p-a-s-s-w-o-r-d
	obfuscated := "p-a-s-s-w-o-r-d"
	input := "Obfuscated: " + obfuscated + "\n"

	_, _ = s.Write([]byte(input))
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, obfuscated, "Separator-obfuscated secret should be scrubbed")
	assert.Contains(t, output, placeholder)
}

// TestConcurrentWrites tests thread-safety with concurrent writes
func TestConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "concurrent-secret"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := strings.Repeat("concurrent-secret ", 100)
			_, _ = s.Write([]byte(msg))
		}(i)
	}

	wg.Wait()
	_ = s.Flush()

	output := buf.String()
	assert.NotContains(t, output, secret, "Concurrent writes should not leak secrets")
}

// TestInvariantNilWriter tests panic on nil writer
func TestInvariantNilWriter(t *testing.T) {
	assert.Panics(t, func() {
		New(nil)
	}, "Should panic on nil writer")
}

// TestInvariantEmptySecret tests panic on empty secret
func TestInvariantEmptySecret(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	assert.Panics(t, func() {
		RegisterSecret(s, "", "<placeholder>")
	}, "Should panic on empty secret")
}

// TestInvariantEmptyPlaceholder tests panic on empty placeholder
func TestInvariantEmptyPlaceholder(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	assert.Panics(t, func() {
		RegisterSecret(s, "secret", "")
	}, "Should panic on empty placeholder")
}

// TestFlushCarry tests that Flush() processes remaining carry bytes
func TestFlushCarry(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "secret-at-end"
	placeholder := "<secret:1>"
	RegisterSecret(s, secret, placeholder)

	// Write secret but don't complete it
	_, _ = s.Write([]byte("prefix secret-at-"))

	// Before flush, secret is incomplete
	assert.NotContains(t, buf.String(), placeholder)

	// Complete the secret
	_, _ = s.Write([]byte("end"))
	_ = s.Flush()

	// After flush, secret should be scrubbed
	output := buf.String()
	assert.NotContains(t, output, secret)
	assert.Contains(t, output, placeholder)
}

// TestEmptyWrite tests handling of empty writes
func TestEmptyWrite(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	n, err := s.Write([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

// TestBinaryData tests handling of non-UTF8 binary data
func TestBinaryData(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := []byte{0xFF, 0xFE, 0xFD, 0xFC}
	placeholder := "<binary>"
	RegisterSecret(s, string(secret), placeholder)

	input := []byte{0x01, 0x02, 0xFF, 0xFE, 0xFD, 0xFC, 0x03, 0x04}
	n, err := s.Write(input)

	require.NoError(t, err)
	assert.Equal(t, len(input), n)

	_ = s.Flush()
	output := buf.Bytes()
	assert.NotContains(t, output, secret)
}

// Helper function to reverse a string
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// TestRunKey tests per-run key generation
func TestRunKey(t *testing.T) {
	s1 := New(&bytes.Buffer{})
	s2 := New(&bytes.Buffer{})

	key1 := s1.RunKey()
	key2 := s2.RunKey()

	// Keys should be 32 bytes
	assert.Len(t, key1, 32)
	assert.Len(t, key2, 32)

	// Different scrubbers should have different keys
	assert.NotEqual(t, key1, key2, "Different scrubbers should have different run keys")
}

// TestFingerprint tests keyed fingerprinting
func TestFingerprint(t *testing.T) {
	s := New(&bytes.Buffer{})

	fp1 := s.Fingerprint([]byte("my-secret"))
	fp2 := s.Fingerprint([]byte("my-secret"))
	fp3 := s.Fingerprint([]byte("different"))

	// Same value should produce same fingerprint
	assert.Equal(t, fp1, fp2)

	// Different value should produce different fingerprint
	assert.NotEqual(t, fp1, fp3)

	// Fingerprint should be hex-encoded (64 chars for BLAKE2b-256)
	assert.Len(t, fp1, 64)
}

// TestFingerprintDifferentKeys tests that different scrubbers produce different fingerprints
func TestFingerprintDifferentKeys(t *testing.T) {
	s1 := New(&bytes.Buffer{})
	s2 := New(&bytes.Buffer{})

	fp1 := s1.Fingerprint([]byte("my-secret"))
	fp2 := s2.Fingerprint([]byte("my-secret"))

	// Same value with different keys should produce different fingerprints
	assert.NotEqual(t, fp1, fp2, "Different run keys should produce different fingerprints")
}

// TestRunKeyCopy verifies that RunKey() returns a copy, not the internal slice
func TestRunKeyCopy(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Get key and mutate it
	key1 := s.RunKey()
	original := make([]byte, len(key1))
	copy(original, key1)

	// Mutate the returned key
	for i := range key1 {
		key1[i] = 0xFF
	}

	// Get key again - should be unchanged
	key2 := s.RunKey()
	assert.Equal(t, original, key2, "RunKey should return a copy, not mutable reference")
	assert.NotEqual(t, key1, key2, "Mutating returned key should not affect internal key")
}

// TestNoSecretsNoBuffering verifies that with no secrets, output is not buffered
func TestNoSecretsNoBuffering(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Write without registering any secrets
	data := []byte("This should pass through immediately")
	n, err := s.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	// Should be written immediately (no carry buffering)
	assert.Equal(t, string(data), buf.String(), "With no secrets, output should not be buffered")
}

// TestCloseFlushes verifies that Close() calls Flush()
func TestCloseFlushes(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	RegisterSecret(s, "secret", "REDACTED")

	// Write data that will be held in carry
	_, _ = s.Write([]byte("prefix"))

	// Before Close, buffer should be empty (data in carry)
	assert.Equal(t, "", buf.String())

	// Close should flush carry
	err := s.Close()
	require.NoError(t, err)

	// Now buffer should have the data
	assert.Equal(t, "prefix", buf.String())
}

// TestHexUpperCase verifies that uppercase hex encoding is caught
func TestHexUpperCase(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "test"
	RegisterSecret(s, secret, "REDACTED")

	// Hex uppercase: 74657374
	_, _ = s.Write([]byte("prefix 74657374 suffix"))
	_ = s.Flush()

	output := buf.String()
	assert.Contains(t, output, "REDACTED", "Uppercase hex should be caught")
	assert.NotContains(t, output, "74657374", "Uppercase hex should be replaced")
}

// TestBase64RawEncoding verifies that base64 raw (no padding) is caught
func TestBase64RawEncoding(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := "test"
	RegisterSecret(s, secret, "REDACTED")

	// Base64 raw (no padding): dGVzdA (vs dGVzdA== with padding)
	_, _ = s.Write([]byte("prefix dGVzdA suffix"))
	_ = s.Flush()

	output := buf.String()
	assert.Contains(t, output, "REDACTED", "Base64 raw encoding should be caught")
	assert.NotContains(t, output, "dGVzdA", "Base64 raw should be replaced")
}
