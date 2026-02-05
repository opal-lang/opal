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
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@var")
	v.recordReference(exprID, "value")

	value, err := v.access(exprID, "value")
	if err != nil {
		t.Fatalf("@var should access its own authorized site, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("@var access() = %q, want %q", value, "secret-value")
	}

	// @env gets different site path than @var
	v.pop()
	v.push("@env")
	value, err = v.access(exprID, "value")

	if err == nil {
		t.Fatal("@env should NOT access @var's secret (different site)")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

func TestPluginSecurity_MissingRecordReference_Fails(t *testing.T) {
	// Malicious plugins might skip RecordReference to bypass authorization

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@malicious")
	// Deliberately skip: v.recordReference(exprID, "value")

	value, err := v.access(exprID, "value")

	if err == nil {
		t.Fatal("access() should fail without prior recordReference()")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

func TestPluginSecurity_ParamNameMismatch_Fails(t *testing.T) {
	// ParamName is part of site path, so mismatches create different sites

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	// Different paramName creates different site path
	value, err := v.access(exprID, "apiKey")

	if err == nil {
		t.Fatal("access() should fail with different paramName (different site)")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
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
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")
	site1 := v.buildSitePathLocked("command")

	// Nested decorator gets different site path
	v.push("@retry")
	v.recordReference(exprID, "command")
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
	v.pop()
	value1, err1 := v.access(exprID, "command")
	if err1 != nil {
		t.Errorf("Access at site 1 should succeed, got error: %v", err1)
	}
	if value1 != "secret-value" {
		t.Errorf("Access at site 1 = %q, want %q", value1, "secret-value")
	}

	// Reset to get same instance index for second @retry
	v.pop() // Pop @shell
	v.pop() // Pop step-1
	v.resetCounts()
	v.push("step-1")
	v.push("@shell")
	v.push("@retry")
	value2, err2 := v.access(exprID, "command")
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
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	value, err := v.access(exprID, "command")
	if err != nil {
		t.Fatalf("@shell should access its authorized site, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("@shell access() = %q, want %q", value, "secret-value")
	}

	// @timeout not authorized yet
	v.pop()
	v.push("@timeout")
	value, err = v.access(exprID, "duration")

	if err == nil {
		t.Fatal("@timeout should NOT access @shell's secret (not authorized)")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}

	// After authorization, @timeout can access
	v.recordReference(exprID, "duration")
	value, err = v.access(exprID, "duration")
	if err != nil {
		t.Errorf("@timeout should access after authorization, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("@timeout access() = %q, want %q", value, "secret-value")
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
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// First @shell instance (index 0)
	v.push("step-1")
	v.push("@shell") // Instance 0
	v.recordReference(exprID, "command")
	site1 := v.buildSitePathLocked("command")
	v.pop()

	// Second @shell instance (index 1)
	v.push("@shell") // Instance 1
	site2 := v.buildSitePathLocked("command")

	// THEN: Sites should be different (different instance index)
	if site1 == site2 {
		t.Errorf("Different @shell instances should have different sites\n"+
			"  Instance 0 site: %s\n"+
			"  Instance 1 site: %s", site1, site2)
	}

	// AND: First instance can access (authorized)
	// Reset to get instance 0 again
	v.pop() // Pop @shell[1]
	v.pop() // Pop step-1
	v.resetCounts()
	v.push("step-1")
	v.push("@shell") // Instance 0
	value, err := v.access(exprID, "command")
	if err != nil {
		t.Errorf("First @shell instance should access, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("First instance access() = %q, want %q", value, "secret-value")
	}

	// AND: Second instance cannot access (not authorized)
	v.pop()          // Pop @shell[0]
	v.push("@shell") // Instance 1 (counter now at 1)
	value, err = v.access(exprID, "command")
	if err == nil {
		t.Fatal("Second @shell instance should NOT access (not authorized)")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
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
	v.StoreUnresolvedValue(exprID, "prod")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// AND: Planner evaluates @if condition
	v.push("@if")
	v.recordReference(exprID, "condition")

	// WHEN: @if accesses for condition evaluation
	value, err := v.access(exprID, "condition")
	// THEN: Should succeed
	if err != nil {
		t.Fatalf("@if should access for condition evaluation, got error: %v", err)
	}
	if value != "prod" {
		t.Errorf("@if access() = %q, want %q", value, "prod")
	}

	// WHEN: @shell in @if body tries to access (different site)
	v.push("step-1") // Inside @if body
	v.push("@shell")
	value, err = v.access(exprID, "command")

	// THEN: Should fail - @shell not authorized (different site than @if)
	if err == nil {
		t.Fatal("@shell should NOT access @if's variable (different site)")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
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
	v.StoreUnresolvedValue(exprID, "prod")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// First @if (outer)
	v.push("@if") // Instance 0
	v.recordReference(exprID, "condition")
	site1 := v.buildSitePathLocked("condition")

	// Second @if (nested inside first)
	v.push("@if") // Instance 1
	v.recordReference(exprID, "condition")
	site2 := v.buildSitePathLocked("condition")

	// THEN: Sites should be different (different nesting)
	if site1 == site2 {
		t.Errorf("Nested @if should have different sites\n"+
			"  Outer @if site: %s\n"+
			"  Nested @if site: %s", site1, site2)
	}

	// AND: Both can access (both authorized)
	v.pop() // Back to outer @if
	value1, err1 := v.access(exprID, "condition")
	if err1 != nil {
		t.Errorf("Outer @if should access, got error: %v", err1)
	}
	if value1 != "prod" {
		t.Errorf("Outer @if access() = %q, want %q", value1, "prod")
	}

	// Reset to get same instance index for nested @if
	v.pop() // Pop @if[0]
	v.resetCounts()
	v.push("@if") // Instance 0 again
	v.push("@if") // Instance 1 (nested)
	value2, err2 := v.access(exprID, "condition")
	if err2 != nil {
		t.Errorf("Nested @if should access, got error: %v", err2)
	}
	if value2 != "prod" {
		t.Errorf("Nested @if access() = %q, want %q", value2, "prod")
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
	v.StoreUnresolvedValue(exprID, "item1,item2,item3")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// AND: Planner evaluates @for items
	v.push("@for")
	v.recordReference(exprID, "items")

	// WHEN: @for accesses for items evaluation
	value, err := v.access(exprID, "items")
	// THEN: Should succeed
	if err != nil {
		t.Fatalf("@for should access for items evaluation, got error: %v", err)
	}
	if value != "item1,item2,item3" {
		t.Errorf("@for access() = %q, want %q", value, "item1,item2,item3")
	}

	// WHEN: @shell in @for body tries to access (different site)
	v.push("step-1") // Inside @for body
	v.push("@shell")
	value, err = v.access(exprID, "command")

	// THEN: Should fail - @shell not authorized
	if err == nil {
		t.Fatal("@shell should NOT access @for's variable (different site)")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}
