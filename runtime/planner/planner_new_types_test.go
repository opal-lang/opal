package planner_test

import (
	"sync"
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
	"github.com/builtwithtofu/sigil/runtime/parser"
	"github.com/builtwithtofu/sigil/runtime/planner"
)

type typesTestPlugin struct{}

type typesTestCapability struct{}

var registerTypesTestPluginOnce sync.Once

func (p *typesTestPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "planner-types-test", Version: "1.0.0", APIVersion: 1}
}

func (p *typesTestPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{typesTestCapability{}}
}

func (c typesTestCapability) Path() string { return "types.test" }

func (c typesTestCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "intVal", Type: types.TypeInt},
			{Name: "floatVal", Type: types.TypeFloat},
			{Name: "boolVal", Type: types.TypeBool},
			{Name: "durationVal", Type: types.TypeDuration},
			{Name: "intArray", Type: types.TypeArray},
			{Name: "stringArray", Type: types.TypeArray},
			{Name: "objectVal", Type: types.TypeObject},
		},
		Block: plugin.BlockOptional,
	}
}

func (c typesTestCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return next
}

func registerTypesTestPlugin() {
	registerTypesTestPluginOnce.Do(func() {
		_ = plugin.Global().Register(&typesTestPlugin{})
	})
}

func TestPlanNew_DecoratorArgTypes(t *testing.T) {
	registerTypesTestPlugin()
	source := `@types.test(
		boolVal=true,
		durationVal=5s,
		floatVal=1.25,
		intArray=[
			1,
			2,
		],
		intVal=3,
		objectVal={
			name: "api",
			count: 2,
			meta: {
				enabled: true,
			},
		},
		stringArray=[
			"alpha",
			"beta",
		],
	) { echo "ok" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{})
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd, ok := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", plan.Steps[0].Tree)
	}

	assertArg := func(key string, wantKind planfmt.ValueKind) planfmt.Value {
		for _, arg := range cmd.Args {
			if arg.Key == key {
				if arg.Val.Kind != wantKind {
					t.Fatalf("Arg %s kind mismatch: want %v, got %v", key, wantKind, arg.Val.Kind)
				}
				return arg.Val
			}
		}
		t.Fatalf("Missing arg %q", key)
		return planfmt.Value{}
	}

	if assertArg("intVal", planfmt.ValueInt).Int != 3 {
		t.Fatalf("intVal mismatch")
	}
	if assertArg("floatVal", planfmt.ValueFloat).Float != 1.25 {
		t.Fatalf("floatVal mismatch")
	}
	if assertArg("boolVal", planfmt.ValueBool).Bool != true {
		t.Fatalf("boolVal mismatch")
	}
	if assertArg("durationVal", planfmt.ValueDuration).Duration != "5s" {
		t.Fatalf("durationVal mismatch")
	}
	intArray := assertArg("intArray", planfmt.ValueArray)
	if len(intArray.Array) != 2 || intArray.Array[0].Int != 1 || intArray.Array[1].Int != 2 {
		t.Fatalf("intArray mismatch: %+v", intArray.Array)
	}
	stringArray := assertArg("stringArray", planfmt.ValueArray)
	if len(stringArray.Array) != 2 || stringArray.Array[0].Str != "alpha" || stringArray.Array[1].Str != "beta" {
		t.Fatalf("stringArray mismatch: %+v", stringArray.Array)
	}
	objectVal := assertArg("objectVal", planfmt.ValueMap)
	if objectVal.Map["name"].Str != "api" || objectVal.Map["count"].Int != 2 || objectVal.Map["meta"].Map["enabled"].Bool != true {
		t.Fatalf("objectVal mismatch: %+v", objectVal.Map)
	}
}
