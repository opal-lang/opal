package executor

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"sync"

	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

type parentDialerTestPlugin struct{}

type multiHopTransportCapability struct{}

type noDialerTransportCapability struct{}

type multiHopOpenedTransport struct {
	id           string
	parent       coreplugin.ParentTransport
	parentDialer coreplugin.NetworkDialer
	snapshot     coreplugin.SessionSnapshot
}

type noDialerOpenedTransport struct {
	id       string
	parent   coreplugin.ParentTransport
	snapshot coreplugin.SessionSnapshot
}

var registerParentDialerTestPluginOnce sync.Once

func (p *parentDialerTestPlugin) Identity() coreplugin.PluginIdentity {
	return coreplugin.PluginIdentity{Name: "executor-parent-dialer-test", Version: "1.0.0", APIVersion: 1}
}

func (p *parentDialerTestPlugin) Capabilities() []coreplugin.Capability {
	return []coreplugin.Capability{multiHopTransportCapability{}, noDialerTransportCapability{}}
}

func (c multiHopTransportCapability) Path() string { return "test.transport.multihop" }

func (c multiHopTransportCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{
		Params: []coreplugin.Param{{Name: "addr", Type: types.TypeString, Required: true}, {Name: "id", Type: types.TypeString, Required: true}},
	}
}

func (c multiHopTransportCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	provider, ok := parent.(coreplugin.NetworkDialerProvider)
	if !ok || provider.NetworkDialer() == nil {
		return nil, fmt.Errorf("parent transport %T does not implement NetworkDialer", parent)
	}

	addr := args.GetString("addr")
	id := args.GetString("id")
	conn, err := provider.NetworkDialer().DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial through parent network dialer to %q: %w", addr, err)
	}
	_ = conn.Close()

	snapshot := parent.Snapshot()
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	return &multiHopOpenedTransport{id: id, parent: parent, parentDialer: provider.NetworkDialer(), snapshot: snapshot}, nil
}

func (c noDialerTransportCapability) Path() string { return "test.transport.nodialer" }

func (c noDialerTransportCapability) Schema() coreplugin.Schema {
	return coreplugin.Schema{Params: []coreplugin.Param{{Name: "id", Type: types.TypeString, Required: true}}}
}

func (c noDialerTransportCapability) Open(ctx context.Context, parent coreplugin.ParentTransport, args coreplugin.ResolvedArgs) (coreplugin.OpenedTransport, error) {
	_ = ctx
	snapshot := parent.Snapshot()
	if snapshot.Env == nil {
		snapshot.Env = map[string]string{}
	}
	return &noDialerOpenedTransport{id: args.GetString("id"), parent: parent, snapshot: snapshot}, nil
}

func (s *multiHopOpenedTransport) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *multiHopOpenedTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *multiHopOpenedTransport) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}

func (s *multiHopOpenedTransport) Snapshot() coreplugin.SessionSnapshot { return s.snapshot }

func (s *multiHopOpenedTransport) WithSnapshot(snapshot coreplugin.SessionSnapshot) coreplugin.OpenedTransport {
	return &multiHopOpenedTransport{id: s.id, parent: s.parent, parentDialer: s.parentDialer, snapshot: snapshot}
}

func (s *multiHopOpenedTransport) Close() error { return nil }

func (s *multiHopOpenedTransport) NetworkDialer() coreplugin.NetworkDialer { return s }

func (s *multiHopOpenedTransport) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	recordMultiHopDial(s.id, addr)
	clientConn, serverConn := net.Pipe()
	_ = serverConn.Close()
	return clientConn, nil
}

func (s *noDialerOpenedTransport) Run(ctx context.Context, argv []string, opts coreplugin.RunOpts) (coreplugin.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *noDialerOpenedTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *noDialerOpenedTransport) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}

func (s *noDialerOpenedTransport) Snapshot() coreplugin.SessionSnapshot { return s.snapshot }

func (s *noDialerOpenedTransport) WithSnapshot(snapshot coreplugin.SessionSnapshot) coreplugin.OpenedTransport {
	return &noDialerOpenedTransport{id: s.id, parent: s.parent, snapshot: snapshot}
}

func (s *noDialerOpenedTransport) Close() error { return nil }

func registerParentDialerTestPlugin() {
	registerParentDialerTestPluginOnce.Do(func() {
		_ = coreplugin.Global().Register(&parentDialerTestPlugin{})
	})
}
