---
title: "Opal Grammar"
audience: "Language Reference"
summary: "Lexer and parser grammar for Opal"
---

# Opal Grammar

Opal is a script builder.

Metaprogramming constructs shape an execution plan, and runtime constructs execute shell and decorator behavior.

- Plan-time metaprogramming: `fun`, function calls, `var`, `if`, `for`, `when`
- Runtime execution: shell commands and execution decorators (for example `@retry`, `@timeout`, `@parallel`)

This document defines the language as authoritative grammar and syntax rules.

## EBNF Notation

- `|` alternation
- `[]` optional
- `{}` repetition (zero or more)
- `()` grouping
- quoted text denotes terminal symbols

## Lexical Grammar

### Identifiers

```ebnf
identifier = ("_" | letter) { "_" | letter | digit } ;
letter     = "a".."z" | "A".."Z" ;
digit      = "0".."9" ;
```

### Keywords

```ebnf
"fun" | "var" | "if" | "else" | "for" | "in" | "when" | "try" | "catch" | "finally"
```

### Literals

```ebnf
string_literal = dq_string | sq_string | bt_string ;
dq_string      = '"' { char | escape } '"' ;
sq_string      = "'" { char | escape } "'" ;
bt_string      = "`" { char | newline } "`" ;

int_literal    = digit { digit } ;
float_literal  = (digit { digit } "." [digit { digit }]) | ("." digit { digit }) ;
bool_literal   = "true" | "false" ;

duration_literal = int_literal duration_unit { int_literal duration_unit } ;
duration_unit    = "ns" | "us" | "ms" | "s" | "m" | "h" | "d" | "w" | "y" ;
```

### Comments

```ebnf
line_comment  = "//" { any_char_except_newline } ;
block_comment = "/*" { any_char } "*/" ;
```

### Operators and punctuation

```ebnf
arith_op      = "+" | "-" | "*" | "/" | "%" ;
cmp_op        = "==" | "!=" | "<" | "<=" | ">" | ">=" ;
logic_op      = "&&" | "||" | "!" ;
assign_op     = "+=" | "-=" | "*=" | "/=" | "%=" ;
incdec_op     = "++" | "--" ;
shell_chain   = "&&" | "||" | "|" | ";" ;
redirect_op   = ">" | ">>" ;

punctuation   = "@" | "." | ":" | "," | "=" | "(" | ")" | "{" | "}" | "[" | "]" | "->" | "..." ;
```

## Syntax Grammar

### Source

```ebnf
source   = { top_item } ;
top_item = function_decl | statement ;
```

### Blocks

```ebnf
block = "{" { statement } "}" ;
```

Braces delimit a block scope. Declarations and mutations inside a block do not leak outside that block.

### Function declarations

```ebnf
function_decl = "fun" identifier [param_list] ("=" shorthand_body | block) ;

param_list    = "(" [param_group { "," param_group }] ")" ;
param_group   = identifier { "," identifier } type_annotation [default_value] ;
type_annotation = [":"] identifier ;

default_value = "=" default_atom ;
default_atom  = string_literal | int_literal | float_literal | bool_literal | duration_literal | identifier ;

shorthand_body = shell_command | expr ;
```

Notes:
- Function declarations are top-level declarations.
- Parameter groups support both `name Type` and `name: Type`.
- Default values parse as one default atom in the declaration syntax.

### Statements

```ebnf
statement = var_decl
          | assignment_stmt
          | if_stmt
          | for_stmt
          | when_stmt
          | try_stmt
          | decorator_stmt
          | function_call_stmt
          | shell_command ;
```

### Variables and assignment

```ebnf
var_decl = "var" identifier "=" expr
         | "var" "(" var_binding { [var_sep] var_binding } ")" ;

var_binding = identifier "=" expr ;
var_sep     = newline | ";" ;

assignment_stmt = identifier assign_op expr ;
```

### Function calls

```ebnf
function_call_stmt = identifier "(" [call_arg { "," call_arg }] ")" ;
call_arg           = [identifier "="] expr ;
```

Disambiguation rule:
- `name(...)` (no space before `(`) parses as a function call statement.
- `name (...)` parses as shell syntax.

### Control flow

```ebnf
if_stmt = "if" expr block ["else" (if_stmt | block)] ;

for_stmt = "for" identifier "in" for_collection block ;
for_collection = identifier | decorator_ref | array_literal | range_expr ;
range_expr = range_endpoint "..." range_endpoint ;
range_endpoint = int_literal | decorator_ref ;

when_stmt = "when" when_subject "{" { when_arm } "}" ;
when_subject = identifier | decorator_ref ;
when_arm = pattern "->" (block | statement) ;

pattern = pattern_primary { "|" pattern_primary } ;
pattern_primary = string_literal
                | "r" string_literal
                | int_literal "..." int_literal
                | "else" ;

try_stmt = "try" block ["catch" block] ["finally" block] ;
```

### Decorators

```ebnf
decorator_stmt = decorator_ref [block] [shell_chain shell_command] ;

decorator_ref  = "@" decorator_path ["." identifier] [decorator_args] ;
decorator_path = identifier { "." identifier } ;

decorator_args = "(" [decorator_arg { "," decorator_arg }] ")" ;
decorator_arg  = [identifier "="] decorator_arg_value ;

decorator_arg_value = string_literal
                    | int_literal
                    | float_literal
                    | bool_literal
                    | duration_literal
                    | identifier
                    | array_literal
                    | object_literal ;
```

Decorator block rules come from decorator capabilities (required, optional, forbidden).

### Shell commands

```ebnf
shell_command = shell_segment { shell_chain shell_segment } ;
shell_segment = shell_arg { shell_arg } [redirect_clause] ;
redirect_clause = redirect_op shell_arg ;

shell_arg = string_literal | identifier | decorator_ref | token_cluster ;
```

`token_cluster` means adjacent non-space tokens form one shell argument; parser uses lexer `HasSpaceBefore` boundaries.

### Expressions

```ebnf
expr            = logical_or ;
logical_or      = logical_and { "||" logical_and } ;
logical_and     = equality { "&&" equality } ;
equality        = comparison { ("==" | "!=") comparison } ;
comparison      = additive { ("<" | "<=" | ">" | ">=") additive } ;
additive        = multiplicative { ("+" | "-") multiplicative } ;
multiplicative  = unary { ("*" | "/" | "%") unary } ;

unary   = ("!" | "-") unary
        | ("++" | "--") unary
        | postfix ;

postfix = primary [ ("++" | "--") ] ;

primary = literal
        | identifier
        | decorator_ref
        | array_literal
        | object_literal ;

literal = string_literal | int_literal | float_literal | bool_literal ;

array_literal  = "[" [expr { "," expr }] [","] "]" ;
object_literal = "{" [object_field { "," object_field }] [","] "}" ;
object_field   = identifier ":" expr ;
```

## Interpolated strings

- Double-quoted and backtick strings allow decorator interpolation.
- Single-quoted strings are literal and do not interpolate.

Examples:

```opal
echo "Deploying @var.service"
echo '@var.service'
echo `HOME=@env.HOME`
```

## Execution mode validation

Parser validation supports two modes:

- Script mode: full language
- Command mode: definition-only mode that rejects top-level shell commands and top-level function call execution

## Language boundary

Opal metaprogramming wraps shell and decorator execution.

- Metaprogramming constructs build the plan and expand scripts.
- Shell commands and execution decorators perform runtime work.

That separation is the core language contract.
