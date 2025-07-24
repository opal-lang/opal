package decorators

import (
	"fmt"
	"sync"
)

// Registry manages all available decorators
type Registry struct {
	functionDecorators map[string]FunctionDecorator
	blockDecorators    map[string]BlockDecorator
	patternDecorators  map[string]PatternDecorator
	mu                 sync.RWMutex
}

// Global registry instance
var globalRegistry = NewRegistry()

// NewRegistry creates a new decorator registry
func NewRegistry() *Registry {
	return &Registry{
		functionDecorators: make(map[string]FunctionDecorator),
		blockDecorators:    make(map[string]BlockDecorator),
		patternDecorators:  make(map[string]PatternDecorator),
	}
}

// RegisterFunction registers a function decorator
func (r *Registry) RegisterFunction(decorator FunctionDecorator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.functionDecorators[decorator.Name()] = decorator
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

// GetFunction retrieves a function decorator by name
func (r *Registry) GetFunction(name string) (FunctionDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.functionDecorators[name]
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

	if decorator, exists := r.functionDecorators[name]; exists {
		return decorator, FunctionType, true
	}
	if decorator, exists := r.blockDecorators[name]; exists {
		return decorator, BlockType, true
	}
	if decorator, exists := r.patternDecorators[name]; exists {
		return decorator, PatternType, true
	}

	return nil, FunctionType, false
}

// ListAll returns all registered decorators by type
func (r *Registry) ListAll() ([]FunctionDecorator, []BlockDecorator, []PatternDecorator) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var functions []FunctionDecorator
	var blocks []BlockDecorator
	var patterns []PatternDecorator

	for _, decorator := range r.functionDecorators {
		functions = append(functions, decorator)
	}
	for _, decorator := range r.blockDecorators {
		blocks = append(blocks, decorator)
	}
	for _, decorator := range r.patternDecorators {
		patterns = append(patterns, decorator)
	}

	return functions, blocks, patterns
}

// Global registry functions for convenience

// RegisterFunction registers a function decorator in the global registry
func RegisterFunction(decorator FunctionDecorator) {
	globalRegistry.RegisterFunction(decorator)
}

// RegisterBlock registers a block decorator in the global registry
func RegisterBlock(decorator BlockDecorator) {
	globalRegistry.RegisterBlock(decorator)
}

// RegisterPattern registers a pattern decorator in the global registry
func RegisterPattern(decorator PatternDecorator) {
	globalRegistry.RegisterPattern(decorator)
}

// GetFunction retrieves a function decorator from the global registry
func GetFunction(name string) (FunctionDecorator, error) {
	decorator, exists := globalRegistry.GetFunction(name)
	if !exists {
		return nil, fmt.Errorf("function decorator @%s not found", name)
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
		return nil, FunctionType, fmt.Errorf("decorator @%s not found", name)
	}
	return decorator, decoratorType, nil
}

// ListAll returns all registered decorators from the global registry
func ListAll() ([]FunctionDecorator, []BlockDecorator, []PatternDecorator) {
	return globalRegistry.ListAll()
}

// Type checking functions for lexer and parser

// IsFunctionDecorator checks if a decorator is a function decorator
func IsFunctionDecorator(name string) bool {
	_, exists := globalRegistry.GetFunction(name)
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

// GetFunctionDecorator is an alias for GetFunction but returns only the decorator (for compatibility)
func GetFunctionDecorator(name string) (FunctionDecorator, bool) {
	return globalRegistry.GetFunction(name)
}

// GetBlockDecorator is an alias for GetBlock but returns only the decorator (for compatibility)
func GetBlockDecorator(name string) (BlockDecorator, bool) {
	return globalRegistry.GetBlock(name)
}

// GetPatternDecorator is an alias for GetPattern but returns only the decorator (for compatibility)
func GetPatternDecorator(name string) (PatternDecorator, bool) {
	return globalRegistry.GetPattern(name)
}
