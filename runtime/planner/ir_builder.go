package planner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/types"
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
		types:     make(map[string]*StructTypeIR),
		enums:     make(map[string]*EnumTypeIR),
		exprSeq:   0,
	}

	stmts, err := b.buildSource()
	if err != nil {
		return nil, err
	}

	return &ExecutionGraph{
		Statements: stmts,
		Functions:  b.functions,
		Types:      b.types,
		Enums:      b.enums,
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
	types     map[string]*StructTypeIR
	enums     map[string]*EnumTypeIR
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

			case parser.NodeStructDecl:
				decl, err := b.buildStructDecl()
				if err != nil {
					return nil, err
				}
				if decl != nil && decl.Name != "" {
					if _, exists := b.types[decl.Name]; exists {
						return nil, fmt.Errorf("duplicate struct declaration %q", decl.Name)
					}
					if _, exists := b.enums[decl.Name]; exists {
						return nil, fmt.Errorf("duplicate type declaration %q", decl.Name)
					}
					b.types[decl.Name] = decl
				}
				continue

			case parser.NodeEnumDecl:
				decl, err := b.buildEnumDecl()
				if err != nil {
					return nil, err
				}
				if decl != nil && decl.Name != "" {
					if _, exists := b.enums[decl.Name]; exists {
						return nil, fmt.Errorf("duplicate enum declaration %q", decl.Name)
					}
					if _, exists := b.types[decl.Name]; exists {
						return nil, fmt.Errorf("duplicate type declaration %q", decl.Name)
					}
					b.enums[decl.Name] = decl
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

func (b *irBuilder) buildStructDecl() (*StructTypeIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeStructDecl

	var decl StructTypeIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeStructDecl {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && decl.Name == "" {
				decl.Name = string(tok.Text)
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeStructField {
			field, err := b.buildStructField()
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, field)
			continue
		}

		b.pos++
	}

	if decl.Name == "" {
		return nil, fmt.Errorf("struct declaration at position %d has no name", startPos)
	}

	seenFields := make(map[string]struct{}, len(decl.Fields))
	for _, field := range decl.Fields {
		if _, exists := seenFields[field.Name]; exists {
			return nil, fmt.Errorf("struct %q has duplicate field %q", decl.Name, field.Name)
		}
		seenFields[field.Name] = struct{}{}
	}

	decl.Span = SourceSpan{Start: startPos, End: b.pos}
	return &decl, nil
}

func (b *irBuilder) buildStructField() (StructFieldIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeStructField

	var field StructFieldIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeStructField {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && field.Name == "" {
				field.Name = string(tok.Text)
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeTypeAnnotation:
				field.Type = b.buildTypeAnnotation()
				continue
			case parser.NodeDefaultValue:
				field.Default = b.buildDefaultValue()
				continue
			}
		}

		b.pos++
	}

	if field.Name == "" {
		return StructFieldIR{}, fmt.Errorf("struct field at position %d has no name", startPos)
	}

	if field.Type == "" {
		return StructFieldIR{}, fmt.Errorf("struct field %q at position %d has no type", field.Name, startPos)
	}

	return field, nil
}

func (b *irBuilder) buildEnumDecl() (*EnumTypeIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeEnumDecl

	var decl EnumTypeIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeEnumDecl {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && decl.Name == "" {
				decl.Name = string(tok.Text)
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeTypeAnnotation:
				decl.BaseType = b.buildTypeAnnotation()
				continue
			case parser.NodeEnumMember:
				member, err := b.buildEnumMember()
				if err != nil {
					return nil, err
				}
				decl.Members = append(decl.Members, member)
				continue
			}
		}

		b.pos++
	}

	if decl.Name == "" {
		return nil, fmt.Errorf("enum declaration at position %d has no name", startPos)
	}

	if decl.BaseType == "" {
		decl.BaseType = "String"
	}

	seenMembers := make(map[string]struct{}, len(decl.Members))
	for _, member := range decl.Members {
		if _, exists := seenMembers[member.Name]; exists {
			return nil, fmt.Errorf("enum %q has duplicate member %q", decl.Name, member.Name)
		}
		seenMembers[member.Name] = struct{}{}
	}

	decl.Span = SourceSpan{Start: startPos, End: b.pos}
	return &decl, nil
}

func (b *irBuilder) buildEnumMember() (EnumMemberIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeEnumMember

	var member EnumMemberIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeEnumMember {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && member.Name == "" {
				member.Name = string(tok.Text)
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeDefaultValue {
			member.Value = b.buildDefaultValue()
			continue
		}

		b.pos++
	}

	if member.Name == "" {
		return EnumMemberIR{}, fmt.Errorf("enum member at position %d has no name", startPos)
	}

	return member, nil
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

	propagateGroupedParamTypes(params)
	return params, nil
}

func propagateGroupedParamTypes(params []ParamIR) {
	for i := len(params) - 1; i >= 0; i-- {
		if params[i].Type == "" {
			continue
		}

		for j := i - 1; j >= 0; j-- {
			if params[j].Type != "" || params[j].Default != nil {
				break
			}
			params[j].Type = params[i].Type
		}
	}
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
	optional := false

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
			if tok.Type == lexer.QUESTION {
				optional = true
			}
			b.pos++
			continue
		}

		b.pos++
	}

	if optional && typeName != "" {
		return typeName + "?"
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
			if node == parser.NodeBinaryExpr && expr != nil {
				expr = b.buildBinaryExprWithLeft(expr)
				continue
			}
			if node == parser.NodeTypeCast && expr != nil {
				expr = b.buildTypeCastExprWithLeft(expr)
				continue
			}
			if parsed, ok := b.buildExprFromNode(node, true); ok {
				expr = b.consumeExprTail(parsed)
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.EQUALS {
				b.pos++
				continue
			}
			if expr != nil {
				b.pos++
				continue
			}
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

			case parser.NodeFunctionCall:
				stmt, err := b.buildFunctionCallStmt()
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

			case parser.NodeRedirect:
				if len(stmts) == 0 {
					return nil, fmt.Errorf("redirect at position %d has no preceding command", b.pos)
				}
				lastStmt := stmts[len(stmts)-1]
				if lastStmt.Kind != StmtCommand || lastStmt.Command == nil {
					return nil, fmt.Errorf("redirect at position %d does not follow a command", b.pos)
				}
				mode, target, err := b.buildRedirect()
				if err != nil {
					return nil, err
				}
				lastStmt.Command.RedirectMode = mode
				lastStmt.Command.RedirectTarget = target
				continue
			}
		}

		if evt.Kind == parser.EventToken {
			// Check for operator tokens between commands within a step
			tok := b.tokens[evt.Data]
			switch tok.Type {
			case lexer.AND_AND, lexer.OR_OR, lexer.PIPE, lexer.SEMICOLON:
				// Apply operator to the last command statement
				if len(stmts) > 0 {
					lastStmt := stmts[len(stmts)-1]
					if lastStmt.Kind == StmtCommand && lastStmt.Command != nil {
						lastStmt.Command.Operator = tok.Symbol()
					}
				}
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return stmts, nil
}

func (b *irBuilder) buildFunctionCallStmt() (*StatementIR, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeFunctionCall

	var name string
	var args []ArgIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeFunctionCall {
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

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeParamList {
			parsedArgs, err := b.buildDecoratorArgs()
			if err != nil {
				return nil, err
			}
			args = parsedArgs
			continue
		}

		b.pos++
	}

	if name == "" {
		return nil, fmt.Errorf("function call at position %d has no name", startPos)
	}

	return &StatementIR{
		Kind: StmtFunctionCall,
		Span: SourceSpan{Start: startPos, End: b.pos},
		FunctionCall: &FunctionCallStmtIR{
			Name: name,
			Args: args,
		},
	}, nil
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
			if node == parser.NodeBinaryExpr && value != nil {
				value = b.buildBinaryExprWithLeft(value)
				continue
			}
			if node == parser.NodeTypeCast && value != nil {
				value = b.buildTypeCastExprWithLeft(value)
				continue
			}
			if parsed, ok := b.buildExprFromNode(node, false); ok {
				value = b.consumeExprTail(parsed)
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

func (b *irBuilder) buildCommandExprParts(closeNode parser.NodeKind, allowShellArgs bool) ([]*ExprIR, error) {
	var parts []*ExprIR

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == closeNode {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)

			switch node {
			case parser.NodeShellArg:
				if allowShellArgs {
					if b.shellArgNeedsSpace() {
						parts = append(parts, &ExprIR{Kind: ExprLiteral, Value: " "})
					}
					argParts, err := b.buildShellArg()
					if err != nil {
						return nil, err
					}
					parts = append(parts, argParts...)
					continue
				}
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
				parts = append(parts, &ExprIR{Kind: ExprLiteral, Value: symbol})
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return parts, nil
}

// buildShellCommand processes a shell command.
func (b *irBuilder) buildShellCommand() (*StatementIR, error) {
	b.pos++ // Move past OPEN NodeShellCommand

	parts, err := b.buildCommandExprParts(parser.NodeShellCommand, true)
	if err != nil {
		return nil, err
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

func (b *irBuilder) buildRedirect() (string, *CommandExpr, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeRedirect

	var mode string
	var target *CommandExpr

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeRedirect {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			switch tok.Type {
			case lexer.GT:
				mode = ">"
			case lexer.APPEND:
				mode = ">>"
			case lexer.LT:
				mode = "<"
			}
			b.pos++
			continue
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeRedirectTarget {
			parsedTarget, err := b.buildRedirectTarget()
			if err != nil {
				return "", nil, err
			}
			target = parsedTarget
			continue
		}

		b.pos++
	}

	if mode == "" {
		return "", nil, fmt.Errorf("redirect at position %d missing operator", startPos)
	}
	if target == nil {
		return "", nil, fmt.Errorf("redirect at position %d missing target", startPos)
	}

	return mode, target, nil
}

func (b *irBuilder) buildRedirectTarget() (*CommandExpr, error) {
	startPos := b.pos
	b.pos++ // Move past OPEN NodeRedirectTarget

	parts, err := b.buildCommandExprParts(parser.NodeRedirectTarget, true)
	if err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("redirect target at position %d has no parts", startPos)
	}

	return &CommandExpr{Parts: parts}, nil
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
				b.scopes.Push()
				blockStmts, err := b.buildBlock()
				b.scopes.Pop()
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
	args = canonicalizeDecoratorArgsForStatement(decoratorName, args)

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
			if node == parser.NodeBinaryExpr && arg.Value != nil {
				arg.Value = b.buildBinaryExprWithLeft(arg.Value)
				continue
			}
			if node == parser.NodeTypeCast && arg.Value != nil {
				arg.Value = b.buildTypeCastExprWithLeft(arg.Value)
				continue
			}
			if parsed, ok := b.buildExprFromNode(node, true); ok {
				arg.Value = b.consumeExprTail(parsed)
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

	return b.buildCommandExprParts(parser.NodeShellArg, false)
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

func (b *irBuilder) buildExprFromNode(node parser.NodeKind, allowBinary bool) (*ExprIR, bool) {
	switch node {
	case parser.NodeLiteral:
		return b.buildLiteralExpr(), true
	case parser.NodeArrayLiteral:
		return b.buildArrayLiteralExpr(), true
	case parser.NodeObjectLiteral:
		return b.buildObjectLiteralExpr(), true
	case parser.NodeDecorator:
		return b.buildDecoratorExpr(), true
	case parser.NodeIdentifier:
		return b.buildIdentifierExpr(), true
	case parser.NodeQualifiedRef:
		return b.buildQualifiedRefExpr(), true
	case parser.NodeBinaryExpr:
		if allowBinary {
			return b.buildBinaryExpr(), true
		}
	}

	return nil, false
}

func (b *irBuilder) buildArrayLiteralExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeArrayLiteral

	// Store elements as []*ExprIR to preserve expressions (decorators, variables, etc.)
	// This allows arrays like ["a", @var.x, 1+2] to be fully represented
	elements := make([]*ExprIR, 0)

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeArrayLiteral {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			// Use buildExprFromNode to handle all expression types uniformly
			// This preserves decorators (@var.x), identifiers, binary expressions, etc.
			if expr, ok := b.buildExprFromNode(node, true); ok {
				elements = append(elements, expr)
				continue
			}
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprLiteral,
		Value: elements,
	}
}

func (b *irBuilder) buildObjectLiteralExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeObjectLiteral

	// Store field values as map[string]*ExprIR to preserve expressions
	// This allows objects like {name: @var.x, count: 1+2} to be fully represented
	fields := make(map[string]*ExprIR)

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeObjectLiteral {
			b.pos++
			break
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeObjectField {
			b.pos++ // Move past OPEN NodeObjectField

			// Get key (should be TOKEN)
			var key string
			if b.pos < len(b.events) && b.events[b.pos].Kind == parser.EventToken {
				key = string(b.tokens[b.events[b.pos].Data].Text)
				b.pos++ // Move past key token
			}

			// Skip colon token
			if b.pos < len(b.events) && b.events[b.pos].Kind == parser.EventToken {
				b.pos++
			}

			// Parse value as *ExprIR to preserve expressions
			var value *ExprIR
			if b.pos < len(b.events) && b.events[b.pos].Kind == parser.EventOpen {
				node := parser.NodeKind(b.events[b.pos].Data)
				if expr, ok := b.buildExprFromNode(node, true); ok {
					value = expr
				}
			}

			if key != "" && value != nil {
				fields[key] = value
			}

			// Skip CLOSE ObjectField
			if b.pos < len(b.events) && b.events[b.pos].Kind == parser.EventClose {
				b.pos++
			}
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:  ExprLiteral,
		Value: fields,
	}
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
	var args []*ExprIR
	var argNames []string

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

		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			if node == parser.NodeParamList {
				parsedArgs, err := b.buildDecoratorArgs()
				if err == nil {
					for _, arg := range parsedArgs {
						if arg.Value != nil {
							args = append(args, arg.Value)
							argNames = append(argNames, arg.Name)
						}
					}
				}
				continue
			}
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
	argNames = canonicalizeDecoratorArgNames(name, argNames, len(selector) > 0)

	// @var.X becomes a VarRef
	if name == "var" && len(selector) > 0 {
		return &ExprIR{
			Kind:    ExprVarRef,
			VarName: selector[0],
		}
	}

	// @var("X") becomes the same VarRef as @var.X.
	if name == "var" && len(args) > 0 {
		if args[0] != nil && args[0].Kind == ExprLiteral {
			if varName, ok := args[0].Value.(string); ok && varName != "" {
				return &ExprIR{
					Kind:    ExprVarRef,
					VarName: varName,
				}
			}
		}
	}

	return &ExprIR{
		Kind: ExprDecoratorRef,
		Decorator: &DecoratorRef{
			Name:     name,
			Selector: selector,
			Args:     args,
			ArgNames: argNames,
		},
	}
}

func canonicalizeDecoratorArgsForStatement(decoratorName string, args []ArgIR) []ArgIR {
	if len(args) == 0 {
		return args
	}

	rawNames := make([]string, len(args))
	for i, arg := range args {
		rawNames[i] = arg.Name
	}

	canonical := canonicalizeDecoratorArgNames(decoratorName, rawNames, false)
	result := make([]ArgIR, len(args))
	copy(result, args)
	for i := range result {
		result[i].Name = canonical[i]
	}

	return result
}

func canonicalizeDecoratorArgNames(path string, rawNames []string, hasPrimary bool) []string {
	if len(rawNames) == 0 {
		return nil
	}

	schema, ok := decoratorSchema(path)
	if !ok {
		result := make([]string, len(rawNames))
		copy(result, rawNames)
		return result
	}

	return canonicalizeNamesWithSchema(rawNames, schema, hasPrimary)
}

func decoratorSchema(path string) (types.DecoratorSchema, bool) {
	candidates := []string{path, strings.TrimPrefix(path, "@")}
	seen := map[string]struct{}{}

	var entry decorator.Entry
	ok := false
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}

		entry, ok = decorator.Global().Lookup(candidate)
		if ok {
			break
		}
	}
	if !ok {
		return types.DecoratorSchema{}, false
	}

	schema := entry.Impl.Descriptor().Schema
	if schema.Path == "" && len(schema.Parameters) == 0 && schema.PrimaryParameter == "" {
		return types.DecoratorSchema{}, false
	}

	return schema, true
}

func canonicalizeNamesWithSchema(rawNames []string, schema types.DecoratorSchema, hasPrimary bool) []string {
	result := make([]string, len(rawNames))
	copy(result, rawNames)

	ordered := plannerPositionalBindingOrder(schema.GetOrderedParameters())
	namedReservations := make(map[string]bool)
	for _, rawName := range rawNames {
		if rawName == "" {
			continue
		}
		namedReservations[canonicalParamName(schema, rawName)] = true
	}

	provided := make(map[string]bool)
	filled := make(map[int]bool)
	if hasPrimary && schema.PrimaryParameter != "" {
		provided[schema.PrimaryParameter] = true
		for i, param := range ordered {
			if param.Name == schema.PrimaryParameter {
				filled[i] = true
				break
			}
		}
	}

	nextPos := 0
	for i, rawName := range rawNames {
		if rawName != "" {
			name := canonicalParamName(schema, rawName)
			result[i] = name
			provided[name] = true
			for pos, param := range ordered {
				if param.Name == name {
					filled[pos] = true
					break
				}
			}
			continue
		}

		for nextPos < len(ordered) {
			candidate := ordered[nextPos]
			if !filled[nextPos] && !provided[candidate.Name] && !namedReservations[candidate.Name] {
				result[i] = candidate.Name
				filled[nextPos] = true
				provided[candidate.Name] = true
				nextPos++
				break
			}
			nextPos++
		}
	}

	return result
}

func canonicalParamName(schema types.DecoratorSchema, name string) string {
	if replacement, ok := schema.DeprecatedParameters[name]; ok {
		return replacement
	}
	return name
}

func plannerPositionalBindingOrder(params []types.ParamSchema) []types.ParamSchema {
	if len(params) == 0 {
		return nil
	}

	ordered := make([]types.ParamSchema, 0, len(params))
	for _, param := range params {
		if param.Required {
			ordered = append(ordered, param)
		}
	}
	for _, param := range params {
		if !param.Required {
			ordered = append(ordered, param)
		}
	}

	return ordered
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

func (b *irBuilder) buildQualifiedRefExpr() *ExprIR {
	b.pos++ // Move past OPEN NodeQualifiedRef

	parts := make([]string, 0, 2)

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeQualifiedRef {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER {
				parts = append(parts, string(tok.Text))
			}
			b.pos++
			continue
		}

		b.pos++
	}

	expr := &ExprIR{Kind: ExprEnumMemberRef}
	if len(parts) > 0 {
		expr.EnumName = parts[0]
	}
	if len(parts) > 1 {
		expr.EnumMember = parts[1]
	}

	return expr
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

			if node == parser.NodeBinaryExpr {
				// Binary expression without explicit left side (shouldn't happen)
				condition = b.buildBinaryExpr()
				continue
			}

			if node == parser.NodeTypeCast && primaryExpr != nil {
				primaryExpr = b.buildTypeCastExprWithLeft(primaryExpr)
				condition = primaryExpr
				continue
			}

			if parsed, ok := b.buildExprFromNode(node, false); ok {
				primaryExpr = b.consumeExprTail(parsed)
				condition = primaryExpr
				continue
			}

			switch node {
			case parser.NodeBlock:
				if thenBranch == nil {
					b.scopes.Push()
					stmts, err := b.buildBlock()
					b.scopes.Pop()
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

		// Handle control flow statements that appear directly in blocks
		// (without a step enter event, e.g., nested if statements)
		if evt.Kind == parser.EventOpen {
			node := parser.NodeKind(evt.Data)
			switch node {
			case parser.NodeIf:
				ifStmt, err := b.buildIfStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, ifStmt)
				continue
			case parser.NodeFor:
				forStmt, err := b.buildForStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, forStmt)
				continue
			case parser.NodeWhen:
				whenStmt, err := b.buildWhenStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, whenStmt)
				continue
			case parser.NodeTry:
				tryStmt, err := b.buildTryStmt()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, tryStmt)
				continue
			}
		}

		b.pos++
	}

	return stmts, nil
}

// buildElseClause processes an else clause.
func (b *irBuilder) buildElseClause() ([]*StatementIR, error) {
	b.pos++ // Move past OPEN NodeElse
	b.scopes.Push()
	defer b.scopes.Pop()

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
			case parser.NodeBlock:
				b.scopes.Push()
				stmts, err := b.buildBlock()
				b.scopes.Pop()
				if err != nil {
					return nil, err
				}
				body = stmts
				continue
			default:
				if parsed, ok := b.buildExprFromNode(node, false); ok {
					collection = b.consumeExprTail(parsed)
					continue
				}
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

			if parsed, ok := b.buildExprFromNode(node, false); ok {
				condition = b.consumeExprTail(parsed)
				continue
			}

			if node == parser.NodeWhenArm {
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
	b.scopes.Push()
	defer b.scopes.Pop()

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
			case parser.NodeQualifiedRef:
				pattern = b.buildQualifiedRefExpr()
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
					b.scopes.Push()
					stmts, err := b.buildBlock()
					b.scopes.Pop()
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
				b.scopes.Push()
				blockStmts, err := b.buildBlock()
				b.scopes.Pop()
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
				b.scopes.Push()
				blockStmts, err := b.buildBlock()
				b.scopes.Pop()
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

func (b *irBuilder) consumeExprTail(left *ExprIR) *ExprIR {
	if left == nil {
		return nil
	}

	for b.pos < len(b.events) {
		evt := b.events[b.pos]
		if evt.Kind != parser.EventOpen {
			break
		}

		node := parser.NodeKind(evt.Data)
		switch node {
		case parser.NodeTypeCast:
			left = b.buildTypeCastExprWithLeft(left)
		case parser.NodeBinaryExpr:
			left = b.buildBinaryExprWithLeft(left)
		default:
			return left
		}
	}

	return left
}

func (b *irBuilder) buildTypeCastExprWithLeft(left *ExprIR) *ExprIR {
	b.pos++ // Move past OPEN NodeTypeCast

	typeName := ""
	optional := false

	for b.pos < len(b.events) {
		evt := b.events[b.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeTypeCast {
			b.pos++
			break
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			if tok.Type == lexer.IDENTIFIER && typeName == "" {
				typeName = string(tok.Text)
			}
			if tok.Type == lexer.QUESTION {
				optional = true
			}
			b.pos++
			continue
		}

		b.pos++
	}

	return &ExprIR{
		Kind:     ExprTypeCast,
		Left:     left,
		TypeName: typeName,
		Optional: optional,
	}
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
			if node == parser.NodeBinaryExpr {
				if right != nil {
					right = b.buildBinaryExprWithLeft(right)
				} else if left != nil {
					right = b.buildBinaryExprWithLeft(left)
					left = nil
				} else {
					right = b.buildBinaryExpr()
				}
				continue
			}
			if node == parser.NodeTypeCast {
				if right != nil {
					right = b.buildTypeCastExprWithLeft(right)
				} else if left != nil {
					left = b.buildTypeCastExprWithLeft(left)
				} else {
					b.pos++
				}
				continue
			}

			expr, ok := b.buildExprFromNode(node, false)
			if !ok {
				b.pos++
				continue
			}

			if left == nil {
				left = expr
			} else if right == nil {
				right = expr
			}
			continue
		}

		if evt.Kind == parser.EventToken {
			tok := b.tokens[evt.Data]
			switch tok.Type {
			case lexer.PLUS:
				op = "+"
			case lexer.MINUS:
				op = "-"
			case lexer.MULTIPLY:
				op = "*"
			case lexer.DIVIDE:
				op = "/"
			case lexer.MODULO:
				op = "%"
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
			case lexer.PLUS:
				op = "+"
			case lexer.MINUS:
				op = "-"
			case lexer.MULTIPLY:
				op = "*"
			case lexer.DIVIDE:
				op = "/"
			case lexer.MODULO:
				op = "%"
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
			if node == parser.NodeBinaryExpr {
				if right != nil {
					right = b.buildBinaryExprWithLeft(right)
				} else {
					right = b.buildBinaryExpr()
				}
				continue
			}

			if node == parser.NodeTypeCast {
				if right != nil {
					right = b.buildTypeCastExprWithLeft(right)
				} else {
					left = b.buildTypeCastExprWithLeft(left)
				}
				continue
			}

			if parsed, ok := b.buildExprFromNode(node, false); ok {
				right = parsed
			} else {
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
		value := string(tok.Text)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
			value = value[1 : len(value)-1]
		}
		return value
	case lexer.INTEGER:
		val, _ := strconv.ParseInt(string(tok.Text), 10, 64)
		return val
	case lexer.FLOAT:
		val, _ := strconv.ParseFloat(string(tok.Text), 64)
		return val
	case lexer.DURATION:
		return durationLiteral(tok.Symbol())
	case lexer.BOOLEAN:
		return string(tok.Text) == "true"
	case lexer.NONE:
		return nil
	case lexer.IDENTIFIER:
		return tok.Symbol()
	default:
		return tok.Symbol()
	}
}
