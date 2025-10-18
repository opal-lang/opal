package types

import (
	"testing"
)

func TestRegisterDecorator(t *testing.T) {
	// Create a fresh registry for testing
	r := NewRegistry()

	// Register a simple decorator
	r.Register("var")

	// Verify it's registered
	if !r.IsRegistered("var") {
		t.Error("decorator 'var' should be registered")
	}
}

func TestRegisterMultipleDecorators(t *testing.T) {
	r := NewRegistry()

	r.Register("var")
	r.Register("env")

	if !r.IsRegistered("var") {
		t.Error("decorator 'var' should be registered")
	}

	if !r.IsRegistered("env") {
		t.Error("decorator 'env' should be registered")
	}
}

func TestUnregisteredDecorator(t *testing.T) {
	r := NewRegistry()

	// Lookup non-existent decorator
	if r.IsRegistered("unknown") {
		t.Error("IsRegistered should return false for unregistered decorator")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should exist
	g := Global()
	if g == nil {
		t.Fatal("Global() should return a registry")
	}

	// Global registry starts empty (built-ins register from their own packages)
	// This test just verifies the global registry exists and works
}

func TestRegisterValueDecorator(t *testing.T) {
	r := NewRegistry()

	// Dummy handlers
	dummyHandler := func(ctx Context, args Args) (Value, error) {
		return nil, nil
	}

	// Register value decorators
	r.RegisterValue("var", dummyHandler)
	r.RegisterValue("env", dummyHandler)
	r.RegisterValue("file.read", dummyHandler)

	// Verify they're registered as value decorators
	if !r.IsValueDecorator("var") {
		t.Error("'var' should be registered as value decorator")
	}

	if !r.IsValueDecorator("file.read") {
		t.Error("'file.read' should be registered as value decorator")
	}

	// Verify they're also registered (general check)
	if !r.IsRegistered("var") {
		t.Error("'var' should be registered")
	}
}

func TestRegisterExecutionDecorator(t *testing.T) {
	r := NewRegistry()

	// Dummy handler
	dummyHandler := func(ctx Context, args Args) error {
		return nil
	}

	// Register execution decorators
	r.RegisterExecution("file.write", dummyHandler)
	r.RegisterExecution("aws.instance.deploy", dummyHandler)

	// Verify they're registered
	if !r.IsRegistered("file.write") {
		t.Error("'file.write' should be registered")
	}

	// Verify they're NOT value decorators
	if r.IsValueDecorator("file.write") {
		t.Error("'file.write' should NOT be a value decorator")
	}

	if r.IsValueDecorator("aws.instance.deploy") {
		t.Error("'aws.instance.deploy' should NOT be a value decorator")
	}
}

func TestMixedDecorators(t *testing.T) {
	r := NewRegistry()

	// Dummy handlers
	valueHandler := func(ctx Context, args Args) (Value, error) {
		return nil, nil
	}
	execHandler := func(ctx Context, args Args) error {
		return nil
	}

	// Same namespace, different methods
	r.RegisterValue("file.read", valueHandler)
	r.RegisterExecution("file.write", execHandler)

	// Both should be registered
	if !r.IsRegistered("file.read") {
		t.Error("'file.read' should be registered")
	}
	if !r.IsRegistered("file.write") {
		t.Error("'file.write' should be registered")
	}

	// But only file.read is a value decorator
	if !r.IsValueDecorator("file.read") {
		t.Error("'file.read' should be a value decorator")
	}
	if r.IsValueDecorator("file.write") {
		t.Error("'file.write' should NOT be a value decorator")
	}
}

func TestRegisterValueWithHandler(t *testing.T) {
	r := NewRegistry()

	// Create a test handler
	called := false
	handler := func(ctx Context, args Args) (Value, error) {
		called = true
		return "test-value", nil
	}

	// Register with handler
	r.RegisterValue("test.decorator", handler)

	// Verify it's registered
	if !r.IsRegistered("test.decorator") {
		t.Error("'test.decorator' should be registered")
	}

	// Verify it's a value decorator
	if !r.IsValueDecorator("test.decorator") {
		t.Error("'test.decorator' should be a value decorator")
	}

	// Retrieve and invoke the handler
	retrievedHandler, ok := r.GetValueHandler("test.decorator")
	if !ok {
		t.Fatal("should be able to retrieve value handler")
	}

	result, err := retrievedHandler(Context{}, Args{})
	if err != nil {
		t.Errorf("handler returned error: %v", err)
	}

	if !called {
		t.Error("handler should have been called")
	}

	if result != "test-value" {
		t.Errorf("expected 'test-value', got %v", result)
	}
}

func TestRegisterExecutionWithHandler(t *testing.T) {
	r := NewRegistry()

	// Create a test handler
	called := false
	handler := func(ctx Context, args Args) error {
		called = true
		return nil
	}

	// Register with handler
	r.RegisterExecution("test.action", handler)

	// Verify it's registered
	if !r.IsRegistered("test.action") {
		t.Error("'test.action' should be registered")
	}

	// Verify it's NOT a value decorator
	if r.IsValueDecorator("test.action") {
		t.Error("'test.action' should NOT be a value decorator")
	}

	// Retrieve and invoke the handler
	retrievedHandler, ok := r.GetExecutionHandler("test.action")
	if !ok {
		t.Fatal("should be able to retrieve execution handler")
	}

	err := retrievedHandler(Context{}, Args{})
	if err != nil {
		t.Errorf("handler returned error: %v", err)
	}

	if !called {
		t.Error("handler should have been called")
	}
}

func TestGetValueHandlerForExecutionDecorator(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx Context, args Args) error {
		return nil
	}

	r.RegisterExecution("test.action", handler)

	// Should not be able to get value handler for execution decorator
	_, ok := r.GetValueHandler("test.action")
	if ok {
		t.Error("should not be able to get value handler for execution decorator")
	}
}

func TestGetExecutionHandlerForValueDecorator(t *testing.T) {
	r := NewRegistry()

	handler := func(ctx Context, args Args) (Value, error) {
		return nil, nil
	}

	r.RegisterValue("test.value", handler)

	// Should not be able to get execution handler for value decorator
	_, ok := r.GetExecutionHandler("test.value")
	if ok {
		t.Error("should not be able to get execution handler for value decorator")
	}
}

func TestHandlerWithArgs(t *testing.T) {
	r := NewRegistry()

	// Handler that uses args
	handler := func(ctx Context, args Args) (Value, error) {
		if args.Primary == nil {
			return nil, nil
		}
		primary := (*args.Primary).(string)
		return "processed-" + primary, nil
	}

	r.RegisterValue("test.processor", handler)

	retrievedHandler, ok := r.GetValueHandler("test.processor")
	if !ok {
		t.Fatal("should be able to retrieve handler")
	}

	// Test with primary property
	primaryVal := Value("input")
	result, err := retrievedHandler(Context{}, Args{Primary: &primaryVal})
	if err != nil {
		t.Errorf("handler returned error: %v", err)
	}

	if result != "processed-input" {
		t.Errorf("expected 'processed-input', got %v", result)
	}

	// Test without primary property
	result, err = retrievedHandler(Context{}, Args{})
	if err != nil {
		t.Errorf("handler returned error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestRegisterSDKHandler verifies SDK-based handler registration
func TestRegisterSDKHandler(t *testing.T) {
	registry := NewRegistry()

	// Register an execution decorator with SDK handler
	handler := func(ctx interface{}, block []interface{}) (int, error) {
		return 0, nil
	}

	registry.RegisterSDKHandler("shell", DecoratorKindExecution, handler)

	// Verify it's registered
	if !registry.IsRegistered("shell") {
		t.Error("shell should be registered")
	}

	// Retrieve the handler
	retrievedHandler, kind, exists := registry.GetSDKHandler("shell")
	if !exists {
		t.Fatal("shell handler should exist")
	}
	if kind != DecoratorKindExecution {
		t.Errorf("expected execution kind, got %v", kind)
	}
	if retrievedHandler == nil {
		t.Error("handler should not be nil")
	}
}

// TestRegisterSDKHandlerWithSchema verifies SDK handler with schema
func TestRegisterSDKHandlerWithSchema(t *testing.T) {
	registry := NewRegistry()

	schema := NewSchema("test", KindExecution).
		Description("Test decorator").
		Build()

	handler := func(ctx interface{}, block []interface{}) (int, error) {
		return 0, nil
	}

	err := registry.RegisterSDKHandlerWithSchema(schema, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify registration
	retrievedHandler, kind, exists := registry.GetSDKHandler("test")
	if !exists {
		t.Fatal("test handler should exist")
	}
	if kind != DecoratorKindExecution {
		t.Errorf("expected execution kind, got %v", kind)
	}
	if retrievedHandler == nil {
		t.Error("handler should not be nil")
	}

	// Verify schema
	retrievedSchema, schemaExists := registry.GetSchema("test")
	if !schemaExists {
		t.Fatal("schema should exist")
	}
	if retrievedSchema.Path != "test" {
		t.Errorf("expected path 'test', got %q", retrievedSchema.Path)
	}
}

// TestGetSDKHandlerNotFound verifies missing handler returns false
func TestGetSDKHandlerNotFound(t *testing.T) {
	registry := NewRegistry()

	handler, kind, exists := registry.GetSDKHandler("nonexistent")
	if exists {
		t.Error("nonexistent handler should not exist")
	}
	if handler != nil {
		t.Error("handler should be nil for nonexistent decorator")
	}
	if kind != 0 {
		t.Error("kind should be zero value for nonexistent decorator")
	}
}
