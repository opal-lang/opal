package planner_test

import (
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/core/types"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/planner"
)

type typesTestDecorator struct{}

func (d *typesTestDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("types.test").
		Summary("Test decorator for plan type coverage").
		Roles(decorator.RoleWrapper).
		ParamInt("intVal", "Int parameter").Done().
		ParamFloat("floatVal", "Float parameter").Done().
		ParamBool("boolVal", "Bool parameter").Done().
		ParamDuration("durationVal", "Duration parameter").Done().
		ParamArray("intArray", "Int array parameter").
		ElementType(types.TypeInt).
		Done().
		ParamArray("stringArray", "String array parameter").
		ElementType(types.TypeString).
		Done().
		ParamObject("objectVal", "Object parameter").
		Field("name", types.TypeString, "Name").
		Field("count", types.TypeInt, "Count").
		FieldObject("meta", "Meta").
		Field("enabled", types.TypeBool, "Enabled").
		DoneField().
		Done().
		Block(decorator.BlockOptional).
		Build()
}

func (d *typesTestDecorator) Wrap(next decorator.ExecNode, _ map[string]any) decorator.ExecNode {
	return next
}

func init() {
	_ = decorator.Register("types.test", &typesTestDecorator{})
}

func TestPlanNew_DecoratorArgTypes(t *testing.T) {
	// Arguments must be sorted alphabetically for plan validation
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

	intVal := assertArg("intVal", planfmt.ValueInt)
	if intVal.Int != 3 {
		t.Fatalf("intVal mismatch: want 3, got %d", intVal.Int)
	}

	floatVal := assertArg("floatVal", planfmt.ValueFloat)
	if floatVal.Float != 1.25 {
		t.Fatalf("floatVal mismatch: want 1.25, got %v", floatVal.Float)
	}

	boolVal := assertArg("boolVal", planfmt.ValueBool)
	if boolVal.Bool != true {
		t.Fatalf("boolVal mismatch: want true, got %v", boolVal.Bool)
	}

	durationVal := assertArg("durationVal", planfmt.ValueDuration)
	if durationVal.Duration != "5s" {
		t.Fatalf("durationVal mismatch: want 5s, got %q", durationVal.Duration)
	}

	intArray := assertArg("intArray", planfmt.ValueArray)
	if len(intArray.Array) != 2 {
		t.Fatalf("intArray length mismatch: want 2, got %d", len(intArray.Array))
	}
	if intArray.Array[0].Kind != planfmt.ValueInt || intArray.Array[0].Int != 1 {
		t.Fatalf("intArray[0] mismatch: want 1, got %+v", intArray.Array[0])
	}
	if intArray.Array[1].Kind != planfmt.ValueInt || intArray.Array[1].Int != 2 {
		t.Fatalf("intArray[1] mismatch: want 2, got %+v", intArray.Array[1])
	}

	stringArray := assertArg("stringArray", planfmt.ValueArray)
	if len(stringArray.Array) != 2 {
		t.Fatalf("stringArray length mismatch: want 2, got %d", len(stringArray.Array))
	}
	if stringArray.Array[0].Kind != planfmt.ValueString || stringArray.Array[0].Str != "alpha" {
		t.Fatalf("stringArray[0] mismatch: want alpha, got %+v", stringArray.Array[0])
	}
	if stringArray.Array[1].Kind != planfmt.ValueString || stringArray.Array[1].Str != "beta" {
		t.Fatalf("stringArray[1] mismatch: want beta, got %+v", stringArray.Array[1])
	}

	objectVal := assertArg("objectVal", planfmt.ValueMap)
	nameVal, ok := objectVal.Map["name"]
	if !ok || nameVal.Kind != planfmt.ValueString || nameVal.Str != "api" {
		t.Fatalf("objectVal.name mismatch: got %+v", nameVal)
	}
	countVal, ok := objectVal.Map["count"]
	if !ok || countVal.Kind != planfmt.ValueInt || countVal.Int != 2 {
		t.Fatalf("objectVal.count mismatch: got %+v", countVal)
	}
	metaVal, ok := objectVal.Map["meta"]
	if !ok || metaVal.Kind != planfmt.ValueMap {
		t.Fatalf("objectVal.meta kind mismatch: got %+v", metaVal)
	}
	enabledVal, ok := metaVal.Map["enabled"]
	if !ok || enabledVal.Kind != planfmt.ValueBool || enabledVal.Bool != true {
		t.Fatalf("objectVal.meta.enabled mismatch: got %+v", enabledVal)
	}
}
