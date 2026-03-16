package decorator_test

import (
	"testing"

	coredecorator "github.com/builtwithtofu/sigil/core/decorator"
	"github.com/google/go-cmp/cmp"
)

func TestSSHTransportRegistration(t *testing.T) {
	entry, ok := coredecorator.Global().Lookup("ssh.connect")
	if !ok {
		t.Fatalf("ssh.connect transport not found in registry")
	}

	transport, typeOK := coredecorator.Global().GetTransport("ssh.connect")
	if typeOK {
		_, typeOK = transport.(*coredecorator.SSHTransport)
	}
	if diff := cmp.Diff(true, typeOK); diff != "" {
		t.Fatalf("registered ssh.connect transport type mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff("ssh.connect", entry.Impl.Descriptor().Path); diff != "" {
		t.Fatalf("descriptor path mismatch (-want +got):\n%s", diff)
	}
}
