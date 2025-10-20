package types

import "fmt"

// ParamType represents the type of a parameter
type ParamType string

const (
	TypeString   ParamType = "string"
	TypeInt      ParamType = "integer"
	TypeFloat    ParamType = "float"
	TypeBool     ParamType = "boolean"
	TypeDuration ParamType = "duration"
	TypeObject   ParamType = "object"
	TypeArray    ParamType = "array"

	// Custom types for handles and references
	TypeAuthHandle   ParamType = "AuthHandle"
	TypeSecretHandle ParamType = "SecretHandle"

	// Enum type for scrub parameter
	TypeScrubMode ParamType = "ScrubMode"
)

// ScrubMode represents scrubbing behavior for I/O decorators
type ScrubMode string

const (
	ScrubNone   ScrubMode = "none"   // No scrubbing (raw data, bash-compatible)
	ScrubStdin  ScrubMode = "stdin"  // Scrub only stdin
	ScrubStdout ScrubMode = "stdout" // Scrub only stdout
	ScrubBoth   ScrubMode = "both"   // Scrub both stdin and stdout
)

// DecoratorKindString represents decorator kind as string
type DecoratorKindString string

const (
	KindValue     DecoratorKindString = "value"
	KindExecution DecoratorKindString = "execution"
)

// BlockRequirement specifies whether a decorator accepts/requires a block
type BlockRequirement string

const (
	BlockForbidden BlockRequirement = "forbidden" // Cannot have block (value decorators, @shell)
	BlockOptional  BlockRequirement = "optional"  // Can have block (@retry with/without block)
	BlockRequired  BlockRequirement = "required"  // Must have block (@parallel, @timeout)
)

// IOFlag represents I/O capabilities for decorators (used with WithIO)
type IOFlag int

const (
	// AcceptsStdin indicates this decorator can read from stdin (piped input).
	// Parser will allow: cmd | @decorator
	// Decorator receives ctx.Stdin() when used after pipe operator.
	AcceptsStdin IOFlag = 1 << iota

	// ProducesStdout indicates this decorator can write to stdout (piped output).
	// Parser will allow: @decorator | cmd
	// Decorator receives ctx.StdoutPipe() when used before pipe operator.
	ProducesStdout

	// ScrubByDefault enables secret scrubbing by default (recommended).
	// Sets default scrub mode based on I/O capabilities:
	//   - Both stdin+stdout: default="both"
	//   - Stdin only: default="stdin"
	//   - Stdout only: default="stdout"
	// Omit this flag for binary/non-text decorators (default="none").
	ScrubByDefault
)

// IOOpts provides fine-grained control over I/O capabilities and scrubbing defaults.
// Use this when you need to set a specific default scrub mode.
type IOOpts struct {
	// AcceptsStdin indicates this decorator can read from stdin
	AcceptsStdin bool

	// ProducesStdout indicates this decorator can write to stdout
	ProducesStdout bool

	// DefaultScrubMode sets the default scrubbing behavior
	// If empty, defaults to ScrubNone (bash-compatible)
	DefaultScrubMode ScrubMode
}

// IOCapability describes a decorator's I/O capabilities for pipe operator support.
//
// Decorators that don't interact with stdin/stdout should leave this nil.
// Only decorators that read from stdin or write to stdout need to declare I/O capabilities.
//
// Use WithIO() with IOFlag constants to declare capabilities.
type IOCapability struct {
	// SupportsStdin indicates this decorator can read from stdin (piped input).
	// If true, the decorator will receive ctx.Stdin() when used after pipe operator.
	// Parser will allow: cmd | @decorator
	SupportsStdin bool

	// SupportsStdout indicates this decorator can write to stdout (piped output).
	// If true, the decorator will receive ctx.StdoutPipe() when used before pipe operator.
	// Parser will allow: @decorator | cmd
	SupportsStdout bool

	// DefaultScrub is the default scrubbing behavior for this decorator.
	// - true: Scrub secrets by default (safe, recommended for most decorators)
	// - false: Pass raw data by default (use for binary/non-text decorators)
	// Users can override with scrub=true/false parameter.
	DefaultScrub bool
}

// DecoratorSchema describes a decorator's interface
type DecoratorSchema struct {
	Path             string                 // "env", "aws.secret"
	Kind             DecoratorKindString    // "value" or "execution"
	Description      string                 // Human-readable description
	PrimaryParameter string                 // Name of primary param ("property", "secretName"), empty if none
	Parameters       map[string]ParamSchema // All parameters (including primary)
	ParameterOrder   []string               // Order of parameter declaration (for positional mapping)
	Returns          *ReturnSchema          // What the decorator returns (value decorators only)
	BlockRequirement BlockRequirement       // Whether decorator accepts/requires a block
	IO               *IOCapability          // I/O capabilities for pipe operator (nil = no I/O)
}

// ParamSchema describes a single parameter
type ParamSchema struct {
	Name        string      // Parameter name
	Type        ParamType   // Type of the parameter
	Description string      // Human-readable description
	Required    bool        // Whether parameter is required
	Default     interface{} // Default value if not provided
	Examples    []string    // Example values

	// Validation (future use)
	Minimum *int     // For int types
	Maximum *int     // For int types
	Enum    []string // Allowed values
	Pattern string   // Regex pattern for string validation
}

// ReturnSchema describes what a value decorator returns
type ReturnSchema struct {
	Type        ParamType              // Type of the return value
	Description string                 // What is returned
	Properties  map[string]ParamSchema // For object returns
}

// SchemaBuilder provides fluent API for building schemas
type SchemaBuilder struct {
	schema         DecoratorSchema
	parameterOrder []string // Track parameter declaration order
}

// NewSchema creates a new schema builder
func NewSchema(path string, kind DecoratorKindString) *SchemaBuilder {
	// Default block requirement based on kind
	blockReq := BlockForbidden
	if kind == KindExecution {
		blockReq = BlockOptional // Execution decorators can optionally have blocks by default
	}

	return &SchemaBuilder{
		schema: DecoratorSchema{
			Path:             path,
			Kind:             kind,
			Parameters:       make(map[string]ParamSchema),
			BlockRequirement: blockReq,
		},
	}
}

// Description sets the decorator description
func (b *SchemaBuilder) Description(desc string) *SchemaBuilder {
	b.schema.Description = desc
	return b
}

// PrimaryParam defines the primary parameter (enables dot syntax)
func (b *SchemaBuilder) PrimaryParam(name string, typ ParamType, description string) *SchemaBuilder {
	b.schema.PrimaryParameter = name
	b.schema.Parameters[name] = ParamSchema{
		Name:        name,
		Type:        typ,
		Description: description,
		Required:    true, // Primary params are always required
	}
	// Track parameter order
	b.parameterOrder = append(b.parameterOrder, name)
	return b
}

// Param adds a named parameter and returns a ParamBuilder
func (b *SchemaBuilder) Param(name string, typ ParamType) *ParamBuilder {
	return &ParamBuilder{
		schemaBuilder: b,
		param: ParamSchema{
			Name: name,
			Type: typ,
		},
	}
}

// Returns sets the return type (for value decorators)
func (b *SchemaBuilder) Returns(typ ParamType, description string) *SchemaBuilder {
	b.schema.Returns = &ReturnSchema{
		Type:        typ,
		Description: description,
	}
	return b
}

// WithBlock sets the block requirement for this decorator
func (b *SchemaBuilder) WithBlock(requirement BlockRequirement) *SchemaBuilder {
	b.schema.BlockRequirement = requirement
	return b
}

// AcceptsBlock marks that this decorator accepts an optional block (deprecated: use WithBlock)
func (b *SchemaBuilder) AcceptsBlock() *SchemaBuilder {
	b.schema.BlockRequirement = BlockOptional
	return b
}

// RequiresBlock marks that this decorator requires a block
func (b *SchemaBuilder) RequiresBlock() *SchemaBuilder {
	b.schema.BlockRequirement = BlockRequired
	return b
}

// WithIO declares I/O capabilities for pipe operator support.
//
// Only call this for decorators that interact with stdin/stdout.
// Decorators that wrap execution (@retry, @timeout) should NOT call this.
//
// Automatically adds a "scrub" parameter (TypeScrubMode) with default based on flags or opts.
//
// Use IOFlag constants (can be combined in any order):
//   - AcceptsStdin: Can read from stdin (allows: cmd | @decorator)
//   - ProducesStdout: Can write to stdout (allows: @decorator | cmd)
//   - ScrubByDefault: Scrub secrets by default (recommended for text data)
//
// The scrub parameter accepts: "none", "stdin", "stdout", "both"
//   - Default is "none" (bash-compatible) unless ScrubByDefault is set
//   - ScrubByDefault sets default based on I/O capabilities
//
// Examples:
//
//	// @shell: reads stdin, writes stdout, no scrubbing by default (bash-compatible)
//	WithIO(AcceptsStdin, ProducesStdout)
//	// Automatically adds: scrub parameter with default="none"
//	// Usage: @shell("grep pass", scrub="both") to enable scrubbing
//
//	// @file.write: reads stdin only, scrubs by default
//	WithIO(AcceptsStdin, ScrubByDefault)
//	// Automatically adds: scrub parameter with default="stdin"
//
//	// @http.get: writes stdout only, scrubs by default
//	WithIO(ProducesStdout, ScrubByDefault)
//	// Automatically adds: scrub parameter with default="stdout"
//
//	// @shell with scrubbing: reads stdin, writes stdout, scrubs by default
//	WithIO(AcceptsStdin, ProducesStdout, ScrubByDefault)
//	// Automatically adds: scrub parameter with default="both"
//
//	// Order doesn't matter
//	WithIO(ScrubByDefault, AcceptsStdin, ProducesStdout)  // Same as above
func (b *SchemaBuilder) WithIO(flags ...IOFlag) *SchemaBuilder {
	capability := &IOCapability{}

	// Process flags in any order
	for _, flag := range flags {
		switch flag {
		case AcceptsStdin:
			capability.SupportsStdin = true
		case ProducesStdout:
			capability.SupportsStdout = true
		case ScrubByDefault:
			capability.DefaultScrub = true
		}
	}

	b.schema.IO = capability

	// Determine default scrub mode based on capabilities and ScrubByDefault flag
	var defaultScrubMode ScrubMode
	if capability.DefaultScrub {
		// ScrubByDefault flag set - scrub based on what we support
		if capability.SupportsStdin && capability.SupportsStdout {
			defaultScrubMode = ScrubBoth
		} else if capability.SupportsStdin {
			defaultScrubMode = ScrubStdin
		} else if capability.SupportsStdout {
			defaultScrubMode = ScrubStdout
		} else {
			defaultScrubMode = ScrubNone
		}
	} else {
		// No ScrubByDefault - raw data (bash-compatible)
		defaultScrubMode = ScrubNone
	}

	// Automatically add "scrub" parameter with enum type
	b.schema.Parameters["scrub"] = ParamSchema{
		Name:        "scrub",
		Type:        TypeScrubMode,
		Description: "Scrub secrets from stdin/stdout",
		Required:    false,
		Default:     string(defaultScrubMode),
		Enum:        []string{string(ScrubNone), string(ScrubStdin), string(ScrubStdout), string(ScrubBoth)},
	}
	// Add to parameter order
	b.parameterOrder = append(b.parameterOrder, "scrub")

	return b
}

// WithIOOpts declares I/O capabilities with fine-grained control over defaults.
//
// Use this when you need to set a specific default scrub mode that doesn't match
// the automatic behavior of WithIO().
//
// Examples:
//
//	// @log.write: writes stdout, but scrub stdin by default (unusual case)
//	WithIOOpts(IOOpts{
//	    AcceptsStdin:     true,
//	    ProducesStdout:   true,
//	    DefaultScrubMode: ScrubStdin,  // Only scrub stdin, not stdout
//	})
//
//	// @binary.encode: I/O but never scrub by default
//	WithIOOpts(IOOpts{
//	    AcceptsStdin:     true,
//	    ProducesStdout:   true,
//	    DefaultScrubMode: ScrubNone,  // Explicit: never scrub
//	})
func (b *SchemaBuilder) WithIOOpts(opts IOOpts) *SchemaBuilder {
	capability := &IOCapability{
		SupportsStdin:  opts.AcceptsStdin,
		SupportsStdout: opts.ProducesStdout,
		DefaultScrub:   opts.DefaultScrubMode != ScrubNone,
	}

	b.schema.IO = capability

	// Use explicit default scrub mode from opts
	defaultScrubMode := opts.DefaultScrubMode
	if defaultScrubMode == "" {
		defaultScrubMode = ScrubNone
	}

	// Automatically add "scrub" parameter with enum type
	b.schema.Parameters["scrub"] = ParamSchema{
		Name:        "scrub",
		Type:        TypeScrubMode,
		Description: "Scrub secrets from stdin/stdout",
		Required:    false,
		Default:     string(defaultScrubMode),
		Enum:        []string{string(ScrubNone), string(ScrubStdin), string(ScrubStdout), string(ScrubBoth)},
	}
	// Add to parameter order
	b.parameterOrder = append(b.parameterOrder, "scrub")

	return b
}

// Build returns the constructed schema
func (b *SchemaBuilder) Build() DecoratorSchema {
	// Copy parameter order to schema
	b.schema.ParameterOrder = b.parameterOrder
	return b.schema
}

// ParamBuilder provides fluent API for building parameters
type ParamBuilder struct {
	schemaBuilder *SchemaBuilder
	param         ParamSchema
}

// Description sets parameter description
func (pb *ParamBuilder) Description(desc string) *ParamBuilder {
	pb.param.Description = desc
	return pb
}

// Required marks parameter as required
func (pb *ParamBuilder) Required() *ParamBuilder {
	pb.param.Required = true
	return pb
}

// Optional marks parameter as optional
func (pb *ParamBuilder) Optional() *ParamBuilder {
	pb.param.Required = false
	return pb
}

// Default sets default value
func (pb *ParamBuilder) Default(val interface{}) *ParamBuilder {
	pb.param.Default = val
	pb.param.Required = false // Has default = optional
	return pb
}

// Examples adds example values
func (pb *ParamBuilder) Examples(examples ...string) *ParamBuilder {
	pb.param.Examples = examples
	return pb
}

// Done finishes building this parameter and returns to schema builder
func (pb *ParamBuilder) Done() *SchemaBuilder {
	pb.schemaBuilder.schema.Parameters[pb.param.Name] = pb.param
	// Track parameter order
	pb.schemaBuilder.parameterOrder = append(pb.schemaBuilder.parameterOrder, pb.param.Name)
	return pb.schemaBuilder
}

// ValidateSchema validates a decorator schema
func ValidateSchema(schema DecoratorSchema) error {
	if schema.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if schema.Kind != KindValue && schema.Kind != KindExecution {
		return fmt.Errorf("kind must be 'value' or 'execution', got %q", schema.Kind)
	}

	// If primary parameter is set, ensure it exists in parameters
	if schema.PrimaryParameter != "" {
		if _, exists := schema.Parameters[schema.PrimaryParameter]; !exists {
			return fmt.Errorf("primary parameter %q not found in parameters", schema.PrimaryParameter)
		}
	}

	// Validate parameters
	for name, param := range schema.Parameters {
		if param.Name == "" {
			return fmt.Errorf("parameter name cannot be empty")
		}
		if param.Name != name {
			return fmt.Errorf("parameter name mismatch: key=%q, param.Name=%q", name, param.Name)
		}
		if param.Type == "" {
			return fmt.Errorf("parameter %q: type cannot be empty", name)
		}
		// Validate type is a known type
		if !isValidParamType(param.Type) {
			return fmt.Errorf("parameter %q: unknown type %q", name, param.Type)
		}
	}

	// Validate parameter order - all names in order must exist in parameters
	for _, name := range schema.ParameterOrder {
		if _, exists := schema.Parameters[name]; !exists {
			return fmt.Errorf("parameter %q in order but not in parameters map", name)
		}
	}

	return nil
}

// isValidParamType checks if a ParamType is valid
func isValidParamType(typ ParamType) bool {
	switch typ {
	case TypeString, TypeInt, TypeFloat, TypeBool, TypeDuration,
		TypeObject, TypeArray, TypeAuthHandle, TypeSecretHandle, TypeScrubMode:
		return true
	default:
		return false
	}
}

// GetOrderedParameters returns parameters in declaration order
// This is used for positional parameter mapping
func (s *DecoratorSchema) GetOrderedParameters() []ParamSchema {
	result := make([]ParamSchema, 0, len(s.ParameterOrder))
	for _, name := range s.ParameterOrder {
		if param, exists := s.Parameters[name]; exists {
			result = append(result, param)
		}
	}
	return result
}
