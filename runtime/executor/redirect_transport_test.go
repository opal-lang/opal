package executor

import (
	"context"
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/google/go-cmp/cmp"
)

func TestRedirectSinkUsesSourceTransportContext(t *testing.T) {
	t.Parallel()
	registerExecutorSessionTestPlugin()

	id := t.TempDir() + "/capture-A"
	pluginTestSinkStore.reset(id)

	plan := &planfmt.Plan{Target: "redirect-source-transport", Transports: localTestTransports("transport:A"), Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.RedirectNode{
			Source: &planfmt.CommandNode{
				Decorator:   "@shell",
				TransportID: "transport:A",
				Args:        []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo routed"}}},
			},
			Target: planfmt.CommandNode{
				Decorator: "@test.capture.sink",
				Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: id}}},
			},
			Mode: planfmt.RedirectOverwrite,
		},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: scopedLocalSessionFactory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	record := pluginTestSinkStore.snapshot(id)
	if diff := cmp.Diff(1, record.openCount); diff != "" {
		t.Fatalf("sink open count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"transport:A"}, record.sessionIDs); diff != "" {
		t.Fatalf("session routing mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("routed\n", record.output.String()); diff != "" {
		t.Fatalf("sink output mismatch (-want +got):\n%s", diff)
	}
}

func TestRedirectSinkInheritsWrapperTransportContext(t *testing.T) {
	t.Parallel()
	registerExecutorSessionTestPlugin()

	id := t.TempDir() + "/capture-boundary"
	pluginTestSinkStore.reset(id)

	plan := &planfmt.Plan{Target: "redirect-wrapper-transport", Transports: localTestTransports("transport:boundary"), Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.CommandNode{
			Decorator: "@test.session.boundary",
			Args:      []planfmt.Arg{{Key: "id", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "transport:boundary"}}},
			Block: []planfmt.Step{{
				ID: 2,
				Tree: &planfmt.RedirectNode{
					Source: &planfmt.CommandNode{
						Decorator: "@shell",
						Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo routed"}}},
					},
					Target: planfmt.CommandNode{
						Decorator: "@test.capture.sink",
						Args:      []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: id}}},
					},
					Mode: planfmt.RedirectOverwrite,
				},
			}},
		},
	}}}

	result, err := ExecutePlan(context.Background(), plan, Config{sessionFactory: scopedLocalSessionFactory}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	record := pluginTestSinkStore.snapshot(id)
	if diff := cmp.Diff(1, record.openCount); diff != "" {
		t.Fatalf("sink open count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"transport:boundary"}, record.sessionIDs); diff != "" {
		t.Fatalf("session routing mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("routed\n", record.output.String()); diff != "" {
		t.Fatalf("sink output mismatch (-want +got):\n%s", diff)
	}
}
