package parser

import (
	"strings"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

func normalizedDecoratorPath(path string) string {
	return strings.TrimPrefix(path, "@")
}

func isRegisteredDecoratorPath(path string) bool {
	path = normalizedDecoratorPath(path)
	if plugin.Global().Lookup(path) != nil {
		return true
	}
	return decorator.Global().IsRegistered(path) || types.Global().IsRegistered(path)
}

func lookupDecoratorSchema(path string) (types.DecoratorSchema, bool) {
	path = normalizedDecoratorPath(path)
	if capability := plugin.Global().Lookup(path); capability != nil {
		return plugin.DecoratorSchema(capability), true
	}
	if entry, ok := decorator.Global().Lookup(path); ok {
		return entry.Impl.Descriptor().Schema, true
	}
	return types.Global().GetSchema(path)
}

func isPluginValueDecorator(path string) bool {
	path = normalizedDecoratorPath(path)
	capability := plugin.Global().Lookup(path)
	return capability != nil && capability.Kind() == plugin.KindValue
}

func isPluginTransportDecorator(path string) bool {
	path = normalizedDecoratorPath(path)
	capability := plugin.Global().Lookup(path)
	return capability != nil && capability.Kind() == plugin.KindTransport
}
