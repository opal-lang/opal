package builtins

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// ParallelWrapperCapability wraps branch execution in parallel.
type ParallelWrapperCapability struct{}

func (c ParallelWrapperCapability) Path() string { return "exec.parallel" }

func (c ParallelWrapperCapability) Schema() plugin.Schema {
	minValue := float64(0)
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "maxConcurrency", Type: types.TypeInt, Default: 0, Minimum: &minValue},
			{Name: "onFailure", Type: types.TypeString, Default: "fail_fast", Enum: []string{"fail_fast", "wait_all"}},
		},
		Block: plugin.BlockRequired,
	}
}

func (c ParallelWrapperCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return parallelNode{next: next, maxConcurrency: args.GetInt("maxConcurrency"), onFailure: args.GetString("onFailure")}
}

type parallelNode struct {
	next           plugin.ExecNode
	maxConcurrency int
	onFailure      string
}

type parallelBranchResult struct {
	result plugin.Result
	err    error
	stdout []byte
	stderr []byte
}

func (n parallelNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	if n.next == nil {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}

	branchNode, ok := n.next.(plugin.BranchExecutor)
	if !ok {
		return n.next.Execute(ctx)
	}

	branchCount := branchNode.BranchCount()
	if branchCount == 0 {
		return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
	}

	maxConcurrency := branchCount
	if n.maxConcurrency > 0 && n.maxConcurrency < branchCount {
		maxConcurrency = n.maxConcurrency
	}

	runCtx := ctx.Context()
	if runCtx == nil {
		runCtx = context.Background()
	}
	cancel := func() {}
	if n.onFailure == "fail_fast" {
		runCtx, cancel = context.WithCancel(runCtx)
	}
	defer cancel()

	results := make([]parallelBranchResult, branchCount)
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var cancelOnce sync.Once

	for i := 0; i < branchCount; i++ {
		wg.Add(1)
		branchIndex := i
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var branchStdout bytes.Buffer
			var branchStderr bytes.Buffer
			branchCtx := parallelExecContext{ctx: runCtx, session: ctx.Session(), stdin: nil, stdout: &branchStdout, stderr: &branchStderr}

			result, err := branchNode.ExecuteBranch(branchIndex, branchCtx)
			results[branchIndex] = parallelBranchResult{result: result, err: err, stdout: branchStdout.Bytes(), stderr: branchStderr.Bytes()}

			if n.onFailure == "fail_fast" && (err != nil || result.ExitCode != plugin.ExitSuccess) {
				cancelOnce.Do(cancel)
			}
		}()
	}

	wg.Wait()

	for _, branch := range results {
		if len(branch.stdout) > 0 && ctx.Stdout() != nil {
			_, _ = ctx.Stdout().Write(branch.stdout)
		}
		if len(branch.stderr) > 0 && ctx.Stderr() != nil {
			_, _ = ctx.Stderr().Write(branch.stderr)
		}
	}

	for _, branch := range results {
		if branch.err != nil {
			return branch.result, branch.err
		}
		if branch.result.ExitCode != plugin.ExitSuccess {
			return branch.result, nil
		}
	}

	return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
}

type parallelExecContext struct {
	ctx     context.Context
	session plugin.ParentTransport
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (c parallelExecContext) Context() context.Context        { return c.ctx }
func (c parallelExecContext) Session() plugin.ParentTransport { return c.session }
func (c parallelExecContext) Stdin() io.Reader                { return c.stdin }
func (c parallelExecContext) Stdout() io.Writer               { return c.stdout }
func (c parallelExecContext) Stderr() io.Writer               { return c.stderr }
