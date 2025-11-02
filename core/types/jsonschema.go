package types

import (
	"encoding/json"
	"fmt"
)

// JSONSchema represents a JSON Schema Draft 2020-12 document
type JSONSchema map[string]any

// ToJSONSchema converts a ParamSchema to JSON Schema format
func (p *ParamSchema) ToJSONSchema() (JSONSchema, error) {
	schema := make(JSONSchema)

	// Set type
	schema["type"] = jsonSchemaType(p.Type)

	// Set description
	if p.Description != "" {
		schema["description"] = p.Description
	}

	// Set default
	if p.Default != nil {
		schema["default"] = p.Default
	}

	// Set examples
	if len(p.Examples) > 0 {
		schema["examples"] = p.Examples
	}

	// Numeric constraints
	if p.Minimum != nil {
		schema["minimum"] = *p.Minimum
	}
	if p.Maximum != nil {
		schema["maximum"] = *p.Maximum
	}

	// String constraints
	if p.MinLength != nil {
		schema["minLength"] = *p.MinLength
	}
	if p.MaxLength != nil {
		schema["maxLength"] = *p.MaxLength
	}
	if p.Pattern != nil {
		schema["pattern"] = *p.Pattern
	}

	// Format
	// Automatically set format for TypeDuration if not explicitly set
	format := p.Format
	if format == nil && p.Type == TypeDuration {
		f := FormatDuration
		format = &f
	}

	if format != nil {
		if IsOpalFormat(*format) {
			// Opal-specific formats use x-opal-format extension
			schema["x-opal-format"] = string(*format)
		} else {
			// Standard JSON Schema format
			schema["format"] = string(*format)
		}
	}

	// Enum (legacy - prefer EnumSchema)
	if len(p.Enum) > 0 {
		schema["enum"] = p.Enum
	}

	// Type-specific schemas
	switch p.Type {
	case TypeEnum:
		if p.EnumSchema != nil {
			schema["enum"] = p.EnumSchema.Values
			if p.EnumSchema.Default != nil {
				schema["default"] = *p.EnumSchema.Default
			}
			if len(p.EnumSchema.DeprecatedValues) > 0 {
				schema["x-opal-deprecated"] = p.EnumSchema.DeprecatedValues
			}
		}

	case TypeObject:
		if p.ObjectSchema != nil {
			properties := make(map[string]JSONSchema)
			for name, field := range p.ObjectSchema.Fields {
				fieldSchema, err := field.ToJSONSchema()
				if err != nil {
					return nil, fmt.Errorf("field %q: %w", name, err)
				}
				properties[name] = fieldSchema
			}
			schema["properties"] = properties

			if len(p.ObjectSchema.Required) > 0 {
				schema["required"] = p.ObjectSchema.Required
			}

			schema["additionalProperties"] = p.ObjectSchema.AdditionalProperties
		}

	case TypeArray:
		if p.ArraySchema != nil {
			if p.ArraySchema.ElementSchema != nil {
				itemSchema, err := p.ArraySchema.ElementSchema.ToJSONSchema()
				if err != nil {
					return nil, fmt.Errorf("array items: %w", err)
				}
				schema["items"] = itemSchema
			} else {
				// Simple type
				schema["items"] = map[string]any{
					"type": jsonSchemaType(p.ArraySchema.ElementType),
				}
			}

			if p.ArraySchema.MinLength != nil {
				schema["minItems"] = *p.ArraySchema.MinLength
			}
			if p.ArraySchema.MaxLength != nil {
				schema["maxItems"] = *p.ArraySchema.MaxLength
			}
			if p.ArraySchema.UniqueItems {
				schema["uniqueItems"] = true
			}
		}
	}

	return schema, nil
}

// jsonSchemaType converts ParamType to JSON Schema type string
func jsonSchemaType(t ParamType) string {
	switch t {
	case TypeString, TypeDuration, TypeEnum:
		return "string"
	case TypeInt:
		return "integer"
	case TypeFloat:
		return "number"
	case TypeBool:
		return "boolean"
	case TypeObject:
		return "object"
	case TypeArray:
		return "array"
	default:
		// Custom types (AuthHandle, SecretHandle, etc.) are strings
		return "string"
	}
}

// ToJSON serializes the JSON Schema to JSON bytes
func (j JSONSchema) ToJSON() ([]byte, error) {
	return json.MarshalIndent(j, "", "  ")
}

// DecoratorSchemaToJSONSchema converts a DecoratorSchema to a full JSON Schema document
func DecoratorSchemaToJSONSchema(schema DecoratorSchema) (JSONSchema, error) {
	doc := make(JSONSchema)

	// JSON Schema metadata
	doc["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	doc["$id"] = fmt.Sprintf("https://opal-lang.com/schemas/decorators/%s.json", schema.Path)
	doc["title"] = schema.Path
	if schema.Description != "" {
		doc["description"] = schema.Description
	}

	// Opal-specific metadata
	doc["x-opal-vocabulary"] = "https://opal-lang.com/vocab/decorator/v1"
	doc["x-opal-kind"] = string(schema.Kind)

	// Convert parameters to properties
	if len(schema.Parameters) > 0 {
		properties := make(map[string]JSONSchema)
		var required []string

		for name, param := range schema.Parameters {
			paramSchema, err := param.ToJSONSchema()
			if err != nil {
				return nil, fmt.Errorf("parameter %q: %w", name, err)
			}
			properties[name] = paramSchema

			if param.Required {
				required = append(required, name)
			}
		}

		doc["type"] = "object"
		doc["properties"] = properties
		if len(required) > 0 {
			doc["required"] = required
		}
		doc["additionalProperties"] = false // Closed by default
	}

	// Primary parameter
	if schema.PrimaryParameter != "" {
		doc["x-opal-primary"] = schema.PrimaryParameter
	}

	// Returns
	if schema.Returns != nil {
		returnSchema := make(JSONSchema)
		returnSchema["type"] = jsonSchemaType(schema.Returns.Type)
		if schema.Returns.Description != "" {
			returnSchema["description"] = schema.Returns.Description
		}
		doc["x-opal-returns"] = returnSchema
	}

	return doc, nil
}
