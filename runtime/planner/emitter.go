package planner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/vault"
)

// Emitter transforms a resolved IR tree into a planfmt.Plan.
// It traverses the pruned tree from the Resolver and builds:
// - planfmt.Step for each command
// - SecretUses for contract verification
// - DisplayID placeholders for secrets
type Emitter struct {
	result           *ResolveResult
	vault            *vault.Vault
	scopes           *ScopeStack
	decoratorExprIDs map[string]string
	target           string

	// State
	nextStepID      uint64
	sitePath        []string
	decoratorCounts []map[string]int
	secretUses      []planfmt.SecretUse
}

// NewEmitter creates a new Emitter.
func NewEmitter(result *ResolveResult, v *vault.Vault, scopes *ScopeStack, target string) *Emitter {
	var decoratorExprIDs map[string]string
	if result != nil {
		decoratorExprIDs = result.DecoratorExprIDs
	}

	return &Emitter{
		result:           result,
		vault:            v,
		scopes:           scopes,
		decoratorExprIDs: decoratorExprIDs,
		target:           target,
		nextStepID:       1,
		sitePath:         []string{"root"},
		decoratorCounts:  []map[string]int{},
	}
}

// Emit transforms the resolved IR into a Plan.
func (e *Emitter) Emit() (*planfmt.Plan, error) {
	if e.result == nil {
		return nil, fmt.Errorf("cannot emit: no resolve result")
	}

	plan := planfmt.NewPlan()
	plan.Target = e.target

	// Emit all statements
	steps, err := e.emitStatements(e.result.Statements)
	if err != nil {
		return nil, err
	}
	plan.Steps = steps

	// Add collected SecretUses
	plan.SecretUses = e.secretUses

	return plan, nil
}

// emitStatements emits a list of statements, returning the resulting steps.
// Consecutive commands chained by operators (&&, ||, |, ;) are grouped into a single step.
func (e *Emitter) emitStatements(stmts []*StatementIR) ([]planfmt.Step, error) {
	var steps []planfmt.Step
	i := 0

	for i < len(stmts) {
		stmt := stmts[i]

		switch stmt.Kind {
		case StmtCommand:
			// Collect all chained commands (commands connected by operators)
			if stmt.Command == nil {
				return nil, fmt.Errorf("nil command in StmtCommand at index %d", i)
			}
			chain := []*CommandStmtIR{stmt.Command}
			for i+1 < len(stmts) && stmt.Command.Operator != "" && stmts[i+1].Kind == StmtCommand && stmts[i+1].Command != nil {
				i++
				chain = append(chain, stmts[i].Command)
				stmt = stmts[i]
			}

			// Build step from command chain
			step, err := e.emitCommandChain(chain)
			if err != nil {
				return nil, err
			}
			steps = append(steps, *step)

		case StmtVarDecl:
			// Variable declarations don't produce steps
			// (values already resolved, just need to track for DisplayID lookup)
			if stmt.VarDecl != nil && stmt.VarDecl.ExprID != "" && e.scopes != nil {
				e.scopes.Define(stmt.VarDecl.Name, stmt.VarDecl.ExprID)
			}

		case StmtBlocker:
			blockerSteps, err := e.emitBlocker(stmt.Blocker)
			if err != nil {
				return nil, err
			}
			steps = append(steps, blockerSteps...)

		case StmtTry:
			trySteps, err := e.emitTry(stmt.Try)
			if err != nil {
				return nil, err
			}
			steps = append(steps, trySteps...)
		}

		i++
	}

	return steps, nil
}

// emitCommandChain emits a chain of commands (possibly connected by operators) as a single Step.
// For a single command, returns a Step with CommandNode.
// For multiple commands, builds an operator tree (AndNode, OrNode, PipelineNode, SequenceNode).
func (e *Emitter) emitCommandChain(chain []*CommandStmtIR) (*planfmt.Step, error) {
	stepID := e.nextStepID
	e.pushStep(stepID)

	// Build execution tree from command chain
	tree, err := e.buildOperatorTree(chain)
	e.popStep()
	if err != nil {
		return nil, err
	}

	step := &planfmt.Step{
		ID:   stepID,
		Tree: tree,
	}
	e.nextStepID++

	return step, nil
}

// buildOperatorTree builds an ExecutionNode tree from a chain of commands.
// Handles operator precedence: | and redirect > && > || > ;
func (e *Emitter) buildOperatorTree(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	if len(chain) == 1 {
		return e.buildCommandNodeOrRedirect(chain[0])
	}

	// Parse by precedence (lowest to highest)
	// 1. Semicolon (lowest)
	if node, err := e.parseSequence(chain); node != nil || err != nil {
		return node, err
	}

	// 2. OR (||)
	if node, err := e.parseOr(chain); node != nil || err != nil {
		return node, err
	}

	// 3. AND (&&)
	if node, err := e.parseAnd(chain); node != nil || err != nil {
		return node, err
	}

	// 4. Pipe and Redirect (highest, left-to-right)
	if node, err := e.parsePipeAndRedirect(chain); node != nil || err != nil {
		return node, err
	}

	// Single command
	return e.buildCommandNodeOrRedirect(chain[0])
}

// buildCommandNode converts a CommandStmtIR to a CommandNode.
func (e *Emitter) buildCommandNode(cmd *CommandStmtIR) (*planfmt.CommandNode, error) {
	if cmd.Decorator != "" {
		e.pushDecorator(cmd.Decorator)
		defer e.popDecorator()
	}

	displayIDs := e.buildDisplayIDMap(cmd)

	args := make([]planfmt.Arg, 0, len(cmd.Args)+1)
	if cmd.Command != nil {
		commandStr := RenderCommand(cmd.Command, displayIDs)
		args = append(args, planfmt.Arg{
			Key: "command",
			Val: planfmt.Value{Kind: planfmt.ValueString, Str: commandStr},
		})
	}

	for _, arg := range cmd.Args {
		args = append(args, planfmt.Arg{
			Key: arg.Name,
			Val: planfmt.Value{Kind: planfmt.ValueString, Str: RenderExpr(arg.Value, displayIDs)},
		})
	}

	cmdNode := &planfmt.CommandNode{
		Decorator: cmd.Decorator,
		Args:      args,
	}

	if len(cmd.Block) > 0 {
		if e.scopes != nil {
			e.scopes.Push()
			defer e.scopes.Pop()
		}
		blockSteps, err := e.emitStatements(cmd.Block)
		if err != nil {
			return nil, err
		}
		cmdNode.Block = blockSteps
	}

	return cmdNode, nil
}

func (e *Emitter) buildCommandNodeOrRedirect(cmd *CommandStmtIR) (planfmt.ExecutionNode, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command is nil")
	}
	if cmd.RedirectMode == "" || cmd.RedirectTarget == nil {
		return e.buildCommandNode(cmd)
	}

	source, err := e.buildCommandNode(cmd)
	if err != nil {
		return nil, err
	}
	return e.buildRedirectNode(source, cmd)
}

func (e *Emitter) buildRedirectNode(source planfmt.ExecutionNode, cmd *CommandStmtIR) (planfmt.ExecutionNode, error) {
	if cmd.RedirectMode == "" || cmd.RedirectTarget == nil {
		return source, nil
	}

	target, err := e.buildRedirectTargetNode(cmd)
	if err != nil {
		return nil, err
	}

	mode := planfmt.RedirectOverwrite
	if cmd.RedirectMode == ">>" {
		mode = planfmt.RedirectAppend
	}

	return &planfmt.RedirectNode{
		Source: source,
		Target: *target,
		Mode:   mode,
	}, nil
}

func (e *Emitter) buildRedirectTargetNode(cmd *CommandStmtIR) (*planfmt.CommandNode, error) {
	if cmd.RedirectTarget == nil {
		return nil, fmt.Errorf("redirect target is nil")
	}

	displayIDs := make(map[string]string)
	for _, part := range cmd.RedirectTarget.Parts {
		e.collectDisplayID(part, displayIDs)
	}

	commandStr := RenderCommand(cmd.RedirectTarget, displayIDs)
	return &planfmt.CommandNode{
		Decorator: "@shell",
		Args: []planfmt.Arg{
			{
				Key: "command",
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: commandStr},
			},
		},
	}, nil
}

// parseSequence splits on semicolon operators (lowest precedence).
func (e *Emitter) parseSequence(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	var segments [][]*CommandStmtIR
	start := 0

	for i, cmd := range chain {
		if cmd.Operator == ";" {
			segments = append(segments, chain[start:i+1])
			start = i + 1
		}
	}

	if len(segments) == 0 {
		return nil, nil
	}

	// Add remaining commands
	if start < len(chain) {
		segments = append(segments, chain[start:])
	}

	var nodes []planfmt.ExecutionNode
	for _, seg := range segments {
		// Clear operator on last command to prevent infinite recursion
		if len(seg) > 0 && seg[len(seg)-1].Operator == ";" {
			seg = cloneCommandChainWithClearedOperator(seg, len(seg)-1)
		}
		node, err := e.buildOperatorTree(seg)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return &planfmt.SequenceNode{Nodes: nodes}, nil
}

func cloneCommandChainWithClearedOperator(chain []*CommandStmtIR, index int) []*CommandStmtIR {
	if len(chain) == 0 {
		return chain
	}

	cloned := make([]*CommandStmtIR, len(chain))
	copy(cloned, chain)

	if index < 0 || index >= len(cloned) || cloned[index] == nil {
		return cloned
	}

	cmdCopy := *cloned[index]
	cmdCopy.Operator = ""
	cloned[index] = &cmdCopy

	return cloned
}

// parseOr splits on || operators.
func (e *Emitter) parseOr(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	// Find rightmost || (left-to-right associativity)
	for i := len(chain) - 1; i >= 0; i-- {
		if chain[i].Operator == "||" {
			leftChain := cloneCommandChainWithClearedOperator(chain[:i+1], i)

			left, err := e.buildOperatorTree(leftChain)
			if err != nil {
				return nil, err
			}
			right, err := e.buildOperatorTree(chain[i+1:])
			if err != nil {
				return nil, err
			}
			return &planfmt.OrNode{Left: left, Right: right}, nil
		}
	}
	return nil, nil
}

// parseAnd splits on && operators.
func (e *Emitter) parseAnd(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	// Find rightmost && (left-to-right associativity)
	for i := len(chain) - 1; i >= 0; i-- {
		if chain[i].Operator == "&&" {
			leftChain := cloneCommandChainWithClearedOperator(chain[:i+1], i)

			left, err := e.buildOperatorTree(leftChain)
			if err != nil {
				return nil, err
			}
			right, err := e.buildOperatorTree(chain[i+1:])
			if err != nil {
				return nil, err
			}
			return &planfmt.AndNode{Left: left, Right: right}, nil
		}
	}
	return nil, nil
}

// parsePipeAndRedirect handles pipe (|) and redirect (>, >>) with equal precedence.
// Scans left-to-right to match bash behavior.
func (e *Emitter) parsePipeAndRedirect(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	for i := 0; i < len(chain); i++ {
		cmd := chain[i]
		if cmd == nil {
			continue
		}

		if cmd.RedirectMode != "" && cmd.RedirectTarget != nil {
			leftCmds := cloneCommandChain(chain[:i+1])
			savedOperator := ""
			if leftCmds[i] != nil {
				leftCmds[i].RedirectMode = ""
				leftCmds[i].RedirectTarget = nil
				savedOperator = leftCmds[i].Operator
				leftCmds[i].Operator = ""
			}

			source, err := e.buildOperatorTree(leftCmds)
			if err != nil {
				return nil, err
			}
			redirectNode, err := e.buildRedirectNode(source, cmd)
			if err != nil {
				return nil, err
			}

			if savedOperator == "|" && i+1 < len(chain) {
				right, err := e.buildOperatorTree(chain[i+1:])
				if err != nil {
					return nil, err
				}
				switch rightNode := right.(type) {
				case *planfmt.CommandNode:
					return &planfmt.PipelineNode{Commands: []planfmt.ExecutionNode{redirectNode, rightNode}}, nil
				case *planfmt.RedirectNode:
					return &planfmt.PipelineNode{Commands: []planfmt.ExecutionNode{redirectNode, rightNode}}, nil
				case *planfmt.PipelineNode:
					nodes := make([]planfmt.ExecutionNode, 1+len(rightNode.Commands))
					nodes[0] = redirectNode
					copy(nodes[1:], rightNode.Commands)
					return &planfmt.PipelineNode{Commands: nodes}, nil
				default:
					return redirectNode, nil
				}
			}

			return redirectNode, nil
		}

		if cmd.Operator == "|" {
			leftCmds := cloneCommandChain(chain[:i+1])
			if leftCmds[i] != nil {
				leftCmds[i].Operator = ""
			}
			left, err := e.buildOperatorTree(leftCmds)
			if err != nil {
				return nil, err
			}

			if i+1 < len(chain) {
				right, err := e.buildOperatorTree(chain[i+1:])
				if err != nil {
					return nil, err
				}

				leftCmd, leftIsCmd := left.(*planfmt.CommandNode)
				rightCmd, rightIsCmd := right.(*planfmt.CommandNode)
				rightPipe, rightIsPipe := right.(*planfmt.PipelineNode)
				leftRedirect, leftIsRedirect := left.(*planfmt.RedirectNode)
				rightRedirect, rightIsRedirect := right.(*planfmt.RedirectNode)

				if leftIsCmd && rightIsCmd {
					return &planfmt.PipelineNode{Commands: []planfmt.ExecutionNode{leftCmd, rightCmd}}, nil
				}
				if leftIsCmd && rightIsPipe {
					nodes := make([]planfmt.ExecutionNode, 1+len(rightPipe.Commands))
					nodes[0] = leftCmd
					copy(nodes[1:], rightPipe.Commands)
					return &planfmt.PipelineNode{Commands: nodes}, nil
				}
				if leftIsCmd && rightIsRedirect {
					return &planfmt.PipelineNode{Commands: []planfmt.ExecutionNode{leftCmd, rightRedirect}}, nil
				}
				if leftIsRedirect && rightIsCmd {
					return &planfmt.PipelineNode{Commands: []planfmt.ExecutionNode{leftRedirect, rightCmd}}, nil
				}
				if leftIsRedirect && rightIsRedirect {
					return &planfmt.PipelineNode{Commands: []planfmt.ExecutionNode{leftRedirect, rightRedirect}}, nil
				}
				if leftIsRedirect && rightIsPipe {
					nodes := make([]planfmt.ExecutionNode, 1+len(rightPipe.Commands))
					nodes[0] = leftRedirect
					copy(nodes[1:], rightPipe.Commands)
					return &planfmt.PipelineNode{Commands: nodes}, nil
				}

				return left, nil
			}

			return left, nil
		}
	}

	return nil, nil
}

func cloneCommandChain(chain []*CommandStmtIR) []*CommandStmtIR {
	cloned := make([]*CommandStmtIR, len(chain))
	for i, cmd := range chain {
		if cmd == nil {
			continue
		}
		cmdCopy := *cmd
		cloned[i] = &cmdCopy
	}
	return cloned
}

// buildDisplayIDMap builds a map of variable/decorator names to DisplayIDs.
// Used by RenderCommand to substitute placeholders.
func (e *Emitter) buildDisplayIDMap(cmd *CommandStmtIR) map[string]string {
	displayIDs := make(map[string]string)

	if cmd.Command != nil {
		for _, part := range cmd.Command.Parts {
			e.collectDisplayID(part, displayIDs)
		}
	}

	for _, arg := range cmd.Args {
		e.collectDisplayID(arg.Value, displayIDs)
	}

	return displayIDs
}

func (e *Emitter) collectDisplayID(expr *ExprIR, displayIDs map[string]string) {
	if expr == nil {
		return
	}

	switch expr.Kind {
	case ExprVarRef:
		// Look up exprID from scopes
		if e.scopes != nil {
			if exprID, ok := e.scopes.Lookup(expr.VarName); ok {
				if displayID := e.vault.GetDisplayID(exprID); displayID != "" {
					displayIDs[expr.VarName] = displayID
					// Record secret use at current site
					e.recordSecretUse(exprID, displayID, expr.VarName)
				}
			}
		}

	case ExprDecoratorRef:
		// Build decorator key and look up DisplayID
		key := decoratorKey(expr.Decorator)
		if exprID, ok := e.decoratorExprIDs[key]; ok {
			if displayID := e.vault.GetDisplayID(exprID); displayID != "" {
				displayIDs[key] = displayID
				e.recordSecretUse(exprID, displayID, key)
			}
		}
	}
}

func (e *Emitter) pushStep(stepID uint64) {
	e.sitePath = append(e.sitePath, fmt.Sprintf("step-%d", stepID))
	e.decoratorCounts = append(e.decoratorCounts, make(map[string]int))
}

func (e *Emitter) popStep() {
	if len(e.sitePath) > 1 {
		e.sitePath = e.sitePath[:len(e.sitePath)-1]
	}
	if len(e.decoratorCounts) > 0 {
		e.decoratorCounts = e.decoratorCounts[:len(e.decoratorCounts)-1]
	}
}

func (e *Emitter) pushDecorator(name string) {
	counts := e.currentDecoratorCounts()
	if counts == nil {
		counts = make(map[string]int)
		e.decoratorCounts = append(e.decoratorCounts, counts)
	}

	index := counts[name]
	counts[name] = index + 1
	e.sitePath = append(e.sitePath, fmt.Sprintf("%s[%d]", name, index))
}

func (e *Emitter) popDecorator() {
	if len(e.sitePath) > 1 {
		e.sitePath = e.sitePath[:len(e.sitePath)-1]
	}
}

func (e *Emitter) currentDecoratorCounts() map[string]int {
	if len(e.decoratorCounts) == 0 {
		return nil
	}
	return e.decoratorCounts[len(e.decoratorCounts)-1]
}

// recordSecretUse records a secret usage at the current site path.
func (e *Emitter) recordSecretUse(exprID, displayID, paramName string) {
	site := e.buildSitePath(paramName)
	siteID := e.computeSiteID(site)

	e.secretUses = append(e.secretUses, planfmt.SecretUse{
		DisplayID: displayID,
		SiteID:    siteID,
		Site:      site,
	})
}

// buildSitePath builds the current site path with the given parameter name.
func (e *Emitter) buildSitePath(paramName string) string {
	// Join site path segments with "/"
	path := ""
	for i, seg := range e.sitePath {
		if i > 0 {
			path += "/"
		}
		path += seg
	}
	path += "/params/" + paramName
	return path
}

// computeSiteID generates an HMAC-based site ID.
func (e *Emitter) computeSiteID(site string) string {
	planKey := e.vault.GetPlanKey()
	if len(planKey) == 0 {
		return ""
	}

	h := hmac.New(sha256.New, planKey)
	h.Write([]byte(site))
	mac := h.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(mac[:16])
}

// emitBlocker emits a blocker (if/for/when) as a single step with LogicNode.
// Preserves structure for rich plan output (HTML, JSON, terminal, etc.).
func (e *Emitter) emitBlocker(blocker *BlockerIR) ([]planfmt.Step, error) {
	switch blocker.Kind {
	case BlockerIf:
		return e.emitIfBlocker(blocker)
	case BlockerFor:
		return e.emitForBlocker(blocker)
	case BlockerWhen:
		return e.emitWhenBlocker(blocker)
	default:
		return nil, nil
	}
}

// emitIfBlocker emits an if statement as a LogicNode step.
// Preserves condition and result for plan display, with taken branch as nested steps.
func (e *Emitter) emitIfBlocker(blocker *BlockerIR) ([]planfmt.Step, error) {
	// Determine which branch was taken and get its steps
	var takenBranch []*StatementIR
	var result string

	if blocker.Taken != nil && *blocker.Taken {
		takenBranch = blocker.ThenBranch
		result = "true"
	} else if blocker.ElseBranch != nil {
		takenBranch = blocker.ElseBranch
		result = "false"
	} else {
		// No else branch and condition was false - nothing to emit
		return nil, nil
	}

	// Emit the taken branch as nested steps
	nestedSteps, err := e.emitStatements(takenBranch)
	if err != nil {
		return nil, err
	}

	// Build condition string from expression
	var conditionStr string
	if blocker.Condition != nil {
		conditionStr = RenderExpr(blocker.Condition, nil)
	}

	// Create LogicNode to preserve structure
	logicNode := &planfmt.LogicNode{
		Kind:      "if",
		Condition: conditionStr,
		Result:    result,
		Block:     nestedSteps,
	}

	step := planfmt.Step{
		ID:   e.nextStepID,
		Tree: logicNode,
	}
	e.nextStepID++

	return []planfmt.Step{step}, nil
}

// emitForBlocker emits a for-loop as LogicNode steps (one per iteration).
// Each iteration is a separate LogicNode showing the loop variable value.
func (e *Emitter) emitForBlocker(blocker *BlockerIR) ([]planfmt.Step, error) {
	var steps []planfmt.Step

	for i, iter := range blocker.Iterations {
		// Emit this iteration's body as nested steps
		nestedSteps, err := e.emitStatements(iter.Body)
		if err != nil {
			return nil, err
		}

		// Build condition showing loop variable assignment
		var collectionStr string
		if blocker.Collection != nil {
			collectionStr = RenderExpr(blocker.Collection, nil)
		}
		conditionStr := fmt.Sprintf("%s in %s", blocker.LoopVar, collectionStr)
		resultStr := fmt.Sprintf("%s = %v (iteration %d)", blocker.LoopVar, iter.Value, i+1)

		logicNode := &planfmt.LogicNode{
			Kind:      "for",
			Condition: conditionStr,
			Result:    resultStr,
			Block:     nestedSteps,
		}

		step := planfmt.Step{
			ID:   e.nextStepID,
			Tree: logicNode,
		}
		e.nextStepID++
		steps = append(steps, step)
	}

	return steps, nil
}

// emitWhenBlocker emits a when statement as a LogicNode step.
func (e *Emitter) emitWhenBlocker(blocker *BlockerIR) ([]planfmt.Step, error) {
	if blocker.MatchedArm < 0 || blocker.MatchedArm >= len(blocker.Arms) {
		return nil, nil
	}

	matchedArm := blocker.Arms[blocker.MatchedArm]

	// Emit the matched arm's body as nested steps
	nestedSteps, err := e.emitStatements(matchedArm.Body)
	if err != nil {
		return nil, err
	}

	// Build condition and result strings
	var conditionStr string
	if blocker.Condition != nil {
		conditionStr = RenderExpr(blocker.Condition, nil)
	}
	var patternStr string
	if matchedArm.Pattern != nil {
		patternStr = RenderExpr(matchedArm.Pattern, nil)
	}
	resultStr := fmt.Sprintf("matched: %s", patternStr)

	logicNode := &planfmt.LogicNode{
		Kind:      "when",
		Condition: conditionStr,
		Result:    resultStr,
		Block:     nestedSteps,
	}

	step := planfmt.Step{
		ID:   e.nextStepID,
		Tree: logicNode,
	}
	e.nextStepID++

	return []planfmt.Step{step}, nil
}

// emitTry emits a try/catch/finally statement as a single step with TryNode.
// All branches are included in the plan (runtime determines which executes).
func (e *Emitter) emitTry(try *TryIR) ([]planfmt.Step, error) {
	stepID := e.nextStepID
	e.nextStepID++
	e.pushStep(stepID)
	defer e.popStep()

	// Emit try block with isolated scope
	var trySteps []planfmt.Step
	if len(try.TryBlock) > 0 {
		if e.scopes != nil {
			e.scopes.Push()
		}
		steps, err := e.emitStatements(try.TryBlock)
		if e.scopes != nil {
			e.scopes.Pop()
		}
		if err != nil {
			return nil, err
		}
		trySteps = steps
	}

	// Emit catch block with isolated scope (if present)
	var catchSteps []planfmt.Step
	if len(try.CatchBlock) > 0 {
		if e.scopes != nil {
			e.scopes.Push()
		}
		steps, err := e.emitStatements(try.CatchBlock)
		if e.scopes != nil {
			e.scopes.Pop()
		}
		if err != nil {
			return nil, err
		}
		catchSteps = steps
	}

	// Emit finally block with isolated scope (if present)
	var finallySteps []planfmt.Step
	if len(try.FinallyBlock) > 0 {
		if e.scopes != nil {
			e.scopes.Push()
		}
		steps, err := e.emitStatements(try.FinallyBlock)
		if e.scopes != nil {
			e.scopes.Pop()
		}
		if err != nil {
			return nil, err
		}
		finallySteps = steps
	}

	// Create TryNode with all blocks
	tryNode := &planfmt.TryNode{
		TryBlock:     trySteps,
		CatchBlock:   catchSteps,
		FinallyBlock: finallySteps,
	}

	// Return as a single step wrapping the TryNode
	return []planfmt.Step{{
		ID:   stepID,
		Tree: tryNode,
	}}, nil
}
