package executor

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveShellName_ExplicitShellTakesPrecedence(t *testing.T) {
	t.Parallel()

	resolved, err := resolveShellName("cmd", map[string]string{"OPAL_SHELL": "pwsh"})
	if err != nil {
		t.Fatalf("resolveShellName failed: %v", err)
	}

	if diff := cmp.Diff("cmd", resolved); diff != "" {
		t.Fatalf("resolved shell mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveShellName_FromEnvironment(t *testing.T) {
	t.Parallel()

	resolved, err := resolveShellName("", map[string]string{"OPAL_SHELL": "pwsh"})
	if err != nil {
		t.Fatalf("resolveShellName failed: %v", err)
	}

	if diff := cmp.Diff("pwsh", resolved); diff != "" {
		t.Fatalf("resolved shell mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveShellName_DefaultShell(t *testing.T) {
	t.Parallel()

	resolved, err := resolveShellName("", map[string]string{})
	if err != nil {
		t.Fatalf("resolveShellName failed: %v", err)
	}

	if diff := cmp.Diff("bash", resolved); diff != "" {
		t.Fatalf("resolved shell mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveShellName_InvalidEnvironmentValue(t *testing.T) {
	t.Parallel()

	_, err := resolveShellName("", map[string]string{"OPAL_SHELL": "fish"})
	if err == nil {
		t.Fatal("expected error")
	}

	want := `invalid OPAL_SHELL "fish": expected one of bash, pwsh, cmd`
	if diff := cmp.Diff(want, err.Error()); diff != "" {
		t.Fatalf("error mismatch (-want +got):\n%s", diff)
	}
}
