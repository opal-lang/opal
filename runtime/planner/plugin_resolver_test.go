package planner

import (
	"context"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/plugin/mockplugin"
	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/builtwithtofu/sigil/runtime/vault"
	"github.com/google/go-cmp/cmp"
)

func init() {
	_ = plugin.Global().Register(&mockplugin.AWSPlugin{})
}

func TestPluginRegistryPlannerResolution(t *testing.T) {
	v := vault.NewWithPlanKey([]byte("01234567890123456789012345678901"))
	exprID := v.DeclareVariable("DB", "@aws.secrets.prod_database")
	r := &Resolver{
		vault:            v,
		session:          &mockSession{},
		config:           ResolveConfig{Context: context.Background()},
		decoratorExprIDs: make(map[string]string),
	}

	calls := []decoratorCall{{
		decorator: &DecoratorRef{Name: "aws", Selector: []string{"secrets", "prod_database"}},
		exprID:    exprID,
	}}

	if err := r.resolveBatch("aws.secrets", calls); err != nil {
		t.Fatalf("resolveBatch() error = %v", err)
	}

	got, ok := r.getValue(decoratorKey(calls[0].decorator))
	if !ok {
		t.Fatal("resolved decorator value not available via resolver lookup")
	}
	if diff := cmp.Diff(any("mock-secret-prod_database"), got); diff != "" {
		t.Fatalf("resolved value mismatch (-want +got):\n%s", diff)
	}
}

func TestPlannerResolvedArgsParsesDuration(t *testing.T) {
	args := plannerResolvedArgs{params: map[string]any{"delay": durationLiteral("2s")}}
	if diff := cmp.Diff(2*time.Second, args.GetDuration("delay")); diff != "" {
		t.Fatalf("duration mismatch (-want +got):\n%s", diff)
	}
}
