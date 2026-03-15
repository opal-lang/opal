package builtins

import (
	"fmt"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// EnvPlugin exposes the built-in env capability.
type EnvPlugin struct{}

func (p *EnvPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{
		Name:       "env",
		Version:    "1.0.0",
		APIVersion: 1,
	}
}

func (p *EnvPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{EnvValueCapability{}}
}

// EnvValueCapability resolves environment variables from the current session.
type EnvValueCapability struct{}

func (c EnvValueCapability) Kind() plugin.CapabilityKind { return plugin.KindValue }

func (c EnvValueCapability) Path() string { return "env" }

func (c EnvValueCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Primary: plugin.Param{Name: "property", Type: types.TypeString, Required: true, Examples: []string{"HOME", "PATH", "USER"}},
		Params: []plugin.Param{
			{Name: "default", Type: types.TypeString},
		},
		Returns:            types.TypeString,
		Block:              plugin.BlockForbidden,
		TransportSensitive: true,
	}
}

func (c EnvValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (string, error) {
	name := args.GetString("property")
	value, exists := ctx.Session().Snapshot().Env[name]
	if exists {
		return value, nil
	}

	defaultValue := args.GetStringOptional("default")
	if defaultValue != "" {
		return defaultValue, nil
	}

	return "", fmt.Errorf("environment variable %q not found", name)
}
