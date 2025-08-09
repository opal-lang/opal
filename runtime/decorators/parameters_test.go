package decorators

import (
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/types"
)

func TestResolvePositionalParameters_Basic(t *testing.T) {
	tests := []struct {
		name     string
		params   []ast.NamedParameter
		schema   []ParameterSchema
		expected []ast.NamedParameter
		wantErr  bool
	}{
		{
			name: "single positional parameter",
			params: []ast.NamedParameter{
				{Name: "", Value: &ast.Identifier{Name: "PORT"}},
			},
			schema: []ParameterSchema{
				{Name: "name", Type: ast.IdentifierType, Required: true},
			},
			expected: []ast.NamedParameter{
				{Name: "name", Value: &ast.Identifier{Name: "PORT"}},
			},
			wantErr: false,
		},
		{
			name: "multiple positional parameters",
			params: []ast.NamedParameter{
				{Name: "", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "", Value: &ast.StringLiteral{Value: "localhost"}},
			},
			schema: []ParameterSchema{
				{Name: "variable", Type: ast.StringType, Required: true},
				{Name: "default", Type: ast.StringType, Required: false},
			},
			expected: []ast.NamedParameter{
				{Name: "variable", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "default", Value: &ast.StringLiteral{Value: "localhost"}},
			},
			wantErr: false,
		},
		{
			name: "all named parameters (no change)",
			params: []ast.NamedParameter{
				{Name: "variable", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "default", Value: &ast.StringLiteral{Value: "localhost"}},
			},
			schema: []ParameterSchema{
				{Name: "variable", Type: ast.StringType, Required: true},
				{Name: "default", Type: ast.StringType, Required: false},
			},
			expected: []ast.NamedParameter{
				{Name: "variable", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "default", Value: &ast.StringLiteral{Value: "localhost"}},
			},
			wantErr: false,
		},
		{
			name: "mixed parameters (positional first)",
			params: []ast.NamedParameter{
				{Name: "", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "default", Value: &ast.StringLiteral{Value: "localhost"}},
			},
			schema: []ParameterSchema{
				{Name: "variable", Type: ast.StringType, Required: true},
				{Name: "default", Type: ast.StringType, Required: false},
			},
			expected: []ast.NamedParameter{
				{Name: "variable", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "default", Value: &ast.StringLiteral{Value: "localhost"}},
			},
			wantErr: false,
		},
		{
			name: "error: positional after named",
			params: []ast.NamedParameter{
				{Name: "default", Value: &ast.StringLiteral{Value: "localhost"}},
				{Name: "", Value: &ast.StringLiteral{Value: "API_URL"}},
			},
			schema: []ParameterSchema{
				{Name: "variable", Type: ast.StringType, Required: true},
				{Name: "default", Type: ast.StringType, Required: false},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "error: too many positional parameters",
			params: []ast.NamedParameter{
				{Name: "", Value: &ast.StringLiteral{Value: "API_URL"}},
				{Name: "", Value: &ast.StringLiteral{Value: "localhost"}},
				{Name: "", Value: &ast.StringLiteral{Value: "extra"}},
			},
			schema: []ParameterSchema{
				{Name: "variable", Type: ast.StringType, Required: true},
				{Name: "default", Type: ast.StringType, Required: false},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "empty parameters and schema",
			params:   []ast.NamedParameter{},
			schema:   []ParameterSchema{},
			expected: []ast.NamedParameter{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolvePositionalParameters(tt.params, tt.schema)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolvePositionalParameters() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolvePositionalParameters() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ResolvePositionalParameters() length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}

			for i, param := range result {
				expectedParam := tt.expected[i]
				if param.Name != expectedParam.Name {
					t.Errorf("Parameter %d: Name mismatch: got %q, want %q", i, param.Name, expectedParam.Name)
				}

				// Compare parameter values by type and content
				if !compareExpressions(param.Value, expectedParam.Value) {
					t.Errorf("Parameter %d: Value mismatch: got %v, want %v", i, param.Value, expectedParam.Value)
				}
			}
		})
	}
}

func TestResolvePositionalParameters_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		params   []ast.NamedParameter
		schema   []ParameterSchema
		expected []ast.NamedParameter
		wantErr  bool
		errMsg   string
	}{
		{
			name: "positional parameter with NameToken should stay as-is",
			params: []ast.NamedParameter{
				{
					Name:      "",
					Value:     &ast.Identifier{Name: "PORT"},
					NameToken: nil, // This indicates it's positional
				},
			},
			schema: []ParameterSchema{
				{Name: "name", Type: ast.IdentifierType, Required: true},
			},
			expected: []ast.NamedParameter{
				{Name: "name", Value: &ast.Identifier{Name: "PORT"}},
			},
			wantErr: false,
		},
		{
			name: "named parameter with NameToken should stay as-is",
			params: []ast.NamedParameter{
				{
					Name:      "name",
					Value:     &ast.Identifier{Name: "PORT"},
					NameToken: &types.Token{}, // This indicates it's named
				},
			},
			schema: []ParameterSchema{
				{Name: "name", Type: ast.IdentifierType, Required: true},
			},
			expected: []ast.NamedParameter{
				{
					Name:      "name",
					Value:     &ast.Identifier{Name: "PORT"},
					NameToken: &types.Token{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolvePositionalParameters(tt.params, tt.schema)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolvePositionalParameters() expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ResolvePositionalParameters() error message mismatch: got %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ResolvePositionalParameters() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ResolvePositionalParameters() length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}

			for i, param := range result {
				expectedParam := tt.expected[i]
				if param.Name != expectedParam.Name {
					t.Errorf("Parameter %d: Name mismatch: got %q, want %q", i, param.Name, expectedParam.Name)
				}

				if !compareExpressions(param.Value, expectedParam.Value) {
					t.Errorf("Parameter %d: Value mismatch: got %v, want %v", i, param.Value, expectedParam.Value)
				}

				// Check NameToken preservation
				if (param.NameToken == nil) != (expectedParam.NameToken == nil) {
					t.Errorf("Parameter %d: NameToken presence mismatch: got %v, want %v", i, param.NameToken != nil, expectedParam.NameToken != nil)
				}
			}
		})
	}
}

// Helper function to compare expressions
func compareExpressions(a, b ast.Expression) bool {
	switch aVal := a.(type) {
	case *ast.Identifier:
		if bVal, ok := b.(*ast.Identifier); ok {
			return aVal.Name == bVal.Name
		}
	case *ast.StringLiteral:
		if bVal, ok := b.(*ast.StringLiteral); ok {
			return aVal.Value == bVal.Value
		}
	case *ast.NumberLiteral:
		if bVal, ok := b.(*ast.NumberLiteral); ok {
			return aVal.Value == bVal.Value
		}
	case *ast.BooleanLiteral:
		if bVal, ok := b.(*ast.BooleanLiteral); ok {
			return aVal.Value == bVal.Value
		}
	case *ast.DurationLiteral:
		if bVal, ok := b.(*ast.DurationLiteral); ok {
			return aVal.Value == bVal.Value
		}
	}
	return false
}
