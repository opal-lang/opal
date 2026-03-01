package executor

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/google/go-cmp/cmp"
)

type materializingTransportExecDecorator struct{}

func (d *materializingTransportExecDecorator) Descriptor() decorator.Descriptor {
	return decorator.Descriptor{Path: "test.transport.materializing"}
}

func (d *materializingTransportExecDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return next
}

func (d *materializingTransportExecDecorator) Capabilities() decorator.TransportCaps {
	return decorator.TransportCapNetwork
}

func (d *materializingTransportExecDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	return parent, nil
}

func (d *materializingTransportExecDecorator) MaterializeSession() bool {
	return true
}

func (d *materializingTransportExecDecorator) IsolationContext() decorator.IsolationContext {
	return nil
}

func TestTransportIDForPlanDecoratorExecution_BlockWithoutInferrableSourceFallsBackToCurrentTransport(t *testing.T) {
	t.Parallel()

	execCtx := &executionContext{transportID: "transport:current"}
	cmd := &planfmt.CommandNode{
		Block: []planfmt.Step{{
			ID:   1,
			Tree: &planfmt.LogicNode{},
		}},
	}

	got := transportIDForPlanDecoratorExecution(execCtx, cmd, &materializingTransportExecDecorator{})
	if diff := cmp.Diff("transport:current", got); diff != "" {
		t.Fatalf("transport fallback mismatch (-want +got):\n%s", diff)
	}
}
