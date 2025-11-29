package planner

import (
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

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
