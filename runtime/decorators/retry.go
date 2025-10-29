package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/types"
)

// RetryDecorator implements the @retry execution decorator.
// Retries failed operations with configurable backoff strategy.
type RetryDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *RetryDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("retry").
		Summary("Retry failed operations with exponential backoff").
		Roles(decorator.RoleWrapper).
		Param("times", types.TypeInt, "Number of retry attempts", "3", "5", "10").
		Param("delay", types.TypeDuration, "Initial delay between retries", "1s", "5s", "30s").
		Param("backoff", types.TypeString, "Backoff strategy", "exponential", "linear", "constant").
		Block(decorator.BlockOptional).
		Build()
}

// Wrap implements the Exec interface.
func (d *RetryDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &retryNode{next: next, params: params}
}

// retryNode wraps an execution node with retry logic.
type retryNode struct {
	next   decorator.ExecNode
	params map[string]any
}

// Execute implements the ExecNode interface.
// Stub implementation: just executes once for now.
func (n *retryNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	// TODO: Implement actual retry logic with backoff
	return n.next.Execute(ctx)
}

// Register @retry decorator with the global registry
func init() {
	if err := decorator.Register("retry", &RetryDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @retry decorator: %v", err))
	}
}
