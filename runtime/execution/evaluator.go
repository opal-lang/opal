package execution

import (
	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/runtime/execution/context"
)

// ================================================================================================
// NODE EVALUATOR - Stubbed for Future Implementation
// ================================================================================================

// NodeEvaluator executes IR trees using registered decorators
// Currently stubbed - runtime execution will be implemented in future phases
type NodeEvaluator struct {
	registry *decorators.Registry
}

// NewNodeEvaluator creates a new node evaluator with the given registry
func NewNodeEvaluator(registry *decorators.Registry) *NodeEvaluator {
	return &NodeEvaluator{
		registry: registry,
	}
}

// ExecuteNode executes an IR node - currently stubbed
func (e *NodeEvaluator) ExecuteNode(ctx *context.Ctx, node interface{}) context.CommandResult {
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "runtime execution not implemented yet - use plan mode",
		ExitCode: 1,
	}
}
