package vault

import (
	"fmt"
	"testing"
)

// ========== Access Control: Zanzibar-Style Tests ==========
// Tuple (Position): (exprID, siteID) - "expression X can be accessed at site Y"
// Caveat (Constraint): Transport restriction - "only if in same transport"
//
// These tests verify the two-rule access control model:
// 1. Site-based authority (HMAC-based SiteID prevents forgery)
// 2. Transport isolation (decorator-specific, currently @env only)

// ========== Happy Path: Authorized Access ==========

func TestAccess_AuthorizedSite_SameTransport_Succeeds(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved in local transport
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	// AND: Authorized site recorded
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	// WHEN: Access at authorized site in same transport
	value, err := v.access(exprID, "command")
	// THEN: Should succeed with correct value
	if err != nil {
		t.Errorf("access() should succeed at authorized site, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("access() = %q, want %q", value, "secret-value")
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

// ========== Edge Case: Transport Boundary Violation ==========

func TestAccess_AuthorizedSite_DifferentTransport_FailsWithTransportError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: @env expression resolved in local transport (transport-sensitive)
	exprID := v.DeclareVariableTransportSensitive("LOCAL_TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	// AND: Authorized site recorded in local transport
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")
	// Site: root/step-1/@shell[0]/params/command

	// WHEN: Change transport but stay at same site path
	v.EnterTransport("ssh:untrusted")
	// Still at: root/step-1/@shell[0] (same path, different transport)

	value, err := v.access(exprID, "command")

	// THEN: Should fail with transport boundary error
	if err == nil {
		t.Fatal("access() should fail when crossing transport boundary")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "transport") {
		t.Errorf("Error should mention 'transport', got: %v", err)
	}
}

// ========== Edge Case: Unauthorized Site ==========

func TestAccess_UnauthorizedSite_SameTransport_FailsWithAuthorityError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression with one authorized site
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	// AND: Authorized site: root/step-1/@shell[0]/params/command
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")
	v.pop()

	// WHEN: Try to access at different site (different decorator)
	v.push("@timeout")
	value, err := v.access(exprID, "duration")

	// THEN: Should fail with authorization error
	if err == nil {
		t.Fatal("access() should fail at unauthorized site")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// ========== Edge Case: Both Violations ==========

func TestAccess_UnauthorizedSite_DifferentTransport_Fails(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved in local transport
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	// AND: Authorized site in local transport
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")
	v.pop()

	// WHEN: Enter different transport AND different site
	v.EnterTransport("ssh:remote")
	v.push("step-1")
	v.push("@timeout")

	value, err := v.access(exprID, "duration")

	// THEN: Should fail (either error type is acceptable)
	if err == nil {
		t.Fatal("access() should fail when both site and transport are wrong")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	// Accept either transport or authority error
	hasTransportError := containsString(err.Error(), "transport")
	hasAuthorityError := containsString(err.Error(), "no authority")
	if !hasTransportError && !hasAuthorityError {
		t.Errorf("Error should mention 'transport' or 'no authority', got: %v", err)
	}
}

// ========== Edge Case: Multiple Authorized Sites ==========

func TestAccess_MultipleSites_EachSiteIndependent(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression authorized at two different sites
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	// Site 1: root/step-1/@shell[0]/params/command
	v.push("step-1")
	v.push("@shell")
	site1 := v.buildSitePath("command")
	v.recordReference(exprID, "command")
	v.pop() // Pop @shell
	v.pop() // Pop step-1

	// Site 2: root/step-2/@retry[0]/params/apiKey
	v.push("step-2")
	v.push("@retry")
	site2 := v.buildSitePath("apiKey")
	v.recordReference(exprID, "apiKey")
	v.pop() // Pop @retry
	v.pop() // Pop step-2

	// Verify sites are different
	if site1 == site2 {
		t.Fatalf("Sites should be different: site1=%q, site2=%q", site1, site2)
	}

	// WHEN: Access at site 1 (need to reconstruct the exact path)
	// Reset to step 1 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.push("step-1") // step-1
	v.push("@shell")
	value1, err1 := v.access(exprID, "command")

	// THEN: Should succeed at site 1
	if err1 != nil {
		t.Errorf("access() should succeed at site 1, got: %v", err1)
	}
	if value1 != "secret-value" {
		t.Errorf("access() at site 1 = %q, want %q", value1, "secret-value")
	}

	v.pop()

	// WHEN: Access at site 2 (need to reconstruct the exact path)
	// Reset to step 2 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.push("step-2")
	v.push("@retry")
	value2, err2 := v.access(exprID, "apiKey")

	// THEN: Should succeed at site 2
	if err2 != nil {
		t.Errorf("access() should succeed at site 2, got: %v", err2)
	}
	if value2 != "secret-value" {
		t.Errorf("access() at site 2 = %q, want %q", value2, "secret-value")
	}
}

// ========== Edge Case: Unresolved Expression ==========

func TestAccess_UnresolvedExpression_FailsWithResolvedError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression declared but not resolved (no Value set)
	exprID := v.DeclareVariable("UNRESOLVED", "@env.FOO")
	// Note: Value is not set (not resolved)

	// AND: Authorized site recorded
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	// WHEN: Try to access unresolved expression
	value, err := v.access(exprID, "command")

	// THEN: Should fail with "not resolved" error
	if err == nil {
		t.Fatal("access() should fail for unresolved expression")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "not resolved") {
		t.Errorf("Error should mention 'not resolved', got: %v", err)
	}
}

// ========== Edge Case: Nonexistent Expression ==========

func TestAccess_NonexistentExpression_FailsWithNotFoundError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// WHEN: Try to access expression that was never declared
	v.push("step-1")
	v.push("@shell")
	value, err := v.access("NONEXISTENT", "command")

	// THEN: Should fail with "not found" error
	if err == nil {
		t.Fatal("access() should fail for nonexistent expression")
	}
	if value != nil {
		t.Errorf("access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// ========== Edge Case: Same Decorator, Different Transports ==========

func TestAccess_SameDecorator_DifferentTransports_IndependentExpressions(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: @env.HOME in local transport
	localID := v.TrackExpression("@env.HOME")
	v.MarkTouched(localID)
	v.StoreUnresolvedValue(localID, "/home/local")
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@shell")
	v.recordReference(localID, "command")
	v.pop() // Pop @shell
	v.pop() // Pop step-1

	// AND: @env.HOME in ssh:server1 transport (different expression!)
	v.EnterTransport("ssh:server1")
	sshID := v.TrackExpression("@env.HOME")
	v.MarkTouched(sshID)
	v.StoreUnresolvedValue(sshID, "/home/server1")
	v.ResolveAllTouched()

	v.resetCounts() // Reset for new step
	v.push("step-2")
	v.push("@shell")
	v.recordReference(sshID, "command")
	v.pop() // Pop @shell
	v.pop() // Pop step-2

	// Verify they are different expressions
	if localID == sshID {
		t.Fatalf("Same decorator in different transports should have different IDs: localID=%q, sshID=%q", localID, sshID)
	}

	// WHEN: Access local expression in local transport
	v.exitTransport() // Back to local
	// Reset to step 1 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.push("step-1") // step-1
	v.push("@shell")
	localValue, localErr := v.access(localID, "command")

	// THEN: Should get local value
	if localErr != nil {
		t.Errorf("access() local should succeed, got: %v", localErr)
	}
	if localValue != "/home/local" {
		t.Errorf("access() local = %q, want %q", localValue, "/home/local")
	}

	v.pop() // Pop @shell
	v.pop() // Pop step-1

	// WHEN: Access SSH expression in SSH transport
	v.EnterTransport("ssh:server1")
	// Reset to step 2 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.push("step-2")
	v.push("@shell")
	sshValue, sshErr := v.access(sshID, "command")

	// THEN: Should get SSH value
	if sshErr != nil {
		t.Errorf("access() SSH should succeed, got: %v", sshErr)
	}
	if sshValue != "/home/server1" {
		t.Errorf("access() SSH = %q, want %q", sshValue, "/home/server1")
	}
}

// ========== Security Requirement: Plan Key Required ==========

func TestAccess_NoPlanKey_PanicsForSecurity(t *testing.T) {
	// Without planKey, all sites have SiteID="" which bypasses authorization.
	// access() enforces planKey requirement to prevent this security vulnerability.

	v := newVault() // No plan key - DANGEROUS!

	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	// access() should panic to prevent security bypass
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("%v", r)
			if !containsString(panicMsg, "planKey") {
				t.Errorf("Panic should mention planKey, got: %v", r)
			}
			t.Logf("✓ Security enforced: access() requires planKey")
		} else {
			t.Error("🚨 SECURITY ISSUE: access() should panic without planKey")
			t.Error("  Without planKey, all sites have SiteID=\"\" which bypasses authorization")
			t.Error("  Production code MUST use NewWithPlanKey()")
		}
	}()

	v.access(exprID, "command") // Should panic
}

// ========== Transport-Agnostic Expressions Can Cross Boundaries ==========

func TestAccess_TransportAgnostic_CrossesTransportBoundary(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Transport-agnostic expression (e.g., @var) resolved in local transport
	// Note: DeclareVariable() defaults to TransportSensitive: false
	exprID := v.DeclareVariable("API_KEY", "@var.API_KEY")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "sk-secret-123")
	v.ResolveAllTouched()

	// AND: Authorized site in local transport
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	// WHEN: Enter different transport but stay at same site
	v.EnterTransport("ssh:remote")

	value, err := v.access(exprID, "command")
	// THEN: Should succeed (transport-agnostic expressions can cross boundaries)
	if err != nil {
		t.Errorf("access() for transport-agnostic expression should succeed across transports, got: %v", err)
	}
	if value != "sk-secret-123" {
		t.Errorf("access() = %q, want %q", value, "sk-secret-123")
	}
}

// ========== Edge Case: Empty String Secret ==========

func TestAccess_EmptyStringSecret_IsValid(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved to empty string (valid secret)
	exprID := v.DeclareVariable("EMPTY_VAR", "@env.EMPTY")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "")
	v.ResolveAllTouched()

	// AND: Authorized site recorded
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")

	// WHEN: Access the empty string secret
	value, err := v.access(exprID, "command")
	// THEN: Should succeed with empty string value
	if err != nil {
		t.Errorf("access() should succeed for empty string secret, got error: %v", err)
	}
	if value != "" {
		t.Errorf("access() = %v, want empty string", value)
	}
}

func TestBuildSecretUses_EmptyStringSecret_IsIncluded(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved to empty string
	exprID := v.DeclareVariable("EMPTY_VAR", "@env.EMPTY")
	v.expressions[exprID].Value = ""
	v.expressions[exprID].Resolved = true
	v.expressions[exprID].DisplayID = "opal:EMPTY"

	// AND: Has reference and is touched
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")
	v.MarkTouched(exprID)

	// WHEN: Build SecretUses
	uses := v.buildSecretUses()

	// THEN: Empty string secret should be included
	if len(uses) != 1 {
		t.Fatalf("Expected 1 SecretUse for empty string secret, got %d", len(uses))
	}
	if uses[0].DisplayID != "opal:EMPTY" {
		t.Errorf("SecretUse.DisplayID = %q, want %q", uses[0].DisplayID, "opal:EMPTY")
	}
}

func TestBuildSecretUses_UnresolvedExpression_IsExcluded(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression NOT resolved (Resolved = false)
	exprID := v.DeclareVariable("UNRESOLVED", "@env.FOO")
	v.expressions[exprID].Value = "some-value"
	v.expressions[exprID].Resolved = false // Explicitly not resolved
	v.expressions[exprID].DisplayID = "opal:UNRES"

	// AND: Has reference and is touched
	v.push("step-1")
	v.push("@shell")
	v.recordReference(exprID, "command")
	v.MarkTouched(exprID)

	// WHEN: Build SecretUses
	uses := v.buildSecretUses()

	// THEN: Unresolved expression should be excluded
	if len(uses) != 0 {
		t.Errorf("Expected 0 SecretUses for unresolved expression, got %d", len(uses))
	}
}

// ========== ResolveDisplayIDWithTransport: Simplified Execution-Time Resolution ==========
// This method is for the new planner architecture where:
// - Decorators receive resolved values, never see Vault
// - Contract verification via plan hash handles integrity
// - Transport boundary is the only runtime check needed (no site authorization)

func TestResolveDisplayIDWithTransport_SameTransport_Succeeds(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Transport-sensitive expression resolved in local transport
	exprID := v.DeclareVariableTransportSensitive("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	displayID := v.GetDisplayID(exprID)

	// WHEN: Resolve in same transport
	value, err := v.ResolveDisplayIDWithTransport(displayID, "local")
	// THEN: Should succeed
	if err != nil {
		t.Errorf("ResolveDisplayIDWithTransport() should succeed in same transport, got: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("ResolveDisplayIDWithTransport() = %q, want %q", value, "secret-value")
	}
}

func TestResolveDisplayIDWithTransport_DifferentTransport_Fails(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Transport-sensitive expression resolved in local transport
	exprID := v.DeclareVariableTransportSensitive("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	displayID := v.GetDisplayID(exprID)

	// WHEN: Resolve in different transport
	_, err := v.ResolveDisplayIDWithTransport(displayID, "ssh://remote-host")

	// THEN: Should fail with transport boundary error
	if err == nil {
		t.Error("ResolveDisplayIDWithTransport() should fail for different transport")
	}
	if !containsString(err.Error(), "transport boundary") {
		t.Errorf("Error should mention transport boundary, got: %v", err)
	}
}

func TestResolveDisplayIDWithTransport_TransportAgnostic_CrossesBoundary(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Transport-agnostic expression (e.g., @var)
	exprID := v.DeclareVariable("API_KEY", "@var.API_KEY") // Not transport-sensitive
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "sk-secret-123")
	v.ResolveAllTouched()

	displayID := v.GetDisplayID(exprID)

	// WHEN: Resolve in different transport
	value, err := v.ResolveDisplayIDWithTransport(displayID, "ssh://remote-host")
	// THEN: Should succeed (transport-agnostic can cross boundaries)
	if err != nil {
		t.Errorf("ResolveDisplayIDWithTransport() should succeed for transport-agnostic, got: %v", err)
	}
	if value != "sk-secret-123" {
		t.Errorf("ResolveDisplayIDWithTransport() = %q, want %q", value, "sk-secret-123")
	}
}

func TestResolveDisplayIDWithTransport_UnknownDisplayID_Fails(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// WHEN: Resolve unknown DisplayID
	_, err := v.ResolveDisplayIDWithTransport("opal:unknown1234567890ab", "local")

	// THEN: Should fail with not found error
	if err == nil {
		t.Error("ResolveDisplayIDWithTransport() should fail for unknown DisplayID")
	}
	if !containsString(err.Error(), "not found") {
		t.Errorf("Error should mention not found, got: %v", err)
	}
}

func TestResolveDisplayIDWithTransport_UnresolvedExpression_Fails(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression tracked but not resolved
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.StoreUnresolvedValue(exprID, "secret-value")
	// Note: NOT calling ResolveAllTouched() - expression is unresolved

	// Manually set DisplayID to simulate partial state
	v.mu.Lock()
	v.expressions[exprID].DisplayID = "opal:test1234567890abcd"
	v.displayIDIndex["opal:test1234567890abcd"] = exprID
	v.mu.Unlock()

	// WHEN: Try to resolve
	_, err := v.ResolveDisplayIDWithTransport("opal:test1234567890abcd", "local")

	// THEN: Should fail because not resolved
	if err == nil {
		t.Error("ResolveDisplayIDWithTransport() should fail for unresolved expression")
	}
	if !containsString(err.Error(), "not resolved") {
		t.Errorf("Error should mention not resolved, got: %v", err)
	}
}

func TestResolveDisplayIDWithTransport_NoSiteAuthRequired(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved but NO site reference recorded
	// (This would fail with access() but should succeed with ResolveDisplayIDWithTransport)
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkTouched(exprID)
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.ResolveAllTouched()

	displayID := v.GetDisplayID(exprID)

	// Note: NOT calling recordReference() - no site authorization

	// WHEN: Resolve with new method (no site auth)
	value, err := v.ResolveDisplayIDWithTransport(displayID, "local")
	// THEN: Should succeed (no site auth required)
	if err != nil {
		t.Errorf("ResolveDisplayIDWithTransport() should not require site auth, got: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("ResolveDisplayIDWithTransport() = %q, want %q", value, "secret-value")
	}
}
