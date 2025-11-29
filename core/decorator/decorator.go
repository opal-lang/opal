package decorator

import "github.com/opal-lang/opal/core/types"

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

	// Schema describes parameters and return type (single source of truth)
	Schema types.DecoratorSchema

	// Capabilities define execution constraints and properties
	Capabilities Capabilities
}

// TransportScope defines where a decorator can be used.
type TransportScope int

const (
	// TransportScopeAny means decorator works in any transport (local, SSH, Docker, etc.)
	TransportScopeAny TransportScope = 0

	// TransportScopeLocal means decorator only works in local transport
	TransportScopeLocal TransportScope = 1

	// TransportScopeSSH means decorator only works in SSH transport
	TransportScopeSSH TransportScope = 2

	// TransportScopeRemote means decorator works in any remote transport (SSH, Docker, etc.)
	TransportScopeRemote TransportScope = 3
)

// String returns the string representation of TransportScope.
func (s TransportScope) String() string {
	switch s {
	case TransportScopeAny:
		return "Any"
	case TransportScopeLocal:
		return "Local"
	case TransportScopeSSH:
		return "SSH"
	case TransportScopeRemote:
		return "Remote"
	default:
		return "Unknown"
	}
}

// Allows checks if the decorator's transport scope allows execution in the current scope.
func (s TransportScope) Allows(current TransportScope) bool {
	// Any scope works everywhere
	if s == TransportScopeAny {
		return true
	}

	// Remote scope allows any remote transport (SSH, Docker, etc.)
	if s == TransportScopeRemote {
		return current == TransportScopeSSH || current == TransportScopeRemote
	}

	// Otherwise, exact match required
	return s == current
}

// Capabilities define execution constraints and properties.
type Capabilities struct {
	// TransportScope defines where this decorator can be used
	TransportScope TransportScope

	// TransportSensitive marks values as tied to the transport where they were resolved.
	// When true, values cannot cross transport boundaries (e.g., local to SSH).
	// Default: false
	TransportSensitive bool

	// Purity indicates if the decorator is deterministic (can be cached/constant-folded)
	// Default: false (safe default - assume side effects)
	Purity bool

	// Idempotent indicates if the decorator is safe to retry
	// Default: false (safe default - assume not idempotent)
	Idempotent bool

	// Block specifies whether decorator accepts/requires a block
	// Default: BlockForbidden (safe default for value decorators)
	Block BlockRequirement

	// IO describes I/O behavior for pipe and redirect operators
	IO IOSemantics
}

// BlockRequirement specifies whether a decorator accepts/requires a block
type BlockRequirement string

const (
	// BlockForbidden means decorator cannot have a block (value decorators like @var, @env)
	BlockForbidden BlockRequirement = "forbidden"

	// BlockOptional means decorator can optionally have a block (e.g., @retry with/without block)
	BlockOptional BlockRequirement = "optional"

	// BlockRequired means decorator must have a block (e.g., @parallel, @timeout)
	BlockRequired BlockRequirement = "required"
)

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

// DescriptorBuilder provides a fluent API for building Descriptor without duplication.
// Only requires parameters and returns - Path, Summary, Roles are set directly.
type DescriptorBuilder struct {
	desc Descriptor
}

// NewDescriptor creates a new descriptor builder.
func NewDescriptor(path string) *DescriptorBuilder {
	return &DescriptorBuilder{
		desc: Descriptor{
			Path:   path,
			Schema: types.DecoratorSchema{Parameters: make(map[string]types.ParamSchema)},
		},
	}
}

// Summary sets the one-line description.
func (b *DescriptorBuilder) Summary(summary string) *DescriptorBuilder {
	b.desc.Summary = summary
	return b
}

// Returns sets the return type (for value decorators).
func (b *DescriptorBuilder) Returns(typ types.ParamType, description string) *DescriptorBuilder {
	b.desc.Schema.Returns = &types.ReturnSchema{
		Type:        typ,
		Description: description,
	}
	return b
}

// TransportScope sets where the decorator can be used.
func (b *DescriptorBuilder) TransportScope(scope TransportScope) *DescriptorBuilder {
	b.desc.Capabilities.TransportScope = scope
	return b
}

// TransportSensitive marks values as tied to the transport where they were resolved.
func (b *DescriptorBuilder) TransportSensitive() *DescriptorBuilder {
	b.desc.Capabilities.TransportSensitive = true
	return b
}

// Pure marks the decorator as deterministic (can be cached/constant-folded).
func (b *DescriptorBuilder) Pure() *DescriptorBuilder {
	b.desc.Capabilities.Purity = true
	return b
}

// Idempotent marks the decorator as safe to retry.
func (b *DescriptorBuilder) Idempotent() *DescriptorBuilder {
	b.desc.Capabilities.Idempotent = true
	return b
}

// Block sets the block requirement.
func (b *DescriptorBuilder) Block(req BlockRequirement) *DescriptorBuilder {
	b.desc.Capabilities.Block = req
	return b
}

// Roles sets the decorator roles (auto-inferred by registry, but can be set explicitly).
func (b *DescriptorBuilder) Roles(roles ...Role) *DescriptorBuilder {
	b.desc.Roles = roles
	return b
}

// Build validates the descriptor and returns it.
func (b *DescriptorBuilder) Build() Descriptor {
	// Ensure primary parameter is first in order if set
	if b.desc.Schema.PrimaryParameter != "" {
		// Remove primary parameter from order if it exists
		newOrder := make([]string, 0, len(b.desc.Schema.ParameterOrder))
		for _, name := range b.desc.Schema.ParameterOrder {
			if name != b.desc.Schema.PrimaryParameter {
				newOrder = append(newOrder, name)
			}
		}
		// Prepend primary parameter
		b.desc.Schema.ParameterOrder = append([]string{b.desc.Schema.PrimaryParameter}, newOrder...)
	}

	return b.desc
}
