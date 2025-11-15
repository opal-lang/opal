package vault

import (
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
	v.MarkResolved(exprID, "secret-value")

	// AND: Authorized site recorded
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Access at authorized site in same transport
	value, err := v.Access(exprID, "command")
	// THEN: Should succeed with correct value
	if err != nil {
		t.Errorf("Access() should succeed at authorized site, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("Access() = %q, want %q", value, "secret-value")
	}
}

// ========== Edge Case: Transport Boundary Violation ==========

func TestAccess_AuthorizedSite_DifferentTransport_FailsWithTransportError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: @env expression resolved in local transport
	exprID := v.DeclareVariable("LOCAL_TOKEN", "@env.TOKEN")
	v.MarkResolved(exprID, "secret-value")

	// AND: Authorized site recorded in local transport
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	// Site: root/step-1/@shell[0]/params/command

	// WHEN: Change transport but stay at same site path
	v.EnterTransport("ssh:untrusted")
	// Still at: root/step-1/@shell[0] (same path, different transport)

	value, err := v.Access(exprID, "command")

	// THEN: Should fail with transport boundary error
	if err == nil {
		t.Fatal("Access() should fail when crossing transport boundary")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
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
	v.MarkResolved(exprID, "secret-value")

	// AND: Authorized site: root/step-1/@shell[0]/params/command
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	v.Pop()

	// WHEN: Try to access at different site (different decorator)
	v.Push("@timeout")
	value, err := v.Access(exprID, "duration")

	// THEN: Should fail with authorization error
	if err == nil {
		t.Fatal("Access() should fail at unauthorized site")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
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
	v.MarkResolved(exprID, "secret-value")

	// AND: Authorized site in local transport
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	v.Pop()

	// WHEN: Enter different transport AND different site
	v.EnterTransport("ssh:remote")
	v.Push("step-1")
	v.Push("@timeout")

	value, err := v.Access(exprID, "duration")

	// THEN: Should fail (either error type is acceptable)
	if err == nil {
		t.Fatal("Access() should fail when both site and transport are wrong")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
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
	v.MarkResolved(exprID, "secret-value")

	// Site 1: root/step-1/@shell[0]/params/command
	v.Push("step-1")
	v.Push("@shell")
	site1 := v.BuildSitePath("command")
	v.RecordReference(exprID, "command")
	v.Pop() // Pop @shell
	v.Pop() // Pop step-1

	// Site 2: root/step-2/@retry[0]/params/apiKey
	v.Push("step-2")
	v.Push("@retry")
	site2 := v.BuildSitePath("apiKey")
	v.RecordReference(exprID, "apiKey")
	v.Pop() // Pop @retry
	v.Pop() // Pop step-2

	// Verify sites are different
	if site1 == site2 {
		t.Fatalf("Sites should be different: site1=%q, site2=%q", site1, site2)
	}

	// WHEN: Access at site 1 (need to reconstruct the exact path)
	// Reset to step 1 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.Push("step-1") // step-1
	v.Push("@shell")
	value1, err1 := v.Access(exprID, "command")

	// THEN: Should succeed at site 1
	if err1 != nil {
		t.Errorf("Access() should succeed at site 1, got: %v", err1)
	}
	if value1 != "secret-value" {
		t.Errorf("Access() at site 1 = %q, want %q", value1, "secret-value")
	}

	v.Pop()

	// WHEN: Access at site 2 (need to reconstruct the exact path)
	// Reset to step 2 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.Push("step-2")
	v.Push("@retry")
	value2, err2 := v.Access(exprID, "apiKey")

	// THEN: Should succeed at site 2
	if err2 != nil {
		t.Errorf("Access() should succeed at site 2, got: %v", err2)
	}
	if value2 != "secret-value" {
		t.Errorf("Access() at site 2 = %q, want %q", value2, "secret-value")
	}
}

// ========== Edge Case: Unresolved Expression ==========

func TestAccess_UnresolvedExpression_FailsWithResolvedError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression declared but not resolved (no Value set)
	exprID := v.DeclareVariable("UNRESOLVED", "@env.FOO")
	// Note: Value is empty string (not resolved)

	// AND: Authorized site recorded
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Try to access unresolved expression
	value, err := v.Access(exprID, "command")

	// THEN: Should fail with "not resolved" error
	if err == nil {
		t.Fatal("Access() should fail for unresolved expression")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
	}
	if !containsString(err.Error(), "not resolved") {
		t.Errorf("Error should mention 'not resolved', got: %v", err)
	}
}

// ========== Edge Case: Nonexistent Expression ==========

func TestAccess_NonexistentExpression_FailsWithNotFoundError(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// WHEN: Try to access expression that was never declared
	v.Push("step-1")
	v.Push("@shell")
	value, err := v.Access("NONEXISTENT", "command")

	// THEN: Should fail with "not found" error
	if err == nil {
		t.Fatal("Access() should fail for nonexistent expression")
	}
	if value != nil {
		t.Errorf("Access() should return nil on error, got %q", value)
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
	v.MarkResolved(localID, "/home/local")

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(localID, "command")
	v.Pop() // Pop @shell
	v.Pop() // Pop step-1

	// AND: @env.HOME in ssh:server1 transport (different expression!)
	v.EnterTransport("ssh:server1")
	sshID := v.TrackExpression("@env.HOME")
	v.MarkResolved(sshID, "/home/server1")

	v.ResetCounts() // Reset for new step
	v.Push("step-2")
	v.Push("@shell")
	v.RecordReference(sshID, "command")
	v.Pop() // Pop @shell
	v.Pop() // Pop step-2

	// Verify they are different expressions
	if localID == sshID {
		t.Fatalf("Same decorator in different transports should have different IDs: localID=%q, sshID=%q", localID, sshID)
	}

	// WHEN: Access local expression in local transport
	v.ExitTransport() // Back to local
	// Reset to step 1 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.Push("step-1") // step-1
	v.Push("@shell")
	localValue, localErr := v.Access(localID, "command")

	// THEN: Should get local value
	if localErr != nil {
		t.Errorf("Access() local should succeed, got: %v", localErr)
	}
	if localValue != "/home/local" {
		t.Errorf("Access() local = %q, want %q", localValue, "/home/local")
	}

	v.Pop() // Pop @shell
	v.Pop() // Pop step-1

	// WHEN: Access SSH expression in SSH transport
	v.EnterTransport("ssh:server1")
	// Reset to step 2 state
	v.pathStack = []PathSegment{{Name: "root", Index: -1}}
	v.decoratorCounts = make(map[string]int)
	v.Push("step-2")
	v.Push("@shell")
	sshValue, sshErr := v.Access(sshID, "command")

	// THEN: Should get SSH value
	if sshErr != nil {
		t.Errorf("Access() SSH should succeed, got: %v", sshErr)
	}
	if sshValue != "/home/server1" {
		t.Errorf("Access() SSH = %q, want %q", sshValue, "/home/server1")
	}
}

// ========== Edge Case: No Plan Key (Testing Mode) ==========

func TestAccess_NoPlanKey_SkipsSiteIDCheck(t *testing.T) {
	v := New() // No plan key

	// GIVEN: Expression without plan key
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.MarkResolved(exprID, "secret-value")

	// AND: Reference recorded (SiteID will be empty without plan key)
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Access without plan key
	value, err := v.Access(exprID, "command")
	// THEN: Should succeed (no SiteID enforcement in test mode)
	if err != nil {
		t.Errorf("Access() without plan key should succeed, got: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("Access() = %q, want %q", value, "secret-value")
	}
}

// ========== Future Feature: Decorator-Specific Transport Isolation ==========

func TestAccess_NonEnvDecorator_CrossesTransportBoundary(t *testing.T) {
	t.Skip("TODO: Implement decorator-level transport isolation capability")

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Non-@env expression (e.g., @var) resolved in local transport
	exprID := v.DeclareVariable("API_KEY", "sk-secret-123")
	v.MarkResolved(exprID, "sk-secret-123")

	// AND: Authorized site in local transport
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Enter different transport but stay at same site
	v.EnterTransport("ssh:remote")

	value, err := v.Access(exprID, "command")
	// THEN: Should succeed (non-@env can cross transport boundaries)
	// This will fail with current implementation - transport isolation is global
	// Need to implement decorator-level capability system
	if err != nil {
		t.Errorf("Access() for non-@env should succeed across transports, got: %v", err)
	}
	if value != "sk-secret-123" {
		t.Errorf("Access() = %q, want %q", value, "sk-secret-123")
	}
}

// ========== Edge Case: Empty String Secret ==========

func TestAccess_EmptyStringSecret_IsValid(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved to empty string (valid secret)
	exprID := v.DeclareVariable("EMPTY_VAR", "@env.EMPTY")
	v.MarkResolved(exprID, "")

	// AND: Authorized site recorded
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Access the empty string secret
	value, err := v.Access(exprID, "command")
	// THEN: Should succeed with empty string value
	if err != nil {
		t.Errorf("Access() should succeed for empty string secret, got error: %v", err)
	}
	if value != "" {
		t.Errorf("Access() = %v, want empty string", value)
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
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	v.MarkTouched(exprID)

	// WHEN: Build SecretUses
	uses := v.BuildSecretUses()

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
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")
	v.MarkTouched(exprID)

	// WHEN: Build SecretUses
	uses := v.BuildSecretUses()

	// THEN: Unresolved expression should be excluded
	if len(uses) != 0 {
		t.Errorf("Expected 0 SecretUses for unresolved expression, got %d", len(uses))
	}
}
