package executor

import (
	"bytes"
	"context"
	"testing"

	"github.com/builtwithtofu/sigil/core/runtime"
	"github.com/builtwithtofu/sigil/core/sdk"
	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/google/go-cmp/cmp"
)

func TestExecuteCommandWithPipesRejectsNonExecutablePluginCapability(t *testing.T) {
	var stderr bytes.Buffer
	e := &executor{sessions: newSessionRuntime(nil), stderr: &stderr}
	execCtx := newExecutionContext(map[string]interface{}{}, e, context.Background())
	cmd := &sdk.CommandNode{Name: "@env", Args: map[string]any{"name": "HOME"}}

	exitCode := e.executeCommandWithPipes(execCtx, cmd, nil, nil)

	if diff := cmp.Diff(runtime.ExitFailure, exitCode); diff != "" {
		t.Fatalf("executeCommandWithPipes() exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("Error: @env is not executable\n", stderr.String()); diff != "" {
		t.Fatalf("stderr mismatch (-want +got):\n%s", diff)
	}
}
