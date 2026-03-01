package decorator_test

import (
	"testing"

	coredecorator "github.com/builtwithtofu/sigil/core/decorator"
	"github.com/google/go-cmp/cmp"
)

func TestSSHTransportRegistration(t *testing.T) {
	entry, ok := coredecorator.Global().Lookup("ssh")
	if !ok {
		t.Fatalf("ssh transport not found in registry")
	}

	_, typeOK := entry.Impl.(*coredecorator.SSHTransport)
	if diff := cmp.Diff(true, typeOK); diff != "" {
		t.Fatalf("registered ssh transport type mismatch (-want +got):\n%s", diff)
	}
}
