package decorator_test

import (
	"testing"

	coredecorator "github.com/builtwithtofu/sigil/core/decorator"
)

func TestSSHTransportRegistration(t *testing.T) {
	if _, ok := coredecorator.Global().Lookup("ssh.connect"); ok {
		t.Fatalf("ssh.connect should not be in legacy registry")
	}
}
