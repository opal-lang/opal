package builtins

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOSGetValueCapabilityResolve(t *testing.T) {
	capability := OSGetValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}, platform: "linux"}}
	args := fakeArgs{}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff("linux", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestOSLinuxValueCapabilityResolve(t *testing.T) {
	capability := OSLinuxValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}, platform: "linux"}}
	args := fakeArgs{}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff("true", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestOSMacOSValueCapabilityResolve(t *testing.T) {
	capability := OSMacOSValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}, platform: "darwin"}}
	args := fakeArgs{}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff("true", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestOSWindowsValueCapabilityResolve(t *testing.T) {
	capability := OSWindowsValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}, platform: "windows"}}
	args := fakeArgs{}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff("true", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}
