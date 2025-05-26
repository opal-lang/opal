/**
 * Devcmd Grammar Definition - POSIX Compatible
 *
 * This ANTLR4 grammar defines the syntax for 'devcmd' (**D**eclarative **E**xecution **V**ocabulary **Cmd**),
 * a domain-specific language for orchestrating build tasks, development environments, and service management.
 *
 * Core features:
 * 1. Named commands with optional modifiers: 'build: npm run build;', 'watch server: node start;'
 * 2. Variables for reuse: 'def SRC = ./src;'
 * 3. Variable references in commands: 'build: cd $(SRC) && make;'
 * 4. Service management with 'watch' and 'stop' commands
 * 5. Multi-statement blocks: 'setup: { npm install; go mod tidy; }'
 * 6. Background processes with ampersand: 'run-all: { server & client & db & }'
 * 7. POSIX shell compatibility: supports find, test, and other shell constructs using {}
 * 8. Consistent semicolon termination: all commands must end with semicolon
 *
 * This grammar handles lexical structure and syntax only. Semantic rules
 * (variable definition before use, watch/stop pairing, unique command names)
 * are enforced during analysis phases after parsing.
 */
grammar devcmd;

/**
 * Parser Rules
 * These define the structural hierarchy of the devcmd language
 */

// A devcmd program consists of multiple lines followed by EOF
program : line* EOF ;

// Each line represents a discrete unit in the program
line
    : variableDefinition   // A variable assignment with 'def'
    | commandDefinition    // A named command pattern
    | NEWLINE              // Empty lines for formatting
    ;

// Variables store reusable text values for reference in commands
// Now with required semicolon
variableDefinition : DEF NAME (EQUALS commandText?)? SEMICOLON (NEWLINE | EOF) ;

// Commands define executable operations, optionally with service lifecycle modifiers
commandDefinition
    : (WATCH | STOP)? NAME COLON (simpleCommand | blockCommand)
    ;

// A simple command contains a single instruction, potentially with line continuations
// Now requires semicolon termination for consistent parsing
simpleCommand : (commandText continuationLine*)? SEMICOLON (NEWLINE | EOF) ;

// Block commands group multiple statements within braces
// Using explicit brace matching to avoid conflicts with command text braces
blockCommand : LBRACE NEWLINE? blockStatements RBRACE (NEWLINE | EOF)? ;

// Block statements can be empty or contain one or more commands
blockStatements
    : /* empty */               // Allow empty blocks
    | nonEmptyBlockStatements   // One or more statements
    ;

// Multiple statements are separated by semicolons
// The optional final semicolon allows for trailing semicolon in blocks like: { cmd1; cmd2; }
nonEmptyBlockStatements
    : blockStatement (SEMICOLON NEWLINE* blockStatement)* SEMICOLON? NEWLINE*
    ;

// Each block statement can be backgrounded with ampersand
// Space before ampersand is implicit since whitespace is skipped by the lexer
blockStatement : commandText AMPERSAND? ;

// Line continuations let commands span multiple lines
continuationLine : BACKSLASH NEWLINE commandText ;

// Command text is the actual instruction content
// Must match at least one element to avoid ambiguity
// Now includes ESCAPED_SEMICOLON for shell escaping
commandText
    : (ESCAPED_CHAR
      | OUR_VARIABLE_REFERENCE      // $(NAME)
      | SHELL_VARIABLE_REFERENCE    // $NAME
      | ESCAPED_SEMICOLON           // \\; for shell commands like find
      | ESCAPED_DOLLAR              // \$ for shell variables and command substitution
      | AMPERSAND
      | COLON
      | EQUALS
      | NUMBER
      | STOP                        // ← allow reserved words here
      | WATCH                       // ← (optional) ditto
      | NAME
      | BACKSLASH                   // ← allow literal backslashes for shell escaping
      | COMMAND_TEXT
      )+
    ;

/**
 * Lexer Rules
 * These define the atomic elements and token patterns of the language
 *
 * CRITICAL: Order matters for lexer precedence!
 * More specific patterns must come before more general ones.
 */

// Keywords that have special meaning in devcmd
DEF : 'def' ;     // Variable definition marker
EQUALS : '=' ;    // Assignment operator
COLON : ':' ;     // Command separator
WATCH : 'watch' ; // Service startup modifier
STOP : 'stop' ;   // Service shutdown modifier

// HIGHEST PRIORITY: Variable references (most specific patterns first)
OUR_VARIABLE_REFERENCE : '$(' NAME ')' ;
SHELL_VARIABLE_REFERENCE : '$' [A-Za-z][A-Za-z0-9_]* ;

// VERY HIGH PRIORITY: Shell escaped semicolon - MUST come before all other escape rules
// This is needed for POSIX commands like: find . -name "*.tmp" -exec rm {} \;
ESCAPED_SEMICOLON : '\\;' ;

// User writes \$ in devcmd source to get $ in the generated shell command
// This is needed for shell variables (\$PATH) and command substitution (\$(date))
ESCAPED_DOLLAR : '\\$' ;

// HIGH PRIORITY: Other escape sequences
// Note: semicolon is NOT included here since it's handled above
ESCAPED_CHAR : '\\' ( [\\nrt${}()"]
                     | 'x' [0-9a-fA-F][0-9a-fA-F]
                     | 'u' [0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]
                     ) ;

// MEDIUM PRIORITY: Structural delimiters
LBRACE : '{' ;    // Block start (also available in command text)
RBRACE : '}' ;    // Block end (also available in command text)
SEMICOLON : ';' ; // Statement separator
AMPERSAND : '&' ; // Background process indicator and also allowed in command text
BACKSLASH : '\\' ; // Line continuation marker and general shell escaping

// MEDIUM PRIORITY: Identifiers and numbers
NAME : [A-Za-z][A-Za-z0-9_-]* ;

// Numeric literals - supports integers, decimals, and decimals starting with dot
// Examples: 42, 3.14, .5, 8080, 1.0
NUMBER : [0-9]* '.' [0-9]+ | [0-9]+ ;

// LOW PRIORITY: General command text content - excludes structural delimiters for proper tokenization
// Note: $(NAME) is handled by OUR_VARIABLE_REFERENCE token with higher precedence
COMMAND_TEXT : ~[\r\n \t:=;\\]+ ;

// LOWEST PRIORITY: Comments and formatting elements
COMMENT : '#' ~[\r\n]* -> channel(HIDDEN) ;  // Comments don't affect execution, so hide from parser
NEWLINE : '\r'? '\n' ;
WS    : [ \t]+ -> channel(HIDDEN) ;

/**
 * Implementation Guidelines
 *
 * A compliant devcmd compiler should implement these features:
 *
 * 1. Runtime Environment
 *    • Commands execute in a POSIX-compatible shell environment
 *    • Environment variables from parent process are preserved
 *    • Working directory is maintained across commands within a block
 *    • Full POSIX shell syntax support including braces, parentheses, pipes
 *
 * 2. Variable Handling
 *    • $(VAR) references expand to their defined value before execution
 *    • Shell variables like $HOME and ${PATH} pass through to the shell
 *    • All devcmd variables must be defined before use
 *
 * 3. Process Management
 *    • 'watch' commands create persistent process groups
 *    • Process groups register with a process registry for cleanup
 *    • 'stop' commands gracefully terminate matching process groups
 *    • Background processes ('&') run concurrently within their block
 *    • Foreground commands block until completion
 *
 * 4. Error Handling
 *    • Syntax errors report line and column of failure
 *    • Command failures propagate exit codes
 *    • Process termination ensures cleanup of all child processes
 *
 * 5. Performance Requirements
 *    • Parsing: O(n) time complexity for n lines of input
 *    • Memory: Peak usage below 5x input file size
 *    • Startup: Command execution begins within 100ms
 *
 * 6. POSIX Shell Compatibility & Syntax
 *    • Support for find, test, and other utilities that use {}
 *    • Proper handling of shell metacharacters in command text
 *    • All commands require semicolon termination for consistent parsing
 *    • Clear distinction between devcmd structure and shell command content
 *    • Escaped semicolons (\\;) are converted to (\;) for shell execution
 */
