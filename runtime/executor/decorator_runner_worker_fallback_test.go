package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

type alwaysFailWriter struct{}

func (w *alwaysFailWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("forced write failure")
}

func TestExecuteShellWithParams_NoFallbackAfterWorkerStarted(t *testing.T) {
	t.Parallel()

	e := &executor{sessions: newSessionRuntime(nil)}
	e.workers = newShellWorkerPool(e.sessions)
	defer e.workers.Close()
	defer e.sessions.Close()

	ctx := newExecutionContext(map[string]interface{}{}, e, context.Background())

	outputFile := filepath.Join(t.TempDir(), "side-effect.txt")
	command := "printf 'once\\n' >> " + shellQuote(outputFile) + "; echo trigger"

	exitCode := e.executeShellWithParams(ctx, map[string]any{"command": command}, nil, &alwaysFailWriter{})
	if diff := cmp.Diff(decorator.ExitFailure, exitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read side-effect file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if diff := cmp.Diff(1, len(lines)); diff != "" {
		t.Fatalf("worker fallback re-executed command (-want +got):\n%s\ncontent=%q", diff, string(content))
	}
}

func TestExecuteShellWithParams_FallbackWhenWorkerNeverStartedCommand(t *testing.T) {
	t.Parallel()

	e := &executor{sessions: newSessionRuntime(nil)}
	failingWorkerSessions := newSessionRuntime(func(transportID string) (decorator.Session, error) {
		return nil, fmt.Errorf("worker session unavailable")
	})
	e.workers = newShellWorkerPool(failingWorkerSessions)
	defer e.workers.Close()
	defer failingWorkerSessions.Close()
	defer e.sessions.Close()

	ctx := newExecutionContext(map[string]interface{}{}, e, context.Background())

	outputFile := filepath.Join(t.TempDir(), "fallback.txt")
	command := "printf 'once\\n' >> " + shellQuote(outputFile)

	exitCode := e.executeShellWithParams(ctx, map[string]any{"command": command}, nil, nil)
	if diff := cmp.Diff(0, exitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read fallback output: %v", err)
	}
	if diff := cmp.Diff("once\n", string(content)); diff != "" {
		t.Fatalf("fallback output mismatch (-want +got):\n%s", diff)
	}
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}
