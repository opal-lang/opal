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
		TypeObject, TypeArray, TypeAuthHandle, TypeSecretHandle:
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
