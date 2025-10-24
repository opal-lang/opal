package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

func TestFormatTree_EmptyPlan(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps:  []planfmt.Step{},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	expected := "test:\n(no steps)\n"
	if buf.String() != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, buf.String())
	}
}

func TestFormatTree_SingleStep(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "deploy",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	if !strings.Contains(output, "deploy:") {
		t.Error("Expected target name 'deploy:'")
	}
	if !strings.Contains(output, "└─ shell echo hello") {
		t.Errorf("Expected tree output with shell command, got:\n%s", output)
	}
}

func TestFormatTree_MultipleSteps(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "build",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "go build"}},
					},
				},
			},
			{
				ID: 2,
				Tree: &planfmt.CommandNode{
					Decorator: "shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "go test"}},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	if !strings.Contains(output, "build:") {
		t.Error("Expected target name 'build:'")
	}
	// First step should use ├─ (not last)
	if !strings.Contains(output, "├─") {
		t.Error("Expected ├─ for first step")
	}
	// Last step should use └─
	if !strings.Contains(output, "└─") {
		t.Error("Expected └─ for last step")
	}
}

func TestFormatTree_WithOperators(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "deploy",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.AndNode{
					Left: &planfmt.CommandNode{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "npm build"}},
						},
					},
					Right: &planfmt.CommandNode{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "docker build"}},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	if !strings.Contains(output, "&&") {
		t.Errorf("Expected && operator in output, got:\n%s", output)
	}
	if !strings.Contains(output, "npm build") {
		t.Error("Expected first command")
	}
	if !strings.Contains(output, "docker build") {
		t.Error("Expected second command")
	}
}

func TestFormatTree_WithPipeline(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.PipelineNode{
					Commands: []planfmt.ExecutionNode{
						&planfmt.CommandNode{
							Decorator: "shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
							},
						},
						&planfmt.CommandNode{
							Decorator: "shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep hello"}},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	if !strings.Contains(output, "|") {
		t.Errorf("Expected | operator in output, got:\n%s", output)
	}
	if !strings.Contains(output, "echo hello") {
		t.Error("Expected first command")
	}
	if !strings.Contains(output, "grep hello") {
		t.Error("Expected second command")
	}
}

func TestFormatTree_WithColor(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, true) // Enable color

	output := buf.String()
	// Should contain ANSI color codes
	if !strings.Contains(output, "\033[") {
		t.Error("Expected ANSI color codes when color is enabled")
	}
}
