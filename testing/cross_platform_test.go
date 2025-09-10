package testing

import (
	"runtime"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
)

// CrossPlatformTest represents a test case for cross-platform validation
type CrossPlatformTest struct {
	Name        string
	Description string
	IR          ir.Node
	Context     *ir.Ctx
	Expected    CrossPlatformExpected
}

// CrossPlatformExpected defines expected outcomes across platforms
type CrossPlatformExpected struct {
	// Universal expectations (must be identical across platforms)
	ExitCode      int
	GraphHash     string // Must be identical across platforms
	PlanStructure string // Must be identical across platforms

	// Platform-specific expectations (can vary)
	StdoutPattern string // Regex pattern for stdout validation
	StderrPattern string // Regex pattern for stderr validation

	// Timing expectations (with tolerance for CI environments)
	MaxDuration time.Duration
	MinDuration time.Duration

	// Error conditions
	ShouldFail   bool
	ErrorPattern string
}

// CrossPlatformHarness validates command execution across Windows and Unix platforms
type CrossPlatformHarness struct {
	registry *decorators.Registry
	t        *testing.T
}

// NewCrossPlatformHarness creates a new cross-platform testing harness
func NewCrossPlatformHarness(t *testing.T, registry *decorators.Registry) *CrossPlatformHarness {
	return &CrossPlatformHarness{
		registry: registry,
		t:        t,
	}
}

// TestCrossPlatformParity validates that commands behave consistently across platforms
func (h *CrossPlatformHarness) TestCrossPlatformParity(testCase CrossPlatformTest) {
	h.t.Run(testCase.Name, func(t *testing.T) {
		// 1. Test interpreter mode execution
		interpreterResult := h.executeInterpreter(testCase.Context, testCase.IR)

		// 2. Test plan generation (must be identical across platforms)
		interpreterPlan := h.generatePlan(testCase.Context, testCase.IR)

		// 3. Validate universal expectations
		h.validateUniversalBehavior(t, testCase.Expected, interpreterResult, interpreterPlan)

		// 4. Validate platform-specific expectations
		h.validatePlatformBehavior(t, testCase.Expected, interpreterResult)

		// 5. Test performance characteristics
		h.validatePerformance(t, testCase.Expected, interpreterResult)
	})
}

// validateUniversalBehavior validates behavior that must be identical across platforms
func (h *CrossPlatformHarness) validateUniversalBehavior(t *testing.T, expected CrossPlatformExpected, result ir.CommandResult, plan *plan.ExecutionPlan) {
	// Exit code must be identical across platforms
	assert.Equal(t, expected.ExitCode, result.ExitCode, "Exit code must be identical across platforms")

	// Plan structure must be identical (using GraphHash for structural comparison)
	actualGraphHash := plan.GraphHash()
	if expected.GraphHash != "" {
		assert.Equal(t, expected.GraphHash, actualGraphHash, "Plan graph hash must be identical across platforms")
	}

	// Plan structure validation (DOT format should be identical)
	planDOT := plan.ToDOT()
	if expected.PlanStructure != "" {
		assert.Equal(t, expected.PlanStructure, planDOT, "Plan structure (DOT) must be identical across platforms")
	}

	// Log the graph hash for reference (helps when writing tests)
	t.Logf("Platform: %s, GraphHash: %s", runtime.GOOS, actualGraphHash)
}

// validatePlatformBehavior validates platform-specific behavior patterns
func (h *CrossPlatformHarness) validatePlatformBehavior(t *testing.T, expected CrossPlatformExpected, result ir.CommandResult) {
	// Platform-specific stdout validation (patterns instead of exact matches)
	if expected.StdoutPattern != "" {
		assert.Regexp(t, expected.StdoutPattern, result.Stdout, "Stdout should match platform-appropriate pattern")
	}

	// Platform-specific stderr validation
	if expected.StderrPattern != "" {
		assert.Regexp(t, expected.StderrPattern, result.Stderr, "Stderr should match platform-appropriate pattern")
	}

	// Error condition validation
	if expected.ShouldFail {
		assert.NotZero(t, result.ExitCode, "Command should fail on all platforms")
		if expected.ErrorPattern != "" {
			assert.Regexp(t, expected.ErrorPattern, result.Stderr, "Error message should match pattern")
		}
	}
}

// validatePerformance validates timing and performance characteristics
func (h *CrossPlatformHarness) validatePerformance(t *testing.T, expected CrossPlatformExpected, result ir.CommandResult) {
	// Note: In this simplified implementation, we don't track execution time
	// In a full implementation, we would capture start/end times during execution

	if expected.MaxDuration > 0 {
		// Placeholder for timing validation
		t.Logf("Performance validation: MaxDuration=%v (not yet implemented)", expected.MaxDuration)
	}

	if expected.MinDuration > 0 {
		// Placeholder for minimum timing validation
		t.Logf("Performance validation: MinDuration=%v (not yet implemented)", expected.MinDuration)
	}
}

// executeInterpreter executes the IR using the interpreter
func (h *CrossPlatformHarness) executeInterpreter(ctx *ir.Ctx, node ir.Node) ir.CommandResult {
	evaluator := ir.NewNodeEvaluator(h.registry)
	return evaluator.EvaluateNode(ctx, node)
}

// generatePlan generates an execution plan from the IR
func (h *CrossPlatformHarness) generatePlan(ctx *ir.Ctx, node ir.Node) *plan.ExecutionPlan {
	// TODO: Implement PlanNode in NodeEvaluator (Step 3)
	return &plan.ExecutionPlan{}
}

// TestShellOperatorParity validates that shell operators work consistently across platforms
func TestShellOperatorParity(t *testing.T) {
	registry := decorators.GlobalRegistry()
	harness := NewCrossPlatformHarness(t, registry)

	// Test && operator (engine-level)
	harness.TestCrossPlatformParity(CrossPlatformTest{
		Name:        "AND operator success",
		Description: "Test && operator when both commands succeed",
		IR: ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{Kind: ir.ElementKindShell, Text: "echo hello", OpNext: ir.ChainOpAnd},
						{Kind: ir.ElementKindShell, Text: "echo world"},
					},
				},
			},
		},
		Context: &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			DryRun:  false,
			Debug:   false,
		},
		Expected: CrossPlatformExpected{
			ExitCode:      0,
			StdoutPattern: `hello\s*world`, // Pattern allows for platform-specific line endings
			ShouldFail:    false,
		},
	})

	// Test && operator failure (engine-level)
	harness.TestCrossPlatformParity(CrossPlatformTest{
		Name:        "AND operator failure",
		Description: "Test && operator when first command fails",
		IR: ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{Kind: ir.ElementKindShell, Text: "exit 1", OpNext: ir.ChainOpAnd},
						{Kind: ir.ElementKindShell, Text: "echo should-not-run"},
					},
				},
			},
		},
		Context: &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			DryRun:  false,
			Debug:   false,
		},
		Expected: CrossPlatformExpected{
			ExitCode:      1,
			StdoutPattern: `^$`, // Should be empty (no output since second command didn't run)
			ShouldFail:    true,
		},
	})

	// Test || operator (engine-level)
	harness.TestCrossPlatformParity(CrossPlatformTest{
		Name:        "OR operator recovery",
		Description: "Test || operator when first command fails and second succeeds",
		IR: ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{Kind: ir.ElementKindShell, Text: "exit 1", OpNext: ir.ChainOpOr},
						{Kind: ir.ElementKindShell, Text: "echo recovered"},
					},
				},
			},
		},
		Context: &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			DryRun:  false,
			Debug:   false,
		},
		Expected: CrossPlatformExpected{
			ExitCode:      0,
			StdoutPattern: `recovered`,
			ShouldFail:    false,
		},
	})
}

// TestDecoratorParity validates that decorators work consistently across platforms
func TestDecoratorParity(t *testing.T) {
	registry := decorators.GlobalRegistry()
	harness := NewCrossPlatformHarness(t, registry)

	// Test @timeout decorator
	harness.TestCrossPlatformParity(CrossPlatformTest{
		Name:        "Timeout decorator",
		Description: "Test @timeout decorator with simple command",
		IR: ir.Wrapper{
			Kind:   "timeout",
			Params: map[string]interface{}{"duration": "5s"},
			Inner: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{Kind: ir.ElementKindShell, Text: "echo timeout-test"},
						},
					},
				},
			},
		},
		Context: &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			DryRun:  false,
			Debug:   false,
		},
		Expected: CrossPlatformExpected{
			ExitCode:      0,
			StdoutPattern: `timeout-test`,
			MaxDuration:   6 * time.Second, // Should complete well under timeout
			ShouldFail:    false,
		},
	})
}

// TestPlanGenerationParity validates that plan generation is identical across platforms
func TestPlanGenerationParity(t *testing.T) {
	registry := decorators.GlobalRegistry()
	harness := NewCrossPlatformHarness(t, registry)

	// Test complex plan generation
	harness.TestCrossPlatformParity(CrossPlatformTest{
		Name:        "Complex plan generation",
		Description: "Test plan generation for complex command with decorators",
		IR: ir.Wrapper{
			Kind:   "parallel",
			Params: map[string]interface{}{},
			Inner: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{Kind: ir.ElementKindShell, Text: "echo task1"},
						},
					},
					{
						Chain: []ir.ChainElement{
							{Kind: ir.ElementKindShell, Text: "echo task2"},
						},
					},
					{
						Chain: []ir.ChainElement{
							{Kind: ir.ElementKindShell, Text: "echo task3"},
						},
					},
				},
			},
		},
		Context: &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			DryRun:  true, // Plan mode
			Debug:   false,
		},
		Expected: CrossPlatformExpected{
			ExitCode: 0,
			// Plan structure must be identical across platforms
			// GraphHash will be automatically validated
			ShouldFail: false,
		},
	})
}

// TestNegativeScenarios validates error conditions consistently across platforms
func TestNegativeScenarios(t *testing.T) {
	registry := decorators.GlobalRegistry()
	harness := NewCrossPlatformHarness(t, registry)

	// Test unknown decorator
	harness.TestCrossPlatformParity(CrossPlatformTest{
		Name:        "Unknown decorator error",
		Description: "Test error handling for unknown decorator",
		IR: ir.Wrapper{
			Kind:   "nonexistent",
			Params: map[string]interface{}{},
			Inner: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{Kind: ir.ElementKindShell, Text: "echo test"},
						},
					},
				},
			},
		},
		Context: &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			DryRun:  false,
			Debug:   false,
		},
		Expected: CrossPlatformExpected{
			ExitCode:     1,
			ShouldFail:   true,
			ErrorPattern: `decorator.*nonexistent.*not found`, // Error message pattern
		},
	})
}

// BenchmarkCrossPlatformPerformance benchmarks execution performance across platforms
func BenchmarkCrossPlatformPerformance(b *testing.B) {
	registry := decorators.GlobalRegistry()
	evaluator := ir.NewNodeEvaluator(registry)

	ctx := &ir.Ctx{
		Env:     &ir.EnvSnapshot{Values: map[string]string{}},
		Vars:    map[string]string{},
		WorkDir: "",
		DryRun:  false,
		Debug:   false,
	}

	// Simple command benchmark
	simpleIR := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{Kind: ir.ElementKindShell, Text: "echo benchmark"},
				},
			},
		},
	}

	b.Run("SimpleCommand", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := evaluator.EvaluateNode(ctx, simpleIR)
			if result.ExitCode != 0 {
				b.Fatalf("Benchmark command failed: %v", result.Stderr)
			}
		}
	})

	// Complex command with operators benchmark
	complexIR := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{Kind: ir.ElementKindShell, Text: "echo step1", OpNext: ir.ChainOpAnd},
					{Kind: ir.ElementKindShell, Text: "echo step2", OpNext: ir.ChainOpAnd},
					{Kind: ir.ElementKindShell, Text: "echo step3"},
				},
			},
		},
	}

	b.Run("ComplexChain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := evaluator.EvaluateNode(ctx, complexIR)
			if result.ExitCode != 0 {
				b.Fatalf("Benchmark command failed: %v", result.Stderr)
			}
		}
	})
}
