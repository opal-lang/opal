package decorators

import (
	"context"
	"io/fs"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

type mockOSSession struct {
	platform string
}

func (m *mockOSSession) Platform() string {
	return m.platform
}

func (m *mockOSSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	return decorator.Result{}, nil
}

func (m *mockOSSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (m *mockOSSession) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockOSSession) Env() map[string]string {
	return nil
}

func (m *mockOSSession) WithEnv(delta map[string]string) decorator.Session {
	return m
}

func (m *mockOSSession) WithWorkdir(dir string) decorator.Session {
	return m
}

func (m *mockOSSession) Cwd() string {
	return ""
}

func (m *mockOSSession) ID() string {
	return "mock-os"
}

func (m *mockOSSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (m *mockOSSession) Close() error {
	return nil
}

func TestOSDecoratorsRegistered(t *testing.T) {
	if !decorator.Global().IsRegistered("os.Get") {
		t.Fatal("built-in decorator 'os.Get' should be registered")
	}
	if !decorator.Global().IsRegistered("os.Linux") {
		t.Fatal("built-in decorator 'os.Linux' should be registered")
	}
	if !decorator.Global().IsRegistered("os.macOS") {
		t.Fatal("built-in decorator 'os.macOS' should be registered")
	}
	if !decorator.Global().IsRegistered("os.Windows") {
		t.Fatal("built-in decorator 'os.Windows' should be registered")
	}
}

func TestOSGetDecoratorResolve(t *testing.T) {
	d := &osGetDecorator{}
	ctx := decorator.ValueEvalContext{
		Session: &mockOSSession{platform: "linux"},
	}

	result := resolveSingle(t, d, ctx, decorator.ValueCall{Path: "os.Get"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	if diff := cmp.Diff("linux", result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

func TestOSLinuxDecoratorResolve(t *testing.T) {
	d := &osLinuxDecorator{}
	ctx := decorator.ValueEvalContext{
		Session: &mockOSSession{platform: "linux"},
	}

	result := resolveSingle(t, d, ctx, decorator.ValueCall{Path: "os.Linux"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	if diff := cmp.Diff("true", result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

func TestOSMacOSDecoratorResolve(t *testing.T) {
	d := &osMacOSDecorator{}
	ctx := decorator.ValueEvalContext{
		Session: &mockOSSession{platform: "darwin"},
	}

	result := resolveSingle(t, d, ctx, decorator.ValueCall{Path: "os.macOS"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	if diff := cmp.Diff("true", result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

func TestOSWindowsDecoratorResolve(t *testing.T) {
	d := &osWindowsDecorator{}
	ctx := decorator.ValueEvalContext{
		Session: &mockOSSession{platform: "windows"},
	}

	result := resolveSingle(t, d, ctx, decorator.ValueCall{Path: "os.Windows"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	if diff := cmp.Diff("true", result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}
