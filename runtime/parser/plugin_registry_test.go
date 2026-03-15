package parser

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/google/go-cmp/cmp"
)

type testPlugin struct{}

func (p *testPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "plug", Version: "1.0.0", APIVersion: 1}
}

func (p *testPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{testValueCapability{}}
}

type testValueCapability struct{}

func (c testValueCapability) Kind() plugin.CapabilityKind { return plugin.KindValue }
func (c testValueCapability) Path() string                { return "plug.value" }
func (c testValueCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Primary: plugin.Param{Name: "name", Type: types.TypeString, Required: true},
		Returns: types.TypeString,
	}
}

func init() {
	_ = plugin.Global().Register(&testPlugin{})
}

func TestPluginRegistryDecoratorDetection(t *testing.T) {
	tree := Parse([]byte(`@plug.value.example`))

	decoratorCount := 0
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
			decoratorCount++
		}
	}

	if diff := cmp.Diff(1, decoratorCount); diff != "" {
		t.Fatalf("decorator count mismatch (-want +got):\n%s", diff)
	}
}

func TestPluginRegistryEnvValidation(t *testing.T) {
	tree := Parse([]byte(`@env.HOME`))
	if len(tree.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", tree.Errors)
	}
}

func TestPluginRegistryExecRetryValidation(t *testing.T) {
	tree := Parse([]byte(`@exec.retry(times=200) { echo "test" }`))
	if len(tree.Errors) == 0 {
		t.Fatal("expected parser validation error for retry max bound")
	}
}
