package types

import (
	"encoding/json"
	"testing"
)

// TestToJSONSchema_String tests converting a string parameter to JSON Schema
func TestToJSONSchema_String(t *testing.T) {
	param := ParamSchema{
		Name:        "name",
		Type:        TypeString,
		Description: "User name",
		Required:    true,
	}

	schema, err := param.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema() error: %v", err)
	}

	// Verify type
	if schema["type"] != "string" {
		t.Errorf("expected type 'string', got %v", schema["type"])
	}

	// Verify description
	if schema["description"] != "User name" {
		t.Errorf("expected description 'User name', got %v", schema["description"])
	}
}

// TestToJSONSchema_Integer tests converting an integer parameter to JSON Schema
func TestToJSONSchema_Integer(t *testing.T) {
	min := 1.0
	max := 100.0
	param := ParamSchema{
		Name:        "count",
		Type:        TypeInt,
		Description: "Item count",
		Minimum:     &min,
		Maximum:     &max,
	}

	schema, err := param.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema() error: %v", err)
	}

	// Verify type
	if schema["type"] != "integer" {
		t.Errorf("expected type 'integer', got %v", schema["type"])
	}

	// Verify constraints
	if schema["minimum"] != 1.0 {
		t.Errorf("expected minimum 1.0, got %v", schema["minimum"])
	}
	if schema["maximum"] != 100.0 {
		t.Errorf("expected maximum 100.0, got %v", schema["maximum"])
	}
}

// TestToJSONSchema_Enum tests converting an enum parameter to JSON Schema
func TestToJSONSchema_Enum(t *testing.T) {
	defaultVal := "read"
	param := ParamSchema{
		Name: "mode",
		Type: TypeEnum,
		EnumSchema: &EnumSchema{
			Values:  []string{"read", "write", "append"},
			Default: &defaultVal,
			DeprecatedValues: map[string]string{
				"old_mode": "read",
			},
		},
	}

	schema, err := param.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema() error: %v", err)
	}

	// Verify type
	if schema["type"] != "string" {
		t.Errorf("expected type 'string', got %v", schema["type"])
	}

	// Verify enum values
	enumVals, ok := schema["enum"].([]string)
	if !ok {
		t.Fatalf("expected enum to be []string, got %T", schema["enum"])
	}
	if len(enumVals) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(enumVals))
	}

	// Verify default
	if schema["default"] != "read" {
		t.Errorf("expected default 'read', got %v", schema["default"])
	}

	// Verify deprecated values
	deprecated, ok := schema["x-opal-deprecated"].(map[string]string)
	if !ok {
		t.Fatalf("expected x-opal-deprecated to be map[string]string, got %T", schema["x-opal-deprecated"])
	}
	if deprecated["old_mode"] != "read" {
		t.Errorf("expected deprecated 'old_mode' -> 'read', got %v", deprecated["old_mode"])
	}
}

// TestToJSONSchema_Object tests converting an object parameter to JSON Schema
func TestToJSONSchema_Object(t *testing.T) {
	param := ParamSchema{
		Name: "config",
		Type: TypeObject,
		ObjectSchema: &ObjectSchema{
			Fields: map[string]ParamSchema{
				"host": {
					Name:     "host",
					Type:     TypeString,
					Required: true,
				},
				"port": {
					Name:    "port",
					Type:    TypeInt,
					Default: 8080,
				},
			},
			Required:             []string{"host"},
			AdditionalProperties: false,
		},
	}

	schema, err := param.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema() error: %v", err)
	}

	// Verify type
	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}

	// Verify properties
	props, ok := schema["properties"].(map[string]JSONSchema)
	if !ok {
		t.Fatalf("expected properties to be map[string]JSONSchema, got %T", schema["properties"])
	}
	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}

	// Verify required
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("expected required to be []string, got %T", schema["required"])
	}
	if len(required) != 1 || required[0] != "host" {
		t.Errorf("expected required ['host'], got %v", required)
	}

	// Verify additionalProperties
	if schema["additionalProperties"] != false {
		t.Errorf("expected additionalProperties false, got %v", schema["additionalProperties"])
	}
}

// TestToJSONSchema_Array tests converting an array parameter to JSON Schema
func TestToJSONSchema_Array(t *testing.T) {
	minLen := 1
	maxLen := 10
	param := ParamSchema{
		Name: "tags",
		Type: TypeArray,
		ArraySchema: &ArraySchema{
			ElementType: TypeString,
			MinLength:   &minLen,
			MaxLength:   &maxLen,
			UniqueItems: true,
		},
	}

	schema, err := param.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema() error: %v", err)
	}

	// Verify type
	if schema["type"] != "array" {
		t.Errorf("expected type 'array', got %v", schema["type"])
	}

	// Verify items
	items, ok := schema["items"].(map[string]any)
	if !ok {
		t.Fatalf("expected items to be map[string]any, got %T", schema["items"])
	}
	if items["type"] != "string" {
		t.Errorf("expected items type 'string', got %v", items["type"])
	}

	// Verify constraints
	if schema["minItems"] != 1 {
		t.Errorf("expected minItems 1, got %v", schema["minItems"])
	}
	if schema["maxItems"] != 10 {
		t.Errorf("expected maxItems 10, got %v", schema["maxItems"])
	}
	if schema["uniqueItems"] != true {
		t.Errorf("expected uniqueItems true, got %v", schema["uniqueItems"])
	}
}

// TestToJSONSchema_Format tests format handling (standard vs Opal-specific)
func TestToJSONSchema_Format(t *testing.T) {
	tests := []struct {
		name          string
		format        Format
		expectedField string
		expectedValue string
	}{
		{"standard URI", FormatURI, "format", "uri"},
		{"standard hostname", FormatHostname, "format", "hostname"},
		{"standard IPv4", FormatIPv4, "format", "ipv4"},
		{"standard IPv6", FormatIPv6, "format", "ipv6"},
		{"standard email", FormatEmail, "format", "email"},
		{"opal CIDR", FormatCIDR, "x-opal-format", "cidr"},
		{"opal semver", FormatSemver, "x-opal-format", "semver"},
		{"opal duration", FormatDuration, "x-opal-format", "duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := ParamSchema{
				Name:   "test",
				Type:   TypeString,
				Format: &tt.format,
			}

			schema, err := param.ToJSONSchema()
			if err != nil {
				t.Fatalf("ToJSONSchema() error: %v", err)
			}

			// Verify correct field is used
			if schema[tt.expectedField] != tt.expectedValue {
				t.Errorf("expected %s=%q, got %v", tt.expectedField, tt.expectedValue, schema[tt.expectedField])
			}

			// Verify other field is not set
			otherField := "format"
			if tt.expectedField == "format" {
				otherField = "x-opal-format"
			}
			if _, exists := schema[otherField]; exists {
				t.Errorf("expected %s to not be set, but it was", otherField)
			}
		})
	}
}

// TestToJSONSchema_Duration_AutoFormat tests that TypeDuration automatically gets format
func TestToJSONSchema_Duration_AutoFormat(t *testing.T) {
	t.Run("without explicit format", func(t *testing.T) {
		// User creates TypeDuration without calling .Format()
		param := ParamSchema{
			Name:        "timeout",
			Type:        TypeDuration,
			Description: "Request timeout",
		}

		schema, err := param.ToJSONSchema()
		if err != nil {
			t.Fatalf("ToJSONSchema() error: %v", err)
		}

		// Should automatically have x-opal-format: duration
		if schema["x-opal-format"] != "duration" {
			t.Errorf("expected x-opal-format='duration', got %v", schema["x-opal-format"])
		}

		// Should be type string
		if schema["type"] != "string" {
			t.Errorf("expected type 'string', got %v", schema["type"])
		}
	})

	t.Run("with explicit format", func(t *testing.T) {
		// User explicitly sets format (should respect it)
		format := FormatDuration
		param := ParamSchema{
			Name:        "timeout",
			Type:        TypeDuration,
			Description: "Request timeout",
			Format:      &format,
		}

		schema, err := param.ToJSONSchema()
		if err != nil {
			t.Fatalf("ToJSONSchema() error: %v", err)
		}

		// Should have x-opal-format: duration
		if schema["x-opal-format"] != "duration" {
			t.Errorf("expected x-opal-format='duration', got %v", schema["x-opal-format"])
		}
	})
}

// TestToJSON tests JSON serialization
func TestToJSON(t *testing.T) {
	schema := JSONSchema{
		"type":        "string",
		"description": "Test parameter",
	}

	jsonBytes, err := schema.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Verify content
	if parsed["type"] != "string" {
		t.Errorf("expected type 'string', got %v", parsed["type"])
	}
}

// TestDecoratorSchemaToJSONSchema tests full decorator schema conversion
func TestDecoratorSchemaToJSONSchema(t *testing.T) {
	schema := DecoratorSchema{
		Path:             "env",
		Kind:             KindValue,
		Description:      "Access environment variables",
		PrimaryParameter: "property",
		Parameters: map[string]ParamSchema{
			"property": {
				Name:        "property",
				Type:        TypeString,
				Description: "Environment variable name",
				Required:    true,
			},
		},
		Returns: &ReturnSchema{
			Type:        TypeString,
			Description: "Environment variable value",
		},
	}

	jsonSchema, err := DecoratorSchemaToJSONSchema(schema)
	if err != nil {
		t.Fatalf("DecoratorSchemaToJSONSchema() error: %v", err)
	}

	// Verify $schema
	if jsonSchema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("expected $schema, got %v", jsonSchema["$schema"])
	}

	// Verify $id
	expectedID := "https://opal-lang.com/schemas/decorators/env.json"
	if jsonSchema["$id"] != expectedID {
		t.Errorf("expected $id %q, got %v", expectedID, jsonSchema["$id"])
	}

	// Verify x-opal-vocabulary
	if jsonSchema["x-opal-vocabulary"] != "https://opal-lang.com/vocab/decorator/v1" {
		t.Errorf("expected x-opal-vocabulary, got %v", jsonSchema["x-opal-vocabulary"])
	}

	// Verify x-opal-kind
	if jsonSchema["x-opal-kind"] != "value" {
		t.Errorf("expected x-opal-kind 'value', got %v", jsonSchema["x-opal-kind"])
	}

	// Verify x-opal-primary
	if jsonSchema["x-opal-primary"] != "property" {
		t.Errorf("expected x-opal-primary 'property', got %v", jsonSchema["x-opal-primary"])
	}

	// Verify properties
	props, ok := jsonSchema["properties"].(map[string]JSONSchema)
	if !ok {
		t.Fatalf("expected properties to be map[string]JSONSchema, got %T", jsonSchema["properties"])
	}
	if len(props) != 1 {
		t.Errorf("expected 1 property, got %d", len(props))
	}

	// Verify returns
	returns, ok := jsonSchema["x-opal-returns"].(JSONSchema)
	if !ok {
		t.Fatalf("expected x-opal-returns to be JSONSchema, got %T", jsonSchema["x-opal-returns"])
	}
	if returns["type"] != "string" {
		t.Errorf("expected returns type 'string', got %v", returns["type"])
	}
}

// TestToJSONSchema_CustomTypes tests that custom ParamTypes map to correct JSON Schema types
func TestToJSONSchema_CustomTypes(t *testing.T) {
	tests := []struct {
		name         string
		paramType    ParamType
		expectedType string
	}{
		{"AuthHandle", TypeAuthHandle, "string"},
		{"SecretHandle", TypeSecretHandle, "string"},
		{"ScrubMode", TypeScrubMode, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := ParamSchema{
				Name: "test",
				Type: tt.paramType,
			}

			schema, err := param.ToJSONSchema()
			if err != nil {
				t.Fatalf("ToJSONSchema() error: %v", err)
			}

			if schema["type"] != tt.expectedType {
				t.Errorf("expected type %q, got %v", tt.expectedType, schema["type"])
			}
		})
	}
}
