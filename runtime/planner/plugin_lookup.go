package planner

import (
	"strings"

	"github.com/builtwithtofu/sigil/core/decorator"
	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

func isRegisteredDecoratorPath(path string) bool {
	path = strings.TrimPrefix(path, "@")
	if coreplugin.Global().Lookup(path) != nil {
		return true
	}
	return decorator.Global().IsRegistered(path) || types.Global().IsRegistered(path)
}

func decoratorSchema(path string) (types.DecoratorSchema, bool) {
	for _, candidate := range []string{path, strings.TrimPrefix(path, "@")} {
		if candidate == "" {
			continue
		}
		if capability := coreplugin.Global().Lookup(candidate); capability != nil {
			return coreplugin.DecoratorSchema(capability), true
		}
		if entry, ok := decorator.Global().Lookup(candidate); ok {
			schema := entry.Impl.Descriptor().Schema
			if schema.Path != "" || len(schema.Parameters) > 0 || schema.PrimaryParameter != "" {
				return schema, true
			}
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
