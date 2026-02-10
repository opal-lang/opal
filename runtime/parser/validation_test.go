package parser

import (
	"testing"
)

func TestValidateScriptMode(t *testing.T) {
	input := `var env = "production"

echo "Starting deployment"

fun deploy(service String) {
  kubectl apply -f deployment.yaml
}

deploy("api")`

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
		{
			name: "top-level function call rejected",
			input: `fun deploy(service String) {
	echo "hello"
}

deploy("api")`,
			shouldErr: true,
			reason:    "command mode is for definitions only, no top-level function call execution",
		},
		{
			name: "function call inside function definition allowed",
			input: `fun helper(service String) {
	echo "hello"
}

fun deploy(service String) {
	helper(@var.service)
}`,
			shouldErr: false,
			reason:    "command mode allows function calls inside definitions",
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

func TestValidateEnvInRemoteTransport(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
		reason    string
	}{
		{
			name:      "@env allowed at top level",
			input:     `var home = @env.HOME`,
			shouldErr: false,
			reason:    "@env is allowed outside transport-switching decorators",
		},
		{
			name: "@env allowed in non-transport decorator",
			input: `@retry(attempts=3) {
				var home = @env.HOME
			}`,
			shouldErr: false,
			reason:    "@retry doesn't switch transport, so @env is allowed",
		},
		{
			name: "shell variables allowed everywhere",
			input: `@ssh.connect(host="remote") {
				echo $HOME
			}`,
			shouldErr: false,
			reason:    "shell variables ($HOME) are always allowed",
		},
		// Note: These tests will pass validation because ssh.connect, docker.exec
		// are not registered decorators yet. When they are registered with
		// SwitchesTransport=true, these tests will start failing as expected.
		{
			name: "@env forbidden in @ssh.connect (when registered)",
			input: `@ssh.connect(host="remote") {
				var home = @env.HOME
			}`,
			shouldErr: false, // Will be true when ssh.connect is registered
			reason:    "@env resolves to local environment, confusing in remote context",
		},
		{
			name: "@env forbidden in @docker.exec (when registered)",
			input: `@docker.exec(container="app") {
				var user = @env.USER
			}`,
			shouldErr: false, // Will be true when docker.exec is registered
			reason:    "@env resolves to local environment, confusing in container context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))
			err := tree.Validate(ModeScript)

			if tt.shouldErr && err == nil {
				t.Errorf("expected error: %s", tt.reason)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v (reason: %s)", err, tt.reason)
			}
		})
	}
}
