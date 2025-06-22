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

// Keywords - must come first for precedence
DEF : 'def' ;
WATCH : 'watch' ;
STOP : 'stop' ;

// REMOVED: Special decorator pattern - let parser handle @name( as separate tokens
// We now handle @name( as AT NAME LPAREN in the parser
// AT_NAME_LPAREN : '@' [A-Za-z] [A-Za-z0-9_-]* '(' ;

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
AMPERSAND : '&' ;
PIPE : '|' ;
LT : '<' ;
GT : '>' ;

// String literals
STRING : '"' (~["\\\r\n] | '\\' .)* '"' ;
SINGLE_STRING : '\'' (~['\\\r\n] | '\\' .)* '\'' ;

// Identifiers and literals
NAME : [A-Za-z][A-Za-z0-9_-]* ;
NUMBER : '-'? [0-9]+ ('.' [0-9]+)? ;

// Path-like content (handles things like ./src, *.tmp, etc.)
PATH_CONTENT : [./~] [A-Za-z0-9._/*-]* ;

// Shell operators and special characters as individual tokens
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

// General content - catch-all for other characters in commands
// This needs to be carefully ordered to not conflict with other tokens
CONTENT : ~[ \t\r\n@{}();:"'#] ;

// Whitespace and comments
COMMENT : '#' ~[\r\n]* -> channel(HIDDEN) ;
NEWLINE : '\r'? '\n' ;
WS : [ \t]+ -> channel(HIDDEN) ;
