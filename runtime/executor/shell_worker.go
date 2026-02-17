package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
)

const workerInstanceEnvVar = "OPAL_INTERNAL_WORKER_INSTANCE"

var shellWorkerSequence atomic.Uint64

type shellWorkerKey struct {
	transportID string
	shellName   string
}

type shellRunRequest struct {
	transportID string
	shellName   string
	command     string
	environ     map[string]string
	workdir     string
	stdout      io.Writer
	stderr      io.Writer
}

type shellWorkerPool struct {
	sessions *sessionRuntime

	mu      sync.Mutex
	workers map[shellWorkerKey][]*shellWorker
}

func newShellWorkerPool(sessions *sessionRuntime) *shellWorkerPool {
	return &shellWorkerPool{
		sessions: sessions,
		workers:  make(map[shellWorkerKey][]*shellWorker),
	}
}

func (p *shellWorkerPool) Run(ctx context.Context, req shellRunRequest) (int, error) {
	invariant.NotNil(p, "shell worker pool")
	invariant.Precondition(req.shellName != "", "shell worker request missing shell name")
	invariant.Precondition(req.command != "", "shell worker request missing command")

	worker, err := p.acquire(req.transportID, req.shellName)
	if err != nil {
		return decorator.ExitFailure, err
	}
	defer p.release(worker)

	return worker.run(ctx, req)
}

func (p *shellWorkerPool) Close() {
	p.mu.Lock()
	workers := make([]*shellWorker, 0)
	for _, workerList := range p.workers {
		workers = append(workers, workerList...)
	}
	p.workers = make(map[shellWorkerKey][]*shellWorker)
	p.mu.Unlock()

	for _, worker := range workers {
		worker.close()
	}
}

func (p *shellWorkerPool) acquire(transportID, shellName string) (*shellWorker, error) {
	key := shellWorkerKey{transportID: normalizedTransportID(transportID), shellName: shellName}

	p.mu.Lock()
	for _, worker := range p.workers[key] {
		if worker.isAlive() && !worker.busy {
			worker.busy = true
			p.mu.Unlock()
			return worker, nil
		}
	}
	p.mu.Unlock()

	worker, err := p.newWorker(key)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	worker.busy = true
	p.workers[key] = append(p.workers[key], worker)
	p.mu.Unlock()

	return worker, nil
}

func (p *shellWorkerPool) release(worker *shellWorker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	worker.busy = false
	if worker.isAlive() {
		return
	}

	key := worker.key
	current := p.workers[key]
	if len(current) == 0 {
		return
	}

	filtered := make([]*shellWorker, 0, len(current)-1)
	for _, candidate := range current {
		if candidate != worker {
			filtered = append(filtered, candidate)
		}
	}
	if len(filtered) == 0 {
		delete(p.workers, key)
		return
	}
	p.workers[key] = filtered
}

func (p *shellWorkerPool) newWorker(key shellWorkerKey) (*shellWorker, error) {
	session, err := p.sessions.SessionFor(key.transportID)
	if err != nil {
		return nil, fmt.Errorf("session for worker %s/%s: %w", key.transportID, key.shellName, err)
	}

	worker := &shellWorker{
		key:      key,
		session:  session,
		instance: strconv.FormatUint(shellWorkerSequence.Add(1), 10),
	}
	worker.alive.Store(true)

	if err := worker.start(); err != nil {
		return nil, err
	}

	return worker, nil
}

type shellWorker struct {
	key      shellWorkerKey
	session  decorator.Session
	instance string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	busy  bool
	alive atomic.Bool

	closeOnce sync.Once

	mu sync.Mutex
}

func (w *shellWorker) start() error {
	var cmd *exec.Cmd
	switch w.key.shellName {
	case "bash":
		cmd = exec.Command("bash", "--noprofile", "--norc")
	case "pwsh", "cmd":
		return fmt.Errorf("shell workers do not support %q", w.key.shellName)
	default:
		return fmt.Errorf("unsupported shell %q", w.key.shellName)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("worker stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("worker stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr
	cmd.Env = toEnvironList(w.session.Env())
	cmd.Dir = w.session.Cwd()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start worker shell: %w", err)
	}

	w.cmd = cmd
	w.stdin = stdin
	w.stdout = bufio.NewReader(stdoutPipe)

	readyMarker := strconv.FormatUint(shellWorkerSequence.Add(1), 10)
	bootstrap := fmt.Sprintf("export %s=%s\nprintf '__OPAL_WORKER_READY_%s__\\n'\n", workerInstanceEnvVar, quoteShellLiteral(w.instance), readyMarker)
	if _, err := io.WriteString(w.stdin, bootstrap); err != nil {
		w.close()
		return fmt.Errorf("bootstrap worker: %w", err)
	}

	for {
		line, err := w.stdout.ReadString('\n')
		if err != nil {
			w.close()
			return fmt.Errorf("read bootstrap marker: %w", err)
		}
		if strings.TrimSpace(line) == "__OPAL_WORKER_READY_"+readyMarker+"__" {
			break
		}
	}

	return nil
}

func (w *shellWorker) run(ctx context.Context, req shellRunRequest) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isAlive() {
		return decorator.ExitFailure, fmt.Errorf("worker %s/%s is closed", w.key.transportID, w.key.shellName)
	}

	stdoutPath, stderrPath, cleanup, err := createWorkerCaptureFiles()
	if err != nil {
		return decorator.ExitFailure, err
	}
	defer cleanup()

	statusMarker := strconv.FormatUint(shellWorkerSequence.Add(1), 10)
	runReq := req
	runReq.environ = envDelta(w.session.Env(), req.environ)
	if req.workdir == w.session.Cwd() {
		runReq.workdir = ""
	}

	script := buildWorkerScript(runReq, stdoutPath, stderrPath, statusMarker)
	if _, err := io.WriteString(w.stdin, script); err != nil {
		w.close()
		return decorator.ExitFailure, fmt.Errorf("write worker request: %w", err)
	}

	type workerResult struct {
		exitCode int
		err      error
	}

	resultCh := make(chan workerResult, 1)
	go func() {
		exitCode, readErr := w.readStatus(statusMarker)
		resultCh <- workerResult{exitCode: exitCode, err: readErr}
	}()

	select {
	case <-ctx.Done():
		w.close()
		return decorator.ExitCanceled, ctx.Err()
	case result := <-resultCh:
		if result.err != nil {
			w.close()
			return decorator.ExitFailure, result.err
		}

		if err := replayWorkerOutput(stdoutPath, req.stdout, os.Stdout); err != nil {
			return decorator.ExitFailure, err
		}
		if err := replayWorkerOutput(stderrPath, req.stderr, os.Stderr); err != nil {
			return decorator.ExitFailure, err
		}

		return result.exitCode, nil
	}
}

func (w *shellWorker) readStatus(marker string) (int, error) {
	statusPrefix := "__OPAL_STATUS_" + marker + ":"
	for {
		line, err := w.stdout.ReadString('\n')
		if err != nil {
			return decorator.ExitFailure, fmt.Errorf("read worker status: %w", err)
		}

		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, statusPrefix) {
			continue
		}

		codeStr := strings.TrimPrefix(trimmed, statusPrefix)
		exitCode, parseErr := strconv.Atoi(codeStr)
		if parseErr != nil {
			return decorator.ExitFailure, fmt.Errorf("parse worker exit code %q: %w", codeStr, parseErr)
		}

		return exitCode, nil
	}
}

func (w *shellWorker) close() {
	w.closeOnce.Do(func() {
		w.alive.Store(false)

		if w.stdin != nil {
			_ = w.stdin.Close()
		}
		if w.cmd != nil && w.cmd.Process != nil {
			_ = w.cmd.Process.Kill()
		}
		if w.cmd != nil {
			_ = w.cmd.Wait()
		}
	})
}

func (w *shellWorker) isAlive() bool {
	return w.alive.Load()
}

func buildWorkerScript(req shellRunRequest, stdoutPath, stderrPath, statusMarker string) string {
	invariant.Precondition(req.command != "", "worker command cannot be empty")

	var b strings.Builder
	b.WriteString("(\n")

	if req.workdir != "" {
		b.WriteString("cd -- ")
		b.WriteString(quoteShellLiteral(req.workdir))
		b.WriteString(" || exit 1\n")
	}

	keys := sortedEnvKeys(req.environ)
	for _, key := range keys {
		if !isValidEnvName(key) {
			continue
		}
		b.WriteString("export ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(quoteShellLiteral(req.environ[key]))
		b.WriteString("\n")
	}

	b.WriteString(req.command)
	if !strings.HasSuffix(req.command, "\n") {
		b.WriteString("\n")
	}

	b.WriteString(") >")
	b.WriteString(quoteShellLiteral(stdoutPath))
	b.WriteString(" 2>")
	b.WriteString(quoteShellLiteral(stderrPath))
	b.WriteString("\n")
	b.WriteString("__opal_status=$?\n")
	b.WriteString("printf '__OPAL_STATUS_")
	b.WriteString(statusMarker)
	b.WriteString(":%d\\n' \"$__opal_status\"\n")

	return b.String()
}

func createWorkerCaptureFiles() (stdoutPath, stderrPath string, cleanup func(), err error) {
	stdoutFile, err := os.CreateTemp("", "opal-worker-stdout-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("create worker stdout file: %w", err)
	}
	stdoutPath = stdoutFile.Name()
	if closeErr := stdoutFile.Close(); closeErr != nil {
		_ = os.Remove(stdoutPath)
		return "", "", nil, fmt.Errorf("close worker stdout file: %w", closeErr)
	}

	stderrFile, err := os.CreateTemp("", "opal-worker-stderr-*")
	if err != nil {
		_ = os.Remove(stdoutPath)
		return "", "", nil, fmt.Errorf("create worker stderr file: %w", err)
	}
	stderrPath = stderrFile.Name()
	if closeErr := stderrFile.Close(); closeErr != nil {
		_ = os.Remove(stdoutPath)
		_ = os.Remove(stderrPath)
		return "", "", nil, fmt.Errorf("close worker stderr file: %w", closeErr)
	}

	cleanup = func() {
		_ = os.Remove(stdoutPath)
		_ = os.Remove(stderrPath)
	}

	return stdoutPath, stderrPath, cleanup, nil
}

func replayWorkerOutput(path string, writer, defaultWriter io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read worker output %q: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	target := writer
	if target == nil {
		target = defaultWriter
	}

	if _, err := io.Copy(target, file); err != nil {
		return fmt.Errorf("write worker output: %w", err)
	}

	return nil
}

func sortedEnvKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isValidEnvName(name string) bool {
	if name == "" {
		return false
	}

	for i := 0; i < len(name); i++ {
		ch := name[i]
		if i == 0 {
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && ch != '_' {
				return false
			}
			continue
		}

		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}

	return true
}

func quoteShellLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func toEnvironList(env map[string]string) []string {
	keys := sortedEnvKeys(env)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+env[key])
	}
	return out
}
