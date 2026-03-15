package plugin

import (
	"context"
	"io"
)

// Capability is the common metadata surface shared by all plugin capabilities.
type Capability interface {
	Path() string
	Schema() Schema
}

// ValueProvider resolves a value during planning.
type ValueProvider interface {
	Capability
	Resolve(ctx ValueContext, args ResolvedArgs) (any, error)
}

// BatchValueProvider optionally resolves multiple calls together.
// Hosts use this to preserve grouped planning behavior when available.
type BatchValueProvider interface {
	ValueProvider
	ResolveBatch(ctx ValueContext, args []ResolvedArgs) ([]any, error)
}

// Wrapper wraps execution of a child node.
type Wrapper interface {
	Capability
	Wrap(next ExecNode, args ResolvedArgs) ExecNode
}

// Transport opens a child execution transport.
//
// The host owns block execution, transport identity, and pooling. Plugins
// receive only resolved parameters and a narrow parent transport view; they do
// not inspect nested block contents.
type Transport interface {
	Capability
	Open(ctx context.Context, parent ParentTransport, args ResolvedArgs) (OpenedTransport, error)
}

// RedirectTarget supports I/O redirection (>, >>, <).
type RedirectTarget interface {
	Capability
	OpenForWrite(ctx ExecContext, args ResolvedArgs, appendMode bool) (io.WriteCloser, error)
	OpenForRead(ctx ExecContext, args ResolvedArgs) (io.ReadCloser, error)
	RedirectCaps() RedirectCaps
}

// RedirectCaps describes redirection capabilities.
type RedirectCaps struct {
	Read   bool
	Write  bool
	Append bool
	Atomic bool
}

// ExecNode is an executable runtime node.
type ExecNode interface {
	Execute(ctx ExecContext) (Result, error)
}

// BranchExecutor is an optional extension for block-aware execution nodes.
// It allows wrapper capabilities (for example parallel execution) to execute
// top-level branches independently.
type BranchExecutor interface {
	ExecNode
	BranchCount() int
	ExecuteBranch(index int, ctx ExecContext) (Result, error)
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
