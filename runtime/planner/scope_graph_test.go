package planner

import (
	"errors"
	"strings"
	"testing"
)

func TestScopeGraphBasics(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable in root scope
	g.Store("HOME", "literal", "/home/alice", VarClassData, VarTaintAgnostic)

	// Resolve from same scope
	val, scope, err := g.Resolve("HOME")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "/home/alice" {
		t.Errorf("Expected /home/alice, got %v", val)
	}
	if scope.sessionID != "local" {
		t.Errorf("Expected scope local, got %s", scope.sessionID)
	}
}

func TestScopeGraphTraversal(t *testing.T) {
	g := NewScopeGraph("local")

	// Store in root scope
	g.Store("HOME", "literal", "/home/alice", VarClassData, VarTaintAgnostic)

	// Enter child scope (non-transport)
	g.EnterScope("retry", false)

	// Should find variable in parent
	val, scope, err := g.Resolve("HOME")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "/home/alice" {
		t.Errorf("Expected /home/alice, got %v", val)
	}
	if scope.sessionID != "local" {
		t.Errorf("Expected scope local, got %s", scope.sessionID)
	}
}

func TestScopeGraphSiblingIsolation(t *testing.T) {
	g := NewScopeGraph("local")

	// Enter first child
	g.EnterScope("ssh:server1", true)
	g.Store("REMOTE_HOME", "@env.HOME", "/home/bob", VarClassData, VarTaintAgnostic)

	// Exit and enter second child
	g.ExitScope()
	g.EnterScope("ssh:server2", true)

	// Should NOT find sibling's variable (will get transport boundary error
	// because it tries to look in parent, but variable isn't there)
	_, _, err := g.Resolve("REMOTE_HOME")
	if err == nil {
		t.Fatal("Expected error for sibling variable, got nil")
	}
	// Error can be either "not found" or "transport boundary" depending on
	// whether it checks parent scope first
}

func TestScopeGraphNesting(t *testing.T) {
	g := NewScopeGraph("local")

	// Root scope
	g.Store("LOCAL", "literal", "local-value", VarClassData, VarTaintAgnostic)

	// First level (SSH)
	g.EnterScope("ssh:server", true)
	g.Store("SSH_VAR", "literal", "ssh-value", VarClassData, VarTaintAgnostic)

	// Second level (Docker inside SSH)
	g.EnterScope("docker:container", true)
	g.Store("DOCKER_VAR", "literal", "docker-value", VarClassData, VarTaintAgnostic)

	// Should find current scope variable
	val, _, err := g.Resolve("DOCKER_VAR")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "docker-value" {
		t.Errorf("Expected docker-value, got %v", val)
	}

	// Check transport depth
	if g.TransportDepth() != 2 {
		t.Errorf("Expected transport depth 2, got %d", g.TransportDepth())
	}
}

func TestScopeGraphExitRoot(t *testing.T) {
	g := NewScopeGraph("local")

	// Try to exit root scope
	err := g.ExitScope()
	if err == nil {
		t.Fatal("Expected error when exiting root scope")
	}
	if !strings.Contains(err.Error(), "cannot exit root scope") {
		t.Errorf("Expected 'cannot exit root scope' error, got: %v", err)
	}
}

func TestTransportBoundarySealing(t *testing.T) {
	g := NewScopeGraph("local")

	// Store secret in local scope
	g.Store("SECRET", "@env.API_KEY", "secret-value", VarClassSecret, VarTaintLocalOnly)

	// Enter transport boundary (SSH)
	g.EnterScope("ssh:server", true)

	// Should be sealed
	if !g.IsSealed() {
		t.Error("Expected scope to be sealed at transport boundary")
	}

	// Try to access parent variable without import
	_, _, err := g.Resolve("SECRET")
	if err == nil {
		t.Fatal("Expected TransportBoundaryError, got nil")
	}

	var boundaryErr *TransportBoundaryError
	if !errors.As(err, &boundaryErr) {
		t.Fatalf("Expected TransportBoundaryError, got %T: %v", err, err)
	}

	// Verify error details
	if boundaryErr.VarName != "SECRET" {
		t.Errorf("Expected VarName=SECRET, got %s", boundaryErr.VarName)
	}
	if boundaryErr.ParentScope != "local" {
		t.Errorf("Expected ParentScope=local, got %s", boundaryErr.ParentScope)
	}
	if boundaryErr.CurrentScope != "ssh:server" {
		t.Errorf("Expected CurrentScope=ssh:server, got %s", boundaryErr.CurrentScope)
	}
}

func TestNonTransportBoundaryNotSealed(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable in root
	g.Store("VAR", "literal", "value", VarClassData, VarTaintAgnostic)

	// Enter non-transport scope (e.g., @retry)
	g.EnterScope("retry", false)

	// Should NOT be sealed
	if g.IsSealed() {
		t.Error("Expected scope NOT to be sealed for non-transport boundary")
	}

	// Should be able to access parent without import
	val, _, err := g.Resolve("VAR")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "value" {
		t.Errorf("Expected value, got %v", val)
	}
}

func TestGetEntry(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable with metadata
	g.Store("SECRET", "@env.API_KEY", "secret-value", VarClassSecret, VarTaintLocalOnly)

	// Get entry
	entry, err := g.GetEntry("SECRET")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	// Verify metadata
	if entry.Value != "secret-value" {
		t.Errorf("Expected secret-value, got %v", entry.Value)
	}
	if entry.Origin != "@env.API_KEY" {
		t.Errorf("Expected @env.API_KEY, got %s", entry.Origin)
	}
	if entry.Class != VarClassSecret {
		t.Errorf("Expected VarClassSecret, got %d", entry.Class)
	}
	if entry.Taint != VarTaintLocalOnly {
		t.Errorf("Expected VarTaintLocalOnly, got %d", entry.Taint)
	}
}

func TestScopePath(t *testing.T) {
	g := NewScopeGraph("local")

	// Check root path
	path := g.ScopePath()
	if len(path) != 1 || path[0] != "local" {
		t.Errorf("Expected [local], got %v", path)
	}

	// Enter child
	g.EnterScope("ssh:server", true)
	path = g.ScopePath()
	if len(path) != 2 || path[0] != "local" || path[1] != "ssh:server" {
		t.Errorf("Expected [local ssh:server], got %v", path)
	}

	// Enter grandchild
	g.EnterScope("docker:container", true)
	path = g.ScopePath()
	if len(path) != 3 {
		t.Errorf("Expected 3 elements, got %v", path)
	}
}

func TestDebugPrint(t *testing.T) {
	g := NewScopeGraph("local")
	g.Store("HOME", "literal", "/home/alice", VarClassData, VarTaintAgnostic)

	g.EnterScope("ssh:server", true)
	g.Store("REMOTE", "@env.HOME", "/home/bob", VarClassData, VarTaintAgnostic)

	output := g.DebugPrint()

	// Check that output contains expected elements
	if !strings.Contains(output, "root") {
		t.Error("Expected output to contain 'root'")
	}
	if !strings.Contains(output, "local") {
		t.Error("Expected output to contain 'local'")
	}
	if !strings.Contains(output, "ssh:server") {
		t.Error("Expected output to contain 'ssh:server'")
	}
	if !strings.Contains(output, "[SEALED]") {
		t.Error("Expected output to contain '[SEALED]'")
	}
	if !strings.Contains(output, "HOME") {
		t.Error("Expected output to contain 'HOME'")
	}
}

func TestScopeGraphAsMap(t *testing.T) {
	g := NewScopeGraph("local")

	// Store in root
	g.Store("ROOT_VAR", "literal", "root", VarClassData, VarTaintAgnostic)

	// Enter child scope
	g.EnterScope("child", false)
	g.Store("CHILD_VAR", "literal", "child", VarClassData, VarTaintAgnostic)

	// AsMap should include both
	vars := g.AsMap()

	if vars["ROOT_VAR"] != "root" {
		t.Errorf("Expected ROOT_VAR=root, got %v", vars["ROOT_VAR"])
	}
	if vars["CHILD_VAR"] != "child" {
		t.Errorf("Expected CHILD_VAR=child, got %v", vars["CHILD_VAR"])
	}

	// Exit to root
	if err := g.ExitScope(); err != nil {
		t.Fatalf("ExitScope failed: %v", err)
	}

	// AsMap should only have root var
	vars = g.AsMap()
	if vars["ROOT_VAR"] != "root" {
		t.Errorf("Expected ROOT_VAR=root, got %v", vars["ROOT_VAR"])
	}
	if _, exists := vars["CHILD_VAR"]; exists {
		t.Error("CHILD_VAR should not be accessible from root scope")
	}
}

func TestScopeGraphAsMapShadowing(t *testing.T) {
	g := NewScopeGraph("local")

	// Store in root
	g.Store("VAR", "literal", "parent", VarClassData, VarTaintAgnostic)

	// Enter child and shadow
	g.EnterScope("child", false)
	g.Store("VAR", "literal", "child", VarClassData, VarTaintAgnostic)

	// AsMap should return child value (shadowing)
	vars := g.AsMap()
	if vars["VAR"] != "child" {
		t.Errorf("Expected VAR=child (shadowed), got %v", vars["VAR"])
	}

	// Exit to root
	if err := g.ExitScope(); err != nil {
		t.Fatalf("ExitScope failed: %v", err)
	}

	// AsMap should return parent value
	vars = g.AsMap()
	if vars["VAR"] != "parent" {
		t.Errorf("Expected VAR=parent, got %v", vars["VAR"])
	}
}

func TestScopeGraphAsMapNested(t *testing.T) {
	g := NewScopeGraph("local")

	// Store in root
	g.Store("A", "literal", "a", VarClassData, VarTaintAgnostic)

	// Enter child
	g.EnterScope("child1", false)
	g.Store("B", "literal", "b", VarClassData, VarTaintAgnostic)

	// Enter grandchild
	g.EnterScope("child2", false)
	g.Store("C", "literal", "c", VarClassData, VarTaintAgnostic)

	// AsMap should include all three
	vars := g.AsMap()
	if len(vars) != 3 {
		t.Errorf("Expected 3 variables, got %d", len(vars))
	}
	if vars["A"] != "a" {
		t.Errorf("Expected A=a, got %v", vars["A"])
	}
	if vars["B"] != "b" {
		t.Errorf("Expected B=b, got %v", vars["B"])
	}
	if vars["C"] != "c" {
		t.Errorf("Expected C=c, got %v", vars["C"])
	}
}

func TestScopeGraphAsMapEmpty(t *testing.T) {
	g := NewScopeGraph("local")

	// AsMap on empty scope should return empty map
	vars := g.AsMap()
	if len(vars) != 0 {
		t.Errorf("Expected empty map, got %d variables", len(vars))
	}
}

func TestScopeGraphAsMapRespectsTransportBoundaries(t *testing.T) {
	g := NewScopeGraph("local")

	// Store in root (local session)
	g.Store("LOCAL_VAR", "literal", "local", VarClassData, VarTaintAgnostic)
	g.Store("SHARED_VAR", "literal", "shared", VarClassData, VarTaintAgnostic)

	// Enter sealed scope (SSH session - transport boundary)
	g.EnterScope("ssh:server", true)
	g.Store("REMOTE_VAR", "literal", "remote", VarClassData, VarTaintAgnostic)

	// AsMap should NOT include parent variables (sealed boundary)
	vars := g.AsMap()
	if len(vars) != 1 {
		t.Errorf("Expected 1 variable (REMOTE_VAR only), got %d: %v", len(vars), vars)
	}
	if vars["REMOTE_VAR"] != "remote" {
		t.Errorf("Expected REMOTE_VAR=remote, got %v", vars["REMOTE_VAR"])
	}
	if _, exists := vars["LOCAL_VAR"]; exists {
		t.Error("LOCAL_VAR should NOT be accessible across sealed boundary")
	}
	if _, exists := vars["SHARED_VAR"]; exists {
		t.Error("SHARED_VAR should NOT be accessible across sealed boundary")
	}
}

func TestScopeGraphAsMapNestedBoundaries(t *testing.T) {
	g := NewScopeGraph("local")

	// Root scope
	g.Store("ROOT_VAR", "literal", "root", VarClassData, VarTaintAgnostic)

	// First boundary (SSH)
	g.EnterScope("ssh:server", true)
	g.Store("SSH_VAR", "literal", "ssh", VarClassData, VarTaintAgnostic)

	// Second boundary (Docker inside SSH)
	g.EnterScope("docker:container", true)
	g.Store("DOCKER_VAR", "literal", "docker", VarClassData, VarTaintAgnostic)

	// AsMap should only see DOCKER_VAR (both parent boundaries are sealed)
	vars := g.AsMap()
	if len(vars) != 1 {
		t.Errorf("Expected 1 variable (DOCKER_VAR only), got %d: %v", len(vars), vars)
	}
	if vars["DOCKER_VAR"] != "docker" {
		t.Errorf("Expected DOCKER_VAR=docker, got %v", vars["DOCKER_VAR"])
	}
	if _, exists := vars["SSH_VAR"]; exists {
		t.Error("SSH_VAR should NOT be accessible across sealed boundary")
	}
	if _, exists := vars["ROOT_VAR"]; exists {
		t.Error("ROOT_VAR should NOT be accessible across sealed boundary")
	}
}
