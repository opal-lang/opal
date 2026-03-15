package builtins

import (
	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// OSPlugin exposes built-in os capabilities.
type OSPlugin struct{}

func (p *OSPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "os", Version: "1.0.0", APIVersion: 1}
}

func (p *OSPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{
		OSGetValueCapability{},
		OSLinuxValueCapability{},
		OSMacOSValueCapability{},
		OSWindowsValueCapability{},
	}
}

type OSGetValueCapability struct{}

func (c OSGetValueCapability) Path() string { return "os.Get" }

func (c OSGetValueCapability) Schema() plugin.Schema {
	return plugin.Schema{Returns: types.TypeString, Block: plugin.BlockForbidden}
}

func (c OSGetValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	return ctx.Session().Snapshot().Platform, nil
}

type OSLinuxValueCapability struct{}

func (c OSLinuxValueCapability) Path() string { return "os.Linux" }

func (c OSLinuxValueCapability) Schema() plugin.Schema {
	return plugin.Schema{Returns: types.TypeString, Block: plugin.BlockForbidden}
}

func (c OSLinuxValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	if ctx.Session().Snapshot().Platform == "linux" {
		return "true", nil
	}
	return "false", nil
}

type OSMacOSValueCapability struct{}

func (c OSMacOSValueCapability) Path() string { return "os.macOS" }

func (c OSMacOSValueCapability) Schema() plugin.Schema {
	return plugin.Schema{Returns: types.TypeString, Block: plugin.BlockForbidden}
}

func (c OSMacOSValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	if ctx.Session().Snapshot().Platform == "darwin" {
		return "true", nil
	}
	return "false", nil
}

type OSWindowsValueCapability struct{}

func (c OSWindowsValueCapability) Path() string { return "os.Windows" }

func (c OSWindowsValueCapability) Schema() plugin.Schema {
	return plugin.Schema{Returns: types.TypeString, Block: plugin.BlockForbidden}
}

func (c OSWindowsValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	if ctx.Session().Snapshot().Platform == "windows" {
		return "true", nil
	}
	return "false", nil
}
