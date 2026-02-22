package executor

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
)

type sinkWriteErrorCapture struct {
	writer io.Writer
	mu     sync.Mutex
	err    error
}

func (c *sinkWriteErrorCapture) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	if err != nil {
		c.mu.Lock()
		if c.err == nil {
			c.err = err
		}
		c.mu.Unlock()
	}
	return n, err
}

func (c *sinkWriteErrorCapture) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// executeRedirect executes a redirect operation (> or >>).
// stdin is optional and used when redirect appears in a pipeline.
func (e *executor) executeRedirect(execCtx sdk.ExecutionContext, redirect *sdk.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	invariant.NotNil(redirect.Sink, "redirect sink")

	redirectExecCtx := withExecutionTransport(execCtx, sourceTransportID(redirect.Source))
	transportID := executionTransportID(redirectExecCtx)
	stderrOnly := redirectStderrEnabled(redirect.Source)
	stream := sdk.SinkStreamStdout
	if stderrOnly {
		stream = sdk.SinkStreamStderr
	}

	if redirect.Mode == sdk.RedirectInput {
		if err := sdk.ValidateSinkForRead(redirect.Sink); err != nil {
			kind, id := redirect.Sink.Identity()
			sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "validate", TransportID: transportID, Cause: err}
			fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
			return 1
		}

		reader, err := redirect.Sink.OpenRead(redirectExecCtx, sdk.SinkOpts{Mode: redirect.Mode, Stream: stream})
		if err != nil {
			kind, id := redirect.Sink.Identity()
			sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "open", TransportID: transportID, Cause: err}
			fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
			return 1
		}

		exitCode := e.executeTreeIO(redirectExecCtx, withRedirectedStderrSource(redirect.Source, stderrOnly), reader, nil)
		if closeErr := reader.Close(); closeErr != nil {
			kind, id := redirect.Sink.Identity()
			sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "close", TransportID: transportID, Cause: closeErr}
			fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
			return 1
		}

		return exitCode
	}

	if err := sdk.ValidateSinkForWrite(redirect.Sink, redirect.Mode); err != nil {
		kind, id := redirect.Sink.Identity()
		sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "validate", TransportID: transportID, Cause: err}
		fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
		return 1
	}

	writer, err := redirect.Sink.OpenWrite(redirectExecCtx, sdk.SinkOpts{Mode: redirect.Mode, Stream: stream})
	if err != nil {
		kind, id := redirect.Sink.Identity()
		sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "open", TransportID: transportID, Cause: err}
		fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
		return 1
	}

	writeCapture := &sinkWriteErrorCapture{writer: writer}
	exitCode := e.executeTreeIO(redirectExecCtx, withRedirectedStderrSource(redirect.Source, stderrOnly), stdin, writeCapture)
	if writeErr := writeCapture.Err(); writeErr != nil {
		kind, id := redirect.Sink.Identity()
		sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "write", TransportID: transportID, Cause: writeErr}
		fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
		if closeErr := writer.Close(); closeErr != nil {
			kind, id := redirect.Sink.Identity()
			sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "close", TransportID: transportID, Cause: closeErr}
			fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
		}
		return 1
	}
	if closeErr := writer.Close(); closeErr != nil {
		kind, id := redirect.Sink.Identity()
		sinkErr := SinkError{SinkID: kind + " (" + id + ")", Operation: "close", TransportID: transportID, Cause: closeErr}
		fmt.Fprintf(os.Stderr, "Error: %v\n", sinkErr)
		return 1
	}

	return exitCode
}

func redirectStderrEnabled(node sdk.TreeNode) bool {
	cmd, ok := node.(*sdk.CommandNode)
	if !ok {
		return false
	}
	stderr, _ := cmd.Args["stderr"].(bool)
	return stderr
}

func withRedirectedStderrSource(node sdk.TreeNode, stderrOnly bool) sdk.TreeNode {
	if !stderrOnly {
		return node
	}

	cmd, ok := node.(*sdk.CommandNode)
	if !ok || !isShellDecorator(cmd.Name) {
		return node
	}

	params := make(map[string]any, len(cmd.Args)+1)
	for k, v := range cmd.Args {
		params[k] = v
	}

	raw, ok := params["command"].(string)
	if !ok || raw == "" {
		return node
	}

	params["command"] = "(" + raw + ") 3>&1 1>&2 2>&3"

	return &sdk.CommandNode{
		Name:        cmd.Name,
		TransportID: cmd.TransportID,
		Args:        params,
		Block:       cmd.Block,
	}
}

func sourceTransportID(node sdk.TreeNode) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *sdk.CommandNode:
		return n.TransportID
	case *sdk.PipelineNode:
		if len(n.Commands) == 0 {
			return ""
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
			return ""
		}
		return sourceTransportID(n.Nodes[0])
	default:
		return ""
	}
}
