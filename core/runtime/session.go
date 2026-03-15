package runtime

import "github.com/builtwithtofu/sigil/core/decorator"

// Session is the canonical runtime execution context interface.
type Session = decorator.Session

// NetworkDialer is the optional network dial interface exposed by sessions.
type NetworkDialer = decorator.NetworkDialer

// NetworkDialerProvider exposes network dial capability for wrapped sessions.
type NetworkDialerProvider = decorator.NetworkDialerProvider

// TransportScope defines where a session can execute.
type TransportScope = decorator.TransportScope

const (
	TransportScopeAny = decorator.TransportScopeAny
)

// GetNetworkDialer resolves a dialer through wrapped sessions.
func GetNetworkDialer(session Session) (NetworkDialer, error) {
	return decorator.GetNetworkDialer(session)
}

// NewLocalSession creates a local host execution session.
func NewLocalSession() Session {
	return decorator.NewLocalSession()
}
