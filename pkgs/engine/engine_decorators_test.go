package engine

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestDecorators tests that decorators work properly with the new execution system
func TestDecorators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // Expected content in generated code
	}{
		{
			name: "parallel decorator",
			input: `build: @parallel {
    echo "Building frontend"
    echo "Building backend"  
}`,
			contains: []string{
				"// Block decorator: @parallel",
				"func main()",
				"cobra.Command",
			},
		},
		{
			name: "timeout decorator",
			input: `test: @timeout(duration=30s) {
    echo "Running tests"
}`,
			contains: []string{
				"// Block decorator: @timeout",
				"func main()",
			},
		},
		{
			name: "when decorator",
			input: `deploy: @when("ENV") {
    prod: kubectl apply -f prod.yaml
    dev: kubectl apply -f dev.yaml
    default: echo "Unknown environment"
}`,
			contains: []string{
				"// Pattern decorator: @when",
				"func main()",
			},
		},
		{
			name: "variable substitution",
			input: `var PORT = "8080"
serve: echo "Server running on @var(PORT)"`,
			contains: []string{
				"PORT := \"8080\"",
				"func main()",
			},
		},
		{
			name:  "environment variable",
			input: `deploy: echo "Deploying to @env("ENVIRONMENT")"`,
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
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			generatedCode := result.String()
			for _, expected := range tt.contains {
				if !strings.Contains(generatedCode, expected) {
					t.Errorf("Generated code should contain %q, but doesn't.\nGenerated code:\n%s", expected, generatedCode)
				}
			}
		})
	}
}

// TestDecoratorImports tests that decorator imports are collected properly
func TestDecoratorImports(t *testing.T) {
	input := `build: @timeout(duration=10s) {
    echo "Building with timeout"
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Check that timeout decorator imports are added
	if !result.HasStandardImport("time") {
		t.Error("Expected 'time' import for timeout decorator")
	}
	if !result.HasStandardImport("context") {
		t.Error("Expected 'context' import for timeout decorator")
	}
}
