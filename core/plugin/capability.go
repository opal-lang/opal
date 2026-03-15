package plugin

import "context"

// CapabilityKind classifies a plugin capability.
type CapabilityKind string

const (
	KindValue     CapabilityKind = "value"
	KindWrapper   CapabilityKind = "wrapper"
	KindTransport CapabilityKind = "transport"
)

// Capability is the common metadata surface shared by all plugin capabilities.
type Capability interface {
	Kind() CapabilityKind
	Path() string
	Schema() Schema
}

// ValueCapability resolves a value during planning.
type ValueCapability interface {
	Capability
	Resolve(ctx ValueContext, args ResolvedArgs) (string, error)
}

// WrapperCapability wraps execution of a child node.
type WrapperCapability interface {
	Capability
	Wrap(next ExecNode, args ResolvedArgs) ExecNode
}

// TransportCapability opens a child execution transport.
//
// The host owns block execution, transport identity, and pooling. Plugins
// receive only resolved parameters and a narrow parent transport view; they do
// not inspect nested block contents.
type TransportCapability interface {
	Capability
	Open(ctx context.Context, parent ParentTransport, args ResolvedArgs) (OpenedTransport, error)
}

// ExecNode is an executable runtime node.
type ExecNode interface {
	Execute(ctx ExecContext) (Result, error)
}

// Result is the outcome of execution.
type Result struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

const (
	ExitSuccess  = 0
	ExitCanceled = -1
	ExitFailure  = 1
)
