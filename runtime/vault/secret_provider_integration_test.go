package vault

import (
	"bytes"
	"io"
	"testing"

	"github.com/opal-lang/opal/runtime/streamscrub"
)

// TestSecretProvider_VariableDeclaration tests that variables declared in Vault
// are properly scrubbed by the SecretProvider
func TestSecretProvider_VariableDeclaration(t *testing.T) {
	planKey := make([]byte, 32)
	v := NewWithPlanKey(planKey)

	// Declare and resolve a variable (simulating what planner does)
	exprID := v.DeclareVariable("API_KEY", "literal:sk-secret-123")
	v.StoreUnresolvedValue(exprID, "sk-secret-123")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Get the SecretProvider
	provider := v.SecretProvider()
	if provider == nil {
		t.Fatal("SecretProvider() returned nil")
	}

	// Test scrubbing
	testInput := "The API key is sk-secret-123 for production"

	var buf bytes.Buffer
	scrubber := streamscrub.New(&buf, streamscrub.WithSecretProvider(provider))

	_, err := io.WriteString(scrubber, testInput)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Flush any remaining data
	if err := scrubber.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	scrubbed := buf.String()

	// The output should NOT contain the raw secret
	if bytes.Contains([]byte(scrubbed), []byte("sk-secret-123")) {
		t.Errorf("Scrubbed output still contains raw secret 'sk-secret-123'")
		t.Logf("Output: %s", scrubbed)
	}

	// The output SHOULD contain a DisplayID (opal:...)
	if !bytes.Contains([]byte(scrubbed), []byte("opal:")) {
		t.Errorf("Scrubbed output should contain DisplayID marker (opal:...)")
		t.Logf("Output: %s", scrubbed)
	}

	t.Logf("Input:  %s", testInput)
	t.Logf("Output: %s", scrubbed)
}

// TestSecretProvider_MultipleVariables tests scrubbing multiple variables
func TestSecretProvider_MultipleVariables(t *testing.T) {
	planKey := make([]byte, 32)
	v := NewWithPlanKey(planKey)

	// Declare multiple variables
	exprID1 := v.DeclareVariable("API_KEY", "literal:sk-secret-123")
	v.StoreUnresolvedValue(exprID1, "sk-secret-123")
	v.MarkTouched(exprID1)
	v.ResolveAllTouched()

	exprID2 := v.DeclareVariable("TOKEN", "literal:token-456")
	v.StoreUnresolvedValue(exprID2, "token-456")
	v.MarkTouched(exprID2)
	v.ResolveAllTouched()

	exprID3 := v.DeclareVariable("PASSWORD", "literal:pass-789")
	v.StoreUnresolvedValue(exprID3, "pass-789")
	v.MarkTouched(exprID3)
	v.ResolveAllTouched()

	// Get the SecretProvider
	provider := v.SecretProvider()

	// Test scrubbing
	testInput := "API: sk-secret-123, Token: token-456, Pass: pass-789"

	var buf bytes.Buffer
	scrubber := streamscrub.New(&buf, streamscrub.WithSecretProvider(provider))

	_, err := io.WriteString(scrubber, testInput)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := scrubber.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	scrubbed := buf.String()

	// None of the raw secrets should be in the output
	secrets := []string{"sk-secret-123", "token-456", "pass-789"}
	for _, secret := range secrets {
		if bytes.Contains([]byte(scrubbed), []byte(secret)) {
			t.Errorf("Scrubbed output still contains raw secret '%s'", secret)
		}
	}

	// Should contain DisplayID markers
	if !bytes.Contains([]byte(scrubbed), []byte("opal:")) {
		t.Error("Scrubbed output should contain DisplayID markers")
	}

	t.Logf("Input:  %s", testInput)
	t.Logf("Output: %s", scrubbed)
}

// TestSecretProvider_UnresolvedNotScrubbed tests that unresolved expressions
// are NOT scrubbed (they don't have values yet)
func TestSecretProvider_UnresolvedNotScrubbed(t *testing.T) {
	planKey := make([]byte, 32)
	v := NewWithPlanKey(planKey)

	// Declare but DON'T resolve
	v.DeclareVariable("API_KEY", "literal:sk-secret-123")
	// Note: NOT calling MarkResolved

	// Get the SecretProvider
	provider := v.SecretProvider()

	// Test scrubbing
	testInput := "The API key is sk-secret-123"

	var buf bytes.Buffer
	scrubber := streamscrub.New(&buf, streamscrub.WithSecretProvider(provider))

	_, err := io.WriteString(scrubber, testInput)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := scrubber.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	scrubbed := buf.String()

	// Since the variable was NOT resolved, the raw value should pass through
	// (SecretProvider only scrubs resolved expressions)
	if scrubbed != testInput {
		t.Errorf("Unresolved expression should not be scrubbed")
		t.Logf("Expected: %s", testInput)
		t.Logf("Got:      %s", scrubbed)
	}
}

// TestSecretProvider_EmptyValue tests that empty string values are handled correctly
func TestSecretProvider_EmptyValue(t *testing.T) {
	planKey := make([]byte, 32)
	v := NewWithPlanKey(planKey)

	// Declare and resolve with empty string
	exprID := v.DeclareVariable("EMPTY", "literal:")
	v.StoreUnresolvedValue(exprID, "")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Get the SecretProvider
	provider := v.SecretProvider()

	// Test scrubbing
	testInput := "Before  After"

	var buf bytes.Buffer
	scrubber := streamscrub.New(&buf, streamscrub.WithSecretProvider(provider))

	_, err := io.WriteString(scrubber, testInput)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := scrubber.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	scrubbed := buf.String()

	// Empty values should not cause issues
	// Output should be unchanged (empty string matches nothing)
	if scrubbed != testInput {
		t.Errorf("Empty value should not affect scrubbing")
		t.Logf("Expected: %s", testInput)
		t.Logf("Got:      %s", scrubbed)
	}
}

// TestSecretProvider_LongestFirst tests that longer secrets are matched first
// (important for cases like "secret" vs "secret-key")
func TestSecretProvider_LongestFirst(t *testing.T) {
	planKey := make([]byte, 32)
	v := NewWithPlanKey(planKey)

	// Declare secrets where one is a prefix of another
	exprID1 := v.DeclareVariable("SHORT", "literal:secret")
	v.StoreUnresolvedValue(exprID1, "secret")
	v.MarkTouched(exprID1)
	v.ResolveAllTouched()

	exprID2 := v.DeclareVariable("LONG", "literal:secret-key-123")
	v.StoreUnresolvedValue(exprID2, "secret-key-123")
	v.MarkTouched(exprID2)
	v.ResolveAllTouched()

	// Get the SecretProvider
	provider := v.SecretProvider()

	// Test scrubbing - the longer secret should be matched first
	testInput := "The value is secret-key-123 and also secret"

	var buf bytes.Buffer
	scrubber := streamscrub.New(&buf, streamscrub.WithSecretProvider(provider))

	_, err := io.WriteString(scrubber, testInput)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := scrubber.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	scrubbed := buf.String()

	// Neither raw secret should be in the output
	if bytes.Contains([]byte(scrubbed), []byte("secret-key-123")) {
		t.Error("Scrubbed output still contains 'secret-key-123'")
	}
	if bytes.Contains([]byte(scrubbed), []byte("secret")) {
		t.Error("Scrubbed output still contains 'secret'")
	}

	t.Logf("Input:  %s", testInput)
	t.Logf("Output: %s", scrubbed)
}
