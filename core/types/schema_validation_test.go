package types

import (
	"testing"
)

// TestParamEnumValidation verifies enum constraint validation
func TestParamEnumValidation(t *testing.T) {
	tests := []struct {
		name      string
		param     ParamSchema
		value     any
		wantValid bool
		wantError string
	}{
		{
			name: "valid string enum",
			param: ParamSchema{
				Name: "strategy",
				Type: TypeString,
				Enum: []any{"constant", "exponential", "linear"},
			},
			value:     "exponential",
			wantValid: true,
		},
		{
			name: "invalid string enum",
			param: ParamSchema{
				Name: "strategy",
				Type: TypeString,
				Enum: []any{"constant", "exponential", "linear"},
			},
			value:     "invalid",
			wantValid: false,
			wantError: "must be one of",
		},
		{
			name: "valid int enum",
			param: ParamSchema{
				Name: "port",
				Type: TypeInt,
				Enum: []any{80, 443, 8080},
			},
			value:     443,
			wantValid: true,
		},
		{
			name: "invalid int enum",
			param: ParamSchema{
				Name: "port",
				Type: TypeInt,
				Enum: []any{80, 443, 8080},
			},
			value:     9000,
			wantValid: false,
			wantError: "must be one of",
		},
		{
			name: "no enum constraint",
			param: ParamSchema{
				Name: "value",
				Type: TypeString,
				Enum: nil,
			},
			value:     "anything",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.ValidateEnum(tt.value)
			if tt.wantValid {
				if err != nil {
					t.Errorf("ValidateEnum() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Error("ValidateEnum() error = nil, want error")
				} else if tt.wantError != "" && !contains(err.Error(), tt.wantError) {
					t.Errorf("ValidateEnum() error = %q, want to contain %q", err.Error(), tt.wantError)
				}
			}
		})
	}
}

// TestParamRangeValidation verifies min/max constraint validation
func TestParamRangeValidation(t *testing.T) {
	min1 := 1.0
	max10 := 10.0
	min0 := 0.0
	max100 := 100.0

	tests := []struct {
		name      string
		param     ParamSchema
		value     any
		wantValid bool
		wantError string
	}{
		{
			name: "valid int in range",
			param: ParamSchema{
				Name:    "attempts",
				Type:    TypeInt,
				Minimum: &min1,
				Maximum: &max10,
			},
			value:     5,
			wantValid: true,
		},
		{
			name: "int below minimum",
			param: ParamSchema{
				Name:    "attempts",
				Type:    TypeInt,
				Minimum: &min1,
				Maximum: &max10,
			},
			value:     0,
			wantValid: false,
			wantError: "must be >= 1",
		},
		{
			name: "int above maximum",
			param: ParamSchema{
				Name:    "attempts",
				Type:    TypeInt,
				Minimum: &min1,
				Maximum: &max10,
			},
			value:     11,
			wantValid: false,
			wantError: "must be <= 10",
		},
		{
			name: "valid float in range",
			param: ParamSchema{
				Name:    "ratio",
				Type:    TypeFloat,
				Minimum: &min0,
				Maximum: &max100,
			},
			value:     50.5,
			wantValid: true,
		},
		{
			name: "float below minimum",
			param: ParamSchema{
				Name:    "ratio",
				Type:    TypeFloat,
				Minimum: &min0,
				Maximum: &max100,
			},
			value:     -1.0,
			wantValid: false,
			wantError: "must be >= 0",
		},
		{
			name: "no range constraint",
			param: ParamSchema{
				Name:    "value",
				Type:    TypeInt,
				Minimum: nil,
				Maximum: nil,
			},
			value:     999999,
			wantValid: true,
		},
		{
			name: "only minimum constraint",
			param: ParamSchema{
				Name:    "value",
				Type:    TypeInt,
				Minimum: &min1,
				Maximum: nil,
			},
			value:     1000,
			wantValid: true,
		},
		{
			name: "only maximum constraint",
			param: ParamSchema{
				Name:    "value",
				Type:    TypeInt,
				Minimum: nil,
				Maximum: &max10,
			},
			value:     5,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.ValidateRange(tt.value)
			if tt.wantValid {
				if err != nil {
					t.Errorf("ValidateRange() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Error("ValidateRange() error = nil, want error")
				} else if tt.wantError != "" && !contains(err.Error(), tt.wantError) {
					t.Errorf("ValidateRange() error = %q, want to contain %q", err.Error(), tt.wantError)
				}
			}
		})
	}
}

// TestParamPatternValidation verifies regex pattern validation
func TestParamPatternValidation(t *testing.T) {
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	semverPattern := `^v?\d+\.\d+\.\d+$`

	tests := []struct {
		name      string
		param     ParamSchema
		value     any
		wantValid bool
		wantError string
	}{
		{
			name: "valid email pattern",
			param: ParamSchema{
				Name:    "email",
				Type:    TypeString,
				Pattern: &emailPattern,
			},
			value:     "user@example.com",
			wantValid: true,
		},
		{
			name: "invalid email pattern",
			param: ParamSchema{
				Name:    "email",
				Type:    TypeString,
				Pattern: &emailPattern,
			},
			value:     "not-an-email",
			wantValid: false,
			wantError: "must match pattern",
		},
		{
			name: "valid semver pattern",
			param: ParamSchema{
				Name:    "version",
				Type:    TypeString,
				Pattern: &semverPattern,
			},
			value:     "v1.2.3",
			wantValid: true,
		},
		{
			name: "invalid semver pattern",
			param: ParamSchema{
				Name:    "version",
				Type:    TypeString,
				Pattern: &semverPattern,
			},
			value:     "1.2",
			wantValid: false,
			wantError: "must match pattern",
		},
		{
			name: "no pattern constraint",
			param: ParamSchema{
				Name:    "value",
				Type:    TypeString,
				Pattern: nil,
			},
			value:     "anything goes",
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.ValidatePattern(tt.value)
			if tt.wantValid {
				if err != nil {
					t.Errorf("ValidatePattern() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Error("ValidatePattern() error = nil, want error")
				} else if tt.wantError != "" && !contains(err.Error(), tt.wantError) {
					t.Errorf("ValidatePattern() error = %q, want to contain %q", err.Error(), tt.wantError)
				}
			}
		})
	}
}

// TestSchemaBuilderWithValidation verifies SchemaBuilder supports new validation
func TestSchemaBuilderWithValidation(t *testing.T) {
	min1 := 1.0
	max10 := 10.0
	pattern := `^\d+$`

	schema := NewSchema("test", KindValue).
		Param("attempts", TypeInt).
		Minimum(&min1).
		Maximum(&max10).
		Done().
		Param("strategy", TypeString).
		Enum([]any{"constant", "exponential", "linear"}).
		Done().
		Param("code", TypeString).
		Pattern(&pattern).
		Done().
		Build()

	// Verify attempts param has min/max
	attemptsParam, exists := schema.Parameters["attempts"]
	if !exists {
		t.Fatal("attempts parameter not found")
	}
	if attemptsParam.Minimum == nil || *attemptsParam.Minimum != 1.0 {
		t.Errorf("attempts.Minimum: got %v, want 1.0", attemptsParam.Minimum)
	}
	if attemptsParam.Maximum == nil || *attemptsParam.Maximum != 10.0 {
		t.Errorf("attempts.Maximum: got %v, want 10.0", attemptsParam.Maximum)
	}

	// Verify strategy param has enum
	strategyParam, exists := schema.Parameters["strategy"]
	if !exists {
		t.Fatal("strategy parameter not found")
	}
	if len(strategyParam.Enum) != 3 {
		t.Errorf("strategy.Enum length: got %d, want 3", len(strategyParam.Enum))
	}

	// Verify code param has pattern
	codeParam, exists := schema.Parameters["code"]
	if !exists {
		t.Fatal("code parameter not found")
	}
	if codeParam.Pattern == nil || *codeParam.Pattern != `^\d+$` {
		t.Errorf("code.Pattern: got %v, want %q", codeParam.Pattern, `^\d+$`)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPrimaryParamIsAlwaysFirst verifies primary param is always first in order
func TestPrimaryParamIsAlwaysFirst(t *testing.T) {
	// Build schema with primary param declared AFTER other params
	schema := NewSchema("test", KindValue).
		Param("other1", TypeString).Done().
		Param("other2", TypeInt).Done().
		PrimaryParam("primary", TypeString, "Primary parameter").
		Param("other3", TypeBool).Done().
		Build()

	// Primary param should be first in order
	if len(schema.ParameterOrder) < 1 {
		t.Fatal("ParameterOrder is empty")
	}

	if schema.ParameterOrder[0] != "primary" {
		t.Errorf("First parameter: got %q, want %q", schema.ParameterOrder[0], "primary")
	}

	// Verify all params are present
	expectedOrder := []string{"primary", "other1", "other2", "other3"}
	if len(schema.ParameterOrder) != len(expectedOrder) {
		t.Fatalf("ParameterOrder length: got %d, want %d", len(schema.ParameterOrder), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if schema.ParameterOrder[i] != expected {
			t.Errorf("ParameterOrder[%d]: got %q, want %q", i, schema.ParameterOrder[i], expected)
		}
	}
}
