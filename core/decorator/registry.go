package decorator

import (
	"fmt"
	"sync"
)

// Registry holds registered decorators with auto-inferred roles.
// Uses the database/sql driver registration pattern.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]Entry // path -> Entry
}

// Entry represents a registered decorator.
type Entry struct {
	Impl  Decorator // The decorator implementation
	Roles []Role    // Auto-inferred from implemented interfaces
}

// NewRegistry creates a new decorator registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]Entry),
	}
}

// Register adds a decorator to the registry.
// Roles are automatically inferred from implemented interfaces.
//
// Example:
//
//	func init() {
//	    decorator.Register("var", &VarDecorator{})
//	    decorator.Register("retry", &RetryDecorator{})
//	    decorator.Register("aws.s3.object", &AWSS3ObjectDecorator{})
//	}
func Register(path string, impl Decorator) error {
	return global.register(path, impl)
}

func (r *Registry) register(path string, impl Decorator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Auto-infer roles from implemented interfaces
	roles := inferRoles(impl)

	r.entries[path] = Entry{
		Impl:  impl,
		Roles: roles,
	}

	return nil
}

// Lookup retrieves a decorator by path (URI-based lookup).
func (r *Registry) Lookup(path string) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[path]
	return entry, ok
}

// IsRegistered checks if a decorator path is registered.
func (r *Registry) IsRegistered(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.entries[path]
	return exists
}

// Export returns all registered decorators (for tooling/docs).
func (r *Registry) Export() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	descriptors := make([]Descriptor, 0, len(r.entries))
	for _, entry := range r.entries {
		desc := entry.Impl.Descriptor()
		desc.Roles = entry.Roles // Use auto-inferred roles
		descriptors = append(descriptors, desc)
	}

	return descriptors
}

// inferRoles automatically determines decorator roles from implemented interfaces.
// This is the key insight: decorators just implement interfaces, registry figures out what they can do.
func inferRoles(decorator Decorator) []Role {
	var roles []Role

	// Check each interface
	if _, ok := decorator.(Value); ok {
		roles = append(roles, RoleProvider)
	}
	if _, ok := decorator.(Exec); ok {
		roles = append(roles, RoleWrapper)
	}
	if _, ok := decorator.(Transport); ok {
		roles = append(roles, RoleBoundary)
	}
	if _, ok := decorator.(Endpoint); ok {
		roles = append(roles, RoleEndpoint)
	}

	// If no roles inferred, something is wrong
	if len(roles) == 0 {
		// Decorator must implement at least one role interface
		panic(fmt.Sprintf("decorator %q implements no role interfaces", decorator.Descriptor().Path))
	}

	return roles
}

// Global registry instance (database/sql pattern)
var global = NewRegistry()

// ResolveValue resolves a value decorator with scope enforcement and parameter validation.
// This is the high-level method that should be used by the planner.
func (r *Registry) ResolveValue(
	ctx ValueEvalContext,
	call ValueCall,
	currentScope TransportScope,
) (ResolvedValue, error) {
	// Step 1: Lookup decorator
	entry, ok := r.Lookup(call.Path)
	if !ok {
		return ResolvedValue{}, fmt.Errorf("decorator %q not found", call.Path)
	}

	// Step 2: Type assert to Value interface
	valueDecorator, ok := entry.Impl.(Value)
	if !ok {
		return ResolvedValue{}, fmt.Errorf("decorator %q does not implement Value interface", call.Path)
	}

	// Step 3: Check transport scope compatibility
	desc := entry.Impl.Descriptor()
	if !desc.Capabilities.TransportScope.Allows(currentScope) {
		return ResolvedValue{}, fmt.Errorf(
			"decorator %q cannot be used in current transport scope (requires %s, current: %s)",
			call.Path,
			desc.Capabilities.TransportScope,
			currentScope,
		)
	}

	// Step 4: TODO - Validate parameters (enum, range, pattern from schema)
	// This will be implemented when we have proper schema validation

	// Step 5: Call decorator's Resolve method
	value, err := valueDecorator.Resolve(ctx, call)
	if err != nil {
		return ResolvedValue{}, err
	}

	// Step 6: TODO - Wrap secrets in ResolvedValue
	// This will be implemented in Phase 4 when we add secret handling
	resolved := ResolvedValue{
		Value:     value,
		Handle:    nil, // TODO: Secret wrapping
		DisplayID: "",  // TODO: Secret ID generation
	}

	return resolved, nil
}

// Global returns the global decorator registry.
func Global() *Registry {
	return global
}
