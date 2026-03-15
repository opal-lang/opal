package decorators

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/plugin"
)

func TestEnvDecoratorTransportScope(t *testing.T) {
	capability := plugin.Global().Lookup("env")
	if capability == nil {
		t.Fatal("@env plugin capability should be registered")
	}

	schema := plugin.DecoratorSchema(capability)
	scope := decorator.TransportScopeAny
	if schema.Path == "" {
		t.Fatal("@env plugin schema should adapt to decorator schema")
	}

	if scope != decorator.TransportScopeAny {
		t.Errorf("expected @env to be TransportScopeAny, got %v", scope)
	}
}
