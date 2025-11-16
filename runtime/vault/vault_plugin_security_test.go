package vault

import (
	"testing"
)

// ========== Decorator Plugin Security Tests ==========
//
// Decorator plugins are untrusted code (could be malicious or buggy).
// These tests validate plugins can only access secrets explicitly given to them.
//
// Security relies on HMAC-based SiteIDs that plugins cannot forge.

func TestPluginSecurity_CrossDecoratorAccess_Fails(t *testing.T) {
	// Different decorators get different site paths, preventing cross-access

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.MarkResolved(exprID, "secret-value")

	v.Push("step-1")
	v.Push("@var")
	v.RecordReference(exprID, "value")

	value, err := v.Access(exprID, "value")
	if err != nil {
		t.Fatalf("@var should access its own authorized site, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("@var Access() = %q, want %q", value, "secret-value")
	}

	// @env gets different site path than @var
	v.Pop()
	v.Push("@env")
	value, err = v.Access(exprID, "value")

	if err == nil {
		t.Fatal("@env should NOT access @var's secret (different site)")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

func TestPluginSecurity_MissingRecordReference_Fails(t *testing.T) {
	// Malicious plugins might skip RecordReference to bypass authorization

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.MarkResolved(exprID, "secret-value")

	v.Push("step-1")
	v.Push("@malicious")
	// Deliberately skip: v.RecordReference(exprID, "value")

	value, err := v.Access(exprID, "value")

	if err == nil {
		t.Fatal("Access() should fail without prior RecordReference()")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

func TestPluginSecurity_ParamNameMismatch_Fails(t *testing.T) {
	// ParamName is part of site path, so mismatches create different sites

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.MarkResolved(exprID, "secret-value")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// Different paramName creates different site path
	value, err := v.Access(exprID, "apiKey")

	if err == nil {
		t.Fatal("Access() should fail with different paramName (different site)")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

func TestPluginSecurity_PathStackManipulation_AffectsSite(t *testing.T) {
	// Nesting level affects site path, preventing nested decorators from
	// accessing parent decorator's secrets

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.MarkResolved(exprID, "secret-value")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	site1 := v.buildSitePathLocked("command")

	// Nested decorator gets different site path
	v.Push("@retry")
	v.RecordReference(exprID, "command")
	site2 := v.buildSitePathLocked("command")

	if site1 == site2 {
		t.Errorf("Different path stack should create different sites\n"+
			"  Site 1: %s\n"+
			"  Site 2: %s", site1, site2)
	}

	siteID1 := v.computeSiteID(site1)
	siteID2 := v.computeSiteID(site2)
	if siteID1 == siteID2 {
		t.Errorf("Different sites should have different SiteIDs\n"+
			"  SiteID 1: %s\n"+
			"  SiteID 2: %s", siteID1, siteID2)
	}

	// Both sites authorized independently
	v.Pop()
	value1, err1 := v.Access(exprID, "command")
	if err1 != nil {
		t.Errorf("Access at site 1 should succeed, got error: %v", err1)
	}
	if value1 != "secret-value" {
		t.Errorf("Access at site 1 = %q, want %q", value1, "secret-value")
	}

	// Reset to get same instance index for second @retry
	v.Pop() // Pop @shell
	v.Pop() // Pop step-1
	v.ResetCounts()
	v.Push("step-1")
	v.Push("@shell")
	v.Push("@retry")
	value2, err2 := v.Access(exprID, "command")
	if err2 != nil {
		t.Errorf("Access at site 2 should succeed, got error: %v", err2)
	}
	if value2 != "secret-value" {
		t.Errorf("Access at site 2 = %q, want %q", value2, "secret-value")
	}
}

func TestPluginSecurity_MultipleDecorators_IndependentAuthorization(t *testing.T) {
	// One decorator's authorization doesn't grant access to another decorator

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.MarkResolved(exprID, "secret-value")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	value, err := v.Access(exprID, "command")
	if err != nil {
		t.Fatalf("@shell should access its authorized site, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("@shell Access() = %q, want %q", value, "secret-value")
	}

	// @timeout not authorized yet
	v.Pop()
	v.Push("@timeout")
	value, err = v.Access(exprID, "duration")

	if err == nil {
		t.Fatal("@timeout should NOT access @shell's secret (not authorized)")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}

	// After authorization, @timeout can access
	v.RecordReference(exprID, "duration")
	value, err = v.Access(exprID, "duration")
	if err != nil {
		t.Errorf("@timeout should access after authorization, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("@timeout Access() = %q, want %q", value, "secret-value")
	}
}

// ========== Test 6: Same Decorator, Multiple Instances ==========

func TestPluginSecurity_SameDecorator_MultipleInstances_DifferentSites(t *testing.T) {
	// SCENARIO: Two instances of the same decorator (e.g., two @shell calls).
	// Each instance should get a different site (different instance index).
	//
	// EXPECTED: Different instances = different sites = independent authorization

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable resolved
	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.MarkResolved(exprID, "secret-value")

	// First @shell instance (index 0)
	v.Push("step-1")
	v.Push("@shell") // Instance 0
	v.RecordReference(exprID, "command")
	site1 := v.buildSitePathLocked("command")
	v.Pop()

	// Second @shell instance (index 1)
	v.Push("@shell") // Instance 1
	site2 := v.buildSitePathLocked("command")

	// THEN: Sites should be different (different instance index)
	if site1 == site2 {
		t.Errorf("Different @shell instances should have different sites\n"+
			"  Instance 0 site: %s\n"+
			"  Instance 1 site: %s", site1, site2)
	}

	// AND: First instance can access (authorized)
	// Reset to get instance 0 again
	v.Pop() // Pop @shell[1]
	v.Pop() // Pop step-1
	v.ResetCounts()
	v.Push("step-1")
	v.Push("@shell") // Instance 0
	value, err := v.Access(exprID, "command")
	if err != nil {
		t.Errorf("First @shell instance should access, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("First instance Access() = %q, want %q", value, "secret-value")
	}

	// AND: Second instance cannot access (not authorized)
	v.Pop()          // Pop @shell[0]
	v.Push("@shell") // Instance 1 (counter now at 1)
	value, err = v.Access(exprID, "command")
	if err == nil {
		t.Fatal("Second @shell instance should NOT access (not authorized)")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
}

// ========== Meta-Programming Security Tests ==========
//
// These tests validate that meta-programming constructs (@if, @for)
// work correctly with the SiteID access control system.
//
// Meta-programming is trusted core code (not plugins), but still needs
// authorization for audit trail and consistency.

// ========== Test 7: @if Condition Access ==========

func TestMetaProgramming_IfCondition_IndependentAuthorization(t *testing.T) {
	// SCENARIO: Variable used in @if condition. The @if construct can access
	// the variable, but a decorator in the @if body cannot (different sites).
	//
	// EXPECTED: @if can access, @shell in body cannot (unless separately authorized)

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable resolved (used in @if condition)
	exprID := v.DeclareVariable("ENV", "prod")
	v.MarkResolved(exprID, "prod")

	// AND: Planner evaluates @if condition
	v.Push("@if")
	v.RecordReference(exprID, "condition")

	// WHEN: @if accesses for condition evaluation
	value, err := v.Access(exprID, "condition")
	// THEN: Should succeed
	if err != nil {
		t.Fatalf("@if should access for condition evaluation, got error: %v", err)
	}
	if value != "prod" {
		t.Errorf("@if Access() = %q, want %q", value, "prod")
	}

	// WHEN: @shell in @if body tries to access (different site)
	v.Push("step-1") // Inside @if body
	v.Push("@shell")
	value, err = v.Access(exprID, "command")

	// THEN: Should fail - @shell not authorized (different site than @if)
	if err == nil {
		t.Fatal("@shell should NOT access @if's variable (different site)")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// ========== Test 8: Nested @if Access ==========

func TestMetaProgramming_NestedIf_DifferentSites(t *testing.T) {
	// SCENARIO: Nested @if constructs. Each @if instance should get a
	// different site (different nesting level).
	//
	// EXPECTED: Different @if nesting = different sites

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable resolved
	exprID := v.DeclareVariable("ENV", "prod")
	v.MarkResolved(exprID, "prod")

	// First @if (outer)
	v.Push("@if") // Instance 0
	v.RecordReference(exprID, "condition")
	site1 := v.buildSitePathLocked("condition")

	// Second @if (nested inside first)
	v.Push("@if") // Instance 1
	v.RecordReference(exprID, "condition")
	site2 := v.buildSitePathLocked("condition")

	// THEN: Sites should be different (different nesting)
	if site1 == site2 {
		t.Errorf("Nested @if should have different sites\n"+
			"  Outer @if site: %s\n"+
			"  Nested @if site: %s", site1, site2)
	}

	// AND: Both can access (both authorized)
	v.Pop() // Back to outer @if
	value1, err1 := v.Access(exprID, "condition")
	if err1 != nil {
		t.Errorf("Outer @if should access, got error: %v", err1)
	}
	if value1 != "prod" {
		t.Errorf("Outer @if Access() = %q, want %q", value1, "prod")
	}

	// Reset to get same instance index for nested @if
	v.Pop() // Pop @if[0]
	v.ResetCounts()
	v.Push("@if") // Instance 0 again
	v.Push("@if") // Instance 1 (nested)
	value2, err2 := v.Access(exprID, "condition")
	if err2 != nil {
		t.Errorf("Nested @if should access, got error: %v", err2)
	}
	if value2 != "prod" {
		t.Errorf("Nested @if Access() = %q, want %q", value2, "prod")
	}
}

// ========== Test 9: @for Loop Access ==========

func TestMetaProgramming_ForLoop_IndependentAuthorization(t *testing.T) {
	// SCENARIO: Variable used in @for loop items. The @for construct can
	// access the variable, but decorators in the loop body cannot.
	//
	// EXPECTED: @for can access, body decorators cannot (unless authorized)

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable resolved (used in @for items)
	exprID := v.DeclareVariable("ITEMS", "item1,item2,item3")
	v.MarkResolved(exprID, "item1,item2,item3")

	// AND: Planner evaluates @for items
	v.Push("@for")
	v.RecordReference(exprID, "items")

	// WHEN: @for accesses for items evaluation
	value, err := v.Access(exprID, "items")
	// THEN: Should succeed
	if err != nil {
		t.Fatalf("@for should access for items evaluation, got error: %v", err)
	}
	if value != "item1,item2,item3" {
		t.Errorf("@for Access() = %q, want %q", value, "item1,item2,item3")
	}

	// WHEN: @shell in @for body tries to access (different site)
	v.Push("step-1") // Inside @for body
	v.Push("@shell")
	value, err = v.Access(exprID, "command")

	// THEN: Should fail - @shell not authorized
	if err == nil {
		t.Fatal("@shell should NOT access @for's variable (different site)")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}
