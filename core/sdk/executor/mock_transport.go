package executor

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"

	"github.com/opal-lang/opal/core/invariant"
)

// MockTransport is a test transport that records all operations and allows
// configurable responses. Useful for testing decorators and execution contexts
// without actually executing commands.
//
// Unlike LocalTransport (which uses os.Environ()), MockTransport uses an
// in-memory base environment that you configure. This demonstrates how
// remote transports (SSH, Docker, SSM) work - they don't use os.Environ().
//
// Example usage:
//
//	mock := NewMockTransport()
//	mock.SetBaseEnv(map[string]string{
//	    "PATH": "/usr/bin:/bin",
//	    "HOME": "/home/testuser",
//	})
//	mock.SetExecResponse("echo hello", ExitSuccess, "hello\n", "")
//
//	// Execute command
//	exitCode, err := mock.Exec(ctx, []string{"echo", "hello"}, opts)
//
//	// Verify what was executed
//	calls := mock.ExecCalls()
//	assert.Equal(t, 1, len(calls))
//	assert.Equal(t, []string{"echo", "hello"}, calls[0].Argv)
type MockTransport struct {
	mu sync.Mutex

	// Base environment (simulates remote environment)
	baseEnv map[string]string

	// Configured responses for Exec calls
	execResponses map[string]*ExecResponse

	// Default response if no specific response configured
	defaultExitCode int
	defaultStdout   string
	defaultStderr   string

	// Recorded calls
	execCalls []ExecCall
	putCalls  []PutCall
	getCalls  []GetCall

	// In-memory file system for Put/Get
	files map[string]*MockFile

	// Close tracking
	closeCalled bool
}

// ExecResponse defines the response for a specific command.
type ExecResponse struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// ExecCall records an Exec invocation.
type ExecCall struct {
	Argv   []string
	Env    map[string]string // Merged environment (base + decorator vars)
	Dir    string
	Stdin  string // Captured stdin content
	Stdout string // Actual stdout written
	Stderr string // Actual stderr written
}

// PutCall records a Put invocation.
type PutCall struct {
	Content []byte
	Dst     string
	Mode    fs.FileMode
}

// GetCall records a Get invocation.
type GetCall struct {
	Src     string
	Content []byte // Content that was returned
}

// MockFile represents a file in the mock filesystem.
type MockFile struct {
	Content []byte
	Mode    fs.FileMode
}

// NewMockTransport creates a new mock transport with sensible defaults.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		baseEnv: map[string]string{
			"PATH": "/usr/bin:/bin",
			"HOME": "/home/mockuser",
			"USER": "mockuser",
		},
		execResponses:   make(map[string]*ExecResponse),
		defaultExitCode: ExitSuccess,
		defaultStdout:   "",
		defaultStderr:   "",
		files:           make(map[string]*MockFile),
	}
}

// SetBaseEnv sets the base environment (simulates remote environment).
// This replaces the default base environment.
func (m *MockTransport) SetBaseEnv(env map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.baseEnv = env
}

// AddBaseEnv adds variables to the base environment.
func (m *MockTransport) AddBaseEnv(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.baseEnv == nil {
		m.baseEnv = make(map[string]string)
	}
	m.baseEnv[key] = value
}

// SetExecResponse configures the response for a specific command.
// The command is matched by joining argv with spaces.
func (m *MockTransport) SetExecResponse(command string, exitCode int, stdout, stderr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execResponses[command] = &ExecResponse{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

// SetExecError configures an error response for a specific command.
func (m *MockTransport) SetExecError(command string, exitCode int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execResponses[command] = &ExecResponse{
		ExitCode: exitCode,
		Err:      err,
	}
}

// SetDefaultResponse sets the default response for commands without specific responses.
func (m *MockTransport) SetDefaultResponse(exitCode int, stdout, stderr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultExitCode = exitCode
	m.defaultStdout = stdout
	m.defaultStderr = stderr
}

// Exec executes a mock command.
func (m *MockTransport) Exec(ctx context.Context, argv []string, opts ExecOpts) (int, error) {
	invariant.Precondition(len(argv) > 0, "argv cannot be empty")
	invariant.NotNil(ctx, "context")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Merge base environment with decorator-added variables
	// This simulates how remote transports work (SSH, Docker, SSM)
	mergedEnv := make(map[string]string)
	for k, v := range m.baseEnv {
		mergedEnv[k] = v
	}
	for k, v := range opts.Env {
		mergedEnv[k] = v
	}

	// Capture stdin
	var stdinContent string
	if opts.Stdin != nil {
		data, _ := io.ReadAll(opts.Stdin)
		stdinContent = string(data)
	}

	// Find response for this command
	command := strings.Join(argv, " ")
	response, exists := m.execResponses[command]
	if !exists {
		response = &ExecResponse{
			ExitCode: m.defaultExitCode,
			Stdout:   m.defaultStdout,
			Stderr:   m.defaultStderr,
		}
	}

	// Write stdout/stderr
	var stdoutContent, stderrContent string
	if opts.Stdout != nil && response.Stdout != "" {
		_, _ = opts.Stdout.Write([]byte(response.Stdout))
		stdoutContent = response.Stdout
	}
	if opts.Stderr != nil && response.Stderr != "" {
		_, _ = opts.Stderr.Write([]byte(response.Stderr))
		stderrContent = response.Stderr
	}

	// Record the call
	m.execCalls = append(m.execCalls, ExecCall{
		Argv:   argv,
		Env:    mergedEnv,
		Dir:    opts.Dir,
		Stdin:  stdinContent,
		Stdout: stdoutContent,
		Stderr: stderrContent,
	})

	return response.ExitCode, response.Err
}

// Put stores a file in the mock filesystem.
func (m *MockTransport) Put(ctx context.Context, src io.Reader, dst string, mode fs.FileMode) error {
	invariant.NotNil(ctx, "context")
	invariant.NotNil(src, "source reader")
	invariant.Precondition(dst != "", "destination path cannot be empty")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Read content
	content, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	// Store in mock filesystem
	m.files[dst] = &MockFile{
		Content: content,
		Mode:    mode,
	}

	// Record the call
	m.putCalls = append(m.putCalls, PutCall{
		Content: content,
		Dst:     dst,
		Mode:    mode,
	})

	return nil
}

// Get retrieves a file from the mock filesystem.
func (m *MockTransport) Get(ctx context.Context, src string, dst io.Writer) error {
	invariant.NotNil(ctx, "context")
	invariant.Precondition(src != "", "source path cannot be empty")
	invariant.NotNil(dst, "destination writer")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if file exists
	file, exists := m.files[src]
	if !exists {
		return fmt.Errorf("file not found: %s", src)
	}

	// Write content
	_, err := dst.Write(file.Content)
	if err != nil {
		return err
	}

	// Record the call
	m.getCalls = append(m.getCalls, GetCall{
		Src:     src,
		Content: file.Content,
	})

	return nil
}

// Close marks the transport as closed.
func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	return nil
}

// ExecCalls returns all recorded Exec calls.
func (m *MockTransport) ExecCalls() []ExecCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	calls := make([]ExecCall, len(m.execCalls))
	copy(calls, m.execCalls)
	return calls
}

// PutCalls returns all recorded Put calls.
func (m *MockTransport) PutCalls() []PutCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]PutCall, len(m.putCalls))
	copy(calls, m.putCalls)
	return calls
}

// GetCalls returns all recorded Get calls.
func (m *MockTransport) GetCalls() []GetCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]GetCall, len(m.getCalls))
	copy(calls, m.getCalls)
	return calls
}

// WasClosed returns true if Close was called.
func (m *MockTransport) WasClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

// Reset clears all recorded calls and responses.
func (m *MockTransport) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execCalls = nil
	m.putCalls = nil
	m.getCalls = nil
	m.execResponses = make(map[string]*ExecResponse)
	m.files = make(map[string]*MockFile)
	m.closeCalled = false
}

// GetFile returns the content and mode of a file in the mock filesystem.
func (m *MockTransport) GetFile(path string) ([]byte, fs.FileMode, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	file, exists := m.files[path]
	if !exists {
		return nil, 0, false
	}
	return file.Content, file.Mode, true
}

// AssertExecCalled is a helper for tests to verify a command was executed.
func (m *MockTransport) AssertExecCalled(t interface {
	Errorf(format string, args ...interface{})
}, expectedArgv []string,
) {
	calls := m.ExecCalls()
	for _, call := range calls {
		if len(call.Argv) != len(expectedArgv) {
			continue
		}
		match := true
		for i := range call.Argv {
			if call.Argv[i] != expectedArgv[i] {
				match = false
				break
			}
		}
		if match {
			return // Found it
		}
	}
	t.Errorf("Expected command %v was not executed. Calls: %v", expectedArgv, calls)
}

// AssertEnvSet is a helper to verify an environment variable was set.
func (m *MockTransport) AssertEnvSet(t interface {
	Errorf(format string, args ...interface{})
}, key, expectedValue string,
) {
	calls := m.ExecCalls()
	if len(calls) == 0 {
		t.Errorf("No exec calls recorded")
		return
	}
	// Check the last call
	lastCall := calls[len(calls)-1]
	actualValue, exists := lastCall.Env[key]
	if !exists {
		t.Errorf("Environment variable %s not set. Env: %v", key, lastCall.Env)
		return
	}
	if actualValue != expectedValue {
		t.Errorf("Environment variable %s = %q, expected %q", key, actualValue, expectedValue)
	}
}

// CaptureStdin returns the stdin content from the last Exec call.
func (m *MockTransport) CaptureStdin() string {
	calls := m.ExecCalls()
	if len(calls) == 0 {
		return ""
	}
	return calls[len(calls)-1].Stdin
}

// CaptureStdout returns the stdout content from the last Exec call.
func (m *MockTransport) CaptureStdout() string {
	calls := m.ExecCalls()
	if len(calls) == 0 {
		return ""
	}
	return calls[len(calls)-1].Stdout
}

// NewMockTransportWithEnv creates a mock transport with custom base environment.
func NewMockTransportWithEnv(env map[string]string) *MockTransport {
	mock := NewMockTransport()
	mock.SetBaseEnv(env)
	return mock
}
