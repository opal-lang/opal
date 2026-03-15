package planner

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/builtwithtofu/sigil/core/decorator"
	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
)

type plannerPluginValueContext struct {
	ctx     context.Context
	session coreplugin.ParentTransport
}

func (c plannerPluginValueContext) Context() context.Context            { return c.ctx }
func (c plannerPluginValueContext) Session() coreplugin.ParentTransport { return c.session }

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
	inner decorator.Session
}

func (s plannerPluginSession) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	result, err := s.inner.Run(ctx, argv, decorator.RunOpts{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr, Dir: opts.Dir})
	return coreplugin.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}
func (s plannerPluginSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.inner.Put(ctx, data, path, mode)
}
func (s plannerPluginSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.inner.Get(ctx, path)
}
func (s plannerPluginSession) Snapshot() coreplugin.SessionSnapshot {
	return coreplugin.SessionSnapshot{Env: s.inner.Env(), Workdir: s.inner.Cwd(), Platform: s.inner.Platform()}
}
func (s plannerPluginSession) Close() error { return nil }

type plannerOpenedTransportSession struct {
	inner coreplugin.OpenedTransport
}

func (s plannerOpenedTransportSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	result, err := s.inner.Run(ctx, argv, coreplugin.RunOpts{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr, Dir: opts.Dir})
	return decorator.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}
func (s plannerOpenedTransportSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.inner.Put(ctx, data, path, mode)
}
func (s plannerOpenedTransportSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.inner.Get(ctx, path)
}
func (s plannerOpenedTransportSession) Env() map[string]string                            { return s.inner.Snapshot().Env }
func (s plannerOpenedTransportSession) WithEnv(delta map[string]string) decorator.Session { return s }
func (s plannerOpenedTransportSession) WithWorkdir(dir string) decorator.Session          { return s }
func (s plannerOpenedTransportSession) Cwd() string                                       { return s.inner.Snapshot().Workdir }
func (s plannerOpenedTransportSession) ID() string                                        { return "planner-plugin-transport" }
func (s plannerOpenedTransportSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeAny
}
func (s plannerOpenedTransportSession) Platform() string { return s.inner.Snapshot().Platform }
func (s plannerOpenedTransportSession) Close() error     { return s.inner.Close() }

func (r *Resolver) resolvePluginBatch(decoratorName string, calls []decoratorCall) error {
	capability := coreplugin.Global().Lookup(decoratorName)
	valueCapability, ok := capability.(coreplugin.ValueCapability)
	if !ok {
		return fmt.Errorf("plugin capability @%s is not a value capability", decoratorName)
	}

	ctx := plannerPluginValueContext{ctx: context.Background(), session: plannerPluginSession{inner: r.session}}
	schema := capability.Schema()
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
