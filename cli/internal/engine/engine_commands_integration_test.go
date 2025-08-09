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
)

// TestProjectCommandsIntegration tests the project's own commands.cli file end-to-end
func TestProjectCommandsIntegration(t *testing.T) {
	// Find the project root commands.cli file
	projectRoot := findProjectRoot(t)
	commandsFile := filepath.Join(projectRoot, "commands.cli")

	// Verify commands.cli exists
	if _, err := os.Stat(commandsFile); os.IsNotExist(err) {
		t.Fatalf("Project commands.cli not found at %s", commandsFile)
	}

	// Read and parse the commands.cli file
	file, err := os.Open(commandsFile)
	if err != nil {
		t.Fatalf("Failed to open commands.cli: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Warning: failed to close commands.cli file: %v", err)
		}
	}()

	// Parse the commands
	program, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse project commands.cli: %v", err)
	}

	// Test 1: Validate that the program structure is valid
	t.Run("ParseValidation", func(t *testing.T) {
		if program == nil {
			t.Fatal("Parsed program is nil")
		}

		if len(program.Commands) == 0 {
			t.Fatal("No commands found in project commands.cli")
		}

		// Verify key commands exist
		expectedCommands := []string{"build", "test", "help", "clean"}
		foundCommands := make(map[string]bool)

		for _, cmd := range program.Commands {
			foundCommands[cmd.Name] = true
		}

		for _, expected := range expectedCommands {
			if !foundCommands[expected] {
				t.Errorf("Expected command '%s' not found in commands.cli", expected)
			}
		}
	})

	// Test 2: Generate Go code successfully
	t.Run("GenerateGoCode", func(t *testing.T) {
		engine := New(program)

		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("Failed to generate Go code from project commands.cli: %v", err)
		}

		code := result.Code.String()
		if code == "" {
			t.Fatal("Generated code is empty")
		}

		// Basic validation of generated code
		if !strings.Contains(code, "package main") {
			t.Error("Generated code missing 'package main'")
		}

		if !strings.Contains(code, "func main()") {
			t.Error("Generated code missing 'func main()'")
		}

		// Validate that all commands have corresponding functions
		for _, cmd := range program.Commands {
			// Commands should have corresponding functions (basic check)
			if !strings.Contains(code, cmd.Name) {
				t.Errorf("Generated code missing reference to command '%s'", cmd.Name)
			}
		}
	})

	// Test 3: Build and execute the generated CLI
	t.Run("BuildAndExecute", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping build test in short mode")
		}

		// Create temporary directory for build
		tempDir, err := os.MkdirTemp("", "devcmd-integration-test-")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Warning: failed to remove temp directory: %v", err)
			}
		}()

		// Generate the Go code with proper module name
		engine := New(program)

		result, err := engine.GenerateCodeWithModule(program, "github.com/aledsdavies/devcmd/test-cli")
		if err != nil {
			t.Fatalf("Failed to generate Go code: %v", err)
		}

		code := result.Code.String()
		goMod := result.GoMod.String()

		// Write generated code to temp file
		mainFile := filepath.Join(tempDir, "main.go")
		if err := os.WriteFile(mainFile, []byte(code), 0o644); err != nil {
			t.Fatalf("Failed to write generated code: %v", err)
		}

		// Write generated go.mod file
		goModFile := filepath.Join(tempDir, "go.mod")
		if err := os.WriteFile(goModFile, []byte(goMod), 0o644); err != nil {
			t.Fatalf("Failed to write generated go.mod: %v", err)
		}

		t.Logf("Generated go.mod content:\n%s", goMod)

		// Download dependencies and create go.sum
		if err := runCommandIntegration(tempDir, "go", "mod", "tidy"); err != nil {
			t.Fatalf("Failed to tidy go module: %v", err)
		}

		// Build the CLI
		binaryPath := filepath.Join(tempDir, "test-cli")
		if err := runCommandIntegration(tempDir, "go", "build", "-o", binaryPath, mainFile); err != nil {
			t.Fatalf("Failed to build generated CLI: %v", err)
		}

		// Test that the binary exists and is executable
		if _, err := os.Stat(binaryPath); err != nil {
			t.Fatalf("Built binary not found: %v", err)
		}

		// Test running --help
		output, err := runCommandWithOutputIntegration(tempDir, binaryPath, "--help")
		if err != nil {
			t.Fatalf("Failed to run generated CLI --help: %v", err)
		}

		// Validate help output contains expected elements
		if !strings.Contains(output, "Available Commands:") {
			t.Error("Help output missing 'Available Commands:' section")
		}

		// Verify key commands appear in help
		helpLines := strings.Split(output, "\n")
		foundInHelp := make(map[string]bool)
		for _, line := range helpLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "build") {
				foundInHelp["build"] = true
			}
			if strings.HasPrefix(line, "test") {
				foundInHelp["test"] = true
			}
			if strings.HasPrefix(line, "help") {
				foundInHelp["help"] = true
			}
		}

		expectedInHelp := []string{"build", "test", "help"}
		for _, expected := range expectedInHelp {
			if !foundInHelp[expected] {
				t.Errorf("Command '%s' not found in help output", expected)
			}
		}
	})

	// Test 4: Command plan generation
	t.Run("CommandPlanGeneration", func(t *testing.T) {
		engine := New(program)

		// Test plan generation for a simple command (help should be safe)
		helpCmd := findCommand(program, "help")
		if helpCmd == nil {
			t.Skip("Help command not found, skipping plan test")
		}

		plan, err := engine.ExecuteCommandPlan(helpCmd)
		if err != nil {
			t.Fatalf("Failed to generate plan for help command: %v", err)
		}

		if plan == nil {
			t.Error("Plan generation returned nil plan")
		}
	})
}

// findProjectRoot walks up the directory tree to find the project root
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current test file location
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Walk up until we find commands.cli or reach root
	for {
		commandsPath := filepath.Join(dir, "commands.cli")
		if _, err := os.Stat(commandsPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	t.Fatalf("Could not find project root with commands.cli")
	return ""
}

// findCommand finds a command by name in the program
func findCommand(program *ast.Program, name string) *ast.CommandDecl {
	for _, cmd := range program.Commands {
		if cmd.Name == name {
			return &cmd
		}
	}
	return nil
}

// runCommandIntegration executes a command in the specified directory
func runCommandIntegration(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCommandWithOutputIntegration executes a command and returns its output
func runCommandWithOutputIntegration(dir string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
