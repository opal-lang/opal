package executor_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/executor"
	"github.com/opal-lang/opal/runtime/vault"
)

// Helper to create a vault for testing
func testVault() *vault.Vault {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}
	return vault.NewWithPlanKey(planKey)
}

// BenchmarkExecutorCore measures step execution performance.
// Target: <100µs per simple shell command, linear scaling with step count.
func BenchmarkExecutorCore(b *testing.B) {
	scenarios := map[string]*planfmt.Plan{
		"single_echo":     generateEchoPlan(1),
		"10_echos":        generateEchoPlan(10),
		"50_echos":        generateEchoPlan(50),
		"complex_command": generateComplexPlan(),
	}

	for name, plan := range scenarios {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := executor.ExecutePlan(context.Background(), plan, executor.Config{
					Telemetry: executor.TelemetryOff, // Zero overhead
				}, testVault())
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
	plan := generateEchoPlan(10)

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
				result, err := executor.ExecutePlan(context.Background(), plan, executor.Config{
					Telemetry: mode,
				}, testVault())
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
		b.Run(fmt.Sprintf("%d_steps", count), func(b *testing.B) {
			plan := generateEchoPlan(count)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := executor.ExecutePlan(context.Background(), plan, executor.Config{
					Telemetry: executor.TelemetryOff,
				}, testVault())
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

func generateEchoPlan(count int) *planfmt.Plan {
	steps := make([]planfmt.Step, count)
	for i := 0; i < count; i++ {
		steps[i] = planfmt.Step{
			ID: uint64(i + 1),
			Tree: &planfmt.CommandNode{
				Decorator: "@shell",
				Args: []planfmt.Arg{{
					Key: "command",
					Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"},
				}},
			},
		}
	}
	return &planfmt.Plan{Target: "benchmark", Steps: steps}
}

func generateComplexPlan() *planfmt.Plan {
	return &planfmt.Plan{
		Target: "benchmark",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{{
						Key: "command",
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'Starting'"},
					}},
				},
			},
			{
				ID: 2,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{{
						Key: "command",
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "sleep 0.01"},
					}},
				},
			},
			{
				ID: 3,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{{
						Key: "command",
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'Done'"},
					}},
				},
			},
		},
	}
}
