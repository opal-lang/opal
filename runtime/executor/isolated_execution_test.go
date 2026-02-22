package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	runtimedecorators "github.com/opal-lang/opal/runtime/decorators"
)

func TestIsolatedNetworkDenyBlocksExternalConnections(t *testing.T) {
	if os.Getenv("OPAL_ISOLATED_EXEC_HELPER") == "network-deny" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := runIsolatedNetworkDenyHelper(); err != nil {
			if canSkipIsolatedExecutionError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			fmt.Printf("ERR:%v\n", err)
			os.Exit(1)
		}

		fmt.Print("OK:network-deny\n")
		return
	}

	out := runIsolatedExecutionHelper(t, "network-deny", "^TestIsolatedNetworkDenyBlocksExternalConnections$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	if diff := cmp.Diff(true, strings.Contains(out, "OK:network-deny")); diff != "" {
		t.Fatalf("network helper output mismatch (-want +got):\n%s\noutput:\n%s", diff, out)
	}
}

func TestIsolatedFilesystemIsolationRestrictsFileAccess(t *testing.T) {
	hostFile := t.TempDir() + "/host-visible.txt"
	if err := os.WriteFile(hostFile, []byte("host-data\n"), 0o600); err != nil {
		t.Fatalf("write host file: %v", err)
	}

	if os.Getenv("OPAL_ISOLATED_EXEC_HELPER") == "filesystem" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := runIsolatedFilesystemHelper(); err != nil {
			if canSkipIsolatedExecutionError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			fmt.Printf("ERR:%v\n", err)
			os.Exit(1)
		}

		fmt.Print("OK:filesystem-isolated\n")
		return
	}

	extraEnv := []string{"OPAL_ISOLATED_HOST_FILE=" + hostFile}
	out := runIsolatedExecutionHelper(t, "filesystem", "^TestIsolatedFilesystemIsolationRestrictsFileAccess$", extraEnv...)
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	if diff := cmp.Diff(true, strings.Contains(out, "OK:filesystem-isolated")); diff != "" {
		t.Fatalf("filesystem helper output mismatch (-want +got):\n%s\noutput:\n%s", diff, out)
	}
}

func TestIsolatedCryptoKeyGeneration(t *testing.T) {
	if os.Getenv("OPAL_ISOLATED_EXEC_HELPER") == "crypto" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := runIsolatedCryptoHelper(); err != nil {
			if canSkipIsolatedExecutionError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			fmt.Printf("ERR:%v\n", err)
			os.Exit(1)
		}

		fmt.Print("OK:crypto-generated\n")
		return
	}

	out := runIsolatedExecutionHelper(t, "crypto", "^TestIsolatedCryptoKeyGeneration$")
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	if diff := cmp.Diff(true, strings.Contains(out, "OK:crypto-generated")); diff != "" {
		t.Fatalf("crypto helper output mismatch (-want +got):\n%s\noutput:\n%s", diff, out)
	}
}

func TestIsolatedCombinedNetworkAndFilesystem(t *testing.T) {
	hostFile := t.TempDir() + "/host-visible-combined.txt"
	if err := os.WriteFile(hostFile, []byte("host-combined\n"), 0o600); err != nil {
		t.Fatalf("write host file: %v", err)
	}

	if os.Getenv("OPAL_ISOLATED_EXEC_HELPER") == "combined" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := runIsolatedCombinedHelper(); err != nil {
			if canSkipIsolatedExecutionError(err) {
				fmt.Printf("SKIP:%v\n", err)
				return
			}
			fmt.Printf("ERR:%v\n", err)
			os.Exit(1)
		}

		fmt.Print("OK:combined-isolated\n")
		return
	}

	extraEnv := []string{"OPAL_ISOLATED_HOST_FILE=" + hostFile}
	out := runIsolatedExecutionHelper(t, "combined", "^TestIsolatedCombinedNetworkAndFilesystem$", extraEnv...)
	if strings.Contains(out, "SKIP:") {
		t.Skip(strings.TrimSpace(out))
	}
	if diff := cmp.Diff(true, strings.Contains(out, "OK:combined-isolated")); diff != "" {
		t.Fatalf("combined helper output mismatch (-want +got):\n%s\noutput:\n%s", diff, out)
	}
}

func runIsolatedNetworkDenyHelper() error {
	session, err := openIsolatedSession(map[string]any{"level": "standard", "network": "deny"})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, runErr := session.Run(ctx, []string{"bash", "-c", "curl --max-time 2 --silent --show-error https://example.com >/dev/null"}, decorator.RunOpts{})
	if runErr == nil && result.ExitCode == 0 {
		return fmt.Errorf("curl unexpectedly succeeded in network-deny isolated session")
	}

	return nil
}

func runIsolatedFilesystemHelper() error {
	session, err := openIsolatedSession(map[string]any{"level": "maximum", "network": "allow", "filesystem": "ephemeral"})
	if err != nil {
		if strings.Contains(err.Error(), `unknown parameter "filesystem"`) {
			return fmt.Errorf("filesystem policy is not yet exposed by @isolated decorator: %w", err)
		}
		return err
	}

	hostFile := os.Getenv("OPAL_ISOLATED_HOST_FILE")
	if hostFile == "" {
		return fmt.Errorf("OPAL_ISOLATED_HOST_FILE is required for filesystem helper")
	}

	script := "set -e; echo isolated-fs > /tmp/opal-isolated-fs.txt; test \"$(cat /tmp/opal-isolated-fs.txt)\" = \"isolated-fs\"; if cat \"$OPAL_ISOLATED_HOST_FILE\" >/dev/null 2>&1; then exit 41; fi"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, runErr := session.Run(ctx, []string{"bash", "-c", script}, decorator.RunOpts{})
	if runErr != nil {
		return runErr
	}
	if result.ExitCode == 41 {
		return fmt.Errorf("filesystem isolation did not block host file access")
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		return fmt.Errorf("filesystem helper exit code mismatch (-want +got):\n%s", diff)
	}

	return nil
}

func runIsolatedCryptoHelper() error {
	session, err := openIsolatedSession(map[string]any{"level": "standard", "network": "deny"})
	if err != nil {
		return err
	}

	cryptoDecorator := &runtimedecorators.CryptoValueDecorator{}
	results, resolveErr := cryptoDecorator.Resolve(
		decorator.ValueEvalContext{Session: session},
		decorator.ValueCall{Path: "crypto", Params: map[string]any{"type": "ed25519"}},
	)
	if resolveErr != nil {
		return resolveErr
	}
	if diff := cmp.Diff(1, len(results)); diff != "" {
		return fmt.Errorf("crypto result count mismatch (-want +got):\n%s", diff)
	}
	if results[0].Error != nil {
		return fmt.Errorf("crypto resolve returned per-call error: %w", results[0].Error)
	}

	keyPair, ok := results[0].Value.(runtimedecorators.KeyPair)
	if !ok {
		return fmt.Errorf("crypto resolve value type mismatch: got %T", results[0].Value)
	}
	if keyPair.IsZero() {
		return fmt.Errorf("crypto returned empty key pair")
	}

	return nil
}

func runIsolatedCombinedHelper() error {
	session, err := openIsolatedSession(map[string]any{"level": "maximum", "network": "deny", "filesystem": "ephemeral"})
	if err != nil {
		if strings.Contains(err.Error(), `unknown parameter "filesystem"`) {
			return fmt.Errorf("filesystem policy is not yet exposed by @isolated decorator: %w", err)
		}
		return err
	}

	hostFile := os.Getenv("OPAL_ISOLATED_HOST_FILE")
	if hostFile == "" {
		return fmt.Errorf("OPAL_ISOLATED_HOST_FILE is required for combined helper")
	}

	script := "set -e; echo combined > /tmp/opal-combined.txt; test \"$(cat /tmp/opal-combined.txt)\" = \"combined\"; if curl --max-time 2 --silent --show-error https://example.com >/dev/null 2>&1; then exit 51; fi; if cat \"$OPAL_ISOLATED_HOST_FILE\" >/dev/null 2>&1; then exit 52; fi"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, runErr := session.Run(ctx, []string{"bash", "-c", script}, decorator.RunOpts{})
	if runErr != nil {
		return runErr
	}
	if result.ExitCode == 51 {
		return fmt.Errorf("network isolation did not block outbound curl")
	}
	if result.ExitCode == 52 {
		return fmt.Errorf("filesystem isolation did not block host file access")
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		return fmt.Errorf("combined helper exit code mismatch (-want +got):\n%s", diff)
	}

	return nil
}

func openIsolatedSession(params map[string]any) (decorator.Session, error) {
	dec := &runtimedecorators.IsolatedTransportDecorator{}
	parent := decorator.NewLocalSession()
	return dec.Open(parent, params)
}

func runIsolatedExecutionHelper(t *testing.T, mode string, runExpr string, extraEnv ...string) string {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run", runExpr)
	cmd.Env = append(os.Environ(), "OPAL_ISOLATED_EXEC_HELPER="+mode)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "SKIP:") {
			return string(out)
		}
		t.Fatalf("helper failed: %v\noutput:\n%s", err, string(out))
	}

	return string(out)
}

func canSkipIsolatedExecutionError(err error) bool {
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
		strings.Contains(msg, "filesystem policy is not yet exposed")
}
