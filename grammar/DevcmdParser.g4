/**
 * Devcmd Parser Grammar
 *
 * Parser for the devcmd language - generates CLI tools from command definitions.
 *
 * Devcmd syntax supports:
 * - Variable definitions: def PORT = 8080;
 * - Simple commands: build: go build -o bin/app ./cmd;
 * - Block workflows: deploy: { build; test; kubectl apply -f k8s/ }
 * - Process management: watch dev / stop dev command pairs
 * - Decorators: @timeout(30s), @retry(3), @parallel, @var(PORT)
 * - Line continuations: command \
 *                        --flag value;
 * - Shell features: pipes, redirections, background processes (&)
 */
parser grammar DevcmdParser;

options {
    tokenVocab = DevcmdLexer;
}

/**
 * TOP LEVEL STRUCTURE
 * Program consists of variable definitions and command definitions
 */

// Entry point - sequence of lines ending with EOF
program : line* EOF ;

// Each line can be a definition, command, or empty line
line
    : variableDefinition   // def NAME = value;
    | commandDefinition    // [watch|stop] NAME: body
    | NEWLINE              // Empty lines for formatting
    ;

/**
 * VARIABLE DEFINITIONS
 * Format: def NAME = value;
 */

// Variable definition with optional value
variableDefinition : DEF NAME EQUALS variableValue? SEMICOLON ;

// Variable value - can contain command text
variableValue : commandText ;

/**
 * COMMAND DEFINITIONS
 * Format: [watch|stop] NAME: body
 * Body can be simple command, block, or decorator
 */

// Command with optional watch/stop modifier
commandDefinition : (WATCH | STOP)? NAME COLON commandBody ;

// Command body - multiple alternatives for different command types
commandBody
    : decoratedCommand     // @name(...) or @name: ...
    | blockCommand         // { ... }
    | simpleCommand        // command;
    ;

/**
 * DECORATOR SYNTAX
 * Three forms:
 * 1. Function: @name(...) - parser handles nested parentheses and newlines
 * 2. Block: @name: { ... }
 * 3. Simple: @name: processed command
 */

// Decorated command with labels for visitor compatibility
decoratedCommand
    : functionDecorator    #functionDecoratorLabel
    | blockDecorator       #blockDecoratorLabel
    | simpleDecorator      #simpleDecoratorLabel
    ;

// UPDATED: Function decorator using separate tokens
functionDecorator : AT NAME LPAREN decoratorContent RPAREN SEMICOLON? ;

// Block decorator: @name: { ... }
blockDecorator : AT decorator COLON blockCommand ;

// Simple decorator: @name: command
simpleDecorator : AT decorator COLON decoratorCommand ;

// Decorator name (kept for compatibility)
decorator : NAME ;

// Content inside @name(...) - handle nested parentheses, @var() decorators, and newlines
decoratorContent : decoratorElement* ;

// Elements that can appear in decorator content
// Enhanced to handle nested @var() decorators as proper structures
decoratorElement
    : nestedDecorator                     // @var(NAME) etc. - parsed as decorators
    | LPAREN decoratorContent RPAREN     // Nested parentheses
    | NEWLINE                            // Allow newlines
    | decoratorTextElement               // Any other content as text
    ;

// Nested decorator within another decorator (e.g., @var inside @sh)
// UPDATED: Use AT NAME LPAREN as separate tokens
nestedDecorator : AT NAME LPAREN decoratorContent RPAREN ;

// Text elements that can appear in decorator content
decoratorTextElement
    : NAME | NUMBER | STRING | SINGLE_STRING | PATH_CONTENT
    | AMPERSAND | PIPE | LT | GT | COLON | EQUALS | BACKSLASH
    | DOT | COMMA | SLASH | DASH | STAR | PLUS | QUESTION | EXCLAIM
    | PERCENT | CARET | TILDE | UNDERSCORE | LBRACKET | RBRACKET
    | LBRACE | RBRACE | DOLLAR | HASH | DOUBLEQUOTE
    | SEMICOLON | WATCH | STOP | DEF | CONTENT
    ;

/**
 * REGULAR COMMANDS
 * Simple and block commands with support for continuations and newlines
 */

// Simple command with optional line continuations and required semicolon
simpleCommand : commandText continuationLine* SEMICOLON ;

// Command text without semicolon requirement (for use in simple decorators)
decoratorCommand : commandText continuationLine* ;

// Block command containing multiple statements with proper newline handling
blockCommand : LBRACE NEWLINE? blockStatements RBRACE ;

// Block content structure (compatible with existing code)
blockStatements
    : /* empty */               // Allow empty blocks
    | nonEmptyBlockStatements   // One or more statements
    ;

// Non-empty block statements separated by semicolons with optional newlines
nonEmptyBlockStatements
    : blockStatement (SEMICOLON NEWLINE* blockStatement)* SEMICOLON? NEWLINE*
    ;

// Individual statement within a block
blockStatement
    : decoratedCommand                    // Decorators work in blocks
    | commandText continuationLine*       // Regular commands (no semicolon in blocks)
    ;

/**
 * LINE CONTINUATIONS
 * Support for multi-line commands using backslash
 */

// Line continuation: backslash + newline + more command text
continuationLine : BACKSLASH NEWLINE commandText ;

/**
 * COMMAND TEXT PARSING
 * Flexible parsing of shell-like command content
 * Enhanced to support inline @var() decorators
 */

// Command text - sequence of content elements
commandText : commandTextElement* ;

// Individual elements that can appear in command text
// Enhanced to include inline decorators like @var(NAME)
commandTextElement
    : inlineDecorator       // @var(NAME) etc. - parsed as decorators in command text
    | NAME                  // Identifiers
    | NUMBER                // Numeric literals
    | STRING                // Double quoted strings
    | SINGLE_STRING         // Single quoted strings
    | PATH_CONTENT          // Path-like content (./src, *.tmp, etc.)
    | LPAREN                // ( - shell subshells, grouping
    | RPAREN                // ) - shell subshells, grouping
    | LBRACE                // { - shell brace expansion
    | RBRACE                // } - shell brace expansion
    | LBRACKET              // [ - shell tests
    | RBRACKET              // ] - shell tests
    | AMPERSAND             // & - shell background processes
    | PIPE                  // | - shell pipes
    | LT                    // < - shell input redirection
    | GT                    // > - shell output redirection
    | COLON                 // : - allowed in commands
    | EQUALS                // = - allowed in commands
    | BACKSLASH             // \ - shell escaping
    | DOT                   // . - paths, decimals
    | COMMA                 // , - lists
    | SLASH                 // / - paths
    | DASH                  // - - command options
    | STAR                  // * - globs
    | PLUS                  // + - expressions
    | QUESTION              // ? - patterns
    | EXCLAIM               // ! - negation
    | PERCENT               // % - modulo
    | CARET                 // ^ - patterns
    | TILDE                 // ~ - home dir
    | UNDERSCORE            // _ - identifiers
    | DOLLAR                // $ - shell variables and command substitution
    | HASH                  // # - when not comment
    | DOUBLEQUOTE           // " - when not in string
    | AT                    // @ - when not decorator start
    | WATCH                 // Allow keywords in command text
    | STOP                  // Allow keywords in command text
    | DEF                   // Allow keywords in command text
    | CONTENT               // General content
    ;

// Inline decorator in command text (e.g., go run @var(MAIN_FILE))
// UPDATED: Use AT NAME LPAREN as separate tokens
inlineDecorator : AT NAME LPAREN decoratorContent RPAREN ;
