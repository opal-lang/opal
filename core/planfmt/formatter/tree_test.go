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
				Commands: []planfmt.Command{
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
						},
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
				Commands: []planfmt.Command{
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "go build"}},
						},
					},
				},
			},
			{
				ID: 2,
				Commands: []planfmt.Command{
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "go test"}},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	if !strings.Contains(output, "├─ shell go build") {
		t.Errorf("Expected first step with ├─, got:\n%s", output)
	}
	if !strings.Contains(output, "└─ shell go test") {
		t.Errorf("Expected last step with └─, got:\n%s", output)
	}
}

func TestFormatTree_WithOperators(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "go build"}},
						},
						Operator: "&&",
					},
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "go test"}},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	if !strings.Contains(output, "shell go build && go test") {
		t.Errorf("Expected operator chain, got:\n%s", output)
	}
}

func TestFormatTree_WithColor(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, true)

	output := buf.String()
	// Should contain ANSI color codes
	if !strings.Contains(output, ColorBlue) {
		t.Error("Expected color codes in output")
	}
	if !strings.Contains(output, ColorReset) {
		t.Error("Expected color reset in output")
	}
}

func TestFormatTree_WithoutColor(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "shell",
						Args: []planfmt.Arg{
							{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	FormatTree(&buf, plan, false)

	output := buf.String()
	// Should NOT contain ANSI color codes
	if strings.Contains(output, ColorBlue) {
		t.Error("Expected no color codes in output")
	}
}

func TestColorize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		color    string
		useColor bool
		want     string
	}{
		{
			name:     "with color enabled",
			text:     "hello",
			color:    ColorBlue,
			useColor: true,
			want:     ColorBlue + "hello" + ColorReset,
		},
		{
			name:     "with color disabled",
			text:     "hello",
			color:    ColorBlue,
			useColor: false,
			want:     "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Colorize(tt.text, tt.color, tt.useColor)
			if got != tt.want {
				t.Errorf("Colorize() = %q, want %q", got, tt.want)
			}
		})
	}
}
