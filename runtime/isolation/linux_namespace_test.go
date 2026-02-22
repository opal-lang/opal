//go:build linux

package isolation

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestLinuxNamespaceIsolator_NetworkIsolation(t *testing.T) {
	if os.Getenv("OPAL_ISOLATION_HELPER") == "network" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		isolator := NewLinuxNamespaceIsolator()
		if err := isolator.DropNetwork(); err != nil {
			if canSkipIsolationError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			t.Fatalf("drop network: %v", err)
		}

		conn, err := net.DialTimeout("tcp", "1.1.1.1:53", 250*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			t.Fatalf("network connection succeeded after isolation")
		}

		fmt.Print("OK:network-isolated\n")
		return
	}

	out := runHelper(t, "network", "^TestLinuxNamespaceIsolator_NetworkIsolation$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}

	if !strings.Contains(out, "OK:network-isolated") {
		t.Fatalf("expected helper success output, got:\n%s", out)
	}
}

func TestLinuxNamespaceIsolator_IsolationLevels(t *testing.T) {
	if os.Getenv("OPAL_ISOLATION_HELPER") == "levels" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		isolator := NewLinuxNamespaceIsolator()
		config := decorator.IsolationConfig{
			NetworkPolicy:    decorator.NetworkPolicyAllow,
			FilesystemPolicy: decorator.FilesystemPolicyFull,
			MemoryLock:       false,
		}

		levels := []decorator.IsolationLevel{
			decorator.IsolationLevelNone,
			decorator.IsolationLevelBasic,
			decorator.IsolationLevelStandard,
			decorator.IsolationLevelMaximum,
		}

		for _, level := range levels {
			if err := isolator.Isolate(level, config); err != nil {
				if canSkipIsolationError(err) {
					fmt.Printf("SKIP:%v\n", err)
					return
				}
				t.Fatalf("isolate level %d: %v", level, err)
			}
		}

		fmt.Print("OK:levels-applied\n")
		return
	}

	out := runHelper(t, "levels", "^TestLinuxNamespaceIsolator_IsolationLevels$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}

	if !strings.Contains(out, "OK:levels-applied") {
		t.Fatalf("expected helper success output, got:\n%s", out)
	}
}

func TestIsSupported(t *testing.T) {
	want := runtime.GOOS == "linux"
	if want {
		if _, err := os.Stat("/proc/self/ns"); err != nil {
			want = false
		}
	}

	got := IsSupported()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("IsSupported mismatch (-want +got):\n%s", diff)
	}
}

func runHelper(t *testing.T, mode, runExpr string) string {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run", runExpr)
	cmd.Env = append(os.Environ(), "OPAL_ISOLATION_HELPER="+mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "SKIP:") {
			return string(out)
		}
		t.Fatalf("helper failed: %v\noutput:\n%s", err, string(out))
	}

	return string(out)
}

func canSkipIsolationError(err error) bool {
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
