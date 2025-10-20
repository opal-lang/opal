package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllExecutionModes verifies all 4 execution modes work correctly with tree structure
func TestAllExecutionModes(t *testing.T) {
	// Build opal binary for testing
	opalBin := buildOpalBinary(t)
	defer os.Remove(opalBin)

	// Create test opal file
	testFile := createTestFile(t, `
fun hello = echo "Hello from Opal!"
fun test_and = echo "A" && echo "B"
fun test_or = echo "A" || echo "B"
fun complex = echo "A" && echo "B" || echo "C"
`)
	defer os.Remove(testFile)

	t.Run("Mode1_DirectExecution", func(t *testing.T) {
		// Direct execution: opal -f file.opl command
		output := runOpal(t, opalBin, "-f", testFile, "hello")
		assert.Equal(t, "Hello from Opal!\n", output)
	})

	t.Run("Mode2_QuickPlan", func(t *testing.T) {
		// Quick plan: opal -f file.opl command --dry-run
		output := runOpal(t, opalBin, "-f", testFile, "hello", "--dry-run", "--no-color")
		assert.Contains(t, output, "hello:")
		assert.Contains(t, output, "@shell echo \"Hello from Opal!\"")
	})

	t.Run("Mode3_ResolvedPlan", func(t *testing.T) {
		// Resolved plan: opal -f file.opl command --dry-run --resolve > plan
		planFile := filepath.Join(t.TempDir(), "test.plan")

		// Generate plan
		cmd := exec.Command(opalBin, "-f", testFile, "hello", "--dry-run", "--resolve")
		planData, err := cmd.Output()
		require.NoError(t, err)

		// Save plan to file
		err = os.WriteFile(planFile, planData, 0o644)
		require.NoError(t, err)

		// Verify plan file exists and has content
		stat, err := os.Stat(planFile)
		require.NoError(t, err)
		assert.Greater(t, stat.Size(), int64(0), "Plan file should not be empty")

		// Verify plan starts with OPAL magic bytes
		assert.True(t, len(planData) > 4, "Plan should have magic bytes")
		assert.Equal(t, "OPAL", string(planData[0:4]), "Plan should start with OPAL magic")
	})

	t.Run("Mode4_ContractVerifiedExecution", func(t *testing.T) {
		// Contract-verified execution: opal --plan file.plan -f file.opl
		planFile := filepath.Join(t.TempDir(), "test.plan")

		// Generate plan
		cmd := exec.Command(opalBin, "-f", testFile, "hello", "--dry-run", "--resolve")
		planData, err := cmd.Output()
		require.NoError(t, err)
		err = os.WriteFile(planFile, planData, 0o644)
		require.NoError(t, err)

		// Execute from plan
		output := runOpal(t, opalBin, "--plan", planFile, "-f", testFile)
		assert.Equal(t, "Hello from Opal!\n", output)
	})
}

// TestOperatorsInAllModes verifies operator tree structure works in all modes
func TestOperatorsInAllModes(t *testing.T) {
	opalBin := buildOpalBinary(t)
	defer os.Remove(opalBin)

	testFile := createTestFile(t, `
fun test_and = echo "A" && echo "B"
fun test_or = echo "A" || echo "B"
fun complex = echo "A" && echo "B" || echo "C"
`)
	defer os.Remove(testFile)

	tests := []struct {
		name           string
		command        string
		expectedOutput string
		expectedTree   string
	}{
		{
			name:           "AND operator",
			command:        "test_and",
			expectedOutput: "A\nB\n",
			expectedTree:   "@shell echo \"A\" && @shell echo \"B\"",
		},
		{
			name:           "OR operator",
			command:        "test_or",
			expectedOutput: "A\n", // B skipped due to short-circuit
			expectedTree:   "@shell echo \"A\" || @shell echo \"B\"",
		},
		{
			name:           "Complex precedence",
			command:        "complex",
			expectedOutput: "A\nB\n", // C skipped: (A && B) succeeds, so || skips
			expectedTree:   "@shell echo \"A\" && @shell echo \"B\" || @shell echo \"C\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test direct execution
			t.Run("DirectExecution", func(t *testing.T) {
				output := runOpal(t, opalBin, "-f", testFile, tt.command)
				assert.Equal(t, tt.expectedOutput, output)
			})

			// Test dry-run shows tree structure
			t.Run("DryRun", func(t *testing.T) {
				output := runOpal(t, opalBin, "-f", testFile, tt.command, "--dry-run", "--no-color")
				assert.Contains(t, output, tt.expectedTree)
			})

			// Test contract-verified execution
			t.Run("ContractVerified", func(t *testing.T) {
				planFile := filepath.Join(t.TempDir(), "test.plan")

				// Generate plan
				cmd := exec.Command(opalBin, "-f", testFile, tt.command, "--dry-run", "--resolve")
				planData, err := cmd.Output()
				require.NoError(t, err)
				err = os.WriteFile(planFile, planData, 0o644)
				require.NoError(t, err)

				// Execute from plan
				output := runOpal(t, opalBin, "--plan", planFile, "-f", testFile)
				assert.Equal(t, tt.expectedOutput, output)
			})
		})
	}
}

// TestContractVerificationFailure verifies that contract verification catches changes
func TestContractVerificationFailure(t *testing.T) {
	opalBin := buildOpalBinary(t)
	defer os.Remove(opalBin)

	// Create original file
	testFile := createTestFile(t, `fun hello = echo "Original"`)
	defer os.Remove(testFile)

	// Generate plan from original
	planFile := filepath.Join(t.TempDir(), "test.plan")
	cmd := exec.Command(opalBin, "-f", testFile, "hello", "--dry-run", "--resolve")
	planData, err := cmd.Output()
	require.NoError(t, err)
	err = os.WriteFile(planFile, planData, 0o644)
	require.NoError(t, err)

	// Modify source file
	err = os.WriteFile(testFile, []byte(`fun hello = echo "Modified"`), 0o644)
	require.NoError(t, err)

	// Try to execute - should fail contract verification
	cmd = exec.Command(opalBin, "--plan", planFile, "-f", testFile)
	output, err := cmd.CombinedOutput()

	// Should fail with contract verification error
	assert.Error(t, err, "Contract verification should fail when source changes")
	assert.Contains(t, string(output), "contract", "Error should mention contract verification")
}

// Helper: Build opal binary for testing
func buildOpalBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	opalBin := filepath.Join(tmpDir, "opal")

	// Build from current directory (cli package)
	cmd := exec.Command("go", "build", "-o", opalBin, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build opal: %v\nOutput: %s", err, output)
	}

	return opalBin
}

// Helper: Create test file with content
func createTestFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "test.opl")
	err := os.WriteFile(tmpFile, []byte(strings.TrimSpace(content)), 0o644)
	require.NoError(t, err)

	return tmpFile
}

// Helper: Run opal and return output
func runOpal(t *testing.T, opalBin string, args ...string) string {
	t.Helper()

	cmd := exec.Command(opalBin, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("opal failed: %v\nStderr: %s\nStdout: %s", err, exitErr.Stderr, output)
		}
		t.Fatalf("opal failed: %v", err)
	}

	return string(output)
}

// TestScriptModeExecution tests script mode (no command name argument)
func TestScriptModeExecution(t *testing.T) {
	opalBin := buildOpalBinary(t)
	defer os.Remove(opalBin)

	t.Run("ScriptModeWithTopLevelCommands", func(t *testing.T) {
		// Create script with only top-level commands (no functions)
		scriptFile := createTestFile(t, `
echo "Line 1"
echo "Line 2"
echo "Line 3"
`)
		defer os.Remove(scriptFile)

		// Execute script mode (no command name argument)
		output := runOpal(t, opalBin, "-f", scriptFile)
		assert.Equal(t, "Line 1\nLine 2\nLine 3\n", output)
	})

	t.Run("ScriptModeWithShebang", func(t *testing.T) {
		// Create script with shebang (shebang is treated as comment)
		scriptFile := createTestFile(t, `
#!/usr/bin/env opal
echo "From shebang script"
echo "Line 2"
`)
		defer os.Remove(scriptFile)

		// Execute script mode
		output := runOpal(t, opalBin, "-f", scriptFile)
		assert.Equal(t, "From shebang script\nLine 2\n", output)
	})

	t.Run("ScriptModeWithFunctionsAndTopLevel", func(t *testing.T) {
		// Script mode should execute only top-level commands, not functions
		scriptFile := createTestFile(t, `
fun deploy = echo "deploying"
fun test = echo "testing"

echo "Top level 1"
echo "Top level 2"
`)
		defer os.Remove(scriptFile)

		// Execute script mode - should only run top-level commands
		output := runOpal(t, opalBin, "-f", scriptFile)
		assert.Equal(t, "Top level 1\nTop level 2\n", output)
		assert.NotContains(t, output, "deploying", "Functions should not execute in script mode")
		assert.NotContains(t, output, "testing", "Functions should not execute in script mode")
	})

	t.Run("ScriptModeDryRun", func(t *testing.T) {
		// Dry-run in script mode should show all top-level commands
		scriptFile := createTestFile(t, `
echo "Line 1"
echo "Line 2"
`)
		defer os.Remove(scriptFile)

		// Dry-run script mode
		output := runOpal(t, opalBin, "-f", scriptFile, "--dry-run", "--no-color")
		assert.Contains(t, output, "@shell echo \"Line 1\"")
		assert.Contains(t, output, "@shell echo \"Line 2\"")
	})

	t.Run("CommandModeStillWorks", func(t *testing.T) {
		// Verify command mode still works (1 arg = command mode)
		scriptFile := createTestFile(t, `
fun hello = echo "Hello from function"

echo "Top level command"
`)
		defer os.Remove(scriptFile)

		// Command mode - should execute only the function
		output := runOpal(t, opalBin, "-f", scriptFile, "hello")
		assert.Equal(t, "Hello from function\n", output)
		assert.NotContains(t, output, "Top level", "Top-level should not execute in command mode")
	})

	t.Run("ShebangPreventsCommandMode", func(t *testing.T) {
		// Files with shebang cannot be used in command mode
		scriptFile := createTestFile(t, `
#!/usr/bin/env opal
fun deploy = echo "deploying"

echo "Top level"
`)
		defer os.Remove(scriptFile)

		// Try to run in command mode - should fail
		cmd := exec.Command(opalBin, "-f", scriptFile, "deploy")
		output, err := cmd.CombinedOutput()

		assert.Error(t, err, "Should fail when trying to run command mode on shebang script")
		assert.Contains(t, string(output), "shebang", "Error should mention shebang")
		assert.Contains(t, string(output), "deploy", "Error should mention the function name")
	})
}
