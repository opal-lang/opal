package planner

import (
	"strings"

	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

func isRegisteredDecoratorPath(path string) bool {
	path = strings.TrimPrefix(path, "@")
	return coreplugin.Global().Lookup(path) != nil
}

func decoratorSchema(path string) (types.DecoratorSchema, bool) {
	for _, candidate := range []string{path, strings.TrimPrefix(path, "@")} {
		if candidate == "" {
			continue
		}
		if capability := coreplugin.Global().Lookup(candidate); capability != nil {
			return coreplugin.DecoratorSchema(capability), true
		}
	}
	return types.DecoratorSchema{}, false
}

func pluginCapability(path string) coreplugin.Capability {
	for _, candidate := range []string{path, strings.TrimPrefix(path, "@")} {
		if candidate == "" {
			continue
		}
		if capability := coreplugin.Global().Lookup(candidate); capability != nil {
			return capability
		}
	}
	return nil
}
