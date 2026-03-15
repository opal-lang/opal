package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/plugin/mockplugin"
	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/builtwithtofu/sigil/runtime/parser"
	"github.com/builtwithtofu/sigil/runtime/planner"
	"github.com/builtwithtofu/sigil/runtime/vault"
	"github.com/google/go-cmp/cmp"
)

func init() {
	_ = plugin.Global().Register(&mockplugin.AWSPlugin{})
}

func TestPluginEndToEndParsePlanExecute(t *testing.T) {
	source := []byte(`
var db_url = @aws.secrets.prod_database

@exec.retry(delay=1ms, times=2) {
    @aws.instance.connect(connectionType="ssh", credentials="creds", instance="i-123") {
        echo @var.db_url
    }
}
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey([]byte("plugin-end-to-end-plan-key-000000"))
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Vault: vlt})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	result, err := ExecutePlan(context.Background(), plan, Config{}, vlt)
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestPluginTransportAllowsEnvInChildSession(t *testing.T) {
	source := []byte(`
@aws.instance.connect(connectionType="ssh", credentials="creds", instance="i-123") {
    echo @env.HOME
}
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey([]byte("plugin-transport-env-plan-key-000"))
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Vault: vlt})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	result, err := ExecutePlan(context.Background(), plan, Config{}, vlt)
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
}

func TestPluginTransportMissingDeclaredSecretFails(t *testing.T) {
	source := []byte(`
@aws.instance.connect(connectionType="ssh", instance="i-123") {
    echo ok
}
`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	vlt := vault.NewWithPlanKey([]byte("plugin-transport-missing-secret-0"))
	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{Vault: vlt})
	if err == nil {
		t.Fatal("expected planning error for missing declared secret")
	}
	if diff := cmp.Diff(true, strings.Contains(err.Error(), `declared secret "credentials" not provided`)); diff != "" {
		t.Fatalf("missing secret error mismatch (-want +got):\n%s", diff)
	}
}
