package executor

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	runtimedecorators "github.com/opal-lang/opal/runtime/decorators"
)

type shellSentinelDecorator struct{}

func (d *shellSentinelDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("shell").
		Summary("Sentinel decorator to detect bypass").
		Roles(decorator.RoleWrapper).
		Block(decorator.BlockForbidden).
		Build()
}

func (d *shellSentinelDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return shellSentinelNode{}
}

type shellSentinelNode struct{}

func (n shellSentinelNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	return decorator.Result{ExitCode: 97}, nil
}

func TestExecuteShellBypassesDecoratorExecPath(t *testing.T) {
	if err := decorator.Register("shell", &shellSentinelDecorator{}); err != nil {
		t.Fatalf("register sentinel shell decorator: %v", err)
	}
	t.Cleanup(func() {
		if err := decorator.Register("shell", &runtimedecorators.ShellDecorator{}); err != nil {
			t.Fatalf("restore shell decorator: %v", err)
		}
	})

	var monitored *decorator.MonitoredSession
	config := Config{
		sessionFactory: func(transportID string) (decorator.Session, error) {
			monitored = decorator.NewMonitoredSession(decorator.NewLocalSession())
			return monitored, nil
		},
	}

	plan := &planfmt.Plan{Target: "authority-shell", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.CommandNode{
			Decorator: "@shell",
			Args: []planfmt.Arg{{
				Key: "command",
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 3"},
			}},
		},
	}}}

	result, err := ExecutePlan(context.Background(), plan, config, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if diff := cmp.Diff(3, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if monitored == nil {
		t.Fatal("expected monitored session to be created")
	}
	if diff := cmp.Diff(1, monitored.Stats().RunCalls); diff != "" {
		t.Fatalf("run calls mismatch (-want +got):\n%s", diff)
	}
}

func TestExecuteNestedShellBypassesDecoratorExecPath(t *testing.T) {
	if err := decorator.Register("shell", &shellSentinelDecorator{}); err != nil {
		t.Fatalf("register sentinel shell decorator: %v", err)
	}
	t.Cleanup(func() {
		if err := decorator.Register("shell", &runtimedecorators.ShellDecorator{}); err != nil {
			t.Fatalf("restore shell decorator: %v", err)
		}
	})

	var monitored *decorator.MonitoredSession
	config := Config{
		sessionFactory: func(transportID string) (decorator.Session, error) {
			monitored = decorator.NewMonitoredSession(decorator.NewLocalSession())
			return monitored, nil
		},
	}

	plan := &planfmt.Plan{Target: "authority-nested-shell", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.CommandNode{
			Decorator: "@retry",
			Args: []planfmt.Arg{{
				Key: "times",
				Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 1},
			}},
			Block: []planfmt.Step{{
				ID: 2,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{{
						Key: "command",
						Val: planfmt.Value{Kind: planfmt.ValueString, Str: "exit 7"},
					}},
				},
			}},
		},
	}}}

	result, err := ExecutePlan(context.Background(), plan, config, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if diff := cmp.Diff(7, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}
	if monitored == nil {
		t.Fatal("expected monitored session to be created")
	}
	if diff := cmp.Diff(1, monitored.Stats().RunCalls); diff != "" {
		t.Fatalf("run calls mismatch (-want +got):\n%s", diff)
	}
}
