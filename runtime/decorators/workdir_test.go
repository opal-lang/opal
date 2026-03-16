package decorators

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/types"
	"github.com/google/go-cmp/cmp"
)

type testWorkdirSession struct {
	cwd string
}

func (s *testWorkdirSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	dir := s.cwd
	if opts.Dir != "" {
		dir = opts.Dir
	}
	if _, err := os.Stat(dir); err != nil {
		return decorator.Result{ExitCode: decorator.ExitFailure}, fmt.Errorf("workdir %q: %w", dir, err)
	}
	return decorator.Result{ExitCode: 0}, nil
}

func (s *testWorkdirSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (s *testWorkdirSession) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (s *testWorkdirSession) Env() map[string]string {
	return map[string]string{}
}

func (s *testWorkdirSession) WithEnv(delta map[string]string) decorator.Session {
	return &testWorkdirSession{cwd: s.cwd}
}

func (s *testWorkdirSession) WithWorkdir(dir string) decorator.Session {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(s.cwd, dir)
	}
	return &testWorkdirSession{cwd: dir}
}

func (s *testWorkdirSession) Cwd() string {
	return s.cwd
}

func (s *testWorkdirSession) ID() string {
	return "test-workdir"
}

func (s *testWorkdirSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (s *testWorkdirSession) Platform() string {
	return "linux"
}

func (s *testWorkdirSession) Close() error {
	return nil
}

func TestWorkdirInfo(t *testing.T) {
	entry, ok := decorator.Global().Lookup("fs.workdir")
	if !ok {
		t.Fatal("built-in decorator 'fs.workdir' should be registered")
	}

	if diff := cmp.Diff([]decorator.Role{decorator.RoleWrapper}, entry.Roles); diff != "" {
		t.Fatalf("roles mismatch (-want +got):\n%s", diff)
	}

	desc := entry.Impl.Descriptor()
	if diff := cmp.Diff("fs.workdir", desc.Path); diff != "" {
		t.Fatalf("path mismatch (-want +got):\n%s", diff)
	}
	param, ok := desc.Schema.Parameters["path"]
	if !ok {
		t.Fatal("descriptor should define required parameter 'path'")
	}
	if diff := cmp.Diff(types.TypeString, param.Type); diff != "" {
		t.Fatalf("parameter type mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, param.Required); diff != "" {
		t.Fatalf("required mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkdirExecution(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "nested")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir target failed: %v", err)
	}

	var got string
	dec := &WorkdirDecorator{}
	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		got = ctx.Session.Cwd()
		return decorator.Result{ExitCode: 0}, nil
	}}, map[string]any{"path": target})

	result, err := node.Execute(decorator.ExecContext{
		Context: context.Background(),
		Session: &testWorkdirSession{cwd: root},
	})
	if err != nil {
		t.Fatalf("workdir execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(target, got); diff != "" {
		t.Fatalf("cwd mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkdirIsolation(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	if err := os.Mkdir(dirA, 0o755); err != nil {
		t.Fatalf("mkdir dirA failed: %v", err)
	}
	if err := os.Mkdir(dirB, 0o755); err != nil {
		t.Fatalf("mkdir dirB failed: %v", err)
	}

	got := make([]string, 2)
	workdir := &WorkdirDecorator{}
	parallel := &ParallelDecorator{}
	next := &testBranchNode{branches: []func(ctx decorator.ExecContext) (decorator.Result, error){
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			node := workdir.Wrap(&testExecNode{execute: func(child decorator.ExecContext) (decorator.Result, error) {
				got[0] = child.Session.Cwd()
				return decorator.Result{ExitCode: 0}, nil
			}}, map[string]any{"path": "a"})
			return node.Execute(ctx)
		},
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			node := workdir.Wrap(&testExecNode{execute: func(child decorator.ExecContext) (decorator.Result, error) {
				got[1] = child.Session.Cwd()
				return decorator.Result{ExitCode: 0}, nil
			}}, map[string]any{"path": "b"})
			return node.Execute(ctx)
		},
	}}

	node := parallel.Wrap(next, map[string]any{"maxConcurrency": int64(2), "onFailure": "wait_all"})
	result, err := node.Execute(decorator.ExecContext{
		Context: context.Background(),
		Session: &testWorkdirSession{cwd: root},
	})
	if err != nil {
		t.Fatalf("parallel workdir execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{dirA, dirB}, got); diff != "" {
		t.Fatalf("parallel workdir mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkdirMissingDirectory(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")

	dec := &WorkdirDecorator{}
	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		return ctx.Session.Run(ctx.Context, []string{"pwd"}, decorator.RunOpts{})
	}}, map[string]any{"path": missing})

	result, err := node.Execute(decorator.ExecContext{
		Context: context.Background(),
		Session: &testWorkdirSession{cwd: root},
	})
	if err == nil {
		t.Fatal("expected missing directory error")
	}
	if diff := cmp.Diff(decorator.ExitFailure, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	want := fmt.Sprintf("workdir %q: stat %s: no such file or directory", missing, missing)
	if diff := cmp.Diff(want, err.Error()); diff != "" {
		t.Fatalf("error mismatch (-want +got):\n%s", diff)
	}
}
