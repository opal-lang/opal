package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/types"
)

// ParallelDecorator implements the @parallel execution decorator.
// Executes tasks in parallel with optional concurrency limit.
type ParallelDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *ParallelDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("parallel").
		Summary("Execute tasks in parallel").
		Roles(decorator.RoleWrapper).
		Param("maxConcurrency", types.TypeInt, "Maximum concurrent tasks (0=unlimited)", "0", "5", "10").
		Block(decorator.BlockRequired).
		Build()
}

// Wrap implements the Exec interface.
func (d *ParallelDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &parallelNode{next: next, params: params}
}

// parallelNode wraps an execution node with parallel execution logic.
type parallelNode struct {
	next   decorator.ExecNode
	params map[string]any
}

// Execute implements the ExecNode interface.
// Stub implementation: just executes sequentially for now.
func (n *parallelNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	// TODO: Implement actual parallel execution logic
	return n.next.Execute(ctx)
}

// Register @parallel decorator with the global registry
func init() {
	if err := decorator.Register("parallel", &ParallelDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @parallel decorator: %v", err))
	}
}
