package decorators

import (
	"context"
	"fmt"
	"time"

	"github.com/opal-lang/opal/core/decorator"
)

// TimeoutDecorator implements the @timeout execution decorator.
// Executes block with a timeout constraint.
type TimeoutDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *TimeoutDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("timeout").
		Summary("Execute block with timeout constraint").
		Roles(decorator.RoleWrapper).
		ParamDuration("duration", "Maximum execution time").
		Required().
		Examples("30s", "5m", "1h").
		Done().
		Block(decorator.BlockRequired).
		Build()
}

// Wrap implements the Exec interface.
func (d *TimeoutDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &timeoutNode{next: next, params: params}
}

// timeoutNode wraps an execution node with timeout logic.
type timeoutNode struct {
	next   decorator.ExecNode
	params map[string]any
}

// Execute implements the ExecNode interface.
func (n *timeoutNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if n.next == nil {
		return decorator.Result{ExitCode: 0}, nil
	}

	duration, err := n.duration()
	if err != nil {
		return decorator.Result{ExitCode: decorator.ExitFailure}, err
	}

	parent := ctx.Context
	if parent == nil {
		parent = context.Background()
	}

	timeoutCtx, cancel := context.WithTimeout(parent, duration)
	defer cancel()

	result, execErr := n.next.Execute(ctx.WithContext(timeoutCtx))
	if timeoutCtx.Err() != nil {
		return decorator.Result{ExitCode: decorator.ExitCanceled}, timeoutCtx.Err()
	}

	return result, execErr
}

func (n *timeoutNode) duration() (time.Duration, error) {
	raw, ok := n.params["duration"]
	if !ok {
		return 0, fmt.Errorf("@timeout requires duration parameter")
	}

	switch v := raw.(type) {
	case time.Duration:
		if v <= 0 {
			return 0, fmt.Errorf("@timeout duration must be > 0")
		}
		return v, nil
	case string:
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("invalid @timeout duration %q: %w", v, err)
		}
		if parsed <= 0 {
			return 0, fmt.Errorf("@timeout duration must be > 0")
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("@timeout duration must be duration string")
	}
}

// Register @timeout decorator with the global registry
func init() {
	if err := decorator.Register("timeout", &TimeoutDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @timeout decorator: %v", err))
	}
}
