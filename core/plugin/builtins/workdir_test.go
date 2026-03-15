package builtins

import (
	"context"
	"io/fs"
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

func TestWorkdirWrapperCapabilityAdjustsSessionSnapshot(t *testing.T) {
	capability := WorkdirWrapperCapability{}
	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		return plugin.Result{ExitCode: plugin.ExitSuccess, Stdout: []byte(ctx.Session().Snapshot().Workdir)}, nil
	}}, fakeArgs{strings: map[string]string{"path": "/tmp/sigil"}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background(), session: fakeParentSession{snapshot: plugin.SessionSnapshot{Workdir: "/home"}}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if diff := cmp.Diff("/tmp/sigil", string(result.Stdout)); diff != "" {
		t.Fatalf("Execute() workdir mismatch (-want +got):\n%s", diff)
	}
}

type fakeParentSession struct {
	snapshot plugin.SessionSnapshot
}

func (s fakeParentSession) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
}

func (s fakeParentSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}
func (s fakeParentSession) Get(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (s fakeParentSession) Snapshot() plugin.SessionSnapshot                     { return s.snapshot }
func (s fakeParentSession) Close() error                                         { return nil }
