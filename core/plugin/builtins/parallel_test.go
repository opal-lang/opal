package builtins

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

func TestParallelWrapperCapabilityFallsBackToSequentialWhenNoBranches(t *testing.T) {
	capability := ParallelWrapperCapability{}
	var calls atomic.Int64
	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		calls.Add(1)
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}}, fakeArgs{ints: map[string]int{"maxConcurrency": 4}, strings: map[string]string{"onFailure": "wait_all"}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background()})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if diff := cmp.Diff(plugin.ExitSuccess, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(1), calls.Load()); diff != "" {
		t.Fatalf("Execute() calls mismatch (-want +got):\n%s", diff)
	}
}

func TestParallelWrapperCapabilityHonorsFailFastCancellation(t *testing.T) {
	capability := ParallelWrapperCapability{}
	node := capability.Wrap(fakeBranchExecNode{
		branches: []func(ctx plugin.ExecContext) (plugin.Result, error){
			func(ctx plugin.ExecContext) (plugin.Result, error) {
				return plugin.Result{ExitCode: 1}, nil
			},
			func(ctx plugin.ExecContext) (plugin.Result, error) {
				select {
				case <-ctx.Context().Done():
					return plugin.Result{ExitCode: plugin.ExitCanceled}, ctx.Context().Err()
				case <-time.After(200 * time.Millisecond):
					return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
				}
			},
		},
	}, fakeArgs{ints: map[string]int{"maxConcurrency": 2}, strings: map[string]string{"onFailure": "fail_fast"}})

	result, _ := node.Execute(fakeExecContext{ctx: context.Background()})
	if diff := cmp.Diff(1, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestParallelWrapperCapabilityConvertsBranchPanicToError(t *testing.T) {
	capability := ParallelWrapperCapability{}
	node := capability.Wrap(fakeBranchExecNode{
		branches: []func(ctx plugin.ExecContext) (plugin.Result, error){
			func(ctx plugin.ExecContext) (plugin.Result, error) {
				panic("boom")
			},
		},
	}, fakeArgs{ints: map[string]int{"maxConcurrency": 1}, strings: map[string]string{"onFailure": "wait_all"}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background()})
	if err == nil {
		t.Fatal("Execute() error = nil, want panic error")
	}
	if diff := cmp.Diff(plugin.ExitFailure, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("branch 0 panicked: boom", err.Error()); diff != "" {
		t.Fatalf("Execute() error mismatch (-want +got):\n%s", diff)
	}
}

type fakeBranchExecNode struct {
	branches []func(ctx plugin.ExecContext) (plugin.Result, error)
}

func (n fakeBranchExecNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	if len(n.branches) == 0 {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}
	return n.branches[0](ctx)
}

func (n fakeBranchExecNode) BranchCount() int { return len(n.branches) }

func (n fakeBranchExecNode) ExecuteBranch(index int, ctx plugin.ExecContext) (plugin.Result, error) {
	return n.branches[index](ctx)
}
