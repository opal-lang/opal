package decorator

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/aledsdavies/opal/core/invariant"
)

// SSHSession implements Session for remote command execution over SSH.
type SSHSession struct {
	client *ssh.Client
	host   string
}

// NewSSHSession creates a new SSH session from connection parameters.
func NewSSHSession(params map[string]any) (*SSHSession, error) {
	host, ok := params["host"].(string)
	if !ok {
		return nil, fmt.Errorf("host parameter required")
	}

	user, ok := params["user"].(string)
	if !ok {
		user = os.Getenv("USER")
	}

	port, ok := params["port"].(int)
	if !ok {
		port = 22
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

	// Connect
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %w", err)
	}

	return &SSHSession{
		client: client,
		host:   host,
	}, nil
}

// Run executes a command on the remote host.
func (s *SSHSession) Run(ctx context.Context, argv []string, opts RunOpts) (Result, error) {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(len(argv) > 0, "argv cannot be empty")

	if ctx.Err() != nil {
		return Result{ExitCode: -1}, ctx.Err()
	}

	session, err := s.client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Build command string with optional directory change
	var cmd string
	if opts.Dir != "" {
		// Prepend cd command if directory specified
		cmd = fmt.Sprintf("cd %s && %s", shellQuote(opts.Dir), shellEscape(argv))
	} else {
		cmd = shellEscape(argv)
	}

	// Wire up I/O
	if opts.Stdin != nil {
		session.Stdin = opts.Stdin // Pass io.Reader directly (was: bytes.NewReader)
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

	// Execute with context cancellation
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL) // Best effort kill
		return Result{ExitCode: -1}, ctx.Err()
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

// Put writes data to a file on the remote host.
func (s *SSHSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(path != "", "path cannot be empty")

	if ctx.Err() != nil {
		return ctx.Err()
	}

	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Use cat to write file
	cmd := fmt.Sprintf("cat > %s && chmod %o %s", shellQuote(path), mode, shellQuote(path))
	session.Stdin = bytes.NewReader(data)

	return session.Run(cmd)
}

// Get reads data from a file on the remote host.
func (s *SSHSession) Get(ctx context.Context, path string) ([]byte, error) {
	invariant.NotNil(ctx, "ctx")
	invariant.Precondition(path != "", "path cannot be empty")

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	cmd := fmt.Sprintf("cat %s", shellQuote(path))
	if err := session.Run(cmd); err != nil {
		return nil, err
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

	if ctx.Err() != nil {
		return Result{ExitCode: -1}, ctx.Err()
	}

	session, err := s.base.client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Set environment variables using session.Setenv (safer than VAR=val cmd)
	for k, v := range s.delta {
		_ = session.Setenv(k, v) // Best effort - some SSH servers don't allow Setenv
	}

	// Build command with optional cd
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

	// Wire up I/O
	if opts.Stdin != nil {
		session.Stdin = opts.Stdin // Pass io.Reader directly (was: bytes.NewReader)
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

	// Execute with context cancellation
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL) // Best effort kill
		return Result{ExitCode: -1}, ctx.Err()
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

func (s *SSHSessionWithEnv) Close() error {
	return s.base.Close()
}

// SSHTransport implements Transport for SSH connections.
type SSHTransport struct{}

func (t *SSHTransport) Descriptor() Descriptor {
	return Descriptor{
		Path: "ssh",
	}
}

func (t *SSHTransport) Open(parent Session, params map[string]any) (Session, error) {
	return NewSSHSession(params)
}

func (t *SSHTransport) Wrap(next ExecNode, params map[string]any) ExecNode {
	// TODO: Implement execution wrapping
	return next
}

// Helper functions

func getHostKeyCallback(params map[string]any) ssh.HostKeyCallback {
	// Check if strict host key checking is disabled (for testing)
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
		// If known_hosts doesn't exist or can't be read, use InsecureIgnoreHostKey
		// This allows first-time connections (TOFU - Trust On First Use)
		return ssh.InsecureIgnoreHostKey()
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

	conn, err := net.Dial("unix", socket)
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
