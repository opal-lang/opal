package planner_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/planner"
	"github.com/opal-lang/opal/runtime/vault"
)

type plannerBenchmarkScenario struct {
	name   string
	source string
}

// BenchmarkPlannerScenarios benchmarks canonical planner entrypoint across
// representative script shapes with deterministic sub-benchmark ordering.
func BenchmarkPlannerScenarios(b *testing.B) {
	scenarios := []plannerBenchmarkScenario{
		{name: "simple_command", source: `echo "Hello, World!"`},
		{name: "multiple_commands", source: generateMultiCommandScript(10)},
		{name: "with_variables", source: generateVariableScript(5)},
		{name: "with_conditionals", source: generateConditionalScript()},
		{name: "with_loops", source: generateLoopScript()},
		{name: "complex_mixed", source: generateComplexMixedScript()},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		tree := parseBenchmarkSource(b, scenario.source)

		b.Run(scenario.name, func(b *testing.B) {
			runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{})
		})
	}
}

// BenchmarkPlannerPipelinePhases measures full pipeline, BuildIR-only,
// and Resolve-only costs.
func BenchmarkPlannerPipelinePhases(b *testing.B) {
	scenarios := []plannerBenchmarkScenario{
		{name: "simple", source: `echo "hello"`},
		{name: "medium", source: generateMultiCommandScript(50)},
		{name: "complex", source: generateComplexMixedScript()},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		tree := parseBenchmarkSource(b, scenario.source)

		b.Run(scenario.name, func(b *testing.B) {
			b.Run("full", func(b *testing.B) {
				runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{})
			})

			b.Run("build_ir", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := planner.BuildIR(tree.Events, tree.Tokens)
					if err != nil {
						b.Fatalf("BuildIR failed: %v", err)
					}
				}
			})

			b.Run("resolve", func(b *testing.B) {
				cfg := planner.ResolveConfig{Context: context.Background()}

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					graph, err := planner.BuildIR(tree.Events, tree.Tokens)
					if err != nil {
						b.Fatalf("BuildIR failed: %v", err)
					}
					b.StartTimer()

					_, err = planner.Resolve(graph, benchmarkVault(), decorator.NewLocalSession(), cfg)
					if err != nil {
						b.Fatalf("Resolve failed: %v", err)
					}
				}
			})
		})
	}
}

// BenchmarkPlannerReferenceCases tracks canonical simple/complex planner paths.
func BenchmarkPlannerReferenceCases(b *testing.B) {
	b.Run("simple", func(b *testing.B) {
		tree := parseBenchmarkSource(b, `echo "Hello"`)
		runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{})
	})

	b.Run("complex", func(b *testing.B) {
		tree := parseBenchmarkSource(b, generateComplexScript())
		runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{})
	})
}

func benchmarkVault() *vault.Vault {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}
	return vault.NewWithPlanKey(planKey)
}

// Helper functions for generating test scripts

func generateMultiCommandScript(count int) string {
	var b strings.Builder
	for i := 0; i < count; i++ {
		_, _ = fmt.Fprintf(&b, "echo \"Command %d\"\n", i)
	}
	return b.String()
}

func generateVariableScript(count int) string {
	var b strings.Builder
	for i := 0; i < count; i++ {
		_, _ = fmt.Fprintf(&b, "var x%d = \"value%d\"\n", i, i)
	}
	for i := 0; i < count; i++ {
		_, _ = fmt.Fprintf(&b, "echo @var.x%d\n", i)
	}
	return b.String()
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
	return `for i in ["1", "2", "3"] {
    echo "Loop item @var.i"
}
echo "Loop complete"`
}

func generateComplexMixedScript() string {
	return `var ENV = "prod"
var COUNT = 3

echo "Starting deployment"

if @var.ENV == "prod" {
    echo "Production deployment"
    for i in ["1", "2", "3"] {
        echo "Deploying instance @var.i"
    }
} else {
    echo "Development deployment"
}

echo "Deployed @var.COUNT instances"
echo "Complete"`
}
