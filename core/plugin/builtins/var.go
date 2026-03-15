package builtins

import (
	"fmt"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// VarPlugin exposes the built-in var capability.
type VarPlugin struct{}

func (p *VarPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "var", Version: "1.0.0", APIVersion: 1}
}

func (p *VarPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{VarValueCapability{}}
}

// VarValueCapability resolves values from the current plan scope.
type VarValueCapability struct{}

func (c VarValueCapability) Path() string { return "var" }

func (c VarValueCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Primary: plugin.Param{Name: "name", Type: types.TypeString, Required: true},
		Returns: types.TypeString,
		Block:   plugin.BlockForbidden,
	}
}

func (c VarValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	name := args.GetString("name")
	if name == "" {
		return nil, fmt.Errorf("@var requires a variable name")
	}

	value, ok := ctx.LookupValue(name)
	if !ok {
		return nil, fmt.Errorf("variable %q not found in any scope", name)
	}

	return value, nil
}
