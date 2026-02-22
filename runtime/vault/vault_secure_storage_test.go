package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVault_EncryptDecrypt_RoundTrip(t *testing.T) {
	v := NewWithPlanKey([]byte("plan-key-for-secure-storage"))
	plaintext := []byte("top-secret")

	encrypted, err := v.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := v.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestVault_StoreRetrieve(t *testing.T) {
	v := NewWithPlanKey([]byte("plan-key-for-secure-storage"))
	encrypted, err := v.Encrypt([]byte("value"))
	require.NoError(t, err)

	handle, err := v.Store(encrypted)
	require.NoError(t, err)
	assert.False(t, handle.IsZero())

	retrieved, err := v.Retrieve(handle)
	require.NoError(t, err)
	assert.Equal(t, encrypted, retrieved)
}

func TestVault_GracefulDegradation_WithoutKey(t *testing.T) {
	v := newVault()
	plaintext := []byte("plain-value")

	encrypted, err := v.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, encrypted)

	decrypted, err := v.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
