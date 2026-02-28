package isolated

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/runtime/isolation"
)

type (
	FilesystemReadonlyDecorator  struct{}
	FilesystemEphemeralDecorator struct{}
)

var (
	_ decorator.Transport = (*FilesystemReadonlyDecorator)(nil)
	_ decorator.Transport = (*FilesystemEphemeralDecorator)(nil)
)

func (d *FilesystemReadonlyDecorator) Descriptor() decorator.Descriptor {
	desc := decorator.NewDescriptor(FullPath("filesystem.readonly")).
		Summary("Execute in isolation with read-only filesystem access").
		Roles(decorator.RoleBoundary).
		Block(decorator.BlockRequired).
		Build()

	desc.Capabilities.SupportedOS = []string{"linux"}

	return desc
}

func (d *FilesystemReadonlyDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork |
		decorator.TransportCapFilesystem |
		decorator.TransportCapEnvironment |
		decorator.TransportCapIsolation
}

func (d *FilesystemReadonlyDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	_ = params

	isolator := isolation.NewIsolator()
	config := decorator.IsolationConfig{
		NetworkPolicy:    decorator.NetworkPolicyAllow,
		FilesystemPolicy: decorator.FilesystemPolicyReadOnly,
		MemoryLock:       false,
	}

	if err := isolator.Isolate(decorator.IsolationLevelStandard, config); err != nil {
		return nil, fmt.Errorf("failed to create read-only filesystem isolation: %w", err)
	}

	return &filesystemReadonlySession{parent: parent, isolator: isolator}, nil
}

func (d *FilesystemReadonlyDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	_ = params
	return next
}

func (d *FilesystemReadonlyDecorator) MaterializeSession() bool {
	return true
}

func (d *FilesystemReadonlyDecorator) IsolationContext() decorator.IsolationContext {
	return isolation.NewIsolator()
}

func (d *FilesystemEphemeralDecorator) Descriptor() decorator.Descriptor {
	desc := decorator.NewDescriptor(FullPath("filesystem.ephemeral")).
		Summary("Execute in isolation with ephemeral filesystem access").
		Roles(decorator.RoleBoundary).
		Block(decorator.BlockRequired).
		Build()

	desc.Capabilities.SupportedOS = []string{"linux"}

	return desc
}

func (d *FilesystemEphemeralDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork |
		decorator.TransportCapFilesystem |
		decorator.TransportCapEnvironment |
		decorator.TransportCapIsolation
}

func (d *FilesystemEphemeralDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	_ = params

	isolator := isolation.NewIsolator()
	config := decorator.IsolationConfig{
		NetworkPolicy:    decorator.NetworkPolicyAllow,
		FilesystemPolicy: decorator.FilesystemPolicyEphemeral,
		MemoryLock:       false,
	}

	if err := isolator.Isolate(decorator.IsolationLevelStandard, config); err != nil {
		return nil, fmt.Errorf("failed to create ephemeral filesystem isolation: %w", err)
	}

	return &filesystemEphemeralSession{parent: parent, isolator: isolator}, nil
}

func (d *FilesystemEphemeralDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	_ = params
	return next
}

func (d *FilesystemEphemeralDecorator) MaterializeSession() bool {
	return true
}

func (d *FilesystemEphemeralDecorator) IsolationContext() decorator.IsolationContext {
	return isolation.NewIsolator()
}

type filesystemReadonlySession struct {
	parent   decorator.Session
	isolator decorator.IsolationContext
}

type filesystemEphemeralSession struct {
	parent   decorator.Session
	isolator decorator.IsolationContext
}

var (
	_ decorator.Session = (*filesystemReadonlySession)(nil)
	_ decorator.Session = (*filesystemEphemeralSession)(nil)
)

func (s *filesystemReadonlySession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *filesystemReadonlySession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *filesystemReadonlySession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}

func (s *filesystemReadonlySession) Env() map[string]string {
	return s.parent.Env()
}

func (s *filesystemReadonlySession) WithEnv(delta map[string]string) decorator.Session {
	return &filesystemReadonlySession{parent: s.parent.WithEnv(delta), isolator: s.isolator}
}

func (s *filesystemReadonlySession) WithWorkdir(dir string) decorator.Session {
	return &filesystemReadonlySession{parent: s.parent.WithWorkdir(dir), isolator: s.isolator}
}

func (s *filesystemReadonlySession) Cwd() string {
	return s.parent.Cwd()
}

func (s *filesystemReadonlySession) Platform() string {
	return s.parent.Platform()
}

func (s *filesystemReadonlySession) ID() string {
	return s.parent.ID() + "/isolated.filesystem.readonly"
}

func (s *filesystemReadonlySession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeIsolated
}

func (s *filesystemReadonlySession) Close() error {
	return nil
}

func (s *filesystemEphemeralSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return s.parent.Run(ctx, argv, opts)
}

func (s *filesystemEphemeralSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.parent.Put(ctx, data, path, mode)
}

func (s *filesystemEphemeralSession) Get(ctx context.Context, path string) ([]byte, error) {
	return s.parent.Get(ctx, path)
}

func (s *filesystemEphemeralSession) Env() map[string]string {
	return s.parent.Env()
}

func (s *filesystemEphemeralSession) WithEnv(delta map[string]string) decorator.Session {
	return &filesystemEphemeralSession{parent: s.parent.WithEnv(delta), isolator: s.isolator}
}

func (s *filesystemEphemeralSession) WithWorkdir(dir string) decorator.Session {
	return &filesystemEphemeralSession{parent: s.parent.WithWorkdir(dir), isolator: s.isolator}
}

func (s *filesystemEphemeralSession) Cwd() string {
	return s.parent.Cwd()
}

func (s *filesystemEphemeralSession) Platform() string {
	return s.parent.Platform()
}

func (s *filesystemEphemeralSession) ID() string {
	return s.parent.ID() + "/isolated.filesystem.ephemeral"
}

func (s *filesystemEphemeralSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeIsolated
}

func (s *filesystemEphemeralSession) Close() error {
	return nil
}

func init() {
	path := FullPath("filesystem.readonly")
	if err := decorator.Register(path, &FilesystemReadonlyDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @%s decorator: %v", path, err))
	}

	path = FullPath("filesystem.ephemeral")
	if err := decorator.Register(path, &FilesystemEphemeralDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @%s decorator: %v", path, err))
	}
}
