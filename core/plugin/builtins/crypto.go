package builtins

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// CryptoPlugin exposes cryptographic helper capabilities.
type CryptoPlugin struct{}

func (p *CryptoPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "crypto", Version: "1.0.0", APIVersion: 1}
}

func (p *CryptoPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{CryptoValueCapability{}}
}

type CryptoValueCapability struct{}

func (c CryptoValueCapability) Path() string { return "crypto" }

func (c CryptoValueCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Primary: plugin.Param{Name: "method", Type: types.TypeString},
		Params: []plugin.Param{
			{Name: "arg0", Type: types.TypeString},
			{Name: "arg1", Type: types.TypeString},
			{Name: "data", Type: types.TypeString},
		},
		Returns: types.TypeString,
		Block:   plugin.BlockForbidden,
	}
}

func (c CryptoValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	method := args.GetString("method")
	if method == "" {
		method = "SHA256"
	}

	if !strings.EqualFold(method, "SHA256") {
		return nil, fmt.Errorf("unsupported crypto method: %s", method)
	}

	input := args.GetString("arg0")
	if input == "" {
		input = args.GetString("arg1")
	}
	if input == "" {
		input = args.GetString("data")
	}
	if input == "" {
		return nil, fmt.Errorf("@crypto.%s requires one string argument", method)
	}

	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:]), nil
}
