package decorators

import (
	"testing"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/runtime/vault"
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

	// Create Vault and declare variables
	v := vault.New()
	nameID := v.DeclareVariable("name", "literal:Alice")
	v.MarkResolved(nameID, "Alice")
	v.RecordReference(nameID, "value") // Authorize access at root/params/value

	// Create context with Vault
	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

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

	v := vault.New()
	countID := v.DeclareVariable("count", "literal:42")
	v.MarkResolved(countID, "42")
	v.RecordReference(countID, "value") // Authorize access

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

	varName := "count"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != "42" {
		t.Errorf("Value: got %v, want \"42\"", result.Value)
	}
}

// TestVarDecoratorResolveBool verifies boolean variable resolution
func TestVarDecoratorResolveBool(t *testing.T) {
	d := &VarDecorator{}

	v := vault.New()
	flagID := v.DeclareVariable("flag", "literal:true")
	v.MarkResolved(flagID, "true")
	v.RecordReference(flagID, "value") // Authorize access

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

	varName := "flag"
	call := decorator.ValueCall{
		Path:    "var",
		Primary: &varName,
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != "true" {
		t.Errorf("Value: got %v, want \"true\"", result.Value)
	}
}

// TestVarDecoratorResolveNotFound verifies error when variable not found
func TestVarDecoratorResolveNotFound(t *testing.T) {
	d := &VarDecorator{}

	v := vault.New()
	nameID := v.DeclareVariable("name", "literal:Alice")
	v.MarkResolved(nameID, "Alice")

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

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

	v := vault.New()
	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

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

	v := vault.New()
	valueID := v.DeclareVariable("value", "literal:local")
	v.MarkResolved(valueID, "local")
	v.RecordReference(valueID, "value") // Authorize access

	// Test with LocalSession
	localCtx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

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

	v := vault.New() // Empty vault

	ctx := decorator.ValueEvalContext{
		Session: decorator.NewLocalSession(),
		Vault:   v,
	}

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
