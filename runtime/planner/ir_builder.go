package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/opal-lang/opal/runtime/lexer"
	"github.com/opal-lang/opal/runtime/parser"
)

// BuildIR constructs an ExecutionGraph from parser events and tokens.
// This is a pure structural pass - no resolution or condition evaluation.
func BuildIR(events []parser.Event, tokens []lexer.Token) (*ExecutionGraph, error) {
	b := &irBuilder{
		events:  events,
		tokens:  tokens,
		pos:     0,
		scopes:  NewScopeStack(),
		exprSeq: 0,
	}

	stmts, err := b.buildSource()
	if err != nil {
		return nil, err
	}

	return &ExecutionGraph{
		Statements: stmts,
		Scopes:     b.scopes,
	}, nil
}

// irBuilder walks parser events and builds the IR.
type irBuilder struct {
	events  []parser.Event
	tokens  []lexer.Token
	pos     int
	scopes  *ScopeStack
	exprSeq int
}

// buildSource processes the top-level source node.
func (b *irBuilder) buildSource() ([]*StatementIR, error) {
	var stmts []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		switch evt.Kind {
		case parser.EventOpen:
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeSource:
				b.pos++
				continue

			case parser.NodeVarDecl:
				stmt, err := b.buildVarDecl()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeIf:
				stmt, err := b.buildIfStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue
			}

		case parser.EventClose:
			b.pos++
			continue

		case parser.EventStepEnter:
			stmt, err := b.buildStep()
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				stmts = append(stmts, stmt...)
			}
			continue

		case parser.EventToken:
			b.pos++
			continue
		}

		b.pos++
	}

	return stmts, nil
}

// buildStep processes a step (EventStepEnter to EventStepExit).
func (b *irBuilder) buildStep() ([]*StatementIR, error) {
	if b.pos >= len(b.events) || b.events[b.pos].Kind != parser.EventStepEnter {
		return nil, fmt.Errorf("buildStep: expected EventStepEnter at pos %d", b.pos)
	}

	b.pos++ // Move past EventStepEnter

	var stmts []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventStepExit {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeVarDecl:
				stmt, err := b.buildVarDecl()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeShellCommand:
				stmt, err := b.buildShellCommand()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeIf:
				stmt, err := b.buildIfStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue
			}
		}

		b.pos++
	}

	return stmts, nil
}

// buildVarDecl processes a variable declaration.
func (b *irBuilder) buildVarDecl() (*StatementIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeVarDecl

	var name string
	var value *ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeVarDecl {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && name == "" {
				name = string(tok.Text)
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeLiteral:
				value = b.buildLiteralExpr()
				continue
			case parser.NodeDecorator:
				value = b.buildDecoratorExpr()
				continue
			case parser.NodeIdentifier:
				value = b.buildIdentifierExpr()
				continue
			}
		}

		b.pos++
	}

	if name == "" {
		return nil, fmt.Errorf("variable declaration at position %d has no name", startPos)
	}

	exprID := b.generateExprID(name)
	b.scopes.Define(name, exprID)

	return &StatementIR{
		Kind: StmtVarDecl,
		VarDecl: &VarDeclIR{
			Name:   name,
			Value:  value,
			ExprID: exprID,
		},
	}, nil
}

// buildShellCommand processes a shell command.
func (b *irBuilder) buildShellCommand() (*StatementIR, error) {
	b.pos++ // Move past OPEN NodeShellCommand

	var parts []*ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeShellCommand {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeShellArg:
				argParts := b.buildShellArg()
				parts = append(parts, argParts...)
				continue
			case parser.NodeDecorator:
				expr := b.buildDecoratorExpr()
				parts = append(parts, expr)
				continue
			case parser.NodeInterpolatedString:
				strParts := b.buildInterpolatedString()
				parts = append(parts, strParts...)
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if len(tok.Text) > 0 {
				parts = append(parts, &ExprIR{
					Kind:  ExprLiteral,
					Value: string(tok.Text),
				})
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return &StatementIR{
		Kind: StmtCommand,
		Command: &CommandStmtIR{
			Decorator: "@shell",
			Command: &CommandExpr{
				Parts: parts,
			},
		},
	}, nil
}

// buildShellArg processes a shell argument node.
func (b *irBuilder) buildShellArg() []*ExprIR {
	b.pos++ // Move past OPEN NodeShellArg

	var parts []*ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeShellArg {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeDecorator:
				expr := b.buildDecoratorExpr()
				parts = append(parts, expr)
				continue
			case parser.NodeInterpolatedString:
				strParts := b.buildInterpolatedString()
				parts = append(parts, strParts...)
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if len(tok.Text) > 0 {
				parts = append(parts, &ExprIR{
					Kind:  ExprLiteral,
					Value: string(tok.Text),
				})
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return parts
}

// buildInterpolatedString processes an interpolated string.
func (b *irBuilder) buildInterpolatedString() []*ExprIR {
	b.pos++ // Move past OPEN NodeInterpolatedString

	var parts []*ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeInterpolatedString {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeStringPart:
				part := b.buildStringPart()
				if part != nil {
					parts = append(parts, part)
				}
				continue
			case parser.NodeDecorator:
				expr := b.buildDecoratorExpr()
				parts = append(parts, expr)
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if len(tok.Text) > 0 && tok.Type == lexer.STRING {
				parts = append(parts, &ExprIR{
					Kind:  ExprLiteral,
					Value: string(tok.Text),
				})
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return parts
}

// buildStringPart processes a string part within an interpolated string.
func (b *irBuilder) buildStringPart() *ExprIR {
	b.pos++ // Move past OPEN NodeStringPart

	var result *ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeStringPart {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			if node == parser.NodeDecorator {
				result = b.buildDecoratorExpr()
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if len(tok.Text) > 0 {
				result = &ExprIR{
					Kind:  ExprLiteral,
					Value: string(tok.Text),
				}
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return result
}

// buildLiteralExpr processes a literal expression.
func (b *irBuilder) buildLiteralExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeLiteral

	var value any

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeLiteral {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			value = tokenToValue(tok)
			b.pos++
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprLiteral,
		Value: value,
	}
}

// buildDecoratorExpr processes a decorator expression (@var.X, @env.HOME).
func (b *irBuilder) buildDecoratorExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeDecorator

	var parts []string

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeDecorator {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			// Collect identifier-like tokens (skip @ and .)
			// Note: "var" is lexed as VAR keyword, not IDENTIFIER
			if len(tok.Text) > 0 && tok.Type != lexer.AT && tok.Type != lexer.DOT {
				parts = append(parts, string(tok.Text))
			}
			b.pos++
			continue
		}

		b.pos++
	}

	if len(parts) == 0 {
		return &ExprIR{
			Kind: ExprDecoratorRef,
			Decorator: &DecoratorRef{
				Name: "",
			},
		}
	}

	name := parts[0]
	var selector []string
	if len(parts) > 1 {
		selector = parts[1:]
	}

	// @var.X becomes a VarRef
	if name == "var" && len(selector) > 0 {
		return &ExprIR{
			Kind:    ExprVarRef,
			VarName: selector[0],
		}
	}

	return &ExprIR{
		Kind: ExprDecoratorRef,
		Decorator: &DecoratorRef{
			Name:     name,
			Selector: selector,
		},
	}
}

// buildIdentifierExpr processes an identifier expression.
func (b *irBuilder) buildIdentifierExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeIdentifier

	var name string

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeIdentifier {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER {
				name = string(tok.Text)
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:    ExprVarRef,
		VarName: name,
	}
}

// buildIfStmt processes an if statement.
func (b *irBuilder) buildIfStmt() (*StatementIR, error) {
	b.pos++ // Move past OPEN NodeIf

	var condition *ExprIR
	var thenBranch []*StatementIR
	var elseBranch []*StatementIR

	// Skip IF token
	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind == parser.EventToken {
			b.pos++
			break
		}
		b.pos++
	}

	// Parse condition expression
	// Pattern: primary expression followed by optional binary expression
	var primaryExpr *ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeIf {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeDecorator:
				primaryExpr = b.buildDecoratorExpr()
				// Check if followed by binary expression
				if b.pos < len(b.events) {
					nextEvt := b.events[b.pos]
					if nextEvt.Kind == parser.EventOpen && parser.NodeKind(nextEvt.Data) == parser.NodeBinaryExpr {
						condition = b.buildBinaryExprWithLeft(primaryExpr)
					} else {
						condition = primaryExpr
					}
				} else {
					condition = primaryExpr
				}
				continue
			case parser.NodeBinaryExpr:
				// Binary expression without explicit left side (shouldn't happen)
				condition = b.buildBinaryExpr()
				continue
			case parser.NodeLiteral:
				primaryExpr = b.buildLiteralExpr()
				// Check if followed by binary expression
				if b.pos < len(b.events) {
					nextEvt := b.events[b.pos]
					if nextEvt.Kind == parser.EventOpen && parser.NodeKind(nextEvt.Data) == parser.NodeBinaryExpr {
						condition = b.buildBinaryExprWithLeft(primaryExpr)
					} else {
						condition = primaryExpr
					}
				} else {
					condition = primaryExpr
				}
				continue
			case parser.NodeBlock:
				if thenBranch == nil {
					stmts, err := b.buildBlock()
					if err != nil {
						return nil, err
					}
					thenBranch = stmts
				}
				continue
			case parser.NodeElse:
				stmts, err := b.buildElseClause()
				if err != nil {
					return nil, err
				}
				elseBranch = stmts
				continue
			}
		}

		b.pos++
	}

	return &StatementIR{
		Kind: StmtBlocker,
		Blocker: &BlockerIR{
			Kind:       BlockerIf,
			Condition:  condition,
			ThenBranch: thenBranch,
			ElseBranch: elseBranch,
		},
	}, nil
}

// buildBlock processes a block of statements.
func (b *irBuilder) buildBlock() ([]*StatementIR, error) {
	b.pos++ // Move past OPEN NodeBlock

	var stmts []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeBlock {
			b.pos++
			break
		}

		if evt.Kind == parser.EventStepEnter {
			stepStmts, err := b.buildStep()
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, stepStmts...)
			continue
		}

		b.pos++
	}

	return stmts, nil
}

// buildElseClause processes an else clause.
func (b *irBuilder) buildElseClause() ([]*StatementIR, error) {
	b.pos++ // Move past OPEN NodeElse

	var stmts []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeElse {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeBlock:
				blockStmts, err := b.buildBlock()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, blockStmts...)
				continue
			case parser.NodeIf:
				ifStmt, err := b.buildIfStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, ifStmt)
				continue
			}
		}

		b.pos++
	}

	return stmts, nil
}

// buildBinaryExpr processes a binary expression.
func (b *irBuilder) buildBinaryExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeBinaryExpr

	var left, right *ExprIR
	var op string

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeBinaryExpr {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			var expr *ExprIR
			switch node {
			case parser.NodeDecorator:
				expr = b.buildDecoratorExpr()
			case parser.NodeLiteral:
				expr = b.buildLiteralExpr()
			case parser.NodeIdentifier:
				expr = b.buildIdentifierExpr()
			case parser.NodeBinaryExpr:
				expr = b.buildBinaryExpr()
			default:
				b.pos++
				continue
			}

			if left == nil {
				left = expr
			} else {
				right = expr
			}
			continue
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			switch tok.Type {
			case lexer.EQ_EQ:
				op = "=="
			case lexer.NOT_EQ:
				op = "!="
			case lexer.LT:
				op = "<"
			case lexer.LT_EQ:
				op = "<="
			case lexer.GT:
				op = ">"
			case lexer.GT_EQ:
				op = ">="
			case lexer.AND_AND:
				op = "&&"
			case lexer.OR_OR:
				op = "||"
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    op,
		Left:  left,
		Right: right,
	}
}

// buildBinaryExprWithLeft processes a binary expression with a known left side.
// This is used when the left expression was already parsed (e.g., in if conditions).
func (b *irBuilder) buildBinaryExprWithLeft(left *ExprIR) *ExprIR {
	b.pos++ // Move past OPEN NodeBinaryExpr

	var right *ExprIR
	var op string

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeBinaryExpr {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			switch tok.Type {
			case lexer.EQ_EQ:
				op = "=="
			case lexer.NOT_EQ:
				op = "!="
			case lexer.LT:
				op = "<"
			case lexer.LT_EQ:
				op = "<="
			case lexer.GT:
				op = ">"
			case lexer.GT_EQ:
				op = ">="
			case lexer.AND_AND:
				op = "&&"
			case lexer.OR_OR:
				op = "||"
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeDecorator:
				right = b.buildDecoratorExpr()
			case parser.NodeLiteral:
				right = b.buildLiteralExpr()
			case parser.NodeIdentifier:
				right = b.buildIdentifierExpr()
			case parser.NodeBinaryExpr:
				right = b.buildBinaryExpr()
			default:
				b.pos++
			}
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    op,
		Left:  left,
		Right: right,
	}
}

// generateExprID generates a unique expression ID.
func (b *irBuilder) generateExprID(name string) string {
	b.exprSeq++
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", name, b.exprSeq)))
	return "expr-" + hex.EncodeToString(hash[:8])
}

// tokenToValue converts a token to its Go value.
func tokenToValue(tok lexer.Token) any {
	switch tok.Type {
	case lexer.STRING:
		return string(tok.Text)
	case lexer.INTEGER:
		val, _ := strconv.ParseInt(string(tok.Text), 10, 64)
		return val
	case lexer.BOOLEAN:
		return string(tok.Text) == "true"
	default:
		return string(tok.Text)
	}
}
