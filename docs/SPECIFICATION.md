# Devcmd Language Specification

A unified declarative DSL where everything is a decorator

---

## Core Mental Model: Everything is a Decorator

Devcmd has a beautifully simple architecture: **everything is a decorator**. There are no special cases, no complex parsing modes, and no different execution paths.

### The Two Decorator Patterns

1. **Value Decorators** - inject values at specific locations
   - Examples: `@var(NAME)`, `@env(API_URL)` 
   - Usage: Inline within shell commands for dynamic substitution
   - Behavior: Return string values for shell interpolation

2. **Execution Decorators** - execute or wrap units of work  
   - Examples: `@cmd(build)`, `@retry(3) { ... }`, `@when(ENV) { ... }`
   - Usage: Standalone or with lambda-style blocks
   - Behavior: Execute commands with optional enhancement/control flow

### Shell Syntax as Decorator Sugar

**The key insight**: All shell syntax becomes `@shell` decorators during parsing.

```devcmd
// What users write (clean, natural syntax)
deploy: {
    echo "Starting deployment"
    npm run build
    kubectl apply -f k8s/
}

// What the parser creates internally  
deploy: {
    @shell("echo \"Starting deployment\"")
    @shell("npm run build")
    @shell("kubectl apply -f k8s/")
}
```

The `@shell` decorator is a first-class execution decorator that enables clean, shell-like syntax while maintaining the unified decorator architecture underneath.

---

## Language Structure

### Newlines Create Decorator Boundaries

**This is the most important rule**: Newlines separate commands, and each command becomes a decorator.

```devcmd
// Multiple commands (newlines = separate @shell decorators)
build: {
    npm install     // Command 1 → @shell("npm install")
    npm run build   // Command 2 → @shell("npm run build") 
    npm test        // Command 3 → @shell("npm test")
}

// Single command with shell operators (operators split into separate decorators)
quick: npm install && npm run build && npm test
// Becomes: @shell("npm install") && @shell("npm run build") && @shell("npm test")
```

### Shell Operators Split Decorator Units

> **🔑 Key Rule: Newline = fail-fast step; `;` = best-effort continue**

Shell operators like `&&`, `||`, `|`, `;` create separate decorator units with standard shell precedence:

```devcmd
// User syntax: Mixed shell and decorators
complex: echo "Starting" && @cmd(build) && @log("Complete") || @parallel { cleanup }

// Internal representation: Each part becomes appropriate decorator
complex: @shell("echo \"Starting\"") && @cmd(build) && @log("Complete") || @parallel { cleanup }

// Shell precedence preserved (left-to-right evaluation)
fallback: command1 || command2 && command3
// Becomes: (@shell("command1") || @shell("command2")) && @shell("command3")
```

**Operator Precedence & Behavior:**

| Operator | Meaning | Precedence | Example |
|----------|---------|------------|---------|
| `\|` | Pipe stdout | Highest | `cmd1 \| cmd2` |
| `&&` | Execute if previous succeeded | High | `cmd1 && cmd2` |
| `\|\|` | Execute if previous failed | High | `cmd1 \|\| cmd2` |
| `;` | Execute unconditionally (shell) | Low | `cmd1; cmd2` |
| Newline | Execute if previous succeeded (fail-fast) | Low | `cmd1`<br>`cmd2` |

**Critical**: Decorators complete their entire block before chain evaluation begins.

### Line Continuation

Use backslash-newline for line continuation (POSIX shell convention):

```devcmd
// Single command split across lines
deploy: kubectl apply -f k8s/production/ && \
        kubectl rollout status deployment/app && \
        kubectl get pods

// Becomes: Three separate chained @shell decorators
deploy: @shell("kubectl apply -f k8s/production/") && @shell("kubectl rollout status deployment/app") && @shell("kubectl get pods")
```

### Semicolon vs Newline Semantics

**CRITICAL DISTINCTION**: Semicolons and newlines have different error handling behavior:

#### **Semicolon (`;`) - Shell Behavior (Continue on Failure)**
```devcmd
@retry(3) { cmd1; cmd2; cmd3 }
// Becomes: @retry(3) { @shell("cmd1; cmd2; cmd3") }
// Traditional shell: all commands execute regardless of individual failures
// @retry succeeds if the overall shell sequence completes
```

#### **Newline - Fail-Fast Behavior**
```devcmd
@retry(3) {
    cmd1
    cmd2
    cmd3
}
// Becomes: @retry(3) { @shell("cmd1"); @shell("cmd2"); @shell("cmd3") }
// Structured execution: cmd2 only runs if cmd1 succeeds
// cmd3 only runs if both cmd1 AND cmd2 succeed
// @retry fails immediately on first command failure
```

#### **Comparison Example**
```devcmd
// Semicolon: "Run all commands, handle errors collectively"
deploy-tolerant: @retry(3) { 
    kubectl apply -f api/; kubectl apply -f worker/; kubectl apply -f ui/ 
}
// If api/ fails, worker/ and ui/ still deploy. Retry if overall sequence fails.

// Newline: "Fail immediately on any error"  
deploy-strict: @retry(3) {
    kubectl apply -f api/
    kubectl apply -f worker/
    kubectl apply -f ui/
}
// If api/ fails, worker/ and ui/ don't deploy. Retry the failed step.
```

**When to use each**:
- **Semicolon**: When you want shell-style "best effort" execution
- **Newline**: When you want structured fail-fast execution (recommended)

---

## Command Definitions

### Basic Command Structure

Commands are defined with a name followed by a colon and either a single command or a block:

```devcmd
// Simple command (becomes @shell decorator)
build: npm run build

// Command block (multiple @shell decorators)
deploy: {
    echo "Starting deployment"
    npm run build
    kubectl apply -f k8s/
    kubectl rollout status
}

// Mixed decorators and shell commands
full-deploy: {
    @log("Starting deployment")
    echo "Building application"
    npm run build
    @retry(3) {
        kubectl apply -f k8s/
        kubectl rollout status
    }
    @log("Deployment complete")
}
```

### Command Block Syntax

Command blocks use curly braces `{}` to group multiple commands:

```devcmd
// Sequential execution (each line = separate decorator)
setup: {
    npm install          // @shell("npm install")
    npm run build        // @shell("npm run build")
    npm test            // @shell("npm test")
}

// Shell operators within blocks (operators split decorators)
quick-check: {
    npm install && npm run build    // @shell("npm install") && @shell("npm run build")
    npm test || echo "Tests failed" // @shell("npm test") || @shell("echo \"Tests failed\"")
}
```

### Dual Mode Execution

Devcmd supports two execution modes that share the same decorator system:

#### **1. Command Mode** (Task Runner)
Files with named command definitions:
```devcmd
// commands.cli - Traditional task runner mode
var ENV = "development"

build: npm run build

deploy: @when(ENV) {
    production: kubectl apply -f k8s/prod/
    staging: kubectl apply -f k8s/staging/
}

test: @retry(3) {
    npm test
    npm run integration
}
```

```bash
devcmd build                   # Execute 'build' command
devcmd deploy                  # Execute 'deploy' command  
devcmd test                    # Execute 'test' command
```

#### **2. Script Mode** (Direct Execution)
Files with commandless execution (no named commands):
```devcmd
#!/usr/bin/env devcmd
// deploy-prod - Direct executable script
var ENV = "production"
var TIMEOUT = 30s

echo "Starting deployment to @var(ENV)"

@timeout(TIMEOUT) {
    echo "Building application"
    npm run build
    
    @retry(3) {
        kubectl apply -f k8s/prod/
        kubectl rollout status deployment/app
    }
}

echo "Deployment complete"
```

```bash
# Direct execution with shebang
chmod +x deploy-prod
./deploy-prod

# Or via devcmd
devcmd deploy-prod
```

#### **Migration Path**
Scripts naturally evolve from simple to organized:
```devcmd
// Start as script (deploy-staging)
echo "Deploy to staging"
kubectl apply -f k8s/staging/

// Evolve to commands (commands.cli)
deploy-staging: {
    echo "Deploy to staging"
    kubectl apply -f k8s/staging/
}

deploy-production: {
    echo "Deploy to production"
    kubectl apply -f k8s/prod/
}
```

#### **3. Advanced: Commands Within Scripts**
Scripts can define local commands and reference them with `@cmd()` for powerful composition:

```devcmd
#!/usr/bin/env devcmd
// complex-deploy - Script with internal command organization
var ENV = "production"
var REPLICAS = 3

// Local command definitions
build-image: {
    echo "Building Docker image"
    docker build -t myapp:@var(VERSION) .
    docker push myapp:@var(VERSION)
}

deploy-k8s: {
    kubectl apply -f k8s/
    kubectl scale deployment/app --replicas=@var(REPLICAS)
    kubectl rollout status deployment/app
}

cleanup-old: {
    kubectl delete pods --field-selector=status.phase=Succeeded
    docker image prune -f
}

// Script execution starts here
echo "Starting complex deployment to @var(ENV)"

@cmd(build-image)              // Execute local 'build-image' command

@retry(3) {
    @cmd(deploy-k8s)           // Execute local 'deploy-k8s' command
}

@parallel {
    @cmd(cleanup-old)          // Execute local 'cleanup-old' command
    @log("Deployment metrics logged")
}

echo "Complex deployment complete"
```

**Benefits of this pattern:**
- **Reusable logic**: Define once, reference multiple times with `@cmd()`
- **Clear organization**: Separate command definitions from execution flow
- **Powerful composition**: Mix local commands with decorators and shell commands
- **Self-contained scripts**: Everything needed in one executable file

Both modes use the same decorator system, variables, and execution semantics - just different entry points and organizational patterns within the same unified architecture.

## Plan-Then-Execute Workflows

Devcmd supports generating resolved execution plans that can be executed deterministically:

### Plan Generation
```bash
# Quick plan (fast preview)
devcmd deploy --dry-run                    # Show what would execute

# Resolved plan (all values resolved)  
devcmd deploy --dry-run --resolve          # Resolve all decorators
devcmd deploy --dry-run --resolve > prod.plan  # Save to file
```

### Plan Execution
```bash
# Execute resolved plan
devcmd --execute prod.plan                 # Run the saved plan

# Works with scripts too
devcmd deploy-script --dry-run --resolve > script.plan
devcmd --execute script.plan
```

### Use Cases
- **Production deployments**: Generate plan in staging, execute in production
- **CI/CD pipelines**: Plan in build phase, execute in deploy phase
- **Audit compliance**: Review exact execution before running
- **Rollback scenarios**: Re-execute previous successful plans
- **Team coordination**: Share exact execution plans for review

### Example Workflow
```bash
# Development: Test with quick plan
devcmd deploy --dry-run

# Staging: Generate resolved plan
devcmd deploy --dry-run --resolve > deploy-v1.2.plan

# Review: Inspect the resolved plan
cat deploy-v1.2.plan

# Production: Execute the exact plan
devcmd --execute deploy-v1.2.plan
```

This enables reliable, auditable deployments where the exact execution is determined and frozen before running.

---

## Decorator Types and Syntax

### Value Decorators (Inline Substitution)

Value decorators inject values inline within shell commands:

```devcmd
var PORT = 3000
var API_URL = "http://localhost:8080"

// Variable substitution  
server: node app.js --port @var(PORT)

// Environment variable with default
deploy: kubectl config use-context @env("KUBE_CONTEXT", default = "local")

// Becomes internally:
server: @shell("node app.js --port @var(PORT)")     // @var resolves during execution
deploy: @shell("kubectl config use-context @env(\"KUBE_CONTEXT\", default = \"local\")")
```

**Standard Value Decorators:**
- `@var(name)` - Devcmd variable substitution
- `@env(variable, default?)` - Environment variable with optional default
- `@aws_secret(name)` - AWS Secrets Manager lookup (expensive operation)
- `@http_get(url)` - HTTP request for dynamic values (expensive operation)

### Execution Decorators (Command Execution)

Execution decorators perform actions or wrap other commands:

#### Simple Execution (No Block)
```devcmd
// Command reference
full-build: @cmd(build) && @cmd(test)

// Becomes: @cmd(build) && @cmd(test) (already decorators)
```

#### Block Execution (Lambda-Style)
```devcmd
// Retry with command block  
deploy: @retry(3) {
    kubectl apply -f k8s/
    kubectl rollout status
}

// Internally: @retry wraps multiple @shell decorators
deploy: @retry(3) {
    @shell("kubectl apply -f k8s/")
    @shell("kubectl rollout status")
}
```

#### Pattern Execution (Conditional Branches)
```devcmd
// Conditional execution
build: @when(ENV) {
    production: docker build -t app:prod .
    development: docker build -t app:dev .
    default: echo "Unknown environment"
}

// Internally: @when selects between @shell decorators
build: @when(ENV) {
    production: @shell("docker build -t app:prod .")
    development: @shell("docker build -t app:dev .")
    default: @shell("echo \"Unknown environment\"")
}
```

**Standard Execution Decorators:**
- `@cmd(command)` - Execute another command by reference
- `@retry(attempts, delay?)` - Retry command block on failure
- `@timeout(duration)` - Execute command block with timeout
- `@parallel` - Execute command block steps concurrently  
- `@when(variable)` - Conditional execution based on variable value
- `@try` - Exception handling with main/catch/finally pattern
- `@log(message, level?)` - Structured logging

---

## Variable System

### Variable Declaration
```devcmd
// Individual variables
var PORT = 3000
var ENV = "development"
var TIMEOUT = 30s
var DEBUG = true

// Grouped variables (Go-style)
var (
    API_URL = "http://localhost:8080"
    DB_URL = "postgres://localhost/mydb"
    RETRY_COUNT = 3
)
```

### Supported Types
Variables support exactly four types:
- **String**: `"quoted text"` (double quotes allow value decorator expansion)
- **Number**: `42`, `3.14`, `-100`
- **Duration**: `30s`, `5m`, `1h`, `500ms`  
- **Boolean**: `true`, `false`

### String Interpolation
```devcmd
var NAME = "myapp"
var PORT = 3000

// Double quotes: Value decorators expand
status: echo "App @var(NAME) running on port @var(PORT)"

// Single quotes: Literal text (no expansion)  
literal: echo 'App @var(NAME) running on port @var(PORT)'
```

### Variable Scoping
- **Global scope**: Variables declared at file level available everywhere
- **Command scope**: Variables maintain same values throughout command execution
- **Immutable values**: Variables cannot be modified after declaration (functional style)

---

## Execution Semantics

### Sequential vs Chain Execution

**Sequential (Newlines)**: Fail-fast execution
```devcmd
deploy: {
    @log("Step 1")    // Executes, shows output
    @log("Step 2")    // Executes only if Step 1 succeeds  
    @log("Step 3")    // Executes only if Step 1 AND Step 2 succeed
}
// Behavior: Stops immediately on first failure
// Result: Success only if ALL commands succeed
```

**Sequential (Semicolons)**: Shell-style execution
```devcmd
deploy: { @log("Step 1"); @log("Step 2"); @log("Step 3") }
// Behavior: All commands execute regardless of individual failures
// Result: Success if overall shell sequence completes
```

**Chain (Operators)**: Commands chain with shell operator semantics
```devcmd
deploy: @log("Starting") && echo "middle" && @log("Ending")
// User sees: All output live
// Result: Accumulated CommandResult from entire chain
```

### Decorator Nesting

Decorators can be nested using explicit block syntax:

```devcmd
// Explicit nesting (recommended)
deploy: @timeout(5m) {
    echo "Starting deployment"
    @retry(3) {
        kubectl apply -f k8s/
        kubectl rollout status
    }
    echo "Deployment complete"
}

// Internally: Nested decorator structure preserved
deploy: @timeout(5m) {
    @shell("echo \"Starting deployment\"")
    @retry(3) {
        @shell("kubectl apply -f k8s/")
        @shell("kubectl rollout status")
    }
    @shell("echo \"Deployment complete\"")
}
```

## Error Handling and Flow Control

### **Shell Operator Behavior**

Shell operators control execution flow based on success/failure:

```devcmd
// Success chaining
build: npm install && npm run build && npm test
// Stops on first failure, succeeds if all succeed

// Failure fallback  
deploy: kubectl apply -f k8s/ || echo "Deployment failed"
// Runs echo only if kubectl fails

// Mixed operators (standard shell precedence)
complex: cmd1 && cmd2 || cmd3    // ((cmd1 && cmd2) || cmd3)
```

### **Decorator Completion Model**

**Critical rule**: Decorators complete their entire block before chain evaluation:

```devcmd
@retry(3) {
    kubectl apply -f k8s/
    kubectl rollout status
} && echo "Deploy success" || echo "Deploy failed"

// Execution flow:
// 1. @retry attempts its block up to 3 times total
// 2. If kubectl apply fails on attempt 1, @retry tries the entire block again
// 3. If all 3 attempts fail, @retry itself fails
// 4. Since @retry failed, && is skipped and || executes "Deploy failed"
// 5. If any attempt succeeds, @retry succeeds and "Deploy success" executes
```

**Important**: @retry applies to the entire block as a unit. If `kubectl apply` fails, the whole block (including `kubectl rollout status`) is retried.

### **Practical Example: Semicolon vs Newline**

```devcmd
// Semicolon: "Best effort" deployment
deploy-tolerant: @retry(3) {
    kubectl apply -f api/; kubectl apply -f worker/; kubectl apply -f ui/
}
// If api/ fails, worker/ and ui/ still attempt to deploy
// Retry only happens if the entire shell command sequence fails

// Newline: "Strict" deployment  
deploy-strict: @retry(3) {
    kubectl apply -f api/
    kubectl apply -f worker/ 
    kubectl apply -f ui/
}
// If api/ fails, worker/ and ui/ never execute
// Retry happens immediately when any individual command fails
```

Choose semicolon when you want shell-style "continue on error" behavior, and newlines when you want structured fail-fast execution.

**Formatter & Lint Guardrails:**
- **Formatter**: Splits multi-operator chains across lines unless user writes `;`
- **Lint**: D001 enforces blocks for mixed `&&/||/|` outside decorator blocks (see [Clean Code Guidelines](CLEAN_CODE_GUIDELINES.md))

**Note**: Newline semantics imply `pipefail=on` for `|` operators unless overridden.

### **Error Propagation**

**Within decorator blocks**:
- Commands execute sequentially
- Block fails on first command failure (unless using `||`)
- Decorator's final status determines chain continuation

**Across decorator chains**:
- Each decorator is an atomic unit for chain evaluation
- Shell operators connect decorator results, not individual commands
- Nested decorators must complete before parent continues

---

## Complete Examples

### Basic Project Workflow
```devcmd
var (
    PORT = 3000
    ENV = "development"
    TIMEOUT = 30s
)

// Simple commands (become @shell decorators)
install: npm install
clean: rm -rf dist node_modules  
lint: eslint src/ --fix

// Command composition
test: @cmd(install) && npm test
check: @cmd(lint) && @cmd(test)

// Enhanced execution
server: @timeout(TIMEOUT) {
    echo "Starting server on port @var(PORT)"
    node app.js --port @var(PORT) --env @var(ENV)
}

// Conditional deployment
deploy: @when(ENV) {
    production: {
        echo "Deploying to production"
        docker build -t myapp:prod .
        kubectl apply -f k8s/prod/
    }
    staging: {
        echo "Deploying to staging"  
        docker build -t myapp:staging .
        kubectl apply -f k8s/staging/
    }
    default: echo "Skipping deployment in @var(ENV) mode"
}
```

### DevOps Pipeline
```devcmd
var (
    CLUSTER = "production"
    TIMEOUT = 10m
    RETRIES = 3
)

// Error handling with retry
deploy: @try {
    main: @timeout(TIMEOUT) {
        echo "Starting deployment to @var(CLUSTER)"
        kubectl config use-context @env("KUBE_CONTEXT")
        @retry(RETRIES) {
            kubectl apply -f k8s/
            kubectl rollout status deployment/app
        }
        kubectl get pods
    }
    catch: {
        echo "Deployment failed, rolling back"
        kubectl rollout undo deployment/app
        kubectl get pods
    }
    finally: echo "Deployment process completed"
}

// Parallel execution
services: @parallel {
    npm run api      // Runs concurrently
    npm run worker   // Runs concurrently  
    npm run ui       // Runs concurrently
}
```

---

## Architecture Benefits

This unified decorator architecture provides:

1. **Conceptual Simplicity**: Everything is a decorator - no special cases
2. **Clean Syntax**: Users write natural shell syntax, parser handles decoration
3. **Powerful Composition**: Lambda-style blocks enable functional composition
4. **Unified Execution**: Single execution model for all constructs
5. **Extensibility**: Adding new decorators requires no language changes
6. **Tool Integration**: LSP, formatting, and analysis tools work uniformly

The result is a language that feels like enhanced shell syntax but has the power and reliability of a structured execution engine underneath.