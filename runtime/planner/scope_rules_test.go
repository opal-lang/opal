package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"

	_ "github.com/opal-lang/opal/runtime/decorators" // Register decorators for parser + resolver
)

func planWithPipeline(t *testing.T, source, target string) (*planfmt.Plan, *vault.Vault, *ExecutionGraph, error) {
	t.Helper()

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	graph, err := BuildIR(tree.Events, tree.Tokens)
	if err != nil {
		return nil, nil, nil, err
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	result, err := Resolve(graph, vlt, decorator.NewLocalSession(), ResolveConfig{TargetFunction: target})
	if err != nil {
		return nil, vlt, graph, err
	}

	scopes := graph.Scopes
	if target != "" {
		if fn, ok := graph.Functions[target]; ok && fn.Scopes != nil {
			scopes = fn.Scopes
		}
	}

	emitter := NewEmitter(result, vlt, scopes, target)
	plan, err := emitter.Emit()
	return plan, vlt, graph, err
}

func TestCommandMode_InheritsTopLevelPrelude(t *testing.T) {
	source := `
var ENV = "prod"
fun deploy {
    echo "@var.ENV"
}
var LATER = "ignored"
`

	plan, _, _, err := planWithPipeline(t, source, "deploy")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	command := getCommandArg(plan.Steps[0].Tree, "command")
	if !containsDisplayID(command) {
		t.Errorf("Expected DisplayID placeholder in command, got %q", command)
	}
}

func TestCommandMode_DoesNotSeeLaterVars(t *testing.T) {
	source := `
fun deploy {
    echo "@var.ENV"
}
var ENV = "prod"
`

	_, _, _, err := planWithPipeline(t, source, "deploy")
	if err == nil {
		t.Fatal("Expected error for @var.ENV declared after function")
	}

	if !strings.Contains(err.Error(), "ENV") {
		t.Errorf("Error should mention ENV, got %v", err)
	}
}

func TestMetaprogrammingLeak_IfBlockMutatesOuterScope(t *testing.T) {
	source := `
var COUNT = "1"
if true {
    var COUNT = "2"
}
echo "@var.COUNT"
`

	_, vlt, graph, err := planWithPipeline(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	exprID, ok := graph.Scopes.Lookup("COUNT")
	if !ok {
		t.Fatal("Expected COUNT to be defined in scope")
	}

	value, ok := vlt.GetUnresolvedValue(exprID)
	if !ok {
		t.Fatal("Expected COUNT value to be resolved")
	}

	if value != "\"2\"" {
		t.Errorf("Expected COUNT to leak from if block as \"2\", got %v", value)
	}
}

func TestExecutionBlockIsolation_RestoresParentValue(t *testing.T) {
	source := `
var COUNT = "1"
@retry {
    var COUNT = "2"
}
echo "@var.COUNT"
`

	_, vlt, graph, err := planWithPipeline(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	exprID, ok := graph.Scopes.Lookup("COUNT")
	if !ok {
		t.Fatal("Expected COUNT to be defined in scope")
	}

	value, ok := vlt.GetUnresolvedValue(exprID)
	if !ok {
		t.Fatal("Expected COUNT value to be resolved")
	}

	if value != "\"1\"" {
		t.Errorf("Expected COUNT to restore parent value \"1\", got %v", value)
	}
}
