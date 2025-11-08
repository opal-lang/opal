package decorator

import (
	"io"
	"testing"
)

// TestAutoInference verifies roles are auto-inferred from interfaces
func TestAutoInference(t *testing.T) {
	r := NewRegistry()

	// Register a value decorator
	varDec := &mockValueDecorator{path: "var"}
	err := r.register("var", varDec)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Lookup and verify role was inferred
	entry, ok := r.Lookup("var")
	if !ok {
		t.Fatal("decorator not found")
	}

	if len(entry.Roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(entry.Roles))
	}

	if entry.Roles[0] != RoleProvider {
		t.Errorf("expected RoleProvider, got %v", entry.Roles[0])
	}
}

// TestMultiRoleInference verifies multi-role decorators
func TestMultiRoleInference(t *testing.T) {
	r := NewRegistry()

	// Register a decorator that implements both Value and Endpoint
	s3Dec := &mockMultiRoleDecorator{path: "aws.s3.object"}
	err := r.register("aws.s3.object", s3Dec)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Lookup and verify both roles were inferred
	entry, ok := r.Lookup("aws.s3.object")
	if !ok {
		t.Fatal("decorator not found")
	}

	if len(entry.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(entry.Roles))
	}

	hasProvider := false
	hasEndpoint := false
	for _, role := range entry.Roles {
		if role == RoleProvider {
			hasProvider = true
		}
		if role == RoleEndpoint {
			hasEndpoint = true
		}
	}

	if !hasProvider {
		t.Error("missing RoleProvider")
	}
	if !hasEndpoint {
		t.Error("missing RoleEndpoint")
	}
}

// TestAllRoleInference verifies all role types can be inferred
func TestAllRoleInference(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		impl     Decorator
		wantRole Role
	}{
		{"value", &mockValueDecorator{path: "test.value"}, RoleProvider},
		{"exec", &mockExecDecorator{path: "test.exec"}, RoleWrapper},
		{"transport", &mockTransportDecorator{path: "test.transport"}, RoleBoundary},
		{"endpoint", &mockEndpointDecorator{path: "test.endpoint"}, RoleEndpoint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.register(tt.impl.Descriptor().Path, tt.impl)
			if err != nil {
				t.Fatalf("register failed: %v", err)
			}

			entry, ok := r.Lookup(tt.impl.Descriptor().Path)
			if !ok {
				t.Fatal("decorator not found")
			}

			found := false
			for _, role := range entry.Roles {
				if role == tt.wantRole {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected role %v, got %v", tt.wantRole, entry.Roles)
			}
		})
	}
}

// TestGlobalRegistration verifies database/sql pattern
func TestGlobalRegistration(t *testing.T) {
	// Simulate init() registration
	varDec := &mockValueDecorator{path: "test.var"}
	err := Register("test.var", varDec)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Lookup from global registry
	entry, ok := Global().Lookup("test.var")
	if !ok {
		t.Fatal("decorator not found in global registry")
	}

	if entry.Impl.Descriptor().Path != "test.var" {
		t.Errorf("path: got %q, want %q", entry.Impl.Descriptor().Path, "test.var")
	}
}

// TestURIBasedLookup verifies path-based lookup
func TestURIBasedLookup(t *testing.T) {
	r := NewRegistry()

	// Register hierarchical paths
	r.register("env", &mockValueDecorator{path: "env"})
	r.register("aws.secret", &mockValueDecorator{path: "aws.secret"})
	r.register("aws.s3.object", &mockMultiRoleDecorator{path: "aws.s3.object"})

	tests := []struct {
		path      string
		wantFound bool
	}{
		{"env", true},
		{"aws.secret", true},
		{"aws.s3.object", true},
		{"aws", false},    // Partial path doesn't match
		{"aws.s3", false}, // Partial path doesn't match
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			_, found := r.Lookup(tt.path)
			if found != tt.wantFound {
				t.Errorf("Lookup(%q): found=%v, want=%v", tt.path, found, tt.wantFound)
			}
		})
	}
}

// TestPrimaryParamHandling verifies PrimaryParam is handled correctly
func TestPrimaryParamHandling(t *testing.T) {
	r := NewRegistry()

	// Register decorator that uses primary param
	envDec := &mockValueDecorator{path: "env"}
	r.register("env", envDec)

	// Lookup decorator by path (not including primary param)
	entry, ok := r.Lookup("env")
	if !ok {
		t.Fatal("decorator not found")
	}

	// Primary param is passed in ValueCall, not in path lookup
	primary := "HOME"
	call := ValueCall{
		Path:    "env",
		Primary: &primary,
		Params:  map[string]any{},
	}

	// Verify decorator receives primary param
	value, ok := entry.Impl.(Value)
	if !ok {
		t.Fatal("decorator doesn't implement Value")
	}

	results, err := value.Resolve(ValueEvalContext{}, call)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error != nil {
		t.Fatalf("result error: %v", results[0].Error)
	}

	// Mock returns "mock-value" regardless, but primary param was passed
	if results[0].Value != "mock-value" {
		t.Errorf("unexpected result: %v", results[0].Value)
	}
}

// TestValueCallConstruction verifies ValueCall is constructed correctly
func TestValueCallConstruction(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		primary     *string
		params      map[string]any
		wantPath    string
		wantPrimary *string
	}{
		{
			name:        "with primary param",
			path:        "env",
			primary:     strPtr("HOME"),
			params:      map[string]any{},
			wantPath:    "env",
			wantPrimary: strPtr("HOME"),
		},
		{
			name:        "without primary param",
			path:        "retry",
			primary:     nil,
			params:      map[string]any{"attempts": 3},
			wantPath:    "retry",
			wantPrimary: nil,
		},
		{
			name:        "hierarchical path with primary",
			path:        "aws.secret",
			primary:     strPtr("API_KEY"),
			params:      map[string]any{},
			wantPath:    "aws.secret",
			wantPrimary: strPtr("API_KEY"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := ValueCall{
				Path:    tt.path,
				Primary: tt.primary,
				Params:  tt.params,
			}

			if call.Path != tt.wantPath {
				t.Errorf("Path: got %q, want %q", call.Path, tt.wantPath)
			}

			if (call.Primary == nil) != (tt.wantPrimary == nil) {
				t.Errorf("Primary nil mismatch: got %v, want %v", call.Primary, tt.wantPrimary)
			}

			if call.Primary != nil && tt.wantPrimary != nil && *call.Primary != *tt.wantPrimary {
				t.Errorf("Primary: got %q, want %q", *call.Primary, *tt.wantPrimary)
			}
		})
	}
}

// TestExport verifies Export returns all registered decorators
func TestExport(t *testing.T) {
	r := NewRegistry()

	// Register multiple decorators
	r.register("var", &mockValueDecorator{path: "var"})
	r.register("retry", &mockExecDecorator{path: "retry"})
	r.register("aws.s3.object", &mockMultiRoleDecorator{path: "aws.s3.object"})

	descriptors := r.Export()

	if len(descriptors) != 3 {
		t.Fatalf("expected 3 descriptors, got %d", len(descriptors))
	}

	// Verify roles are included in exported descriptors
	for _, desc := range descriptors {
		if len(desc.Roles) == 0 {
			t.Errorf("descriptor %q has no roles", desc.Path)
		}
	}
}

// TestConcurrentAccess verifies registry is thread-safe
func TestConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	// Register initial decorator
	r.register("var", &mockValueDecorator{path: "var"})

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, ok := r.Lookup("var")
			if !ok {
				t.Error("concurrent lookup failed")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestIsRegistered verifies IsRegistered method
func TestIsRegistered(t *testing.T) {
	r := NewRegistry()

	r.register("env", &mockValueDecorator{path: "env"})

	if !r.IsRegistered("env") {
		t.Error("IsRegistered(env) = false, want true")
	}

	if r.IsRegistered("nonexistent") {
		t.Error("IsRegistered(nonexistent) = true, want false")
	}
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

// Mock decorators for testing

type mockValueDecorator struct {
	path string
}

// Compile-time check that mockValueDecorator implements Value
var _ Value = (*mockValueDecorator)(nil)

// Compile-time check that mockValueDecorator implements Value
var _ Value = (*mockValueDecorator)(nil)

func (m *mockValueDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockValueDecorator) Resolve(ctx ValueEvalContext, calls ...ValueCall) ([]ResolveResult, error) {
	results := make([]ResolveResult, len(calls))
	for i := range calls {
		results[i] = ResolveResult{
			Value:  "mock-value",
			Origin: "mock.value",
			Error:  nil,
		}
	}
	return results, nil
}

type mockExecDecorator struct {
	path string
}

func (m *mockExecDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockExecDecorator) Wrap(next ExecNode, params map[string]any) ExecNode {
	return nil // Stub
}

type mockTransportDecorator struct {
	path string
}

func (m *mockTransportDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockTransportDecorator) Open(parent Session, params map[string]any) (Session, error) {
	return nil, nil // Stub
}

func (m *mockTransportDecorator) Wrap(next ExecNode, params map[string]any) ExecNode {
	return nil // Stub
}

type mockEndpointDecorator struct {
	path string
}

func (m *mockEndpointDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockEndpointDecorator) Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error) {
	return nil, nil // Stub
}

type mockMultiRoleDecorator struct {
	path string
}

func (m *mockMultiRoleDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockMultiRoleDecorator) Resolve(ctx ValueEvalContext, calls ...ValueCall) ([]ResolveResult, error) {
	results := make([]ResolveResult, len(calls))
	for i := range calls {
		results[i] = ResolveResult{
			Value:  map[string]any{"size": 1024},
			Origin: "mock.multi",
			Error:  nil,
		}
	}
	return results, nil
}

func (m *mockMultiRoleDecorator) Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error) {
	return nil, nil // Stub
}

// Mock decorator with transport scope for testing
type mockScopedValueDecorator struct {
	path  string
	scope TransportScope
}

func (m *mockScopedValueDecorator) Descriptor() Descriptor {
	return Descriptor{
		Path: m.path,
		Capabilities: Capabilities{
			TransportScope: m.scope,
		},
	}
}

func (m *mockScopedValueDecorator) Resolve(ctx ValueEvalContext, calls ...ValueCall) ([]ResolveResult, error) {
	results := make([]ResolveResult, len(calls))
	for i := range calls {
		results[i] = ResolveResult{
			Value:  "scoped-value",
			Origin: "mock.scoped",
			Error:  nil,
		}
	}
	return results, nil
}

// TestTransportScopeAllows verifies TransportScope.Allows() logic
func TestTransportScopeAllows(t *testing.T) {
	tests := []struct {
		name     string
		scope    TransportScope
		current  TransportScope
		expected bool
	}{
		// TransportScopeAny allows everything
		{"Any allows Local", TransportScopeAny, TransportScopeLocal, true},
		{"Any allows SSH", TransportScopeAny, TransportScopeSSH, true},
		{"Any allows Remote", TransportScopeAny, TransportScopeRemote, true},
		{"Any allows Any", TransportScopeAny, TransportScopeAny, true},

		// TransportScopeLocal only allows Local
		{"Local allows Local", TransportScopeLocal, TransportScopeLocal, true},
		{"Local denies SSH", TransportScopeLocal, TransportScopeSSH, false},
		{"Local denies Remote", TransportScopeLocal, TransportScopeRemote, false},

		// TransportScopeSSH only allows SSH
		{"SSH allows SSH", TransportScopeSSH, TransportScopeSSH, true},
		{"SSH denies Local", TransportScopeSSH, TransportScopeLocal, false},

		// TransportScopeRemote allows any remote (SSH, Docker, etc.)
		{"Remote allows SSH", TransportScopeRemote, TransportScopeSSH, true},
		{"Remote denies Local", TransportScopeRemote, TransportScopeLocal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.scope.Allows(tt.current)
			if result != tt.expected {
				t.Errorf("TransportScope(%v).Allows(%v) = %v, want %v",
					tt.scope, tt.current, result, tt.expected)
			}
		})
	}
}

// TestResolveValueSuccess verifies successful value resolution
func TestResolveValueSuccess(t *testing.T) {
	r := NewRegistry()

	// Register a value decorator
	dec := &mockValueDecorator{path: "var"}
	r.register("var", dec)

	// Create context
	session := NewLocalSession()
	ctx := ValueEvalContext{
		Session: session,
		Vars:    map[string]any{"name": "test"},
	}

	// Create call
	varName := "name"
	call := ValueCall{
		Path:    "var",
		Primary: &varName,
		Params:  map[string]any{},
	}

	// Resolve value
	resolved, err := r.ResolveValue(ctx, call, TransportScopeLocal)
	if err != nil {
		t.Fatalf("ResolveValue failed: %v", err)
	}

	// Verify result
	if resolved.Value != "mock-value" {
		t.Errorf("Value: got %v, want %q", resolved.Value, "mock-value")
	}
}

// TestResolveValueNotFound verifies error when decorator doesn't exist
func TestResolveValueNotFound(t *testing.T) {
	r := NewRegistry()

	session := NewLocalSession()
	ctx := ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	call := ValueCall{
		Path:    "nonexistent",
		Primary: nil,
		Params:  map[string]any{},
	}

	_, err := r.ResolveValue(ctx, call, TransportScopeLocal)
	if err == nil {
		t.Fatal("Expected error for nonexistent decorator")
	}

	expectedMsg := `decorator "nonexistent" not found`
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

// TestResolveValueNotValueDecorator verifies error when decorator doesn't implement Value
func TestResolveValueNotValueDecorator(t *testing.T) {
	r := NewRegistry()

	// Register an Exec decorator (not Value)
	dec := &mockExecDecorator{path: "retry"}
	r.register("retry", dec)

	session := NewLocalSession()
	ctx := ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	call := ValueCall{
		Path:    "retry",
		Primary: nil,
		Params:  map[string]any{},
	}

	_, err := r.ResolveValue(ctx, call, TransportScopeLocal)
	if err == nil {
		t.Fatal("Expected error for non-value decorator")
	}

	expectedMsg := `decorator "retry" does not implement Value interface`
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

// TestResolveValueScopeViolation verifies transport scope enforcement
func TestResolveValueScopeViolation(t *testing.T) {
	r := NewRegistry()

	// Register a Local-only decorator
	dec := &mockScopedValueDecorator{
		path:  "local.file",
		scope: TransportScopeLocal,
	}
	r.register("local.file", dec)

	session := NewLocalSession()
	ctx := ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	call := ValueCall{
		Path:    "local.file",
		Primary: nil,
		Params:  map[string]any{},
	}

	// Try to resolve in SSH context (should fail)
	_, err := r.ResolveValue(ctx, call, TransportScopeSSH)
	if err == nil {
		t.Fatal("Expected error for scope violation")
	}

	expectedMsg := `decorator "local.file" cannot be used in current transport scope (requires Local, current: SSH)`
	if err.Error() != expectedMsg {
		t.Errorf("Error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

// TestResolveValueScopeAllowed verifies scope enforcement allows valid scopes
func TestResolveValueScopeAllowed(t *testing.T) {
	r := NewRegistry()

	// Register an Any-scope decorator
	dec := &mockScopedValueDecorator{
		path:  "env",
		scope: TransportScopeAny,
	}
	r.register("env", dec)

	session := NewLocalSession()
	ctx := ValueEvalContext{
		Session: session,
		Vars:    map[string]any{},
	}

	call := ValueCall{
		Path:    "env",
		Primary: nil,
		Params:  map[string]any{},
	}

	// Should work in Local context
	_, err := r.ResolveValue(ctx, call, TransportScopeLocal)
	if err != nil {
		t.Errorf("ResolveValue in Local failed: %v", err)
	}

	// Should work in SSH context
	_, err = r.ResolveValue(ctx, call, TransportScopeSSH)
	if err != nil {
		t.Errorf("ResolveValue in SSH failed: %v", err)
	}
}
