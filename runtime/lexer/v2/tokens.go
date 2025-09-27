package v2

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
	VAR    // var
	AT     // @
	DOT    // .
	COLON  // :
	EQUALS // =
	COMMA  // ,
	ARROW  // -> (for when patterns)

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
	Type           TokenType
	Text           []byte // Use []byte for zero-allocation performance
	Position       Position
	HasSpaceBefore bool // True if whitespace preceded this token
}

// String returns the token text as a string (for testing and debugging)
func (t Token) String() string {
	return string(t.Text)
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
	case VAR:
		return "VAR"
	case AT:
		return "AT"
	case DOT:
		return "DOT"
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
