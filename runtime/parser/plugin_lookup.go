package parser

import (
	"strings"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

func normalizedDecoratorPath(path string) string {
	return strings.TrimPrefix(path, "@")
}

func isRegisteredDecoratorPath(path string) bool {
	path = normalizedDecoratorPath(path)
	return plugin.Global().Lookup(path) != nil
}

func lookupDecoratorSchema(path string) (types.DecoratorSchema, bool) {
	path = normalizedDecoratorPath(path)
	if capability := plugin.Global().Lookup(path); capability != nil {
		return plugin.DecoratorSchema(capability), true
	}
	return types.DecoratorSchema{}, false
}

func isPluginValueDecorator(path string) bool {
	path = normalizedDecoratorPath(path)
	entry := plugin.Global().LookupEntry(path)
	return entry != nil && entry.IsValue()
}

func isPluginTransportDecorator(path string) bool {
	path = normalizedDecoratorPath(path)
	entry := plugin.Global().LookupEntry(path)
	return entry != nil && entry.IsTransport()
}
