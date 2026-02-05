// Package streamscrub provides streaming secret redaction.
package streamscrub

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"sync"

	"github.com/opal-lang/opal/core/invariant"
)

// PlaceholderFunc generates a placeholder for a secret.
// The function should return a deterministic placeholder for the same secret+key.
type PlaceholderFunc func(secret []byte) string

// Scrubber redacts secrets from output streams.
type Scrubber struct {
	mu              sync.Mutex // Protects all fields below
	out             io.Writer
	provider        SecretProvider // Provider for secret detection and replacement
	frames          []frame
	carry           []byte // Rolling window for chunk-boundary secrets
	placeholderFunc PlaceholderFunc
}

// frame represents a buffering scope.
type frame struct {
	label string
	buf   bytes.Buffer
}

// Option configures a Scrubber.
type Option func(*Scrubber)

// WithPlaceholderFunc sets a custom placeholder generation function.
// If not provided, uses a default simple placeholder.
func WithPlaceholderFunc(fn PlaceholderFunc) Option {
	return func(s *Scrubber) {
		s.placeholderFunc = fn
	}
}

// WithSecretProvider sets a secret provider for on-demand detection.
// The provider is queried for each chunk of data to determine if it contains
// secrets. This enables automatic secret detection without manual registration.
//
// Example:
//
//	vault := vault.NewWithPlanKey(planKey)
//	scrubber := streamscrub.New(output, streamscrub.WithSecretProvider(vault))
//	// Secrets automatically detected via vault
func WithSecretProvider(provider SecretProvider) Option {
	return func(s *Scrubber) {
		s.provider = provider
	}
}

// New creates a new Scrubber that writes to w.
// By default, uses keyed BLAKE2b placeholders with a random per-run key.
// This prevents correlation attacks across runs.
func New(w io.Writer, opts ...Option) *Scrubber {
	// INPUT CONTRACT
	invariant.NotNil(w, "writer")

	// Create default keyed placeholder generator (security by default)
	gen, err := NewPlaceholderGenerator()
	if err != nil {
		// Fallback to simple placeholder if random key generation fails
		// (should never happen in practice)
		gen = nil
	}

	var defaultFunc PlaceholderFunc
	if gen != nil {
		defaultFunc = gen.PlaceholderFunc()
	} else {
		defaultFunc = simplePlaceholder
	}

	s := &Scrubber{
		out:             w,
		placeholderFunc: defaultFunc,
	}

	// Apply options (can override default placeholder)
	for _, opt := range opts {
		opt(s)
	}

	// OUTPUT CONTRACT
	invariant.Postcondition(s.out != nil, "scrubber must have output writer")
	invariant.Postcondition(len(s.frames) == 0, "scrubber must start with no active frames")
	invariant.Postcondition(s.placeholderFunc != nil, "scrubber must have placeholder function")

	return s
}

// simplePlaceholder is a fallback placeholder generator (used only if keyed generation fails).
func simplePlaceholder(secret []byte) string {
	return "<REDACTED>"
}

// StartFrame begins a new buffering scope.
func (s *Scrubber) StartFrame(label string) {
	// INPUT CONTRACT
	invariant.Precondition(label != "", "frame label cannot be empty")

	s.mu.Lock()
	defer s.mu.Unlock()

	oldFrameCount := len(s.frames)

	s.frames = append(s.frames, frame{
		label: label,
		buf:   bytes.Buffer{},
	})

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.frames) == oldFrameCount+1, "frame must be pushed onto stack")
	invariant.Postcondition(s.frames[len(s.frames)-1].label == label, "frame label must match")
}

// EndFrame ends the current frame and flushes scrubbed output.
// Secrets are provided by the SecretProvider, not passed as parameters.
func (s *Scrubber) EndFrame() error {
	// INPUT CONTRACT
	invariant.Precondition(len(s.frames) > 0, "cannot end frame when no frame is active")

	s.mu.Lock()
	defer s.mu.Unlock()

	oldFrameCount := len(s.frames)

	// Pop current frame
	currentFrame := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]

	// Get frame buffer bytes
	frameBuf := currentFrame.buf.Bytes()

	// Scrub frame buffer with provider (if available)
	scrubbed, err := s.scrubAll(frameBuf)
	if err != nil {
		// Provider rejected chunk - do not write unsanitized data
		// Zeroize buffer and return error
		for i := range frameBuf {
			frameBuf[i] = 0
		}
		currentFrame.buf.Reset()
		return err
	}

	// Flush to output BEFORE zeroizing (scrubbed may share underlying array with frameBuf)
	_, err = s.out.Write(scrubbed)

	// Zeroize frame buffer after writing
	for i := range frameBuf {
		frameBuf[i] = 0
	}
	currentFrame.buf.Reset()

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.frames) == oldFrameCount-1, "frame must be popped from stack")

	return err
}

// scrubAll replaces all secrets in buf using the SecretProvider.
// Returns an error if the provider rejects the chunk (e.g., fail-fast mode).
// Assumes mu is held.
func (s *Scrubber) scrubAll(buf []byte) ([]byte, error) {
	// Use provider-based scrubbing if available
	if s.provider != nil {
		return s.scrubAllProvider(buf)
	}

	// No provider - pass through unchanged
	return buf, nil
}

// scrubAllProvider uses SecretProvider to process chunk.
// Provider handles all secret detection and replacement internally.
// Returns an error if the provider rejects the chunk.
// Assumes mu is held.
func (s *Scrubber) scrubAllProvider(buf []byte) ([]byte, error) {
	// Provider does all the work - finds and replaces all secrets
	processed, err := s.provider.HandleChunk(buf)
	if err != nil {
		// Provider rejected chunk (e.g., fail-fast mode detected secret)
		// Return error to prevent writing unsanitized data
		return nil, err
	}

	return processed, nil
}

// Helper functions for encoding variants

func toHex(b []byte) string {
	return hex.EncodeToString(b)
}

func toUpperHex(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		if s[i] >= 'a' && s[i] <= 'f' {
			result[i] = s[i] - 32 // 'a' -> 'A'
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func toBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func toBase64Raw(b []byte) string {
	return base64.RawStdEncoding.EncodeToString(b)
}

func toBase64URL(b []byte) string {
	return base64.URLEncoding.EncodeToString(b)
}

func toPercentEncoding(b []byte, uppercase bool) string {
	result := make([]byte, 0, len(b)*3)
	for _, c := range b {
		result = append(result, '%')
		if uppercase {
			result = append(result, "0123456789ABCDEF"[c>>4])
			result = append(result, "0123456789ABCDEF"[c&0xF])
		} else {
			result = append(result, "0123456789abcdef"[c>>4])
			result = append(result, "0123456789abcdef"[c&0xF])
		}
	}
	return string(result)
}

func insertSeparators(b []byte, sep string) string {
	if len(b) == 0 {
		return ""
	}
	result := make([]byte, 0, len(b)*2)
	result = append(result, b[0])
	for i := 1; i < len(b); i++ {
		result = append(result, []byte(sep)...)
		result = append(result, b[i])
	}
	return string(result)
}

// LockdownStreams redirects stdout and stderr through the scrubber.
// Returns a restore function that MUST be deferred to restore original streams.
//
// Usage:
//
//	scrubber := streamscrub.New(os.Stdout)
//	restore := scrubber.LockdownStreams()
//	defer restore()
//	// All stdout/stderr now goes through scrubber
func (s *Scrubber) LockdownStreams() func() {
	// INPUT CONTRACT
	invariant.Precondition(s.out != nil, "scrubber must have output writer")

	// Save original streams
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create pipes for stdout and stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		panic("streamscrub: failed to create stdout pipe: " + err.Error())
	}

	rErr, wErr, err := os.Pipe()
	if err != nil {
		panic("streamscrub: failed to create stderr pipe: " + err.Error())
	}

	// Redirect os.Stdout and os.Stderr to write ends of pipes
	os.Stdout = wOut
	os.Stderr = wErr

	// Copy from pipes to scrubber in background
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(s, rOut)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(s, rErr)
	}()

	// Return idempotent restore function
	var once sync.Once
	return func() {
		once.Do(func() {
			// Close write ends to signal EOF to copy goroutines
			_ = wOut.Close()
			_ = wErr.Close()

			// Wait for copy goroutines to finish
			wg.Wait()

			// Close read ends
			_ = rOut.Close()
			_ = rErr.Close()

			// Restore original streams
			os.Stdout = originalStdout
			os.Stderr = originalStderr

			// Flush any remaining buffered data
			_ = s.Flush()
		})
	}
}

// Write implements io.Writer - scrubs secrets before writing.
func (s *Scrubber) Write(p []byte) (int, error) {
	// INPUT CONTRACT
	invariant.Precondition(s.out != nil, "output writer must not be nil")

	if len(p) == 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// If we're in a frame, buffer the output
	if len(s.frames) > 0 {
		currentFrame := &s.frames[len(s.frames)-1]
		n, err := currentFrame.buf.Write(p)

		// OUTPUT CONTRACT (frame mode)
		invariant.Postcondition(n == len(p) || err != nil, "must write all bytes or return error")
		return n, err
	}

	// Streaming mode: merge with carry from previous write
	buf := append(append([]byte{}, s.carry...), p...)

	// Scrub all secrets (longest-first)
	result, err := s.scrubAll(buf)
	if err != nil {
		// Provider rejected chunk - do not write unsanitized data
		return 0, err
	}

	// Keep last maxLen-1 bytes as carry for next write
	// (in case secret is split across chunk boundary)
	carrySize := 0
	maxLen := 0

	// Get max secret length from provider
	if s.provider != nil {
		maxLen = s.provider.MaxSecretLength()
	}

	if maxLen > 0 {
		carrySize = maxLen - 1
		// UTF-8 safety: hold back at least 3 bytes for multi-byte code points
		if carrySize < 3 {
			carrySize = 3
		}
	}

	// INVARIANT: carrySize must be reasonable
	invariant.Postcondition(carrySize >= 0, "carrySize must be non-negative")
	invariant.Postcondition(carrySize < 1024*1024, "carrySize must be reasonable (<1MB)")

	if carrySize > 0 && len(result) > carrySize {
		// Write everything except the carry
		toWrite := result[:len(result)-carrySize]
		s.carry = append(s.carry[:0], result[len(result)-carrySize:]...)

		// INVARIANT: carry size matches expected
		invariant.Postcondition(len(s.carry) == carrySize, "carry must be exactly carrySize bytes")

		_, err := s.out.Write(toWrite)
		if err != nil {
			return 0, err
		}
	} else if carrySize > 0 {
		// Buffer is smaller than carry size, accumulate
		s.carry = append(s.carry[:0], result...)

		// INVARIANT: carry doesn't exceed expected size
		invariant.Postcondition(len(s.carry) <= carrySize, "carry must not exceed carrySize")
	} else {
		// No secrets registered, write everything immediately
		_, err := s.out.Write(result)
		if err != nil {
			return 0, err
		}
	}

	// OUTPUT CONTRACT (streaming mode)
	// Return original length (io.Writer contract)
	return len(p), nil
}

// Flush writes any remaining carry bytes after redaction.
func (s *Scrubber) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.carry) == 0 {
		return nil
	}

	// Scrub carry one final time (longest-first)
	result, err := s.scrubAll(s.carry)
	if err != nil {
		// Provider rejected chunk - zeroize carry and return error
		for i := range s.carry {
			s.carry[i] = 0
		}
		s.carry = s.carry[:0]
		return err
	}

	// Write and zeroize carry
	_, err = s.out.Write(result)

	// Zeroize carry buffer
	for i := range s.carry {
		s.carry[i] = 0
	}
	s.carry = s.carry[:0]

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.carry) == 0, "carry must be cleared after flush")

	return err
}

// Close flushes remaining data and zeroizes sensitive buffers.
// Callers MUST defer Close() to prevent secret leakage.
//
// WARNING: Any open frames are discarded (not flushed). This prevents
// secret leakage but may lose buffered output. Ensure all frames are
// ended before calling Close().
func (s *Scrubber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flush any remaining carry data (Flush locks, so unlock first)
	s.mu.Unlock()
	err := s.Flush()
	s.mu.Lock()

	// Zeroize any open frame buffers (discarded, not flushed)
	// LOOP INVARIANT: Track progress through frames
	prevIdx := -1
	for idx := range s.frames {
		// Assert loop makes progress
		invariant.Postcondition(idx > prevIdx, "loop must make progress")
		prevIdx = idx

		buf := s.frames[idx].buf.Bytes()
		for j := range buf {
			buf[j] = 0
		}
	}
	s.frames = s.frames[:0]

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.carry) == 0, "carry must be cleared")
	invariant.Postcondition(len(s.frames) == 0, "frames must be cleared")

	return err
}
