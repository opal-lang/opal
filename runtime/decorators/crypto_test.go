package decorators

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/stretchr/testify/require"
)

func TestCryptoValueDecorator_Descriptor(t *testing.T) {
	d := &CryptoValueDecorator{}
	desc := d.Descriptor()

	if diff := cmp.Diff("crypto", desc.Path); diff != "" {
		t.Fatalf("descriptor path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("Cryptographic helper functions", desc.Summary); diff != "" {
		t.Fatalf("descriptor summary mismatch (-want +got):\n%s", diff)
	}
}

func TestCryptoValueDecorator_Resolve_SHA256_Hello(t *testing.T) {
	d := &CryptoValueDecorator{}
	fn := "SHA256"

	results, err := d.Resolve(decorator.ValueEvalContext{}, decorator.ValueCall{
		Path:    "crypto",
		Primary: &fn,
		Params: map[string]any{
			"arg1": "hello",
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)

	hash, ok := results[0].Value.(string)
	require.True(t, ok)
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if diff := cmp.Diff(expected, hash); diff != "" {
		t.Fatalf("sha256 hash mismatch (-want +got):\n%s", diff)
	}
}

func TestCryptoValueDecorator_Resolve_SHA256_EmptyString(t *testing.T) {
	d := &CryptoValueDecorator{}
	fn := "SHA256"

	results, err := d.Resolve(decorator.ValueEvalContext{}, decorator.ValueCall{
		Path:    "crypto",
		Primary: &fn,
		Params: map[string]any{
			"arg1": "",
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error)

	hash, ok := results[0].Value.(string)
	require.True(t, ok)
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if diff := cmp.Diff(expected, hash); diff != "" {
		t.Fatalf("sha256 hash mismatch (-want +got):\n%s", diff)
	}
}

func TestCryptoValueDecorator_Resolve_UnknownFunction(t *testing.T) {
	d := &CryptoValueDecorator{}
	fn := "Ed25519"

	results, err := d.Resolve(decorator.ValueEvalContext{}, decorator.ValueCall{
		Path:    "crypto",
		Primary: &fn,
		Params: map[string]any{
			"arg1": "hello",
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Error(t, results[0].Error)

	if diff := cmp.Diff("unsupported crypto method: Ed25519", results[0].Error.Error()); diff != "" {
		t.Fatalf("error mismatch (-want +got):\n%s", diff)
	}
}
