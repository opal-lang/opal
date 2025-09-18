package plan

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/ir"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import builtin decorators for testing
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin"
)

func TestPlanGenerator_BasicShellCommand(t *testing.T) {
	// Setup
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a basic shell command IR
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo hello"},
							},
						},
					},
				},
			},
		},
	}

	// Create execution context
	ctx := createTestContext()

	// Generate plan
	executionPlan := generator.GenerateFromIR(ctx, seq, "test-command")

	// Test the complete output
	got := executionPlan.StringNoColor()
	want := `test-command:
└─ Execute 1 command steps
   └─ echo hello`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}

	// Verify structure assertions
	assert.Equal(t, "test-command", executionPlan.Context["command_name"])
	assert.Equal(t, "dry_run", executionPlan.Context["mode"])
	assert.Len(t, executionPlan.Steps, 1)
}

func TestPlanGenerator_ChainWithAndOperator(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a chain: echo hello && echo world
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo hello"},
							},
						},
						OpNext: ir.ChainOpAnd,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo world"},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "chain-test")

	got := executionPlan.StringNoColor()
	want := `chain-test:
└─ Execute 1 command steps
   └─ echo hello && echo world`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ChainWithOrOperator(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a chain: echo hello || echo fallback
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo hello"},
							},
						},
						OpNext: ir.ChainOpOr,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo fallback"},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "or-test")

	got := executionPlan.StringNoColor()
	want := `or-test:
└─ Execute 1 command steps
   └─ echo hello || echo fallback`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ChainWithPipeOperator(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a chain: echo hello | grep hello
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo hello"},
							},
						},
						OpNext: ir.ChainOpPipe,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "grep hello"},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "pipe-test")

	got := executionPlan.StringNoColor()
	want := `pipe-test:
└─ Execute 1 command steps
   └─ echo hello | grep hello`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ChainWithAppendOperator(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a chain: echo hello >> output.txt
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo hello"},
							},
						},
						OpNext: ir.ChainOpAppend,
						Target: "output.txt",
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "append-test")

	got := executionPlan.StringNoColor()
	want := `append-test:
└─ Execute 1 command steps
   └─ echo hello >> output.txt`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}

	// Verify metadata includes the target
	require.Len(t, executionPlan.Steps, 1)
	mainStep := executionPlan.Steps[0]
	require.Len(t, mainStep.Children, 1)
	shellStep := mainStep.Children[0]
	assert.Equal(t, ">>", shellStep.Metadata["op_next"])
}

func TestPlanGenerator_MultipleSteps(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create multiple command steps (newline separated)
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo step1"},
							},
						},
					},
				},
			},
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo step2"},
							},
						},
					},
				},
			},
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo step3"},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "multi-step")

	got := executionPlan.StringNoColor()
	want := `multi-step:
└─ Execute 3 command steps
   ├─ echo step1
   ├─ echo step2
   └─ echo step3`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_EmptyCommand(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create an empty command sequence
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{Chain: []ir.ChainElement{}}, // Empty chain
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "empty")

	got := executionPlan.StringNoColor()
	want := `empty:
└─ Execute 1 command steps
   └─ (empty)`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ComplexChainOperators(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a complex chain: echo start && (echo middle | grep middle) || echo fallback
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo start"},
							},
						},
						OpNext: ir.ChainOpAnd,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo middle"},
							},
						},
						OpNext: ir.ChainOpPipe,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "grep middle"},
							},
						},
						OpNext: ir.ChainOpOr,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo fallback"},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "complex-chain")

	got := executionPlan.StringNoColor()
	want := `complex-chain:
└─ Execute 1 command steps
   └─ echo start && echo middle | grep middle || echo fallback`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_NotFound(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create an action decorator that doesn't exist
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "nonexistent",
						Args: []decorators.Param{
							decorators.NewParam("message", "test"),
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "missing-action")

	got := executionPlan.StringNoColor()
	want := `missing-action:
└─ Execute 1 command steps
   └─ Unknown action: @nonexistent`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_MissingContent(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Create a shell element without content (should not happen in new code)
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind:    ir.ElementKindShell,
						Content: nil, // Missing content
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "missing-content")

	got := executionPlan.StringNoColor()
	want := `missing-content:
└─ Execute 1 command steps
   └─ <missing structured content>`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

// ======================================================================================
// DECORATOR TESTS - Testing all decorator types in plan mode
// ======================================================================================

func TestPlanGenerator_ActionDecorator_Cmd(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @cmd(build) action decorator
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "cmd",
						Args: []decorators.Param{
							decorators.NewPositionalParam("build"),
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "cmd-test")

	got := executionPlan.StringNoColor()
	want := `cmd-test:
└─ Execute 1 command steps
   └─ @cmd(build)`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}

	// Verify new expansion hint metadata format
	require.Len(t, executionPlan.Steps, 1)
	mainStep := executionPlan.Steps[0]
	require.Len(t, mainStep.Children, 1)
	cmdStep := mainStep.Children[0]
	assert.Equal(t, "command_reference", cmdStep.Metadata["expansion_type"])
	assert.Equal(t, "build", cmdStep.Metadata["command_name"])
}

func TestPlanGenerator_ActionDecorator_Log(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @log("Starting build") action decorator
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							decorators.NewPositionalParam("Starting build"),
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-test")

	got := executionPlan.StringNoColor()
	want := `log-test:
└─ Execute 1 command steps
   └─ Log: [INFO] Starting build`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_LogMultiline(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @log("Line 1\nLine 2\nLine 3") - multiline message should be truncated
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "Line 1\nLine 2\nLine 3"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-multiline-test")

	got := executionPlan.StringNoColor()
	want := `log-multiline-test:
└─ Execute 1 command steps
   └─ Log: [INFO] Line 1 ...`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_LogWithLevel(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @log("Error occurred", level="error") action decorator
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "Error occurred"},
							{ParamName: "level", ParamValue: "error"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-error-test")

	got := executionPlan.StringNoColor()
	want := `log-error-test:
└─ Execute 1 command steps
   └─ Log: [ERROR] Error occurred`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_LogPlain(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @log("Simple message", plain=true) action decorator
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "Simple message"},
							{ParamName: "plain", ParamValue: true},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-plain-test")

	got := executionPlan.StringNoColor()
	want := `log-plain-test:
└─ Execute 1 command steps
   └─ Log (plain): Simple message`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_MultipleLogs(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test multiple @log calls in sequence (multiple steps)
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "First message"},
						},
					},
				},
			},
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "Second message"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "multiple-logs-test")

	got := executionPlan.StringNoColor()
	want := `multiple-logs-test:
└─ Execute 2 command steps
   ├─ Log: [INFO] First message
   └─ Log: [INFO] Second message`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_LogInChain(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @log("Starting") && echo "test" && @log("Done") - logs in same step chain
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "Starting"},
						},
						OpNext: ir.ChainOpAnd,
					},
					{
						Kind: ir.ElementKindShell,
						Content: &ir.ElementContent{
							Parts: []ir.ContentPart{
								{Kind: ir.PartKindLiteral, Text: "echo test"},
							},
						},
						OpNext: ir.ChainOpAnd,
					},
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "Done"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-chain-test")

	got := executionPlan.StringNoColor()
	want := `log-chain-test:
└─ Execute 1 command steps
   └─ @log(Starting) && echo test && @log(Done)`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_LogWithColorTags(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @log("{green}Success!{/green}") - color tags should be removed in plan
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "{green}Success!{/green}"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-color-test")

	got := executionPlan.StringNoColor()
	want := `log-color-test:
└─ Execute 1 command steps
   └─ Log: [INFO] Success!`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_ActionDecorator_LogLongMessage(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test a very long log message that should be truncated
	longMessage := "This is a very long log message that should definitely be truncated when displayed in the plan because it exceeds the reasonable length limit"
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "log",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: longMessage},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "log-long-test")

	got := executionPlan.StringNoColor()
	// Should be truncated to 77 chars + "..."
	// "This is a very long log message that should definitely be truncat" (67 chars) + "..."
	want := `log-long-test:
└─ Execute 1 command steps
   └─ Log: [INFO] This is a very long log message that should definitely be truncat...`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_BlockDecorator_Timeout(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @timeout(30s) { echo test }
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindBlock,
						Name: "timeout",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "30s"},
						},
						InnerSteps: []ir.CommandStep{
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindShell,
										Content: &ir.ElementContent{
											Parts: []ir.ContentPart{
												{Kind: ir.PartKindLiteral, Text: "echo test"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "timeout-test")

	got := executionPlan.StringNoColor()
	want := `timeout-test:
└─ Execute 1 command steps
   └─ @timeout {30s timeout}
      └─ Inner commands
         └─ echo test`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_BlockDecorator_Parallel(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	tests := []struct {
		name        string
		seq         ir.CommandSeq
		commandName string
		want        string
	}{
		{
			name: "two tasks default concurrency",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task1"},
													},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task2"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "parallel-two",
			want: `parallel-two:
└─ Execute 1 command steps
   └─ @parallel {2 concurrent}
      └─ Inner commands
         ├─ echo task1
         └─ echo task2`,
		},
		{
			name: "single task",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo single"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "parallel-single",
			want: `parallel-single:
└─ Execute 1 command steps
   └─ @parallel {1 concurrent}
      └─ Inner commands
         └─ echo single`,
		},
		{
			name: "empty tasks",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind:       ir.ElementKindBlock,
								Name:       "parallel",
								Args:       []decorators.Param{},
								InnerSteps: []ir.CommandStep{},
							},
						},
					},
				},
			},
			commandName: "parallel-empty",
			want: `parallel-empty:
└─ Execute 1 command steps
   └─ @parallel {0 concurrent}
      └─ Inner commands`,
		},
		{
			name: "explicit concurrency limit",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{
									{ParamName: "concurrency", ParamValue: 3},
								},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task1"},
													},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task2"},
													},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task3"},
													},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task4"},
													},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task5"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "parallel-limited",
			want: `parallel-limited:
└─ Execute 1 command steps
   └─ @parallel {3 concurrent}
      └─ Inner commands
         ├─ echo task1
         ├─ echo task2
         ├─ echo task3
         ├─ echo task4
         └─ echo task5`,
		},
		{
			name: "concurrency higher than task count",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{
									{ParamName: "concurrency", ParamValue: "5"},
								},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task1"},
													},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "echo task2"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "parallel-over-limit",
			want: `parallel-over-limit:
└─ Execute 1 command steps
   └─ @parallel {2 concurrent}
      └─ Inner commands
         ├─ echo task1
         └─ echo task2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext()
			executionPlan := generator.GenerateFromIR(ctx, tt.seq, tt.commandName)

			got := executionPlan.StringNoColor()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanGenerator_BlockDecorator_ParallelWithActionDecorators(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	tests := []struct {
		name        string
		seq         ir.CommandSeq
		commandName string
		want        string
	}{
		{
			name: "parallel with cmd decorators only",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "cmd",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "core-deps"},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "cmd",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "runtime-deps"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "setup-parallel-cmds",
			want: `setup-parallel-cmds:
└─ Execute 1 command steps
   └─ @parallel {2 concurrent}
      └─ Inner commands
         ├─ @cmd(core-deps)
         └─ @cmd(runtime-deps)`,
		},
		{
			name: "parallel with mixed cmd and shell commands",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "cmd",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "build"},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindShell,
												Content: &ir.ElementContent{
													Parts: []ir.ContentPart{
														{Kind: ir.PartKindLiteral, Text: "go work sync"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "setup-mixed",
			want: `setup-mixed:
└─ Execute 1 command steps
   └─ @parallel {2 concurrent}
      └─ Inner commands
         ├─ @cmd(build)
         └─ go work sync`,
		},
		{
			name: "parallel with log and cmd decorators",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{
									{ParamName: "mode", ParamValue: "all"},
								},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "log",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "Starting task 1"},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "cmd",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "test-all"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			commandName: "parallel-with-log",
			want: `parallel-with-log:
└─ Execute 1 command steps
   └─ @parallel {2 concurrent}
      └─ Inner commands
         ├─ Log: [INFO] Starting task 1
         └─ @cmd(test-all)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext()
			executionPlan := generator.GenerateFromIR(ctx, tt.seq, tt.commandName)

			got := executionPlan.StringNoColor()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanGenerator_BlockDecorator_Retry(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @retry(3) { echo flaky }
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindBlock,
						Name: "retry",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "3"},
						},
						InnerSteps: []ir.CommandStep{
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindShell,
										Content: &ir.ElementContent{
											Parts: []ir.ContentPart{
												{Kind: ir.PartKindLiteral, Text: "echo flaky"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "retry-test")

	got := executionPlan.StringNoColor()
	want := `retry-test:
└─ Execute 1 command steps
   └─ @retry {3 attempts, 1s delay}
      └─ Inner commands
         └─ echo flaky`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_BlockDecorator_Workdir(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test @workdir("/tmp") { echo pwd }
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindBlock,
						Name: "workdir",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "/tmp"},
						},
						InnerSteps: []ir.CommandStep{
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindShell,
										Content: &ir.ElementContent{
											Parts: []ir.ContentPart{
												{Kind: ir.PartKindLiteral, Text: "echo pwd"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "workdir-test")

	// The exact output depends on the workdir decorator implementation
	// but we verify it generates a plan without errors
	require.Len(t, executionPlan.Steps, 1)
	mainStep := executionPlan.Steps[0]
	require.Len(t, mainStep.Children, 1)
	assert.Contains(t, executionPlan.StringNoColor(), "workdir-test")
}

func TestPlanGenerator_PatternDecorator_Try(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test try-catch pattern with wrapper format
	wrapper := ir.Wrapper{
		Kind:   "try",
		Params: map[string]interface{}{},
		Inner: ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{
							Kind: ir.ElementKindShell,
							Content: &ir.ElementContent{
								Parts: []ir.ContentPart{
									{Kind: ir.PartKindLiteral, Text: "risky command"},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, wrapper, "try-test")

	// Verify it generates a plan without errors
	require.Len(t, executionPlan.Steps, 1)
	assert.Contains(t, executionPlan.StringNoColor(), "try-test")
}

func TestPlanGenerator_PatternDecorator_When(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test when pattern with wrapper format
	wrapper := ir.Wrapper{
		Kind: "when",
		Params: map[string]interface{}{
			"var": "BUILD_TYPE",
		},
		Inner: ir.CommandSeq{
			Steps: []ir.CommandStep{
				{
					Chain: []ir.ChainElement{
						{
							Kind: ir.ElementKindShell,
							Content: &ir.ElementContent{
								Parts: []ir.ContentPart{
									{Kind: ir.PartKindLiteral, Text: "echo conditional"},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	ctx.Vars = map[string]string{"BUILD_TYPE": "release"}
	executionPlan := generator.GenerateFromIR(ctx, wrapper, "when-test")

	// Verify it generates a plan without errors
	require.Len(t, executionPlan.Steps, 1)
	assert.Contains(t, executionPlan.StringNoColor(), "when-test")
}

func TestPlanGenerator_MultipleDecorators(t *testing.T) {
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	// Test complex scenario: @timeout(30s) { @parallel { @log("task1"); @log("task2") } }
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindBlock,
						Name: "timeout",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "30s"},
						},
						InnerSteps: []ir.CommandStep{
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindBlock,
										Name: "parallel",
										Args: []decorators.Param{},
										InnerSteps: []ir.CommandStep{
											{
												Chain: []ir.ChainElement{
													{
														Kind: ir.ElementKindAction,
														Name: "log",
														Args: []decorators.Param{
															{ParamName: "", ParamValue: "task1"},
														},
													},
												},
											},
											{
												Chain: []ir.ChainElement{
													{
														Kind: ir.ElementKindAction,
														Name: "log",
														Args: []decorators.Param{
															{ParamName: "", ParamValue: "task2"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generator.GenerateFromIR(ctx, seq, "nested-test")

	// Verify nested structure generates successfully
	require.Len(t, executionPlan.Steps, 1)
	assert.Contains(t, executionPlan.StringNoColor(), "nested-test")

	// Check that both timeout and parallel are mentioned in the output
	output := executionPlan.StringNoColor()
	assert.Contains(t, output, "timeout")
	assert.Contains(t, output, "parallel")
}

func TestPlanGenerator_CommandCallingParallel(t *testing.T) {
	// This tests the critical case: @cmd(setup) where setup contains @parallel
	registry := decorators.GlobalRegistry()

	// Create a mock command resolver that returns a parallel command
	mockResolver := &MockCommandResolver{
		commands: map[string]ir.Node{
			"core-deps": ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "go mod download"},
									},
								},
							},
						},
					},
				},
			},
			"runtime-deps": ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "npm install"},
									},
								},
							},
						},
					},
				},
			},
			"setup": ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindAction,
								Name: "log",
								Args: []decorators.Param{
									{ParamName: "", ParamValue: "Setting up development environment..."},
								},
							},
						},
					},
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindBlock,
								Name: "parallel",
								Args: []decorators.Param{},
								InnerSteps: []ir.CommandStep{
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "cmd",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "core-deps"},
												},
											},
										},
									},
									{
										Chain: []ir.ChainElement{
											{
												Kind: ir.ElementKindAction,
												Name: "cmd",
												Args: []decorators.Param{
													{ParamName: "", ParamValue: "runtime-deps"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	generatorWithResolver := NewGeneratorWithResolver(registry, mockResolver)

	// Test calling @cmd(setup) which contains @parallel with @cmd decorators
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "cmd",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "setup"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generatorWithResolver.GenerateFromIR(ctx, seq, "main")

	got := executionPlan.StringNoColor()
	want := `main:
└─ Execute 1 command steps
   └─ @cmd(setup)
      ├─ Log: [INFO] Setting up development environment...
      └─ @parallel {2 concurrent}
         └─ Inner commands
            ├─ @cmd(core-deps)
            │  └─ go mod download
            └─ @cmd(runtime-deps)
               └─ npm install`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanGenerator_CommandCallingPatternDecorator(t *testing.T) {
	// This tests the critical missing case: @cmd(deploy) where deploy contains @try pattern decorator
	registry := decorators.GlobalRegistry()

	// Create a mock command resolver that includes pattern decorators
	mockResolver := &MockCommandResolver{
		commands: map[string]ir.Node{
			"backup-db": ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "pg_dump mydb > backup.sql"},
									},
								},
							},
						},
					},
				},
			},
			"cleanup": ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "rm -f backup.sql"},
									},
								},
							},
						},
					},
				},
			},
			"deploy": ir.Pattern{
				Kind:   "try",
				Params: map[string]interface{}{},
				Branches: map[string]ir.CommandSeq{
					"main": {
						Steps: []ir.CommandStep{
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindAction,
										Name: "cmd",
										Args: []decorators.Param{
											{ParamName: "", ParamValue: "backup-db"},
										},
									},
								},
							},
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindShell,
										Content: &ir.ElementContent{
											Parts: []ir.ContentPart{
												{Kind: ir.PartKindLiteral, Text: "kubectl apply -f k8s/"},
											},
										},
									},
								},
							},
						},
					},
					"catch": {
						Steps: []ir.CommandStep{
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindAction,
										Name: "log",
										Args: []decorators.Param{
											{ParamName: "", ParamValue: "Deployment failed, cleaning up"},
										},
									},
								},
							},
							{
								Chain: []ir.ChainElement{
									{
										Kind: ir.ElementKindAction,
										Name: "cmd",
										Args: []decorators.Param{
											{ParamName: "", ParamValue: "cleanup"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	generatorWithResolver := NewGeneratorWithResolver(registry, mockResolver)

	// Test calling @cmd(deploy) which contains @try pattern decorator with nested @cmd decorators
	seq := ir.CommandSeq{
		Steps: []ir.CommandStep{
			{
				Chain: []ir.ChainElement{
					{
						Kind: ir.ElementKindAction,
						Name: "cmd",
						Args: []decorators.Param{
							{ParamName: "", ParamValue: "deploy"},
						},
					},
				},
			},
		},
	}

	ctx := createTestContext()
	executionPlan := generatorWithResolver.GenerateFromIR(ctx, seq, "main")

	got := executionPlan.StringNoColor()
	// Note: @try pattern decorators evaluate conditions and show the selected branch
	// This test assumes the @try decorator shows both branches in dry-run mode
	want := `main:
└─ Execute 1 command steps
   └─ @cmd(deploy)
      └─ @try
         ├─ main: Execute 2 command steps
         │  ├─ @cmd(backup-db)
         │  │  └─ pg_dump mydb > backup.sql
         │  └─ kubectl apply -f k8s/
         └─ catch (optional): Execute 2 command steps
            ├─ Log: [INFO] Deployment failed, cleaning up
            └─ @cmd(cleanup)
               └─ rm -f backup.sql`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Plan output mismatch (-want +got):\n%s", diff)
	}
}

// ======================================================================================
// SHELL CHAINING TESTS - Testing correct spec behavior for shell operators
// ======================================================================================

func TestPlanGenerator_ShellChaining_SpecBehavior(t *testing.T) {
	// According to spec: shell chains should be displayed as single commands, not trees
	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	tests := []struct {
		name     string
		chain    []ir.ChainElement
		expected string
	}{
		{
			name: "simple AND chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo world"},
						},
					},
				},
			},
			expected: `simple-AND-chain:
└─ Execute 1 command steps
   └─ echo hello && echo world`,
		},
		{
			name: "simple OR chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpOr,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo fallback"},
						},
					},
				},
			},
			expected: `simple-OR-chain:
└─ Execute 1 command steps
   └─ echo hello || echo fallback`,
		},
		{
			name: "simple PIPE chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpPipe,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "grep hello"},
						},
					},
				},
			},
			expected: `simple-PIPE-chain:
└─ Execute 1 command steps
   └─ echo hello | grep hello`,
		},
		{
			name: "complex multi-operator chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo start"},
						},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo middle"},
						},
					},
					OpNext: ir.ChainOpPipe,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "grep middle"},
						},
					},
					OpNext: ir.ChainOpOr,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo fallback"},
						},
					},
				},
			},
			expected: `complex-multi-operator-chain:
└─ Execute 1 command steps
   └─ echo start && echo middle | grep middle || echo fallback`,
		},
		{
			name: "append operator chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpAppend,
					Target: "output.txt",
				},
			},
			expected: `append-operator-chain:
└─ Execute 1 command steps
   └─ echo hello >> output.txt`,
		},
		{
			name: "action decorators in chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindAction,
					Name: "log",
					Args: []decorators.Param{
						{ParamName: "", ParamValue: "starting"},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo middle"},
						},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindAction,
					Name: "log",
					Args: []decorators.Param{
						{ParamName: "", ParamValue: "done"},
					},
				},
			},
			expected: `action-decorators-in-chain:
└─ Execute 1 command steps
   └─ @log(starting) && echo middle && @log(done)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := ir.CommandSeq{
				Steps: []ir.CommandStep{
					{Chain: tt.chain},
				},
			}

			ctx := createTestContext()
			executionPlan := generator.GenerateFromIR(ctx, seq, strings.ReplaceAll(tt.name, " ", "-"))

			got := executionPlan.StringNoColor()
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("SPEC VIOLATION: Shell chains should be single commands, not trees (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanGenerator_MultipleStepsVsChains_CorrectSpec(t *testing.T) {
	// This test demonstrates the difference between:
	// 1. Multiple command steps (newline-separated) - should be separate
	// 2. Single command step with chains (shell operators) - should be single

	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	tests := []struct {
		name     string
		seq      ir.CommandSeq
		expected string
	}{
		{
			name: "multiple command steps (newline-separated)",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step1"},
									},
								},
							},
						},
					},
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step2"},
									},
								},
							},
						},
					},
				},
			},
			expected: `multi-step:
└─ Execute 2 command steps
   ├─ echo step1
   └─ echo step2`,
		},
		{
			name: "single command step with chain (shell operators)",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step1"},
									},
								},
								OpNext: ir.ChainOpAnd,
							},
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step2"},
									},
								},
							},
						},
					},
				},
			},
			expected: `chain-step:
└─ Execute 1 command steps
   └─ echo step1 && echo step2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext()
			var commandName string
			if strings.Contains(tt.name, "multiple") {
				commandName = "multi-step"
			} else {
				commandName = "chain-step"
			}

			executionPlan := generator.GenerateFromIR(ctx, tt.seq, commandName)
			got := executionPlan.StringNoColor()

			if diff := cmp.Diff(tt.expected, got); diff != "" {
				if strings.Contains(tt.name, "multiple") {
					t.Errorf("Multiple command steps should be separate (-want +got):\n%s", diff)
				} else {
					t.Errorf("SPEC VIOLATION: Single chain step should be single command (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// MockCommandResolver for testing command resolution
type MockCommandResolver struct {
	commands map[string]ir.Node
}

func (m *MockCommandResolver) GetCommand(name string) (ir.Node, error) {
	if cmd, exists := m.commands[name]; exists {
		return cmd, nil
	}
	return nil, fmt.Errorf("command not found: %s", name)
}

// ======================================================================================
// SHELL CHAINING TESTS - Testing correct spec behavior (Table-driven)
// ======================================================================================

func TestPlanGenerator_ShellChaining_CorrectSpecBehavior(t *testing.T) {
	// According to spec: shell chains should be displayed as single commands
	// NOT as separate tree elements

	tests := []struct {
		name     string
		chain    []ir.ChainElement
		expected string // Expected single command output per spec
	}{
		{
			name: "simple AND chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo world"},
						},
					},
				},
			},
			expected: "echo hello && echo world",
		},
		{
			name: "simple OR chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpOr,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo fallback"},
						},
					},
				},
			},
			expected: "echo hello || echo fallback",
		},
		{
			name: "simple PIPE chain",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpPipe,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "grep hello"},
						},
					},
				},
			},
			expected: "echo hello | grep hello",
		},
		{
			name: "APPEND operation",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo hello"},
						},
					},
					OpNext: ir.ChainOpAppend,
					Target: "output.txt",
				},
			},
			expected: "echo hello >> output.txt",
		},
		{
			name: "complex chain with multiple operators",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo start"},
						},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo middle"},
						},
					},
					OpNext: ir.ChainOpPipe,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "grep middle"},
						},
					},
					OpNext: ir.ChainOpOr,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo fallback"},
						},
					},
				},
			},
			expected: "echo start && echo middle | grep middle || echo fallback",
		},
		{
			name: "chain with action decorators",
			chain: []ir.ChainElement{
				{
					Kind: ir.ElementKindAction,
					Name: "log",
					Args: []decorators.Param{
						{ParamName: "", ParamValue: "starting"},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindShell,
					Content: &ir.ElementContent{
						Parts: []ir.ContentPart{
							{Kind: ir.PartKindLiteral, Text: "echo middle"},
						},
					},
					OpNext: ir.ChainOpAnd,
				},
				{
					Kind: ir.ElementKindAction,
					Name: "log",
					Args: []decorators.Param{
						{ParamName: "", ParamValue: "done"},
					},
				},
			},
			expected: "@log(starting) && echo middle && @log(done)",
		},
	}

	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := ir.CommandSeq{
				Steps: []ir.CommandStep{
					{Chain: tt.chain},
				},
			}

			ctx := createTestContext()
			executionPlan := generator.GenerateFromIR(ctx, seq, "test-command")

			got := executionPlan.StringNoColor()
			want := fmt.Sprintf(`test-command:
└─ Execute 1 command steps
   └─ %s`, tt.expected)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("SPEC VIOLATION: Shell chains should be single commands (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanGenerator_MultipleStepsVsChains(t *testing.T) {
	// Test the difference between multiple command steps (newlines) vs chains (operators)

	tests := []struct {
		name        string
		seq         ir.CommandSeq
		commandName string
		expected    string
	}{
		{
			name: "multiple command steps (newline-separated) should be separate",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step1"},
									},
								},
							},
						},
					},
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step2"},
									},
								},
							},
						},
					},
				},
			},
			commandName: "multi-step",
			expected: `multi-step:
└─ Execute 2 command steps
   ├─ echo step1
   └─ echo step2`,
		},
		{
			name: "single command step with chain (shell operators) should be single",
			seq: ir.CommandSeq{
				Steps: []ir.CommandStep{
					{
						Chain: []ir.ChainElement{
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step1"},
									},
								},
								OpNext: ir.ChainOpAnd,
							},
							{
								Kind: ir.ElementKindShell,
								Content: &ir.ElementContent{
									Parts: []ir.ContentPart{
										{Kind: ir.PartKindLiteral, Text: "echo step2"},
									},
								},
							},
						},
					},
				},
			},
			commandName: "chain-step",
			expected: `chain-step:
└─ Execute 1 command steps
   └─ echo step1 && echo step2`,
		},
	}

	registry := decorators.GlobalRegistry()
	generator := NewGenerator(registry)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext()
			executionPlan := generator.GenerateFromIR(ctx, tt.seq, tt.commandName)

			got := executionPlan.StringNoColor()

			if diff := cmp.Diff(tt.expected, got); diff != "" {
				if tt.name == "single command step with chain (shell operators) should be single" {
					t.Errorf("SPEC VIOLATION: Chain steps should be single commands (-want +got):\n%s", diff)
				} else {
					t.Errorf("Multiple steps behavior incorrect (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// Helper function to create a test context
func createTestContext() *context.Ctx {
	return &context.Ctx{
		DryRun: true,
		Vars:   make(map[string]string),
		Env: ir.EnvSnapshot{
			Values: make(map[string]string),
		},
		SysInfo: context.SystemInfoSnapshot{
			NumCPU:   4, // Deterministic value for tests
			MemoryMB: 8192,
			OS:       "linux",
			Arch:     "amd64",
			Hostname: "test-host",
			TempDir:  "/tmp",
			HomeDir:  "/home/test",
			UserName: "test",
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}
