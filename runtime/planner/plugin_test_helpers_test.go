package planner

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net"
	"sync"

	"github.com/builtwithtofu/sigil/core/decorator"
	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	coreruntime "github.com/builtwithtofu/sigil/core/runtime"
	"github.com/builtwithtofu/sigil/core/types"
)

type plannerSingleCapabilityPlugin struct {
	name       string
	capability coreplugin.Capability
}

type captureValuePluginCapability struct {
	path string
	dec  *captureValueDecorator
}

type orderedValuePluginCapability struct {
	path string
	impl *orderedValueDecorator
}

type countingValuePluginCapability struct {
	path string
	impl *countingValueDecorator
}

type poolCloseTransportPluginCapability struct {
	path       string
	openCount  *int
	closeCount *int
}

type transportEnvPluginCapability struct{}

type plannerMultiHopTransportPluginCapability struct{}

type contextAwareTransportPluginCapability struct {
	path      string
	valueKey  any
	seenValue *string
	openErr   error
}

type contextAwareValuePluginCapability struct {
	path      string
	valueKey  any
	seenValue *string
}

type defaultAwareValuePluginCapability struct {
	path         string
	seenDelay    *string
	seenAttempts *int
}

type plannerWrappedOpenedTransport struct {
	snapshot pluginSnapshot
	parent   coreplugin.ParentTransport
	closeFn  func() error
}

type pluginSnapshot = coreplugin.SessionSnapshot

var (
	plannerTestPluginMu sync.Mutex
	plannerDialCallsMu  sync.Mutex
	plannerDialCalls    []string
)

func (p *plannerSingleCapabilityPlugin) Identity() coreplugin.PluginIdentity {
	return coreplugin.PluginIdentity{Name: p.name, Version: "1.0.0", APIVersion: 1}
}

func (p *plannerSingleCapabilityPlugin) Capabilities() []coreplugin.Capability {
	return []coreplugin.Capability{p.capability}
}

func registerPlannerCapability(path string, capability coreplugin.Capability) error {
	plannerTestPluginMu.Lock()
	defer plannerTestPluginMu.Unlock()
	if coreplugin.Global().Lookup(path) != nil {
		return nil
	}
	name := "planner-test-" + path
	return coreplugin.Global().Register(&plannerSingleCapabilityPlugin{name: name, capability: capability})
}

func (c captureValuePluginCapability) Path() string { return c.path }
func (c captureValuePluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{
		Primary: coreplugin.Param{Name: "primary", Type: types.TypeString},
		Params:  []coreplugin.Param{{Name: "fallback", Type: types.TypeString}},
		Returns: types.TypeString,
		Block:   coreplugin.BlockForbidden,
	}
}

func (c captureValuePluginCapability) Resolve(ctx coreplugin.ValueContext, args coreplugin.ResolvedArgs) (any, error) {
	values, err := c.ResolveBatch(ctx, []coreplugin.ResolvedArgs{args})
	if err != nil {
		return nil, err
	}
	return values[0], nil
}

func (c captureValuePluginCapability) ResolveBatch(ctx coreplugin.ValueContext, args []coreplugin.ResolvedArgs) ([]any, error) {
	valueCtx := plannerDecoratorValueContext(ctx)
	calls := make([]decorator.ValueCall, 0, len(args))
	for _, arg := range args {
		call := decorator.ValueCall{Path: c.path, Params: map[string]any{}}
		if primary := arg.GetStringOptional("primary"); primary != "" {
			call.Primary = &primary
		}
		if fallback := arg.GetStringOptional("fallback"); fallback != "" {
			call.Params["fallback"] = fallback
		}
		calls = append(calls, call)
	}
	results, err := c.dec.Resolve(valueCtx, calls...)
	if err != nil {
		return nil, err
	}
	values := make([]any, 0, len(results))
	for _, result := range results {
		values = append(values, result.Value)
	}
	return values, nil
}

func (c orderedValuePluginCapability) Path() string { return c.path }
func (c orderedValuePluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Returns: types.TypeString, Block: coreplugin.BlockForbidden}
}

func (c orderedValuePluginCapability) Resolve(ctx coreplugin.ValueContext, args coreplugin.ResolvedArgs) (any, error) {
	values, err := c.ResolveBatch(ctx, []coreplugin.ResolvedArgs{args})
	if err != nil {
		return nil, err
	}
	return values[0], nil
}

func (c orderedValuePluginCapability) ResolveBatch(ctx coreplugin.ValueContext, args []coreplugin.ResolvedArgs) ([]any, error) {
	results, err := c.impl.Resolve(plannerDecoratorValueContext(ctx), make([]decorator.ValueCall, len(args))...)
	if err != nil {
		return nil, err
	}
	values := make([]any, 0, len(results))
	for _, result := range results {
		values = append(values, result.Value)
	}
	return values, nil
}

func (c countingValuePluginCapability) Path() string { return c.path }
func (c countingValuePluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Primary: coreplugin.Param{Name: "primary", Type: types.TypeString}, Returns: types.TypeBool, Block: coreplugin.BlockForbidden}
}

func (c countingValuePluginCapability) Resolve(ctx coreplugin.ValueContext, args coreplugin.ResolvedArgs) (any, error) {
	values, err := c.ResolveBatch(ctx, []coreplugin.ResolvedArgs{args})
	if err != nil {
		return nil, err
	}
	return values[0], nil
}

func (c countingValuePluginCapability) ResolveBatch(ctx coreplugin.ValueContext, args []coreplugin.ResolvedArgs) ([]any, error) {
	calls := make([]decorator.ValueCall, 0, len(args))
	for _, arg := range args {
		call := decorator.ValueCall{Path: c.path, Params: map[string]any{}}
		if primary := arg.GetStringOptional("primary"); primary != "" {
			call.Primary = &primary
		}
		calls = append(calls, call)
	}
	results, err := c.impl.Resolve(plannerDecoratorValueContext(ctx), calls...)
	if err != nil {
		return nil, err
	}
	values := make([]any, 0, len(results))
	for _, result := range results {
		values = append(values, result.Value)
	}
	return values, nil
}

func (c poolCloseTransportPluginCapability) Path() string { return c.path }
func (c poolCloseTransportPluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Block: coreplugin.BlockRequired}
}

func (c poolCloseTransportPluginCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	_ = ctx
	_ = args
	*c.openCount = *c.openCount + 1
	snapshot := parent.Snapshot()
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	return &plannerWrappedOpenedTransport{snapshot: snapshot, parent: parent, closeFn: func() error {
		*c.closeCount = *c.closeCount + 1
		return nil
	}}, nil
}

func (c transportEnvPluginCapability) Path() string { return "test.transport.env" }
func (c transportEnvPluginCapability) AllowTransportSensitiveValuesInPlan() bool {
	return true
}

func (c transportEnvPluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Block: coreplugin.BlockRequired}
}

func (c transportEnvPluginCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	_ = ctx
	_ = args
	snapshot := parent.Snapshot()
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	snapshot.Env["HOME"] = "remote-home"
	return &plannerWrappedOpenedTransport{snapshot: snapshot, parent: parent}, nil
}

func (c plannerMultiHopTransportPluginCapability) Path() string { return "test.transport.multihop" }

func (c contextAwareTransportPluginCapability) Path() string { return c.path }

func (c contextAwareTransportPluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Block: coreplugin.BlockRequired}
}

func (c contextAwareTransportPluginCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	_ = args
	if value, ok := ctx.Value(c.valueKey).(string); ok && c.seenValue != nil {
		*c.seenValue = value
	}
	if c.openErr != nil {
		return nil, c.openErr
	}
	snapshot := parent.Snapshot()
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	return &plannerWrappedOpenedTransport{snapshot: snapshot, parent: parent}, nil
}

func (c contextAwareValuePluginCapability) Path() string { return c.path }

func (c contextAwareValuePluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Returns: types.TypeString, Block: coreplugin.BlockForbidden}
}

func (c contextAwareValuePluginCapability) Resolve(ctx coreplugin.ValueContext, args coreplugin.ResolvedArgs) (any, error) {
	_ = args
	if value, ok := ctx.Context().Value(c.valueKey).(string); ok && c.seenValue != nil {
		*c.seenValue = value
	}
	return "ok", nil
}

func (c defaultAwareValuePluginCapability) Path() string { return c.path }

func (c defaultAwareValuePluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{
		Params: []coreplugin.Param{
			{Name: "delay", Type: types.TypeString, Default: "5s"},
			{Name: "attempts", Type: types.TypeInt, Default: 3},
		},
		Returns: types.TypeString,
		Block:   coreplugin.BlockForbidden,
	}
}

func (c defaultAwareValuePluginCapability) Resolve(ctx coreplugin.ValueContext, args coreplugin.ResolvedArgs) (any, error) {
	_ = ctx
	if c.seenDelay != nil {
		*c.seenDelay = args.GetStringOptional("delay")
	}
	if c.seenAttempts != nil {
		*c.seenAttempts = args.GetInt("attempts")
	}
	return "ok", nil
}

func (c plannerMultiHopTransportPluginCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "addr", Type: types.TypeString, Required: true}, {Name: "id", Type: types.TypeString, Required: true}}, Block: coreplugin.BlockRequired}
}

func (c plannerMultiHopTransportPluginCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	provider, ok := parent.(coreplugin.NetworkDialerProvider)
	if !ok || provider.NetworkDialer() == nil {
		return nil, fmt.Errorf("parent transport %T does not implement NetworkDialer", parent)
	}
	addr := args.GetString("addr")
	id := args.GetString("id")
	conn, err := provider.NetworkDialer().DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.Close()
	snapshot := parent.Snapshot()
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	return &plannerMultiHopOpenedTransport{id: id, parent: parent, parentDialer: provider.NetworkDialer(), snapshot: snapshot}, nil
}

type plannerMultiHopOpenedTransport struct {
	id           string
	parent       coreplugin.ParentTransport
	parentDialer coreplugin.NetworkDialer
	snapshot     coreplugin.SessionSnapshot
}

func (s *plannerMultiHopOpenedTransport) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *plannerMultiHopOpenedTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *plannerMultiHopOpenedTransport) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}
func (s *plannerMultiHopOpenedTransport) Snapshot() coreplugin.SessionSnapshot { return s.snapshot }
func (s *plannerMultiHopOpenedTransport) WithSnapshot(snapshot coreplugin.SessionSnapshot) coreplugin.OpenedTransport {
	return &plannerMultiHopOpenedTransport{id: s.id, parent: s.parent, parentDialer: s.parentDialer, snapshot: snapshot}
}
func (s *plannerMultiHopOpenedTransport) Close() error { return nil }
func (s *plannerMultiHopOpenedTransport) NetworkDialer() coreplugin.NetworkDialer {
	return plannerMultiHopDialer{id: s.id, parent: s.parentDialer}
}

type plannerMultiHopDialer struct {
	id     string
	parent coreplugin.NetworkDialer
}

func (d plannerMultiHopDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	recordPlannerDial(d.id, addr)
	clientConn, serverConn := net.Pipe()
	_ = serverConn.Close()
	return clientConn, nil
}

type plannerRootDialerSession struct{ mockSession }

func (s *plannerRootDialerSession) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	recordPlannerDial("local", addr)
	clientConn, serverConn := net.Pipe()
	_ = serverConn.Close()
	return clientConn, nil
}

func resetPlannerDialCalls() {
	plannerDialCallsMu.Lock()
	defer plannerDialCallsMu.Unlock()
	plannerDialCalls = nil
}

func recordPlannerDial(sourceID, addr string) {
	plannerDialCallsMu.Lock()
	defer plannerDialCallsMu.Unlock()
	plannerDialCalls = append(plannerDialCalls, sourceID+"->"+addr)
}

func plannerDialCallsValue() []string {
	plannerDialCallsMu.Lock()
	defer plannerDialCallsMu.Unlock()
	return append([]string(nil), plannerDialCalls...)
}

func (s *plannerWrappedOpenedTransport) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	if s.parent == nil {
		return coreplugin.Result{ExitCode: coreplugin.ExitSuccess}, nil
	}
	return s.parent.Run(ctx, argv, opts)
}

func (s *plannerWrappedOpenedTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	if s.parent == nil {
		return nil
	}
	return s.parent.Put(ctx, data, path, mode)
}

func (s *plannerWrappedOpenedTransport) Get(ctx context.Context, path string) ([]byte, error) {
	if s.parent == nil {
		return nil, fmt.Errorf("path %q not found", path)
	}
	return s.parent.Get(ctx, path)
}
func (s *plannerWrappedOpenedTransport) Snapshot() coreplugin.SessionSnapshot { return s.snapshot }
func (s *plannerWrappedOpenedTransport) WithSnapshot(snapshot coreplugin.SessionSnapshot) coreplugin.OpenedTransport {
	return &plannerWrappedOpenedTransport{snapshot: snapshot, parent: s.parent, closeFn: s.closeFn}
}

func (s *plannerWrappedOpenedTransport) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}

func plannerDecoratorValueContext(ctx coreplugin.ValueContext) decorator.ValueEvalContext {
	planHash, _ := hex.DecodeString(ctx.PlanHash())
	var session coreruntime.Session
	if pluginSession, ok := ctx.Session().(interface{ decoratorSession() coreruntime.Session }); ok {
		session = pluginSession.decoratorSession()
	} else {
		session = &mockSession{}
	}
	return decorator.ValueEvalContext{Session: session, LookupValue: ctx.LookupValue, PlanHash: planHash, StepPath: "phase3.step.test.capture.ctxmeta"}
}
