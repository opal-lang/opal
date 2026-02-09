package executor

import (
	"io"

	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
)

// executeStep executes a single step by executing its tree.
//
// Transport context is explicit per command via TransportID. DisplayID resolution
// uses that TransportID for boundary checks. Site-based authorization is handled
// by contract verification (plan hash), not runtime checks.
func (e *executor) executeStep(execCtx sdk.ExecutionContext, step sdk.Step) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.Precondition(step.Tree != nil, "step must have a tree")

	var stdin io.Reader
	var stdout io.Writer
	if ec, ok := execCtx.(*executionContext); ok {
		stdin = ec.stdin
		stdout = ec.stdoutPipe
	}

	return e.executeTreeIO(execCtx, step.Tree, stdin, stdout)
}

// executeTreeIO executes a tree node with optional stdin/stdout overrides.
// stdin is only applied where input is meaningful (commands, pipelines, right side of &&/||,
// and last element of sequence), matching shell behavior.
func (e *executor) executeTreeIO(execCtx sdk.ExecutionContext, node sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommandWithPipes(execCtx, n, stdin, stdout)

	case *sdk.PipelineNode:
		return e.executePipelineIO(execCtx, n, stdin, stdout)

	case *sdk.AndNode:
		leftExit := e.executeTreeIO(execCtx, n.Left, nil, stdout)
		if leftExit != 0 {
			return leftExit
		}
		return e.executeTreeIO(execCtx, n.Right, stdin, stdout)

	case *sdk.OrNode:
		leftExit := e.executeTreeIO(execCtx, n.Left, nil, stdout)
		if leftExit == 0 {
			return leftExit
		}
		return e.executeTreeIO(execCtx, n.Right, stdin, stdout)

	case *sdk.SequenceNode:
		var lastExit int
		for i, child := range n.Nodes {
			childStdin := io.Reader(nil)
			if i == len(n.Nodes)-1 {
				childStdin = stdin
			}
			lastExit = e.executeTreeIO(execCtx, child, childStdin, stdout)
		}
		return lastExit

	case *sdk.RedirectNode:
		return e.executeRedirect(execCtx, n, stdin)

	default:
		invariant.Invariant(false, "unknown TreeNode type: %T", node)
		return 1 // Unreachable
	}
}

// executeTreeNode executes a tree node (CommandNode or RedirectNode) with optional pipes.
// This is used by executePipeline to handle both commands and redirects in pipelines.
func (e *executor) executeTreeNode(execCtx sdk.ExecutionContext, node sdk.TreeNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := node.(type) {
	case *sdk.CommandNode:
		return e.executeCommandWithPipes(execCtx, n, stdin, stdout)
	case *sdk.RedirectNode:
		// The redirect source receives piped stdin and writes to sink directly.
		return e.executeRedirect(execCtx, n, stdin)
	default:
		invariant.Invariant(false, "invalid pipeline element type %T", node)
		return 1
	}
}
