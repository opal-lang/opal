package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/runtime/streamscrub"
)

// Vault manages variable scoping and secret tracking with site-based access control.
//
// # Architecture
//
// Vault uses pathStack as a scope trie where each level stores variables.
// Variable lookup walks up the trie (current → parent → root) enabling parent → child flow.
// Expression IDs are hash-based (content + transport) to support:
//   - Variable shadowing (same name, different values in different scopes)
//   - Transport-sensitive expressions (@env.HOME differs per SSH session)
//   - Expression deduplication (multiple variables can share same expression)
//
// # Security Model
//
// ALL value decorators produce secrets. Vault enforces:
//  1. Transport boundaries - secrets cannot cross transport boundaries
//  2. Site authorization - secrets only accessible at authorized sites (HMAC-based SiteID)
//
// # Usage
//
//	// Pass 1: Declare variables and track references
//	vault := vault.NewWithPlanKey(planKey)
//	exprID := vault.DeclareVariable("API_KEY", "@env.API_KEY")  // Returns hash-based ID
//	vault.RecordReference(exprID, "command")
//
//	// Pass 2: Store value and resolve
//	vault.MarkTouched(exprID)
//	vault.StoreUnresolvedValue(exprID, "ghp_abc123")
//	vault.ResolveAllTouched()
//
//	// Pass 3: Access during execution
//	value, _ := vault.AccessByDisplayID("opal:abc123", "command")
//
//	// Pass 4: Finalize
//	vault.PruneUntouched()
//	uses := vault.BuildSecretUses()
//
// # Variable Lookup
//
//	// Declare at root
//	rootID := vault.DeclareVariable("COUNT", "5")
//
//	// Enter child scope
//	vault.EnterDecorator("@retry")
//
//	// Lookup walks up trie
//	foundID, _ := vault.LookupVariable("COUNT")  // Finds rootID from parent scope
//
// # Rules
//
//  1. Call ResolveAllTouched when resolving (captures transport context)
//  2. Call MarkTouched for expressions in execution path
//  3. Call PruneUntouched before BuildSecretUses
//  4. Do not copy Vault after first use
//
// See docs/ARCHITECTURE.md for complete architecture.
type Vault struct {
	mu sync.RWMutex // Protects all fields below (RWMutex for better read performance)

	// Path tracking (DAG traversal)
	pathStack       []PathSegment
	stepCount       int
	decoratorCounts map[string]int // Decorator instance counts at current level

	// Expression tracking
	expressions    map[string]*Expression // exprID → Expression
	displayIDIndex map[string]string      // DisplayID → exprID (reverse lookup for execution)
	references     map[string][]SiteRef   // exprID → sites that use it
	touched        map[string]bool        // exprID → in execution path

	// Scope-aware variable storage (pathStack IS the trie)
	scopes map[string]*VaultScope // scopePath → scope

	// Transport boundary tracking
	currentTransport string            // Current transport scope
	exprTransport    map[string]string // exprID → transport where resolved

	// Security
	planKey []byte // For HMAC-based SiteIDs

	// Secret scrubbing (lazy initialization)
	provider streamscrub.SecretProvider
}

// Expression represents a secret-producing expression.
// In our security model: ALL expressions are secrets.
type Expression struct {
	Raw       string // Original source: "@var.X", "@aws.secret('key')", etc.
	Value     any    // Resolved value (preserves original type: string, int, bool, map, slice)
	DisplayID string // Placeholder ID for plan (e.g., "opal:3J98t56A")
	Resolved  bool   // True if expression has been resolved (even if Value is nil)
}

// Note: No ExprType, no IsSecret - everything is a secret.
// Vault stores raw values directly - access control via SiteID + transport checks.

// VaultScope represents a scope in the variable trie.
// Each scope corresponds to a level in the pathStack.
// Variables declared at a scope are stored in that scope.
// Lookup walks up the trie (current → parent → grandparent → root).
type VaultScope struct {
	path   string            // "root/step-1/@retry[0]"
	parent string            // Parent scope path (empty for root)
	vars   map[string]string // varName → exprID
}

// SiteRef represents a reference to an expression at a specific site.
type SiteRef struct {
	Site      string // "root/step-1/@shell[0]/params/command"
	SiteID    string // HMAC-based unforgeable ID
	ParamName string // "command", "apiKey", etc.
}

// PathSegment represents one level in the scope/site path.
// Generic representation - Vault doesn't know what the name means.
// The caller (planner) decides: "root", "step-1", "@retry", etc.
type PathSegment struct {
	Name  string // Scope name: "root", "step-1", "@retry", etc.
	Index int    // Instance index (-1 if not applicable)
}

// New creates a new Vault.
func New() *Vault {
	v := &Vault{
		pathStack:        []PathSegment{{Name: "root", Index: -1}},
		stepCount:        0,
		decoratorCounts:  make(map[string]int),
		expressions:      make(map[string]*Expression),
		displayIDIndex:   make(map[string]string),
		references:       make(map[string][]SiteRef),
		touched:          make(map[string]bool),
		scopes:           make(map[string]*VaultScope),
		currentTransport: "local",
		exprTransport:    make(map[string]string),
	}

	// Initialize root scope
	v.scopes["root"] = &VaultScope{
		path:   "root",
		parent: "",
		vars:   make(map[string]string),
	}

	return v
}

// NewWithPlanKey creates a new Vault with a specific plan key for HMAC-based SiteIDs.
func NewWithPlanKey(planKey []byte) *Vault {
	v := New()
	v.planKey = planKey
	return v
}

// Push adds a segment to the path stack.
// The caller (planner) decides what the segment represents: "step-1", "@retry", etc.
// Returns the index for this segment name at the current level.
func (v *Vault) Push(name string) int {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Get next instance index for this name at current level
	index := v.decoratorCounts[name]
	v.decoratorCounts[name]++

	v.pathStack = append(v.pathStack, PathSegment{
		Name:  name,
		Index: index,
	})

	return index
}

// Pop removes the top segment from the path stack.
// Panics if attempting to pop root (programmer error).
func (v *Vault) Pop() {
	v.mu.Lock()
	defer v.mu.Unlock()

	invariant.Precondition(len(v.pathStack) > 1, "cannot pop root from path stack")
	v.pathStack = v.pathStack[:len(v.pathStack)-1]
}

// ResetCounts resets the decorator instance counters.
// Used when entering a new step to reset decorator indices to 0.
// The caller (planner) decides when to reset - typically when starting a new step.
func (v *Vault) ResetCounts() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.decoratorCounts = make(map[string]int)
}

// BuildSitePath constructs the canonical site path for a parameter.
// Format: root/step-N/@decorator[index]/params/paramName
// Thread-safe: Acquires read lock.
func (v *Vault) BuildSitePath(paramName string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.buildSitePathLocked(paramName)
}

// buildSitePathLocked is the internal unlocked version of BuildSitePath.
// Caller must hold at least a read lock.
func (v *Vault) buildSitePathLocked(paramName string) string {
	var parts []string

	for _, seg := range v.pathStack {
		// Decorators (starting with @) include instance index
		if strings.HasPrefix(seg.Name, "@") {
			parts = append(parts, fmt.Sprintf("%s[%d]", seg.Name, seg.Index))
		} else {
			// Non-decorators (root, step-N) are just the name
			parts = append(parts, seg.Name)
		}
	}

	// Add parameter path
	parts = append(parts, "params", paramName)

	return strings.Join(parts, "/")
}

// ========== Scope Management ==========

// currentScopePath converts pathStack to a scope path string.
// Used for site paths (authorization). Includes all segments.
func (v *Vault) currentScopePath() string {
	var parts []string
	for _, seg := range v.pathStack {
		// Decorators (starting with @) include instance index
		if strings.HasPrefix(seg.Name, "@") {
			parts = append(parts, fmt.Sprintf("%s[%d]", seg.Name, seg.Index))
		} else {
			// Non-decorators (root, step-N) are just the name
			parts = append(parts, seg.Name)
		}
	}
	return strings.Join(parts, "/")
}

// currentVariableScopePath converts pathStack to a variable scope path.
// Variable scopes exclude step segments (steps are not scopes).
// Only root and decorator blocks create variable scopes.
func (v *Vault) currentVariableScopePath() string {
	var parts []string
	for _, seg := range v.pathStack {
		// Skip step segments (step-N) - they're for site paths, not variable scoping
		if strings.HasPrefix(seg.Name, "step-") {
			continue
		}

		// Include decorators with instance index
		if strings.HasPrefix(seg.Name, "@") {
			parts = append(parts, fmt.Sprintf("%s[%d]", seg.Name, seg.Index))
		} else {
			// Include root
			parts = append(parts, seg.Name)
		}
	}

	// If no parts (only step segments), return root
	if len(parts) == 0 {
		return "root"
	}

	return strings.Join(parts, "/")
}

// getOrCreateScope ensures a scope exists at the given path.
// Creates parent link to enable trie walk during variable lookup.
func (v *Vault) getOrCreateScope(scopePath string) *VaultScope {
	if scope, exists := v.scopes[scopePath]; exists {
		return scope
	}

	parentPath := v.parentScopePath(scopePath)

	scope := &VaultScope{
		path:   scopePath,
		parent: parentPath,
		vars:   make(map[string]string),
	}
	v.scopes[scopePath] = scope
	return scope
}

// parentScopePath computes the parent scope path for trie traversal.
// Returns empty string for root since it has no parent.
func (v *Vault) parentScopePath(scopePath string) string {
	lastSlash := strings.LastIndex(scopePath, "/")
	if lastSlash == -1 {
		return ""
	}
	return scopePath[:lastSlash]
}

// LookupVariable resolves a variable name to its expression ID.
// Walks up the scope trie from current scope to root, enabling parent → child flow.
// Handles missing scopes by computing parent path directly (scopes created lazily).
func (v *Vault) LookupVariable(varName string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	scopePath := v.currentScopePath()
	visited := make(map[string]bool)

	for scopePath != "" {
		// Safety: Detect cycles in scope trie (visiting same scope twice)
		invariant.Invariant(!visited[scopePath], "cycle detected in scope trie at %q", scopePath)
		visited[scopePath] = true

		scope := v.scopes[scopePath]
		if scope != nil {
			if exprID, exists := scope.vars[varName]; exists {
				return exprID, nil
			}
			scopePath = scope.parent
		} else {
			// Scope not created yet, compute parent directly
			scopePath = v.parentScopePath(scopePath)
		}
	}

	return "", fmt.Errorf("variable %q not found in any scope", varName)
}

// ========== Expression Tracking ==========

// DeclareVariable registers a variable in the current variable scope.
// Variable scope excludes step segments (steps are not scopes, only decorator blocks are).
// Uses hash-based exprID (not variable name) to support:
// - Same variable name with different values in different scopes (shadowing)
// - Same expression shared by multiple variables (deduplication)
// - Transport-sensitive expressions (@env.HOME differs per SSH session)
func (v *Vault) DeclareVariable(name, raw string) string {
	return v.declareVariableAt(name, raw, v.currentVariableScopePath())
}

// declareVariableAt is the internal implementation for declaring variables at a specific scope.
func (v *Vault) declareVariableAt(name, raw, scopePath string) string {
	v.mu.Lock()
	defer v.mu.Unlock()

	exprID := v.generateExprID(raw)

	if _, exists := v.expressions[exprID]; !exists {
		// DisplayID will be generated in ResolveAllTouched() when we have the actual value
		// This ensures DisplayID = HMAC(planKey, value) for unlinkability
		v.expressions[exprID] = &Expression{
			Raw:       raw,
			DisplayID: "", // Empty until resolved
		}
	}

	scope := v.getOrCreateScope(scopePath)
	scope.vars[name] = exprID

	return exprID
}

// TrackExpression registers a direct decorator call (e.g., @env.HOME).
// Returns a deterministic hash-based ID that includes transport context.
// Format: "transport:hash"
func (v *Vault) TrackExpression(raw string) string {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Generate deterministic ID including transport
	exprID := v.generateExprID(raw)

	// Store expression if not already tracked
	if _, exists := v.expressions[exprID]; !exists {
		// DisplayID will be generated in ResolveAllTouched() when we have the actual value
		// This ensures DisplayID = HMAC(planKey, value) for unlinkability
		v.expressions[exprID] = &Expression{
			Raw:       raw,
			DisplayID: "", // Empty until resolved
		}
	}

	return exprID
}

// generateExprID creates a deterministic expression ID including transport context.
func (v *Vault) generateExprID(raw string) string {
	// Include current transport for context-sensitive IDs
	h := sha256.New()
	h.Write([]byte(v.currentTransport))
	h.Write([]byte(":"))
	h.Write([]byte(raw))
	hash := h.Sum(nil)

	// Format: "transport:hash"
	// Use first 8 bytes of hash for reasonable ID length
	hashStr := fmt.Sprintf("%x", hash[:8])
	return fmt.Sprintf("%s:%s", v.currentTransport, hashStr)
}

// RecordReference records that an expression is used at the current site.
// Transport boundary check is deferred to Access() time (after resolution).
func (v *Vault) RecordReference(exprID, paramName string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	site := v.buildSitePathLocked(paramName)
	siteID := v.computeSiteID(site)

	v.references[exprID] = append(v.references[exprID], SiteRef{
		Site:      site,
		SiteID:    siteID,
		ParamName: paramName,
	})

	return nil
}

// computeSiteID generates an unforgeable site identifier using HMAC.
func (v *Vault) computeSiteID(canonicalPath string) string {
	if len(v.planKey) == 0 {
		// No plan key set - return empty (tests without security)
		return ""
	}

	h := hmac.New(sha256.New, v.planKey)
	h.Write([]byte(canonicalPath))
	mac := h.Sum(nil)

	// Truncate to 16 bytes and base64 encode
	return base64.RawURLEncoding.EncodeToString(mac[:16])
}

// computeDisplayID generates a DisplayID from a resolved value using HMAC.
// DisplayID = HMAC(planKey, value) ensures unlinkability across plans.
// Same secret in different plans → different DisplayIDs (prevents correlation).
// Same secret in same plan → same DisplayID (enables contract verification).
//
// Maps have non-deterministic iteration order in Go, so JSON marshaling
// provides canonical representation with sorted keys.
func (v *Vault) computeDisplayID(value any) string {
	var canonical []byte

	switch v := value.(type) {
	case string:
		canonical = []byte(v)
	case []byte:
		// Must match getPatterns() representation for scrubbing to work
		canonical = v
	default:
		// JSON marshaling sorts map keys for determinism
		var err error
		canonical, err = json.Marshal(value)
		invariant.Invariant(err == nil, "computeDisplayID: failed to marshal value to JSON: %v", err)
	}

	if len(v.planKey) == 0 {
		// Backward compatibility for tests that don't set planKey
		h := sha256.New()
		h.Write(canonical)
		hash := h.Sum(nil)
		return base64.RawURLEncoding.EncodeToString(hash[:16])
	}

	h := hmac.New(sha256.New, v.planKey)
	h.Write(canonical)
	mac := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(mac[:16])
}

// PruneUnused removes expressions that have no site references.
// This eliminates variables that were declared but never used.
func (v *Vault) PruneUnused() {
	for id := range v.expressions {
		if len(v.references[id]) == 0 {
			delete(v.expressions, id)
			delete(v.references, id)
			delete(v.touched, id)
			delete(v.exprTransport, id)
		}
	}
}

// BuildSecretUses constructs the final SecretUse list for the plan.
// Auto-prunes: Only includes expressions that:
// 1. Have been resolved (Resolved flag is true) - unresolved are skipped
// 2. Have at least one site reference - unreferenced are skipped
// 3. Are marked as touched - untouched are skipped
//
// In our security model: ALL value decorators are secrets.
// Note: Empty string values are valid secrets (e.g., empty env vars).
//
// Returns a deterministically sorted slice (by DisplayID, then Site) to ensure
// stable contract hashes across runs. Map iteration order is non-deterministic.
func (v *Vault) BuildSecretUses() []SecretUse {
	v.mu.Lock()
	defer v.mu.Unlock()

	var uses []SecretUse

	for id, expr := range v.expressions {
		// Auto-prune: Skip unresolved expressions (check Resolved flag, not Value)
		if !expr.Resolved {
			continue
		}

		// Auto-prune: Skip expressions with no references (unused)
		refs := v.references[id]
		if len(refs) == 0 {
			continue
		}

		// Auto-prune: Skip untouched expressions (not in execution path)
		if !v.touched[id] {
			continue
		}

		// Build SecretUse for each reference site
		for _, ref := range refs {
			uses = append(uses, SecretUse{
				DisplayID: expr.DisplayID,
				SiteID:    ref.SiteID,
				Site:      ref.Site,
			})
		}
	}

	// Sort for deterministic contract hashing
	// Primary: DisplayID (same secret used at multiple sites)
	// Secondary: Site (deterministic order for same DisplayID)
	sort.Slice(uses, func(i, j int) bool {
		if uses[i].DisplayID != uses[j].DisplayID {
			return uses[i].DisplayID < uses[j].DisplayID
		}
		return uses[i].Site < uses[j].Site
	})

	return uses
}

// SecretUse represents an authorized secret usage at a specific site.
// This is what gets added to the Plan for executor enforcement.
type SecretUse struct {
	DisplayID string // "opal:3J98t56A"
	SiteID    string // HMAC-based unforgeable ID
	Site      string // "root/step-1/@shell[0]/params/command" (diagnostic)
}

// MarkTouched marks an expression as touched (in execution path).
func (v *Vault) MarkTouched(exprID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.touched[exprID] = true
}

// IsTouched checks if an expression is marked as touched.
func (v *Vault) IsTouched(exprID string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.touched[exprID]
}

// PruneUntouched removes expressions not in execution path.
func (v *Vault) PruneUntouched() {
	v.mu.Lock()
	defer v.mu.Unlock()

	for id := range v.expressions {
		if !v.touched[id] {
			delete(v.expressions, id)
			delete(v.references, id)
			delete(v.touched, id)
			delete(v.exprTransport, id)
		}
	}
}

// EnterTransport enters a new transport scope.
func (v *Vault) EnterTransport(scope string) {
	v.currentTransport = scope
}

// ExitTransport exits current transport scope (returns to local).
func (v *Vault) ExitTransport() {
	v.currentTransport = "local"
}

// CurrentTransport returns the current transport scope.
func (v *Vault) CurrentTransport() string {
	return v.currentTransport
}

// ResolveAllTouched marks all touched expressions as resolved and generates DisplayIDs.
// Enables batching efficiency: decorators can batch multiple API calls (e.g., @aws.secret).
func (v *Vault) ResolveAllTouched() {
	v.mu.Lock()
	defer v.mu.Unlock()

	for exprID := range v.touched {
		expr, exists := v.expressions[exprID]
		if !exists {
			continue
		}

		if expr.Resolved {
			continue
		}

		invariant.Invariant(expr.Value != nil,
			"ResolveAllTouched: expression %q is touched but has no value stored", exprID)

		// Mark as resolved and capture transport context
		expr.Resolved = true
		v.exprTransport[exprID] = v.currentTransport

		// Generate DisplayID from value using HMAC for unlinkability
		hash := v.computeDisplayID(expr.Value)
		expr.DisplayID = fmt.Sprintf("opal:%s", hash)

		// Build reverse index for execution (DisplayID → exprID lookup)
		v.displayIDIndex[expr.DisplayID] = exprID
	}
}

// StoreUnresolvedValue stores a parsed value without marking it resolved.
// Enables deferred resolution for batching efficiency (multiple @aws.secret calls batched into one API request).
// No-op if value already stored (expression deduplication).
func (v *Vault) StoreUnresolvedValue(exprID string, value any) {
	v.mu.Lock()
	defer v.mu.Unlock()

	expr, exists := v.expressions[exprID]
	invariant.Precondition(exists, "StoreUnresolvedValue: expression %q not found", exprID)

	if expr.Value != nil {
		return
	}

	expr.Value = value
}

// GetDisplayID returns the placeholder ID for an expression.
// Safe to call because it returns only the DisplayID, not the actual secret value.
func (v *Vault) GetDisplayID(exprID string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	expr, exists := v.expressions[exprID]
	if !exists || !expr.Resolved {
		return ""
	}
	return expr.DisplayID
}

// IsResolved checks if an expression has been resolved.
// Safe to call - returns only resolution status, not the actual value.
func (v *Vault) IsResolved(exprID string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	expr, exists := v.expressions[exprID]
	return exists && expr.Resolved
}

// GetPlanKey returns the plan key used for HMAC-based DisplayID generation.
// This should be stored in plan.PlanSalt for contract verification.
// Returns a copy to prevent external modification.
func (v *Vault) GetPlanKey() []byte {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.planKey) == 0 {
		return nil
	}

	// Return a copy to prevent external modification
	keyCopy := make([]byte, len(v.planKey))
	copy(keyCopy, v.planKey)
	return keyCopy
}

// checkTransportBoundary checks if expression can be used in current transport.
func (v *Vault) checkTransportBoundary(exprID string) error {
	// Get transport where expression was resolved
	exprTransport, exists := v.exprTransport[exprID]

	// CRITICAL: This should NEVER happen in production!
	// If it does, it means ResolveAllTouched() wasn't called (programmer error).
	invariant.Invariant(exists,
		"expression %q has no transport recorded (ResolveAllTouched not called?)",
		exprID)

	// Check if crossing transport boundary (legitimate security check - return error)
	if exprTransport != v.currentTransport {
		return fmt.Errorf(
			"transport boundary violation: expression %q resolved in %q, cannot use in %q",
			exprID, exprTransport, v.currentTransport,
		)
	}

	return nil
}

// Access returns the raw value for an expression at the current site.
//
// Implements Zanzibar-style access control:
//   - Tuple (Position): Checks if (exprID, siteID) is authorized
//   - Caveat (Constraint): Checks transport boundary (if decorator requires it)
//
// Used by planner for meta-programming (e.g., @if conditionals, @for loops).
//
// Parameters:
//   - exprID: Expression identifier (from DeclareVariable or TrackExpression)
//   - paramName: Parameter name accessing the value (e.g., "command", "apiKey")
//
// Returns:
//   - Resolved value (preserves original type: string, int, bool, map, slice) if both checks pass
//   - Error if expression not found, not resolved, unauthorized site, or transport violation
//
// Example:
//
//	vault.EnterDecorator("@shell")
//	value, err := vault.Access("API_KEY", "command")  // Checks site: root/@shell[0]/params/command
func (v *Vault) Access(exprID, paramName string) (any, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 0. Security: Require planKey for authorization checks
	// Without planKey, all sites have SiteID="" which bypasses authorization
	invariant.Precondition(len(v.planKey) > 0,
		"Access() requires planKey for security - use NewWithPlanKey() instead of New()")

	// 1. Get expression
	expr, exists := v.expressions[exprID]
	if !exists {
		return nil, fmt.Errorf("expression %q not found", exprID)
	}
	if !expr.Resolved {
		return nil, fmt.Errorf("expression %q not resolved yet", exprID)
	}

	// 2. Check transport boundary (Caveat - checked first as more fundamental)
	if err := v.checkTransportBoundary(exprID); err != nil {
		return nil, err
	}

	// 3. Build current site with parameter name
	currentSite := v.buildSitePathLocked(paramName)
	currentSiteID := v.computeSiteID(currentSite)

	// 4. Check if current site is authorized (Tuple)
	authorized := false
	for _, ref := range v.references[exprID] {
		if ref.SiteID == currentSiteID {
			authorized = true
			break
		}
	}
	if !authorized {
		return nil, fmt.Errorf("no authority to unwrap %q at site %q", exprID, currentSite)
	}

	// 5. Return value (preserves original type)
	return expr.Value, nil
}

// AccessByDisplayID resolves a DisplayID to its actual value.
// This is used during execution to resolve DisplayID placeholders in commands.
// The DisplayID is looked up in the reverse index to find the exprID,
// then Access() is called with the exprID for full authorization checks.
func (v *Vault) AccessByDisplayID(displayID, paramName string) (any, error) {
	v.mu.RLock()
	exprID, found := v.displayIDIndex[displayID]
	v.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("DisplayID %q not found in vault", displayID)
	}

	// Use existing Access method for full authorization checks
	return v.Access(exprID, paramName)
}

// ============================================================================
// SecretProvider Implementation (for streamscrub integration)
// ============================================================================

// getPatterns returns all resolved expressions as scrubbing patterns.
// This is called by the pattern provider on each HandleChunk invocation.
// Converts values to strings for pattern matching (scrubbing only needs string representation).
func (v *Vault) getPatterns() []streamscrub.Pattern {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var patterns []streamscrub.Pattern

	for _, expr := range v.expressions {
		if !expr.Resolved {
			continue
		}

		// Must match computeDisplayID() representation for scrubbing to work
		var valueBytes []byte
		switch v := expr.Value.(type) {
		case string:
			valueBytes = []byte(v)
		case []byte:
			valueBytes = v
		case nil:
			continue
		default:
			valueStr := fmt.Sprintf("%v", v)
			if valueStr == "" || valueStr == "<nil>" {
				continue
			}
			valueBytes = []byte(valueStr)
		}

		if len(valueBytes) == 0 {
			continue
		}

		patterns = append(patterns, streamscrub.Pattern{
			Value:       valueBytes,
			Placeholder: []byte(expr.DisplayID),
		})
	}

	return patterns
}

// SecretProvider returns a streamscrub.SecretProvider for this vault.
// The provider replaces all resolved expression values with their DisplayIDs.
//
// This enables automatic secret scrubbing in output streams without manual
// registration. The scrubber calls the provider to process each chunk.
//
// The provider is lazily initialized on first call and reused.
// Thread-safe: Safe for concurrent calls.
func (v *Vault) SecretProvider() streamscrub.SecretProvider {
	// Fast path: check with read lock first
	v.mu.RLock()
	if v.provider != nil {
		defer v.mu.RUnlock()
		return v.provider
	}
	v.mu.RUnlock()

	// Slow path: initialize with write lock
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have initialized)
	if v.provider == nil {
		v.provider = streamscrub.NewPatternProviderWithVariants(v.getPatterns)
	}

	return v.provider
}
