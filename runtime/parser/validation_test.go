package parser

import (
	"testing"
)

func TestValidateScriptMode(t *testing.T) {
	input := `var env = "production"

echo "Starting deployment"

fun deploy(service) {
  kubectl apply -f deployment.yaml
}`

	tree := Parse([]byte(input))

	err := tree.Validate(ModeScript)
	if err != nil {
		t.Errorf("script mode should allow vars, functions, and shell commands: %v", err)
	}
}

func TestValidateCommandMode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
		reason    string
	}{
		{
			name:      "function definition allowed",
			input:     `fun deploy() { echo "hello" }`,
			shouldErr: false,
			reason:    "command mode allows function definitions (like just/make)",
		},
		{
			name:      "variable declaration allowed",
			input:     `var env = "prod"`,
			shouldErr: false,
			reason:    "command mode allows variable declarations for parameterization",
		},
		{
			name: "multiple definitions allowed",
			input: `var env = "prod"
fun deploy() { echo "hello" }
fun test() { echo "test" }`,
			shouldErr: false,
			reason:    "command mode allows multiple definitions",
		},
		{
			name:      "top-level shell command rejected",
			input:     `echo "hello"`,
			shouldErr: true,
			reason:    "command mode is for definitions only, no top-level execution",
		},
		{
			name:      "bare identifier rejected",
			input:     `deploy`,
			shouldErr: true,
			reason:    "command mode is for definitions only, no execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))
			err := tree.Validate(ModeCommand)

			if tt.shouldErr && err == nil {
				t.Errorf("expected error: %s", tt.reason)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v (reason: %s)", err, tt.reason)
			}
		})
	}
}
