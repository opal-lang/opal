package planner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/opal-lang/opal/runtime/lexer"
	"github.com/opal-lang/opal/runtime/parser"
)

// BuildIR constructs an ExecutionGraph from parser events and tokens.
// This is a pure structural pass - no resolution or condition evaluation.
func BuildIR(events []parser.Event, tokens []lexer.Token) (*ExecutionGraph, error) {
	b := &irBuilder{
		events:    events,
		tokens:    tokens,
		pos:       0,
		scopes:    NewScopeStack(),
		functions: make(map[string]*FunctionIR),
		exprSeq:   0,
	}

	stmts, err := b.buildSource()
	if err != nil {
		return nil, err
	}

	return &ExecutionGraph{
		Statements: stmts,
		Functions:  b.functions,
		Scopes:     b.scopes,
	}, nil
}

// irBuilder walks parser events and builds the IR.
type irBuilder struct {
	events    []parser.Event
	tokens    []lexer.Token
	pos       int
	scopes    *ScopeStack
	functions map[string]*FunctionIR
	exprSeq   int
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

			case parser.NodeFunction:
				fn, err := b.buildFunction()
				if err != nil {
					return nil, err
				}
				if fn != nil && fn.Name != "" {
					b.functions[fn.Name] = fn
				}
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

			case parser.NodeFor:
				stmt, err := b.buildForStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeWhen:
				stmt, err := b.buildWhenStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeTry:
				stmt, err := b.buildTryStmt()
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

// buildFunction processes a function definition.
func (b *irBuilder) buildFunction() (*FunctionIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeFunction

	var name string
	var params []ParamIR
	var body []*StatementIR

	// Snapshot outer scopes and use a cloned scope stack for the function body
	outerScopes := b.scopes
	functionScopes := b.scopes.Clone()
	b.scopes = functionScopes
	defer func() { b.scopes = outerScopes }()

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeFunction {
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
			case parser.NodeParamList:
				parsedParams, err := b.buildParamList()
				if err != nil {
					return nil, err
				}
				params = parsedParams
				continue
			case parser.NodeBlock:
				blockStmts, err := b.buildBlock()
				if err != nil {
					return nil, err
				}
				body = append(body, blockStmts...)
				continue
			}
		}

		if evt.Kind == parser.EventStepEnter {
			stepStmts, err := b.buildStep()
			if err != nil {
				return nil, err
			}
			body = append(body, stepStmts...)
			continue
		}

		b.pos++
	}

	if name == "" {
		return nil, fmt.Errorf("function declaration at position %d has no name", startPos)
	}

	return &FunctionIR{
		Name:   name,
		Params: params,
		Body:   body,
		Span:   SourceSpan{Start: startPos, End: b.pos},
		Scopes: functionScopes,
	}, nil
}

// buildParamList processes a function parameter list.
func (b *irBuilder) buildParamList() ([]ParamIR, error) {
	b.pos++ // Move past OPEN NodeParamList

	var params []ParamIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParamList {
			b.pos++
			break
		}
		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeParam {
			param, err := b.buildParam()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
			continue
		}
		if evt.Kind == parser.EventToken {
			b.pos++
			continue
		}
		b.pos++
	}

	return params, nil
}

func (b *irBuilder) buildParam() (ParamIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeParam

	var param ParamIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParam {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && param.Name == "" {
				param.Name = string(tok.Text)
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeTypeAnnotation:
				param.Type = b.buildTypeAnnotation()
				continue
			case parser.NodeDefaultValue:
				param.Default = b.buildDefaultValue()
				continue
			}
		}

		b.pos++
	}

	if param.Name == "" {
		return ParamIR{}, fmt.Errorf("parameter at position %d has no name", startPos)
	}

	return param, nil
}

func (b *irBuilder) buildTypeAnnotation() string {
	b.pos++ // Move past OPEN NodeTypeAnnotation

	var typeName string

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeTypeAnnotation {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER {
				typeName = string(tok.Text)
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return typeName
}

func (b *irBuilder) buildDefaultValue() *ExprIR {
	b.pos++ // Move past OPEN NodeDefaultValue

	var expr *ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeDefaultValue {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeLiteral:
				expr = b.buildLiteralExpr()
				continue
			case parser.NodeDecorator:
				expr = b.buildDecoratorExpr()
				continue
			case parser.NodeIdentifier:
				expr = b.buildIdentifierExpr()
				continue
			case parser.NodeBinaryExpr:
				expr = b.buildBinaryExpr()
				continue
			}
		}

		if evt.Kind == parser.EventToken && expr == nil {
			tok := b.tokens[evt.Data]
			expr = &ExprIR{
				Kind:  ExprLiteral,
				Value: tokenToValue(tok),
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return expr
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

			case parser.NodeDecorator:
				stmt, err := b.buildDecoratorStmt()
				if err != nil {
					return nil, err
				}
				if stmt != nil {
					stmts = append(stmts, stmt)
				}
				continue

			case parser.NodeIf:
				stmt, err := b.buildIfStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeFor:
				stmt, err := b.buildForStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeWhen:
				stmt, err := b.buildWhenStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				continue

			case parser.NodeTry:
				stmt, err := b.buildTryStmt()
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

	// ExprID is NOT generated here - it will be generated during resolution
	// based on the actual scope context (loop iteration, transport, etc.).
	// This ensures each loop iteration gets a unique ExprID for variables
	// declared in the loop body.
	//
	// We still define the variable in scopes with a placeholder so that
	// subsequent VarRefs can find it. The actual ExprID will be set
	// during resolution when collectVarDecl processes this VarDecl.
	placeholderID := fmt.Sprintf("placeholder:%s:%d", name, b.exprSeq)
	b.exprSeq++
	b.scopes.Define(name, placeholderID)

	return &StatementIR{
		Kind: StmtVarDecl,
		Span: SourceSpan{Start: startPos, End: b.pos},
		VarDecl: &VarDeclIR{
			Name:  name,
			Value: value,
			// ExprID intentionally empty - generated during resolution
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
				// Check if this arg needs a space before it
				needsSpace := b.shellArgNeedsSpace()
				if needsSpace {
					parts = append(parts, &ExprIR{
						Kind:  ExprLiteral,
						Value: " ",
					})
				}
				argParts, err := b.buildShellArg()
				if err != nil {
					return nil, err
				}
				parts = append(parts, argParts...)
				continue
			case parser.NodeDecorator:
				expr := b.buildDecoratorExpr()
				parts = append(parts, expr)
				continue
			case parser.NodeInterpolatedString:
				strParts, err := b.buildInterpolatedString()
				if err != nil {
					return nil, err
				}
				parts = append(parts, strParts...)
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			symbol := tok.Symbol()
			if symbol != "" {
				parts = append(parts, &ExprIR{
					Kind:  ExprLiteral,
					Value: symbol,
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

// buildDecoratorStmt processes a decorator statement with optional args and block.
func (b *irBuilder) buildDecoratorStmt() (*StatementIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeDecorator

	var nameParts []string
	var args []ArgIR
	var block []*StatementIR
	parsingName := true

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeDecorator {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeParamList:
				parsedArgs, err := b.buildDecoratorArgs()
				if err != nil {
					return nil, err
				}
				args = parsedArgs
				parsingName = false
				continue
			case parser.NodeBlock:
				blockStmts, err := b.buildBlock()
				if err != nil {
					return nil, err
				}
				block = append(block, blockStmts...)
				parsingName = false
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			if parsingName {
				tok := b.tokens[evt.Data]
				switch tok.Type {
				case lexer.AT, lexer.DOT:
					// Skip
				case lexer.IDENTIFIER, lexer.VAR:
					nameParts = append(nameParts, string(tok.Text))
				}
			}
			b.pos++
			continue
		}

		b.pos++
	}

	if len(nameParts) == 0 {
		return nil, fmt.Errorf("decorator statement at position %d has no name", startPos)
	}

	decoratorName := "@" + strings.Join(nameParts, ".")

	return &StatementIR{
		Kind:         StmtCommand,
		CreatesScope: len(block) > 0,
		Command: &CommandStmtIR{
			Decorator: decoratorName,
			Args:      args,
			Block:     block,
		},
	}, nil
}

func (b *irBuilder) buildDecoratorArgs() ([]ArgIR, error) {
	b.pos++ // Move past OPEN NodeParamList

	var args []ArgIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParamList {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeParam {
			arg, err := b.buildDecoratorArg()
			if err != nil {
				return nil, err
			}
			if arg.Name == "" {
				arg.Name = fmt.Sprintf("arg%d", len(args)+1)
			}
			args = append(args, arg)
			continue
		}

		if evt.Kind == parser.EventToken {
			b.pos++
			continue
		}

		b.pos++
	}

	return args, nil
}

func (b *irBuilder) buildDecoratorArg() (ArgIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeParam

	var arg ArgIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParam {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			switch tok.Type {
			case lexer.IDENTIFIER:
				if arg.Name == "" && b.decoratorArgHasName() {
					arg.Name = string(tok.Text)
					b.pos++
					continue
				}
				if arg.Value == nil {
					arg.Value = &ExprIR{Kind: ExprLiteral, Value: tokenToValue(tok)}
				}
				b.pos++
				continue
			case lexer.EQUALS:
				b.pos++
				continue
			default:
				if arg.Value == nil {
					arg.Value = &ExprIR{Kind: ExprLiteral, Value: tokenToValue(tok)}
				}
				b.pos++
				continue
			}
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeLiteral:
				arg.Value = b.buildLiteralExpr()
				continue
			case parser.NodeDecorator:
				arg.Value = b.buildDecoratorExpr()
				continue
			case parser.NodeIdentifier:
				arg.Value = b.buildIdentifierExpr()
				continue
			case parser.NodeBinaryExpr:
				arg.Value = b.buildBinaryExpr()
				continue
			}
		}

		b.pos++
	}

	if arg.Value == nil {
		return ArgIR{}, fmt.Errorf("decorator parameter at position %d has no value", startPos)
	}

	return arg, nil
}

func (b *irBuilder) decoratorArgHasName() bool {
	for i := b.pos + 1; i < len(b.events); i++ {
		evt := b.events[i]
		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			return tok.Type == lexer.EQUALS
		}
		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParam {
			return false
		}
		if evt.Kind == parser.EventOpen {
			return false
		}
	}

	return false
}

// shellArgNeedsSpace checks if the upcoming shell arg needs a space before it.
// This looks ahead to find the first token in the NodeShellArg and checks HasSpaceBefore.
func (b *irBuilder) shellArgNeedsSpace() bool {
	// Look ahead from current position (which is at OPEN NodeShellArg)
	for i := b.pos + 1; i < len(b.events); i++ {
		evt := b.events[i]

		// If we hit the close of the shell arg, no token found
		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeShellArg {
			return false
		}

		// First token in the arg
		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			return tok.HasSpaceBefore
		}
	}
	return false
}

// buildShellArg processes a shell argument node.
func (b *irBuilder) buildShellArg() ([]*ExprIR, error) {
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
				strParts, err := b.buildInterpolatedString()
				if err != nil {
					return nil, err
				}
				parts = append(parts, strParts...)
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			symbol := tok.Symbol()
			if symbol != "" {
				parts = append(parts, &ExprIR{
					Kind:  ExprLiteral,
					Value: symbol,
				})
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return parts, nil
}

// buildInterpolatedString processes an interpolated string.
func (b *irBuilder) buildInterpolatedString() ([]*ExprIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeInterpolatedString

	var tokenText []byte
	var quoteType byte

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.STRING {
				tokenText = tok.Text
				if len(tokenText) > 0 {
					quoteType = tokenText[0]
				}
				b.pos++
				break
			}
		}

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeInterpolatedString {
			b.pos++
			return nil, fmt.Errorf("interpolated string at position %d has no token", startPos)
		}

		b.pos++
	}

	// Malformed string literal: must have opening and closing quotes
	if len(tokenText) < 2 || tokenText[0] != tokenText[len(tokenText)-1] {
		return nil, fmt.Errorf("malformed interpolated string literal at position %d", startPos)
	}

	var parts []*ExprIR
	content := tokenText[1 : len(tokenText)-1]
	stringParts := parser.TokenizeString(content, quoteType)

	if len(tokenText) >= 2 {
		parts = append(parts, &ExprIR{Kind: ExprLiteral, Value: string(quoteType)})
	}

	for _, part := range stringParts {
		segment := string(content[part.Start:part.End])
		if part.IsLiteral {
			if segment != "" {
				parts = append(parts, &ExprIR{Kind: ExprLiteral, Value: segment})
			}
			continue
		}

		if segment == "" {
			continue
		}

		if segment == "var" && part.PropertyStart >= 0 {
			parts = append(parts, &ExprIR{
				Kind:    ExprVarRef,
				VarName: string(content[part.PropertyStart:part.PropertyEnd]),
			})
			continue
		}

		selector := []string{}
		if part.PropertyStart >= 0 {
			selector = append(selector, string(content[part.PropertyStart:part.PropertyEnd]))
		}

		parts = append(parts, &ExprIR{
			Kind: ExprDecoratorRef,
			Decorator: &DecoratorRef{
				Name:     segment,
				Selector: selector,
			},
		})
	}

	if len(tokenText) >= 2 {
		parts = append(parts, &ExprIR{Kind: ExprLiteral, Value: string(quoteType)})
	}

	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeInterpolatedString {
			b.pos++
			break
		}
		b.pos++
	}

	return parts, nil
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
			case parser.NodeIdentifier:
				primaryExpr = b.buildIdentifierExpr()
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

// buildForStmt processes a for loop statement.
func (b *irBuilder) buildForStmt() (*StatementIR, error) {
	b.pos++ // Move past OPEN NodeFor

	var loopVar string
	var collection *ExprIR
	var body []*StatementIR

	// Skip FOR token
	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind == parser.EventToken {
			b.pos++
			break
		}
		b.pos++
	}

	// Parse: loopVar in collection { body }
	// Tokens: FOR, IDENTIFIER (loopVar), IN, IDENTIFIER (collection)
	seenIn := false
	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeFor {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IN {
				seenIn = true
				b.pos++
				continue
			}
			if tok.Type == lexer.IDENTIFIER {
				if !seenIn && loopVar == "" {
					loopVar = string(tok.Text)
				} else if seenIn && collection == nil {
					collection = &ExprIR{
						Kind:    ExprVarRef,
						VarName: string(tok.Text),
					}
				}
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeDecorator:
				collection = b.buildDecoratorExpr()
				continue
			case parser.NodeIdentifier:
				collection = b.buildIdentifierExpr()
				continue
			case parser.NodeLiteral:
				collection = b.buildLiteralExpr()
				continue
			case parser.NodeBlock:
				stmts, err := b.buildBlock()
				if err != nil {
					return nil, err
				}
				body = stmts
				continue
			}
		}

		b.pos++
	}

	return &StatementIR{
		Kind: StmtBlocker,
		Blocker: &BlockerIR{
			Kind:       BlockerFor,
			LoopVar:    loopVar,
			Collection: collection,
			ThenBranch: body,
		},
	}, nil
}

// buildWhenStmt processes a when statement (pattern matching).
func (b *irBuilder) buildWhenStmt() (*StatementIR, error) {
	b.pos++ // Move past OPEN NodeWhen

	var condition *ExprIR
	var arms []*WhenArmIR

	// Skip WHEN token
	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind == parser.EventToken {
			b.pos++
			break
		}
		b.pos++
	}

	// Parse condition expression and arms
	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeWhen {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeDecorator:
				condition = b.buildDecoratorExpr()
				continue
			case parser.NodeLiteral:
				condition = b.buildLiteralExpr()
				continue
			case parser.NodeIdentifier:
				condition = b.buildIdentifierExpr()
				continue
			case parser.NodeWhenArm:
				arm, err := b.buildWhenArm()
				if err != nil {
					return nil, err
				}
				arms = append(arms, arm)
				continue
			}
		}

		b.pos++
	}

	return &StatementIR{
		Kind: StmtBlocker,
		Blocker: &BlockerIR{
			Kind:      BlockerWhen,
			Condition: condition,
			Arms:      arms,
		},
	}, nil
}

// buildWhenArm processes a single when arm (pattern -> body).
func (b *irBuilder) buildWhenArm() (*WhenArmIR, error) {
	b.pos++ // Move past OPEN NodeWhenArm

	var pattern *ExprIR
	var body []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeWhenArm {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodePatternLiteral:
				pattern = b.buildPatternLiteral()
				continue
			case parser.NodePatternElse:
				pattern = b.buildPatternElse()
				continue
			case parser.NodePatternRegex:
				pattern = b.buildPatternRegex()
				continue
			case parser.NodePatternRange:
				pattern = b.buildPatternRange()
				continue
			}
		}

		if evt.Kind == parser.EventStepEnter {
			stmts, err := b.buildStep()
			if err != nil {
				return nil, err
			}
			body = append(body, stmts...)
			continue
		}

		b.pos++
	}

	return &WhenArmIR{
		Pattern: pattern,
		Body:    body,
	}, nil
}

// buildPatternLiteral processes a literal pattern in a when arm.
func (b *irBuilder) buildPatternLiteral() *ExprIR {
	b.pos++ // Move past OPEN NodePatternLiteral

	var value any

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodePatternLiteral {
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

// buildPatternElse processes an else pattern (catch-all) in a when arm.
func (b *irBuilder) buildPatternElse() *ExprIR {
	b.pos++ // Move past OPEN NodePatternElse

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodePatternElse {
			b.pos++
			break
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprLiteral,
		Value: "_", // Special marker for else pattern
	}
}

// buildPatternRegex processes a regex pattern in a when arm.
func (b *irBuilder) buildPatternRegex() *ExprIR {
	b.pos++ // Move past OPEN NodePatternRegex

	var pattern string

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodePatternRegex {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			pattern = string(tok.Text)
			b.pos++
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprLiteral,
		Value: pattern,
	}
}

// buildPatternRange processes a range pattern in a when arm.
func (b *irBuilder) buildPatternRange() *ExprIR {
	b.pos++ // Move past OPEN NodePatternRange

	var start, end int64
	var hasStart bool

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodePatternRange {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.INTEGER {
				val, _ := strconv.ParseInt(string(tok.Text), 10, 64)
				if !hasStart {
					start = val
					hasStart = true
				} else {
					end = val
				}
			}
			b.pos++
			continue
		}

		b.pos++
	}

	// Return as a binary expression representing the range
	return &ExprIR{
		Kind: ExprBinaryOp,
		Op:   "...",
		Left: &ExprIR{
			Kind:  ExprLiteral,
			Value: start,
		},
		Right: &ExprIR{
			Kind:  ExprLiteral,
			Value: end,
		},
	}
}

// buildTryStmt processes a try/catch/finally statement.
func (b *irBuilder) buildTryStmt() (*StatementIR, error) {
	b.pos++ // Move past OPEN NodeTry

	var tryBlock []*StatementIR
	var catchBlock []*StatementIR
	var finallyBlock []*StatementIR

	// Skip TRY token
	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind == parser.EventToken {
			b.pos++
			break
		}
		b.pos++
	}

	// Parse try/catch/finally blocks
	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeTry {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeBlock:
				if tryBlock == nil {
					stmts, err := b.buildBlock()
					if err != nil {
						return nil, err
					}
					tryBlock = stmts
				}
				continue
			case parser.NodeCatch:
				catchStmts, err := b.buildCatchClause()
				if err != nil {
					return nil, err
				}
				catchBlock = catchStmts
				continue
			case parser.NodeFinally:
				finallyStmts, err := b.buildFinallyClause()
				if err != nil {
					return nil, err
				}
				finallyBlock = finallyStmts
				continue
			}
		}

		b.pos++
	}

	return &StatementIR{
		Kind:         StmtTry,
		CreatesScope: true, // Try/catch creates a scope for error variable
		Try: &TryIR{
			TryBlock:     tryBlock,
			CatchBlock:   catchBlock,
			FinallyBlock: finallyBlock,
		},
	}, nil
}

// buildCatchClause processes a catch clause.
func (b *irBuilder) buildCatchClause() ([]*StatementIR, error) {
	b.pos++ // Move past OPEN NodeCatch

	var stmts []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeCatch {
			b.pos++
			break
		}

		// Skip error variable identifier if present (not used until runtime variables exist)
		if evt.Kind == parser.EventToken {
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			if node == parser.NodeBlock {
				blockStmts, err := b.buildBlock()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, blockStmts...)
				continue
			}
		}

		b.pos++
	}

	return stmts, nil
}

// buildFinallyClause processes a finally clause.
func (b *irBuilder) buildFinallyClause() ([]*StatementIR, error) {
	b.pos++ // Move past OPEN NodeFinally

	var stmts []*StatementIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeFinally {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			if node == parser.NodeBlock {
				blockStmts, err := b.buildBlock()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, blockStmts...)
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
		return tok.Symbol()
	}
}
