package decorators

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/types"
	"github.com/builtwithtofu/sigil/runtime/vault"
)

func init() {
	if err := decorator.Register("secrets", NewSecretsDecorator()); err != nil {
		panic(fmt.Sprintf("failed to register @secrets decorator: %v", err))
	}
}

type secretsScope struct {
	service *vault.Vault
	handles map[string]vault.SecretHandle
}

type SecretsDecorator struct {
	mu     sync.RWMutex
	scopes map[string]*secretsScope
}

func NewSecretsDecorator() *SecretsDecorator {
	return &SecretsDecorator{scopes: make(map[string]*secretsScope)}
}

func (d *SecretsDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("secrets").
		Summary("Store and retrieve encrypted secrets").
		Roles(decorator.RoleProvider).
		PrimaryParamString("method", "Operation to perform: put or get").
		Examples("put", "get").
		Done().
		ParamString("path", "Logical secret path").
		Examples("keys/deploy", "tokens/github").
		Done().
		ParamString("value", "Secret value for put operations").
		Examples("s3cr3t").
		Done().
		Returns(types.TypeSecretHandle, "Secret handle for put, or []byte for get").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

func (d *SecretsDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	scope := d.scopeForPlan(ctx.PlanHash)

	for i, call := range calls {
		method := extractSecretsMethod(call)
		path, err := extractSecretsPath(call)
		if err != nil {
			results[i] = decorator.ResolveResult{Value: nil, Origin: "@secrets", Error: err}
			continue
		}

		switch method {
		case "put":
			value, err := extractSecretsValue(call)
			if err != nil {
				results[i] = decorator.ResolveResult{Value: nil, Origin: "@secrets", Error: err}
				continue
			}

			handle, err := d.put(scope, path, value)
			if err != nil {
				results[i] = decorator.ResolveResult{Value: nil, Origin: "@secrets", Error: err}
				continue
			}

			results[i] = decorator.ResolveResult{Value: handle, Origin: "@secrets", Error: nil}
		case "get":
			value, err := d.get(scope, path)
			if err != nil {
				results[i] = decorator.ResolveResult{Value: nil, Origin: "@secrets", Error: err}
				continue
			}

			results[i] = decorator.ResolveResult{Value: value, Origin: "@secrets", Error: nil}
		default:
			results[i] = decorator.ResolveResult{
				Value:  nil,
				Origin: "@secrets",
				Error:  fmt.Errorf("@secrets unsupported method %q (expected put or get)", method),
			}
		}
	}

	return results, nil
}

func (d *SecretsDecorator) put(scope *secretsScope, path string, value []byte) (vault.SecretHandle, error) {
	encrypted, err := scope.service.Encrypt(value)
	if err != nil {
		return vault.SecretHandle{}, fmt.Errorf("encrypt secret %q: %w", path, err)
	}

	handle, err := scope.service.Store(encrypted)
	if err != nil {
		return vault.SecretHandle{}, fmt.Errorf("store secret %q: %w", path, err)
	}

	d.mu.Lock()
	scope.handles[path] = handle
	d.mu.Unlock()

	return handle, nil
}

func (d *SecretsDecorator) get(scope *secretsScope, path string) ([]byte, error) {
	d.mu.RLock()
	handle, ok := scope.handles[path]
	d.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("secret path %q not found", path)
	}

	encrypted, err := scope.service.Retrieve(handle)
	if err != nil {
		return nil, err
	}

	plaintext, err := scope.service.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (d *SecretsDecorator) scopeForPlan(planHash []byte) *secretsScope {
	scopeKey := "default"
	if len(planHash) > 0 {
		scopeKey = hex.EncodeToString(planHash)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	scope, ok := d.scopes[scopeKey]
	if ok {
		return scope
	}

	v := vault.NewWithPlanKey(planHash)
	scope = &secretsScope{
		service: v,
		handles: make(map[string]vault.SecretHandle),
	}
	d.scopes[scopeKey] = scope
	return scope
}

func extractSecretsMethod(call decorator.ValueCall) string {
	if call.Primary != nil && *call.Primary != "" {
		return *call.Primary
	}
	if method, ok := call.Params["method"].(string); ok && method != "" {
		return method
	}
	if _, ok := call.Params["value"]; ok {
		return "put"
	}
	return "get"
}

func extractSecretsPath(call decorator.ValueCall) (string, error) {
	path, ok := call.Params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("@secrets requires path parameter")
	}
	return path, nil
}

func extractSecretsValue(call decorator.ValueCall) ([]byte, error) {
	raw, ok := call.Params["value"]
	if !ok {
		return nil, fmt.Errorf("@secrets.put requires value parameter")
	}

	switch v := raw.(type) {
	case []byte:
		return append([]byte(nil), v...), nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("@secrets.put value must be string or []byte")
	}
}
