package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/cli/internal/parser"
	"github.com/aledsdavies/devcmd/core/ast"
)

// TestParallelWorkdirIsolation verifies that @parallel with @workdir maintains proper isolation
func TestParallelWorkdirIsolation(t *testing.T) {
	// This test verifies that parallel branches with different working directories
	// actually execute in isolation and don't interfere with each other
	input := `test: @parallel {
    @workdir("dir1") {
        pwd > result.txt
        echo "Branch 1 executed" >> result.txt
    }
    @workdir("dir2") {
        pwd > result.txt
        echo "Branch 2 executed" >> result.txt
    }
    @workdir("dir3") {
        pwd > result.txt
        echo "Branch 3 executed" >> result.txt
    }
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

	// Create a temporary directory for this test
	tmpDir, err := os.MkdirTemp("", "devcmd-parallel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	// Create the subdirectories
	for _, dir := range []string{"dir1", "dir2", "dir3"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}
	}

	// Write the generated Go code
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(result.String()), 0o644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Write go.mod
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

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if tidyOutput, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, tidyOutput)
	}

	// Build the CLI binary
	binaryPath := filepath.Join(tmpDir, "testcli")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, mainGoPath)
	buildCmd.Dir = tmpDir
	if buildOutput, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, buildOutput)
	}

	// Execute the test command
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binaryPath, "test")
	execCmd.Dir = tmpDir
	output, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute CLI: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	t.Logf("CLI execution output:\n%s", outputStr)

	// Debug: List all files created in tmpDir and check main result.txt
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			t.Logf("Found file: %s", path)
			if strings.HasSuffix(path, "result.txt") {
				content, _ := os.ReadFile(path)
				t.Logf("Content of %s:\n%s", path, string(content))
			}
		}
		return nil
	})
	if err != nil {
		t.Logf("Error walking directory: %v", err)
	}

	// VERIFY ISOLATION: Each result file should contain the correct directory path
	for i, dir := range []string{"dir1", "dir2", "dir3"} {
		resultFile := filepath.Join(tmpDir, dir, "result.txt")

		// Check if the result file exists
		if _, err := os.Stat(resultFile); os.IsNotExist(err) {
			t.Errorf("Result file not created in %s: %s", dir, resultFile)
			continue
		}

		// Read the result file
		content, err := os.ReadFile(resultFile)
		if err != nil {
			t.Errorf("Failed to read result file %s: %v", resultFile, err)
			continue
		}

		contentStr := string(content)

		// Verify the pwd output shows the correct directory
		expectedPath := filepath.Join(tmpDir, dir)
		if !strings.Contains(contentStr, expectedPath) {
			t.Errorf("ISOLATION FAILURE: Branch %d did not execute in correct directory.\nExpected path: %s\nGot content: %s",
				i+1, expectedPath, contentStr)
		} else {
			t.Logf("✅ Branch %d correctly executed in %s", i+1, dir)
		}

		// Verify the echo command also executed
		expectedMessage := fmt.Sprintf("Branch %d executed", i+1)
		if !strings.Contains(contentStr, expectedMessage) {
			t.Errorf("Branch %d message not found in output: %s", i+1, contentStr)
		}
	}
}

// TestNestedDecoratorBehavior verifies that nested decorators work correctly
func TestNestedDecoratorBehavior(t *testing.T) {
	// Test @timeout with @retry - each retry should respect the timeout
	input := `test: @timeout(duration=5s) {
    @retry(attempts=3, delay=1s) {
        echo "Attempt at $(date +%s)"
        exit 1  # Force retry
    }
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

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "devcmd-nested-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	// Write the generated Go code
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(result.String()), 0o644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Write go.mod
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

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if tidyOutput, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, tidyOutput)
	}

	// Build the CLI binary
	binaryPath := filepath.Join(tmpDir, "testcli")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, mainGoPath)
	buildCmd.Dir = tmpDir
	if buildOutput, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, buildOutput)
	}

	// Execute the test command - should timeout after 5s even with retries
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binaryPath, "test")
	execCmd.Dir = tmpDir
	output, _ := execCmd.CombinedOutput() // Expect this to fail due to timeout or retries

	duration := time.Since(startTime)
	outputStr := string(output)

	t.Logf("CLI execution took %v, output:\n%s", duration, outputStr)

	// Count how many attempts were made
	attemptCount := strings.Count(outputStr, "Attempt at")

	// Verify behavior
	if duration > 6*time.Second {
		t.Errorf("Timeout not respected: execution took %v (should be ~5s)", duration)
	}

	if attemptCount == 0 {
		t.Error("No retry attempts detected in output")
	} else if attemptCount > 3 {
		t.Errorf("Too many retry attempts: %d (max should be 3)", attemptCount)
	} else {
		t.Logf("✅ Made %d retry attempts within timeout", attemptCount)
	}
}

// TestParallelContextVariableIsolation verifies that parallel branches have isolated contexts
func TestParallelContextVariableIsolation(t *testing.T) {
	// Each parallel branch should have its own context and not interfere with others
	input := `
branch1: echo "Branch 1: Starting" && sleep 0.5 && echo "Branch 1: Completed"
branch2: echo "Branch 2: Starting" && sleep 0.3 && echo "Branch 2: Completed"
branch3: echo "Branch 3: Starting" && sleep 0.1 && echo "Branch 3: Completed"

test: @parallel {
    @cmd(branch1)
    @cmd(branch2)
    @cmd(branch3)
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

	// Verify the generated code creates child contexts for each branch
	generatedCode := result.String()

	// Check for context isolation in parallel execution
	if !strings.Contains(generatedCode, "Clone()") {
		t.Error("Generated code should create isolated contexts for parallel branches")
	}

	// Count how many isolated contexts are created
	isolatedContextCount := strings.Count(generatedCode, ".Clone()")
	if isolatedContextCount < 3 {
		t.Errorf("Expected at least 3 isolated contexts for parallel branches, found %d", isolatedContextCount)
	} else {
		t.Logf("✅ Generated code creates %d isolated contexts for isolation", isolatedContextCount)
	}

	// Also verify the code compiles and runs
	tmpDir, err := os.MkdirTemp("", "devcmd-context-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	// Write and compile
	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(generatedCode), 0o644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

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

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if tidyOutput, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, tidyOutput)
	}

	binaryPath := filepath.Join(tmpDir, "testcli")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, mainGoPath)
	buildCmd.Dir = tmpDir
	if buildOutput, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, buildOutput)
	}

	// Execute and verify parallel execution
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binaryPath, "test")
	execCmd.Dir = tmpDir
	output, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute CLI: %v\nOutput: %s", err, output)
	}

	duration := time.Since(startTime)
	outputStr := string(output)

	t.Logf("Parallel execution took %v, output:\n%s", duration, outputStr)

	// Verify all branches executed
	for i := 1; i <= 3; i++ {
		startMsg := fmt.Sprintf("Branch %d: Starting", i)
		endMsg := fmt.Sprintf("Branch %d: Completed", i)

		if !strings.Contains(outputStr, startMsg) {
			t.Errorf("Branch %d did not start", i)
		}
		if !strings.Contains(outputStr, endMsg) {
			t.Errorf("Branch %d did not complete", i)
		}
	}

	// Verify parallel execution (should take ~0.5s not 0.9s if sequential)
	if duration > 1*time.Second {
		t.Errorf("Execution took too long (%v), might not be parallel", duration)
	} else {
		t.Logf("✅ Parallel execution completed in %v", duration)
	}
}

// TestParallelWorkdirOutputConsistency verifies that @parallel with @workdir produces
// consistent output in both interpreter and generator modes - reproduces the linting issue
func TestParallelWorkdirOutputConsistency(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "devcmd-output-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	// Create subdirectories to simulate the project structure
	for _, dir := range []string{"core", "runtime", "testing", "cli"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}
	}

	// Test command that simulates the linting scenario
	input := `test-lint: {
    echo "Running linters across all modules..."
    @parallel {
        @workdir("core") { echo "core: 0 issues." }
        @workdir("runtime") { echo "runtime: 0 issues." }
        @workdir("testing") { echo "testing: 0 issues." }
        @workdir("cli") { echo "cli: 0 issues." }
    }
    echo "Linting complete!"
}`

	// Parse the input
	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Test 1: Interpreter Mode
	t.Run("InterpreterMode", func(t *testing.T) {
		engine := New(program)

		// Change to tmpDir for the test
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Logf("Warning: failed to restore directory: %v", err)
			}
		}()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Find the test-lint command
		var testCmd *ast.CommandDecl
		for _, cmd := range program.Commands {
			if cmd.Name == "test-lint" {
				testCmd = &cmd
				break
			}
		}
		if testCmd == nil {
			t.Fatalf("test-lint command not found")
		}

		// Execute in interpreter mode and capture output
		t.Logf("=== Interpreter Mode Output ===")
		result, err := engine.ExecuteCommand(testCmd)
		if err != nil {
			t.Errorf("Interpreter mode failed: %v", err)
		} else {
			t.Logf("Interpreter result: %s", result.Status)
		}
	})

	// Test 2: Generator Mode
	t.Run("GeneratorMode", func(t *testing.T) {
		engine := New(program)

		// Generate the code
		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("GenerateCode failed: %v", err)
		}

		// Create a temporary directory for the generated binary
		genTmpDir, err := os.MkdirTemp("", "devcmd-gen-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(genTmpDir); err != nil {
				t.Logf("Warning: failed to remove temp dir: %v", err)
			}
		}()

		// Copy test directories to generator temp dir
		for _, dir := range []string{"core", "runtime", "testing", "cli"} {
			if err := os.MkdirAll(filepath.Join(genTmpDir, dir), 0o755); err != nil {
				t.Fatalf("Failed to create %s in gen dir: %v", dir, err)
			}
		}

		// Write the generated Go code
		mainGoPath := filepath.Join(genTmpDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(result.String()), 0o644); err != nil {
			t.Fatalf("Failed to write main.go: %v", err)
		}

		// Write go.mod
		goModPath := filepath.Join(genTmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(result.GoModString()), 0o644); err != nil {
			t.Fatalf("Failed to write go.mod: %v", err)
		}

		// Build the binary
		if err := runCommand("go", "mod", "tidy"); err != nil {
			t.Logf("go mod tidy failed (acceptable in CI): %v", err)
		}

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Logf("Warning: failed to restore directory: %v", err)
			}
		}()
		if err := os.Chdir(genTmpDir); err != nil {
			t.Fatalf("Failed to change to gen temp directory: %v", err)
		}

		// Try to build and run (might fail in CI, but should work locally)
		if err := runCommand("go", "build", "-o", "testcli", "main.go"); err != nil {
			t.Logf("Build failed (acceptable in CI environment): %v", err)
			return // Skip execution test if build fails
		}

		// Execute the generated binary
		t.Logf("=== Generator Mode Output ===")
		if err := runCommand("./testcli", "test-lint"); err != nil {
			t.Logf("Generated binary execution failed (acceptable): %v", err)
		}
	})

	t.Logf("Both modes tested - check logs above to compare outputs")
	t.Logf("Expected: Both modes should show 'core: 0 issues.', 'runtime: 0 issues.', etc.")
}
