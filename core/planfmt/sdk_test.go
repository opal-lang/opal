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

// TestToSDKCommands tests command conversion
func TestToSDKCommands(t *testing.T) {
	tests := []struct {
		name     string
		planCmds []Command
		want     []sdk.Command
	}{
		{
			name:     "empty commands",
			planCmds: []Command{},
			want:     []sdk.Command{},
		},
		{
			name: "single command",
			planCmds: []Command{
				{
					Decorator: "shell",
					Args: []Arg{
						{Key: "command", Val: Value{Kind: ValueString, Str: "echo test"}},
					},
					Block:    []Step{},
					Operator: "",
				},
			},
			want: []sdk.Command{
				{
					Name: "shell",
					Args: map[string]interface{}{
						"command": "echo test",
					},
					Block:    []sdk.Step{},
					Operator: "",
				},
			},
		},
		{
			name: "command with block",
			planCmds: []Command{
				{
					Decorator: "retry",
					Args: []Arg{
						{Key: "times", Val: Value{Kind: ValueInt, Int: 3}},
					},
					Block: []Step{
						{
							ID: 100,
							Commands: []Command{
								{
									Decorator: "shell",
									Args: []Arg{
										{Key: "command", Val: Value{Kind: ValueString, Str: "npm test"}},
									},
								},
							},
						},
					},
					Operator: "",
				},
			},
			want: []sdk.Command{
				{
					Name: "retry",
					Args: map[string]interface{}{
						"times": int64(3),
					},
					Block: []sdk.Step{
						{
							ID: 100,
							Commands: []sdk.Command{
								{
									Name: "shell",
									Args: map[string]interface{}{
										"command": "npm test",
									},
									Block:    []sdk.Step{},
									Operator: "",
								},
							},
						},
					},
					Operator: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSDKCommands(tt.planCmds)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ToSDKCommands() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestToSDKStep tests single step conversion
func TestToSDKStep(t *testing.T) {
	planStep := Step{
		ID: 42,
		Commands: []Command{
			{
				Decorator: "shell",
				Args: []Arg{
					{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}},
				},
			},
		},
	}

	want := sdk.Step{
		ID: 42,
		Commands: []sdk.Command{
			{
				Name: "shell",
				Args: map[string]interface{}{
					"command": "echo hello",
				},
				Block:    []sdk.Step{},
				Operator: "",
			},
		},
	}

	got := ToSDKStep(planStep)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ToSDKStep() mismatch (-want +got):\n%s", diff)
	}
}

// TestToSDKSteps tests multiple steps conversion
func TestToSDKSteps(t *testing.T) {
	planSteps := []Step{
		{
			ID: 1,
			Commands: []Command{
				{
					Decorator: "shell",
					Args: []Arg{
						{Key: "command", Val: Value{Kind: ValueString, Str: "echo first"}},
					},
				},
			},
		},
		{
			ID: 2,
			Commands: []Command{
				{
					Decorator: "shell",
					Args: []Arg{
						{Key: "command", Val: Value{Kind: ValueString, Str: "echo second"}},
					},
				},
			},
		},
	}

	want := []sdk.Step{
		{
			ID: 1,
			Commands: []sdk.Command{
				{
					Name: "shell",
					Args: map[string]interface{}{
						"command": "echo first",
					},
					Block:    []sdk.Step{},
					Operator: "",
				},
			},
		},
		{
			ID: 2,
			Commands: []sdk.Command{
				{
					Name: "shell",
					Args: map[string]interface{}{
						"command": "echo second",
					},
					Block:    []sdk.Step{},
					Operator: "",
				},
			},
		},
	}

	got := ToSDKSteps(planSteps)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ToSDKSteps() mismatch (-want +got):\n%s", diff)
	}
}

// TestToSDKSteps_Nested tests nested block conversion
func TestToSDKSteps_Nested(t *testing.T) {
	planSteps := []Step{
		{
			ID: 1,
			Commands: []Command{
				{
					Decorator: "retry",
					Args: []Arg{
						{Key: "times", Val: Value{Kind: ValueInt, Int: 3}},
					},
					Block: []Step{
						{
							ID: 10,
							Commands: []Command{
								{
									Decorator: "shell",
									Args: []Arg{
										{Key: "command", Val: Value{Kind: ValueString, Str: "npm test"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	want := []sdk.Step{
		{
			ID: 1,
			Commands: []sdk.Command{
				{
					Name: "retry",
					Args: map[string]interface{}{
						"times": int64(3),
					},
					Block: []sdk.Step{
						{
							ID: 10,
							Commands: []sdk.Command{
								{
									Name: "shell",
									Args: map[string]interface{}{
										"command": "npm test",
									},
									Block:    []sdk.Step{},
									Operator: "",
								},
							},
						},
					},
					Operator: "",
				},
			},
		},
	}

	got := ToSDKSteps(planSteps)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ToSDKSteps() nested mismatch (-want +got):\n%s", diff)
	}
}
