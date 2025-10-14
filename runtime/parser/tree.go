package parser

import (
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
