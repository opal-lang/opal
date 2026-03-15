package builtins

import (
	"fmt"
	"sync"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
	"github.com/builtwithtofu/sigil/runtime/vault"
)

// SecretsPlugin exposes encrypted secret storage capabilities.
type SecretsPlugin struct{}

func (p *SecretsPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "secrets", Version: "1.0.0", APIVersion: 1}
}

func (p *SecretsPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{NewSecretsValueCapability()}
}

type secretsScope struct {
	service *vault.Vault
	handles map[string]vault.SecretHandle
}

// SecretsValueCapability stores and retrieves encrypted secrets per plan hash.
type SecretsValueCapability struct {
	mu     sync.RWMutex
	scopes map[string]*secretsScope
}

func NewSecretsValueCapability() *SecretsValueCapability {
	return &SecretsValueCapability{scopes: make(map[string]*secretsScope)}
}

func (c *SecretsValueCapability) Path() string { return "secrets" }

func (c *SecretsValueCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Primary: plugin.Param{Name: "method", Type: types.TypeString},
		Params: []plugin.Param{
			{Name: "path", Type: types.TypeString, Required: true},
			{Name: "value", Type: types.TypeString},
		},
		Returns: types.TypeSecretHandle,
		Block:   plugin.BlockForbidden,
	}
}

func (c *SecretsValueCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (any, error) {
	scope := c.scopeForPlan(ctx.PlanHash())

	method := args.GetStringOptional("method")
	if method == "" {
		if args.GetStringOptional("value") != "" {
			method = "put"
		} else {
			method = "get"
		}
	}

	path := args.GetString("path")
	if path == "" {
		return nil, fmt.Errorf("@secrets requires path parameter")
	}

	switch method {
	case "put":
		value := []byte(args.GetStringOptional("value"))
		if len(value) == 0 {
			return nil, fmt.Errorf("@secrets.put requires value parameter")
		}
		handle, err := c.put(scope, path, value)
		if err != nil {
			return nil, err
		}
		return handle.ID, nil
	case "get":
		return c.get(scope, path)
	default:
		return nil, fmt.Errorf("@secrets unsupported method %q (expected put or get)", method)
	}
}

func (c *SecretsValueCapability) put(scope *secretsScope, path string, value []byte) (vault.SecretHandle, error) {
	encrypted, err := scope.service.Encrypt(value)
	if err != nil {
		return vault.SecretHandle{}, fmt.Errorf("encrypt secret %q: %w", path, err)
	}

	handle, err := scope.service.Store(encrypted)
	if err != nil {
		return vault.SecretHandle{}, fmt.Errorf("store secret %q: %w", path, err)
	}

	c.mu.Lock()
	scope.handles[path] = handle
	c.mu.Unlock()

	return handle, nil
}

func (c *SecretsValueCapability) get(scope *secretsScope, path string) ([]byte, error) {
	c.mu.RLock()
	handle, ok := scope.handles[path]
	c.mu.RUnlock()
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

func (c *SecretsValueCapability) scopeForPlan(planHash string) *secretsScope {
	scopeKey := planHash
	if scopeKey == "" {
		scopeKey = "default"
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	scope, ok := c.scopes[scopeKey]
	if ok {
		return scope
	}

	v := vault.NewWithPlanKey([]byte(scopeKey))
	scope = &secretsScope{service: v, handles: make(map[string]vault.SecretHandle)}
	c.scopes[scopeKey] = scope
	return scope
}
