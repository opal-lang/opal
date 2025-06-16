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
 * - Decorators: @timeout(30s), @retry(3), @parallel
 * - Variable expansion: $(PORT), shell variables $USER
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
    : AT_NAME_LPAREN decoratorContent RPAREN SEMICOLON?    #functionDecorator
    | AT decorator COLON blockCommand                      #blockDecorator
    | AT decorator COLON decoratorCommand                  #simpleDecorator
    ;

// Decorator name (kept for compatibility)
decorator : NAME ;

// Content inside @name(...) - handle nested parentheses and newlines
decoratorContent : decoratorElement* ;

// Elements that can appear in decorator content
// This handles nested parentheses by recursively parsing them
// Also allows newlines and all other content
decoratorElement
    : LPAREN decoratorContent RPAREN         // Nested parentheses
    | NEWLINE                                // Allow newlines
    | ~(LPAREN | RPAREN | NEWLINE)+         // Any sequence of non-paren, non-newline tokens
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
 */

// Command text - sequence of content elements
commandText : commandTextElement* ;

// Individual elements that can appear in command text
commandTextElement
    : VAR_REF           // $(VAR) - devcmd variable
    | SHELL_VAR         // $VAR - shell variable
    | ESCAPED_DOLLAR    // \$ - literal dollar
    | ESCAPED_SEMICOLON // \; - literal semicolon
    | ESCAPED_BRACE     // \{ or \} - literal braces
    | NAME              // Identifiers
    | NUMBER            // Numeric literals
    | STRING            // Double quoted strings
    | SINGLE_STRING     // Single quoted strings
    | PATH_CONTENT      // Path-like content (./src, *.tmp, etc.)
    | LPAREN            // ( - shell subshells, grouping
    | RPAREN            // ) - shell subshells, grouping
    | LBRACE            // { - shell brace expansion
    | RBRACE            // } - shell brace expansion
    | LBRACKET          // [ - shell tests
    | RBRACKET          // ] - shell tests
    | AMPERSAND         // & - shell background processes
    | PIPE              // | - shell pipes
    | LT                // < - shell input redirection
    | GT                // > - shell output redirection
    | COLON             // : - allowed in commands
    | EQUALS            // = - allowed in commands
    | BACKSLASH         // \ - shell escaping
    | DOT               // . - paths, decimals
    | COMMA             // , - lists
    | SLASH             // / - paths
    | DASH              // - - command options
    | STAR              // * - globs
    | PLUS              // + - expressions
    | QUESTION          // ? - patterns
    | EXCLAIM           // ! - negation
    | PERCENT           // % - modulo
    | CARET             // ^ - patterns
    | TILDE             // ~ - home dir
    | UNDERSCORE        // _ - identifiers
    | DOLLAR            // $ - when not part of variable ref
    | HASH              // # - when not comment
    | DOUBLEQUOTE       // " - when not in string
    | AT                // @ - when not decorator start
    | WATCH             // Allow keywords in command text
    | STOP              // Allow keywords in command text
    | DEF               // Allow keywords in command text
    | CONTENT           // General content
    ;
