package builtin

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWhenDecorator tests the @when pattern decorator
func TestWhenDecorator(t *testing.T) {
	tests := []struct {
		name         string
		params       []decorators.DecoratorParam
		branches     map[string]decorators.CommandSeq
		envVars      map[string]string
		wantOut      string
		wantErr      string
		wantExit     int
		wantSelected string
	}{
		{
			name: "exact environment variable match",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "ENV"},
			},
			branches: map[string]decorators.CommandSeq{
				"prod": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo production"},
						}},
					},
				},
				"dev": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo development"},
						}},
					},
				},
			},
			envVars:      map[string]string{"ENV": "prod"},
			wantOut:      "production\n",
			wantExit:     0,
			wantSelected: "prod",
		},
		{
			name: "wildcard fallback match",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "ENV"},
			},
			branches: map[string]decorators.CommandSeq{
				"prod": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo production"},
						}},
					},
				},
				"default": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo unknown environment"},
						}},
					},
				},
			},
			envVars:      map[string]string{"ENV": "staging"},
			wantOut:      "unknown environment\n",
			wantExit:     0,
			wantSelected: "default",
		},
		{
			name: "missing environment variable uses wildcard",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "MISSING_VAR"},
			},
			branches: map[string]decorators.CommandSeq{
				"value1": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo value1"},
						}},
					},
				},
				"default": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo default"},
						}},
					},
				},
			},
			envVars:      map[string]string{}, // MISSING_VAR not set
			wantOut:      "default\n",
			wantExit:     0,
			wantSelected: "default",
		},
		{
			name: "empty environment variable matches empty string pattern",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "EMPTY_VAR"},
			},
			branches: map[string]decorators.CommandSeq{
				"": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo empty"},
						}},
					},
				},
				"nonempty": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo has value"},
						}},
					},
				},
			},
			envVars:      map[string]string{"EMPTY_VAR": ""},
			wantOut:      "empty\n",
			wantExit:     0,
			wantSelected: "",
		},
		{
			name: "no matching branch error",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "ENV"},
			},
			branches: map[string]decorators.CommandSeq{
				"prod": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo production"},
						}},
					},
				},
			},
			envVars:  map[string]string{"ENV": "staging"},
			wantErr:  "no matching branch for ENV=\"staging\"",
			wantExit: 1,
		},
		{
			name:   "missing environment variable parameter error",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"branch1": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo test"},
						}},
					},
				},
			},
			envVars:  map[string]string{},
			wantErr:  "@when parameter error: @when requires an environment variable name",
			wantExit: 1,
		},
		{
			name: "named env parameter",
			params: []decorators.DecoratorParam{
				{Name: "env", Value: "TARGET"},
			},
			branches: map[string]decorators.CommandSeq{
				"linux": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo linux build"},
						}},
					},
				},
			},
			envVars:      map[string]string{"TARGET": "linux"},
			wantOut:      "linux build\n",
			wantExit:     0,
			wantSelected: "linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			when := NewWhenDecorator()
			ctx := createTestCtxWithEnv(tt.envVars)

			result := when.SelectBranch(ctx, tt.params, tt.branches)

			assert.Equal(t, tt.wantExit, result.ExitCode, "unexpected exit code")

			if tt.wantOut != "" {
				assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch")
			}

			if tt.wantErr != "" {
				assert.Contains(t, result.Stderr, tt.wantErr, "stderr mismatch")
			}
		})
	}
}

// TestWhenDecoratorDescribe tests the plan/describe functionality
func TestWhenDecoratorDescribe(t *testing.T) {
	tests := []struct {
		name           string
		params         []decorators.DecoratorParam
		branches       map[string]plan.ExecutionStep
		envVars        map[string]string
		wantDesc       string
		wantSelected   string
		wantEnvVar     string
		wantCurrentVal string
	}{
		{
			name: "environment variable with value",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "ENV"},
			},
			branches: map[string]plan.ExecutionStep{
				"prod": {Description: "production deployment"},
				"dev":  {Description: "development deployment"},
			},
			envVars:        map[string]string{"ENV": "prod"},
			wantDesc:       "@when(ENV) → ENV=\"prod\" (selected: prod)",
			wantSelected:   "prod",
			wantEnvVar:     "ENV",
			wantCurrentVal: "prod",
		},
		{
			name: "missing environment variable",
			params: []decorators.DecoratorParam{
				{Name: "", Value: "MISSING"},
			},
			branches: map[string]plan.ExecutionStep{
				"default": {Description: "default branch"},
			},
			envVars:        map[string]string{},
			wantDesc:       "@when(MISSING) → MISSING=<unset> (selected: default)",
			wantSelected:   "default",
			wantEnvVar:     "MISSING",
			wantCurrentVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			when := NewWhenDecorator()
			ctx := createTestCtxWithEnv(tt.envVars)

			step := when.Describe(ctx, tt.params, tt.branches)

			assert.Equal(t, tt.wantDesc, step.Description, "description mismatch")
			assert.Equal(t, tt.wantEnvVar, step.Metadata["envVar"], "envVar metadata mismatch")
			assert.Equal(t, tt.wantCurrentVal, step.Metadata["currentValue"], "currentValue metadata mismatch")
			assert.Equal(t, tt.wantSelected, step.Metadata["selectedBranch"], "selectedBranch metadata mismatch")
		})
	}
}

// TestTryDecorator tests the @try pattern decorator
func TestTryDecorator(t *testing.T) {
	tests := []struct {
		name     string
		params   []decorators.DecoratorParam
		branches map[string]decorators.CommandSeq
		wantOut  string
		wantErr  string
		wantExit int
	}{
		{
			name:   "successful main execution",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo success"},
						}},
					},
				},
				"catch": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo catch"},
						}},
					},
				},
			},
			wantOut:  "success\n",
			wantExit: 0,
		},
		{
			name:   "main fails, catch executes successfully",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "exit 1"},
						}},
					},
				},
				"catch": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo recovered"},
						}},
					},
				},
			},
			wantOut:  "recovered\n",
			wantExit: 0,
		},
		{
			name:   "main fails, catch also fails",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "exit 1"},
						}},
					},
				},
				"catch": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "exit 2"},
						}},
					},
				},
			},
			wantExit: 2, // Catch block's exit code
		},
		{
			name:   "finally block always executes",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo main"},
						}},
					},
				},
				"finally": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo cleanup"},
						}},
					},
				},
			},
			wantOut:  "main\ncleanup\n",
			wantExit: 0,
		},
		{
			name:   "finally failure overrides success",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo main"},
						}},
					},
				},
				"finally": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "exit 3"},
						}},
					},
				},
			},
			wantOut:  "main\n",
			wantExit: 3, // Finally block's exit code overrides
		},
		{
			name:   "complete try/catch/finally flow",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "exit 1"},
						}},
					},
				},
				"catch": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo recovered"},
						}},
					},
				},
				"finally": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo cleanup"},
						}},
					},
				},
			},
			wantOut:  "recovered\ncleanup\n",
			wantExit: 0,
		},
		{
			name:   "missing main branch error",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"catch": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo catch"},
						}},
					},
				},
			},
			wantErr:  "@try requires a 'main' branch",
			wantExit: 1,
		},
		{
			name:   "main only (no catch or finally)",
			params: []decorators.DecoratorParam{},
			branches: map[string]decorators.CommandSeq{
				"main": {
					Steps: []decorators.CommandStep{
						{Chain: []decorators.ChainElement{
							{Kind: decorators.ElementKindShell, Text: "echo main only"},
						}},
					},
				},
			},
			wantOut:  "main only\n",
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			try := NewTryDecorator()
			ctx := createTestCtx()

			result := try.SelectBranch(ctx, tt.params, tt.branches)

			assert.Equal(t, tt.wantExit, result.ExitCode, "unexpected exit code")

			if tt.wantOut != "" {
				assert.Equal(t, tt.wantOut, result.Stdout, "stdout mismatch")
			}

			if tt.wantErr != "" {
				assert.Contains(t, result.Stderr, tt.wantErr, "stderr mismatch")
			}
		})
	}
}

// TestTryDecoratorDescribe tests the plan/describe functionality
func TestTryDecoratorDescribe(t *testing.T) {
	tests := []struct {
		name         string
		params       []decorators.DecoratorParam
		branches     map[string]plan.ExecutionStep
		wantDesc     string
		wantHasCatch bool
		wantFinally  bool
	}{
		{
			name:   "try with all branches",
			params: []decorators.DecoratorParam{},
			branches: map[string]plan.ExecutionStep{
				"main":    {Description: "main operation"},
				"catch":   {Description: "error handling"},
				"finally": {Description: "cleanup"},
			},
			wantDesc:     "@try",
			wantHasCatch: true,
			wantFinally:  true,
		},
		{
			name:   "try with main only",
			params: []decorators.DecoratorParam{},
			branches: map[string]plan.ExecutionStep{
				"main": {Description: "main operation"},
			},
			wantDesc:     "@try",
			wantHasCatch: false,
			wantFinally:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			try := NewTryDecorator()
			ctx := createTestCtx()

			step := try.Describe(ctx, tt.params, tt.branches)

			assert.Equal(t, tt.wantDesc, step.Description, "description mismatch")
			assert.Equal(t, tt.wantHasCatch, step.Metadata["hasCatch"] == "true", "hasCatch metadata mismatch")
			assert.Equal(t, tt.wantFinally, step.Metadata["hasFinally"] == "true", "hasFinally metadata mismatch")
			assert.Equal(t, "try", step.Metadata["decorator"], "decorator metadata mismatch")
		})
	}
}

// TestPatternDecoratorsParameterSchema tests parameter schemas are correct
func TestPatternDecoratorsParameterSchema(t *testing.T) {
	t.Run("WhenDecorator schema", func(t *testing.T) {
		when := NewWhenDecorator()
		schema := when.ParameterSchema()

		require.Len(t, schema, 1, "expected 1 parameter")

		// Check env parameter
		assert.Equal(t, "env", schema[0].Name)
		assert.Equal(t, decorators.ArgTypeString, schema[0].Type)
		assert.True(t, schema[0].Required)
	})

	t.Run("TryDecorator schema", func(t *testing.T) {
		try := NewTryDecorator()
		schema := try.ParameterSchema()

		require.Len(t, schema, 0, "expected 0 parameters (uses pattern branches)")
	})
}

// TestPatternDecoratorsRegistration tests that decorators are registered correctly
func TestPatternDecoratorsRegistration(t *testing.T) {
	registry := decorators.GlobalRegistry()

	t.Run("when decorator registered", func(t *testing.T) {
		whenDecorator, found := registry.GetPattern("when")
		require.True(t, found, "when decorator should be registered")
		assert.Equal(t, "when", whenDecorator.Name())
	})

	t.Run("try decorator registered", func(t *testing.T) {
		tryDecorator, found := registry.GetPattern("try")
		require.True(t, found, "try decorator should be registered")
		assert.Equal(t, "try", tryDecorator.Name())
	})
}

// TestPatternDecoratorPatternSchemas tests pattern schema validation
func TestPatternDecoratorPatternSchemas(t *testing.T) {
	t.Run("WhenDecorator pattern schema", func(t *testing.T) {
		when := NewWhenDecorator()
		schema := when.PatternSchema()

		assert.True(t, schema.AllowsWildcard, "should allow wildcard patterns")
		assert.True(t, schema.AllowsAnyIdentifier, "should allow any identifier patterns")
		assert.Empty(t, schema.RequiredPatterns, "should not require specific patterns")
	})

	t.Run("TryDecorator pattern schema", func(t *testing.T) {
		try := NewTryDecorator()
		schema := try.PatternSchema()

		assert.False(t, schema.AllowsWildcard, "should not allow wildcard patterns")
		assert.False(t, schema.AllowsAnyIdentifier, "should not allow arbitrary identifiers")
		assert.Contains(t, schema.AllowedPatterns, "main")
		assert.Contains(t, schema.AllowedPatterns, "catch")
		assert.Contains(t, schema.AllowedPatterns, "finally")
		assert.Contains(t, schema.RequiredPatterns, "main")
	})
}

// Helper functions for testing

// createTestCtxWithEnv creates a test context with environment variables
func createTestCtxWithEnv(envVars map[string]string) *decorators.Ctx {
	env := &ir.EnvSnapshot{Values: envVars}
	return &decorators.Ctx{
		Env:     env,
		Vars:    map[string]string{},
		WorkDir: "",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		DryRun:  false,
		Debug:   false,
		NumCPU:  4, // Deterministic value for tests
	}
}
