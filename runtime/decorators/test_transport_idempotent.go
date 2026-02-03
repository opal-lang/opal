package decorators

import (
	"fmt"

	"github.com/opal-lang/opal/core/decorator"
)

type idempotentTestTransport struct {
	base *decorator.TestTransport
}

func (t *idempotentTestTransport) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.transport.idempotent").
		Summary("Idempotent test transport for planning").
		Roles(decorator.RoleBoundary).
		Block(decorator.BlockRequired).
		Idempotent().
		Build()
}

func (t *idempotentTestTransport) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	return t.base.Open(parent, params)
}

func (t *idempotentTestTransport) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return t.base.Wrap(next, params)
}

func init() {
	if err := decorator.Register("test.transport.idempotent", &idempotentTestTransport{
		base: decorator.NewTestTransport("test-idempotent"),
	}); err != nil {
		panic(fmt.Sprintf("failed to register @test.transport.idempotent decorator: %v", err))
	}
}
