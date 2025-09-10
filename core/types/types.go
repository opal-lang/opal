package types

import "fmt"

// ExpressionType represents the type of an expression
type ExpressionType int

const (
	StringType ExpressionType = iota
	NumberType
	DurationType
	BooleanType
	IdentifierType
)

// String returns a string representation of the ExpressionType
func (t ExpressionType) String() string {
	switch t {
	case StringType:
		return "string"
	case NumberType:
		return "number"
	case DurationType:
		return "duration"
	case BooleanType:
		return "boolean"
	case IdentifierType:
		return "identifier"
	default:
		return "unknown"
	}
}

// TokenType represents the type of token in Devcmd
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Language structure tokens
	VAR      // var
	WATCH    // watch
	STOP     // stop
	WHEN     // when (for @when pattern decorator)
	TRY      // try (for @try pattern decorator)
	AT       // @
	COLON    // :
	EQUALS   // =
	COMMA    // ,
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	ASTERISK // * (wildcard in patterns)

	// Shell operators
	AND    // &&
	OR     // ||
	PIPE   // |
	APPEND // >>

	// Literals and Content
	IDENTIFIER   // command names, variable names, decorator names, patterns
	SHELL_TEXT   // shell command text
	SHELL_END    // marks end of shell content
	NUMBER       // 8080, 3.14, -100
	STRING_START // opening quote for strings (" for interpolated, ' for literal)
	STRING_TEXT  // text content within strings
	STRING_END   // closing quote for strings (" or ')
	DURATION     // 30s, 5m, 1h
	BOOLEAN      // true, false

	// Comments
	COMMENT           // #
	MULTILINE_COMMENT // /* */
)

// Pre-computed token name lookup for fast debugging
var tokenNames = [...]string{
	EOF:        "EOF",
	ILLEGAL:    "ILLEGAL",
	VAR:        "VAR",
	WATCH:      "WATCH",
	STOP:       "STOP",
	WHEN:       "WHEN",
	TRY:        "TRY",
	AT:         "AT",
	COLON:      "COLON",
	EQUALS:     "EQUALS",
	COMMA:      "COMMA",
	LPAREN:     "LPAREN",
	RPAREN:     "RPAREN",
	LBRACE:     "LBRACE",
	RBRACE:     "RBRACE",
	ASTERISK:   "ASTERISK",
	AND:        "AND",
	OR:         "OR",
	PIPE:       "PIPE",
	APPEND:     "APPEND",
	IDENTIFIER: "IDENTIFIER",
	SHELL_TEXT: "SHELL_TEXT",
	SHELL_END:  "SHELL_END",
	NUMBER:     "NUMBER",

	STRING_START:      "STRING_START",
	STRING_TEXT:       "STRING_TEXT",
	STRING_END:        "STRING_END",
	DURATION:          "DURATION",
	BOOLEAN:           "BOOLEAN",
	COMMENT:           "COMMENT",
	MULTILINE_COMMENT: "MULTILINE_COMMENT",
}

func (t TokenType) String() string {
	if int(t) < len(tokenNames) && int(t) >= 0 {
		return tokenNames[t]
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// StringLiteralType represents the type of string literal
type StringLiteralType int

const (
	DoubleQuoted StringLiteralType = iota // "string"
	SingleQuoted                          // 'string'
	Backtick                              // `string`
)

// SemanticTokenType represents semantic categories for syntax highlighting
type SemanticTokenType int

const (
	SemKeyword   SemanticTokenType = iota // var, watch, stop, when, try
	SemCommand                            // command names
	SemVariable                           // variable names
	SemString                             // string literals
	SemNumber                             // numeric literals
	SemComment                            // comments
	SemOperator                           // :, =, {, }, (, ), @, *
	SemShellText                          // shell text content
	SemDecorator                          // decorators like @timeout, @retry
	SemPattern                            // pattern-matching decorators (@when, @try)
	SemBoolean                            // boolean literals (true, false)
	SemParameter                          // parameter names in decorators
)

// SourceSpan represents a precise location in source code
type SourceSpan struct {
	Start SourcePosition `json:"start"`
	End   SourcePosition `json:"end"`
}

// SourcePosition represents a position in source code
type SourcePosition struct {
	Line   int `json:"line"`   // 1-based
	Column int `json:"column"` // 1-based
	Offset int `json:"offset"` // 0-based byte offset
}

// Token represents a single token with enhanced position information
type Token struct {
	Type       TokenType
	Semantic   SemanticTokenType
	Line       int
	Column     int
	EndLine    int
	EndColumn  int
	Value      string
	Raw        string
	StringType StringLiteralType

	// Enhanced positioning for shell content
	Span SourceSpan `json:"span"`
}

// Position returns a formatted position string for error reporting
func (t Token) Position() string {
	if t.Line == t.EndLine {
		return fmt.Sprintf("%d:%d-%d", t.Line, t.Column, t.EndColumn)
	}
	return fmt.Sprintf("%d:%d-%d:%d", t.Line, t.Column, t.EndLine, t.EndColumn)
}
