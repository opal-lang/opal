package plugin

import (
	"context"
	"io"
	"io/fs"
	"net"
)

// RunOpts configures command execution for plugin sessions.
type RunOpts struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
}

// SessionSnapshot is the host-visible execution state exposed by a transport.
type SessionSnapshot struct {
	Env      map[string]string
	Workdir  string
	Platform string
}

// ParentTransport is the narrow host-controlled parent execution context exposed
// to value, wrapper, and transport capabilities.
type ParentTransport interface {
	Run(ctx context.Context, argv []string, opts RunOpts) (Result, error)
	Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error
	Get(ctx context.Context, path string) ([]byte, error)
	Snapshot() SessionSnapshot
	Close() error
}

// NetworkDialer opens outbound network connections through a parent transport.
// Nested transports use this for bastion/IAP-style double-hop connections.
type NetworkDialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// NetworkDialerProvider optionally exposes parent-routed dialing.
type NetworkDialerProvider interface {
	NetworkDialer() NetworkDialer
}

// ParentTransportWrapper optionally exposes an underlying parent transport.
// Wrappers that adjust snapshot state (for example workdir) should implement
// this so hosts can recover base transport identity when needed.
type ParentTransportWrapper interface {
	UnwrapParentTransport() ParentTransport
}

// OpenedTransport is the narrow runtime object returned by a transport capability.
// The host wraps it with transport identity, pooling, and boundary enforcement.
type OpenedTransport interface {
	Run(ctx context.Context, argv []string, opts RunOpts) (Result, error)
	Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error
	Get(ctx context.Context, path string) ([]byte, error)
	Snapshot() SessionSnapshot
	WithSnapshot(snapshot SessionSnapshot) OpenedTransport
	Close() error
}
