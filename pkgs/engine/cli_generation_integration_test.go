package engine

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestCLIGenerationIntegration tests the complete pipeline:
// 1. Parse a commands file
// 2. Generate Go CLI code
// 3. Compile the generated code
// 4. Test that individual commands work properly
// 5. Verify it doesn't execute all commands at once
func TestCLIGenerationIntegration(t *testing.T) {
	// Test commands that should generate proper subcommands
	testCommands := `
# Test CLI commands
var PROJECT = "testproject"

# Simple command
hello: {
    echo "Hello from @var(PROJECT)!"
}

# Command with multiple steps
build: {
    echo "Building @var(PROJECT)..."
    echo "Build complete"
}

# Command that would be problematic if run automatically
dangerous: {
    echo "This should only run when called explicitly"
    echo "Not during CLI initialization"
}
`

	t.Run("GenerateProperCLI", func(t *testing.T) {
		// Create temporary directory for test
		tempDir := t.TempDir()

		// Parse the test commands
		program, err := parser.Parse(strings.NewReader(testCommands))
		if err != nil {
			t.Fatalf("Failed to parse commands: %v", err)
		}

		// Generate CLI code
		engine := New(program)

		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("Failed to generate CLI code: %v", err)
		}

		generationResult := result

		generatedCode := generationResult.String()

		// Basic sanity checks on generated code
		if len(generatedCode) < 100 {
			t.Fatalf("Generated code too short, likely failed: %d chars", len(generatedCode))
		}

		// Use engine to write both main.go and go.mod files
		err = engine.WriteFiles(generationResult, tempDir, "testcli")
		if err != nil {
			t.Fatalf("Failed to write generated files: %v", err)
		}

		// Test 1: Code should compile
		t.Run("CodeCompiles", func(t *testing.T) {
			// First run go mod tidy to generate go.sum
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = tempDir
			tidyOutput, err := tidyCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, string(tidyOutput))
			}

			cmd := exec.Command("go", "build", "-o", "testcli", ".")
			cmd.Dir = tempDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Generated code failed to compile: %v\nOutput: %s\nGenerated code:\n%s",
					err, string(output), generatedCode)
			}
		})

		binaryPath := filepath.Join(tempDir, "testcli")

		// Test 2: CLI should show help and available commands
		t.Run("ShowsHelpWithCommands", func(t *testing.T) {
			cmd := exec.Command(binaryPath, "--help")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI help failed: %v\nOutput: %s", err, string(output))
			}

			helpOutput := string(output)

			// Should show available commands
			expectedCommands := []string{"hello", "build", "dangerous"}
			for _, cmdName := range expectedCommands {
				if !strings.Contains(helpOutput, cmdName) {
					t.Errorf("Help output missing command '%s'\nOutput: %s", cmdName, helpOutput)
				}
			}
		})

		// Test 3: Individual commands should work
		t.Run("IndividualCommandsWork", func(t *testing.T) {
			// Test hello command
			cmd := exec.Command(binaryPath, "hello")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Hello command failed: %v\nOutput: %s", err, string(output))
			} else {
				outputStr := string(output)
				if !strings.Contains(outputStr, "Hello from testproject!") {
					t.Errorf("Hello command output incorrect. Expected 'Hello from testproject!', got: %s", outputStr)
				}
			}

			// Test build command
			cmd = exec.Command(binaryPath, "build")
			output, err = cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Build command failed: %v\nOutput: %s", err, string(output))
			} else {
				outputStr := string(output)
				if !strings.Contains(outputStr, "Building testproject...") {
					t.Errorf("Build command output incorrect. Expected 'Building testproject...', got: %s", outputStr)
				}
				if !strings.Contains(outputStr, "Build complete") {
					t.Errorf("Build command output incomplete. Expected 'Build complete', got: %s", outputStr)
				}
			}
		})

		// Test 4: CRITICAL - CLI should NOT execute all commands when just showing help
		t.Run("DoesNotExecuteAllCommandsOnHelp", func(t *testing.T) {
			cmd := exec.Command(binaryPath, "--help")
			cmd.Dir = tempDir

			// Set a timeout to prevent hanging
			timeout := 5 * time.Second
			timer := time.AfterFunc(timeout, func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			})
			defer timer.Stop()

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Help command failed or timed out: %v\nOutput: %s", err, string(output))
			}

			helpOutput := string(output)

			// These strings should NOT appear in help output (they're from command execution)
			forbiddenOutputs := []string{
				"Hello from testproject!",
				"Building testproject...",
				"Build complete",
				"This should only run when called explicitly",
				"Not during CLI initialization",
			}

			for _, forbidden := range forbiddenOutputs {
				if strings.Contains(helpOutput, forbidden) {
					t.Errorf("CRITICAL: Help output contains command execution output '%s' - CLI is executing all commands!\nFull output: %s",
						forbidden, helpOutput)
				}
			}
		})

		// Test 5: Running without arguments should show help, not execute commands
		t.Run("NoArgsShowsHelpNotExecution", func(t *testing.T) {
			cmd := exec.Command(binaryPath)
			cmd.Dir = tempDir

			timeout := 5 * time.Second
			timer := time.AfterFunc(timeout, func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			})
			defer timer.Stop()

			output, err := cmd.CombinedOutput()

			// It's OK if it exits with error (showing help), but it shouldn't execute commands
			outputStr := string(output)

			// We don't fail the test on error since CLI might exit with error showing help
			_ = err

			// Should contain help/usage information
			if !strings.Contains(outputStr, "Usage:") && !strings.Contains(outputStr, "Available Commands:") {
				t.Errorf("CLI without args should show usage help, got: %s", outputStr)
			}

			// Should NOT contain command execution output
			forbiddenOutputs := []string{
				"Hello from testproject!",
				"Building testproject...",
				"This should only run when called explicitly",
			}

			for _, forbidden := range forbiddenOutputs {
				if strings.Contains(outputStr, forbidden) {
					t.Errorf("CRITICAL: CLI without args executed commands! Found '%s' in output: %s",
						forbidden, outputStr)
				}
			}
		})

		// Test 6: Verify Cobra CLI structure in generated code
		t.Run("GeneratesCobraCLI", func(t *testing.T) {
			// Check that generated code uses Cobra properly
			requiredPatterns := []string{
				"github.com/spf13/cobra", // Should import cobra
				"&cobra.Command{",        // Should create cobra commands
				"rootCmd.AddCommand",     // Should add subcommands
				"rootCmd.Execute()",      // Should execute root command
			}

			for _, pattern := range requiredPatterns {
				if !strings.Contains(generatedCode, pattern) {
					t.Errorf("Generated code missing required Cobra pattern '%s'\nGenerated code:\n%s",
						pattern, generatedCode)
				}
			}

			// Should NOT contain immediate execution patterns that would execute all commands
			forbiddenPatterns := []string{
				"func() {\n\t\techo", // Immediate function execution with command content
			}

			for _, pattern := range forbiddenPatterns {
				if strings.Contains(generatedCode, pattern) {
					t.Errorf("Generated code contains forbidden immediate execution pattern '%s'\nThis suggests it's generating an execution script instead of a CLI\nGenerated code:\n%s",
						pattern, generatedCode)
				}
			}
		})
	})
}

// TestComplexCLIGeneration tests with more complex scenarios including decorators
func TestComplexCLIGeneration(t *testing.T) {
	complexCommands := `
var ENV = "test"
var PORT = "8080"

# Simple command
start: {
    echo "Starting server on port @var(PORT)"
}

# Command with parallel decorator
deploy: @parallel {
    echo "Deploying frontend..."  
    echo "Deploying backend..."
}

# Command with variable usage
info: {
    echo "Environment: @var(ENV)"
    echo "Port: @var(PORT)"
}
`

	tempDir := t.TempDir()

	// Parse the complex commands
	program, err := parser.Parse(strings.NewReader(complexCommands))
	if err != nil {
		t.Fatalf("Failed to parse complex commands: %v", err)
	}

	// Generate CLI code
	engine := New(program)

	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate complex CLI code: %v", err)
	}

	generationResult := result

	generatedCode := generationResult.String()

	// Use engine to write both main.go and go.mod files
	err = engine.WriteFiles(generationResult, tempDir, "complexcli")
	if err != nil {
		t.Fatalf("Failed to write generated files: %v", err)
	}

	// First run go mod tidy to generate go.sum
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, string(tidyOutput))
	}

	// Compile
	cmd := exec.Command("go", "build", "-o", "complexcli", ".")
	cmd.Dir = tempDir
	output2, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Complex CLI failed to compile: %v\nOutput: %s\nGenerated code:\n%s",
			err, string(output2), generatedCode)
	}

	binaryPath := filepath.Join(tempDir, "complexcli")

	// Test that decorators work in CLI mode
	t.Run("DecoratorCommandsWork", func(t *testing.T) {
		// Test parallel command
		cmd := exec.Command(binaryPath, "deploy")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Deploy command failed: %v\nOutput: %s", err, string(output))
		} else {
			outputStr := string(output)
			if !strings.Contains(outputStr, "Deploying frontend...") || !strings.Contains(outputStr, "Deploying backend...") {
				t.Errorf("Parallel deploy command output incorrect: %s", outputStr)
			}
		}

		// Test variable expansion
		cmd = exec.Command(binaryPath, "info")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Info command failed: %v\nOutput: %s", err, string(output))
		} else {
			outputStr := string(output)
			if !strings.Contains(outputStr, "Environment: test") || !strings.Contains(outputStr, "Port: 8080") {
				t.Errorf("Variable expansion in info command incorrect: %s", outputStr)
			}
		}
	})
}
