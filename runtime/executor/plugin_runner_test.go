package executor

import (
	"context"
	"io/fs"
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/plugin/mockplugin"
	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/google/go-cmp/cmp"
)

func init() {
	_ = plugin.Global().Register(&mockplugin.AWSPlugin{})
}

func TestPluginTransportExecution(t *testing.T) {
	vlt := testVault()
	exprID := vlt.TrackExpression("creds")
	vlt.StoreUnresolvedValue(exprID, "creds")
	vlt.MarkTouched(exprID)
	vlt.ResolveAllTouched()
	displayID := vlt.GetDisplayID(exprID)

	plan := &planfmt.Plan{
		Target: "plugin-transport",
		Steps: []planfmt.Step{{
			ID: 1,
			Tree: &planfmt.CommandNode{
				Decorator: "@aws.instance.connect",
				Args: []planfmt.Arg{
					{Key: "connectionType", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "ssh"}},
					{Key: "credentials", Val: planfmt.Value{Kind: planfmt.ValueString, Str: displayID}},
					{Key: "instance", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "i-123"}},
				},
				Block: []planfmt.Step{{
					ID:   2,
					Tree: shellCmd("echo 'Hello, World!'"),
				}},
			},
		}},
	}

	result, err := ExecutePlan(context.Background(), plan, Config{}, vlt)
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, result.StepsRun); diff != "" {
		t.Fatalf("steps run mismatch (-want +got):\n%s", diff)
	}
}

type overlayOpenedTransport struct {
	snapshot plugin.SessionSnapshot
}

func (o overlayOpenedTransport) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
}

func (o overlayOpenedTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (o overlayOpenedTransport) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}
func (o overlayOpenedTransport) Snapshot() plugin.SessionSnapshot { return o.snapshot }
func (o overlayOpenedTransport) WithSnapshot(snapshot plugin.SessionSnapshot) plugin.OpenedTransport {
	return overlayOpenedTransport{snapshot: snapshot}
}
func (o overlayOpenedTransport) Close() error { return nil }

func TestPluginTransportSessionPreservesEnvAndWorkdirOverlays(t *testing.T) {
	base := pluginTransportSession{
		id: "plugin:test",
		inner: overlayOpenedTransport{snapshot: plugin.SessionSnapshot{
			Env:      map[string]string{"HOME": "/home/tester"},
			Workdir:  "/tmp/base",
			Platform: "linux",
		}},
	}

	withEnv := base.WithEnv(map[string]string{"APP_ENV": "prod"}).(pluginTransportSession)
	if diff := cmp.Diff(map[string]string{"HOME": "/home/tester", "APP_ENV": "prod"}, withEnv.Env()); diff != "" {
		t.Fatalf("env overlay mismatch (-want +got):\n%s", diff)
	}

	withDir := withEnv.WithWorkdir("/srv/app").(pluginTransportSession)
	if diff := cmp.Diff("/srv/app", withDir.Cwd()); diff != "" {
		t.Fatalf("workdir overlay mismatch (-want +got):\n%s", diff)
	}
}
