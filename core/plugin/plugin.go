package plugin

// Plugin is a namespace that exposes one or more capabilities.
//
// Plugins are opaque to users. The host discovers them through this metadata,
// validates their schemas, and keeps execution of nested blocks host-driven.
type Plugin interface {
	Identity() PluginIdentity
	Capabilities() []Capability
}

// PluginIdentity identifies a plugin package and its host-compatibility window.
type PluginIdentity struct {
	Name       string
	Version    string
	APIVersion int
	SigilRange string
}
