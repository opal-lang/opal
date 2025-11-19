package decorator

import (
	"fmt"
	"regexp"

	"github.com/opal-lang/opal/core/types"
)

// DeprecationInfo holds information about a deprecated parameter.
type DeprecationInfo struct {
	// ReplacedBy is the new parameter name that replaces this deprecated one
	ReplacedBy string

	// Message is an optional custom deprecation message
	// If empty, a default message will be generated: "parameter 'old' is deprecated, use 'new' instead"
	Message string
}

// ParamBuilder provides a fluent API for building parameters with type-specific constraints.
// It returns to the parent DescriptorBuilder when Done() is called.
type ParamBuilder struct {
	parent           *DescriptorBuilder
	schema           types.ParamSchema
	requiredExplicit bool // Track if Required() was explicitly called
	deprecation      *DeprecationInfo
}

// Required marks the parameter as required.
func (pb *ParamBuilder) Required() *ParamBuilder {
	pb.schema.Required = true
	pb.requiredExplicit = true
	return pb
}

// Default sets the default value for the parameter.
// Automatically marks the parameter as optional.
func (pb *ParamBuilder) Default(value any) *ParamBuilder {
	pb.schema.Default = value
	pb.schema.Required = false
	return pb
}

// Examples adds example values for documentation.
func (pb *ParamBuilder) Examples(examples ...string) *ParamBuilder {
	pb.schema.Examples = examples
	return pb
}

// MinLength sets the minimum length constraint (for strings and arrays).
func (pb *ParamBuilder) MinLength(n int) *ParamBuilder {
	pb.schema.MinLength = &n
	return pb
}

// MaxLength sets the maximum length constraint (for strings and arrays).
func (pb *ParamBuilder) MaxLength(n int) *ParamBuilder {
	pb.schema.MaxLength = &n
	return pb
}

// Pattern sets a regex pattern constraint (for strings).
// Pattern is validated when Build() is called.
func (pb *ParamBuilder) Pattern(regex string) *ParamBuilder {
	pb.schema.Pattern = &regex
	return pb
}

// Format sets a typed format constraint (for strings).
// Examples: FormatURI, FormatHostname, FormatIPv4, FormatCIDR, FormatSemver, FormatDuration
func (pb *ParamBuilder) Format(format types.Format) *ParamBuilder {
	pb.schema.Format = &format
	return pb
}

// Min sets the minimum value constraint (for numeric types).
func (pb *ParamBuilder) Min(minVal float64) *ParamBuilder {
	pb.schema.Minimum = &minVal
	return pb
}

// Max sets the maximum value constraint (for numeric types).
func (pb *ParamBuilder) Max(maxVal float64) *ParamBuilder {
	pb.schema.Maximum = &maxVal
	return pb
}

// Deprecation marks this parameter as deprecated and replaced by another parameter.
// This parameter will still be accepted but will emit a warning.
// Example: ParamInt("maxConcurrency", "...").Deprecation(DeprecationInfo{ReplacedBy: "max_workers"})
func (pb *ParamBuilder) Deprecation(info DeprecationInfo) *ParamBuilder {
	pb.deprecation = &info
	return pb
}

// Done finishes building this parameter and returns to the parent DescriptorBuilder.
// Validates the parameter schema before adding it.
func (pb *ParamBuilder) Done() *DescriptorBuilder {
	// Validate parameter schema
	if err := pb.validate(); err != nil {
		panic(fmt.Sprintf("invalid parameter %q: %v", pb.schema.Name, err))
	}

	// Register deprecated parameter name if specified
	if pb.deprecation != nil {
		if pb.parent.desc.Schema.DeprecatedParameters == nil {
			pb.parent.desc.Schema.DeprecatedParameters = make(map[string]string)
		}
		// Map: old name (this param) -> new name (replacement)
		pb.parent.desc.Schema.DeprecatedParameters[pb.schema.Name] = pb.deprecation.ReplacedBy
	}

	// Add parameter to parent descriptor
	pb.parent.desc.Schema.Parameters[pb.schema.Name] = pb.schema
	pb.parent.desc.Schema.ParameterOrder = append(pb.parent.desc.Schema.ParameterOrder, pb.schema.Name)

	return pb.parent
}

// validate checks parameter schema for common errors.
func (pb *ParamBuilder) validate() error {
	// Check for required + default (invalid combination)
	// Only error if Required() was explicitly called before Default()
	if pb.requiredExplicit && pb.schema.Default != nil {
		return fmt.Errorf("parameter cannot be both required and have a default value")
	}

	// Validate regex pattern if set
	if pb.schema.Pattern != nil {
		if _, err := regexp.Compile(*pb.schema.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", *pb.schema.Pattern, err)
		}
	}

	// Validate min <= max for numeric types
	if pb.schema.Minimum != nil && pb.schema.Maximum != nil {
		if *pb.schema.Minimum > *pb.schema.Maximum {
			return fmt.Errorf("minimum (%v) cannot be greater than maximum (%v)", *pb.schema.Minimum, *pb.schema.Maximum)
		}
	}

	return nil
}

// ParamString creates a string parameter builder.
func (b *DescriptorBuilder) ParamString(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeString,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamInt creates an integer parameter builder.
func (b *DescriptorBuilder) ParamInt(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeInt,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamFloat creates a float parameter builder.
func (b *DescriptorBuilder) ParamFloat(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeFloat,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamBool creates a boolean parameter builder.
func (b *DescriptorBuilder) ParamBool(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeBool,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamDuration creates a duration parameter builder.
func (b *DescriptorBuilder) ParamDuration(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeDuration,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// EnumParamBuilder provides a fluent API for building enum parameters.
type EnumParamBuilder struct {
	parent      *DescriptorBuilder
	schema      types.ParamSchema
	deprecation *DeprecationInfo
}

// Values sets the allowed enum values.
func (eb *EnumParamBuilder) Values(values ...string) *EnumParamBuilder {
	if eb.schema.EnumSchema == nil {
		eb.schema.EnumSchema = &types.EnumSchema{}
	}
	eb.schema.EnumSchema.Values = values
	return eb
}

// Default sets the default enum value.
// The default must be one of the allowed values.
func (eb *EnumParamBuilder) Default(value string) *EnumParamBuilder {
	if eb.schema.EnumSchema == nil {
		eb.schema.EnumSchema = &types.EnumSchema{}
	}
	eb.schema.EnumSchema.Default = &value
	eb.schema.Default = value
	eb.schema.Required = false
	return eb
}

// Required marks the enum parameter as required.
func (eb *EnumParamBuilder) Required() *EnumParamBuilder {
	eb.schema.Required = true
	return eb
}

// Deprecated marks an old enum value as deprecated and maps it to a replacement.
// The old value must NOT be in the current values list.
// The replacement must be in the current values list.
func (eb *EnumParamBuilder) Deprecated(oldValue, replacement string) *EnumParamBuilder {
	if eb.schema.EnumSchema == nil {
		eb.schema.EnumSchema = &types.EnumSchema{}
	}
	if eb.schema.EnumSchema.DeprecatedValues == nil {
		eb.schema.EnumSchema.DeprecatedValues = make(map[string]string)
	}
	eb.schema.EnumSchema.DeprecatedValues[oldValue] = replacement
	return eb
}

// Examples adds example values for documentation.
func (eb *EnumParamBuilder) Examples(examples ...string) *EnumParamBuilder {
	eb.schema.Examples = examples
	return eb
}

// Deprecation marks this parameter as deprecated and replaced by another parameter.
// This parameter will still be accepted but will emit a warning.
func (eb *EnumParamBuilder) Deprecation(info DeprecationInfo) *EnumParamBuilder {
	eb.deprecation = &info
	return eb
}

// Done finishes building this enum parameter and returns to the parent DescriptorBuilder.
// Validates the enum schema before adding it.
func (eb *EnumParamBuilder) Done() *DescriptorBuilder {
	// Validate enum schema
	if err := eb.validate(); err != nil {
		panic(fmt.Sprintf("invalid enum parameter %q: %v", eb.schema.Name, err))
	}

	// Register deprecated parameter name if specified
	if eb.deprecation != nil {
		if eb.parent.desc.Schema.DeprecatedParameters == nil {
			eb.parent.desc.Schema.DeprecatedParameters = make(map[string]string)
		}
		// Map: old name (this param) -> new name (replacement)
		eb.parent.desc.Schema.DeprecatedParameters[eb.schema.Name] = eb.deprecation.ReplacedBy
	}

	// Add parameter to parent descriptor
	eb.parent.desc.Schema.Parameters[eb.schema.Name] = eb.schema
	eb.parent.desc.Schema.ParameterOrder = append(eb.parent.desc.Schema.ParameterOrder, eb.schema.Name)

	return eb.parent
}

// validate checks enum schema for common errors.
func (eb *EnumParamBuilder) validate() error {
	if eb.schema.EnumSchema == nil {
		return fmt.Errorf("enum schema not set (call Values() first)")
	}

	// Check that values is not empty
	if len(eb.schema.EnumSchema.Values) == 0 {
		return fmt.Errorf("enum values cannot be empty")
	}

	// Check that default is in values
	if eb.schema.EnumSchema.Default != nil {
		found := false
		for _, v := range eb.schema.EnumSchema.Values {
			if v == *eb.schema.EnumSchema.Default {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("default value %q must be one of the allowed values: %v", *eb.schema.EnumSchema.Default, eb.schema.EnumSchema.Values)
		}
	}

	// Check that deprecated values are NOT in current values
	for oldValue, replacement := range eb.schema.EnumSchema.DeprecatedValues {
		// Old value should NOT be in current values
		for _, v := range eb.schema.EnumSchema.Values {
			if v == oldValue {
				return fmt.Errorf("deprecated value %q cannot be in current values list (use replacement %q instead)", oldValue, replacement)
			}
		}

		// Replacement should be in current values
		found := false
		for _, v := range eb.schema.EnumSchema.Values {
			if v == replacement {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("replacement value %q for deprecated %q must be in current values: %v", replacement, oldValue, eb.schema.EnumSchema.Values)
		}
	}

	return nil
}

// ParamEnum creates an enum parameter builder.
func (b *DescriptorBuilder) ParamEnum(name, description string) *EnumParamBuilder {
	return &EnumParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeEnum,
			Description: description,
			Required:    false, // Optional by default
			EnumSchema:  &types.EnumSchema{},
		},
	}
}

// ObjectParamBuilder provides a fluent API for building object parameters.
type ObjectParamBuilder struct {
	parent      *DescriptorBuilder
	schema      types.ParamSchema
	deprecation *DeprecationInfo
}

// Field adds a field to the object schema.
func (ob *ObjectParamBuilder) Field(name string, typ types.ParamType, description string) *ObjectParamBuilder {
	if ob.schema.ObjectSchema == nil {
		ob.schema.ObjectSchema = &types.ObjectSchema{
			Fields: make(map[string]types.ParamSchema),
		}
	}
	ob.schema.ObjectSchema.Fields[name] = types.ParamSchema{
		Name:        name,
		Type:        typ,
		Description: description,
	}
	return ob
}

// FieldObject adds a nested object field and returns a nested builder.
func (ob *ObjectParamBuilder) FieldObject(name, description string) *NestedObjectBuilder {
	if ob.schema.ObjectSchema == nil {
		ob.schema.ObjectSchema = &types.ObjectSchema{
			Fields: make(map[string]types.ParamSchema),
		}
	}
	return &NestedObjectBuilder{
		parent:    ob,
		fieldName: name,
		schema: types.ParamSchema{
			Name:         name,
			Type:         types.TypeObject,
			Description:  description,
			ObjectSchema: &types.ObjectSchema{Fields: make(map[string]types.ParamSchema)},
		},
	}
}

// RequiredFields marks fields as required.
func (ob *ObjectParamBuilder) RequiredFields(fields ...string) *ObjectParamBuilder {
	if ob.schema.ObjectSchema == nil {
		ob.schema.ObjectSchema = &types.ObjectSchema{
			Fields: make(map[string]types.ParamSchema),
		}
	}
	ob.schema.ObjectSchema.Required = append(ob.schema.ObjectSchema.Required, fields...)
	return ob
}

// AllowAdditionalProperties allows extra fields not defined in the schema.
// By default, objects are closed (AdditionalProperties = false) to catch typos.
func (ob *ObjectParamBuilder) AllowAdditionalProperties() *ObjectParamBuilder {
	if ob.schema.ObjectSchema == nil {
		ob.schema.ObjectSchema = &types.ObjectSchema{
			Fields: make(map[string]types.ParamSchema),
		}
	}
	ob.schema.ObjectSchema.AdditionalProperties = true
	return ob
}

// Required marks the object parameter as required.
func (ob *ObjectParamBuilder) Required() *ObjectParamBuilder {
	ob.schema.Required = true
	return ob
}

// Examples adds example values for documentation.
func (ob *ObjectParamBuilder) Examples(examples ...string) *ObjectParamBuilder {
	ob.schema.Examples = examples
	return ob
}

// Deprecation marks this parameter as deprecated and replaced by another parameter.
// This parameter will still be accepted but will emit a warning.
func (ob *ObjectParamBuilder) Deprecation(info DeprecationInfo) *ObjectParamBuilder {
	ob.deprecation = &info
	return ob
}

// Done finishes building this object parameter and returns to the parent DescriptorBuilder.
// Validates the object schema before adding it.
func (ob *ObjectParamBuilder) Done() *DescriptorBuilder {
	// Validate object schema
	if err := ob.validate(); err != nil {
		panic(fmt.Sprintf("invalid object parameter %q: %v", ob.schema.Name, err))
	}

	// Register deprecated parameter name if specified
	if ob.deprecation != nil {
		if ob.parent.desc.Schema.DeprecatedParameters == nil {
			ob.parent.desc.Schema.DeprecatedParameters = make(map[string]string)
		}
		// Map: old name (this param) -> new name (replacement)
		ob.parent.desc.Schema.DeprecatedParameters[ob.schema.Name] = ob.deprecation.ReplacedBy
	}

	// Add parameter to parent descriptor
	ob.parent.desc.Schema.Parameters[ob.schema.Name] = ob.schema
	ob.parent.desc.Schema.ParameterOrder = append(ob.parent.desc.Schema.ParameterOrder, ob.schema.Name)

	return ob.parent
}

// validate checks object schema for common errors.
func (ob *ObjectParamBuilder) validate() error {
	if ob.schema.ObjectSchema == nil {
		return fmt.Errorf("object schema not set")
	}

	// Check that required fields exist in fields map
	for _, reqField := range ob.schema.ObjectSchema.Required {
		if _, exists := ob.schema.ObjectSchema.Fields[reqField]; !exists {
			return fmt.Errorf("required field %q does not exist in object fields", reqField)
		}
	}

	return nil
}

// NestedObjectBuilder provides a fluent API for building nested object fields.
type NestedObjectBuilder struct {
	parent    *ObjectParamBuilder
	fieldName string
	schema    types.ParamSchema
}

// Field adds a field to the nested object.
func (nb *NestedObjectBuilder) Field(name string, typ types.ParamType, description string) *NestedObjectBuilder {
	nb.schema.ObjectSchema.Fields[name] = types.ParamSchema{
		Name:        name,
		Type:        typ,
		Description: description,
	}
	return nb
}

// RequiredFields marks fields as required in the nested object.
func (nb *NestedObjectBuilder) RequiredFields(fields ...string) *NestedObjectBuilder {
	nb.schema.ObjectSchema.Required = append(nb.schema.ObjectSchema.Required, fields...)
	return nb
}

// DoneField finishes building the nested object field and returns to the parent object builder.
func (nb *NestedObjectBuilder) DoneField() *ObjectParamBuilder {
	// Add nested object field to parent
	nb.parent.schema.ObjectSchema.Fields[nb.fieldName] = nb.schema
	return nb.parent
}

// ParamObject creates an object parameter builder.
func (b *DescriptorBuilder) ParamObject(name, description string) *ObjectParamBuilder {
	return &ObjectParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:         name,
			Type:         types.TypeObject,
			Description:  description,
			Required:     false, // Optional by default
			ObjectSchema: &types.ObjectSchema{Fields: make(map[string]types.ParamSchema)},
		},
	}
}

// ArrayParamBuilder provides a fluent API for building array parameters.
type ArrayParamBuilder struct {
	parent      *DescriptorBuilder
	schema      types.ParamSchema
	deprecation *DeprecationInfo
}

// ElementType sets the type of array elements.
func (ab *ArrayParamBuilder) ElementType(typ types.ParamType) *ArrayParamBuilder {
	if ab.schema.ArraySchema == nil {
		ab.schema.ArraySchema = &types.ArraySchema{}
	}
	ab.schema.ArraySchema.ElementType = typ
	return ab
}

// ElementObject starts building an object element schema and returns a nested builder.
func (ab *ArrayParamBuilder) ElementObject() *ArrayElementObjectBuilder {
	if ab.schema.ArraySchema == nil {
		ab.schema.ArraySchema = &types.ArraySchema{}
	}
	ab.schema.ArraySchema.ElementType = types.TypeObject
	ab.schema.ArraySchema.ElementSchema = &types.ParamSchema{
		Type:         types.TypeObject,
		ObjectSchema: &types.ObjectSchema{Fields: make(map[string]types.ParamSchema)},
	}
	return &ArrayElementObjectBuilder{
		parent: ab,
		schema: ab.schema.ArraySchema.ElementSchema,
	}
}

// MinLength sets the minimum number of array elements.
func (ab *ArrayParamBuilder) MinLength(n int) *ArrayParamBuilder {
	if ab.schema.ArraySchema == nil {
		ab.schema.ArraySchema = &types.ArraySchema{}
	}
	ab.schema.ArraySchema.MinLength = &n
	return ab
}

// MaxLength sets the maximum number of array elements.
func (ab *ArrayParamBuilder) MaxLength(n int) *ArrayParamBuilder {
	if ab.schema.ArraySchema == nil {
		ab.schema.ArraySchema = &types.ArraySchema{}
	}
	ab.schema.ArraySchema.MaxLength = &n
	return ab
}

// UniqueItems requires all array elements to be unique.
func (ab *ArrayParamBuilder) UniqueItems() *ArrayParamBuilder {
	if ab.schema.ArraySchema == nil {
		ab.schema.ArraySchema = &types.ArraySchema{}
	}
	ab.schema.ArraySchema.UniqueItems = true
	return ab
}

// Required marks the array parameter as required.
func (ab *ArrayParamBuilder) Required() *ArrayParamBuilder {
	ab.schema.Required = true
	return ab
}

// Examples adds example values for documentation.
func (ab *ArrayParamBuilder) Examples(examples ...string) *ArrayParamBuilder {
	ab.schema.Examples = examples
	return ab
}

// Deprecation marks this parameter as deprecated and replaced by another parameter.
// This parameter will still be accepted but will emit a warning.
func (ab *ArrayParamBuilder) Deprecation(info DeprecationInfo) *ArrayParamBuilder {
	ab.deprecation = &info
	return ab
}

// Done finishes building this array parameter and returns to the parent DescriptorBuilder.
// Validates the array schema before adding it.
func (ab *ArrayParamBuilder) Done() *DescriptorBuilder {
	// Validate array schema
	if err := ab.validate(); err != nil {
		panic(fmt.Sprintf("invalid array parameter %q: %v", ab.schema.Name, err))
	}

	// Register deprecated parameter name if specified
	if ab.deprecation != nil {
		if ab.parent.desc.Schema.DeprecatedParameters == nil {
			ab.parent.desc.Schema.DeprecatedParameters = make(map[string]string)
		}
		// Map: old name (this param) -> new name (replacement)
		ab.parent.desc.Schema.DeprecatedParameters[ab.schema.Name] = ab.deprecation.ReplacedBy
	}

	// Add parameter to parent descriptor
	ab.parent.desc.Schema.Parameters[ab.schema.Name] = ab.schema
	ab.parent.desc.Schema.ParameterOrder = append(ab.parent.desc.Schema.ParameterOrder, ab.schema.Name)

	return ab.parent
}

// validate checks array schema for common errors.
func (ab *ArrayParamBuilder) validate() error {
	if ab.schema.ArraySchema == nil {
		return fmt.Errorf("array schema not set")
	}

	// Check that element type is set
	if ab.schema.ArraySchema.ElementType == "" {
		return fmt.Errorf("element type must be set (call ElementType() or ElementObject())")
	}

	// Check min <= max if both set
	if ab.schema.ArraySchema.MinLength != nil && ab.schema.ArraySchema.MaxLength != nil {
		if *ab.schema.ArraySchema.MinLength > *ab.schema.ArraySchema.MaxLength {
			return fmt.Errorf("MinLength (%d) cannot be greater than MaxLength (%d)", *ab.schema.ArraySchema.MinLength, *ab.schema.ArraySchema.MaxLength)
		}
	}

	return nil
}

// ArrayElementObjectBuilder provides a fluent API for building object element schemas.
type ArrayElementObjectBuilder struct {
	parent *ArrayParamBuilder
	schema *types.ParamSchema
}

// Field adds a field to the element object schema.
func (aob *ArrayElementObjectBuilder) Field(name string, typ types.ParamType, description string) *ArrayElementObjectBuilder {
	aob.schema.ObjectSchema.Fields[name] = types.ParamSchema{
		Name:        name,
		Type:        typ,
		Description: description,
	}
	return aob
}

// RequiredFields marks fields as required in the element object.
func (aob *ArrayElementObjectBuilder) RequiredFields(fields ...string) *ArrayElementObjectBuilder {
	aob.schema.ObjectSchema.Required = append(aob.schema.ObjectSchema.Required, fields...)
	return aob
}

// DoneElement finishes building the element object and returns to the array builder.
func (aob *ArrayElementObjectBuilder) DoneElement() *ArrayParamBuilder {
	return aob.parent
}

// ParamArray creates an array parameter builder.
func (b *DescriptorBuilder) ParamArray(name, description string) *ArrayParamBuilder {
	return &ArrayParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeArray,
			Description: description,
			Required:    false, // Optional by default
			ArraySchema: &types.ArraySchema{},
		},
	}
}

// PrimaryParamString creates a string primary parameter builder.
// Primary parameters are always required and appear first in parameter order.
//
// IMPORTANT: Primary parameters MUST be strings. They are used in dot syntax
// (@env.HOME, @var.count) where the value after the dot is an identifier.
// Non-string primary parameters don't make semantic sense in this context.
func (b *DescriptorBuilder) PrimaryParamString(name, description string) *ParamBuilder {
	// Check for duplicate primary parameter
	if b.desc.Schema.PrimaryParameter != "" {
		panic(fmt.Sprintf("primary parameter already set to %q, cannot set to %q", b.desc.Schema.PrimaryParameter, name))
	}

	b.desc.Schema.PrimaryParameter = name

	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeString,
			Description: description,
			Required:    true, // Primary parameters are always required
		},
	}
}
