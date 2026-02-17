package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestShellWorkerCloseIsConcurrentSafe(t *testing.T) {
	t.Parallel()

	runtime := newSessionRuntime(nil)
	defer runtime.Close()

	pool := newShellWorkerPool(runtime)
	worker, err := pool.acquire("local", "bash")
	if err != nil {
		t.Fatalf("acquire worker: %v", err)
	}
	pool.release(worker)

	const closers = 24
	done := make(chan struct{})
	for i := 0; i < closers; i++ {
		go func() {
			worker.close()
			done <- struct{}{}
		}()
	}

	for i := 0; i < closers; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("concurrent close %d timed out", i+1)
		}
	}

	if diff := cmp.Diff(false, worker.isAlive()); diff != "" {
		t.Fatalf("worker alive state mismatch (-want +got):\n%s", diff)
	}
}

func TestShellWorkerRunReturnsWhenPoolCloses(t *testing.T) {
	t.Parallel()

	runtime := newSessionRuntime(nil)
	defer runtime.Close()

	pool := newShellWorkerPool(runtime)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	type runResult struct {
		exitCode int
		err      error
	}
	resultCh := make(chan runResult, 1)
	go func() {
		exitCode, err := pool.Run(ctx, shellRunRequest{
			transportID: "local",
			shellName:   "bash",
			command:     "sleep 5",
		})
		resultCh <- runResult{exitCode: exitCode, err: err}
	}()

	time.Sleep(50 * time.Millisecond)
	pool.Close()

	select {
	case result := <-resultCh:
		if result.exitCode == 0 {
			t.Fatalf("expected non-zero exit code after pool close")
		}
		if result.err == nil {
			return
		}
		if result.err != context.Canceled && result.err != context.DeadlineExceeded {
			if diff := cmp.Diff(decorator.ExitFailure, result.exitCode); diff != "" {
				t.Fatalf("unexpected exit code for worker failure (-want +got):\n%s", diff)
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("worker run did not return after pool close")
	}
}
