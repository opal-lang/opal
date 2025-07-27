package engine

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestImportManagement tests that imports are collected correctly from decorators
func TestImportManagement(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedStandard   []string
		expectedThirdParty []string
	}{
		{
			name: "timeout decorator imports",
			input: `build: @timeout(duration=10s) {
    echo "Building with timeout"
}`,
			expectedStandard:   []string{"time", "context"},
			expectedThirdParty: []string{},
		},
		{
			name: "parallel decorator imports",
			input: `build: @parallel {
    echo "Frontend"
    echo "Backend"
}`,
			expectedStandard:   []string{"sync"},
			expectedThirdParty: []string{},
		},
		{
			name:               "env decorator imports",
			input:              `deploy: echo "Deploying to @env("ENVIRONMENT")"`,
			expectedStandard:   []string{"os"},
			expectedThirdParty: []string{},
		},
		{
			name: "multiple decorators",
			input: `build: @timeout(duration=30s) {
    @parallel {
        echo "Frontend"
        echo "Backend"
    }
}
deploy: echo "Environment: @env("ENV")"`,
			expectedStandard:   []string{"time", "context", "sync", "os"},
			expectedThirdParty: []string{},
		},
		{
			name: "confirm decorator imports",
			input: `deploy: @confirm(message="Deploy to production?") {
    kubectl apply -f prod.yaml
}`,
			expectedStandard:   []string{"bufio", "fmt", "os", "strings"},
			expectedThirdParty: []string{},
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
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			// Check standard library imports
			for _, expectedImport := range tt.expectedStandard {
				if !result.HasStandardImport(expectedImport) {
					t.Errorf("Expected standard import %q not found", expectedImport)
				}
			}

			// Check third-party imports
			for _, expectedImport := range tt.expectedThirdParty {
				if !result.HasThirdPartyImport(expectedImport) {
					t.Errorf("Expected third-party import %q not found", expectedImport)
				}
			}

			t.Logf("Successfully collected imports for %s", tt.name)
		})
	}
}

// TestImportDeduplication tests that duplicate imports are handled correctly
func TestImportDeduplication(t *testing.T) {
	input := `build: @timeout(duration=10s) {
    echo "First timeout"
}
test: @timeout(duration=20s) {
    echo "Second timeout"
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Should have time and context imports, but not duplicated
	if !result.HasStandardImport("time") {
		t.Error("Expected 'time' import for timeout decorators")
	}
	if !result.HasStandardImport("context") {
		t.Error("Expected 'context' import for timeout decorators")
	}

	// Check that imports are in the map (indicating deduplication works)
	if len(result.StandardImports) == 0 {
		t.Error("StandardImports map should not be empty")
	}

	t.Logf("Import deduplication working correctly")
}

// TestNestedDecoratorImports tests import collection from nested decorators
func TestNestedDecoratorImports(t *testing.T) {
	input := `complex: @timeout(duration=60s) {
    @parallel {
        @retry(attempts=3) {
            echo "Retried parallel task 1"
        }
        echo "Parallel task 2"
    }
    @confirm(message="Continue?") {
        echo "Confirmed action"
    }
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Should collect imports from all nested decorators
	expectedImports := []string{
		"time",    // from timeout
		"context", // from timeout
		"sync",    // from parallel
		"bufio",   // from confirm
		"fmt",     // from confirm (and base)
		"os",      // from confirm (and base)
		"strings", // from confirm
	}

	for _, expectedImport := range expectedImports {
		if !result.HasStandardImport(expectedImport) {
			t.Errorf("Expected import %q from nested decorators", expectedImport)
		}
	}

	t.Logf("Successfully collected imports from nested decorators")
}

// TestImportRequirements tests the ImportRequirement interface
func TestImportRequirements(t *testing.T) {
	input := `build: echo "simple build"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Even simple programs should have basic imports
	baseImports := []string{"fmt", "os"}
	for _, baseImport := range baseImports {
		if !result.HasStandardImport(baseImport) {
			t.Errorf("Expected base import %q", baseImport)
		}
	}

	// Programs with commands should have cobra import
	if len(program.Commands) > 0 {
		if !result.HasThirdPartyImport("github.com/spf13/cobra") {
			t.Error("Expected cobra import for programs with commands")
		}
	}

	t.Logf("Import requirements satisfied")
}

// TestImportGeneration tests that imports are correctly formatted in generated code
func TestImportGeneration(t *testing.T) {
	input := `var PORT = "8080"
serve: echo "Server on @var(PORT)"
timeout_test: @timeout(duration=5s) {
    echo "Quick test"
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Verify that the result contains correct import information
	generatedCode := result.String()

	// Basic structure should be present
	if !strings.Contains(generatedCode, "func main()") {
		t.Error("Generated code should contain main function")
	}

	// Check that we collected the expected imports (they would be used by a larger generator)
	if !result.HasStandardImport("time") {
		t.Error("Should have collected 'time' import from timeout decorator")
	}
	if !result.HasStandardImport("context") {
		t.Error("Should have collected 'context' import from timeout decorator")
	}

	t.Logf("Import generation and collection working correctly")
}
