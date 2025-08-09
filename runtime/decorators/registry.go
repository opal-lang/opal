package decorators

import (
	"fmt"
	"sync"
)

// Registry manages all available decorators
type Registry struct {
	valueDecorators   map[string]ValueDecorator
	actionDecorators  map[string]ActionDecorator
	blockDecorators   map[string]BlockDecorator
	patternDecorators map[string]PatternDecorator
	mu                sync.RWMutex
}

// Global registry instance
var globalRegistry = NewRegistry()

// NewRegistry creates a new decorator registry
func NewRegistry() *Registry {
	return &Registry{
		valueDecorators:   make(map[string]ValueDecorator),
		actionDecorators:  make(map[string]ActionDecorator),
		blockDecorators:   make(map[string]BlockDecorator),
		patternDecorators: make(map[string]PatternDecorator),
	}
}

// RegisterValue registers a value decorator
func (r *Registry) RegisterValue(decorator ValueDecorator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.valueDecorators[decorator.Name()] = decorator
}

// RegisterAction registers an action decorator
func (r *Registry) RegisterAction(decorator ActionDecorator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actionDecorators[decorator.Name()] = decorator
}

// RegisterBlock registers a block decorator
func (r *Registry) RegisterBlock(decorator BlockDecorator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blockDecorators[decorator.Name()] = decorator
}

// RegisterPattern registers a pattern decorator
func (r *Registry) RegisterPattern(decorator PatternDecorator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.patternDecorators[decorator.Name()] = decorator
}

// GetValue retrieves a value decorator by name
func (r *Registry) GetValue(name string) (ValueDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.valueDecorators[name]
	return decorator, exists
}

// GetAction retrieves an action decorator by name
func (r *Registry) GetAction(name string) (ActionDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.actionDecorators[name]
	return decorator, exists
}

// GetBlock retrieves a block decorator by name
func (r *Registry) GetBlock(name string) (BlockDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.blockDecorators[name]
	return decorator, exists
}

// GetPattern retrieves a pattern decorator by name
func (r *Registry) GetPattern(name string) (PatternDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.patternDecorators[name]
	return decorator, exists
}

// GetAny retrieves any decorator by name, returning the decorator and its type
func (r *Registry) GetAny(name string) (Decorator, DecoratorType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if decorator, exists := r.valueDecorators[name]; exists {
		return decorator, ValueType, true
	}
	if decorator, exists := r.actionDecorators[name]; exists {
		return decorator, ActionType, true
	}
	if decorator, exists := r.blockDecorators[name]; exists {
		return decorator, BlockType, true
	}
	if decorator, exists := r.patternDecorators[name]; exists {
		return decorator, PatternType, true
	}

	return nil, ValueType, false
}

// ListAll returns all registered decorators by type
func (r *Registry) ListAll() ([]ValueDecorator, []ActionDecorator, []BlockDecorator, []PatternDecorator) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var values []ValueDecorator
	var actions []ActionDecorator
	var blocks []BlockDecorator
	var patterns []PatternDecorator

	for _, decorator := range r.valueDecorators {
		values = append(values, decorator)
	}
	for _, decorator := range r.actionDecorators {
		actions = append(actions, decorator)
	}
	for _, decorator := range r.blockDecorators {
		blocks = append(blocks, decorator)
	}
	for _, decorator := range r.patternDecorators {
		patterns = append(patterns, decorator)
	}

	return values, actions, blocks, patterns
}

// Global registry functions for convenience

// RegisterValue registers a value decorator in the global registry
func RegisterValue(decorator ValueDecorator) {
	globalRegistry.RegisterValue(decorator)
}

// RegisterAction registers an action decorator in the global registry
func RegisterAction(decorator ActionDecorator) {
	globalRegistry.RegisterAction(decorator)
}

// RegisterBlock registers a block decorator in the global registry
func RegisterBlock(decorator BlockDecorator) {
	globalRegistry.RegisterBlock(decorator)
}

// RegisterPattern registers a pattern decorator in the global registry
func RegisterPattern(decorator PatternDecorator) {
	globalRegistry.RegisterPattern(decorator)
}

// GetValue retrieves a value decorator from the global registry
func GetValue(name string) (ValueDecorator, error) {
	decorator, exists := globalRegistry.GetValue(name)
	if !exists {
		return nil, fmt.Errorf("value decorator @%s not found", name)
	}
	return decorator, nil
}

// GetAction retrieves an action decorator from the global registry
func GetAction(name string) (ActionDecorator, error) {
	decorator, exists := globalRegistry.GetAction(name)
	if !exists {
		return nil, fmt.Errorf("action decorator @%s not found", name)
	}
	return decorator, nil
}

// GetBlock retrieves a block decorator from the global registry
func GetBlock(name string) (BlockDecorator, error) {
	decorator, exists := globalRegistry.GetBlock(name)
	if !exists {
		return nil, fmt.Errorf("block decorator @%s not found", name)
	}
	return decorator, nil
}

// GetPattern retrieves a pattern decorator from the global registry
func GetPattern(name string) (PatternDecorator, error) {
	decorator, exists := globalRegistry.GetPattern(name)
	if !exists {
		return nil, fmt.Errorf("pattern decorator @%s not found", name)
	}
	return decorator, nil
}

// GetAny retrieves any decorator from the global registry
func GetAny(name string) (Decorator, DecoratorType, error) {
	decorator, decoratorType, exists := globalRegistry.GetAny(name)
	if !exists {
		return nil, ValueType, fmt.Errorf("decorator @%s not found", name)
	}
	return decorator, decoratorType, nil
}

// ListAll returns all registered decorators from the global registry
func ListAll() ([]ValueDecorator, []ActionDecorator, []BlockDecorator, []PatternDecorator) {
	return globalRegistry.ListAll()
}

// Type checking functions for lexer and parser

// IsValueDecorator checks if a decorator is a value decorator
func IsValueDecorator(name string) bool {
	_, exists := globalRegistry.GetValue(name)
	return exists
}

// IsActionDecorator checks if a decorator is an action decorator
func IsActionDecorator(name string) bool {
	_, exists := globalRegistry.GetAction(name)
	return exists
}

// IsBlockDecorator checks if a decorator is a block decorator
func IsBlockDecorator(name string) bool {
	_, exists := globalRegistry.GetBlock(name)
	return exists
}

// IsPatternDecorator checks if a decorator is a pattern decorator
func IsPatternDecorator(name string) bool {
	_, exists := globalRegistry.GetPattern(name)
	return exists
}

// IsDecorator checks if a name is any type of decorator
func IsDecorator(name string) bool {
	_, _, exists := globalRegistry.GetAny(name)
	return exists
}

// GetValueDecorator is an alias for GetValue but returns only the decorator (for compatibility)
func GetValueDecorator(name string) (ValueDecorator, bool) {
	return globalRegistry.GetValue(name)
}

// GetActionDecorator is an alias for GetAction but returns only the decorator (for compatibility)
func GetActionDecorator(name string) (ActionDecorator, bool) {
	return globalRegistry.GetAction(name)
}

// GetBlockDecorator is an alias for GetBlock but returns only the decorator (for compatibility)
func GetBlockDecorator(name string) (BlockDecorator, bool) {
	return globalRegistry.GetBlock(name)
}

// GetPatternDecorator is an alias for GetPattern but returns only the decorator (for compatibility)
func GetPatternDecorator(name string) (PatternDecorator, bool) {
	return globalRegistry.GetPattern(name)
}
