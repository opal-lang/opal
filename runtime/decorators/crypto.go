package decorators

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/opal-lang/opal/core/decorator"
)

func init() {
	if err := decorator.Register("crypto", &CryptoValueDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @crypto decorator: %v", err))
	}
}

// CryptoValueDecorator provides cryptographic hash functions.
type CryptoValueDecorator struct{}

func (d *CryptoValueDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("crypto").
		Summary("Cryptographic helper functions").
		Build()
}

// Resolve implements decorator.Value interface
func (d *CryptoValueDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))

	for i, call := range calls {
		method := d.methodForCall(call)
		switch {
		case strings.EqualFold(method, "SHA256"):
			input, err := d.argumentForCall(call)
			if err != nil {
				results[i] = decorator.ResolveResult{Error: err, Origin: "@crypto"}
				continue
			}

			hash := sha256.Sum256([]byte(input))
			results[i] = decorator.ResolveResult{
				Value:  hex.EncodeToString(hash[:]),
				Origin: "@crypto",
			}
		default:
			results[i] = decorator.ResolveResult{
				Error:  fmt.Errorf("unsupported crypto method: %s", method),
				Origin: "@crypto",
			}
		}
	}

	return results, nil
}

func (d *CryptoValueDecorator) methodForCall(call decorator.ValueCall) string {
	if call.Primary != nil && *call.Primary != "" {
		return *call.Primary
	}

	if method, ok := call.Params["method"].(string); ok && method != "" {
		return method
	}

	return "SHA256"
}

func (d *CryptoValueDecorator) argumentForCall(call decorator.ValueCall) (string, error) {
	if value, ok := call.Params["arg0"]; ok {
		return coerceString(value)
	}

	if value, ok := call.Params["arg1"]; ok {
		return coerceString(value)
	}

	if value, ok := call.Params["data"]; ok {
		return coerceString(value)
	}

	return "", fmt.Errorf("@crypto.%s requires one string argument", d.methodForCall(call))
}

func coerceString(value any) (string, error) {
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("crypto argument must be string, got %T", value)
	}
	return str, nil
}
