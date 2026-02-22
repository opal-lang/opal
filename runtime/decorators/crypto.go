package decorators

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/isolation"
	"github.com/opal-lang/opal/runtime/vault"
)

var cryptoVault = vault.NewWithPlanKey(nil)

func init() {
	decorator.Register("crypto", &CryptoValueDecorator{})
}

// CryptoValueDecorator provides cryptographic operations in isolated environment
type CryptoValueDecorator struct{}

func (d *CryptoValueDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("crypto").
		Summary("Cryptographic operations in isolated environment").
		Build()
}

// Resolve implements decorator.Value interface
func (d *CryptoValueDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))

	for i, call := range calls {
		// Get key type from parameters
		keyType := "ed25519" // default
		if kt, ok := call.Params["type"].(string); ok {
			keyType = kt
		}

		// Generate key
		keyPair, err := d.Generate(context.Background(), keyType)
		if err != nil {
			results[i] = decorator.ResolveResult{
				Error:  err,
				Origin: "@crypto",
			}
			continue
		}

		storeInVault, _ := call.Params["store"].(bool)
		vaultPath, _ := call.Params["vault_path"].(string)
		if storeInVault || vaultPath != "" {
			handle, err := d.storePrivateKeyInVault(keyPair.PrivateKey)
			if err != nil {
				results[i] = decorator.ResolveResult{
					Error:  err,
					Origin: "@crypto",
				}
				continue
			}

			keyPair.PrivateKeyHandle = handle.ID
			keyPair.PrivateKey = ""
		}

		results[i] = decorator.ResolveResult{
			Value:  keyPair,
			Origin: "@crypto",
		}
	}

	return results, nil
}

func (d *CryptoValueDecorator) storePrivateKeyInVault(privateKey string) (vault.SecretHandle, error) {
	value, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		value = []byte(privateKey)
	}

	encrypted, err := cryptoVault.Encrypt(value)
	if err != nil {
		return vault.SecretHandle{}, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	handle, err := cryptoVault.Store(encrypted)
	if err != nil {
		return vault.SecretHandle{}, fmt.Errorf("failed to store private key: %w", err)
	}

	return handle, nil
}

func (d *CryptoValueDecorator) Generate(ctx context.Context, keyType string) (KeyPair, error) {
	// Check if isolation is supported
	if !isolation.IsSupported() {
		// Fallback: generate without isolation
		return d.generateWithoutIsolation(keyType)
	}

	// Try to create isolated environment for key generation
	isolator := isolation.NewIsolator()

	// Setup maximum isolation: no network, restricted filesystem
	config := decorator.IsolationConfig{
		NetworkPolicy:    decorator.NetworkPolicyDeny,
		FilesystemPolicy: decorator.FilesystemPolicyEphemeral,
		MemoryLock:       true,
	}

	// Attempt isolation - if it fails, fall back to non-isolated generation
	if err := isolator.Isolate(decorator.IsolationLevelMaximum, config); err != nil {
		// Isolation failed - fall back gracefully
		return d.generateWithoutIsolation(keyType)
	}

	// Drop network to ensure no connectivity
	if err := isolator.DropNetwork(); err != nil {
		// Non-fatal: continue without network drop
		// Fall back to generate without full isolation
		return d.generateWithoutIsolation(keyType)
	}

	// Lock memory to prevent swapping
	if err := isolator.LockMemory(); err != nil {
		// Non-fatal: continue without memory lock
	}

	// Generate key in isolated environment
	keyPair, err := d.generateKey(keyType)
	if err != nil {
		return KeyPair{}, fmt.Errorf("failed to generate key: %w", err)
	}

	return keyPair, nil
}

func (d *CryptoValueDecorator) generateKey(keyType string) (KeyPair, error) {
	switch keyType {
	case "ed25519":
		return d.generateEd25519()
	default:
		return KeyPair{}, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

func (d *CryptoValueDecorator) generateEd25519() (KeyPair, error) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	pubKey := privKey.Public().(ed25519.PublicKey)

	return KeyPair{
		Type:       "ed25519",
		PublicKey:  base64.StdEncoding.EncodeToString(pubKey),
		PrivateKey: base64.StdEncoding.EncodeToString(privKey.Seed()),
	}, nil
}

func (d *CryptoValueDecorator) generateWithoutIsolation(keyType string) (KeyPair, error) {
	// Generate without isolation - for environments where namespaces aren't available
	// This is less secure but provides graceful degradation
	return d.generateKey(keyType)
}

// KeyPair represents a generated cryptographic key pair
type KeyPair struct {
	Type             string `json:"type"`
	PublicKey        string `json:"public_key"`
	PrivateKey       string `json:"private_key"`
	PrivateKeyHandle string `json:"private_key_handle,omitempty"`
}

// IsZero returns true if the KeyPair is empty
func (k KeyPair) IsZero() bool {
	return k.Type == "" && k.PublicKey == "" && k.PrivateKey == "" && k.PrivateKeyHandle == ""
}
