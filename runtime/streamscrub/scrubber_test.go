package streamscrub

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
)

// ============================================================================
// Test Helpers
// ============================================================================

// safeBuffer is a thread-safe bytes.Buffer for concurrent testing
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// testProvider creates a simple pattern provider for testing
func testProvider(patterns map[string]string) SecretProvider {
	return NewPatternProvider(func() []Pattern {
		var result []Pattern
		for secret, placeholder := range patterns {
			result = append(result, Pattern{
				Value:       []byte(secret),
				Placeholder: []byte(placeholder),
			})
		}
		return result
	})
}

// ============================================================================
// Basic Functionality Tests
// ============================================================================

// TestBasicScrubbing verifies scrubber passes through data when no provider
func TestBasicScrubbing(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	input := []byte("hello world")
	n, err := s.Write(input)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("Write returned %d, want %d", n, len(input))
	}

	if err := s.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if got := buf.String(); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

// TestSimpleSecretRedaction verifies basic secret scrubbing
func TestSimpleSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"my-secret-key": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("The key is: my-secret-key"))
	s.Flush()

	want := "The key is: <REDACTED>"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestMultipleSecrets verifies multiple secrets are replaced
func TestMultipleSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"secret1": "<REDACTED-1>",
		"secret2": "<REDACTED-2>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("First: secret1, Second: secret2"))
	s.Flush()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("secret1")) {
		t.Errorf("secret1 not scrubbed: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("secret2")) {
		t.Errorf("secret2 not scrubbed: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("<REDACTED-1>")) {
		t.Errorf("placeholder 1 not found: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("<REDACTED-2>")) {
		t.Errorf("placeholder 2 not found: %q", got)
	}
}

// ============================================================================
// Chunk Boundary Tests
// ============================================================================

// TestChunkBoundarySafety verifies secrets split across writes are detected
func TestChunkBoundarySafety(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"secret-value-123": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	// Split secret across 3 chunks
	s.Write([]byte("prefix secret-"))
	s.Write([]byte("value-"))
	s.Write([]byte("123 suffix"))
	s.Flush()

	want := "prefix <REDACTED> suffix"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestSplitBoundaryFourWrites verifies secrets split across many writes
func TestSplitBoundaryFourWrites(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"LONG_SECRET_TOKEN": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("prefix LONG_"))
	s.Write([]byte("SECRET_"))
	s.Write([]byte("TOKEN"))
	s.Write([]byte(" suffix"))
	s.Flush()

	want := "prefix <REDACTED> suffix"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ============================================================================
// Frame Tests
// ============================================================================

// TestFrameBuffering verifies frames buffer output
func TestFrameBuffering(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	s.StartFrame("test-frame")
	s.Write([]byte("buffered output"))

	// Nothing written yet
	if buf.Len() != 0 {
		t.Errorf("output written during frame, want buffered")
	}

	s.EndFrame()

	// Now flushed
	want := "buffered output"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFrameScrubbingHierarchical verifies frames scrub with provider
func TestFrameScrubbingHierarchical(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"secret123": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.StartFrame("decorator-frame")
	s.Write([]byte("Loading secret: secret123"))
	s.EndFrame()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("secret123")) {
		t.Errorf("secret leaked: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("<REDACTED>")) {
		t.Errorf("placeholder not found: %q", got)
	}
}

// TestNestedFrames verifies nested frame behavior
func TestNestedFrames(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"outer-secret": "<OUTER>",
		"inner-secret": "<INNER>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.StartFrame("outer")
	s.Write([]byte("outer: outer-secret "))

	s.StartFrame("inner")
	s.Write([]byte("inner: inner-secret"))
	s.EndFrame()

	s.EndFrame()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("outer-secret")) {
		t.Errorf("outer secret leaked: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("inner-secret")) {
		t.Errorf("inner secret leaked: %q", got)
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

// TestConcurrentWrites verifies thread safety
func TestConcurrentWrites(t *testing.T) {
	var buf safeBuffer
	provider := testProvider(map[string]string{
		"secret": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := fmt.Sprintf("goroutine %d: secret\n", id)
			s.Write([]byte(msg))
		}(i)
	}

	wg.Wait()
	s.Flush()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("secret")) {
		t.Errorf("secret leaked in concurrent writes: %q", got)
	}
}

// ============================================================================
// Lockdown Tests
// ============================================================================

// TestLockdownStreams verifies stdout/stderr redirection
func TestLockdownStreams(t *testing.T) {
	var buf safeBuffer
	provider := testProvider(map[string]string{
		"my-password": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	restore := s.LockdownStreams()
	defer restore()

	fmt.Println("Password is: my-password")
	fmt.Fprintln(os.Stderr, "Error: my-password failed")

	restore()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("my-password")) {
		t.Errorf("secret leaked through lockdown: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("<REDACTED>")) {
		t.Errorf("placeholder not found: %q", got)
	}
}

// ============================================================================
// Longest-Match Tests
// ============================================================================

// TestOverlappingSecrets verifies longest-first matching
func TestOverlappingSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"SECRET":          "<SHORT>",
		"SECRET_EXTENDED": "<LONG>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("Value: SECRET_EXTENDED"))
	s.Flush()

	got := buf.String()
	// Should use longest match
	if got != "Value: <LONG>" {
		t.Errorf("got %q, want %q", got, "Value: <LONG>")
	}
	// Should not have partial replacement
	if bytes.Contains([]byte(got), []byte("SECRET")) {
		t.Errorf("secret not fully replaced: %q", got)
	}
}

// ============================================================================
// Encoding Variants Tests
// ============================================================================

// TestEncodingVariants verifies variant detection
func TestEncodingVariants(t *testing.T) {
	var buf bytes.Buffer

	source := func() []Pattern {
		return []Pattern{
			{Value: []byte("test"), Placeholder: []byte("<REDACTED>")},
		}
	}
	provider := NewPatternProviderWithVariants(source)
	s := New(&buf, WithSecretProvider(provider))

	// Write secret in various encodings
	s.Write([]byte("raw: test\n"))
	s.Write([]byte("hex: 74657374\n"))
	s.Write([]byte("base64: dGVzdA==\n"))
	s.Flush()

	got := buf.String()

	// All variants should be scrubbed
	if bytes.Contains([]byte(got), []byte("test")) {
		t.Errorf("raw secret leaked: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("74657374")) {
		t.Errorf("hex variant leaked: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("dGVzdA")) {
		t.Errorf("base64 variant leaked: %q", got)
	}

	// Should have placeholders
	wantCount := 3
	gotCount := bytes.Count([]byte(got), []byte("<REDACTED>"))
	if gotCount != wantCount {
		t.Errorf("got %d placeholders, want %d in: %q", gotCount, wantCount, got)
	}
}

// ============================================================================
// Zeroization Tests
// ============================================================================

// TestCloseZeroization verifies Close() zeroizes buffers
func TestCloseZeroization(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"secret123": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	// Write to create carry buffer
	s.Write([]byte("secret123"))

	// Close should zeroize
	s.Close()

	// Verify output was flushed
	if !bytes.Contains(buf.Bytes(), []byte("<REDACTED>")) {
		t.Errorf("output not flushed on close")
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

// TestEmptyWrite verifies empty writes are handled
func TestEmptyWrite(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	n, err := s.Write([]byte{})
	if err != nil {
		t.Errorf("empty write failed: %v", err)
	}
	if n != 0 {
		t.Errorf("empty write returned %d, want 0", n)
	}
}

// TestNoProvider verifies scrubber works without provider
func TestNoProvider(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf) // No provider

	s.Write([]byte("data with no secrets"))
	s.Flush()

	want := "data with no secrets"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEmptySecret verifies empty secrets are ignored
func TestEmptySecret(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"":       "<EMPTY>",
		"secret": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("Value: secret"))
	s.Flush()

	want := "Value: <REDACTED>"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestProviderError verifies that provider errors prevent unsanitized output.
func TestProviderError(t *testing.T) {
	var output bytes.Buffer

	// Create provider that rejects all chunks
	rejectProvider := &errorProvider{err: errors.New("secret detected")}
	scrubber := New(&output, WithSecretProvider(rejectProvider))

	// Write should fail and not write anything
	_, err := scrubber.Write([]byte("sensitive data"))
	if err == nil {
		t.Fatal("Expected error from provider, got nil")
	}
	if err.Error() != "secret detected" {
		t.Errorf("Expected 'secret detected', got %v", err)
	}

	// Output should be empty (no unsanitized data written)
	if output.Len() != 0 {
		t.Errorf("Expected no output, got: %q", output.String())
	}
}

// TestProviderErrorInFrame verifies that provider errors in frames prevent output.
func TestProviderErrorInFrame(t *testing.T) {
	var output bytes.Buffer

	// Create provider that rejects all chunks
	rejectProvider := &errorProvider{err: errors.New("secret in frame")}
	scrubber := New(&output, WithSecretProvider(rejectProvider))

	// Start frame and write
	scrubber.StartFrame("test")
	scrubber.Write([]byte("sensitive data in frame"))

	// EndFrame should fail and not write anything
	err := scrubber.EndFrame()
	if err == nil {
		t.Fatal("Expected error from provider, got nil")
	}
	if err.Error() != "secret in frame" {
		t.Errorf("Expected 'secret in frame', got %v", err)
	}

	// Output should be empty (no unsanitized data written)
	if output.Len() != 0 {
		t.Errorf("Expected no output, got: %q", output.String())
	}
}

// TestProviderErrorInFlush verifies that provider errors in Flush prevent output.
func TestProviderErrorInFlush(t *testing.T) {
	var output bytes.Buffer

	// Create provider that accepts first write but rejects flush
	callCount := 0
	conditionalProvider := &conditionalErrorProvider{
		shouldError: func() bool {
			callCount++
			return callCount > 1 // Error on second call (flush)
		},
		err: errors.New("secret in flush"),
	}

	scrubber := New(&output, WithSecretProvider(conditionalProvider))

	// First write succeeds (provider accepts it)
	_, err := scrubber.Write([]byte("ok"))
	if err != nil {
		t.Fatalf("First write should succeed, got: %v", err)
	}

	// Flush should fail
	err = scrubber.Flush()
	if err == nil {
		t.Fatal("Expected error from provider in Flush, got nil")
	}
	if err.Error() != "secret in flush" {
		t.Errorf("Expected 'secret in flush', got %v", err)
	}
}

// errorProvider always returns an error
type errorProvider struct {
	err error
}

func (p *errorProvider) HandleChunk(chunk []byte) ([]byte, error) {
	return nil, p.err
}

func (p *errorProvider) MaxSecretLength() int {
	return 100
}

// conditionalErrorProvider returns error based on condition
type conditionalErrorProvider struct {
	shouldError func() bool
	err         error
}

func (p *conditionalErrorProvider) HandleChunk(chunk []byte) ([]byte, error) {
	if p.shouldError() {
		return nil, p.err
	}
	return chunk, nil
}

func (p *conditionalErrorProvider) MaxSecretLength() int {
	return 100
}
