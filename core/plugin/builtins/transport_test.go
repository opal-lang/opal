package builtins

import (
	"context"
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

func TestTestTransportCapabilityOpen(t *testing.T) {
	capability := TestTransportCapability{}
	parent := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Env: map[string]string{"BASE": "1"}, Workdir: "/tmp", Platform: "linux"}, files: map[string][]byte{}}

	opened, err := capability.Open(context.Background(), parent, fakeArgs{strings: map[string]string{"name": "test"}})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer opened.Close()

	if diff := cmp.Diff("/tmp", opened.Snapshot().Workdir); diff != "" {
		t.Fatalf("Snapshot().Workdir mismatch (-want +got):\n%s", diff)
	}
}

func TestTestTransportIdempotentCapabilityOpen(t *testing.T) {
	capability := TestTransportIdempotentCapability{}
	parent := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Env: map[string]string{}, Workdir: "/w", Platform: "linux"}, files: map[string][]byte{}}

	opened, err := capability.Open(context.Background(), parent, fakeArgs{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer opened.Close()

	if diff := cmp.Diff("/w", opened.Snapshot().Workdir); diff != "" {
		t.Fatalf("Snapshot().Workdir mismatch (-want +got):\n%s", diff)
	}
}

func TestSandboxTransportCapabilityOpenFailsClosed(t *testing.T) {
	capability := SandboxTransportCapability{}
	parent := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Env: map[string]string{"BASE": "1"}, Workdir: "/tmp", Platform: "linux"}, files: map[string][]byte{}}

	_, err := capability.Open(context.Background(), parent, fakeArgs{strings: map[string]string{"level": "none", "network": "allow"}})
	if err == nil {
		t.Fatal("Open() error = nil, want sandbox fail-closed error")
	}
	if diff := cmp.Diff("sandbox transport is not available through plugin capabilities", err.Error()); diff != "" {
		t.Fatalf("Open() error mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedTransportCapabilitiesFailClosed(t *testing.T) {
	parent := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Env: map[string]string{}, Workdir: "/tmp", Platform: "linux"}, files: map[string][]byte{}}
	cases := []plugin.Transport{
		IsolatedNetworkLoopbackCapability{},
		IsolatedFilesystemReadonlyCapability{},
		IsolatedFilesystemEphemeralCapability{},
		IsolatedMemoryLockCapability{},
		IsolatedPrivilegesDropCapability{},
	}

	for _, capability := range cases {
		_, err := capability.Open(context.Background(), parent, fakeArgs{})
		if err == nil {
			t.Fatal("Open() error = nil, want isolated fail-closed error")
		}
		if diff := cmp.Diff("isolated transport is not available through plugin capabilities", err.Error()); diff != "" {
			t.Fatalf("Open() error mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestIsolatedPrivilegesDropCapabilityIsRegistered(t *testing.T) {
	if plugin.Global().Lookup("isolated.privileges.drop") == nil {
		t.Fatal("isolated.privileges.drop should be registered in plugin registry")
	}
}
