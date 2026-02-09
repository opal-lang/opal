package executor

import (
	"fmt"
	"sync"

	"github.com/opal-lang/opal/core/decorator"
)

type sessionFactory func(transportID string) (decorator.Session, error)

type sessionRuntime struct {
	mu       sync.Mutex
	sessions map[string]decorator.Session
	factory  sessionFactory
}

func newSessionRuntime(factory sessionFactory) *sessionRuntime {
	if factory == nil {
		factory = defaultSessionFactory
	}

	return &sessionRuntime{
		sessions: make(map[string]decorator.Session),
		factory:  factory,
	}
}

func defaultSessionFactory(transportID string) (decorator.Session, error) {
	session := decorator.NewLocalSession()
	if normalizedTransportID(transportID) == "local" {
		return session, nil
	}

	return &transportScopedSession{id: normalizedTransportID(transportID), session: session}, nil
}

func normalizedTransportID(transportID string) string {
	if transportID == "" {
		return "local"
	}
	return transportID
}

func (r *sessionRuntime) SessionFor(transportID string) (decorator.Session, error) {
	key := normalizedTransportID(transportID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if session, ok := r.sessions[key]; ok {
		return session, nil
	}

	session, err := r.factory(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create session for transport %q: %w", key, err)
	}

	r.sessions[key] = session
	return session, nil
}

func (r *sessionRuntime) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, session := range r.sessions {
		_ = session.Close()
	}

	r.sessions = make(map[string]decorator.Session)
}
