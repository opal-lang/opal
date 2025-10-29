package types

import (
	"testing"
)

// TestRoleEnumValues verifies all Role enum values are defined
func TestRoleEnumValues(t *testing.T) {
	roles := []Role{
		RoleProvider,
		RoleWrapper,
		RoleBoundary,
		RoleEndpoint,
		RoleAnnotate,
	}

	// Verify each role has a non-empty string value
	for _, r := range roles {
		if string(r) == "" {
			t.Errorf("Role enum value is empty: %v", r)
		}
	}

	// Verify expected string representations
	tests := []struct {
		role Role
		want string
	}{
		{RoleProvider, "provider"},
		{RoleWrapper, "wrapper"},
		{RoleBoundary, "boundary"},
		{RoleEndpoint, "endpoint"},
		{RoleAnnotate, "annotate"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.want {
			t.Errorf("Role %v: got %q, want %q", tt.role, string(tt.role), tt.want)
		}
	}
}

// TestDescriptorCreation verifies Descriptor struct can be created with all fields
func TestDescriptorCreation(t *testing.T) {
	desc := Descriptor{
		Path:    "test.decorator",
		Roles:   []Role{RoleProvider},
		Version: "1.0.0",
		Summary: "Test decorator",
		DocURL:  "https://example.com/docs",
		Schema: DecoratorSchema{
			Path: "test.decorator",
			Kind: "value",
		},
		Capabilities: Capabilities{
			TransportScope: ScopeAgnostic,
			Purity:         true,
			Idempotent:     true,
		},
	}

	// Verify all fields are accessible
	if desc.Path != "test.decorator" {
		t.Errorf("Path: got %q, want %q", desc.Path, "test.decorator")
	}
	if len(desc.Roles) != 1 || desc.Roles[0] != RoleProvider {
		t.Errorf("Roles: got %v, want [%v]", desc.Roles, RoleProvider)
	}
	if desc.Version != "1.0.0" {
		t.Errorf("Version: got %q, want %q", desc.Version, "1.0.0")
	}
	if desc.Summary != "Test decorator" {
		t.Errorf("Summary: got %q, want %q", desc.Summary, "Test decorator")
	}
	if desc.DocURL != "https://example.com/docs" {
		t.Errorf("DocURL: got %q, want %q", desc.DocURL, "https://example.com/docs")
	}
	if desc.Capabilities.TransportScope != ScopeAgnostic {
		t.Errorf("TransportScope: got %v, want %v", desc.Capabilities.TransportScope, ScopeAgnostic)
	}
	if !desc.Capabilities.Purity {
		t.Error("Purity: got false, want true")
	}
	if !desc.Capabilities.Idempotent {
		t.Error("Idempotent: got false, want true")
	}
}

// TestCapabilitiesDefaults verifies Capabilities struct has sensible zero values
func TestCapabilitiesDefaults(t *testing.T) {
	var caps Capabilities

	// Zero values should be safe defaults
	if caps.TransportScope != TransportScope(0) {
		t.Errorf("Default TransportScope: got %v, want 0", caps.TransportScope)
	}
	if caps.Purity {
		t.Error("Default Purity: got true, want false (safe default)")
	}
	if caps.Idempotent {
		t.Error("Default Idempotent: got true, want false (safe default)")
	}

	// IO should have zero values
	if caps.IO.PipeIn {
		t.Error("Default IO.PipeIn: got true, want false")
	}
	if caps.IO.PipeOut {
		t.Error("Default IO.PipeOut: got true, want false")
	}
	if caps.IO.RedirectIn {
		t.Error("Default IO.RedirectIn: got true, want false")
	}
	if caps.IO.RedirectOut {
		t.Error("Default IO.RedirectOut: got true, want false")
	}
}

// TestDecoratorInterface verifies Decorator interface contract
func TestDecoratorInterface(t *testing.T) {
	// Create a mock decorator
	mock := &mockDecorator{
		desc: Descriptor{
			Path:  "mock",
			Roles: []Role{RoleProvider},
		},
	}

	// Verify it implements Decorator interface
	var _ Decorator = mock

	// Verify Descriptor() returns expected value
	desc := mock.Descriptor()
	if desc.Path != "mock" {
		t.Errorf("Descriptor().Path: got %q, want %q", desc.Path, "mock")
	}
	if len(desc.Roles) != 1 || desc.Roles[0] != RoleProvider {
		t.Errorf("Descriptor().Roles: got %v, want [%v]", desc.Roles, RoleProvider)
	}
}

// TestIOSemantics verifies IOSemantics struct
func TestIOSemantics(t *testing.T) {
	io := IOSemantics{
		PipeIn:      true,
		PipeOut:     true,
		RedirectIn:  false,
		RedirectOut: true,
	}

	if !io.PipeIn {
		t.Error("PipeIn: got false, want true")
	}
	if !io.PipeOut {
		t.Error("PipeOut: got false, want true")
	}
	if io.RedirectIn {
		t.Error("RedirectIn: got true, want false")
	}
	if !io.RedirectOut {
		t.Error("RedirectOut: got false, want true")
	}
}

// TestMultiRoleDecorator verifies decorators can have multiple roles
func TestMultiRoleDecorator(t *testing.T) {
	desc := Descriptor{
		Path:  "aws.s3.object",
		Roles: []Role{RoleProvider, RoleEndpoint}, // Multi-role!
	}

	if len(desc.Roles) != 2 {
		t.Errorf("Roles length: got %d, want 2", len(desc.Roles))
	}

	// Verify both roles are present
	hasProvider := false
	hasEndpoint := false
	for _, role := range desc.Roles {
		if role == RoleProvider {
			hasProvider = true
		}
		if role == RoleEndpoint {
			hasEndpoint = true
		}
	}

	if !hasProvider {
		t.Error("Missing RoleProvider in multi-role decorator")
	}
	if !hasEndpoint {
		t.Error("Missing RoleEndpoint in multi-role decorator")
	}
}

// mockDecorator is a test implementation of Decorator interface
type mockDecorator struct {
	desc Descriptor
}

func (m *mockDecorator) Descriptor() Descriptor {
	return m.desc
}
