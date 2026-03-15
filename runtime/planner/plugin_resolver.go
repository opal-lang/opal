package planner

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net"
	"time"

	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	coreruntime "github.com/builtwithtofu/sigil/core/runtime"
)

type plannerPluginValueContext struct {
	ctx      context.Context
	session  coreplugin.ParentTransport
	planHash string
	lookup   func(name string) (any, bool)
}

func (c plannerPluginValueContext) Context() context.Context            { return c.ctx }
func (c plannerPluginValueContext) Session() coreplugin.ParentTransport { return c.session }
func (c plannerPluginValueContext) PlanHash() string                    { return c.planHash }
func (c plannerPluginValueContext) LookupValue(name string) (any, bool) {
	if c.lookup == nil {
		return nil, false
	}
	return c.lookup(name)
}

type plannerResolvedArgs struct {
	strings map[string]string
	ints    map[string]int
	params  map[string]any
	schema  coreplugin.Schema
}

func (a plannerResolvedArgs) GetString(name string) string {
	if value, ok := a.params[name].(string); ok {
		return value
	}
	return a.strings[name]
}

func (a plannerResolvedArgs) GetStringOptional(name string) string { return a.GetString(name) }

func (a plannerResolvedArgs) GetInt(name string) int {
	switch value := a.params[name].(type) {
	case int:
		return value
	case int64:
		return int(value)
	}
	return a.ints[name]
}

func (a plannerResolvedArgs) GetDuration(name string) time.Duration {
	if value, ok := a.params[name].(time.Duration); ok {
		return value
	}
	if value, ok := a.params[name].(durationLiteral); ok {
		parsed, err := time.ParseDuration(string(value))
		if err == nil {
			return parsed
		}
	}
	if value, ok := a.params[name].(string); ok {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func (a plannerResolvedArgs) ResolveSecret(name string) (string, error) {
	if !a.schema.DeclaresSecret(name) {
		return "", fmt.Errorf("capability did not declare secret %q", name)
	}
	if value, ok := a.params[name].(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("declared secret %q not provided", name)
}

type plannerPluginSession struct {
	inner coreruntime.Session
}

func (s plannerPluginSession) decoratorSession() coreruntime.Session { return s.inner }

type plannerRuntimeNetworkDialerAdapter struct{ inner coreruntime.NetworkDialer }

func (d plannerRuntimeNetworkDialerAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.inner.DialContext(ctx, network, addr)
}

type plannerPluginNetworkDialerAdapter struct{ inner coreplugin.NetworkDialer }

func (d plannerPluginNetworkDialerAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.inner.DialContext(ctx, network, addr)
}

func (s plannerPluginSession) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	result, err := s.inner.Run(ctx, argv, coreruntime.RunOpts{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr, Dir: opts.Dir})
	return coreplugin.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}

func (s plannerPluginSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.inner.Put(ctx, data, path, mode)
}

func (s plannerPluginSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.inner.Get(ctx, path)
}

func (s plannerPluginSession) Snapshot() coreplugin.SessionSnapshot {
	env := s.inner.Env()
	copyEnv := make(map[string]string, len(env))
	for key, value := range env {
		copyEnv[key] = value
	}
	return coreplugin.SessionSnapshot{Env: copyEnv, Workdir: s.inner.Cwd(), Platform: s.inner.Platform()}
}
func (s plannerPluginSession) Close() error { return nil }
func (s plannerPluginSession) NetworkDialer() coreplugin.NetworkDialer {
	if dialer, ok := s.inner.(coreruntime.NetworkDialer); ok {
		if dialer != nil {
			return plannerRuntimeNetworkDialerAdapter{inner: dialer}
		}
	}
	dialer, err := coreruntime.GetNetworkDialer(s.inner)
	if err != nil || dialer == nil {
		return nil
	}
	return plannerRuntimeNetworkDialerAdapter{inner: dialer}
}

type plannerOpenedTransportSession struct {
	inner coreplugin.OpenedTransport
}

func (s plannerOpenedTransportSession) Run(ctx context.Context, argv []string, opts coreruntime.RunOpts) (coreruntime.Result, error) {
	result, err := s.inner.Run(ctx, argv, coreplugin.RunOpts{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr, Dir: opts.Dir})
	return coreruntime.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}

func (s plannerOpenedTransportSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.inner.Put(ctx, data, path, mode)
}

func (s plannerOpenedTransportSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.inner.Get(ctx, path)
}

func (s plannerOpenedTransportSession) Env() map[string]string {
	snapshot := s.inner.Snapshot()
	env := make(map[string]string, len(snapshot.Env))
	for key, value := range snapshot.Env {
		env[key] = value
	}
	return env
}
func (s plannerOpenedTransportSession) WithEnv(delta map[string]string) coreruntime.Session { return s }
func (s plannerOpenedTransportSession) WithWorkdir(dir string) coreruntime.Session          { return s }
func (s plannerOpenedTransportSession) Cwd() string                                         { return s.inner.Snapshot().Workdir }
func (s plannerOpenedTransportSession) ID() string                                          { return "planner-plugin-transport" }
func (s plannerOpenedTransportSession) TransportScope() coreruntime.TransportScope {
	return coreruntime.TransportScopeAny
}
func (s plannerOpenedTransportSession) Platform() string { return s.inner.Snapshot().Platform }
func (s plannerOpenedTransportSession) Close() error     { return s.inner.Close() }
func (s plannerOpenedTransportSession) NetworkDialer() coreruntime.NetworkDialer {
	provider, ok := s.inner.(coreplugin.NetworkDialerProvider)
	if !ok {
		return nil
	}
	dialer := provider.NetworkDialer()
	if dialer == nil {
		return nil
	}
	return plannerPluginNetworkDialerAdapter{inner: dialer}
}

func (r *Resolver) resolvePluginBatch(decoratorName string, calls []decoratorCall) error {
	capability := coreplugin.Global().Lookup(decoratorName)
	valueCapability, ok := capability.(coreplugin.ValueProvider)
	if !ok {
		return fmt.Errorf("plugin capability @%s is not a value capability", decoratorName)
	}

	ctx := plannerPluginValueContext{
		ctx:      context.Background(),
		session:  plannerPluginSession{inner: r.session},
		planHash: plannerPlanHashString(r),
		lookup:   r.getValue,
	}
	schema := capability.Schema()
	if batchCapability, ok := capability.(coreplugin.BatchValueProvider); ok {
		batchArgs := make([]coreplugin.ResolvedArgs, 0, len(calls))
		for _, call := range calls {
			valueCall, err := buildValueCall(call.decorator, r.getValue)
			if err != nil {
				return err
			}
			params := make(map[string]any, len(valueCall.Params)+1)
			for key, value := range valueCall.Params {
				params[key] = value
			}
			if valueCall.Primary != nil && schema.Primary.Name != "" {
				params[schema.Primary.Name] = *valueCall.Primary
			}
			batchArgs = append(batchArgs, plannerResolvedArgs{params: params, schema: schema})
		}
		resolved, err := batchCapability.ResolveBatch(ctx, batchArgs)
		if err != nil {
			return fmt.Errorf("failed to resolve @%s: %w (cannot plan if cannot resolve)", decoratorName, err)
		}
		if len(resolved) != len(calls) {
			return fmt.Errorf("failed to resolve @%s: expected %d results, got %d", decoratorName, len(calls), len(resolved))
		}
		for i, call := range calls {
			r.vault.StoreUnresolvedValue(call.exprID, resolved[i])
			r.vault.MarkTouched(call.exprID)
			r.decoratorExprIDs[decoratorKey(call.decorator)] = call.exprID
		}
		return nil
	}

	for _, call := range calls {
		valueCall, err := buildValueCall(call.decorator, r.getValue)
		if err != nil {
			return err
		}
		params := make(map[string]any, len(valueCall.Params)+1)
		for key, value := range valueCall.Params {
			params[key] = value
		}
		if valueCall.Primary != nil && schema.Primary.Name != "" {
			params[schema.Primary.Name] = *valueCall.Primary
		}
		resolved, err := valueCapability.Resolve(ctx, plannerResolvedArgs{params: params, schema: schema})
		if err != nil {
			return fmt.Errorf("failed to resolve @%s: %w (cannot plan if cannot resolve)", decoratorName, err)
		}
		r.vault.StoreUnresolvedValue(call.exprID, resolved)
		r.vault.MarkTouched(call.exprID)
		r.decoratorExprIDs[decoratorKey(call.decorator)] = call.exprID
	}

	return nil
}

func plannerPlanHashString(r *Resolver) string {
	if r == nil {
		return ""
	}
	if len(r.config.PlanHash) > 0 {
		return hex.EncodeToString(r.config.PlanHash)
	}
	if r.vault != nil {
		key := r.vault.GetPlanKey()
		if len(key) > 0 {
			return hex.EncodeToString(key)
		}
	}
	return ""
}
