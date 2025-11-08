package decorators

import (
	"os"
	"testing"

	"github.com/aledsdavies/opal/core/decorator"
)

// TestEnvDecoratorDescriptor verifies the decorator metadata
func TestEnvDecoratorDescriptor(t *testing.T) {
	d := &EnvDecorator{}
	desc := d.Descriptor()

	// Verify path
	if desc.Path != "env" {
		t.Errorf("Path: got %q, want %q", desc.Path, "env")
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

	// Verify transport scope is Any (reads from session env)
	if desc.Capabilities.TransportScope != decorator.TransportScopeAny {
		t.Errorf("TransportScope: got %v, want TransportScopeAny", desc.Capabilities.TransportScope)
	}

	// Verify NOT pure (reads external state)
	if desc.Capabilities.Purity {
		t.Error("Purity should be false (reads external state)")
	}

	// Verify idempotent
	if !desc.Capabilities.Idempotent {
		t.Error("Idempotent should be true")
	}
}

// TestEnvDecoratorResolveFromLocalSession verifies reading env from local session
func TestEnvDecoratorResolveFromLocalSession(t *testing.T) {
	d := &EnvDecorator{}

	// Create local session (reads from os.Environ)
	session := decorator.NewLocalSession()

	ctx := decorator.ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	// Read USER env var (should exist in most environments)
	envVar := "USER"
	call := decorator.ValueCall{
		Path:    "env",
		Primary: &envVar,
		Params:  map[string]any{},
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	// Should return the actual USER value
	expectedUser := os.Getenv("USER")
	if result.Value != expectedUser {
		t.Errorf("Value: got %v, want %q", result.Value, expectedUser)
	}
}

// TestEnvDecoratorResolveFromSessionWithEnv verifies reading from modified session
func TestEnvDecoratorResolveFromSessionWithEnv(t *testing.T) {
	d := &EnvDecorator{}

	// Create session with custom env
	baseSession := decorator.NewLocalSession()
	session := baseSession.WithEnv(map[string]string{
		"OPAL_TEST_VAR": "test_value",
	})

	ctx := decorator.ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	envVar := "OPAL_TEST_VAR"
	call := decorator.ValueCall{
		Path:    "env",
		Primary: &envVar,
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != "test_value" {
		t.Errorf("Value: got %v, want %q", result.Value, "test_value")
	}
}

// TestEnvDecoratorResolveNotFound verifies error when env var not found
func TestEnvDecoratorResolveNotFound(t *testing.T) {
	d := &EnvDecorator{}

	session := decorator.NewLocalSession()
	ctx := decorator.ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	envVar := "NONEXISTENT_VAR_12345"
	call := decorator.ValueCall{
		Path:    "env",
		Primary: &envVar,
	}

	results, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Fatal("Expected error for missing env var")
	}

	expectedMsg := `environment variable "NONEXISTENT_VAR_12345" not found`
	if results[0].Error.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", results[0].Error.Error(), expectedMsg)
	}
}

// TestEnvDecoratorResolveWithDefault verifies default parameter
func TestEnvDecoratorResolveWithDefault(t *testing.T) {
	d := &EnvDecorator{}

	session := decorator.NewLocalSession()
	ctx := decorator.ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	envVar := "NONEXISTENT_VAR_12345"
	call := decorator.ValueCall{
		Path:    "env",
		Primary: &envVar,
		Params: map[string]any{
			"default": "fallback_value",
		},
	}

	result := resolveSingle(t, d, ctx, call)

	if result.Error != nil {
		t.Fatalf("Result error: %v", result.Error)
	}

	if result.Value != "fallback_value" {
		t.Errorf("Value: got %v, want %q", result.Value, "fallback_value")
	}
}

// TestEnvDecoratorResolveNoPrimary verifies error when no env var name provided
func TestEnvDecoratorResolveNoPrimary(t *testing.T) {
	d := &EnvDecorator{}

	session := decorator.NewLocalSession()
	ctx := decorator.ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	call := decorator.ValueCall{
		Path:    "env",
		Primary: nil, // No env var name
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

	expectedMsg := "@env requires an environment variable name"
	if results[0].Error.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", results[0].Error.Error(), expectedMsg)
	}
}

// TestEnvDecoratorTransportAware verifies @env reads from session environment
func TestEnvDecoratorTransportAware(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	d := &EnvDecorator{}

	// Create local session
	localSession := decorator.NewLocalSession()
	localCtx := decorator.ValueEvalContext{
		Session: localSession,
		Vars:    map[string]any{},
	}

	// Read HOME from local session
	envVar := "HOME"
	call := decorator.ValueCall{
		Path:    "env",
		Primary: &envVar,
	}

	localResult := resolveSingle(t, d, localCtx, call)

	if localResult.Error != nil {
		t.Fatalf("Result error: %v", localResult.Error)
	}

	// Should match os.Getenv
	expectedHome := os.Getenv("HOME")
	if localResult.Value != expectedHome {
		t.Errorf("Local HOME: got %v, want %q", localResult.Value, expectedHome)
	}

	// @env reads from Session.Env(), so it's transport-aware
	// In SSH context, it would read remote HOME
}

// TestEnvDecoratorSessionEnvIsolation verifies env changes don't leak
func TestEnvDecoratorSessionEnvIsolation(t *testing.T) {
	d := &EnvDecorator{}

	// Create two sessions with different env
	base := decorator.NewLocalSession()
	session1 := base.WithEnv(map[string]string{
		"OPAL_VAR": "value1",
	})
	session2 := base.WithEnv(map[string]string{
		"OPAL_VAR": "value2",
	})

	ctx1 := decorator.ValueEvalContext{
		Session: session1,
		Vars:    map[string]any{},
	}
	ctx2 := decorator.ValueEvalContext{
		Session: session2,
		Vars:    map[string]any{},
	}

	envVar := "OPAL_VAR"
	call := decorator.ValueCall{
		Path:    "env",
		Primary: &envVar,
	}

	// Resolve in session1
	result1 := resolveSingle(t, d, ctx1, call)
	if result1.Error != nil {
		t.Fatalf("Result error: %v", result1.Error)
	}
	if result1.Value != "value1" {
		t.Errorf("Session1 value: got %v, want %q", result1.Value, "value1")
	}

	// Resolve in session2
	result2 := resolveSingle(t, d, ctx2, call)
	if result2.Error != nil {
		t.Fatalf("Result error: %v", result2.Error)
	}
	if result2.Value != "value2" {
		t.Errorf("Session2 value: got %v, want %q", result2.Value, "value2")
	}
}
