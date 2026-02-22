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

	// MaterializeSession reports whether this transport creates a dedicated
	// session via Open() instead of using the default session factory.
	MaterializeSession() bool

	// IsolationContext returns the transport's isolation capabilities.
	// Returns nil if transport doesn't support isolation.
	IsolationContext() IsolationContext
}

// IsolationContext provides isolation capabilities for transports.
type IsolationContext interface {
	// Isolate applies the specified isolation level.
	Isolate(level IsolationLevel, config IsolationConfig) error

	// DropNetwork removes network access.
	DropNetwork() error

	// RestrictFilesystem limits filesystem access.
	RestrictFilesystem(readOnly, writable []string) error

	// DropPrivileges drops unnecessary privileges.
	DropPrivileges() error

	// LockMemory prevents memory from being swapped.
	LockMemory() error
}

// IsolationLevel defines the degree of isolation.
type IsolationLevel int

const (
	IsolationLevelNone     IsolationLevel = iota
	IsolationLevelBasic                   // Basic process isolation
	IsolationLevelStandard                // Linux namespaces (network, PID, mount)
	IsolationLevelMaximum                 // Full isolation + seccomp + landlock
)

// NetworkPolicy defines network isolation policy.
type NetworkPolicy int

const (
	NetworkPolicyAllow NetworkPolicy = iota
	NetworkPolicyDeny
	NetworkPolicyLoopbackOnly
)

// FilesystemPolicy defines filesystem isolation policy.
type FilesystemPolicy int

const (
	FilesystemPolicyFull FilesystemPolicy = iota
	FilesystemPolicyReadOnly
	FilesystemPolicyEphemeral
)

// IsolationConfig configures isolation behavior.
type IsolationConfig struct {
	NetworkPolicy    NetworkPolicy
	FilesystemPolicy FilesystemPolicy
	MemoryLock       bool
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
func (c TransportCaps) Has(capability TransportCaps) bool {
	return c&capability != 0
}
