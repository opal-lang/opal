package plugin

import (
	"time"

	"github.com/builtwithtofu/sigil/core/types"
)

// BlockMode specifies whether a capability accepts a block.
type BlockMode = types.BlockRequirement

const (
	BlockForbidden BlockMode = types.BlockForbidden
	BlockOptional  BlockMode = types.BlockOptional
	BlockRequired  BlockMode = types.BlockRequired
)

// Param describes a capability parameter.
type Param struct {
	Name     string
	Type     types.ParamType
	Required bool
	Default  any
	Enum     []string
	Examples []string
	Minimum  *float64
	Maximum  *float64
}

// Schema describes the host-visible contract of a capability.
type Schema struct {
	Primary            Param
	Params             []Param
	Returns            types.ParamType
	Block              BlockMode
	Secrets            []string
	Effects            []string
	TransportSensitive bool
	CancelHint         time.Duration
}

// DeclaresSecret reports whether the schema allows resolving a secret.
func (s Schema) DeclaresSecret(name string) bool {
	for _, declared := range s.Secrets {
		if declared == name {
			return true
		}
	}
	return false
}
