package decorators

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/runtime/isolation"
)

const (
	sandboxHelperEnv       = "OPAL_SANDBOX_HELPER"
	sandboxHelperLevelEnv  = "OPAL_SANDBOX_LEVEL"
	sandboxHelperNetEnv    = "OPAL_SANDBOX_NETWORK"
	sandboxHelperTokenEnv  = "OPAL_SANDBOX_TOKEN"
	sandboxHelperToken     = "opal-sandbox-v1"
	sandboxIDSuffix        = "/sandbox"
	sandboxHelperErrPrefix = "sandbox helper"
)

type SandboxTransportDecorator struct{}

var _ decorator.Transport = (*SandboxTransportDecorator)(nil)

type sandboxConfig struct {
	Level   string `decorator:"level"`
	Network string `decorator:"network"`
}

type sandboxRequest struct {
	Argv  []string          `json:"argv"`
	Dir   string            `json:"dir,omitempty"`
	Env   map[string]string `json:"env,omitempty"`
	Stdin []byte            `json:"stdin,omitempty"`
}

type sandboxResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   []byte `json:"stdout,omitempty"`
	Stderr   []byte `json:"stderr,omitempty"`
	Error    string `json:"error,omitempty"`
}

type sandboxHello struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type sandboxProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	enc    *json.Encoder
	dec    *json.Decoder
	stderr *bytes.Buffer

	mu     sync.Mutex
	closed atomic.Bool
	once   sync.Once
}

type sandboxSession struct {
	parent  decorator.Session
	process *sandboxProcess
	env     map[string]string
	cwd     string
}

var _ decorator.Session = (*sandboxSession)(nil)

func (d *SandboxTransportDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("sandbox").
		Summary("Execute in a subprocess sandbox transport context").
		Roles(decorator.RoleBoundary).
		ParamEnum("level", "Isolation level").
		Values("none", "basic", "standard", "maximum").
		Default("standard").
		Done().
		ParamEnum("network", "Network isolation policy").
		Values("allow", "deny", "loopback").
		Default("allow").
		Done().
		Block(decorator.BlockRequired).
		Build()
}

func (d *SandboxTransportDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork |
		decorator.TransportCapFilesystem |
		decorator.TransportCapEnvironment |
		decorator.TransportCapIsolation
}

func (d *SandboxTransportDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	cfg, _, err := decorator.DecodeInto[sandboxConfig](
		d.Descriptor().Schema,
		nil,
		params,
	)
	if err != nil {
		return nil, err
	}

	helpPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve sandbox helper executable: %w", err)
	}

	cmd := exec.Command(helpPath)
	cmd.Env = append(os.Environ(),
		sandboxHelperEnv+"=1",
		sandboxHelperTokenEnv+"="+sandboxHelperToken,
		sandboxHelperLevelEnv+"="+cfg.Level,
		sandboxHelperNetEnv+"="+cfg.Network,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create sandbox helper stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("create sandbox helper stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start sandbox helper: %w", err)
	}

	dec := json.NewDecoder(stdout)
	enc := json.NewEncoder(stdin)

	var hello sandboxHello
	if err := dec.Decode(&hello); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("read sandbox helper handshake: %w", err)
	}
	if !hello.OK {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if hello.Error != "" {
			return nil, fmt.Errorf("failed to create sandbox environment: %s", hello.Error)
		}
		return nil, errors.New("failed to create sandbox environment")
	}

	process := &sandboxProcess{
		cmd:    cmd,
		stdin:  stdin,
		enc:    enc,
		dec:    dec,
		stderr: &stderr,
	}

	return &sandboxSession{parent: parent, process: process}, nil
}

func (d *SandboxTransportDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return next
}

func (d *SandboxTransportDecorator) MaterializeSession() bool {
	return false
}

func (d *SandboxTransportDecorator) IsolationContext() decorator.IsolationContext {
	return isolation.NewIsolator()
}

func (s *sandboxSession) Run(ctx context.Context, argv []string, opts decorator.RunOpts) (decorator.Result, error) {
	if len(argv) == 0 {
		return decorator.Result{ExitCode: decorator.ExitFailure}, errors.New("argv cannot be empty")
	}

	if ctx.Err() != nil {
		return decorator.Result{ExitCode: decorator.ExitCanceled}, ctx.Err()
	}

	var stdinData []byte
	if opts.Stdin != nil {
		readData, err := io.ReadAll(opts.Stdin)
		if err != nil {
			return decorator.Result{ExitCode: decorator.ExitFailure}, fmt.Errorf("read stdin input: %w", err)
		}
		stdinData = readData
	}

	dir := opts.Dir
	if dir == "" {
		dir = s.cwd
	}

	req := sandboxRequest{
		Argv:  append([]string(nil), argv...),
		Dir:   dir,
		Env:   s.combinedEnv(),
		Stdin: stdinData,
	}

	type execResult struct {
		resp sandboxResponse
		err  error
	}
	resCh := make(chan execResult, 1)
	go func() {
		resp, execErr := s.process.execute(req)
		resCh <- execResult{resp: resp, err: execErr}
	}()

	select {
	case <-ctx.Done():
		s.process.terminate()
		return decorator.Result{ExitCode: decorator.ExitCanceled}, ctx.Err()
	case res := <-resCh:
		if res.err != nil {
			return decorator.Result{ExitCode: decorator.ExitFailure}, res.err
		}
		if res.resp.Error != "" {
			return decorator.Result{ExitCode: res.resp.ExitCode}, errors.New(res.resp.Error)
		}

		if opts.Stdout != nil {
			if _, err := opts.Stdout.Write(res.resp.Stdout); err != nil {
				return decorator.Result{ExitCode: decorator.ExitFailure}, fmt.Errorf("write stdout: %w", err)
			}
		}
		if opts.Stderr != nil {
			if _, err := opts.Stderr.Write(res.resp.Stderr); err != nil {
				return decorator.Result{ExitCode: decorator.ExitFailure}, fmt.Errorf("write stderr: %w", err)
			}
		}

		result := decorator.Result{ExitCode: res.resp.ExitCode}
		if opts.Stdout == nil {
			result.Stdout = res.resp.Stdout
		}
		if opts.Stderr == nil {
			result.Stderr = res.resp.Stderr
		}
		return result, nil
	}
}

func (s *sandboxSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	parent := s.parent
	if s.cwd != "" {
		parent = parent.WithWorkdir(s.cwd)
	}
	return parent.Put(ctx, data, path, mode)
}

func (s *sandboxSession) Get(ctx context.Context, path string) ([]byte, error) {
	parent := s.parent
	if s.cwd != "" {
		parent = parent.WithWorkdir(s.cwd)
	}
	return parent.Get(ctx, path)
}

func (s *sandboxSession) Env() map[string]string {
	return copyEnvMap(s.combinedEnv())
}

func (s *sandboxSession) WithEnv(delta map[string]string) decorator.Session {
	nextEnv := copyEnvMap(s.env)
	for k, v := range delta {
		nextEnv[k] = v
	}

	return &sandboxSession{
		parent:  s.parent,
		process: s.process,
		env:     nextEnv,
		cwd:     s.cwd,
	}
}

func (s *sandboxSession) WithWorkdir(dir string) decorator.Session {
	return &sandboxSession{
		parent:  s.parent,
		process: s.process,
		env:     copyEnvMap(s.env),
		cwd:     dir,
	}
}

func (s *sandboxSession) Cwd() string {
	if s.cwd != "" {
		return s.cwd
	}
	return s.parent.Cwd()
}

func (s *sandboxSession) Platform() string {
	return s.parent.Platform()
}

func (s *sandboxSession) ID() string {
	return s.parent.ID() + sandboxIDSuffix
}

func (s *sandboxSession) TransportScope() decorator.TransportScope {
	return decorator.TransportScopeIsolated
}

func (s *sandboxSession) Close() error {
	return s.process.Close()
}

func (s *sandboxSession) combinedEnv() map[string]string {
	env := s.parent.Env()
	for k, v := range s.env {
		env[k] = v
	}
	return env
}

func (p *sandboxProcess) execute(req sandboxRequest) (sandboxResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return sandboxResponse{}, errors.New("sandbox session is closed")
	}

	if err := p.enc.Encode(req); err != nil {
		return sandboxResponse{}, fmt.Errorf("send command to sandbox helper: %w", err)
	}

	var resp sandboxResponse
	if err := p.dec.Decode(&resp); err != nil {
		helperErr := p.stderr.String()
		if helperErr != "" {
			return sandboxResponse{}, fmt.Errorf("read command result from sandbox helper: %w (%s: %s)", err, sandboxHelperErrPrefix, helperErr)
		}
		return sandboxResponse{}, fmt.Errorf("read command result from sandbox helper: %w", err)
	}

	return resp, nil
}

func (p *sandboxProcess) Close() error {
	var closeErr error
	p.once.Do(func() {
		p.closed.Store(true)
		p.terminate()

		if err := p.cmd.Wait(); err != nil {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				closeErr = err
			}
		}
	})

	return closeErr
}

func (p *sandboxProcess) terminate() {
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

func runSandboxHelper() error {
	level := os.Getenv(sandboxHelperLevelEnv)
	network := os.Getenv(sandboxHelperNetEnv)

	if level == "" {
		level = "standard"
	}
	if network == "" {
		network = "allow"
	}

	var isolationLevel decorator.IsolationLevel
	switch level {
	case "none":
		isolationLevel = decorator.IsolationLevelNone
	case "basic":
		isolationLevel = decorator.IsolationLevelBasic
	case "standard":
		isolationLevel = decorator.IsolationLevelStandard
	case "maximum":
		isolationLevel = decorator.IsolationLevelMaximum
	default:
		return fmt.Errorf("invalid sandbox level %q", level)
	}

	var networkPolicy decorator.NetworkPolicy
	switch network {
	case "allow":
		networkPolicy = decorator.NetworkPolicyAllow
	case "deny":
		networkPolicy = decorator.NetworkPolicyDeny
	case "loopback":
		networkPolicy = decorator.NetworkPolicyLoopbackOnly
	default:
		return fmt.Errorf("invalid sandbox network policy %q", network)
	}

	config := decorator.IsolationConfig{
		NetworkPolicy:    networkPolicy,
		FilesystemPolicy: decorator.FilesystemPolicyFull,
		MemoryLock:       false,
	}

	isolator := isolation.NewIsolator()
	hello := sandboxHello{OK: true}
	if err := isolator.Isolate(isolationLevel, config); err != nil {
		hello.OK = false
		hello.Error = err.Error()
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(hello); err != nil {
		return fmt.Errorf("send helper handshake: %w", err)
	}
	if !hello.OK {
		return errors.New(hello.Error)
	}

	dec := json.NewDecoder(os.Stdin)
	for {
		var req sandboxRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode request: %w", err)
		}

		resp := runSandboxCommand(req)
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
	}
}

func runSandboxCommand(req sandboxRequest) sandboxResponse {
	if len(req.Argv) == 0 {
		return sandboxResponse{
			ExitCode: decorator.ExitFailure,
			Error:    "argv cannot be empty",
		}
	}

	cmd := exec.Command(req.Argv[0], req.Argv[1:]...)
	if req.Dir != "" {
		cmd.Dir = req.Dir
	}
	cmd.Env = sandboxMapToEnv(req.Env)
	if req.Stdin != nil {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := decorator.ExitSuccess
	respErr := ""
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = decorator.ExitFailure
			respErr = err.Error()
		}
	}

	return sandboxResponse{
		ExitCode: exitCode,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Error:    respErr,
	}
}

func sandboxMapToEnv(envMap map[string]string) []string {
	environ := make([]string, 0, len(envMap))
	for k, v := range envMap {
		environ = append(environ, k+"="+v)
	}
	return environ
}

func copyEnvMap(env map[string]string) map[string]string {
	if len(env) == 0 {
		return map[string]string{}
	}

	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = v
	}
	return out
}

func init() {
	if os.Getenv(sandboxHelperEnv) == "1" && os.Getenv(sandboxHelperTokenEnv) == sandboxHelperToken {
		if err := runSandboxHelper(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s failed: %v\n", sandboxHelperErrPrefix, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
