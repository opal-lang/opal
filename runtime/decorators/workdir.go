package decorators

import (
	"fmt"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/invariant"
)

// WorkdirDecorator implements the @workdir execution decorator.
type WorkdirDecorator struct{}

func (d *WorkdirDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("workdir").
		Summary("Execute block in a different working directory").
		Roles(decorator.RoleWrapper).
		ParamString("path", "Working directory for nested block").
		Required().
		Done().
		Block(decorator.BlockRequired).
		Build()
}

func (d *WorkdirDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &workdirNode{next: next, params: params}
}

type workdirNode struct {
	next   decorator.ExecNode
	params map[string]any
}

type workdirConfig struct {
	Path string `decorator:"path"`
}

func (n *workdirNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if n.next == nil {
		return decorator.Result{ExitCode: 0}, nil
	}

	invariant.NotNil(ctx.Session, "ctx.Session")

	cfg, _, err := decorator.DecodeInto[workdirConfig](
		(&WorkdirDecorator{}).Descriptor().Schema,
		nil,
		n.params,
	)
	if err != nil {
		return decorator.Result{ExitCode: decorator.ExitFailure}, err
	}
	if cfg.Path == "" {
		return decorator.Result{ExitCode: decorator.ExitFailure}, fmt.Errorf("@workdir path must not be empty")
	}

	child := ctx.WithSession(ctx.Session.WithWorkdir(cfg.Path))
	return n.next.Execute(child)
}

func init() {
	if err := decorator.Register("workdir", &WorkdirDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @workdir decorator: %v", err))
	}
}
