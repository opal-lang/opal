package builtins

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSecretsValueCapabilityPutThenGet(t *testing.T) {
	capability := NewSecretsValueCapability()
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}, planKey: "plan-key"}

	putValue, err := capability.Resolve(ctx, fakeArgs{strings: map[string]string{"method": "put", "path": "keys/deploy", "value": "secret-value"}})
	if err != nil {
		t.Fatalf("Resolve(put) error = %v", err)
	}
	if _, ok := putValue.(string); !ok {
		t.Fatalf("Resolve(put) type = %T, want string", putValue)
	}

	getValue, err := capability.Resolve(ctx, fakeArgs{strings: map[string]string{"method": "get", "path": "keys/deploy"}})
	if err != nil {
		t.Fatalf("Resolve(get) error = %v", err)
	}
	if diff := cmp.Diff([]byte("secret-value"), getValue); diff != "" {
		t.Fatalf("Resolve(get) mismatch (-want +got):\n%s", diff)
	}
}

func TestSecretsValueCapabilityPlanHashIsolation(t *testing.T) {
	capability := NewSecretsValueCapability()
	ctxA := fakeValueContext{session: &fakeSession{env: map[string]string{}}, planKey: "plan-a"}
	ctxB := fakeValueContext{session: &fakeSession{env: map[string]string{}}, planKey: "plan-b"}

	_, err := capability.Resolve(ctxA, fakeArgs{strings: map[string]string{"method": "put", "path": "keys/deploy", "value": "secret-a"}})
	if err != nil {
		t.Fatalf("Resolve(put) error = %v", err)
	}

	_, err = capability.Resolve(ctxB, fakeArgs{strings: map[string]string{"method": "get", "path": "keys/deploy"}})
	if err == nil {
		t.Fatal("Resolve(get) error = nil, want isolation error")
	}
}

func TestSecretsValueCapabilityRejectsUnsupportedMethod(t *testing.T) {
	capability := NewSecretsValueCapability()
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}, planKey: "plan-key"}

	_, err := capability.Resolve(ctx, fakeArgs{strings: map[string]string{"method": "delete", "path": "keys/deploy"}})
	if err == nil {
		t.Fatal("Resolve() error = nil, want unsupported method error")
	}
}
