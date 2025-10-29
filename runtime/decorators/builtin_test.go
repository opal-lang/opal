package decorators

import (
	"testing"

	"github.com/aledsdavies/opal/core/decorator"
)

func TestBuiltinDecoratorsRegistered(t *testing.T) {
	// Built-in decorators should be registered via init()
	if !decorator.Global().IsRegistered("var") {
		t.Error("built-in decorator 'var' should be registered")
	}

	if !decorator.Global().IsRegistered("env") {
		t.Error("built-in decorator 'env' should be registered")
	}
}

func TestUnknownDecoratorNotRegistered(t *testing.T) {
	// Unknown decorators should not be registered
	if decorator.Global().IsRegistered("unknown") {
		t.Error("'unknown' should not be registered")
	}

	if decorator.Global().IsRegistered("example") {
		t.Error("'example' should not be registered")
	}
}
