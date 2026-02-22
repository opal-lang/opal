package decorator

// Transport is the interface for decorators that create transport boundaries.
// Transport decorators implement BOTH Open() and Wrap() methods.
// Examples: @ssh.connect, @docker.exec, @k8s.pod
type Transport interface {
	Decorator

	// Capabilities returns the transport capabilities this transport provides.
	Capabilities() TransportCaps

	// Open creates a new Session with the specified transport
	Open(parent Session, params map[string]any) (Session, error)

	// Wrap wraps execution to use the transport session
	Wrap(next ExecNode, params map[string]any) ExecNode
}

// TransportCaps defines transport capabilities as a bitset.
type TransportCaps uint32

const (
	// TransportCapNetwork indicates the transport can reach network endpoints.
	TransportCapNetwork TransportCaps = 1 << iota

	// TransportCapFilesystem indicates the transport can read/write filesystem data.
	TransportCapFilesystem

	// TransportCapEnvironment indicates the transport can modify execution environment variables.
	TransportCapEnvironment

	// TransportCapIsolation indicates the transport supports isolation guarantees.
	TransportCapIsolation
)

// Has reports whether the transport includes a given capability.
func (c TransportCaps) Has(cap TransportCaps) bool {
	return c&cap != 0
}
