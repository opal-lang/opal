# Devcmd Language Syntax Guide

A declarative DSL for defining CLI tools from command definitions

---

## Core Mental Model

Devcmd separates **language structure** from **shell command content**:

### Language Structure (Newlines as Boundaries)
- **Variable definitions**: `var PORT = 8080` or `var ( PORT = 8080 )`
- **Command definitions**: `build: go build ./cmd`
- **Block boundaries**: `{ ... }`
- **Statement separation**: Each line is a distinct Devcmd statement

### Shell Command Content (POSIX/Shell Compliant)
- **Command text**: Must be valid POSIX shell syntax
- **Shell operators**: `;`, `&&`, `||`, `|`, `>`, `<`, `&` all supported
- **Shell constructs**: Variables, command substitution, conditionals, loops
- **Generated as**: `exec.Command("sh", "-c", "your-command-here")`

**Mental Model**:
- **Newlines** = Devcmd statement boundaries (except when escaped)
- **Everything else** = Shell command content

---

## **CRITICAL: Newline Rules**

### **Newlines Create Multiple Commands**
**This is the most important rule in Devcmd**: Newlines create separate command statements everywhere in the language, **except when explicitly escaped with backslash**.

```devcmd
// ✅ Multiple commands - each line is a separate command
deploy: {
    npm run build     // Command 1
    npm test          // Command 2
    kubectl apply     // Command 3
}

// ✅ Single command with shell operators
deploy: npm run build && npm test && kubectl apply

// ✅ Pattern with multiple commands per branch
backup: @when(ENV) {
    production: {
        pg_dump mydb > backup.sql    // Command 1
        aws s3 cp backup.sql s3://   // Command 2
        rm backup.sql                // Command 3
    }
    staging: echo "staging backup"  // Single command
}

// ✅ Block decorator with multiple commands
services: @parallel {
    npm run api       // Command 1 (runs concurrently)
    npm run worker    // Command 2 (runs concurrently)
    npm run ui        // Command 3 (runs concurrently)
}
```

### **Line Continuation with Backslash-Newline**
Following POSIX shell convention, a backslash immediately followed by a newline acts as line continuation - both characters are removed and the lines are joined:

```devcmd
// ✅ Single command (line continuation)
build: npm run build && \
       npm run test && \
       npm run package

// ✅ Equivalent to: npm run build && npm run test && npm run package

// ✅ Line continuation in complex commands
deploy: kubectl apply -f k8s/production/ && \
        kubectl rollout status deployment/api && \
        kubectl get pods
```

**Line Continuation Rules**:
- The `\` must be the **very last character** before the newline
- **No spaces or tabs** allowed after the `\`
- Both `\` and newline are completely removed from the input
- Following whitespace on the next line is preserved
- Lines are joined with a single space character

**Examples:**
```devcmd
// ✅ Correct line continuation
server: docker run -d \
        --name myapp \
        --port 3000:3000 \
        myapp:latest

// ❌ Invalid - space after backslash
server: docker run -d \
        --name myapp     // This creates TWO commands

// ❌ Invalid - tab after backslash
server: docker run -d \
        --name myapp     // This creates TWO commands
```

### **Semicolons Are Shell Operators, Not Statement Separators**
```devcmd
// ✅ Shell operators within single command
build: echo "Starting"; npm install; echo "Done"  // ONE command with shell operators

// ✅ Multiple commands (newlines)
build: {
    echo "Starting"   // Command 1
    npm install       // Command 2
    echo "Done"       // Command 3
}
```

### **Newline Rules Apply EVERYWHERE**
The newline rule is consistent across all Devcmd constructs:

**Top-level commands:**
```devcmd
build: npm run build    // Command 1
test: npm test          // Command 2
deploy: kubectl apply   // Command 3
```

**Command blocks:**
```devcmd
deploy: {
    npm run build       // Command 1
    npm test            // Command 2
    kubectl apply       // Command 3
}
```

**Pattern branches:**
```devcmd
deploy: @when(ENV) {
    production: {
        kubectl config use-context prod  // Command 1
        kubectl apply -f k8s/prod/       // Command 2
        kubectl rollout status           // Command 3
    }
    staging: kubectl apply -f k8s/staging/  // Single command
}
```

**Block decorators:**
```devcmd
services: @timeout(30s) {
    docker-compose up -d    // Command 1
    sleep 5                 // Command 2
    curl localhost:3000     // Command 3
}

parallel-services: @parallel {
    npm run api             // Command 1 (concurrent)
    npm run worker          // Command 2 (concurrent)
    npm run ui              // Command 3 (concurrent)
}
```

**Nested decorators:**
```devcmd
deploy: @timeout(5m) {
    echo "Starting deployment"  // Command 1
    @retry(attempts = 3) {
        kubectl apply -f k8s/   // Command 1 inside retry
        kubectl rollout status  // Command 2 inside retry
    }
    echo "Deployment complete"  // Command 2
}
```

**Line continuations work in all contexts:**
```devcmd
// ✅ Line continuation in decorator blocks
deploy: @timeout(5m) {
    echo "Starting deployment..."        // Command 1
    kubectl apply -f k8s/production/ && \
    kubectl rollout status && \
    kubectl get pods                     // Command 2 (with continuation)
    echo "Deployment complete"           // Command 3
}

// ✅ Line continuation in pattern branches
backup: @when(ENV) {
    production: {
        pg_dump --verbose \              // Command 1 (with continuation)
                --no-owner \
                --format=custom \
                mydb > backup.sql
        aws s3 cp backup.sql \           // Command 2 (with continuation)
                  s3://backups/$(date +%Y%m%d)/
        rm backup.sql                    // Command 3
    }
}
```

---

## Command Structure and Hierarchy

### Commands Are Always Top-Level
Commands are the primary construct in Devcmd, similar to classes in Java:

```devcmd
// ✅ Valid - top-level commands
build: npm run build
test: npm test
deploy: kubectl apply -f k8s/

// ❌ Invalid - commands cannot be nested inside other commands
build: {
    test: npm test  // Commands cannot be defined inside other commands
}
```

### Command Types
```devcmd
// Regular command
build: npm run build

// Watch command (process management)
watch server: node app.js

// Stop command (cleanup)
stop server: pkill -f "node app.js"
```

---

## Syntax Sugar Rules

### Simple Command Sugar
**The only syntax sugar in Devcmd**: Simple commands without decorators get automatic braces.

```devcmd
// These are equivalent:
build: npm run build
build: { npm run build }

// These are equivalent:
watch server: node app.js
watch server: { node app.js }
```

### Block Decorators Require Explicit Braces
**Block decorators NEVER get syntax sugar** - they always require explicit block syntax when they wrap commands:

```devcmd
// ✅ Correct - explicit braces required for block decorators
server: @timeout(30s) { node app.js }

// ❌ Invalid - no syntax sugar for block decorators
server: @timeout(30s) node app.js
```

### Multi-line Commands Must Use Braces
```devcmd
// ✅ Correct - explicit braces for multi-line
deploy: {
    npm run build
    npm test
    kubectl apply -f k8s/
}

// ❌ Invalid - no automatic braces for multi-line
deploy: npm run build
        npm test
```

---

## Lexer Modes and Parsing Context

The lexer implements a **three-mode system** to handle the different parsing contexts in devcmd:

### LanguageMode (Top Level)
**When**: Top-level parsing and decorator parsing
**Recognizes**:
- Keywords: `var`, `watch`, `stop`
- Decorators: `@timeout`, `@parallel`, `@var`, etc.
- Language structure: `:`, `=`, `{`, `}`, `(`, `)`
- Literals: strings, numbers, durations

**Transition Rules**:
- `:` followed by non-structural content → **CommandMode**
- `{` for regular commands → **CommandMode**
- `{` for pattern decorators → **PatternMode**
- `@` → stay in **LanguageMode** (parse decorator)

### CommandMode (Inside Command Bodies)
**When**: After `:` or inside `{}` blocks, or parsing shell content in pattern branches
**Recognizes**:
- Shell text as complete units (tokenized as `SHELL_TEXT`)
- Line continuations (`\` + newline)
- Decorators (switches back to LanguageMode)
- Block boundaries

**Transition Rules**:
- `@` followed by identifier → **LanguageMode** (parse decorator)
- `}` → **LanguageMode** (exit command body) or **PatternMode** (exit pattern branch)
- `\n` in simple commands → **LanguageMode** (exit command)
- `\n` in pattern branches → **PatternMode** (return to pattern parsing)
- `\` + `\n` → continue in **CommandMode** (line continuation)

### PatternMode (Pattern Decorator Context)
**When**: Inside pattern decorator blocks (`@when`, `@try`, etc.)
**Recognizes**:
- Pattern identifiers (decorator-specific - see below)
- Structural tokens: `:`, `{`, `}`
- Nested decorators

**Transition Rules**:
- `:` after pattern identifier → **CommandMode** (parse shell content)
- `{` after pattern identifier → **CommandMode** (parse multi-command shell content)
- `}` → previous context via mode stack (exit pattern decorator)
- `@` → **LanguageMode** (nested decorator)

**Pattern Identifier Rules**:
- **@when**: Accepts any identifier for matching + `default` as wildcard
- **@try**: Only accepts `main` (required), `catch`, `finally`
- Each pattern decorator defines its own valid pattern identifier set

### Mode Transition Examples
```devcmd
// LanguageMode
deploy: {                    // : and { → CommandMode
    echo "deploying"         // CommandMode: shell text
    @parallel {              // @ → LanguageMode, parse @parallel, { → CommandMode
        npm run build        // CommandMode: shell text
        npm test            // CommandMode: shell text
    }                       // } → back to outer CommandMode
    echo "done"             // CommandMode: shell text
}                           // } → LanguageMode

// Pattern decorator mode transitions
backup: @when(ENV) {        // @ → LanguageMode, parse @when, { → PatternMode
    production: {            // PatternMode: identifier, : → CommandMode, { → CommandMode  
        pg_dump mydb         // CommandMode: shell text
        aws s3 cp backup.sql // CommandMode: shell text
    }                       // } → PatternMode (return to pattern parsing)
    staging: kubectl apply  // PatternMode: identifier, : → CommandMode, \n → PatternMode
    default: echo "unknown" // PatternMode: default wildcard, : → CommandMode, \n → PatternMode
}                           // } → LanguageMode (exit pattern decorator)

// Simple command with sugar
build: npm run build        // : → CommandMode, \n → LanguageMode

// Line continuation
build: npm run build && \   // : → CommandMode, \ + \n → stay in CommandMode
       npm test             // CommandMode: continuation, \n → LanguageMode
```

---

## Decorator Types and Parameter Syntax

Devcmd uses **Kotlin-style named parameters** for all decorators. Parameters can be specified by name or by position when unambiguous.

### Value Decorators (Inline Value Substitution)
Value decorators provide values for shell interpolation and are used inline within shell commands. They return values that are substituted into the command text at the exact location where they appear.

```devcmd
// @var(name) - Variable substitution from Devcmd variables
build: echo "Building on port @var(PORT)"
build: echo "Building on port @var(name = PORT)"  // Named parameter (equivalent)

// @env - Environment variable substitution with optional default
deploy: kubectl config use-context @env("KUBE_CONTEXT")
deploy: kubectl config use-context @env(variable = "KUBE_CONTEXT")  // Named parameter
deploy: kubectl config use-context @env(variable = "KUBE_CONTEXT", default = "local")  // With default

// Mixed parameter styles (positional first, then named)
setup: echo "API: @env("API_URL", default = "http://localhost:3000")"
```

**Value Decorator Characteristics**:
- Return values substituted into shell text at exact position
- Used inline within command content for variable expansion
- Support both positional and named parameters
- No braces required around the decorator itself
- Execute in place during shell command composition

**Standard Value Decorators**:
- `@var(name)` - Substitutes Devcmd variable value
- `@env(variable, default?)` - Substitutes environment variable with optional default

### Action Decorators (Command Execution)
Action decorators execute commands and return structured results that can be chained with shell operators. They perform actions rather than just providing values.

```devcmd
// @cmd(command) - Execute another command defined in the same CLI
deploy: @cmd(build) && kubectl apply -f k8s/
test: @cmd(build) && @cmd(lint) && npm test

// Action decorators can be standalone or chained
full-deploy: {
    @cmd(build)                    // Standalone action
    @cmd(test) && echo "Tests passed"  // Chained with shell operator
    kubectl apply -f k8s/
}

// Action decorators return CommandResult for chaining
check: @cmd(build) || (echo "Build failed" && exit 1)
```

#### Shell Chaining with ActionDecorators

ActionDecorators support full shell chaining semantics with `&&`, `||`, `|`, and `>>` operators:

```devcmd
// AND chaining - next runs only if previous succeeds
deploy: @cmd(build) && @cmd(test) && kubectl apply -f k8s/

// OR chaining - next runs only if previous fails  
verify: @cmd(test) || echo "Tests failed - check logs"

// Pipe chaining - stdout feeds to next command
logs: @cmd(get-logs) | grep ERROR | head -10

// File append - stdout appends to file
backup: @cmd(export-data) >> backup.txt
```

**Chaining Implementation**:
- **Interpreter Mode**: Uses shell for native chaining execution
- **Generator Mode**: Produces Go code with `CommandResult` logic that respects shell semantics
- **Mixed Chaining**: ActionDecorators can be chained with regular shell commands
- **Error Propagation**: Failed ActionDecorators properly terminate chains with `&&` or continue with `||`

**Generated Code Example**:
```go
// devcmd: @cmd(build) && @cmd(test) || echo "failed"
buildResult := executeBuild()
if buildResult.Success() {
    testResult := executeTest()
    if testResult.Failed() {
        return executeShellCommand("echo \"failed\"")
    }
    return testResult
}
return executeShellCommand("echo \"failed\"")
```

**Action Decorator Characteristics**:
- Execute commands and return structured CommandResult objects
- Can be standalone or chained with shell operators (&&, ||, |, >>)
- Support Go-native conditional execution logic in generated code
- Enable command composition and reusability
- Support both positional and named parameters
- Preserve shell chaining semantics across interpreter and generator modes

**Standard Action Decorators**:
- `@cmd(command)` - Execute another command defined in the same CLI

### Block Decorators (Command Wrapping)
Block decorators wrap commands inside their block with enhancement functionality and always require explicit braces. **Newlines within block decorators create multiple commands.**

```devcmd
// @parallel - Concurrent execution (each newline = separate concurrent command)
services: @parallel {
    npm run api      // Runs concurrently as Command 1
    npm run worker   // Runs concurrently as Command 2
    npm run ui       // Runs concurrently as Command 3
}

// @timeout - Execution timeout wraps all commands in block
api: @timeout(30s) {
    node server.js
}

// Multiple commands within timeout wrapper
deploy: @timeout(5m) {
    echo "Starting deployment"    // Command 1 (wrapped with 5m timeout)
    kubectl apply -f k8s/         // Command 2 (wrapped with 5m timeout)
    kubectl rollout status        // Command 3 (wrapped with 5m timeout)
}

// @retry - Retry wrapper applies to entire command sequence
deploy: @retry(3) {
    kubectl apply -f k8s/         // Command 1 (retried as unit)
    kubectl rollout status        // Command 2 (retried as unit)
}

// @debounce - Debounce execution with delay
watch: @debounce(500ms) {
    npm run build
}

// Multiple parameters with named syntax
backup: @retry(attempts = 3, delay = 1s) {
    rsync -av /data/ /backup/     // Command 1
    echo "Backup completed"       // Command 2
}
```

**Block Decorator Characteristics**:
- Always require explicit `{` `}` braces
- Wrap all commands within the block with enhancement functionality
- **Each newline creates a separate command within the decorator's wrapping scope**
- Support both positional and named parameters
- Apply enhancement behavior to all commands within the block

**Standard Block Decorators**:
- `@parallel` - Wraps commands to execute concurrently (each newline = separate goroutine)
- `@timeout(duration)` - Wraps command sequence with execution timeout
- `@retry(attempts, delay?)` - Wraps command sequence with retry logic on failure
- `@debounce(delay, pattern?)` - Wraps command sequence with debounce execution

### Nesting & Stacking Rules

**What's allowed**

* **Block→Block** (e.g. `@timeout{ @retry{ … } }`)
* **Block→Pattern** and **Pattern→Block** (wrappers outside apply to the selected branch inside)
* **Pattern→Pattern** (e.g. `@when{ production: @try{ … } }`)
* **Action** and **Value** decorators anywhere inside command text/branches

**What's not allowed**

* **Inline chaining of blocks** on the same level without braces (must nest, not chain):
  `@timeout(5m) @retry(3) { … }` ❌ (use `@timeout{ @retry{ … } }` instead).

**Evaluation order (authoritative)**

1. Parse to IR as a nested tree: `Wrapper{Inner CommandSeq}`, `Pattern{Branches map[string]CommandSeq}`, `CommandSeq{Steps []Step}`.
2. **CommandSeq nesting**: Each decorator operates on CommandSeq and returns CommandResult. A decorator is just another CommandSeq with nesting.
3. **Pattern selection** happens where it appears; the **selected branch CommandSeq** executes directly.
4. Within a `CommandSeq`, steps run in order; `&&`, `||`, `|`, `>>` apply inside each step/chain with the same semantics in interpreter and generated modes.

**Examples**

```devcmd
deploy: @timeout(5m) {               # outer wrapper
  @retry(attempts=3) {               # inner wrapper
    kubectl apply -f k8s/ && kubectl rollout status
  }
}

policy: @when(ENV) {                  # pattern selects branch
  prod:   @timeout(2m) { run-prod }   # wrapper inside selected branch
  default: @retry(2) { run-staging }
}

mix: @timeout(1m) {
  @when(MODE) {
    A: @retry(2) { step-a1 
                   step-a2 }
    B: { step-b1 | filter >> out.log } # plain block inside branch
  }
}
```

### Pattern Decorators (Conditional Branching)
Pattern decorators enable conditional execution based on variable values or execution flow. **Each pattern branch supports multiple commands separated by newlines.**

```devcmd
// @when - Conditional execution based on variable value
deploy: @when(MODE) {
    production: {
        kubectl config use-context prod       // Command 1
        kubectl apply -f k8s/prod/           // Command 2
        kubectl rollout status deployment/app // Command 3
        echo "Production deployment complete" // Command 4
    }
    staging: {
        kubectl config use-context staging   // Command 1
        kubectl apply -f k8s/staging/       // Command 2
        kubectl rollout status              // Command 3
    }
    development: echo "Skipping deployment in dev mode"  // Single command
    default: echo "Unknown mode: @var(MODE)"     // Wildcard pattern - single command
}

// @try - Exception handling with multiple commands per block
backup: @try {
    main: {
        echo "Starting backup process"       // Command 1
        pg_dump mydb > backup.sql           // Command 2
        aws s3 cp backup.sql s3://backups/  // Command 3
        echo "Backup uploaded successfully" // Command 4
    }
    catch: {
        echo "Backup failed: cleaning up..." // Command 1
        rm -f backup.sql                     // Command 2
        exit 1                               // Command 3
    }
    finally: {
        echo "Backup process completed"      // Command 1
        rm -f temp_files/*                   // Command 2
    }
}
```

**Pattern Decorator Characteristics**:
- Enable conditional execution and error handling
- Always require explicit `{` `}` braces
- Support pattern matching with identifiers and wildcards
- **Each pattern branch can contain multiple commands separated by newlines**
- Each command in a pattern branch executes sequentially

**Pattern Syntax**:
- **Identifier patterns**: Decorator-specific (e.g., `production`, `staging` for @when; `main`, `catch`, `finally` for @try)
- **Wildcard pattern**: `default` (only supported by @when, matches any value not explicitly handled)
- **Branch syntax**: `pattern: command` or `pattern: { commands }`

**Standard Pattern Decorators**:
- `@when(variable)` - Branch based on variable value
  - Accepts any identifier patterns + `default` wildcard
  - Example: `@when(ENV) { production: ..., staging: ..., default: ... }`
- `@try` - Exception handling with fixed semantic blocks
  - Only accepts: `main` (required), `catch`, `finally` (at least one of catch/finally required)
  - Example: `@try { main: ..., catch: ..., finally: ... }

### Nested Decorators (Explicit Block Syntax)
Decorators can be nested using explicit block syntax. **Newlines create multiple commands at every nesting level.**

```devcmd
// ✅ Correct - explicit nesting with multiple commands
deploy: @timeout(5m) {
    echo "Starting deployment process"  // Command 1
    @retry(attempts = 3) {
        kubectl apply -f k8s/           // Command 1 inside retry
        kubectl rollout status          // Command 2 inside retry
        kubectl get pods               // Command 3 inside retry
    }
    echo "Deployment completed"        // Command 2
    @parallel {
        kubectl logs deployment/api    // Command 1 concurrent
        kubectl logs deployment/worker // Command 2 concurrent
    }
    echo "All done"                   // Command 3
}

// ✅ Complex nested example
release: @parallel {
    @timeout(2m) {
        npm run build                  // Command 1 in timeout
        npm run test:unit             // Command 2 in timeout
    }
    @timeout(1m) {
        npm run test:e2e              // Command 1 in timeout
        npm run lint                  // Command 2 in timeout
    }
    @retry(attempts = 2) {
        npm run deploy                // Command 1 in retry
        npm run smoke-test            // Command 2 in retry
    }
}

// ❌ Invalid - no decorator chaining syntax
deploy: @timeout(5m) @retry(3) {
    npm run deploy
}
```

### Parameter Type System
Decorator parameters follow a strict type system with compile-time validation:

**Allowed parameter types:**
- **String literals**: `"value"`, `'value'`, `` `value` ``
- **Number literals**: `42`, `3.14`, `-100`
- **Duration literals**: `30s`, `5m`, `1h`, `500ms`
- **Boolean literals**: `true`, `false`
- **Variable references**: Must be identifiers referencing declared variables

**Type validation rules:**
```devcmd
var TIMEOUT = 30s            // Duration type
var RETRIES = 3              // Number type
var PROJECT_DIR = "/app"     // String type
var CONFIRM_DEPLOY = true    // Boolean type

// ✅ Valid - types match decorator expectations
deploy: @timeout(TIMEOUT) { npm run deploy }           // duration variable
build: @retry(RETRIES) { npm run build }               // number variable
server: @env("PORT", default = "3000") { node app.js } // string literals
test: @confirm(CONFIRM_DEPLOY) { npm test }            // boolean variable

// ❌ Invalid - type mismatches
deploy: @timeout(RETRIES) { npm run deploy }           // number to duration parameter
build: @retry(PROJECT_DIR) { npm run build }           // string to number parameter
```

**Disallowed constructs:**
```devcmd
// ❌ Invalid - nested function decorators in parameters
deploy: @timeout(@var(DURATION)) { npm run deploy }
build: @env(@var(ENV_NAME)) { npm run build }

// ❌ Invalid - complex expressions in parameters
server: @timeout(30 + 5) { node app.js }
```

**Correct approach - use direct variable references:**
```devcmd
var TIMEOUT = 35s
var ENV_VAR = "NODE_ENV"

// ✅ Correct - direct variable references
deploy: @timeout(TIMEOUT) { npm run deploy }
server: @env(ENV_VAR) { node app.js }
```

---

## Variable Definitions

### Individual Variable Syntax
```devcmd
var VARIABLE_NAME = value
```

### Grouped Variable Syntax (Go-like)
```devcmd
var (
    VARIABLE_NAME = value
    ANOTHER_VAR = value
)
```

### Variable Types
Variables must be one of exactly four supported types, automatically inferred from their assigned values:

```devcmd
// The four supported types:
var PORT = 8080           // Number type
var HOST = "localhost"    // String type (must be quoted)
var TIMEOUT = 30s         // Duration type
var DEBUG = true          // Boolean type (true or false)

// ❌ Invalid - no other types supported
var PATH = ./src          // Unquoted paths not supported - use "./src"
var DATA = [1, 2, 3]      // Arrays not supported
var CONFIG = { key: val } // Objects not supported
```

**Type System Rules:**
- **String**: Must be quoted with `"` or `'` or `` ` `` 
  - **Single quotes** `'text'`: Literal strings (no value decorator expansion)
  - **Double quotes** `"text"`: Interpolated strings (allows value decorator expansion)  
  - **Backticks** `` `text` ``: Multi-line strings (allows value decorator expansion)
- **Number**: Integer or decimal numbers (positive or negative)
- **Duration**: Number followed by time unit (`ns`, `us`, `ms`, `s`, `m`, `h`)
- **Boolean**: Exactly `true` or `false` (case-sensitive)

All other data types are unsupported and will result in compilation errors.

### String Interpolation

**Value decorator expansion in strings:**
```devcmd
var NAME = "World"
var PORT = 8080

// ✅ Single quotes - literal (no expansion)
literal: echo 'Hello @var(NAME) on port @var(PORT)'
// Output: Hello @var(NAME) on port @var(PORT)

// ✅ Double quotes - interpolated (value decorators expanded)  
interpolated: echo "Hello @var(NAME) on port @var(PORT)"
// Output: Hello World on port 8080

// ✅ Backticks - multi-line strings (value decorators expanded)
command: echo `Server @var(NAME) 
running on @var(PORT)`
// Output: Server World \n running on 8080

// ✅ Mixed content with shell operators preserved literally inside strings
complex: echo "Command: @var(NAME) | grep pattern" | actual-grep
// String content: "Command: World | grep pattern" 
// The | inside quotes is literal, the | outside is a shell pipe
```

**Supported value decorators in interpolated strings:**
- `@var(NAME)` - Variable expansion
- `@env(PATH)` - Environment variable expansion  
- Any other registered value decorators

### Variable References
Use `@var(NAME)` to reference variables in commands and `@env(variable)` to reference environment variables:

```devcmd
var PORT = 8080
var APP = "myapp"

// Using Devcmd variables
serve: go run main.go --port=@var(PORT) --name=@var(APP)

// Using environment variables with defaults
deploy: kubectl config use-context @env("KUBE_CONTEXT", default = "local")
build: echo "Building @var(APP) in @env("NODE_ENV", default = "development") mode"
```

---

## Statement Termination

### Newline Termination
All Devcmd statements are terminated by newlines, not semicolons:

```devcmd
// Variable definitions
var PORT = 8080
var HOST = "localhost"

// Grouped variable definitions
var (
    SRC = "./src"
    DIST = "./dist"
)

// Command definitions
build: go build ./cmd
watch dev: npm start
```

### Shell Command Content (Standard Shell Semantics)
```devcmd
// Semicolons in shell commands are shell operators, not language separators
build: echo "Step 1"; npm install; echo "Step 2"; npm run build

// Multiple commands (newlines create separate commands)
deploy: {
    npm run build && npm test || (echo "Build failed" && exit 1)  // Command 1
    echo "Success"; docker build -t myapp .                       // Command 2
}
```

---

## Execution Semantics

Understanding how commands execute and return results is essential for writing effective devcmd configurations.

### Sequential vs Chain Execution

Devcmd distinguishes between **sequential execution** (newline-separated) and **chain execution** (shell operators).

#### Sequential Execution (Newlines)

Commands separated by newlines execute in sequence, following standard shell semantics:

```bash
test: {
    @log("Step 1")
    @log("Step 2") 
    @log("Step 3")
}
```

**Live Output** (what user sees):
```
Step 1
Step 2
Step 3
```

**CommandResult** (return value):
```json
{
  "stdout": "Step 3\n",
  "stderr": "", 
  "exitCode": 0
}
```

**Key Point**: All commands execute and show live output, but CommandResult contains only the final command's result.

#### Chain Execution (Shell Operators)

Shell operators (`&&`, `||`, `|`) accumulate results differently:

```bash
test: @log("Starting") && echo "middle" && @log("Ending")
```

**Live Output**:
```
Starting
middle
Ending
```

**CommandResult**:
```json
{
  "stdout": "Starting\nmiddle\nEnding\n",
  "stderr": "",
  "exitCode": 0
}
```

**Key Point**: Shell chains accumulate output from all successful commands.

### Action Decorators

#### @log Decorator
The efficient equivalent of `echo` with structured logging:

```bash
info: @log("Info message", level="info")      # → stdout
error: @log("Error message", level="error")   # → stderr  
simple: @log("Hello world")                   # → stdout (default)
```

- Includes newlines like `echo`
- Error level messages go to stderr
- Supports color formatting: `@log("{green}Success!{/green}")`

#### @cmd Decorator
Execute other commands by reference for composition and reuse:

```bash
build: echo "Building..."
test: echo "Testing..."
ci: @cmd(build) && @cmd(test) && echo "CI complete"
```

Enables clean command composition without repetition.

### Variable Expansion

#### Shell Variables
Standard shell behavior - variables expand normally:
```bash
test: echo "User: $USER, Files: $(ls | wc -l)"
```

#### @var Variables  
CLI variables expand consistently throughout devcmd syntax:
```bash
var PROJECT = "myapp"

build: echo "Building @var(PROJECT)"     # → "Building myapp"
log: @log("Building @var(PROJECT)")      # → "Building myapp"
```

### Output and Error Handling

- **Live Output**: Users see all command output in real-time during execution
- **CommandResult**: Contains the final result for programmatic use
- **Exit Codes**: Determined by the last executed command
- **Error Propagation**: Failed commands stop `&&` chains, continue with `||` chains

---

## Execution Modes

Devcmd supports three execution modes that handle the same command definitions with different execution strategies:

### Interpreter Mode (Default)
Commands are executed directly by the devcmd engine at runtime.

```bash
# Direct execution
devcmd build
devcmd test
devcmd deploy
```

### Generated Mode (Phase 2 Roadmap)
Future: Command definitions will be compiled into standalone CLI binaries.

```bash
# Future capability - not yet implemented  
devcmd generate -f commands.cli --binary mycli

# Use generated CLI (future)
./mycli build
./mycli --help
```

### Plan Mode (Dry-Run)
Commands are analyzed and visualized without execution.

```bash
# Show execution plan
devcmd --dry-run deploy

# Generated CLI also supports dry-run
./mycli --dry-run deploy
```

**Plan Mode Output Example:**
```
🔍 Execution Plan for: deploy

deploy:
└─ @when(ENV) {production}
   ├─ echo "Deploying myapp to production"
   ├─ kubectl apply -f k8s/prod/
   └─ kubectl rollout status deployment/api
```

**Plan Mode Features:**
- Shows resolved variable values
- Displays conditional branch selection  
- Visualizes decorator behavior
- Safe exploration without side effects
- Tree-structured execution flow

For detailed information about execution modes, see [Execution Modes Documentation](execution_modes.md).

---

## Complete Examples

### Basic Project Structure
```devcmd
// Variables using only the four supported types
var SRC = "./src"            // String type (quoted)
var DIST = "./dist"          // String type (quoted)
var PORT = 3000              // Number type
var SERVER_TIMEOUT = 30s     // Duration type
var DEPLOY_TIMEOUT = 5m      // Duration type
var RETRY_COUNT = 3          // Number type
var DEBOUNCE_DELAY = 500ms   // Duration type

// Simple commands (with automatic braces)
build: npm run build
clean: rm -rf @var(DIST)
lint: eslint @var(SRC) --fix

// Action decorators for command chaining
test: @cmd(build) && npm test
check: @cmd(lint) && @cmd(test)
full-check: @cmd(build) && @cmd(lint) && @cmd(test) && echo "All checks passed"

// Environment variable usage in shell commands
status: echo "Running in @env("NODE_ENV", default = "development") mode on port @var(PORT)"

// Commands with decorators (explicit braces required)
server: @timeout(SERVER_TIMEOUT) {
    node app.js --port @var(PORT)
}

// Complex workflow with nested decorators and multiple commands
deploy: @timeout(DEPLOY_TIMEOUT) {
    echo "Starting deployment..."        // Command 1
    @parallel {
        npm run build                    // Command 1 (concurrent)
        npm run test                     // Command 2 (concurrent)
    }
    echo "Build and test completed"      // Command 2
    @retry(attempts = RETRY_COUNT, delay = 2s) {
        kubectl apply -f k8s/            // Command 1 (retried)
        kubectl rollout status deployment/app  // Command 2 (retried)
    }
    echo "Deployment complete"           // Command 3
}

// Process management with multiple commands
watch dev: @debounce(DEBOUNCE_DELAY) {
    echo "File changed, rebuilding..."   // Command 1
    npm run build:watch                  // Command 2
    echo "Build complete"                // Command 3
}

stop dev: pkill -f "npm.*build:watch"

// Line continuation examples
setup: docker run -d \
       --name myapp \
       --port 3000:3000 \
       --env NODE_ENV=production \
       myapp:latest

cleanup: kubectl delete deployment myapp && \
         kubectl delete service myapp && \
         echo "Cleanup complete"
```

### Frontend Development Example with Pattern Decorators
```devcmd
var (
    NODE_ENV = "development"     // String type
    WEBPACK_MODE = "development" // String type
    API_URL = "http://localhost:3000"  // String type
    BUILD_TIMEOUT = 2m           // Duration type
    MODE = "development"         // String type for pattern matching
)

// Simple commands
install: npm install
clean: rm -rf dist node_modules

// Development server with conditional behavior and multiple commands
dev: @when(MODE) {
    production: @timeout(BUILD_TIMEOUT) {
        echo "Building for production..."                    // Command 1
        NODE_ENV=production webpack --mode production        // Command 2
        echo "Starting production server..."                 // Command 3
        serve -s dist -l @var(PORT)                         // Command 4
    }
    development: @timeout(30s) {
        echo "Starting development server..."                // Command 1
        NODE_ENV=@env("NODE_ENV") webpack serve --mode @var(WEBPACK_MODE) --hot  // Command 2
    }
    default: echo "Unknown mode: @var(MODE)"
}

// Build process with error handling and multiple commands per branch
build: @try {
    main: @timeout(BUILD_TIMEOUT) {
        echo "Building for production..."                    // Command 1
        npm run clean                                        // Command 2
        NODE_ENV=production webpack \                        // Command 3 (with continuation)
                              --mode production \
                              --optimize-minimize
        npm run optimize                                     // Command 4
        echo "Build complete"                                // Command 5
    }
    catch: {
        echo "Build failed - cleaning up..."                // Command 1
        rm -rf dist                                          // Command 2
        npm run clean:cache                                  // Command 3
        exit 1                                              // Command 4
    }
    finally: {
        echo "Build process finished"                        // Command 1
        npm run cleanup                                      // Command 2
    }
}

// Testing with parallel execution and multiple commands
test: @parallel {
    @retry(attempts = 2) {
        echo "Running unit tests..."     // Command 1 (retried)
        npm run test:unit                // Command 2 (retried)
    }
    @retry(attempts = 2) {
        echo "Running e2e tests..."      // Command 1 (retried)
        npm run test:e2e                 // Command 2 (retried)
    }
    npm run lint                         // Command 3 (concurrent)
}
```

### DevOps Deployment Example
```devcmd
var (
    CLUSTER = "production"       // String type
    NAMESPACE = "myapp"          // String type
    IMAGE_TAG = "latest"         // String type
    DEPLOY_TIMEOUT = 10m         // Duration type
    ROLLBACK_RETRIES = 2         // Number type
    ENVIRONMENT = "production"   // String type for pattern matching
)

// Environment-specific deployment with multiple commands per pattern
deploy: @when(ENVIRONMENT) {
    production: @timeout(DEPLOY_TIMEOUT) {
        echo "Deploying to production cluster..."           // Command 1
        kubectl config use-context @env("PROD_KUBE_CONTEXT") // Command 2
        kubectl create namespace @var(NAMESPACE) \           // Command 3 (with continuation)
                --dry-run=client \
                -o yaml | kubectl apply -f -
        @retry(attempts = 3, delay = 30s) {
            kubectl apply -f k8s/prod/ -n @var(NAMESPACE)    // Command 1 (retried)
            kubectl rollout status deployment/api \         // Command 2 (retried, with continuation)
                    -n @var(NAMESPACE)
        }
        kubectl get pods -n @var(NAMESPACE)                  // Command 4
        echo "Production deployment successful"              // Command 5
    }
    staging: @timeout(5m) {
        echo "Deploying to staging cluster..."              // Command 1
        kubectl config use-context \                        // Command 2 (with continuation)
                @env("STAGING_KUBE_CONTEXT", default = "staging")
        kubectl apply -f k8s/staging/ -n @var(NAMESPACE)     // Command 3
        kubectl rollout status deployment/api -n @var(NAMESPACE)  // Command 4
        echo "Staging deployment complete"                   // Command 5
    }
    development: echo "Skipping deployment in development mode"
    default: {
        echo "Unknown environment: @var(ENVIRONMENT)"       // Command 1
        echo "Valid environments: production, staging, development"  // Command 2
        exit 1                                              // Command 3
    }
}

// Log monitoring with error handling and multiple commands
logs: @try {
    main: {
        echo "Connecting to cluster..."                      // Command 1
        kubectl config current-context                       // Command 2
        kubectl logs -f deployment/api -n @var(NAMESPACE)    // Command 3
    }
    catch: {
        echo "Failed to access logs - check cluster connection"  // Command 1
        kubectl cluster-info                                 // Command 2
        kubectl get pods -n @var(NAMESPACE)                  // Command 3
    }
}

// Rollback with retry and multiple verification commands
rollback: @retry(attempts = ROLLBACK_RETRIES, delay = 10s) {
    echo "Starting rollback process..."                      // Command 1
    kubectl rollout undo deployment/api -n @var(NAMESPACE)   // Command 2
    kubectl rollout status deployment/api \                 // Command 3 (with continuation)
            -n @var(NAMESPACE)
    kubectl get pods -n @var(NAMESPACE)                      // Command 4
    echo "Rollback completed successfully"                   // Command 5
}

// Cleanup with confirmation and multiple cleanup commands
cleanup: @when(ENVIRONMENT) {
    production: {
        echo "WARNING: This will delete production resources!"  // Command 1
        read -p "Type 'DELETE' to confirm: " confirm           // Command 2
        [ "$confirm" = "DELETE" ] && \                         // Command 3 (with continuation)
                kubectl delete deployment --all -n @var(NAMESPACE)
        [ "$confirm" = "DELETE" ] && \                         // Command 4 (with continuation)
                kubectl delete service --all -n @var(NAMESPACE)
        [ "$confirm" = "DELETE" ] && \                         // Command 5 (with continuation)
                echo "Production resources deleted"
    }
    default: {
        kubectl delete deployment --all -n @var(NAMESPACE)      // Command 1
        kubectl delete service --all -n @var(NAMESPACE)         // Command 2
        echo "Resources cleaned up"                             // Command 3
    }
}
```

---

## Key Design Principles

1. **Commands are top-level constructs** - never nested inside other commands
2. **Newlines create multiple commands** - except when escaped with backslash-newline (`\` + `\n`)
3. **Line continuation follows POSIX rules** - backslash must be the last character before newline
4. **Minimal syntax sugar** - only simple commands get automatic braces
5. **Block decorators require explicit braces** - they wrap commands with enhancement functionality
6. **Shell semantics preserved** - semicolons, pipes, etc. work as expected in shell
7. **Clear mode boundaries** - LanguageMode for structure, CommandMode for shell content
8. **Kotlin-style parameters** - named and positional parameters for all decorators
9. **Newline termination** - all statements terminated by newlines, not semicolons
10. **Four-type system** - variables must be string, boolean, number, or duration only
11. **Type-safe decorator parameters** - decorator parameters have type requirements validated at compile-time
12. **No nested function decorators** - only primitive types and identifiers allowed in parameters
13. **Consistent newline behavior** - newlines create commands everywhere, with only backslash-newline as exception
14. **Specialized decorator types** - Value decorators for inline substitution, Action decorators for command execution
15. **Go-native chaining** - Action decorators return structured results for reliable operator chaining
16. **Unified execution modes** - Same command definitions work across interpreter, generated, and plan modes
17. **Command composition** - Action decorators enable reusable command building blocks

This design keeps the language simple, predictable, and shell-friendly while providing powerful process management, workflow capabilities, and reliable command composition.
