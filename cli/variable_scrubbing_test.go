package main

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/opal/runtime/streamscrub"
	"github.com/aledsdavies/opal/runtime/vault"
	"github.com/spf13/cobra"
)

// TestVariableScrubbing_EndToEnd tests the complete CLI→Planner→Scrubber integration.
// Verifies that variables declared during planning are properly scrubbed in output.
func TestVariableScrubbing_EndToEnd(t *testing.T) {
	// Create temporary opal file
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "test.opl")

	secretValue := "my-secret-value"
	source := `var SECRET = "my-secret-value"
echo "The secret is: @var.SECRET"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create vault and scrubber (simulating CLI setup)
	planKey := make([]byte, 32)
	_, err = rand.Read(planKey)
	if err != nil {
		t.Fatalf("Failed to generate plan key: %v", err)
	}
	vlt := vault.NewWithPlanKey(planKey)

	opalGen, err := streamscrub.NewOpalPlaceholderGenerator()
	if err != nil {
		t.Fatalf("Failed to create placeholder generator: %v", err)
	}

	var outputBuf bytes.Buffer
	scrubber := streamscrub.New(&outputBuf,
		streamscrub.WithPlaceholderFunc(opalGen.PlaceholderFunc()),
		streamscrub.WithSecretProvider(vlt.SecretProvider()))

	// Redirect stdout/stderr through scrubber
	restore := scrubber.LockdownStreams()
	defer restore()

	// Run command (script mode - no command name)
	cmd := &cobra.Command{}
	exitCode, err := runCommand(cmd, "", opalFile, false, false, false, true, false, vlt, scrubber, &outputBuf)
	if err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}

	if err := scrubber.Close(); err != nil {
		t.Fatalf("Failed to close scrubber: %v", err)
	}

	output := outputBuf.String()

	// The echo command outputs the secret, which should be scrubbed
	// CRITICAL: Verify raw secret is NOT in output
	if strings.Contains(output, secretValue) {
		t.Errorf("Output contains raw secret %q - scrubbing failed!", secretValue)
		t.Logf("Output: %s", output)
	}

	// Verify output contains DisplayID marker (scrubbed value)
	if !strings.Contains(output, "opal:") {
		t.Error("Output should contain DisplayID marker (opal:...)")
		t.Logf("Output: %s", output)
	}

	t.Logf("Scrubbing successful - secret replaced with DisplayID")
}

// TestVariableScrubbing_MultipleVariables tests that multiple variables are scrubbed in output.
func TestVariableScrubbing_MultipleVariables(t *testing.T) {
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "multiple.opl")

	source := `var API_KEY = "sk-secret-123"
var TOKEN = "token-456"
var PASSWORD = "pass-789"
echo "API: @var.API_KEY, Token: @var.TOKEN, Pass: @var.PASSWORD"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create vault and scrubber
	planKey := make([]byte, 32)
	_, err = rand.Read(planKey)
	if err != nil {
		t.Fatalf("Failed to generate plan key: %v", err)
	}
	vlt := vault.NewWithPlanKey(planKey)

	opalGen, err := streamscrub.NewOpalPlaceholderGenerator()
	if err != nil {
		t.Fatalf("Failed to create placeholder generator: %v", err)
	}

	var outputBuf bytes.Buffer
	scrubber := streamscrub.New(&outputBuf,
		streamscrub.WithPlaceholderFunc(opalGen.PlaceholderFunc()),
		streamscrub.WithSecretProvider(vlt.SecretProvider()))

	// Run command in dry-run mode (plan only, don't execute)
	// Executor doesn't yet support DisplayID resolution, so we can't execute
	cmd := &cobra.Command{}
	dryRun := true
	exitCode, err := runCommand(cmd, "", opalFile, dryRun, false, false, true, false, vlt, scrubber, &outputBuf)
	if err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}

	// Now test scrubbing by manually writing secrets to the scrubber
	// The vault was populated during planning, so scrubber should have the patterns
	secrets := []string{"sk-secret-123", "token-456", "pass-789"}
	testInput := "API: sk-secret-123, Token: token-456, Pass: pass-789"
	_, err = io.WriteString(scrubber, testInput)
	if err != nil {
		t.Fatalf("Failed to write to scrubber: %v", err)
	}

	if err := scrubber.Close(); err != nil {
		t.Fatalf("Failed to close scrubber: %v", err)
	}

	output := outputBuf.String()

	// Verify all secrets are scrubbed
	for _, secret := range secrets {
		if strings.Contains(output, secret) {
			t.Errorf("Output contains raw secret %q - scrubbing failed!", secret)
		}
	}

	// Verify output contains DisplayID markers (scrubbed values)
	if !strings.Contains(output, "opal:") {
		t.Error("Output should contain DisplayID markers")
	}

	t.Logf("All %d secrets scrubbed successfully", len(secrets))
}
