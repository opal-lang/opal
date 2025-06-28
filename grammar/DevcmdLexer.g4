/**
 * Devcmd Lexer Grammar
 *
 * Lexer for the devcmd language - a declarative syntax for defining
 * CLI tools from simple command definitions. Devcmd transforms command
 * definitions into standalone CLI binaries with process management,
 * variable substitution, and workflow automation.
 *
 * Language features:
 * - Variable definitions: def NAME = value;
 * - Simple commands: build: go build ./cmd;
 * - Block commands: deploy: { build; test; kubectl apply -f k8s/ }
 * - Process management: watch/stop command pairs
 * - Decorators: @name(...) for command metadata, processing, and variables
 * - Shell command syntax: pipes, redirections, background processes
 */
lexer grammar DevcmdLexer;

// Add predicate function for comment detection
@lexer::members {
    func (p *DevcmdLexer) isCommentLine() bool {
        // Get current position in line
        pos := p.GetCharPositionInLine()

        // If at start of line, it's a comment
        if pos == 0 {
            return true
        }

        // Look back from current position to start of line
        // Check if only whitespace (spaces/tabs) precedes current position
        for i := 1; i <= pos; i++ {
            char := p.GetInputStream().LA(-i)
            if char != ' ' && char != '\t' {
                return false // Non-whitespace found, not a comment line
            }
        }
        return true // Only whitespace found, this is a comment line
    }
}

// Keywords - must come first for precedence
DEF : 'def' ;
WATCH : 'watch' ;
STOP : 'stop' ;

// Regular decorator start - for @name: syntax and single @
AT : '@' ;

// Structural operators and delimiters
EQUALS : '=' ;
COLON : ':' ;
SEMICOLON : ';' ;
LBRACE : '{' ;
RBRACE : '}' ;
LPAREN : '(' ;
RPAREN : ')' ;
BACKSLASH : '\\' ;

// String literals - must come before other character tokens
STRING : '"' (~["\\\r\n] | '\\' .)* '"' ;
SINGLE_STRING : '\'' (~['\\\r\n] | '\\' .)* '\'' ;

// Semantic token definitions with specific naming conventions
// Note: We use a single NAME token and handle semantic validation in the parser
// This avoids lexer ambiguity issues while maintaining semantic correctness

// NAME: General identifier token that covers all naming patterns
// - Commands: build, nix-build, docker-compose-up, deploy-v2
// - Decorators: var, sh, parallel, retry-on-fail, wait-for
// - Variables: SRC, BUILD_DIR, NODE_ENV, PORT_8080, MY_VAR
NAME : [A-Za-z] [A-Za-z0-9_-]* ;

// NUMBER: Numeric literals including decimals
NUMBER : '-'? [0-9]+ ('.' [0-9]+)? ;

// Path-like content (handles things like ./src, *.tmp, etc.)
// More specific pattern to avoid conflicts with names
PATH_CONTENT : [./~] [A-Za-z0-9._/*-]+ ;

// Shell operators and special characters as individual tokens
// Reordered to put more specific tokens first
AMPERSAND : '&' ;
PIPE : '|' ;
LT : '<' ;
GT : '>' ;
DOT : '.' ;
COMMA : ',' ;
SLASH : '/' ;
DASH : '-' ;
STAR : '*' ;
PLUS : '+' ;
QUESTION : '?' ;
EXCLAIM : '!' ;
PERCENT : '%' ;
CARET : '^' ;
TILDE : '~' ;
UNDERSCORE : '_' ;
LBRACKET : '[' ;
RBRACKET : ']' ;
DOLLAR : '$' ;
HASH : '#' ;
DOUBLEQUOTE : '"' ;
BACKTICK : '`' ;

// Whitespace and comments - must be at the end
// Support # comments that start at beginning of line or after whitespace only
COMMENT : {p.isCommentLine()}? [ \t]* '#' ~[\r\n]* -> channel(HIDDEN) ;

NEWLINE : '\r'? '\n' ;
WS : [ \t]+ -> channel(HIDDEN) ;
