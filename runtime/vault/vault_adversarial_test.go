package vault

import (
	"fmt"
	"sync"
	"testing"
)

// ========== Red Team / Adversarial Security Tests ==========
//
// These tests simulate malicious actors trying to bypass the SiteID access control system.
// Each test represents a specific attack vector identified during security analysis.

// ========== Attack Vector 4: Empty planKey Bypass ==========

func TestAdversarial_EmptyPlanKey_PanicsOnAccess(t *testing.T) {
	// Empty planKey would bypass authorization, but Access() now panics to prevent this
	// MITIGATION: Added invariant in Access() to fail fast if planKey is empty

	v := New() // No plan key!

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Authorize at one site
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// Try to access - should panic due to empty planKey
	v.Pop()
	v.Push("@malicious")

	defer func() {
		if r := recover(); r != nil {
			// Expected panic
			panicMsg := fmt.Sprintf("%v", r)
			if !containsString(panicMsg, "planKey") {
				t.Errorf("Panic should mention planKey, got: %v", r)
			}
			t.Logf("✓ SECURITY MITIGATION WORKING: Access() panics without planKey")
			t.Logf("  Panic message: %v", r)
		} else {
			t.Error("🚨 Access() should panic without planKey (security mitigation missing)")
		}
	}()

	v.Access(exprID, "apiKey") // Should panic
}

func TestAdversarial_EmptyPlanKey_SiteIDIsEmpty(t *testing.T) {
	// Verify that empty planKey produces empty SiteID

	v := New() // No plan key

	v.Push("step-1")
	v.Push("@shell")

	siteID := v.computeSiteID("root/step-1/@shell[0]/params/command")

	if siteID != "" {
		t.Errorf("Empty planKey should produce empty SiteID, got: %q", siteID)
	}
}

func TestAdversarial_WithPlanKey_SiteIDIsNotEmpty(t *testing.T) {
	// Verify that planKey produces non-empty SiteID

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	v.Push("step-1")
	v.Push("@shell")

	siteID := v.computeSiteID("root/step-1/@shell[0]/params/command")

	if siteID == "" {
		t.Error("Non-empty planKey should produce non-empty SiteID")
	}
	if len(siteID) == 0 {
		t.Error("SiteID should have non-zero length")
	}
}

// ========== Attack Vector 6: Instance Counter Manipulation ==========

func TestAdversarial_ResetCounts_DoesNotAllowCrossDecoratorAccess(t *testing.T) {
	// Attacker tries to match instance index of authorized decorator by resetting counter

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Authorize @shell[0]
	v.Push("step-1")
	v.Push("@shell") // Instance 0
	v.RecordReference(exprID, "command")
	authorizedSite := v.buildSitePathLocked("command")
	authorizedSiteID := v.computeSiteID(authorizedSite)

	// Attacker resets counter to get instance 0 again
	v.Pop() // Pop @shell
	v.Pop() // Pop step-1
	v.ResetCounts()
	v.Push("step-1")
	v.Push("@malicious") // Also instance 0 (same index as @shell!)
	attackSite := v.buildSitePathLocked("command")
	attackSiteID := v.computeSiteID(attackSite)

	// Sites should be different (decorator name is part of site path)
	if authorizedSite == attackSite {
		t.Errorf("Different decorators should have different sites even with same instance index\n"+
			"  Authorized: %s\n"+
			"  Attack:     %s", authorizedSite, attackSite)
	}

	// SiteIDs should be different
	if authorizedSiteID == attackSiteID {
		t.Errorf("Different decorators should have different SiteIDs\n"+
			"  Authorized SiteID: %s\n"+
			"  Attack SiteID:     %s", authorizedSiteID, attackSiteID)
	}

	// Access should fail
	value, err := v.Access(exprID, "command")
	if err == nil {
		t.Errorf("⚠️  SECURITY ISSUE: ResetCounts allowed cross-decorator access!")
		t.Errorf("  Got value: %q (should have been denied)", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// ========== Attack Vector 5: ParamName Confusion ==========

func TestAdversarial_ParamName_IsPartOfSitePath(t *testing.T) {
	// Verify that different paramNames create different site paths

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	v.Push("step-1")
	v.Push("@shell")

	site1 := v.buildSitePathLocked("command")
	site2 := v.buildSitePathLocked("apiKey")

	if site1 == site2 {
		t.Errorf("Different paramNames should create different sites\n"+
			"  Param 'command': %s\n"+
			"  Param 'apiKey':  %s", site1, site2)
	}

	// Should differ only in the last segment
	expectedSite1 := "root/step-1/@shell[0]/params/command"
	expectedSite2 := "root/step-1/@shell[0]/params/apiKey"

	if site1 != expectedSite1 {
		t.Errorf("Site for 'command' = %q, want %q", site1, expectedSite1)
	}
	if site2 != expectedSite2 {
		t.Errorf("Site for 'apiKey' = %q, want %q", site2, expectedSite2)
	}
}

func TestAdversarial_ParamName_EnforcedInAccess(t *testing.T) {
	// Attacker authorizes with one paramName, tries to access with another

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command") // Authorize "command"

	// Try to access with different paramName
	value, err := v.Access(exprID, "apiKey") // Different param!

	if err == nil {
		t.Errorf("⚠️  SECURITY ISSUE: ParamName not enforced in Access()!")
		t.Errorf("  Authorized param: command")
		t.Errorf("  Access param:     apiKey")
		t.Errorf("  Got value: %q (should have been denied)", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// ========== Attack Vector 1: Race Condition on pathStack Manipulation ==========

func TestAdversarial_ConcurrentAccess_ThreadSafe(t *testing.T) {
	// Multiple goroutines try to access the SAME authorized site concurrently
	// This tests that the mutex properly protects concurrent reads

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Authorize a specific site
	v.Push("step-1")
	v.Push("@shell") // Instance 0
	v.RecordReference(exprID, "command")
	authorizedSite := v.BuildSitePath("command")
	v.Pop()
	v.Pop()

	// Reset decorator counts so first goroutine gets instance 0
	v.ResetCounts()

	// Concurrent access from multiple goroutines to the SAME site
	var wg sync.WaitGroup
	successes := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// All goroutines access the same authorized site
			v.Push("step-1")
			v.Push("@shell") // Will be instance 0 for first goroutine, then 1, 2, etc.
			// This is actually testing that DIFFERENT instances are properly isolated!
			_, err := v.Access(exprID, "command")
			v.Pop()
			v.Pop()

			successes <- (err == nil)
		}()
	}

	wg.Wait()
	close(successes)

	// Count successes
	successCount := 0
	for success := range successes {
		if success {
			successCount++
		}
	}

	// Only the first goroutine should succeed (instance 0)
	// All others get different instance indices and should fail
	t.Logf("Concurrent access: %d/50 succeeded", successCount)
	t.Logf("Authorized site: %s", authorizedSite)

	if successCount != 1 {
		t.Fatalf("Expected exactly 1 success (instance 0), got %d", successCount)
	}
}

func TestAdversarial_PathStackManipulation_BetweenRecordAndAccess(t *testing.T) {
	// Attacker manipulates pathStack between RecordReference and Access

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command") // Authorize @shell site

	// Attacker manipulates stack
	v.Pop()                                   // Remove @shell
	v.Push("@malicious")                      // Add different decorator
	value, err := v.Access(exprID, "command") // Try to access at malicious site

	if err == nil {
		t.Errorf("⚠️  SECURITY ISSUE: pathStack manipulation allowed unauthorized access!")
		t.Errorf("  Got value: %q (should have been denied)", value)
	}
	if !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// ========== Attack Vector 8: Reference List Pollution ==========

func TestAdversarial_ReferenceListPollution_PerformanceDegradation(t *testing.T) {
	// Attacker adds many references to slow down Access() checks

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Add 1000 references (pollution attack)
	v.Push("step-1")
	for i := 0; i < 1000; i++ {
		v.Push(fmt.Sprintf("@attack-%d", i))
		v.RecordReference(exprID, "command")
		v.Pop()
	}

	// Add legitimate reference
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// Access should still work (but might be slow)
	value, err := v.Access(exprID, "command")
	if err != nil {
		t.Errorf("Access failed with polluted reference list: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("Access() = %q, want %q", value, "secret-value")
	}

	// Check reference count
	refCount := len(v.references[exprID])
	if refCount != 1001 {
		t.Errorf("Reference count = %d, want 1001", refCount)
	}

	t.Logf("✓ Access still works with %d references (linear scan)", refCount)
	t.Logf("  Note: This could be a DoS vector if reference count is unbounded")
}

// ========== Attack Vector 7: Transport Boundary Bypass ==========

func TestAdversarial_MarkResolved_CapturesCurrentTransport(t *testing.T) {
	// Verify that MarkResolved captures transport at call time, not resolution time

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// Resolve in local transport
	v.EnterTransport("local")
	exprID := v.DeclareVariable("SECRET", "secret-value")

	// Attacker changes transport before MarkResolved
	v.EnterTransport("ssh:server")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched() // Should capture "ssh:server"

	// Check which transport was captured
	capturedTransport := v.exprTransport[exprID]
	if capturedTransport != "ssh:server" {
		t.Errorf("MarkResolved should capture current transport\n"+
			"  Expected: ssh:server\n"+
			"  Got:      %s", capturedTransport)
	}

	// This is actually CORRECT behavior - MarkResolved captures current transport
	// The CALLER must ensure they call MarkResolved in the correct transport
	t.Logf("✓ MarkResolved correctly captures transport at call time: %s", capturedTransport)
	t.Logf("  Note: Caller must call MarkResolved immediately after resolution")
}

func TestAdversarial_TransportBoundary_EnforcedInAccess(t *testing.T) {
	// Verify that Access enforces transport boundaries

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// Resolve in local transport
	v.EnterTransport("local")
	exprID := v.DeclareVariable("SECRET", "secret-value")
	v.StoreUnresolvedValue(exprID, "secret-value")
	v.MarkTouched(exprID)
	v.ResolveAllTouched()

	// Authorize in local transport
	v.Push("step-1")
	v.Push("@shell")
	v.RecordReference(exprID, "command")

	// Try to access from different transport
	v.EnterTransport("ssh:server")
	value, err := v.Access(exprID, "command")

	if err == nil {
		t.Errorf("⚠️  SECURITY ISSUE: Transport boundary not enforced!")
		t.Errorf("  Resolved in: local")
		t.Errorf("  Accessed in: ssh:server")
		t.Errorf("  Got value: %q (should have been denied)", value)
	}
	if !containsString(err.Error(), "transport boundary violation") {
		t.Errorf("Error should mention 'transport boundary violation', got: %v", err)
	}
}

// ========== Attack Vector 9: DisplayID Prediction ==========

func TestAdversarial_DisplayID_RequiresPlanKey(t *testing.T) {
	// Verify that DisplayID cannot be predicted without planKey

	planKey := []byte("test-key-32-bytes-long!!!!!!")
	v1 := NewWithPlanKey(planKey)
	v2 := NewWithPlanKey(planKey)

	// Same value, same planKey → same DisplayID (deterministic)
	exprID1 := v1.DeclareVariable("SECRET", "secret-value")
	v1.StoreUnresolvedValue(exprID1, "secret-value")
	v1.MarkTouched(exprID1)
	v1.ResolveAllTouched()
	displayID1 := v1.GetDisplayID(exprID1)

	exprID2 := v2.DeclareVariable("SECRET", "secret-value")
	v2.StoreUnresolvedValue(exprID2, "secret-value")
	v2.MarkTouched(exprID2)
	v2.ResolveAllTouched()
	displayID2 := v2.GetDisplayID(exprID2)

	if displayID1 != displayID2 {
		t.Errorf("Same planKey + same value should produce same DisplayID\n"+
			"  DisplayID 1: %s\n"+
			"  DisplayID 2: %s", displayID1, displayID2)
	}

	// Different planKey → different DisplayID (unpredictable without key)
	v3 := NewWithPlanKey([]byte("different-key-32-bytes-long!"))
	exprID3 := v3.DeclareVariable("SECRET", "secret-value")
	v3.StoreUnresolvedValue(exprID3, "secret-value")
	v3.MarkTouched(exprID3)
	v3.ResolveAllTouched()
	displayID3 := v3.GetDisplayID(exprID3)

	if displayID1 == displayID3 {
		t.Errorf("⚠️  SECURITY ISSUE: Different planKeys produce same DisplayID!")
		t.Errorf("  This means DisplayID is predictable without knowing planKey")
	}

	t.Logf("✓ DisplayID requires planKey to predict")
	t.Logf("  Same key:      %s", displayID1)
	t.Logf("  Different key: %s", displayID3)
}

// ========== Attack Vector: SiteID Uniqueness ==========

func TestAdversarial_SiteID_UniquePerSite(t *testing.T) {
	// Verify that different sites produce different SiteIDs

	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	sites := []string{
		"root/step-1/@shell[0]/params/command",
		"root/step-1/@shell[1]/params/command",
		"root/step-1/@shell[0]/params/apiKey",
		"root/step-2/@shell[0]/params/command",
		"root/step-1/@retry[0]/params/command",
	}

	siteIDs := make(map[string]string)
	for _, site := range sites {
		siteID := v.computeSiteID(site)
		if siteID == "" {
			t.Errorf("SiteID should not be empty for site: %s", site)
		}
		siteIDs[site] = siteID
	}

	// Check for collisions
	seen := make(map[string]string)
	for site, siteID := range siteIDs {
		if otherSite, exists := seen[siteID]; exists {
			t.Errorf("⚠️  SECURITY ISSUE: SiteID collision detected!\n"+
				"  Site 1: %s\n"+
				"  Site 2: %s\n"+
				"  SiteID: %s", otherSite, site, siteID)
		}
		seen[siteID] = site
	}

	t.Logf("✓ All %d sites have unique SiteIDs", len(sites))
}
