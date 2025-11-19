package decorators

import (
	"testing"

	"github.com/opal-lang/opal/core/decorator"
)

func TestEnvDecoratorTransportScope(t *testing.T) {
	// Get the transport scope for @env from new registry
	entry, ok := decorator.Global().Lookup("env")
	if !ok {
		t.Fatal("@env should be registered")
	}

	desc := entry.Impl.Descriptor()
	scope := desc.Capabilities.TransportScope

	if scope != decorator.TransportScopeAny {
		t.Errorf("expected @env to be TransportScopeAny, got %v", scope)
	}
}
