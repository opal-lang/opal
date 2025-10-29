package decorator

// Span represents a telemetry span for observability.
//
// Telemetry Ownership Model:
//   - Opal runtime: Automatically tracks decorator entry/exit, total timing, and call hierarchy
//   - Decorator implementations: Can create child spans for internal operations (optional)
//
// Example:
//
//	func (d *RetryDecorator) Wrap(next ExecNode, params map[string]any) ExecNode {
//	    // Opal runtime creates parent span: "decorator.retry"
//	    return &retryNode{
//	        next:     next,
//	        attempts: params["attempts"].(int),
//	    }
//	}
//
//	func (n *retryNode) Execute(ctx ExecContext) (Result, error) {
//	    // Decorator can create child spans for internal tracking
//	    for i := 0; i < n.attempts; i++ {
//	        attemptSpan := ctx.Trace.Child("retry.attempt", map[string]any{"attempt": i})
//	        result, err := n.next.Execute(ctx)
//	        attemptSpan.End()
//	        if err == nil {
//	            return result, nil
//	        }
//	    }
//	    return Result{}, fmt.Errorf("all attempts failed")
//	}
//
// This is a stub interface - full implementation in Phase 7.
type Span interface {
	// End marks the span as complete
	End()

	// Child creates a child span for internal operations (optional)
	// Decorators can use this to track internal logic
	Child(name string, attrs map[string]any) Span
}

// NoOpSpan is a no-op implementation of Span.
type NoOpSpan struct{}

// End does nothing.
func (NoOpSpan) End() {}
