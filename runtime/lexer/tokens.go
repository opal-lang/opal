package lexer

// TokenType represents lexical tokens for the v2 language design
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Chaining operations (execution flow)
	NEWLINE   // \n - sequential execution
	SEMICOLON // ; - sequential execution

	// Meta-programming keywords
	FOR     // for
	IN      // in
	IF      // if
	ELSE    // else
	WHEN    // when - pattern matching
	TRY     // try - error handling decorator
	CATCH   // catch - error handling decorator
	FINALLY // finally - error handling decorator

	// Language structure
	FUN       // fun - command definition
	VAR       // var
	AT        // @
	DOT       // .
	DOTDOTDOT // ... (range operator for for loops and when patterns)
	COLON     // :
	EQUALS    // =
	COMMA     // ,
	ARROW     // -> (for when patterns)

	// Brackets and braces
	LPAREN  // (
	RPAREN  // )
	LBRACE  // {
	RBRACE  // }
	LSQUARE // [
	RSQUARE // ]

	// Comparison operators (for if statements)
	EQ_EQ  // ==
	NOT_EQ // !=
	LT     // <
	LT_EQ  // <=
	GT     // >
	GT_EQ  // >=

	// Logical operators
	AND_AND // && (logical and)
	OR_OR   // || (logical or)
	NOT     // !

	// Arithmetic operators
	PLUS     // +
	MINUS    // -
	MULTIPLY // *
	DIVIDE   // /
	MODULO   // %

	// Increment/Decrement
	INCREMENT // ++
	DECREMENT // --

	// Assignment operators
	PLUS_ASSIGN     // +=
	MINUS_ASSIGN    // -=
	MULTIPLY_ASSIGN // *=
	DIVIDE_ASSIGN   // /=
	MODULO_ASSIGN   // %=

	// Shell chain operators
	AND    // && (chain success)
	OR     // || (chain failure)
	PIPE   // |
	APPEND // >>

	// Literals and content
	IDENTIFIER // command names, variable names, decorator names
	INTEGER    // 123, 0, -456
	FLOAT      // 3.14, -0.5, 123.0
	SCIENTIFIC // 1e6, 2.5e-3, 1.23e+4
	STRING     // "string" or 'string' content
	DURATION   // 30s, 5m, 1h30m, 500ms
	BOOLEAN    // true, false

	// Comments
	COMMENT // # single line comment
)

// Token represents a lexical token
type Token struct {
	Type     TokenType
	Text     []byte // Use []byte for zero-allocation performance
	Position Position
	// HasSpaceBefore is a parsing hint, not semantic data.
	// Helps parser distinguish cases like "-- released" vs "--release" (two tokens vs one).
	// Parser uses this during parsing then discards it.
	// Not part of token identity - "fun greet()" and "  fun greet()" are semantically identical.
	// We capture it here because we're already scanning whitespace in the lexer.
	HasSpaceBefore bool
}

// String returns the token text as a string (for testing and debugging)
func (t Token) String() string {
	return string(t.Text)
}

// Symbol returns the token's symbol or text representation.
// For tokens with Text (identifiers, literals), returns the text.
// For operator tokens with empty Text, returns the symbol (e.g., "-", "+", "&&").
// This method is allocation-free - it returns a static string for operators.
func (t Token) Symbol() string {
	if len(t.Text) > 0 {
		return string(t.Text)
	}

	// Return static symbols for operators (no allocation)
	switch t.Type {
	case MINUS:
		return "-"
	case PLUS:
		return "+"
	case MULTIPLY:
		return "*"
	case DIVIDE:
		return "/"
	case MODULO:
		return "%"
	case EQ_EQ:
		return "=="
	case NOT_EQ:
		return "!="
	case LT:
		return "<"
	case LT_EQ:
		return "<="
	case GT:
		return ">"
	case GT_EQ:
		return ">="
	case AND:
		return "&"
	case OR:
		return "|"
	case AND_AND:
		return "&&"
	case OR_OR:
		return "||"
	case NOT:
		return "!"
	case INCREMENT:
		return "++"
	case DECREMENT:
		return "--"
	case PLUS_ASSIGN:
		return "+="
	case MINUS_ASSIGN:
		return "-="
	case MULTIPLY_ASSIGN:
		return "*="
	case DIVIDE_ASSIGN:
		return "/="
	case MODULO_ASSIGN:
		return "%="
	case PIPE:
		return "|"
	case APPEND:
		return ">>"
	case EQUALS:
		return "="
	case COLON:
		return ":"
	case COMMA:
		return ","
	case SEMICOLON:
		return ";"
	case DOT:
		return "."
	case DOTDOTDOT:
		return "..."
	case ARROW:
		return "->"
	case AT:
		return "@"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case LSQUARE:
		return "["
	case RSQUARE:
		return "]"
	default:
		return ""
	}
}

// Position represents a position in the source code
type Position struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case NEWLINE:
		return "NEWLINE"
	case SEMICOLON:
		return "SEMICOLON"
	case FOR:
		return "FOR"
	case IN:
		return "IN"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case WHEN:
		return "WHEN"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case FINALLY:
		return "FINALLY"
	case FUN:
		return "FUN"
	case VAR:
		return "VAR"
	case AT:
		return "AT"
	case DOT:
		return "DOT"
	case DOTDOTDOT:
		return "DOTDOTDOT"
	case COLON:
		return "COLON"
	case EQUALS:
		return "EQUALS"
	case COMMA:
		return "COMMA"
	case ARROW:
		return "ARROW"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LSQUARE:
		return "LSQUARE"
	case RSQUARE:
		return "RSQUARE"
	case EQ_EQ:
		return "EQ_EQ"
	case NOT_EQ:
		return "NOT_EQ"
	case LT:
		return "LT"
	case LT_EQ:
		return "LT_EQ"
	case GT:
		return "GT"
	case GT_EQ:
		return "GT_EQ"
	case AND_AND:
		return "AND_AND"
	case OR_OR:
		return "OR_OR"
	case NOT:
		return "NOT"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case MULTIPLY:
		return "MULTIPLY"
	case DIVIDE:
		return "DIVIDE"
	case MODULO:
		return "MODULO"
	case INCREMENT:
		return "INCREMENT"
	case DECREMENT:
		return "DECREMENT"
	case PLUS_ASSIGN:
		return "PLUS_ASSIGN"
	case MINUS_ASSIGN:
		return "MINUS_ASSIGN"
	case MULTIPLY_ASSIGN:
		return "MULTIPLY_ASSIGN"
	case DIVIDE_ASSIGN:
		return "DIVIDE_ASSIGN"
	case MODULO_ASSIGN:
		return "MODULO_ASSIGN"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case PIPE:
		return "PIPE"
	case APPEND:
		return "APPEND"
	case IDENTIFIER:
		return "IDENTIFIER"

	case INTEGER:
		return "INTEGER"
	case FLOAT:
		return "FLOAT"
	case SCIENTIFIC:
		return "SCIENTIFIC"
	case STRING:
		return "STRING"
	case DURATION:
		return "DURATION"
	case BOOLEAN:
		return "BOOLEAN"
	case COMMENT:
		return "COMMENT"
	default:
		return "UNKNOWN"
	}
}

// Keywords maps string literals to their corresponding token types
var Keywords = map[string]TokenType{
	"fun":     FUN,
	"for":     FOR,
	"in":      IN,
	"if":      IF,
	"else":    ELSE,
	"when":    WHEN,
	"try":     TRY,
	"catch":   CATCH,
	"finally": FINALLY,
	"var":     VAR,
	"true":    BOOLEAN,
	"false":   BOOLEAN,
}

// SingleCharTokens maps single characters to their token types
var SingleCharTokens = map[byte]TokenType{
	'@':  AT,
	'.':  DOT,
	':':  COLON,
	'=':  EQUALS,
	',':  COMMA,
	'(':  LPAREN,
	')':  RPAREN,
	'{':  LBRACE,
	'}':  RBRACE,
	'[':  LSQUARE,
	']':  RSQUARE,
	'|':  PIPE,
	'<':  LT,
	'>':  GT,
	'!':  NOT,
	'+':  PLUS,
	'-':  MINUS,
	'*':  MULTIPLY,
	'/':  DIVIDE,
	'%':  MODULO,
	'\n': NEWLINE,
	';':  SEMICOLON,
}

// TwoCharTokens maps two-character sequences to their token types
var TwoCharTokens = map[string]TokenType{
	"->": ARROW,
	"==": EQ_EQ,
	"!=": NOT_EQ,
	"<=": LT_EQ,
	">=": GT_EQ,
	"&&": AND_AND,
	"||": OR_OR,
	">>": APPEND,
	"++": INCREMENT,
	"--": DECREMENT,
	"+=": PLUS_ASSIGN,
	"-=": MINUS_ASSIGN,
	"*=": MULTIPLY_ASSIGN,
	"/=": DIVIDE_ASSIGN,
	"%=": MODULO_ASSIGN,
}
