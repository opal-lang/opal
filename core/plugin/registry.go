package plugin

import (
	"fmt"
	"slices"
	"strings"
)

// Registry stores plugins and their capabilities by namespace and path.
type Registry struct {
	plugins      map[string]Plugin
	capabilities map[string]Capability
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:      make(map[string]Plugin),
		capabilities: make(map[string]Capability),
	}
}

var global = NewRegistry()

// Global returns the process-wide plugin registry.
func Global() *Registry {
	return global
}

// Register adds a plugin and all of its capabilities.
func (r *Registry) Register(plugin Plugin) error {
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
		r.capabilities[capability.Path()] = capability
	}

	return nil
}

// Lookup returns the capability at the given path, if present.
func (r *Registry) Lookup(path string) Capability {
	return r.capabilities[path]
}

// Plugin returns the registered plugin namespace, if present.
func (r *Registry) Plugin(name string) Plugin {
	return r.plugins[name]
}

// ListNamespace returns registered capability paths under a namespace.
func (r *Registry) ListNamespace(namespace string) []string {
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
