package types

import "testing"

// TestMeasureSchemaDepth_BaseCase tests empty schema with no nesting
func TestMeasureSchemaDepth_BaseCase(t *testing.T) {
	schema := map[string]any{
		"type": "string",
	}

	depth := measureSchemaDepth(schema)
	if depth != 0 {
		t.Errorf("Expected depth 0, got %d", depth)
	}
}

// TestMeasureSchemaDepth_SingleLevel tests object with one level of properties
func TestMeasureSchemaDepth_SingleLevel(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
	}

	depth := measureSchemaDepth(schema)
	if depth != 1 {
		t.Errorf("Expected depth 1, got %d", depth)
	}
}

// TestMeasureSchemaDepth_TwoLevels tests nested object
func TestMeasureSchemaDepth_TwoLevels(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"user": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
				},
			},
		},
	}

	depth := measureSchemaDepth(schema)
	if depth != 2 {
		t.Errorf("Expected depth 2, got %d", depth)
	}
}

// TestMeasureSchemaDepth_ArrayItems tests array with nested items
func TestMeasureSchemaDepth_ArrayItems(t *testing.T) {
	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type": "string",
				},
			},
		},
	}

	depth := measureSchemaDepth(schema)
	if depth != 2 {
		t.Errorf("Expected depth 2 (items=1, properties=2), got %d", depth)
	}
}

// TestMeasureSchemaDepth_MultipleBranches tests max depth across multiple fields
func TestMeasureSchemaDepth_MultipleBranches(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shallow": map[string]any{
				"type": "string",
			},
			"deep": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nested": map[string]any{
						"type": "string",
					},
				},
			},
		},
	}

	depth := measureSchemaDepth(schema)
	if depth != 2 {
		t.Errorf("Expected depth 2 (max of shallow=1 and deep=2), got %d", depth)
	}
}

// TestMeasureSchemaDepth_AllOf tests schema combinators
func TestMeasureSchemaDepth_AllOf(t *testing.T) {
	schema := map[string]any{
		"allOf": []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
				},
			},
		},
	}

	depth := measureSchemaDepth(schema)
	if depth != 2 {
		t.Errorf("Expected depth 2 (allOf=1, properties=2), got %d", depth)
	}
}

// TestMeasureSchemaDepth_DeepNesting tests deeply nested schema
func TestMeasureSchemaDepth_DeepNesting(t *testing.T) {
	// Create schema with depth 5
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"level1": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"level2": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"level3": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"level4": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"level5": map[string]any{
												"type": "string",
											},
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

	depth := measureSchemaDepth(schema)
	if depth != 5 {
		t.Errorf("Expected depth 5, got %d", depth)
	}
}
