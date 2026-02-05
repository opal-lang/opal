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

func resolveOnly(t *testing.T, source, target string) (*ResolveResult, *vault.Vault, *ExecutionGraph, error) {
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
	return result, vlt, graph, err
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

func TestCommandMode_PreludeIfBlockDoesNotLeakToFunction(t *testing.T) {
	source := `
if true {
    var ENV = "prod"
}
fun deploy {
    echo "@var.ENV"
}
`

	_, _, _, err := planWithPipeline(t, source, "deploy")
	if err == nil {
		t.Fatal("Expected error: ENV declared in prelude if block should not leak into function scope")
	}
}

func TestCommandMode_PreludeWhenBlockDoesNotLeakToFunction(t *testing.T) {
	source := `
var MODE = "prod"
when @var.MODE {
    "prod" -> var ENV = "prod"
    else -> var ENV = "dev"
}
fun deploy {
    echo "@var.ENV"
}
`

	_, _, _, err := planWithPipeline(t, source, "deploy")
	if err == nil {
		t.Fatal("Expected error: ENV declared in prelude when arm should not leak into function scope")
	}
}

func TestCommandMode_PreludeForBlockDoesNotLeakToFunction(t *testing.T) {
	source := `
for item in ["prod"] {
    var ENV = "prod"
}
fun deploy {
    echo "@var.ENV"
}
`

	_, _, _, err := planWithPipeline(t, source, "deploy")
	if err == nil {
		t.Fatal("Expected error: ENV declared in prelude for block should not leak into function scope")
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

func TestLexicalScoping_IfBlockPreservesOuterBinding(t *testing.T) {
	source := `
var COUNT = "1"
if true {
    var COUNT = "2"
}
if @var.COUNT == "1" {
    echo "ok"
} else {
    echo "bad"
}
`

	result, _, _, err := resolveOnly(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(result.Statements) < 3 {
		t.Fatalf("Expected at least 3 statements, got %d", len(result.Statements))
	}

	probe := result.Statements[2].Blocker
	if probe == nil || probe.Kind != BlockerIf {
		t.Fatalf("Expected third statement to be probe if blocker")
	}
	if probe.Taken == nil || !*probe.Taken {
		t.Errorf("Expected probe if to be taken (outer COUNT should remain \"1\")")
	}
}

func TestLexicalScoping_WhenBlockPreservesOuterBinding(t *testing.T) {
	source := `
var MODE = "prod"
var COUNT = "1"
when @var.MODE {
    "prod" -> var COUNT = "2"
    else -> var COUNT = "3"
}
if @var.COUNT == "1" {
    echo "ok"
} else {
    echo "bad"
}
`

	result, _, _, err := resolveOnly(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(result.Statements) < 4 {
		t.Fatalf("Expected at least 4 statements, got %d", len(result.Statements))
	}

	probe := result.Statements[3].Blocker
	if probe == nil || probe.Kind != BlockerIf {
		t.Fatalf("Expected fourth statement to be probe if blocker")
	}
	if probe.Taken == nil || !*probe.Taken {
		t.Errorf("Expected probe if to be taken (outer COUNT should remain \"1\")")
	}
}

func TestLexicalScoping_ForBlockPreservesOuterBinding(t *testing.T) {
	source := `
var COUNT = "1"
for item in ["a", "b"] {
    var COUNT = "2"
}
if @var.COUNT == "1" {
    echo "ok"
} else {
    echo "bad"
}
`

	result, _, _, err := resolveOnly(t, source, "")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(result.Statements) < 3 {
		t.Fatalf("Expected at least 3 statements, got %d", len(result.Statements))
	}

	probe := result.Statements[2].Blocker
	if probe == nil || probe.Kind != BlockerIf {
		t.Fatalf("Expected third statement to be probe if blocker")
	}
	if probe.Taken == nil || !*probe.Taken {
		t.Errorf("Expected probe if to be taken (outer COUNT should remain \"1\")")
	}
}

func TestLexicalScoping_ForLoopVarNotVisibleOutside(t *testing.T) {
	source := `
for item in ["a", "b"] {
    echo "@var.item"
}
echo "@var.item"
`

	_, _, _, err := planWithPipeline(t, source, "")
	if err == nil {
		t.Fatal("Expected error: loop variable item should not be visible outside for block")
	}
}

func TestLexicalScoping_FunctionScopeDoesNotLeakToRoot(t *testing.T) {
	source := `
fun deploy {
    var INNER = "secret"
}
echo "@var.INNER"
`

	_, _, _, err := planWithPipeline(t, source, "")
	if err == nil {
		t.Fatal("Expected error: INNER declared in function should not leak to root scope")
	}
}

func TestLexicalScoping_IfUntakenDeclarationIsUnavailable(t *testing.T) {
	source := `
if false {
    var INNER = "secret"
}
echo "@var.INNER"
`

	_, _, _, err := planWithPipeline(t, source, "")
	if err == nil {
		t.Fatal("Expected error: INNER declared in untaken if branch should not be visible")
	}
}

func TestLexicalScoping_WhenUntakenArmDeclarationIsUnavailable(t *testing.T) {
	source := `
var MODE = "prod"
when @var.MODE {
    "dev" -> var INNER = "secret"
    else -> echo "noop"
}
echo "@var.INNER"
`

	_, _, _, err := planWithPipeline(t, source, "")
	if err == nil {
		t.Fatal("Expected error: INNER declared in untaken when arm should not be visible")
	}
}

func TestLexicalScoping_ForZeroIterationsDoesNotExposeBodyVars(t *testing.T) {
	source := `
for item in [] {
    var INNER = "secret"
}
echo "@var.INNER"
`

	_, _, _, err := planWithPipeline(t, source, "")
	if err == nil {
		t.Fatal("Expected error: INNER declared in zero-iteration for body should not be visible")
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

	if value != "1" {
		t.Errorf("Expected COUNT to restore parent value \"1\", got %v", value)
	}
}
