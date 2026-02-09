package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/stretchr/testify/assert"
)

// TestContextCloneInheritsParent verifies that Clone inherits parent's Go context
func TestContextCloneInheritsParent(t *testing.T) {
	// Create parent with timeout
	parentCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	parent := newExecutionContext(map[string]interface{}{}, nil, parentCtx)

	// Clone with new args
	child := parent.Clone(map[string]interface{}{"foo": "bar"}, nil, nil)

	// Child should inherit parent's Go context (with timeout)
	assert.Equal(t, parentCtx, child.Context())
	assert.Equal(t, "bar", child.ArgString("foo"))
}

// TestContextCloneCancellation verifies that canceling parent cancels child
func TestContextCloneCancellation(t *testing.T) {
	// Create parent with cancellable context
	parentCtx, cancel := context.WithCancel(context.Background())

	parent := newExecutionContext(map[string]interface{}{}, nil, parentCtx)
	child := parent.Clone(map[string]interface{}{}, nil, nil)

	// Cancel parent
	cancel()

	// Child's context should be cancelled too
	select {
	case <-child.Context().Done():
		// Good - child context is cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Child context was not cancelled when parent was cancelled")
	}
}

// TestContextCloneWithPipes verifies Clone sets stdin/stdout correctly
func TestContextCloneWithPipes(t *testing.T) {
	parent := newExecutionContext(map[string]interface{}{}, nil, context.Background())

	stdin := strings.NewReader("test input")
	stdout := &strings.Builder{}

	child := parent.Clone(map[string]interface{}{}, stdin, stdout)

	assert.Equal(t, stdin, child.Stdin())
	assert.Equal(t, stdout, child.StdoutPipe())
}

// TestContextCloneWithoutPipes verifies Clone works with nil pipes
func TestContextCloneWithoutPipes(t *testing.T) {
	parent := newExecutionContext(map[string]interface{}{}, nil, context.Background())

	child := parent.Clone(map[string]interface{}{}, nil, nil)

	assert.Nil(t, child.Stdin())
	assert.Nil(t, child.StdoutPipe())
}

// TestContextCloneInheritsEnvironment verifies Clone inherits environment
func TestContextCloneInheritsEnvironment(t *testing.T) {
	parent := newExecutionContext(map[string]interface{}{}, nil, context.Background())
	parentWithEnv := parent.WithEnviron(map[string]string{"FOO": "bar"})

	child := parentWithEnv.Clone(map[string]interface{}{}, nil, nil)

	assert.Equal(t, "bar", child.Environ()["FOO"])
}

// TestContextCloneInheritsWorkdir verifies Clone inherits workdir
func TestContextCloneInheritsWorkdir(t *testing.T) {
	parent := newExecutionContext(map[string]interface{}{}, nil, context.Background())
	parentWithWd := parent.WithWorkdir("/tmp")

	child := parentWithWd.Clone(map[string]interface{}{}, nil, nil)

	assert.Equal(t, "/tmp", child.Workdir())
}

// TestContextWithMethodsPreservePipes verifies With* methods preserve pipes
func TestContextWithMethodsPreservePipes(t *testing.T) {
	stdin := strings.NewReader("test")
	stdout := &strings.Builder{}

	parent := newExecutionContext(map[string]interface{}{}, nil, context.Background())
	withPipes := parent.Clone(map[string]interface{}{}, stdin, stdout)

	// WithContext should preserve pipes
	newCtx := withPipes.WithContext(context.Background())
	assert.Equal(t, stdin, newCtx.Stdin())
	assert.Equal(t, stdout, newCtx.StdoutPipe())

	// WithEnviron should preserve pipes
	withEnv := withPipes.WithEnviron(map[string]string{"FOO": "bar"})
	assert.Equal(t, stdin, withEnv.Stdin())
	assert.Equal(t, stdout, withEnv.StdoutPipe())

	// WithWorkdir should preserve pipes
	withWd := withPipes.WithWorkdir("/tmp")
	assert.Equal(t, stdin, withWd.Stdin())
	assert.Equal(t, stdout, withWd.StdoutPipe())
}

func TestContextArgDurationParsesString(t *testing.T) {
	ctx := newExecutionContext(map[string]interface{}{"duration": "150ms"}, nil, context.Background())
	assert.Equal(t, 150*time.Millisecond, ctx.ArgDuration("duration"))
}

func TestContextArgDurationReturnsZeroForInvalid(t *testing.T) {
	ctx := newExecutionContext(map[string]interface{}{"duration": "invalid"}, nil, context.Background())
	assert.Equal(t, time.Duration(0), ctx.ArgDuration("duration"))
}

func TestContextTransportUsesTransportSession(t *testing.T) {
	exec := &executor{sessions: newSessionRuntime(func(transportID string) (decorator.Session, error) {
		base := decorator.NewLocalSession()
		if transportID == "local" {
			return base, nil
		}
		return &transportScopedSession{id: transportID, session: base}, nil
	})}

	ctx := newExecutionContext(map[string]interface{}{}, exec, context.Background())
	transportCtx := ctx.(*executionContext).withTransportID("transport:A")

	transport, ok := transportCtx.Transport().(*sessionTransport)
	if !ok {
		t.Fatalf("expected sessionTransport, got %T", transportCtx.Transport())
	}
	assert.Equal(t, "transport:A", transport.session.ID())
}
