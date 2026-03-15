package builtins

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// TestTransportPlugin exposes test transport capabilities.
type TestTransportPlugin struct{}

func (p *TestTransportPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "test.transport", Version: "1.0.0", APIVersion: 1}
}

func (p *TestTransportPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{TestTransportCapability{}, TestTransportIdempotentCapability{}}
}

type TestTransportCapability struct{}

func (c TestTransportCapability) Path() string { return "test.transport" }

func (c TestTransportCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{{Name: "name", Type: types.TypeString, Default: "test"}},
		Block:  plugin.BlockRequired,
	}
}

func (c TestTransportCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	name := args.GetStringOptional("name")
	if name == "" {
		name = "test"
	}
	return newWrappedOpenedTransport(parent, "/"+name), nil
}

type TestTransportIdempotentCapability struct{}

func (c TestTransportIdempotentCapability) Path() string { return "test.transport.idempotent" }

func (c TestTransportIdempotentCapability) AllowTransportSensitiveValuesInPlan() bool {
	return true
}

func (c TestTransportIdempotentCapability) Schema() plugin.Schema {
	return plugin.Schema{Block: plugin.BlockRequired}
}

func (c TestTransportIdempotentCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newWrappedOpenedTransport(parent, "/test-idempotent"), nil
}

// SandboxPlugin exposes sandbox transport capability.
type SandboxPlugin struct{}

func (p *SandboxPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "sandbox", Version: "1.0.0", APIVersion: 1}
}

func (p *SandboxPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{SandboxTransportCapability{}}
}

type SandboxTransportCapability struct{}

func (c SandboxTransportCapability) Path() string { return "sandbox" }

func (c SandboxTransportCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "level", Type: types.TypeString, Default: "standard", Enum: []string{"none", "basic", "standard", "maximum"}},
			{Name: "network", Type: types.TypeString, Default: "allow", Enum: []string{"allow", "deny", "loopback"}},
		},
		Block: plugin.BlockRequired,
	}
}

func (c SandboxTransportCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	_ = parent
	_ = args.GetStringOptional("level")
	_ = args.GetStringOptional("network")
	return nil, fmt.Errorf("sandbox transport is not available through plugin capabilities")
}

// IsolatedPlugin exposes isolated.* transport capabilities.
type IsolatedPlugin struct{}

func (p *IsolatedPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "isolated", Version: "1.0.0", APIVersion: 1}
}

func (p *IsolatedPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{
		IsolatedNetworkLoopbackCapability{},
		IsolatedFilesystemReadonlyCapability{},
		IsolatedFilesystemEphemeralCapability{},
		IsolatedMemoryLockCapability{},
	}
}

type IsolatedNetworkLoopbackCapability struct{}

func (c IsolatedNetworkLoopbackCapability) Path() string { return "isolated.network.loopback" }
func (c IsolatedNetworkLoopbackCapability) Schema() plugin.Schema {
	return plugin.Schema{Block: plugin.BlockRequired}
}

func (c IsolatedNetworkLoopbackCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newWrappedOpenedTransport(parent, "/isolated.network.loopback"), nil
}

type IsolatedFilesystemReadonlyCapability struct{}

func (c IsolatedFilesystemReadonlyCapability) Path() string { return "isolated.filesystem.readonly" }
func (c IsolatedFilesystemReadonlyCapability) Schema() plugin.Schema {
	return plugin.Schema{Block: plugin.BlockRequired}
}

func (c IsolatedFilesystemReadonlyCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newWrappedOpenedTransport(parent, "/isolated.filesystem.readonly"), nil
}

type IsolatedFilesystemEphemeralCapability struct{}

func (c IsolatedFilesystemEphemeralCapability) Path() string { return "isolated.filesystem.ephemeral" }
func (c IsolatedFilesystemEphemeralCapability) Schema() plugin.Schema {
	return plugin.Schema{Block: plugin.BlockRequired}
}

func (c IsolatedFilesystemEphemeralCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newWrappedOpenedTransport(parent, "/isolated.filesystem.ephemeral"), nil
}

type IsolatedMemoryLockCapability struct{}

func (c IsolatedMemoryLockCapability) Path() string { return "isolated.memory.lock" }
func (c IsolatedMemoryLockCapability) Schema() plugin.Schema {
	return plugin.Schema{Block: plugin.BlockRequired}
}

func (c IsolatedMemoryLockCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return newWrappedOpenedTransport(parent, "/isolated.memory.lock"), nil
}

type wrappedOpenedTransport struct {
	parent   plugin.ParentTransport
	snapshot plugin.SessionSnapshot
}

func newWrappedOpenedTransport(parent plugin.ParentTransport, idSuffix string) plugin.OpenedTransport {
	snapshot := plugin.SessionSnapshot{Env: map[string]string{}, Platform: "linux"}
	if parent != nil {
		snapshot = parent.Snapshot()
	}
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	if idSuffix != "" {
		snapshot.Env["SIGIL_TRANSPORT_SUFFIX"] = idSuffix
	}
	return &wrappedOpenedTransport{parent: parent, snapshot: snapshot}
}

func (s *wrappedOpenedTransport) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	if s.parent == nil {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}
	return s.parent.Run(ctx, argv, opts)
}

func (s *wrappedOpenedTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	if s.parent == nil {
		return nil
	}
	return s.parent.Put(ctx, data, path, mode)
}

func (s *wrappedOpenedTransport) Get(ctx context.Context, path string) ([]byte, error) {
	if s.parent == nil {
		return nil, fmt.Errorf("path %q not found", path)
	}
	return s.parent.Get(ctx, path)
}

func (s *wrappedOpenedTransport) Snapshot() plugin.SessionSnapshot {
	env := make(map[string]string, len(s.snapshot.Env))
	for key, value := range s.snapshot.Env {
		env[key] = value
	}
	copySnapshot := s.snapshot
	copySnapshot.Env = env
	return copySnapshot
}

func (s *wrappedOpenedTransport) WithSnapshot(snapshot plugin.SessionSnapshot) plugin.OpenedTransport {
	copySnapshot := snapshot
	if copySnapshot.Env == nil {
		copySnapshot.Env = map[string]string{}
	}
	return &wrappedOpenedTransport{parent: s.parent, snapshot: copySnapshot}
}

func (s *wrappedOpenedTransport) Close() error { return nil }
