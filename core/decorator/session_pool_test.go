package decorator

import (
	"context"
	"io/fs"
	"testing"
)

// TestSessionPoolGetOrCreate verifies session pooling creates sessions on-demand
func TestSessionPoolGetOrCreate(t *testing.T) {
	pool := NewSessionPool()
	defer pool.CloseAll()

	// Mock transport
	transport := &mockTransport{}
	parent := NewLocalSession()
	params := map[string]any{"host": "prod"}

	// First call creates session
	session1, err := pool.GetOrCreate(transport, parent, params)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if session1 == nil {
		t.Fatal("Expected session, got nil")
	}

	// Verify transport.Open was called
	if transport.openCount != 1 {
		t.Errorf("Open count: got %d, want 1", transport.openCount)
	}

	// Second call with same params reuses session
	session2, err := pool.GetOrCreate(transport, parent, params)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Should be same session instance
	if session1 != session2 {
		t.Error("Expected same session instance, got different")
	}

	// Verify transport.Open was NOT called again
	if transport.openCount != 1 {
		t.Errorf("Open count: got %d, want 1 (should reuse)", transport.openCount)
	}
}

// TestSessionPoolDifferentParams verifies different params create different sessions
func TestSessionPoolDifferentParams(t *testing.T) {
	pool := NewSessionPool()
	defer pool.CloseAll()

	transport := &mockTransport{}
	parent := NewLocalSession()

	// Create session for prod
	params1 := map[string]any{"host": "prod"}
	session1, err := pool.GetOrCreate(transport, parent, params1)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Create session for staging (different params)
	params2 := map[string]any{"host": "staging"}
	session2, err := pool.GetOrCreate(transport, parent, params2)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Should be different sessions
	if session1 == session2 {
		t.Error("Expected different sessions for different params")
	}

	// Verify two Open calls
	if transport.openCount != 2 {
		t.Errorf("Open count: got %d, want 2", transport.openCount)
	}
}

// TestSessionPoolCloseAll verifies all sessions are closed
func TestSessionPoolCloseAll(t *testing.T) {
	pool := NewSessionPool()

	transport := &mockTransport{}
	parent := NewLocalSession()

	// Create two sessions
	params1 := map[string]any{"host": "prod"}
	session1, _ := pool.GetOrCreate(transport, parent, params1)
	mockSession1 := session1.(*mockSession)

	params2 := map[string]any{"host": "staging"}
	session2, _ := pool.GetOrCreate(transport, parent, params2)
	mockSession2 := session2.(*mockSession)

	// Close all
	pool.CloseAll()

	// Verify both sessions were closed
	if !mockSession1.closed {
		t.Error("Session 1 was not closed")
	}
	if !mockSession2.closed {
		t.Error("Session 2 was not closed")
	}
}

// TestSessionPoolConcurrentAccess verifies thread-safe access
func TestSessionPoolConcurrentAccess(t *testing.T) {
	pool := NewSessionPool()
	defer pool.CloseAll()

	transport := &mockTransport{}
	parent := NewLocalSession()
	params := map[string]any{"host": "prod"}

	// Create sessions concurrently
	done := make(chan Session, 10)
	for i := 0; i < 10; i++ {
		go func() {
			session, _ := pool.GetOrCreate(transport, parent, params)
			done <- session
		}()
	}

	// Collect all sessions
	sessions := make([]Session, 10)
	for i := 0; i < 10; i++ {
		sessions[i] = <-done
	}

	// All should be the same instance
	for i := 1; i < 10; i++ {
		if sessions[i] != sessions[0] {
			t.Errorf("Session %d is different from session 0", i)
		}
	}

	// Only one Open call should have happened
	if transport.openCount != 1 {
		t.Errorf("Open count: got %d, want 1", transport.openCount)
	}
}

// TestSessionKeyDeterministic verifies session keys are deterministic
func TestSessionKeyDeterministic(t *testing.T) {
	params := map[string]any{
		"host": "prod",
		"port": 22,
		"user": "deploy",
	}

	// Generate key multiple times
	key1 := sessionKey("ssh", params)
	key2 := sessionKey("ssh", params)

	if key1 != key2 {
		t.Errorf("Keys not deterministic: %s != %s", key1, key2)
	}
}

// TestSessionKeyDifferentOrder verifies key is same regardless of param order
func TestSessionKeyDifferentOrder(t *testing.T) {
	params1 := map[string]any{
		"host": "prod",
		"port": 22,
		"user": "deploy",
	}

	params2 := map[string]any{
		"user": "deploy",
		"host": "prod",
		"port": 22,
	}

	key1 := sessionKey("ssh", params1)
	key2 := sessionKey("ssh", params2)

	if key1 != key2 {
		t.Errorf("Keys differ for same params in different order: %s != %s", key1, key2)
	}
}

// TestSessionKeyDifferentParams verifies different params produce different keys
func TestSessionKeyDifferentParams(t *testing.T) {
	params1 := map[string]any{"host": "prod"}
	params2 := map[string]any{"host": "staging"}

	key1 := sessionKey("ssh", params1)
	key2 := sessionKey("ssh", params2)

	if key1 == key2 {
		t.Error("Expected different keys for different params")
	}
}

// Mock implementations for testing

type mockTransport struct {
	openCount int
}

func (m *mockTransport) Descriptor() Descriptor {
	return Descriptor{Path: "mock"}
}

func (m *mockTransport) Open(parent Session, params map[string]any) (Session, error) {
	m.openCount++
	return &mockSession{}, nil
}

func (m *mockTransport) Wrap(next ExecNode, params map[string]any) ExecNode {
	return nil
}

type mockSession struct {
	closed bool
}

func (m *mockSession) Run(ctx context.Context, argv []string, opts RunOpts) (Result, error) {
	return Result{}, nil
}

func (m *mockSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (m *mockSession) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockSession) Env() map[string]string {
	return nil
}

func (m *mockSession) WithEnv(delta map[string]string) Session {
	return m
}

func (m *mockSession) WithWorkdir(dir string) Session {
	return m
}

func (m *mockSession) Cwd() string {
	return ""
}

// ID returns a mock session identifier for testing.
func (m *mockSession) ID() string {
	return "mock"
}

// TransportScope returns a mock transport scope for testing.
func (m *mockSession) TransportScope() TransportScope {
	return TransportScopeLocal
}

func (m *mockSession) Close() error {
	m.closed = true
	return nil
}
