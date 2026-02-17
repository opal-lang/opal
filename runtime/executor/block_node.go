package executor

import (
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
)

type blockNode struct {
	execCtx sdk.ExecutionContext
	steps   []sdk.Step
}

func (n *blockNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	child := n.childContext(ctx)
	exitCode, err := child.ExecuteBlock(n.steps)
	return decorator.Result{ExitCode: exitCode}, err
}

func (n *blockNode) BranchCount() int {
	return len(n.steps)
}

func (n *blockNode) ExecuteBranch(index int, ctx decorator.ExecContext) (decorator.Result, error) {
	invariant.Precondition(index >= 0 && index < len(n.steps), "branch index out of bounds: %d", index)

	child := n.childContext(ctx)
	exitCode, err := child.ExecuteBlock([]sdk.Step{n.steps[index]})
	return decorator.Result{ExitCode: exitCode}, err
}

func (n *blockNode) childContext(ctx decorator.ExecContext) sdk.ExecutionContext {
	return childExecutionContextFromDecorator(n.execCtx, ctx)
}

func childExecutionContextFromDecorator(base sdk.ExecutionContext, ctx decorator.ExecContext) sdk.ExecutionContext {
	child := base.WithContext(ctx.Context)
	if typed, ok := child.(*executionContext); ok {
		child = typed.withPipes(ctx.Stdin, ctx.Stdout)
	}
	if ctx.Session != nil {
		if typed, ok := child.(*executionContext); ok {
			child = typed.withTransportID(normalizedTransportID(ctx.Session.ID()))
		}
		child = child.WithEnviron(ctx.Session.Env())
		child = child.WithWorkdir(ctx.Session.Cwd())
	}
	return child
}
