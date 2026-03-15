package plugin

import (
	"context"
	"io"
	"time"
)

// ValueContext provides plan-time resolution state.
type ValueContext interface {
	Context() context.Context
	Session() ParentTransport
	PlanHash() string
	LookupValue(name string) (any, bool)
}

// ExecContext provides runtime execution state.
type ExecContext interface {
	Context() context.Context
	Session() ParentTransport
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
}

// ResolvedArgs provides typed access to resolved capability arguments.
//
// ResolveSecret must reject access to secrets not declared by the capability's
// schema.
type ResolvedArgs interface {
	GetString(name string) string
	GetStringOptional(name string) string
	GetInt(name string) int
	GetDuration(name string) time.Duration
	ResolveSecret(name string) (string, error)
}
