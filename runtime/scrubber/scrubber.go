package scrubber

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/aledsdavies/opal/core/invariant"
	"golang.org/x/crypto/blake2b"
)

// Scrubber redacts secrets from output streams with robust obfuscation detection
type Scrubber struct {
	writer io.Writer

	// Per-run key for keyed fingerprints (prevents cross-run correlation)
	runKey []byte

	// Secret entries sorted by descending length (longest first)
	secrets []secretEntry
	rmu     sync.RWMutex // Protects secrets slice

	// Write serialization (prevents interleaved output)
	wmu sync.Mutex

	// Rolling window for chunk-boundary secrets
	carry  []byte
	maxLen int
}

// secretEntry holds a secret and its placeholder in byte form
type secretEntry struct {
	value       []byte
	placeholder []byte
}

// New creates a new Scrubber that writes to w
// Generates a per-run key for keyed fingerprints (prevents cross-run correlation)
func New(w io.Writer) *Scrubber {
	invariant.NotNil(w, "writer")

	// Generate per-run key (32 bytes for BLAKE2b-256)
	runKey := make([]byte, 32)
	if _, err := rand.Read(runKey); err != nil {
		panic(fmt.Sprintf("failed to generate scrubber run key: %v", err))
	}

	return &Scrubber{
		writer: w,
		runKey: runKey,
		maxLen: 0, // Will be set to longest secret when secrets are registered
		carry:  make([]byte, 0, 1024),
	}
}

// RunKey returns a copy of the per-run key for keyed fingerprints
// This is used by secret.Handle.Fingerprint() for consistent hashing within a run
// Returns a copy to prevent external mutation of the internal key
func (s *Scrubber) RunKey() []byte {
	k := make([]byte, len(s.runKey))
	copy(k, s.runKey)
	return k
}

// Fingerprint computes a keyed fingerprint of a value using the per-run key
// This is used internally for detection, NOT for display
func (s *Scrubber) Fingerprint(value []byte) string {
	hash, err := blake2b.New256(s.runKey)
	if err != nil {
		panic(fmt.Sprintf("failed to create BLAKE2b hash: %v", err))
	}
	hash.Write(value)
	digest := hash.Sum(nil)
	return hex.EncodeToString(digest)
}

// RegisterSecret registers a secret value to be scrubbed
// The placeholder will ALWAYS be used regardless of encoding
func RegisterSecret(s *Scrubber, value, placeholder string) {
	invariant.Precondition(value != "", "secret value cannot be empty")
	invariant.Precondition(placeholder != "", "placeholder cannot be empty")

	s.rmu.Lock()
	defer s.rmu.Unlock()

	// Register the raw secret
	s.addEntry([]byte(value), []byte(placeholder))

	// Register common encodings (all use same placeholder for obfuscation)
	// Hex: lowercase and uppercase
	s.addEntry([]byte(hex.EncodeToString([]byte(value))), []byte(placeholder))
	s.addEntry([]byte(strings.ToUpper(hex.EncodeToString([]byte(value)))), []byte(placeholder))

	// Base64: standard, URL, and raw (no padding) variants
	s.addEntry([]byte(base64.StdEncoding.EncodeToString([]byte(value))), []byte(placeholder))
	s.addEntry([]byte(base64.URLEncoding.EncodeToString([]byte(value))), []byte(placeholder))
	s.addEntry([]byte(base64.RawStdEncoding.EncodeToString([]byte(value))), []byte(placeholder))
	s.addEntry([]byte(base64.RawURLEncoding.EncodeToString([]byte(value))), []byte(placeholder))

	// URL encoding
	s.addEntry([]byte(url.QueryEscape(value)), []byte(placeholder))
	s.addEntry([]byte(url.PathEscape(value)), []byte(placeholder))

	// Register reversed (common obfuscation)
	s.addEntry(reverse([]byte(value)), []byte(placeholder))

	// Register separator-tolerant variants (p-a-s-s-w-o-r-d)
	s.addEntry(withSeparators([]byte(value), '-'), []byte(placeholder))
	s.addEntry(withSeparators([]byte(value), '_'), []byte(placeholder))
	s.addEntry(withSeparators([]byte(value), '.'), []byte(placeholder))
	s.addEntry(withSeparators([]byte(value), ':'), []byte(placeholder))

	// Sort by descending length (longest first to handle substrings)
	sort.Slice(s.secrets, func(i, j int) bool {
		return len(s.secrets[i].value) > len(s.secrets[j].value)
	})
}

// addEntry adds a secret entry (internal, assumes lock held)
// Updates maxLen to track the longest registered variant
func (s *Scrubber) addEntry(value, placeholder []byte) {
	if len(value) == 0 {
		return // Skip empty values
	}
	s.secrets = append(s.secrets, secretEntry{
		value:       value,
		placeholder: placeholder,
	})

	// Update maxLen to longest variant (not just raw value)
	if len(value) > s.maxLen {
		s.maxLen = len(value)
	}
}

// Write implements io.Writer with secret scrubbing
func (s *Scrubber) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Serialize writes to prevent interleaving
	s.wmu.Lock()
	defer s.wmu.Unlock()

	// Merge with carry from previous write
	buf := append(append([]byte{}, s.carry...), p...)

	// Redact all secrets (byte-wise, longest first)
	s.rmu.RLock()
	redacted := s.redactAll(buf)
	s.rmu.RUnlock()

	// Keep last maxLen-1 bytes as carry for next write
	// (in case secret is split across chunk boundary)
	// If no secrets registered (maxLen==0), don't buffer anything
	carrySize := 0
	if s.maxLen > 0 {
		carrySize = s.maxLen - 1
	}

	if carrySize > 0 && len(redacted) > carrySize {
		// Write everything except the carry
		toWrite := redacted[:len(redacted)-carrySize]
		s.carry = append(s.carry[:0], redacted[len(redacted)-carrySize:]...)

		n, err := s.writer.Write(toWrite)
		if err != nil {
			return n, err
		}
		if n < len(toWrite) {
			return n, io.ErrShortWrite
		}
	} else if carrySize > 0 {
		// Buffer is smaller than carry size, accumulate
		s.carry = append(s.carry[:0], redacted...)
	} else {
		// No secrets registered, write everything immediately
		n, err := s.writer.Write(redacted)
		if err != nil {
			return n, err
		}
		if n < len(redacted) {
			return n, io.ErrShortWrite
		}
	}

	// Report original chunk size as written
	return len(p), nil
}

// Close implements io.WriteCloser by flushing remaining data
// This ensures trailing secrets at chunk boundaries are caught
// Callers MUST defer Close() to prevent secret leakage
func (s *Scrubber) Close() error {
	return s.Flush()
}

// Flush writes any remaining carry bytes after redaction
// Must be called at end of stream to ensure trailing secrets are caught
func (s *Scrubber) Flush() error {
	s.wmu.Lock()
	defer s.wmu.Unlock()

	if len(s.carry) == 0 {
		return nil
	}

	// Redact carry one final time
	s.rmu.RLock()
	redacted := s.redactAll(s.carry)
	s.rmu.RUnlock()

	// Write and clear carry
	_, err := s.writer.Write(redacted)
	s.carry = s.carry[:0]

	return err
}

// redactAll replaces all secrets in buf (byte-wise, longest first)
// Assumes rmu is held for reading
func (s *Scrubber) redactAll(buf []byte) []byte {
	result := buf

	// Process secrets longest-first to handle substrings correctly
	for _, entry := range s.secrets {
		result = bytes.ReplaceAll(result, entry.value, entry.placeholder)
	}

	return result
}

// reverse reverses a byte slice
func reverse(b []byte) []byte {
	result := make([]byte, len(b))
	for i := range b {
		result[i] = b[len(b)-1-i]
	}
	return result
}

// withSeparators inserts separator between each byte
// e.g., "pass" with '-' becomes "p-a-s-s"
func withSeparators(b []byte, sep byte) []byte {
	if len(b) == 0 {
		return b
	}

	result := make([]byte, 0, len(b)*2-1)
	for i, c := range b {
		if i > 0 {
			result = append(result, sep)
		}
		result = append(result, c)
	}
	return result
}
