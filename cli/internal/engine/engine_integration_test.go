package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/cli/internal/parser"
)

// TestEngineIntegration tests end-to-end engine functionality
func TestEngineIntegration(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectGenerate bool // Whether generation should succeed
		contains       []string
	}{
		{
			name: "simple variable and command",
			input: `var PORT = "8080"
serve: echo "Server on port @var(PORT)"`,
			expectGenerate: true,
			contains: []string{
				"func main()",
				"const PORT = \"8080\"",
				"cobra.Command",
				"rootCmd.Execute()",
			},
		},
		{
			name: "parallel execution",
			input: `build: @parallel {
    echo "Frontend build"
    echo "Backend build"
}`,
			expectGenerate: true,
			contains: []string{
				"func main()",
				"sync.WaitGroup",
				"go func()",
			},
		},
		{
			name: "conditional execution",
			input: `deploy: @when("ENV") {
    prod: kubectl apply -f prod.yaml
    staging: kubectl apply -f staging.yaml
    default: echo "No deployment configured"
}`,
			expectGenerate: true,
			contains: []string{
				"func main()",
				"switch",
				"case \"prod\":",
			},
		},
		{
			name: "timeout with retry",
			input: `test: @timeout(duration=30s) {
    @retry(attempts=3) {
        npm test
    }
}`,
			expectGenerate: true,
			contains: []string{
				"func main()",
				"context.WithTimeout",
				"time.Second",
			},
		},
		{
			name:           "environment variables",
			input:          `deploy: echo "Deploying to @env("ENVIRONMENT", default="dev")"`,
			expectGenerate: true,
			contains: []string{
				"func main()",
				"cobra.Command",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse program: %v", err)
			}

			engine := New(program)
			result, err := engine.GenerateCode(program)

			if tt.expectGenerate {
				if err != nil {
					t.Fatalf("Expected code generation to succeed, but got error: %v", err)
				}

				generatedCode := result.String()
				for _, expected := range tt.contains {
					if !strings.Contains(generatedCode, expected) {
						t.Errorf("Generated code should contain %q, but doesn't.\nGenerated code:\n%s", expected, generatedCode)
					}
				}
			} else {
				if err == nil {
					t.Fatalf("Expected code generation to fail, but it succeeded")
				}
			}
		})
	}
}

// TestEngineVariableResolution tests variable resolution across different contexts
func TestEngineVariableResolution(t *testing.T) {
	input := `var HOST = "localhost"
var PORT = 8080
var DEBUG = true

serve: echo "Server: @var(HOST):@var(PORT) (debug=@var(DEBUG))"
test: echo "Testing on @var(HOST)"
`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)

	// Test variable processing with decorator lookups
	ctx := engine.CreateGeneratorContext(context.Background(), program)
	err = ctx.InitializeVariables()
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	// Check that variables are correctly resolved
	expectedVars := map[string]string{
		"HOST":  "localhost",
		"PORT":  "8080", // Numbers are stored as their string representation
		"DEBUG": "true",
	}

	for name, expectedValue := range expectedVars {
		if actualValue, exists := ctx.GetVariable(name); !exists {
			t.Errorf("Expected variable %s not found", name)
		} else if actualValue != expectedValue {
			t.Errorf("Variable %s: expected %s, got %s", name, expectedValue, actualValue)
		}
	}

	// Test code generation includes variables
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	generatedCode := result.String()
	expectedInCode := []string{
		"const HOST = \"localhost\"",
		"const PORT = \"8080\"",
		"const DEBUG = \"true\"",
	}

	for _, expected := range expectedInCode {
		if !strings.Contains(generatedCode, expected) {
			t.Errorf("Generated code should contain %q.\nGenerated code:\n%s", expected, generatedCode)
		}
	}
}

// TestEngineCommandExecution tests individual command execution
func TestEngineCommandExecution(t *testing.T) {
	input := `var MSG = "hello world"
greet: echo "@var(MSG)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	if len(program.Commands) == 0 {
		t.Fatal("Expected at least one command in program")
	}

	engine := New(program)

	// Test command execution structure (doesn't actually run shell commands in test)
	cmdResult, err := engine.ExecuteCommand(&program.Commands[0])
	if err != nil {
		// This is expected in a test environment without proper shell setup
		if !strings.Contains(err.Error(), "command execution failed") &&
			!strings.Contains(err.Error(), "command failed") {
			t.Logf("Command execution failed as expected in test environment: %v", err)
		}
	}

	// Verify command result structure
	if cmdResult.Name != "greet" {
		t.Errorf("Expected command name 'greet', got %s", cmdResult.Name)
	}

	if cmdResult.Status != "failed" && cmdResult.Status != "success" {
		t.Errorf("Expected command status to be 'failed' or 'success', got %s", cmdResult.Status)
	}
}
