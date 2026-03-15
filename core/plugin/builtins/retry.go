package builtins

import (
	"math"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// RetryWrapperCapability wraps execution with retry behavior.
type RetryWrapperCapability struct{}

var (
	retryMin = float64(1)
	retryMax = float64(100)
)

func (c RetryWrapperCapability) Kind() plugin.CapabilityKind { return plugin.KindWrapper }

func (c RetryWrapperCapability) Path() string { return "exec.retry" }

func (c RetryWrapperCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "times", Type: types.TypeInt, Default: 3, Minimum: &retryMin, Maximum: &retryMax},
			{Name: "delay", Type: types.TypeDuration, Default: time.Second},
			{Name: "backoff", Type: types.TypeString, Default: "exponential", Enum: []string{"exponential", "linear", "constant"}},
		},
		Block: plugin.BlockOptional,
	}
}

func (c RetryWrapperCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return retryNode{
		next:    next,
		times:   args.GetInt("times"),
		delay:   args.GetDuration("delay"),
		backoff: args.GetString("backoff"),
	}
}

type retryNode struct {
	next    plugin.ExecNode
	times   int
	delay   time.Duration
	backoff string
}

func (n retryNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	if n.next == nil {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}

	var last plugin.Result
	var lastErr error

	for attempt := 1; attempt <= n.times; attempt++ {
		if err := ctx.Context().Err(); err != nil {
			return plugin.Result{ExitCode: plugin.ExitCanceled}, err
		}

		last, lastErr = n.next.Execute(ctx)
		if lastErr == nil && last.ExitCode == plugin.ExitSuccess {
			return last, nil
		}

		if attempt == n.times || n.delay <= 0 {
			continue
		}

		select {
		case <-time.After(retryDelay(n.delay, n.backoff, attempt)):
		case <-ctx.Context().Done():
			return plugin.Result{ExitCode: plugin.ExitCanceled}, ctx.Context().Err()
		}
	}

	return last, lastErr
}

func retryDelay(base time.Duration, backoff string, attempt int) time.Duration {
	if attempt < 1 {
		return base
	}

	switch backoff {
	case "constant":
		return base
	case "linear":
		return time.Duration(attempt) * base
	default:
		scale := math.Pow(2, float64(attempt-1))
		return time.Duration(float64(base) * scale)
	}
}
