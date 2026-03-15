package builtins

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCryptoValueCapabilitySHA256(t *testing.T) {
	capability := CryptoValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}}
	args := fakeArgs{strings: map[string]string{"method": "SHA256", "arg0": "hello"}}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	const expected = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestCryptoValueCapabilityRejectsUnsupportedMethod(t *testing.T) {
	capability := CryptoValueCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}}
	args := fakeArgs{strings: map[string]string{"method": "MD5", "arg0": "hello"}}

	_, err := capability.Resolve(ctx, args)
	if err == nil {
		t.Fatal("Resolve() error = nil, want unsupported method error")
	}
}
