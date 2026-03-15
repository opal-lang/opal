package executor

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/builtwithtofu/sigil/core/invariant"
	"github.com/builtwithtofu/sigil/core/planfmt"
	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	coreruntime "github.com/builtwithtofu/sigil/core/runtime"
	"github.com/builtwithtofu/sigil/core/sdk"
)

func (e *executor) executePlanStep(execCtx sdk.ExecutionContext, step planfmt.Step) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.Precondition(step.Tree != nil, "step must have a tree")
	if isExecutionCanceled(execCtx) {
		return coreruntime.ExitCanceled
	}

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
		if isExecutionCanceled(execCtx) {
			return coreruntime.ExitCanceled
		}
		exitCode := e.executePlanStep(execCtx, step)
		if exitCode != 0 {
			return exitCode
		}
	}
	return 0
}

func (e *executor) executePlanTreeIO(execCtx sdk.ExecutionContext, node planfmt.ExecutionNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	if isExecutionCanceled(execCtx) {
		return coreruntime.ExitCanceled
	}

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
		if leftExit == 0 || isExecutionCanceled(execCtx) {
			return leftExit
		}
		return e.executePlanTreeIO(execCtx, n.Right, stdin, stdout)

	case *planfmt.SequenceNode:
		var lastExit int
		for i, child := range n.Nodes {
			if isExecutionCanceled(execCtx) {
				return coreruntime.ExitCanceled
			}
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
		_, _ = fmt.Fprintln(e.stderr, "Error: try/catch/finally execution is not implemented")
		return coreruntime.ExitFailure

	default:
		invariant.Invariant(false, "unknown planfmt.ExecutionNode type: %T", node)
		return coreruntime.ExitFailure
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
		return coreruntime.ExitFailure
	}
}

func (e *executor) executePlanPipelineIO(execCtx sdk.ExecutionContext, pipeline *planfmt.PipelineNode, initialStdin io.Reader, finalStdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(pipeline, "pipeline")
	if isExecutionCanceled(execCtx) {
		return coreruntime.ExitCanceled
	}
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
			return coreruntime.ExitFailure
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
	if isExecutionCanceled(execCtx) {
		return coreruntime.ExitCanceled
	}

	commandExecCtx := withExecutionTransport(execCtx, cmd.TransportID)

	params, ok := e.resolveCommandParams(commandExecCtx, cmd.Decorator, planArgsToMap(cmd.Args))
	if !ok {
		return coreruntime.ExitFailure
	}

	if isShellDecorator(cmd.Decorator) {
		return e.executeShellWithParams(commandExecCtx, params, stdin, stdout)
	}

	decoratorName := normalizeDecoratorName(cmd.Decorator)
	capability := isPluginOnlyCapability(decoratorName)
	if capability == nil {
		_, _ = fmt.Fprintf(e.stderr, "Error: unknown decorator: %s\n", cmd.Decorator)
		return coreruntime.ExitFailure
	}

	var next coreruntime.ExecNode
	if len(cmd.Block) > 0 {
		next = &planBlockNode{executor: e, execCtx: commandExecCtx, steps: cmd.Block}
	}

	switch typed := capability.(type) {
	case coreplugin.Wrapper:
		return e.executePluginWrapper(commandExecCtx, next, typed, params, stdin, stdout)
	case coreplugin.Transport:
		return e.executePlanPluginTransport(commandExecCtx, cmd.Block, typed, params)
	default:
		_, _ = fmt.Fprintf(e.stderr, "Error: @%s is not executable\n", decoratorName)
		return coreruntime.ExitFailure
	}
}

func (e *executor) executePlanRedirect(execCtx sdk.ExecutionContext, redirect *planfmt.RedirectNode, stdin io.Reader) int {
	invariant.NotNil(execCtx, "execCtx")
	invariant.NotNil(redirect, "redirect node")
	if isExecutionCanceled(execCtx) {
		return coreruntime.ExitCanceled
	}

	redirectExecCtx := withExecutionTransport(execCtx, sourceTransportIDForPlan(redirect.Source))
	stderrOnly := planRedirectStderrEnabled(redirect.Source)
	transportID := executionTransportID(redirectExecCtx)

	redirectTarget, args, sinkIdentity, ok := resolvePlanIOSink(&redirect.Target, e.stderr)
	if !ok {
		return coreruntime.ExitFailure
	}

	caps := redirectTarget.RedirectCaps()
	if redirect.Mode == planfmt.RedirectInput && !caps.Read {
		_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "validate", TransportID: transportID, Cause: fmt.Errorf("does not support input (<)")})
		return coreruntime.ExitFailure
	}
	if redirect.Mode == planfmt.RedirectOverwrite && !caps.Write {
		_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "validate", TransportID: transportID, Cause: fmt.Errorf("does not support overwrite (>)")})
		return coreruntime.ExitFailure
	}
	if redirect.Mode == planfmt.RedirectAppend && !caps.Append {
		_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "validate", TransportID: transportID, Cause: fmt.Errorf("does not support append (>>)")})
		return coreruntime.ExitFailure
	}

	baseSession, sessionErr := e.sessions.SessionFor(executionTransportID(redirectExecCtx))
	if sessionErr != nil {
		_, _ = fmt.Fprintf(e.stderr, "Error creating session: %v\n", sessionErr)
		return coreruntime.ExitFailure
	}
	session := sessionForExecutionContext(baseSession, redirectExecCtx)
	if isExecutionCanceled(redirectExecCtx) {
		return coreruntime.ExitCanceled
	}

	pluginCtx := pluginExecContext{ctx: redirectExecCtx.Context(), session: pluginParentSession{session: session}, stdin: stdin, stdout: nil, stderr: e.stderr}
	resolver := func(displayID string) (string, error) {
		if e.vault == nil {
			return "", fmt.Errorf("secret resolver unavailable")
		}
		value, err := e.vault.ResolveDisplayIDWithTransport(displayID, executionTransportID(redirectExecCtx))
		if err != nil {
			return "", err
		}
		return fmt.Sprint(value), nil
	}
	resolvedArgs := newPluginArgs(args, redirectTarget.Schema(), resolver)

	if redirect.Mode == planfmt.RedirectInput {
		reader, err := redirectTarget.OpenForRead(pluginCtx, resolvedArgs)
		if err != nil {
			_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "open", TransportID: transportID, Cause: err})
			return coreruntime.ExitFailure
		}

		exitCode := e.executePlanTreeIO(redirectExecCtx, withPlanRedirectedStderrSource(redirect.Source, stderrOnly), reader, nil)
		if closeErr := reader.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "close", TransportID: transportID, Cause: closeErr})
			return coreruntime.ExitFailure
		}
		return exitCode
	}

	writer, err := redirectTarget.OpenForWrite(pluginCtx, resolvedArgs, redirect.Mode == planfmt.RedirectAppend)
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "open", TransportID: transportID, Cause: err})
		return coreruntime.ExitFailure
	}

	exitCode := e.executePlanTreeIO(redirectExecCtx, withPlanRedirectedStderrSource(redirect.Source, stderrOnly), stdin, writer)
	if closeErr := writer.Close(); closeErr != nil {
		_, _ = fmt.Fprintf(e.stderr, "Error: %v\n", SinkError{SinkID: sinkIdentity, Operation: "close", TransportID: transportID, Cause: closeErr})
		return coreruntime.ExitFailure
	}

	return exitCode
}

func planRedirectStderrEnabled(node planfmt.ExecutionNode) bool {
	cmd, ok := node.(*planfmt.CommandNode)
	if !ok {
		return false
	}

	for _, arg := range cmd.Args {
		if arg.Key == "stderr" && arg.Val.Kind == planfmt.ValueBool {
			return arg.Val.Bool
		}
	}

	return false
}

func withPlanRedirectedStderrSource(node planfmt.ExecutionNode, stderrOnly bool) planfmt.ExecutionNode {
	if !stderrOnly {
		return node
	}

	cmd, ok := node.(*planfmt.CommandNode)
	if !ok || normalizeDecoratorName(cmd.Decorator) != "shell" {
		return node
	}

	args := make([]planfmt.Arg, len(cmd.Args))
	copy(args, cmd.Args)

	for i := range args {
		if args[i].Key != "command" || args[i].Val.Kind != planfmt.ValueString || args[i].Val.Str == "" {
			continue
		}

		args[i].Val.Str = "(" + args[i].Val.Str + ") 3>&1 1>&2 2>&3 3>&-"
		return &planfmt.CommandNode{
			Decorator:   cmd.Decorator,
			TransportID: cmd.TransportID,
			Args:        args,
			Block:       cmd.Block,
		}
	}

	return node
}

func isExecutionCanceled(execCtx sdk.ExecutionContext) bool {
	if execCtx == nil {
		return false
	}
	ctx := execCtx.Context()
	if ctx == nil {
		return false
	}
	return ctx.Err() != nil
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
	case *planfmt.LogicNode:
		if len(n.Block) == 0 {
			return ""
		}
		return sourceTransportIDForPlan(n.Block[0].Tree)
	case *planfmt.TryNode:
		if len(n.TryBlock) > 0 {
			return sourceTransportIDForPlan(n.TryBlock[0].Tree)
		}
		if len(n.CatchBlock) > 0 {
			return sourceTransportIDForPlan(n.CatchBlock[0].Tree)
		}
		if len(n.FinallyBlock) > 0 {
			return sourceTransportIDForPlan(n.FinallyBlock[0].Tree)
		}
		return ""
	default:
		return ""
	}
}

type planBlockNode struct {
	executor *executor
	execCtx  sdk.ExecutionContext
	steps    []planfmt.Step
}

func (n *planBlockNode) Execute(ctx coreruntime.ExecContext) (coreruntime.Result, error) {
	child := childExecutionContextFromDecorator(n.execCtx, ctx)
	exitCode := n.executor.executePlanBlock(child, n.steps)
	return coreruntime.Result{ExitCode: exitCode}, nil
}

func (n *planBlockNode) BranchCount() int {
	return len(n.steps)
}

func (n *planBlockNode) ExecuteBranch(index int, ctx coreruntime.ExecContext) (coreruntime.Result, error) {
	invariant.Precondition(index >= 0 && index < len(n.steps), "branch index out of bounds: %d", index)

	child := childExecutionContextFromDecorator(n.execCtx, ctx)
	exitCode := n.executor.executePlanStep(child, n.steps[index])
	return coreruntime.Result{ExitCode: exitCode}, nil
}

func normalizeDecoratorName(name string) string {
	if name != "" && name[0] == '@' {
		return name[1:]
	}
	return name
}

func resolvePlanIOSink(target *planfmt.CommandNode, stderr io.Writer) (coreplugin.RedirectTarget, map[string]any, string, bool) {
	if target == nil {
		_, _ = fmt.Fprintln(stderr, "Error: redirect target is nil")
		return nil, nil, "", false
	}

	decoratorPath, args := normalizePlanIOArgs(target.Decorator, planArgsToMap(target.Args))
	if entry := coreplugin.Global().LookupEntry(decoratorPath); entry != nil && entry.IsRedirect() {
		capability, ok := coreplugin.Global().Lookup(decoratorPath).(coreplugin.RedirectTarget)
		if !ok {
			_, _ = fmt.Fprintf(stderr, "Error: decorator %s registered as redirect target but does not implement redirect interface\n", target.Decorator)
			return nil, nil, target.Decorator, false
		}

		identity := target.Decorator
		if decoratorPath == "file" {
			if path, ok := args["path"].(string); ok && path != "" {
				identity = "@file(" + path + ")"
			}
		}

		return capability, args, identity, true
	}

	_, _ = fmt.Fprintf(stderr, "Error: unknown redirect sink decorator: %s\n", target.Decorator)
	return nil, nil, target.Decorator, false
}

func normalizePlanIOArgs(decoratorName string, args map[string]any) (string, map[string]any) {
	normalizedDecorator := normalizeDecoratorName(decoratorName)
	if normalizedDecorator == "shell" {
		if path, ok := args["command"].(string); ok && path != "" {
			normalizedArgs := map[string]any{"path": path}
			if perm, ok := args["perm"]; ok {
				normalizedArgs["perm"] = perm
			}
			return "file", normalizedArgs
		}
	}

	return normalizedDecorator, args
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
