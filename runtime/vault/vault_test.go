package vault

import (
	"testing"
)

// ========== Path Tracking Tests ==========

// TestVault_PathTracking_SingleDecorator tests building a simple path
// with one decorator instance.
func TestVault_PathTracking_SingleDecorator(t *testing.T) {
	v := New()

	// GIVEN: We enter a step and a decorator
	v.EnterStep()
	v.EnterDecorator("@shell")

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
	v.EnterStep()
	v.EnterDecorator("@shell")
	path1 := v.BuildSitePath("command")
	v.ExitDecorator()

	v.EnterStep()
	v.EnterDecorator("@shell")
	path2 := v.BuildSitePath("command")
	v.ExitDecorator()

	v.EnterStep()
	v.EnterDecorator("@shell")
	path3 := v.BuildSitePath("command")
	v.ExitDecorator()

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
	v.EnterDecorator("@retry")
	v.EnterDecorator("@timeout")
	v.EnterDecorator("@shell")

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

	v.EnterStep()

	// First shell command
	v.EnterDecorator("@shell")
	path1 := v.BuildSitePath("command")
	v.ExitDecorator()

	// Second shell command
	v.EnterDecorator("@shell")
	path2 := v.BuildSitePath("command")
	v.ExitDecorator()

	// A retry decorator
	v.EnterDecorator("@retry")
	path3 := v.BuildSitePath("times")
	v.ExitDecorator()

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
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.ExitDecorator()
	v.EnterDecorator("@shell")
	path1 := v.BuildSitePath("command")
	v.ExitDecorator()

	// Step 2: One shell command (should be [0] again)
	v.EnterStep()
	v.EnterDecorator("@shell")
	path2 := v.BuildSitePath("command")
	v.ExitDecorator()

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
	v.EnterStep()
	v.EnterDecorator("@shell")
	err := v.RecordReference(exprID, "command")
	if err != nil {
		t.Fatalf("RecordReference should succeed in same transport: %v", err)
	}

	// WHEN: We enter SSH transport and try to access it
	v.EnterTransport("ssh:untrusted")
	v.EnterStep()
	v.EnterDecorator("@shell")
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
	v.EnterStep()
	v.EnterDecorator("@shell")
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
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")
	v.ExitDecorator()

	v.EnterStep()
	v.EnterDecorator("@shell")
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

	v.EnterStep()
	v.EnterDecorator("@shell")
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

	v.EnterStep()
	v.EnterDecorator("@shell")
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

// TestVault_DeclareVariable_ReturnsVariableName tests that DeclareVariable returns the variable name as ID.
func TestVault_DeclareVariable_ReturnsVariableName(t *testing.T) {
	v := New()

	// WHEN: We declare a variable
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// THEN: Should return variable name as ID
	if exprID != "API_KEY" {
		t.Errorf("DeclareVariable() = %q, want %q", exprID, "API_KEY")
	}
}

// TestVault_DeclareVariable_StoresExpression tests that the expression is stored.
func TestVault_DeclareVariable_StoresExpression(t *testing.T) {
	v := New()

	// WHEN: We declare a variable
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// THEN: Expression should be stored (check via internal state)
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
	v.EnterStep()
	v.EnterDecorator("@shell")
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
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")
	v.ExitDecorator()

	// WHEN: We try to access at different site (not authorized)
	v.EnterStep()
	v.EnterDecorator("@timeout")
	value, err := v.Access(exprID, "duration")

	// THEN: Should fail with authorization error
	if err == nil {
		t.Error("Access() at unauthorized site should fail")
	}
	if value != "" {
		t.Errorf("Access() should return empty string on error, got %q", value)
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
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: We try to access
	value, err := v.Access(exprID, "command")

	// THEN: Should fail (not resolved yet)
	if err == nil {
		t.Error("Access() on unresolved expression should fail")
	}
	if value != "" {
		t.Errorf("Access() should return empty string on error, got %q", value)
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

	v.EnterStep()
	v.EnterDecorator("@shell")
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
	v.EnterStep()
	v.EnterDecorator("@shell")
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

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
