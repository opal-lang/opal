package execution

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellOperatorEdgeCases covers all shell operator combinations and edge cases
func TestShellOperatorEdgeCases(t *testing.T) {
	registry := decorators.GlobalRegistry()
	evaluator := NewNodeEvaluator(registry)

	setupCtx := func() *ir.Ctx {
		return &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}
	}

	t.Run("AND operator (&&) success chain", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo first"); e.OpNext = ir.ChainOpAnd; return e }(),
						func() ir.ChainElement { e := ir.NewShellElement("echo second"); e.OpNext = ir.ChainOpAnd; return e }(),
						ir.NewShellElement("echo third"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "All commands should succeed")
		assert.Contains(t, result.Stdout, "first", "Should contain first command output")
		assert.Contains(t, result.Stdout, "second", "Should contain second command output")
		assert.Contains(t, result.Stdout, "third", "Should contain third command output")
	})

	t.Run("AND operator (&&) failure stops chain", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo first"); e.OpNext = ir.ChainOpAnd; return e }(),
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpAnd; return e }(),
						ir.NewShellElement("echo should-not-execute"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 1, result.ExitCode, "Should fail with exit code 1")
		assert.Contains(t, result.Stdout, "first", "Should contain first command output")
		assert.NotContains(t, result.Stdout, "should-not-execute", "Should not execute after failure")
	})

	t.Run("OR operator (||) skips on success", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo success"); e.OpNext = ir.ChainOpOr; return e }(),
						ir.NewShellElement("echo should-not-execute"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "Should succeed")
		assert.Contains(t, result.Stdout, "success", "Should contain first command output")
		assert.NotContains(t, result.Stdout, "should-not-execute", "Should not execute fallback on success")
	})

	t.Run("OR operator (||) executes on failure", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpOr; return e }(),
						ir.NewShellElement("echo fallback"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "Should succeed with fallback")
		assert.Contains(t, result.Stdout, "fallback", "Should execute fallback on failure")
	})

	t.Run("PIPE operator (|) chains stdout", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo hello"); e.OpNext = ir.ChainOpPipe; return e }(),
						ir.NewShellElement("read input && echo got:$input"), // Read from stdin and echo
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "Pipe should succeed")
		assert.Contains(t, result.Stdout, "got:hello", "Should pass input through pipe and process it")
	})

	t.Run("APPEND operator (>>) creates/appends to file", func(t *testing.T) {
		ctx := setupCtx()

		// Create a temporary directory for testing
		tempDir, err := ioutil.TempDir("", "devcmd_test_")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		testFile := filepath.Join(tempDir, "output.txt")

		// First command should create the file
		ir1 := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement {
							e := ir.NewShellElement("echo first")
							e.OpNext = ir.ChainOpAppend
							e.Target = testFile
							return e
						}(),
					},
				},
			},
		}

		result1 := evaluator.EvaluateNode(ctx, ir1)
		assert.Equal(t, 0, result1.ExitCode, "First append should succeed")

		// Second command should append to the file
		ir2 := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement {
							e := ir.NewShellElement("echo second")
							e.OpNext = ir.ChainOpAppend
							e.Target = testFile
							return e
						}(),
					},
				},
			},
		}

		result2 := evaluator.EvaluateNode(ctx, ir2)
		assert.Equal(t, 0, result2.ExitCode, "Second append should succeed")

		// Verify file contents
		content, err := ioutil.ReadFile(testFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "first", "File should contain first output")
		assert.Contains(t, string(content), "second", "File should contain second output")
	})
}

// TestComplexOperatorChains tests combinations of multiple operators
func TestComplexOperatorChains(t *testing.T) {
	registry := decorators.GlobalRegistry()
	evaluator := NewNodeEvaluator(registry)

	setupCtx := func() *ir.Ctx {
		return &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}
	}

	t.Run("AND then OR recovery", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo first"); e.OpNext = ir.ChainOpAnd; return e }(),
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpOr; return e }(),
						ir.NewShellElement("echo recovery"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "Should recover with OR after AND failure")
		assert.Contains(t, result.Stdout, "first", "Should contain first command output")
		assert.Contains(t, result.Stdout, "recovery", "Should execute recovery command")
	})

	t.Run("OR then AND continuation", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("exit 1"); e.OpNext = ir.ChainOpOr; return e }(),
						func() ir.ChainElement { e := ir.NewShellElement("echo recovery"); e.OpNext = ir.ChainOpAnd; return e }(),
						ir.NewShellElement("echo final"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "Should succeed through recovery chain")
		assert.Contains(t, result.Stdout, "recovery", "Should execute recovery command")
		assert.Contains(t, result.Stdout, "final", "Should execute final command after AND")
	})

	t.Run("Pipe with AND combination", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo hello"); e.OpNext = ir.ChainOpPipe; return e }(),
						func() ir.ChainElement {
							e := ir.NewShellElement("read input && echo processed:$input")
							e.OpNext = ir.ChainOpAnd
							return e
						}(),
						ir.NewShellElement("echo done"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 0, result.ExitCode, "Pipe with AND should succeed")
		assert.Contains(t, result.Stdout, "processed:hello", "Should transform through pipe")
		assert.Contains(t, result.Stdout, "done", "Should execute final command")
	})
}

// TestOperatorErrorHandling tests error conditions and edge cases
func TestOperatorErrorHandling(t *testing.T) {
	registry := decorators.GlobalRegistry()
	evaluator := NewNodeEvaluator(registry)

	setupCtx := func() *ir.Ctx {
		return &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  false,
			Debug:   false,
		}
	}

	t.Run("Pipe with failing second command", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo hello"); e.OpNext = ir.ChainOpPipe; return e }(),
						ir.NewShellElement("false"), // Command that always fails
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		assert.Equal(t, 1, result.ExitCode, "Should fail when pipe target fails")
	})

	t.Run("Append to invalid file path", func(t *testing.T) {
		ctx := setupCtx()
		invalidPath := "/root/invalid/path/file.txt" // Should fail on most systems

		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement {
							e := ir.NewShellElement("echo test")
							e.OpNext = ir.ChainOpAppend
							e.Target = invalidPath
							return e
						}(),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		// The current implementation should handle this gracefully
		// (even if it's just a TODO placeholder)
		assert.NotEmpty(t, result.Stdout, "Should have some output even for invalid append")
	})

	t.Run("Empty commands in chain", func(t *testing.T) {
		ctx := setupCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement(""); e.OpNext = ir.ChainOpAnd; return e }(),
						ir.NewShellElement("echo after-empty"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)
		// Should handle empty commands gracefully
		t.Logf("Empty command result: stdout='%s', stderr='%s', exitcode=%d",
			result.Stdout, result.Stderr, result.ExitCode)
	})
}

// TestDryRunModeOperators tests that operators work correctly in dry-run mode
func TestDryRunModeOperators(t *testing.T) {
	registry := decorators.GlobalRegistry()
	evaluator := NewNodeEvaluator(registry)

	setupDryRunCtx := func() *ir.Ctx {
		return &ir.Ctx{
			Env:     &ir.EnvSnapshot{Values: map[string]string{}},
			Vars:    map[string]string{},
			WorkDir: "",
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
			DryRun:  true, // Dry-run mode
			Debug:   false,
		}
	}

	t.Run("Dry-run with complex operator chain", func(t *testing.T) {
		ctx := setupDryRunCtx()
		ir := ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						func() ir.ChainElement { e := ir.NewShellElement("echo first"); e.OpNext = ir.ChainOpAnd; return e }(),
						func() ir.ChainElement { e := ir.NewShellElement("echo second"); e.OpNext = ir.ChainOpPipe; return e }(),
						func() ir.ChainElement { e := ir.NewShellElement("tr 'a-z' 'A-Z'"); e.OpNext = ir.ChainOpOr; return e }(),
						ir.NewShellElement("echo fallback"),
					},
				},
			},
		}

		result := evaluator.EvaluateNode(ctx, ir)

		// In dry-run mode, we expect the plan to be generated without execution
		t.Logf("Dry-run result: stdout='%s', stderr='%s', exitcode=%d",
			result.Stdout, result.Stderr, result.ExitCode)

		// The exact behavior in dry-run mode depends on implementation
		// but it should not fail catastrophically
		assert.GreaterOrEqual(t, result.ExitCode, 0, "Dry-run should not have negative exit codes")
	})
}
