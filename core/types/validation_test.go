package types

import (
	"strings"
	"testing"
)

func TestValidator_ValidateParams_String(t *testing.T) {
	validator := NewValidator(nil)

	minLen := 3
	maxLen := 10
	schema := &ParamSchema{
		Name:      "name",
		Type:      TypeString,
		MinLength: &minLen,
		MaxLength: &maxLen,
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid", "hello", false},
		{"too short", "hi", true},
		{"too long", "hello world!", true},
		{"not string", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateParams_Integer(t *testing.T) {
	validator := NewValidator(nil)

	min := 1.0
	max := 100.0
	schema := &ParamSchema{
		Name:    "count",
		Type:    TypeInt,
		Minimum: &min,
		Maximum: &max,
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid", 50, false},
		{"valid float", 50.0, false},
		{"too small", 0, true},
		{"too large", 101, true},
		{"not number", "50", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateParams_Enum(t *testing.T) {
	validator := NewValidator(nil)

	schema := &ParamSchema{
		Name: "mode",
		Type: TypeEnum,
		EnumSchema: &EnumSchema{
			Values: []string{"read", "write", "append"},
		},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid", "read", false},
		{"invalid", "delete", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateParams_Duration(t *testing.T) {
	validator := NewValidator(nil)

	schema := &ParamSchema{
		Name: "timeout",
		Type: TypeDuration,
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid", "1h30m", false},
		{"valid simple", "30s", false},
		{"invalid format", "invalid", true},
		{"not string", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateParams_Object(t *testing.T) {
	validator := NewValidator(nil)

	schema := &ParamSchema{
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
					Name: "port",
					Type: TypeInt,
				},
			},
			Required:             []string{"host"},
			AdditionalProperties: false,
		},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid", map[string]interface{}{"host": "localhost", "port": 8080}, false},
		{"missing required", map[string]interface{}{"port": 8080}, true},
		{"additional property", map[string]interface{}{"host": "localhost", "extra": "value"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateParams_Array(t *testing.T) {
	validator := NewValidator(nil)

	minLen := 1
	maxLen := 3
	schema := &ParamSchema{
		Name: "tags",
		Type: TypeArray,
		ArraySchema: &ArraySchema{
			ElementType: TypeString,
			MinLength:   &minLen,
			MaxLength:   &maxLen,
			UniqueItems: true,
		},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid", []interface{}{"tag1", "tag2"}, false},
		{"empty", []interface{}{}, true},
		{"too many", []interface{}{"tag1", "tag2", "tag3", "tag4"}, true},
		{"duplicates", []interface{}{"tag1", "tag1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_Security_SchemaSize(t *testing.T) {
	config := DefaultValidationConfig()
	config.MaxSchemaSize = 100 // Very small limit
	validator := NewValidator(config)

	// Create a schema that will be large when serialized
	fields := make(map[string]ParamSchema)
	for i := 0; i < 100; i++ {
		fields[string(rune('a'+i%26))] = ParamSchema{
			Name: "field",
			Type: TypeString,
		}
	}

	schema := &ParamSchema{
		Name: "large",
		Type: TypeObject,
		ObjectSchema: &ObjectSchema{
			Fields: fields,
		},
	}

	err := validator.ValidateParams(schema, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for schema too large")
	}
	if !strings.Contains(err.Error(), "schema too large") {
		t.Errorf("Expected 'schema too large' error, got: %v", err)
	}
}

func TestValidator_Security_SchemaDepth(t *testing.T) {
	config := DefaultValidationConfig()
	config.MaxSchemaDepth = 2 // Shallow limit for testing
	validator := NewValidator(config)

	// Create a schema with depth 3 (exceeds limit of 2)
	// Depth 0: root
	// Depth 1: properties -> field1
	// Depth 2: properties -> field2
	// Depth 3: properties -> field3 (exceeds limit)
	schema := &ParamSchema{
		Name: "root",
		Type: TypeObject,
		ObjectSchema: &ObjectSchema{
			Fields: map[string]ParamSchema{
				"field1": {
					Name: "field1",
					Type: TypeObject,
					ObjectSchema: &ObjectSchema{
						Fields: map[string]ParamSchema{
							"field2": {
								Name: "field2",
								Type: TypeObject,
								ObjectSchema: &ObjectSchema{
									Fields: map[string]ParamSchema{
										"field3": {
											Name: "field3",
											Type: TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := validator.ValidateParams(schema, map[string]interface{}{
		"field1": map[string]interface{}{
			"field2": map[string]interface{}{
				"field3": "value",
			},
		},
	})

	if err == nil {
		t.Fatal("Expected error for schema too deep")
	}
	if !strings.Contains(err.Error(), "schema too deep") {
		t.Errorf("Expected 'schema too deep' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "3 levels") {
		t.Errorf("Expected error to mention '3 levels', got: %v", err)
	}
	if !strings.Contains(err.Error(), "max: 2") {
		t.Errorf("Expected error to mention 'max: 2', got: %v", err)
	}
}

func TestValidator_Cache(t *testing.T) {
	config := DefaultValidationConfig()
	config.EnableCache = true
	config.MaxCacheSize = 2
	validator := NewValidator(config)

	schema := &ParamSchema{
		Name: "test",
		Type: TypeString,
	}

	// First validation - cache miss
	err1 := validator.ValidateParams(schema, "hello")
	if err1 != nil {
		t.Fatalf("First validation failed: %v", err1)
	}

	// Second validation - cache hit
	err2 := validator.ValidateParams(schema, "world")
	if err2 != nil {
		t.Fatalf("Second validation failed: %v", err2)
	}

	// Verify cache has entry
	if validator.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestValidator_NoCache(t *testing.T) {
	config := DefaultValidationConfig()
	config.EnableCache = false
	validator := NewValidator(config)

	if validator.cache != nil {
		t.Error("Expected cache to be nil when disabled")
	}

	schema := &ParamSchema{
		Name: "test",
		Type: TypeString,
	}

	err := validator.ValidateParams(schema, "hello")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
}

// TestValidator_StandardFormats verifies that standard JSON Schema formats still work
// (regression test: we extend compiler.Formats, not replace it)
func TestValidator_StandardFormats(t *testing.T) {
	validator := NewValidator(nil)

	tests := []struct {
		name    string
		format  Format
		value   interface{}
		wantErr bool
	}{
		{"valid email", FormatEmail, "user@example.com", false},
		{"invalid email", FormatEmail, "not-an-email", true},
		{"valid URI", FormatURI, "https://example.com/path", false},
		{"invalid URI", FormatURI, "not a uri", true},
		{"valid IPv4", FormatIPv4, "192.168.1.1", false},
		{"invalid IPv4", FormatIPv4, "999.999.999.999", true},
		{"valid hostname", FormatHostname, "example.com", false},
		{"invalid hostname", FormatHostname, "invalid_hostname!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &ParamSchema{
				Name:   "test",
				Type:   TypeString,
				Format: &tt.format,
			}

			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidator_OpalFormats verifies that Opal-specific formats work
func TestValidator_OpalFormats(t *testing.T) {
	validator := NewValidator(nil)

	tests := []struct {
		name    string
		format  Format
		value   interface{}
		wantErr bool
	}{
		// CIDR format
		{"valid CIDR IPv4", FormatCIDR, "10.0.0.0/8", false},
		{"valid CIDR IPv6", FormatCIDR, "2001:db8::/32", false},
		{"invalid CIDR", FormatCIDR, "not-a-cidr", true},
		{"invalid CIDR no prefix", FormatCIDR, "10.0.0.0", true},

		// Semver format (follows Go module semver rules, accepts with/without v prefix)
		{"valid semver with v", FormatSemver, "v1.2.3", false},
		{"valid semver without v", FormatSemver, "1.2.3", false},
		{"valid semver short with v", FormatSemver, "v1.2", false},   // Go semver allows this
		{"valid semver short without v", FormatSemver, "1.2", false}, // We add v prefix
		{"valid semver prerelease", FormatSemver, "v1.2.3-alpha", false},
		{"valid semver build", FormatSemver, "v1.2.3+build", false},
		{"invalid semver", FormatSemver, "not-semver", true},
		{"invalid semver empty", FormatSemver, "", true},

		// Duration format (already tested elsewhere, but include for completeness)
		{"valid duration", FormatDuration, "1h30m", false},
		{"invalid duration", FormatDuration, "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &ParamSchema{
				Name:   "test",
				Type:   TypeString,
				Format: &tt.format,
			}

			err := validator.ValidateParams(schema, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()

	if config.MaxSchemaSize != 1024*1024 {
		t.Errorf("Expected MaxSchemaSize 1MB, got %d", config.MaxSchemaSize)
	}
	if config.MaxSchemaDepth != 10 {
		t.Errorf("Expected MaxSchemaDepth 10, got %d", config.MaxSchemaDepth)
	}
	if config.AllowRemoteRef {
		t.Error("Expected AllowRemoteRef to be false")
	}
	if !config.EnableCache {
		t.Error("Expected EnableCache to be true")
	}
	if !config.AssertFormat {
		t.Error("Expected AssertFormat to be true")
	}
}
