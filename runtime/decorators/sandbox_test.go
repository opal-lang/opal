package decorators

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/runtime/isolation"
	"github.com/google/go-cmp/cmp"
)

func TestSandboxDecoratorRegistered(t *testing.T) {
	if decorator.Global().IsRegistered("sandbox") {
		t.Error("decorator 'sandbox' should NOT be registered - it's a namespace now")
	}
}

func TestSandboxTransportCapabilities(t *testing.T) {
	dec := &SandboxTransportDecorator{}
	got := dec.Capabilities()

	if !got.Has(decorator.TransportCapIsolation) {
		t.Fatal("expected isolation capability")
	}

	if !got.Has(decorator.TransportCapNetwork) {
		t.Fatal("expected network capability")
	}
}

func TestSandboxIsolationContextUsesFactoryType(t *testing.T) {
	dec := &SandboxTransportDecorator{}
	ctx := dec.IsolationContext()
	if ctx == nil {
		t.Fatal("expected non-nil isolation context")
	}

	if diff := cmp.Diff(fmt.Sprintf("%T", isolation.NewIsolator()), fmt.Sprintf("%T", ctx)); diff != "" {
		t.Fatalf("isolation context type mismatch (-want +got):\n%s", diff)
	}
}

func TestSandboxOpenLevelNoneCreatesSessionAndRunsCommand(t *testing.T) {
	dec := &SandboxTransportDecorator{}
	parent := decorator.NewLocalSession()

	session, err := dec.Open(parent, map[string]any{"level": "none"})
	if err != nil {
		t.Fatalf("open sandbox session: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	if diff := cmp.Diff(decorator.TransportScopeIsolated, session.TransportScope()); diff != "" {
		t.Fatalf("transport scope mismatch (-want +got):\n%s", diff)
	}

	if !strings.HasSuffix(session.ID(), "/sandbox") {
		t.Fatalf("expected sandbox session id suffix, got %q", session.ID())
	}

	result, runErr := session.Run(context.Background(), []string{"sh", "-c", "printf sandbox-ok"}, decorator.RunOpts{})
	if runErr != nil {
		t.Fatalf("run through sandbox session: %v", runErr)
	}

	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("sandbox-ok", string(result.Stdout)); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestSandboxWithEnvAndWorkdirApplyToRun(t *testing.T) {
	dec := &SandboxTransportDecorator{}
	parent := decorator.NewLocalSession()

	session, err := dec.Open(parent, map[string]any{"level": "none"})
	if err != nil {
		t.Fatalf("open sandbox session: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	dir := t.TempDir()
	derived := session.WithEnv(map[string]string{"OPAL_SANDBOX_TEST": "sandbox-env-ok"}).WithWorkdir(dir)

	result, runErr := derived.Run(context.Background(), []string{"sh", "-c", "printf '%s|%s' \"$OPAL_SANDBOX_TEST\" \"$(pwd)\""}, decorator.RunOpts{})
	if runErr != nil {
		t.Fatalf("run through derived sandbox session: %v", runErr)
	}

	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	want := "sandbox-env-ok|" + dir
	if diff := cmp.Diff(want, string(result.Stdout)); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestSandboxOpenStandardAppliesIsolation(t *testing.T) {
	dec := &SandboxTransportDecorator{}
	parent := decorator.NewLocalSession()

	session, err := dec.Open(parent, map[string]any{"level": "standard", "network": "allow"})
	if err != nil {
		if canSkipSandboxError(err) {
			t.Skipf("sandbox isolation not available: %v", err)
		}
		t.Fatalf("open sandbox session at standard level: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	result, runErr := session.Run(context.Background(), []string{"sh", "-c", "printf sandbox-standard-ok"}, decorator.RunOpts{})
	if runErr != nil {
		t.Fatalf("run in standard sandbox session: %v", runErr)
	}

	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("sandbox-standard-ok", string(result.Stdout)); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func canSkipSandboxError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.ENOSYS) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "operation not permitted") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "read-only file system") ||
		strings.Contains(msg, "sandbox helper failed")
}

func TestSandboxHelperEnvironmentNotSetInParent(t *testing.T) {
	if os.Getenv(sandboxHelperEnv) != "" {
		t.Fatalf("expected %s to be unset in parent test process", sandboxHelperEnv)
	}
}
