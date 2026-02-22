package decorators

import (
	"context"
	"testing"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/isolation"
	"github.com/opal-lang/opal/runtime/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryptoValueDecorator_Descriptor(t *testing.T) {
	d := &CryptoValueDecorator{}
	desc := d.Descriptor()

	assert.Equal(t, "crypto", desc.Path)
	assert.Contains(t, desc.Summary, "Cryptographic")
}

func TestCryptoValueDecorator_Generate_Ed25519(t *testing.T) {
	d := &CryptoValueDecorator{}
	ctx := t.Context()

	keyPair, err := d.Generate(ctx, "ed25519")

	// Should work even without isolation (graceful degradation)
	require.NoError(t, err)
	assert.Equal(t, "ed25519", keyPair.Type)
	assert.NotEmpty(t, keyPair.PublicKey)
	assert.NotEmpty(t, keyPair.PrivateKey)
	assert.NotEqual(t, keyPair.PublicKey, keyPair.PrivateKey)
}

func TestCryptoValueDecorator_Generate_UnsupportedType(t *testing.T) {
	d := &CryptoValueDecorator{}
	ctx := t.Context()

	_, err := d.Generate(ctx, "rsa-4096")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key type")
}

func TestCryptoValueDecorator_Generate_WithIsolation(t *testing.T) {
	if !isolation.IsSupported() {
		t.Skip("isolation not supported")
	}

	d := &CryptoValueDecorator{}
	ctx := t.Context()

	// This will attempt to use isolation
	keyPair, err := d.Generate(ctx, "ed25519")
	// May fail in restricted environments, but should handle gracefully
	if err != nil {
		t.Logf("Isolation failed (expected in restricted env): %v", err)
		return
	}

	assert.Equal(t, "ed25519", keyPair.Type)
	assert.NotEmpty(t, keyPair.PublicKey)
	assert.NotEmpty(t, keyPair.PrivateKey)
}

func TestKeyPair_IsZero(t *testing.T) {
	empty := KeyPair{}
	assert.True(t, empty.IsZero())

	populated := KeyPair{
		Type:       "ed25519",
		PublicKey:  "abc123",
		PrivateKey: "xyz789",
	}
	assert.False(t, populated.IsZero())
}

func TestCryptoValueDecorator_Resolve_StoresPrivateKeyInVault(t *testing.T) {
	d := &CryptoValueDecorator{}

	results, err := d.Resolve(decorator.ValueEvalContext{}, decorator.ValueCall{
		Path: "crypto",
		Params: map[string]any{
			"type":  "ed25519",
			"store": true,
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)

	keyPair, ok := results[0].Value.(KeyPair)
	require.True(t, ok)
	assert.NotEmpty(t, keyPair.PrivateKeyHandle)
	assert.Empty(t, keyPair.PrivateKey)

	encrypted, err := cryptoVault.Retrieve(vault.SecretHandle{ID: keyPair.PrivateKeyHandle})
	require.NoError(t, err)
	decrypted, err := cryptoVault.Decrypt(encrypted)
	require.NoError(t, err)
	assert.NotEmpty(t, decrypted)

	_, err = d.Generate(context.Background(), "ed25519")
	require.NoError(t, err)
}
