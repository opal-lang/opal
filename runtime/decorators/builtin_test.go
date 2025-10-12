package decorators

import (
	"testing"

	"github.com/aledsdavies/opal/core/types"
)

func TestBuiltinDecoratorsRegistered(t *testing.T) {
	// Built-in decorators should be registered via init()
	if !types.Global().IsRegistered("var") {
		t.Error("built-in decorator 'var' should be registered")
	}

	if !types.Global().IsRegistered("env") {
		t.Error("built-in decorator 'env' should be registered")
	}
}

func TestUnknownDecoratorNotRegistered(t *testing.T) {
	// Unknown decorators should not be registered
	if types.Global().IsRegistered("unknown") {
		t.Error("'unknown' should not be registered")
	}

	if types.Global().IsRegistered("example") {
		t.Error("'example' should not be registered")
	}
}
