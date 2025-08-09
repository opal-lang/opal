package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/cli/internal/parser"
)

// TestWorkdirSequentialExecution tests that commands before and after @workdir execute
func TestWorkdirSequentialExecution(t *testing.T) {
	input := `test: {
    echo "Before workdir"
    pwd
    @workdir("testdir") {
        echo "Inside workdir"
        pwd
    }
    echo "After workdir"
    pwd
}`

	// Parse the input
	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Create engine and generate code
	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	generatedCode := result.String()
	if generatedCode == "" {
		t.Fatal("Generated code should not be empty")
	}

	// Create a temporary directory for this test
	tmpDir, err := os.MkdirTemp("", "devcmd-workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Create the testdir subdirectory that the workdir command expects
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create testdir: %v", err)
	}

	// Write the generated Go code
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(generatedCode), 0o644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Write a minimal go.mod
	goModContent := `module testcli

go 1.24.3

require github.com/spf13/cobra v1.9.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Run go mod tidy to get dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, tidyOutput)
	}

	// Build the CLI binary
	binaryPath := filepath.Join(tmpDir, "testcli")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, mainGoPath)
	buildCmd.Dir = tmpDir
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, buildOutput)
	}

	// Execute the test command and capture ALL output
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binaryPath, "test")
	execCmd.Dir = tmpDir // Run from tmpDir so testdir is available
	output, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute CLI: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	t.Logf("CLI execution output:\n%s", outputStr)

	// CRITICAL TEST: Verify both sequential execution AND directory changes
	expectedOutputs := []string{
		"Before workdir",
		"Inside workdir",
		"After workdir",
	}

	var missingOutputs []string
	for _, expected := range expectedOutputs {
		if !strings.Contains(outputStr, expected) {
			missingOutputs = append(missingOutputs, expected)
		}
	}

	if len(missingOutputs) > 0 {
		t.Errorf("CRITICAL BUG: Sequential execution failed with @workdir decorator. Missing outputs: %v", missingOutputs)
		t.Errorf("Full output was: %s", outputStr)
	}

	// VERIFY WORKING DIRECTORY ACTUALLY CHANGED
	lines := strings.Split(outputStr, "\n")
	var pwdOutputs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// pwd outputs are absolute paths
		if strings.HasPrefix(line, "/") && !strings.Contains(line, "echo") {
			pwdOutputs = append(pwdOutputs, line)
		}
	}

	if len(pwdOutputs) >= 3 {
		// First pwd should be tmpDir
		// Second pwd should be tmpDir/testdir (inside @workdir)
		// Third pwd should be tmpDir again (after @workdir)

		if !strings.HasSuffix(pwdOutputs[1], "/testdir") {
			t.Errorf("CRITICAL BUG: @workdir did not change directory! Inside @workdir pwd was: %s", pwdOutputs[1])
		} else {
			t.Logf("✅ Verified @workdir changed to testdir: %s", pwdOutputs[1])
		}

		// Verify we returned to original directory after @workdir
		if pwdOutputs[0] != pwdOutputs[2] {
			t.Errorf("WARNING: Directory not restored after @workdir. Before: %s, After: %s", pwdOutputs[0], pwdOutputs[2])
		} else {
			t.Logf("✅ Verified directory restored after @workdir")
		}
	} else {
		t.Errorf("Could not verify directory changes - expected 3 pwd outputs, got %d: %v", len(pwdOutputs), pwdOutputs)
	}
}

// TestBuildCommandSequentialExecution tests the actual build command from commands.cli
func TestBuildCommandSequentialExecution(t *testing.T) {
	// This tests the actual build command that was failing
	input := `build: {
    echo "🔨 Building PROJECT CLI..."
    @workdir("cli") { go build -o ../PROJECT ./main.go }
    echo "✅ Built: ./PROJECT"
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	generatedCode := result.String()
	t.Logf("Generated code length: %d characters", len(generatedCode))

	// Test that the generated code has the correct structure:
	// 1. First echo command
	// 2. Workdir decorator code
	// 3. Second echo command

	// Check that all expected elements are present in generated code
	expectedElements := []string{
		`"echo \"🔨 Building PROJECT CLI...\""`, // First command (with proper Go string escaping)
		`@workdir(\"cli\")`,                    // Workdir decorator comment (with proper Go string escaping)
		`"go build -o ../PROJECT ./main.go"`,   // Command inside workdir
		`"echo \"✅ Built: ./PROJECT\""`,        // Final command (with proper Go string escaping)
	}

	for _, element := range expectedElements {
		if !strings.Contains(generatedCode, element) {
			t.Errorf("Generated code missing expected element: %s", element)
		}
	}

	// Most importantly, check that there are NO early returns that would break the sequence
	lines := strings.Split(generatedCode, "\n")
	var foundEarlyReturn bool

	for i, line := range lines {
		if strings.Contains(line, "return CommandResult{") &&
			!strings.Contains(line, "// Final return") {
			// Check if there are more commands after this return
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if strings.Contains(nextLine, "echo") && strings.Contains(nextLine, "ExecCmd") {
					foundEarlyReturn = true
					t.Errorf("CRITICAL: Found early return at line %d that would prevent execution of command at line %d", i+1, j+1)
					t.Errorf("Return line: %s", strings.TrimSpace(line))
					t.Errorf("Unreachable line: %s", nextLine)
				}
			}
		}
	}

	if !foundEarlyReturn {
		t.Logf("✅ Generated code has correct sequential execution structure")
	}
}
