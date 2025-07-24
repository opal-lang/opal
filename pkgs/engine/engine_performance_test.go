package engine

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// BenchmarkEngine_Interpreter benchmarks interpreter mode performance
func BenchmarkEngine_Interpreter(b *testing.B) {
	input := `var PORT = "8080"
var HOST = "localhost"
build: echo "Building on @var(HOST):@var(PORT)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		b.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test variable processing performance
		_ = engine.processVariablesIntoContext(program)
	}
}

// BenchmarkEngine_Generator benchmarks generator mode performance
func BenchmarkEngine_Generator(b *testing.B) {
	input := `var PORT = "8080"
var HOST = "localhost"
build: echo "Building on @var(HOST):@var(PORT)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		b.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.GenerateCode(program)
	}
}

// BenchmarkEngine_ComplexProgram benchmarks performance with complex programs
func BenchmarkEngine_ComplexProgram(b *testing.B) {
	// Generate a complex program with many variables and commands
	var inputBuilder strings.Builder

	// Add many variables
	for i := 0; i < 100; i++ {
		inputBuilder.WriteString(fmt.Sprintf("var VAR%d = \"value%d\"\n", i, i))
	}

	// Add commands with variable references
	for i := 0; i < 50; i++ {
		inputBuilder.WriteString(fmt.Sprintf("cmd%d: echo \"Command %d uses @var(VAR%d)\"\n", i, i, i%100))
	}

	program, err := parser.Parse(strings.NewReader(inputBuilder.String()))
	if err != nil {
		b.Fatalf("Failed to parse complex program: %v", err)
	}

	engine := New(program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.GenerateCode(program)
	}
}

// TestEngine_Performance tests performance characteristics
func TestEngine_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	input := `var PORT = "8080"
build: echo "Building on port @var(PORT)"
test: echo "Testing"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)

	// Measure generation time
	start := time.Now()
	_, err = engine.GenerateCode(program)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Expect reasonable performance (adjust threshold as needed)
	if duration > 100*time.Millisecond {
		t.Logf("Code generation took %v, which seems slow for a simple program", duration)
	}

	t.Logf("Code generation completed in %v", duration)
}

// TestEngine_ScalabilityByVariables tests how performance scales with variable count
func TestEngine_ScalabilityByVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	variableCounts := []int{10, 50, 100, 200}

	for _, count := range variableCounts {
		t.Run(fmt.Sprintf("variables_%d", count), func(t *testing.T) {
			var inputBuilder strings.Builder

			// Add variables
			for i := 0; i < count; i++ {
				inputBuilder.WriteString(fmt.Sprintf("var VAR%d = \"value%d\"\n", i, i))
			}

			// Add a command that uses some variables
			inputBuilder.WriteString("build: echo \"Using @var(VAR0) and @var(VAR1)\"\n")

			program, err := parser.Parse(strings.NewReader(inputBuilder.String()))
			if err != nil {
				t.Fatalf("Failed to parse program with %d variables: %v", count, err)
			}

			engine := New(program)

			start := time.Now()
			_, err = engine.GenerateCode(program)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Code generation failed with %d variables: %v", count, err)
			}

			t.Logf("Generated code for %d variables in %v", count, duration)
		})
	}
}
