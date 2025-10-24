package parser

import (
	"github.com/aledsdavies/opal/core/types"
	"github.com/aledsdavies/opal/runtime/lexer"
)

// ParseTree represents the result of parsing
type ParseTree struct {
	Source      []byte          // Original source (for reference)
	Tokens      []lexer.Token   // Tokens from lexer
	Events      []Event         // Parse events
	Errors      []ParseError    // Parse errors
	Telemetry   *ParseTelemetry // Performance metrics (nil if disabled)
	DebugEvents []DebugEvent    // Debug events (nil if disabled)
}

// Event represents a parse tree construction event
type Event struct {
	Kind EventKind
	Data uint32
}

// EventKind represents the type of parse event
type EventKind uint8

const (
	EventOpen      EventKind = iota // Open syntax node
	EventClose                      // Close syntax node
	EventToken                      // Consume token
	EventStepEnter                  // Enter a step (newline-separated executable statement)
	EventStepExit                   // Exit a step
)

// NodeKind represents syntax node types
//
// IMPORTANT: When adding new node types, ALWAYS add them at the END of the enum
// (after the last node type, before the closing parenthesis). Adding nodes in the
// middle will shift all subsequent node numbers and break existing tests.
type NodeKind uint32

const (
	NodeSource NodeKind = iota // Top-level source (file, stdin, string)
	NodeFunction
	NodeParamList
	NodeBlock
	NodeParam          // Function parameter
	NodeTypeAnnotation // Type annotation (: Type)
	NodeDefaultValue   // Default value (= expression)

	// Statements
	NodeVarDecl      // Variable declaration
	NodeShellCommand // Shell command (converts to @shell decorator internally)
	NodeShellArg     // Shell command argument (may contain multiple tokens)
	NodeIf           // If statement: if condition { ... }
	NodeElse         // Else clause: else { ... } or else if { ... }
	NodeFor          // For loop: for item in collection { ... }

	// Expressions
	NodeLiteral            // Literal value (int, string, bool, duration)
	NodeIdentifier         // Identifier reference
	NodeBinaryExpr         // Binary expression (a + b, a == b, etc.)
	NodeInterpolatedString // String with interpolated decorators: "Hello @var.name"
	NodeStringPart         // Part of interpolated string (literal or decorator reference)

	// Decorators
	NodeDecorator // Decorator with property access: @var.name, @env.HOME

	// Error handling - added at end to preserve existing node numbers
	NodeTry     // Try block: try { ... }
	NodeCatch   // Catch block: catch { ... }
	NodeFinally // Finally block: finally { ... }

	// Pattern matching - added at end to preserve existing node numbers
	NodeWhen           // When statement: when expr { pattern -> block ... }
	NodeWhenArm        // When arm: pattern -> block
	NodePatternLiteral // String literal pattern: "production"
	NodePatternElse    // Else pattern (catch-all)
	NodePatternRegex   // Regex pattern: r"^pattern$"
	NodePatternRange   // Numeric range pattern: 200...299

	// For loop ranges - added at end to preserve existing node numbers
	NodeRange // Range expression in for loops: 1...10, @var.start...@var.end

	// OR patterns - added at end to preserve existing node numbers
	NodePatternOr // OR pattern: "a" | "b" | "c"

	// Unary expressions - added at end to preserve existing node numbers
	NodeUnaryExpr // Unary expression: !expr, -expr

	// Increment/decrement expressions - added at end to preserve existing node numbers
	NodePrefixExpr  // Prefix increment/decrement: ++expr, --expr
	NodePostfixExpr // Postfix increment/decrement: expr++, expr--

	// Assignment statements - added at end to preserve existing node numbers
	NodeAssignment // Assignment: x += 5, total -= cost

	// Output redirection - added at end to preserve existing node numbers
	NodeRedirect       // Redirect operator: > or >>
	NodeRedirectTarget // Redirect target (path, variable, or decorator)
)

// ParseError represents a parse error with rich context for user-friendly messages
type ParseError struct {
	// Location
	Filename string         // Source filename (empty for stdin/string)
	Position lexer.Position // Line, column, offset

	// Core error info
	Message string // Clear, specific: "missing closing parenthesis"
	Context string // What we were parsing: "parameter list"

	// What went wrong
	Expected []lexer.TokenType // What tokens would be valid
	Got      lexer.TokenType   // What we found instead

	// How to fix it (educational)
	Suggestion string // Actionable fix: "Add ')' after the last parameter"
	Example    string // Valid syntax: "fun greet(name) {}"
	Note       string // Optional explanation for learning
}

// ValidateSemantics performs post-parse semantic validation
// This includes checking pipe operator I/O compatibility and other semantic rules
func (tree *ParseTree) ValidateSemantics() {
	v := &semanticValidator{
		tree:   tree,
		tokens: tree.Tokens,
		events: tree.Events,
	}
	v.validate()
}

type semanticValidator struct {
	tree   *ParseTree
	tokens []lexer.Token
	events []Event
	errors []ParseError
}

func (v *semanticValidator) validate() {
	v.validatePipeOperators()
	v.validateRedirectOperators()
	v.tree.Errors = append(v.tree.Errors, v.errors...)
}

func (v *semanticValidator) validatePipeOperators() {
	// Walk through tokens to find pipe operators
	// For each pipe, check what's on the left and right
	for i := 0; i < len(v.tokens); i++ {
		if v.tokens[i].Type != lexer.PIPE {
			continue
		}

		// Found a pipe operator at position i
		pipeToken := v.tokens[i]

		// Look backward to find what produces output
		leftDecorator := v.findDecoratorBefore(i)

		// Look forward to find what consumes input
		rightDecorator := v.findDecoratorAfter(i)

		// Validate left side (must support stdout)
		if leftDecorator != "" {
			v.validateStdoutSupport(leftDecorator, pipeToken)
		}

		// Validate right side (must support stdin)
		if rightDecorator != "" {
			v.validateStdinSupport(rightDecorator, pipeToken)
		}
	}
}

func (v *semanticValidator) findDecoratorBefore(pipePos int) string {
	// Walk backward from pipe position to find @ token
	// If we see }, we need to find the decorator that owns that block
	for i := pipePos - 1; i >= 0; i-- {
		tok := v.tokens[i]

		// Skip whitespace, newlines
		if tok.Type == lexer.NEWLINE {
			continue
		}

		// If we hit }, find the matching decorator
		if tok.Type == lexer.RBRACE {
			// Find matching { and then the decorator before it
			return v.findDecoratorBeforeBlock(i)
		}

		// If we hit @, this is a decorator without a block
		if tok.Type == lexer.AT {
			// Next token should be the decorator name
			if i+1 < len(v.tokens) && (v.tokens[i+1].Type == lexer.IDENTIFIER || v.tokens[i+1].Type == lexer.VAR) {
				return v.extractDecoratorName(i + 1)
			}
		}

		// If we hit something else, it's plain shell (no validation needed)
		break
	}
	return ""
}

func (v *semanticValidator) findDecoratorBeforeBlock(rbracePos int) string {
	// Find matching { by counting braces backward
	braceDepth := 1
	for i := rbracePos - 1; i >= 0; i-- {
		switch v.tokens[i].Type {
		case lexer.RBRACE:
			braceDepth++
		case lexer.LBRACE:
			braceDepth--
			if braceDepth == 0 {
				// Found matching {, now look backward for @
				return v.findDecoratorBeforeLBrace(i)
			}
		}
	}
	return ""
}

func (v *semanticValidator) findDecoratorBeforeLBrace(lbracePos int) string {
	// Walk backward from { to find @
	// Skip ) and parameter lists
	for i := lbracePos - 1; i >= 0; i-- {
		tok := v.tokens[i]

		// Skip whitespace, newlines
		if tok.Type == lexer.NEWLINE {
			continue
		}

		// Skip ) - could be decorator parameters
		if tok.Type == lexer.RPAREN {
			// Skip the entire parameter list
			i = v.skipParameterListBackward(i)
			if i < 0 {
				break
			}
			continue
		}

		// Skip identifiers and dots (decorator name)
		if tok.Type == lexer.IDENTIFIER || tok.Type == lexer.DOT {
			continue
		}

		// If we hit @, extract the decorator name
		if tok.Type == lexer.AT {
			// Next token should be the decorator name
			if i+1 < len(v.tokens) && (v.tokens[i+1].Type == lexer.IDENTIFIER || v.tokens[i+1].Type == lexer.VAR) {
				return v.extractDecoratorName(i + 1)
			}
		}

		// Something else - stop
		break
	}
	return ""
}

func (v *semanticValidator) skipParameterListBackward(rparenPos int) int {
	// Skip backward from ) to matching (
	parenDepth := 1
	for i := rparenPos - 1; i >= 0; i-- {
		switch v.tokens[i].Type {
		case lexer.RPAREN:
			parenDepth++
		case lexer.LPAREN:
			parenDepth--
			if parenDepth == 0 {
				return i - 1 // Return position before (
			}
		}
	}
	return -1 // No matching (
}

func (v *semanticValidator) findDecoratorAfter(pipePos int) string {
	// Walk forward from pipe position to find @ token
	// Skip whitespace
	for i := pipePos + 1; i < len(v.tokens); i++ {
		tok := v.tokens[i]

		// Skip whitespace, newlines
		if tok.Type == lexer.NEWLINE {
			continue
		}

		// If we hit @, this is a decorator
		if tok.Type == lexer.AT {
			// Next token should be the decorator name
			if i+1 < len(v.tokens) && (v.tokens[i+1].Type == lexer.IDENTIFIER || v.tokens[i+1].Type == lexer.VAR) {
				return v.extractDecoratorName(i + 1)
			}
		}

		// If we hit something else, it's plain shell (no validation needed)
		break
	}
	return ""
}

func (v *semanticValidator) extractDecoratorName(startPos int) string {
	// Build decorator name by following dot-separated identifiers
	// e.g., "file.read" or "timeout"
	name := string(v.tokens[startPos].Text)

	// Check for dot-separated parts
	i := startPos + 1
	for i < len(v.tokens) && v.tokens[i].Type == lexer.DOT {
		i++ // Skip dot
		if i < len(v.tokens) && v.tokens[i].Type == lexer.IDENTIFIER {
			name += "." + string(v.tokens[i].Text)
			i++
		} else {
			break
		}
	}

	return name
}

func (v *semanticValidator) validateStdoutSupport(decoratorName string, pipeToken lexer.Token) {
	schema, exists := v.tree.getSchema(decoratorName)
	if !exists {
		return // Decorator not registered - parser already reported error
	}

	if schema.IO == nil || !schema.IO.SupportsStdout {
		v.errors = append(v.errors, ParseError{
			Position:   pipeToken.Position,
			Message:    "@" + decoratorName + " does not produce stdout",
			Context:    "pipe operator",
			Got:        pipeToken.Type,
			Suggestion: "Only shell commands and decorators with stdout support can be piped from",
			Example:    "echo \"test\" | grep \"pattern\"",
			Note:       "Only decorators that produce stdout can be piped from",
		})
	}
}

func (v *semanticValidator) validateStdinSupport(decoratorName string, pipeToken lexer.Token) {
	schema, exists := v.tree.getSchema(decoratorName)
	if !exists {
		return // Decorator not registered - parser already reported error
	}

	if schema.IO == nil || !schema.IO.SupportsStdin {
		v.errors = append(v.errors, ParseError{
			Position:   pipeToken.Position,
			Message:    "@" + decoratorName + " does not accept stdin",
			Context:    "pipe operator",
			Got:        pipeToken.Type,
			Suggestion: "Only shell commands and decorators with stdin support can receive piped data",
			Example:    "echo \"test\" | grep \"pattern\"",
			Note:       "Only decorators that accept stdin can receive piped data",
		})
	}
}

func (v *semanticValidator) validateRedirectOperators() {
	// Walk through tokens to find redirect operators (> and >>)
	for i := 0; i < len(v.tokens); i++ {
		if v.tokens[i].Type != lexer.GT && v.tokens[i].Type != lexer.APPEND {
			continue
		}

		// Found a redirect operator at position i
		redirectToken := v.tokens[i]

		// Look forward to find the redirect target
		targetDecorator := v.findDecoratorAfter(i)

		// Validate target (must support redirect)
		if targetDecorator != "" {
			v.validateRedirectSupport(targetDecorator, redirectToken)
		}
	}
}

func (v *semanticValidator) validateRedirectSupport(decoratorName string, redirectToken lexer.Token) {
	schema, exists := v.tree.getSchema(decoratorName)
	if !exists {
		return // Decorator not registered - parser already reported error
	}

	// Check if decorator supports redirect at all
	if schema.Redirect == nil {
		v.errors = append(v.errors, ParseError{
			Position:   redirectToken.Position,
			Message:    "@" + decoratorName + " does not support redirection",
			Context:    "redirect operator",
			Got:        redirectToken.Type,
			Suggestion: "Only decorators with redirect support can be used as redirect targets",
			Example:    "echo \"test\" > output.txt",
			Note:       "Use @shell(\"output.txt\") or decorators that support redirect",
		})
		return
	}

	// Check if decorator supports the specific mode (> or >>)
	var mode types.RedirectMode
	if redirectToken.Type == lexer.GT {
		mode = types.RedirectOverwrite
	} else {
		mode = types.RedirectAppend
	}

	if !schema.Redirect.Support.SupportsMode(mode) {
		var operatorStr, modeStr string
		if redirectToken.Type == lexer.GT {
			operatorStr = ">"
			modeStr = "overwrite"
		} else {
			operatorStr = ">>"
			modeStr = "append"
		}

		v.errors = append(v.errors, ParseError{
			Position:   redirectToken.Position,
			Message:    "@" + decoratorName + " does not support " + modeStr + " (" + operatorStr + ")",
			Context:    "redirect operator",
			Got:        redirectToken.Type,
			Suggestion: "Use a different redirect mode or a decorator that supports " + modeStr,
			Example:    "echo \"test\" " + operatorStr + " output.txt",
			Note:       "@" + decoratorName + " only supports " + schema.Redirect.Support.String(),
		})
	}
}

// getSchema is a helper to look up decorator schemas
func (tree *ParseTree) getSchema(decoratorName string) (schema types.DecoratorSchema, exists bool) {
	return types.Global().GetSchema(decoratorName)
}
