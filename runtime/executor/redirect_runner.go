package executor

import (
	"fmt"
	"io"
	"os"

	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
)

// executeRedirect executes a redirect operation (> or >>).
// stdin is optional and used when redirect appears in a pipeline.
func (e *executor) executeRedirect(execCtx sdk.ExecutionContext, redirect *sdk.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")

	redirectExecCtx := withExecutionTransport(execCtx, sourceTransportID(redirect.Source))

	caps := redirect.Sink.Caps()
	if redirect.Mode == sdk.RedirectOverwrite && !caps.Overwrite {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: sink %s (%s) does not support overwrite (>)\n", kind, path)
		return 1
	}
	if redirect.Mode == sdk.RedirectAppend && !caps.Append {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: sink %s (%s) does not support append (>>)\n", kind, path)
		return 1
	}

	writer, err := redirect.Sink.Open(redirectExecCtx, redirect.Mode, nil)
	if err != nil {
		kind, path := redirect.Sink.Identity()
		fmt.Fprintf(os.Stderr, "Error: failed to open sink %s (%s): %v\n", kind, path, err)
		return 1
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			kind, path := redirect.Sink.Identity()
			fmt.Fprintf(os.Stderr, "Error: failed to close sink %s (%s): %v\n", kind, path, closeErr)
		}
	}()

	return e.executeTreeIO(redirectExecCtx, redirect.Source, stdin, writer)
}

func sourceTransportID(node sdk.TreeNode) string {
	if node == nil {
		return "local"
	}

	switch n := node.(type) {
	case *sdk.CommandNode:
		return normalizedTransportID(n.TransportID)
	case *sdk.PipelineNode:
		if len(n.Commands) == 0 {
			return "local"
		}
		return sourceTransportID(n.Commands[0])
	case *sdk.RedirectNode:
		return sourceTransportID(n.Source)
	case *sdk.AndNode:
		return sourceTransportID(n.Left)
	case *sdk.OrNode:
		return sourceTransportID(n.Left)
	case *sdk.SequenceNode:
		if len(n.Nodes) == 0 {
			return "local"
		}
		return sourceTransportID(n.Nodes[0])
	default:
		return "local"
	}
}
