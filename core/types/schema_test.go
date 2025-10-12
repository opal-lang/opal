package types

import "testing"

func TestSchemaBuilderBasic(t *testing.T) {
	schema := NewSchema("env", KindValue).
		Description("Access environment variables").
		PrimaryParam("property", TypeString, "Env var name").
		Build()

	if schema.Path != "env" {
		t.Errorf("expected path 'env', got %q", schema.Path)
	}
	if schema.Kind != "value" {
		t.Errorf("expected kind 'value', got %q", schema.Kind)
	}
	if schema.Description != "Access environment variables" {
		t.Errorf("expected description, got %q", schema.Description)
	}
	if schema.PrimaryParameter != "property" {
		t.Errorf("expected primary parameter 'property', got %q", schema.PrimaryParameter)
	}
	if len(schema.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(schema.Parameters))
	}

	// Check primary param exists in parameters
	param, exists := schema.Parameters["property"]
	if !exists {
		t.Fatal("primary parameter 'property' not found in parameters map")
	}
	if param.Type != TypeString {
		t.Errorf("expected type 'string', got %q", param.Type)
	}
	if !param.Required {
		t.Error("primary parameter should be required")
	}
}

func TestSchemaWithMultipleParams(t *testing.T) {
	schema := NewSchema("retry", KindExecution).
		Description("Retry with backoff").
		Param("times", TypeInt).
		Description("Number of retries").
		Default(3).
		Done().
		Param("delay", TypeDuration).
		Description("Delay between retries").
		Default("1s").
		Examples("1s", "5s", "30s").
		Done().
		AcceptsBlock().
		Build()

	if schema.Path != "retry" {
		t.Errorf("expected path 'retry', got %q", schema.Path)
	}
	if schema.BlockRequirement != BlockOptional {
		t.Error("expected BlockRequirement to be BlockOptional")
	}
	if len(schema.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(schema.Parameters))
	}

	// Check times parameter
	times, exists := schema.Parameters["times"]
	if !exists {
		t.Fatal("parameter 'times' not found")
	}
	if times.Type != TypeInt {
		t.Errorf("expected type 'int', got %q", times.Type)
	}
	if times.Default != 3 {
		t.Errorf("expected default 3, got %v", times.Default)
	}
	if times.Required {
		t.Error("parameter with default should not be required")
	}

	// Check delay parameter
	delay, exists := schema.Parameters["delay"]
	if !exists {
		t.Fatal("parameter 'delay' not found")
	}
	if len(delay.Examples) != 3 {
		t.Errorf("expected 3 examples, got %d", len(delay.Examples))
	}
}

func TestSchemaWithReturns(t *testing.T) {
	schema := NewSchema("env", KindValue).
		PrimaryParam("property", TypeString, "Env var name").
		Returns(TypeString, "Environment variable value").
		Build()

	if schema.Returns == nil {
		t.Fatal("expected Returns to be set")
	}
	if schema.Returns.Type != "string" {
		t.Errorf("expected return type 'string', got %q", schema.Returns.Type)
	}
	if schema.Returns.Description != "Environment variable value" {
		t.Errorf("unexpected return description: %q", schema.Returns.Description)
	}
}

func TestValidateSchemaSuccess(t *testing.T) {
	schema := NewSchema("test", KindValue).
		PrimaryParam("prop", TypeString, "Test property").
		Build()

	err := ValidateSchema(schema)
	if err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

func TestValidateSchemaEmptyPath(t *testing.T) {
	schema := DecoratorSchema{
		Path: "",
		Kind: "value",
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestValidateSchemaInvalidKind(t *testing.T) {
	schema := DecoratorSchema{
		Path: "test",
		Kind: "invalid",
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for invalid type, got nil")
	}
}

func TestParameterDeclarationOrder(t *testing.T) {
	schema := NewSchema("retry", KindExecution).
		Description("Retry with backoff").
		Param("times", TypeInt).
		Description("Number of retries").
		Default(3).
		Done().
		Param("delay", TypeDuration).
		Description("Delay between retries").
		Default("1s").
		Done().
		Param("backoff", TypeString).
		Description("Backoff strategy").
		Default("exponential").
		Done().
		Build()

	// Get parameters in declaration order
	ordered := schema.GetOrderedParameters()

	// Should have 3 parameters
	if len(ordered) != 3 {
		t.Fatalf("expected 3 ordered parameters, got %d", len(ordered))
	}

	// Check order matches declaration order
	expectedOrder := []string{"times", "delay", "backoff"}
	for i, expected := range expectedOrder {
		if ordered[i].Name != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, ordered[i].Name)
		}
	}

	// Verify parameter details are correct
	if ordered[0].Type != TypeInt {
		t.Errorf("times: expected TypeInt, got %v", ordered[0].Type)
	}
	if ordered[1].Type != TypeDuration {
		t.Errorf("delay: expected TypeDuration, got %v", ordered[1].Type)
	}
	if ordered[2].Type != TypeString {
		t.Errorf("backoff: expected TypeString, got %v", ordered[2].Type)
	}
}

func TestParameterOrderWithPrimary(t *testing.T) {
	schema := NewSchema("env", KindValue).
		PrimaryParam("property", TypeString, "Env var name").
		Param("default", TypeString).
		Description("Default value").
		Optional().
		Done().
		Build()

	ordered := schema.GetOrderedParameters()

	// Should have 2 parameters: property (primary) and default
	if len(ordered) != 2 {
		t.Fatalf("expected 2 ordered parameters, got %d", len(ordered))
	}

	// Primary parameter should be first (declared first)
	if ordered[0].Name != "property" {
		t.Errorf("expected primary 'property' first, got %q", ordered[0].Name)
	}
	if ordered[1].Name != "default" {
		t.Errorf("expected 'default' second, got %q", ordered[1].Name)
	}
}

func TestParameterOrderEmpty(t *testing.T) {
	schema := NewSchema("parallel", KindExecution).
		Description("Execute in parallel").
		Build()

	ordered := schema.GetOrderedParameters()

	// Should have 0 parameters
	if len(ordered) != 0 {
		t.Errorf("expected 0 ordered parameters, got %d", len(ordered))
	}
}

func TestValidateSchemaPrimaryNotInParams(t *testing.T) {
	schema := DecoratorSchema{
		Path:             "test",
		Kind:             KindValue,
		PrimaryParameter: "missing",
		Parameters:       make(map[string]ParamSchema),
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for primary parameter not in parameters map")
	}
}

func TestRegisterWithSchema(t *testing.T) {
	r := NewRegistry()

	schema := NewSchema("test", "value").
		PrimaryParam("prop", "string", "Test property").
		Param("default", "string").
		Optional().
		Done().
		Build()

	handler := func(ctx Context, args Args) (Value, error) {
		return "test", nil
	}

	err := r.RegisterValueWithSchema(schema, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve schema
	retrieved, exists := r.GetSchema("test")
	if !exists {
		t.Fatal("schema not found after registration")
	}
	if retrieved.PrimaryParameter != "prop" {
		t.Errorf("expected primary parameter 'prop', got %q", retrieved.PrimaryParameter)
	}
	if len(retrieved.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(retrieved.Parameters))
	}
}

func TestRegisterWithSchemaWrongKind(t *testing.T) {
	r := NewRegistry()

	schema := NewSchema("test", KindExecution). // Wrong kind
							Build()

	handler := func(ctx Context, args Args) (Value, error) {
		return nil, nil
	}

	err := r.RegisterValueWithSchema(schema, handler)
	if err == nil {
		t.Error("expected error when registering execution schema with RegisterValueWithSchema")
	}
}

func TestParameterOrder(t *testing.T) {
	schema := NewSchema("retry", KindExecution).
		Param("times", TypeInt).Required().Done().
		Param("delay", TypeDuration).Required().Done().
		Param("backoff", TypeString).Optional().Done().
		Build()

	if len(schema.ParameterOrder) != 3 {
		t.Errorf("expected 3 parameters in order, got %d", len(schema.ParameterOrder))
	}

	// Check order matches declaration order
	expectedOrder := []string{"times", "delay", "backoff"}
	for i, expected := range expectedOrder {
		if i >= len(schema.ParameterOrder) {
			t.Errorf("missing parameter at index %d", i)
			continue
		}
		if schema.ParameterOrder[i] != expected {
			t.Errorf("parameter order[%d]: expected %q, got %q", i, expected, schema.ParameterOrder[i])
		}
	}
}

func TestPrimaryParameterFirst(t *testing.T) {
	schema := NewSchema("timeout", KindExecution).
		PrimaryParam("duration", TypeDuration, "Timeout duration").
		Param("onTimeout", TypeString).Optional().Done().
		Build()

	if len(schema.ParameterOrder) != 2 {
		t.Errorf("expected 2 parameters in order, got %d", len(schema.ParameterOrder))
	}

	// Primary parameter should be first
	if schema.ParameterOrder[0] != "duration" {
		t.Errorf("primary parameter should be first, got %q", schema.ParameterOrder[0])
	}
	if schema.ParameterOrder[1] != "onTimeout" {
		t.Errorf("second parameter should be 'onTimeout', got %q", schema.ParameterOrder[1])
	}
}

func TestParameterOrderValidation(t *testing.T) {
	// Schema with parameter order that doesn't match parameters map
	schema := DecoratorSchema{
		Path: "test",
		Kind: KindExecution,
		Parameters: map[string]ParamSchema{
			"a": {Name: "a", Type: TypeString},
			"b": {Name: "b", Type: TypeInt},
		},
		ParameterOrder: []string{"a", "b", "c"}, // "c" doesn't exist
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for parameter in order but not in parameters map")
	}
}
