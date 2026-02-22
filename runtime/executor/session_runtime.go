package executor

import (
	"fmt"
	"strings"
	"sync"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
)

type sessionFactory func(transportID string) (decorator.Session, error)

type sessionRuntime struct {
	mu         sync.Mutex
	sessions   map[string]decorator.Session
	transports map[string]planfmt.Transport
	factory    sessionFactory
}

func newSessionRuntime(factory sessionFactory) *sessionRuntime {
	if factory == nil {
		factory = defaultSessionFactory
	}

	return &sessionRuntime{
		sessions:   make(map[string]decorator.Session),
		transports: make(map[string]planfmt.Transport),
		factory:    factory,
	}
}

func (r *sessionRuntime) registerPlanTransports(transports []planfmt.Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.transports = make(map[string]planfmt.Transport, len(transports))
	for _, transport := range transports {
		if transport.ID == "" {
			continue
		}
		r.transports[normalizedTransportID(transport.ID)] = transport
	}
}

func defaultSessionFactory(transportID string) (decorator.Session, error) {
	// Factory is only called for local transports (either literal "local" or derived IDs).
	// Create a LocalSession regardless of the specific transport ID format.
	return decorator.NewLocalSession(), nil
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
	if session, ok := r.sessions[key]; ok {
		r.mu.Unlock()
		return session, nil
	}
	r.mu.Unlock()

	session, err := r.createSession(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create session for transport %q: %w", key, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.sessions[key]; ok {
		return existing, nil
	}
	r.sessions[key] = session
	return session, nil
}

func (r *sessionRuntime) createSession(transportID string) (decorator.Session, error) {
	transport, ok := r.lookupTransport(transportID)
	if !ok {
		if transportID == "local" {
			return r.factory(transportID)
		}

		return nil, fmt.Errorf("unknown transport %q: transport not registered", transportID)
	}

	if normalizedTransportID(transport.Decorator) == "local" {
		return r.factory(transportID)
	}

	name := strings.TrimPrefix(transport.Decorator, "@")
	entry, exists := decorator.Global().Lookup(name)
	if !exists {
		return nil, fmt.Errorf("unknown transport decorator %q", transport.Decorator)
	}

	transportDecorator, ok := entry.Impl.(decorator.Transport)
	if !ok {
		return nil, fmt.Errorf("decorator %q is not a transport", transport.Decorator)
	}

	if !isTransportSessionMaterialized(name) {
		return r.factory(transportID)
	}

	parentID := normalizedTransportID(transport.ParentID)
	if parentID == "" {
		parentID = "local"
	}
	parentSession, err := r.SessionFor(parentID)
	if err != nil {
		return nil, err
	}

	openedSession, err := transportDecorator.Open(parentSession, planArgsToMap(transport.Args))
	if err != nil {
		return nil, fmt.Errorf("open transport %q: %w", transport.Decorator, err)
	}

	return &transportScopedSession{id: transportID, session: openedSession}, nil
}

func (r *sessionRuntime) lookupTransport(transportID string) (planfmt.Transport, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	transport, ok := r.transports[normalizedTransportID(transportID)]
	return transport, ok
}

func isTransportSessionMaterialized(name string) bool {
	entry, exists := decorator.Global().Lookup(name)
	if !exists {
		return false
	}

	transport, ok := entry.Impl.(decorator.Transport)
	if !ok {
		return false
	}

	return transport.MaterializeSession()
}

func (r *sessionRuntime) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, session := range r.sessions {
		_ = session.Close()
	}

	r.sessions = make(map[string]decorator.Session)
}
