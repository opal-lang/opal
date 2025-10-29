package decorator

import "context"

import (
	"strings"
	"testing"
)

// TestSessionIsolationEnvironment verifies env changes don't leak between sessions
func TestSessionIsolationEnvironment(t *testing.T) {
	// Create two independent sessions
	session1 := NewLocalSession()
	session2 := NewLocalSession()

	// Modify session1's environment
	session1Modified := session1.WithEnv(map[string]string{
		"TEST_VAR": "session1_value",
	})

	// Modify session2's environment
	session2Modified := session2.WithEnv(map[string]string{
		"TEST_VAR": "session2_value",
	})

	// Run command in session1
	result1, err := session1Modified.Run(context.Background(), []string{"sh", "-c", "echo $TEST_VAR"}, RunOpts{})
	if err != nil {
		t.Fatalf("Session1 run failed: %v", err)
	}

	// Run command in session2
	result2, err := session2Modified.Run(context.Background(), []string{"sh", "-c", "echo $TEST_VAR"}, RunOpts{})
	if err != nil {
		t.Fatalf("Session2 run failed: %v", err)
	}

	// Verify each session sees its own value
	output1 := strings.TrimSpace(string(result1.Stdout))
	output2 := strings.TrimSpace(string(result2.Stdout))

	if output1 != "session1_value" {
		t.Errorf("Session1 output: got %q, want %q", output1, "session1_value")
	}

	if output2 != "session2_value" {
		t.Errorf("Session2 output: got %q, want %q", output2, "session2_value")
	}

	// Verify original sessions are unchanged
	env1 := session1.Env()
	env2 := session2.Env()

	if _, ok := env1["TEST_VAR"]; ok {
		t.Error("Original session1 was mutated (TEST_VAR should not exist)")
	}

	if _, ok := env2["TEST_VAR"]; ok {
		t.Error("Original session2 was mutated (TEST_VAR should not exist)")
	}
}

// TestSessionIsolationWithEnvDoesNotMutateParent verifies WithEnv is copy-on-write
func TestSessionIsolationWithEnvDoesNotMutateParent(t *testing.T) {
	parent := NewLocalSession()

	// Create child with modified env
	child := parent.WithEnv(map[string]string{
		"CHILD_VAR": "child_value",
	})

	// Verify parent doesn't have child's var
	parentEnv := parent.Env()
	if _, ok := parentEnv["CHILD_VAR"]; ok {
		t.Error("Parent session was mutated by WithEnv")
	}

	// Verify child has the var
	childEnv := child.Env()
	if childEnv["CHILD_VAR"] != "child_value" {
		t.Errorf("Child env: got %q, want %q", childEnv["CHILD_VAR"], "child_value")
	}

	// Verify child still has parent's env
	if _, ok := childEnv["PATH"]; !ok {
		t.Error("Child session missing parent's PATH")
	}
}

// TestSessionIsolationEnvMutationDoesNotAffectOthers verifies env mutations are isolated
func TestSessionIsolationEnvMutationDoesNotAffectOthers(t *testing.T) {
	base := NewLocalSession()

	// Create two children from same parent
	child1 := base.WithEnv(map[string]string{"VAR": "child1"})
	child2 := base.WithEnv(map[string]string{"VAR": "child2"})

	// Verify each child has its own value
	env1 := child1.Env()
	env2 := child2.Env()

	if env1["VAR"] != "child1" {
		t.Errorf("Child1 VAR: got %q, want %q", env1["VAR"], "child1")
	}

	if env2["VAR"] != "child2" {
		t.Errorf("Child2 VAR: got %q, want %q", env2["VAR"], "child2")
	}

	// Verify base is unchanged
	baseEnv := base.Env()
	if _, ok := baseEnv["VAR"]; ok {
		t.Error("Base session was mutated")
	}
}

// TestSessionIsolationEnvSnapshotImmutable verifies Env() returns immutable snapshot
func TestSessionIsolationEnvSnapshotImmutable(t *testing.T) {
	session := NewLocalSession()

	// Get env snapshot
	env1 := session.Env()

	// Try to mutate the snapshot
	env1["INJECTED_VAR"] = "malicious_value"

	// Get another snapshot
	env2 := session.Env()

	// Verify mutation didn't affect session
	if _, ok := env2["INJECTED_VAR"]; ok {
		t.Error("Session env was mutated via returned map (not immutable)")
	}

	// Verify session commands don't see the injected var
	result, err := session.Run(context.Background(), []string{"sh", "-c", "echo $INJECTED_VAR"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "" {
		t.Errorf("Session saw injected var: %q", output)
	}
}

// TestSessionPoolIsolation verifies pooled sessions don't leak state
func TestSessionPoolIsolation(t *testing.T) {
	pool := NewSessionPool()
	defer pool.CloseAll()

	// Use LocalSession-based transport for proper WithEnv behavior
	transport := &localTransport{}
	parent := NewLocalSession()

	// Create session with params1
	params1 := map[string]any{"id": "session1"}
	session1, err := pool.GetOrCreate(transport, parent, params1)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Modify session1's environment
	session1Modified := session1.WithEnv(map[string]string{
		"SESSION_ID": "session1",
	})

	// Create session with params2 (different params)
	params2 := map[string]any{"id": "session2"}
	session2, err := pool.GetOrCreate(transport, parent, params2)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Verify session2 doesn't have session1's modifications
	env2 := session2.Env()
	if _, ok := env2["SESSION_ID"]; ok {
		t.Error("Session2 has SESSION_ID from session1 (state leaked)")
	}

	// Verify session1Modified has the var
	env1Modified := session1Modified.Env()
	if env1Modified["SESSION_ID"] != "session1" {
		t.Errorf("Session1Modified SESSION_ID: got %q, want %q", env1Modified["SESSION_ID"], "session1")
	}

	// Verify original session1 is unchanged (copy-on-write)
	env1 := session1.Env()
	if _, ok := env1["SESSION_ID"]; ok {
		t.Error("Original session1 was mutated by WithEnv")
	}
}

// localTransport creates LocalSession instances for testing
type localTransport struct {
	openCount int
}

func (t *localTransport) Descriptor() Descriptor {
	return Descriptor{Path: "local"}
}

func (t *localTransport) Open(parent Session, params map[string]any) (Session, error) {
	t.openCount++
	return NewLocalSession(), nil
}

func (t *localTransport) Wrap(next ExecNode, params map[string]any) ExecNode {
	return nil
}

// TestSessionPoolReuseDoesNotLeakState verifies reused sessions are clean
func TestSessionPoolReuseDoesNotLeakState(t *testing.T) {
	pool := NewSessionPool()
	defer pool.CloseAll()

	transport := &mockTransport{}
	parent := NewLocalSession()
	params := map[string]any{"host": "prod"}

	// First use: get session and modify it
	session1, _ := pool.GetOrCreate(transport, parent, params)
	session1Modified := session1.WithEnv(map[string]string{"FIRST_USE": "true"})

	// Use the modified session
	_ = session1Modified.Env()

	// Second use: get same session (should be reused)
	session2, _ := pool.GetOrCreate(transport, parent, params)

	// Verify session2 is the same instance as session1 (not session1Modified)
	if session2 != session1 {
		t.Error("Pool returned different session for same params")
	}

	// Verify session2 doesn't have modifications from session1Modified
	env2 := session2.Env()
	if _, ok := env2["FIRST_USE"]; ok {
		t.Error("Reused session has state from previous WithEnv (leaked)")
	}
}

// TestSessionIsolationConcurrentModifications verifies concurrent env modifications are isolated
func TestSessionIsolationConcurrentModifications(t *testing.T) {
	base := NewLocalSession()

	// Create 10 children concurrently, each with unique env
	done := make(chan Session, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			child := base.WithEnv(map[string]string{
				"GOROUTINE_ID": string(rune('0' + id)),
			})
			done <- child
		}(i)
	}

	// Collect all children
	children := make([]Session, 10)
	for i := 0; i < 10; i++ {
		children[i] = <-done
	}

	// Verify each child has its own unique env
	seen := make(map[string]bool)
	for i, child := range children {
		env := child.Env()
		id := env["GOROUTINE_ID"]

		if id == "" {
			t.Errorf("Child %d missing GOROUTINE_ID", i)
			continue
		}

		if seen[id] {
			t.Errorf("Duplicate GOROUTINE_ID: %s", id)
		}
		seen[id] = true
	}

	// Verify base is unchanged
	baseEnv := base.Env()
	if _, ok := baseEnv["GOROUTINE_ID"]; ok {
		t.Error("Base session was mutated by concurrent WithEnv calls")
	}
}
