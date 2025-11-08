package planner

import (
	"fmt"
	"strings"
)

// VarClass categorizes variables by sensitivity level.
type VarClass int

const (
	VarClassData   VarClass = iota // Regular data
	VarClassConfig                 // Configuration (non-secret)
	VarClassSecret                 // Sensitive credential
)

// VarTaint tracks whether a variable can cross transport boundaries.
type VarTaint int

const (
	VarTaintAgnostic         VarTaint = iota // Can cross any boundary
	VarTaintLocalOnly                        // Cannot leave local scope
	VarTaintBoundaryImported                 // Crossed via @import
)

// VarEntry stores a variable with its security metadata.
type VarEntry struct {
	Value  any
	Origin string   // "@env.HOME", "literal", etc.
	Class  VarClass // Data, Config, Secret
	Taint  VarTaint // Agnostic, LocalOnly, BoundaryImported
}

// ScopeGraph manages hierarchical variable scoping across sessions.
// Variables are resolved by traversing up the parent chain.
// Transport boundaries require explicit imports for security.
type ScopeGraph struct {
	root    *Scope
	current *Scope
}

// Scope represents a variable scope tied to a session context.
type Scope struct {
	// Identity
	id        string // Unique scope ID
	sessionID string // Session identifier from Session.ID()

	// Variables
	vars     map[string]VarEntry
	imported map[string]bool // Explicitly imported from parent

	// Graph structure
	parent   *Scope
	children []*Scope

	// Security
	sealedFromParent bool // Transport boundary guard
	transportDepth   int  // 0=local, 1=ssh, 2=ssh→docker

	// Metadata
	depth int      // Distance from root
	path  []string // Path from root (for debugging)
}

// NewScopeGraph creates a new scope graph with a root scope.
func NewScopeGraph(rootSessionID string) *ScopeGraph {
	root := &Scope{
		id:               "root",
		sessionID:        rootSessionID,
		vars:             make(map[string]VarEntry),
		imported:         make(map[string]bool),
		parent:           nil,
		children:         nil,
		sealedFromParent: false,
		transportDepth:   0,
		depth:            0,
		path:             []string{rootSessionID},
	}

	return &ScopeGraph{
		root:    root,
		current: root,
	}
}

// EnterScope creates a new child scope and makes it current.
// Called when entering a new session (e.g., @ssh block).
// If the session changes transport, the scope is sealed from parent.
func (g *ScopeGraph) EnterScope(sessionID string, isTransportBoundary bool) {
	newDepth := g.current.transportDepth
	if isTransportBoundary {
		newDepth++
	}

	child := &Scope{
		id:               fmt.Sprintf("%s.%d", sessionID, len(g.current.children)),
		sessionID:        sessionID,
		vars:             make(map[string]VarEntry),
		imported:         make(map[string]bool),
		parent:           g.current,
		children:         nil,
		sealedFromParent: isTransportBoundary,
		transportDepth:   newDepth,
		depth:            g.current.depth + 1,
		path:             append(g.current.path, sessionID),
	}

	g.current.children = append(g.current.children, child)
	g.current = child
}

// ExitScope returns to the parent scope.
// Called when exiting a session block.
func (g *ScopeGraph) ExitScope() error {
	if g.current.parent == nil {
		return fmt.Errorf("cannot exit root scope")
	}
	g.current = g.current.parent
	return nil
}

// Store adds a variable to the current scope.
func (g *ScopeGraph) Store(varName, origin string, value any, class VarClass, taint VarTaint) {
	g.current.vars[varName] = VarEntry{
		Value:  value,
		Origin: origin,
		Class:  class,
		Taint:  taint,
	}
}

// Resolve looks up a variable by traversing up the scope chain.
// Returns the value, the scope where it was found, and any error.
// Respects transport boundary sealing.
func (g *ScopeGraph) Resolve(varName string) (any, *Scope, error) {
	// Check current scope first
	if entry, ok := g.current.vars[varName]; ok {
		return entry.Value, g.current, nil
	}

	// If sealed from parent, check imports
	if g.current.sealedFromParent {
		if !g.current.imported[varName] {
			return nil, nil, &TransportBoundaryError{
				VarName:      varName,
				CurrentScope: g.current.sessionID,
				ParentScope:  g.current.parent.sessionID,
			}
		}
	}

	// Traverse up parent chain
	return g.resolveInParent(varName)
}

// resolveInParent looks up a variable in the parent chain.
func (g *ScopeGraph) resolveInParent(varName string) (any, *Scope, error) {
	scope := g.current.parent

	for scope != nil {
		if entry, ok := scope.vars[varName]; ok {
			return entry.Value, scope, nil
		}
		scope = scope.parent
	}

	return nil, nil, fmt.Errorf("variable %q not found in scope chain", varName)
}

// GetEntry retrieves the full VarEntry (including metadata) for a variable.
func (g *ScopeGraph) GetEntry(varName string) (*VarEntry, error) {
	// Check current scope
	if entry, ok := g.current.vars[varName]; ok {
		return &entry, nil
	}

	// If sealed, check imports
	if g.current.sealedFromParent && !g.current.imported[varName] {
		return nil, &TransportBoundaryError{
			VarName:      varName,
			CurrentScope: g.current.sessionID,
			ParentScope:  g.current.parent.sessionID,
		}
	}

	// Traverse parent chain
	scope := g.current.parent
	for scope != nil {
		if entry, ok := scope.vars[varName]; ok {
			return &entry, nil
		}
		scope = scope.parent
	}

	return nil, fmt.Errorf("variable %q not found in scope chain", varName)
}

// CurrentSessionID returns the session ID of the current scope.
func (g *ScopeGraph) CurrentSessionID() string {
	return g.current.sessionID
}

// ScopePath returns the path from root to current scope.
func (g *ScopeGraph) ScopePath() []string {
	return g.current.path
}

// IsSealed returns true if the current scope is sealed from parent.
func (g *ScopeGraph) IsSealed() bool {
	return g.current.sealedFromParent
}

// TransportDepth returns the transport nesting depth (0=local, 1=ssh, 2=ssh→docker).
func (g *ScopeGraph) TransportDepth() int {
	return g.current.transportDepth
}

// DebugPrint prints the scope tree for debugging.
func (g *ScopeGraph) DebugPrint() string {
	var b strings.Builder
	g.root.debugPrint(&b, 0)
	return b.String()
}

// AsMap returns all accessible variables as a flat map.
// Respects transport boundaries: when traversing to a parent across a sealed
// boundary, only explicitly imported variables are included.
// Used for decorator evaluation context.
func (g *ScopeGraph) AsMap() map[string]any {
	result := make(map[string]any)

	// Traverse from current scope up to root
	scope := g.current
	for scope != nil {
		// Add variables from this scope (don't overwrite - child shadows parent)
		for name, entry := range scope.vars {
			if _, exists := result[name]; !exists {
				result[name] = entry.Value
			}
		}

		// Move to parent, respecting transport boundaries
		if scope.parent != nil && scope.sealedFromParent {
			// This scope is sealed from its parent (transport boundary).
			// Only traverse to parent if we have explicit imports.
			// Add only imported variables from parent chain.
			for name := range scope.imported {
				if _, exists := result[name]; !exists {
					// Look up the imported variable in parent chain
					if value, _, err := g.resolveInParentChain(scope.parent, name); err == nil {
						result[name] = value
					}
				}
			}
			// Stop traversal - sealed boundary blocks implicit parent access
			break
		}

		scope = scope.parent
	}

	return result
}

// resolveInParentChain looks up a variable in the parent chain starting from the given scope.
// Used by AsMap to resolve imported variables across sealed boundaries.
func (g *ScopeGraph) resolveInParentChain(startScope *Scope, varName string) (any, *Scope, error) {
	scope := startScope
	for scope != nil {
		if entry, ok := scope.vars[varName]; ok {
			return entry.Value, scope, nil
		}
		scope = scope.parent
	}
	return nil, nil, fmt.Errorf("variable %q not found in parent chain", varName)
}

func (s *Scope) debugPrint(b *strings.Builder, indent int) {
	prefix := strings.Repeat("  ", indent)

	sealed := ""
	if s.sealedFromParent {
		sealed = " [SEALED]"
	}

	fmt.Fprintf(b, "%s%s (session=%s, depth=%d)%s\n", prefix, s.id, s.sessionID, s.transportDepth, sealed)

	// Print variables
	for name, entry := range s.vars {
		fmt.Fprintf(b, "%s  %s = %v (origin=%s, class=%d, taint=%d)\n",
			prefix, name, entry.Value, entry.Origin, entry.Class, entry.Taint)
	}

	// Print imported variables
	if len(s.imported) > 0 {
		fmt.Fprintf(b, "%s  imports: %v\n", prefix, s.imported)
	}

	// Print children
	for _, child := range s.children {
		child.debugPrint(b, indent+1)
	}
}

// TransportBoundaryError is returned when attempting to access a parent variable
// across a transport boundary without explicitly passing it.
// Follows the CrossSessionLeakageError format for consistency.
type TransportBoundaryError struct {
	VarName      string
	CurrentScope string
	ParentScope  string
}

func (e *TransportBoundaryError) Error() string {
	var b strings.Builder

	// Error header (Rust-style, matches CrossSessionLeakageError)
	fmt.Fprintf(&b, "Error: Transport boundary violation\n")
	fmt.Fprintf(&b, "  --> Variable '%s' cannot cross from %s to %s\n",
		e.VarName, e.ParentScope, e.CurrentScope)
	fmt.Fprintf(&b, "   |\n")

	// Details
	fmt.Fprintf(&b, "   | Variable: %s\n", e.VarName)
	fmt.Fprintf(&b, "   | Parent:   %s session\n", e.ParentScope)
	fmt.Fprintf(&b, "   | Current:  %s session\n", e.CurrentScope)
	fmt.Fprintf(&b, "   |\n")

	// Suggestion
	fmt.Fprintf(&b, "   = Suggestion: Pass variables explicitly via decorator parameters\n")
	fmt.Fprintf(&b, "   = Example:\n")
	fmt.Fprintf(&b, "       @ssh(host=\"server\", env={%s: @var.%s}) {\n", e.VarName, e.VarName)
	fmt.Fprintf(&b, "           echo $%s  # Available as environment variable\n", e.VarName)
	fmt.Fprintf(&b, "       }\n")
	fmt.Fprintf(&b, "   = Note: Transport boundaries block implicit variable access for security.\n")

	return b.String()
}
