package executor

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/builtwithtofu/sigil/core/planfmt"
	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	coreruntime "github.com/builtwithtofu/sigil/core/runtime"
	"github.com/builtwithtofu/sigil/core/sdk"
)

type pluginExecContext struct {
	ctx     context.Context
	session coreplugin.ParentTransport
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (c pluginExecContext) Context() context.Context            { return c.ctx }
func (c pluginExecContext) Session() coreplugin.ParentTransport { return c.session }
func (c pluginExecContext) Stdin() io.Reader                    { return c.stdin }
func (c pluginExecContext) Stdout() io.Writer                   { return c.stdout }
func (c pluginExecContext) Stderr() io.Writer                   { return c.stderr }

type pluginArgs struct {
	params  map[string]any
	secrets map[string]any
	schema  coreplugin.Schema
	resolve func(displayID string) (string, error)
}

func (a pluginArgs) GetString(name string) string {
	if value, ok := a.params[name].(string); ok {
		return value
	}
	return ""
}
func (a pluginArgs) GetStringOptional(name string) string { return a.GetString(name) }
func (a pluginArgs) GetInt(name string) int {
	switch value := a.params[name].(type) {
	case int:
		return value
	case int64:
		return int(value)
	}
	return 0
}

func (a pluginArgs) GetDuration(name string) time.Duration {
	switch value := a.params[name].(type) {
	case time.Duration:
		return value
	case string:
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func (a pluginArgs) ResolveSecret(name string) (string, error) {
	if !a.schema.DeclaresSecret(name) {
		return "", fmt.Errorf("capability did not declare secret %q", name)
	}
	if value, ok := a.secrets[name]; ok {
		if displayID, ok := value.(string); ok && strings.HasPrefix(displayID, "sigil:") && a.resolve != nil {
			return a.resolve(displayID)
		}
		if secret, ok := value.(string); ok {
			return secret, nil
		}
	}
	return "", fmt.Errorf("declared secret %q not provided", name)
}

type pluginParentSession struct{ session coreruntime.Session }

type runtimeNetworkDialerAdapter struct{ inner coreruntime.NetworkDialer }

func (d runtimeNetworkDialerAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.inner.DialContext(ctx, network, addr)
}

type pluginNetworkDialerAdapter struct{ inner coreplugin.NetworkDialer }

func (d pluginNetworkDialerAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.inner.DialContext(ctx, network, addr)
}

func (s pluginParentSession) decoratorSession() coreruntime.Session { return s.session }

func (s pluginParentSession) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	result, err := s.session.Run(ctx, argv, coreruntime.RunOpts{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr, Dir: opts.Dir})
	return coreplugin.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}

func (s pluginParentSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.session.Put(ctx, data, path, mode)
}

func (s pluginParentSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.session.Get(ctx, path)
}

func (s pluginParentSession) Snapshot() coreplugin.SessionSnapshot {
	env := s.session.Env()
	copyEnv := make(map[string]string, len(env))
	for key, value := range env {
		copyEnv[key] = value
	}
	return coreplugin.SessionSnapshot{Env: copyEnv, Workdir: s.session.Cwd(), Platform: s.session.Platform()}
}
func (s pluginParentSession) Close() error { return nil }
func (s pluginParentSession) NetworkDialer() coreplugin.NetworkDialer {
	if dialer, ok := s.session.(coreruntime.NetworkDialer); ok {
		if dialer == nil {
			return nil
		}
		return runtimeNetworkDialerAdapter{inner: dialer}
	}
	dialer, err := coreruntime.GetNetworkDialer(s.session)
	if err != nil {
		return nil
	}
	if dialer == nil {
		return nil
	}
	return runtimeNetworkDialerAdapter{inner: dialer}
}

type pluginNextNode struct {
	node    coreruntime.ExecNode
	session coreruntime.Session
}

func (n pluginNextNode) Execute(ctx coreplugin.ExecContext) (coreplugin.Result, error) {
	execSession := n.session
	if parent := ctx.Session(); parent != nil {
		if session, ok := decoratorSessionFromParent(parent); ok {
			execSession = session
		}
	}
	result, err := n.node.Execute(coreruntime.ExecContext{
		Context: ctx.Context(),
		Session: execSession,
		Stdin:   ctx.Stdin(),
		Stdout:  ctx.Stdout(),
		Stderr:  ctx.Stderr(),
	})
	return coreplugin.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}

func decoratorSessionFromParent(parent coreplugin.ParentTransport) (coreruntime.Session, bool) {
	root := parent
	for {
		wrapper, ok := root.(coreplugin.ParentTransportWrapper)
		if !ok {
			break
		}
		next := wrapper.UnwrapParentTransport()
		if next == nil || next == root {
			break
		}
		root = next
	}

	base, ok := root.(interface{ decoratorSession() coreruntime.Session })
	if !ok {
		return nil, false
	}

	session := base.decoratorSession()
	snapshot := parent.Snapshot()
	if snapshot.Workdir != "" {
		session = session.WithWorkdir(snapshot.Workdir)
	}
	if len(snapshot.Env) > 0 {
		env := make(map[string]string, len(snapshot.Env))
		for key, value := range snapshot.Env {
			env[key] = value
		}
		session = session.WithEnv(env)
	}

	return session, true
}

func isPluginOnlyCapability(path string) coreplugin.Capability {
	return coreplugin.Global().Lookup(path)
}

func applyPluginSession(execCtx sdk.ExecutionContext, session coreplugin.OpenedTransport) sdk.ExecutionContext {
	child := execCtx
	snapshot := session.Snapshot()
	if env := snapshot.Env; env != nil {
		replaced := make(map[string]string, len(env))
		for key, value := range env {
			replaced[key] = value
		}
		child = child.WithEnviron(replaced)
	}
	if workdir := snapshot.Workdir; workdir != "" {
		child = child.WithWorkdir(workdir)
	}
	return child
}

type pluginTransportSession struct {
	id    string
	inner coreplugin.OpenedTransport
}

func (s pluginTransportSession) Run(ctx context.Context, argv []string, opts coreruntime.RunOpts) (coreruntime.Result, error) {
	result, err := s.inner.Run(ctx, argv, coreplugin.RunOpts{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr, Dir: opts.Dir})
	return coreruntime.Result{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, err
}

func (s pluginTransportSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.inner.Put(ctx, data, path, mode)
}

func (s pluginTransportSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.inner.Get(ctx, path)
}

func (s pluginTransportSession) Env() map[string]string {
	snapshot := s.inner.Snapshot()
	env := make(map[string]string, len(snapshot.Env))
	for key, value := range snapshot.Env {
		env[key] = value
	}
	return env
}

func (s pluginTransportSession) WithEnv(delta map[string]string) coreruntime.Session {
	snapshot := s.inner.Snapshot()
	merged := make(map[string]string, len(snapshot.Env)+len(delta))
	for key, value := range snapshot.Env {
		merged[key] = value
	}
	for key, value := range delta {
		merged[key] = value
	}
	snapshot.Env = merged
	return pluginTransportSession{id: s.id, inner: s.inner.WithSnapshot(snapshot)}
}

func (s pluginTransportSession) WithWorkdir(dir string) coreruntime.Session {
	snapshot := s.inner.Snapshot()
	snapshot.Workdir = dir
	return pluginTransportSession{id: s.id, inner: s.inner.WithSnapshot(snapshot)}
}
func (s pluginTransportSession) Cwd() string { return s.inner.Snapshot().Workdir }
func (s pluginTransportSession) ID() string  { return s.id }
func (s pluginTransportSession) TransportScope() coreruntime.TransportScope {
	return coreruntime.TransportScopeAny
}
func (s pluginTransportSession) Platform() string { return s.inner.Snapshot().Platform }
func (s pluginTransportSession) Close() error     { return s.inner.Close() }
func (s pluginTransportSession) NetworkDialer() coreruntime.NetworkDialer {
	provider, ok := s.inner.(coreplugin.NetworkDialerProvider)
	if !ok {
		return nil
	}
	dialer := provider.NetworkDialer()
	if dialer == nil {
		return nil
	}
	return pluginNetworkDialerAdapter{inner: dialer}
}

func newPluginArgs(params map[string]any, schema coreplugin.Schema, resolve func(string) (string, error)) pluginArgs {
	plain := make(map[string]any, len(params))
	secrets := make(map[string]any)
	for key, value := range params {
		if schema.DeclaresSecret(key) {
			secrets[key] = value
			continue
		}
		plain[key] = value
	}
	return pluginArgs{params: plain, secrets: secrets, schema: schema, resolve: resolve}
}

func syntheticPluginTransportID(currentID, decoratorName string, params map[string]any) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(currentID))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(decoratorName))
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		value := params[key]
		_, _ = h.Write([]byte("|" + key + "=" + fmt.Sprint(value)))
	}
	return fmt.Sprintf("plugin:%x", h.Sum64())
}

func (e *executor) executePluginWrapper(execCtx sdk.ExecutionContext, next coreruntime.ExecNode, capability coreplugin.Wrapper, params map[string]any, stdin io.Reader, stdout io.Writer) int {
	baseSession, sessionErr := e.sessions.SessionFor(executionTransportID(execCtx))
	if sessionErr != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error creating session: %v\n", sessionErr)
		return coreruntime.ExitFailure
	}
	session := sessionForExecutionContext(baseSession, execCtx)
	resolver := func(displayID string) (string, error) {
		if e.vault == nil {
			return "", fmt.Errorf("secret resolver unavailable")
		}
		value, err := e.vault.ResolveDisplayIDWithTransport(displayID, executionTransportID(execCtx))
		if err != nil {
			return "", err
		}
		return fmt.Sprint(value), nil
	}
	node := capability.Wrap(pluginNextNode{node: next, session: session}, newPluginArgs(params, capability.Schema(), resolver))
	result, err := node.Execute(pluginExecContext{ctx: execCtx.Context(), session: pluginParentSession{session: session}, stdin: stdin, stdout: stdout, stderr: e.stderr})
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", err)
	}
	return result.ExitCode
}

func (e *executor) executePluginTransport(execCtx sdk.ExecutionContext, steps []sdk.Step, capability coreplugin.Transport, params map[string]any) int {
	baseSession, sessionErr := e.sessions.SessionFor(executionTransportID(execCtx))
	if sessionErr != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error creating session: %v\n", sessionErr)
		return coreruntime.ExitFailure
	}
	session := sessionForExecutionContext(baseSession, execCtx)
	transportID := syntheticPluginTransportID(executionTransportID(execCtx), capability.Path(), params)
	resolver := func(displayID string) (string, error) {
		if e.vault == nil {
			return "", fmt.Errorf("secret resolver unavailable")
		}
		value, err := e.vault.ResolveDisplayIDWithTransport(displayID, executionTransportID(execCtx))
		if err != nil {
			return "", err
		}
		return fmt.Sprint(value), nil
	}
	adapterSession, err := e.sessions.AcquirePooledPluginSession(transportID, capability.Path(), session, params, func() (coreplugin.OpenedTransport, error) {
		return capability.Open(execCtx.Context(), pluginParentSession{session: session}, newPluginArgs(params, capability.Schema(), resolver))
	})
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", err)
		return coreruntime.ExitFailure
	}
	e.sessions.installSession(transportID, adapterSession)
	defer e.sessions.removeSession(transportID)
	child := withExecutionTransport(execCtx, transportID)
	if opened := pluginOpenedTransportSnapshot(adapterSession); opened != nil {
		child = applyPluginSession(child, opened)
	}
	exitCode, blockErr := child.ExecuteBlock(steps)
	if blockErr != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", blockErr)
		return coreruntime.ExitFailure
	}
	return exitCode
}

func (e *executor) executePlanPluginTransport(execCtx sdk.ExecutionContext, steps []planfmt.Step, capability coreplugin.Transport, params map[string]any) int {
	baseSession, sessionErr := e.sessions.SessionFor(executionTransportID(execCtx))
	if sessionErr != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error creating session: %v\n", sessionErr)
		return coreruntime.ExitFailure
	}
	session := sessionForExecutionContext(baseSession, execCtx)
	childTransportID := syntheticPluginTransportID(executionTransportID(execCtx), capability.Path(), params)
	if len(steps) > 0 {
		if derived := sourceTransportIDForPlan(steps[0].Tree); derived != "" {
			childTransportID = normalizedTransportID(derived)
		}
	}
	resolver := func(displayID string) (string, error) {
		if e.vault == nil {
			return "", fmt.Errorf("secret resolver unavailable")
		}
		value, err := e.vault.ResolveDisplayIDWithTransport(displayID, executionTransportID(execCtx))
		if err != nil {
			return "", err
		}
		return fmt.Sprint(value), nil
	}
	adapterSession, err := e.sessions.AcquirePooledPluginSession(childTransportID, capability.Path(), session, params, func() (coreplugin.OpenedTransport, error) {
		return capability.Open(execCtx.Context(), pluginParentSession{session: session}, newPluginArgs(params, capability.Schema(), resolver))
	})
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", err)
		return coreruntime.ExitFailure
	}
	e.sessions.installSession(childTransportID, adapterSession)
	defer e.sessions.removeSession(childTransportID)
	child := withExecutionTransport(execCtx, childTransportID)
	if opened := pluginOpenedTransportSnapshot(adapterSession); opened != nil {
		child = applyPluginSession(child, opened)
	}
	exitCode := e.executePlanBlock(child, steps)
	return exitCode
}

func pluginOpenedTransportSnapshot(session coreruntime.Session) coreplugin.OpenedTransport {
	if scoped, ok := session.(*transportScopedSession); ok {
		session = scoped.session
	}
	if pluginSession, ok := session.(pluginTransportSession); ok {
		return pluginSession.inner
	}
	return nil
}
