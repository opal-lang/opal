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

// TestSequentialExecutionBug_RealWorld tests the actual bug we observed where
// generated CLI commands stop executing after the first command in a block
func TestSequentialExecutionBug_RealWorld(t *testing.T) {
	// This test recreates the exact scenario where we observed the bug:
	// 1. Generate a CLI with multiple commands in a block
	// 2. Build the CLI binary
	// 3. Execute the CLI command
	// 4. Verify ALL commands in the block execute (not just the first one)

	input := `build: {
    echo "Step 1: Starting"  
    echo "Step 2: Middle"
    echo "Step 3: Finished"
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
	tmpDir, err := os.MkdirTemp("", "devcmd-sequential-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

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

	// Execute the build command and capture ALL output
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binaryPath, "build")
	output, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute CLI: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	t.Logf("CLI execution output:\n%s", outputStr)

	// CRITICAL TEST: All three echo commands should appear in the output
	expectedOutputs := []string{
		"Step 1: Starting",
		"Step 2: Middle",
		"Step 3: Finished",
	}

	var missingOutputs []string
	for _, expected := range expectedOutputs {
		if !strings.Contains(outputStr, expected) {
			missingOutputs = append(missingOutputs, expected)
		}
	}

	if len(missingOutputs) > 0 {
		t.Errorf("CRITICAL BUG: Sequential execution failed. Missing outputs: %v", missingOutputs)
		t.Errorf("This indicates commands stopped executing after the first one")
		t.Errorf("Full output was: %s", outputStr)

		// This is the critical bug - only the first command executes
		if strings.Contains(outputStr, "Step 1: Starting") &&
			!strings.Contains(outputStr, "Step 2: Middle") {
			t.Error("CONFIRMED BUG: First command executed but subsequent commands did not")
		}
	} else {
		t.Logf("✅ All commands executed successfully in sequence")
	}
}

// TestSequentialExecution_CompilerError tests that we don't generate invalid Go code
// that has unreachable statements after early returns
func TestSequentialExecution_CompilerError(t *testing.T) {
	input := `build: {
    echo "First command"
    echo "Second command" 
    echo "Third command"
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	generatedCode := result.String()

	// Check for the specific pattern that causes the bug:
	// A return statement followed by more executable code (which becomes unreachable)
	lines := strings.Split(generatedCode, "\n")

	var hasEarlyReturn bool
	var hasUnreachableCode bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for return statements that aren't at the end of a function
		if strings.Contains(trimmed, "return CommandResult{") && !strings.Contains(trimmed, "// Final return") {
			// Check if there are more executable statements after this return
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])

				// Skip empty lines and comments
				if nextLine == "" || strings.HasPrefix(nextLine, "//") {
					continue
				}

				// If we find another command execution after a return, that's the bug
				if strings.Contains(nextLine, "ExecCmd") || strings.Contains(nextLine, "exec.Command") {
					hasEarlyReturn = true
					hasUnreachableCode = true
					t.Errorf("CRITICAL BUG: Found unreachable command execution after return statement")
					t.Errorf("Return statement at line %d: %s", i+1, trimmed)
					t.Errorf("Unreachable code at line %d: %s", j+1, nextLine)
					break
				}

				// If we hit a closing brace, we're out of the function
				if nextLine == "}" {
					break
				}
			}
		}
	}

	if hasEarlyReturn && hasUnreachableCode {
		t.Error("Generated code has the sequential execution bug pattern")
	} else {
		t.Log("✅ Generated code does not have early return bug pattern")
	}
}
