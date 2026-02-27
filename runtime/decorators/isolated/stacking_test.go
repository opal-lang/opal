package isolated

import (
	"context"
	"fmt"
	"io/fs"
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/google/go-cmp/cmp"
)

func TestStackingNestedSessionIDPath(t *testing.T) {
	base := &recordingSession{id: "parent"}

	networkSession := &networkLoopbackSession{
		parent:   base,
		isolator: (&NetworkLoopbackDecorator{}).IsolationContext(),
	}
	memorySession := &memoryLockSession{
		parent:   networkSession,
		isolator: (&MemoryLockDecorator{}).IsolationContext(),
	}

	if diff := cmp.Diff("parent/isolated.network.loopback/isolated.memory.lock", memorySession.ID()); diff != "" {
		t.Fatalf("stacked session id mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(decorator.TransportScopeIsolated, memorySession.TransportScope()); diff != "" {
		t.Fatalf("transport scope mismatch (-want +got):\n%s", diff)
	}
}

func TestStackingChainsIsolationContextAndExecution(t *testing.T) {
	base := &recordingSession{id: "parent", hasResult: true, runResult: decorator.Result{ExitCode: 0, Stdout: []byte("stacked-ok")}}

	networkDecorator := &NetworkLoopbackDecorator{}
	memoryDecorator := &MemoryLockDecorator{}

	networkSession := &networkLoopbackSession{parent: base, isolator: networkDecorator.IsolationContext()}
	memorySession := &memoryLockSession{parent: networkSession, isolator: memoryDecorator.IsolationContext()}

	parentSession, ok := memorySession.parent.(*networkLoopbackSession)
	if !ok {
		t.Fatalf("expected memory parent to be networkLoopbackSession, got %T", memorySession.parent)
	}

	if memorySession.isolator == nil {
		t.Fatal("expected memory session isolator to be non-nil")
	}
	if parentSession.isolator == nil {
		t.Fatal("expected network session isolator to be non-nil")
	}
	if memorySession.isolator == parentSession.isolator {
		t.Fatal("expected stacked sessions to have distinct isolation contexts")
	}

	if diff := cmp.Diff(fmt.Sprintf("%T", memoryDecorator.IsolationContext()), fmt.Sprintf("%T", memorySession.isolator)); diff != "" {
		t.Fatalf("memory isolation context type mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(fmt.Sprintf("%T", networkDecorator.IsolationContext()), fmt.Sprintf("%T", parentSession.isolator)); diff != "" {
		t.Fatalf("network isolation context type mismatch (-want +got):\n%s", diff)
	}

	result, err := memorySession.Run(context.Background(), []string{"echo", "stacked"}, decorator.RunOpts{})
	if err != nil {
		t.Fatalf("run through stacked sessions: %v", err)
	}

	if diff := cmp.Diff(1, base.runCalls); diff != "" {
		t.Fatalf("run call count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"echo", "stacked"}, base.lastArgv); diff != "" {
		t.Fatalf("run argv mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("stacked-ok", string(result.Stdout)); diff != "" {
		t.Fatalf("run stdout mismatch (-want +got):\n%s", diff)
	}
}

type recordingSession struct {
	id        string
	runCalls  int
	lastArgv  []string
	hasResult bool
	runResult decorator.Result
}

func (s *recordingSession) Run(_ context.Context, argv []string, _ decorator.RunOpts) (decorator.Result, error) {
	s.runCalls++
	s.lastArgv = append([]string(nil), argv...)
	if !s.hasResult {
		return decorator.Result{ExitCode: 0}, nil
	}
	return s.runResult, nil
}

func (s *recordingSession) Put(context.Context, []byte, string, fs.FileMode) error {
	return nil
}

func (s *recordingSession) Get(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (s *recordingSession) Env() map[string]string {
	return map[string]string{}
}

func (s *recordingSession) WithEnv(map[string]string) decorator.Session {
	return s
}

func (s *recordingSession) WithWorkdir(string) decorator.Session {
	return s
}

func (s *recordingSession) Cwd() string {
	return ""
}

func (s *recordingSession) ID() string {
	return s.id
}

func (s *recordingSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeLocal
}

func (s *recordingSession) Platform() string {
	return "linux"
}

func (s *recordingSession) Close() error {
	return nil
}
