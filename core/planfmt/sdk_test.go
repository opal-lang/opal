package planfmt

import (
	"testing"

	"github.com/aledsdavies/opal/core/sdk"
	"github.com/google/go-cmp/cmp"
)

// TestToSDKArgs tests argument conversion
func TestToSDKArgs(t *testing.T) {
	tests := []struct {
		name     string
		planArgs []Arg
		want     map[string]interface{}
	}{
		{
			name:     "empty args",
			planArgs: []Arg{},
			want:     map[string]interface{}{},
		},
		{
			name: "string arg",
			planArgs: []Arg{
				{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}},
			},
			want: map[string]interface{}{
				"command": "echo hello",
			},
		},
		{
			name: "int arg",
			planArgs: []Arg{
				{Key: "times", Val: Value{Kind: ValueInt, Int: 3}},
			},
			want: map[string]interface{}{
				"times": int64(3),
			},
		},
		{
			name: "bool arg",
			planArgs: []Arg{
				{Key: "enabled", Val: Value{Kind: ValueBool, Bool: true}},
			},
			want: map[string]interface{}{
				"enabled": true,
			},
		},
		{
			name: "mixed args",
			planArgs: []Arg{
				{Key: "command", Val: Value{Kind: ValueString, Str: "npm test"}},
				{Key: "retries", Val: Value{Kind: ValueInt, Int: 5}},
				{Key: "verbose", Val: Value{Kind: ValueBool, Bool: false}},
			},
			want: map[string]interface{}{
				"command": "npm test",
				"retries": int64(5),
				"verbose": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSDKArgs(tt.planArgs)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ToSDKArgs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestToSDKStep_CommandNode tests single command node conversion
func TestToSDKStep_CommandNode(t *testing.T) {
	planStep := Step{
		ID: 42,
		Tree: &CommandNode{
			Decorator: "shell",
			Args: []Arg{
				{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}},
			},
		},
	}

	got := ToSDKStep(planStep)

	if got.ID != 42 {
		t.Errorf("expected ID 42, got %d", got.ID)
	}
	if got.Tree == nil {
		t.Fatal("expected tree to be non-nil")
	}
	cmd, ok := got.Tree.(*sdk.CommandNode)
	if !ok {
		t.Fatalf("expected *sdk.CommandNode, got %T", got.Tree)
	}
	if cmd.Name != "shell" {
		t.Errorf("expected name 'shell', got %q", cmd.Name)
	}
	if cmd.Args["command"] != "echo hello" {
		t.Errorf("expected command 'echo hello', got %v", cmd.Args["command"])
	}
}

// TestToSDKStep_AndNode tests AND node conversion
func TestToSDKStep_AndNode(t *testing.T) {
	planStep := Step{
		ID: 1,
		Tree: &AndNode{
			Left: &CommandNode{
				Decorator: "shell",
				Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "npm build"}}},
			},
			Right: &CommandNode{
				Decorator: "shell",
				Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "docker build"}}},
			},
		},
	}

	got := ToSDKStep(planStep)

	andNode, ok := got.Tree.(*sdk.AndNode)
	if !ok {
		t.Fatalf("expected *sdk.AndNode, got %T", got.Tree)
	}
	if andNode.Left == nil {
		t.Error("expected Left to be non-nil")
	}
	if andNode.Right == nil {
		t.Error("expected Right to be non-nil")
	}
}

// TestToSDKStep_PipelineNode tests pipeline node conversion
func TestToSDKStep_PipelineNode(t *testing.T) {
	planStep := Step{
		ID: 1,
		Tree: &PipelineNode{
			Commands: []ExecutionNode{
				&CommandNode{
					Decorator: "shell",
					Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}}},
				},
				&CommandNode{
					Decorator: "shell",
					Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "grep hello"}}},
				},
			},
		},
	}

	got := ToSDKStep(planStep)

	pipeNode, ok := got.Tree.(*sdk.PipelineNode)
	if !ok {
		t.Fatalf("expected *sdk.PipelineNode, got %T", got.Tree)
	}
	if len(pipeNode.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(pipeNode.Commands))
	}
}

// TestToSDKStep_WithBlock tests command with nested block
func TestToSDKStep_WithBlock(t *testing.T) {
	planStep := Step{
		ID: 1,
		Tree: &CommandNode{
			Decorator: "retry",
			Args:      []Arg{{Key: "times", Val: Value{Kind: ValueInt, Int: 3}}},
			Block: []Step{
				{
					ID: 2,
					Tree: &CommandNode{
						Decorator: "shell",
						Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "npm test"}}},
					},
				},
			},
		},
	}

	got := ToSDKStep(planStep)

	cmd, ok := got.Tree.(*sdk.CommandNode)
	if !ok {
		t.Fatalf("expected *sdk.CommandNode, got %T", got.Tree)
	}
	if cmd.Name != "retry" {
		t.Errorf("expected name 'retry', got %q", cmd.Name)
	}
	if len(cmd.Block) != 1 {
		t.Fatalf("expected 1 block step, got %d", len(cmd.Block))
	}
	if cmd.Block[0].ID != 2 {
		t.Errorf("expected block step ID 2, got %d", cmd.Block[0].ID)
	}
}

// TestToSDKSteps tests multiple steps conversion
func TestToSDKSteps(t *testing.T) {
	planSteps := []Step{
		{
			ID: 1,
			Tree: &CommandNode{
				Decorator: "shell",
				Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo first"}}},
			},
		},
		{
			ID: 2,
			Tree: &CommandNode{
				Decorator: "shell",
				Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo second"}}},
			},
		},
	}

	got := ToSDKSteps(planSteps)

	if len(got) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(got))
	}
	if got[0].ID != 1 {
		t.Errorf("expected step 0 ID 1, got %d", got[0].ID)
	}
	if got[1].ID != 2 {
		t.Errorf("expected step 1 ID 2, got %d", got[1].ID)
	}
}

// TestToSDKSteps_ComplexTree tests complex operator tree conversion
func TestToSDKSteps_ComplexTree(t *testing.T) {
	// echo "a" | grep "a" && echo "b" || echo "c"
	// Parsed as: ((echo "a" | grep "a") && echo "b") || echo "c"
	planSteps := []Step{
		{
			ID: 1,
			Tree: &OrNode{
				Left: &AndNode{
					Left: &PipelineNode{
						Commands: []ExecutionNode{
							&CommandNode{Decorator: "shell", Args: []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo a"}}}},
							&CommandNode{Decorator: "shell", Args: []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "grep a"}}}},
						},
					},
					Right: &CommandNode{
						Decorator: "shell",
						Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo b"}}},
					},
				},
				Right: &CommandNode{
					Decorator: "shell",
					Args:      []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo c"}}},
				},
			},
		},
	}

	got := ToSDKSteps(planSteps)

	if len(got) != 1 {
		t.Fatalf("expected 1 step, got %d", len(got))
	}

	orNode, ok := got[0].Tree.(*sdk.OrNode)
	if !ok {
		t.Fatalf("expected *sdk.OrNode, got %T", got[0].Tree)
	}

	andNode, ok := orNode.Left.(*sdk.AndNode)
	if !ok {
		t.Fatalf("expected Left to be *sdk.AndNode, got %T", orNode.Left)
	}

	pipeNode, ok := andNode.Left.(*sdk.PipelineNode)
	if !ok {
		t.Fatalf("expected AndNode.Left to be *sdk.PipelineNode, got %T", andNode.Left)
	}

	if len(pipeNode.Commands) != 2 {
		t.Errorf("expected 2 commands in pipeline, got %d", len(pipeNode.Commands))
	}
}
