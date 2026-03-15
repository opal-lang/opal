package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/google/go-cmp/cmp"
)

func TestParallelBranchWorkdirIsolation(t *testing.T) {
	t.Parallel()

	registerExecutorSessionTestPlugin()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	dirA := t.TempDir()
	outPath := filepath.Join(t.TempDir(), "parallel-output.txt")

	plan := &planfmt.Plan{Target: "parallel-workdir", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.RedirectNode{
			Source: &planfmt.CommandNode{
				Decorator: "@exec.parallel",
				Args: []planfmt.Arg{
					{Key: "maxConcurrency", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 2}},
					{Key: "onFailure", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "wait_all"}},
				},
				Block: []planfmt.Step{
					{
						ID: 2,
						Tree: &planfmt.CommandNode{
							Decorator: "@test.chdir",
							Args:      []planfmt.Arg{{Key: "dir", Val: planfmt.Value{Kind: planfmt.ValueString, Str: dirA}}},
							Block: []planfmt.Step{{
								ID:   3,
								Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "pwd"}}}},
							}},
						},
					},
					{
						ID:   4,
						Tree: &planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "pwd"}}}},
					},
				},
			},
			Target: planfmt.CommandNode{Decorator: "@shell", Args: []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: outPath}}}},
			Mode:   planfmt.RedirectOverwrite,
		},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	want := dirA + "\n" + originalWd + "\n"
	content, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("read redirected output: %v", readErr)
	}
	if diff := cmp.Diff(want, string(content)); diff != "" {
		t.Fatalf("parallel branch isolation mismatch (-want +got):\n%s", diff)
	}
}
