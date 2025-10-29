package decorator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SessionPool manages session reuse across transport blocks.
// This is a runtime optimization - NOT part of the contract plan.
//
// Sessions are pooled by deterministic key based on transport and params.
// Same params → same session instance (connection reused).
type SessionPool struct {
	sessions map[string]Session
	mu       sync.Mutex
}

// NewSessionPool creates a new session pool.
func NewSessionPool() *SessionPool {
	return &SessionPool{
		sessions: make(map[string]Session),
	}
}

// GetOrCreate returns an existing session or creates a new one.
// Sessions are keyed by transport name and params hash.
//
// Thread-safe: Multiple goroutines can call this concurrently.
func (p *SessionPool) GetOrCreate(
	transport Transport,
	parent Session,
	params map[string]any,
) (Session, error) {
	// Create deterministic key
	key := sessionKey(transport.Descriptor().Path, params)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Return existing session if available
	if session, ok := p.sessions[key]; ok {
		return session, nil
	}

	// Create new session
	session, err := transport.Open(parent, params)
	if err != nil {
		return nil, err
	}

	// Cache for reuse
	p.sessions[key] = session
	return session, nil
}

// CloseAll closes all pooled sessions.
// Should be called when planning/execution completes.
func (p *SessionPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, session := range p.sessions {
		_ = session.Close() // Best effort cleanup
	}

	// Clear the pool
	p.sessions = make(map[string]Session)
}

// sessionKey generates a deterministic key for session pooling.
// Same transport + params → same key (regardless of param order).
func sessionKey(transport string, params map[string]any) string {
	// Sort keys for determinism
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical string
	var buf strings.Builder
	buf.WriteString(transport)
	buf.WriteString(":")
	for _, k := range keys {
		fmt.Fprintf(&buf, "%s=%v;", k, params[k])
	}

	// Hash for compact key
	h := sha256.Sum256([]byte(buf.String()))
	return hex.EncodeToString(h[:8]) // First 64 bits sufficient
}
