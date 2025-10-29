package types

// Role represents behavioral capabilities of a decorator.
// Decorators can have multiple roles (e.g., @aws.s3.object is both Provider and Endpoint).
// Roles are auto-inferred from implemented interfaces.
type Role string

const (
	// RoleProvider produces data (@var, @env, @aws.secret)
	RoleProvider Role = "provider"

	// RoleWrapper wraps execution (@retry, @timeout)
	RoleWrapper Role = "wrapper"

	// RoleBoundary creates scoped context (@ssh.connect, @docker.exec)
	RoleBoundary Role = "boundary"

	// RoleEndpoint reads/writes data (@file.read, @s3.put)
	RoleEndpoint Role = "endpoint"

	// RoleAnnotate augments plan metadata (@trace, @measure)
	RoleAnnotate Role = "annotate"
)

// Decorator is the base interface all decorators must implement.
// It provides reflectable metadata for LSP, CLI, docs, and telemetry.
type Decorator interface {
	Descriptor() Descriptor
}

// Descriptor holds rich metadata about a decorator.
// This is the single source of truth for validation, documentation, and tooling.
type Descriptor struct {
	// Path is the decorator's full path (e.g., "env", "retry", "aws.s3.object")
	Path string

	// Roles are behavioral capabilities (auto-inferred from implemented interfaces)
	// A decorator can have multiple roles (e.g., @aws.s3.object is both Provider and Endpoint)
	Roles []Role

	// Version is the decorator version (semver string)
	Version string

	// Summary is a one-line description
	Summary string

	// DocURL links to full documentation
	DocURL string

	// Schema describes parameters and return type
	Schema DecoratorSchema

	// Capabilities define execution constraints and properties
	Capabilities Capabilities
}

// Capabilities define execution constraints and properties.
type Capabilities struct {
	// TransportScope defines where this decorator can be used
	TransportScope TransportScope

	// Purity indicates if the decorator is deterministic (can be cached/constant-folded)
	// Default: false (safe default - assume side effects)
	Purity bool

	// Idempotent indicates if the decorator is safe to retry
	// Default: false (safe default - assume not idempotent)
	Idempotent bool

	// IO describes I/O behavior for pipe and redirect operators
	IO IOSemantics
}

// IOSemantics describes I/O capabilities for decorators.
// This is a simplified v1.0 model - decorators are inherently concurrency-safe
// by design (pure/monadic), so no ConcurrentSafe flag is needed.
type IOSemantics struct {
	// PipeIn indicates decorator can read from stdin (supports: cmd | @decorator)
	PipeIn bool

	// PipeOut indicates decorator can write to stdout (supports: @decorator | cmd)
	PipeOut bool

	// RedirectIn indicates decorator can read from file (supports: @decorator < file)
	RedirectIn bool

	// RedirectOut indicates decorator can write to file (supports: cmd > @decorator)
	RedirectOut bool
}
