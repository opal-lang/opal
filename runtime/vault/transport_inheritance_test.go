package vault

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// testKey32 is a fixed 32-byte key for deterministic testing
var testKey32 = []byte("test-key-32-bytes-for-hmac-12345")

// ========== DeclaredTransport Tests ==========

// TestExpression_DeclaredTransport_RecordedAtDeclaration verifies that
// DeclaredTransport is set when the expression is declared, not when resolved.
func TestExpression_DeclaredTransport_RecordedAtDeclaration(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: We're in "local" transport
	// (default transport is "local")

	// WHEN: We declare a transport-sensitive variable
	exprID := v.DeclareVariableTransportSensitive("HOME", "@env.HOME")

	// THEN: The expression should have DeclaredTransport = "local"
	v.mu.Lock()
	expr := v.expressions[exprID]
	v.mu.Unlock()

	if expr == nil {
		t.Fatal("Expression not found")
	}
	if expr.DeclaredTransport != "local" {
		t.Errorf("DeclaredTransport = %q, want %q", expr.DeclaredTransport, "local")
	}
}

// TestExpression_DeclaredTransport_CapturesCurrentTransport verifies that
// DeclaredTransport captures the transport context at declaration time.
func TestExpression_DeclaredTransport_CapturesCurrentTransport(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: We enter an SSH transport
	v.EnterTransport("ssh:server1")

	// WHEN: We declare a transport-sensitive variable
	exprID := v.DeclareVariableTransportSensitive("REMOTE_HOME", "@env.HOME")

	// THEN: The expression should have DeclaredTransport = "ssh:server1"
	v.mu.Lock()
	expr := v.expressions[exprID]
	v.mu.Unlock()

	if expr == nil {
		t.Fatal("Expression not found")
	}
	if expr.DeclaredTransport != "ssh:server1" {
		t.Errorf("DeclaredTransport = %q, want %q", expr.DeclaredTransport, "ssh:server1")
	}
}

// TestExpression_DeclaredTransport_NotAffectedByResolution verifies that
// DeclaredTransport is NOT changed when ResolveAllTouched is called.
func TestExpression_DeclaredTransport_NotAffectedByResolution(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: We declare a variable in "local" transport
	exprID := v.DeclareVariableTransportSensitive("HOME", "@env.HOME")
	v.StoreUnresolvedValue(exprID, "/home/user")
	v.MarkTouched(exprID)

	// WHEN: We enter a different transport and resolve
	v.EnterTransport("ssh:server1")
	v.ResolveAllTouched()

	// THEN: DeclaredTransport should still be "local" (not "ssh:server1")
	v.mu.Lock()
	expr := v.expressions[exprID]
	v.mu.Unlock()

	if expr.DeclaredTransport != "local" {
		t.Errorf("DeclaredTransport = %q after resolution, want %q", expr.DeclaredTransport, "local")
	}
}

// ========== Transport Boundary Check Tests ==========

// TestCheckTransportBoundary_UsesDeclaredTransport verifies that
// checkTransportBoundary compares against DeclaredTransport, not exprTransport.
func TestCheckTransportBoundary_UsesDeclaredTransport(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A transport-sensitive variable declared in "local"
	exprID := v.DeclareVariableTransportSensitive("HOME", "@env.HOME")
	v.StoreUnresolvedValue(exprID, "/home/user")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// WHEN: We try to use it in a different transport
	v.EnterTransport("ssh:server1")
	err := v.CheckTransportBoundary(exprID)

	// THEN: Should fail with transport boundary error
	if err == nil {
		t.Fatal("Expected transport boundary error, got nil")
	}
	if !strings.Contains(err.Error(), "transport boundary") {
		t.Errorf("Expected transport boundary error, got: %v", err)
	}
}

// TestCheckTransportBoundary_AllowsSameTransport verifies that
// using a variable in the same transport where it was declared is allowed.
func TestCheckTransportBoundary_AllowsSameTransport(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A transport-sensitive variable declared in "ssh:server1"
	v.EnterTransport("ssh:server1")
	exprID := v.DeclareVariableTransportSensitive("REMOTE_HOME", "@env.HOME")
	v.StoreUnresolvedValue(exprID, "/home/remote")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// WHEN: We use it in the same transport
	// (still in "ssh:server1")
	err := v.CheckTransportBoundary(exprID)
	// THEN: Should succeed
	if err != nil {
		t.Errorf("Expected no error for same transport, got: %v", err)
	}
}

// TestCheckTransportBoundary_AllowsNonSensitive verifies that
// non-transport-sensitive variables can cross boundaries.
func TestCheckTransportBoundary_AllowsNonSensitive(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A non-transport-sensitive variable declared in "local"
	exprID := v.DeclareVariable("VERSION", "1.0.0")
	v.StoreUnresolvedValue(exprID, "1.0.0")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// WHEN: We try to use it in a different transport
	v.EnterTransport("ssh:server1")
	err := v.CheckTransportBoundary(exprID)
	// THEN: Should succeed (non-sensitive can cross boundaries)
	if err != nil {
		t.Errorf("Expected no error for non-sensitive variable, got: %v", err)
	}
}

// ========== IsExpressionTransportSensitive Tests ==========

// TestIsExpressionTransportSensitive_ReturnsTrueForSensitive verifies
// that IsExpressionTransportSensitive returns true for transport-sensitive expressions.
func TestIsExpressionTransportSensitive_ReturnsTrueForSensitive(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A transport-sensitive variable
	exprID := v.DeclareVariableTransportSensitive("HOME", "@env.HOME")

	// WHEN: We check if it's transport-sensitive
	result := v.IsExpressionTransportSensitive(exprID)

	// THEN: Should return true
	if !result {
		t.Error("Expected IsExpressionTransportSensitive to return true")
	}
}

// TestIsExpressionTransportSensitive_ReturnsFalseForNonSensitive verifies
// that IsExpressionTransportSensitive returns false for non-sensitive expressions.
func TestIsExpressionTransportSensitive_ReturnsFalseForNonSensitive(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A non-transport-sensitive variable
	exprID := v.DeclareVariable("VERSION", "1.0.0")

	// WHEN: We check if it's transport-sensitive
	result := v.IsExpressionTransportSensitive(exprID)

	// THEN: Should return false
	if result {
		t.Error("Expected IsExpressionTransportSensitive to return false")
	}
}

// TestIsExpressionTransportSensitive_ReturnsFalseForUnknown verifies
// that IsExpressionTransportSensitive returns false for unknown expressions.
func TestIsExpressionTransportSensitive_ReturnsFalseForUnknown(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// WHEN: We check an unknown expression
	result := v.IsExpressionTransportSensitive("unknown:12345678")

	// THEN: Should return false (safe default)
	if result {
		t.Error("Expected IsExpressionTransportSensitive to return false for unknown")
	}
}

// ========== GetUnresolvedValue Tests ==========

// TestGetUnresolvedValue_ReturnsStoredValue verifies that
// GetUnresolvedValue returns the value stored with StoreUnresolvedValue.
func TestGetUnresolvedValue_ReturnsStoredValue(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A variable with a stored value
	exprID := v.DeclareVariable("NAME", "literal:Aled")
	v.StoreUnresolvedValue(exprID, "Aled")

	// WHEN: We get the unresolved value
	value, ok := v.GetUnresolvedValue(exprID)

	// THEN: Should return the stored value
	if !ok {
		t.Fatal("Expected GetUnresolvedValue to return ok=true")
	}
	if diff := cmp.Diff("Aled", value); diff != "" {
		t.Errorf("GetUnresolvedValue mismatch (-want +got):\n%s", diff)
	}
}

// TestGetUnresolvedValue_ReturnsFalseForUnknown verifies that
// GetUnresolvedValue returns false for unknown expressions.
func TestGetUnresolvedValue_ReturnsFalseForUnknown(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// WHEN: We get an unknown expression
	_, ok := v.GetUnresolvedValue("unknown:12345678")

	// THEN: Should return ok=false
	if ok {
		t.Error("Expected GetUnresolvedValue to return ok=false for unknown")
	}
}

// TestGetUnresolvedValue_ReturnsFalseForNoValue verifies that
// GetUnresolvedValue returns false when no value has been stored.
func TestGetUnresolvedValue_ReturnsFalseForNoValue(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A variable declared but no value stored
	exprID := v.DeclareVariable("NAME", "literal:Aled")

	// WHEN: We get the unresolved value
	_, ok := v.GetUnresolvedValue(exprID)

	// THEN: Should return ok=false
	if ok {
		t.Error("Expected GetUnresolvedValue to return ok=false when no value stored")
	}
}

// ========== Variable Chaining Tests ==========

// TestVariableChaining_InheritsTransportSensitivity verifies that
// when a variable is assigned from another transport-sensitive variable,
// the new variable inherits transport sensitivity.
func TestVariableChaining_InheritsTransportSensitivity(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A transport-sensitive variable HOME
	homeExprID := v.DeclareVariableTransportSensitive("HOME", "@env.HOME")
	v.StoreUnresolvedValue(homeExprID, "/home/user")

	// WHEN: We check if HOME is transport-sensitive
	isSensitive := v.IsExpressionTransportSensitive(homeExprID)

	// THEN: HOME should be transport-sensitive
	if !isSensitive {
		t.Error("Expected HOME to be transport-sensitive")
	}

	// AND WHEN: We declare HOME2 = @var.HOME (chaining)
	// The planner should check IsExpressionTransportSensitive(homeExprID)
	// and call DeclareVariableTransportSensitive for HOME2
	home2ExprID := v.DeclareVariableTransportSensitive("HOME2", "chained:HOME")
	v.StoreUnresolvedValue(home2ExprID, "/home/user")

	// THEN: HOME2 should also be transport-sensitive
	if !v.IsExpressionTransportSensitive(home2ExprID) {
		t.Error("Expected HOME2 to be transport-sensitive (inherited from HOME)")
	}
}

// TestVariableChaining_NonSensitiveStaysNonSensitive verifies that
// when a variable is assigned from a non-sensitive variable,
// the new variable is also non-sensitive.
func TestVariableChaining_NonSensitiveStaysNonSensitive(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: A non-transport-sensitive variable VERSION
	versionExprID := v.DeclareVariable("VERSION", "literal:1.0.0")
	v.StoreUnresolvedValue(versionExprID, "1.0.0")

	// WHEN: We check if VERSION is transport-sensitive
	isSensitive := v.IsExpressionTransportSensitive(versionExprID)

	// THEN: VERSION should NOT be transport-sensitive
	if isSensitive {
		t.Error("Expected VERSION to NOT be transport-sensitive")
	}

	// AND WHEN: We declare COPY = @var.VERSION (chaining)
	// The planner should check IsExpressionTransportSensitive(versionExprID)
	// and call DeclareVariable (not TransportSensitive) for COPY
	copyExprID := v.DeclareVariable("COPY", "chained:VERSION")
	v.StoreUnresolvedValue(copyExprID, "1.0.0")

	// THEN: COPY should also NOT be transport-sensitive
	if v.IsExpressionTransportSensitive(copyExprID) {
		t.Error("Expected COPY to NOT be transport-sensitive")
	}
}

// ========== Integration: Transport Boundary with Chaining ==========

// TestTransportBoundary_ChainedVariableBlocked verifies that
// a chained transport-sensitive variable is blocked across boundaries.
func TestTransportBoundary_ChainedVariableBlocked(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: HOME declared as transport-sensitive in "local"
	homeExprID := v.DeclareVariableTransportSensitive("HOME", "@env.HOME")
	v.StoreUnresolvedValue(homeExprID, "/home/user")
	v.MarkTouched(homeExprID)

	// AND: HOME2 declared as transport-sensitive (chained from HOME) in "local"
	home2ExprID := v.DeclareVariableTransportSensitive("HOME2", "chained:HOME")
	v.StoreUnresolvedValue(home2ExprID, "/home/user")
	v.MarkTouched(home2ExprID)

	// Resolve all
	v.ResolveAllTouched()

	// WHEN: We try to use HOME2 in a different transport
	v.EnterTransport("ssh:server1")
	err := v.CheckTransportBoundary(home2ExprID)

	// THEN: Should fail with transport boundary error
	if err == nil {
		t.Fatal("Expected transport boundary error for chained variable")
	}
	if !strings.Contains(err.Error(), "transport boundary") {
		t.Errorf("Expected transport boundary error, got: %v", err)
	}
}

// TestTransportBoundary_ChainedNonSensitiveAllowed verifies that
// a chained non-sensitive variable is allowed across boundaries.
func TestTransportBoundary_ChainedNonSensitiveAllowed(t *testing.T) {
	v := NewWithPlanKey(testKey32)

	// GIVEN: VERSION declared as non-sensitive in "local"
	versionExprID := v.DeclareVariable("VERSION", "literal:1.0.0")
	v.StoreUnresolvedValue(versionExprID, "1.0.0")
	v.MarkTouched(versionExprID)

	// AND: COPY declared as non-sensitive (chained from VERSION) in "local"
	copyExprID := v.DeclareVariable("COPY", "chained:VERSION")
	v.StoreUnresolvedValue(copyExprID, "1.0.0")
	v.MarkTouched(copyExprID)

	// Resolve all
	v.ResolveAllTouched()

	// WHEN: We try to use COPY in a different transport
	v.EnterTransport("ssh:server1")
	err := v.CheckTransportBoundary(copyExprID)
	// THEN: Should succeed (non-sensitive can cross boundaries)
	if err != nil {
		t.Errorf("Expected no error for chained non-sensitive variable, got: %v", err)
	}
}
