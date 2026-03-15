package planner

import (
	"fmt"
	"testing"

	_ "github.com/builtwithtofu/sigil/runtime/decorators"
	"github.com/builtwithtofu/sigil/runtime/parser"
	"github.com/builtwithtofu/sigil/runtime/vault"
	"github.com/google/go-cmp/cmp"
)

func parsePlanNew(t *testing.T, source string, v *vault.Vault) (*PlanResult, error) {
	t.Helper()
	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}
	return PlanWithObservability(tree.Events, tree.Tokens, Config{Vault: v})
}

func TestPlanNew_TransportBoundary_EnvBlockedAcrossBoundary(t *testing.T) {
	source := `
var LOCAL_HOME = @env.HOME
@test.transport {
    echo "Home: @var.LOCAL_HOME"
}
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	_, err := parsePlanNew(t, source, v)
	if err == nil {
		t.Fatal("Expected transport boundary error, got nil")
	}

	refVault := vault.NewWithPlanKey(planKey)
	refVault.EnterTransport(localTransportID(planKey))
	exprID := refVault.DeclareVariableTransportSensitive("LOCAL_HOME", "@env.HOME")
	transportID, transportErr := deriveTransportID(planKey, "@test.transport", nil, localTransportID(planKey))
	if transportErr != nil {
		t.Fatalf("derive transport ID failed: %v", transportErr)
	}
	expected := fmt.Sprintf(
		"failed to resolve: transport boundary violation: expression %q declared in %q, cannot use in %q",
		exprID,
		localTransportID(planKey),
		transportID,
	)

	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanNew_TransportBoundary_FunctionArgPreservesTransportSensitivity(t *testing.T) {
	source := `
fun deploy(home String) {
    @test.transport {
        echo @var.home
    }
}

var LOCAL_HOME = @env.HOME
deploy(@var.LOCAL_HOME)
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	_, err := parsePlanNew(t, source, v)
	if err == nil {
		t.Fatal("Expected transport boundary error, got nil")
	}

	refVault := vault.NewWithPlanKey(planKey)
	refVault.EnterTransport(localTransportID(planKey))
	exprID := refVault.DeclareVariableTransportSensitive("LOCAL_HOME", "@env.HOME")
	transportID, transportErr := deriveTransportID(planKey, "@test.transport", nil, localTransportID(planKey))
	if transportErr != nil {
		t.Fatalf("derive transport ID failed: %v", transportErr)
	}
	expected := fmt.Sprintf(
		"failed to resolve: transport boundary violation: expression %q declared in %q, cannot use in %q",
		exprID,
		localTransportID(planKey),
		transportID,
	)

	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanNew_TransportBoundary_VarFromLiteralAllowedAcrossBoundary(t *testing.T) {
	source := `
var VERSION = "1.0.0"
@test.transport {
    echo @var.VERSION
}
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	result, err := parsePlanNew(t, source, v)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}
	if result == nil || result.Plan == nil {
		t.Fatal("Expected plan result, got nil")
	}
}

func TestPlanNew_TransportBoundary_DirectEnvInIdempotentTransportWorks(t *testing.T) {
	source := `
@test.transport.idempotent {
    echo "Home: @env.HOME"
}
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	result, err := parsePlanNew(t, source, v)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}
	if result == nil || result.Plan == nil {
		t.Fatal("Expected plan result, got nil")
	}
}

func TestPlanNew_TransportBoundary_DirectEnvInNonIdempotentTransportFails(t *testing.T) {
	source := `
@test.transport {
    echo "Home: @env.HOME"
}
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	_, err := parsePlanNew(t, source, v)
	if err == nil {
		t.Fatal("Expected transport env error, got nil")
	}

	if diff := cmp.Diff("failed to resolve: @env cannot be used inside @test.transport", err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestPlanNew_TransportBoundary_DirectEnvUsesNearestTransportSession(t *testing.T) {
	source := `
@test.transport.env {
    echo "Home: @env.HOME"
}
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	result, err := parsePlanNew(t, source, v)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}
	if result == nil || result.Plan == nil {
		t.Fatal("Expected plan result, got nil")
	}
}

func TestPlanNew_TransportBoundary_NestedEnvUsesNearestParentTransport(t *testing.T) {
	source := `
@test.transport.env {
    @test.transport.env {
        echo "Home: @env.HOME"
    }
}
`

	planKey := []byte("plan-key-transport-boundary-0000")
	v := vault.NewWithPlanKey(planKey)

	result, err := parsePlanNew(t, source, v)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}
	if result == nil || result.Plan == nil {
		t.Fatal("Expected plan result, got nil")
	}
}
