package stdlib

import (
	"fmt"
	"testing"
)

func TestDecoratorRegistry(t *testing.T) {
	// Test standard decorators - only var and parallel exist now
	tests := []struct {
		name     string
		expected DecoratorSignature
	}{
		{
			name: "var",
			expected: DecoratorSignature{
				Name:     "var",
				Type:     FunctionDecorator,
				Semantic: SemVariable,
			},
		},
		{
			name: "parallel",
			expected: DecoratorSignature{
				Name:     "parallel",
				Type:     BlockDecorator,
				Semantic: SemDecorator,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decorator, exists := GetDecorator(test.name)
			if !exists {
				t.Errorf("Expected decorator %s to exist", test.name)
				return
			}

			if decorator.Name != test.expected.Name {
				t.Errorf("Expected name %s, got %s", test.expected.Name, decorator.Name)
			}
			if decorator.Type != test.expected.Type {
				t.Errorf("Expected type %v, got %v", test.expected.Type, decorator.Type)
			}
			if decorator.Semantic != test.expected.Semantic {
				t.Errorf("Expected semantic %v, got %v", test.expected.Semantic, decorator.Semantic)
			}
		})
	}
}

func TestSemanticTypes(t *testing.T) {
	tests := []struct {
		name     string
		expected SemanticType
	}{
		{"var", SemVariable},
		{"parallel", SemDecorator},
		{"unknown", SemDecorator}, // Default fallback
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			semanticType := GetDecoratorSemanticType(test.name)
			if semanticType != test.expected {
				t.Errorf("Expected semantic type %v for %s, got %v", test.expected, test.name, semanticType)
			}
		})
	}
}

func TestCustomDecoratorRegistration(t *testing.T) {
	// Register a custom function decorator
	customFunctionDecorator := &DecoratorSignature{
		Name:        "custom-func",
		Type:        FunctionDecorator,
		Semantic:    SemVariable,
		Description: "Custom function test decorator",
		Args: []ArgumentSpec{
			{Name: "value", Type: StringArg, Optional: false},
		},
	}

	RegisterDecorator(customFunctionDecorator)

	// Test that it was registered
	if !IsValidDecorator("custom-func") {
		t.Error("Custom function decorator should be valid after registration")
	}

	if !IsFunctionDecorator("custom-func") {
		t.Error("Custom decorator should be a function decorator")
	}

	if GetDecoratorSemanticType("custom-func") != SemVariable {
		t.Error("Custom function decorator should have SemVariable semantic type")
	}

	// Register a custom block decorator
	customBlockDecorator := &DecoratorSignature{
		Name:          "custom-block",
		Type:          BlockDecorator,
		Semantic:      SemDecorator,
		Description:   "Custom block test decorator",
		RequiresBlock: true,
		Args: []ArgumentSpec{
			{Name: "timeout", Type: DurationArg, Optional: true, Default: "30s"},
		},
	}

	RegisterDecorator(customBlockDecorator)

	// Test that it was registered
	if !IsValidDecorator("custom-block") {
		t.Error("Custom block decorator should be valid after registration")
	}

	if !IsBlockDecorator("custom-block") {
		t.Error("Custom decorator should be a block decorator")
	}

	if !RequiresExplicitBlock("custom-block") {
		t.Error("Custom block decorator should require explicit block")
	}

	// Test retrieval
	decorator, exists := GetDecorator("custom-block")
	if !exists {
		t.Error("Custom block decorator should exist")
	}

	if decorator.Name != "custom-block" {
		t.Errorf("Expected name 'custom-block', got %s", decorator.Name)
	}

	if !decorator.RequiresBlock {
		t.Error("Custom block decorator should require block")
	}
}

func TestDecoratorOverride(t *testing.T) {
	// Override an existing decorator
	originalDecorator, exists := GetDecorator("var")
	if !exists {
		t.Fatal("Expected var decorator to exist")
	}

	// Register a modified version
	modifiedDecorator := &DecoratorSignature{
		Name:        "var",
		Type:        FunctionDecorator,
		Semantic:    SemFunction, // Changed semantic type
		Description: "Modified variable decorator",
		Args: []ArgumentSpec{
			{Name: "name", Type: IdentifierArg, Optional: false},
			{Name: "default", Type: StringArg, Optional: true}, // Added optional arg
		},
	}

	RegisterDecorator(modifiedDecorator)

	// Test that it was overridden
	newDecorator, exists := GetDecorator("var")
	if !exists {
		t.Error("var decorator should still exist after override")
	}

	if newDecorator.Semantic != SemFunction {
		t.Error("var decorator should have new semantic type after override")
	}

	if len(newDecorator.Args) != 2 {
		t.Errorf("Expected 2 arguments after override, got %d", len(newDecorator.Args))
	}

	if newDecorator.Description != "Modified variable decorator" {
		t.Error("Description should be updated after override")
	}

	// Restore original for other tests
	RegisterDecorator(originalDecorator)
}

func TestDecoratorTypeFiltering(t *testing.T) {
	// Test getting decorators by type
	functionDecorators := GetFunctionDecorators()
	blockDecorators := GetBlockDecorators()

	// Should have at least the standard ones
	if len(functionDecorators) < 1 { // var only
		t.Errorf("Expected at least 1 function decorators, got %d", len(functionDecorators))
	}

	if len(blockDecorators) < 1 { // parallel only
		t.Errorf("Expected at least 1 block decorators, got %d", len(blockDecorators))
	}

	// Test semantic filtering
	variableDecorators := GetDecoratorsBySemanticType(SemVariable)
	if len(variableDecorators) < 1 { // var (and potentially custom from previous test)
		t.Errorf("Expected at least 1 variable decorators, got %d", len(variableDecorators))
	}

	decoratorDecorators := GetDecoratorsBySemanticType(SemDecorator)
	if len(decoratorDecorators) < 1 { // parallel
		t.Errorf("Expected at least 1 decorator-type decorators, got %d", len(decoratorDecorators))
	}
}

func TestDecoratorValidation(t *testing.T) {
	// Test argument validation - only testing current decorators
	tests := []struct {
		name        string
		args        []string
		hasBlock    bool
		expectError bool
		description string
	}{
		{"var", []string{"PORT"}, false, false, "Valid var usage"},
		{"var", []string{}, false, true, "Missing required arg for var"},
		{"parallel", []string{}, true, false, "Valid parallel usage"},
		{"parallel", []string{}, false, true, "parallel missing required block"},
		{"unknown", []string{}, false, true, "Unknown decorator"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := ValidateDecorator(test.name, test.args, test.hasBlock)
			if test.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", test.name)
			}
			if !test.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", test.name, err)
			}
		})
	}
}

func TestDecoratorDocumentation(t *testing.T) {
	// Test that documentation can be generated
	doc := GetDecoratorDocumentation()
	if len(doc) == 0 {
		t.Error("Expected non-empty documentation")
	}

	// Test individual decorator documentation
	decorator, exists := GetDecorator("var")
	if !exists {
		t.Error("Expected var decorator to exist")
		return
	}

	usage := decorator.GetUsageString()
	if usage == "" {
		t.Error("Expected non-empty usage string")
	}

	expectedUsage := "@var(name)"
	if usage != expectedUsage {
		t.Errorf("Expected usage %s, got %s", expectedUsage, usage)
	}

	docString := decorator.GetDocumentationString()
	if docString == "" {
		t.Error("Expected non-empty documentation string")
	}

	// Test block decorator with RequiresBlock
	parallelDecorator, exists := GetDecorator("parallel")
	if !exists {
		t.Error("Expected parallel decorator to exist")
		return
	}

	parallelUsage := parallelDecorator.GetUsageString()
	expectedParallelUsage := "@parallel { ... }"
	if parallelUsage != expectedParallelUsage {
		t.Errorf("Expected parallel usage %s, got %s", expectedParallelUsage, parallelUsage)
	}
}

func TestConcurrentRegistration(t *testing.T) {
	// Test that concurrent registration is safe
	done := make(chan bool, 10)

	// Launch multiple goroutines to register decorators
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			decorator := &DecoratorSignature{
				Name:        fmt.Sprintf("concurrent%d", id),
				Type:        FunctionDecorator,
				Semantic:    SemDecorator,
				Description: "Concurrent test decorator",
				Args:        []ArgumentSpec{},
			}

			RegisterDecorator(decorator)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all decorators were registered
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("concurrent%d", i)
		if !IsValidDecorator(name) {
			t.Errorf("Decorator %s should be valid after concurrent registration", name)
		}
	}
}

func TestDecoratorArguments(t *testing.T) {
	// Register a decorator with complex arguments for testing
	complexDecorator := &DecoratorSignature{
		Name:          "complex",
		Type:          BlockDecorator,
		Semantic:      SemDecorator,
		Description:   "Complex decorator for testing",
		RequiresBlock: true,
		Args: []ArgumentSpec{
			{Name: "required1", Type: StringArg, Optional: false},
			{Name: "required2", Type: NumberArg, Optional: false},
			{Name: "optional1", Type: DurationArg, Optional: true, Default: "1s"},
			{Name: "optional2", Type: BooleanArg, Optional: true, Default: "false"},
		},
	}

	RegisterDecorator(complexDecorator)

	// Test argument validation
	argTests := []struct {
		args        []string
		expectError bool
		description string
	}{
		{[]string{}, true, "Missing all required args"},
		{[]string{"value1"}, true, "Missing one required arg"},
		{[]string{"value1", "42"}, false, "All required args provided"},
		{[]string{"value1", "42", "2s"}, false, "Required + one optional"},
		{[]string{"value1", "42", "2s", "true"}, false, "All args provided"},
		{[]string{"value1", "42", "2s", "true", "extra"}, true, "Too many args"},
	}

	for _, test := range argTests {
		t.Run(test.description, func(t *testing.T) {
			err := ValidateDecorator("complex", test.args, true)
			if test.expectError && err == nil {
				t.Errorf("Expected error for args %v, got nil", test.args)
			}
			if !test.expectError && err != nil {
				t.Errorf("Expected no error for args %v, got %v", test.args, err)
			}
		})
	}
}

func TestRegistryState(t *testing.T) {
	// Test that we can get all decorators and they include our registered ones
	allDecorators := GetAllDecorators()

	// Should have at least the standard ones: var, parallel
	if len(allDecorators) < 2 {
		t.Errorf("Expected at least 2 decorators, got %d", len(allDecorators))
	}

	// Test that we have the expected standard decorators
	standardNames := []string{"var", "parallel"}
	for _, name := range standardNames {
		if !IsValidDecorator(name) {
			t.Errorf("Expected standard decorator %s to be registered", name)
		}
	}

	// Test decorator counts by type
	functionCount := len(GetFunctionDecorators())
	blockCount := len(GetBlockDecorators())

	if functionCount < 1 {
		t.Errorf("Expected at least 1 function decorators, got %d", functionCount)
	}

	if blockCount < 1 {
		t.Errorf("Expected at least 1 block decorators, got %d", blockCount)
	}

	// Note: Total might be more than function + block due to custom decorators
	// registered in other tests, so we don't check exact equality
}

func TestMinimalDecoratorSet(t *testing.T) {
	// Test the exact minimal set we currently have
	expectedDecorators := map[string]struct {
		decoratorType DecoratorType
		semantic      SemanticType
		requiresBlock bool
	}{
		"var": {
			decoratorType: FunctionDecorator,
			semantic:      SemVariable,
			requiresBlock: false,
		},
		"parallel": {
			decoratorType: BlockDecorator,
			semantic:      SemDecorator,
			requiresBlock: true,
		},
	}

	for name, expected := range expectedDecorators {
		t.Run(name, func(t *testing.T) {
			decorator, exists := GetDecorator(name)
			if !exists {
				t.Errorf("Expected decorator %s to exist", name)
				return
			}

			if decorator.Type != expected.decoratorType {
				t.Errorf("Expected %s to be %v, got %v", name, expected.decoratorType, decorator.Type)
			}

			if decorator.Semantic != expected.semantic {
				t.Errorf("Expected %s semantic to be %v, got %v", name, expected.semantic, decorator.Semantic)
			}

			if decorator.RequiresBlock != expected.requiresBlock {
				t.Errorf("Expected %s RequiresBlock to be %v, got %v", name, expected.requiresBlock, decorator.RequiresBlock)
			}
		})
	}
}
