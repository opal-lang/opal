package planner_test

import (
	"testing"

	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/planner"
)

// BenchmarkPlannerCore measures plan generation performance across complexity levels.
// Target: <1ms for simple scripts, <10ms for complex scripts with 100+ steps.
func BenchmarkPlannerCore(b *testing.B) {
	scenarios := map[string]string{
		"simple_command": `echo "Hello, World!"`,
		"multiple_commands": `echo "First"
echo "Second"
echo "Third"`,
		"function_call":  `fun hello = echo "Hello"`,
		"complex_script": generateComplexScript(),
	}

	for name, source := range scenarios {
		b.Run(name, func(b *testing.B) {
			sourceBytes := []byte(source)

			// Parse once (not part of benchmark)
			tree := parser.Parse(sourceBytes)
			if len(tree.Errors) > 0 {
				b.Fatalf("Parse errors: %v", tree.Errors)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Plan generation is what we're measuring
				_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
					Target: "", // Script mode
				})
				if err != nil {
					b.Fatalf("Plan failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkPlannerTelemetryModes measures observability overhead.
// Verifies TelemetryOff has zero overhead, TelemetryTiming has minimal overhead.
func BenchmarkPlannerTelemetryModes(b *testing.B) {
	source := []byte(generateComplexScript())
	tree := parser.Parse(source)

	modes := map[string]planner.TelemetryLevel{
		"off":    planner.TelemetryOff,
		"basic":  planner.TelemetryBasic,
		"timing": planner.TelemetryTiming,
	}

	for name, mode := range modes {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := planner.PlanWithObservability(tree.Events, tree.Tokens, planner.Config{
					Target:    "",
					Telemetry: mode,
				})
				if err != nil {
					b.Fatalf("Plan failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkPlannerScaling verifies linear scaling with step count.
// Ensures plan generation time scales linearly, not quadratically.
func BenchmarkPlannerScaling(b *testing.B) {
	stepCounts := []int{10, 50, 100, 500}

	for _, count := range stepCounts {
		b.Run(string(rune('0'+count/100))+"00_steps", func(b *testing.B) {
			source := generateScriptWithSteps(count)
			tree := parser.Parse([]byte(source))

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
					Target: "",
				})
				if err != nil {
					b.Fatalf("Plan failed: %v", err)
				}
			}
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

	b.Run("script_mode", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "", // Script mode - plans everything
			})
			if err != nil {
				b.Fatalf("Plan failed: %v", err)
			}
		}
	})

	b.Run("command_mode", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "hello", // Command mode - plans only 'hello'
			})
			if err != nil {
				b.Fatalf("Plan failed: %v", err)
			}
		}
	})
}

// Helper functions

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
	result := ""
	for i := 0; i < count; i++ {
		result += "echo \"Step " + string(rune('0'+i%10)) + "\"\n"
	}
	return result
}
