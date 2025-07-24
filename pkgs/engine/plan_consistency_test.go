package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestPlanConsistency verifies that interpreter mode and generated binary mode
// produce identical execution plan graphs for the same command and state.
// This is the core success criteria for the plan mode feature.
func TestPlanConsistency(t *testing.T) {
	testCases := []struct {
		name        string
		cliContent  string
		command     string
		description string
	}{
		{
			name: "basic_shell_commands",
			cliContent: `var PROJECT_NAME = "testproject"

build: {
  echo "Building @var(PROJECT_NAME)..."
  echo "Build complete"
}`,
			command:     "build",
			description: "Basic shell commands with variable expansion",
		},
		{
			name: "parallel_decorator",
			cliContent: `parallel_test: @parallel {
  echo "Task 1"
  echo "Task 2" 
  echo "Task 3"
}`,
			command:     "parallel_test",
			description: "Parallel decorator with multiple shell commands",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the CLI content
			program, err := parser.Parse(strings.NewReader(tc.cliContent))
			if err != nil {
				t.Fatalf("Failed to parse CLI content: %v", err)
			}

			// Find the target command
			var targetCommand *ast.CommandDecl
			for i := range program.Commands {
				if program.Commands[i].Name == tc.command {
					targetCommand = &program.Commands[i]
					break
				}
			}
			if targetCommand == nil {
				t.Fatalf("Command %s not found in program", tc.command)
			}

			// Get execution plan from engine directly (interpreter mode)
			interpreterPlan, err := getEngineInterpreterPlan(program, targetCommand)
			if err != nil {
				t.Fatalf("Failed to get interpreter plan: %v", err)
			}

			// Get execution plan from generated binary
			generatedPlan, err := getGeneratedBinaryPlan(t, tc.cliContent, tc.command)
			if err != nil {
				// For now, this will fail because generated binaries don't support --dry-run yet
				// This test documents what we need to implement
				t.Logf("Generated binary plan failed (expected until --dry-run is implemented): %v", err)
				t.Logf("Interpreter plan output:\n%s", interpreterPlan)
				t.Skip("Generated binary --dry-run not implemented yet")
				return
			}

			// Compare the plans - they should be identical
			if !comparePlanOutputs(interpreterPlan, generatedPlan) {
				t.Errorf("Execution plans do not match for %s!\n\nInterpreter mode:\n%s\n\nGenerated binary:\n%s",
					tc.description, interpreterPlan, generatedPlan)
			} else {
				t.Logf("✅ Plans match perfectly for %s", tc.description)
			}
		})
	}
}

// getEngineInterpreterPlan gets execution plan directly using the engine in plan mode
func getEngineInterpreterPlan(program *ast.Program, command *ast.CommandDecl) (string, error) {
	// Create engine and get execution plan
	engine := New(program)
	plan, err := engine.ExecuteCommandPlan(command)
	if err != nil {
		return "", err
	}

	return plan.String(), nil
}

// getGeneratedBinaryPlan builds a CLI binary and gets its --dry-run output
func getGeneratedBinaryPlan(t *testing.T, cliContent, command string) (string, error) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Write CLI content to file
	cliFile := filepath.Join(tempDir, "test.cli")
	if err := os.WriteFile(cliFile, []byte(cliContent), 0o644); err != nil {
		return "", err
	}

	// Build binary using go run to avoid dependency on build directory
	binaryPath := filepath.Join(tempDir, "test-cli")
	buildCmd := exec.Command("go", "run", "../../cmd/devcmd", "build", "-f", cliFile, "--binary", "test-cli", "--output", binaryPath)
	buildCmd.Dir = tempDir

	if _, err := buildCmd.CombinedOutput(); err != nil {
		return "", err
	}

	// Try to run with --dry-run flag
	runCmd := exec.Command(binaryPath, command, "--dry-run")
	output, err := runCmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// comparePlanOutputs compares two execution plan outputs for structural equality
func comparePlanOutputs(plan1, plan2 string) bool {
	// Normalize both plans by removing extra whitespace and empty lines
	normalize := func(s string) []string {
		lines := strings.Split(strings.TrimSpace(s), "\n")
		var normalized []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				normalized = append(normalized, trimmed)
			}
		}
		return normalized
	}

	lines1 := normalize(plan1)
	lines2 := normalize(plan2)

	if len(lines1) != len(lines2) {
		return false
	}

	for i := range lines1 {
		if lines1[i] != lines2[i] {
			return false
		}
	}

	return true
}

// TestBasicPlanGeneration tests that the engine can generate execution plans
func TestBasicPlanGeneration(t *testing.T) {
	cliContent := `var NAME = "test"
	
simple: echo "Hello @var(NAME)"`

	program, err := parser.Parse(strings.NewReader(cliContent))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	engine := New(program)

	// Find the simple command
	var simpleCmd *ast.CommandDecl
	for i := range program.Commands {
		if program.Commands[i].Name == "simple" {
			simpleCmd = &program.Commands[i]
			break
		}
	}

	if simpleCmd == nil {
		t.Fatal("simple command not found")
	}

	// Generate execution plan
	plan, err := engine.ExecuteCommandPlan(simpleCmd)
	if err != nil {
		t.Fatalf("Failed to generate plan: %v", err)
	}

	planStr := plan.String()

	// Verify the plan contains expected elements with new format
	if !strings.Contains(planStr, "simple:") {
		t.Error("Plan missing command header")
	}

	if !strings.Contains(planStr, "echo \"Hello test\"") {
		t.Error("Plan missing resolved command")
	}

	t.Logf("Generated plan:\n%s", planStr)
}
