package builtins

import (
	"context"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

func TestTimeoutWrapperCapabilityCancelsAfterDeadline(t *testing.T) {
	capability := TimeoutWrapperCapability{}
	node := capability.Wrap(fakeExecNode{execute: func(ctx plugin.ExecContext) (plugin.Result, error) {
		<-ctx.Context().Done()
		return plugin.Result{ExitCode: plugin.ExitCanceled}, ctx.Context().Err()
	}}, fakeArgs{durations: map[string]time.Duration{"duration": 10 * time.Millisecond}})

	result, err := node.Execute(fakeExecContext{ctx: context.Background()})
	if err == nil {
		t.Fatal("Execute() error = nil, want timeout error")
	}
	if diff := cmp.Diff(plugin.ExitCanceled, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit code mismatch (-want +got):\n%s", diff)
	}
}
