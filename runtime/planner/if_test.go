package planner_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/planner"

	// Import decorators to register @var in the global registry
	_ "github.com/opal-lang/opal/runtime/decorators"
)

// getCommandFromStep extracts a CommandNode from a step, handling LogicNode wrappers.
// The new planner preserves structure with LogicNode for if/for/when statements.
// For elseif chains, it recursively searches through nested LogicNodes.
func getCommandFromStep(step *planfmt.Step) *planfmt.CommandNode {
	return getCommandFromNode(step.Tree)
}

// getCommandFromNode recursively searches for a CommandNode in a tree node.
func getCommandFromNode(node planfmt.ExecutionNode) *planfmt.CommandNode {
	// First try direct CommandNode
	if cmd, ok := node.(*planfmt.CommandNode); ok {
		return cmd
	}

	// If it's a LogicNode, recursively search its block
	if logic, ok := node.(*planfmt.LogicNode); ok {
		for _, blockStep := range logic.Block {
			if cmd := getCommandFromNode(blockStep.Tree); cmd != nil {
				return cmd
			}
		}
	}

	return nil
}

// =============================================================================
// Category 1: Basic If (boolean literals)
// =============================================================================

// TestIfTrue verifies that `if true { ... }` plans the block
func TestIfTrue(t *testing.T) {
	source := `if true { echo "yes" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (the echo command from the if block)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Verify it's the echo command (may be wrapped in LogicNode)
	step := plan.Steps[0]
	cmd := getCommandFromStep(&step)
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", step.Tree)
	}

	if cmd.Decorator != "@shell" {
		t.Errorf("Expected @shell decorator, got %q", cmd.Decorator)
	}

	// Find command argument
	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "yes"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfFalse verifies that `if false { ... }` prunes the block (empty plan)
func TestIfFalse(t *testing.T) {
	source := `if false { echo "no" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps (block pruned)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps (pruned), got %d", len(plan.Steps))
	}
}

// TestIfTrueEmptyBlock verifies that `if true { }` produces empty plan
func TestIfTrueEmptyBlock(t *testing.T) {
	source := `if true { }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Empty blocks now return a LogicNode with no nested commands
	// This preserves structure for display purposes
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step (empty LogicNode), got %d", len(plan.Steps))
		return
	}

	logic, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Errorf("Expected LogicNode for empty if block, got %T", plan.Steps[0].Tree)
		return
	}

	if len(logic.Block) != 0 {
		t.Errorf("Expected empty LogicNode block, got %d steps", len(logic.Block))
	}
}

// =============================================================================
// Category 2: If-Else
// =============================================================================

// TestIfTrueElse verifies that `if true { A } else { B }` plans A, prunes B
func TestIfTrueElse(t *testing.T) {
	source := `if true { echo "A" } else { echo "B" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "A")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "A"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfFalseElse verifies that `if false { A } else { B }` prunes A, plans B
func TestIfFalseElse(t *testing.T) {
	source := `if false { echo "A" } else { echo "B" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "B")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "B"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// =============================================================================
// Category 3: Else-If Chains
// =============================================================================

// TestElseIfFirstMatch verifies first matching branch is taken
func TestElseIfFirstMatch(t *testing.T) {
	source := `if true { echo "A" } else if true { echo "B" } else { echo "C" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "A" - first match wins)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "A"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestElseIfSecondMatch verifies second branch taken when first is false
func TestElseIfSecondMatch(t *testing.T) {
	source := `if false { echo "A" } else if true { echo "B" } else { echo "C" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "B")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "B"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestElseIfFallthrough verifies else branch taken when all conditions false
func TestElseIfFallthrough(t *testing.T) {
	source := `if false { echo "A" } else if false { echo "B" } else { echo "C" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "C")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "C"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestElseIfNoElseAllFalse verifies all branches pruned when no else and all false
func TestElseIfNoElseAllFalse(t *testing.T) {
	source := `if false { echo "A" } else if false { echo "B" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// When all branches are pruned, we get a LogicNode with empty block
	// This preserves structure for display purposes
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step (empty LogicNode), got %d", len(plan.Steps))
		return
	}

	logic, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Errorf("Expected LogicNode for pruned if-elseif, got %T", plan.Steps[0].Tree)
		return
	}

	if len(logic.Block) != 0 {
		t.Errorf("Expected empty LogicNode block, got %d steps", len(logic.Block))
	}
}

// =============================================================================
// Category 4: Variable Conditions (Wave Resolution)
// =============================================================================

// TestIfVarConditionTrue verifies variable-based condition evaluation
func TestIfVarConditionTrue(t *testing.T) {
	source := `var ENV = "prod"
if @var.ENV == "prod" { echo "production" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "production")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "production"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfVarConditionFalse verifies variable-based condition prunes block
func TestIfVarConditionFalse(t *testing.T) {
	source := `var ENV = "dev"
if @var.ENV == "prod" { echo "production" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps (pruned)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps (pruned), got %d", len(plan.Steps))
	}
}

// TestIfVarConditionNotEqual verifies != operator
func TestIfVarConditionNotEqual(t *testing.T) {
	source := `var ENV = "prod"
if @var.ENV != "dev" { echo "not dev" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (condition is true: "prod" != "dev")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "not dev"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfBoolVarTrue verifies boolean variable as condition (truthy)
func TestIfBoolVarTrue(t *testing.T) {
	source := `var ENABLED = true
if @var.ENABLED { echo "enabled" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "enabled")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "enabled"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfBoolVarFalse verifies boolean variable as condition (falsy)
func TestIfBoolVarFalse(t *testing.T) {
	source := `var ENABLED = false
if @var.ENABLED { echo "enabled" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps (pruned)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps (pruned), got %d", len(plan.Steps))
	}
}

// =============================================================================
// Category 5: Comparison with Decorator on Right Side
// =============================================================================

// TestIfLiteralEqualsVar verifies "literal" == @var.X (decorator on right)
func TestIfLiteralEqualsVar(t *testing.T) {
	source := `var ENV = "prod"
if "prod" == @var.ENV { echo "matched" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (condition is true)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "matched"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfLiteralNotEqualsVar verifies "literal" != @var.X (decorator on right, false case)
func TestIfLiteralNotEqualsVar(t *testing.T) {
	source := `var ENV = "prod"
if "dev" == @var.ENV { echo "matched" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps (condition is false: "dev" != "prod")
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps (pruned), got %d", len(plan.Steps))
	}
}

// =============================================================================
// Category 6: Comparison with Decorators on Both Sides
// =============================================================================

// TestIfVarEqualsVarTrue verifies @var.A == @var.B (both decorators, equal)
func TestIfVarEqualsVarTrue(t *testing.T) {
	source := `var A = "same"
var B = "same"
if @var.A == @var.B { echo "equal" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (condition is true)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "equal"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestIfVarEqualsVarFalse verifies @var.A == @var.B (both decorators, not equal)
func TestIfVarEqualsVarFalse(t *testing.T) {
	source := `var A = "one"
var B = "two"
if @var.A == @var.B { echo "equal" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps (condition is false)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps (pruned), got %d", len(plan.Steps))
	}
}

// TestIfVarNotEqualsVar verifies @var.A != @var.B
func TestIfVarNotEqualsVar(t *testing.T) {
	source := `var A = "one"
var B = "two"
if @var.A != @var.B { echo "different" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (condition is true)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "different"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// =============================================================================
// Category 6b: Else-If with Variable Conditions
// =============================================================================

// TestElseIfWithVarConditions verifies else-if chain with variable conditions
func TestElseIfWithVarConditions(t *testing.T) {
	source := `var ENV = "staging"
if @var.ENV == "prod" { echo "production" } else if @var.ENV == "staging" { echo "staging" } else { echo "dev" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "staging" - second condition matches)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "staging"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestElseIfWithVarConditionsFallthrough verifies else-if falls through to else
func TestElseIfWithVarConditionsFallthrough(t *testing.T) {
	source := `var ENV = "local"
if @var.ENV == "prod" { echo "production" } else if @var.ENV == "staging" { echo "staging" } else { echo "dev" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "dev" - no conditions match, falls through to else)
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "dev"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// =============================================================================
// Category 7: Nested If (Multiple Waves)
// =============================================================================

// TestNestedIfBothTrue verifies nested if with both conditions true
func TestNestedIfBothTrue(t *testing.T) {
	source := `if true { if true { echo "inner" } }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (echo "inner")
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "inner"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// TestNestedIfInnerFalse verifies nested if with inner condition false
func TestNestedIfInnerFalse(t *testing.T) {
	source := `if true { if false { echo "inner" } }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// When inner if is pruned (false condition), the inner block should be completely removed
	// and we should get an empty outer block since there's nothing left to execute.
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step (LogicNode), got %d", len(plan.Steps))
	}

	logic, ok := plan.Steps[0].Tree.(*planfmt.LogicNode)
	if !ok {
		t.Fatalf("Expected LogicNode for nested if, got %T", plan.Steps[0].Tree)
	}

	if len(logic.Block) != 0 {
		t.Errorf("Expected 0 block steps (inner pruned), got %d", len(logic.Block))
	}
}

// TestNestedIfOuterFalse verifies nested if with outer condition false
func TestNestedIfOuterFalse(t *testing.T) {
	source := `if false { if true { echo "inner" } }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 0 steps (outer pruned, inner never evaluated)
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps (outer pruned), got %d", len(plan.Steps))
	}
}

// =============================================================================
// Category 10: Error Cases
// =============================================================================

// TestIfUndefinedVariable verifies error for undefined variable in condition
func TestIfUndefinedVariable(t *testing.T) {
	source := `if @var.UNDEFINED { echo "never" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})

	// Should fail with undefined variable error
	if err == nil {
		t.Fatal("Expected error for undefined variable, got nil")
	}

	// Error should mention the undefined variable
	if !contains(err.Error(), "UNDEFINED") && !contains(err.Error(), "not found") {
		t.Errorf("Expected error about undefined variable, got: %v", err)
	}
}

// TestIfUndefinedVariableComparison verifies error for undefined variable in comparison
func TestIfUndefinedVariableComparison(t *testing.T) {
	source := `if @var.UNDEFINED == "x" { echo "never" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})

	// Should fail with undefined variable error
	if err == nil {
		t.Fatal("Expected error for undefined variable, got nil")
	}

	// Error should mention the undefined variable
	if !contains(err.Error(), "UNDEFINED") && !contains(err.Error(), "not found") {
		t.Errorf("Expected error about undefined variable, got: %v", err)
	}
}

// =============================================================================
// Category 11: Function Mode
// =============================================================================

// TestIfInFunction verifies if works inside function definitions
func TestIfInFunction(t *testing.T) {
	source := `fun deploy { if true { echo "deploying" } }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "deploy",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step
	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	cmd := getCommandFromStep(&plan.Steps[0])
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", plan.Steps[0].Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	expected := `echo "deploying"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Silence unused import warning for cmp
var _ = cmp.Diff

// =============================================================================
// Category 12: Chained Operators (Fix for multi-operator conditions)
// =============================================================================

// TestIfChainedAndOr verifies that chained && and || operators are fully evaluated.
// Previously, only the first binary operator was evaluated.
func TestIfChainedAndOr(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectSteps int
		expectCmd   string
	}{
		{
			name:        "false || true evaluates to true",
			source:      `if false || true { echo "yes" }`,
			expectSteps: 1,
			expectCmd:   `echo "yes"`,
		},
		{
			name:        "true && false evaluates to false",
			source:      `if true && false { echo "yes" } else { echo "no" }`,
			expectSteps: 1,
			expectCmd:   `echo "no"`,
		},
		{
			name:        "false || true && false evaluates to false (right-to-left)",
			source:      `if false || true && false { echo "yes" } else { echo "no" }`,
			expectSteps: 1,
			expectCmd:   `echo "no"`,
		},
		{
			name:        "true || false && false evaluates to true",
			source:      `if true || false && false { echo "yes" } else { echo "no" }`,
			expectSteps: 1,
			expectCmd:   `echo "yes"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.ParseString(tt.source)
			if len(tree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", tree.Errors)
			}

			plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "",
			})
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			if len(plan.Steps) != tt.expectSteps {
				t.Fatalf("Expected %d steps, got %d", tt.expectSteps, len(plan.Steps))
			}

			if tt.expectSteps > 0 {
				step := plan.Steps[0]
				cmd := getCommandFromStep(&step)
				if cmd == nil {
					t.Fatalf("Expected CommandNode in step, got %T", step.Tree)
				}

				var cmdArg string
				for _, arg := range cmd.Args {
					if arg.Key == "command" {
						cmdArg = arg.Val.Str
						break
					}
				}

				if cmdArg != tt.expectCmd {
					t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", tt.expectCmd, cmdArg)
				}
			}
		})
	}
}

// =============================================================================
// Category 13: @env Decorator in Conditions (Fix for non-@var decorators)
// =============================================================================

// TestIfEnvDecorator verifies that @env decorators are properly resolved in conditions.
// Previously, non-@var decorators always returned true.
func TestIfEnvDecorator(t *testing.T) {
	// Set environment variable for test
	t.Setenv("TEST_ENV_VAR", "production")

	tests := []struct {
		name        string
		source      string
		expectSteps int
		expectCmd   string
	}{
		{
			name:        "@env.TEST_ENV_VAR equals production",
			source:      `if @env.TEST_ENV_VAR == "production" { echo "prod" } else { echo "dev" }`,
			expectSteps: 1,
			expectCmd:   `echo "prod"`,
		},
		{
			name:        "@env.TEST_ENV_VAR not equals staging",
			source:      `if @env.TEST_ENV_VAR == "staging" { echo "stage" } else { echo "other" }`,
			expectSteps: 1,
			expectCmd:   `echo "other"`,
		},
		{
			name:        "@env.TEST_ENV_VAR truthy check",
			source:      `if @env.TEST_ENV_VAR { echo "set" } else { echo "unset" }`,
			expectSteps: 1,
			expectCmd:   `echo "set"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.ParseString(tt.source)
			if len(tree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", tree.Errors)
			}

			plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "",
			})
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			if len(plan.Steps) != tt.expectSteps {
				t.Fatalf("Expected %d steps, got %d", tt.expectSteps, len(plan.Steps))
			}

			if tt.expectSteps > 0 {
				step := plan.Steps[0]
				cmd := getCommandFromStep(&step)
				if cmd == nil {
					t.Fatalf("Expected CommandNode in step, got %T", step.Tree)
				}

				var cmdArg string
				for _, arg := range cmd.Args {
					if arg.Key == "command" {
						cmdArg = arg.Val.Str
						break
					}
				}

				if cmdArg != tt.expectCmd {
					t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", tt.expectCmd, cmdArg)
				}
			}
		})
	}
}

// TestIfEnvUnset verifies that unset @env variables are handled correctly.
func TestIfEnvUnset(t *testing.T) {
	// Ensure the variable is NOT set
	t.Setenv("TEST_UNSET_VAR", "")

	source := `if @env.TEST_UNSET_VAR { echo "set" } else { echo "unset" }`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	cmd := getCommandFromStep(&step)
	if cmd == nil {
		t.Fatalf("Expected CommandNode in step, got %T", step.Tree)
	}

	var cmdArg string
	for _, arg := range cmd.Args {
		if arg.Key == "command" {
			cmdArg = arg.Val.Str
			break
		}
	}

	// Empty string is falsy, so should take else branch
	expected := `echo "unset"`
	if cmdArg != expected {
		t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", expected, cmdArg)
	}
}
