package secret

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"

	"golang.org/x/crypto/blake2b"

	"github.com/opal-lang/opal/core/invariant"
)

const (
	// redactionMask is used to hide secret values in logs and output
	redactionMask = "***"
)

// DebugMode enables panic-on-leak for testing
// Set OPAL_SECRET_DEBUG=1 to enable
var DebugMode = os.Getenv("OPAL_SECRET_DEBUG") == "1"

// Capability is a token required to unwrap secrets in production
// Only the executor can issue capabilities
type Capability struct {
	token uint64 // Opaque token (checked internally)
}

// globalCapability is set by the executor at runtime
var globalCapability *Capability

// SetCapability sets the global capability (executor only)
// This enables UnsafeUnwrap/Bytes/ForEnv in production
func SetCapability(capability *Capability) {
	globalCapability = capability
}

// Handle wraps a secret value with taint tracking
// Prevents accidental leakage by making unsafe operations explicit
type Handle struct {
	value     string
	tainted   bool
	displayID string    // Opaque display ID from IDFactory
	factory   IDFactory // Factory for generating display IDs
	context   IDContext // Context for ID generation
}

// NewHandle creates a new tainted secret handle with a default run-mode factory
// For production use, prefer NewHandleWithFactory to control determinism
func NewHandle(value string) *Handle {
	// Generate random run key for ModeRun (non-deterministic)
	var runKey [32]byte
	if _, err := rand.Read(runKey[:]); err != nil {
		panic(fmt.Sprintf("failed to generate run key: %v", err))
	}

	factory := NewIDFactory(ModeRun, runKey[:])

	// Default context (minimal - should be overridden with NewHandleWithFactory)
	ctx := IDContext{
		PlanHash:  []byte("default-run"),
		StepPath:  "unknown",
		Decorator: "unknown",
		KeyName:   "unknown",
		Kind:      "s",
	}

	return &Handle{
		value:     value,
		tainted:   true,
		displayID: factory.Make(ctx, []byte(value)),
		factory:   factory,
		context:   ctx,
	}
}

// NewHandleWithFactory creates a new tainted secret handle with explicit factory and context
// Use this for deterministic DisplayIDs in resolved plans (ModePlan)
func NewHandleWithFactory(value string, factory IDFactory, context IDContext) *Handle {
	return &Handle{
		value:     value,
		tainted:   true,
		displayID: factory.Make(context, []byte(value)),
		factory:   factory,
		context:   context,
	}
}

// IsTainted returns true if the secret is still tainted
func (h *Handle) IsTainted() bool {
	return h.tainted
}

// String implements fmt.Stringer but PANICS on tainted secrets
// This prevents accidental printing of secrets
func (h *Handle) String() string {
	if h.tainted {
		panic("attempted to print tainted secret - use UnwrapWithMask() or UnsafeUnwrap()")
	}
	return h.value
}

// UnwrapWithMask returns a masked version of the secret
// Safe to print: "sec***123" for "secret-password-123"
func (h *Handle) UnwrapWithMask() string {
	if len(h.value) <= 6 {
		return redactionMask
	}
	// Show first 3 and last 3 characters
	return h.value[:3] + redactionMask + h.value[len(h.value)-3:]
}

// UnwrapLast4 returns only the last 4 characters
// Safe to print: "...-123" for "secret-password-123"
func (h *Handle) UnwrapLast4() string {
	if len(h.value) <= 4 {
		return redactionMask
	}
	return "..." + h.value[len(h.value)-4:]
}

// Mask returns a masked version with custom visible character count
// n specifies how many characters to show at start and end
// Safe to print: Mask(2) -> "se***23" for "secret-password-123"
func (h *Handle) Mask(n int) string {
	invariant.Precondition(n >= 0, "mask count must be non-negative")
	if len(h.value) <= n*2 {
		return redactionMask
	}
	return h.value[:n] + redactionMask + h.value[len(h.value)-n:]
}

// ForEnv returns a safe environment variable assignment string
// Format: "KEY=<value>" - safe to pass to subprocess environment
// Requires capability in production (issued by executor)
// Panics in debug mode or without capability
func (h *Handle) ForEnv(key string) string {
	invariant.Precondition(key != "", "environment variable key cannot be empty")
	if DebugMode {
		panic("ForEnv() called in debug mode - only use within executor context")
	}
	if globalCapability == nil {
		panic("ForEnv() requires capability - only call from executor-issued decorators")
	}
	return key + "=" + h.value
}

// Bytes returns the secret as bytes
// Requires capability in production (issued by executor)
// Panics in debug mode or without capability
func (h *Handle) Bytes() []byte {
	if DebugMode {
		panic("Bytes() called in debug mode - only use within executor context")
	}
	if globalCapability == nil {
		panic("Bytes() requires capability - only call from executor-issued decorators")
	}
	return []byte(h.value)
}

// UnsafeUnwrap returns the raw secret value
// ONLY use when absolutely necessary (e.g., passing to subprocess)
// Requires capability in production (issued by executor)
// Panics in debug mode or without capability
// Consider using ForEnv() or Bytes() instead for safer alternatives
func (h *Handle) UnsafeUnwrap() string {
	if DebugMode {
		panic("UnsafeUnwrap() called in debug mode - secret leak detected")
	}
	if globalCapability == nil {
		panic("UnsafeUnwrap() requires capability - only call from executor-issued decorators")
	}
	return h.value
}

// IsEmpty returns true if the secret is empty
func (h *Handle) IsEmpty() bool {
	return h.value == ""
}

// Len returns the length of the secret without exposing the value
func (h *Handle) Len() int {
	return len(h.value)
}

// Equal compares two secrets without exposing values
// Uses constant-time comparison to prevent timing attacks
func (h *Handle) Equal(other *Handle) bool {
	invariant.NotNil(other, "other")
	if h.Len() != other.Len() {
		return false
	}
	// Use standard library constant-time comparison
	return subtle.ConstantTimeCompare([]byte(h.value), []byte(other.value)) == 1
}

// ID returns the opaque identifier for display (user-visible)
// Format: opal:s:3J98t56A (Base58 encoded, context-aware)
// In ModePlan: deterministic (same context+value → same ID)
// In ModeRun: random-looking (different run → different ID)
func (h *Handle) ID() string {
	return h.displayID
}

// IDWithEmoji returns the opaque identifier with emoji for display
// Format: 🔒 opal:secret:3J98t56A
// Used in terminal output and logs
func (h *Handle) IDWithEmoji() string {
	return fmt.Sprintf("🔒 %s", h.ID())
}

// Placeholder returns the scrubber placeholder (alias for ID for backward compatibility)
func (h *Handle) Placeholder() string {
	return h.ID()
}

// Fingerprint returns a keyed hash for scrubber matching (internal use only)
// Uses BLAKE2b with a per-run key to prevent correlation across runs
// This is what the scrubber uses internally for detection, NOT what users see
func (h *Handle) Fingerprint(key []byte) string {
	invariant.Precondition(len(key) >= 32, "fingerprint key must be at least 32 bytes")

	// BLAKE2b-256 with per-run key
	hash, err := blake2b.New256(key)
	if err != nil {
		panic(fmt.Sprintf("failed to create BLAKE2b hash: %v", err))
	}
	hash.Write([]byte(h.value))
	digest := hash.Sum(nil)

	return hex.EncodeToString(digest)
}

// GoString implements fmt.GoStringer for %#v formatting
// Returns placeholder instead of raw value
func (h *Handle) GoString() string {
	return fmt.Sprintf("secret.Handle{%s}", h.Placeholder())
}

// Format implements fmt.Formatter to prevent %v bypass
// All format verbs return the placeholder, never the raw value
func (h *Handle) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('#') {
			// %#v -> GoString()
			_, _ = fmt.Fprint(f, h.GoString())
		} else {
			// %v -> Placeholder()
			_, _ = fmt.Fprint(f, h.Placeholder())
		}
	case 's':
		// %s -> Placeholder()
		_, _ = fmt.Fprint(f, h.Placeholder())
	default:
		// Unknown verb -> Placeholder()
		_, _ = fmt.Fprint(f, h.Placeholder())
	}
}

// MarshalJSON implements json.Marshaler
// Returns placeholder instead of raw value
func (h *Handle) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, h.Placeholder())), nil
}

// MarshalText implements encoding.TextMarshaler
// Returns placeholder instead of raw value
func (h *Handle) MarshalText() ([]byte, error) {
	return []byte(h.Placeholder()), nil
}
