package secret

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewHandle tests creating a new secret handle
func TestNewHandle(t *testing.T) {
	h := NewHandle("my-secret-value")

	assert.NotNil(t, h)
	assert.True(t, h.IsTainted(), "Secret should be tainted by default")
}

// TestHandleStringPanics tests that String() panics on tainted secrets
func TestHandleStringPanics(t *testing.T) {
	h := NewHandle("secret")

	assert.Panics(t, func() {
		_ = h.String()
	}, "String() should panic on tainted secret")
}

// TestHandleUnwrapWithMask tests safe unwrapping with masking
func TestHandleUnwrapWithMask(t *testing.T) {
	h := NewHandle("secret-password-123")

	masked := h.UnwrapWithMask()

	assert.NotContains(t, masked, "secret-password-123")
	assert.Contains(t, masked, "***")
}

// TestHandleUnwrapLast4 tests unwrapping last 4 characters
func TestHandleUnwrapLast4(t *testing.T) {
	h := NewHandle("secret-password-123")

	last4 := h.UnwrapLast4()

	assert.Equal(t, "...-123", last4)
	assert.NotContains(t, last4, "secret-password")
}

// TestHandleUnsafeUnwrap tests explicit unsafe unwrapping
func TestHandleUnsafeUnwrap(t *testing.T) {
	// Set capability for testing
	SetCapability(&Capability{token: 12345})
	defer SetCapability(nil)
	h := NewHandle("my-secret")

	// Unsafe unwrap requires explicit acknowledgment
	value := h.UnsafeUnwrap()

	assert.Equal(t, "my-secret", value)
}

// TestHandleDebugMode tests panic-on-leak in debug mode
func TestHandleDebugMode(t *testing.T) {
	// Enable debug mode
	oldDebug := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebug }()

	h := NewHandle("secret")

	// In debug mode, even UnsafeUnwrap should panic
	assert.Panics(t, func() {
		_ = h.UnsafeUnwrap()
	}, "UnsafeUnwrap should panic in debug mode")
}

// TestHandleIsEmpty tests empty secret detection
func TestHandleIsEmpty(t *testing.T) {
	h := NewHandle("")

	assert.True(t, h.IsEmpty())

	h2 := NewHandle("not-empty")
	assert.False(t, h2.IsEmpty())
}

// TestHandleLen tests getting secret length without exposing value
func TestHandleLen(t *testing.T) {
	h := NewHandle("12345")

	assert.Equal(t, 5, h.Len())
}

// TestHandleEqual tests comparing secrets without exposing values
func TestHandleEqual(t *testing.T) {
	h1 := NewHandle("secret")
	h2 := NewHandle("secret")
	h3 := NewHandle("different")
	h4 := NewHandle("secret2") // Same length as h1, different value

	assert.True(t, h1.Equal(h2), "Same values should be equal")
	assert.False(t, h1.Equal(h3), "Different values should not be equal")
	assert.False(t, h1.Equal(h4), "Same length, different values should not be equal")
}

// TestHandleEqualNilPanics tests that Equal panics on nil
func TestHandleEqualNilPanics(t *testing.T) {
	h := NewHandle("secret")
	assert.Panics(t, func() {
		h.Equal(nil)
	})
}

// TestHandleWithPlaceholder tests placeholder generation
func TestHandleWithPlaceholder(t *testing.T) {
	h1 := NewHandle("my-secret")
	h2 := NewHandle("my-secret")

	p1 := h1.Placeholder()
	p2 := h2.Placeholder()

	assert.NotEmpty(t, p1)
	assert.Contains(t, p1, "opal:s:")
	// Different handles for same value should have different placeholders (oracle protection)
	assert.NotEqual(t, p1, p2, "Same value should produce different placeholders")
}

// TestHandleMask tests custom masking
func TestHandleMask(t *testing.T) {
	h := NewHandle("secret-password-123")

	mask2 := h.Mask(2)
	assert.Equal(t, "se***23", mask2)

	mask3 := h.Mask(3)
	assert.Equal(t, "sec***123", mask3)

	// Short value
	short := NewHandle("abc")
	assert.Equal(t, "***", short.Mask(2))
}

// TestHandleMaskNegativePanics tests that negative mask panics
func TestHandleMaskNegativePanics(t *testing.T) {
	h := NewHandle("secret")
	assert.Panics(t, func() {
		h.Mask(-1)
	})
}

// TestHandleForEnv tests environment variable formatting
func TestHandleForEnv(t *testing.T) {
	// Set capability for testing
	SetCapability(&Capability{token: 12345})
	defer SetCapability(nil)
	// Skip in debug mode (would panic)
	if DebugMode {
		t.Skip("Skipping ForEnv test in debug mode")
	}

	h := NewHandle("secret-value")
	env := h.ForEnv("API_KEY")

	assert.Equal(t, "API_KEY=secret-value", env)
}

// TestHandleForEnvEmptyKeyPanics tests that empty key panics
func TestHandleForEnvEmptyKeyPanics(t *testing.T) {
	// Set capability for testing
	SetCapability(&Capability{token: 12345})
	defer SetCapability(nil)
	if DebugMode {
		t.Skip("Skipping in debug mode")
	}

	h := NewHandle("secret")
	assert.Panics(t, func() {
		h.ForEnv("")
	})
}

// TestHandleBytes tests byte conversion
func TestHandleBytes(t *testing.T) {
	// Set capability for testing
	SetCapability(&Capability{token: 12345})
	defer SetCapability(nil)
	// Skip in debug mode (would panic)
	if DebugMode {
		t.Skip("Skipping Bytes test in debug mode")
	}

	h := NewHandle("secret")
	b := h.Bytes()

	assert.Equal(t, []byte("secret"), b)
}

// TestHandleFormat tests fmt.Formatter implementation
func TestHandleFormat(t *testing.T) {
	h := NewHandle("my-actual-secret-value")

	// %v should return placeholder, not the actual value
	v := fmt.Sprintf("%v", h)
	assert.Contains(t, v, "opal:s:")
	assert.NotContains(t, v, "my-actual-secret-value")

	// %s should return placeholder, not the actual value
	s := fmt.Sprintf("%s", h)
	assert.Contains(t, s, "opal:s:")
	assert.NotContains(t, s, "my-actual-secret-value")

	// %#v should return GoString, not the actual value
	gv := fmt.Sprintf("%#v", h)
	assert.Contains(t, gv, "secret.Handle{")
	assert.Contains(t, gv, "opal:s:")
	assert.NotContains(t, gv, "my-actual-secret-value")
}

// TestHandleFingerprint tests keyed fingerprinting
func TestHandleFingerprint(t *testing.T) {
	h1 := NewHandle("my-secret")
	h2 := NewHandle("my-secret")

	// Same key should produce same fingerprint for same value
	key := make([]byte, 32)
	fp1 := h1.Fingerprint(key)
	fp2 := h2.Fingerprint(key)
	assert.Equal(t, fp1, fp2, "Same value with same key should produce same fingerprint")

	// Different key should produce different fingerprint
	key2 := make([]byte, 32)
	key2[0] = 1
	fp3 := h1.Fingerprint(key2)
	assert.NotEqual(t, fp1, fp3, "Same value with different key should produce different fingerprint")

	// Fingerprint should be hex-encoded (64 chars for BLAKE2b-256)
	assert.Len(t, fp1, 64)
}

// TestHandleFingerprintShortKeyPanics tests that short key panics
func TestHandleFingerprintShortKeyPanics(t *testing.T) {
	h := NewHandle("secret")
	assert.Panics(t, func() {
		h.Fingerprint([]byte("short"))
	})
}

// TestHandleID tests opaque ID generation
func TestHandleID(t *testing.T) {
	h1 := NewHandle("my-secret")
	h2 := NewHandle("my-secret")

	id1 := h1.ID()
	id2 := h2.ID()

	// IDs should be different even for same value (oracle protection)
	assert.NotEqual(t, id1, id2, "Same value should produce different IDs")

	// IDs should be opaque (not reveal value)
	assert.Contains(t, id1, "opal:s:")
	assert.NotContains(t, id1, "my-secret")
}

// TestCapabilityGate tests that raw access requires capability
func TestCapabilityGate(t *testing.T) {
	h := NewHandle("secret")

	// Without capability, should panic
	assert.Panics(t, func() {
		h.UnsafeUnwrap()
	}, "UnsafeUnwrap should panic without capability")

	assert.Panics(t, func() {
		h.Bytes()
	}, "Bytes should panic without capability")

	assert.Panics(t, func() {
		h.ForEnv("KEY")
	}, "ForEnv should panic without capability")

	// Set capability
	SetCapability(&Capability{token: 12345})
	defer SetCapability(nil) // Clean up

	// With capability, should work
	assert.NotPanics(t, func() {
		_ = h.UnsafeUnwrap()
	}, "UnsafeUnwrap should work with capability")

	assert.NotPanics(t, func() {
		_ = h.Bytes()
	}, "Bytes should work with capability")

	assert.NotPanics(t, func() {
		_ = h.ForEnv("KEY")
	}, "ForEnv should work with capability")
}

// TestHandleQuotedFormat tests %q formatting
func TestHandleQuotedFormat(t *testing.T) {
	h := NewHandle("my-secret-value")

	// %q should return quoted placeholder, not the actual value
	quoted := fmt.Sprintf("%q", h)
	assert.Contains(t, quoted, "opal:s:")
	assert.NotContains(t, quoted, "my-secret-value")
}
