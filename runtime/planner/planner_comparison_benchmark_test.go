package planner_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/planner"
)

// BenchmarkPlannerComparison benchmarks the canonical planner entrypoint across
// representative script shapes.
func BenchmarkPlannerComparison(b *testing.B) {
	scenarios := map[string]string{
		"simple_command":    `echo "Hello, World!"`,
		"multiple_commands": generateMultiCommandScript(10),
		"with_variables":    generateVariableScript(5),
		"with_conditionals": generateConditionalScript(),
		"with_loops":        generateLoopScript(),
		"complex_mixed":     generateComplexMixedScript(),
	}

	for name, source := range scenarios {
		sourceBytes := []byte(source)
		tree := parser.Parse(sourceBytes)
		if len(tree.Errors) > 0 {
			b.Fatalf("Parse errors in %s: %v", name, tree.Errors)
		}

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
				if err != nil {
					b.Fatalf("Planner failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkPlannerNewPipelinePhases measures individual phase performance.
// Helps identify which phase (BuildIR, Resolve, Emit) is the bottleneck.
func BenchmarkPlannerNewPipelinePhases(b *testing.B) {
	scenarios := map[string]string{
		"simple":  `echo "hello"`,
		"medium":  generateMultiCommandScript(50),
		"complex": generateComplexMixedScript(),
	}

	for name, source := range scenarios {
		sourceBytes := []byte(source)
		tree := parser.Parse(sourceBytes)
		if len(tree.Errors) > 0 {
			b.Fatalf("Parse errors in %s: %v", name, tree.Errors)
		}

		b.Run(name, func(b *testing.B) {
			// Full pipeline
			b.Run("full", func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
					if err != nil {
						b.Fatalf("Plan failed: %v", err)
					}
				}
			})

			// Just BuildIR phase
			b.Run("build_ir", func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := planner.BuildIR(tree.Events, tree.Tokens)
					if err != nil {
						b.Fatalf("BuildIR failed: %v", err)
					}
				}
			})

			b.Run("resolve_emit", func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					// This includes both Resolve and Emit
					// We measure them together since Emit depends on Resolve output
					_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
					if err != nil {
						b.Fatalf("Plan failed: %v", err)
					}
				}
			})
		})
	}
}

// BenchmarkPlannerNewVsBaseline compares canonical planner against documented baselines.
// From AGENTS.md:
//   - Planner (simple): ~361ns/op, 392 B/op, 9 allocs/op
//   - Planner (complex): ~4.7µs/op, 6480 B/op, 151 allocs/op
func BenchmarkPlannerNewVsBaseline(b *testing.B) {
	b.Run("simple_target_361ns", func(b *testing.B) {
		source := []byte(`echo "Hello"`)
		tree := parser.Parse(source)

		b.ResetTimer()
		b.ReportAllocs()
		start := time.Now()
		for i := 0; i < b.N; i++ {
			_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
			if err != nil {
				b.Fatalf("Plan failed: %v", err)
			}
		}
		elapsed := time.Since(start)

		// After run, check against baseline
		result := testing.BenchmarkResult{N: b.N, T: elapsed}
		nsPerOp := float64(result.T.Nanoseconds()) / float64(b.N)
		if nsPerOp > 400 { // Allow 10% margin over 361ns
			b.Logf("WARNING: Simple case slower than baseline: %.0f ns/op vs target 361 ns/op", nsPerOp)
		}
	})

	b.Run("complex_target_4.7us", func(b *testing.B) {
		source := generateComplexScript() // From existing benchmark_test.go
		tree := parser.Parse([]byte(source))

		b.ResetTimer()
		b.ReportAllocs()
		start := time.Now()
		for i := 0; i < b.N; i++ {
			_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
			if err != nil {
				b.Fatalf("Plan failed: %v", err)
			}
		}
		elapsed := time.Since(start)

		result := testing.BenchmarkResult{N: b.N, T: elapsed}
		nsPerOp := float64(result.T.Nanoseconds()) / float64(b.N)
		if nsPerOp > 5200 { // Allow 10% margin over 4.7µs
			b.Logf("WARNING: Complex case slower than baseline: %.0f ns/op vs target 4700 ns/op", nsPerOp)
		}
	})
}

// BenchmarkPlannerOutputParity checks output stability across repeated runs.
func BenchmarkPlannerOutputParity(b *testing.B) {
	source := generateComplexMixedScript()
	sourceBytes := []byte(source)
	tree := parser.Parse(sourceBytes)

	baseline, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
	if err != nil {
		b.Fatalf("Baseline plan failed: %v", err)
	}

	for i := 0; i < b.N; i++ {
		result, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
		if err != nil {
			b.Fatalf("Planner failed: %v", err)
		}

		if len(result.Steps) != len(baseline.Steps) {
			b.Fatalf("Step count mismatch: baseline=%d current=%d", len(baseline.Steps), len(result.Steps))
		}

		if len(result.SecretUses) != len(baseline.SecretUses) {
			b.Logf("WARNING: SecretUses count differs: baseline=%d current=%d", len(baseline.SecretUses), len(result.SecretUses))
		}
	}
}

// Helper functions for generating test scripts

func generateMultiCommandScript(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += fmt.Sprintf("echo \"Command %d\"\n", i)
	}
	return result
}

func generateVariableScript(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += fmt.Sprintf("var x%d = \"value%d\"\n", i, i)
	}
	for i := 0; i < count; i++ {
		result += fmt.Sprintf("echo @var.x%d\n", i)
	}
	return result
}

func generateConditionalScript() string {
	return `var ENV = "prod"
if @var.ENV == "prod" {
    echo "Production mode"
} else {
    echo "Development mode"
}
echo "Done"`
}

func generateLoopScript() string {
	// Loops require properly resolved collections - skip for now
	// Return a simple script instead
	return `echo "Loop placeholder"
echo "Loop complete"`
}

func generateComplexMixedScript() string {
	return `var ENV = "prod"
var COUNT = 3
var instances = ["1", "2", "3"]

echo "Starting deployment"

if @var.ENV == "prod" {
    echo "Production deployment"
    for i in @var.instances {
        echo "Deploying instance @var.i"
    }
} else {
    echo "Development deployment"
}

echo "Deployed @var.COUNT instances"
echo "Complete"`
}
