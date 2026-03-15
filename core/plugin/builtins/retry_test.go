package builtins

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

type fakeExecContext struct {
	ctx context.Context

	session plugin.ParentTransport
}

func (f fakeExecContext) Context() context.Context        { return f.ctx }
func (f fakeExecContext) Session() plugin.ParentTransport { return f.session }
func (f fakeExecContext) Stdin() io.Reader                { return nil }
func (f fakeExecContext) Stdout() io.Writer               { return nil }
func (f fakeExecContext) Stderr() io.Writer               { return nil }

type fakeExecNode struct {
	execute func(ctx plugin.ExecContext) (plugin.Result, error)
}

func (n fakeExecNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	return n.execute(ctx)
}

func TestRetryWrapperSucceedsAfterFailures(t *testing.T) {
	capability := RetryWrapperCapability{}
	var attempts atomic.Int64

	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		attempt := attempts.Add(1)
		if attempt < 3 {
			return plugin.Result{ExitCode: 2}, nil
		}
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}}, fakeArgs{ints: map[string]int{"times": 3}, durations: map[string]time.Duration{"delay": time.Millisecond}, strings: map[string]string{"backoff": "constant"}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background()})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if diff := cmp.Diff(plugin.ExitSuccess, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(3), attempts.Load()); diff != "" {
		t.Fatalf("Execute() attempts mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryWrapperStopsAtMaxAttempts(t *testing.T) {
	capability := RetryWrapperCapability{}
	var attempts atomic.Int64

	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		attempts.Add(1)
		return plugin.Result{ExitCode: 9}, nil
	}}, fakeArgs{ints: map[string]int{"times": 2}, durations: map[string]time.Duration{"delay": time.Millisecond}, strings: map[string]string{"backoff": "constant"}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background()})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if diff := cmp.Diff(9, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(2), attempts.Load()); diff != "" {
		t.Fatalf("Execute() attempts mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryWrapperRespectsCancellation(t *testing.T) {
	capability := RetryWrapperCapability{}
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	var attempts atomic.Int64
	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		attempts.Add(1)
		return plugin.Result{ExitCode: 1}, errors.New("should not execute")
	}}, fakeArgs{ints: map[string]int{"times": 3}, durations: map[string]time.Duration{"delay": time.Millisecond}, strings: map[string]string{"backoff": "constant"}})

	result, err := node.Execute(fakeExecContext{ctx: cancelledCtx})
	if err == nil {
		t.Fatal("Execute() error = nil, want cancellation error")
	}
	if diff := cmp.Diff(plugin.ExitCanceled, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(0), attempts.Load()); diff != "" {
		t.Fatalf("Execute() attempts mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryWrapperCancellationDuringDelay(t *testing.T) {
	capability := RetryWrapperCapability{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts atomic.Int64
	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		attempts.Add(1)
		cancel()
		return plugin.Result{ExitCode: 1}, errors.New("boom")
	}}, fakeArgs{ints: map[string]int{"times": 3}, durations: map[string]time.Duration{"delay": 50 * time.Millisecond}, strings: map[string]string{"backoff": "constant"}})

	result, err := node.Execute(fakeExecContext{ctx: ctx})
	if err == nil {
		t.Fatal("Execute() error = nil, want cancellation error")
	}
	if diff := cmp.Diff(plugin.ExitCanceled, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(1), attempts.Load()); diff != "" {
		t.Fatalf("Execute() attempts mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryDelayExponential(t *testing.T) {
	if diff := cmp.Diff(5*time.Millisecond, retryDelay(5*time.Millisecond, "constant", 3)); diff != "" {
		t.Fatalf("retryDelay(constant) mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(20*time.Millisecond, retryDelay(5*time.Millisecond, "exponential", 3)); diff != "" {
		t.Fatalf("retryDelay(exponential) mismatch (-want +got):\n%s", diff)
	}
}

func TestRetryWrapperWithoutBlockIsNoOp(t *testing.T) {
	capability := RetryWrapperCapability{}
	node := capability.Wrap(nil, fakeArgs{ints: map[string]int{"times": 3}, strings: map[string]string{"backoff": "constant"}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background()})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if diff := cmp.Diff(plugin.ExitSuccess, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit mismatch (-want +got):\n%s", diff)
	}
}
