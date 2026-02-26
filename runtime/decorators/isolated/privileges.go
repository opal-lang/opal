package isolated

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/isolation"
)

// IsolationLevelMinimal applies minimum isolation that still drops privileges.
const IsolationLevelMinimal decorator.IsolationLevel = decorator.IsolationLevelBasic

type PrivilegesDropDecorator struct{}

var _ decorator.Transport = (*PrivilegesDropDecorator)(nil)

func (d *PrivilegesDropDecorator) Descriptor() decorator.Descriptor {
	desc := decorator.NewDescriptor(FullPath("privileges.drop")).
		Summary("Execute in isolation with dropped supplementary privileges").
		Roles(decorator.RoleBoundary).
		Block(decorator.BlockRequired).
		Build()

	desc.Capabilities.SupportedOS = []string{"linux", "darwin"}

	return desc
}

func (d *PrivilegesDropDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork |
		decorator.TransportCapFilesystem |
		decorator.TransportCapEnvironment |
		decorator.TransportCapIsolation
}

func (d *PrivilegesDropDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	_ = params

	isolator := isolation.NewIsolator()
	config := decorator.IsolationConfig{
		NetworkPolicy:    decorator.NetworkPolicyAllow,
		FilesystemPolicy: decorator.FilesystemPolicyFull,
		MemoryLock:       false,
	}

	if err := isolator.Isolate(IsolationLevelMinimal, config); err != nil {
		return nil, fmt.Errorf("failed to drop privileges: %w", err)
	}

	return &privilegesDropSession{parent: parent, isolator: isolator}, nil
}

func (d *PrivilegesDropDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	_ = params
	return next
}

func (d *PrivilegesDropDecorator) MaterializeSession() bool {
	return true
}

func (d *PrivilegesDropDecorator) IsolationContext() decorator.IsolationContext {
	return isolation.NewIsolator()
}

type privilegesDropSession struct {
	parent   decorator.Session
	isolator decorator.IsolationContext
}

var _ decorator.Session = (*privilegesDropSession)(nil)

func (s *privilegesDropSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *privilegesDropSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *privilegesDropSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}

func (s *privilegesDropSession) Env() map[string]string {
	return s.parent.Env()
}

func (s *privilegesDropSession) WithEnv(delta map[string]string) decorator.Session {
	return &privilegesDropSession{parent: s.parent.WithEnv(delta), isolator: s.isolator}
}

func (s *privilegesDropSession) WithWorkdir(dir string) decorator.Session {
	return &privilegesDropSession{parent: s.parent.WithWorkdir(dir), isolator: s.isolator}
}

func (s *privilegesDropSession) Cwd() string {
	return s.parent.Cwd()
}

func (s *privilegesDropSession) Platform() string {
	return s.parent.Platform()
}

func (s *privilegesDropSession) ID() string {
	return s.parent.ID() + "/isolated.privileges.drop"
}

func (s *privilegesDropSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeIsolated
}

func (s *privilegesDropSession) Close() error {
	return nil
}

func init() {
	path := FullPath("privileges.drop")
	if err := decorator.Register(path, &PrivilegesDropDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @%s decorator: %v", path, err))
	}
}
