package plugin

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Registry stores plugins and their capabilities by namespace and path.
type Registry struct {
	mu           sync.RWMutex
	plugins      map[string]Plugin
	capabilities map[string]Capability
	entries      map[string]*Entry
}

// Entry preserves plugin identity and discovered capabilities.
type Entry struct {
	Path     string
	Version  string
	Plugin   any
	Schema   Schema
	canValue bool
	canWrap  bool
	canTrans bool
	canRedir bool
}

func (e *Entry) IsValue() bool     { return e != nil && e.canValue }
func (e *Entry) IsWrapper() bool   { return e != nil && e.canWrap }
func (e *Entry) IsTransport() bool { return e != nil && e.canTrans }
func (e *Entry) IsRedirect() bool  { return e != nil && e.canRedir }

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:      make(map[string]Plugin),
		capabilities: make(map[string]Capability),
		entries:      make(map[string]*Entry),
	}
}

var global = NewRegistry()

// Global returns the process-wide plugin registry.
func Global() *Registry {
	return global
}

// Register adds a plugin and all of its capabilities.
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity := plugin.Identity()
	if identity.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if _, exists := r.plugins[identity.Name]; exists {
		return fmt.Errorf("plugin namespace %q already registered", identity.Name)
	}

	for _, capability := range plugin.Capabilities() {
		path := capability.Path()
		if path == "" {
			return fmt.Errorf("plugin %q capability path cannot be empty", identity.Name)
		}
		if _, exists := r.capabilities[path]; exists {
			return fmt.Errorf("capability %q already registered", path)
		}
	}

	r.plugins[identity.Name] = plugin
	for _, capability := range plugin.Capabilities() {
		path := capability.Path()
		r.capabilities[path] = capability
		_, isValue := capability.(ValueProvider)
		_, isWrap := capability.(Wrapper)
		_, isTrans := capability.(Transport)
		_, isRedir := capability.(RedirectTarget)
		r.entries[path] = &Entry{
			Path:     path,
			Version:  identity.Version,
			Plugin:   capability,
			Schema:   capability.Schema(),
			canValue: isValue,
			canWrap:  isWrap,
			canTrans: isTrans,
			canRedir: isRedir,
		}
	}

	return nil
}

// Lookup returns the capability at the given path, if present.
func (r *Registry) Lookup(path string) Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.capabilities[path]
}

// LookupEntry returns the discovered capability entry at the given path.
func (r *Registry) LookupEntry(path string) *Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries[path]
}

// Plugin returns the registered plugin namespace, if present.
func (r *Registry) Plugin(name string) Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins[name]
}

// ListNamespace returns registered capability paths under a namespace.
func (r *Registry) ListNamespace(namespace string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	prefix := namespace + "."
	paths := make([]string, 0)
	for path := range r.capabilities {
		if strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths
}
