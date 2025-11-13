package vault

import (
	"bytes"
	"sync"
	"testing"
)

// testKey is a fixed 32-byte key for deterministic testing
var testKey = []byte("test-key-32-bytes-for-hmac-12345")

// ========== Path Tracking Tests ==========

// TestVault_PathTracking_SingleDecorator tests building a simple path
// with one decorator instance.
func TestVault_PathTracking_SingleDecorator(t *testing.T) {
	v := New()

	// GIVEN: We enter a step and a decorator
	v.Push("step-1")
	v.Push("@shell")

	// WHEN: We build the site path for a parameter
	path := v.BuildSitePath("command")

	// THEN: Path should include decorator with index [0]
	expected := "root/step-1/@shell[0]/params/command"
	if path != expected {
		t.Errorf("BuildSitePath() = %q, want %q", path, expected)
	}
}

// TestVault_PathTracking_MultipleInstances tests that multiple instances
// of the same decorator get different indices.
func TestVault_PathTracking_MultipleInstances(t *testing.T) {
	v := New()

	// GIVEN: Three shell commands in three steps
	v.Push("step-1")
	v.Push("@shell")
	path1 := v.BuildSitePath("command")
	v.Pop() // Pop @shell
	v.Pop() // Pop step-1

	v.ResetCounts() // Reset for new step
	v.Push("step-2")
	v.Push("@shell")
	path2 := v.BuildSitePath("command")
	v.Pop() // Pop @shell
	v.Pop() // Pop step-2

	v.ResetCounts() // Reset for new step
	v.Push("step-3")
	v.Push("@shell")
	path3 := v.BuildSitePath("command")
	v.Pop() // Pop @shell
	v.Pop() // Pop step-3

	// THEN: Each should have different step but same decorator index [0]
	expected := []string{
		"root/step-1/@shell[0]/params/command",
		"root/step-2/@shell[0]/params/command",
		"root/step-3/@shell[0]/params/command",
	}

	paths := []string{path1, path2, path3}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("Path[%d] = %q, want %q", i, path, expected[i])
		}
	}
}

// TestVault_PathTracking_NestedDecorators tests building paths through
// nested decorator contexts.
func TestVault_PathTracking_NestedDecorators(t *testing.T) {
	v := New()

	// GIVEN: Nested decorators @retry -> @timeout -> @shell
	v.Push("@retry")
	v.Push("@timeout")
	v.Push("@shell")

	// WHEN: We build the site path
	path := v.BuildSitePath("command")

	// THEN: Path should show full nesting
	expected := "root/@retry[0]/@timeout[0]/@shell[0]/params/command"
	if path != expected {
		t.Errorf("BuildSitePath() = %q, want %q", path, expected)
	}
}

// TestVault_PathTracking_MultipleDecoratorsAtSameLevel tests that
// different decorators at the same level get independent indices.
func TestVault_PathTracking_MultipleDecoratorsAtSameLevel(t *testing.T) {
	v := New()

	v.Push("step-1")

	// First shell command
	v.Push("@shell")
	path1 := v.BuildSitePath("command")
	v.Pop()

	// Second shell command
	v.Push("@shell")
	path2 := v.BuildSitePath("command")
	v.Pop()

	// A retry decorator
	v.Push("@retry")
	path3 := v.BuildSitePath("times")
	v.Pop()

	// THEN: Shell commands get [0] and [1], retry gets [0]
	expected := []string{
		"root/step-1/@shell[0]/params/command",
		"root/step-1/@shell[1]/params/command",
		"root/step-1/@retry[0]/params/times",
	}

	paths := []string{path1, path2, path3}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("Path[%d] = %q, want %q", i, path, expected[i])
		}
	}
}

// TestVault_PathTracking_ResetCountsPerStep tests that decorator counts
// reset when entering a new step.
func TestVault_PathTracking_ResetCountsPerStep(t *testing.T) {
	v := New()

	// Step 1: Two shell commands
	v.Push("step-1")
	v.Push("@shell")
	v.Pop()
	v.Push("@shell")
	path1 := v.BuildSitePath("command")
	v.Pop()
	v.Pop() // Pop step-1

	// Step 2: One shell command (should be [0] again)
	v.ResetCounts() // Reset for new step
	v.Push("step-2")
	v.Push("@shell")
	path2 := v.BuildSitePath("command")
	v.Pop()
	v.Pop() // Pop step-2

	// THEN: Step 1 has @shell[1], Step 2 has @shell[0]
	if path1 != "root/step-1/@shell[1]/params/command" {
		t.Errorf("Step 1 path = %q, want %q", path1, "root/step-1/@shell[1]/params/command")
	}
	if path2 != "root/step-2/@shell[0]/params/command" {
		t.Errorf("Step 2 path = %q, want %q", path2, "root/step-2/@shell[0]/params/command")
	}
}

// ========== Transport Boundary Tests ==========

// TestVault_EnterExitTransport tests transport scope tracking.
func TestVault_EnterExitTransport(t *testing.T) {
	v := New()

	// GIVEN: We start in local transport
	if v.CurrentTransport() != "local" {
		t.Errorf("CurrentTransport = %q, want %q", v.CurrentTransport(), "local")
	}

	// WHEN: We enter SSH transport
	v.EnterTransport("ssh:server1")

	// THEN: Current transport changes
	if v.CurrentTransport() != "ssh:server1" {
		t.Errorf("CurrentTransport = %q, want %q", v.CurrentTransport(), "ssh:server1")
	}

	// WHEN: We exit transport
	v.ExitTransport()

	// THEN: Back to local
	if v.CurrentTransport() != "local" {
		t.Errorf("CurrentTransport = %q, want %q", v.CurrentTransport(), "local")
	}
}

// TestVault_TransportBoundaryViolation tests that crossing boundaries is blocked.
func TestVault_TransportBoundaryViolation(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved in local transport
	exprID := v.DeclareVariable("LOCAL_TOKEN", "@env.TOKEN")
	v.MarkResolved(exprID, "secret")
	v.MarkTouched(exprID)

	// AND: Record reference in local transport (allowed)
	v.Push("step-1")
	v.Push("@shell")
	err := v.RecordReference(exprID, "command")
	if err != nil {
		t.Fatalf("RecordReference should succeed in same transport: %v", err)
	}

	// WHEN: We enter SSH transport and try to access it
	v.EnterTransport("ssh:untrusted")
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command") // Recording is allowed

	// THEN: Access should fail with transport boundary violation
	_, err = v.Access(exprID, "command")
	if err == nil {
		t.Fatal("Expected transport boundary error, got nil")
	}
	if !containsString(err.Error(), "transport boundary") {
		t.Errorf("Error should mention transport boundary, got: %v", err)
	}
}

// TestVault_SameTransportAllowed tests that same transport is allowed.
func TestVault_SameTransportAllowed(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved in local transport
	exprID := v.DeclareVariable("LOCAL_VAR", "@env.HOME")
	v.MarkResolved(exprID, "value") // MarkResolved captures transport automatically
	v.MarkTouched(exprID)

	// WHEN: We use it in same transport
	v.Push("step-1")
	v.Push("@shell")
	err := v.RecordReference(exprID, "command")
	// THEN: Should succeed
	if err != nil {
		t.Errorf("Expected no error in same transport, got: %v", err)
	}
}

// ========== Execution Path Tracking Tests ==========

// TestVault_MarkTouched tests marking expressions as touched (in execution path).
func TestVault_MarkTouched(t *testing.T) {
	v := New()

	// GIVEN: We have two expressions
	id1 := v.DeclareVariable("USED", "@env.HOME")
	id2 := v.DeclareVariable("UNUSED", "@env.PATH")

	// WHEN: We mark one as touched
	v.MarkTouched(id1)

	// THEN: USED is touched, UNUSED is not
	if !v.IsTouched(id1) {
		t.Error("USED should be marked as touched")
	}
	if v.IsTouched(id2) {
		t.Error("UNUSED should not be marked as touched")
	}
}

// TestVault_PruneUntouched tests removing expressions not in execution path.
func TestVault_PruneUntouched(t *testing.T) {
	v := New()

	// GIVEN: Three expressions, only two touched
	id1 := v.DeclareVariable("TOUCHED1", "@env.HOME")
	id2 := v.DeclareVariable("TOUCHED2", "@env.USER")
	id3 := v.DeclareVariable("UNTOUCHED", "@env.PATH")

	v.MarkTouched(id1)
	v.MarkTouched(id2)

	// WHEN: We prune untouched
	v.PruneUntouched()

	// THEN: Only touched expressions remain
	if v.expressions[id1] == nil {
		t.Error("TOUCHED1 should still exist")
	}
	if v.expressions[id2] == nil {
		t.Error("TOUCHED2 should still exist")
	}
	if v.expressions[id3] != nil {
		t.Error("UNTOUCHED should be pruned")
	}
}

// ========== BuildSecretUses Tests ==========

// TestVault_BuildSecretUses tests building final SecretUse list.
func TestVault_BuildSecretUses(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variables with references
	exprID := v.DeclareVariable("API_KEY", "sk-secret")
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	v.Pop()

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// Resolve expression (normally done during planning)
	v.expressions[exprID].Value = "sk-secret"
	v.expressions[exprID].Resolved = true
	v.expressions[exprID].DisplayID = "opal:v:ABC123"

	// Mark as touched (in execution path)
	v.MarkTouched(exprID)

	// WHEN: We build final SecretUses
	uses := v.BuildSecretUses()

	// THEN: Should have 2 SecretUse entries
	if len(uses) != 2 {
		t.Fatalf("Expected 2 SecretUses, got %d", len(uses))
	}

	// Both should have same DisplayID
	if uses[0].DisplayID != uses[1].DisplayID {
		t.Error("Same secret should have same DisplayID")
	}

	// Different SiteIDs
	if uses[0].SiteID == uses[1].SiteID {
		t.Error("Different sites should have different SiteIDs")
	}
}

// TestVault_BuildSecretUses_RequiresDisplayID tests that expressions without
// DisplayID are skipped (not yet resolved).
func TestVault_BuildSecretUses_RequiresDisplayID(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable with reference but no DisplayID (not resolved yet)
	exprID := v.DeclareVariable("UNRESOLVED", "value")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// Don't assign DisplayID - simulates unresolved expression

	// WHEN: We build SecretUses
	uses := v.BuildSecretUses()

	// THEN: Should be empty (no DisplayID = not resolved = skip)
	if len(uses) != 0 {
		t.Fatalf("Expected 0 SecretUses (unresolved), got %d", len(uses))
	}
}

// TestVault_BuildSecretUses_OnlyTouched tests that only touched expressions are included.
func TestVault_BuildSecretUses_OnlyTouched(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Two expressions with references, only one touched
	id1 := v.DeclareVariable("TOUCHED", "@env.HOME")
	id2 := v.DeclareVariable("UNTOUCHED", "@env.PATH")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(id1, "command")
	v.RecordReference(id2, "command")

	// Resolve both
	v.expressions[id1].Value = "value1"
	v.expressions[id1].Resolved = true
	v.expressions[id1].DisplayID = "opal:v:AAA"
	v.expressions[id2].Value = "value2"
	v.expressions[id2].Resolved = true
	v.expressions[id2].DisplayID = "opal:v:BBB"

	// Mark only one as touched
	v.MarkTouched(id1)

	// WHEN: We build SecretUses
	uses := v.BuildSecretUses()

	// THEN: Only TOUCHED is included
	if len(uses) != 1 {
		t.Fatalf("Expected 1 SecretUse, got %d", len(uses))
	}
	if uses[0].DisplayID != "opal:v:AAA" {
		t.Errorf("Expected DisplayID opal:v:AAA, got %s", uses[0].DisplayID)
	}
}

// ========== DeclareVariable Tests ==========

// TestVault_DeclareVariable_StoresExpression tests that the expression is stored with hash-based ID.
func TestVault_DeclareVariable_StoresExpression(t *testing.T) {
	v := New()

	// WHEN: We declare a variable
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// THEN: Should return hash-based ID (not variable name)
	if exprID == "API_KEY" {
		t.Error("DeclareVariable() should return hash-based ID, not variable name")
	}

	// THEN: Expression should be stored with hash-based ID
	if v.expressions[exprID] == nil {
		t.Error("Expression should be stored")
	}
	if v.expressions[exprID].Raw != "@env.API_KEY" {
		t.Errorf("Expression.Raw = %q, want %q", v.expressions[exprID].Raw, "@env.API_KEY")
	}
}

// ========== TrackExpression Tests ==========

// TestVault_TrackExpression_ReturnsHashBasedID tests that TrackExpression returns hash-based ID.
func TestVault_TrackExpression_ReturnsHashBasedID(t *testing.T) {
	v := New()

	// WHEN: We track a direct decorator call
	exprID := v.TrackExpression("@env.HOME")

	// THEN: Should return hash-based ID with transport
	// Format: "transport:decorator:params:hash"
	if exprID == "" {
		t.Error("TrackExpression() should return non-empty ID")
	}
	if exprID == "@env.HOME" {
		t.Error("TrackExpression() should not return raw expression as ID")
	}
	// Should include transport prefix
	if len(exprID) < 6 || exprID[:6] != "local:" {
		t.Errorf("TrackExpression() = %q, should start with 'local:'", exprID)
	}
}

// TestVault_TrackExpression_IncludesTransport tests that expression ID includes transport context.
func TestVault_TrackExpression_IncludesTransport(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: We're in local transport
	localID := v.TrackExpression("@env.HOME")

	// WHEN: We enter SSH transport
	v.EnterTransport("ssh:server1")
	sshID := v.TrackExpression("@env.HOME")

	// THEN: IDs should be different (different transport context)
	if localID == sshID {
		t.Errorf("Same expression in different transports should have different IDs: local=%q, ssh=%q", localID, sshID)
	}

	// Both should include their transport
	if len(localID) < 6 || localID[:6] != "local:" {
		t.Errorf("Local ID should start with 'local:', got %q", localID)
	}
	if len(sshID) < 12 || sshID[:12] != "ssh:server1:" {
		t.Errorf("SSH ID should start with 'ssh:server1:', got %q", sshID)
	}
}

// TestVault_TrackExpression_Deterministic tests that same expression returns same ID.
func TestVault_TrackExpression_Deterministic(t *testing.T) {
	v := New()

	// WHEN: We track the same expression twice
	id1 := v.TrackExpression("@env.HOME")
	id2 := v.TrackExpression("@env.HOME")

	// THEN: Should return same ID (deterministic)
	if id1 != id2 {
		t.Errorf("Same expression should return same ID: id1=%q, id2=%q", id1, id2)
	}
}

// ========== Access Tests ==========

// TestVault_Access_ChecksSiteID tests that Access checks SiteID authorization.
func TestVault_Access_ChecksSiteID(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable with resolved value
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// Resolve expression (sets Value, Resolved, and transport)
	v.MarkResolved(exprID, "sk-secret-123")

	// Record authorized site
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: We try to access at authorized site (current site)
	value, err := v.Access(exprID, "command")
	// THEN: Should succeed
	if err != nil {
		t.Errorf("Access() at authorized site should succeed, got error: %v", err)
	}
	if value != "sk-secret-123" {
		t.Errorf("Access() = %q, want %q", value, "sk-secret-123")
	}
}

// TestVault_Access_RejectsUnauthorizedSite tests that Access rejects unauthorized sites.
func TestVault_Access_RejectsUnauthorizedSite(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable with resolved value
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")
	v.MarkResolved(exprID, "sk-secret-123")

	// Record authorized site
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	v.Pop()

	// WHEN: We try to access at different site (not authorized)
	v.Push("step-1")
	v.Push("@timeout")
	value, err := v.Access(exprID, "duration")

	// THEN: Should fail with authorization error
	if err == nil {
		t.Error("Access() at unauthorized site should fail")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %v", value)
	}
	if err != nil && !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// TestVault_Access_ChecksTransportBoundary tests that Access checks transport boundaries.

// TestVault_Access_UnresolvedExpression tests that Access fails for unresolved expressions.
func TestVault_Access_UnresolvedExpression(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable that hasn't been resolved yet
	exprID := v.DeclareVariable("UNRESOLVED", "@env.FOO")

	// Record use-site
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: We try to access
	value, err := v.Access(exprID, "command")

	// THEN: Should fail (not resolved yet)
	if err == nil {
		t.Error("Access() on unresolved expression should fail")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %v", value)
	}
	if err != nil && !containsString(err.Error(), "not resolved") {
		t.Errorf("Error should mention 'not resolved', got: %v", err)
	}
}

// ========== Pruning Tests ==========

// TestVault_PruneUnusedExpressions tests removing expressions with no references.
func TestVault_PruneUnusedExpressions(t *testing.T) {
	v := New()

	// GIVEN: Two variables, one used, one unused
	id1 := v.DeclareVariable("USED", "sk-used")
	id2 := v.DeclareVariable("UNUSED", "sk-unused")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(id1, "command")

	// WHEN: We prune unused expressions
	v.PruneUnused()

	// THEN: Only USED should remain
	if v.expressions[id1] == nil {
		t.Error("USED expression should still exist")
	}
	if v.expressions[id2] != nil {
		t.Error("UNUSED expression should be pruned")
	}
}

// TestVault_EndToEnd_PruneAndBuild tests complete workflow.
func TestVault_EndToEnd_PruneAndBuild(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Multiple variables, some used, some unused
	id1 := v.DeclareVariable("USED_SECRET", "sk-used")
	id2 := v.DeclareVariable("UNUSED_SECRET", "sk-unused")
	id3 := v.DeclareVariable("ANOTHER_USED", "value")

	// Only reference USED_SECRET and ANOTHER_USED
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(id1, "command")
	v.RecordReference(id3, "command")

	// WHEN: We prune and build
	v.PruneUnused()

	// Resolve expressions (normally done during planning)
	if v.expressions[id1] != nil {
		v.expressions[id1].Value = "sk-used"
		v.expressions[id1].Resolved = true
		v.expressions[id1].DisplayID = "opal:v:AAA"
	}
	if v.expressions[id3] != nil {
		v.expressions[id3].Value = "value"
		v.expressions[id3].Resolved = true
		v.expressions[id3].DisplayID = "opal:v:BBB"
	}

	// Mark as touched (in execution path)
	v.MarkTouched(id1)
	v.MarkTouched(id3)

	uses := v.BuildSecretUses()

	// THEN: Should have 2 SecretUses (UNUSED_SECRET pruned)
	if len(uses) != 2 {
		t.Fatalf("Expected 2 SecretUses, got %d", len(uses))
	}

	// Verify UNUSED_SECRET was pruned
	if v.expressions[id2] != nil {
		t.Error("UNUSED_SECRET should have been pruned")
	}
}

// ========== Scope-Aware Variable Tests ==========

// TestVault_LookupVariable tests basic variable lookup.
func TestVault_LookupVariable(t *testing.T) {
	v := New()

	// GIVEN: Variable declared at root
	exprID := v.DeclareVariable("NAME", "Aled")

	// WHEN: We lookup the variable
	foundID, err := v.LookupVariable("NAME")
	// THEN: Should find it
	if err != nil {
		t.Fatalf("LookupVariable() failed: %v", err)
	}
	if foundID != exprID {
		t.Errorf("LookupVariable() = %q, want %q", foundID, exprID)
	}
}

// TestVault_LookupVariable_NotFound tests error handling for missing variables.
func TestVault_LookupVariable_NotFound(t *testing.T) {
	v := New()

	// WHEN: We lookup a non-existent variable
	_, err := v.LookupVariable("DOES_NOT_EXIST")

	// THEN: Should return error
	if err == nil {
		t.Error("LookupVariable() should fail for non-existent variable")
	}
	if err != nil && !containsString(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// TestVault_DeclareVariable_RegistersName tests that DeclareVariable registers the name.
func TestVault_DeclareVariable_RegistersName(t *testing.T) {
	v := New()

	// GIVEN: We declare a variable
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// WHEN: We lookup the variable
	foundID, err := v.LookupVariable("API_KEY")
	// THEN: Should find it with same exprID
	if err != nil {
		t.Fatalf("LookupVariable() failed: %v", err)
	}
	if foundID != exprID {
		t.Errorf("LookupVariable() = %q, want %q", foundID, exprID)
	}
}

// TestVault_ScopeAwareVariables_BasicDeclaration tests variable storage in root scope.
func TestVault_ScopeAwareVariables_BasicDeclaration(t *testing.T) {
	v := New()

	// GIVEN: Variable declared at root
	exprID := v.DeclareVariable("COUNT", "5")

	// THEN: Should be stored in root scope
	rootScope := v.scopes["root"]
	if rootScope == nil {
		t.Fatal("Root scope should exist")
	}
	if rootScope.vars["COUNT"] != exprID {
		t.Errorf("Root scope vars[COUNT] = %q, want %q", rootScope.vars["COUNT"], exprID)
	}
}

// TestVault_ScopeAwareVariables_ParentToChild tests parent → child variable flow.
func TestVault_ScopeAwareVariables_ParentToChild(t *testing.T) {
	v := New()

	// GIVEN: Variable declared at root
	exprID := v.DeclareVariable("COUNT", "5")

	// WHEN: We enter a child scope
	v.Push("@retry")

	// THEN: Child can lookup parent's variable
	foundID, err := v.LookupVariable("COUNT")
	if err != nil {
		t.Fatalf("LookupVariable() failed: %v", err)
	}
	if foundID != exprID {
		t.Errorf("LookupVariable() = %q, want %q", foundID, exprID)
	}
}

// TestVault_ScopeAwareVariables_Shadowing tests variable shadowing.
func TestVault_ScopeAwareVariables_Shadowing(t *testing.T) {
	v := New()

	// GIVEN: Variable declared at root
	rootExprID := v.DeclareVariable("COUNT", "5")

	// WHEN: We enter child scope and shadow the variable
	v.Push("@retry")
	childExprID := v.DeclareVariable("COUNT", "3")

	// THEN: Child lookup should find child's value (shadows parent)
	foundID, err := v.LookupVariable("COUNT")
	if err != nil {
		t.Fatalf("LookupVariable() in child failed: %v", err)
	}
	if foundID != childExprID {
		t.Errorf("Child LookupVariable() = %q, want %q (child shadows parent)", foundID, childExprID)
	}

	// WHEN: We exit child scope
	v.Pop()

	// THEN: Parent lookup should find parent's value (unchanged)
	foundID, err = v.LookupVariable("COUNT")
	if err != nil {
		t.Fatalf("LookupVariable() in parent failed: %v", err)
	}
	if foundID != rootExprID {
		t.Errorf("Parent LookupVariable() = %q, want %q (parent unchanged)", foundID, rootExprID)
	}
}

// TestVault_ScopeAwareVariables_NotFound tests not found in any scope.
func TestVault_ScopeAwareVariables_NotFound(t *testing.T) {
	v := New()

	// GIVEN: Some variables exist
	v.DeclareVariable("EXISTS", "value")

	// WHEN: We enter nested scopes and lookup non-existent variable
	v.Push("@retry")
	v.Push("@timeout")

	_, err := v.LookupVariable("DOES_NOT_EXIST")

	// THEN: Should return error
	if err == nil {
		t.Error("LookupVariable() should fail for non-existent variable")
	}
	if err != nil && !containsString(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// TestVault_ScopeAwareVariables_NestedScopes tests multiple scope levels.
func TestVault_ScopeAwareVariables_NestedScopes(t *testing.T) {
	v := New()

	// GIVEN: Variables at different scope levels
	aID := v.DeclareVariable("A", "1")

	v.Push("@retry")
	bID := v.DeclareVariable("B", "2")

	v.Push("@timeout")
	cID := v.DeclareVariable("C", "3")

	// WHEN: We lookup from deepest scope
	// THEN: Should find all three variables
	foundA, err := v.LookupVariable("A")
	if err != nil {
		t.Fatalf("LookupVariable(A) failed: %v", err)
	}
	if foundA != aID {
		t.Errorf("LookupVariable(A) = %q, want %q", foundA, aID)
	}

	foundB, err := v.LookupVariable("B")
	if err != nil {
		t.Fatalf("LookupVariable(B) failed: %v", err)
	}
	if foundB != bID {
		t.Errorf("LookupVariable(B) = %q, want %q", foundB, bID)
	}

	foundC, err := v.LookupVariable("C")
	if err != nil {
		t.Fatalf("LookupVariable(C) failed: %v", err)
	}
	if foundC != cID {
		t.Errorf("LookupVariable(C) = %q, want %q", foundC, cID)
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// SecretProvider Tests
// ============================================================================

// TestVault_SecretProvider_NoExpressions tests provider with no expressions
func TestVault_SecretProvider_NoExpressions(t *testing.T) {
	v := New()
	provider := v.SecretProvider()

	chunk := []byte("No secrets here")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	if !bytes.Equal(result, chunk) {
		t.Errorf("Expected unchanged chunk, got %q", result)
	}
}

// TestVault_SecretProvider_UnresolvedExpression tests provider with unresolved expression
func TestVault_SecretProvider_UnresolvedExpression(t *testing.T) {
	v := New()

	// Declare but don't resolve
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")
	_ = exprID

	provider := v.SecretProvider()
	chunk := []byte("The key is: secret123")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Should not replace (expression not resolved)
	if !bytes.Equal(result, chunk) {
		t.Errorf("Expected unchanged chunk, got %q", result)
	}
}

// TestVault_SecretProvider_ResolvedExpression tests provider with resolved expression
func TestVault_SecretProvider_ResolvedExpression(t *testing.T) {
	v := NewWithPlanKey(testKey)

	// Declare and resolve
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")
	v.MarkResolved(exprID, "secret123")

	provider := v.SecretProvider()
	chunk := []byte("The key is: secret123")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Should replace with DisplayID
	if bytes.Contains(result, []byte("secret123")) {
		t.Errorf("Secret not replaced: %q", result)
	}

	// Should contain DisplayID (format: opal:v:...)
	if !bytes.Contains(result, []byte("opal:v:")) {
		t.Errorf("DisplayID not found in result: %q", result)
	}
}

// TestVault_SecretProvider_MultipleSecrets tests provider with multiple secrets
func TestVault_SecretProvider_MultipleSecrets(t *testing.T) {
	v := NewWithPlanKey(testKey)

	// Declare and resolve multiple secrets
	id1 := v.DeclareVariable("KEY1", "@env.KEY1")
	v.MarkResolved(id1, "secret1")

	id2 := v.DeclareVariable("KEY2", "@env.KEY2")
	v.MarkResolved(id2, "secret2")

	provider := v.SecretProvider()
	chunk := []byte("First: secret1, Second: secret2")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Both secrets should be replaced
	if bytes.Contains(result, []byte("secret1")) {
		t.Errorf("secret1 not replaced: %q", result)
	}
	if bytes.Contains(result, []byte("secret2")) {
		t.Errorf("secret2 not replaced: %q", result)
	}

	// Should contain DisplayIDs
	if !bytes.Contains(result, []byte("opal:v:")) {
		t.Errorf("DisplayIDs not found in result: %q", result)
	}
}

// TestVault_SecretProvider_LongestFirst tests longest-first matching
func TestVault_SecretProvider_LongestFirst(t *testing.T) {
	v := NewWithPlanKey(testKey)

	// Declare overlapping secrets (use different raw expressions)
	id1 := v.DeclareVariable("SHORT", "@env.SHORT")
	v.MarkResolved(id1, "SECRET")

	id2 := v.DeclareVariable("LONG", "@env.LONG")
	v.MarkResolved(id2, "SECRET_EXTENDED")

	provider := v.SecretProvider()
	chunk := []byte("Value: SECRET_EXTENDED")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Should replace entire "SECRET_EXTENDED", not just "SECRET"
	if bytes.Contains(result, []byte("SECRET")) {
		t.Errorf("Secret not fully replaced (longest-first failed): %q", result)
	}

	// Should have one DisplayID
	count := bytes.Count(result, []byte("opal:v:"))
	if count != 1 {
		t.Errorf("Expected 1 DisplayID, got %d in: %q", count, result)
	}
}

// TestVault_SecretProvider_EmptyValue tests provider with empty resolved value
func TestVault_SecretProvider_EmptyValue(t *testing.T) {
	v := NewWithPlanKey(testKey)

	// Resolve to empty string
	id := v.DeclareVariable("EMPTY", "@env.EMPTY")
	v.MarkResolved(id, "")

	provider := v.SecretProvider()
	chunk := []byte("No secrets here")
	result, err := provider.HandleChunk(chunk)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}

	// Should not replace (empty value ignored)
	if !bytes.Equal(result, chunk) {
		t.Errorf("Expected unchanged chunk, got %q", result)
	}
}

// TestVault_SecretProvider_ThreadSafety tests concurrent access to provider
func TestVault_SecretProvider_ThreadSafety(t *testing.T) {
	v := NewWithPlanKey(testKey)

	// Declare and resolve
	id := v.DeclareVariable("KEY", "@env.KEY")
	v.MarkResolved(id, "secret")

	provider := v.SecretProvider()

	// Call HandleChunk concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chunk := []byte("The secret is: secret")
			result, err := provider.HandleChunk(chunk)
			if err != nil {
				t.Errorf("HandleChunk failed: %v", err)
			}
			if bytes.Contains(result, []byte("secret")) {
				t.Errorf("Secret not replaced in concurrent call: %q", result)
			}
		}()
	}
	wg.Wait()
}

// TestVault_SecretProvider_DynamicPatterns tests that patterns update dynamically
func TestVault_SecretProvider_DynamicPatterns(t *testing.T) {
	v := NewWithPlanKey(testKey)

	// Declare and resolve first secret
	id1 := v.DeclareVariable("KEY1", "@env.KEY1")
	v.MarkResolved(id1, "secret1")

	provider := v.SecretProvider()

	// First call - should replace secret1
	chunk1 := []byte("Value: secret1")
	result1, err := provider.HandleChunk(chunk1)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}
	if bytes.Contains(result1, []byte("secret1")) {
		t.Errorf("secret1 not replaced: %q", result1)
	}

	// Add second secret
	id2 := v.DeclareVariable("KEY2", "@env.KEY2")
	v.MarkResolved(id2, "secret2")

	// Second call - should replace both secrets
	chunk2 := []byte("Value: secret1 and secret2")
	result2, err := provider.HandleChunk(chunk2)
	if err != nil {
		t.Fatalf("HandleChunk failed: %v", err)
	}
	if bytes.Contains(result2, []byte("secret1")) {
		t.Errorf("secret1 not replaced: %q", result2)
	}
	if bytes.Contains(result2, []byte("secret2")) {
		t.Errorf("secret2 not replaced: %q", result2)
	}
}

// TestVault_SecretProvider_MaxSecretLength tests MaxSecretLength method
func TestVault_SecretProvider_MaxSecretLength(t *testing.T) {
	v := NewWithPlanKey(testKey)

	provider := v.SecretProvider()

	// Initially no secrets
	if got := provider.MaxSecretLength(); got != 0 {
		t.Errorf("MaxSecretLength() = %d, want 0", got)
	}

	// Add short secret
	id1 := v.DeclareVariable("SHORT", "@env.SHORT")
	v.MarkResolved(id1, "short")

	// With variants, max length includes encoded forms (hex, base64, percent-encoding, separators)
	// "short" (5 bytes) -> percent-encoded "%73%68%6F%72%74" (15 bytes) is longest
	if got := provider.MaxSecretLength(); got != 15 {
		t.Errorf("MaxSecretLength() = %d, want 15 (includes percent-encoded variant)", got)
	}

	// Add longer secret
	id2 := v.DeclareVariable("LONG", "@env.LONG")
	v.MarkResolved(id2, "this_is_a_much_longer_secret")

	// "this_is_a_much_longer_secret" (28 bytes) -> percent-encoded (84 bytes) is longest
	if got := provider.MaxSecretLength(); got != 84 {
		t.Errorf("MaxSecretLength() = %d, want 84 (includes percent-encoded variant)", got)
	}
}
