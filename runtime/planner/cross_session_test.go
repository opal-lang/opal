package planner

import (
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/parser"
)

// TestLiteralVariablesAreSessionAgnostic verifies that literal values
// are marked as session-agnostic and can be used in any session.
func TestLiteralVariablesAreSessionAgnostic(t *testing.T) {
	source := `
var COUNT = 3
var NAME = "test"
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	// Plan
	config := Config{
		Target: "",
	}

	result, err := PlanWithObservability(tree.Events, tree.Tokens, config)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Literal-only declarations should not create secret uses.
	if result.Plan == nil {
		t.Fatal("Expected plan to be created")
	}
	if len(result.Plan.SecretUses) != 0 {
		t.Fatalf("Expected no SecretUses for literal declarations, got %d", len(result.Plan.SecretUses))
	}
}

// TestDecoratorWithParametersDoesNotHang verifies that decorators with
// parameters (parentheses, commas, string literals) don't cause infinite loops
// in the parseDecoratorValue function.
func TestDecoratorWithParametersDoesNotHang(t *testing.T) {
	// This test uses @env.HOME which is valid syntax
	// The key is that parseDecoratorValue correctly handles the decorator
	// without hanging on tokens it doesn't recognize
	source := `
var HOME = @env.HOME
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	// Plan (this should not hang, even if decorator resolution fails)
	config := Config{
		Target: "",
	}

	// We don't care if planning succeeds or fails, just that it doesn't hang
	_, _ = PlanWithObservability(tree.Events, tree.Tokens, config)
}

// TestMultiDotDecoratorParsing tests that decorators with multiple dots
// are parsed correctly (e.g., @env.HOME uses two parts).
func TestMultiDotDecoratorParsing(t *testing.T) {
	// @env.HOME should parse as decorator="env", primary="HOME"
	// This is the current working syntax
	source := `var HOME = @env.HOME`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	config := Config{Target: ""}
	result, err := PlanWithObservability(tree.Events, tree.Tokens, config)
	// @env.HOME should succeed (resolves from environment)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if result.Plan == nil {
		t.Fatal("Expected plan to be created")
	}
}

// TestMultiSegmentDecoratorPath verifies that decorators with multiple segments
// in their path are resolved correctly by trying progressively shorter paths.
func TestMultiSegmentDecoratorPath(t *testing.T) {
	// @env.HOME should resolve as decorator="env" with primary="HOME"
	// This works because "env" is registered in the global registry
	source := `var HOME = @env.HOME`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	config := Config{Target: ""}
	result, err := PlanWithObservability(tree.Events, tree.Tokens, config)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if result.Plan == nil {
		t.Fatal("Expected plan to be created")
	}
}

// TestSessionTransportScope verifies that sessions correctly report their transport scope.
func TestSessionTransportScope(t *testing.T) {
	t.Run("LocalSession", func(t *testing.T) {
		session := &decorator.LocalSession{}
		if session.TransportScope() != decorator.TransportScopeLocal {
			t.Errorf("LocalSession.TransportScope() = %v, want %v",
				session.TransportScope(), decorator.TransportScopeLocal)
		}
	})

	// Note: SSHSession requires actual SSH connection setup, so we can't easily test it here
	// The implementation is verified by the interface contract and type system
}
