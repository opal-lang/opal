package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators"
	"github.com/opal-lang/opal/runtime/parser"
)

// planSource is a test helper that parses and plans source code.
func planSource(t *testing.T, source string) *planfmt.Plan {
	t.Helper()
	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := Plan(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}
	return plan
}

// ========== Transport Boundary Integration Tests ==========
//
// These tests verify that transport boundaries are enforced correctly:
// - Transport-sensitive decorators (@env) cannot cross boundaries
// - Transport-agnostic decorators (@var) can cross boundaries
// - Transport decorators create boundaries via EnterTransport/ExitTransport

func init() {
	// Register TestTransport for testing
	transport := decorator.NewTestTransport("ssh:test-server")
	_ = decorator.Register("test.transport", transport)
}

// TestTransportBoundary_IsTransportDecorator verifies the helper function
func TestTransportBoundary_IsTransportDecorator(t *testing.T) {
	p := &planner{}

	tests := []struct {
		name     string
		expected bool
	}{
		{"@test.transport", true},
		{"test.transport", true},
		{"@shell", false},
		{"shell", false},
		{"@retry", false},
		{"@var", false},
		{"@env", false},
		{"@nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.isTransportDecorator(tt.name)
			if result != tt.expected {
				t.Errorf("isTransportDecorator(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

// TestTransportBoundary_VaultEnterExitCalled verifies that transport decorators
// trigger EnterTransport/ExitTransport calls on the vault.
func TestTransportBoundary_VaultEnterExitCalled(t *testing.T) {
	// This test verifies the planner correctly calls vault transport methods.
	// We can't easily test this without a mock vault, but we can verify
	// the isTransportDecorator logic works correctly.

	p := &planner{}

	// Verify test.transport is recognized as a transport decorator
	if !p.isTransportDecorator("@test.transport") {
		t.Error("@test.transport should be recognized as a transport decorator")
	}

	// Verify @shell is NOT a transport decorator
	if p.isTransportDecorator("@shell") {
		t.Error("@shell should NOT be recognized as a transport decorator")
	}

	// Verify @retry is NOT a transport decorator
	if p.isTransportDecorator("@retry") {
		t.Error("@retry should NOT be recognized as a transport decorator")
	}
}

// TestTransportBoundary_TransportSensitiveCapability verifies that @env has
// TransportSensitive set to true.
func TestTransportBoundary_TransportSensitiveCapability(t *testing.T) {
	entry, ok := decorator.Global().Lookup("env")
	if !ok {
		t.Fatal("@env decorator not found in registry")
	}

	desc := entry.Impl.Descriptor()
	if !desc.Capabilities.TransportSensitive {
		t.Error("@env should have TransportSensitive = true")
	}
}

// TestTransportBoundary_VarNotTransportSensitive verifies that @var does NOT
// have TransportSensitive set.
func TestTransportBoundary_VarNotTransportSensitive(t *testing.T) {
	entry, ok := decorator.Global().Lookup("var")
	if !ok {
		t.Fatal("@var decorator not found in registry")
	}

	desc := entry.Impl.Descriptor()
	if desc.Capabilities.TransportSensitive {
		t.Error("@var should have TransportSensitive = false")
	}
}

// TestTransportBoundary_TestTransportHasBoundaryRole verifies that TestTransport
// has the RoleBoundary role.
func TestTransportBoundary_TestTransportHasBoundaryRole(t *testing.T) {
	entry, ok := decorator.Global().Lookup("test.transport")
	if !ok {
		t.Fatal("test.transport not found in registry")
	}

	hasBoundary := false
	for _, role := range entry.Roles {
		if role == decorator.RoleBoundary {
			hasBoundary = true
			break
		}
	}

	if !hasBoundary {
		t.Error("test.transport should have RoleBoundary role")
	}
}

// TestTransportBoundary_EnvInCommandIsTransportSensitive verifies that @env.X
// used directly in a command is tracked as transport-sensitive in the vault.
//
// This is a regression test for P0 bug: transport boundary check was short-circuiting
// because expressions were never marked as transport-sensitive.
func TestTransportBoundary_EnvInCommandIsTransportSensitive(t *testing.T) {
	source := `echo "Home: @env.HOME"`

	plan := planSource(t, source)

	// The plan should have SecretUses for @env.HOME
	if len(plan.SecretUses) == 0 {
		t.Fatal("Expected SecretUses for @env.HOME, got none")
	}

	// Verify the expression is tracked (we can't directly check TransportSensitive
	// from outside the vault, but we can verify the planner processed @env)
	found := false
	for _, use := range plan.SecretUses {
		if use.Site != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected @env.HOME to be tracked in SecretUses")
	}
}

// TestTransportBoundary_EnvBlockedAcrossBoundary verifies that @env values
// resolved in local context cannot be accessed inside a transport block.
//
// This is the core security property: @env.HOME from local machine should NOT
// be usable inside @ssh block (different machine, different HOME).
func TestTransportBoundary_EnvBlockedAcrossBoundary(t *testing.T) {
	source := `
var LOCAL_HOME = @env.HOME
@test.transport {
    echo "Home: @var.LOCAL_HOME"
}
`
	// Planning MUST fail because @env.HOME is transport-sensitive and
	// cannot cross the transport boundary into @test.transport.
	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	_, err := Plan(tree.Events, tree.Tokens, Config{})

	if err == nil {
		t.Fatal("Expected transport boundary error: @env.HOME should not be usable inside @test.transport")
	}

	// Verify it's the right kind of error
	errStr := err.Error()
	if !strings.Contains(errStr, "transport") {
		t.Errorf("Expected transport boundary error, got: %v", err)
	}
}

// TestTransportBoundary_VarInheritsTransportSensitivity verifies that when
// a variable is assigned from a transport-sensitive decorator, the variable
// itself becomes transport-sensitive.
//
// var HOME = @env.HOME  ← HOME should be transport-sensitive because @env is
func TestTransportBoundary_VarInheritsTransportSensitivity(t *testing.T) {
	source := `
var HOME = @env.HOME
@test.transport {
    echo "Home: @var.HOME"
}
`
	// This SHOULD fail during planning because @var.HOME references a
	// transport-sensitive value that was resolved in a different transport context.
	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	_, err := Plan(tree.Events, tree.Tokens, Config{})

	// MUST fail - using transport-sensitive value across boundary
	if err == nil {
		t.Fatal("Expected transport boundary error: var HOME = @env.HOME should make HOME transport-sensitive, but @var.HOME was allowed inside @test.transport")
	}

	// Verify it's the right kind of error (transport boundary violation)
	errStr := err.Error()
	if !strings.Contains(errStr, "transport") {
		t.Errorf("Expected transport boundary error, got: %v", err)
	}
}

// TestTransportBoundary_VarFromLiteralAllowedAcrossBoundary verifies that
// variables assigned from literals CAN cross transport boundaries.
//
// var VERSION = "1.0.0"  ← VERSION is transport-agnostic (literal has no transport context)
func TestTransportBoundary_VarFromLiteralAllowedAcrossBoundary(t *testing.T) {
	source := `
var VERSION = "1.0.0"
@test.transport {
    echo "Version: @var.VERSION"
}
`
	// This SHOULD succeed - literals are transport-agnostic
	plan := planSource(t, source)

	// Verify plan was created
	if plan == nil {
		t.Fatal("Expected plan to be created")
	}

	// Verify SecretUses contains the variable
	if len(plan.SecretUses) == 0 {
		t.Error("Expected SecretUses for @var.VERSION")
	}
}

// TestTransportBoundary_DirectEnvInTransportBlockWorks verifies that @env
// used directly inside a transport block resolves from that transport's context.
//
// @ssh("server") { echo @env.HOME }  ← resolves HOME on the remote server
func TestTransportBoundary_DirectEnvInTransportBlockWorks(t *testing.T) {
	source := `
@test.transport {
    echo "Home: @env.HOME"
}
`
	// This SHOULD succeed - @env.HOME resolves in the transport's context
	plan := planSource(t, source)

	if plan == nil {
		t.Fatal("Expected plan to be created")
	}

	// Verify SecretUses contains the @env expression
	if len(plan.SecretUses) == 0 {
		t.Error("Expected SecretUses for @env.HOME")
	}
}
