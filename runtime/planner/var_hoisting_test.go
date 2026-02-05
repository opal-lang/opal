package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"
)

// ========== No Hoisting Tests ==========
//
// Variables cannot be used before declaration.
// This prevents confusing code and enforces clear declaration order.

func TestVarHoisting_UseBeforeDeclare_Errors(t *testing.T) {
	// Variable used before declaration should error

	source := []byte(`
echo "@var.NAME"
var NAME = "Aled"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})

	if err == nil {
		t.Fatal("Should error when variable used before declaration")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "NAME") {
		t.Errorf("Error should mention variable name 'NAME', got: %v", err)
	}

	// Error message now explicitly says "undefined variable (no hoisting allowed)"
	// which is more descriptive than "not found"
	if !strings.Contains(errMsg, "undefined variable") {
		t.Errorf("Error should mention variable is undefined, got: %v", err)
	}
}

func TestVarHoisting_DeclareBeforeUse_Works(t *testing.T) {
	// Variable declared before use should work

	source := []byte(`
var NAME = "Aled"
echo "@var.NAME"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	plan, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should work when variable declared before use, got error: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Plan should succeed - that's the main test
	// (Detailed command structure validation is in other tests)
}

func TestVarHoisting_UseInEarlierStep_DeclareInLaterStep_Errors(t *testing.T) {
	// Variable used in step 1, declared in step 2 should error
	// Steps execute sequentially, so this is use-before-declare

	source := []byte(`
echo "@var.COUNT"

var COUNT = 5
echo "done"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})

	if err == nil {
		t.Fatal("Should error when variable used in earlier step than declaration")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "COUNT") {
		t.Errorf("Error should mention variable name 'COUNT', got: %v", err)
	}
}

func TestVarHoisting_NeverDeclared_ClearError(t *testing.T) {
	// Variable never declared should give clear "not found" error

	source := []byte(`
echo "@var.UNDEFINED"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})

	if err == nil {
		t.Fatal("Should error when variable never declared")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "UNDEFINED") {
		t.Errorf("Error should mention variable name 'UNDEFINED', got: %v", err)
	}
	// Error message now explicitly says "undefined variable (no hoisting allowed)"
	if !strings.Contains(errMsg, "undefined variable") {
		t.Errorf("Error should say 'undefined variable' for undeclared variable, got: %v", err)
	}
}

// ========== Edge Cases ==========

func TestVarHoisting_UseInSameStepBeforeDeclare_Errors(t *testing.T) {
	// Variable used and declared in same step, but use comes first

	source := []byte(`
echo "@var.COUNT"
var COUNT = 5
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})

	if err == nil {
		t.Fatal("Should error when variable used before declaration in same step")
	}

	if !strings.Contains(err.Error(), "COUNT") {
		t.Errorf("Error should mention variable name, got: %v", err)
	}
}

func TestVarHoisting_MultipleUsesBeforeDeclare_AllError(t *testing.T) {
	// Multiple uses before declaration - first use should error

	source := []byte(`
echo "@var.NAME"
echo "@var.NAME again"
var NAME = "Aled"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})

	if err == nil {
		t.Fatal("Should error on first use before declaration")
	}

	if !strings.Contains(err.Error(), "NAME") {
		t.Errorf("Error should mention variable name, got: %v", err)
	}
}

func TestVarHoisting_MultipleDeclarations_UsesAfterWork(t *testing.T) {
	// Multiple declarations - uses after each declaration should work

	source := []byte(`
var FIRST = "one"
echo "@var.FIRST"
var SECOND = "two"
echo "@var.SECOND"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should work when variables declared before use, got error: %v", err)
	}
}

func TestVarHoisting_MultipleSteps_DeclarationOrder_Matters(t *testing.T) {
	// Variables declared in step 1 can be used in step 2

	source := []byte(`
var NAME = "Aled"

echo "@var.NAME"
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})
	if err != nil {
		t.Fatalf("Should work when variable declared in earlier step, got error: %v", err)
	}
}

func TestVarHoisting_SameVariableName_RedeclaredLater_FirstUseErrors(t *testing.T) {
	// Use variable, then declare it twice - first use should error

	source := []byte(`
echo "@var.COUNT"
var COUNT = 5
var COUNT = 10
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey(make([]byte, 32))
	_, err := Plan(tree.Events, tree.Tokens, Config{
		Vault: vlt,
	})

	if err == nil {
		t.Fatal("Should error when variable used before first declaration")
	}

	if !strings.Contains(err.Error(), "COUNT") {
		t.Errorf("Error should mention variable name, got: %v", err)
	}
}
