package decorator

import (
	"context"
	"io/fs"
	"sync"
)

// MonitoredSession wraps a Session to track method calls for testing.
// This is a test helper that doesn't change behavior, just observes it.
type MonitoredSession struct {
	wrapped Session
	stats   *SessionStats
}

// SessionStats tracks session method calls for testing.
type SessionStats struct {
	mu sync.Mutex

	RunCalls     int
	PutCalls     int
	GetCalls     int
	EnvCalls     int
	WithEnvCalls int
	CwdCalls     int
	CloseCalls   int

	// Track WithEnv deltas to verify isolation
	WithEnvDeltas []map[string]string
}

// NewMonitoredSession wraps a session with monitoring.
func NewMonitoredSession(session Session) *MonitoredSession {
	return &MonitoredSession{
		wrapped: session,
		stats:   &SessionStats{},
	}
}

// Stats returns the accumulated statistics.
func (m *MonitoredSession) Stats() *SessionStats {
	return m.stats
}

// Session interface implementation (delegates to wrapped session)

func (m *MonitoredSession) Run(ctx context.Context, argv []string, opts RunOpts) (Result, error) {
	m.stats.mu.Lock()
	m.stats.RunCalls++
	m.stats.mu.Unlock()
	return m.wrapped.Run(ctx, argv, opts)
}

func (m *MonitoredSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	m.stats.mu.Lock()
	m.stats.PutCalls++
	m.stats.mu.Unlock()
	return m.wrapped.Put(ctx, data, path, mode)
}

func (m *MonitoredSession) Get(ctx context.Context, path string) ([]byte, error) {
	m.stats.mu.Lock()
	m.stats.GetCalls++
	m.stats.mu.Unlock()
	return m.wrapped.Get(ctx, path)
}

func (m *MonitoredSession) Env() map[string]string {
	m.stats.mu.Lock()
	m.stats.EnvCalls++
	m.stats.mu.Unlock()
	return m.wrapped.Env()
}

func (m *MonitoredSession) WithEnv(delta map[string]string) Session {
	m.stats.mu.Lock()
	m.stats.WithEnvCalls++
	// Deep copy delta for inspection
	deltaCopy := make(map[string]string, len(delta))
	for k, v := range delta {
		deltaCopy[k] = v
	}
	m.stats.WithEnvDeltas = append(m.stats.WithEnvDeltas, deltaCopy)
	m.stats.mu.Unlock()

	// Wrap the returned session too
	newSession := m.wrapped.WithEnv(delta)
	return &MonitoredSession{
		wrapped: newSession,
		stats:   m.stats, // Share stats with parent
	}
}

func (m *MonitoredSession) WithWorkdir(dir string) Session {
	// Don't track WithWorkdir calls for now
	newSession := m.wrapped.WithWorkdir(dir)
	return &MonitoredSession{
		wrapped: newSession,
		stats:   m.stats, // Share stats with parent
	}
}

func (m *MonitoredSession) Cwd() string {
	m.stats.mu.Lock()
	m.stats.CwdCalls++
	m.stats.mu.Unlock()
	return m.wrapped.Cwd()
}

// ID returns the session identifier, delegating to the wrapped session.
func (m *MonitoredSession) ID() string {
	return m.wrapped.ID()
}

// TransportScope returns the transport scope, delegating to the wrapped session.
func (m *MonitoredSession) TransportScope() TransportScope {
	return m.wrapped.TransportScope()
}

func (m *MonitoredSession) Close() error {
	m.stats.mu.Lock()
	m.stats.CloseCalls++
	m.stats.mu.Unlock()
	return m.wrapped.Close()
}

// MonitoredTransport wraps a Transport to track Open calls.
type MonitoredTransport struct {
	wrapped   Transport
	OpenCalls int
	mu        sync.Mutex
}

// NewMonitoredTransport wraps a transport with monitoring.
func NewMonitoredTransport(transport Transport) *MonitoredTransport {
	return &MonitoredTransport{
		wrapped: transport,
	}
}

func (m *MonitoredTransport) Descriptor() Descriptor {
	return m.wrapped.Descriptor()
}

func (m *MonitoredTransport) Open(parent Session, params map[string]any) (Session, error) {
	m.mu.Lock()
	m.OpenCalls++
	m.mu.Unlock()

	session, err := m.wrapped.Open(parent, params)
	if err != nil {
		return nil, err
	}

	// Wrap returned session for monitoring
	return NewMonitoredSession(session), nil
}

func (m *MonitoredTransport) Wrap(next ExecNode, params map[string]any) ExecNode {
	return m.wrapped.Wrap(next, params)
}
