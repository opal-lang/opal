package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

func TestShellWorkerReusesPerTransportAndShell(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localA := filepath.Join(tmpDir, "local-a.txt")
	localB := filepath.Join(tmpDir, "local-b.txt")
	transportA1 := filepath.Join(tmpDir, "transport-a-1.txt")
	transportA2 := filepath.Join(tmpDir, "transport-a-2.txt")
	transportB := filepath.Join(tmpDir, "transport-b.txt")

	plan := &planfmt.Plan{Target: "worker-reuse", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			shellPlanCommand("echo \"$" + workerInstanceEnvVar + "\" > " + shellLiteral(localA)),
			shellPlanCommand("echo \"$" + workerInstanceEnvVar + "\" > " + shellLiteral(localB)),
			shellPlanCommandOn("transport:A", "echo \"$"+workerInstanceEnvVar+"\" > "+shellLiteral(transportA1)),
			shellPlanCommandOn("transport:A", "echo \"$"+workerInstanceEnvVar+"\" > "+shellLiteral(transportA2)),
			shellPlanCommandOn("transport:B", "echo \"$"+workerInstanceEnvVar+"\" > "+shellLiteral(transportB)),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	localAID := readTrimmedFile(t, localA)
	localBID := readTrimmedFile(t, localB)
	transportAID1 := readTrimmedFile(t, transportA1)
	transportAID2 := readTrimmedFile(t, transportA2)
	transportBID := readTrimmedFile(t, transportB)

	if diff := cmp.Diff(localAID, localBID); diff != "" {
		t.Fatalf("local worker ID mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(transportAID1, transportAID2); diff != "" {
		t.Fatalf("transport:A worker ID mismatch (-want +got):\n%s", diff)
	}

	if localAID == "" || transportAID1 == "" || transportBID == "" {
		t.Fatalf("worker IDs must be non-empty: local=%q transportA=%q transportB=%q", localAID, transportAID1, transportBID)
	}
	if diff := cmp.Diff(false, localAID == transportAID1); diff != "" {
		t.Fatalf("expected different worker IDs for local and transport:A (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, transportAID1 == transportBID); diff != "" {
		t.Fatalf("expected different worker IDs for transport:A and transport:B (-want +got):\n%s", diff)
	}
}

func TestShellWorkerSubshellIsolation(t *testing.T) {
	t.Parallel()

	plan := &planfmt.Plan{Target: "worker-env-isolation", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			shellPlanCommand("export OPAL_WORKER_LEAK_TEST=leaked"),
			shellPlanCommand("if [ -n \"$OPAL_WORKER_LEAK_TEST\" ]; then exit 41; fi"),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestShellWorkerSubshellIsolationForWorkdir(t *testing.T) {
	t.Parallel()

	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	firstPwd := filepath.Join(t.TempDir(), "first_pwd.txt")
	secondPwd := filepath.Join(t.TempDir(), "second_pwd.txt")

	plan := &planfmt.Plan{Target: "worker-workdir-isolation", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.SequenceNode{Nodes: []planfmt.ExecutionNode{
			shellPlanCommand("cd " + shellLiteral(tmpDir) + "; pwd > " + shellLiteral(firstPwd)),
			shellPlanCommand("pwd > " + shellLiteral(secondPwd)),
		}},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(tmpDir, readTrimmedFile(t, firstPwd)); diff != "" {
		t.Fatalf("first command cwd mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(originalCwd, readTrimmedFile(t, secondPwd)); diff != "" {
		t.Fatalf("second command cwd mismatch (-want +got):\n%s", diff)
	}
}

func TestShellWorkerStreamsStdoutBeforeCommandExit(t *testing.T) {
	t.Parallel()

	runtime := newSessionRuntime(nil)
	defer runtime.Close()

	pool := newShellWorkerPool(runtime)
	defer pool.Close()

	writer := newStreamingProbeWriter("first\n")
	type runResult struct {
		exitCode int
		err      error
	}
	resultCh := make(chan runResult, 1)

	go func() {
		exitCode, err := pool.Run(context.Background(), shellRunRequest{
			transportID: "local",
			shellName:   "bash",
			command:     "printf 'first\\n'; sleep 1; printf 'second\\n'",
			stdout:      writer,
		})
		resultCh <- runResult{exitCode: exitCode, err: err}
	}()

	select {
	case <-writer.Trigger():
	case <-time.After(700 * time.Millisecond):
		t.Fatal("timed out waiting for first streamed stdout chunk")
	}

	select {
	case <-resultCh:
		t.Fatal("worker run finished before first chunk was observed")
	default:
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("worker run failed: %v", result.err)
		}
		if diff := cmp.Diff(0, result.exitCode); diff != "" {
			t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker run completion")
	}

	if diff := cmp.Diff("first\nsecond\n", writer.String()); diff != "" {
		t.Fatalf("streamed stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestShellWorkerStreamsStderrBeforeCommandExit(t *testing.T) {
	t.Parallel()

	runtime := newSessionRuntime(nil)
	defer runtime.Close()

	pool := newShellWorkerPool(runtime)
	defer pool.Close()

	writer := newStreamingProbeWriter("err-first\n")
	type runResult struct {
		exitCode int
		err      error
	}
	resultCh := make(chan runResult, 1)

	go func() {
		exitCode, err := pool.Run(context.Background(), shellRunRequest{
			transportID: "local",
			shellName:   "bash",
			command:     "printf 'err-first\\n' >&2; sleep 1; printf 'err-second\\n' >&2",
			stderr:      writer,
		})
		resultCh <- runResult{exitCode: exitCode, err: err}
	}()

	select {
	case <-writer.Trigger():
	case <-time.After(700 * time.Millisecond):
		t.Fatal("timed out waiting for first streamed stderr chunk")
	}

	select {
	case <-resultCh:
		t.Fatal("worker run finished before first stderr chunk was observed")
	default:
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("worker run failed: %v", result.err)
		}
		if diff := cmp.Diff(0, result.exitCode); diff != "" {
			t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker run completion")
	}

	if diff := cmp.Diff("err-first\nerr-second\n", writer.String()); diff != "" {
		t.Fatalf("streamed stderr mismatch (-want +got):\n%s", diff)
	}
}

type streamingProbeWriter struct {
	mu        sync.Mutex
	triggerOn string
	triggered bool
	buf       strings.Builder
	trigger   chan struct{}
}

func newStreamingProbeWriter(triggerOn string) *streamingProbeWriter {
	return &streamingProbeWriter{
		triggerOn: triggerOn,
		trigger:   make(chan struct{}),
	}
}

func (w *streamingProbeWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.buf.Write(p); err != nil {
		return 0, err
	}

	if !w.triggered && strings.Contains(w.buf.String(), w.triggerOn) {
		w.triggered = true
		close(w.trigger)
	}

	return len(p), nil
}

func (w *streamingProbeWriter) Trigger() <-chan struct{} {
	return w.trigger
}

func (w *streamingProbeWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func readTrimmedFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

func shellLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func shellPlanCommand(command string) *planfmt.CommandNode {
	return shellPlanCommandOn("", command)
}

func shellPlanCommandOn(transportID, command string) *planfmt.CommandNode {
	return &planfmt.CommandNode{
		Decorator:   "@shell",
		TransportID: transportID,
		Args: []planfmt.Arg{{
			Key: "command",
			Val: planfmt.Value{Kind: planfmt.ValueString, Str: command},
		}},
	}
}
