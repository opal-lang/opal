package decorators

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

type testBranchNode struct {
	branches []func(ctx decorator.ExecContext) (decorator.Result, error)
}

func (n *testBranchNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if len(n.branches) == 0 {
		return decorator.Result{ExitCode: 0}, nil
	}
	return n.branches[0](ctx)
}

func (n *testBranchNode) BranchCount() int {
	return len(n.branches)
}

func (n *testBranchNode) ExecuteBranch(index int, ctx decorator.ExecContext) (decorator.Result, error) {
	return n.branches[index](ctx)
}

func TestParallelDeterministicOutputOrder(t *testing.T) {
	dec := &ParallelDecorator{}
	next := &testBranchNode{branches: []func(ctx decorator.ExecContext) (decorator.Result, error){
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			time.Sleep(30 * time.Millisecond)
			_, _ = ctx.Stdout.Write([]byte("A\n"))
			return decorator.Result{ExitCode: 0}, nil
		},
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			time.Sleep(5 * time.Millisecond)
			_, _ = ctx.Stdout.Write([]byte("B\n"))
			return decorator.Result{ExitCode: 0}, nil
		},
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			_, _ = ctx.Stdout.Write([]byte("C\n"))
			return decorator.Result{ExitCode: 0}, nil
		},
	}}

	node := dec.Wrap(next, map[string]any{"onFailure": "wait_all"})
	var stdout bytes.Buffer

	result, err := node.Execute(decorator.ExecContext{Context: context.Background(), Stdout: &stdout})
	if err != nil {
		t.Fatalf("parallel execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("A\nB\nC\n", stdout.String()); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestParallelDefaultFailFastCancelsSiblings(t *testing.T) {
	dec := &ParallelDecorator{}
	var canceled atomic.Bool

	next := &testBranchNode{branches: []func(ctx decorator.ExecContext) (decorator.Result, error){
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			return decorator.Result{ExitCode: 7}, nil
		},
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			select {
			case <-ctx.Context.Done():
				canceled.Store(true)
				return decorator.Result{ExitCode: decorator.ExitCanceled}, ctx.Context.Err()
			case <-time.After(2 * time.Second):
				return decorator.Result{ExitCode: 0}, nil
			}
		},
	}}

	node := dec.Wrap(next, map[string]any{"maxConcurrency": int64(2)})
	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err != nil {
		t.Fatalf("parallel should return branch failure without top-level error: %v", err)
	}
	if diff := cmp.Diff(7, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if !canceled.Load() {
		t.Fatal("expected sibling branch to be cancelled under fail_fast mode")
	}
}

func TestParallelWaitAllRunsAllBranches(t *testing.T) {
	dec := &ParallelDecorator{}
	var secondRan atomic.Bool

	next := &testBranchNode{branches: []func(ctx decorator.ExecContext) (decorator.Result, error){
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			_, _ = ctx.Stdout.Write([]byte("first\n"))
			return decorator.Result{ExitCode: 5}, nil
		},
		func(ctx decorator.ExecContext) (decorator.Result, error) {
			secondRan.Store(true)
			_, _ = ctx.Stdout.Write([]byte("second\n"))
			return decorator.Result{ExitCode: 0}, nil
		},
	}}

	node := dec.Wrap(next, map[string]any{"onFailure": "wait_all"})
	var stdout bytes.Buffer

	result, err := node.Execute(decorator.ExecContext{Context: context.Background(), Stdout: &stdout})
	if err != nil {
		t.Fatalf("parallel wait_all should not return top-level error: %v", err)
	}
	if diff := cmp.Diff(5, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if !secondRan.Load() {
		t.Fatal("expected second branch to run under wait_all mode")
	}
	if diff := cmp.Diff("first\nsecond\n", stdout.String()); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestParallelHonorsMaxConcurrency(t *testing.T) {
	dec := &ParallelDecorator{}
	var active atomic.Int64
	var maxSeen atomic.Int64

	branches := make([]func(ctx decorator.ExecContext) (decorator.Result, error), 6)
	for i := range branches {
		branches[i] = func(ctx decorator.ExecContext) (decorator.Result, error) {
			current := active.Add(1)
			for {
				seen := maxSeen.Load()
				if current <= seen || maxSeen.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			active.Add(-1)
			return decorator.Result{ExitCode: 0}, nil
		}
	}

	node := dec.Wrap(&testBranchNode{branches: branches}, map[string]any{
		"maxConcurrency": int64(2),
		"onFailure":      "wait_all",
	})

	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err != nil {
		t.Fatalf("parallel execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(2), maxSeen.Load()); diff != "" {
		t.Fatalf("max concurrency mismatch (-want +got):\n%s", diff)
	}
}
