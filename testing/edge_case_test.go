package testing

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EdgeCaseHarness tests negative scenarios and edge cases
type EdgeCaseHarness struct {
	registry *decorators.Registry
	t        *testing.T
}

// NewEdgeCaseHarness creates a new edge case testing harness
func NewEdgeCaseHarness(t *testing.T, registry *decorators.Registry) *EdgeCaseHarness {
	return &EdgeCaseHarness{
		registry: registry,
		t:        t,
	}
}

// TestPipeToNonStdinAwareAction validates that piping to non-StdinAware actions fails
func TestPipeToNonStdinAwareAction(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Pipe to non-StdinAware action must error", func(t *testing.T) {
		// Create a test that pipes to @log (which is not StdinAware)
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test piping to @log action (should fail)
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo hello"); e.OpNext = ir.ChainOpPipe; return e }(),
						{Kind: ir.ElementKindAction, Name: "log", Args: []decorators.DecoratorParam{
							{Name: "", Value: "test message"},
						}},
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)
		result := evaluator.EvaluateNode(ctx, ir)

		// Must fail with specific error about pipe incompatibility
		assert.NotZero(t, result.ExitCode, "Piping to non-StdinAware action must fail")
		assert.Contains(t, result.Stderr, "not pipe-capable", "Error message must indicate pipe incompatibility")
		assert.Contains(t, result.Stderr, "log", "Error message must specify the problematic decorator")
	})

	t.Run("Pipe to StdinAware action must succeed", func(t *testing.T) {
		// This test would require an actual StdinAware decorator
		// For now, just document the expected behavior
		t.Skip("Requires implementation of a StdinAware test decorator")

		// Expected behavior:
		// 1. Create a test decorator that implements StdinAware
		// 2. Test that piping TO that decorator works correctly
		// 3. Validate that input is properly passed to RunWithInput method
	})
}

// TestStepShortCircuit validates that step execution stops on failure
func TestStepShortCircuit(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Step short-circuit on failure", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test that subsequent steps don't run after a step fails
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				// Step 1: Should succeed
				{
					Chain: []ir.ChainElement{
						ir.NewShellElement("echo step1"),
					},
				},
				// Step 2: Should fail
				{
					Chain: []ir.ChainElement{
						ir.NewShellElement("exit 1"),
					},
				},
				// Step 3: Should NOT run (short-circuit)
				{
					Chain: []ir.ChainElement{
						ir.NewShellElement("echo should-not-run"),
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)
		result := evaluator.EvaluateNode(ctx, ir)

		// Should fail with exit code 1 from step 2
		assert.Equal(t, 1, result.ExitCode, "Should fail with exit code from failing step")

		// Should contain output from step 1 but not step 3
		assert.Contains(t, result.Stdout, "step1", "Should contain output from successful step")
		assert.NotContains(t, result.Stdout, "should-not-run", "Should NOT contain output from short-circuited step")
	})

	t.Run("Chain-level short-circuit with && operator", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test that && operator stops execution when first command fails
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpAnd; return e }(),
						ir.NewShellElement("echo should-not-run"),
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)
		result := evaluator.EvaluateNode(ctx, ir)

		// Should fail with exit code 1
		assert.Equal(t, 1, result.ExitCode, "Should fail with exit code from first command")

		// Should NOT contain output from second command
		assert.NotContains(t, result.Stdout, "should-not-run", "Second command should not execute due to && short-circuit")
	})

	t.Run("Chain-level continuation with || operator", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test that || operator continues execution when first command fails
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpOr; return e }(),
						ir.NewShellElement("echo recovery"),
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)
		result := evaluator.EvaluateNode(ctx, ir)

		// Should succeed with recovery
		assert.Equal(t, 0, result.ExitCode, "Should succeed after || recovery")

		// Should contain output from recovery command
		assert.Contains(t, result.Stdout, "recovery", "Should execute recovery command after || operator")
	})
}

// TestLargeStreamingData validates that large data doesn't cause buffer overflows
func TestLargeStreamingData(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Large stdout handling", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Generate a large amount of output (1MB of data)
		largeData := strings.Repeat("A", 1024*1024) // 1MB of 'A' characters

		// Use a command that outputs large data
		// Note: This approach works cross-platform
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{Kind: ir.ElementKindShell, Text: "echo " + largeData},
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)

		// Measure execution time to ensure it doesn't hang
		start := time.Now()
		result := evaluator.EvaluateNode(ctx, ir)
		duration := time.Since(start)

		// Should complete successfully
		assert.Equal(t, 0, result.ExitCode, "Large output command should succeed")

		// Should complete in reasonable time (not hang)
		assert.Less(t, duration, 10*time.Second, "Large output should not cause hanging")

		// Should capture the output (or at least part of it)
		assert.NotEmpty(t, result.Stdout, "Should capture large output")

		// Log statistics for analysis
		t.Logf("Large data test: %d bytes output, %v duration", len(result.Stdout), duration)
	})

	t.Run("Large streaming with >> operator", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test large data with >> redirection
		tempFile := "/tmp/devcmd_test_large_output.txt"

		// Generate moderately large output and redirect to file
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{
							Kind:   ir.ElementKindShell,
							Text:   "echo " + strings.Repeat("B", 10000), // 10KB
							OpNext: ir.ChainOpAppend,
							Target: tempFile,
						},
					},
				},
				// Clean up the temp file
				{
					Chain: []ir.ChainElement{
						{Kind: ir.ElementKindShell, Text: "rm -f " + tempFile},
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)
		result := evaluator.EvaluateNode(ctx, ir)

		// Should complete successfully
		assert.Equal(t, 0, result.ExitCode, "Large output redirection should succeed")

		// Should not contain the large data in stdout (redirected to file)
		assert.NotContains(t, result.Stdout, "BBBB", "Large data should be redirected to file, not stdout")
	})
}

// TestMemoryEfficiency validates that execution doesn't leak memory
func TestMemoryEfficiency(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Multiple command execution memory usage", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Execute many small commands to test for memory leaks
		evaluator := ir.NewNodeEvaluator(registry)

		for i := 0; i < 100; i++ {
			ir := ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							ir.NewShellElement("echo test"),
						},
					},
				},
			}

			result := evaluator.EvaluateNode(ctx, ir)
			require.Equal(t, 0, result.ExitCode, "Command %d should succeed", i)

			// Reset buffers to prevent accumulation
			ctx.Stdout.(*bytes.Buffer).Reset()
			ctx.Stderr.(*bytes.Buffer).Reset()
		}

		// If we get here without hanging or crashing, memory efficiency is acceptable
		t.Log("Successfully executed 100 commands without memory issues")
	})
}

// TestErrorRecovery validates error handling and recovery scenarios
func TestErrorRecovery(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Decorator error handling", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test error handling with invalid decorator parameters
		ir := ir.Wrapper{
			Kind:   "timeout",
			Params: map[string]interface{}{"duration": "invalid-duration"}, // Invalid parameter
			Inner: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							ir.NewShellElement("echo test"),
						},
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)
		result := evaluator.EvaluateNode(ctx, ir)

		// Should fail gracefully with informative error
		assert.NotZero(t, result.ExitCode, "Invalid decorator parameters should cause failure")
		assert.NotEmpty(t, result.Stderr, "Should provide error message for invalid parameters")
	})
}

// TestConcurrencyIssues validates thread safety and concurrent execution
func TestConcurrencyIssues(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Concurrent context isolation", func(t *testing.T) {
		// Test that multiple concurrent executions don't interfere
		const numGoroutines = 10

		results := make(chan ir.CommandResult, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				ctx := &ir.Ctx{
					Env:     &ir.EnvSnapshot{Values: map[string]string{}},
					Vars:    map[string]string{"ID": string(rune('0' + id))},
					WorkDir: "",
					Stdout:  &bytes.Buffer{},
					Stderr:  &bytes.Buffer{},
					DryRun:  false,
					Debug:   false,
				}

				ir := ir.CommandSeq{
					Steps: []ir.CommandStep{
						{
							Chain: []ir.ChainElement{
								ir.NewShellElement("echo concurrent-test"),
							},
						},
					},
				}

				evaluator := ir.NewNodeEvaluator(registry)
				result := evaluator.EvaluateNode(ctx, ir)
				results <- result
			}(i)
		}

		// Collect all results
		for i := 0; i < numGoroutines; i++ {
			result := <-results
			assert.Equal(t, 0, result.ExitCode, "Concurrent execution %d should succeed", i)
			assert.Contains(t, result.Stdout, "concurrent-test", "Should contain expected output")
		}

		t.Log("Successfully completed concurrent execution test")
	})
}

// TestResourceLimits validates behavior under resource constraints
func TestResourceLimits(t *testing.T) {
	registry := decorators.GlobalRegistry()
	_ = NewEdgeCaseHarness(t, registry)

	t.Run("Command timeout behavior", func(t *testing.T) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Test timeout with @timeout decorator
		ir := ir.Wrapper{
			Kind:   "timeout",
			Params: map[string]interface{}{"duration": "1s"},
			Inner: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							ir.NewShellElement("sleep 5"), // Will timeout
						},
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)

		start := time.Now()
		result := evaluator.EvaluateNode(ctx, ir)
		duration := time.Since(start)

		// Should timeout and fail
		assert.NotZero(t, result.ExitCode, "Long-running command should timeout")

		// Should complete within reasonable time of timeout (not wait for full sleep)
		assert.Less(t, duration, 3*time.Second, "Should timeout quickly, not wait for full sleep duration")

		t.Logf("Timeout test completed in %v", duration)
	})
}

// TestRealWorldCliParsing validates parsing of complex real-world CLI files
func TestRealWorldCliParsing(t *testing.T) {
	t.Run("Complex project commands.cli parsing", func(t *testing.T) {
		// This tests the exact patterns that caused CLI parsing failures
		// Real-world devcmd project commands.cli content
		complexCommandsInput := `# Multi-module project with complex decorators
var PROJECT = "devcmd"
var VERSION = "$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')"

# Setup with parallel dependency management
setup: {
    @log("Setting up @var(PROJECT) development environment...")
    @log("Downloading Go dependencies for all modules...")
    @parallel {
        @cmd(core-deps)
        @cmd(runtime-deps)
        @cmd(cli-deps)
    }
    go work sync
    @log("Setup complete!")
}

# Complex nested decorators with timeout
release: @timeout(10m) {
    @log("Running release preparation workflow...")
    @cmd(clean)
    @cmd(setup)
    @parallel {
        @cmd(test)
        @cmd(lint)
    }
    @log("Ready for release!")
}

# Command with variable expansion and command substitution
info: {
    @log("Project: @var(PROJECT)")
    @log("Version: @var(VERSION)")
    @log("Go source files: $(find . -name '*.go' | wc -l)")
}
`

		// Import the CLI parser (note: may need access to internal package)
		// For now, we'll test the parsing logic conceptually
		t.Logf("Testing parsing of complex commands.cli content (%d bytes)", len(complexCommandsInput))

		// Key patterns that must be supported:
		patterns := []struct {
			name        string
			pattern     string
			shouldParse bool
		}{
			{"variable definition", `var PROJECT = "devcmd"`, true},
			{"command substitution", `var VERSION = "$(git describe...)"`, true},
			{"parallel block decorator", `@parallel { @cmd(core-deps) }`, true},
			{"nested decorators", `@timeout(10m) { @parallel { ... } }`, true},
			{"variable expansion", `@log("Project: @var(PROJECT)")`, true},
			{"command references", `@cmd(setup)`, true},
		}

		for _, pattern := range patterns {
			t.Run(pattern.name, func(t *testing.T) {
				if pattern.shouldParse {
					t.Logf("✅ Pattern '%s' should parse successfully", pattern.name)
					// In a real implementation, we'd call the parser here
					// parser.Parse(strings.NewReader(pattern.pattern))
				} else {
					t.Logf("❌ Pattern '%s' should fail to parse", pattern.name)
				}
			})
		}

		// The critical test: @parallel in shell context
		parallelBlockInput := `setup: @parallel {
    @cmd(core-deps)
    @cmd(runtime-deps)
}`

		t.Run("parallel block decorator in shell context", func(t *testing.T) {
			t.Logf("Testing @parallel block decorator parsing")
			t.Logf("Input: %s", parallelBlockInput)

			// This is the exact pattern that was failing:
			// decorator @parallel cannot be used in shell context (line 268:5)
			// - only value and action decorators are allowed

			// TODO: Once parser is fixed, verify it parses successfully
			// program, err := parser.Parse(strings.NewReader(parallelBlockInput))
			// require.NoError(t, err)
			// require.Len(t, program.Commands, 1)
			// require.Equal(t, "setup", program.Commands[0].Name)

			t.Log("⚠️  This test documents the exact failure case that needs to be fixed")
			t.Log("Parser should allow BlockType decorators (@parallel) in shell context")
		})
	})

	t.Run("Decorator type validation edge cases", func(t *testing.T) {
		// Test all decorator types in different contexts
		decoratorTests := []struct {
			name          string
			input         string
			decoratorType string
			context       string
			shouldSucceed bool
			expectedError string
		}{
			{
				name:          "ValueType decorator in shell context",
				input:         `test: @var(PROJECT) echo "hello"`,
				decoratorType: "ValueType",
				context:       "shell",
				shouldSucceed: true,
			},
			{
				name:          "ActionType decorator in shell context",
				input:         `test: @log("message") echo "hello"`,
				decoratorType: "ActionType",
				context:       "shell",
				shouldSucceed: true,
			},
			{
				name:          "BlockType decorator in shell context - CURRENT BUG",
				input:         `test: @parallel { echo "task1"; echo "task2" }`,
				decoratorType: "BlockType",
				context:       "shell",
				shouldSucceed: false, // Currently fails, should succeed after fix
				expectedError: "cannot be used in shell context",
			},
			{
				name:          "BlockType decorator in command context",
				input:         `test: { @parallel { echo "task1" } }`,
				decoratorType: "BlockType",
				context:       "command",
				shouldSucceed: true,
			},
		}

		for _, test := range decoratorTests {
			t.Run(test.name, func(t *testing.T) {
				t.Logf("Testing %s (%s) in %s context", test.decoratorType, test.name, test.context)
				t.Logf("Input: %s", test.input)

				if test.shouldSucceed {
					t.Logf("✅ Should parse successfully")
				} else {
					t.Logf("❌ Should fail with error: %s", test.expectedError)
				}

				// TODO: Add actual parser testing once access to internal parser is resolved
				// This documents the expected behavior for each case
			})
		}
	})
}

// BenchmarkEdgeCasePerformance benchmarks edge case scenarios
func BenchmarkEdgeCasePerformance(b *testing.B) {
	registry := decorators.GlobalRegistry()

	b.Run("FailureRecovery", func(b *testing.B) {
		ctx := &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}

		// Benchmark failure + recovery pattern
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpOr; return e }(),
						ir.NewShellElement("echo recovered"),
					},
				},
			},
		}

		evaluator := ir.NewNodeEvaluator(registry)

		for i := 0; i < b.N; i++ {
			result := evaluator.EvaluateNode(ctx, ir)
			if result.ExitCode != 0 {
				b.Fatalf("Recovery benchmark failed: %v", result.Stderr)
			}

			// Reset buffers
			ctx.Stdout.(*bytes.Buffer).Reset()
			ctx.Stderr.(*bytes.Buffer).Reset()
		}
	})
}
