package executor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

func monitoredFactory() (sessionFactory, map[string]*decorator.SessionStats) {
	stats := map[string]*decorator.SessionStats{}
	var mu sync.Mutex

	factory := func(transportID string) (decorator.Session, error) {
		var base decorator.Session = decorator.NewLocalSession()
		if transportID != "local" {
			base = &transportScopedSession{id: transportID, session: base}
		}

		monitored := decorator.NewMonitoredSession(base)
		mu.Lock()
		stats[transportID] = monitored.Stats()
		mu.Unlock()
		return monitored, nil
	}

	return factory, stats
}

func TestExecuteClosesSessionsOnSuccess(t *testing.T) {
	t.Parallel()

	factory, stats := monitoredFactory()
	plan := &planfmt.Plan{Target: "session-close-success", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			planShell("echo local"),
			planShellOn("transport:A", "echo remote"),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: factory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, stats["local"].CloseCalls); diff != "" {
		t.Fatalf("local close calls mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, stats["transport:A"].CloseCalls); diff != "" {
		t.Fatalf("transport:A close calls mismatch (-want +got):\n%s", diff)
	}
}

func TestExecuteClosesSessionsOnFailure(t *testing.T) {
	t.Parallel()

	factory, stats := monitoredFactory()
	plan := &planfmt.Plan{Target: "session-close-failure", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			planShell("echo local"),
			planShellOn("transport:A", "exit 7"),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: factory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(7, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, stats["local"].CloseCalls); diff != "" {
		t.Fatalf("local close calls mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, stats["transport:A"].CloseCalls); diff != "" {
		t.Fatalf("transport:A close calls mismatch (-want +got):\n%s", diff)
	}
}

func TestExecuteClosesSessionsOnCancellation(t *testing.T) {
	t.Parallel()

	factory, stats := monitoredFactory()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	plan := &planfmt.Plan{Target: "session-close-cancel", Steps: []planfmt.Step{{
		ID:   1,
		Tree: planShellOn("transport:A", "sleep 5"),
	}}}

	result, err := ExecutePlan(ctx, plan, Config{sessionFactory: factory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(true, result.ExitCode != 0); diff != "" {
		t.Fatalf("expected non-zero exit code on cancellation (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, stats["transport:A"].CloseCalls); diff != "" {
		t.Fatalf("transport:A close calls mismatch (-want +got):\n%s", diff)
	}
}

func planShell(command string) *planfmt.CommandNode {
	return planShellOn("", command)
}

func planShellOn(transportID, command string) *planfmt.CommandNode {
	return &planfmt.CommandNode{
		Decorator:   "@shell",
		TransportID: transportID,
		Args: []planfmt.Arg{{
			Key: "command",
			Val: planfmt.Value{Kind: planfmt.ValueString, Str: command},
		}},
	}
}
