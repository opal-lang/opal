package execution

import (
	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/ir"
	"github.com/aledsdavies/devcmd/core/transform"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// ================================================================================================
// HELPER FUNCTIONS - Bridge between core and runtime execution
// ================================================================================================

// TransformCommand transforms an AST command to IR (bridge to core/transform)
func TransformCommand(cmd *ast.CommandDecl) (ir.Node, error) {
	return transform.TransformCommand(cmd)
}

// NewCtx creates a new execution context with the given options (bridge to context package)
func NewCtx(opts context.CtxOptions) (*context.Ctx, error) {
	return context.NewCtx(opts)
}

// Re-export types for convenience
type (
	CtxOptions = context.CtxOptions
	EnvOptions = context.EnvOptions
)
