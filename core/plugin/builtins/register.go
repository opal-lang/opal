package builtins

import (
	"github.com/builtwithtofu/sigil/core/invariant"
	"github.com/builtwithtofu/sigil/core/plugin"
)

// ExecPlugin exposes builtin execution capabilities under the exec namespace.
type ExecPlugin struct{}

func (p *ExecPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "exec", Version: "1.0.0", APIVersion: 1}
}

func (p *ExecPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{RetryWrapperCapability{}}
}

func init() {
	invariant.ExpectNoError(plugin.Global().Register(&EnvPlugin{}), "register builtin env plugin")
	invariant.ExpectNoError(plugin.Global().Register(&ExecPlugin{}), "register builtin exec plugin")
}
