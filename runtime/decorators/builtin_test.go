package decorators

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
)

func TestBuiltinDecoratorsRegistered(t *testing.T) {
	// Built-in decorators should be registered via init()
	if !decorator.Global().IsRegistered("var") {
		t.Error("built-in decorator 'var' should be registered")
	}

	if !decorator.Global().IsRegistered("exec.timeout") {
		t.Error("built-in decorator 'exec.timeout' should be registered")
	}

	if !decorator.Global().IsRegistered("exec.parallel") {
		t.Error("built-in decorator 'exec.parallel' should be registered")
	}

	if !decorator.Global().IsRegistered("fs.workdir") {
		t.Error("built-in decorator 'fs.workdir' should be registered")
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
