package builtins

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestVarValueCapabilityResolveString(t *testing.T) {
	capability := VarValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}, vars: map[string]any{"NAME": "sigil"}}
	args := fakeArgs{strings: map[string]string{"name": "NAME"}}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff("sigil", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestVarValueCapabilityResolveInt(t *testing.T) {
	capability := VarValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}, vars: map[string]any{"COUNT": 3}}
	args := fakeArgs{strings: map[string]string{"name": "COUNT"}}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff(any(3), got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestVarValueCapabilityResolveMissing(t *testing.T) {
	capability := VarValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}, vars: map[string]any{}}
	args := fakeArgs{strings: map[string]string{"name": "MISSING"}}

	_, err := capability.Resolve(ctx, args)
	if err == nil {
		t.Fatal("Resolve() error = nil, want missing variable error")
	}
}
