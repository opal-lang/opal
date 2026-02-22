package decorators

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/isolation"
)

func TestIsolatedDecoratorRegistered(t *testing.T) {
	if !decorator.Global().IsRegistered("isolated") {
		t.Fatal("built-in decorator 'isolated' should be registered")
	}
}

func TestIsolatedTransportCapabilities(t *testing.T) {
	dec := &IsolatedTransportDecorator{}
	got := dec.Capabilities()

	if !got.Has(decorator.TransportCapIsolation) {
		t.Fatal("expected isolation capability")
	}

	if !got.Has(decorator.TransportCapNetwork) {
		t.Fatal("expected network capability")
	}
}

func TestIsolatedIsolationContextUsesFactoryType(t *testing.T) {
	dec := &IsolatedTransportDecorator{}
	ctx := dec.IsolationContext()
	if ctx == nil {
		t.Fatal("expected non-nil isolation context")
	}

	if diff := cmp.Diff(fmt.Sprintf("%T", isolation.NewIsolator()), fmt.Sprintf("%T", ctx)); diff != "" {
		t.Fatalf("isolation context type mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedOpenLevelNoneCreatesIsolatedSession(t *testing.T) {
	dec := &IsolatedTransportDecorator{}
	parent := decorator.NewLocalSession()

	session, err := dec.Open(parent, map[string]any{"level": "none"})
	if err != nil {
		t.Fatalf("open isolated session: %v", err)
	}

	if diff := cmp.Diff(decorator.TransportScopeIsolated, session.TransportScope()); diff != "" {
		t.Fatalf("transport scope mismatch (-want +got):\n%s", diff)
	}

	if !strings.HasSuffix(session.ID(), "/isolated") {
		t.Fatalf("expected isolated session id suffix, got %q", session.ID())
	}

	result, runErr := session.Run(context.Background(), []string{"sh", "-c", "printf isolated-ok"}, decorator.RunOpts{})
	if runErr != nil {
		t.Fatalf("run through isolated session: %v", runErr)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("isolated-ok", string(result.Stdout)); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedOpenStandardAppliesIsolation(t *testing.T) {
	if os.Getenv("OPAL_ISOLATED_HELPER") == "standard" {
		dec := &IsolatedTransportDecorator{}
		parent := decorator.NewLocalSession()

		_, err := dec.Open(parent, map[string]any{"level": "standard", "network": "allow"})
		if err != nil {
			if canSkipIsolatedError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			fmt.Printf("ERR:%v\n", err)
			os.Exit(1)
		}

		fmt.Print("OK:isolated-standard-applied\n")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestIsolatedOpenStandardAppliesIsolation$")
	cmd.Env = append(os.Environ(), "OPAL_ISOLATED_HELPER=standard")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "SKIP:") {
			t.Skip(strings.TrimSpace(string(out)))
		}
		t.Fatalf("helper failed: %v\noutput:\n%s", err, string(out))
	}

	output := string(out)
	if strings.Contains(output, "SKIP:") {
		t.Skip(strings.TrimSpace(output))
	}
	if !strings.Contains(output, "OK:isolated-standard-applied") {
		t.Fatalf("expected helper success output, got:\n%s", output)
	}
}

func canSkipIsolatedError(err error) bool {
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
		strings.Contains(msg, "read-only file system")
}
