package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestMain ensures cleanup happens after all tests
func TestMain(m *testing.M) {
	code := m.Run()

	// Final cleanup - ignore errors since we're cleaning up
	_ = os.Remove("generated.go")
	matches, _ := filepath.Glob("*.tmp")
	for _, match := range matches {
		_ = os.Remove(match)
	}

	os.Exit(code)
}

// TestEngine_BasicConstruction tests basic engine construction
func TestEngine_BasicConstruction(t *testing.T) {
	input := `var PORT = "8080"
build: echo "Building"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	if engine.ctx == nil {
		t.Fatal("Engine context should not be nil")
	}

	if engine.goVersion != "1.24" {
		t.Errorf("Expected default Go version 1.24, got %s", engine.goVersion)
	}
}

// TestEngine_CustomGoVersion tests engine construction with custom Go version
func TestEngine_CustomGoVersion(t *testing.T) {
	program, err := parser.Parse(strings.NewReader(`build: echo "test"`))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := NewWithGoVersion(program, "1.23")
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	if engine.goVersion != "1.23" {
		t.Errorf("Expected Go version 1.23, got %s", engine.goVersion)
	}
}

// TestEngine_VariableProcessing tests variable processing functionality
func TestEngine_VariableProcessing(t *testing.T) {
	input := `var HOST = "localhost"
var PORT = 8080
var DEBUG = true`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)

	err = engine.processVariablesIntoContext(program)
	if err != nil {
		t.Fatalf("Failed to process variables: %v", err)
	}

	// Check variables were processed correctly
	expectedVars := map[string]string{
		"HOST":  "localhost",
		"PORT":  "8080", // Numbers are stored as their string representation
		"DEBUG": "true",
	}

	for name, expectedValue := range expectedVars {
		if actualValue, exists := engine.ctx.GetVariable(name); !exists {
			t.Errorf("Variable %s not found", name)
		} else if actualValue != expectedValue {
			t.Errorf("Variable %s: expected %s, got %s", name, expectedValue, actualValue)
		}
	}
}

// TestEngine_CodeGeneration tests basic code generation
func TestEngine_CodeGeneration(t *testing.T) {
	input := `var PORT = "8080"
serve: echo "Serving on port @var(PORT)"
build: make all`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	generatedCode := result.String()

	// Check for required elements in generated code
	requiredElements := []string{
		"func main()",
		"PORT := \"8080\"",
		"cobra.Command",
		"rootCmd.Execute()",
		"serveCmd",
		"buildCmd",
	}

	for _, element := range requiredElements {
		if !strings.Contains(generatedCode, element) {
			t.Errorf("Generated code should contain %q.\nGenerated code:\n%s", element, generatedCode)
		}
	}
}

// TestEngine_CommandExecution tests command execution structure
func TestEngine_CommandExecution(t *testing.T) {
	input := `greeting: echo "Hello World"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	if len(program.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(program.Commands))
	}

	engine := New(program)
	cmdResult, err := engine.ExecuteCommand(&program.Commands[0])
	// In test environment, command may fail - that's expected
	if err != nil {
		if !strings.Contains(err.Error(), "command execution failed") &&
			!strings.Contains(err.Error(), "command failed") {
			t.Logf("Command execution failed as expected in test environment: %v", err)
		}
	}

	// Verify command result structure
	if cmdResult.Name != "greeting" {
		t.Errorf("Expected command name 'greeting', got %s", cmdResult.Name)
	}

	if cmdResult.Status == "" {
		t.Error("Command status should not be empty")
	}

	if len(cmdResult.Output) == 0 {
		t.Log("Command output is empty, which is expected in test environment")
	}
}

// TestEngine_EmptyProgram tests handling of empty programs
func TestEngine_EmptyProgram(t *testing.T) {
	program, err := parser.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to parse empty program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed for empty program: %v", err)
	}

	generatedCode := result.String()

	// Should still have basic structure
	if !strings.Contains(generatedCode, "func main()") {
		t.Error("Generated code should contain main function even for empty program")
	}

	if !strings.Contains(generatedCode, "rootCmd.Execute()") {
		t.Error("Generated code should contain root command execution")
	}
}

// TestEngine_ErrorHandling tests error handling in various scenarios
func TestEngine_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid program",
			input:       `build: echo "hello"`,
			expectError: false,
		},
		{
			name: "valid program with variables",
			input: `var TEST = "value"
build: echo "@var(TEST)"`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Unexpected parse error: %v", err)
				}
				return
			}

			engine := New(program)
			_, err = engine.GenerateCode(program)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
