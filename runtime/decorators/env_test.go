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

	value, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Should return the actual USER value
	expectedUser := os.Getenv("USER")
	if value != expectedUser {
		t.Errorf("Value: got %v, want %q", value, expectedUser)
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

	value, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Value: got %v, want %q", value, "test_value")
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

	_, err := d.Resolve(ctx, call)
	if err == nil {
		t.Fatal("Expected error for missing env var")
	}

	expectedMsg := `environment variable "NONEXISTENT_VAR_12345" not found`
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
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

	value, err := d.Resolve(ctx, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != "fallback_value" {
		t.Errorf("Value: got %v, want %q", value, "fallback_value")
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

	_, err := d.Resolve(ctx, call)
	if err == nil {
		t.Fatal("Expected error when no env var name provided")
	}

	expectedMsg := "@env requires an environment variable name"
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
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

	localValue, err := d.Resolve(localCtx, call)
	if err != nil {
		t.Fatalf("Local resolve failed: %v", err)
	}

	// Should match os.Getenv
	expectedHome := os.Getenv("HOME")
	if localValue != expectedHome {
		t.Errorf("Local HOME: got %v, want %q", localValue, expectedHome)
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
	value1, err := d.Resolve(ctx1, call)
	if err != nil {
		t.Fatalf("Session1 resolve failed: %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Session1 value: got %v, want %q", value1, "value1")
	}

	// Resolve in session2
	value2, err := d.Resolve(ctx2, call)
	if err != nil {
		t.Fatalf("Session2 resolve failed: %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Session2 value: got %v, want %q", value2, "value2")
	}
}
