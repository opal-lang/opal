//go:build linux

package isolation

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkIsolation(t *testing.T) {
	if !IsSupported() {
		t.Skip("Linux namespaces not supported")
	}

	if os.Getenv("OPAL_ISOLATION_VERIFICATION_HELPER") == "network" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		isolator := NewLinuxNamespaceIsolator()
		err := isolator.DropNetwork()
		if err != nil {
			if canSkipIsolationError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			t.Fatalf("drop network: %v", err)
		}

		_, err = net.DialTimeout("tcp", "8.8.8.8:53", 500*time.Millisecond)
		assert.Error(t, err)
		fmt.Print("OK:network-isolated\n")
		return
	}

	out := runIsolationVerificationHelper(t, "network", "^TestNetworkIsolation$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	require.Contains(t, out, "OK:network-isolated")
}

func TestFilesystemIsolation(t *testing.T) {
	if !IsSupported() {
		t.Skip("Linux namespaces not supported")
	}

	if os.Getenv("OPAL_ISOLATION_VERIFICATION_HELPER") == "filesystem" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		isolator := NewLinuxNamespaceIsolator()
		err := isolator.RestrictFilesystem([]string{"/"}, nil)
		if err != nil {
			if canSkipIsolationError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			t.Fatalf("restrict filesystem: %v", err)
		}

		testFile := "/tmp/opal-isolation-fs-test"
		err = os.WriteFile(testFile, []byte("blocked"), 0o600)
		assert.Error(t, err)
		fmt.Print("OK:filesystem-isolated\n")
		return
	}

	out := runIsolationVerificationHelper(t, "filesystem", "^TestFilesystemIsolation$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	require.Contains(t, out, "OK:filesystem-isolated")
}

func TestProcessIsolation(t *testing.T) {
	if !IsSupported() {
		t.Skip("Linux namespaces not supported")
	}

	if os.Getenv("OPAL_ISOLATION_VERIFICATION_HELPER") == "process" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		isolator := NewLinuxNamespaceIsolator()
		err := isolator.Isolate(decorator.IsolationLevelStandard, decorator.IsolationConfig{})
		if err != nil {
			if canSkipIsolationError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			t.Fatalf("isolate process namespace: %v", err)
		}

		cmd := exec.Command("sh", "-c", "echo $$")
		pidBytes, err := cmd.Output()
		require.NoError(t, err)

		pidText := strings.TrimSpace(string(pidBytes))
		pid, err := strconv.Atoi(pidText)
		require.NoError(t, err)
		assert.Equal(t, 1, pid)
		fmt.Print("OK:process-isolated\n")
		return
	}

	out := runIsolationVerificationHelper(t, "process", "^TestProcessIsolation$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	require.Contains(t, out, "OK:process-isolated")
}

func TestMemoryLock(t *testing.T) {
	if !IsSupported() {
		t.Skip("Linux namespaces not supported")
	}

	if os.Getenv("OPAL_ISOLATION_VERIFICATION_HELPER") == "memory" {
		isolator := NewLinuxNamespaceIsolator()
		err := isolator.LockMemory()
		if err != nil {
			if canSkipMemoryLockError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			t.Fatalf("lock memory: %v", err)
		}

		assert.NoError(t, syscall.Munlockall())
		fmt.Print("OK:memory-locked\n")
		return
	}

	out := runIsolationVerificationHelper(t, "memory", "^TestMemoryLock$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	require.Contains(t, out, "OK:memory-locked")
}

func TestCryptoIsolation(t *testing.T) {
	if !IsSupported() {
		t.Skip("Linux namespaces not supported")
	}

	if os.Getenv("OPAL_ISOLATION_VERIFICATION_HELPER") == "crypto" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		isolator := NewLinuxNamespaceIsolator()
		err := isolator.Isolate(decorator.IsolationLevelStandard, decorator.IsolationConfig{
			NetworkPolicy:    decorator.NetworkPolicyDeny,
			FilesystemPolicy: decorator.FilesystemPolicyFull,
			MemoryLock:       false,
		})
		if err != nil {
			if canSkipIsolationError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			t.Fatalf("isolate for crypto: %v", err)
		}

		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		require.Len(t, pub, ed25519.PublicKeySize)
		require.Len(t, priv, ed25519.PrivateKeySize)
		assert.NotEmpty(t, pub)
		assert.NotEmpty(t, priv)
		fmt.Print("OK:crypto-generated\n")
		return
	}

	out := runIsolationVerificationHelper(t, "crypto", "^TestCryptoIsolation$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	require.Contains(t, out, "OK:crypto-generated")
}

func runIsolationVerificationHelper(t *testing.T, mode, runExpr string) string {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run", runExpr)
	cmd.Env = append(os.Environ(), "OPAL_ISOLATION_VERIFICATION_HELPER="+mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "SKIP:") {
			return string(out)
		}
		t.Fatalf("helper failed: %v\noutput:\n%s", err, string(out))
	}

	return string(out)
}

func canSkipMemoryLockError(err error) bool {
	if err == nil {
		return false
	}

	if canSkipIsolationError(err) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "cannot allocate memory") ||
		strings.Contains(strings.ToLower(err.Error()), "resource temporarily unavailable")
}
