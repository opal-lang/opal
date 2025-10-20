package types

import (
	"fmt"
	"sync"
)

// Value represents a runtime value in Opal
// This is a placeholder - will be expanded as we implement the runtime
type Value interface{}

// Block represents a code block passed to a decorator
// This is a placeholder - will be expanded when we implement block execution
type Block interface {
	// Execute runs the block in the given context
	Execute(ctx Context) error
}

// Context holds the execution context for decorator handlers
type Context struct {
	Variables  map[string]Value  // Variable bindings: var x = "value"
	Env        map[string]string // Environment variables
	WorkingDir string            // Current working directory
}

// Args holds the arguments passed to a decorator handler
type Args struct {
	Primary *Value           // Primary property: @env.HOME → "HOME"
	Params  map[string]Value // Named parameters: (default="", times=3)
	Block   *Block           // Lambda/block for execution decorators
}

// ValueHandler is a function that implements a value decorator
// Returns data with no side effects
type ValueHandler func(ctx Context, args Args) (Value, error)

// ExecutionHandler is a function that implements an execution decorator
// Performs actions with side effects
type ExecutionHandler func(ctx Context, args Args) error

// DecoratorKind represents the type of decorator
type DecoratorKind int

const (
	// DecoratorKindValue returns data with no side effects (can be interpolated in strings)
	DecoratorKindValue DecoratorKind = iota
	// DecoratorKindExecution performs actions with side effects (cannot be interpolated)
	DecoratorKindExecution
)

// DecoratorInfo holds metadata about a registered decorator
type DecoratorInfo struct {
	Path             string           // Full path: "var", "env", "file.read", "aws.instance.data"
	Kind             DecoratorKind    // Value or Execution
	Schema           DecoratorSchema  // Schema describing the decorator's interface
	ValueHandler     ValueHandler     // Handler for value decorators (nil for execution)
	ExecutionHandler ExecutionHandler // Handler for execution decorators (nil for value) - OLD STYLE
	RawHandler       interface{}      // Raw handler (type depends on decorator kind) - NEW STYLE
}

// Registry holds registered decorator paths and their metadata
type Registry struct {
	mu         sync.RWMutex
	decorators map[string]DecoratorInfo
}

// NewRegistry creates a new decorator registry
func NewRegistry() *Registry {
	return &Registry{
		decorators: make(map[string]DecoratorInfo),
	}
}

// RegisterValue registers a value decorator (returns data, no side effects)
// Can be used in string interpolation
func (r *Registry) RegisterValue(path string, handler ValueHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[path] = DecoratorInfo{
		Path:         path,
		Kind:         DecoratorKindValue,
		Schema:       DecoratorSchema{Path: path, Kind: "value"}, // Minimal schema
		ValueHandler: handler,
	}
}

// RegisterExecution registers an execution decorator (performs actions with side effects)
// Cannot be used in string interpolation
func (r *Registry) RegisterExecution(path string, handler ExecutionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[path] = DecoratorInfo{
		Path:             path,
		Kind:             DecoratorKindExecution,
		Schema:           DecoratorSchema{Path: path, Kind: "execution"}, // Minimal schema
		ExecutionHandler: handler,
	}
}

// GetValueHandler retrieves the handler for a value decorator
func (r *Registry) GetValueHandler(path string) (ValueHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	if !exists || info.Kind != DecoratorKindValue {
		return nil, false
	}
	return info.ValueHandler, true
}

// GetExecutionHandler retrieves the handler for an execution decorator
func (r *Registry) GetExecutionHandler(path string) (ExecutionHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	if !exists || info.Kind != DecoratorKindExecution {
		return nil, false
	}
	return info.ExecutionHandler, true
}

// Register adds a decorator (defaults to value for backward compatibility)
// Deprecated: Use RegisterValue or RegisterExecution instead
func (r *Registry) Register(name string) {
	// For backward compatibility, register with a nil handler
	// This allows existing tests to pass while we migrate to the new pattern
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[name] = DecoratorInfo{
		Path: name,
		Kind: DecoratorKindValue,
	}
}

// IsRegistered checks if a decorator path is registered
func (r *Registry) IsRegistered(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.decorators[path]
	return exists
}

// IsValueDecorator checks if a decorator path is registered as a value decorator
func (r *Registry) IsValueDecorator(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	return exists && info.Kind == DecoratorKindValue
}

// GetSchema retrieves the schema for a decorator
func (r *Registry) GetSchema(path string) (DecoratorSchema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.decorators[path]
	return info.Schema, exists
}

// GetInfo retrieves the full decorator info
func (r *Registry) GetInfo(path string) (DecoratorInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	return info, exists
}

// RegisterValueWithSchema registers a value decorator with a schema
func (r *Registry) RegisterValueWithSchema(schema DecoratorSchema, handler ValueHandler) error {
	// Validate schema
	if err := ValidateSchema(schema); err != nil {
		return fmt.Errorf("invalid schema for %s: %w", schema.Path, err)
	}

	if schema.Kind != "value" {
		return fmt.Errorf("schema kind must be 'value' for RegisterValueWithSchema, got %q", schema.Kind)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.decorators[schema.Path] = DecoratorInfo{
		Path:         schema.Path,
		Kind:         DecoratorKindValue,
		Schema:       schema,
		ValueHandler: handler,
	}
	return nil
}

// RegisterExecutionWithSchema registers an execution decorator with a schema
func (r *Registry) RegisterExecutionWithSchema(schema DecoratorSchema, handler ExecutionHandler) error {
	// Validate schema
	if err := ValidateSchema(schema); err != nil {
		return fmt.Errorf("invalid schema for %s: %w", schema.Path, err)
	}

	if schema.Kind != KindExecution {
		return fmt.Errorf("schema kind must be 'execution' for RegisterExecutionWithSchema, got %q", schema.Kind)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.decorators[schema.Path] = DecoratorInfo{
		Path:             schema.Path,
		Kind:             DecoratorKindExecution,
		Schema:           schema,
		ExecutionHandler: handler,
	}
	return nil
}

// RegisterSDKHandler registers a decorator with an SDK-based handler.
// This is the new style that uses sdk.ExecutionContext and sdk.Step.
// The handler type depends on the decorator kind:
// - Value decorators: sdk.ValueHandler
// - Execution decorators: sdk.ExecutionHandler
//
// This avoids circular dependencies: core/types imports core/sdk (both in core).
func (r *Registry) RegisterSDKHandler(path string, kind DecoratorKind, handler interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kindStr := KindValue
	if kind == DecoratorKindExecution {
		kindStr = KindExecution
	}

	r.decorators[path] = DecoratorInfo{
		Path:       path,
		Kind:       kind,
		Schema:     DecoratorSchema{Path: path, Kind: kindStr},
		RawHandler: handler,
	}
}

// RegisterSDKHandlerWithSchema registers a decorator with schema and SDK-based handler.
func (r *Registry) RegisterSDKHandlerWithSchema(schema DecoratorSchema, handler interface{}) error {
	// Validate schema
	if err := ValidateSchema(schema); err != nil {
		return fmt.Errorf("invalid schema for %s: %w", schema.Path, err)
	}

	kind := DecoratorKindValue
	if schema.Kind == KindExecution {
		kind = DecoratorKindExecution
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.decorators[schema.Path] = DecoratorInfo{
		Path:       schema.Path,
		Kind:       kind,
		Schema:     schema,
		RawHandler: handler,
	}
	return nil
}

// GetSDKHandler retrieves the SDK-based handler for a decorator.
// Returns the handler, decorator kind, and whether it exists.
// Caller must type-assert to the appropriate handler type:
// - Value: sdk.ValueHandler
// - Execution: sdk.ExecutionHandler
func (r *Registry) GetSDKHandler(path string) (handler interface{}, kind DecoratorKind, exists bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	if !exists || info.RawHandler == nil {
		return nil, 0, false
	}
	return info.RawHandler, info.Kind, true
}

// Global registry instance
var globalRegistry = NewRegistry()

// Global returns the global decorator registry
func Global() *Registry {
	return globalRegistry
}
