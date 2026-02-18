package executor

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/core/sdk"
)

func (e *executor) executePlanStep(execCtx sdk.ExecutionContext, step planfmt.Step) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.Precondition(step.Tree != nil, "step must have a tree")

	if logic, ok := step.Tree.(*planfmt.LogicNode); ok {
		return e.executePlanBlock(execCtx, logic.Block)
	}

	var stdin io.Reader
	var stdout io.Writer
	if ec, ok := execCtx.(*executionContext); ok {
		stdin = ec.stdin
		stdout = ec.stdoutPipe
	}

	return e.executePlanTreeIO(execCtx, step.Tree, stdin, stdout)
}

func (e *executor) executePlanBlock(execCtx sdk.ExecutionContext, steps []planfmt.Step) int {
	for _, step := range steps {
		exitCode := e.executePlanStep(execCtx, step)
		if exitCode != 0 {
			return exitCode
		}
	}
	return 0
}

func (e *executor) executePlanTreeIO(execCtx sdk.ExecutionContext, node planfmt.ExecutionNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")

	switch n := node.(type) {
	case *planfmt.CommandNode:
		return e.executePlanCommandWithPipes(execCtx, n, stdin, stdout)

	case *planfmt.PipelineNode:
		return e.executePlanPipelineIO(execCtx, n, stdin, stdout)

	case *planfmt.AndNode:
		leftExit := e.executePlanTreeIO(execCtx, n.Left, nil, stdout)
		if leftExit != 0 {
			return leftExit
		}
		return e.executePlanTreeIO(execCtx, n.Right, stdin, stdout)

	case *planfmt.OrNode:
		leftExit := e.executePlanTreeIO(execCtx, n.Left, nil, stdout)
		if leftExit == 0 {
			return leftExit
		}
		return e.executePlanTreeIO(execCtx, n.Right, stdin, stdout)

	case *planfmt.SequenceNode:
		var lastExit int
		for i, child := range n.Nodes {
			childStdin := io.Reader(nil)
			if i == len(n.Nodes)-1 {
				childStdin = stdin
			}
			lastExit = e.executePlanTreeIO(execCtx, child, childStdin, stdout)
		}
		return lastExit

	case *planfmt.RedirectNode:
		return e.executePlanRedirect(execCtx, n, stdin)

	case *planfmt.LogicNode:
		return e.executePlanBlock(execCtx, n.Block)

	case *planfmt.TryNode:
		fmt.Fprintln(os.Stderr, "Error: try/catch/finally execution is not implemented")
		return decorator.ExitFailure

	default:
		invariant.Invariant(false, "unknown planfmt.ExecutionNode type: %T", node)
		return decorator.ExitFailure
	}
}

func (e *executor) executePlanTreeNode(execCtx sdk.ExecutionContext, node planfmt.ExecutionNode, stdin io.Reader, stdout io.Writer) int {
	switch n := node.(type) {
	case *planfmt.CommandNode:
		return e.executePlanCommandWithPipes(execCtx, n, stdin, stdout)
	case *planfmt.RedirectNode:
		return e.executePlanRedirect(execCtx, n, stdin)
	default:
		invariant.Invariant(false, "invalid pipeline element type %T", node)
		return decorator.ExitFailure
	}
}

func (e *executor) executePlanPipelineIO(execCtx sdk.ExecutionContext, pipeline *planfmt.PipelineNode, initialStdin io.Reader, finalStdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(pipeline, "pipeline")
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")
	for i, node := range pipeline.Commands {
		invariant.Precondition(node != nil, "pipeline command %d cannot be nil", i)
		switch node.(type) {
		case *planfmt.CommandNode, *planfmt.RedirectNode:
			// valid pipeline node types
		default:
			invariant.Precondition(false, "invalid pipeline element type %T", node)
		}
	}

	if numCommands == 1 {
		return e.executePlanTreeNode(execCtx, pipeline.Commands[0], initialStdin, finalStdout)
	}

	pipeReaders := make([]*os.File, numCommands-1)
	pipeWriters := make([]*os.File, numCommands-1)
	for i := 0; i < numCommands-1; i++ {
		pr, pw, err := os.Pipe()
		if err != nil {
			for j := 0; j < i; j++ {
				_ = pipeReaders[j].Close()
				_ = pipeWriters[j].Close()
			}
			return decorator.ExitFailure
		}
		pipeReaders[i] = pr
		pipeWriters[i] = pw
	}

	exitCodes := make([]int, numCommands)
	pipeReaderCloseOnce := make([]sync.Once, numCommands-1)
	pipeWriterCloseOnce := make([]sync.Once, numCommands-1)

	var wg sync.WaitGroup
	wg.Add(numCommands)

	for i := 0; i < numCommands; i++ {
		cmdIndex := i
		node := pipeline.Commands[i]

		go func() {
			defer wg.Done()

			var stdin io.Reader
			if cmdIndex == 0 {
				stdin = initialStdin
			} else {
				stdin = pipeReaders[cmdIndex-1]
				defer func() {
					idx := cmdIndex - 1
					pipeReaderCloseOnce[idx].Do(func() {
						_ = pipeReaders[idx].Close()
					})
				}()
			}

			var stdout io.Writer
			if cmdIndex < numCommands-1 {
				stdout = pipeWriters[cmdIndex]
				defer func() {
					idx := cmdIndex
					pipeWriterCloseOnce[idx].Do(func() {
						_ = pipeWriters[idx].Close()
					})
				}()
			} else if finalStdout != nil {
				stdout = finalStdout
			} else {
				stdout = os.Stdout
			}

			exitCodes[cmdIndex] = e.executePlanTreeNode(execCtx, node, stdin, stdout)
		}()
	}

	wg.Wait()
	return exitCodes[numCommands-1]
}

func (e *executor) executePlanCommandWithPipes(execCtx sdk.ExecutionContext, cmd *planfmt.CommandNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	commandExecCtx := withExecutionTransport(execCtx, cmd.TransportID)

	params, ok := e.resolveCommandParams(commandExecCtx, cmd.Decorator, planArgsToMap(cmd.Args))
	if !ok {
		return decorator.ExitFailure
	}

	if isShellDecorator(cmd.Decorator) {
		return e.executeShellWithParams(commandExecCtx, params, stdin, stdout)
	}

	decoratorName := normalizeDecoratorName(cmd.Decorator)
	entry, exists := decorator.Global().Lookup(decoratorName)
	invariant.Invariant(exists, "unknown decorator: %s", cmd.Decorator)

	execDec, ok := entry.Impl.(decorator.Exec)
	invariant.Invariant(ok, "%s is not an execution decorator", cmd.Decorator)

	return e.executePlanDecorator(commandExecCtx, cmd, execDec, params, stdin, stdout)
}

func (e *executor) executePlanDecorator(
	execCtx sdk.ExecutionContext,
	cmd *planfmt.CommandNode,
	execDec decorator.Exec,
	params map[string]any,
	stdin io.Reader,
	stdout io.Writer,
) int {
	var next decorator.ExecNode
	if len(cmd.Block) > 0 {
		next = &planBlockNode{executor: e, execCtx: execCtx, steps: cmd.Block}
	}

	node := execDec.Wrap(next, params)
	if node == nil {
		if next == nil {
			return 0
		}
		node = next
	}

	baseSession, sessionErr := e.sessions.SessionFor(executionTransportID(execCtx))
	if sessionErr != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", sessionErr)
		return decorator.ExitFailure
	}
	session := sessionForExecutionContext(baseSession, execCtx)

	if stdout == nil {
		stdout = os.Stdout
	}

	decoratorExecCtx := decorator.ExecContext{
		Context: execCtx.Context(),
		Session: session,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  os.Stderr,
		Trace:   nil,
	}

	result, err := node.Execute(decoratorExecCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return result.ExitCode
}

func (e *executor) executePlanRedirect(execCtx sdk.ExecutionContext, redirect *planfmt.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")

	redirectExecCtx := withExecutionTransport(execCtx, sourceTransportIDForPlan(redirect.Source))

	ioDecorator, sinkIdentity, ok := resolvePlanIOSink(&redirect.Target)
	if !ok {
		return decorator.ExitFailure
	}

	caps := ioDecorator.IOCaps()
	if redirect.Mode == planfmt.RedirectOverwrite && !caps.Write {
		fmt.Fprintf(os.Stderr, "Error: sink %s does not support overwrite (>)\n", sinkIdentity)
		return decorator.ExitFailure
	}
	if redirect.Mode == planfmt.RedirectAppend && !caps.Append {
		fmt.Fprintf(os.Stderr, "Error: sink %s does not support append (>>)\n", sinkIdentity)
		return decorator.ExitFailure
	}

	baseSession, sessionErr := e.sessions.SessionFor(executionTransportID(redirectExecCtx))
	if sessionErr != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", sessionErr)
		return decorator.ExitFailure
	}
	session := sessionForExecutionContext(baseSession, redirectExecCtx)

	writer, err := ioDecorator.OpenWrite(decorator.ExecContext{
		Context: redirectExecCtx.Context(),
		Session: session,
		Stderr:  os.Stderr,
	}, redirect.Mode == planfmt.RedirectAppend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open sink %s: %v\n", sinkIdentity, err)
		return decorator.ExitFailure
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to close sink %s: %v\n", sinkIdentity, closeErr)
		}
	}()

	return e.executePlanTreeIO(redirectExecCtx, redirect.Source, stdin, writer)
}

func sourceTransportIDForPlan(node planfmt.ExecutionNode) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *planfmt.CommandNode:
		return n.TransportID
	case *planfmt.PipelineNode:
		if len(n.Commands) == 0 {
			return ""
		}
		return sourceTransportIDForPlan(n.Commands[0])
	case *planfmt.RedirectNode:
		return sourceTransportIDForPlan(n.Source)
	case *planfmt.AndNode:
		return sourceTransportIDForPlan(n.Left)
	case *planfmt.OrNode:
		return sourceTransportIDForPlan(n.Left)
	case *planfmt.SequenceNode:
		if len(n.Nodes) == 0 {
			return ""
		}
		return sourceTransportIDForPlan(n.Nodes[0])
	default:
		return ""
	}
}

type planBlockNode struct {
	executor *executor
	execCtx  sdk.ExecutionContext
	steps    []planfmt.Step
}

func (n *planBlockNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	child := childExecutionContextFromDecorator(n.execCtx, ctx)
	exitCode := n.executor.executePlanBlock(child, n.steps)
	return decorator.Result{ExitCode: exitCode}, nil
}

func (n *planBlockNode) BranchCount() int {
	return len(n.steps)
}

func (n *planBlockNode) ExecuteBranch(index int, ctx decorator.ExecContext) (decorator.Result, error) {
	invariant.Precondition(index >= 0 && index < len(n.steps), "branch index out of bounds: %d", index)

	child := childExecutionContextFromDecorator(n.execCtx, ctx)
	exitCode := n.executor.executePlanStep(child, n.steps[index])
	return decorator.Result{ExitCode: exitCode}, nil
}

func normalizeDecoratorName(name string) string {
	if name != "" && name[0] == '@' {
		return name[1:]
	}
	return name
}

func resolvePlanIOSink(target *planfmt.CommandNode) (decorator.IO, string, bool) {
	if target == nil {
		fmt.Fprintln(os.Stderr, "Error: redirect target is nil")
		return nil, "", false
	}

	decoratorPath := normalizeDecoratorName(target.Decorator)
	entry, exists := decorator.Global().Lookup(decoratorPath)
	if !exists {
		fmt.Fprintf(os.Stderr, "Error: unknown redirect sink decorator: %s\n", target.Decorator)
		return nil, target.Decorator, false
	}

	ioDecorator, ok := entry.Impl.(decorator.IO)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: decorator %s does not support redirect sink I/O\n", target.Decorator)
		return nil, target.Decorator, false
	}

	args := planArgsToMap(target.Args)
	if factory, ok := ioDecorator.(decorator.IOFactory); ok {
		ioDecorator = factory.WithParams(args)
	}

	identity := target.Decorator
	if commandPath, ok := args["command"].(string); ok && commandPath != "" {
		identity = identity + "(" + commandPath + ")"
	}

	return ioDecorator, identity, true
}

func planArgsToMap(args []planfmt.Arg) map[string]any {
	out := make(map[string]any, len(args))
	for _, arg := range args {
		out[arg.Key] = planValueToAny(arg.Val)
	}
	return out
}

func planValueToAny(val planfmt.Value) any {
	switch val.Kind {
	case planfmt.ValueString:
		return val.Str
	case planfmt.ValueInt:
		return val.Int
	case planfmt.ValueBool:
		return val.Bool
	case planfmt.ValueFloat:
		return val.Float
	case planfmt.ValueDuration:
		return val.Duration
	case planfmt.ValueArray:
		items := make([]any, len(val.Array))
		for i, item := range val.Array {
			items[i] = planValueToAny(item)
		}
		return items
	case planfmt.ValueMap:
		mapped := make(map[string]any, len(val.Map))
		for key, item := range val.Map {
			mapped[key] = planValueToAny(item)
		}
		return mapped
	case planfmt.ValuePlaceholder:
		invariant.Invariant(false, "ValuePlaceholder reached execution runtime (ref=%d)", val.Ref)
		return fmt.Sprintf("$%d", val.Ref)
	default:
		return nil
	}
}
