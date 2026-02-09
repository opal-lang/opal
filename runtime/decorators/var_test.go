package decorators

import (
	"testing"

	"github.com/opal-lang/opal/core/decorator"
)

// resolveSingle is a test helper that calls Resolve with a single call and returns the result.
// It fails the test if there are any errors.
func resolveSingle(t *testing.T, d decorator.Value, ctx decorator.ValueEvalContext, call decorator.ValueCall) decorator.ResolveResult {
	t.Helper()

	results, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	return results[0]
}

func valueCtx(vars map[string]any) decorator.ValueEvalContext {
	lookup := func(name string) (any, bool) {
		v, ok := vars[name]
		return v, ok
	}

	return decorator.ValueEvalContext{
		Session:     decorator.NewLocalSession(),
		LookupValue: lookup,
	}
}

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
	ctx := valueCtx(map[string]any{"name": "Alice"})

	// Test string variable
	varName := "name"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
		Params:  map[string]any{},
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != "Alice" {
		t.Errorf("Value: got %v, want %q", result.Value, "Alice")
	}

	if result.Origin != "var.name" {
		t.Errorf("Origin: got %q, want %q", result.Origin, "var.name")
	}
}

// TestVarDecoratorResolveInt verifies integer variable resolution
func TestVarDecoratorResolveInt(t *testing.T) {
	d := &VarDecorator{}
	ctx := valueCtx(map[string]any{"count": int64(42)})

	varName := "count"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != int64(42) {
		t.Errorf("Value: got %v, want 42", result.Value)
	}
}

// TestVarDecoratorResolveBool verifies boolean variable resolution
func TestVarDecoratorResolveBool(t *testing.T) {
	d := &VarDecorator{}
	ctx := valueCtx(map[string]any{"flag": true})

	varName := "flag"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != true {
		t.Errorf("Value: got %v, want true", result.Value)
	}
}

// TestVarDecoratorResolveNotFound verifies error when variable not found
func TestVarDecoratorResolveNotFound(t *testing.T) {
	d := &VarDecorator{}
	ctx := valueCtx(map[string]any{"name": "Alice"})

	varName := "missing"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	results, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Fatal("Expected error for missing variable")
	}

	expectedMsg := `variable "missing" not found in any scope`
	if results[0].Error.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", results[0].Error.Error(), expectedMsg)
	}
}

// TestVarDecoratorResolveNoPrimary verifies error when no variable name provided
func TestVarDecoratorResolveNoPrimary(t *testing.T) {
	d := &VarDecorator{}
	ctx := valueCtx(map[string]any{})

	call := decorator.ValueCall{
		Path:    "var",
		Primary: nil, // No primary parameter
	}

	results, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Fatal("Expected error when no primary parameter")
	}

	expectedMsg := "@var requires a variable name"
	if results[0].Error.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", results[0].Error.Error(), expectedMsg)
	}
}

// TestVarDecoratorTransportAgnostic verifies @var works in different transports
func TestVarDecoratorTransportAgnostic(t *testing.T) {
	d := &VarDecorator{}
	localCtx := valueCtx(map[string]any{"value": "local"})

	varName := "value"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	result := resolveSingle(t, d, localCtx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != "local" {
		t.Errorf("LocalSession value: got %v, want %q", result.Value, "local")
	}

	// @var should work the same regardless of transport
	// (We can't test SSH here without a server, but the point is it's transport-agnostic)
}

// TestVarDecoratorEmptyVars verifies behavior with empty variable map
func TestVarDecoratorEmptyVars(t *testing.T) {
	d := &VarDecorator{}
	ctx := valueCtx(map[string]any{})

	varName := "anything"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	results, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Fatal("Expected error when vault is empty")
	}

	expectedMsg := `variable "anything" not found in any scope`
	if results[0].Error.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", results[0].Error.Error(), expectedMsg)
	}
}

func TestVarDecoratorResolveCallFormArg1Rejected(t *testing.T) {
	d := &VarDecorator{}
	ctx := valueCtx(map[string]any{"name": "Alice"})

	call := decorator.ValueCall{
		Path:   "var",
		Params: map[string]any{"arg1": "name"},
	}

	result := resolveSingle(t, d, ctx, call)
	if result.Error == nil {
		t.Fatal("expected error")
	}
	if got, want := result.Error.Error(), "@var requires a variable name"; got != want {
		t.Errorf("Error mismatch: got %q, want %q", got, want)
	}
}
