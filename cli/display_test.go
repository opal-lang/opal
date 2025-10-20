package main

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/stretchr/testify/assert"
)

// Helper to create a simple shell command tree
func shellCmd(cmd string) *planfmt.CommandNode {
	return &planfmt.CommandNode{
		Decorator: "@shell",
		Args: []planfmt.Arg{
			{
				Key: "command",
				Val: planfmt.Value{
					Kind: planfmt.ValueString,
					Str:  cmd,
				},
			},
		},
	}
}

func TestDisplayPlan_SingleStep(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd(`echo "Hello, World!"`),
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
				ID:   1,
				Tree: shellCmd("kubectl apply -f k8s/"),
			},
			{
				ID:   2,
				Tree: shellCmd("kubectl rollout status deployment/app"),
			},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false)

	output := buf.String()
	expected := `deploy:
├─ @shell kubectl apply -f k8s/
└─ @shell kubectl rollout status deployment/app
`

	assert.Equal(t, expected, output)
}

func TestDisplayPlan_WithSecrets(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "deploy",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo $SECRET"),
			},
		},
		Secrets: []planfmt.Secret{
			{Key: "SECRET", DisplayID: "opal:s:abc123", RuntimeValue: "secret123"},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false)

	output := buf.String()
	expected := `deploy:
└─ @shell echo $SECRET
`

	assert.Equal(t, expected, output)
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

func TestDisplayPlan_NestedDecorator(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "retry",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@retry",
					Args: []planfmt.Arg{
						{
							Key: "max",
							Val: planfmt.Value{
								Kind: planfmt.ValueInt,
								Int:  3,
							},
						},
					},
					Block: []planfmt.Step{
						{
							ID:   2,
							Tree: shellCmd("kubectl apply -f k8s/"),
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	DisplayPlan(&buf, plan, false)

	output := buf.String()
	expected := `retry:
└─ @retry max=3
   └─ @shell kubectl apply -f k8s/
`

	assert.Equal(t, expected, output)
}
