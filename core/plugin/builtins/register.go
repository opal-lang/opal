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
	return []plugin.Capability{RetryWrapperCapability{}, TimeoutWrapperCapability{}, ParallelWrapperCapability{}}
}

func init() {
	invariant.ExpectNoError(plugin.Global().Register(&EnvPlugin{}), "register builtin env plugin")
	invariant.ExpectNoError(plugin.Global().Register(&ExecPlugin{}), "register builtin exec plugin")
	invariant.ExpectNoError(plugin.Global().Register(&VarPlugin{}), "register builtin var plugin")
	invariant.ExpectNoError(plugin.Global().Register(&OSPlugin{}), "register builtin os plugin")
	invariant.ExpectNoError(plugin.Global().Register(&CryptoPlugin{}), "register builtin crypto plugin")
	invariant.ExpectNoError(plugin.Global().Register(&FSPlugin{}), "register builtin fs plugin")
	invariant.ExpectNoError(plugin.Global().Register(&FilePlugin{}), "register builtin file plugin")
	invariant.ExpectNoError(plugin.Global().Register(&ShellPlugin{}), "register builtin shell plugin")
	invariant.ExpectNoError(plugin.Global().Register(&SecretsPlugin{}), "register builtin secrets plugin")
	invariant.ExpectNoError(plugin.Global().Register(&TestTransportPlugin{}), "register builtin test transport plugin")
	invariant.ExpectNoError(plugin.Global().Register(&SandboxPlugin{}), "register builtin sandbox plugin")
	invariant.ExpectNoError(plugin.Global().Register(&IsolatedPlugin{}), "register builtin isolated plugin")
}
