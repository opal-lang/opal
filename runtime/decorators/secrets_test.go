package decorators

import (
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsDecorator_Descriptor(t *testing.T) {
	d := NewSecretsDecorator()
	desc := d.Descriptor()

	assert.Equal(t, "secrets", desc.Path)
	assert.Contains(t, desc.Summary, "encrypted")
}

func TestSecretsDecorator_PutThenGet(t *testing.T) {
	d := NewSecretsDecorator()
	ctx := decorator.ValueEvalContext{PlanHash: []byte("plan-key")}

	putMethod := "put"
	putResult, err := d.Resolve(ctx, decorator.ValueCall{
		Primary: &putMethod,
		Params: map[string]any{
			"path":  "keys/deploy",
			"value": []byte("secret-value"),
		},
	})
	require.NoError(t, err)
	require.Len(t, putResult, 1)
	require.NoError(t, putResult[0].Error)

	handle, ok := putResult[0].Value.(vault.SecretHandle)
	require.True(t, ok)
	assert.False(t, handle.IsZero())

	getMethod := "get"
	getResult, err := d.Resolve(ctx, decorator.ValueCall{
		Primary: &getMethod,
		Params: map[string]any{
			"path": "keys/deploy",
		},
	})
	require.NoError(t, err)
	require.Len(t, getResult, 1)
	require.NoError(t, getResult[0].Error)

	value, ok := getResult[0].Value.([]byte)
	require.True(t, ok)
	assert.Equal(t, []byte("secret-value"), value)
}

func TestSecretsDecorator_GetMissingPath(t *testing.T) {
	d := NewSecretsDecorator()
	ctx := decorator.ValueEvalContext{PlanHash: []byte("plan-key")}

	getMethod := "get"
	result, err := d.Resolve(ctx, decorator.ValueCall{
		Primary: &getMethod,
		Params:  map[string]any{"path": "missing/path"},
	})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Error(t, result[0].Error)
	assert.Contains(t, result[0].Error.Error(), "not found")
}
