package decorator

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/builtwithtofu/sigil/core/invariant"
)

// SSHSession implements Session for remote command execution over SSH.
type SSHSession struct {
	client   *ssh.Client
	host     string
	platform string
}

const (
	defaultNetworkDialTimeout  = 30 * time.Second
	defaultSSHHandshakeTimeout = 10 * time.Second
	maxSSHDialPoolSize         = 100
)

var sshDialPool = make(chan struct{}, maxSSHDialPoolSize)

var sshClientDialContext = func(client *ssh.Client, ctx context.Context, network, addr string) (net.Conn, error) {
	return client.DialContext(ctx, network, addr)
}

var sshNewClientConn = func(conn net.Conn, addr string, config *ssh.ClientConfig) (ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
	return ssh.NewClientConn(conn, addr, config)
}

// NewSSHSession creates a new SSH session from connection parameters.
func NewSSHSession(params map[string]any) (*SSHSession, error) {
	dialCtx, cancel := withDefaultDialDeadline(context.Background())
	defer cancel()

	client, host, err := dialSSHClient(dialCtx, (&net.Dialer{}).DialContext, params)
	if err != nil {
		return nil, err
	}

	platform := detectRemotePlatform(client)

	return &SSHSession{
		client:   client,
		host:     host,
		platform: platform,
	}, nil
}

func dialSSHClient(ctx context.Context, dialContext func(context.Context, string, string) (net.Conn, error), params map[string]any) (*ssh.Client, string, error) {
	if err := acquireSSHDialSlot(ctx); err != nil {
		return nil, "", err
	}
	defer releaseSSHDialSlot()

	host, ok := params["host"].(string)
	if !ok {
		return nil, "", fmt.Errorf("host parameter required")
	}

	user, ok := params["user"].(string)
	if !ok {
		user = os.Getenv("USER")
	}

	port := 22
	switch v := params["port"].(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	}

	// Validate host
	if host == "" || strings.TrimSpace(host) == "" {
		return nil, "", TransportError{
			Code:      TransportErrorCodeValidationFailed,
			Message:   "SSH host cannot be empty",
			Retryable: false,
		}
	}

	// Validate port range (1-65535)
	if port < 1 || port > 65535 {
		return nil, "", TransportError{
			Code:      TransportErrorCodeValidationFailed,
			Message:   fmt.Sprintf("SSH port must be between 1 and 65535, got %d", port),
			Retryable: false,
		}
	}

	// Validate key file if provided as string path
	if keyStr, ok := params["key"].(string); ok && keyStr != "" {
		if _, err := os.Stat(keyStr); err != nil {
			return nil, "", TransportError{
				Code:      TransportErrorCodeValidationFailed,
				Message:   fmt.Sprintf("SSH key file not accessible: %v", err),
				Retryable: false,
				Cause:     err,
			}
		}
	}

	// Create SSH client config
	var authMethods []ssh.AuthMethod

	// Try direct signer first (for testing)
	switch key := params["key"].(type) {
	case ssh.Signer:
		authMethods = append(authMethods, ssh.PublicKeys(key))
	case string:
		// Try keyfile auth if string path provided
		if keyAuth := sshKeyAuth(key); keyAuth != nil {
			authMethods = append(authMethods, keyAuth)
		}
	}

	// Fall back to SSH agent
	if len(authMethods) == 0 {
		if agentAuth := sshAgentAuth(); agentAuth != nil {
			authMethods = append(authMethods, agentAuth)
		}
	}

	// Host key verification
	hostKeyCallback := getHostKeyCallback(params)

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	handshakeTimeout := getSSHHandshakeTimeout(params)

	// Connect
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := dialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", TransportError{
			Code:      TransportErrorCodeConnect,
			Message:   "ssh dial failed",
			Retryable: true,
			Cause:     err,
		}
	}

	sshConn, chans, reqs, err := sshNewClientConnWithTimeout(ctx, conn, addr, config, handshakeTimeout)
	if err != nil {
		_ = conn.Close()
		return nil, "", TransportError{
			Code:      TransportErrorCodeConnect,
			Message:   "ssh dial failed",
			Retryable: true,
			Cause:     err,
		}
	}

	client := ssh.NewClient(sshConn, chans, reqs)

	return client, host, nil
}

// Run executes a command on the remote host.
func (s *SSHSession) Run(ctx context.Context, argv []string, opts RunOpts) (Result, error) {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(len(argv) > 0, "argv cannot be empty")

	var cmd string
	if opts.Dir != "" {
		cmd = fmt.Sprintf("cd %s && %s", shellQuote(opts.Dir), shellEscape(argv))
	} else {
		cmd = shellEscape(argv)
	}

	return runSSHSession(ctx, s.client, cmd, opts, nil)
}

// Put writes data to a file on the remote host.
func (s *SSHSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(path != "", "path cannot be empty")

	if ctx.Err() != nil {
		return TransportError{
			Code:      TransportErrorCodeContext,
			Message:   "put context cancelled",
			Retryable: false,
			Cause:     ctx.Err(),
		}
	}

	session, err := s.client.NewSession()
	if err != nil {
		return TransportError{
			Code:      TransportErrorCodeSession,
			Message:   "failed to create ssh session",
			Retryable: true,
			Cause:     err,
		}
	}
	defer func() { _ = session.Close() }()

	// Use cat to write file
	cmd := fmt.Sprintf("cat > %s && chmod %o %s", shellQuote(path), mode, shellQuote(path))
	session.Stdin = bytes.NewReader(data)

	if err := session.Run(cmd); err != nil {
		return TransportError{
			Code:      TransportErrorCodeIO,
			Message:   "failed to write remote file",
			Retryable: true,
			Cause:     err,
		}
	}

	return nil
}

// Get reads data from a file on the remote host.
func (s *SSHSession) Get(ctx context.Context, path string) ([]byte, error) {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(path != "", "path cannot be empty")

	if ctx.Err() != nil {
		return nil, TransportError{
			Code:      TransportErrorCodeContext,
			Message:   "get context cancelled",
			Retryable: false,
			Cause:     ctx.Err(),
		}
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, TransportError{
			Code:      TransportErrorCodeSession,
			Message:   "failed to create ssh session",
			Retryable: true,
			Cause:     err,
		}
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	cmd := fmt.Sprintf("cat %s", shellQuote(path))
	if err := session.Run(cmd); err != nil {
		return nil, TransportError{
			Code:      TransportErrorCodeIO,
			Message:   "failed to read remote file",
			Retryable: true,
			Cause:     err,
		}
	}

	return stdout.Bytes(), nil
}

// Env returns the remote environment variables.
func (s *SSHSession) Env() map[string]string {
	session, err := s.client.NewSession()
	if err != nil {
		return make(map[string]string)
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	// Run env command
	if err := session.Run("env"); err != nil {
		return make(map[string]string)
	}

	// Parse env output
	return parseEnv(stdout.String())
}

// WithEnv returns a new Session with environment delta applied.
// For SSH, this creates a wrapper that sets env vars before commands.
func (s *SSHSession) WithEnv(delta map[string]string) Session {
	return &SSHSessionWithEnv{
		base:  s,
		delta: delta,
		cwd:   "",
	}
}

// WithWorkdir returns a new Session with working directory set.
func (s *SSHSession) WithWorkdir(dir string) Session {
	invariant.Precondition(dir != "", "dir cannot be empty")
	return &SSHSessionWithEnv{
		base:  s,
		delta: make(map[string]string),
		cwd:   dir,
	}
}

// Cwd returns the current working directory on the remote host.
func (s *SSHSession) Cwd() string {
	session, err := s.client.NewSession()
	if err != nil {
		return ""
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	if err := session.Run("pwd"); err != nil {
		return ""
	}

	return strings.TrimSpace(stdout.String())
}

// ID returns the session identifier for SSH sessions.
// Format: "ssh:hostname" to distinguish from local and other SSH sessions.
func (s *SSHSession) ID() string {
	return "ssh:" + s.host
}

// TransportScope returns the transport scope for SSH sessions.
func (s *SSHSession) TransportScope() TransportScope {
	return TransportScopeSSH
}

// Platform returns the remote target OS for this SSH session.
func (s *SSHSession) Platform() string {
	if s.platform == "" {
		return ""
	}
	return s.platform
}

func (s *SSHSession) IsolationContext() IsolationContext {
	return nil
}

func (s *SSHSession) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if s.client == nil {
		return nil, errors.New("ssh session not connected")
	}

	if ctx == nil {
		return nil, fmt.Errorf("ssh dial requires context with deadline (recommended timeout <= %s)", defaultNetworkDialTimeout)
	}

	if _, ok := ctx.Deadline(); !ok {
		return nil, fmt.Errorf("ssh dial requires context deadline to prevent hangs (recommended timeout <= %s)", defaultNetworkDialTimeout)
	}

	conn, err := sshClientDialContext(s.client, ctx, network, addr)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("ssh dial timed out for %s %s: %w", network, addr, err)
		}
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("ssh dial canceled for %s %s: %w", network, addr, err)
		}
		return nil, err
	}

	return conn, nil
}

func (s *SSHSession) NetworkDialer() NetworkDialer {
	return s
}

// Close closes the SSH connection.
func (s *SSHSession) Close() error {
	return s.client.Close()
}

// SSHSessionWithEnv wraps SSHSession to inject environment variables and working directory.
type SSHSessionWithEnv struct {
	base  *SSHSession
	delta map[string]string
	cwd   string
}

func (s *SSHSessionWithEnv) Run(ctx context.Context, argv []string, opts RunOpts) (Result, error) {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(len(argv) > 0, "argv cannot be empty")

	var cmd string
	workdir := opts.Dir
	if workdir == "" {
		workdir = s.cwd
	}
	if workdir != "" {
		cmd = fmt.Sprintf("cd %s && %s", shellQuote(workdir), shellEscape(argv))
	} else {
		cmd = shellEscape(argv)
	}

	return runSSHSession(ctx, s.base.client, cmd, opts, s.delta)
}

func (s *SSHSessionWithEnv) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return s.base.Put(ctx, data, path, mode)
}

func (s *SSHSessionWithEnv) Get(ctx context.Context, path string) ([]byte, error) {
	return s.base.Get(ctx, path)
}

func (s *SSHSessionWithEnv) Env() map[string]string {
	// Merge base env with delta
	env := s.base.Env()
	for k, v := range s.delta {
		env[k] = v
	}
	return env
}

func (s *SSHSessionWithEnv) WithEnv(delta map[string]string) Session {
	// Merge deltas
	merged := make(map[string]string)
	for k, v := range s.delta {
		merged[k] = v
	}
	for k, v := range delta {
		merged[k] = v
	}
	return &SSHSessionWithEnv{
		base:  s.base,
		delta: merged,
		cwd:   s.cwd,
	}
}

func (s *SSHSessionWithEnv) WithWorkdir(dir string) Session {
	invariant.Precondition(dir != "", "dir cannot be empty")
	return &SSHSessionWithEnv{
		base:  s.base,
		delta: s.delta,
		cwd:   dir,
	}
}

func (s *SSHSessionWithEnv) Cwd() string {
	if s.cwd != "" {
		return s.cwd
	}
	return s.base.Cwd()
}

// ID returns the session identifier, delegating to the base SSH session.
func (s *SSHSessionWithEnv) ID() string {
	return s.base.ID()
}

// TransportScope returns the transport scope, delegating to the base SSH session.
func (s *SSHSessionWithEnv) TransportScope() TransportScope {
	return s.base.TransportScope()
}

// Platform returns the remote target OS, delegating to the base SSH session.
func (s *SSHSessionWithEnv) Platform() string {
	return s.base.Platform()
}

func (s *SSHSessionWithEnv) Close() error {
	return nil
}

// SSHTransport implements Transport for SSH connections.
type SSHTransport struct{}

func init() {
	if err := Register("ssh.connect", &SSHTransport{}); err != nil {
		panic(fmt.Sprintf("failed to register @ssh.connect decorator: %v", err))
	}
}

func (t *SSHTransport) Descriptor() Descriptor {
	return Descriptor{
		Path: "ssh.connect",
	}
}

func (t *SSHTransport) Capabilities() TransportCaps {
	return TransportCapNetwork | TransportCapEnvironment
}

func (t *SSHTransport) Open(parent Session, params map[string]any) (Session, error) {
	dialer, err := getNetworkDialer(parent)
	if err != nil {
		return nil, err
	}

	dialCtx, cancel := withDefaultDialDeadline(context.Background())
	defer cancel()

	sshSession, err := dialSSHSession(dialCtx, dialer, params)
	if err != nil {
		return nil, err
	}

	return sshSession, nil
}

func (t *SSHTransport) Wrap(next ExecNode, params map[string]any) ExecNode {
	return &sshTransportNode{
		next:   next,
		params: params,
	}
}

func (t *SSHTransport) MaterializeSession() bool {
	return true
}

func (t *SSHTransport) IsolationContext() IsolationContext {
	return nil
}

type sshTransportNode struct {
	next   ExecNode
	params map[string]any
}

func (n *sshTransportNode) Execute(ctx ExecContext) (Result, error) {
	if ctx.Session != nil && ctx.Session.TransportScope() == TransportScopeSSH && !strings.HasPrefix(ctx.Session.ID(), "ssh:") {
		session := Session(ctx.Session)
		if env := sshEnvDelta(n.params); len(env) > 0 {
			session = session.WithEnv(env)
		}
		if workdir, ok := n.params["workdir"].(string); ok && workdir != "" {
			session = session.WithWorkdir(workdir)
		}

		if n.next != nil {
			return n.next.Execute(ctx.WithSession(session))
		}

		command, ok := n.params["command"].(string)
		if !ok || strings.TrimSpace(command) == "" {
			return Result{ExitCode: ExitSuccess}, nil
		}

		shellName := "bash"
		if v, ok := n.params["shell"].(string); ok && v != "" {
			shellName = v
		}

		argv, err := sshShellCommandArgs(shellName, command)
		if err != nil {
			return Result{ExitCode: ExitFailure}, err
		}

		execCtx := ctx.Context
		if execCtx == nil {
			execCtx = context.Background()
		}

		return session.Run(execCtx, argv, RunOpts{
			Stdin:  ctx.Stdin,
			Stdout: ctx.Stdout,
			Stderr: ctx.Stderr,
		})
	}

	dialer, err := getNetworkDialer(ctx.Session)
	if err != nil {
		return Result{ExitCode: ExitFailure}, err
	}

	execCtx := ctx.Context
	if execCtx == nil {
		execCtx = context.Background()
	}
	execCtx, cancel := withDefaultDialDeadline(execCtx)
	defer cancel()

	sshSession, err := dialSSHSession(execCtx, dialer, n.params)
	if err != nil {
		return Result{ExitCode: ExitFailure}, err
	}
	defer func() { _ = sshSession.Close() }()

	session := Session(sshSession)
	if env := sshEnvDelta(n.params); len(env) > 0 {
		session = session.WithEnv(env)
	}
	if workdir, ok := n.params["workdir"].(string); ok && workdir != "" {
		session = session.WithWorkdir(workdir)
	}

	if n.next != nil {
		return n.next.Execute(ctx.WithSession(session))
	}

	command, ok := n.params["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return Result{ExitCode: ExitSuccess}, nil
	}

	shellName := "bash"
	if v, ok := n.params["shell"].(string); ok && v != "" {
		shellName = v
	}

	argv, err := sshShellCommandArgs(shellName, command)
	if err != nil {
		return Result{ExitCode: ExitFailure}, err
	}

	return session.Run(execCtx, argv, RunOpts{
		Stdin:  ctx.Stdin,
		Stdout: ctx.Stdout,
		Stderr: ctx.Stderr,
	})
}

func dialSSHSession(ctx context.Context, dialer NetworkDialer, params map[string]any) (*SSHSession, error) {
	client, host, err := dialSSHClient(ctx, dialer.DialContext, params)
	if err != nil {
		return nil, err
	}

	platform := detectRemotePlatform(client)

	return &SSHSession{
		client:   client,
		host:     host,
		platform: platform,
	}, nil
}

// Helper functions

func runSSHSession(ctx context.Context, client *ssh.Client, cmd string, opts RunOpts, env map[string]string) (Result, error) {
	if ctx.Err() != nil {
		return Result{ExitCode: -1}, TransportError{
			Code:      TransportErrorCodeContext,
			Message:   "command context cancelled",
			Retryable: false,
			Cause:     ctx.Err(),
		}
	}

	session, err := client.NewSession()
	if err != nil {
		return Result{}, TransportError{
			Code:      TransportErrorCodeSession,
			Message:   "failed to create ssh session",
			Retryable: true,
			Cause:     err,
		}
	}
	defer func() { _ = session.Close() }()

	for k, v := range env {
		_ = session.Setenv(k, v)
	}

	if opts.Stdin != nil {
		session.Stdin = opts.Stdin
	}

	var stdout, stderr bytes.Buffer
	if opts.Stdout != nil {
		session.Stdout = opts.Stdout
	} else {
		session.Stdout = &stdout
	}
	if opts.Stderr != nil {
		session.Stderr = opts.Stderr
	} else {
		session.Stderr = &stderr
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return Result{ExitCode: -1}, TransportError{
			Code:      TransportErrorCodeContext,
			Message:   "command context cancelled",
			Retryable: false,
			Cause:     ctx.Err(),
		}
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				exitCode = 1
			}
		}
		return Result{
			ExitCode: exitCode,
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
		}, nil
	}
}

func getHostKeyCallback(params map[string]any) ssh.HostKeyCallback {
	// Check if strict host key checking is disabled (opt-in insecure mode)
	if strictHostKey, ok := params["strict_host_key"].(bool); ok && !strictHostKey {
		return ssh.InsecureIgnoreHostKey()
	}

	// Get known_hosts path (default: ~/.ssh/known_hosts)
	knownHostsPath := os.ExpandEnv("$HOME/.ssh/known_hosts")
	if path, ok := params["known_hosts_path"].(string); ok {
		knownHostsPath = path
	}

	// Try to load known_hosts file
	callback, err := loadKnownHosts(knownHostsPath)
	if err != nil {
		// Fail closed: reject connection if host-key verification cannot be established
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return fmt.Errorf("host-key verification failed: known_hosts file not found or unreadable at %s (set strict_host_key=false to allow insecure connections)", knownHostsPath)
		}
	}

	return callback
}

func loadKnownHosts(path string) (ssh.HostKeyCallback, error) {
	// Read known_hosts file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse known_hosts
	knownHosts := make(map[string]ssh.PublicKey)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: hostname key-type key-data
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		hostname := parts[0]
		keyType := parts[1]
		keyData := parts[2]

		// Decode base64 key
		keyBytes, err := base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			continue
		}

		pubKey, err := ssh.ParsePublicKey(keyBytes)
		if err != nil {
			continue
		}

		knownHosts[hostname+":"+keyType] = pubKey
	}

	// Return callback that checks against known_hosts
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Build lookup key
		lookupKey := hostname + ":" + key.Type()

		// Check if host key is known
		knownKey, ok := knownHosts[lookupKey]
		if !ok {
			return fmt.Errorf("host key not found in known_hosts: %s", hostname)
		}

		// Compare keys
		if !bytes.Equal(key.Marshal(), knownKey.Marshal()) {
			return fmt.Errorf("host key mismatch for %s", hostname)
		}

		return nil
	}, nil
}

func sshKeyAuth(keyPath string) ssh.AuthMethod {
	// Read private key file
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil
	}

	return ssh.PublicKeys(signer)
}

func sshAgentAuth() ssh.AuthMethod {
	// Connect to SSH agent
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", socket)
	if err != nil {
		return nil
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

func parseEnv(output string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.IndexByte(line, '='); idx > 0 {
			env[line[:idx]] = line[idx+1:]
		}
	}
	return env
}

func detectRemotePlatform(client *ssh.Client) string {
	platform := probeRemoteUname(client)
	if platform != "" {
		return platform
	}

	platform = probeRemoteWindowsVer(client)
	if platform != "" {
		return platform
	}

	return ""
}

func probeRemoteUname(client *ssh.Client) string {
	session, err := client.NewSession()
	if err != nil {
		return ""
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	session.Stdout = &stdout
	if err := session.Run("uname -s"); err != nil {
		return ""
	}

	return normalizePlatform(stdout.String())
}

func probeRemoteWindowsVer(client *ssh.Client) string {
	session, err := client.NewSession()
	if err != nil {
		return ""
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	session.Stdout = &stdout
	if err := session.Run("cmd /C ver"); err != nil {
		return ""
	}

	return normalizePlatform(stdout.String())
}

func normalizePlatform(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}

	switch {
	case strings.Contains(value, "linux"):
		return "linux"
	case strings.Contains(value, "darwin"):
		return "darwin"
	case strings.Contains(value, "windows"):
		return "windows"
	case strings.Contains(value, "freebsd"):
		return "freebsd"
	case strings.Contains(value, "openbsd"):
		return "openbsd"
	case strings.Contains(value, "netbsd"):
		return "netbsd"
	default:
		return ""
	}
}

func sshEnvDelta(params map[string]any) map[string]string {
	raw, ok := params["env"]
	if !ok || raw == nil {
		return nil
	}

	delta := make(map[string]string)

	switch typed := raw.(type) {
	case map[string]string:
		for k, v := range typed {
			delta[k] = v
		}
	case map[string]any:
		for k, v := range typed {
			delta[k] = fmt.Sprint(v)
		}
	}

	if len(delta) == 0 {
		return nil
	}

	return delta
}

func sshShellCommandArgs(shellName, command string) ([]string, error) {
	switch shellName {
	case "bash":
		return []string{"bash", "-c", command}, nil
	case "pwsh":
		return []string{"pwsh", "-NoProfile", "-NonInteractive", "-Command", command}, nil
	case "cmd":
		return []string{"cmd", "/C", command}, nil
	default:
		return nil, fmt.Errorf("unsupported shell %q: expected one of bash, pwsh, cmd", shellName)
	}
}

func getSSHHandshakeTimeout(params map[string]any) time.Duration {
	if params == nil {
		return defaultSSHHandshakeTimeout
	}

	if value, ok := params["handshake_timeout"]; ok {
		switch typed := value.(type) {
		case time.Duration:
			if typed > 0 {
				return typed
			}
		case string:
			parsed, err := time.ParseDuration(typed)
			if err == nil && parsed > 0 {
				return parsed
			}
		case int:
			if typed > 0 {
				return time.Duration(typed) * time.Second
			}
		case int64:
			if typed > 0 {
				return time.Duration(typed) * time.Second
			}
		case float64:
			if typed > 0 {
				return time.Duration(typed * float64(time.Second))
			}
		}
	}

	return defaultSSHHandshakeTimeout
}

func sshNewClientConnWithTimeout(ctx context.Context, conn net.Conn, addr string, config *ssh.ClientConfig, handshakeTimeout time.Duration) (ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
	handshakeCtx := ctx
	var cancel context.CancelFunc
	if handshakeTimeout > 0 {
		handshakeCtx, cancel = context.WithTimeout(ctx, handshakeTimeout)
		defer cancel()
	}

	type result struct {
		sshConn ssh.Conn
		chans   <-chan ssh.NewChannel
		reqs    <-chan *ssh.Request
		err     error
	}

	done := make(chan result, 1)
	go func() {
		sshConn, chans, reqs, err := sshNewClientConn(conn, addr, config)
		done <- result{sshConn: sshConn, chans: chans, reqs: reqs, err: err}
	}()

	select {
	case <-handshakeCtx.Done():
		_ = conn.Close()
		if errors.Is(handshakeCtx.Err(), context.DeadlineExceeded) {
			return nil, nil, nil, fmt.Errorf("ssh handshake timed out after %s for %s: %w", handshakeTimeout, addr, handshakeCtx.Err())
		}
		return nil, nil, nil, fmt.Errorf("ssh handshake canceled for %s: %w", addr, handshakeCtx.Err())
	case out := <-done:
		return out.sshConn, out.chans, out.reqs, out.err
	}
}

func acquireSSHDialSlot(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("ssh dial pool requires context")
	}

	select {
	case sshDialPool <- struct{}{}:
		return nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("ssh dial pool exhausted and timed out waiting for slot (max=%d): %w", maxSSHDialPoolSize, ctx.Err())
		}
		return fmt.Errorf("ssh dial pool wait canceled: %w", ctx.Err())
	}
}

func releaseSSHDialSlot() {
	select {
	case <-sshDialPool:
	default:
	}
}

func withDefaultDialDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, defaultNetworkDialTimeout)
}

func shellEscape(argv []string) string {
	escaped := make([]string, len(argv))
	for i, arg := range argv {
		escaped[i] = shellQuote(arg)
	}
	return strings.Join(escaped, " ")
}

func shellQuote(s string) string {
	// Simple quoting - wrap in single quotes and escape single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
