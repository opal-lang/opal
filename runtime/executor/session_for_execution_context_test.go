package executor

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestSessionForExecutionContextUsesBaseSnapshotWithoutSessionEnvCall(t *testing.T) {
	t.Parallel()

	baseSession := decorator.NewMonitoredSession(decorator.NewLocalSession())
	execCtx := &executionContext{
		ctx:         context.Background(),
		transportID: "local",
		environ:     map[string]string{"A": "1"},
		workdir:     "/tmp/base",
		baseEnviron: map[string]string{"A": "1"},
		baseWorkdir: "/tmp/base",
	}

	resolved := sessionForExecutionContext(baseSession, execCtx)

	if resolved != baseSession {
		t.Fatalf("session mismatch: expected original session pointer to be reused")
	}

	stats := baseSession.Stats()
	if diff := cmp.Diff(0, stats.EnvCalls); diff != "" {
		t.Fatalf("Env calls mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, stats.WithEnvCalls); diff != "" {
		t.Fatalf("WithEnv calls mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionForExecutionContextComputesDeltaFromBaseSnapshot(t *testing.T) {
	t.Parallel()

	baseSession := decorator.NewMonitoredSession(decorator.NewLocalSession())
	execCtx := &executionContext{
		ctx:         context.Background(),
		transportID: "local",
		environ: map[string]string{
			"A": "1",
			"B": "2",
		},
		workdir: "/tmp/base",
		baseEnviron: map[string]string{
			"A": "1",
		},
		baseWorkdir: "/tmp/base",
	}

	_ = sessionForExecutionContext(baseSession, execCtx)

	stats := baseSession.Stats()
	if diff := cmp.Diff(0, stats.EnvCalls); diff != "" {
		t.Fatalf("Env calls mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, stats.WithEnvCalls); diff != "" {
		t.Fatalf("WithEnv calls mismatch (-want +got):\n%s", diff)
	}

	wantDelta := map[string]string{"B": "2"}
	if len(stats.WithEnvDeltas) != 1 {
		t.Fatalf("WithEnv deltas length mismatch: got %d want 1", len(stats.WithEnvDeltas))
	}
	if diff := cmp.Diff(wantDelta, stats.WithEnvDeltas[0]); diff != "" {
		t.Fatalf("WithEnv delta mismatch (-want +got):\n%s", diff)
	}
}
