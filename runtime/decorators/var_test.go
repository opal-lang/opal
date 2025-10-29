package decorators

import (
	"testing"

	"github.com/aledsdavies/opal/core/decorator"
)

// TestVarDecoratorDescriptor verifies the decorator metadata
func TestVarDecoratorDescriptor(t *testing.T) {
	d := &VarDecorator{}
	desc := d.Descriptor()

	// Verify path
	if desc.Path != "var" {
		t.Errorf("Path: got %q, want %q", desc.Path, "var")
	}

	// Verify roles include Provider
	hasProvider := false
	for _, role := range desc.Roles {
		if role == decorator.RoleProvider {
			hasProvider = true
			break
		}
	}
	if !hasProvider {
		t.Error("Roles should include RoleProvider")
	}

	// Verify transport scope is Any (works everywhere)
	if desc.Capabilities.TransportScope != decorator.TransportScopeAny {
		t.Errorf("TransportScope: got %v, want TransportScopeAny", desc.Capabilities.TransportScope)
	}

	// Verify purity (deterministic)
	if !desc.Capabilities.Purity {
		t.Error("Purity should be true (deterministic)")
	}

	// Verify idempotent
	if !desc.Capabilities.Idempotent {
		t.Error("Idempotent should be true")
	}
}

// TestVarDecoratorResolveSuccess verifies successful variable resolution
func TestVarDecoratorResolveSuccess(t *testing.T) {
	d := &VarDecorator{}

	// Create context with variables
	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars: map[string]any{
			"name":  "Alice",
			"count": 42,
			"flag":  true,
		},
	}

	// Test string variable
	varName := "name"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
		Params:  map[string]any{},
	}

	value, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != "Alice" {
		t.Errorf("Value: got %v, want %q", value, "Alice")
	}
}

// TestVarDecoratorResolveInt verifies integer variable resolution
func TestVarDecoratorResolveInt(t *testing.T) {
	d := &VarDecorator{}

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars: map[string]any{
			"count": 42,
		},
	}

	varName := "count"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	value, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != 42 {
		t.Errorf("Value: got %v, want 42", value)
	}
}

// TestVarDecoratorResolveBool verifies boolean variable resolution
func TestVarDecoratorResolveBool(t *testing.T) {
	d := &VarDecorator{}

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars: map[string]any{
			"flag": true,
		},
	}

	varName := "flag"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	value, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != true {
		t.Errorf("Value: got %v, want true", value)
	}
}

// TestVarDecoratorResolveNotFound verifies error when variable not found
func TestVarDecoratorResolveNotFound(t *testing.T) {
	d := &VarDecorator{}

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars: map[string]any{
			"name": "Alice",
		},
	}

	varName := "missing"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	_, err := d.Resolve(ctx, call)
	if err == nil {
		t.Fatal("Expected error for missing variable")
	}

	expectedMsg := `variable "missing" not found`
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

// TestVarDecoratorResolveNoPrimary verifies error when no variable name provided
func TestVarDecoratorResolveNoPrimary(t *testing.T) {
	d := &VarDecorator{}

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars:    map[string]any{},
	}

	call := decorator.ValueCall{
		Path:    "var",
		Primary: nil, // No variable name
	}

	_, err := d.Resolve(ctx, call)
	if err == nil {
		t.Fatal("Expected error when no variable name provided")
	}

	expectedMsg := "@var requires a variable name"
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

// TestVarDecoratorTransportAgnostic verifies @var works in different transports
func TestVarDecoratorTransportAgnostic(t *testing.T) {
	d := &VarDecorator{}

	// Test with LocalSession
	localCtx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars: map[string]any{
			"value": "local",
		},
	}

	varName := "value"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	value, err := d.Resolve(localCtx, call)
	if err != nil {
		t.Fatalf("Resolve with LocalSession failed: %v", err)
	}
	if value != "local" {
		t.Errorf("LocalSession value: got %v, want %q", value, "local")
	}

	// @var should work the same regardless of transport
	// (We can't test SSH here without a server, but the point is it's transport-agnostic)
}

// TestVarDecoratorEmptyVars verifies behavior with empty variable map
func TestVarDecoratorEmptyVars(t *testing.T) {
	d := &VarDecorator{}

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vars:    map[string]any{}, // Empty
	}

	varName := "anything"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	_, err := d.Resolve(ctx, call)
	if err == nil {
		t.Fatal("Expected error when vars map is empty")
	}
}
