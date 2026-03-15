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

func TestSandboxTransportCapabilityOpen(t *testing.T) {
	capability := SandboxTransportCapability{}
	parent := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Env: map[string]string{"BASE": "1"}, Workdir: "/tmp", Platform: "linux"}, files: map[string][]byte{}}

	opened, err := capability.Open(context.Background(), parent, fakeArgs{strings: map[string]string{"level": "none", "network": "allow"}})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer opened.Close()

	if diff := cmp.Diff("/tmp", opened.Snapshot().Workdir); diff != "" {
		t.Fatalf("Snapshot().Workdir mismatch (-want +got):\n%s", diff)
	}
}

func TestIsolatedTransportCapabilitiesOpen(t *testing.T) {
	parent := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Env: map[string]string{}, Workdir: "/tmp", Platform: "linux"}, files: map[string][]byte{}}
	cases := []plugin.Transport{
		IsolatedNetworkLoopbackCapability{},
		IsolatedFilesystemReadonlyCapability{},
		IsolatedFilesystemEphemeralCapability{},
		IsolatedMemoryLockCapability{},
	}

	for _, capability := range cases {
		opened, err := capability.Open(context.Background(), parent, fakeArgs{})
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		_ = opened.Close()
	}
}
