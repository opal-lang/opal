package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestCommandsCLIGeneration tests that the actual commands.cli file from the project root
// can be parsed and generates valid Go code that compiles and executes properly.
// This ensures our real-world dogfooding CLI doesn't have issues.
func TestCommandsCLIGeneration(t *testing.T) {
	// Get project root directory (../../ from pkgs/engine/)
	projectRoot := filepath.Join("..", "..")
	commandsCliPath := filepath.Join(projectRoot, "commands.cli")

	// Check if commands.cli exists
	if _, err := os.Stat(commandsCliPath); os.IsNotExist(err) {
		t.Skipf("commands.cli not found at %s, skipping test", commandsCliPath)
	}

	// Read the commands.cli file
	commandsContent, err := os.ReadFile(commandsCliPath)
	if err != nil {
		t.Fatalf("Failed to read commands.cli: %v", err)
	}

	t.Run("ParseCommandsCLI", func(t *testing.T) {
		// Parse the commands.cli file
		program, err := parser.Parse(strings.NewReader(string(commandsContent)))
		if err != nil {
			t.Fatalf("Failed to parse commands.cli: %v", err)
		}

		// Basic validation - should have variables and commands
		if len(program.Variables) == 0 {
			t.Error("commands.cli should contain variable declarations")
		}

		if len(program.Commands) == 0 {
			t.Error("commands.cli should contain command definitions")
		}

		// Check for expected key variables
		expectedVars := []string{"PROJECT", "VERSION", "BUILD_TIME", "GO_VERSION"}
		foundVars := make(map[string]bool)
		for _, variable := range program.Variables {
			foundVars[variable.Name] = true
		}

		for _, expectedVar := range expectedVars {
			if !foundVars[expectedVar] {
				t.Errorf("Expected variable '%s' not found in commands.cli", expectedVar)
			}
		}

		// Check for expected key commands
		expectedCommands := []string{"setup", "build", "test", "help", "ci"}
		foundCommands := make(map[string]bool)
		for _, command := range program.Commands {
			foundCommands[command.Name] = true
		}

		for _, expectedCmd := range expectedCommands {
			if !foundCommands[expectedCmd] {
				t.Errorf("Expected command '%s' not found in commands.cli", expectedCmd)
			}
		}

		t.Logf("Successfully parsed commands.cli with %d variables and %d commands",
			len(program.Variables), len(program.Commands))
	})

	t.Run("GenerateCodeFromCommandsCLI", func(t *testing.T) {
		// Parse the commands.cli file
		program, err := parser.Parse(strings.NewReader(string(commandsContent)))
		if err != nil {
			t.Fatalf("Failed to parse commands.cli: %v", err)
		}

		// Generate CLI code
		engine := New(program)
		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("Failed to generate code from commands.cli: %v", err)
		}

		generatedCode := result.String()

		// Basic validation of generated code
		if len(generatedCode) < 1000 {
			t.Fatalf("Generated code too short (%d chars), likely incomplete", len(generatedCode))
		}

		// Check for essential Go CLI patterns
		requiredPatterns := []string{
			"package main",
			"func main()",
			"github.com/spf13/cobra",
			"rootCmd.Execute()",
			"&cobra.Command{",
		}

		for _, pattern := range requiredPatterns {
			if !strings.Contains(generatedCode, pattern) {
				t.Errorf("Generated code missing required pattern '%s'", pattern)
			}
		}

		// Check that some key commands are present in generated code
		keyCommands := []string{"setup", "build", "test", "help"}
		for _, cmd := range keyCommands {
			cmdPattern := `"` + cmd + `"`
			if !strings.Contains(generatedCode, cmdPattern) {
				t.Errorf("Generated code missing command '%s'", cmd)
			}
		}

		t.Logf("Successfully generated %d chars of Go code from commands.cli", len(generatedCode))
	})

	t.Run("CompileGeneratedCLI", func(t *testing.T) {
		// Create temporary directory for compilation test
		tempDir := t.TempDir()

		// Parse and generate
		program, err := parser.Parse(strings.NewReader(string(commandsContent)))
		if err != nil {
			t.Fatalf("Failed to parse commands.cli: %v", err)
		}

		engine := New(program)
		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}

		// Write generated files
		err = engine.WriteFiles(result, tempDir, "dev")
		if err != nil {
			t.Fatalf("Failed to write generated files: %v", err)
		}

		// Run go mod tidy
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = tempDir
		tidyOutput, err := tidyCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, string(tidyOutput))
		}

		// Compile the generated code
		buildCmd := exec.Command("go", "build", "-o", "dev", ".")
		buildCmd.Dir = tempDir
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Compilation failed: %v\nOutput: %s\nGenerated code preview:\n%s",
				err, string(buildOutput), result.String()[:2000]+"...")
		}

		t.Logf("Successfully compiled CLI from commands.cli")
	})

	t.Run("TestGeneratedCLIBasicFunctionality", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()

		// Parse and generate
		program, err := parser.Parse(strings.NewReader(string(commandsContent)))
		if err != nil {
			t.Fatalf("Failed to parse commands.cli: %v", err)
		}

		engine := New(program)
		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}

		// Write and compile
		err = engine.WriteFiles(result, tempDir, "dev")
		if err != nil {
			t.Fatalf("Failed to write generated files: %v", err)
		}

		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = tempDir
		if tidyOutput, err := tidyCmd.CombinedOutput(); err != nil {
			t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, string(tidyOutput))
		}

		buildCmd := exec.Command("go", "build", "-o", "dev", ".")
		buildCmd.Dir = tempDir
		if buildOutput, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Build failed: %v\nOutput: %s", err, string(buildOutput))
		}

		binaryPath := filepath.Join(tempDir, "dev")

		// Test 1: Help should work and not execute commands
		t.Run("HelpWorks", func(t *testing.T) {
			cmd := exec.Command(binaryPath, "--help")
			cmd.Dir = tempDir

			// Set timeout to prevent hanging
			timeout := 10 * time.Second
			timer := time.AfterFunc(timeout, func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			})
			defer timer.Stop()

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Help command failed: %v\nOutput: %s", err, string(output))
			}

			helpOutput := string(output)

			// Should show available commands
			expectedInHelp := []string{"Available Commands:", "Use", "help"}
			for _, expected := range expectedInHelp {
				if !strings.Contains(helpOutput, expected) {
					t.Errorf("Help output missing '%s'\nOutput: %s", expected, helpOutput)
				}
			}

			// Should contain some of our commands
			someCommands := []string{"setup", "build", "test", "help"}
			foundAny := false
			for _, cmd := range someCommands {
				if strings.Contains(helpOutput, cmd) {
					foundAny = true
					break
				}
			}
			if !foundAny {
				t.Errorf("Help output should contain at least one of our commands %v\nOutput: %s",
					someCommands, helpOutput)
			}

			// CRITICAL: Should NOT contain command execution output
			forbiddenInHelp := []string{
				"Setting up devcmd",
				"Building devcmd",
				"Running Go tests",
				"go build -o",
				"echo",
			}

			for _, forbidden := range forbiddenInHelp {
				if strings.Contains(helpOutput, forbidden) {
					t.Errorf("CRITICAL: Help output contains execution output '%s' - commands being executed during help!\nOutput: %s",
						forbidden, helpOutput)
				}
			}
		})

		// Test 2: Individual command should work (use help since it's safe)
		t.Run("IndividualCommandWorks", func(t *testing.T) {
			cmd := exec.Command(binaryPath, "help")
			cmd.Dir = tempDir

			timeout := 10 * time.Second
			timer := time.AfterFunc(timeout, func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			})
			defer timer.Stop()

			output, err := cmd.CombinedOutput()
			// help command might exit with error code, but should produce output
			outputStr := string(output)

			if len(outputStr) < 50 {
				t.Errorf("Help command produced very little output: %s\nError: %v", outputStr, err)
			}

			// Should contain our expected help content
			if !strings.Contains(outputStr, "Development Commands") ||
				!strings.Contains(outputStr, "Quick Start") {
				t.Errorf("Help command should contain our custom help content\nOutput: %s", outputStr)
			}
		})

		t.Logf("Successfully tested basic functionality of generated CLI from commands.cli")
	})
}

// TestCommandsCLIVariableResolution tests that variables in commands.cli are properly resolved
func TestCommandsCLIVariableResolution(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	commandsCliPath := filepath.Join(projectRoot, "commands.cli")

	if _, err := os.Stat(commandsCliPath); os.IsNotExist(err) {
		t.Skipf("commands.cli not found at %s, skipping test", commandsCliPath)
	}

	commandsContent, err := os.ReadFile(commandsCliPath)
	if err != nil {
		t.Fatalf("Failed to read commands.cli: %v", err)
	}

	program, err := parser.Parse(strings.NewReader(string(commandsContent)))
	if err != nil {
		t.Fatalf("Failed to parse commands.cli: %v", err)
	}

	engine := New(program)

	// Process variables into context
	err = engine.processVariablesIntoContext(program)
	if err != nil {
		t.Fatalf("Failed to process variables: %v", err)
	}

	// Check that key variables are properly resolved
	keyVars := []string{"PROJECT", "GO_VERSION"}
	for _, varName := range keyVars {
		if value, exists := engine.ctx.GetVariable(varName); !exists {
			t.Errorf("Variable '%s' not found in context", varName)
		} else if value == "" {
			t.Errorf("Variable '%s' has empty value", varName)
		} else {
			t.Logf("Variable '%s' = '%s'", varName, value)
		}
	}

	// Generate code and check variable usage
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	generatedCode := result.String()

	// Check that variables are properly declared in generated code
	expectedDeclarations := []string{
		"PROJECT := \"devcmd\"",
		"GO_VERSION := \"1.22\"",
	}

	for _, declaration := range expectedDeclarations {
		if !strings.Contains(generatedCode, declaration) {
			t.Errorf("Generated code should contain variable declaration '%s'", declaration)
		}
	}
}
