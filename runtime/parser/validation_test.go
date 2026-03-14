package parser

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/types"
)

// mockTransportDecorator is a test decorator that simulates a transport-switching decorator.
// It has RoleBoundary role which makes isTransportSwitchingDecorator return true.
type mockTransportDecorator struct {
	path string
}

func (m *mockTransportDecorator) Descriptor() decorator.Descriptor {
	return decorator.Descriptor{
		Path: m.path,
		Schema: types.DecoratorSchema{
			Path: m.path,
			Kind: types.KindExecution,
		},
		Capabilities: decorator.Capabilities{
			Block: decorator.BlockRequired,
		},
	}
}

func (m *mockTransportDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	return parent, nil
}

func (m *mockTransportDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return next
}

func (m *mockTransportDecorator) MaterializeSession() bool {
	return false
}

func (m *mockTransportDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork | decorator.TransportCapEnvironment
}

func (m *mockTransportDecorator) IsolationContext() decorator.IsolationContext {
	return nil
}

// mockRootOnlyValueDecorator is a test decorator that simulates a root-only value decorator like @env.
// It has RoleProvider role and TransportScopeLocal capability.
type mockRootOnlyValueDecorator struct {
	path string
}

func (m *mockRootOnlyValueDecorator) Descriptor() decorator.Descriptor {
	return decorator.Descriptor{
		Path: m.path,
		Schema: types.DecoratorSchema{
			Path: m.path,
			Kind: types.KindValue,
		},
		Capabilities: decorator.Capabilities{
			TransportScope: decorator.TransportScopeLocal,
		},
	}
}

func (m *mockRootOnlyValueDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	for i := range calls {
		results[i] = decorator.ResolveResult{Value: "mock-value"}
	}
	return results, nil
}

func init() {
	// Register test decorators that simulate @ssh.connect and @env behavior
	decorator.Register("test.ssh", &mockTransportDecorator{path: "test.ssh"})
	decorator.Register("test.env", &mockRootOnlyValueDecorator{path: "test.env"})
}

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
			name: "@exec.retry allowed in non-transport decorator",
			input: `@exec.retry(times=3) {
				var home = @env.HOME
			}`,
			shouldErr: false,
			reason:    "@exec.retry doesn't switch transport, so @env is allowed",
		},
		{
			name: "shell variables allowed everywhere",
			input: `@test.ssh(host="remote") {
				echo $HOME
			}`,
			shouldErr: false,
			reason:    "shell variables ($HOME) are always allowed",
		},
		{
			name: "root-only value decorator forbidden in transport decorator",
			input: `@test.ssh(host="remote") {
				var home = @test.env.KEY
			}`,
			shouldErr: true,
			reason:    "root-only value decorators cannot be used inside transport decorators",
		},
		{
			name: "@env forbidden in @docker.exec (when registered)",
			input: `@docker.exec(container="app") {
				var user = @env.USER
			}`,
			shouldErr: false, // Will be true when docker.exec is registered with RoleBoundary
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
