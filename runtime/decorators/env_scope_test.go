package decorators

import (
	"testing"

	"github.com/aledsdavies/opal/core/types"
)

func TestEnvDecoratorTransportScope(t *testing.T) {
	// Get the transport scope for @env
	scope := types.Global().GetTransportScope("env")

	if scope != types.ScopeRootOnly {
		t.Errorf("expected @env to be ScopeRootOnly, got %v", scope)
	}
}
