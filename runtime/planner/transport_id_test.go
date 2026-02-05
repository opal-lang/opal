package planner

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"
)

type envTransportDecorator struct{}

func (d *envTransportDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.transport.env").
		Summary("Test transport that provides a custom environment").
		Roles(decorator.RoleBoundary).
		Block(decorator.BlockRequired).
		Idempotent().
		Build()
}

func (d *envTransportDecorator) Open(parent decorator.Session, params map[string]any) (decorator.Session, error) {
	base, err := decorator.NewTestTransport("test-env").Open(parent, params)
	if err != nil {
		return nil, err
	}
	return base.WithEnv(map[string]string{"HOME": "remote-home"}), nil
}

func (d *envTransportDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return next
}

func init() {
	_ = decorator.Register("test.transport.env", &envTransportDecorator{})
}

func TestPlanNew_TransportIDs(t *testing.T) {
	source := `
@test.transport {
    echo "hello"
}
`
	planKey := []byte("plan-key-transport-ids-000000000")
	v := vault.NewWithPlanKey(planKey)

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}
	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{Vault: v})
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	plan := result.Plan
	localID := localTransportID(planKey)
	expectedTransportID, err := deriveTransportID(planKey, "@test.transport", nil, localID)
	if err != nil {
		t.Fatalf("derive transport ID failed: %v", err)
	}

	if len(plan.Transports) != 2 {
		t.Fatalf("expected 2 transports, got %d", len(plan.Transports))
	}

	var localTransport planfmt.Transport
	var transport planfmt.Transport
	for _, entry := range plan.Transports {
		switch entry.ID {
		case localID:
			localTransport = entry
		case expectedTransportID:
			transport = entry
		}
	}

	if diff := cmp.Diff(localID, localTransport.ID); diff != "" {
		t.Errorf("local transport ID mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("local", localTransport.Decorator); diff != "" {
		t.Errorf("local transport decorator mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", localTransport.ParentID); diff != "" {
		t.Errorf("local transport parent ID mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(expectedTransportID, transport.ID); diff != "" {
		t.Errorf("transport ID mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("@test.transport", transport.Decorator); diff != "" {
		t.Errorf("transport decorator mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(localID, transport.ParentID); diff != "" {
		t.Errorf("transport parent ID mismatch (-want +got):\n%s", diff)
	}

	cmd, ok := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", plan.Steps[0].Tree)
	}
	if diff := cmp.Diff(localID, cmd.TransportID); diff != "" {
		t.Errorf("root command transport ID mismatch (-want +got):\n%s", diff)
	}
	if len(cmd.Block) != 1 {
		t.Fatalf("expected 1 block step, got %d", len(cmd.Block))
	}
	innerCmd, ok := cmd.Block[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("expected inner CommandNode, got %T", cmd.Block[0].Tree)
	}
	if diff := cmp.Diff(expectedTransportID, innerCmd.TransportID); diff != "" {
		t.Errorf("inner command transport ID mismatch (-want +got):\n%s", diff)
	}
}

func TestResolve_EnvUsesTransportSession(t *testing.T) {
	source := `
@test.transport.env {
    var HOME = @env.HOME
}
`
	planKey := []byte("plan-key-transport-env-000000000")
	v := vault.NewWithPlanKey(planKey)
	session := &mockSession{env: map[string]string{"HOME": "local-home"}}

	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}
	graph, err := BuildIR(tree.Events, tree.Tokens)
	if err != nil {
		t.Fatalf("BuildIR failed: %v", err)
	}

	result, err := Resolve(graph, v, session, ResolveConfig{Context: context.Background()})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	cmd := result.Statements[0].Command
	if cmd == nil || len(cmd.Block) != 1 {
		t.Fatalf("expected transport block with 1 statement")
	}
	varDecl := cmd.Block[0].VarDecl
	if varDecl == nil {
		t.Fatalf("expected var declaration inside transport block")
	}
	value, ok := v.GetUnresolvedValue(varDecl.ExprID)
	if !ok {
		t.Fatalf("expected vault value for exprID %q", varDecl.ExprID)
	}
	if diff := cmp.Diff("remote-home", value); diff != "" {
		t.Errorf("env value mismatch (-want +got):\n%s", diff)
	}
}
