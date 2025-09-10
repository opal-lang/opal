package decorators

import (
	"fmt"
	"sync"
)

// ================================================================================================
// DECORATOR REGISTRY
// ================================================================================================

// Registry holds all registered decorators with fully-qualified names and collision detection
type Registry struct {
	mu sync.RWMutex

	Actions  map[string]ActionDecorator
	Blocks   map[string]BlockDecorator
	Values   map[string]ValueDecorator
	Patterns map[string]PatternDecorator
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		Actions:  make(map[string]ActionDecorator),
		Blocks:   make(map[string]BlockDecorator),
		Values:   make(map[string]ValueDecorator),
		Patterns: make(map[string]PatternDecorator),
	}
}

// RegisterValue registers a value decorator with collision detection across all categories
func (r *Registry) RegisterValue(decorator ValueDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.Values[name] = decorator
	return nil
}

// RegisterAction registers an action decorator with collision detection across all categories
func (r *Registry) RegisterAction(decorator ActionDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.Actions[name] = decorator
	return nil
}

// RegisterBlock registers a block decorator with collision detection across all categories
func (r *Registry) RegisterBlock(decorator BlockDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.Blocks[name] = decorator
	return nil
}

// RegisterPattern registers a pattern decorator with collision detection across all categories
func (r *Registry) RegisterPattern(decorator PatternDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.Patterns[name] = decorator
	return nil
}

// checkCollision checks if a name is already registered in any category
// Must be called with lock held
func (r *Registry) checkCollision(name string) error {
	if _, exists := r.Values[name]; exists {
		return fmt.Errorf("decorator %q already registered as Value", name)
	}
	if _, exists := r.Actions[name]; exists {
		return fmt.Errorf("decorator %q already registered as Action", name)
	}
	if _, exists := r.Blocks[name]; exists {
		return fmt.Errorf("decorator %q already registered as Block", name)
	}
	if _, exists := r.Patterns[name]; exists {
		return fmt.Errorf("decorator %q already registered as Pattern", name)
	}
	return nil
}

// GetValue retrieves a value decorator
func (r *Registry) GetValue(name string) (ValueDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.Values[name]
	return decorator, exists
}

// GetAction retrieves an action decorator
func (r *Registry) GetAction(name string) (ActionDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.Actions[name]
	return decorator, exists
}

// GetBlock retrieves a block decorator
func (r *Registry) GetBlock(name string) (BlockDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.Blocks[name]
	return decorator, exists
}

// GetPattern retrieves a pattern decorator
func (r *Registry) GetPattern(name string) (PatternDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.Patterns[name]
	return decorator, exists
}

// ListAll returns all registered decorators organized by category
func (r *Registry) ListAll() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)

	// Collect values
	var values []string
	for name := range r.Values {
		values = append(values, name)
	}
	result["values"] = values

	// Collect actions
	var actions []string
	for name := range r.Actions {
		actions = append(actions, name)
	}
	result["actions"] = actions

	// Collect blocks
	var blocks []string
	for name := range r.Blocks {
		blocks = append(blocks, name)
	}
	result["blocks"] = blocks

	// Collect patterns
	var patterns []string
	for name := range r.Patterns {
		patterns = append(patterns, name)
	}
	result["patterns"] = patterns

	return result
}

// ================================================================================================
// GLOBAL REGISTRY - Database/SQL Driver Pattern
// ================================================================================================

// Global registry instance - decorators register themselves via init() functions
var globalRegistry = NewRegistry()

// Register registers a value decorator in the global registry (called from init functions)
func Register(decorator ValueDecorator) {
	if err := globalRegistry.RegisterValue(decorator); err != nil {
		panic(fmt.Sprintf("failed to register value decorator: %v", err))
	}
}

// RegisterAction registers an action decorator in the global registry (called from init functions)
func RegisterAction(decorator ActionDecorator) {
	if err := globalRegistry.RegisterAction(decorator); err != nil {
		panic(fmt.Sprintf("failed to register action decorator: %v", err))
	}
}

// RegisterBlock registers a block decorator in the global registry (called from init functions)
func RegisterBlock(decorator BlockDecorator) {
	if err := globalRegistry.RegisterBlock(decorator); err != nil {
		panic(fmt.Sprintf("failed to register block decorator: %v", err))
	}
}

// RegisterPattern registers a pattern decorator in the global registry (called from init functions)
func RegisterPattern(decorator PatternDecorator) {
	if err := globalRegistry.RegisterPattern(decorator); err != nil {
		panic(fmt.Sprintf("failed to register pattern decorator: %v", err))
	}
}

// GlobalRegistry returns the global registry instance
func GlobalRegistry() *Registry {
	return globalRegistry
}

// ================================================================================================
// CONVENIENCE FUNCTIONS
// ================================================================================================

// NewStandardRegistry creates a new registry pre-populated with all registered decorators
// This is useful for creating isolated registries while inheriting global registrations
func NewStandardRegistry() *Registry {
	r := NewRegistry()

	// Copy all decorators from global registry
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	for name, decorator := range globalRegistry.Values {
		r.Values[name] = decorator
	}
	for name, decorator := range globalRegistry.Actions {
		r.Actions[name] = decorator
	}
	for name, decorator := range globalRegistry.Blocks {
		r.Blocks[name] = decorator
	}
	for name, decorator := range globalRegistry.Patterns {
		r.Patterns[name] = decorator
	}

	return r
}

// IsValueDecorator checks if a decorator is registered as a value decorator
func (r *Registry) IsValueDecorator(name string) bool {
	_, exists := r.GetValue(name)
	return exists
}

// IsActionDecorator checks if a decorator is registered as an action decorator
func (r *Registry) IsActionDecorator(name string) bool {
	_, exists := r.GetAction(name)
	return exists
}

// IsBlockDecorator checks if a decorator is registered as a block decorator
func (r *Registry) IsBlockDecorator(name string) bool {
	_, exists := r.GetBlock(name)
	return exists
}

// IsPatternDecorator checks if a decorator is registered as a pattern decorator
func (r *Registry) IsPatternDecorator(name string) bool {
	_, exists := r.GetPattern(name)
	return exists
}

// IsDecorator checks if a name is registered as any type of decorator
func (r *Registry) IsDecorator(name string) bool {
	return r.IsValueDecorator(name) || r.IsActionDecorator(name) ||
		r.IsBlockDecorator(name) || r.IsPatternDecorator(name)
}

// GetDecoratorType returns the type of decorator ("value", "action", "block", "pattern") or empty string if not found
func (r *Registry) GetDecoratorType(name string) string {
	if r.IsValueDecorator(name) {
		return "value"
	}
	if r.IsActionDecorator(name) {
		return "action"
	}
	if r.IsBlockDecorator(name) {
		return "block"
	}
	if r.IsPatternDecorator(name) {
		return "pattern"
	}
	return ""
}

// ================================================================================================
// GLOBAL CONVENIENCE FUNCTIONS FOR LEXER/PARSER
// ================================================================================================

// IsDecorator checks if a name is registered as any type of decorator in the global registry
func IsDecorator(name string) bool {
	return globalRegistry.IsDecorator(name)
}

// IsValueDecorator checks if a decorator is registered as a value decorator in the global registry
func IsValueDecorator(name string) bool {
	return globalRegistry.IsValueDecorator(name)
}

// IsActionDecorator checks if a decorator is registered as an action decorator in the global registry
func IsActionDecorator(name string) bool {
	return globalRegistry.IsActionDecorator(name)
}

// IsBlockDecorator checks if a decorator is registered as a block decorator in the global registry
func IsBlockDecorator(name string) bool {
	return globalRegistry.IsBlockDecorator(name)
}

// IsPatternDecorator checks if a decorator is registered as a pattern decorator in the global registry
func IsPatternDecorator(name string) bool {
	return globalRegistry.IsPatternDecorator(name)
}

// GetDecoratorType returns the type of decorator ("value", "action", "block", "pattern") or empty string if not found
func GetDecoratorType(name string) string {
	return globalRegistry.GetDecoratorType(name)
}

// GetValue retrieves a value decorator from the global registry
func GetValue(name string) (ValueDecorator, error) {
	decorator, exists := globalRegistry.GetValue(name)
	if !exists {
		return nil, fmt.Errorf("value decorator %q not found", name)
	}
	return decorator, nil
}

// GetAction retrieves an action decorator from the global registry
func GetAction(name string) (ActionDecorator, error) {
	decorator, exists := globalRegistry.GetAction(name)
	if !exists {
		return nil, fmt.Errorf("action decorator %q not found", name)
	}
	return decorator, nil
}

// GetBlock retrieves a block decorator from the global registry
func GetBlock(name string) (BlockDecorator, error) {
	decorator, exists := globalRegistry.GetBlock(name)
	if !exists {
		return nil, fmt.Errorf("block decorator %q not found", name)
	}
	return decorator, nil
}

// GetPattern retrieves a pattern decorator from the global registry
func GetPattern(name string) (PatternDecorator, error) {
	decorator, exists := globalRegistry.GetPattern(name)
	if !exists {
		return nil, fmt.Errorf("pattern decorator %q not found", name)
	}
	return decorator, nil
}

// ================================================================================================
// ADDITIONAL PARSER SUPPORT TYPES AND FUNCTIONS
// ================================================================================================

// DecoratorType constants for parser
const (
	ValueType   = "value"
	ActionType  = "action"
	BlockType   = "block"
	PatternType = "pattern"
)

// GetAny retrieves any decorator by name and returns it with its type
// Returns DecoratorBase interface since all decorator types implement it
func GetAny(name string) (DecoratorBase, string, error) {
	// Check each type in order
	if decorator, exists := globalRegistry.GetValue(name); exists {
		return decorator, ValueType, nil
	}
	if decorator, exists := globalRegistry.GetAction(name); exists {
		return decorator, ActionType, nil
	}
	if decorator, exists := globalRegistry.GetBlock(name); exists {
		return decorator, BlockType, nil
	}
	if decorator, exists := globalRegistry.GetPattern(name); exists {
		return decorator, PatternType, nil
	}

	return nil, "", fmt.Errorf("decorator %q not found", name)
}
