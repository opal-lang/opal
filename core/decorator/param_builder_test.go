package decorator

import (
	"testing"

	"github.com/aledsdavies/opal/core/types"
	"github.com/google/go-cmp/cmp"
)

// TestParamString_Basic tests basic string parameter creation
func TestParamString_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Done().
		Build()

	param, exists := desc.Schema.Parameters["name"]
	if !exists {
		t.Fatal("parameter 'name' not found")
	}

	if param.Type != types.TypeString {
		t.Errorf("expected type string, got %v", param.Type)
	}
	if param.Description != "User name" {
		t.Errorf("expected description 'User name', got %q", param.Description)
	}
	if param.Required {
		t.Error("expected parameter to be optional by default")
	}
}

// TestParamString_Required tests required string parameter
func TestParamString_Required(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Required().
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	if !param.Required {
		t.Error("expected parameter to be required")
	}
}

// TestParamString_Default tests string parameter with default value
func TestParamString_Default(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Default("anonymous").
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	if param.Required {
		t.Error("parameter with default should be optional")
	}
	if param.Default != "anonymous" {
		t.Errorf("expected default 'anonymous', got %v", param.Default)
	}
}

// TestParamString_MinMaxLength tests string length constraints
func TestParamString_MinMaxLength(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		MinLength(3).
		MaxLength(50).
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	if param.MinLength == nil || *param.MinLength != 3 {
		t.Errorf("expected MinLength=3, got %v", param.MinLength)
	}
	if param.MaxLength == nil || *param.MaxLength != 50 {
		t.Errorf("expected MaxLength=50, got %v", param.MaxLength)
	}
}

// TestParamString_Pattern tests regex pattern constraint
func TestParamString_Pattern(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("email", "Email address").
		Pattern(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`).
		Done().
		Build()

	param := desc.Schema.Parameters["email"]
	if param.Pattern == nil {
		t.Fatal("expected pattern to be set")
	}
	expected := `^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`
	if *param.Pattern != expected {
		t.Errorf("expected pattern %q, got %q", expected, *param.Pattern)
	}
}

// TestParamString_Format tests typed format constraint
func TestParamString_Format(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("url", "Website URL").
		Format(types.FormatURI).
		Done().
		Build()

	param := desc.Schema.Parameters["url"]
	if param.Format == nil {
		t.Fatal("expected format to be set")
	}
	if *param.Format != types.FormatURI {
		t.Errorf("expected format URI, got %v", *param.Format)
	}
}

// TestParamString_Examples tests example values
func TestParamString_Examples(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Examples("alice", "bob", "charlie").
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	expected := []string{"alice", "bob", "charlie"}
	if diff := cmp.Diff(expected, param.Examples); diff != "" {
		t.Errorf("examples mismatch (-want +got):\n%s", diff)
	}
}

// TestParamString_Chaining tests method chaining
func TestParamString_Chaining(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("email", "Email address").
		Required().
		Pattern(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`).
		MinLength(5).
		MaxLength(100).
		Examples("user@example.com").
		Done().
		Build()

	param := desc.Schema.Parameters["email"]
	if !param.Required {
		t.Error("expected parameter to be required")
	}
	if param.Pattern == nil {
		t.Error("expected pattern to be set")
	}
	if param.MinLength == nil || *param.MinLength != 5 {
		t.Error("expected MinLength=5")
	}
	if param.MaxLength == nil || *param.MaxLength != 100 {
		t.Error("expected MaxLength=100")
	}
	if len(param.Examples) != 1 || param.Examples[0] != "user@example.com" {
		t.Error("expected example 'user@example.com'")
	}
}

// TestParamInt_Basic tests basic integer parameter creation
func TestParamInt_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("count", "Number of items").
		Done().
		Build()

	param := desc.Schema.Parameters["count"]
	if param.Type != types.TypeInt {
		t.Errorf("expected type int, got %v", param.Type)
	}
	if param.Description != "Number of items" {
		t.Errorf("expected description 'Number of items', got %q", param.Description)
	}
}

// TestParamInt_MinMax tests integer min/max constraints
func TestParamInt_MinMax(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("port", "Port number").
		Min(1).
		Max(65535).
		Done().
		Build()

	param := desc.Schema.Parameters["port"]
	if param.Minimum == nil || *param.Minimum != 1.0 {
		t.Errorf("expected Minimum=1, got %v", param.Minimum)
	}
	if param.Maximum == nil || *param.Maximum != 65535.0 {
		t.Errorf("expected Maximum=65535, got %v", param.Maximum)
	}
}

// TestParamInt_DefaultValue tests integer with default
func TestParamInt_DefaultValue(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("retries", "Number of retries").
		Default(3).
		Done().
		Build()

	param := desc.Schema.Parameters["retries"]
	if param.Default != 3 {
		t.Errorf("expected default 3, got %v", param.Default)
	}
	if param.Required {
		t.Error("parameter with default should be optional")
	}
}

// TestParamDuration_Basic tests basic duration parameter creation
func TestParamDuration_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamDuration("timeout", "Timeout duration").
		Done().
		Build()

	param := desc.Schema.Parameters["timeout"]
	if param.Type != types.TypeDuration {
		t.Errorf("expected type duration, got %v", param.Type)
	}
	if param.Description != "Timeout duration" {
		t.Errorf("expected description 'Timeout duration', got %q", param.Description)
	}
}

// TestParamDuration_Default tests duration with default value
func TestParamDuration_Default(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamDuration("timeout", "Timeout duration").
		Default("30s").
		Done().
		Build()

	param := desc.Schema.Parameters["timeout"]
	if param.Default != "30s" {
		t.Errorf("expected default '30s', got %v", param.Default)
	}
}

// TestParamBool_Basic tests basic boolean parameter creation
func TestParamBool_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamBool("verbose", "Enable verbose output").
		Done().
		Build()

	param := desc.Schema.Parameters["verbose"]
	if param.Type != types.TypeBool {
		t.Errorf("expected type bool, got %v", param.Type)
	}
	if param.Description != "Enable verbose output" {
		t.Errorf("expected description 'Enable verbose output', got %q", param.Description)
	}
}

// TestParamBool_Default tests boolean with default value
func TestParamBool_Default(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamBool("verbose", "Enable verbose output").
		Default(false).
		Done().
		Build()

	param := desc.Schema.Parameters["verbose"]
	if param.Default != false {
		t.Errorf("expected default false, got %v", param.Default)
	}
}

// TestMultipleParams tests multiple parameters in order
func TestMultipleParams(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "Name").Required().Done().
		ParamInt("age", "Age").Min(0).Max(150).Done().
		ParamBool("active", "Is active").Default(true).Done().
		Build()

	if len(desc.Schema.Parameters) != 3 {
		t.Errorf("expected 3 parameters, got %d", len(desc.Schema.Parameters))
	}

	// Check parameter order
	expectedOrder := []string{"name", "age", "active"}
	if diff := cmp.Diff(expectedOrder, desc.Schema.ParameterOrder); diff != "" {
		t.Errorf("parameter order mismatch (-want +got):\n%s", diff)
	}
}

// TestPrimaryParam_WithBuilder tests primary parameter with builder
func TestPrimaryParam_WithBuilder(t *testing.T) {
	desc := NewDescriptor("env").
		Summary("Get environment variable").
		PrimaryParamString("name", "Variable name").
		Pattern(`^[A-Z_][A-Z0-9_]*$`).
		Examples("PATH", "HOME").
		Done().
		Build()

	if desc.Schema.PrimaryParameter != "name" {
		t.Errorf("expected primary parameter 'name', got %q", desc.Schema.PrimaryParameter)
	}

	param := desc.Schema.Parameters["name"]
	if !param.Required {
		t.Error("primary parameter should be required")
	}
	if param.Pattern == nil {
		t.Error("expected pattern to be set")
	}

	// Primary parameter should be first in order
	if len(desc.Schema.ParameterOrder) == 0 || desc.Schema.ParameterOrder[0] != "name" {
		t.Error("primary parameter should be first in order")
	}
}

// TestGuardrails_RequiredAndDefault tests that required+default is caught
func TestGuardrails_RequiredAndDefault(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for required parameter with default value")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "Name").
		Required().
		Default("value").
		Done().
		Build()
}

// TestGuardrails_InvalidPattern tests that invalid regex pattern is caught
func TestGuardrails_InvalidPattern(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid regex pattern")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "Name").
		Pattern(`[invalid(`). // Invalid regex
		Done().
		Build()
}

// TestGuardrails_MinGreaterThanMax tests that min > max is caught
func TestGuardrails_MinGreaterThanMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for min > max")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("count", "Count").
		Min(100).
		Max(10). // min > max
		Done().
		Build()
}

// TestGuardrails_DuplicatePrimaryParam tests that duplicate primary parameter is caught
func TestGuardrails_DuplicatePrimaryParam(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate primary parameter")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		PrimaryParamString("first", "First").Done().
		PrimaryParamString("second", "Second").Done(). // Duplicate primary
		Build()
}

// TestParamEnum_Basic tests basic enum parameter creation
func TestParamEnum_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Done().
		Build()

	param := desc.Schema.Parameters["level"]
	if param.Type != types.TypeEnum {
		t.Errorf("expected type enum, got %v", param.Type)
	}
	if param.Description != "Log level" {
		t.Errorf("expected description 'Log level', got %q", param.Description)
	}
	if param.EnumSchema == nil {
		t.Fatal("expected EnumSchema to be set")
	}

	expected := []string{"debug", "info", "warn", "error"}
	if diff := cmp.Diff(expected, param.EnumSchema.Values); diff != "" {
		t.Errorf("enum values mismatch (-want +got):\n%s", diff)
	}
}

// TestParamEnum_WithDefault tests enum with default value
func TestParamEnum_WithDefault(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Default("info").
		Done().
		Build()

	param := desc.Schema.Parameters["level"]
	if param.EnumSchema.Default == nil {
		t.Fatal("expected default to be set")
	}
	if *param.EnumSchema.Default != "info" {
		t.Errorf("expected default 'info', got %q", *param.EnumSchema.Default)
	}
	if param.Required {
		t.Error("parameter with default should be optional")
	}
}

// TestParamEnum_Required tests required enum parameter
func TestParamEnum_Required(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Required().
		Done().
		Build()

	param := desc.Schema.Parameters["level"]
	if !param.Required {
		t.Error("expected parameter to be required")
	}
}

// TestParamEnum_WithDeprecated tests enum with deprecated values
func TestParamEnum_WithDeprecated(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Deprecated("verbose", "debug").
		Deprecated("warning", "warn").
		Done().
		Build()

	param := desc.Schema.Parameters["level"]
	if param.EnumSchema.DeprecatedValues == nil {
		t.Fatal("expected DeprecatedValues to be set")
	}

	expected := map[string]string{
		"verbose": "debug",
		"warning": "warn",
	}
	if diff := cmp.Diff(expected, param.EnumSchema.DeprecatedValues); diff != "" {
		t.Errorf("deprecated values mismatch (-want +got):\n%s", diff)
	}
}

// TestParamEnum_Chaining tests method chaining
func TestParamEnum_Chaining(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Default("info").
		Deprecated("verbose", "debug").
		Examples("info", "warn").
		Done().
		Build()

	param := desc.Schema.Parameters["level"]
	if param.EnumSchema == nil {
		t.Fatal("expected EnumSchema to be set")
	}
	if param.EnumSchema.Default == nil || *param.EnumSchema.Default != "info" {
		t.Error("expected default 'info'")
	}
	if len(param.EnumSchema.DeprecatedValues) != 1 {
		t.Error("expected 1 deprecated value")
	}
	if len(param.Examples) != 2 {
		t.Error("expected 2 examples")
	}
}

// TestParamEnum_GuardrailDefaultNotInValues tests that default must be in values
func TestParamEnum_GuardrailDefaultNotInValues(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for default value not in enum values")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Default("invalid"). // Not in values
		Done().
		Build()
}

// TestParamEnum_GuardrailDeprecatedNotInValues tests that deprecated values must not be in values
func TestParamEnum_GuardrailDeprecatedNotInValues(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for deprecated value in enum values")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Values("debug", "info", "warn", "error").
		Deprecated("debug", "info"). // "debug" is in values - should error
		Done().
		Build()
}

// TestParamEnum_GuardrailEmptyValues tests that values cannot be empty
func TestParamEnum_GuardrailEmptyValues(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty enum values")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("level", "Log level").
		Done(). // No values set
		Build()
}

// TestParamObject_Basic tests basic object parameter creation
func TestParamObject_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		Field("host", types.TypeString, "Hostname").
		Field("port", types.TypeInt, "Port number").
		Done().
		Build()

	param := desc.Schema.Parameters["config"]
	if param.Type != types.TypeObject {
		t.Errorf("expected type object, got %v", param.Type)
	}
	if param.ObjectSchema == nil {
		t.Fatal("expected ObjectSchema to be set")
	}
	if len(param.ObjectSchema.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(param.ObjectSchema.Fields))
	}

	// Check fields
	hostField, exists := param.ObjectSchema.Fields["host"]
	if !exists {
		t.Fatal("expected 'host' field to exist")
	}
	if hostField.Type != types.TypeString {
		t.Errorf("expected host type string, got %v", hostField.Type)
	}

	portField, exists := param.ObjectSchema.Fields["port"]
	if !exists {
		t.Fatal("expected 'port' field to exist")
	}
	if portField.Type != types.TypeInt {
		t.Errorf("expected port type int, got %v", portField.Type)
	}
}

// TestParamObject_RequiredFields tests object with required fields
func TestParamObject_RequiredFields(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		Field("host", types.TypeString, "Hostname").
		Field("port", types.TypeInt, "Port number").
		RequiredFields("host").
		Done().
		Build()

	param := desc.Schema.Parameters["config"]
	expected := []string{"host"}
	if diff := cmp.Diff(expected, param.ObjectSchema.Required); diff != "" {
		t.Errorf("required fields mismatch (-want +got):\n%s", diff)
	}
}

// TestParamObject_AdditionalProperties tests object with additional properties allowed
func TestParamObject_AdditionalProperties(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		Field("host", types.TypeString, "Hostname").
		AllowAdditionalProperties().
		Done().
		Build()

	param := desc.Schema.Parameters["config"]
	if !param.ObjectSchema.AdditionalProperties {
		t.Error("expected AdditionalProperties to be true")
	}
}

// TestParamObject_ClosedByDefault tests that objects are closed by default
func TestParamObject_ClosedByDefault(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		Field("host", types.TypeString, "Hostname").
		Done().
		Build()

	param := desc.Schema.Parameters["config"]
	if param.ObjectSchema.AdditionalProperties {
		t.Error("expected AdditionalProperties to be false by default (closed objects)")
	}
}

// TestParamObject_NestedObject tests nested object fields
func TestParamObject_NestedObject(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		FieldObject("database", "Database configuration").
		Field("host", types.TypeString, "DB hostname").
		Field("port", types.TypeInt, "DB port").
		DoneField().
		Done().
		Build()

	param := desc.Schema.Parameters["config"]
	dbField, exists := param.ObjectSchema.Fields["database"]
	if !exists {
		t.Fatal("expected 'database' field to exist")
	}
	if dbField.Type != types.TypeObject {
		t.Errorf("expected database type object, got %v", dbField.Type)
	}
	if dbField.ObjectSchema == nil {
		t.Fatal("expected nested ObjectSchema to be set")
	}
	if len(dbField.ObjectSchema.Fields) != 2 {
		t.Errorf("expected 2 nested fields, got %d", len(dbField.ObjectSchema.Fields))
	}
}

// TestParamObject_Chaining tests method chaining
func TestParamObject_Chaining(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		Field("host", types.TypeString, "Hostname").
		Field("port", types.TypeInt, "Port number").
		RequiredFields("host", "port").
		Examples(`{"host": "localhost", "port": 8080}`).
		Done().
		Build()

	param := desc.Schema.Parameters["config"]
	if len(param.ObjectSchema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(param.ObjectSchema.Required))
	}
	if len(param.Examples) != 1 {
		t.Errorf("expected 1 example, got %d", len(param.Examples))
	}
}

// TestParamObject_GuardrailRequiredFieldNotExists tests that required field must exist
func TestParamObject_GuardrailRequiredFieldNotExists(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for required field that doesn't exist")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("config", "Configuration object").
		Field("host", types.TypeString, "Hostname").
		RequiredFields("port"). // "port" doesn't exist
		Done().
		Build()
}

// TestParamArray_Basic tests basic array parameter creation
func TestParamArray_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("tags", "List of tags").
		ElementType(types.TypeString).
		Done().
		Build()

	param := desc.Schema.Parameters["tags"]
	if param.Type != types.TypeArray {
		t.Errorf("expected type array, got %v", param.Type)
	}
	if param.ArraySchema == nil {
		t.Fatal("expected ArraySchema to be set")
	}
	if param.ArraySchema.ElementType != types.TypeString {
		t.Errorf("expected element type string, got %v", param.ArraySchema.ElementType)
	}
}

// TestParamArray_MinMaxLength tests array length constraints
func TestParamArray_MinMaxLength(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("tags", "List of tags").
		ElementType(types.TypeString).
		MinLength(1).
		MaxLength(10).
		Done().
		Build()

	param := desc.Schema.Parameters["tags"]
	if param.ArraySchema.MinLength == nil || *param.ArraySchema.MinLength != 1 {
		t.Errorf("expected MinLength=1, got %v", param.ArraySchema.MinLength)
	}
	if param.ArraySchema.MaxLength == nil || *param.ArraySchema.MaxLength != 10 {
		t.Errorf("expected MaxLength=10, got %v", param.ArraySchema.MaxLength)
	}
}

// TestParamArray_UniqueItems tests unique items constraint
func TestParamArray_UniqueItems(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("tags", "List of tags").
		ElementType(types.TypeString).
		UniqueItems().
		Done().
		Build()

	param := desc.Schema.Parameters["tags"]
	if !param.ArraySchema.UniqueItems {
		t.Error("expected UniqueItems to be true")
	}
}

// TestParamArray_ObjectElements tests array of objects
func TestParamArray_ObjectElements(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("servers", "List of servers").
		ElementObject().
		Field("host", types.TypeString, "Hostname").
		Field("port", types.TypeInt, "Port").
		DoneElement().
		Done().
		Build()

	param := desc.Schema.Parameters["servers"]
	if param.ArraySchema.ElementType != types.TypeObject {
		t.Errorf("expected element type object, got %v", param.ArraySchema.ElementType)
	}
	if param.ArraySchema.ElementSchema == nil {
		t.Fatal("expected ElementSchema to be set")
	}
	if param.ArraySchema.ElementSchema.ObjectSchema == nil {
		t.Fatal("expected element ObjectSchema to be set")
	}
	if len(param.ArraySchema.ElementSchema.ObjectSchema.Fields) != 2 {
		t.Errorf("expected 2 fields in element object, got %d", len(param.ArraySchema.ElementSchema.ObjectSchema.Fields))
	}
}

// TestParamArray_Chaining tests method chaining
func TestParamArray_Chaining(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("tags", "List of tags").
		ElementType(types.TypeString).
		MinLength(1).
		MaxLength(10).
		UniqueItems().
		Examples(`["tag1", "tag2"]`).
		Done().
		Build()

	param := desc.Schema.Parameters["tags"]
	if param.ArraySchema.MinLength == nil || *param.ArraySchema.MinLength != 1 {
		t.Error("expected MinLength=1")
	}
	if param.ArraySchema.MaxLength == nil || *param.ArraySchema.MaxLength != 10 {
		t.Error("expected MaxLength=10")
	}
	if !param.ArraySchema.UniqueItems {
		t.Error("expected UniqueItems=true")
	}
	if len(param.Examples) != 1 {
		t.Error("expected 1 example")
	}
}

// TestParamArray_GuardrailNoElementType tests that element type must be set
func TestParamArray_GuardrailNoElementType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for array without element type")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("tags", "List of tags").
		Done(). // No element type set
		Build()
}

// TestDeprecatedParam_MultipleDeprecations tests multiple deprecated parameters using fluent API
func TestDeprecatedParam_MultipleDeprecations(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("maxConcurrency", "Old parameter name").
		Deprecation(DeprecationInfo{ReplacedBy: "max_workers"}).
		Done().
		ParamInt("max_workers", "Maximum workers").Done().
		ParamDuration("retryDelay", "Old parameter name").
		Deprecation(DeprecationInfo{ReplacedBy: "retry_delay"}).
		Done().
		ParamDuration("retry_delay", "Retry delay").Done().
		Build()

	if len(desc.Schema.DeprecatedParameters) != 2 {
		t.Errorf("expected 2 deprecated parameters, got %d", len(desc.Schema.DeprecatedParameters))
	}

	// Verify both mappings
	if newName, exists := desc.Schema.DeprecatedParameters["maxConcurrency"]; !exists {
		t.Error("expected 'maxConcurrency' to be in DeprecatedParameters")
	} else if newName != "max_workers" {
		t.Errorf("expected 'maxConcurrency' to map to 'max_workers', got %q", newName)
	}

	if newName, exists := desc.Schema.DeprecatedParameters["retryDelay"]; !exists {
		t.Error("expected 'retryDelay' to be in DeprecatedParameters")
	} else if newName != "retry_delay" {
		t.Errorf("expected 'retryDelay' to map to 'retry_delay', got %q", newName)
	}
}

// TestParamBuilder_Deprecation tests the fluent .Deprecation() API
func TestParamBuilder_Deprecation(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("maxConcurrency", "Old parameter name").
		Deprecation(DeprecationInfo{ReplacedBy: "max_workers"}).
		Done().
		ParamInt("max_workers", "New parameter name").Done().
		Build()

	// Check that deprecated parameter mapping exists
	if desc.Schema.DeprecatedParameters == nil {
		t.Fatal("expected DeprecatedParameters to be set")
	}

	if newName, exists := desc.Schema.DeprecatedParameters["maxConcurrency"]; !exists {
		t.Error("expected 'maxConcurrency' to be in DeprecatedParameters")
	} else if newName != "max_workers" {
		t.Errorf("expected 'maxConcurrency' to map to 'max_workers', got %q", newName)
	}

	// Check that both parameters exist
	if _, exists := desc.Schema.Parameters["maxConcurrency"]; !exists {
		t.Error("expected 'maxConcurrency' parameter to exist (deprecated params are still defined)")
	}
	if _, exists := desc.Schema.Parameters["max_workers"]; !exists {
		t.Error("expected 'max_workers' parameter to exist")
	}
}

// TestEnumParamBuilder_Deprecation tests enum parameter deprecation
func TestEnumParamBuilder_Deprecation(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamEnum("logLevel", "Old parameter name").
		Values("info", "warn", "error").
		Deprecation(DeprecationInfo{ReplacedBy: "log_level"}).
		Done().
		ParamEnum("log_level", "New parameter name").
		Values("info", "warn", "error").
		Done().
		Build()

	if newName, exists := desc.Schema.DeprecatedParameters["logLevel"]; !exists {
		t.Error("expected 'logLevel' to be in DeprecatedParameters")
	} else if newName != "log_level" {
		t.Errorf("expected 'logLevel' to map to 'log_level', got %q", newName)
	}
}

// TestObjectParamBuilder_Deprecation tests object parameter deprecation
func TestObjectParamBuilder_Deprecation(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamObject("httpConfig", "Old parameter name").
		Field("url", types.TypeString, "URL").
		Deprecation(DeprecationInfo{ReplacedBy: "http_config"}).
		Done().
		ParamObject("http_config", "New parameter name").
		Field("url", types.TypeString, "URL").
		Done().
		Build()

	if newName, exists := desc.Schema.DeprecatedParameters["httpConfig"]; !exists {
		t.Error("expected 'httpConfig' to be in DeprecatedParameters")
	} else if newName != "http_config" {
		t.Errorf("expected 'httpConfig' to map to 'http_config', got %q", newName)
	}
}

// TestArrayParamBuilder_Deprecation tests array parameter deprecation
func TestArrayParamBuilder_Deprecation(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamArray("allowedHosts", "Old parameter name").
		ElementType(types.TypeString).
		Deprecation(DeprecationInfo{ReplacedBy: "allowed_hosts"}).
		Done().
		ParamArray("allowed_hosts", "New parameter name").
		ElementType(types.TypeString).
		Done().
		Build()

	if newName, exists := desc.Schema.DeprecatedParameters["allowedHosts"]; !exists {
		t.Error("expected 'allowedHosts' to be in DeprecatedParameters")
	} else if newName != "allowed_hosts" {
		t.Errorf("expected 'allowedHosts' to map to 'allowed_hosts', got %q", newName)
	}
}
