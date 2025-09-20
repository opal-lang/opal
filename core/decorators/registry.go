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

	// Legacy interfaces (will be removed in Phase 4)
	LegacyActions  map[string]LegacyActionDecorator
	LegacyBlocks   map[string]LegacyBlockDecorator
	LegacyValues   map[string]LegacyValueDecorator
	LegacyPatterns map[string]LegacyPatternDecorator

	// Target interfaces (generic with any for registry storage)
	Values    map[string]ValueDecorator[any]
	Execution map[string]ExecutionDecorator[any]
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		// Legacy interfaces
		LegacyActions:  make(map[string]LegacyActionDecorator),
		LegacyBlocks:   make(map[string]LegacyBlockDecorator),
		LegacyValues:   make(map[string]LegacyValueDecorator),
		LegacyPatterns: make(map[string]LegacyPatternDecorator),

		// Target interfaces
		Values:    make(map[string]ValueDecorator[any]),
		Execution: make(map[string]ExecutionDecorator[any]),
	}
}

// RegisterValue registers a legacy value decorator with collision detection across all categories
func (r *Registry) RegisterValue(decorator LegacyValueDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.LegacyValues[name] = decorator
	return nil
}

// RegisterAction registers a legacy action decorator with collision detection across all categories
func (r *Registry) RegisterAction(decorator LegacyActionDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.LegacyActions[name] = decorator
	return nil
}

// RegisterBlock registers a legacy block decorator with collision detection across all categories
func (r *Registry) RegisterBlock(decorator LegacyBlockDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.LegacyBlocks[name] = decorator
	return nil
}

// RegisterPattern registers a legacy pattern decorator with collision detection across all categories
func (r *Registry) RegisterPattern(decorator LegacyPatternDecorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.LegacyPatterns[name] = decorator
	return nil
}

// checkCollision checks if a name is already registered in any category
// Must be called with lock held
// During Phase 1-3: Allow same decorator to be registered in both legacy and new interfaces
func (r *Registry) checkCollision(name string) error {
	// During migration: Only check for conflicts between categories of the same generation

	// Check within new interfaces
	valueExists := false
	if _, exists := r.Values[name]; exists {
		valueExists = true
	}
	executionExists := false
	if _, exists := r.Execution[name]; exists {
		executionExists = true
	}

	// Error if same name in both new interfaces
	if valueExists && executionExists {
		return fmt.Errorf("decorator %q already registered as both ValueDecorator and ExecutionDecorator", name)
	}

	// Check within legacy interfaces
	legacyValueExists := false
	if _, exists := r.LegacyValues[name]; exists {
		legacyValueExists = true
	}
	legacyActionExists := false
	if _, exists := r.LegacyActions[name]; exists {
		legacyActionExists = true
	}
	legacyBlockExists := false
	if _, exists := r.LegacyBlocks[name]; exists {
		legacyBlockExists = true
	}
	legacyPatternExists := false
	if _, exists := r.LegacyPatterns[name]; exists {
		legacyPatternExists = true
	}

	// Error if same name in multiple legacy interfaces
	legacyCount := 0
	if legacyValueExists {
		legacyCount++
	}
	if legacyActionExists {
		legacyCount++
	}
	if legacyBlockExists {
		legacyCount++
	}
	if legacyPatternExists {
		legacyCount++
	}

	if legacyCount > 1 {
		return fmt.Errorf("decorator %q already registered in multiple legacy interfaces", name)
	}

	// Allow same name between legacy and new (migration phase)
	return nil
}

// GetValue retrieves a legacy value decorator
func (r *Registry) GetValue(name string) (LegacyValueDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.LegacyValues[name]
	return decorator, exists
}

// GetAction retrieves a legacy action decorator
func (r *Registry) GetAction(name string) (LegacyActionDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.LegacyActions[name]
	return decorator, exists
}

// GetBlock retrieves a legacy block decorator
func (r *Registry) GetBlock(name string) (LegacyBlockDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.LegacyBlocks[name]
	return decorator, exists
}

// GetPattern retrieves a legacy pattern decorator
func (r *Registry) GetPattern(name string) (LegacyPatternDecorator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.LegacyPatterns[name]
	return decorator, exists
}

// ================================================================================================
// UNIFIED INTERFACE REGISTRATION - Target architecture
// ================================================================================================

// RegisterValueDecorator registers a value decorator with the new interface
func (r *Registry) RegisterValueDecorator(decorator ValueDecorator[any]) error {
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

// RegisterExecutionDecorator registers an execution decorator with the new interface
func (r *Registry) RegisterExecutionDecorator(decorator ExecutionDecorator[any]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := decorator.Name()

	// Check for collisions across ALL categories
	if err := r.checkCollision(name); err != nil {
		return err
	}

	r.Execution[name] = decorator
	return nil
}

// GetValueDecorator retrieves a value decorator
func (r *Registry) GetValueDecorator(name string) (ValueDecorator[any], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.Values[name]
	return decorator, exists
}

// GetExecutionDecorator retrieves an execution decorator
func (r *Registry) GetExecutionDecorator(name string) (ExecutionDecorator[any], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.Execution[name]
	return decorator, exists
}

// ListAll returns all registered decorators organized by category
func (r *Registry) ListAll() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)

	// Collect unified values
	var values []string
	for name := range r.Values {
		values = append(values, name)
	}
	result["values"] = values

	// Collect unified execution
	var execution []string
	for name := range r.Execution {
		execution = append(execution, name)
	}
	result["execution"] = execution

	// Collect legacy values
	var legacyValues []string
	for name := range r.LegacyValues {
		legacyValues = append(legacyValues, name)
	}
	result["legacy_values"] = legacyValues

	// Collect legacy actions
	var actions []string
	for name := range r.LegacyActions {
		actions = append(actions, name)
	}
	result["actions"] = actions

	// Collect legacy blocks
	var blocks []string
	for name := range r.LegacyBlocks {
		blocks = append(blocks, name)
	}
	result["blocks"] = blocks

	// Collect legacy patterns
	var patterns []string
	for name := range r.LegacyPatterns {
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
func Register(decorator LegacyValueDecorator) {
	if err := globalRegistry.RegisterValue(decorator); err != nil {
		panic(fmt.Sprintf("failed to register value decorator: %v", err))
	}
}

// RegisterAction registers an action decorator in the global registry (called from init functions)
func RegisterAction(decorator LegacyActionDecorator) {
	if err := globalRegistry.RegisterAction(decorator); err != nil {
		panic(fmt.Sprintf("failed to register action decorator: %v", err))
	}
}

// RegisterBlock registers a block decorator in the global registry (called from init functions)
func RegisterBlock(decorator LegacyBlockDecorator) {
	if err := globalRegistry.RegisterBlock(decorator); err != nil {
		panic(fmt.Sprintf("failed to register block decorator: %v", err))
	}
}

// RegisterPattern registers a pattern decorator in the global registry (called from init functions)
func RegisterPattern(decorator LegacyPatternDecorator) {
	if err := globalRegistry.RegisterPattern(decorator); err != nil {
		panic(fmt.Sprintf("failed to register pattern decorator: %v", err))
	}
}

// RegisterValueDecorator registers a value decorator globally
func RegisterValueDecorator(decorator ValueDecorator[any]) {
	if err := globalRegistry.RegisterValueDecorator(decorator); err != nil {
		panic(fmt.Sprintf("Failed to register value decorator %q: %v", decorator.Name(), err))
	}
}

// RegisterExecutionDecorator registers an execution decorator globally
func RegisterExecutionDecorator(decorator ExecutionDecorator[any]) {
	if err := globalRegistry.RegisterExecutionDecorator(decorator); err != nil {
		panic(fmt.Sprintf("Failed to register execution decorator %q: %v", decorator.Name(), err))
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

	// Copy unified decorators
	for name, decorator := range globalRegistry.Values {
		r.Values[name] = decorator
	}
	for name, decorator := range globalRegistry.Execution {
		r.Execution[name] = decorator
	}

	// Copy legacy decorators
	for name, decorator := range globalRegistry.LegacyValues {
		r.LegacyValues[name] = decorator
	}
	for name, decorator := range globalRegistry.LegacyActions {
		r.LegacyActions[name] = decorator
	}
	for name, decorator := range globalRegistry.LegacyBlocks {
		r.LegacyBlocks[name] = decorator
	}
	for name, decorator := range globalRegistry.LegacyPatterns {
		r.LegacyPatterns[name] = decorator
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

// GetValue retrieves a legacy value decorator from the global registry
func GetValue(name string) (LegacyValueDecorator, error) {
	decorator, exists := globalRegistry.GetValue(name)
	if !exists {
		return nil, fmt.Errorf("value decorator %q not found", name)
	}
	return decorator, nil
}

// GetAction retrieves a legacy action decorator from the global registry
func GetAction(name string) (LegacyActionDecorator, error) {
	decorator, exists := globalRegistry.GetAction(name)
	if !exists {
		return nil, fmt.Errorf("action decorator %q not found", name)
	}
	return decorator, nil
}

// GetBlock retrieves a legacy block decorator from the global registry
func GetBlock(name string) (LegacyBlockDecorator, error) {
	decorator, exists := globalRegistry.GetBlock(name)
	if !exists {
		return nil, fmt.Errorf("block decorator %q not found", name)
	}
	return decorator, nil
}

// GetPattern retrieves a legacy pattern decorator from the global registry
func GetPattern(name string) (LegacyPatternDecorator, error) {
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
	ValueType      = "value"
	ActionType     = "action"
	BlockDecorType = "block" // Renamed to avoid conflict with new BlockType
	PatternType    = "pattern"
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
		return decorator, BlockDecorType, nil
	}
	if decorator, exists := globalRegistry.GetPattern(name); exists {
		return decorator, PatternType, nil
	}

	return nil, "", fmt.Errorf("decorator %q not found", name)
}
