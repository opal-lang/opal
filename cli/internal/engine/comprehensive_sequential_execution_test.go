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
	"github.com/aledsdavies/devcmd/core/ast"

	// Import builtins to register decorators
	_ "github.com/aledsdavies/devcmd/cli/internal/builtins"
)

// TestSequentialExecution_BothModes is the comprehensive test that verifies
// sequential execution works correctly in both interpreted and generated modes
func TestSequentialExecution_BothModes(t *testing.T) {
	testCases := []struct {
		name            string
		commands        string
		expectSuccess   bool
		expectedFiles   []string // Files that should be created
		unexpectedFiles []string // Files that should NOT be created
		description     string
	}{
		{
			name: "successful_sequential_execution",
			commands: `test: {
    echo "Step 1 executed" > step1.txt
    echo "Step 2 executed" > step2.txt  
    echo "Step 3 executed" > step3.txt
}`,
			expectSuccess: true,
			expectedFiles: []string{"step1.txt", "step2.txt", "step3.txt"},
			description:   "All commands should execute sequentially and create their files",
		},
		{
			name: "failure_stops_execution",
			commands: `test: {
    echo "Step 1 executed" > step1.txt
    false
    echo "Step 3 should not execute" > step3.txt
}`,
			expectSuccess:   false,
			expectedFiles:   []string{"step1.txt"},
			unexpectedFiles: []string{"step3.txt"},
			description:     "Execution should stop after failure, step 3 should not execute",
		},
		{
			name: "complex_operations_sequential",
			commands: `test: {
    mkdir -p testdir
    echo "content" > testdir/file1.txt
    cat testdir/file1.txt > testdir/file2.txt
    ls testdir > listing.txt
}`,
			expectSuccess: true,
			expectedFiles: []string{"testdir/file1.txt", "testdir/file2.txt", "listing.txt"},
			description:   "Complex file operations should execute sequentially",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.description)

			// Test both modes
			t.Run("interpreted_mode", func(t *testing.T) {
				testSequentialExecutionInterpreted(t, tc.commands, tc.expectSuccess, tc.expectedFiles, tc.unexpectedFiles)
			})

			t.Run("generated_mode", func(t *testing.T) {
				testSequentialExecutionGenerated(t, tc.commands, tc.expectSuccess, tc.expectedFiles, tc.unexpectedFiles)
			})
		})
	}
}

// testSequentialExecutionInterpreted tests sequential execution in interpreted mode
func testSequentialExecutionInterpreted(t *testing.T, commands string, expectSuccess bool, expectedFiles, unexpectedFiles []string) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Logf("Warning: failed to restore original directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Parse the commands
	program, err := parser.Parse(strings.NewReader(commands))
	if err != nil {
		t.Fatalf("Failed to parse commands: %v", err)
	}

	// Create engine and execute in interpreted mode
	engine := New(program)

	// Find the test command
	var testCmd *ast.CommandDecl
	for _, cmd := range program.Commands {
		if cmd.Name == "test" {
			testCmd = &cmd
			break
		}
	}
	if testCmd == nil {
		t.Fatal("Test command not found in parsed program")
	}

	// Execute the command
	result, err := engine.ExecuteCommand(testCmd)

	// Check execution result matches expectation
	if expectSuccess {
		if err != nil {
			t.Errorf("Expected success but got error: %v", err)
		}
		if result != nil && result.Status != "success" {
			t.Errorf("Expected success but command failed with status: %s", result.Status)
		}
	} else {
		if err == nil && (result == nil || result.Status == "success") {
			t.Errorf("Expected failure but command succeeded")
		}
	}

	// Verify expected files were created
	for _, filename := range expectedFiles {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created in interpreted mode", filename)
		} else {
			t.Logf("✅ File %s created successfully in interpreted mode", filename)
		}
	}

	// Verify unexpected files were NOT created
	for _, filename := range unexpectedFiles {
		if _, err := os.Stat(filename); err == nil {
			t.Errorf("Unexpected file %s was created in interpreted mode", filename)
		} else {
			t.Logf("✅ File %s correctly not created in interpreted mode", filename)
		}
	}
}

// testSequentialExecutionGenerated tests sequential execution in generated mode
func testSequentialExecutionGenerated(t *testing.T, commands string, expectSuccess bool, expectedFiles, unexpectedFiles []string) {
	// Create temporary directory for generated CLI
	tmpDir := t.TempDir()

	// Parse the commands
	program, err := parser.Parse(strings.NewReader(commands))
	if err != nil {
		t.Fatalf("Failed to parse commands: %v", err)
	}

	// Generate the CLI code
	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate CLI: %v", err)
	}

	// Write the generated code
	mainGoPath := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainGoPath, []byte(result.String()), 0o644)
	if err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Create go.mod
	goModContent := `module testcli

go 1.22

require github.com/spf13/cobra v1.8.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(goModPath, []byte(goModContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, output)
	}

	// Build the CLI
	binaryPath := filepath.Join(tmpDir, "testcli")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = tmpDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Create test execution directory
	testDir := filepath.Join(tmpDir, "testrun")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Execute the CLI command
	execCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(execCtx, binaryPath, "test")
	execCmd.Dir = testDir // Run in test directory so files are created there
	output, err := execCmd.CombinedOutput()

	// Check execution result matches expectation
	if expectSuccess {
		if err != nil {
			t.Errorf("Expected success but CLI failed: %v\nOutput: %s", err, output)
		}
	} else {
		if err == nil {
			t.Errorf("Expected failure but CLI succeeded\nOutput: %s", output)
		} else {
			t.Logf("✅ CLI failed as expected: %v", err)
		}
	}

	t.Logf("CLI output:\n%s", output)

	// Verify expected files were created in test directory
	for _, filename := range expectedFiles {
		fullPath := filepath.Join(testDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created in generated mode", filename)
		} else {
			t.Logf("✅ File %s created successfully in generated mode", filename)
		}
	}

	// Verify unexpected files were NOT created
	for _, filename := range unexpectedFiles {
		fullPath := filepath.Join(testDir, filename)
		if _, err := os.Stat(fullPath); err == nil {
			t.Errorf("Unexpected file %s was created in generated mode", filename)
		} else {
			t.Logf("✅ File %s correctly not created in generated mode", filename)
		}
	}
}

// TestSequentialExecutionConsistency verifies that both modes produce identical results
func TestSequentialExecutionConsistency(t *testing.T) {
	commands := `test: {
    echo "line1" > output.txt
    echo "line2" >> output.txt
    echo "line3" >> output.txt
}`

	// Test both modes and compare file contents
	t.Run("consistency_check", func(t *testing.T) {
		// Get interpreted mode results
		interpretedDir := t.TempDir()
		interpretedContent := testModeAndGetFileContent(t, commands, interpretedDir, "interpreted")

		// Get generated mode results
		generatedDir := t.TempDir()
		generatedContent := testModeAndGetFileContent(t, commands, generatedDir, "generated")

		// Compare results
		if interpretedContent != generatedContent {
			t.Errorf("Mode inconsistency detected!\nInterpreted result: %q\nGenerated result: %q",
				interpretedContent, generatedContent)
		} else {
			t.Logf("✅ Both modes produced identical results: %q", interpretedContent)
		}
	})
}

// testModeAndGetFileContent is a helper that tests a mode and returns the content of output.txt
func testModeAndGetFileContent(t *testing.T, commands, testDir, mode string) string {
	if mode == "interpreted" {
		// Test interpreted mode
		oldDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(oldDir); err != nil {
				t.Logf("Warning: failed to restore directory: %v", err)
			}
		}()
		if err := os.Chdir(testDir); err != nil {
			t.Logf("Warning: failed to change directory: %v", err)
		}

		program, _ := parser.Parse(strings.NewReader(commands))

		engine := New(program)
		var testCmd *ast.CommandDecl
		for _, cmd := range program.Commands {
			if cmd.Name == "test" {
				testCmd = &cmd
				break
			}
		}

		_, err := engine.ExecuteCommand(testCmd)
		if err != nil {
			t.Errorf("Interpreted mode failed: %v", err)
			return ""
		}

	} else {
		// Test generated mode
		// Parse, generate, build and execute (similar to testSequentialExecutionGenerated)
		program, _ := parser.Parse(strings.NewReader(commands))

		engine := New(program)
		result, _ := engine.GenerateCode(program)

		mainGoPath := filepath.Join(testDir, "main.go")
		if err := os.WriteFile(mainGoPath, []byte(result.String()), 0o644); err != nil {
			t.Logf("Warning: failed to write main.go: %v", err)
			return ""
		}

		goModContent := `module testcli
go 1.22
require github.com/spf13/cobra v1.8.1
require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)`
		goModPath := filepath.Join(testDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
			t.Logf("Warning: failed to write go.mod: %v", err)
			return ""
		}

		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = testDir
		if err := tidyCmd.Run(); err != nil {
			t.Logf("Warning: go mod tidy failed: %v", err)
		}

		buildCmd := exec.Command("go", "build", "-o", "testcli", ".")
		buildCmd.Dir = testDir
		if err := buildCmd.Run(); err != nil {
			t.Logf("Warning: build failed: %v", err)
			return ""
		}

		execCmd := exec.Command("./testcli", "test")
		execCmd.Dir = testDir
		if err := execCmd.Run(); err != nil {
			t.Logf("Warning: execution failed: %v", err)
		}
	}

	// Read the output file
	outputPath := filepath.Join(testDir, "output.txt")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Errorf("Failed to read output.txt in %s mode: %v", mode, err)
		return ""
	}

	return strings.TrimSpace(string(content))
}
