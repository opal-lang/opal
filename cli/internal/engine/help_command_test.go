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

// TestGeneratedCliHelpCommand tests that the help command works on generated CLIs
func TestGeneratedCliHelpCommand(t *testing.T) {
	// Create a simple CLI with a few commands
	input := `
build: echo "Building..."
test: echo "Testing..."
deploy: echo "Deploying..."
`

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
	tmpDir, err := os.MkdirTemp("", "devcmd-help-test-*")
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

	// TEST 1: Execute help command with timeout to see if it hangs
	t.Run("HelpCommandDoesNotHang", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		execCmd := exec.CommandContext(ctx, binaryPath, "help")
		output, err := execCmd.CombinedOutput()

		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("CRITICAL BUG: help command hangs and times out after 5 seconds")
		}

		if err != nil {
			t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("Help command output:\n%s", outputStr)

		// Check that help output contains expected elements
		expectedElements := []string{
			"Available Commands:", // Cobra help format
			"build",               // Our commands should be listed
			"test",
			"deploy",
		}

		for _, element := range expectedElements {
			if !strings.Contains(outputStr, element) {
				t.Errorf("Help output missing expected element: %s", element)
			}
		}
	})

	// TEST 2: Execute help with specific command
	t.Run("HelpForSpecificCommand", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		execCmd := exec.CommandContext(ctx, binaryPath, "help", "build")
		output, err := execCmd.CombinedOutput()

		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("CRITICAL BUG: help build command hangs and times out")
		}

		if err != nil {
			t.Fatalf("Help build command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("Help build command output:\n%s", outputStr)

		// Should contain build-specific help
		if !strings.Contains(outputStr, "build") {
			t.Error("Help for build command should contain 'build'")
		}
	})

	// TEST 3: Execute with no arguments (should show help)
	t.Run("NoArgumentsShowsHelp", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		execCmd := exec.CommandContext(ctx, binaryPath)
		output, err := execCmd.CombinedOutput()

		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("CRITICAL BUG: CLI with no arguments hangs and times out")
		}

		// No arguments might return error code but should still produce output
		outputStr := string(output)
		t.Logf("No arguments output (err=%v):\n%s", err, outputStr)

		// Should show some kind of help or usage information
		helpIndicators := []string{
			"Available Commands:",
			"Usage:",
			"help",
		}

		hasHelpIndicator := false
		for _, indicator := range helpIndicators {
			if strings.Contains(outputStr, indicator) {
				hasHelpIndicator = true
				break
			}
		}

		if !hasHelpIndicator {
			t.Error("CLI with no arguments should show help information")
		}
	})
}

// TestGeneratedCliBasicExecution tests that basic commands work
func TestGeneratedCliBasicExecution(t *testing.T) {
	input := `simple: echo "Hello from generated CLI"`

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

	tmpDir, err := os.MkdirTemp("", "devcmd-basic-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Write files
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
)`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Build
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if tidyOutput, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, tidyOutput)
	}

	binaryPath := filepath.Join(tmpDir, "testcli")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, mainGoPath)
	buildCmd.Dir = tmpDir
	if buildOutput, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, buildOutput)
	}

	// Execute simple command
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binaryPath, "simple")
	output, err := execCmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("CRITICAL BUG: simple command hangs and times out")
	}

	if err != nil {
		t.Fatalf("Simple command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	t.Logf("Simple command output: %s", outputStr)

	if !strings.Contains(outputStr, "Hello from generated CLI") {
		t.Error("Simple command output missing expected text")
	}
}
