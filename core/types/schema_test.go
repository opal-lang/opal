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

func TestIOCapabilityWithFlags(t *testing.T) {
	tests := []struct {
		name         string
		flags        []IOFlag
		expectStdin  bool
		expectStdout bool
		expectScrub  bool
	}{
		{
			name:         "shell: full I/O with scrubbing",
			flags:        []IOFlag{AcceptsStdin, ProducesStdout, ScrubByDefault},
			expectStdin:  true,
			expectStdout: true,
			expectScrub:  true,
		},
		{
			name:         "file.write: stdin only with scrubbing",
			flags:        []IOFlag{AcceptsStdin, ScrubByDefault},
			expectStdin:  true,
			expectStdout: false,
			expectScrub:  true,
		},
		{
			name:         "http.get: stdout only with scrubbing",
			flags:        []IOFlag{ProducesStdout, ScrubByDefault},
			expectStdin:  false,
			expectStdout: true,
			expectScrub:  true,
		},
		{
			name:         "image.process: binary data, no scrubbing",
			flags:        []IOFlag{AcceptsStdin, ProducesStdout},
			expectStdin:  true,
			expectStdout: true,
			expectScrub:  false,
		},
		{
			name:         "order doesn't matter",
			flags:        []IOFlag{ScrubByDefault, ProducesStdout, AcceptsStdin},
			expectStdin:  true,
			expectStdout: true,
			expectScrub:  true,
		},
		{
			name:         "empty flags",
			flags:        []IOFlag{},
			expectStdin:  false,
			expectStdout: false,
			expectScrub:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewSchema("test", KindExecution).
				Description("Test decorator").
				WithIO(tt.flags...).
				Build()

			if schema.IO == nil {
				t.Fatal("expected IO capability to be set")
			}

			if schema.IO.SupportsStdin != tt.expectStdin {
				t.Errorf("SupportsStdin: expected %v, got %v", tt.expectStdin, schema.IO.SupportsStdin)
			}
			if schema.IO.SupportsStdout != tt.expectStdout {
				t.Errorf("SupportsStdout: expected %v, got %v", tt.expectStdout, schema.IO.SupportsStdout)
			}
			if schema.IO.DefaultScrub != tt.expectScrub {
				t.Errorf("DefaultScrub: expected %v, got %v", tt.expectScrub, schema.IO.DefaultScrub)
			}
		})
	}
}

func TestIOCapabilityNotSet(t *testing.T) {
	// Decorators that don't call WithIO should have nil IO capability
	schema := NewSchema("retry", KindExecution).
		Description("Retry decorator").
		RequiresBlock().
		Build()

	if schema.IO != nil {
		t.Error("expected IO capability to be nil for decorator without WithIO")
	}
}

func TestIOCapabilityScrubParameter(t *testing.T) {
	tests := []struct {
		name            string
		flags           []IOFlag
		expectedDefault ScrubMode
		expectedStdin   bool
		expectedStdout  bool
	}{
		{
			name:            "shell: no scrubbing by default (bash-compatible)",
			flags:           []IOFlag{AcceptsStdin, ProducesStdout},
			expectedDefault: ScrubNone,
			expectedStdin:   true,
			expectedStdout:  true,
		},
		{
			name:            "shell with ScrubByDefault: scrub both",
			flags:           []IOFlag{AcceptsStdin, ProducesStdout, ScrubByDefault},
			expectedDefault: ScrubBoth,
			expectedStdin:   true,
			expectedStdout:  true,
		},
		{
			name:            "file.write: scrub stdin only",
			flags:           []IOFlag{AcceptsStdin, ScrubByDefault},
			expectedDefault: ScrubStdin,
			expectedStdin:   true,
			expectedStdout:  false,
		},
		{
			name:            "http.get: scrub stdout only",
			flags:           []IOFlag{ProducesStdout, ScrubByDefault},
			expectedDefault: ScrubStdout,
			expectedStdin:   false,
			expectedStdout:  true,
		},
		{
			name:            "image.process: no scrubbing (binary data)",
			flags:           []IOFlag{AcceptsStdin, ProducesStdout},
			expectedDefault: ScrubNone,
			expectedStdin:   true,
			expectedStdout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewSchema("test", KindExecution).
				Description("Test decorator").
				WithIO(tt.flags...).
				Build()

			// Verify IO capability
			if schema.IO == nil {
				t.Fatal("expected IO capability to be set")
			}
			if schema.IO.SupportsStdin != tt.expectedStdin {
				t.Errorf("SupportsStdin: expected %v, got %v", tt.expectedStdin, schema.IO.SupportsStdin)
			}
			if schema.IO.SupportsStdout != tt.expectedStdout {
				t.Errorf("SupportsStdout: expected %v, got %v", tt.expectedStdout, schema.IO.SupportsStdout)
			}

			// Verify scrub parameter was added
			scrubParam, exists := schema.Parameters["scrub"]
			if !exists {
				t.Fatal("expected scrub parameter to be added automatically")
			}

			// Verify scrub parameter type
			if scrubParam.Type != TypeScrubMode {
				t.Errorf("scrub parameter type: expected %v, got %v", TypeScrubMode, scrubParam.Type)
			}

			// Verify scrub parameter default
			if scrubParam.Default != string(tt.expectedDefault) {
				t.Errorf("scrub parameter default: expected %q, got %q", tt.expectedDefault, scrubParam.Default)
			}

			// Verify scrub parameter enum values
			expectedEnum := []string{string(ScrubNone), string(ScrubStdin), string(ScrubStdout), string(ScrubBoth)}
			if len(scrubParam.Enum) != len(expectedEnum) {
				t.Errorf("scrub parameter enum: expected %v, got %v", expectedEnum, scrubParam.Enum)
			}

			// Verify scrub parameter is optional
			if scrubParam.Required {
				t.Error("scrub parameter should be optional")
			}
		})
	}
}

func TestIOCapabilityWithOpts(t *testing.T) {
	tests := []struct {
		name            string
		opts            IOOpts
		expectedDefault ScrubMode
		expectedStdin   bool
		expectedStdout  bool
	}{
		{
			name: "explicit scrub stdin only",
			opts: IOOpts{
				AcceptsStdin:     true,
				ProducesStdout:   true,
				DefaultScrubMode: ScrubStdin,
			},
			expectedDefault: ScrubStdin,
			expectedStdin:   true,
			expectedStdout:  true,
		},
		{
			name: "explicit scrub stdout only",
			opts: IOOpts{
				AcceptsStdin:     true,
				ProducesStdout:   true,
				DefaultScrubMode: ScrubStdout,
			},
			expectedDefault: ScrubStdout,
			expectedStdin:   true,
			expectedStdout:  true,
		},
		{
			name: "explicit scrub both",
			opts: IOOpts{
				AcceptsStdin:     true,
				ProducesStdout:   true,
				DefaultScrubMode: ScrubBoth,
			},
			expectedDefault: ScrubBoth,
			expectedStdin:   true,
			expectedStdout:  true,
		},
		{
			name: "explicit scrub none",
			opts: IOOpts{
				AcceptsStdin:     true,
				ProducesStdout:   true,
				DefaultScrubMode: ScrubNone,
			},
			expectedDefault: ScrubNone,
			expectedStdin:   true,
			expectedStdout:  true,
		},
		{
			name: "empty default scrub mode defaults to none",
			opts: IOOpts{
				AcceptsStdin:   true,
				ProducesStdout: true,
			},
			expectedDefault: ScrubNone,
			expectedStdin:   true,
			expectedStdout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewSchema("test", KindExecution).
				Description("Test decorator").
				WithIOOpts(tt.opts).
				Build()

			// Verify IO capability
			if schema.IO == nil {
				t.Fatal("expected IO capability to be set")
			}
			if schema.IO.SupportsStdin != tt.expectedStdin {
				t.Errorf("SupportsStdin: expected %v, got %v", tt.expectedStdin, schema.IO.SupportsStdin)
			}
			if schema.IO.SupportsStdout != tt.expectedStdout {
				t.Errorf("SupportsStdout: expected %v, got %v", tt.expectedStdout, schema.IO.SupportsStdout)
			}

			// Verify scrub parameter
			scrubParam, exists := schema.Parameters["scrub"]
			if !exists {
				t.Fatal("expected scrub parameter to be added automatically")
			}
			if scrubParam.Default != string(tt.expectedDefault) {
				t.Errorf("scrub parameter default: expected %q, got %q", tt.expectedDefault, scrubParam.Default)
			}
		})
	}
}

// TestEnumSchema tests EnumSchema type
func TestEnumSchema(t *testing.T) {
	defaultVal := "option1"
	enumSchema := &EnumSchema{
		Values:  []string{"option1", "option2", "option3"},
		Default: &defaultVal,
		DeprecatedValues: map[string]string{
			"old_option": "option1",
		},
	}

	// Verify values
	if len(enumSchema.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(enumSchema.Values))
	}
	if enumSchema.Values[0] != "option1" {
		t.Errorf("expected first value 'option1', got %q", enumSchema.Values[0])
	}

	// Verify default
	if enumSchema.Default == nil {
		t.Fatal("expected default to be set")
	}
	if *enumSchema.Default != "option1" {
		t.Errorf("expected default 'option1', got %q", *enumSchema.Default)
	}

	// Verify deprecated values
	if len(enumSchema.DeprecatedValues) != 1 {
		t.Errorf("expected 1 deprecated value, got %d", len(enumSchema.DeprecatedValues))
	}
	if enumSchema.DeprecatedValues["old_option"] != "option1" {
		t.Errorf("expected deprecated 'old_option' -> 'option1', got %q", enumSchema.DeprecatedValues["old_option"])
	}
}

// TestObjectSchema tests ObjectSchema type
func TestObjectSchema(t *testing.T) {
	objectSchema := &ObjectSchema{
		Fields: map[string]ParamSchema{
			"name": {
				Name:        "name",
				Type:        TypeString,
				Description: "User name",
				Required:    true,
			},
			"age": {
				Name:        "age",
				Type:        TypeInt,
				Description: "User age",
				Required:    false,
			},
		},
		Required:             []string{"name"},
		AdditionalProperties: false,
	}

	// Verify fields
	if len(objectSchema.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(objectSchema.Fields))
	}

	// Verify name field
	nameField, exists := objectSchema.Fields["name"]
	if !exists {
		t.Fatal("expected 'name' field to exist")
	}
	if nameField.Type != TypeString {
		t.Errorf("expected name type 'string', got %q", nameField.Type)
	}
	if !nameField.Required {
		t.Error("expected name field to be required")
	}

	// Verify age field
	ageField, exists := objectSchema.Fields["age"]
	if !exists {
		t.Fatal("expected 'age' field to exist")
	}
	if ageField.Type != TypeInt {
		t.Errorf("expected age type 'integer', got %q", ageField.Type)
	}
	if ageField.Required {
		t.Error("expected age field to be optional")
	}

	// Verify required list
	if len(objectSchema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(objectSchema.Required))
	}
	if objectSchema.Required[0] != "name" {
		t.Errorf("expected required field 'name', got %q", objectSchema.Required[0])
	}

	// Verify closed object (no additional properties)
	if objectSchema.AdditionalProperties {
		t.Error("expected AdditionalProperties to be false (closed object)")
	}
}

// TestArraySchema tests ArraySchema type
func TestArraySchema(t *testing.T) {
	minLen := 1
	maxLen := 10
	arraySchema := &ArraySchema{
		ElementType: TypeString,
		MinLength:   &minLen,
		MaxLength:   &maxLen,
		UniqueItems: true,
	}

	// Verify element type
	if arraySchema.ElementType != TypeString {
		t.Errorf("expected element type 'string', got %q", arraySchema.ElementType)
	}

	// Verify min length
	if arraySchema.MinLength == nil {
		t.Fatal("expected MinLength to be set")
	}
	if *arraySchema.MinLength != 1 {
		t.Errorf("expected MinLength 1, got %d", *arraySchema.MinLength)
	}

	// Verify max length
	if arraySchema.MaxLength == nil {
		t.Fatal("expected MaxLength to be set")
	}
	if *arraySchema.MaxLength != 10 {
		t.Errorf("expected MaxLength 10, got %d", *arraySchema.MaxLength)
	}

	// Verify unique items
	if !arraySchema.UniqueItems {
		t.Error("expected UniqueItems to be true")
	}
}

// TestArraySchemaWithComplexElements tests ArraySchema with object elements
func TestArraySchemaWithComplexElements(t *testing.T) {
	arraySchema := &ArraySchema{
		ElementType: TypeObject,
		ElementSchema: &ParamSchema{
			Name: "item",
			Type: TypeObject,
			ObjectSchema: &ObjectSchema{
				Fields: map[string]ParamSchema{
					"id": {
						Name:     "id",
						Type:     TypeString,
						Required: true,
					},
					"value": {
						Name:     "value",
						Type:     TypeInt,
						Required: false,
					},
				},
				Required:             []string{"id"},
				AdditionalProperties: false,
			},
		},
		UniqueItems: false,
	}

	// Verify element type
	if arraySchema.ElementType != TypeObject {
		t.Errorf("expected element type 'object', got %q", arraySchema.ElementType)
	}

	// Verify element schema
	if arraySchema.ElementSchema == nil {
		t.Fatal("expected ElementSchema to be set")
	}
	if arraySchema.ElementSchema.Type != TypeObject {
		t.Errorf("expected element schema type 'object', got %q", arraySchema.ElementSchema.Type)
	}

	// Verify object schema
	if arraySchema.ElementSchema.ObjectSchema == nil {
		t.Fatal("expected ObjectSchema to be set")
	}
	if len(arraySchema.ElementSchema.ObjectSchema.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(arraySchema.ElementSchema.ObjectSchema.Fields))
	}
}

// TestParamSchemaWithEnumSchema tests ParamSchema with EnumSchema
func TestParamSchemaWithEnumSchema(t *testing.T) {
	defaultVal := "read"
	param := ParamSchema{
		Name:        "mode",
		Type:        TypeEnum,
		Description: "Operation mode",
		Required:    false,
		EnumSchema: &EnumSchema{
			Values:  []string{"read", "write", "append"},
			Default: &defaultVal,
		},
	}

	// Verify basic fields
	if param.Type != TypeEnum {
		t.Errorf("expected type 'enum', got %q", param.Type)
	}

	// Verify enum schema
	if param.EnumSchema == nil {
		t.Fatal("expected EnumSchema to be set")
	}
	if len(param.EnumSchema.Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(param.EnumSchema.Values))
	}
	if *param.EnumSchema.Default != "read" {
		t.Errorf("expected default 'read', got %q", *param.EnumSchema.Default)
	}
}

// TestParamSchemaWithObjectSchema tests ParamSchema with ObjectSchema
func TestParamSchemaWithObjectSchema(t *testing.T) {
	param := ParamSchema{
		Name:        "config",
		Type:        TypeObject,
		Description: "Configuration object",
		Required:    true,
		ObjectSchema: &ObjectSchema{
			Fields: map[string]ParamSchema{
				"host": {
					Name:     "host",
					Type:     TypeString,
					Required: true,
				},
				"port": {
					Name:     "port",
					Type:     TypeInt,
					Required: false,
					Default:  8080,
				},
			},
			Required:             []string{"host"},
			AdditionalProperties: false,
		},
	}

	// Verify basic fields
	if param.Type != TypeObject {
		t.Errorf("expected type 'object', got %q", param.Type)
	}

	// Verify object schema
	if param.ObjectSchema == nil {
		t.Fatal("expected ObjectSchema to be set")
	}
	if len(param.ObjectSchema.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(param.ObjectSchema.Fields))
	}

	// Verify closed object
	if param.ObjectSchema.AdditionalProperties {
		t.Error("expected closed object (AdditionalProperties=false)")
	}
}

// TestParamSchemaWithArraySchema tests ParamSchema with ArraySchema
func TestParamSchemaWithArraySchema(t *testing.T) {
	minLen := 1
	param := ParamSchema{
		Name:        "tags",
		Type:        TypeArray,
		Description: "List of tags",
		Required:    false,
		ArraySchema: &ArraySchema{
			ElementType: TypeString,
			MinLength:   &minLen,
			UniqueItems: true,
		},
	}

	// Verify basic fields
	if param.Type != TypeArray {
		t.Errorf("expected type 'array', got %q", param.Type)
	}

	// Verify array schema
	if param.ArraySchema == nil {
		t.Fatal("expected ArraySchema to be set")
	}
	if param.ArraySchema.ElementType != TypeString {
		t.Errorf("expected element type 'string', got %q", param.ArraySchema.ElementType)
	}
	if !param.ArraySchema.UniqueItems {
		t.Error("expected UniqueItems to be true")
	}
}

// TestTypeEnumConstant tests that TypeEnum is a valid ParamType
func TestTypeEnumConstant(t *testing.T) {
	if !isValidParamType(TypeEnum) {
		t.Error("TypeEnum should be a valid ParamType")
	}
}

// TestStringConstraints tests MinLength and MaxLength on ParamSchema
func TestStringConstraints(t *testing.T) {
	minLen := 3
	maxLen := 50
	param := ParamSchema{
		Name:      "username",
		Type:      TypeString,
		MinLength: &minLen,
		MaxLength: &maxLen,
	}

	// Verify constraints
	if param.MinLength == nil {
		t.Fatal("expected MinLength to be set")
	}
	if *param.MinLength != 3 {
		t.Errorf("expected MinLength 3, got %d", *param.MinLength)
	}

	if param.MaxLength == nil {
		t.Fatal("expected MaxLength to be set")
	}
	if *param.MaxLength != 50 {
		t.Errorf("expected MaxLength 50, got %d", *param.MaxLength)
	}
}

// TestParamSchemaWithFormat tests ParamSchema with Format field
func TestParamSchemaWithFormat(t *testing.T) {
	uriFormat := FormatURI
	param := ParamSchema{
		Name:        "endpoint",
		Type:        TypeString,
		Description: "API endpoint URL",
		Required:    true,
		Format:      &uriFormat,
	}

	// Verify basic fields
	if param.Type != TypeString {
		t.Errorf("expected type 'string', got %q", param.Type)
	}

	// Verify format
	if param.Format == nil {
		t.Fatal("expected Format to be set")
	}
	if *param.Format != FormatURI {
		t.Errorf("expected format 'uri', got %q", *param.Format)
	}
}

// TestParamBuilderFormat tests Format() method on ParamBuilder
func TestParamBuilderFormat(t *testing.T) {
	schema := NewSchema("test", KindValue).
		Param("url", TypeString).
		Description("URL parameter").
		Format(FormatURI).
		Required().
		Done().
		Build()

	// Verify parameter exists
	param, exists := schema.Parameters["url"]
	if !exists {
		t.Fatal("parameter 'url' not found")
	}

	// Verify format is set
	if param.Format == nil {
		t.Fatal("expected Format to be set")
	}
	if *param.Format != FormatURI {
		t.Errorf("expected format 'uri', got %q", *param.Format)
	}
}
