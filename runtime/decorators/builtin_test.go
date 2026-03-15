package decorators

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/plugin"
)

func TestBuiltinDecoratorsRegistered(t *testing.T) {
	// Built-in decorators should be registered via init()
	if !decorator.Global().IsRegistered("var") {
		t.Error("built-in decorator 'var' should be registered")
	}

	if plugin.Global().Lookup("exec.timeout") == nil {
		t.Error("built-in plugin capability 'exec.timeout' should be registered")
	}

	if plugin.Global().Lookup("exec.parallel") == nil {
		t.Error("built-in plugin capability 'exec.parallel' should be registered")
	}

	if plugin.Global().Lookup("fs.workdir") == nil {
		t.Error("built-in plugin capability 'fs.workdir' should be registered")
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
