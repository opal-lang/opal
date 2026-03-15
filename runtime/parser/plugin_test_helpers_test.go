package parser

import (
	"sync"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

type parserTestPlugin struct{}

type parserConfigValueCapability struct{}

type parserDeployProductionCapability struct{}

type parserDeployStagingCapability struct{}

type parserDeployTestCapability struct{}

type parserRequiredShiftCapability struct{}

var registerParserTestPluginOnce sync.Once

func (p *parserTestPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "parser-test", Version: "1.0.0", APIVersion: 1}
}

func (p *parserTestPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{
		parserConfigValueCapability{},
		parserDeployProductionCapability{},
		parserDeployStagingCapability{},
		parserDeployTestCapability{},
		parserRequiredShiftCapability{},
	}
}

func (c parserConfigValueCapability) Path() string { return "config.myconfig" }
func (c parserConfigValueCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params:  []plugin.Param{{Name: "settings", Type: types.TypeObject}},
		Returns: types.TypeString,
		Block:   plugin.BlockForbidden,
	}
}

func (c parserConfigValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	return "test", nil
}

func (c parserDeployProductionCapability) Path() string { return "deploy.production" }
func (c parserDeployProductionCapability) Schema() plugin.Schema {
	return plugin.Schema{Params: []plugin.Param{{Name: "hosts", Type: types.TypeArray}}, Block: plugin.BlockForbidden}
}

func (c parserDeployProductionCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return nil
}

func (c parserDeployStagingCapability) Path() string { return "deploy.staging" }
func (c parserDeployStagingCapability) Schema() plugin.Schema {
	return plugin.Schema{Params: []plugin.Param{{Name: "hosts", Type: types.TypeArray}}, Block: plugin.BlockForbidden}
}

func (c parserDeployStagingCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return nil
}

func (c parserDeployTestCapability) Path() string { return "deploy.test" }
func (c parserDeployTestCapability) Schema() plugin.Schema {
	return plugin.Schema{Params: []plugin.Param{{Name: "hosts", Type: types.TypeArray}}, Block: plugin.BlockForbidden}
}

func (c parserDeployTestCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return nil
}

func (c parserRequiredShiftCapability) Path() string { return "required_shift" }
func (c parserRequiredShiftCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "a", Type: types.TypeString},
			{Name: "b", Type: types.TypeInt, Required: true},
			{Name: "c", Type: types.TypeString},
		},
		Returns: types.TypeString,
		Block:   plugin.BlockForbidden,
	}
}

func (c parserRequiredShiftCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	return "ok", nil
}

func registerParserTestPlugin() {
	registerParserTestPluginOnce.Do(func() {
		_ = plugin.Global().Register(&parserTestPlugin{})
	})
}

func init() {
	registerParserTestPlugin()
}
