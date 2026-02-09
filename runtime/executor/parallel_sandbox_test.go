package executor

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/sdk"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

type testChdirDecorator struct{}

func (d *testChdirDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.chdir").
		Summary("Test-only decorator that changes session workdir for its block").
		Roles(decorator.RoleWrapper).
		ParamString("dir", "Workdir for nested block").
		Required().
		Done().
		Block(decorator.BlockRequired).
		Build()
}

func (d *testChdirDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &testChdirNode{next: next, params: params}
}

type testChdirNode struct {
	next   decorator.ExecNode
	params map[string]any
}

func (n *testChdirNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if n.next == nil {
		return decorator.Result{ExitCode: 0}, nil
	}

	dir, _ := n.params["dir"].(string)
	if dir == "" {
		return decorator.Result{ExitCode: 1}, nil
	}

	child := ctx.WithSession(ctx.Session.WithWorkdir(dir))
	return n.next.Execute(child)
}

var registerTestChdirDecoratorOnce sync.Once

func registerTestChdirDecorator(t *testing.T) {
	t.Helper()
	var registerErr error
	registerTestChdirDecoratorOnce.Do(func() {
		registerErr = decorator.Register("test.chdir", &testChdirDecorator{})
	})
	if registerErr != nil {
		t.Fatalf("register @test.chdir failed: %v", registerErr)
	}
}

func TestParallelBranchWorkdirIsolation(t *testing.T) {
	registerTestChdirDecorator(t)
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	dirA := t.TempDir()
	sink := &captureSink{}

	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.RedirectNode{
			Source: &sdk.CommandNode{
				Name: "@parallel",
				Args: map[string]any{
					"maxConcurrency": int64(2),
					"onFailure":      "wait_all",
				},
				Block: []sdk.Step{
					{
						ID: 2,
						Tree: &sdk.CommandNode{
							Name: "@test.chdir",
							Args: map[string]any{"dir": dirA},
							Block: []sdk.Step{{
								ID:   3,
								Tree: &sdk.CommandNode{Name: "@shell", Args: map[string]any{"command": "pwd"}},
							}},
						},
					},
					{
						ID:   4,
						Tree: &sdk.CommandNode{Name: "@shell", Args: map[string]any{"command": "pwd"}},
					},
				},
			},
			Sink: sink,
			Mode: sdk.RedirectOverwrite,
		},
	}}

	result, err := Execute(context.Background(), steps, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	want := dirA + "\n" + originalWd + "\n"
	if diff := cmp.Diff(want, sink.output.String()); diff != "" {
		t.Fatalf("parallel branch isolation mismatch (-want +got):\n%s", diff)
	}
}
