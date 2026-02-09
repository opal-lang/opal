package decorators

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/opal-lang/opal/core/decorator"
)

// RetryDecorator implements the @retry execution decorator.
// Retries failed operations with configurable backoff strategy.
type RetryDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *RetryDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("retry").
		Summary("Retry failed operations with exponential backoff").
		Roles(decorator.RoleWrapper).
		ParamInt("times", "Number of retry attempts").
		Min(1).
		Max(100).
		Default(3).
		Examples("3", "5", "10").
		Done().
		ParamDuration("delay", "Initial delay between retries").
		Default("1s").
		Examples("1s", "5s", "30s").
		Done().
		ParamEnum("backoff", "Backoff strategy").
		Values("exponential", "linear", "constant").
		Default("exponential").
		Done().
		Block(decorator.BlockOptional).
		Build()
}

// Wrap implements the Exec interface.
func (d *RetryDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &retryNode{next: next, params: params}
}

// retryNode wraps an execution node with retry logic.
type retryNode struct {
	next   decorator.ExecNode
	params map[string]any
}

// Execute implements the ExecNode interface.
func (n *retryNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if n.next == nil {
		return decorator.Result{ExitCode: 0}, nil
	}

	attempts := n.attempts()
	delay := n.delay()
	backoff := n.backoff()

	var lastResult decorator.Result
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Context != nil {
			if err := ctx.Context.Err(); err != nil {
				return decorator.Result{ExitCode: decorator.ExitCanceled}, err
			}
		}

		lastResult, lastErr = n.next.Execute(ctx)
		if lastErr == nil && lastResult.ExitCode == 0 {
			return lastResult, nil
		}

		if attempt == attempts || delay <= 0 {
			continue
		}

		wait := retryDelay(delay, backoff, attempt)
		if waitErr := waitContext(ctx.Context, wait); waitErr != nil {
			return decorator.Result{ExitCode: decorator.ExitCanceled}, waitErr
		}
	}

	return lastResult, lastErr
}

func (n *retryNode) attempts() int {
	raw, ok := n.params["times"]
	if !ok {
		return 3
	}

	switch v := raw.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	}

	return 3
}

func (n *retryNode) delay() time.Duration {
	raw, ok := n.params["delay"]
	if !ok {
		return time.Second
	}

	switch v := raw.(type) {
	case time.Duration:
		return v
	case string:
		parsed, err := time.ParseDuration(v)
		if err == nil {
			return parsed
		}
	}

	return time.Second
}

func (n *retryNode) backoff() string {
	mode, _ := n.params["backoff"].(string)
	switch mode {
	case "constant", "linear", "exponential":
		return mode
	default:
		return "exponential"
	}
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

func waitContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}

	if ctx == nil {
		time.Sleep(wait)
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Register @retry decorator with the global registry
func init() {
	if err := decorator.Register("retry", &RetryDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @retry decorator: %v", err))
	}
}
