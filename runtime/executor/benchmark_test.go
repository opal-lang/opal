package executor_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/executor"
	"github.com/opal-lang/opal/runtime/vault"
)

type executorScenario struct {
	name string
	plan *planfmt.Plan
}

// Helper to create a vault for testing
func testVault() *vault.Vault {
	planKey := make([]byte, 32)
	for i := range planKey {
		planKey[i] = byte(i)
	}
	return vault.NewWithPlanKey(planKey)
}

// BenchmarkExecutorCore measures step execution performance.
// Scenarios suppress command output so measurements focus on runtime overhead.
func BenchmarkExecutorCore(b *testing.B) {
	scenarios := []executorScenario{
		{name: "single_noop", plan: generateCommandPlan(1, ":")},
		{name: "single_echo_devnull", plan: generateCommandPlan(1, "echo test >/dev/null")},
		{name: "10_noop", plan: generateCommandPlan(10, ":")},
		{name: "50_noop", plan: generateCommandPlan(50, ":")},
		{name: "complex_shell_work", plan: generateComplexPlan()},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		b.Run(scenario.name, func(b *testing.B) {
			runExecutorBenchmark(b, scenario.plan, executor.TelemetryOff)
		})
	}
}

// BenchmarkExecutorTelemetryModes measures observability overhead.
// Verifies TelemetryOff has zero overhead, TelemetryTiming has minimal overhead.
func BenchmarkExecutorTelemetryModes(b *testing.B) {
	plan := generateCommandPlan(10, ":")

	modes := []struct {
		name  string
		level executor.TelemetryLevel
	}{
		{name: "off", level: executor.TelemetryOff},
		{name: "basic", level: executor.TelemetryBasic},
		{name: "timing", level: executor.TelemetryTiming},
	}

	for _, mode := range modes {
		mode := mode
		b.Run(mode.name, func(b *testing.B) {
			runExecutorBenchmark(b, plan, mode.level)
		})
	}
}

// BenchmarkExecutorScaling verifies linear scaling with step count.
// Ensures execution time scales linearly, not quadratically.
func BenchmarkExecutorScaling(b *testing.B) {
	stepCounts := []int{1, 10, 50, 100}

	for _, count := range stepCounts {
		count := count
		b.Run(fmt.Sprintf("%d_steps", count), func(b *testing.B) {
			plan := generateCommandPlan(count, ":")
			runExecutorBenchmark(b, plan, executor.TelemetryOff)
		})
	}
}

// Helper functions

func runExecutorBenchmark(b *testing.B, plan *planfmt.Plan, telemetry executor.TelemetryLevel) {
	b.Helper()

	ctx := context.Background()
	vlt := testVault()
	cfg := executor.Config{Telemetry: telemetry}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := executor.ExecutePlan(ctx, plan, cfg, vlt)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
		if result.ExitCode != 0 {
			b.Fatalf("Command failed with exit code %d", result.ExitCode)
		}
	}
}

func generateCommandPlan(count int, command string) *planfmt.Plan {
	steps := make([]planfmt.Step, count)
	for i := 0; i < count; i++ {
		steps[i] = planfmt.Step{
			ID: uint64(i + 1),
			Tree: &planfmt.CommandNode{
				Decorator: "@shell",
				Args: []planfmt.Arg{{
					Key: "command",
					Val: planfmt.Value{Kind: planfmt.ValueString, Str: command},
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
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value=1; value=$((value + 41)); [ \"$value\" -eq 42 ]"},
					}},
				},
			},
			{
				ID: 2,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{{
						Key: "command",
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "printf 'alpha beta gamma' | wc -w >/dev/null"},
					}},
				},
			},
			{
				ID: 3,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{{
						Key: "command",
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo done >/dev/null"},
					}},
				},
			},
		},
	}
}
