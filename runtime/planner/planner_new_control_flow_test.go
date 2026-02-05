package planner_test

import (
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/planner"

	_ "github.com/opal-lang/opal/runtime/decorators"
)

func TestPlanNew_WhenStatement_Match(t *testing.T) {
	source := `var ENV = "production"
when @var.ENV { "production" -> echo "prod" else -> echo "other" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Target: ""})
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	logic, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Fatalf("Expected LogicNode, got %T", plan.Steps[0].Tree)
	}
	if logic.Kind != "when" {
		t.Errorf("Expected LogicNode kind 'when', got %q", logic.Kind)
	}
	if logic.Result != "matched: production" {
		t.Errorf("Expected result %q, got %q", "matched: production", logic.Result)
	}
	if len(logic.Block) != 1 {
		t.Fatalf("Expected 1 nested step, got %d", len(logic.Block))
	}
	if getDecorator(logic.Block[0].Tree) != "@shell" {
		t.Errorf("Expected @shell decorator, got %q", getDecorator(logic.Block[0].Tree))
	}
	if getCommandArg(logic.Block[0].Tree, "command") != `echo "prod"` {
		t.Errorf("Expected command %q, got %q", `echo "prod"`, getCommandArg(logic.Block[0].Tree, "command"))
	}
}

func TestPlanNew_WhenStatement_NoMatch(t *testing.T) {
	source := `var ENV = "dev"
when @var.ENV { "production" -> echo "prod" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Target: ""})
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	if len(plan.Steps) != 0 {
		t.Fatalf("Expected 0 steps, got %d", len(plan.Steps))
	}
}

func TestPlanNew_TryCatchFinally(t *testing.T) {
	source := `try { echo "try" } catch { echo "catch" } finally { echo "finally" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Target: ""})
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	tryNode, ok := plan.Steps[0].Tree.(*planfmt.TryNode)
	if !ok {
		t.Fatalf("Expected TryNode, got %T", plan.Steps[0].Tree)
	}

	if len(tryNode.TryBlock) != 1 {
		t.Fatalf("Expected 1 try step, got %d", len(tryNode.TryBlock))
	}
	if len(tryNode.CatchBlock) != 1 {
		t.Fatalf("Expected 1 catch step, got %d", len(tryNode.CatchBlock))
	}
	if len(tryNode.FinallyBlock) != 1 {
		t.Fatalf("Expected 1 finally step, got %d", len(tryNode.FinallyBlock))
	}

	if getCommandArg(tryNode.TryBlock[0].Tree, "command") != `echo "try"` {
		t.Errorf("Expected try command %q, got %q", `echo "try"`, getCommandArg(tryNode.TryBlock[0].Tree, "command"))
	}
	if getCommandArg(tryNode.CatchBlock[0].Tree, "command") != `echo "catch"` {
		t.Errorf("Expected catch command %q, got %q", `echo "catch"`, getCommandArg(tryNode.CatchBlock[0].Tree, "command"))
	}
	if getCommandArg(tryNode.FinallyBlock[0].Tree, "command") != `echo "finally"` {
		t.Errorf("Expected finally command %q, got %q", `echo "finally"`, getCommandArg(tryNode.FinallyBlock[0].Tree, "command"))
	}
}

func TestPlanNew_ForLoopIterations(t *testing.T) {
	source := `for region in ["us", "eu"] { echo "ok" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Target: ""})
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(plan.Steps))
	}

	first, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Fatalf("Expected LogicNode for first step, got %T", plan.Steps[0].Tree)
	}
	if first.Kind != "for" {
		t.Errorf("Expected LogicNode kind 'for', got %q", first.Kind)
	}
	if first.Result != "region = us (iteration 1)" {
		t.Errorf("Expected first result %q, got %q", "region = us (iteration 1)", first.Result)
	}
	if len(first.Block) != 1 {
		t.Fatalf("Expected 1 nested step in first iteration, got %d", len(first.Block))
	}
	if getCommandArg(first.Block[0].Tree, "command") != `echo "ok"` {
		t.Errorf("Expected first command %q, got %q", `echo "ok"`, getCommandArg(first.Block[0].Tree, "command"))
	}

	second, ok := plan.Steps[1].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Fatalf("Expected LogicNode for second step, got %T", plan.Steps[1].Tree)
	}
	if second.Kind != "for" {
		t.Errorf("Expected LogicNode kind 'for', got %q", second.Kind)
	}
	if second.Result != "region = eu (iteration 2)" {
		t.Errorf("Expected second result %q, got %q", "region = eu (iteration 2)", second.Result)
	}
	if len(second.Block) != 1 {
		t.Fatalf("Expected 1 nested step in second iteration, got %d", len(second.Block))
	}
	if getCommandArg(second.Block[0].Tree, "command") != `echo "ok"` {
		t.Errorf("Expected second command %q, got %q", `echo "ok"`, getCommandArg(second.Block[0].Tree, "command"))
	}
}
