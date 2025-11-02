package types

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"golang.org/x/mod/semver"
)

// Validator validates parameter values against schemas
type Validator struct {
	config *ValidationConfig
	cache  *validatorCache
}

// NewValidator creates a new validator with given config
func NewValidator(config *ValidationConfig) *Validator {
	if config == nil {
		config = DefaultValidationConfig()
	}

	var cache *validatorCache
	if config.EnableCache {
		cache = newValidatorCache(config.MaxCacheSize)
	}

	return &Validator{
		config: config,
		cache:  cache,
	}
}

// ValidateParams validates a value against a parameter schema
func (v *Validator) ValidateParams(schema *ParamSchema, value interface{}) error {
	// Convert to JSON Schema
	jsonSchema, err := schema.ToJSONSchema()
	if err != nil {
		return fmt.Errorf("schema conversion failed: %w", err)
	}

	// Check schema size (security)
	schemaBytes, err := json.Marshal(jsonSchema)
	if err != nil {
		return fmt.Errorf("schema marshal failed: %w", err)
	}
	if len(schemaBytes) > v.config.MaxSchemaSize {
		return fmt.Errorf("schema too large: %d bytes (max: %d)",
			len(schemaBytes), v.config.MaxSchemaSize)
	}

	// Check schema depth (security)
	depth := measureSchemaDepth(jsonSchema)
	if depth > v.config.MaxSchemaDepth {
		return fmt.Errorf("schema too deep: %d levels (max: %d)",
			depth, v.config.MaxSchemaDepth)
	}

	// Get or compile validator
	validator, err := v.getValidator(jsonSchema)
	if err != nil {
		return fmt.Errorf("validator compilation failed: %w", err)
	}

	// Validate
	if err := validator.Validate(value); err != nil {
		return convertValidationError(err)
	}

	return nil
}

// getValidator gets cached validator or compiles new one
func (v *Validator) getValidator(schema JSONSchema) (*jsonschema.Schema, error) {
	// Compute schema hash for cache lookup
	schemaHash, err := hashSchema(schema)
	if err != nil {
		return nil, err
	}

	// Check cache
	if v.cache != nil {
		if validator, ok := v.cache.get(schemaHash); ok {
			return validator, nil
		}
	}

	// Compile new validator
	validator, err := v.compileSchema(schema)
	if err != nil {
		return nil, err
	}

	// Cache it
	if v.cache != nil {
		v.cache.put(schemaHash, validator)
	}

	return validator, nil
}

// compileSchema compiles JSON Schema with security controls
func (v *Validator) compileSchema(schema JSONSchema) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	compiler.AssertFormat = v.config.AssertFormat
	compiler.AssertContent = v.config.AssertContent

	// Extend (not replace) format validators with our custom ones
	// The compiler already has standard validators (email, uri, ipv4, etc.)
	// We add Opal-specific formats on top
	if compiler.Formats == nil {
		compiler.Formats = make(map[string]func(interface{}) bool)
	}
	for name, validator := range getFormatValidators() {
		compiler.Formats[name] = validator
	}

	// Security: Control $ref resolution
	compiler.LoadURL = v.createSecureLoader()

	// Add schema as resource
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	url := "schema://main.json"
	if err := compiler.AddResource(url, strings.NewReader(string(schemaJSON))); err != nil {
		return nil, err
	}

	// Compile
	return compiler.Compile(url)
}

// createSecureLoader creates a LoadURL function with security controls
func (v *Validator) createSecureLoader() func(string) (io.ReadCloser, error) {
	return func(url string) (io.ReadCloser, error) {
		// Block remote refs if not allowed
		if !v.config.AllowRemoteRef {
			if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
				return nil, fmt.Errorf("remote $ref not allowed: %s", url)
			}
		}

		// Check allowed schemes
		allowed := false
		for _, scheme := range v.config.AllowedSchemes {
			if strings.HasPrefix(url, scheme+"://") || strings.HasPrefix(url, scheme+":") {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("URL scheme not allowed: %s", url)
		}

		// Use default loader
		return jsonschema.LoadURL(url)
	}
}

// getFormatValidators returns our custom format validators
func getFormatValidators() map[string]func(interface{}) bool {
	return map[string]func(interface{}) bool{
		"duration": func(v interface{}) bool {
			s, ok := v.(string)
			if !ok {
				return true // Type validation happens separately
			}
			_, err := ParseDuration(s)
			return err == nil
		},
		"cidr": func(v interface{}) bool {
			s, ok := v.(string)
			if !ok {
				return true // Type validation happens separately
			}
			_, err := netip.ParsePrefix(s)
			return err == nil
		},
		"semver": func(v interface{}) bool {
			s, ok := v.(string)
			if !ok {
				return true // Type validation happens separately
			}
			// semver.IsValid requires "v" prefix (e.g., "v1.2.3")
			// Accept both with and without prefix
			if !strings.HasPrefix(s, "v") {
				s = "v" + s
			}
			return semver.IsValid(s)
		},
	}
}

// convertValidationError converts jsonschema.ValidationError to our format
func convertValidationError(err error) error {
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return err
	}

	// For now, return the error as-is
	// In Phase 5, we'll convert to our custom error format
	return ve
}

// measureSchemaDepth measures the maximum nesting depth of a JSON Schema
// to prevent resource exhaustion from deeply nested schemas.
// Depth is counted by the number of times we encounter nesting keywords
// like "properties", "items", "allOf", "anyOf", "oneOf".
func measureSchemaDepth(schema JSONSchema) int {
	return measureDepth(schema, 0)
}

// measureDepth recursively measures schema depth
func measureDepth(obj any, currentDepth int) int {
	// Try to convert to map[string]any (handles both JSONSchema and plain maps)
	var m map[string]any
	switch v := obj.(type) {
	case JSONSchema:
		m = map[string]any(v)
	case map[string]any:
		m = v
	default:
		return currentDepth
	}

	maxDepth := currentDepth

	// Properties: each field schema is one level deeper
	if propsVal, hasProps := m["properties"]; hasProps {
		// Handle both map[string]any and map[string]JSONSchema
		switch props := propsVal.(type) {
		case map[string]any:
			for _, fieldSchema := range props {
				depth := measureDepth(fieldSchema, currentDepth+1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		case map[string]JSONSchema:
			for _, fieldSchema := range props {
				depth := measureDepth(fieldSchema, currentDepth+1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		}
	}

	// Items: array element schema is one level deeper
	if items, ok := m["items"]; ok {
		depth := measureDepth(items, currentDepth+1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	// Combinators: each schema in array is one level deeper
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := m[key].([]any); ok {
			for _, schema := range arr {
				depth := measureDepth(schema, currentDepth+1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		}
	}

	return maxDepth
}
