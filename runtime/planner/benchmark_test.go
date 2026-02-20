package planner_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opal-lang/opal/runtime/lexer"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/planner"
)

type plannerScenario struct {
	name   string
	source string
}

// BenchmarkPlannerCore measures plan generation performance across complexity levels.
// Target: <1ms for simple scripts, <10ms for complex scripts with 100+ steps.
func BenchmarkPlannerCore(b *testing.B) {
	scenarios := []plannerScenario{
		{name: "simple_command", source: `echo "Hello, World!"`},
		{name: "multiple_commands", source: `echo "First"
echo "Second"
	echo "Third"`},
		{name: "function_call", source: `fun hello = echo "Hello"
hello()`},
		{name: "complex_script", source: generateComplexScript()},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		b.Run(scenario.name, func(b *testing.B) {
			tree := parseBenchmarkSource(b, scenario.source)
			runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{Target: ""})
		})
	}
}

// BenchmarkPlannerTelemetryModes measures observability overhead.
// Verifies TelemetryOff has zero overhead, TelemetryTiming has minimal overhead.
func BenchmarkPlannerTelemetryModes(b *testing.B) {
	source := []byte(generateComplexScript())
	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		b.Fatalf("Parse errors: %v", tree.Errors)
	}

	modes := []struct {
		name  string
		level planner.TelemetryLevel
	}{
		{name: "off", level: planner.TelemetryOff},
		{name: "basic", level: planner.TelemetryBasic},
		{name: "timing", level: planner.TelemetryTiming},
	}

	for _, mode := range modes {
		mode := mode
		b.Run(mode.name, func(b *testing.B) {
			runPlannerWithObservabilityBenchmark(b, tree.Events, tree.Tokens, planner.Config{
				Target:    "",
				Telemetry: mode.level,
			})
		})
	}
}

// BenchmarkPlannerScaling verifies linear scaling with step count.
// Ensures plan generation time scales linearly, not quadratically.
func BenchmarkPlannerScaling(b *testing.B) {
	stepCounts := []int{10, 50, 100, 500}

	for _, count := range stepCounts {
		count := count
		b.Run(fmt.Sprintf("%d_steps", count), func(b *testing.B) {
			tree := parseBenchmarkSource(b, generateScriptWithSteps(count))
			runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{Target: ""})
		})
	}
}

// BenchmarkPlannerCommandMode measures function-scoped planning performance.
// Tests planning a specific function vs script mode.
func BenchmarkPlannerCommandMode(b *testing.B) {
	source := []byte(`fun hello = echo "Hello"
fun world = echo "World"
fun deploy = kubectl apply -f k8s/`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		b.Fatalf("Parse errors: %v", tree.Errors)
	}

	b.Run("script_mode", func(b *testing.B) {
		runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{
			Target: "", // Script mode - plans everything
		})
	})

	b.Run("command_mode", func(b *testing.B) {
		runPlannerBenchmark(b, tree.Events, tree.Tokens, planner.Config{
			Target: "hello", // Command mode - plans only 'hello'
		})
	})
}

// Helper functions

func parseBenchmarkSource(b *testing.B, source string) *parser.ParseTree {
	b.Helper()

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		b.Fatalf("Parse errors: %v", tree.Errors)
	}

	return tree
}

func runPlannerBenchmark(b *testing.B, events []parser.Event, tokens []lexer.Token, cfg planner.Config) {
	b.Helper()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := planner.Plan(events, tokens, cfg)
		if err != nil {
			b.Fatalf("Plan failed: %v", err)
		}
	}
}

func runPlannerWithObservabilityBenchmark(b *testing.B, events []parser.Event, tokens []lexer.Token, cfg planner.Config) {
	b.Helper()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := planner.PlanWithObservability(events, tokens, cfg)
		if err != nil {
			b.Fatalf("Plan failed: %v", err)
		}
	}
}

func generateComplexScript() string {
	return `echo "Starting deployment"
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl rollout status deployment/app
echo "Deployment complete"`
}

func generateScriptWithSteps(count int) string {
	var b strings.Builder
	for i := 0; i < count; i++ {
		b.WriteString(fmt.Sprintf("echo \"Step %d\"\n", i))
	}
	return b.String()
}
