package execution

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/runtime/ir"
)

// ================================================================================================
// NODE EVALUATOR - IR Runtime Execution Engine
// ================================================================================================

// NodeEvaluator executes IR trees using registered decorators
// This is the core engine that provides semantic equivalence between interpreter and generated modes
type NodeEvaluator struct {
	registry *decorators.Registry
}

// ExecuteAction implements the ExecutionDelegate interface for action decorators
func (e *NodeEvaluator) ExecuteAction(ctx *decorators.Ctx, name string, args []decorators.DecoratorParam) decorators.CommandResult {
	// Convert decorator context back to execution context
	execCtx := e.toExecutionContext(ctx)

	// Execute the action using our existing executeAction method
	result := e.executeAction(execCtx, name, args)

	// Convert back to decorator result format
	return decorators.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// ExecuteBlock implements the ExecutionDelegate interface for block decorators
func (e *NodeEvaluator) ExecuteBlock(ctx *decorators.Ctx, name string, args []decorators.DecoratorParam, innerSteps []decorators.CommandStep) decorators.CommandResult {
	// Convert decorator context back to execution context
	execCtx := e.toExecutionContext(ctx)

	// Convert decorator command steps to execution command steps
	var execSteps []ir.CommandStep
	for _, step := range innerSteps {
		execStep := ir.CommandStep{
			Chain: make([]ir.ChainElement, len(step.Chain)),
		}
		for i, element := range step.Chain {
			// Convert Text to structured Content if it exists
			var content *ir.ElementContent
			if element.Text != "" {
				content = &ir.ElementContent{
					Parts: []ir.ContentPart{
						{
							Kind: ir.PartKindLiteral,
							Text: element.Text,
						},
					},
				}
			}

			execStep.Chain[i] = ir.ChainElement{
				Kind:       ir.ElementKind(element.Kind),
				Name:       element.Name,
				Content:    content,
				Args:       element.Args,
				InnerSteps: nil, // Block decorators shouldn't have nested inner steps in chain elements
				OpNext:     ir.ChainOp(element.OpNext),
				Target:     element.Target,
			}
		}
		execSteps = append(execSteps, execStep)
	}

	// Execute the block using our existing executeBlockDecorator method
	result := e.executeBlockDecorator(execCtx, name, args, execSteps)

	// Convert back to decorator result format
	return decorators.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// ExecuteCommand implements the ExecutionDelegate interface for command execution by name
func (e *NodeEvaluator) ExecuteCommand(ctx *decorators.Ctx, commandName string) decorators.CommandResult {
	// Convert decorator context to execution context
	execCtx := e.toExecutionContext(ctx)

	// Look up the command IR node from the context
	if execCtx.Commands == nil {
		return decorators.CommandResult{
			Stderr:   "Commands not available in execution context for @cmd",
			ExitCode: 1,
		}
	}

	irNode, exists := execCtx.Commands[commandName]
	if !exists {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("Command '%s' not found", commandName),
			ExitCode: 1,
		}
	}

	// Execute the IR node recursively
	result := e.EvaluateNode(execCtx, irNode)

	// Convert back to decorator result format
	return decorators.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// NewNodeEvaluator creates a new evaluator with the given registry
func NewNodeEvaluator(registry *decorators.Registry) *NodeEvaluator {
	return &NodeEvaluator{
		registry: registry,
	}
}

// resolveElementText resolves the text content from an IR ChainElement
func (e *NodeEvaluator) resolveElementText(ctx *ir.Ctx, element *ir.ChainElement) (string, error) {
	if element.Content == nil {
		return "", fmt.Errorf("shell element missing content")
	}

	// Convert IR context to decorator context for text resolution
	decoratorCtx := &decorators.Ctx{
		Env:  ctx.Env, // ir.EnvSnapshot implements decorators.EnvSnapshot interface
		Vars: ctx.Vars,
		// Note: Some fields are omitted but not needed for text resolution
	}

	return element.Content.Resolve(decoratorCtx, e.registry)
}

// EvaluateNode executes an IR node and returns the result
func (e *NodeEvaluator) EvaluateNode(ctx *ir.Ctx, node ir.Node) ir.CommandResult {
	switch n := node.(type) {
	case ir.CommandSeq:
		return e.evaluateCommandSeq(ctx, n)
	case ir.Wrapper:
		return e.evaluateWrapper(ctx, n)
	case ir.Pattern:
		return e.evaluatePattern(ctx, n)
	default:
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Unknown node type: %T", node),
			ExitCode: 1,
		}
	}
}

// evaluateCommandSeq executes a sequence of command steps
func (e *NodeEvaluator) evaluateCommandSeq(ctx *ir.Ctx, seq ir.CommandSeq) ir.CommandResult {
	var lastResult ir.CommandResult
	for _, step := range seq.Steps {
		lastResult = e.evaluateStep(ctx, step)
		if lastResult.Failed() {
			return lastResult // Short-circuit on failure
		}
	}
	return lastResult // Return the last result, not a new empty one
}

// evaluateWrapper executes a block decorator around inner content
func (e *NodeEvaluator) evaluateWrapper(ctx *ir.Ctx, wrapper ir.Wrapper) ir.CommandResult {
	// Get the block decorator from registry
	decorator, exists := e.registry.GetBlock(wrapper.Kind)
	if !exists {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Block decorator @%s not found", wrapper.Kind),
			ExitCode: 1,
		}
	}

	// Convert parameters to decorator format
	args := e.convertParams(wrapper.Params)

	// Convert inner CommandSeq to decorator format
	innerCommandSeq := e.convertToDecoratorCommandSeq(wrapper.Inner)

	// Convert execution context to decorator context
	decorCtx := e.toDecoratorContext(ctx)

	// Execute the wrapper with CommandSeq
	result := decorator.WrapCommands(decorCtx, args, innerCommandSeq)

	// Convert back to execution result
	return ir.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// evaluatePattern executes a pattern decorator (conditional execution)
func (e *NodeEvaluator) evaluatePattern(ctx *ir.Ctx, pattern ir.Pattern) ir.CommandResult {
	// Get the pattern decorator from registry
	decorator, exists := e.registry.GetPattern(pattern.Kind)
	if !exists {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Pattern decorator @%s not found", pattern.Kind),
			ExitCode: 1,
		}
	}

	// Convert parameters to decorator format
	args := e.convertParams(pattern.Params)

	// Build branches map with CommandSeq
	branches := make(map[string]decorators.CommandSeq)
	for branchName, branchSeq := range pattern.Branches {
		branches[branchName] = e.convertToDecoratorCommandSeq(branchSeq)
	}

	// Convert execution context to decorator context
	decorCtx := e.toDecoratorContext(ctx)

	// Execute the pattern
	result := decorator.SelectBranch(decorCtx, args, branches)

	// Convert back to execution result
	return ir.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// evaluateStep executes a single command step (chain of elements)
func (e *NodeEvaluator) evaluateStep(ctx *ir.Ctx, step ir.CommandStep) ir.CommandResult {
	if len(step.Chain) == 0 {
		return ir.CommandResult{ExitCode: 0} // Empty chain succeeds
	}

	var lastResult ir.CommandResult
	var combinedOutput strings.Builder

	for i, element := range step.Chain {
		// Check if we should execute this element based on previous operator
		if i > 0 {
			prevOp := step.Chain[i-1].OpNext
			if !e.shouldExecute(prevOp, lastResult) {
				continue
			}
		}

		// Execute the element
		var result ir.CommandResult
		switch ir.ElementKind(element.Kind) {
		case ir.ElementKindShell:
			// Resolve shell command text from IR element
			text, err := e.resolveElementText(ctx, &element)
			if err != nil {
				return ir.CommandResult{
					Stderr:   fmt.Sprintf("Failed to resolve shell command text: %v", err),
					ExitCode: 1,
				}
			}

			// Handle pipe operator - pass previous stdout as stdin
			if i > 0 && step.Chain[i-1].OpNext == ir.ChainOpPipe {
				result = e.executeShellWithInput(ctx, text, strings.NewReader(lastResult.Stdout))
			} else {
				result = e.executeShell(ctx, text)
			}
		case ir.ElementKindAction:
			result = e.executeAction(ctx, element.Name, element.Args)
		case ir.ElementKindBlock:
			// Special handling for block decorators - they have InnerSteps
			result = e.executeBlockDecorator(ctx, element.Name, element.Args, element.InnerSteps)
		case ir.ElementKindPattern:
			// Pattern decorators should not be in ChainElements anymore - they should be Pattern nodes
			result = ir.CommandResult{
				Stderr:   fmt.Sprintf("Pattern decorator @%s should be a Pattern node, not ir.ChainElement", element.Name),
				ExitCode: 1,
			}
		default:
			result = ir.CommandResult{
				Stderr:   fmt.Sprintf("Unknown element kind: %s", element.Kind),
				ExitCode: 1,
			}
		}

		// Handle output redirection
		if element.OpNext == ir.ChainOpAppend && element.Target != "" {
			err := e.appendToFile(element.Target, result.Stdout)
			if err != nil {
				result.Stderr = fmt.Sprintf("Failed to append to %s: %v", element.Target, err)
				result.ExitCode = 1
			} else {
				// Clear stdout since it was written to file
				result.Stdout = ""
			}
		}

		// Accumulate output based on operation type
		if i == 0 {
			// First element: include output unless it's piped to the next command
			if element.OpNext != ir.ChainOpPipe && result.Stdout != "" {
				combinedOutput.WriteString(result.Stdout)
			}
		} else {
			prevOp := step.Chain[i-1].OpNext
			switch prevOp {
			case ir.ChainOpPipe:
				// For pipes, include the output of the pipe target (current element)
				// unless this element is also piped to the next one
				if element.OpNext != ir.ChainOpPipe && result.Stdout != "" {
					combinedOutput.WriteString(result.Stdout)
				}
			case ir.ChainOpAnd, ir.ChainOpOr, ir.ChainOpNone:
				// For &&, ||, and no operator, always accumulate outputs
				if result.Stdout != "" {
					combinedOutput.WriteString(result.Stdout)
				}
			}
		}

		lastResult = result
	}

	// Return the result with combined output
	finalOutput := combinedOutput.String()

	return ir.CommandResult{
		Stdout:   finalOutput,
		Stderr:   lastResult.Stderr,
		ExitCode: lastResult.ExitCode,
	}
}

// executeShell runs a shell command with real-time streaming output
func (e *NodeEvaluator) executeShell(ctx *ir.Ctx, command string) ir.CommandResult {
	// Only expand value decorators in normal execution mode, not in dry-run
	var expandedCommand string
	if ctx.DryRun {
		expandedCommand = command // Keep original for plan display
	} else {
		// Expand value decorators before execution
		var err error
		expandedCommand, err = expandValueDecorators(ctx, command)
		if err != nil {
			return ir.CommandResult{
				Stderr:   fmt.Sprintf("Failed to expand value decorators: %v", err),
				ExitCode: 1,
			}
		}
	}

	if ctx.Debug {
		fmt.Printf("[DEBUG] Executing shell command: %s\n", expandedCommand)
	}

	cmd := exec.Command("sh", "-c", expandedCommand)
	cmd.Dir = ctx.WorkDir

	// For real-time streaming, we need separate pipes for stdout/stderr
	return e.executeWithStreaming(ctx, cmd)
}

// executeWithStreaming executes a command with real-time output streaming
// This provides both live feedback to the user and captures output for CommandResult
func (e *NodeEvaluator) executeWithStreaming(ctx *ir.Ctx, cmd *exec.Cmd) ir.CommandResult {
	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Failed to create stdout pipe: %v", err),
			ExitCode: 1,
		}
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Failed to create stderr pipe: %v", err),
			ExitCode: 1,
		}
	}

	// Set up environment if available
	if ctx.Env != nil {
		envMap := ctx.Env.GetAll()
		cmd.Env = make([]string, 0, len(envMap))
		for k, v := range envMap {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	} else {
		cmd.Env = os.Environ()
	}

	// Connect stdin if available
	if ctx.Stdin != nil {
		cmd.Stdin = ctx.Stdin
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Failed to start command: %v", err),
			ExitCode: 1,
		}
	}

	// Capture output while streaming
	var stdoutBuf, stderrBuf strings.Builder
	var wg sync.WaitGroup

	// Determine live output streams based on UI config
	var liveStdout, liveStderr io.Writer
	if ctx.UIConfig != nil && ctx.UIConfig.Quiet {
		// In quiet mode, don't stream stdout live, only capture for result
		liveStdout = nil
		liveStderr = ctx.Stderr // Still show errors in quiet mode
	} else {
		// Normal mode: stream both stdout and stderr live
		liveStdout = ctx.Stdout
		liveStderr = ctx.Stderr
	}

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stdoutPipe, liveStdout, &stdoutBuf, ctx.Debug, "STDOUT")
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stderrPipe, liveStderr, &stderrBuf, ctx.Debug, "STDERR")
	}()

	// Wait for streaming to complete
	wg.Wait()

	// Wait for command to finish
	err = cmd.Wait()

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	if ctx.Debug {
		fmt.Printf("[DEBUG] Command finished with exit code: %d\n", exitCode)
	}

	return ir.CommandResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
	}
}

// executeShellWithInput runs a shell command with provided stdin (for pipe operator)
func (e *NodeEvaluator) executeShellWithInput(ctx *ir.Ctx, command string, stdin io.Reader) ir.CommandResult {
	// Only expand value decorators in normal execution mode, not in dry-run
	var expandedCommand string
	if ctx.DryRun {
		expandedCommand = command // Keep original for plan display
	} else {
		// Expand value decorators before execution
		var err error
		expandedCommand, err = expandValueDecorators(ctx, command)
		if err != nil {
			return ir.CommandResult{
				Stderr:   fmt.Sprintf("Failed to expand value decorators: %v", err),
				ExitCode: 1,
			}
		}
	}

	if ctx.Debug {
		fmt.Printf("[DEBUG] Executing shell command with input: %s\n", expandedCommand)
	}

	// Create command
	cmd := exec.Command("sh", "-c", expandedCommand)

	// Set working directory
	if ctx.WorkDir != "" {
		cmd.Dir = ctx.WorkDir
	}

	// Apply environment
	if ctx.Env != nil {
		envMap := ctx.Env.GetAll()
		cmd.Env = make([]string, 0, len(envMap))
		for k, v := range envMap {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	} else {
		cmd.Env = os.Environ()
	}

	// Connect stdin
	cmd.Stdin = stdin

	// Create pipes for stdout and stderr capture and streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Failed to create stdout pipe: %v", err),
			ExitCode: 1,
		}
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Failed to create stderr pipe: %v", err),
			ExitCode: 1,
		}
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Failed to start command: %v", err),
			ExitCode: 1,
		}
	}

	// Capture output with streaming
	var stdoutBuf, stderrBuf strings.Builder
	var wg sync.WaitGroup

	// Determine live output streams based on UI config
	var liveStdout, liveStderr io.Writer
	if ctx.UIConfig != nil && ctx.UIConfig.Quiet {
		liveStdout = nil
		liveStderr = ctx.Stderr
	} else {
		liveStdout = ctx.Stdout
		liveStderr = ctx.Stderr
	}

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stdoutPipe, liveStdout, &stdoutBuf, ctx.Debug, "STDOUT")
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stderrPipe, liveStderr, &stderrBuf, ctx.Debug, "STDERR")
	}()

	// Wait for streaming to complete
	wg.Wait()

	// Wait for command to finish
	err = cmd.Wait()

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	if ctx.Debug {
		fmt.Printf("[DEBUG] Command with input finished with exit code: %d\n", exitCode)
	}

	return ir.CommandResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
	}
}

// streamOutput reads from a pipe and writes to both a live output stream and a buffer
// This enables real-time feedback while preserving output for the final result
func (e *NodeEvaluator) streamOutput(pipe io.ReadCloser, liveOutput io.Writer, buffer *strings.Builder, debug bool, label string) {
	defer func() { _ = pipe.Close() }()

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text() + "\n"

		// Write to live output for real-time feedback (if not quiet mode)
		if liveOutput != nil {
			_, _ = liveOutput.Write([]byte(line))
		}

		// Capture in buffer for final CommandResult
		buffer.WriteString(line)

		if debug {
			fmt.Printf("[DEBUG] %s: %s", label, line)
		}
	}

	if err := scanner.Err(); err != nil && debug {
		fmt.Printf("[DEBUG] Error reading %s: %v\n", label, err)
	}
}

// executeAction runs an action decorator
func (e *NodeEvaluator) executeAction(ctx *ir.Ctx, name string, args []decorators.DecoratorParam) ir.CommandResult {
	// Get the action decorator from registry
	decorator, exists := e.registry.GetAction(name)
	if !exists {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Action decorator @%s not found", name),
			ExitCode: 1,
		}
	}

	// Convert execution context to decorator context
	decorCtx := e.toDecoratorContext(ctx)

	// Execute the action
	result := decorator.Run(decorCtx, args)

	// Convert back to execution result
	return ir.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// executeBlockDecorator runs a block decorator with inner steps
func (e *NodeEvaluator) executeBlockDecorator(ctx *ir.Ctx, name string, args []decorators.DecoratorParam, innerSteps []ir.CommandStep) ir.CommandResult {
	// Get the block decorator from registry
	decorator, exists := e.registry.GetBlock(name)
	if !exists {
		return ir.CommandResult{
			Stderr:   fmt.Sprintf("Block decorator @%s not found", name),
			ExitCode: 1,
		}
	}

	// Convert inner steps to CommandSeq
	innerCommandSeq := e.convertToDecoratorCommandSeq(ir.CommandSeq{Steps: innerSteps})

	// Convert execution context to decorator context
	decorCtx := e.toDecoratorContext(ctx)

	// Execute the block decorator with CommandSeq
	result := decorator.WrapCommands(decorCtx, args, innerCommandSeq)

	// Convert back to execution result
	return ir.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
}

// shouldExecute determines if an element should execute based on the previous operator and result
func (e *NodeEvaluator) shouldExecute(op ir.ChainOp, prevResult ir.CommandResult) bool {
	switch op {
	case ir.ChainOpAnd: // &&
		return prevResult.Success()
	case ir.ChainOpOr: // ||
		return prevResult.Failed()
	case ir.ChainOpPipe: // |
		return true // Always execute for pipes
	case ir.ChainOpAppend: // >>
		return true // Always execute for append
	default:
		return true
	}
}

// convertToDecoratorCommandSeq converts execution CommandSeq to decorator CommandSeq
func (e *NodeEvaluator) convertToDecoratorCommandSeq(seq ir.CommandSeq) decorators.CommandSeq {
	var decoratorSteps []decorators.CommandStep

	for _, step := range seq.Steps {
		decoratorStep := decorators.CommandStep{
			Chain: make([]decorators.ChainElement, len(step.Chain)),
		}

		for i, element := range step.Chain {
			// Convert inner steps if they exist (for block decorators inside pattern branches)
			var innerSteps []decorators.CommandStep
			for _, innerStep := range element.InnerSteps {
				convertedInnerStep := decorators.CommandStep{
					Chain: make([]decorators.ChainElement, len(innerStep.Chain)),
				}
				for j, innerElem := range innerStep.Chain {
					convertedInnerStep.Chain[j] = decorators.ChainElement{
						Kind:   decorators.ElementKind(innerElem.Kind),
						Name:   innerElem.Name,
						Text:   e.irElementToText(&innerElem),
						Args:   innerElem.Args,
						OpNext: decorators.ChainOp(innerElem.OpNext),
						Target: innerElem.Target,
						// Note: Not converting nested InnerSteps to avoid infinite recursion
					}
				}
				innerSteps = append(innerSteps, convertedInnerStep)
			}

			decoratorStep.Chain[i] = decorators.ChainElement{
				Kind:       decorators.ElementKind(element.Kind),
				Name:       element.Name,
				Text:       e.irElementToText(&element),
				Args:       element.Args,
				InnerSteps: innerSteps, // Include converted inner steps
				OpNext:     decorators.ChainOp(element.OpNext),
				Target:     element.Target,
			}
		}

		decoratorSteps = append(decoratorSteps, decoratorStep)
	}

	return decorators.CommandSeq{
		Steps: decoratorSteps,
	}
}

// convertParams converts parameter map to decorator parameters
func (e *NodeEvaluator) convertParams(params map[string]interface{}) []decorators.DecoratorParam {
	var result []decorators.DecoratorParam
	for name, value := range params {
		result = append(result, decorators.DecoratorParam{
			Name:  name,
			Value: value,
		})
	}
	return result
}

// toDecoratorContext converts execution context to decorator context
func (e *NodeEvaluator) toDecoratorContext(ctx *ir.Ctx) *decorators.Ctx {
	// Convert UI config to decorator format
	var ui *decorators.UIConfig
	if ctx.UIConfig != nil {
		ui = &decorators.UIConfig{
			ColorMode:   ctx.UIConfig.ColorMode,
			Quiet:       ctx.UIConfig.Quiet,
			Verbose:     ctx.UIConfig.Verbose,
			Interactive: ctx.UIConfig.Interactive,
			AutoConfirm: ctx.UIConfig.AutoConfirm,
			CI:          ctx.UIConfig.CI,
		}
	}

	return &decorators.Ctx{
		Env:      ctx.Env,
		Vars:     ctx.Vars,
		WorkDir:  ctx.WorkDir,
		Stdout:   ctx.Stdout,
		Stderr:   ctx.Stderr,
		Stdin:    ctx.Stdin,
		DryRun:   ctx.DryRun,
		Debug:    ctx.Debug,
		UI:       ui,
		Executor: e, // Set the evaluator as the execution delegate
	}
}

// toExecutionContext converts decorator context back to execution context
func (e *NodeEvaluator) toExecutionContext(ctx *decorators.Ctx) *ir.Ctx {
	// Convert UI config to execution format
	var ui *ir.UIConfig
	if ctx.UI != nil {
		ui = &ir.UIConfig{
			ColorMode:   ctx.UI.ColorMode,
			Quiet:       ctx.UI.Quiet,
			Verbose:     ctx.UI.Verbose,
			Interactive: ctx.UI.Interactive,
			AutoConfirm: ctx.UI.AutoConfirm,
			CI:          ctx.UI.CI,
		}
	}

	// Cast the EnvSnapshot interface to concrete type
	// This is safe because we control both the creation and usage
	var env *ir.EnvSnapshot
	if ctx.Env != nil {
		if snapshot, ok := ctx.Env.(*ir.EnvSnapshot); ok {
			env = snapshot
		}
	}

	return &ir.Ctx{
		Env:      env,
		Vars:     ctx.Vars,
		WorkDir:  ctx.WorkDir,
		Stdout:   ctx.Stdout,
		Stderr:   ctx.Stderr,
		Stdin:    ctx.Stdin,
		DryRun:   ctx.DryRun,
		Debug:    ctx.Debug,
		UIConfig: ui,
	}
}

// ================================================================================================
// VALUE DECORATOR EXPANSION
// ================================================================================================

// expandValueDecorators expands @var and @env placeholders in shell commands
func expandValueDecorators(ctx *ir.Ctx, command string) (string, error) {
	// Pattern to match @var(name=VAR_NAME) and @env(key=ENV_KEY) placeholders
	// The transform code creates placeholders like: @var(name=BUILD_DIR) or @env(key=HOME)

	if ctx.Debug {
		fmt.Printf("[DEBUG] expandValueDecorators input: %q\n", command)
	}

	result := command

	// We need to find and replace value decorator placeholders
	// For now, implement a simple regex-based approach

	// Import regex at the top if needed
	var err error

	// Handle @var(name=VAR_NAME) patterns
	result, err = expandVarDecorators(ctx, result)
	if err != nil {
		return "", fmt.Errorf("failed to expand @var decorators: %w", err)
	}

	// Handle @env(key=ENV_KEY) patterns
	result, err = expandEnvDecorators(ctx, result)
	if err != nil {
		return "", fmt.Errorf("failed to expand @env decorators: %w", err)
	}

	return result, nil
}

// expandVarDecorators expands @var(name=VAR_NAME) placeholders
func expandVarDecorators(ctx *ir.Ctx, command string) (string, error) {
	// Match @var(name=VAR_NAME) patterns
	re := regexp.MustCompile(`@var\(name=([^)]+)\)`)

	return re.ReplaceAllStringFunc(command, func(match string) string {
		// Extract variable name from the match
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // Keep original if can't parse
		}

		varName := submatches[1]

		// Look up variable in context
		if value, exists := ctx.Vars[varName]; exists {
			return value
		}

		// Variable not found - this should cause an error
		// Return a safe error placeholder that won't break shell syntax
		return fmt.Sprintf("UNDEFINED_VAR_%s", varName)
	}), nil
}

// expandEnvDecorators expands @env(key=ENV_KEY) and more complex patterns
func expandEnvDecorators(ctx *ir.Ctx, command string) (string, error) {
	// Match @env(key=ENV_KEY, default=DEFAULT_VALUE) patterns
	// This matches the format created by the transform code
	re := regexp.MustCompile(`@env\(key=([^,)]+)(?:,\s*default=([^)]+))?\)`)

	result := re.ReplaceAllStringFunc(command, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // Keep original if can't parse
		}

		envKey := submatches[1]
		defaultValue := ""
		if len(submatches) >= 3 && submatches[2] != "" {
			defaultValue = submatches[2]
		}

		if ctx.Debug {
			fmt.Printf("[DEBUG] @env expansion: key=%q, default=%q\n", envKey, defaultValue)
		}

		// Look up environment variable in frozen context
		if value, exists := ctx.Env.Get(envKey); exists && value != "" {
			if ctx.Debug {
				fmt.Printf("[DEBUG] @env found value: %q\n", value)
			}
			return value
		}

		// Environment variable not found or empty - use default
		if ctx.Debug {
			fmt.Printf("[DEBUG] @env using default: %q\n", defaultValue)
		}
		return defaultValue
	})

	return result, nil
}

// ================================================================================================
// FILE OPERATIONS
// ================================================================================================

// appendToFile appends content to a file, creating it if it doesn't exist
func (e *NodeEvaluator) appendToFile(filepath, content string) error {
	// Create directories if they don't exist
	if lastSlash := strings.LastIndex(filepath, "/"); lastSlash != -1 {
		dir := filepath[:lastSlash]
		if dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	// Open file for appending, create if it doesn't exist
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer func() { _ = file.Close() }()

	// Write content with newline if content doesn't end with one
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filepath, err)
	}

	return nil
}

// irElementToText converts an IR ChainElement to a text string suitable for decorators.ChainElement.Text
func (e *NodeEvaluator) irElementToText(element *ir.ChainElement) string {
	if element.Content == nil {
		// For non-shell elements (action decorators), return empty text
		return ""
	}

	// Build the text from content parts (this is a simplified conversion)
	// In a full implementation, this would resolve value decorators properly
	var text strings.Builder
	for _, part := range element.Content.Parts {
		if part.Kind == ir.PartKindLiteral {
			text.WriteString(part.Text)
		} else {
			// For decorator parts, create a placeholder representation
			text.WriteString("@" + part.DecoratorName + "()")
		}
	}
	return text.String()
}
