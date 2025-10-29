package decorator

import "context"

import (
	"testing"
)

// TestMonitoredSessionTracksMethodCalls verifies monitoring works
func TestMonitoredSessionTracksMethodCalls(t *testing.T) {
	base := NewLocalSession()
	monitored := NewMonitoredSession(base)

	// Call various methods
	monitored.Env()
	monitored.Cwd()
	monitored.Run(context.Background(), []string{"echo", "test"}, RunOpts{})

	stats := monitored.Stats()

	if stats.EnvCalls != 1 {
		t.Errorf("EnvCalls: got %d, want 1", stats.EnvCalls)
	}

	if stats.CwdCalls != 1 {
		t.Errorf("CwdCalls: got %d, want 1", stats.CwdCalls)
	}

	if stats.RunCalls != 1 {
		t.Errorf("RunCalls: got %d, want 1", stats.RunCalls)
	}
}

// TestMonitoredSessionWithEnvTracking verifies WithEnv deltas are captured
func TestMonitoredSessionWithEnvTracking(t *testing.T) {
	base := NewLocalSession()
	monitored := NewMonitoredSession(base)

	// Create child with env delta
	delta1 := map[string]string{"VAR1": "value1"}
	child1 := monitored.WithEnv(delta1)

	// Create another child
	delta2 := map[string]string{"VAR2": "value2"}
	child1.WithEnv(delta2)

	stats := monitored.Stats()

	if stats.WithEnvCalls != 2 {
		t.Errorf("WithEnvCalls: got %d, want 2", stats.WithEnvCalls)
	}

	// Verify deltas were captured
	if len(stats.WithEnvDeltas) != 2 {
		t.Fatalf("WithEnvDeltas length: got %d, want 2", len(stats.WithEnvDeltas))
	}

	if stats.WithEnvDeltas[0]["VAR1"] != "value1" {
		t.Errorf("First delta VAR1: got %q, want %q", stats.WithEnvDeltas[0]["VAR1"], "value1")
	}

	if stats.WithEnvDeltas[1]["VAR2"] != "value2" {
		t.Errorf("Second delta VAR2: got %q, want %q", stats.WithEnvDeltas[1]["VAR2"], "value2")
	}
}

// TestMonitoredSessionSharedStats verifies child sessions share stats
func TestMonitoredSessionSharedStats(t *testing.T) {
	base := NewLocalSession()
	monitored := NewMonitoredSession(base)

	// Create child
	child := monitored.WithEnv(map[string]string{"VAR": "value"})

	// Call method on child
	child.Env()

	// Stats should be shared
	stats := monitored.Stats()

	if stats.WithEnvCalls != 1 {
		t.Errorf("WithEnvCalls: got %d, want 1", stats.WithEnvCalls)
	}

	if stats.EnvCalls != 1 {
		t.Errorf("EnvCalls: got %d, want 1 (child call should be tracked)", stats.EnvCalls)
	}
}

// TestMonitoredTransportTracksOpenCalls verifies transport monitoring
func TestMonitoredTransportTracksOpenCalls(t *testing.T) {
	// Create a simple transport that returns LocalSession
	baseTransport := &localTransport{}
	monitored := NewMonitoredTransport(baseTransport)

	parent := NewLocalSession()
	params := map[string]any{"host": "test"}

	// Call Open twice
	monitored.Open(parent, params)
	monitored.Open(parent, params)

	if monitored.OpenCalls != 2 {
		t.Errorf("OpenCalls: got %d, want 2", monitored.OpenCalls)
	}
}

// TestSessionPoolWithMonitoring verifies pool behavior with monitoring
func TestSessionPoolWithMonitoring(t *testing.T) {
	pool := NewSessionPool()
	defer pool.CloseAll()

	// Use monitored transport
	baseTransport := &localTransport{}
	transport := NewMonitoredTransport(baseTransport)

	parent := NewLocalSession()
	params := map[string]any{"host": "prod"}

	// First call creates session
	session1, _ := pool.GetOrCreate(transport, parent, params)

	// Second call reuses session
	session2, _ := pool.GetOrCreate(transport, parent, params)

	// Verify only one Open call
	if transport.OpenCalls != 1 {
		t.Errorf("OpenCalls: got %d, want 1 (should reuse)", transport.OpenCalls)
	}

	// Verify same session returned
	if session1 != session2 {
		t.Error("Expected same session instance")
	}

	// Both should be MonitoredSession
	monitored1, ok := session1.(*MonitoredSession)
	if !ok {
		t.Fatal("Session1 is not MonitoredSession")
	}

	monitored2, ok := session2.(*MonitoredSession)
	if !ok {
		t.Fatal("Session2 is not MonitoredSession")
	}

	// Should share stats (same underlying session)
	if monitored1.Stats() != monitored2.Stats() {
		t.Error("Expected shared stats for reused session")
	}
}

// TestMonitoredSessionIsolation verifies monitoring doesn't break isolation
func TestMonitoredSessionIsolation(t *testing.T) {
	base := NewLocalSession()
	monitored := NewMonitoredSession(base)

	// Create two children with different env
	child1 := monitored.WithEnv(map[string]string{"VAR": "child1"})
	child2 := monitored.WithEnv(map[string]string{"VAR": "child2"})

	// Verify each child has its own env
	env1 := child1.Env()
	env2 := child2.Env()

	if env1["VAR"] != "child1" {
		t.Errorf("Child1 VAR: got %q, want %q", env1["VAR"], "child1")
	}

	if env2["VAR"] != "child2" {
		t.Errorf("Child2 VAR: got %q, want %q", env2["VAR"], "child2")
	}

	// Verify stats tracked both WithEnv calls
	stats := monitored.Stats()
	if stats.WithEnvCalls != 2 {
		t.Errorf("WithEnvCalls: got %d, want 2", stats.WithEnvCalls)
	}

	// Verify both Env calls were tracked
	if stats.EnvCalls != 2 {
		t.Errorf("EnvCalls: got %d, want 2", stats.EnvCalls)
	}
}
