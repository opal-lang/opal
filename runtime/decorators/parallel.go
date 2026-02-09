package decorators

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/opal-lang/opal/core/decorator"
)

// ParallelDecorator implements the @parallel execution decorator.
// Executes tasks in parallel with optional concurrency limit.
type ParallelDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *ParallelDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("parallel").
		Summary("Execute tasks in parallel").
		Roles(decorator.RoleWrapper).
		ParamInt("maxConcurrency", "Maximum concurrent tasks (0=unlimited)").
		Min(0).
		Default(int64(0)).
		Examples("0", "5", "10").
		Done().
		ParamEnum("onFailure", "Failure behavior for parallel branches").
		Values("fail_fast", "wait_all").
		Default("fail_fast").
		Done().
		Block(decorator.BlockRequired).
		Build()
}

// Wrap implements the Exec interface.
func (d *ParallelDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &parallelNode{next: next, params: params}
}

// parallelNode wraps an execution node with parallel execution logic.
type parallelNode struct {
	next   decorator.ExecNode
	params map[string]any
}

type parallelConfig struct {
	MaxConcurrency int64  `decorator:"maxConcurrency"`
	OnFailure      string `decorator:"onFailure"`
}

// Execute implements the ExecNode interface.
func (n *parallelNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	if n.next == nil {
		return decorator.Result{ExitCode: 0}, nil
	}

	branchNode, ok := n.next.(decorator.BranchExecutor)
	if !ok {
		return n.next.Execute(ctx)
	}

	branchCount := branchNode.BranchCount()
	if branchCount == 0 {
		return decorator.Result{ExitCode: 0}, nil
	}

	cfg, _, err := decorator.DecodeInto[parallelConfig](
		(&ParallelDecorator{}).Descriptor().Schema,
		nil,
		n.params,
	)
	if err != nil {
		return decorator.Result{ExitCode: decorator.ExitFailure}, err
	}

	maxConcurrency := branchCount
	if cfg.MaxConcurrency > 0 && cfg.MaxConcurrency < int64(branchCount) {
		maxConcurrency = int(cfg.MaxConcurrency)
	}
	failureMode := cfg.OnFailure

	runCtx := ctx.Context
	if runCtx == nil {
		runCtx = context.Background()
	}
	cancel := func() {}
	if failureMode == "fail_fast" {
		runCtx, cancel = context.WithCancel(runCtx)
	}
	defer cancel()

	type branchResult struct {
		result decorator.Result
		err    error
		stdout []byte
		stderr []byte
	}

	results := make([]branchResult, branchCount)
	sem := make(chan struct{}, maxConcurrency)
	var cancelOnce sync.Once
	var wg sync.WaitGroup

	for i := 0; i < branchCount; i++ {
		wg.Add(1)
		branchIndex := i

		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var branchStdout bytes.Buffer
			var branchStderr bytes.Buffer

			branchCtx := ctx.
				WithContext(runCtx).
				WithIO(nil, &branchStdout, &branchStderr)

			result, err := branchNode.ExecuteBranch(branchIndex, branchCtx)
			results[branchIndex] = branchResult{
				result: result,
				err:    err,
				stdout: branchStdout.Bytes(),
				stderr: branchStderr.Bytes(),
			}

			if failureMode == "fail_fast" && (err != nil || result.ExitCode != 0) {
				cancelOnce.Do(cancel)
			}
		}()
	}

	wg.Wait()

	for _, branch := range results {
		if len(branch.stdout) > 0 && ctx.Stdout != nil {
			_, _ = ctx.Stdout.Write(branch.stdout)
		}
		if len(branch.stderr) > 0 && ctx.Stderr != nil {
			_, _ = ctx.Stderr.Write(branch.stderr)
		}
	}

	for _, branch := range results {
		if branch.err != nil {
			return branch.result, branch.err
		}
		if branch.result.ExitCode != 0 {
			return branch.result, nil
		}
	}

	return decorator.Result{ExitCode: 0}, nil
}

// Register @parallel decorator with the global registry
func init() {
	if err := decorator.Register("parallel", &ParallelDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @parallel decorator: %v", err))
	}
}
