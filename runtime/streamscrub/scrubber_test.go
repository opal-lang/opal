package streamscrub

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

// safeBuffer is a thread-safe bytes.Buffer for testing
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

// TestBasicScrubbing - simplest possible test
func TestBasicScrubbing(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Write some output
	input := []byte("hello world")
	n, err := s.Write(input)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("Write returned %d, want %d", n, len(input))
	}

	// Flush to get output
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Should pass through unchanged (no secrets registered)
	if got := buf.String(); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

// TestSimpleSecretRedaction - register a secret and scrub it
func TestSimpleSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register a secret
	secret := []byte("my-secret-key")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Write output containing the secret
	input := []byte("The key is: my-secret-key")
	s.Write(input)
	s.Flush()

	// Secret should be redacted
	want := "The key is: <REDACTED>"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFrameBuffering - buffer output during frame, flush after
func TestFrameBuffering(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Start a frame
	s.StartFrame("test-frame")

	// Write during frame - should be buffered
	s.Write([]byte("buffered output"))

	// Nothing should be written yet
	if buf.Len() != 0 {
		t.Errorf("output written during frame, want buffered")
	}

	// End frame with a secret
	secret := []byte("secret123")
	s.EndFrame([][]byte{secret})

	// Now output should be flushed (no secret in this output, so unchanged)
	want := "buffered output"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFrameScrubbingHierarchical - secret registered in frame scrubs frame output
func TestFrameScrubbingHierarchical(t *testing.T) {
	var buf bytes.Buffer
	// Use fixed key for deterministic testing
	key := make([]byte, 32)
	gen, _ := NewPlaceholderGeneratorWithKey(key)
	s := New(&buf, WithPlaceholderFunc(gen.PlaceholderFunc()))
	defer s.Close()

	// Start a frame
	s.StartFrame("decorator-frame")

	// Decorator prints its secret during execution
	s.Write([]byte("Loading secret: secret123"))

	// End frame and register the secret
	secret := []byte("secret123")
	s.EndFrame([][]byte{secret})

	// Frame output should be scrubbed before flushing
	// With fixed key, placeholder is deterministic
	got := buf.String()
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("secret leaked: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("<REDACTED:")) {
		t.Errorf("keyed placeholder not found: %q", got)
	}
}

// TestChunkBoundarySafety - secret split across multiple writes
func TestChunkBoundarySafety(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register a secret
	secret := []byte("secret-value-123")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Write secret split across 3 chunks
	s.Write([]byte("prefix secret-"))
	s.Write([]byte("value-"))
	s.Write([]byte("123 suffix"))

	// Flush to get final output
	s.Flush()

	// Secret should be scrubbed even though split
	want := "prefix <REDACTED> suffix"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestNestedFrames - inner frame can access outer frame's secrets
func TestNestedFrames(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Outer frame
	s.StartFrame("outer")
	s.Write([]byte("outer: "))

	// Register outer secret
	outerSecret := []byte("outer-secret")
	s.EndFrame([][]byte{outerSecret})

	// Inner frame
	s.StartFrame("inner")
	s.Write([]byte("inner prints outer: outer-secret"))

	// Register inner secret
	innerSecret := []byte("inner-secret")
	s.EndFrame([][]byte{innerSecret})

	// Both secrets should be scrubbed
	got := buf.String()
	if bytes.Contains([]byte(got), outerSecret) {
		t.Errorf("outer secret leaked: %q", got)
	}
	if bytes.Contains([]byte(got), innerSecret) {
		t.Errorf("inner secret leaked: %q", got)
	}
}

// TestEncodingVariants - secrets in hex/base64 are also scrubbed
func TestEncodingVariants(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register secret with variants
	secret := []byte("test")
	s.RegisterSecretWithVariants(secret)

	// Write secret in various encodings
	// Hex: 74657374
	s.Write([]byte("hex: 74657374\n"))
	// Base64: dGVzdA==
	s.Write([]byte("base64: dGVzdA==\n"))
	// Raw
	s.Write([]byte("raw: test\n"))

	s.Flush()

	got := buf.String()
	// All variants should be scrubbed
	if bytes.Contains([]byte(got), []byte("74657374")) {
		t.Errorf("hex variant leaked: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("dGVzdA")) {
		t.Errorf("base64 variant leaked: %q", got)
	}
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("raw secret leaked: %q", got)
	}
	// Should have keyed placeholder
	if !bytes.Contains([]byte(got), []byte("<REDACTED:")) {
		t.Errorf("keyed placeholder missing: %q", got)
	}
}

// TestLockdownStreams - stdout/stderr are redirected through scrubber
func TestLockdownStreams(t *testing.T) {
	var buf safeBuffer // Use thread-safe buffer for concurrent writes
	s := New(&buf)

	// Register a secret
	secret := []byte("my-password")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Lockdown streams
	restore := s.LockdownStreams()
	defer restore()

	// Print to stdout (should go through scrubber)
	fmt.Println("Password is: my-password")

	// Print to stderr (should also go through scrubber)
	fmt.Fprintln(os.Stderr, "Error: my-password failed")

	// Restore streams (defer will call this, but we call explicitly for testing)
	restore()

	// Check output was scrubbed
	got := buf.String()
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("secret leaked through lockdown: %q", got)
	}
	if !bytes.Contains([]byte(got), placeholder) {
		t.Errorf("placeholder not found in output: %q", got)
	}
}

// TestCloseZeroization - Close() zeroizes sensitive buffers
func TestCloseZeroization(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Write some data to carry buffer
	secret := []byte("secret123")
	s.RegisterSecret(secret, []byte("<REDACTED>"))
	s.Write([]byte("partial"))

	// Start a frame with buffered data
	s.StartFrame("test")
	s.Write([]byte("buffered data"))

	// Close should flush and zeroize
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify buffers are cleared
	if len(s.carry) != 0 {
		t.Errorf("carry not cleared after Close, len=%d", len(s.carry))
	}
	if len(s.frames) != 0 {
		t.Errorf("frames not cleared after Close, len=%d", len(s.frames))
	}
}

// TestIdempotentRestore - verify restore function can be called multiple times
func TestIdempotentRestore(t *testing.T) {
	var buf safeBuffer
	s := New(&buf)

	// Lockdown streams
	restore := s.LockdownStreams()

	// Call restore multiple times - should not panic
	restore()
	restore()
	restore()

	// Should still work after multiple calls
	fmt.Println("test output")
}

// TestConcurrentWrites - verify thread safety with concurrent writes
func TestConcurrentWrites(t *testing.T) {
	var buf safeBuffer
	s := New(&buf)

	// Register a secret
	secret := []byte("secret123")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Launch multiple goroutines writing concurrently
	var wg sync.WaitGroup
	numWriters := 10
	writesPerWriter := 100

	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				msg := fmt.Sprintf("writer %d: message %d with secret123\n", id, j)
				s.Write([]byte(msg))
			}
		}(i)
	}

	wg.Wait()
	s.Flush()

	// Verify no secrets leaked
	got := buf.String()
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("secret leaked in concurrent writes: %q", got)
	}

	// Verify placeholder is present
	if !bytes.Contains([]byte(got), placeholder) {
		t.Errorf("placeholder not found in concurrent writes output")
	}
}

// ============================================================================
// RED TEAM SECURITY TESTS - Phase 1 (Critical)
// ============================================================================

// TestSplitBoundaryRedaction - secret split across multiple writes must be caught
func TestSplitBoundaryRedaction(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	defer s.Close()

	secret := []byte("SECRET_TOKEN_12345")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Split secret across 2 writes
	part1 := []byte("start-of-output " + string(secret[:8]))
	part2 := []byte(string(secret[8:]) + " end-of-output\n")

	// No active frame (execution mode): streaming should redact safely
	if _, err := s.Write(part1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write(part2); err != nil {
		t.Fatal(err)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if bytes.Contains([]byte(got), secret) {
		t.Fatalf("SECURITY: secret leaked across boundary: %q in %q", secret, got)
	}
	if !bytes.Contains([]byte(got), placeholder) {
		t.Fatalf("placeholder not present, want %q in %q", placeholder, got)
	}
}

// TestSplitBoundaryFourWrites - secret split across 4 writes (worst case)
func TestSplitBoundaryFourWrites(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	defer s.Close()

	secret := []byte("VERYLONGSECRETTOKEN123456789")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Split into 4 chunks
	chunk1 := []byte("prefix " + string(secret[:7]))
	chunk2 := []byte(string(secret[7:14]))
	chunk3 := []byte(string(secret[14:21]))
	chunk4 := []byte(string(secret[21:]) + " suffix\n")

	s.Write(chunk1)
	s.Write(chunk2)
	s.Write(chunk3)
	s.Write(chunk4)
	s.Flush()

	got := buf.String()
	if bytes.Contains([]byte(got), secret) {
		t.Fatalf("SECURITY: secret leaked across 4-way split: %q", got)
	}
	if !bytes.Contains([]byte(got), placeholder) {
		t.Fatalf("placeholder missing: %q", got)
	}
}

// TestNestedFramesDontLeak - inner frame must scrub with outer secrets
func TestNestedFramesDontLeak(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	defer s.Close()

	outerSecret := []byte("OUTER_SECRET")
	innerSecret := []byte("INNER_SECRET")

	// Start outer frame
	s.StartFrame("outer")
	s.Write([]byte("outer prints pre-secret\n"))

	// End outer frame, registering outer secret
	s.EndFrame([][]byte{outerSecret})

	// Start inner frame AFTER outer ended
	s.StartFrame("inner")
	// Inner frame prints BOTH secrets
	s.Write([]byte("inner prints outer: " + string(outerSecret) + "\n"))
	s.Write([]byte("inner prints inner: " + string(innerSecret) + "\n"))
	s.EndFrame([][]byte{innerSecret})

	got := buf.String()
	// Both secrets should be scrubbed
	if bytes.Contains([]byte(got), outerSecret) {
		t.Fatalf("SECURITY: outer secret leaked in inner frame: %q", got)
	}
	if bytes.Contains([]byte(got), innerSecret) {
		t.Fatalf("SECURITY: inner secret leaked: %q", got)
	}
	// Should have keyed placeholders
	if !bytes.Contains([]byte(got), []byte("<REDACTED:")) {
		t.Fatalf("keyed placeholders missing in output: %q", got)
	}
}

// TestTrulyNestedFrames - inner frame starts BEFORE outer ends
func TestTrulyNestedFrames(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	defer s.Close()

	outerSecret := []byte("OUTER")
	innerSecret := []byte("INNER")

	// Start outer frame
	s.StartFrame("outer")
	s.Write([]byte("outer-start\n"))

	// Start inner frame WHILE outer is still active
	s.StartFrame("inner")
	s.Write([]byte("inner has: " + string(innerSecret) + "\n"))
	s.EndFrame([][]byte{innerSecret}) // End inner first

	// Continue outer frame
	s.Write([]byte("outer has: " + string(outerSecret) + "\n"))
	s.EndFrame([][]byte{outerSecret}) // End outer second

	got := buf.String()
	if bytes.Contains([]byte(got), outerSecret) || bytes.Contains([]byte(got), innerSecret) {
		t.Fatalf("SECURITY: nested frame leaked secrets: %q", got)
	}
}

// TestPanicMidFrameZeroizes - panic during frame must not leak secrets
func TestPanicMidFrameZeroizes(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := []byte("LEAKYSECRET")

	// Ensure Close is called even on panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
		// Close should zeroize buffers
		s.Close()

		// Check that secret didn't leak
		got := buf.String()
		if bytes.Contains([]byte(got), secret) {
			t.Fatalf("SECURITY: secret leaked on panic: %q", got)
		}

		// Verify zeroization (internal check)
		if len(s.carry) != 0 {
			t.Errorf("carry not cleared after panic+close")
		}
		if len(s.frames) != 0 {
			t.Errorf("frames not cleared after panic+close")
		}
	}()

	s.StartFrame("panic-test")
	s.Write([]byte("about to leak: " + string(secret)))
	panic("boom") // Simulate crash mid-frame
}

// TestOverlappingSecrets - longer secret should win (longest-first matching)
func TestOverlappingSecrets(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	defer s.Close()

	// Register shorter secret first
	short := []byte("SECRET")
	shortPH := []byte("<SHORT>")
	s.RegisterSecret(short, shortPH)

	// Register longer secret that contains the short one
	long := []byte("SECRET_EXTENDED")
	longPH := []byte("<LONG>")
	s.RegisterSecret(long, longPH)

	// Write the long secret
	s.Write([]byte("value: " + string(long) + "\n"))
	s.Flush()

	got := buf.String()
	// Should use LONG placeholder (longest-first)
	if !bytes.Contains([]byte(got), longPH) {
		t.Fatalf("longest-first failed: want %q in %q", longPH, got)
	}
	// Should NOT have the short placeholder
	if bytes.Contains([]byte(got), shortPH) {
		t.Fatalf("longest-first failed: got short placeholder %q in %q", shortPH, got)
	}
	// Should NOT leak the secret
	if bytes.Contains([]byte(got), long) {
		t.Fatalf("SECURITY: overlapping secret leaked: %q", got)
	}
}

// TestZeroizationInternals - verify buffers are actually zeroed
func TestZeroizationInternals(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	secret := []byte("SENSITIVE")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Write partial data to create carry buffer
	s.Write([]byte("prefix"))

	// Start a frame to create frame buffer
	s.StartFrame("test")
	s.Write([]byte("frame data with " + string(secret)))

	// Close should zeroize everything
	s.Close()

	// Check carry is zeroed
	for i, b := range s.carry {
		if b != 0 {
			t.Fatalf("SECURITY: carry[%d] not zeroized: %x", i, b)
		}
	}

	// Check frames are cleared
	if len(s.frames) != 0 {
		t.Fatalf("SECURITY: frames not cleared after Close")
	}
}

// TestPropertyNoSecretLeak - property test: scrubbed output never contains secrets
func TestPropertyNoSecretLeak(t *testing.T) {
	secret := []byte("PROPERTY_SECRET_123")
	placeholder := []byte("<REDACTED>")

	// Test with various input patterns
	testCases := []struct {
		name  string
		input []byte
	}{
		{"empty", []byte("")},
		{"no-secret", []byte("just normal text")},
		{"secret-only", secret},
		{"secret-prefix", append([]byte("prefix "), secret...)},
		{"secret-suffix", append(secret, []byte(" suffix")...)},
		{"secret-middle", append(append([]byte("before "), secret...), []byte(" after")...)},
		{"secret-repeated", bytes.Repeat(secret, 3)},
		{"secret-with-newlines", append(append([]byte("line1\n"), secret...), []byte("\nline3")...)},
		{"secret-with-nulls", append(append([]byte{0, 0}, secret...), []byte{0, 0}...)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			s := New(&buf)
			defer s.Close()

			s.RegisterSecret(secret, placeholder)

			// Write and flush
			s.Write(tc.input)
			s.Flush()

			got := buf.Bytes()

			// PROPERTY: Output must never contain the secret
			if bytes.Contains(got, secret) {
				t.Fatalf("SECURITY VIOLATION: secret leaked in %q case: %q", tc.name, got)
			}

			// If input contained secret, output must contain placeholder
			if bytes.Contains(tc.input, secret) && !bytes.Contains(got, placeholder) {
				t.Fatalf("placeholder missing in %q case: %q", tc.name, got)
			}
		})
	}
}

// TestPlaceholderDeterminism - same secret should produce same placeholder
func TestPlaceholderDeterminism(t *testing.T) {
	secret := []byte("MY_SECRET")

	// Create a deterministic placeholder function (simulates keyed hash)
	callCount := 0
	deterministicPlaceholder := func(s []byte) string {
		callCount++
		// Simulate keyed BLAKE2b: same input -> same output
		if bytes.Equal(s, secret) {
			return "<PLACEHOLDER_ABC123>"
		}
		return "<PLACEHOLDER_OTHER>"
	}

	var buf1, buf2 bytes.Buffer
	s1 := New(&buf1, WithPlaceholderFunc(deterministicPlaceholder))
	s2 := New(&buf2, WithPlaceholderFunc(deterministicPlaceholder))

	// Register same secret in both scrubbers
	s1.RegisterSecretWithVariants(secret)
	s2.RegisterSecretWithVariants(secret)

	// Write same content to both
	input := []byte("The secret is: " + string(secret) + "\n")
	s1.Write(input)
	s1.Flush()
	s2.Write(input)
	s2.Flush()

	// Both should produce identical output
	if buf1.String() != buf2.String() {
		t.Fatalf("non-deterministic placeholders:\n  s1: %q\n  s2: %q", buf1.String(), buf2.String())
	}

	// Output should contain the deterministic placeholder
	want := "The secret is: <PLACEHOLDER_ABC123>\n"
	if buf1.String() != want {
		t.Errorf("got %q, want %q", buf1.String(), want)
	}
}

// TestPlaceholderFunctionCalled - verify custom placeholder function is used
func TestPlaceholderFunctionCalled(t *testing.T) {
	var called bool
	customPlaceholder := func(secret []byte) string {
		called = true
		return "<CUSTOM>"
	}

	var buf bytes.Buffer
	s := New(&buf, WithPlaceholderFunc(customPlaceholder))

	secret := []byte("test")
	s.RegisterSecretWithVariants(secret)

	if !called {
		t.Fatal("custom placeholder function was not called")
	}

	// Verify custom placeholder is used
	s.Write([]byte("value: test"))
	s.Flush()

	if !bytes.Contains(buf.Bytes(), []byte("<CUSTOM>")) {
		t.Errorf("custom placeholder not found in output: %q", buf.String())
	}
}

// ============================================================================
// END RED TEAM SECURITY TESTS - Phase 1
// ============================================================================

// TestExpandedEncodingVariants - all encoding variants are scrubbed
func TestExpandedEncodingVariants(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register secret with all variants
	secret := []byte("pass")
	s.RegisterSecretWithVariants(secret)

	tests := []struct {
		name  string
		input string
		want  string // empty means should be redacted
	}{
		{"raw", "raw: pass", ""},
		{"hex-lower", "hex: 70617373", ""},
		{"hex-upper", "hex: 70617373", ""},
		{"base64-std", "b64: cGFzcw==", ""},
		{"base64-raw", "b64raw: cGFzcw", ""},
		{"base64-url", "b64url: cGFzcw==", ""},
		{"percent-lower", "url: %70%61%73%73", ""},
		{"percent-upper", "url: %70%61%73%73", ""},
		{"separator-dash", "sep: p-a-s-s", ""},
		{"separator-underscore", "sep: p_a_s_s", ""},
		{"separator-colon", "sep: p:a:s:s", ""},
		{"separator-dot", "sep: p.a.s.s", ""},
		{"separator-space", "sep: p a s s", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			s.Write([]byte(tt.input))
			s.Flush()

			got := buf.String()
			// Check that the secret variant was redacted
			if tt.want == "" {
				// Should contain keyed placeholder, not original
				if bytes.Contains([]byte(got), secret) {
					t.Errorf("%s: secret leaked in output: %q", tt.name, got)
				}
				if !bytes.Contains([]byte(got), []byte("<REDACTED:")) {
					t.Errorf("%s: keyed placeholder not found in output: %q", tt.name, got)
				}
			}
		})
	}
}
