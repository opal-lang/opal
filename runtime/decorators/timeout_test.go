package decorators

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestTimeoutCancelsLongRunningExecution(t *testing.T) {
	dec := &TimeoutDecorator{}
	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		<-ctx.Context.Done()
		return decorator.Result{ExitCode: decorator.ExitCanceled}, ctx.Context.Err()
	}}, map[string]any{"duration": "30ms"})

	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if diff := cmp.Diff(decorator.ExitCanceled, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestTimeoutPassesThroughSuccessfulExecution(t *testing.T) {
	dec := &TimeoutDecorator{}
	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		return decorator.Result{ExitCode: 0}, nil
	}}, map[string]any{"duration": "1s"})

	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err != nil {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestTimeoutRejectsInvalidDuration(t *testing.T) {
	dec := &TimeoutDecorator{}
	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		return decorator.Result{ExitCode: 0}, nil
	}}, map[string]any{"duration": "not-a-duration"})

	result, err := node.Execute(decorator.ExecContext{Context: context.Background()})
	if err == nil {
		t.Fatal("expected invalid duration error")
	}
	if diff := cmp.Diff(decorator.ExitFailure, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestTimeoutUsesParentDeadlineWhenSooner(t *testing.T) {
	dec := &TimeoutDecorator{}
	parent, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	node := dec.Wrap(&testExecNode{execute: func(ctx decorator.ExecContext) (decorator.Result, error) {
		<-ctx.Context.Done()
		return decorator.Result{ExitCode: decorator.ExitCanceled}, ctx.Context.Err()
	}}, map[string]any{"duration": "1s"})

	result, err := node.Execute(decorator.ExecContext{Context: parent})
	if err == nil {
		t.Fatal("expected parent deadline error")
	}
	if diff := cmp.Diff(decorator.ExitCanceled, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}
