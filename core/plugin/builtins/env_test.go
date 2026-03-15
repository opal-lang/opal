package builtins

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

type fakeValueContext struct {
	session plugin.ParentTransport
}

func (f fakeValueContext) Context() context.Context        { return context.Background() }
func (f fakeValueContext) Session() plugin.ParentTransport { return f.session }

type fakeArgs struct {
	strings   map[string]string
	ints      map[string]int
	durations map[string]time.Duration
}

func (f fakeArgs) GetString(name string) string              { return f.strings[name] }
func (f fakeArgs) GetStringOptional(name string) string      { return f.strings[name] }
func (f fakeArgs) GetInt(name string) int                    { return f.ints[name] }
func (f fakeArgs) GetDuration(name string) time.Duration     { return f.durations[name] }
func (f fakeArgs) ResolveSecret(name string) (string, error) { return "", nil }

type fakeSession struct {
	env map[string]string
	cwd string
}

func (f *fakeSession) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	return plugin.Result{}, nil
}
func (f *fakeSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}
func (f *fakeSession) Get(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (f *fakeSession) Snapshot() plugin.SessionSnapshot {
	workdir := f.cwd
	if workdir == "" {
		workdir = "/tmp"
	}
	return plugin.SessionSnapshot{Env: f.env, Workdir: workdir, Platform: "linux"}
}
func (f *fakeSession) Close() error { return nil }

func TestEnvValueCapabilityResolveFromSession(t *testing.T) {
	capability := EnvValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{"HOME": "/home/tester"}}}
	args := fakeArgs{strings: map[string]string{"property": "HOME"}}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if diff := cmp.Diff("/home/tester", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestEnvValueCapabilityResolveDefault(t *testing.T) {
	capability := EnvValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}}
	args := fakeArgs{strings: map[string]string{"property": "MISSING", "default": "fallback"}}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if diff := cmp.Diff("fallback", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestEnvValueCapabilityResolveMissing(t *testing.T) {
	capability := EnvValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}}
	args := fakeArgs{strings: map[string]string{"property": "MISSING"}}

	_, err := capability.Resolve(ctx, args)
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if diff := cmp.Diff("environment variable \"MISSING\" not found", err.Error()); diff != "" {
		t.Fatalf("Resolve() error mismatch (-want +got):\n%s", diff)
	}
}
