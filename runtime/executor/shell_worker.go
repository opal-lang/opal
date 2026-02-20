package executor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
)

const (
	workerInstanceEnvVar   = "OPAL_INTERNAL_WORKER_INSTANCE"
	workerStreamBufferSize = 64 * 1024
)

var (
	shellWorkerSequence    atomic.Uint64
	workerStreamBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, workerStreamBufferSize)
			return &buf
		},
	}
)

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

type workerRunError struct {
	cause          error
	commandStarted bool
}

func (e *workerRunError) Error() string {
	return e.cause.Error()
}

func (e *workerRunError) Unwrap() error {
	return e.cause
}

func newWorkerRunError(cause error, commandStarted bool) error {
	if cause == nil {
		return nil
	}
	return &workerRunError{cause: cause, commandStarted: commandStarted}
}

type shellWorkerPool struct {
	sessions *sessionRuntime

	mu            sync.Mutex
	workers       map[shellWorkerKey][]*shellWorker
	commandCounts map[shellWorkerKey]int
}

func newShellWorkerPool(sessions *sessionRuntime) *shellWorkerPool {
	return &shellWorkerPool{
		sessions:      sessions,
		workers:       make(map[shellWorkerKey][]*shellWorker),
		commandCounts: make(map[shellWorkerKey]int),
	}
}

func (p *shellWorkerPool) shouldUseWorker(transportID, shellName string) bool {
	key := shellWorkerKey{transportID: normalizedTransportID(transportID), shellName: shellName}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.commandCounts[key]++
	return p.commandCounts[key] > 1
}

func (p *shellWorkerPool) Run(ctx context.Context, req shellRunRequest) (int, error) {
	invariant.NotNil(p, "shell worker pool")
	invariant.Precondition(req.shellName != "", "shell worker request missing shell name")
	invariant.Precondition(req.command != "", "shell worker request missing command")

	worker, err := p.acquire(req.transportID, req.shellName)
	if err != nil {
		return decorator.ExitFailure, newWorkerRunError(err, false)
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
	p.commandCounts = make(map[shellWorkerKey]int)
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
	baseEnv  map[string]string
	baseCwd  string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	ctrl   *bufio.Reader
	ctrlR  *os.File

	streamCh    chan workerStreamChunk
	streamErrCh chan error
	closedCh    chan struct{}

	busy  bool
	alive atomic.Bool

	closeOnce sync.Once

	mu sync.Mutex
}

type workerStreamChunk struct {
	stderr bool
	data   []byte
	pool   *[]byte
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

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("worker stderr pipe: %w", err)
	}

	ctrlR, ctrlW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("worker control pipe: %w", err)
	}

	cmd.ExtraFiles = []*os.File{ctrlW}
	w.baseEnv = w.session.Env()
	w.baseCwd = w.session.Cwd()
	cmd.Env = toEnvironList(w.baseEnv)
	cmd.Dir = w.baseCwd

	if err := cmd.Start(); err != nil {
		_ = ctrlR.Close()
		_ = ctrlW.Close()
		return fmt.Errorf("start worker shell: %w", err)
	}
	_ = ctrlW.Close()

	w.cmd = cmd
	w.stdin = stdin
	w.stdout = stdoutPipe
	w.stderr = stderrPipe
	w.ctrlR = ctrlR
	w.ctrl = bufio.NewReader(ctrlR)
	w.streamCh = make(chan workerStreamChunk, 128)
	w.streamErrCh = make(chan error, 2)
	w.closedCh = make(chan struct{})

	go w.pumpStream(stdoutPipe, false)
	go w.pumpStream(stderrPipe, true)

	readyMarker := strconv.FormatUint(shellWorkerSequence.Add(1), 10)
	bootstrap := fmt.Sprintf("export %s=%s\nprintf '__OPAL_WORKER_READY_%s__\\n' >&3\n", workerInstanceEnvVar, quoteShellLiteral(w.instance), readyMarker)
	if _, err := io.WriteString(w.stdin, bootstrap); err != nil {
		w.close()
		return fmt.Errorf("bootstrap worker: %w", err)
	}

	for {
		line, err := w.ctrl.ReadString('\n')
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

	if err := w.drainPendingStreams(); err != nil {
		w.close()
		return decorator.ExitFailure, newWorkerRunError(err, true)
	}

	statusMarker := strconv.FormatUint(shellWorkerSequence.Add(1), 10)
	runReq := req
	runReq.environ = envDelta(w.baseEnv, req.environ)
	if req.workdir == w.baseCwd {
		runReq.workdir = ""
	}

	script := buildWorkerScript(runReq, statusMarker)
	if _, err := io.WriteString(w.stdin, script); err != nil {
		w.close()
		return decorator.ExitFailure, newWorkerRunError(fmt.Errorf("write worker request: %w", err), false)
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

	var status workerResult
	statusReady := false
	var streamErr error
	var flushTimer *time.Timer
	var flushCh <-chan time.Time
	flushIdle := 2 * time.Millisecond
	stopFlush := func() {
		if flushTimer == nil {
			return
		}
		if !flushTimer.Stop() {
			select {
			case <-flushTimer.C:
			default:
			}
		}
		flushCh = nil
	}
	armFlush := func() {
		if flushTimer == nil {
			flushTimer = time.NewTimer(flushIdle)
			flushCh = flushTimer.C
			return
		}
		if !flushTimer.Stop() {
			select {
			case <-flushTimer.C:
			default:
			}
		}
		flushTimer.Reset(flushIdle)
		flushCh = flushTimer.C
	}
	defer func() {
		if flushTimer != nil {
			stopFlush()
		}
	}()

	recordStatus := func(result workerResult) {
		status = result
		statusReady = true
		armFlush()
	}

	for {
		if !statusReady {
			select {
			case result := <-resultCh:
				recordStatus(result)
				continue
			default:
			}
		}

		cancelCh := ctx.Done()
		if statusReady {
			cancelCh = nil
		}

		select {
		case <-cancelCh:
			if !statusReady {
				select {
				case result := <-resultCh:
					recordStatus(result)
					continue
				default:
				}
			}

			w.close()
			return decorator.ExitCanceled, newWorkerRunError(ctx.Err(), true)

		case result := <-resultCh:
			recordStatus(result)

		case chunk := <-w.streamCh:
			if err := writeWorkerChunk(chunk, req.stdout, req.stderr); err != nil {
				w.close()
				return decorator.ExitFailure, newWorkerRunError(err, true)
			}
			if statusReady {
				armFlush()
			}

		case err := <-w.streamErrCh:
			if err != nil && streamErr == nil {
				streamErr = err
				w.close()
			}

		case <-flushCh:
			if !statusReady {
				continue
			}
			select {
			case chunk := <-w.streamCh:
				if err := writeWorkerChunk(chunk, req.stdout, req.stderr); err != nil {
					w.close()
					return decorator.ExitFailure, newWorkerRunError(err, true)
				}
				armFlush()
				continue
			default:
			}

			if status.err != nil {
				w.close()
				return decorator.ExitFailure, newWorkerRunError(status.err, true)
			}

			if streamErr != nil {
				w.close()
				return decorator.ExitFailure, newWorkerRunError(streamErr, true)
			}

			return status.exitCode, nil
		}
	}
}

func (w *shellWorker) readStatus(marker string) (int, error) {
	statusPrefix := "__OPAL_STATUS_" + marker + ":"
	for {
		line, err := w.ctrl.ReadString('\n')
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
		if w.closedCh != nil {
			close(w.closedCh)
		}
		if w.ctrlR != nil {
			_ = w.ctrlR.Close()
		}

		if w.stdin != nil {
			_ = w.stdin.Close()
		}

		if w.stdout != nil {
			_ = w.stdout.Close()
		}
		if w.stderr != nil {
			_ = w.stderr.Close()
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

func buildWorkerScript(req shellRunRequest, statusMarker string) string {
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
	b.WriteString(")\n")
	b.WriteString("__opal_status=$?\n")
	b.WriteString("printf '__OPAL_STATUS_")
	b.WriteString(statusMarker)
	b.WriteString(":%d\\n' \"$__opal_status\" >&3\n")

	return b.String()
}

func (w *shellWorker) pumpStream(reader io.Reader, stderr bool) {
	bufPtr := getWorkerStreamBuffer()
	buf := (*bufPtr)[:workerStreamBufferSize]
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := workerStreamChunk{stderr: stderr, data: buf[:n], pool: bufPtr}
			bufPtr = nil
			buf = nil
			select {
			case w.streamCh <- chunk:
			case <-w.closedCh:
				releaseWorkerStreamChunk(chunk)
				return
			}
		}

		if err == nil {
			if bufPtr == nil {
				bufPtr = getWorkerStreamBuffer()
				buf = (*bufPtr)[:workerStreamBufferSize]
			}
			continue
		}

		if bufPtr != nil {
			putWorkerStreamBuffer(bufPtr, 0)
			buf = nil
		}

		if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
			return
		}

		select {
		case w.streamErrCh <- fmt.Errorf("read worker stream: %w", err):
		case <-w.closedCh:
		}
		return
	}
}

func (w *shellWorker) drainPendingStreams() error {
	for {
		select {
		case chunk := <-w.streamCh:
			releaseWorkerStreamChunk(chunk)
			continue
		default:
		}
		break
	}

	for {
		select {
		case err := <-w.streamErrCh:
			if err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func writeWorkerChunk(chunk workerStreamChunk, stdout, stderr io.Writer) error {
	defer releaseWorkerStreamChunk(chunk)

	target := stdout
	defaultTarget := os.Stdout
	if chunk.stderr {
		target = stderr
		defaultTarget = os.Stderr
	}
	if target == nil {
		target = defaultTarget
	}
	if target == nil || len(chunk.data) == 0 {
		return nil
	}
	if _, err := target.Write(chunk.data); err != nil {
		return fmt.Errorf("write worker output: %w", err)
	}
	return nil
}

func getWorkerStreamBuffer() *[]byte {
	bufPtr, ok := workerStreamBufferPool.Get().(*[]byte)
	if !ok || bufPtr == nil {
		buf := make([]byte, workerStreamBufferSize)
		return &buf
	}

	buf := *bufPtr
	if cap(buf) < workerStreamBufferSize {
		buf = make([]byte, workerStreamBufferSize)
	}
	buf = buf[:workerStreamBufferSize]
	*bufPtr = buf
	return bufPtr
}

func putWorkerStreamBuffer(bufPtr *[]byte, used int) {
	if bufPtr == nil {
		return
	}

	buf := *bufPtr
	if cap(buf) < workerStreamBufferSize {
		buf = make([]byte, workerStreamBufferSize)
	}
	buf = buf[:workerStreamBufferSize]

	if used < 0 {
		used = 0
	}
	if used > len(buf) {
		used = len(buf)
	}
	if used > 0 {
		clear(buf[:used])
	}
	*bufPtr = buf
	workerStreamBufferPool.Put(bufPtr)
}

func releaseWorkerStreamChunk(chunk workerStreamChunk) {
	if chunk.pool == nil {
		return
	}
	putWorkerStreamBuffer(chunk.pool, len(chunk.data))
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
