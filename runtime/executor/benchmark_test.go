package executor_test

import (
	"context"
	"testing"

	"github.com/aledsdavies/opal/core/sdk"
	"github.com/aledsdavies/opal/runtime/executor"
)

// BenchmarkExecutorCore measures step execution performance.
// Target: <100µs per simple shell command, linear scaling with step count.
func BenchmarkExecutorCore(b *testing.B) {
	scenarios := map[string][]sdk.Step{
		"single_echo":     generateEchoSteps(1),
		"10_echos":        generateEchoSteps(10),
		"50_echos":        generateEchoSteps(50),
		"complex_command": generateComplexSteps(),
	}

	for name, steps := range scenarios {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := executor.Execute(context.Background(), steps, executor.Config{
					Telemetry: executor.TelemetryOff, // Zero overhead
				})
				if err != nil {
					b.Fatalf("Execute failed: %v", err)
				}
				if result.ExitCode != 0 {
					b.Fatalf("Command failed with exit code %d", result.ExitCode)
				}
			}
		})
	}
}

// BenchmarkExecutorTelemetryModes measures observability overhead.
// Verifies TelemetryOff has zero overhead, TelemetryTiming has minimal overhead.
func BenchmarkExecutorTelemetryModes(b *testing.B) {
	steps := generateEchoSteps(10)

	modes := map[string]executor.TelemetryLevel{
		"off":    executor.TelemetryOff,
		"basic":  executor.TelemetryBasic,
		"timing": executor.TelemetryTiming,
	}

	for name, mode := range modes {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := executor.Execute(context.Background(), steps, executor.Config{
					Telemetry: mode,
				})
				if err != nil {
					b.Fatalf("Execute failed: %v", err)
				}
				if result.ExitCode != 0 {
					b.Fatalf("Command failed with exit code %d", result.ExitCode)
				}
			}
		})
	}
}

// BenchmarkExecutorScaling verifies linear scaling with step count.
// Ensures execution time scales linearly, not quadratically.
func BenchmarkExecutorScaling(b *testing.B) {
	stepCounts := []int{1, 10, 50, 100}

	for _, count := range stepCounts {
		b.Run(string(rune('0'+count/100))+"_steps", func(b *testing.B) {
			steps := generateEchoSteps(count)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := executor.Execute(context.Background(), steps, executor.Config{
					Telemetry: executor.TelemetryOff,
				})
				if err != nil {
					b.Fatalf("Execute failed: %v", err)
				}
				if result.ExitCode != 0 {
					b.Fatalf("Command failed with exit code %d", result.ExitCode)
				}
			}
		})
	}
}

// Helper functions

func generateEchoSteps(count int) []sdk.Step {
	steps := make([]sdk.Step, count)
	for i := 0; i < count; i++ {
		steps[i] = sdk.Step{
			ID: uint64(i + 1),
			Tree: &sdk.CommandNode{
				Name: "shell",
				Args: map[string]interface{}{
					"command": "echo test",
				},
			},
		}
	}
	return steps
}

func generateComplexSteps() []sdk.Step {
	return []sdk.Step{
		{
			ID: 1,
			Tree: &sdk.CommandNode{
				Name: "shell",
				Args: map[string]interface{}{
					"command": "echo 'Starting'",
				},
			},
		},
		{
			ID: 2,
			Tree: &sdk.CommandNode{
				Name: "shell",
				Args: map[string]interface{}{
					"command": "sleep 0.01",
				},
			},
		},
		{
			ID: 3,
			Tree: &sdk.CommandNode{
				Name: "shell",
				Args: map[string]interface{}{
					"command": "echo 'Done'",
				},
			},
		},
	}
}
