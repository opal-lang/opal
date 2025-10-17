package main

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/stretchr/testify/assert"
)

func TestDisplayPlan_SingleStep(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  `echo "Hello, World!"`,
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false) // no color

	output := buf.String()
	expected := `hello:
└─ @shell echo "Hello, World!"
`

	assert.Equal(t, expected, output)
}

func TestDisplayPlan_MultipleSteps(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "deploy",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  "kubectl apply -f k8s/",
								},
							},
						},
					},
				},
			},
			{
				ID: 2,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  "kubectl rollout status deployment/app",
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false) // no color

	output := buf.String()
	expected := `deploy:
├─ @shell kubectl apply -f k8s/
└─ @shell kubectl rollout status deployment/app
`

	assert.Equal(t, expected, output)
}

func TestDisplayPlan_WithOperators(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  `echo "First"`,
								},
							},
						},
						Operator: "&&",
					},
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  `echo "Second"`,
								},
							},
						},
						Operator: "||",
					},
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  `echo "Fallback"`,
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false) // no color

	output := buf.String()
	expected := `test:
└─ @shell echo "First" && echo "Second" || echo "Fallback"
`

	assert.Equal(t, expected, output)
}

func TestDisplayPlan_WithColor(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Commands: []planfmt.Command{
					{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  `echo "Hello"`,
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, true) // with color

	output := buf.String()

	// Should contain ANSI color codes
	assert.Contains(t, output, "\033[") // ANSI escape sequence
	assert.Contains(t, output, "@shell")
	assert.Contains(t, output, "hello:")
}

func TestDisplayPlan_EmptyPlan(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "empty",
		Steps:  []planfmt.Step{},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false)

	output := buf.String()
	expected := `empty:
(no steps)
`

	assert.Equal(t, expected, output)
}
