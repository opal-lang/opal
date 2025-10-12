---
title: "Opal Formal Grammar"
audience: "Language Reference"
summary: "EBNF grammar specification and syntax rules"
---

# Opal Formal Grammar

EBNF grammar specification for the Opal language.

## Grammar Notation

- `|` - alternation (or)
- `*` - zero or more
- `+` - one or more
- `?` - optional (zero or one)
- `()` - grouping
- `[]` - character class
- `""` - literal string

## Lexical Elements

### Identifiers

```ebnf
identifier = letter (letter | digit | "_")*
letter     = [a-zA-Z]
digit      = [0-9]
```

**Examples**: `deploy`, `apiUrl`, `PORT`, `serviceName`, `buildAndTest`

**Naming conventions** (not enforced by grammar):
- Variables/parameters: `camelCase` preferred
- Constants: `SCREAMING_SNAKE` common
- Commands: any valid identifier style

### Keywords

```ebnf
fun when if else for in try catch finally var
```

### Literals

```ebnf
string_literal   = '"' string_char* '"'
int_literal      = digit+
float_literal    = digit+ "." digit+
bool_literal     = "true" | "false"
duration_literal = duration_component+

duration_component = digit+ duration_unit
duration_unit      = "y" | "w" | "d" | "h" | "m" | "s" | "ms" | "us" | "ns"
```

**Duration examples**: `30s`, `5m`, `2h`, `1h30m`, `2d12h`

**Semantic Notes** (validation rules, not syntax):
- Components must be in descending order: `1h30m` ✓, `30m1h` ✗
- No duplicate units: `1h30m` ✓, `1h2h` ✗
- Integer values only: `1h30m` ✓, `1.5h` ✗

### Operators

```ebnf
arithmetic = "+" | "-" | "*" | "/" | "%"
comparison = "==" | "!=" | "<" | ">" | "<=" | ">="
logical    = "&&" | "||" | "!"
assignment = "=" | "+=" | "-=" | "*=" | "/=" | "%="
increment  = "++" | "--"
shell      = "|" | "&&" | "||" | ";"
```

### Delimiters

```ebnf
( ) { } [ ] , : . @
```

## Syntax Grammar

### Source File

```ebnf
source = declaration*

declaration = function_decl
            | var_decl
```

### Function Declarations

```ebnf
function_decl = "fun" identifier param_list? ("=" expression | block)

param_list = "(" (param ("," param)*)? ")"

param = identifier type_annotation? default_value?

type_annotation = ":" type_name

default_value = "=" expression

type_name = "String" | "Int" | "Float" | "Bool" | "Duration" | "Array" | "Map"
```

**Examples**:
```opal
fun deploy = kubectl apply -f k8s/
fun greet(name) = echo "Hello @var.name"
fun build(module, target = "dist") { ... }
fun deploy(env: String, replicas: Int = 3) { ... }
```

**Semantic Notes** (calling conventions):
- Positional: `@cmd.retry(3, 2s)`
- Named: `@cmd.retry(attempts=3, delay=2s)`
- Mixed: `@cmd.retry(3, delay=2s)`

All three forms are valid (Kotlin-style flexibility).

### Variable Declarations

```ebnf
var_decl = "var" (var_spec | "(" var_spec+ ")")

var_spec = identifier ("=" expression)?
```

**Examples**:
```opal
var ENV = @env.ENVIRONMENT
var PORT = 3000
var (
    apiUrl = @env.API_URL
    replicas = 3
)
```

### Statements

```ebnf
statement = var_decl
          | assignment
          | expression
          | if_stmt
          | when_stmt
          | for_stmt
          | try_stmt
          | block

assignment = identifier assign_op expression

assign_op = "=" | "+=" | "-=" | "*=" | "/=" | "%="
```

### Control Flow

```ebnf
if_stmt = "if" expression block ("else" (if_stmt | block))?

when_stmt = "when" expression "{" when_arm+ "}"

when_arm = pattern "->" (expression | block)

pattern = string_literal
        | pattern "|" pattern              # OR patterns
        | "r" string_literal               # Regex patterns
        | expression "..." expression      # Range patterns (three dots)
        | "else"                           # Catch-all

for_stmt = "for" identifier "in" expression block

try_stmt = "try" block ("catch" block)? ("finally" block)?
```

**Examples**:
```opal
if @var.ENV == "production" { ... }

when @var.ENV {
    "production" -> { ... }
    "staging" | "dev" -> { ... }
    else -> { ... }
}

for service in @var.SERVICES { ... }

try { ... } catch { ... } finally { ... }
```

### Expressions

```ebnf
expression = primary
           | unary_expr
           | binary_expr
           | call_expr
           | decorator_expr

primary = identifier
        | literal
        | "(" expression ")"
        | array_literal
        | map_literal

array_literal = "[" (expression ("," expression)*)? "]"

map_literal = "{" (map_entry ("," map_entry)*)? "}"

map_entry = (string_literal | identifier) ":" expression

unary_expr = ("!" | "-" | "++" | "--") expression

binary_expr = expression binary_op expression

binary_op = arithmetic | comparison | logical | shell

call_expr = "@cmd." identifier "(" argument_list? ")"

argument_list = argument ("," argument)*

argument = (identifier "=")? expression
```

**Argument forms** (all valid):
```opal
@cmd.retry(3, 2s)                    # Positional
@cmd.retry(attempts=3, delay=2s)     # Named
@cmd.retry(3, delay=2s)              # Mixed
```

### Decorators

```ebnf
decorator_expr = value_decorator | execution_decorator

value_decorator = "@" decorator_path decorator_args?

execution_decorator = "@" decorator_path decorator_args? block

decorator_path = identifier ("." identifier)*

decorator_args = "(" argument_list? ")"
```

**Value decorators**:
```opal
@env.PORT
@var.REPLICAS
@aws.secret.apiKey(auth=prodAuth)
```

**Execution decorators**:
```opal
@retry(attempts=3, delay=2s) { ... }
@timeout(duration=5m) { ... }
@parallel { ... }
```

### Variable Interpolation

```ebnf
interpolation = "@var." identifier terminator?
              | "@env." identifier terminator?
              | decorator_expr

terminator = "()"
```

**In strings and commands**:
```opal
echo "Deploying @var.service"
kubectl scale --replicas=@var.replicas deployment/@var.service
kubectl apply -f k8s/@var.service/
```

**Termination with `()`**:
```opal
echo "@var.service()_backup"  // Expands to "api_backup"
```

### Blocks

```ebnf
block = "{" statement* "}"
```

### Shell Commands

Shell commands are parsed as execution decorators internally:

```ebnf
shell_command = shell_token+

shell_token = identifier | string_literal | interpolation | shell_operator
```

**Parser transformation**:
```opal
// Source
npm run build

// Parser generates
@shell("npm run build")
```

## Operator Precedence

From highest to lowest:

1. `()` - grouping, function calls
2. `++`, `--` - increment, decrement
3. `!`, unary `-` - logical not, negation
4. `*`, `/`, `%` - multiplication, division, modulo
5. `+`, `-` - addition, subtraction
6. `<`, `>`, `<=`, `>=` - comparison
7. `==`, `!=` - equality
8. `&&` - logical and
9. `||` - logical or
10. `|` - pipe (shell)
11. `;` - sequence (shell)
12. `=`, `+=`, `-=`, etc. - assignment

## Whitespace and Comments

```ebnf
whitespace = " " | "\t" | "\n" | "\r"

comment = "//" [^\n]* "\n"
        | "/*" .* "*/"
```

**Whitespace significance**:
- Newlines separate statements (fail-fast semantics)
- Semicolons override newline semantics (continue on error)
- `HasSpaceBefore` token flag preserved for shell command parsing

## Semantic Rules

### Plan-Time vs Runtime

**Plan-time constructs** (deterministic):
- `for` loops - unroll to concrete steps
- `if`/`when` - select single branch
- `fun` - template expansion
- Variable declarations and assignments (in most blocks)

**Runtime constructs**:
- `try`/`catch` - path selection based on exceptions
- Execution decorators - modify command execution
- Shell commands - actual work execution

### Scope Rules

**Outer scope mutations allowed**:
- Regular blocks: `{ ... }`
- `for` loops
- `if`/`when` branches
- `fun` bodies

**Scope isolation** (read outer, mutations stay local):
- `try`/`catch`/`finally` blocks
- Execution decorator blocks (`@retry { ... }`, etc.)

### Type System

**Optional typing** (TypeScript-style):
- Variables untyped by default
- Function parameters can have type annotations
- Type checking at plan-time when types specified
- Future: `--strict-types` flag

**Type inference**:
- From literals: `var x = 3` → Int
- From defaults: `fun f(x = "hi")` → x is String
- From decorators: `@env.PORT` → String (can be converted)

## Plan File Format

Plans are JSON with the following structure:

```json
{
  "version": "1.0",
  "source_commit": "abc123...",
  "spec_version": "0.1.0",
  "compiler_version": "0.1.0",
  "plan_hash": "sha256:def456...",
  "steps": [
    {
      "id": 0,
      "decorator": "shell",
      "args": {
        "cmd": "kubectl apply -f k8s/"
      },
      "dependencies": []
    },
    {
      "id": 1,
      "decorator": "shell",
      "args": {
        "cmd": "kubectl scale --replicas=<1:sha256:abc123> deployment/app"
      },
      "dependencies": [0]
    }
  ],
  "values": {
    "REPLICAS": "<1:sha256:abc123>"
  }
}
```

**Hash placeholder format**: `<length:algorithm:hash>`
- `length` - character count of actual value
- `algorithm` - hash algorithm (sha256)
- `hash` - truncated hash for verification

## Decorator Plugin Interface

Decorators are loaded as Go plugins (`.so` files):

```go
// Decorator plugin interface (similar to database/sql)
type ValueDecorator interface {
    Plan(ctx Context, args []Param) (Value, error)
}

type ExecutionDecorator interface {
    Plan(ctx Context, args []Param, block Block) (Plan, error)
    Execute(ctx Context, plan Plan) error
}
```

**Loading at runtime:**
```bash
# CLI discovers decorators in:
~/.opal/decorators/*.so
./.opal/decorators/*.so
```

## Implementation Notes

### Parser Strategy
- Event-based parse tree (rust-analyzer style)
- Dual-path: Events → Plan (execution) or Events → AST (tooling)
- Resilient parsing with error recovery
- FIRST/FOLLOW sets for predictive parsing

### Performance Targets
- Lexer: >5000 lines/ms
- Parser: >3000 lines/ms (events)
- Plan generation: <10ms for typical scripts

### Error Recovery
- Synchronization points: `}`, `;`, newline
- Error nodes in parse tree for tooling
- Continue parsing after errors for better diagnostics
