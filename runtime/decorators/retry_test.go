package decorators

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

type testExecNode struct {
	execute func(ctx decorator.ExecContext) (decorator.Result, error)
}

func (n *testExecNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	return n.execute(ctx)
}

func TestRetrySucceedsAfterFailures(t *testing.T) {
	dec := &RetryDecorator{}
	var attempts atomic.Int64

	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		attempt := attempts.Add(1)
		if attempt < 3 {
			return decorator.Result{ExitCode: 2}, nil
		}
		return decorator.Result{ExitCode: 0}, nil
	}}, map[string]any{"times": int64(3), "delay": "1ms", "backoff": "constant"})

	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err != nil {
		t.Fatalf("retry execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(3), attempts.Load()); diff != "" {
		t.Fatalf("attempt count mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryStopsAtMaxAttempts(t *testing.T) {
	dec := &RetryDecorator{}
	var attempts atomic.Int64

	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		attempts.Add(1)
		return decorator.Result{ExitCode: 9}, nil
	}}, map[string]any{"times": int64(2), "delay": "1ms", "backoff": "constant"})

	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err != nil {
		t.Fatalf("retry execute failed: %v", err)
	}
	if diff := cmp.Diff(9, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(2), attempts.Load()); diff != "" {
		t.Fatalf("attempt count mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryRespectsCancellation(t *testing.T) {
	dec := &RetryDecorator{}
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	var attempts atomic.Int64
	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		attempts.Add(1)
		return decorator.Result{ExitCode: 1}, nil
	}}, map[string]any{"times": int64(3), "delay": "1ms"})

	result, err := node.Execute(decorator.ExecContext{Context: cancelledCtx})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if diff := cmp.Diff(decorator.ExitCanceled, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(0), attempts.Load()); diff != "" {
		t.Fatalf("attempt count mismatch (-want +got):\n%s", diff)
	}
}
