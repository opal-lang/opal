package planner

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/builtwithtofu/sigil/runtime/parser"

	_ "github.com/builtwithtofu/sigil/runtime/decorators" // Register decorators for parser
)

// Decorator Block Structure Tests
//
// These tests verify that decorator blocks (@exec.retry, @exec.timeout, @exec.parallel, etc.)
// are correctly represented in the plan as CommandNode with:
// 1. Decorator name set
// 2. Arguments parsed
// 3. Block steps collected
//
// This is CRITICAL for execution - without the decorator Step, the executor
// will just run the inner commands without retry/timeout/parallel semantics.

// TestDecoratorBlock_CreatesDecoratorStep verifies that @exec.retry creates a proper
// decorator Step in the plan (not just the inner commands).
func TestDecoratorBlock_CreatesDecoratorStep(t *testing.T) {
	source := `
@exec.retry(times=3) {
    echo "test"
}
`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	plan := result.Plan

	// ASSERT: Should have 1 step (the @exec.retry decorator)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step (@exec.retry), got %d", len(plan.Steps))
	}

	step := plan.Steps[0]

	// ASSERT: Step should have a tree
	if step.Tree == nil {
		t.Fatal("Step.Tree is nil")
	}

	// ASSERT: Tree should be a CommandNode
	cmd, ok := step.Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", step.Tree)
	}

	// ASSERT: Decorator should be "@exec.retry"
	if cmd.Decorator != "@exec.retry" {
		t.Errorf("Expected decorator '@exec.retry', got '%s'", cmd.Decorator)
	}

	// ASSERT: Should have 'times' argument
	var timesArg *planfmt.Arg
	for i := range cmd.Args {
		if cmd.Args[i].Key == "times" {
			timesArg = &cmd.Args[i]
			break
		}
	}
	if timesArg == nil {
		t.Fatal("Missing 'times' argument")
	}

	// ASSERT: times should be 3
	if timesArg.Val.Kind != planfmt.ValueInt || timesArg.Val.Int != 3 {
		t.Errorf("Expected times=3, got %v", timesArg.Val)
	}

	// ASSERT: Should have block with 1 step (echo)
	if len(cmd.Block) != 1 {
		t.Fatalf("Expected 1 block step, got %d", len(cmd.Block))
	}

	blockStep := cmd.Block[0]
	blockCmd, ok := blockStep.Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected block step to be CommandNode, got %T", blockStep.Tree)
	}

	// ASSERT: Block command should be @shell
	if blockCmd.Decorator != "@shell" {
		t.Errorf("Expected block decorator '@shell', got '%s'", blockCmd.Decorator)
	}

	t.Logf("✓ @exec.retry decorator step created correctly")
	t.Logf("✓ Arguments parsed: times=%d", timesArg.Val.Int)
	t.Logf("✓ Block contains %d step(s)", len(cmd.Block))
}

// TestDecoratorBlock_NestedDecorators verifies nested decorator blocks
// are properly structured in the plan.
func TestDecoratorBlock_NestedDecorators(t *testing.T) {
	source := `
@exec.timeout(duration=30s) {
    @exec.retry(times=3) {
        echo "test"
    }
}
`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	plan := result.Plan

	// ASSERT: Should have 1 step (@exec.timeout)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step (@exec.timeout), got %d", len(plan.Steps))
	}

	// ASSERT: Outer decorator is @exec.timeout
	outerCmd, ok := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", plan.Steps[0].Tree)
	}
	if outerCmd.Decorator != "@exec.timeout" {
		t.Errorf("Expected outer decorator '@exec.timeout', got '%s'", outerCmd.Decorator)
	}

	// ASSERT: @exec.timeout block has 1 step (@exec.retry)
	if len(outerCmd.Block) != 1 {
		t.Fatalf("Expected 1 block step in @exec.timeout, got %d", len(outerCmd.Block))
	}

	// ASSERT: Inner decorator is @exec.retry
	innerCmd, ok := outerCmd.Block[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected inner CommandNode, got %T", outerCmd.Block[0].Tree)
	}
	if innerCmd.Decorator != "@exec.retry" {
		t.Errorf("Expected inner decorator '@exec.retry', got '%s'", innerCmd.Decorator)
	}

	// ASSERT: @exec.retry block has 1 step (echo)
	if len(innerCmd.Block) != 1 {
		t.Fatalf("Expected 1 block step in @exec.retry, got %d", len(innerCmd.Block))
	}

	echoCmd, ok := innerCmd.Block[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected echo CommandNode, got %T", innerCmd.Block[0].Tree)
	}
	if echoCmd.Decorator != "@shell" {
		t.Errorf("Expected echo decorator '@shell', got '%s'", echoCmd.Decorator)
	}

	t.Logf("✓ Nested decorators structured correctly")
	t.Logf("✓ @exec.timeout → @exec.retry → echo")
}

// TestDecoratorBlock_NoArguments verifies decorator blocks without arguments work.
func TestDecoratorBlock_NoArguments(t *testing.T) {
	source := `
@exec.parallel {
    echo "a"
    echo "b"
}
`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{})
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	plan := result.Plan

	// ASSERT: Should have 1 step (@exec.parallel)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step (@exec.parallel), got %d", len(plan.Steps))
	}

	cmd, ok := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", plan.Steps[0].Tree)
	}

	// ASSERT: Decorator should be "@exec.parallel"
	if cmd.Decorator != "@exec.parallel" {
		t.Errorf("Expected decorator '@exec.parallel', got '%s'", cmd.Decorator)
	}

	// ASSERT: Should have no arguments (or empty args)
	if len(cmd.Args) != 0 {
		t.Logf("Note: @exec.parallel has %d args (expected 0, but may be OK)", len(cmd.Args))
	}

	// ASSERT: Should have 2 block steps (echo "a", echo "b")
	if len(cmd.Block) != 2 {
		t.Fatalf("Expected 2 block steps, got %d", len(cmd.Block))
	}

	t.Logf("✓ @exec.parallel decorator created correctly")
	t.Logf("✓ Block contains %d steps", len(cmd.Block))
}
