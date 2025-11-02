package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorator"
)

// TimeoutDecorator implements the @timeout execution decorator.
// Executes block with a timeout constraint.
type TimeoutDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *TimeoutDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("timeout").
		Summary("Execute block with timeout constraint").
		Roles(decorator.RoleWrapper).
		ParamDuration("duration", "Maximum execution time").
		Required().
		Examples("30s", "5m", "1h").
		Done().
		Block(decorator.BlockRequired).
		Build()
}

// Wrap implements the Exec interface.
func (d *TimeoutDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &timeoutNode{next: next, params: params}
}

// timeoutNode wraps an execution node with timeout logic.
type timeoutNode struct {
	next   decorator.ExecNode
	params map[string]any
}

// Execute implements the ExecNode interface.
// Stub implementation: just executes without timeout for now.
func (n *timeoutNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	// TODO: Implement actual timeout logic with context cancellation
	return n.next.Execute(ctx)
}

// Register @timeout decorator with the global registry
func init() {
	if err := decorator.Register("timeout", &TimeoutDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @timeout decorator: %v", err))
	}
}
