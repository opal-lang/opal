package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/types"
	"github.com/opal-lang/opal/runtime/lexer"
)

// Parse parses the input bytes and returns a parse tree
// Takes []byte directly for zero-copy performance
func Parse(source []byte, opts ...ParserOpt) *ParseTree {
	config := &ParserConfig{}
	for _, opt := range opts {
		opt(config)
	}

	var telemetry *ParseTelemetry
	var debugEvents []DebugEvent
	var startTotal time.Time

	// Initialize telemetry if enabled
	if config.telemetry >= TelemetryBasic {
		telemetry = &ParseTelemetry{}
		if config.telemetry >= TelemetryTiming {
			startTotal = time.Now()
		}
	}

	// Initialize debug if enabled
	if config.debug > DebugOff {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	// Lex the input first
	var startLex time.Time
	if config.telemetry >= TelemetryTiming {
		startLex = time.Now()
	}

	lex := lexer.NewLexer()
	lex.Init(source)
	tokens := lex.GetTokens()

	if config.telemetry >= TelemetryBasic {
		telemetry.TokenCount = len(tokens)
		if config.telemetry >= TelemetryTiming {
			telemetry.LexTime = time.Since(startLex)
		}
	}

	// Create parser with pre-allocated buffers
	// Heuristic: ~3 events per token (Open, Token, Close for simple nodes)
	eventCap := len(tokens) * 3
	if eventCap < 16 {
		eventCap = 16
	}

	p := &parser{
		tokens:        tokens,
		pos:           0,
		events:        make([]Event, 0, eventCap),
		errors:        make([]ParseError, 0, 4), // Most parses have 0-4 errors
		config:        config,
		debugEvents:   debugEvents,
		functionNames: collectTopLevelFunctionNames(tokens),
	}

	// Parse the file
	var startParse time.Time
	if config.telemetry >= TelemetryTiming {
		startParse = time.Now()
	}

	p.file()

	if config.telemetry >= TelemetryBasic {
		telemetry.EventCount = len(p.events)
		telemetry.ErrorCount = len(p.errors)
		if config.telemetry >= TelemetryTiming {
			telemetry.ParseTime = time.Since(startParse)
			telemetry.TotalTime = time.Since(startTotal)
		}
	}

	return &ParseTree{
		Source:      source,
		Tokens:      tokens,
		Events:      p.events,
		Errors:      p.errors,
		Warnings:    p.warnings,
		Telemetry:   telemetry,
		DebugEvents: p.debugEvents,
	}
}

// ParseString is a convenience wrapper for tests
func ParseString(input string, opts ...ParserOpt) *ParseTree {
	return Parse([]byte(input), opts...)
}

// ParseTokens parses pre-lexed tokens (for benchmarking pure parse performance)
func ParseTokens(source []byte, tokens []lexer.Token, opts ...ParserOpt) *ParseTree {
	config := &ParserConfig{}
	for _, opt := range opts {
		opt(config)
	}

	var telemetry *ParseTelemetry
	var debugEvents []DebugEvent
	var startTotal time.Time

	// Initialize telemetry if enabled
	if config.telemetry >= TelemetryBasic {
		telemetry = &ParseTelemetry{}
		if config.telemetry >= TelemetryTiming {
			startTotal = time.Now()
		}
	}

	// Initialize debug if enabled
	if config.debug > DebugOff {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	// Create parser with pre-allocated buffers
	eventCap := len(tokens) * 3
	if eventCap < 16 {
		eventCap = 16
	}

	p := &parser{
		tokens:        tokens,
		pos:           0,
		events:        make([]Event, 0, eventCap),
		errors:        make([]ParseError, 0, 4),
		config:        config,
		debugEvents:   debugEvents,
		functionNames: collectTopLevelFunctionNames(tokens),
	}

	// Parse the file
	var startParse time.Time
	if config.telemetry >= TelemetryTiming {
		startParse = time.Now()
	}

	p.file()

	if config.telemetry >= TelemetryBasic {
		telemetry.EventCount = len(p.events)
		telemetry.ErrorCount = len(p.errors)
		telemetry.TokenCount = len(tokens)
		if config.telemetry >= TelemetryTiming {
			telemetry.ParseTime = time.Since(startParse)
			telemetry.TotalTime = time.Since(startTotal)
		}
	}

	return &ParseTree{
		Source:      source,
		Tokens:      tokens,
		Events:      p.events,
		Errors:      p.errors,
		Warnings:    p.warnings,
		Telemetry:   telemetry,
		DebugEvents: p.debugEvents,
	}
}

// parser is the internal parser state
type parser struct {
	tokens        []lexer.Token
	pos           int
	events        []Event
	errors        []ParseError
	warnings      []ParseWarning
	config        *ParserConfig
	debugEvents   []DebugEvent
	functionNames map[string]struct{}
}

func collectTopLevelFunctionNames(tokens []lexer.Token) map[string]struct{} {
	names := make(map[string]struct{})
	braceDepth := 0

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok.Type {
		case lexer.LBRACE:
			braceDepth++
		case lexer.RBRACE:
			if braceDepth > 0 {
				braceDepth--
			}
		case lexer.FUN:
			if braceDepth != 0 {
				continue
			}
			next := i + 1
			for next < len(tokens) && (tokens[next].Type == lexer.NEWLINE || tokens[next].Type == lexer.COMMENT) {
				next++
			}
			if next < len(tokens) && tokens[next].Type == lexer.IDENTIFIER {
				names[string(tokens[next].Text)] = struct{}{}
			}
		}
	}

	return names
}

// recordDebugEvent records debug events when debug tracing is enabled
func (p *parser) recordDebugEvent(event, context string) {
	if p.config.debug == DebugOff || p.debugEvents == nil {
		return
	}

	p.debugEvents = append(p.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		TokenPos:  p.pos,
		Context:   context,
	})
}

// file parses the top-level source structure (file, stdin, or string)
func (p *parser) file() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_source", "parsing source")
	}

	kind := p.start(NodeSource)

	// Parse top-level declarations
	for !p.at(lexer.EOF) {
		prevPos := p.pos

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("file_loop_iteration", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		// Skip newlines at top level
		if p.at(lexer.NEWLINE) {
			p.advance()
			continue
		}

		// Check if this is an executable step (not control flow)
		// Steps: var declarations, decorators, shell commands
		// NOT steps: fun, if, for, when, try (control flow/metaprogramming/definitions)
		isStep := p.at(lexer.VAR) || p.at(lexer.AT) || p.at(lexer.IDENTIFIER)

		if isStep {
			p.events = append(p.events, Event{Kind: EventStepEnter, Data: 0})
		}

		if p.at(lexer.FUN) {
			p.function()
		} else if p.at(lexer.STRUCT) {
			p.structDecl()
		} else if p.at(lexer.ENUM) {
			p.enumDecl()
		} else if p.at(lexer.VAR) {
			p.varDecl()
		} else if p.at(lexer.IF) {
			// If statement at top level (script mode)
			p.ifStmt()
		} else if p.at(lexer.FOR) {
			// For loop at top level (script mode)
			p.forStmt()
		} else if p.at(lexer.TRY) {
			// Try/catch at top level (script mode)
			p.tryStmt()
		} else if p.at(lexer.WHEN) {
			// When statement at top level (script mode)
			p.whenStmt()
		} else if p.at(lexer.CATCH) {
			// Catch without try at top level
			p.errors = append(p.errors, ParseError{
				Position:   p.current().Position,
				Message:    "catch without try",
				Context:    "top-level statement",
				Got:        lexer.CATCH,
				Suggestion: "catch must follow a try block",
				Example:    "try { kubectl apply } catch { kubectl rollback }",
			})
			p.advance() // Skip the catch keyword
		} else if p.at(lexer.FINALLY) {
			// Finally without try at top level
			p.errors = append(p.errors, ParseError{
				Position:   p.current().Position,
				Message:    "finally without try",
				Context:    "top-level statement",
				Got:        lexer.FINALLY,
				Suggestion: "finally must follow a try block",
				Example:    "try { kubectl apply } finally { echo \"done\" }",
			})
			p.advance() // Skip the finally keyword
		} else if p.at(lexer.AT) {
			// Decorator at top level (script mode)
			p.decorator()
		} else if p.at(lexer.IDENTIFIER) {
			p.identifierStatement()
		} else {
			// Unknown token, skip for now
			p.advance()
		}

		if isStep {
			p.events = append(p.events, Event{Kind: EventStepExit, Data: 0})
		}

		// INVARIANT: Parser must make progress in each iteration
		invariant.Invariant(p.pos > prevPos || p.at(lexer.EOF), "parser stuck in file() at pos %d - no progress made", p.pos)
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_source", "source complete")
	}
}

// function parses a function declaration: fun IDENTIFIER ParamList Block
func (p *parser) function() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_function", "parsing function")
	}

	kind := p.start(NodeFunction)

	// Consume 'fun' keyword
	p.token()

	// Consume function name
	if p.at(lexer.IDENTIFIER) {
		if p.functionNames == nil {
			p.functionNames = make(map[string]struct{})
		}
		p.functionNames[string(p.current().Text)] = struct{}{}
		p.token()
	}

	// Parse parameter list (optional)
	if p.at(lexer.LPAREN) {
		p.paramList()
	}

	// Parse body: either = expression/shell or block (required)
	if p.at(lexer.EQUALS) {
		p.token() // Consume '='

		// Emit step boundary for function body (consistency with block syntax)
		p.events = append(p.events, Event{Kind: EventStepEnter, Data: 0})

		// After '=', could be shell command or expression
		if p.at(lexer.IDENTIFIER) {
			// Shell command
			p.shellCommand()
		} else {
			// Expression
			p.expression()
		}

		p.events = append(p.events, Event{Kind: EventStepExit, Data: 0})
	} else if p.at(lexer.LBRACE) {
		// Block
		p.block()
	} else {
		// Missing function body - report error
		p.errorExpected(lexer.LBRACE, "function body")
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_function", "function complete")
	}
}

// structDecl parses a struct declaration: struct IDENTIFIER { fields }
func (p *parser) structDecl() {
	kind := p.start(NodeStructDecl)

	// Consume 'struct' keyword
	p.token()

	// Consume struct name
	p.expect(lexer.IDENTIFIER, "struct declaration")

	if p.at(lexer.COLON) || (p.at(lexer.IDENTIFIER) && strings.EqualFold(string(p.current().Text), "extends")) {
		p.errorWithDetails(
			"struct inheritance is not supported",
			"struct declaration",
			"Use nested fields for composition instead of inheritance",
		)
		for !p.at(lexer.LBRACE) && !p.at(lexer.NEWLINE) && !p.at(lexer.EOF) {
			p.advance()
		}
	}

	// Parse body
	if !p.expect(lexer.LBRACE, "struct declaration") {
		p.finish(kind)
		return
	}

	p.skipNewlines()
	for !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		prevPos := p.pos

		p.skipNewlines()
		if p.at(lexer.RBRACE) || p.at(lexer.EOF) {
			break
		}

		if p.at(lexer.FUN) || (p.at(lexer.IDENTIFIER) && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.LPAREN) {
			p.errorWithDetails(
				"struct methods are not supported",
				"struct declaration",
				"Move behavior into top-level functions and keep structs as data-only declarations",
			)
			p.skipUnsupportedStructMember()
			p.skipNewlines()
			continue
		}

		p.structField()
		p.skipNewlines()

		if p.at(lexer.COMMA) {
			p.token()
			p.skipNewlines()
		}

		if p.pos == prevPos && !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
			p.advance()
		}
	}

	p.expect(lexer.RBRACE, "struct declaration")
	p.finish(kind)
}

// enumDecl parses an enum declaration: enum IDENTIFIER [Type] { members }
func (p *parser) enumDecl() {
	kind := p.start(NodeEnumDecl)

	// Consume 'enum' keyword
	p.token()

	// Consume enum name
	p.expect(lexer.IDENTIFIER, "enum declaration")

	// Optional base type annotation
	if p.hasGoStyleTypeAnnotation() {
		p.typeAnnotation()
	}

	// Parse body
	if !p.expect(lexer.LBRACE, "enum declaration") {
		p.finish(kind)
		return
	}

	p.skipNewlines()
	for !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		prevPos := p.pos

		p.skipNewlines()
		if p.at(lexer.RBRACE) || p.at(lexer.EOF) {
			break
		}

		p.enumMember()
		p.skipNewlines()

		if p.at(lexer.COMMA) {
			p.token()
			p.skipNewlines()
		}

		if p.pos == prevPos && !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
			p.advance()
		}
	}

	p.expect(lexer.RBRACE, "enum declaration")
	p.finish(kind)
}

func (p *parser) enumMember() {
	kind := p.start(NodeEnumMember)

	// Member name
	p.expect(lexer.IDENTIFIER, "enum member")

	// Optional explicit enum value
	if p.at(lexer.EQUALS) {
		p.enumMemberDefaultValue()
	}

	p.finish(kind)
}

func (p *parser) enumMemberDefaultValue() {
	kind := p.start(NodeDefaultValue)

	// Consume '='
	p.token()
	p.skipNewlines()

	if p.at(lexer.EOF) || p.at(lexer.NEWLINE) || p.at(lexer.COMMA) || p.at(lexer.RBRACE) {
		p.errorWithDetails(
			"missing enum member value",
			"enum member",
			"Add a quoted string after '='",
		)
		p.finish(kind)
		return
	}

	if p.at(lexer.STRING) {
		p.token()
		p.finish(kind)
		return
	}

	p.errors = append(p.errors, ParseError{
		Position:   p.current().Position,
		Message:    "enum member value must be a string literal",
		Context:    "enum member",
		Got:        p.current().Type,
		Expected:   []lexer.TokenType{lexer.STRING},
		Suggestion: "Wrap enum member values in quotes",
		Example:    `Prod = "production"`,
	})

	if !p.at(lexer.EOF) && !p.at(lexer.NEWLINE) && !p.at(lexer.COMMA) && !p.at(lexer.RBRACE) {
		p.advance()
	}

	p.finish(kind)
}

func (p *parser) skipUnsupportedStructMember() {
	braceDepth := 0
	parenDepth := 0
	bracketDepth := 0
	for !p.at(lexer.EOF) {
		if braceDepth == 0 && parenDepth == 0 && bracketDepth == 0 && (p.at(lexer.NEWLINE) || p.at(lexer.COMMA) || p.at(lexer.RBRACE)) {
			return
		}

		if p.at(lexer.LBRACE) {
			braceDepth++
			p.advance()
			continue
		}

		if p.at(lexer.LPAREN) {
			parenDepth++
			p.advance()
			continue
		}

		if p.at(lexer.LSQUARE) {
			bracketDepth++
			p.advance()
			continue
		}

		if p.at(lexer.RBRACE) {
			if braceDepth == 0 {
				return
			}
			braceDepth--
			p.advance()
			continue
		}

		if p.at(lexer.RPAREN) {
			if parenDepth > 0 {
				parenDepth--
			}
			p.advance()
			continue
		}

		if p.at(lexer.RSQUARE) {
			if bracketDepth > 0 {
				bracketDepth--
			}
			p.advance()
			continue
		}

		p.advance()
	}
}

func (p *parser) structField() {
	kind := p.start(NodeStructField)

	// Field name
	p.expect(lexer.IDENTIFIER, "struct field")

	// Required field type annotation (go-style only)
	if p.hasGoStyleTypeAnnotation() {
		p.typeAnnotation()
	} else {
		p.errorWithDetails(
			"missing field type annotation",
			"struct field",
			"Use typed fields: struct Config { retries Int }",
		)
	}

	// Optional field default value
	if p.at(lexer.EQUALS) {
		p.defaultValue()
	}

	p.finish(kind)
}

// paramList parses a parameter list: ( params )
func (p *parser) paramList() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_paramList", "parsing param list")
	}

	kind := p.start(NodeParamList)

	// Expect '('
	p.expect(lexer.LPAREN, "parameter list")
	p.skipNewlines()

	// Parse parameters (comma-separated)
	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		p.skipNewlines()
		if p.at(lexer.RPAREN) || p.at(lexer.EOF) {
			break
		}

		p.param()
		p.skipNewlines()

		// If there's a comma, consume it and continue
		if !p.at(lexer.COMMA) {
			// No comma means we're done with parameters
			break
		}
		p.token()
		p.skipNewlines()
	}

	// Expect ')'
	p.expect(lexer.RPAREN, "parameter list")

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_paramList", "param list complete")
	}
}

// param parses a single parameter.
//
// Function parameters require explicit type annotations.
// Supported forms:
//   - name Type
//   - name Type = expression
func (p *parser) param() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_param", "parsing parameter")
	}

	kind := p.start(NodeParam)

	// Consume parameter name
	if p.at(lexer.IDENTIFIER) {
		p.token()
	}

	// Parse required type annotation
	hasType := false
	if p.hasGoStyleTypeAnnotation() {
		p.typeAnnotation()
		hasType = true
	}
	if !hasType && !p.hasGroupedTypeAhead() {
		p.errorWithDetails(
			"missing parameter type annotation",
			"function parameter",
			"Use typed parameters: fun deploy(env String) { ... }",
		)
	}

	// Parse optional default value
	if p.at(lexer.EQUALS) {
		p.defaultValue()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_param", "parameter complete")
	}
}

func (p *parser) hasGoStyleTypeAnnotation() bool {
	if !p.at(lexer.IDENTIFIER) {
		return false
	}

	if p.pos+1 >= len(p.tokens) {
		return true
	}

	if p.tokens[p.pos+1].Type == lexer.QUESTION {
		if p.pos+2 >= len(p.tokens) {
			return true
		}

		switch p.tokens[p.pos+2].Type {
		case lexer.COMMA, lexer.RPAREN, lexer.EQUALS, lexer.NEWLINE, lexer.LBRACE, lexer.EOF:
			return true
		default:
			return false
		}
	}

	switch p.tokens[p.pos+1].Type {
	case lexer.COMMA, lexer.RPAREN, lexer.EQUALS, lexer.NEWLINE, lexer.LBRACE, lexer.EOF:
		return true
	default:
		return false
	}
}

// hasGroupedTypeAhead checks whether a trailing parameter group provides a type,
// allowing forms like: fun f(name, alias String)
func (p *parser) hasGroupedTypeAhead() bool {
	if !p.at(lexer.COMMA) {
		return false
	}

	parenDepth := 0
	braceDepth := 0
	bracketDepth := 0

	for i := p.pos + 1; i < len(p.tokens); i++ {
		tok := p.tokens[i]

		switch tok.Type {
		case lexer.LPAREN:
			parenDepth++
		case lexer.RPAREN:
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				return false
			}
			if parenDepth > 0 {
				parenDepth--
			}
		case lexer.LBRACE:
			braceDepth++
		case lexer.RBRACE:
			if braceDepth > 0 {
				braceDepth--
			}
		case lexer.LSQUARE:
			bracketDepth++
		case lexer.RSQUARE:
			if bracketDepth > 0 {
				bracketDepth--
			}
		}

		if parenDepth != 0 || braceDepth != 0 || bracketDepth != 0 {
			continue
		}

		if tok.Type != lexer.IDENTIFIER || i+1 >= len(p.tokens) {
			continue
		}

		next := p.tokens[i+1].Type
		if next == lexer.IDENTIFIER {
			return true
		}
	}

	return false
}

// typeAnnotation parses a type annotation: Type
func (p *parser) typeAnnotation() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_typeAnnotation", "parsing type annotation")
	}

	kind := p.start(NodeTypeAnnotation)

	// Consume type name
	p.expect(lexer.IDENTIFIER, "type annotation")
	if p.at(lexer.QUESTION) {
		p.token()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_typeAnnotation", "type annotation complete")
	}
}

// defaultValue parses a default value: = expression
func (p *parser) defaultValue() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_defaultValue", "parsing default value")
	}

	kind := p.start(NodeDefaultValue)

	// Consume '='
	if p.at(lexer.EQUALS) {
		p.token()
	}
	p.skipNewlines()

	// Parse expression (for now, just consume one token - string literal, number, etc.)
	// TODO: Full expression parsing in later iteration
	if p.at(lexer.EOF) || p.at(lexer.RPAREN) || p.at(lexer.COMMA) {
		p.errorWithDetails(
			"missing default parameter value",
			"function parameter default value",
			"Add a value after '='",
		)
	} else {
		p.token()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_defaultValue", "default value complete")
	}
}

// block parses a block: { statements }
func (p *parser) block() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_block", "parsing block")
	}

	kind := p.start(NodeBlock)

	// Expect '{'
	p.expect(lexer.LBRACE, "function body")

	// Parse statements
	for !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		prevPos := p.pos

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("block_loop_iteration", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		p.statement()

		// INVARIANT: Parser must make progress in each iteration
		// If statement() didn't advance, we need to force progress to avoid infinite loop
		if p.pos == prevPos && !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
			if p.config.debug >= DebugDetailed {
				p.recordDebugEvent("block_force_progress", fmt.Sprintf("pos: %d, forcing advance on %v", p.pos, p.current().Type))
			}
			// Force progress by advancing past the problematic token
			p.advance()
		}
	}

	// Expect '}'
	p.expect(lexer.RBRACE, "function body")

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_block", "block complete")
	}
}

// statement parses a statement
func (p *parser) statement() {
	// Skip newlines (statement separators)
	for p.at(lexer.NEWLINE) {
		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("statement_skip_newline", fmt.Sprintf("pos: %d", p.pos))
		}
		p.advance()
	}

	// Check if this is an executable step (not control flow)
	// Steps: var declarations, decorators, assignments, shell commands
	// NOT steps: if, for, when, try (control flow/metaprogramming)
	isStep := p.at(lexer.VAR) || p.at(lexer.AT) || p.at(lexer.IDENTIFIER)

	if isStep {
		p.events = append(p.events, Event{Kind: EventStepEnter, Data: 0})
	}

	if p.at(lexer.FUN) {
		// Function declarations not allowed inside blocks
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "function declarations must be at top level",
			Context:    "statement",
			Got:        lexer.FUN,
			Suggestion: "Move function declaration outside of blocks",
			Example:    "fun helper() { ... } at top level, not inside if/for/etc",
		})
		p.advance() // Skip the fun keyword
	} else if p.at(lexer.STRUCT) {
		// Struct declarations not allowed inside blocks
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "struct declarations must be at top level",
			Context:    "statement",
			Got:        lexer.STRUCT,
			Suggestion: "Move struct declaration outside of blocks",
			Example:    "struct Config { retries Int } at top level",
		})
		p.advance() // Skip the struct keyword
	} else if p.at(lexer.ENUM) {
		// Enum declarations not allowed inside blocks
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "enum declarations must be at top level",
			Context:    "statement",
			Got:        lexer.ENUM,
			Suggestion: "Move enum declaration outside of blocks",
			Example:    "enum Stage { Dev Prod } at top level",
		})
		p.advance() // Skip the enum keyword
	} else if p.at(lexer.VAR) {
		p.varDecl()
	} else if p.at(lexer.IF) {
		p.ifStmt()
	} else if p.at(lexer.FOR) {
		p.forStmt()
	} else if p.at(lexer.TRY) {
		p.tryStmt()
	} else if p.at(lexer.WHEN) {
		p.whenStmt()
	} else if p.at(lexer.ELSE) {
		// Else without matching if
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "else without matching if",
			Context:    "statement",
			Got:        lexer.ELSE,
			Suggestion: "else must follow an if statement",
			Example:    "if condition { ... } else { ... }",
		})
		p.advance() // Skip the else keyword
	} else if p.at(lexer.CATCH) {
		// Catch without try
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "catch without try",
			Context:    "statement",
			Got:        lexer.CATCH,
			Suggestion: "catch must follow a try block",
			Example:    "try { ... } catch { ... }",
			Note:       "catch handles errors from the preceding try block",
		})
		p.advance() // Skip the catch keyword
	} else if p.at(lexer.FINALLY) {
		// Finally without try
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "finally without try",
			Context:    "statement",
			Got:        lexer.FINALLY,
			Suggestion: "finally must follow a try block",
			Example:    "try { ... } finally { ... }",
			Note:       "finally always executes after try (and catch if present)",
		})
		p.advance() // Skip the finally keyword
	} else if p.at(lexer.AT) {
		// Decorator (execution decorator with block)
		p.decorator()

		// Check for shell operators after decorator (for piping, chaining)
		// e.g., @timeout(5s) { echo "test" } | grep "pattern"
		if p.isShellOperator() {
			p.token() // Consume operator (&&, ||, |)

			// Parse next command after operator
			if !p.isStatementBoundary() && !p.at(lexer.EOF) {
				p.shellCommand()
			}
		}
	} else if p.at(lexer.IDENTIFIER) {
		p.identifierStatement()
	} else if !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		// Unknown statement - error recovery
		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("error_recovery_start", fmt.Sprintf("pos: %d, unexpected %v in statement", p.pos, p.current().Type))
		}

		p.errorUnexpected("statement")
		p.recover()

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("recovery_sync_found", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		// Consume separator to guarantee progress
		if p.at(lexer.NEWLINE) || p.at(lexer.SEMICOLON) {
			if p.config.debug >= DebugDetailed {
				p.recordDebugEvent("consumed_separator", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
			}
			p.advance()
		}
	}

	if isStep {
		p.events = append(p.events, Event{Kind: EventStepExit, Data: 0})
	}
}

// ifStmt parses an if statement: if condition { ... } [else { ... }]
func (p *parser) ifStmt() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_if", "parsing if statement")
	}

	kind := p.start(NodeIf)

	// Consume 'if' keyword
	p.token()

	// Check for missing condition (block immediately after if)
	if p.at(lexer.LBRACE) {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing condition after 'if'",
			Context:    "if statement",
			Got:        lexer.LBRACE,
			Expected:   []lexer.TokenType{lexer.BOOLEAN, lexer.IDENTIFIER},
			Suggestion: "Add a boolean condition before the block",
			Example:    "if true { ... } or if @var.enabled { ... }",
		})
		// Continue parsing the block despite the error
	} else if !p.at(lexer.EOF) {
		// Parse condition expression (supports comparisons like @var.X == "value")
		p.expression()
	}

	// Parse then block
	if p.at(lexer.LBRACE) {
		p.block()
	} else {
		p.errorExpected(lexer.LBRACE, "if statement body")
	}

	// Parse optional else clause
	if p.at(lexer.ELSE) {
		p.elseClause()
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_if", "if statement complete")
	}
}

// elseClause parses an else clause: else { ... } or else if { ... }
func (p *parser) elseClause() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_else", "parsing else clause")
	}

	kind := p.start(NodeElse)

	// Consume 'else' keyword
	p.token()

	// Check for 'else if' pattern
	if p.at(lexer.IF) {
		// Recursive: else if is parsed as else { if ... }
		p.ifStmt()
	} else if p.at(lexer.LBRACE) {
		// Regular else block
		p.block()
	} else {
		p.errorExpected(lexer.LBRACE, "else clause body")
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_else", "else clause complete")
	}
}

// forStmt parses a for loop: for item in collection { ... }
func (p *parser) forStmt() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_for", "parsing for loop")
	}

	kind := p.start(NodeFor)

	// Consume 'for' keyword
	p.token()

	// Parse loop variable (item)
	if p.at(lexer.IDENTIFIER) {
		p.token()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing loop variable after 'for'",
			Context:    "for loop",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.IDENTIFIER},
			Suggestion: "Add a variable name to hold each item",
			Example:    "for item in items { ... }",
			Note:       "for loops unroll at plan-time; the loop variable is resolved during planning",
		})
	}

	// Expect 'in' keyword
	if p.at(lexer.IN) {
		p.token()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing 'in' keyword in for loop",
			Context:    "for loop",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.IN},
			Suggestion: "Add 'in' between loop variable and collection",
			Example:    "for item in items { ... }",
		})
	}

	// Parse collection expression (identifier, decorator, or range)
	p.parseForCollection()

	// Parse loop body
	if p.at(lexer.LBRACE) {
		p.block()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing block after for loop header",
			Context:    "for loop body",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.LBRACE},
			Suggestion: "Add a block with the loop body",
			Example:    "for item in items { echo @var.item }",
		})
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_for", "for loop complete")
	}
}

// parseForCollection parses the collection expression in a for loop.
// Handles: identifier, decorator (@var.list), or range (1...10).
func (p *parser) parseForCollection() {
	// Check if this is a range expression by looking ahead
	isRange := false
	if p.at(lexer.INTEGER) {
		// Check if next token is DOTDOTDOT
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.DOTDOTDOT {
			isRange = true
		}
	} else if p.at(lexer.AT) {
		// Check if there's a DOTDOTDOT after the decorator
		// Decorator is: @ IDENTIFIER [. IDENTIFIER]
		lookahead := p.pos + 1 // skip @
		if lookahead < len(p.tokens) && (p.tokens[lookahead].Type == lexer.IDENTIFIER || p.tokens[lookahead].Type == lexer.VAR) {
			lookahead++ // skip decorator name
			if lookahead < len(p.tokens) && p.tokens[lookahead].Type == lexer.DOT {
				lookahead++ // skip .
				if lookahead < len(p.tokens) && p.tokens[lookahead].Type == lexer.IDENTIFIER {
					lookahead++ // skip property
				}
			}
			if lookahead < len(p.tokens) && p.tokens[lookahead].Type == lexer.DOTDOTDOT {
				isRange = true
			}
		}
	}

	if isRange {
		// Parse as range expression
		rangeKind := p.start(NodeRange)

		// Parse start expression (number or decorator)
		if p.at(lexer.INTEGER) {
			p.token()
		} else if p.at(lexer.AT) {
			p.parseDecorator()
		}

		// Consume ...
		if p.at(lexer.DOTDOTDOT) {
			p.token()
		}

		// Parse end expression (number or decorator)
		if p.at(lexer.INTEGER) {
			p.token()
		} else if p.at(lexer.AT) {
			p.parseDecorator()
		} else {
			p.errors = append(p.errors, ParseError{
				Position:   p.current().Position,
				Message:    "missing end expression in range",
				Context:    "for loop range",
				Got:        p.current().Type,
				Expected:   []lexer.TokenType{lexer.INTEGER, lexer.AT},
				Suggestion: "Provide the end value for the range",
				Example:    "for i in 1...10 { ... }",
			})
		}

		p.finish(rangeKind)
	} else if p.at(lexer.IDENTIFIER) {
		p.token()
	} else if p.at(lexer.AT) {
		p.parseDecorator()
	} else if p.at(lexer.LSQUARE) {
		// Array literal: ["a", "b", "c"]
		p.arrayLiteral()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing collection expression in for loop",
			Context:    "for loop",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.IDENTIFIER, lexer.AT, lexer.INTEGER, lexer.LSQUARE},
			Suggestion: "Provide a collection to iterate over",
			Example:    "for item in items { ... } or for i in 1...10 { ... } or for x in [1, 2, 3] { ... }",
			Note:       "collection must resolve at plan-time to a list of concrete values",
		})
	}
}

// parseDecorator parses a decorator expression: @var.name, @env.HOME
func (p *parser) parseDecorator() {
	kind := p.start(NodeDecorator)
	p.token() // @
	if p.at(lexer.IDENTIFIER) || p.at(lexer.VAR) {
		p.token() // decorator name
	}
	if p.at(lexer.DOT) {
		p.token() // .
		if p.at(lexer.IDENTIFIER) {
			p.token() // property name
		}
	}
	p.finish(kind)
}

// tryStmt parses a try statement: try { ... } [catch { ... }] [finally { ... }]
func (p *parser) tryStmt() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_try", "parsing try statement")
	}

	kind := p.start(NodeTry)

	// Consume 'try' keyword
	p.token()

	// Parse try block
	if p.at(lexer.LBRACE) {
		p.block()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing block after 'try'",
			Context:    "try statement",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.LBRACE},
			Suggestion: "Add a block with the try body",
			Example:    "try { kubectl apply } catch { kubectl rollback }",
			Note:       "try/catch is the only non-deterministic construct; plan records all paths",
		})
	}

	// Parse optional catch block
	if p.at(lexer.CATCH) {
		p.catchClause()
	}

	// Parse optional finally block
	if p.at(lexer.FINALLY) {
		p.finallyClause()
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_try", "try statement complete")
	}
}

// catchClause parses a catch clause: catch { ... }
func (p *parser) catchClause() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_catch", "parsing catch clause")
	}

	kind := p.start(NodeCatch)

	// Consume 'catch' keyword
	p.token()

	// Parse catch block
	if p.at(lexer.LBRACE) {
		p.block()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing block after 'catch'",
			Context:    "catch clause",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.LBRACE},
			Suggestion: "Add a block with the catch body",
			Example:    "catch { echo \"Error occurred\" }",
		})
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_catch", "catch clause complete")
	}
}

// finallyClause parses a finally clause: finally { ... }
func (p *parser) finallyClause() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_finally", "parsing finally clause")
	}

	kind := p.start(NodeFinally)

	// Consume 'finally' keyword
	p.token()

	// Parse finally block
	if p.at(lexer.LBRACE) {
		p.block()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing block after 'finally'",
			Context:    "finally clause",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.LBRACE},
			Suggestion: "Add a block with the finally body",
			Example:    "finally { echo \"Cleanup\" }",
			Note:       "finally always executes, regardless of try or catch outcome",
		})
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_finally", "finally clause complete")
	}
}

// whenStmt parses a when statement: when expression { pattern -> body ... }
func (p *parser) whenStmt() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_when", "parsing when statement")
	}

	kind := p.start(NodeWhen)

	// Consume 'when' keyword
	p.token()

	// Parse match expression (what we're matching against)
	if p.at(lexer.LBRACE) {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing expression after 'when'",
			Context:    "when statement",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.IDENTIFIER, lexer.AT},
			Suggestion: "Add an expression to match against",
			Example:    "when @var.ENV { ... }",
		})
	} else if p.at(lexer.EOF) {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing expression after 'when'",
			Context:    "when statement",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.IDENTIFIER, lexer.AT},
			Suggestion: "Add an expression to match against",
			Example:    "when @var.ENV { ... }",
		})
	} else {
		p.expression()
	}

	// Expect opening brace
	if !p.at(lexer.LBRACE) {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing '{' after when expression",
			Context:    "when statement",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.LBRACE},
			Suggestion: "Add '{' to start the pattern arms",
			Example:    `when @var.ENV { "prod" -> deploy else -> echo "skip" }`,
			Note:       "when is plan-time pattern matching; only the matching branch expands",
		})
	} else {
		p.token() // consume '{'
	}

	// Parse when arms (pattern -> body)
	for !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		// Skip newlines between arms
		for p.at(lexer.NEWLINE) {
			p.advance()
		}

		// Check again after skipping newlines
		if p.at(lexer.RBRACE) || p.at(lexer.EOF) {
			break
		}

		p.whenArm()
	}

	// Expect closing brace
	if p.at(lexer.RBRACE) {
		p.token()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing '}' after when arms",
			Context:    "when statement",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.RBRACE},
			Suggestion: "Add '}' to close the when statement",
			Example:    `when @var.ENV { "prod" -> deploy }`,
		})
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_when", "when statement complete")
	}
}

// whenArm parses a single when arm: pattern -> (expression | block)
func (p *parser) whenArm() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_when_arm", "parsing when arm")
	}

	kind := p.start(NodeWhenArm)

	// Parse pattern
	p.pattern()

	// Expect arrow
	if !p.at(lexer.ARROW) {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "missing '->' after pattern",
			Context:    "when arm",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.ARROW},
			Suggestion: "Add '->' between pattern and body",
			Example:    `"production" -> deploy`,
		})
	} else {
		p.token() // consume '->'
	}

	// Parse body (can be expression or block)
	if p.at(lexer.LBRACE) {
		p.block()
	} else {
		// Single statement - use statement() to handle all cases including FUN check
		p.statement()
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_when_arm", "when arm complete")
	}
}

// pattern parses a pattern for when statements (Phase 1: string literals and else only)
func (p *parser) pattern() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_pattern", "parsing pattern")
	}

	// Parse the first (left) pattern
	p.patternPrimary()

	// Handle OR patterns: "a" | "b" | "c"
	// Left-associative: parses as (("a" | "b") | "c")
	for p.at(lexer.PIPE) {
		orKind := p.start(NodePatternOr)
		p.token()          // consume |
		p.patternPrimary() // parse right side
		p.finish(orKind)
	}

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_pattern", "pattern complete")
	}
}

// patternPrimary parses a single pattern (without OR)
func (p *parser) patternPrimary() {
	if p.at(lexer.ELSE) {
		// else pattern (catch-all)
		kind := p.start(NodePatternElse)
		p.token()
		p.finish(kind)
	} else if p.at(lexer.STRING) {
		// String literal pattern
		kind := p.start(NodePatternLiteral)
		p.token()
		p.finish(kind)
	} else if p.at(lexer.IDENTIFIER) && string(p.current().Text) == "r" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.STRING {
		// Regex pattern: r"^pattern$"
		kind := p.start(NodePatternRegex)
		p.token() // consume 'r'
		p.token() // consume string
		p.finish(kind)
	} else if p.at(lexer.INTEGER) && p.pos+2 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.DOTDOTDOT {
		// Numeric range pattern: 200...299
		kind := p.start(NodePatternRange)
		p.token() // consume start integer
		p.token() // consume ...
		if p.at(lexer.INTEGER) {
			p.token() // consume end integer
		} else {
			p.errors = append(p.errors, ParseError{
				Position:   p.current().Position,
				Message:    "missing end value in range pattern",
				Context:    "when arm",
				Got:        p.current().Type,
				Expected:   []lexer.TokenType{lexer.INTEGER},
				Suggestion: "Add end value after ...",
				Example:    `200...299 -> success`,
			})
		}
		p.finish(kind)
	} else if p.isQualifiedRefExpression() {
		// Enum member pattern: Type.Member
		p.qualifiedRef()
	} else {
		p.errors = append(p.errors, ParseError{
			Position:   p.current().Position,
			Message:    "invalid pattern",
			Context:    "when arm",
			Got:        p.current().Type,
			Expected:   []lexer.TokenType{lexer.STRING, lexer.ELSE, lexer.IDENTIFIER, lexer.INTEGER, lexer.DOT},
			Suggestion: "Use a string literal, regex pattern, numeric range, Type.Member, or else",
			Example:    `"production" -> deploy or OS.Windows -> deploy or 200...299 -> success`,
			Note:       "Range patterns use ... (three dots); validation happens at plan-time",
		})
		p.advance()
	}
}

// shellCommand parses a shell command and its arguments
// Uses HasSpaceBefore to determine argument boundaries
// Consumes tokens until a shell operator (&&, ||, |) or statement boundary
func (p *parser) shellCommand() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_shell_command", "parsing shell command")
	}

	kind := p.start(NodeShellCommand)

	// Parse shell arguments until we hit an operator or boundary
	for !p.isShellOperator() && !p.isStatementBoundary() {
		prevPos := p.pos

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("shell_arg_start", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		// Parse a single shell argument (may be multiple tokens without spaces)
		p.shellArg()

		// INVARIANT: must make progress
		invariant.Invariant(p.pos > prevPos, "parser stuck in shellCommand() at pos %d, token: %v", p.pos, p.current().Type)
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_shell_command", "shell command complete")
	}

	// If we stopped at a shell operator, validate and consume it
	if p.isShellOperator() {
		// Check if it's a redirect operator
		if p.at(lexer.GT) || p.at(lexer.APPEND) || p.at(lexer.LT) {
			p.shellRedirect()

			// CRITICAL FIX: After redirect, check for chaining operators (&&, ||, |, ;)
			// This allows: echo a > out && echo b (both redirect AND chaining)
			if p.isShellOperator() && !p.at(lexer.GT) && !p.at(lexer.APPEND) && !p.at(lexer.LT) {
				p.token() // Consume chaining operator (&&, ||, |, ;)

				// Parse next command after operator
				if !p.isStatementBoundary() && !p.at(lexer.EOF) {
					p.shellCommand()
				}
			}
		} else {
			p.token() // Consume operator (&&, ||, |, ;)

			// Parse next command after operator
			if !p.isStatementBoundary() && !p.at(lexer.EOF) {
				p.shellCommand()
			}
		}
	}
}

// shellRedirect parses output redirection (> or >>)
// PRECONDITION: Current token is GT or APPEND
func (p *parser) shellRedirect() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_shell_redirect", "parsing redirect")
	}

	kind := p.start(NodeRedirect)

	// Consume redirect operator (> or >>)
	p.token()

	// Parse redirect target
	targetKind := p.start(NodeRedirectTarget)

	// Target can be:
	// - A path: output.txt
	// - A variable: @var.OUTPUT_FILE
	// - A decorator: @file.temp() (future)
	if !p.isStatementBoundary() && !p.isShellOperator() && !p.at(lexer.EOF) {
		p.shellArg() // Parse target as shell argument
	}

	p.finish(targetKind)
	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_shell_redirect", "redirect complete")
	}
}

// shellArg parses a single shell argument
// Consumes tokens until we hit a space (HasSpaceBefore on next token)
// or a shell operator or statement boundary
// PRECONDITION: Must NOT be called when at operator or boundary (caller's responsibility)
func (p *parser) shellArg() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_shell_arg", "parsing shell argument")
	}

	// PRECONDITION CHECK: shellArg should never be called at operator/boundary
	invariant.Precondition(!p.isShellOperator() && !p.isStatementBoundary(),
		"shellArg() called at operator/boundary, pos: %d, token: %v", p.pos, p.current().Type)

	kind := p.start(NodeShellArg)

	// Check if first token is a STRING that needs interpolation
	if p.at(lexer.STRING) && p.stringNeedsInterpolation() {
		// Parse string with interpolation
		p.stringLiteral()
	} else if p.at(lexer.AT) {
		// Decorator: @var.HOME, @env.PATH, etc.
		p.decorator()
	} else {
		// Consume first token (guaranteed to exist due to precondition)
		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("shell_arg_first_token", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}
		p.token()

		// Consume additional tokens that form this argument (no space between them)
		// Loop continues while: not at operator, not at boundary, and no space before current token
		for !p.isShellOperator() && !p.isStatementBoundary() && !p.current().HasSpaceBefore {
			prevPos := p.pos

			if p.config.debug >= DebugDetailed {
				p.recordDebugEvent("shell_arg_continue_token", fmt.Sprintf("pos: %d, token: %v, hasSpace: %v",
					p.pos, p.current().Type, p.current().HasSpaceBefore))
			}

			p.token() // Emit token event (planner will group based on HasSpaceBefore)

			// INVARIANT: p.token() MUST increment p.pos
			invariant.Invariant(p.pos > prevPos, "parser stuck in shellArg() at pos %d (was %d), token: %v - token() failed to increment position",
				p.pos, prevPos, p.current().Type)
		}
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_shell_arg", "shell argument complete")
	}
}

// isShellOperator checks if current token is a shell operator that splits commands
func (p *parser) isShellOperator() bool {
	return p.at(lexer.AND_AND) || // &&
		p.at(lexer.OR_OR) || // ||
		p.at(lexer.PIPE) || // |
		p.at(lexer.SEMICOLON) || // ;
		p.at(lexer.GT) || // >
		p.at(lexer.APPEND) || // >>
		p.at(lexer.LT)
}

// isStatementBoundary checks if current token ends a statement
func (p *parser) isStatementBoundary() bool {
	return p.at(lexer.NEWLINE) ||
		p.at(lexer.RBRACE) ||
		p.at(lexer.EOF) ||
		p.at(lexer.ELSE) // Stop at else (for when arms and if/else)
}

// varDecl parses a variable declaration:
//   - Simple form: var IDENTIFIER = expression
//   - Block form: var ( IDENTIFIER = expression; ... )
func (p *parser) varDecl() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_var_decl", "parsing variable declaration")
	}

	// Check for block form: var ( ... )
	// Peek ahead: if next token after VAR is LPAREN, it's a block
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.LPAREN {
		p.varDeclBlock()
	} else {
		// Simple form: var name = value
		p.varDeclSingle()
	}

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_var_decl", "variable declaration complete")
	}
}

// varDeclSingle parses a single variable declaration: var IDENTIFIER = expression
func (p *parser) varDeclSingle() {
	kind := p.start(NodeVarDecl)

	// Consume 'var' keyword
	p.token()

	// Expect identifier
	if !p.expect(lexer.IDENTIFIER, "variable declaration") {
		p.finish(kind)
		return
	}

	// Expect '='
	if !p.expect(lexer.EQUALS, "variable declaration") {
		p.finish(kind)
		return
	}

	// Parse expression
	p.expression()

	p.finish(kind)
}

// varDeclBlock parses a block of variable declarations: var ( IDENTIFIER = expression; ... )
func (p *parser) varDeclBlock() {
	// Consume 'var' keyword
	p.token()

	// Consume '('
	p.token()

	// Skip any leading newlines
	for p.at(lexer.NEWLINE) {
		p.token()
	}

	// Parse variable declarations until ')'
	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		// Each declaration is wrapped in NodeVarDecl (but without 'var' keyword)
		p.varDeclSingleWithoutVar()

		// Consume optional newline or semicolon separators (can be multiple)
		for p.at(lexer.NEWLINE) || p.at(lexer.SEMICOLON) {
			p.token()
		}

		// Break if we hit closing paren
		if p.at(lexer.RPAREN) {
			break
		}
	}

	// Expect closing ')'
	if !p.expect(lexer.RPAREN, "variable declaration block") {
		return
	}
}

// varDeclSingleWithoutVar parses a single variable declaration without the 'var' keyword: IDENTIFIER = expression
func (p *parser) varDeclSingleWithoutVar() {
	kind := p.start(NodeVarDecl)

	// Expect identifier
	if !p.expect(lexer.IDENTIFIER, "variable declaration") {
		p.finish(kind)
		return
	}

	// Expect '='
	if !p.expect(lexer.EQUALS, "variable declaration") {
		p.finish(kind)
		return
	}

	// Parse expression
	p.expression()

	p.finish(kind)
}

// assignmentStmt parses an assignment statement: IDENTIFIER OP= expression
func (p *parser) assignmentStmt() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_assignment", "parsing assignment statement")
	}

	kind := p.start(NodeAssignment)

	// Consume identifier
	p.token()

	// Consume assignment operator (+=, -=, *=, /=, %=)
	p.token()

	// Parse expression
	p.expression()

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_assignment", "assignment statement complete")
	}
}

func (p *parser) identifierStatement() {
	if p.isFunctionCallSyntax() {
		functionName := string(p.current().Text)
		if !p.isKnownFunction(functionName) {
			p.errorWithDetails(
				fmt.Sprintf("unknown function %q", functionName),
				"function call",
				fmt.Sprintf("Define it with fun %s(...) { ... } or run a shell command by adding a space: %s (...)", functionName, functionName),
			)
		}
		p.functionCall()
		return
	}

	// Check if this is an assignment statement or shell command
	// Look ahead to see if next token is an assignment operator
	nextPos := p.pos + 1
	if nextPos < len(p.tokens) {
		nextType := p.tokens[nextPos].Type
		if nextType == lexer.PLUS_ASSIGN ||
			nextType == lexer.MINUS_ASSIGN ||
			nextType == lexer.MULTIPLY_ASSIGN ||
			nextType == lexer.DIVIDE_ASSIGN ||
			nextType == lexer.MODULO_ASSIGN {
			p.assignmentStmt()
		} else {
			p.shellCommand()
		}
		return
	}

	p.shellCommand()
}

func (p *parser) isFunctionCallSyntax() bool {
	if !p.at(lexer.IDENTIFIER) {
		return false
	}

	nextPos := p.pos + 1
	if nextPos >= len(p.tokens) {
		return false
	}

	nextToken := p.tokens[nextPos]
	if nextToken.Type != lexer.LPAREN {
		return false
	}

	current := p.current()
	return nextToken.Position.Offset == current.Position.Offset+len(current.Text)
}

func (p *parser) isKnownFunction(name string) bool {
	if p.functionNames == nil {
		return false
	}
	_, exists := p.functionNames[name]
	return exists
}

// functionCall parses a function call statement: name(arg1, key=value)
func (p *parser) functionCall() {
	kind := p.start(NodeFunctionCall)

	// Function name
	p.token()

	paramListKind := p.start(NodeParamList)
	if !p.expect(lexer.LPAREN, "function call arguments") {
		p.finish(paramListKind)
		p.finish(kind)
		return
	}
	p.skipNewlines()

	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		p.skipNewlines()
		if p.at(lexer.RPAREN) || p.at(lexer.EOF) {
			break
		}

		argKind := p.start(NodeParam)

		// Named argument: key = expr
		if p.at(lexer.IDENTIFIER) {
			nextPos := p.pos + 1
			if nextPos < len(p.tokens) && p.tokens[nextPos].Type == lexer.EQUALS {
				p.token() // key
				p.token() // =
				p.skipNewlines()

				if p.at(lexer.COMMA) || p.at(lexer.RPAREN) || p.at(lexer.EOF) {
					p.errorWithDetails(
						"missing argument value",
						"function call argument",
						"Add a value after '='",
					)
				} else {
					p.expression()
				}
			} else {
				p.expression()
			}
		} else {
			p.expression()
		}

		p.finish(argKind)
		p.skipNewlines()

		if p.at(lexer.COMMA) {
			p.token()
			p.skipNewlines()
		} else if !p.at(lexer.RPAREN) {
			p.errorWithDetails(
				"expected ',' or ')' in function call",
				"function call arguments",
				"Separate arguments with ',' and close with ')'",
			)
			break
		}
	}

	p.expect(lexer.RPAREN, "function call arguments")
	p.finish(paramListKind)
	p.finish(kind)
}

// expression parses an expression
func (p *parser) expression() {
	p.binaryExpr(0) // Start with lowest precedence
}

// binaryExpr parses binary expressions with precedence
func (p *parser) binaryExpr(minPrec int) {
	// Parse left side (primary expression)
	p.primary()

	// Check for postfix increment/decrement (++ and --)
	// These have highest precedence, so handle before binary operators
	if p.at(lexer.INCREMENT) || p.at(lexer.DECREMENT) {
		kind := p.start(NodePostfixExpr)
		p.token() // Consume ++ or --
		p.finish(kind)
	}

	for p.at(lexer.AS) {
		kind := p.start(NodeTypeCast)
		p.token() // Consume AS keyword
		p.expect(lexer.IDENTIFIER, "type cast")
		if p.at(lexer.QUESTION) {
			p.token()
		}
		p.finish(kind)
	}

	// Parse binary operators
	for {
		prec := p.precedence()
		if prec == 0 || prec < minPrec {
			break
		}

		// We have a binary operator
		kind := p.start(NodeBinaryExpr)
		p.token() // Consume operator

		// Parse right side with higher precedence
		p.binaryExpr(prec + 1)

		p.finish(kind)
	}
}

// primary parses a primary expression (literal, identifier, etc.)
func (p *parser) primary() {
	// Check for unary operators (! and -)
	if p.at(lexer.NOT) || p.at(lexer.MINUS) {
		kind := p.start(NodeUnaryExpr)
		p.token()   // Consume ! or -
		p.primary() // Parse operand (recursive for multiple unary operators)
		p.finish(kind)
		return
	}

	// Check for prefix increment/decrement (++ and --)
	if p.at(lexer.INCREMENT) || p.at(lexer.DECREMENT) {
		kind := p.start(NodePrefixExpr)
		p.token()   // Consume ++ or --
		p.primary() // Parse operand
		p.finish(kind)
		return
	}

	switch {
	case p.at(lexer.INTEGER), p.at(lexer.FLOAT), p.at(lexer.BOOLEAN), p.at(lexer.NONE):
		// Literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)

	case p.at(lexer.STRING):
		// String - check if it needs interpolation
		p.stringLiteral()

	case p.at(lexer.AT):
		// Decorator: @var.name, @env.HOME
		// In expression context, don't check for blocks (the { might be part of if/for/etc.)
		p.decoratorInExpressionContext()

	case p.at(lexer.IDENTIFIER):
		if p.isQualifiedRefExpression() {
			p.qualifiedRef()
		} else {
			// Identifier
			kind := p.start(NodeIdentifier)
			p.token()
			p.finish(kind)
		}

	case p.at(lexer.LSQUARE):
		// Array literal: [expr, expr, ...]
		p.arrayLiteral()

	case p.at(lexer.LBRACE):
		// Object literal: {key: value, ...}
		p.objectLiteral()

	default:
		// Unexpected token - report error and create error node
		p.errorUnexpected("expression")
		// Advance to prevent infinite loop
		if !p.at(lexer.EOF) {
			p.advance()
		}
	}
}

func (p *parser) isQualifiedRefExpression() bool {
	if !p.at(lexer.IDENTIFIER) {
		return false
	}

	nextPos := p.pos + 1
	if nextPos >= len(p.tokens) || p.tokens[nextPos].Type != lexer.DOT {
		return false
	}

	memberPos := p.pos + 2
	if memberPos >= len(p.tokens) || p.tokens[memberPos].Type != lexer.IDENTIFIER {
		return false
	}

	return true
}

func (p *parser) qualifiedRef() {
	kind := p.start(NodeQualifiedRef)

	// Parse exactly Type.Member
	p.token() // Type
	p.token() // .
	p.token() // Member

	if p.at(lexer.DOT) {
		p.errorWithDetails(
			"qualified reference must use Type.Member",
			"expression",
			"Use exactly two segments for qualified constants in this phase",
		)

		for p.at(lexer.DOT) {
			p.token()
			if p.at(lexer.IDENTIFIER) {
				p.token()
				continue
			}
			break
		}
	}

	p.finish(kind)
}

// arrayLiteral parses an array literal: [expr, expr, ...]
func (p *parser) arrayLiteral() {
	kind := p.start(NodeArrayLiteral)
	p.expect(lexer.LSQUARE, "array literal") // Consume '['
	p.skipNewlines()

	// Parse elements
	for !p.at(lexer.RSQUARE) && !p.at(lexer.EOF) {
		p.skipNewlines()
		if p.at(lexer.RSQUARE) || p.at(lexer.EOF) {
			break
		}

		// Parse element expression
		p.expression()
		p.skipNewlines()

		// Check for comma or end of array
		if p.at(lexer.COMMA) {
			p.token() // Consume comma
			p.skipNewlines()
			// Allow trailing comma before ]
			if p.at(lexer.RSQUARE) {
				break
			}
		} else if !p.at(lexer.RSQUARE) {
			// Expected comma or ]
			p.errorWithDetails("expected ',' or ']' in array literal", "array literal", "")
			break
		}
	}

	p.expect(lexer.RSQUARE, "array literal") // Consume ']'
	p.finish(kind)
}

// objectLiteral parses an object literal: {key: value, ...}
func (p *parser) objectLiteral() {
	kind := p.start(NodeObjectLiteral)
	p.expect(lexer.LBRACE, "object literal") // Consume '{'
	p.skipNewlines()

	// Parse fields
	for !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		p.skipNewlines()
		if p.at(lexer.RBRACE) || p.at(lexer.EOF) {
			break
		}

		// Parse field: key: value
		p.objectField()
		p.skipNewlines()

		// Check for comma or end of object
		if p.at(lexer.COMMA) {
			p.token() // Consume comma
			p.skipNewlines()
			// Allow trailing comma before }
			if p.at(lexer.RBRACE) {
				break
			}
		} else if !p.at(lexer.RBRACE) {
			// Expected comma or }
			p.errorWithDetails("expected ',' or '}' in object literal", "object literal", "")
			break
		}
	}

	p.expect(lexer.RBRACE, "object literal") // Consume '}'
	p.finish(kind)
}

// objectField parses a single object field: key: value
func (p *parser) objectField() {
	kind := p.start(NodeObjectField)

	// Parse key (must be identifier)
	if !p.at(lexer.IDENTIFIER) {
		p.errorExpected(lexer.IDENTIFIER, "object field")
		p.finish(kind)
		return
	}
	p.token() // Consume identifier

	// Expect colon
	if !p.at(lexer.COLON) {
		p.errorExpected(lexer.COLON, "object field")
		p.finish(kind)
		return
	}
	p.token() // Consume ':'
	p.skipNewlines()

	// Parse value expression
	p.expression()

	p.finish(kind)
}

// decorator parses @identifier.property
// Only creates decorator node if identifier is registered
func (p *parser) decorator() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_decorator", "parsing decorator")
	}

	// Look ahead to check if this is a registered decorator
	// (we need to peek before consuming @ to decide if it's a decorator)
	atPos := p.pos
	p.advance() // Move past @

	// Check if next token is an identifier or VAR keyword
	if !p.at(lexer.IDENTIFIER) && !p.at(lexer.VAR) {
		// Not a decorator, treat @ as literal
		// TODO: This needs better handling for literal @ in strings
		return
	}

	// Build the decorator path by trying progressively longer dot-separated names
	//
	// Decorator syntax: @namespace.subnamespace.function.primaryParam
	//   - Namespace can be arbitrarily long (like a URI): @aws.secret.api_key
	//   - Primary param is dot syntax for the main parameter: @var.name where "name" is the primary param
	//   - We try progressively longer paths until we find a registered decorator
	//
	// Examples:
	//   @var.name        → try "var" (registered) → use "var", ".name" is primary param
	//   @file.read       → try "file" (not registered), try "file.read" (registered) → use "file.read"
	//   @aws.secret.key  → try "aws", "aws.secret", "aws.secret.key" until one is registered
	//   @file.read.path  → try "file", "file.read" (registered) → use "file.read", ".path" is primary param
	decoratorName := string(p.current().Text)
	tempPos := p.pos

	// Scan entire dotted sequence to find longest registered match
	var longestMatch string
	var longestMatchPos int
	currentName := decoratorName
	currentPos := tempPos

	// Check if first identifier is registered
	if types.Global().IsRegistered(currentName) || decorator.Global().IsRegistered(currentName) {
		longestMatch = currentName
		longestMatchPos = currentPos
	}

	// Continue scanning for longer matches
	for {
		p.advance() // Move to next token
		if !p.at(lexer.DOT) {
			// No more dots
			break
		}
		p.advance() // Move past dot
		if !p.at(lexer.IDENTIFIER) {
			// Dot not followed by identifier - stop here
			break
		}
		// Extend the candidate name
		currentName = currentName + "." + string(p.current().Text)
		currentPos = p.pos

		// Check if this longer name is registered
		if types.Global().IsRegistered(currentName) || decorator.Global().IsRegistered(currentName) {
			longestMatch = currentName
			longestMatchPos = currentPos
		}
	}

	// If no registered decorator found, reset position and treat @ as literal
	if longestMatch == "" {
		p.pos = tempPos
		return
	}

	// Use the longest registered match
	decoratorName = longestMatch
	p.pos = longestMatchPos

	// Get the schema for validation
	// Try new registry first, fall back to old registry for backward compatibility
	var schema types.DecoratorSchema
	var hasSchema bool

	entry, hasNewEntry := decorator.Global().Lookup(decoratorName)
	if hasNewEntry {
		// Extract schema from new registry
		desc := entry.Impl.Descriptor()
		schema = desc.Schema
		hasSchema = true
	} else {
		// Fall back to old registry
		schema, hasSchema = types.Global().GetSchema(decoratorName)
	}

	// It's a registered decorator, parse it
	// Reset position to @ and start the node
	p.pos = atPos
	kind := p.start(NodeDecorator)

	// Consume @ token (emit it)
	p.token()

	// Consume decorator name (may be dot-separated: file.read, aws.secret.api_key)
	// Count dots in decorator name to know how many tokens to consume
	dotCount := 0
	for _, ch := range decoratorName {
		if ch == '.' {
			dotCount++
		}
	}

	// Consume first identifier
	p.token()

	// Consume remaining dot + identifier pairs
	for i := 0; i < dotCount; i++ {
		p.token() // Consume DOT
		p.token() // Consume IDENTIFIER
	}

	// Track if primary parameter was provided via dot syntax
	hasPrimaryViaDot := false

	// Parse primary parameter via dot syntax (e.g., @var.name where "name" is the primary param)
	// This is AFTER the decorator name, so @file.read.property would have:
	//   - decorator: "file.read"
	//   - primary param: "property"
	if p.at(lexer.DOT) {
		p.token() // Consume DOT
		if p.at(lexer.IDENTIFIER) {
			p.token() // Consume property name
			hasPrimaryViaDot = true
		}
	}

	// Track provided parameters for validation
	providedParams := make(map[string]bool)
	if hasPrimaryViaDot && hasSchema && schema.PrimaryParameter != "" {
		providedParams[schema.PrimaryParameter] = true
	}

	// Parse parameters: (param1=value1, param2=value2)
	if p.at(lexer.LPAREN) {
		p.decoratorParamsWithValidation(decoratorName, schema, providedParams)
	}

	// Validate required parameters
	if hasSchema {
		p.validateRequiredParameters(decoratorName, schema, providedParams)
	}

	// Parse optional block (use new registry's Block capability)
	if hasNewEntry {
		desc := entry.Impl.Descriptor()
		blockReq := desc.Capabilities.Block

		// Default to BlockForbidden if not specified (safe default for value decorators)
		if blockReq == "" {
			blockReq = decorator.BlockForbidden
		}

		switch blockReq {
		case decorator.BlockRequired:
			// Block is required
			if !p.at(lexer.LBRACE) {
				p.errorWithDetails(
					fmt.Sprintf("@%s requires a block", decoratorName),
					"decorator block",
					fmt.Sprintf("Add a block: @%s(...) { ... }", decoratorName),
				)
			} else {
				p.block()
			}
		case decorator.BlockOptional:
			// Block is optional
			if p.at(lexer.LBRACE) {
				p.block()
			}
		case decorator.BlockForbidden:
			// Block is not allowed
			if p.at(lexer.LBRACE) {
				p.errorWithDetails(
					fmt.Sprintf("@%s cannot have a block", decoratorName),
					"decorator block",
					fmt.Sprintf("@%s is a value decorator and does not accept blocks", decoratorName),
				)
			}
		}
	} else if hasSchema {
		// Fall back to old schema-based validation for decorators not in new registry
		switch schema.BlockRequirement {
		case types.BlockRequired:
			// Block is required
			if !p.at(lexer.LBRACE) {
				p.errorWithDetails(
					fmt.Sprintf("@%s requires a block", decoratorName),
					"decorator block",
					fmt.Sprintf("Add a block: @%s(...) { ... }", decoratorName),
				)
			} else {
				p.block()
			}
		case types.BlockOptional:
			// Block is optional
			if p.at(lexer.LBRACE) {
				p.block()
			}
		case types.BlockForbidden:
			// Block is not allowed
			if p.at(lexer.LBRACE) {
				p.errorWithDetails(
					fmt.Sprintf("@%s cannot have a block", decoratorName),
					"decorator block",
					fmt.Sprintf("@%s is a %s decorator and does not accept blocks", decoratorName, schema.Kind),
				)
			}
		}
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_decorator", "decorator complete")
	}
}

// decoratorInExpressionContext parses a decorator in expression context (e.g., if condition).
// Unlike decorator(), this does NOT check for or consume blocks, because in expression
// context a following { is likely part of the enclosing statement (if/for/etc.), not a decorator block.
func (p *parser) decoratorInExpressionContext() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_decorator_expr", "parsing decorator in expression context")
	}

	// Look ahead to check if this is a registered decorator
	atPos := p.pos
	p.advance() // Move past @

	// Check if next token is an identifier or VAR keyword
	if !p.at(lexer.IDENTIFIER) && !p.at(lexer.VAR) {
		return
	}

	// Build the decorator path by trying progressively longer dot-separated names
	decoratorName := string(p.current().Text)
	tempPos := p.pos

	var longestMatch string
	var longestMatchPos int
	currentName := decoratorName
	currentPos := tempPos

	if types.Global().IsRegistered(currentName) || decorator.Global().IsRegistered(currentName) {
		longestMatch = currentName
		longestMatchPos = currentPos
	}

	for {
		p.advance()
		if !p.at(lexer.DOT) {
			break
		}
		p.advance()
		if !p.at(lexer.IDENTIFIER) {
			break
		}
		currentName = currentName + "." + string(p.current().Text)
		currentPos = p.pos

		if types.Global().IsRegistered(currentName) || decorator.Global().IsRegistered(currentName) {
			longestMatch = currentName
			longestMatchPos = currentPos
		}
	}

	if longestMatch == "" {
		p.pos = tempPos
		return
	}

	decoratorName = longestMatch
	p.pos = longestMatchPos

	// Reset position to @ and start the node
	p.pos = atPos
	kind := p.start(NodeDecorator)

	// Consume @ token
	p.token()

	// Count dots in decorator name
	dotCount := 0
	for _, ch := range decoratorName {
		if ch == '.' {
			dotCount++
		}
	}

	// Consume first identifier
	p.token()

	// Consume remaining dot + identifier pairs
	for i := 0; i < dotCount; i++ {
		p.token() // DOT
		p.token() // IDENTIFIER
	}

	// Get schema for validation (needed for primary parameter tracking)
	var schema types.DecoratorSchema
	var hasSchema bool
	entry, hasNewEntry := decorator.Global().Lookup(decoratorName)
	if hasNewEntry {
		desc := entry.Impl.Descriptor()
		schema = desc.Schema
		hasSchema = true
	} else {
		schema, hasSchema = types.Global().GetSchema(decoratorName)
	}

	// Track if primary parameter was provided via dot syntax
	hasPrimaryViaDot := false

	// Parse primary parameter via dot syntax
	if p.at(lexer.DOT) {
		p.token() // DOT
		if p.at(lexer.IDENTIFIER) {
			p.token() // property name
			hasPrimaryViaDot = true
		}
	}

	// Track provided parameters for validation
	providedParams := make(map[string]bool)
	if hasPrimaryViaDot && hasSchema && schema.PrimaryParameter != "" {
		providedParams[schema.PrimaryParameter] = true
	}

	// Parse parameters if present
	if p.at(lexer.LPAREN) {
		p.decoratorParamsWithValidation(decoratorName, schema, providedParams)
	}

	// NOTE: We intentionally do NOT check for blocks here.
	// In expression context, a following { is part of the enclosing statement.

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_decorator_expr", "decorator in expression context complete")
	}
}

// decoratorParamsWithValidation parses and validates decorator parameters
func (p *parser) decoratorParamsWithValidation(decoratorName string, schema types.DecoratorSchema, providedParams map[string]bool) {
	if !p.at(lexer.LPAREN) {
		return
	}

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_decorator_params", fmt.Sprintf("decorator=%s, schema_params=%d", decoratorName, len(schema.Parameters)))
	}

	paramListKind := p.start(NodeParamList)
	p.token() // Consume (
	p.skipNewlines()

	// Get positional binding order (required parameters shift left)
	orderedParams := schema.GetOrderedParameters()
	positionalParams := positionalBindingOrder(orderedParams)
	namedReservations := collectNamedReservations(p.tokens, p.pos, schema)
	filledPositions := make(map[int]bool)
	nextPositionIndex := 0

	// Parse parameters until we hit )
	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		p.skipNewlines()
		if p.at(lexer.RPAREN) || p.at(lexer.EOF) {
			break
		}

		paramKind := p.start(NodeParam)

		// Determine if this is a named or positional parameter
		isNamed := false
		var paramName string
		var paramSchema types.ParamSchema
		paramExists := false

		// Check if this is a named parameter (identifier followed by =)
		if p.at(lexer.IDENTIFIER) {
			// Look ahead to see if there's an EQUALS
			nextPos := p.pos + 1
			if nextPos < len(p.tokens) && p.tokens[nextPos].Type == lexer.EQUALS {
				// Named parameter: name=value
				isNamed = true
				paramNameToken := p.current()
				paramName = string(paramNameToken.Text)
				p.token() // Consume parameter name
				p.token() // Consume =

				// Check for duplicate parameter
				if providedParams[paramName] {
					p.errorWithDetails(
						fmt.Sprintf("duplicate parameter '%s'", paramName),
						"decorator parameter",
						"Each parameter can only be specified once",
					)
				}

				// Find this parameter's position in positionalParams
				for pos, param := range positionalParams {
					if param.Name == paramName {
						filledPositions[pos] = true
						break
					}
				}

				// Mark as provided immediately (before positional check)
				providedParams[paramName] = true

				// Check if parameter exists in schema
				paramSchema, paramExists = schema.Parameters[paramName]
				if !paramExists {
					// Check if it's a deprecated parameter name
					if schema.DeprecatedParameters != nil {
						if newName, isDeprecated := schema.DeprecatedParameters[paramName]; isDeprecated {
							oldParamName := paramName
							// Emit warning about deprecated parameter name
							p.warningWithDetails(
								fmt.Sprintf("parameter '%s' is deprecated for @%s", paramName, decoratorName),
								"decorator parameter",
								fmt.Sprintf("Use '%s' instead", newName),
							)
							// Map to new parameter name
							paramName = newName
							paramSchema, paramExists = schema.Parameters[paramName]
							// Update providedParams to use new name
							delete(providedParams, oldParamName) // Remove old name
							providedParams[newName] = true       // Add new name
						}
					}

					if !paramExists {
						// Unknown parameter
						p.errorWithDetails(
							fmt.Sprintf("unknown parameter '%s' for @%s", paramName, decoratorName),
							"decorator parameter",
							p.validParametersSuggestion(schema),
						)
					}
				}
			}
		}

		// If not named, treat as positional
		if !isNamed {
			// Find next unfilled position
			found := false
			for nextPositionIndex < len(positionalParams) {
				candidate := positionalParams[nextPositionIndex]
				if !filledPositions[nextPositionIndex] && !providedParams[candidate.Name] && !namedReservations[candidate.Name] {
					// Use this position
					paramSchema = candidate
					paramName = candidate.Name
					paramExists = true
					filledPositions[nextPositionIndex] = true
					nextPositionIndex++
					found = true
					break
				}
				nextPositionIndex++
			}

			if !found {
				p.errorWithDetails(
					"too many positional arguments",
					"decorator parameters",
					fmt.Sprintf("@%s accepts %d positional parameters", decoratorName, len(positionalParams)),
				)
				p.finish(paramKind)
				break
			}
		}

		// Check for duplicate parameter (only for positional, named already checked)
		if !isNamed {
			if providedParams[paramName] {
				p.errorWithDetails(
					fmt.Sprintf("duplicate parameter '%s'", paramName),
					"decorator parameter",
					"Each parameter can only be specified once",
				)
			} else {
				providedParams[paramName] = true
			}
		}

		// Parse and validate parameter value
		p.skipNewlines()
		valueToken := p.current()

		// Check if this is a simple literal or a complex expression (object/array)
		if p.at(lexer.LBRACE) || p.at(lexer.LSQUARE) {
			// Complex expression (object or array literal)
			// Track event position before parsing
			eventStartPos := len(p.events)
			p.expression()

			// Validate complex literal if parameter exists in schema
			if paramExists {
				p.validateComplexLiteral(paramName, paramSchema, eventStartPos)
			}
		} else if p.at(lexer.STRING) || p.at(lexer.INTEGER) || p.at(lexer.FLOAT) ||
			p.at(lexer.BOOLEAN) || p.at(lexer.DURATION) || p.at(lexer.IDENTIFIER) {

			// Validate type if parameter exists in schema
			if paramExists {
				p.validateParameterType(paramName, paramSchema, valueToken)
				// Validate constraints (min/max, pattern, format) for literal values
				p.validateParameterConstraints(paramName, paramSchema, valueToken)
			}

			p.token() // Consume value
		} else {
			p.errorUnexpected("parameter value")
			p.finish(paramKind)
			break
		}

		p.finish(paramKind)
		p.skipNewlines()

		// Check for comma (more parameters)
		if p.at(lexer.COMMA) {
			p.token() // Consume comma
			p.skipNewlines()
		} else if !p.at(lexer.RPAREN) {
			p.errorUnexpected("',' or ')'")
			break
		}
	}
	p.skipNewlines()

	if !p.at(lexer.RPAREN) {
		p.errorExpected(lexer.RPAREN, "')'")
		p.finish(paramListKind)
		return
	}
	p.token() // Consume )
	p.finish(paramListKind)
}

func positionalBindingOrder(params []types.ParamSchema) []types.ParamSchema {
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

func collectNamedReservations(tokens []lexer.Token, start int, schema types.DecoratorSchema) map[string]bool {
	reserved := make(map[string]bool)
	parenDepth := 0
	braceDepth := 0
	bracketDepth := 0
	atArgStart := true

	for i := start; i < len(tokens); i++ {
		tok := tokens[i]

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && tok.Type == lexer.RPAREN {
			break
		}

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
			switch tok.Type {
			case lexer.NEWLINE:
				continue
			case lexer.COMMA:
				atArgStart = true
				continue
			}

			if atArgStart && tok.Type == lexer.IDENTIFIER {
				if i+1 < len(tokens) && tokens[i+1].Type == lexer.EQUALS {
					paramName := string(tok.Text)
					if replacement, ok := schema.DeprecatedParameters[paramName]; ok {
						paramName = replacement
					}
					reserved[paramName] = true
				}
			}

			atArgStart = false
		}

		switch tok.Type {
		case lexer.LPAREN:
			parenDepth++
		case lexer.RPAREN:
			if parenDepth > 0 {
				parenDepth--
			}
		case lexer.LBRACE:
			braceDepth++
		case lexer.RBRACE:
			if braceDepth > 0 {
				braceDepth--
			}
		case lexer.LSQUARE:
			bracketDepth++
		case lexer.RSQUARE:
			if bracketDepth > 0 {
				bracketDepth--
			}
		}
	}

	return reserved
}

// validateParameterType checks if the token type matches the expected parameter type
func (p *parser) validateParameterType(paramName string, paramSchema types.ParamSchema, valueToken lexer.Token) {
	expectedType := paramSchema.Type
	actualType := p.tokenToParamType(valueToken.Type)

	if p.config.debug >= DebugDetailed {
		p.recordDebugEvent("validate_param_type",
			fmt.Sprintf("param=%s, expected=%s, actual=%s, match=%v",
				paramName, expectedType, actualType, actualType == expectedType))
	}

	// Special case: Enum parameters accept STRING tokens
	// The enum type (e.g., TypeScrubMode) is just a string with restricted values
	// Support both old (paramSchema.Enum) and new (paramSchema.EnumSchema) formats
	var enumValues []any
	var deprecatedValues map[string]string

	if len(paramSchema.Enum) > 0 {
		enumValues = paramSchema.Enum
	} else if paramSchema.EnumSchema != nil && len(paramSchema.EnumSchema.Values) > 0 {
		// Convert []string to []any for compatibility
		enumValues = make([]any, len(paramSchema.EnumSchema.Values))
		for i, v := range paramSchema.EnumSchema.Values {
			enumValues[i] = v
		}
		// Track deprecated values for warning messages
		deprecatedValues = paramSchema.EnumSchema.DeprecatedValues
	}

	if len(enumValues) > 0 && valueToken.Type == lexer.STRING {
		// Validate the string value is in the allowed enum values
		value := string(valueToken.Text)
		// Remove quotes from string literal
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		// Check if value is in current allowed values
		validValue := false
		for _, allowed := range enumValues {
			if value == allowed {
				validValue = true
				break
			}
		}

		// If not in current values, check if it's a deprecated value
		if !validValue && deprecatedValues != nil {
			if replacement, isDeprecated := deprecatedValues[value]; isDeprecated {
				// Accept deprecated value but emit a warning
				p.warningSchema(
					ErrorCodeSchemaEnumDeprecated,
					paramName,
					fmt.Sprintf("parameter '%s' uses deprecated value %q", paramName, value),
					fmt.Sprintf("Use %q instead", replacement),
				)
				validValue = true // Accept it, just warn
			}
		}

		if !validValue {
			// Format enum values for display
			enumStrs := make([]string, len(enumValues))
			for i, v := range enumValues {
				enumStrs[i] = fmt.Sprintf("%q", v)
			}
			p.errorSchema(
				ErrorCodeSchemaEnumInvalid,
				paramName,
				fmt.Sprintf("parameter '%s' has invalid value %q", paramName, value),
				fmt.Sprintf("Use one of: %s", strings.Join(enumStrs, ", ")),
				fmt.Sprintf("one of %v", enumValues),
				fmt.Sprintf("%q", value),
			)
		}
		return // Enum validation complete
	}

	if actualType != expectedType {
		// Build detailed expected type description
		expectedDesc := p.expectedTypeFromSchema(paramSchema)

		p.errorSchema(
			ErrorCodeSchemaTypeMismatch,
			paramName,
			fmt.Sprintf("parameter '%s' expects %s, got %s", paramName, expectedDesc, actualType),
			p.generateConcreteSuggestion(paramName, paramSchema, ""),
			expectedDesc,
			string(actualType),
		)
	}
}

// tokenToParamType converts a lexer token type to a ParamType
func (p *parser) tokenToParamType(tokType lexer.TokenType) types.ParamType {
	switch tokType {
	case lexer.STRING:
		return types.TypeString
	case lexer.INTEGER:
		return types.TypeInt
	case lexer.FLOAT:
		return types.TypeFloat
	case lexer.BOOLEAN:
		return types.TypeBool
	case lexer.DURATION:
		return types.TypeDuration
	case lexer.IDENTIFIER:
		// Identifiers could be variable references, for now treat as string
		return types.TypeString
	default:
		return types.TypeString
	}
}

// validateParameterConstraints validates parameter constraints (min/max, pattern, format)
// for literal values. Skips validation for variables and expressions.
func (p *parser) validateParameterConstraints(paramName string, paramSchema types.ParamSchema, valueToken lexer.Token) {
	// Only validate literal values (skip variables, expressions)
	if valueToken.Type == lexer.IDENTIFIER || valueToken.Type == lexer.AT {
		return // Skip validation for variables/expressions
	}

	// Extract literal value from token
	value, ok := p.extractLiteralValue(valueToken)
	if !ok {
		return // Not a literal, skip validation
	}

	// Use core/types validator for constraint validation
	validator := types.NewValidator(types.DefaultValidationConfig())

	if err := validator.ValidateParams(&paramSchema, value); err != nil {
		// Convert validation error to parser error with rich context
		code := p.errorCodeFromValidationError(err, paramSchema)
		p.errorSchema(
			code,
			paramName,
			fmt.Sprintf("invalid value for parameter '%s'", paramName),
			p.suggestionFromSchema(paramSchema, err),
			p.expectedTypeFromSchema(paramSchema),
			fmt.Sprintf("%v", value),
		)
	}
}

// validateComplexLiteral validates object and array literals against schema
func (p *parser) validateComplexLiteral(paramName string, paramSchema types.ParamSchema, eventStartPos int) {
	// Extract value from events
	value, ok := p.extractComplexValue(eventStartPos)
	if !ok {
		// Not a literal (contains variables/expressions), skip validation
		return
	}

	// Validate using core/types validator
	validator := types.NewValidator(&types.ValidationConfig{
		MaxSchemaSize:  1024 * 1024, // 1MB
		MaxSchemaDepth: 10,
		AllowRemoteRef: false,
		EnableCache:    true,
		MaxCacheSize:   100,
	})

	// Validate the value against the schema
	if err := validator.ValidateParams(&paramSchema, value); err != nil {
		code := p.errorCodeFromValidationError(err, paramSchema)
		p.errorSchema(
			code,
			paramName,
			fmt.Sprintf("invalid value for parameter '%s'", paramName),
			p.suggestionFromSchema(paramSchema, err),
			p.expectedTypeFromSchema(paramSchema),
			fmt.Sprintf("%v", value),
		)
	}
}

// extractComplexValue extracts a Go value from object/array literal events
// Returns (value, true) for pure literals, (nil, false) if it contains variables/expressions
func (p *parser) extractComplexValue(eventStartPos int) (any, bool) {
	if eventStartPos >= len(p.events) {
		return nil, false
	}

	// Start from the first event after eventStartPos
	evt := p.events[eventStartPos]
	if evt.Kind != EventOpen {
		return nil, false
	}

	nodeKind := NodeKind(evt.Data)
	switch nodeKind {
	case NodeObjectLiteral:
		return p.extractObjectLiteral(eventStartPos)
	case NodeArrayLiteral:
		return p.extractArrayLiteral(eventStartPos)
	default:
		return nil, false
	}
}

// extractObjectLiteral extracts a map[string]any from object literal events
func (p *parser) extractObjectLiteral(startPos int) (any, bool) {
	result := make(map[string]any)
	pos := startPos + 1 // Skip NodeObjectLiteral open event
	depth := 1

	for pos < len(p.events) && depth > 0 {
		evt := p.events[pos]

		if evt.Kind == EventOpen {
			nodeKind := NodeKind(evt.Data)

			if nodeKind == NodeObjectField {
				// Extract field name and value
				fieldName, fieldValue, fieldOk, newPos := p.extractObjectField(pos)
				if !fieldOk {
					return nil, false // Contains non-literal
				}
				result[fieldName] = fieldValue
				pos = newPos
				continue
			} else if nodeKind == NodeObjectLiteral {
				depth++
			}
		} else if evt.Kind == EventClose {
			nodeKind := NodeKind(evt.Data)
			if nodeKind == NodeObjectLiteral {
				depth--
				if depth == 0 {
					return result, true
				}
			}
		}

		pos++
	}

	return result, true
}

// extractObjectField extracts a single object field (name and value)
// Returns (name, value, ok, nextPos)
func (p *parser) extractObjectField(startPos int) (string, any, bool, int) {
	pos := startPos + 1 // Skip NodeObjectField open event

	// Next should be the field name token
	if pos >= len(p.events) {
		return "", nil, false, pos
	}

	nameEvt := p.events[pos]
	if nameEvt.Kind != EventToken {
		return "", nil, false, pos
	}

	nameToken := p.tokens[nameEvt.Data]
	fieldName := string(nameToken.Text)
	pos++

	// Skip colon token
	if pos < len(p.events) && p.events[pos].Kind == EventToken {
		pos++
	}

	// Extract value
	if pos >= len(p.events) {
		return "", nil, false, pos
	}

	valueEvt := p.events[pos]
	var fieldValue any
	var ok bool

	switch valueEvt.Kind {
	case EventToken:
		// Simple literal value
		valueToken := p.tokens[valueEvt.Data]
		fieldValue, ok = p.extractLiteralValue(valueToken)
		if !ok {
			return "", nil, false, pos
		}
		pos++
	case EventOpen:
		// Nested object or array
		nodeKind := NodeKind(valueEvt.Data)
		switch nodeKind {
		case NodeObjectLiteral:
			fieldValue, ok = p.extractObjectLiteral(pos)
		case NodeArrayLiteral:
			fieldValue, ok = p.extractArrayLiteral(pos)
		default:
			return "", nil, false, pos
		}

		if !ok {
			return "", nil, false, pos
		}

		// Skip to close event
		depth := 1
		pos++
		for pos < len(p.events) && depth > 0 {
			evt := p.events[pos]
			switch evt.Kind {
			case EventOpen:
				depth++
			case EventClose:
				depth--
			}
			pos++
		}
	}

	// Skip to NodeObjectField close event
	for pos < len(p.events) {
		evt := p.events[pos]
		if evt.Kind == EventClose && NodeKind(evt.Data) == NodeObjectField {
			pos++
			break
		}
		pos++
	}

	return fieldName, fieldValue, true, pos
}

// extractArrayLiteral extracts a []any from array literal events
func (p *parser) extractArrayLiteral(startPos int) (any, bool) {
	result := make([]any, 0)
	pos := startPos + 1 // Skip NodeArrayLiteral open event
	depth := 1

	for pos < len(p.events) && depth > 0 {
		evt := p.events[pos]

		if evt.Kind == EventOpen {
			nodeKind := NodeKind(evt.Data)

			if nodeKind == NodeArrayLiteral {
				depth++
			} else if nodeKind == NodeObjectLiteral {
				// Nested object
				objValue, ok := p.extractObjectLiteral(pos)
				if !ok {
					return nil, false
				}
				result = append(result, objValue)

				// Skip to close event
				nestedDepth := 1
				pos++
				for pos < len(p.events) && nestedDepth > 0 {
					e := p.events[pos]
					switch e.Kind {
					case EventOpen:
						nestedDepth++
					case EventClose:
						nestedDepth--
					}
					pos++
				}
				continue
			}
		} else if evt.Kind == EventClose {
			nodeKind := NodeKind(evt.Data)
			if nodeKind == NodeArrayLiteral {
				depth--
				if depth == 0 {
					return result, true
				}
			}
		} else if evt.Kind == EventToken && depth == 1 {
			// Element token (not inside nested structure)
			token := p.tokens[evt.Data]
			value, ok := p.extractLiteralValue(token)
			if !ok {
				return nil, false
			}
			result = append(result, value)
		}

		pos++
	}

	return result, true
}

// extractLiteralValue extracts a Go value from a literal token
// Returns (value, true) for literals, (nil, false) for non-literals
func (p *parser) extractLiteralValue(tok lexer.Token) (any, bool) {
	switch tok.Type {
	case lexer.STRING:
		// Remove quotes from string literal
		value := string(tok.Text)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		return value, true
	case lexer.INTEGER:
		// Parse integer
		var val int64
		_, _ = fmt.Sscanf(string(tok.Text), "%d", &val)
		return int(val), true
	case lexer.FLOAT:
		// Parse float
		var val float64
		_, _ = fmt.Sscanf(string(tok.Text), "%f", &val)
		return val, true
	case lexer.BOOLEAN:
		// Parse boolean
		value := string(tok.Text)
		return value == "true", true
	case lexer.DURATION:
		// Duration is validated as a string by the validator
		return string(tok.Text), true
	default:
		return nil, false
	}
}

// suggestionFromSchema generates a helpful suggestion based on schema constraints
func (p *parser) suggestionFromSchema(schema types.ParamSchema, err error) string {
	switch schema.Type {
	case types.TypeInt:
		if schema.Minimum != nil && schema.Maximum != nil {
			return fmt.Sprintf("Use an integer between %v and %v", *schema.Minimum, *schema.Maximum)
		} else if schema.Minimum != nil {
			return fmt.Sprintf("Use an integer >= %v", *schema.Minimum)
		} else if schema.Maximum != nil {
			return fmt.Sprintf("Use an integer <= %v", *schema.Maximum)
		}
		return "Use a valid integer"
	case types.TypeFloat:
		if schema.Minimum != nil && schema.Maximum != nil {
			return fmt.Sprintf("Use a number between %v and %v", *schema.Minimum, *schema.Maximum)
		}
		return "Use a valid number"
	case types.TypeString:
		if schema.Pattern != nil {
			return fmt.Sprintf("Use a string matching pattern: %s", *schema.Pattern)
		}
		if schema.Format != nil {
			return fmt.Sprintf("Use a valid %s", *schema.Format)
		}
		return "Use a valid string"
	case types.TypeDuration:
		return "Use a duration like \"5m\", \"1h\", or \"30s\""
	case types.TypeEnum:
		if schema.EnumSchema != nil && len(schema.EnumSchema.Values) > 0 {
			return fmt.Sprintf("Use one of: %v", schema.EnumSchema.Values)
		}
		return "Use a valid enum value"
	default:
		return err.Error()
	}
}

// errorCodeFromValidationError determines the appropriate error code from a validation error
func (p *parser) errorCodeFromValidationError(err error, schema types.ParamSchema) ErrorCode {
	errMsg := err.Error()

	// Check error message patterns to determine error code
	if strings.Contains(errMsg, "minimum") || strings.Contains(errMsg, "maximum") {
		return ErrorCodeSchemaRangeViolation
	}
	if strings.Contains(errMsg, "pattern") {
		return ErrorCodeSchemaPatternMismatch
	}
	if strings.Contains(errMsg, "format") {
		return ErrorCodeSchemaFormatInvalid
	}
	if strings.Contains(errMsg, "minLength") || strings.Contains(errMsg, "maxLength") {
		return ErrorCodeSchemaLengthViolation
	}
	if strings.Contains(errMsg, "additionalProperties") {
		return ErrorCodeSchemaAdditionalProp
	}
	if strings.Contains(errMsg, "type") {
		if schema.Type == types.TypeObject {
			return ErrorCodeSchemaObjectFieldType
		}
		if schema.Type == types.TypeArray {
			return ErrorCodeSchemaArrayElementType
		}
		return ErrorCodeSchemaTypeMismatch
	}

	// Default to range violation for numeric constraints
	if schema.Type == types.TypeInt || schema.Type == types.TypeFloat {
		return ErrorCodeSchemaRangeViolation
	}

	return ErrorCodeSchemaTypeMismatch
}

// expectedTypeFromSchema generates a human-readable expected type description
func (p *parser) expectedTypeFromSchema(schema types.ParamSchema) string {
	switch schema.Type {
	case types.TypeInt:
		if schema.Minimum != nil && schema.Maximum != nil {
			return fmt.Sprintf("integer between %v and %v", *schema.Minimum, *schema.Maximum)
		} else if schema.Minimum != nil {
			return fmt.Sprintf("integer >= %v", *schema.Minimum)
		} else if schema.Maximum != nil {
			return fmt.Sprintf("integer <= %v", *schema.Maximum)
		}
		return "integer"
	case types.TypeFloat:
		if schema.Minimum != nil && schema.Maximum != nil {
			return fmt.Sprintf("number between %v and %v", *schema.Minimum, *schema.Maximum)
		}
		return "number"
	case types.TypeString:
		if schema.Pattern != nil {
			return fmt.Sprintf("string matching /%s/", *schema.Pattern)
		}
		if schema.Format != nil {
			return fmt.Sprintf("%s format", *schema.Format)
		}
		if schema.MinLength != nil && schema.MaxLength != nil {
			return fmt.Sprintf("string (length %d-%d)", *schema.MinLength, *schema.MaxLength)
		}
		return "string"
	case types.TypeBool:
		return "boolean"
	case types.TypeDuration:
		return "duration (e.g., \"5m\", \"1h\")"
	case types.TypeEnum:
		if schema.EnumSchema != nil && len(schema.EnumSchema.Values) > 0 {
			return fmt.Sprintf("one of %v", schema.EnumSchema.Values)
		}
		return "enum value"
	case types.TypeObject:
		return "object"
	case types.TypeArray:
		return "array"
	default:
		return string(schema.Type)
	}
}

// generateConcreteSuggestion generates a realistic example based on schema constraints
func (p *parser) generateConcreteSuggestion(paramName string, schema types.ParamSchema, decoratorName string) string {
	const placeholderValue = "\"value\""
	var exampleValue string

	switch schema.Type {
	case types.TypeInt:
		// Use midpoint of range if available, not arbitrary "42"
		if schema.Minimum != nil && schema.Maximum != nil {
			midpoint := (*schema.Minimum + *schema.Maximum) / 2
			exampleValue = fmt.Sprintf("%d", int(midpoint))
		} else if schema.Minimum != nil {
			// Use minimum + 1 for a realistic example
			exampleValue = fmt.Sprintf("%d", int(*schema.Minimum)+1)
		} else if schema.Maximum != nil {
			// Use maximum - 1 for a realistic example
			exampleValue = fmt.Sprintf("%d", int(*schema.Maximum)-1)
		} else {
			exampleValue = "1"
		}

	case types.TypeFloat:
		if schema.Minimum != nil && schema.Maximum != nil {
			midpoint := (*schema.Minimum + *schema.Maximum) / 2
			exampleValue = fmt.Sprintf("%.1f", midpoint)
		} else {
			exampleValue = "1.0"
		}

	case types.TypeString:
		if schema.Format != nil {
			// Use format-specific examples
			switch *schema.Format {
			case "uri":
				exampleValue = "\"https://example.com\""
			case "hostname":
				exampleValue = "\"example.com\""
			case "ipv4":
				exampleValue = "\"192.0.2.1\""
			case "cidr":
				exampleValue = "\"10.0.0.0/8\""
			case "semver":
				exampleValue = "\"1.0.0\""
			case "duration":
				exampleValue = "\"5m\""
			default:
				exampleValue = placeholderValue
			}
		} else if len(schema.Examples) > 0 && schema.Examples[0] != "" {
			// Use first non-empty example
			exampleValue = fmt.Sprintf("%q", schema.Examples[0])
		} else {
			exampleValue = placeholderValue
		}

	case types.TypeBool:
		exampleValue = "true"

	case types.TypeDuration:
		exampleValue = "\"5m\""

	case types.TypeEnum:
		// Use first valid enum value
		if schema.EnumSchema != nil && len(schema.EnumSchema.Values) > 0 {
			exampleValue = fmt.Sprintf("%q", schema.EnumSchema.Values[0])
		} else {
			exampleValue = placeholderValue
		}

	case types.TypeObject:
		exampleValue = "{...}"

	case types.TypeArray:
		exampleValue = "[...]"

	default:
		exampleValue = "value"
	}

	// Format suggestion based on context
	if decoratorName != "" {
		// Full decorator syntax for complete examples
		return fmt.Sprintf("Use @%s(%s=%s) { ... }", decoratorName, paramName, exampleValue)
	}

	// Simple format for type mismatch errors (matches existing test expectations)
	typeDesc := string(schema.Type)
	if schema.Type == types.TypeEnum && schema.EnumSchema != nil && len(schema.EnumSchema.Values) > 0 {
		return fmt.Sprintf("Use one of: %s", strings.Join(func() []string {
			quoted := make([]string, len(schema.EnumSchema.Values))
			for i, v := range schema.EnumSchema.Values {
				quoted[i] = fmt.Sprintf("%q", v)
			}
			return quoted
		}(), ", "))
	}

	// Use correct article (a/an) based on type
	article := "a"
	if typeDesc == "integer" || typeDesc == "array" || typeDesc == "object" {
		article = "an"
	}
	return fmt.Sprintf("Use %s %s value like %s", article, typeDesc, exampleValue)
}

// validateRequiredParameters checks that all required parameters were provided
func (p *parser) validateRequiredParameters(decoratorName string, schema types.DecoratorSchema, providedParams map[string]bool) {
	for paramName, paramSchema := range schema.Parameters {
		if paramSchema.Required && !providedParams[paramName] {
			suggestion := fmt.Sprintf("Provide %s parameter", paramName)
			if paramName == schema.PrimaryParameter {
				// Use first example from schema if available, otherwise generic
				exampleValue := "VALUE"
				if len(paramSchema.Examples) > 0 && paramSchema.Examples[0] != "" {
					exampleValue = paramSchema.Examples[0]
				}
				suggestion = fmt.Sprintf("Use dot syntax like @%s.%s or provide %s=\"%s\"", decoratorName, exampleValue, paramName, exampleValue)
			}

			p.errorSchema(
				ErrorCodeSchemaRequiredMissing,
				paramName,
				fmt.Sprintf("missing required parameter '%s'", paramName),
				suggestion,
				p.expectedTypeFromSchema(paramSchema),
				"(not provided)",
			)
		}
	}
}

// validParametersSuggestion returns a suggestion listing valid parameters
func (p *parser) validParametersSuggestion(schema types.DecoratorSchema) string {
	if len(schema.Parameters) == 0 {
		return "This decorator accepts no parameters"
	}

	params := make([]string, 0, len(schema.Parameters))
	for name := range schema.Parameters {
		params = append(params, name)
	}

	// Simple alphabetical sort
	for i := 0; i < len(params); i++ {
		for j := i + 1; j < len(params); j++ {
			if params[i] > params[j] {
				params[i], params[j] = params[j], params[i]
			}
		}
	}

	result := "Valid parameters: "
	for i, param := range params {
		if i > 0 {
			result += ", "
		}
		result += param
	}
	return result
}

// errorWithDetails creates a parse error with full context
func (p *parser) errorWithDetails(message, context, suggestion string) {
	tok := p.current()
	p.errors = append(p.errors, ParseError{
		Position:   tok.Position,
		Message:    message,
		Context:    context,
		Got:        tok.Type,
		Suggestion: suggestion,
	})
}

// warningWithDetails adds a warning (non-fatal) to the parse tree
func (p *parser) warningWithDetails(message, context, suggestion string) {
	tok := p.current()
	p.warnings = append(p.warnings, ParseWarning{
		Position:   tok.Position,
		Message:    message,
		Context:    context,
		Suggestion: suggestion,
	})
}

// errorSchema adds a schema validation error with structured error code
func (p *parser) errorSchema(code ErrorCode, paramName, message, suggestion, expectedType, gotValue string) {
	tok := p.current()
	p.errors = append(p.errors, ParseError{
		Position:     tok.Position,
		Message:      message,
		Context:      "decorator parameter",
		Suggestion:   suggestion,
		Code:         code,
		Path:         paramName,
		ExpectedType: expectedType,
		GotValue:     gotValue,
	})
}

// warningSchema adds a schema validation warning with structured error code
func (p *parser) warningSchema(code ErrorCode, paramName, message, suggestion string) {
	tok := p.current()
	p.warnings = append(p.warnings, ParseWarning{
		Position:   tok.Position,
		Message:    message,
		Context:    "decorator parameter",
		Suggestion: suggestion,
		Note:       fmt.Sprintf("Code: %s, Parameter: %s", code, paramName),
	})
}

// stringNeedsInterpolation checks if the current STRING token needs interpolation
func (p *parser) stringNeedsInterpolation() bool {
	tok := p.current()

	if len(tok.Text) == 0 {
		return false
	}

	quoteType := tok.Text[0]

	// Single quotes never interpolate
	if quoteType == '\'' {
		return false
	}

	// Extract content without quotes
	content := tok.Text
	if len(content) < 2 {
		return false
	}
	content = content[1 : len(content)-1]

	// Tokenize and check if there are multiple parts or decorator parts
	parts := TokenizeString(content, quoteType)

	// Needs interpolation if there are multiple parts or if the single part is a decorator
	return len(parts) > 1 || (len(parts) == 1 && !parts[0].IsLiteral)
}

// stringLiteral parses a string literal, checking for interpolation
func (p *parser) stringLiteral() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_string_literal", "parsing string")
	}

	tok := p.current()

	// Check quote type - single quotes have no interpolation
	if len(tok.Text) == 0 {
		// Empty string token, treat as simple literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	quoteType := tok.Text[0]

	// Single quotes never interpolate
	if quoteType == '\'' {
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	// Extract content without quotes
	content := tok.Text
	if len(content) < 2 {
		// Malformed string, treat as simple literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}
	content = content[1 : len(content)-1] // Remove surrounding quotes

	// Tokenize the string content
	parts := TokenizeString(content, quoteType)

	// If no parts or only one literal part, treat as simple literal
	if len(parts) == 0 || (len(parts) == 1 && parts[0].IsLiteral) {
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	// Has interpolation - create interpolated string node
	kind := p.start(NodeInterpolatedString)
	p.token() // Consume the STRING token

	// Create nodes for each part
	for _, part := range parts {
		partKind := p.start(NodeStringPart)

		if part.IsLiteral {
			// Literal part - no additional nodes needed
			// The part's byte offsets are stored in the StringPart
		} else {
			// Decorator part - create decorator node
			decoratorKind := p.start(NodeDecorator)
			// Note: We don't consume tokens here because the decorator is embedded in the string
			// The decorator name and property are in the string content at part.Start:part.End
			p.finish(decoratorKind)
		}

		p.finish(partKind)
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_string_literal", "string complete")
	}
}

// precedence returns the precedence of the current token as a binary operator
func (p *parser) precedence() int {
	switch p.current().Type {
	case lexer.OR_OR:
		return 1
	case lexer.AND_AND:
		return 2
	case lexer.EQ_EQ, lexer.NOT_EQ:
		return 3
	case lexer.LT, lexer.LT_EQ, lexer.GT, lexer.GT_EQ:
		return 4
	case lexer.PLUS, lexer.MINUS:
		return 5
	case lexer.MULTIPLY, lexer.DIVIDE, lexer.MODULO:
		return 6
	default:
		return 0 // Not a binary operator
	}
}

// at checks if current token is of given type
func (p *parser) at(typ lexer.TokenType) bool {
	return p.current().Type == typ
}

// current returns the current token
func (p *parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		// Return EOF token if we're past the end
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token
func (p *parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// skipNewlines advances past consecutive newline separator tokens.
func (p *parser) skipNewlines() {
	for p.at(lexer.NEWLINE) {
		p.advance()
	}
}

// start emits an Open event with the given node kind and returns it for matching close
func (p *parser) start(kind NodeKind) NodeKind {
	p.events = append(p.events, Event{
		Kind: EventOpen,
		Data: uint32(kind),
	})
	return kind
}

// finish emits a Close event with the given node kind
func (p *parser) finish(kind NodeKind) {
	p.events = append(p.events, Event{
		Kind: EventClose,
		Data: uint32(kind),
	})
}

// token emits a Token event and advances
func (p *parser) token() {
	p.events = append(p.events, Event{
		Kind: EventToken,
		Data: uint32(p.pos),
	})
	p.advance()
}

// expect checks for expected token and reports error if not found
func (p *parser) expect(expected lexer.TokenType, context string) bool {
	if p.at(expected) {
		p.token()
		return true
	}
	p.errorExpected(expected, context)
	return false
}

// errorExpected reports an error for missing expected token
func (p *parser) errorExpected(expected lexer.TokenType, context string) {
	current := p.current()

	err := ParseError{
		Position: current.Position,
		Message:  "missing " + tokenName(expected),
		Context:  context,
		Expected: []lexer.TokenType{expected},
		Got:      current.Type,
	}

	// Add helpful suggestions based on context
	switch expected {
	case lexer.RPAREN:
		err.Suggestion = "Add ')' to close the " + context
		err.Example = "fun greet(name String) {}"
	case lexer.RBRACE:
		err.Suggestion = "Add '}' to close the " + context
		err.Example = "fun greet() { echo \"hello\" }"
	case lexer.LBRACE:
		err.Suggestion = "Add '{' to start the function body"
		err.Example = "fun greet() {}"
	case lexer.IDENTIFIER:
		switch context {
		case "function declaration":
			err.Suggestion = "Add a function name after 'fun'"
			err.Example = "fun greet() {}"
		case "parameter":
			err.Suggestion = "Add a parameter name"
			err.Example = "fun greet(name String) {}"
		}
	}

	p.errors = append(p.errors, err)
}

// errorUnexpected reports an error for unexpected token
func (p *parser) errorUnexpected(context string) {
	current := p.current()

	err := ParseError{
		Position: current.Position,
		Message:  "unexpected " + tokenName(current.Type),
		Context:  context,
		Got:      current.Type,
	}

	p.errors = append(p.errors, err)
}

// isSyncToken checks if current token is a synchronization point
func (p *parser) isSyncToken() bool {
	switch p.current().Type {
	case lexer.RBRACE, // End of block
		lexer.SEMICOLON, // Statement terminator
		lexer.FUN,       // Start of new function
		lexer.EOF:       // End of file
		return true
	}

	// Newline can be a sync point in some contexts
	// For now, we'll rely on explicit tokens
	return false
}

// recover skips tokens until we reach a synchronization point
// This allows the parser to continue after errors and report multiple issues
func (p *parser) recover() {
	for !p.isSyncToken() && !p.at(lexer.EOF) {
		p.advance()
	}
}
