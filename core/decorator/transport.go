package decorator

// Transport is the interface for decorators that create transport boundaries.
// Transport decorators implement BOTH Open() and Wrap() methods.
// Examples: @ssh.connect, @docker.exec, @k8s.pod
type Transport interface {
	Decorator

	// Open creates a new Session with the specified transport
	Open(parent Session, params map[string]any) (Session, error)

	// Wrap wraps execution to use the transport session
	Wrap(next ExecNode, params map[string]any) ExecNode
}
