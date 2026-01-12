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
	nextStepID uint64
	sitePath   []string
	secretUses []planfmt.SecretUse
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
	}
}

// Emit transforms the resolved IR into a Plan.
func (e *Emitter) Emit() (*planfmt.Plan, error) {
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
			chain := []*CommandStmtIR{stmt.Command}
			for i+1 < len(stmts) && stmt.Command.Operator != "" && stmts[i+1].Kind == StmtCommand {
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
	if len(chain) == 1 {
		return e.emitCommand(chain[0])
	}

	// Build execution tree from command chain
	tree, err := e.buildOperatorTree(chain)
	if err != nil {
		return nil, err
	}

	step := &planfmt.Step{
		ID:   e.nextStepID,
		Tree: tree,
	}
	e.nextStepID++

	return step, nil
}

// buildOperatorTree builds an ExecutionNode tree from a chain of commands.
// Handles operator precedence: | > && > || > ;
func (e *Emitter) buildOperatorTree(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	if len(chain) == 1 {
		return e.buildCommandNode(chain[0])
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

	// 4. Pipe (|) - highest
	if node, err := e.parsePipe(chain); node != nil || err != nil {
		return node, err
	}

	// Single command
	return e.buildCommandNode(chain[0])
}

// buildCommandNode converts a CommandStmtIR to a CommandNode.
func (e *Emitter) buildCommandNode(cmd *CommandStmtIR) (*planfmt.CommandNode, error) {
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
			seg[len(seg)-1].Operator = ""
		}
		node, err := e.buildOperatorTree(seg)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return &planfmt.SequenceNode{Nodes: nodes}, nil
}

// parseOr splits on || operators.
func (e *Emitter) parseOr(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	// Find rightmost || (left-to-right associativity)
	for i := len(chain) - 1; i >= 0; i-- {
		if chain[i].Operator == "||" {
			leftChain := make([]*CommandStmtIR, i+1)
			copy(leftChain, chain[:i+1])
			leftChain[i].Operator = "" // Clear operator

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
			leftChain := make([]*CommandStmtIR, i+1)
			copy(leftChain, chain[:i+1])
			leftChain[i].Operator = "" // Clear operator

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

// parsePipe splits on | operators (highest precedence).
func (e *Emitter) parsePipe(chain []*CommandStmtIR) (planfmt.ExecutionNode, error) {
	var pipeCommands []planfmt.ExecutionNode

	for i, cmd := range chain {
		cmdNode, err := e.buildCommandNode(cmd)
		if err != nil {
			return nil, err
		}
		pipeCommands = append(pipeCommands, cmdNode)
		if cmd.Operator != "|" && i < len(chain)-1 {
			// Non-pipe operator in the middle - shouldn't happen at this point
			break
		}
	}

	if len(pipeCommands) <= 1 {
		return nil, nil
	}

	return &planfmt.PipelineNode{Commands: pipeCommands}, nil
}

// emitCommand emits a single command statement as a Step.
func (e *Emitter) emitCommand(cmd *CommandStmtIR) (*planfmt.Step, error) {
	cmdNode, err := e.buildCommandNode(cmd)
	if err != nil {
		return nil, err
	}

	step := &planfmt.Step{
		ID:   e.nextStepID,
		Tree: cmdNode,
	}
	e.nextStepID++

	return step, nil
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

// emitTry emits a try/catch/finally statement.
// All branches are included in the plan (runtime determines which executes).
func (e *Emitter) emitTry(try *TryIR) ([]planfmt.Step, error) {
	// TODO: Implement try/catch emission
	// For now, just emit the try block with isolated scope
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
	return steps, nil
}
