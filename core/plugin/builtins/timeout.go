package builtins

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// TimeoutWrapperCapability wraps execution with a timeout.
type TimeoutWrapperCapability struct{}

func (c TimeoutWrapperCapability) Path() string { return "exec.timeout" }

func (c TimeoutWrapperCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{{Name: "duration", Type: types.TypeDuration, Required: true}},
		Block:  plugin.BlockRequired,
	}
}

func (c TimeoutWrapperCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return timeoutNode{next: next, duration: args.GetDuration("duration")}
}

type timeoutNode struct {
	next     plugin.ExecNode
	duration time.Duration
}

func (n timeoutNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	if n.next == nil {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}
	if n.duration <= 0 {
		return plugin.Result{ExitCode: plugin.ExitFailure}, fmt.Errorf("@exec.timeout duration must be > 0")
	}

	parent := ctx.Context()
	if parent == nil {
		parent = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(parent, n.duration)
	defer cancel()

	result, err := n.next.Execute(execContextWithContext{ExecContext: ctx, ctx: timeoutCtx})
	if timeoutCtx.Err() != nil {
		return plugin.Result{ExitCode: plugin.ExitCanceled}, fmt.Errorf("timeout: execution exceeded %s", n.duration)
	}
	return result, err
}

type execContextWithContext struct {
	plugin.ExecContext
	ctx context.Context
}

func (c execContextWithContext) Context() context.Context { return c.ctx }
func (c execContextWithContext) Session() plugin.ParentTransport {
	return c.ExecContext.Session()
}
func (c execContextWithContext) Stdin() io.Reader  { return c.ExecContext.Stdin() }
func (c execContextWithContext) Stdout() io.Writer { return c.ExecContext.Stdout() }
func (c execContextWithContext) Stderr() io.Writer { return c.ExecContext.Stderr() }
