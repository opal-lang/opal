---
title: "Opal Language Specification"
audience: "End Users & Script Authors"
summary: "Complete language guide with syntax, semantics, and real-world examples"
---

# Opal Language Specification

Deterministic task runner for operations and developer workflows.

> Opal automates *what happens after infrastructure exists* — deployment, validation, and operational workflows — without managing infrastructure state.

**See also**: [Formal Grammar](GRAMMAR.md) for EBNF syntax specification.

## The Gap

After infrastructure is provisioned, teams use shell scripts or Makefiles for:
- Deployments and migrations
- Health checks and restarts
- Build, test, release workflows
- Day-2 operational tasks

These are brittle, non-deterministic, and hard to audit. Opal fills this gap.

## Design Philosophy

**Outcome-focused, not state-driven:**

Opal queries reality, plans actions, and executes - without maintaining state files.

- **Reality is truth**: Query current state (does this instance exist? what's the current version?)
- **Plan from reality**: Generate execution plan based on what exists now
- **Execute the plan**: Perform the work to achieve outcomes
- **Contract verification**: Detect if reality changed between plan and execute

The **plan is your contract** - deterministic, verifiable, hash-based. This comes from contract-first development: define the agreement upfront, execute against it.

No state files to maintain or synchronize. Reality is the only source of truth.

## Core Principles

**Plans as execution contracts**: Resolved plans show exactly what will execute, with hash-based verification to catch changes between planning and execution.

**Decorator-based execution**: Everything that performs work becomes a decorator internally - no special cases. You write natural syntax, the parser converts to decorators.

```opal
// You write natural syntax
deploy: {
    echo "Starting deployment"
    npm run build
    kubectl apply -f k8s/
}

// Parser converts to execution decorators
deploy: {
    @shell("echo \"Starting deployment\"")
    @shell("npm run build")
    @shell("kubectl apply -f k8s/")
}
```

**Deterministic planning**: Same inputs always produce the same plan. Control flow resolves at plan-time.

**Scope semantics**: All blocks can read outer values; only language control blocks (`for`, `if`, `when`, `fun`) can mutate outer scope. Execution decorator blocks (`@retry`, `@timeout`, etc.) and `try/catch` blocks use scope isolation—mutations don't leak out. See [Variable Scope](#variable-scope) for details.

**Scope**: Operations and task running. We're proving the model here before extending to infrastructure provisioning.

## Mental Model

```
                    Direct Execution:
Source Code → Parse → Plan → Execute
              ↓       ↓       ↓
           Events  Actions  Work

                    Contract Execution (with plan file):
Source Code → Parse → Replan → Verify → Execute
              ↓       ↓        ↓         ↓
           Events  Actions  Match?    Work
                              ↓
                         Plan File
                        (Contract)
```

**Critical insight**: Plan files are NEVER executed directly. They are verification contracts.

**Contract execution always:**
1. Replans from current source and infrastructure state
2. Compares fresh plan against contract (hash verification)
3. Executes ONLY if they match, aborts if they differ

**Unlike Terraform** (which applies old plan to new state), Opal verifies current reality would still produce the same plan. This prevents executing stale or tampered plans.

## Error Handling Model

**Error precedence rule** (normative for all executors and decorators):

1. **`err != nil`** → Failure. Exit code becomes informational (logged but not used for control flow).
2. **`err == nil`** → Success if and only if `exitCode == 0`. Non-zero exit code with `nil` error is a failure.

**Typed errors** enable policy decisions:
- `RetryableError` - Decorator can retry (network timeout, rate limit)
- `TimeoutError` - Decorator exceeded time limit
- `CancelledError` - Context cancelled (user interrupt, deadline)

**Example:**
```opal
@retry(times=3) {
    curl https://api.example.com/health
}
```

If `curl` returns exit code 7 (connection failed) with `err == nil`, retry decorator sees non-zero exit and retries. If it returns `TimeoutError`, retry decorator can apply backoff strategy.

## Two Ways to Run

**Command mode** - organized tasks:
```opal
// commands.opl
fun install = npm install
fun test = npm test
fun deploy = kubectl apply -f k8s/
```
```bash
opal deploy    # Run specific task
```

**Script mode** - direct execution:
```opal
#!/usr/bin/env opal
echo "Deploying version $VERSION"
kubectl apply -f k8s/
```
```bash
./deploy-script    # Execute directly
```

## Line-by-Line Execution

How commands connect matters:

```opal
// Newlines = fail-fast (stop on first failure)
deploy: {
    npm run build
    npm test
    kubectl apply -f k8s/
}

// Semicolons = keep going (shell behavior)
setup: npm install; npm run build; npm test

// Shell operators = standard behavior with pipefail
check: npm run build && npm test || echo "Build failed"
logs: kubectl logs app | grep ERROR
```

**Operator precedence**: `|` (pipe) > `&&`, `||` > `;` > newlines

## Example: Deployment with Conditionals

```opal
deploy: {
    when @env.ENV {
        "production" -> {
            kubectl apply -f k8s/prod/
            @retry(attempts=3) { kubectl rollout status deployment/app }
        }
        else -> kubectl apply -f k8s/dev/
    }
}
```

### Command Definitions with `fun` (Template Functions)

`fun` enables **template-based code reuse** at plan-time. Command definitions can be parameterized and called with different arguments.

**IMPORTANT**: `fun` definitions must be at the top level. You cannot define `fun` inside `for` loops, `if` statements, or other control flow constructs. Define template functions first, then call them inside loops.

```opal
var MODULES = ["cli", "runtime"]

# ✅ CORRECT: Define template functions at top level
fun build_module(module) {
    @workdir(@var.module) {
        npm ci
        npm run build
    }
}

fun test_module(module, retries=2) {
    @workdir(@var.module) {
        @retry(attempts=@var.retries, delay=5s) { npm test }
    }
}

# ✅ CORRECT: Call templates inside loops
fun build_all {
    for module in @var.MODULES {
        @cmd.build_module(module=@var.module)
    }
}

fun test_all {
    for module in @var.MODULES {
        @cmd.test_module(module=@var.module, retries=3)
    }
}

# ❌ INCORRECT: Cannot define fun inside for loops
fun build_all {
    for module in @var.MODULES {
        fun build_@var.module {  # ERROR: fun inside for is not allowed
            npm run build
        }
    }
}
```

**Metaprogramming semantics**:
- **Plan-time expansion**: `for` loops and `@var.NAME` resolve during plan construction
- **Parameter binding**: All parameters resolve to concrete values when `@cmd.function_name()` is called
- **Template inlining**: `@cmd.fun_name(args...)` expands to the `fun` body with parameters substituted
- **Template expansion**: `@cmd.function_name()` calls expand function templates with parameters
- **Static command names**: After metaprogramming expansion, all command names are concrete identifiers
- **No runtime reflection**: All metaprogramming happens at plan-time, execution is deterministic

**Syntax forms**:
```opal
# Assignment form (concise one-liners)
fun deploy = kubectl apply -f k8s/
fun greet(name) = echo "Hello, @var.name!"

# Block form (multi-line)
fun complex {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
}

fun build_module(module, target="dist") {
    @workdir(@var.module) {
        npm ci
        npm run build --output=@var.target
    }
}
```

**Parameter types** (optional, TypeScript-style):

Type annotations are optional and enable plan-time type checking when specified:

```opal
# Untyped (simple, flexible)
fun greet(name) {
    echo "Hello @var.name"
}

# Typed (explicit validation)
fun deploy(env: String, replicas: Int = 3, timeout: Duration = 30s) {
    kubectl scale deployment/app --replicas=@var.replicas
}

# Mixed (practical)
fun build(module, target = "dist", verbose: Bool = false) {
    npm run build -- --target=@var.target
}
```

**Parameter syntax**:
- `name` - required, untyped (no validation)
- `name: String` - required, typed (validated at plan-time)
- `name = "default"` - optional, untyped (type inferred from default)
- `name: String = "default"` - optional, typed with explicit type

**Supported types**:
- `String` - text values
- `Int` - integer numbers
- `Float` - floating-point numbers
- `Bool` - true/false
- `Duration` - time durations (30s, 2h, 1d)
- `Array` - lists of values
- `Map` - key-value pairs

**Type checking**:
- Types are validated at plan-time when values are resolved
- Untyped parameters accept any value
- Type mismatches produce clear error messages before execution
- Future: `--strict-types` flag for requiring all parameters to be typed

**Command mode CLI flags**:

In command mode, parameters with defaults become CLI flags:

```opal
# Definition
fun deploy(env: String, replicas: Int = 3, timeout: Duration = 30s) {
    kubectl scale deployment/app --replicas=@var.replicas
}
```

```bash
# CLI usage
opal deploy production              # uses defaults: replicas=3, timeout=30s
opal deploy production --replicas=5 # override replicas
opal deploy production --timeout=60s --replicas=10
```

**Example expansion**:
```opal
# Template function definition
fun test_module(module) {
    @workdir(@var.module) { go test ./... }
}

# Usage in commands
test_all: {
    for module in ["cli", "runtime"] {
        @cmd.test_module(module=@var.module)
    }
}

# Expands at plan-time to:
# @workdir("cli") { go test ./... }
# @workdir("runtime") { go test ./... }
```

## Decorator Syntax

**For advanced patterns** (opaque handles, resource collections, memoization, composition), see [DECORATOR_GUIDE.md](DECORATOR_GUIDE.md).

Opal distinguishes between value decorators (return data) and execution decorators (perform work) with clear syntax patterns.

### **Value Decorators**

**Dot syntax for simple access:**
```opal
# Variables (namespaced for visibility)
kubectl scale --replicas=@var.REPLICAS deployment/app
psql @var.CONFIG.database.url
docker run -p @var.SERVICES[0]:3000 app

# Simple value decorators
kubectl create secret --token=@aws.secret.api_token
echo "Environment: @env.NODE_ENV"
curl -H "Authorization: Bearer @oauth.tokens.access_token"
```

**Optional parameters when needed:**
```opal
# Defaults and complex parameters
kubectl apply -f @env.MANIFEST_PATH(default="k8s/")
curl @aws.secret.api_key(region="us-west-2")
echo "Version: @env.APP_VERSION(default="latest")"
```

### **Execution Decorators**

**Always function syntax with blocks:**
```opal
@retry(attempts=3, delay=2s) { kubectl apply -f k8s/ }
@workdir("/tmp") { ls -la }
@timeout(30s) { npm test }
@parallel {
    echo "Task A"
    echo "Task B"
}
```

**Clear distinction:**
- **Value decorators**: Use dot syntax, return data for command arguments
- **Execution decorators**: Use function syntax, modify how commands execute

**See [Decorator Design Guide](DECORATOR_GUIDE.md)** for advanced patterns: opaque handles, resource collections, memoization, and composition best practices.

## Variables

### Variable Access and Interpolation

**Opal uses `@var.NAME` syntax for all variable access.** Variables are plan-time values that get expanded during plan generation.

```opal
var service = "api"
var replicas = 3

// In command arguments (unquoted)
kubectl scale --replicas=@var.replicas deployment/@var.service

// In strings (quoted)
echo "Deploying @var.service with @var.replicas replicas"

// In paths
kubectl apply -f k8s/@var.service/

// Terminate decorator with () if followed by ASCII with no spaces
echo "@var.service()_backup"  // Expands to "api_backup"
```

**Plan-time expansion:** The parser expands `@var.NAME` during plan generation into literal values:

```opal
// Source code
for service in ["api", "worker"] {
    kubectl apply -f k8s/@var.service/
}

// Expands to plan
kubectl apply -f k8s/api/
kubectl apply -f k8s/worker/
```

**Shell variables (`${}`) are NOT Opal syntax.** If you need actual shell environment variables, they stay inside shell commands and are evaluated by the shell at runtime, not by Opal:

```opal
// Shell variable (rare - evaluated by shell at runtime)
@shell("echo Current user: $USER")

// Opal variable (common - expanded by Opal at plan-time)
echo "Deploying to: @var.ENV"
```

### Variable Declaration

Pull values from real sources:

```opal
// Required variables (error if not set)
var DATABASE_URL = @env.DATABASE_URL
var API_KEY = @env.API_KEY

// Optional variables (use default if not set)
var ENV = @env.ENVIRONMENT(default="development")
var PORT = @env.PORT(default=3000)
var DEBUG = @env.DEBUG(default=false)
var TIMEOUT = @env.DEPLOY_TIMEOUT(default=30s)

// Arrays and maps
var SERVICES = ["api", "worker", "ui"]
var CONFIG = {
    "database": @env.DATABASE_URL,  // Required
    "redis": @env.REDIS_URL(default="redis://localhost:6379")  // Optional
}

// Go-style grouped declarations
var (
    API_URL = @env.API_URL(default="https://api.dev.com")
    REPLICAS = @env.REPLICAS(default=1)
    SERVICES = ["api", "worker"]
)
```

**Environment variable behavior:**
- `@env.VAR` without `default` → Error if not set (required)
- `@env.VAR(default="value")` → Use default if not set (optional)
- Defaults can be typed: `default=3000` (int), `default=false` (bool), `default=30s` (duration)

**Types**: `String | Bool | Int | Float | Duration | Array | Map`. 

Type checking is optional (TypeScript-style):
- Variables are untyped by default (flexible, inferred from values)
- Function parameters can have optional type annotations
- Type errors are caught at plan-time when types are specified
- Future: `--strict-types` flag for requiring explicit types

### Identifier Names

Variable names, command names, and value decorator and execution decorator names follow ASCII identifier rules for fast tokenization:

```opal
// Valid identifiers - start with letter, then letters/numbers/underscores
var apiUrl = "https://api.example.com"
var PORT = 3000
var service_name = "api-gateway"
var deployToProduction = true

// Command names follow same rules
deployToProduction: kubectl apply -f prod/
check_api_health: curl /health
buildAndTest: npm run build && npm test
```

**Identifier rules**:
- Must start with letter: `[a-zA-Z]`
- Can contain letters, numbers, underscores: `[a-zA-Z0-9_]*`
- Case-sensitive: `myVar` ≠ `MyVar` ≠ `MYVAR`
- No hyphens to avoid parsing ambiguity with minus operator
- ASCII-only for optimal performance

**Supported naming styles**:
- `camelCase` - common in JavaScript/Java
- `snake_case` - common in Python/Go
- `PascalCase` - common for types/commands
- `SCREAMING_SNAKE` - common for constants

### Duration Format

Duration literals use human-readable time units common in operations:

```opal
// Simple durations
var TIMEOUT = 30s           // 30 seconds
var RETRY_DELAY = 5m        // 5 minutes
var DEPLOY_WINDOW = 2h      // 2 hours
var RETENTION = 7d          // 7 days
var BACKUP_CYCLE = 1w       // 1 week
var LICENSE_EXPIRY = 2y     // 2 years

// Compound durations (high to low order)
var SESSION_TIMEOUT = 1h30m     // 1 hour 30 minutes
var MAINTENANCE_WINDOW = 2d12h  // 2 days 12 hours
var GRACE_PERIOD = 5m30s        // 5 minutes 30 seconds
var API_TIMEOUT = 1s500ms       // 1 second 500 milliseconds
```

**Supported units** (in descending order):
- `y` - years
- `w` - weeks
- `d` - days
- `h` - hours
- `m` - minutes
- `s` - seconds
- `ms` - milliseconds
- `us` - microseconds
- `ns` - nanoseconds

**Compound duration rules**:
- Must be in descending order: `1h30m` ✓, `30m1h` ✗
- No duplicate units: `1h30m` ✓, `1h2h` ✗
- No whitespace within: `1h30m` ✓, `1h 30m` = two separate durations
- Can skip units: `1d30m` ✓ (skipping hours is fine)
- Integer values only: `1h30m` ✓, `1.5h` ✗ (use compound format for precision)

**Duration arithmetic**:
```opal
// Duration arithmetic with other durations
var total = 1h30m + 45m        // total = 2h15m
var remaining = 5m - 2m30s     // remaining = 2m30s
var scaled = 30s * 3           // scaled = 1m30s

// Variables can hold negative durations
var grace = 1m - 5m            // grace = -4m (preserved for logic)
var timeout = 30s - 1h         // timeout = -29m30s (preserved for logic)

// Conditional logic with negative durations
if grace < 0s {
    echo "No grace period remaining"
    exit 1
} else {
    @timeout(grace) { deploy() }
}
```

**Duration execution rules**:
```opal
// Runtime functions clamp negative durations to zero
@timeout(-4m) { cmd }          // Executes with 0s timeout
@retry(attempts=3, delay=-30s) { cmd }  // Uses 0s delay
sleep(-1h)                     // Sleeps for 0s (no-op)

// Variables preserve negative values for arithmetic/logic
var remaining = deadline - current_time
echo "Time remaining: @var.remaining"     // Shows "-5m" if past deadline
@timeout(remaining) { task() }          // Uses max(remaining, 0s) = 0s
```

**Duration evaluation rules**:
- All duration arithmetic evaluated at plan time with resolved values
- Variables can store negative durations for conditional logic
- Runtime functions automatically clamp negative durations to zero
- Duration literals are always non-negative (`30s`, `1h30m`)
- Negative expressions use minus operator: `-30s` = `MINUS` + `DURATION`

### Accessing Data

Use dot notation for nested access:

```opal
// Array access
start-api: docker run -p @var.SERVICES.0:3000 app
start-worker: docker run -p @var.SERVICES.1:3001 app

// Map access
connect: psql @var.CONFIG.database
cache: redis-cli -u @var.CONFIG.redis

// Wildcards expand to space-separated values
list-services: echo "Services: @var.SERVICES.*"    # "api worker ui"

// All equivalent ways to access arrays
@var.SERVICES.0    # Dot notation
@var.SERVICES[0]   # Bracket notation
@var.SERVICES.[0]  # Mixed notation
```

## Arithmetic and Assignment

Opal supports arithmetic operations for deterministic calculations in operations scripts. All arithmetic is evaluated at plan time, ensuring predictable results.

### Basic Arithmetic

```opal
// Deterministic calculations for operations
var total_replicas = base_replicas * environments
var batch_size = total_items / worker_count
var shard_id = item_id % shard_count
var timeout = base_timeout + (retry_attempt * backoff_multiplier)

// Memory and resource calculations
var total_memory = service_memory * replica_count
var disk_space = data_size + (log_retention * daily_logs)
var cpu_limit = base_cpu + (load_factor * peak_multiplier)
```

**Supported operators**:
- `+` - addition
- `-` - subtraction
- `*` - multiplication
- `/` - division
- `%` - modulo (remainder)

**Operator precedence** (highest to lowest):
1. `*`, `/`, `%` (multiplication, division, modulo)
2. `+`, `-` (addition, subtraction)
3. Use parentheses `()` for explicit ordering

### Assignment Operators

```opal
// Accumulation in deterministic loops
var total_cost = 0
for service in @var.SERVICES {
    total_cost += @var.SERVICE_COSTS[service]
}

// Resource scaling
var replicas = 1
replicas *= @var.ENVIRONMENTS.length  // multiply by environment count
replicas += 1                          // add monitoring replica

// Batch processing
var remaining = total_items
for batch in batches {
    remaining -= batch.size
    echo "Processing batch, @var.remaining items left"
}
```

**Supported assignment operators**:
- `+=` - add and assign
- `-=` - subtract and assign
- `*=` - multiply and assign
- `/=` - divide and assign
- `%=` - modulo and assign

### Increment and Decrement

```opal
// Counting in deterministic loops
var counter = 0
for service in @var.SERVICES {
    counter++
    echo "Processing service @var.counter: @var.service"
}

// Countdown with for loop (plan-time range)
var max_retries = 3
for attempt in 1...@var.max_retries {
    echo "Retry attempt @var.attempt"
}
```

**Supported operators**:
- `++` - increment by 1
- `--` - decrement by 1

### Deterministic Evaluation

All arithmetic operations are evaluated at plan time with known values:

```opal
// Plan shows exact calculations
var replicas = 3 * 2 + 1    // Plan: replicas = 7
var timeout = 30 + (2 * 5)  // Plan: timeout = 40

// Variable resolution then calculation
var base = @env.BASE_REPLICAS(default=2)  // Resolves to 2
var total = base * 3                         // Plan: total = 6
```

**Type rules**:
- `int` + `int` = `int`
- `float` + `float` = `float`
- `int` + `float` = `float` (automatic promotion)
- `duration` + `duration` = `duration`
- Division by zero detected at plan time

## Control Flow

Control flow happens at **plan time**, creating deterministic execution sequences.

### Block Phases and Deterministic Execution

Every `{ ... }` block in opal represents a **phase** - a unit of execution with strong deterministic guarantees. When you write a block, the planner expands it at plan time into a finite, ordered sequence of steps that will execute in a predictable way.

**Phase boundaries create execution order.** Each phase completes entirely before the next phase begins. This means all steps within a phase finish before any step in a subsequent phase can start. Within a phase, steps execute according to their canonical order - the order they appear after plan-time expansion.

**Variable mutations follow block semantics.** Most blocks (`for`, `if`, `when`, command blocks) allow variable mutations to affect the outer scope, since their execution is deterministic at plan time. However, `try/catch` blocks and execution decorator blocks use scope isolation to maintain predictable behavior (detailed below).

**Plans are verifiable contracts.** The resolved plan captures everything needed to verify execution: which steps run in what order, what commands they execute (with placeholders for resolved values), and how they handle retries or timeouts. If any of this changes between plan and execution, verification fails.

**Outputs are deterministically merged.** Each step produces its own stdout and stderr streams. When these need to be combined (for logging or display), they're merged in the canonical order, ensuring the same plan always produces the same combined output.

Block-specific constructs like `for`, `if`, `when`, `try/catch`, and `@parallel` work within this framework. They define how blocks expand (unrolling loops, selecting branches) or add constraints (parallel independence), but they all inherit the same phase execution guarantees.

### @parallel Output Determinism

**Output merge contract**: Stdout/stderr from parallel branches are buffered per-step and emitted in **plan order** (by step ID), not completion order.

**Example:**
```opal
@parallel {
    sleep 3 && echo "Branch A (slow)"     # Step ID: 1
    sleep 1 && echo "Branch B (fast)"     # Step ID: 2  
    echo "Branch C (instant)"             # Step ID: 3
}
```

**Output (always in this order):**
```
Branch A (slow)
Branch B (fast)
Branch C (instant)
```

Even though Branch C completes first, output is deterministic by step ID. This ensures:
- **Reproducible logs** - Same plan always produces same output order
- **Contract verification** - Output order is part of the execution contract
- **Debugging** - Output matches plan structure, not race conditions

**TUI/live progress**: Separate from final output. TUI can show live progress in completion order, but final stdout/stderr follows plan order.

### Command Definitions

Commands are defined using the `fun` keyword for reusable, parameterized blocks that expand at plan-time:

```opal
// Simple one-liner definitions
fun deploy = kubectl apply -f k8s/
fun hello = echo "Hello World!"

// Parameterized one-liners
fun greet(name) = echo "Hello, @var.name!"

// Multi-line block form
fun build_module(module, target="dist") {
    @workdir(@var.module) {
        npm ci
        npm run build --output=@var.target
    }
}

// Template function for reuse
fun test_module(module) {
    @workdir(@var.module) { go test ./... }
}

// Calling parameterized commands
build_all: {
    @cmd.build_module(module="frontend", target="public")
    @cmd.build_module(module="backend")  // uses default target="dist"
}
```

**Plan-time expansion**: `fun` definitions are **macros** that expand at plan-time when called via `@cmd.function_name()`. All parameters must be resolvable at plan-time using value decorators.

**DAG constraint**: Command calls must form a directed acyclic graph. Recursive calls or cycles result in plan generation errors.

**Parameter binding**: Arguments are bound to their resolved values at plan-time. Default values are supported and must be plan-time expressions.

**Deterministic**: All `fun` bodies must have finite execution paths - no unbounded loops or dynamic fan-out beyond normal metaprogramming expansion.

**Scope isolation**: `fun` bodies follow the same scope rules as other blocks - regular statements propagate mutations to outer scope, execution decorator blocks isolate scope.

### Loops

```opal
deploy-all: {
    for service in @var.SERVICES {
        echo "Deploying @var.service"
        kubectl apply -f k8s/@var.service/
        kubectl rollout status deployment/@var.service
    }
}

// Plan expands to independent steps:
// ├─ echo "Deploying api"
// ├─ kubectl apply -f k8s/api/
// ├─ kubectl rollout status deployment/api
// ├─ echo "Deploying worker"
// └─ ... (and so on)
```

For loops unroll at plan time into a known number of steps. The collection (`@var.SERVICES`) is resolved during planning, and each item creates a separate step in the canonical order. Empty collections produce zero steps.

### Conditionals

```opal
deploy: {
    if @var.ENV == "production" {
        kubectl apply -f k8s/prod/
        kubectl scale --replicas=3 deployment/app
    } else {
        kubectl apply -f k8s/dev/
        kubectl scale --replicas=1 deployment/app
    }
}
```

Conditionals are evaluated at plan time using resolved variable values. Only the taken branch expands into steps - the other branch becomes dead code that doesn't appear in the final plan.

### Pattern Matching

```opal
deploy: {
    when @var.ENV {
        "production" -> {
            kubectl apply -f k8s/prod/
            kubectl scale --replicas=3 deployment/app
        }
        "staging" -> kubectl apply -f k8s/staging/
        else -> echo "Unknown environment: @var.ENV"
    }
}
```

Pattern matching uses first-match-wins evaluation at plan time. Supported patterns include exact strings (`"production"`), OR expressions (`"main" | "develop"`), regex patterns (`r"^release/"`), numeric ranges (`200...299`), and catch-all (`else`). Only the matching branch expands into the plan.

### Error Handling

Try/catch is the **only** non-deterministic construct:

```opal
deploy: {
    try {
        kubectl apply -f k8s/
        kubectl rollout status deployment/app
        echo "Deployment successful"
    } catch {
        echo "Deployment failed, rolling back"
        kubectl rollout undo deployment/app
    } finally {
        echo "Cleaning up temporary resources"
        kubectl delete pod -l job=deployment-helper
    }
}
```

The plan records all possible execution paths through try/catch blocks. At runtime, only one of `try` or `catch` executes, while `finally` always runs. Execution logs show which path was actually taken.

## Scope Isolation

The rule is simple: **values can flow in from the outer scope, but mutations never flow back out**.

```opal
var counter = 0
var status = "pending"

try {
    // Can READ outer scope values
    echo "Starting with counter=@var.counter"  // counter = 0 ✓

    // Can MODIFY local copies
    counter++           // Local counter = 1
    status = "running"  // Local status = "running"

    kubectl apply -f k8s/
} catch {
    // Can READ outer scope values
    echo "Failed with counter=@var.counter"    // counter = 0 ✓

    // Can MODIFY local copies
    counter += 5        // Local counter = 5
    status = "failed"   // Local status = "failed"
}

// Outer scope unchanged after try/catch
echo "Final: counter=@var.counter, status=@var.status"  // counter=0, status="pending" ✓
```

**Decorator blocks work the same way:**

```opal
var attempts = 0
var result = ""

@retry(attempts=3) {
    // Can READ outer scope
    echo "Base attempts: @var.attempts"  // attempts = 0 ✓

    // Can MODIFY local copies
    attempts++         // Local attempts = 1, 2, 3...
    result = "done"    // Local result = "done"

    kubectl apply -f k8s/
}

// Outer scope unchanged after execution decorator
echo "Final: attempts=@var.attempts, result=@var.result"  // attempts=0, result="" ✓
```

This pattern ensures that both non-deterministic execution (try/catch) and execution decorator behaviors don't create unpredictable state mutations in the outer scope.

## Decorators

### Value Decorators

Inject values inline:

```opal
// Environment variables
start: node app.js --port @env.PORT(default=3000)

// Opal variables
scale: kubectl scale --replicas=@var.REPLICAS deployment/app

// Expensive lookups (resolved lazily)
deploy: kubectl apply --token=@aws.secret.api_token
config: curl -H "Authorization: @http.get('https://auth.com/token')"
```

### Execution Decorators

Enhance command execution:

```opal
// Retry with named parameters
deploy: @retry(attempts=3, delay=2s, backoff=1.5) {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
}

// Timeout protection
build: @timeout(5m) {
    npm run build
    npm test
}

// Parallel execution
services: @parallel {
    npm run api
    npm run worker
    npm run ui
}

// Command references
deploy: @cmd.build && @cmd.test && @cmd.apply
```

### Value Decorators and Remote Execution

**IMPORTANT**: `@env` reads from the **current session's environment**, which changes based on context (local, remote, container).

#### How @env Works

**`@env` is session-aware:**
- In local context: reads from `os.Environ()` snapshot
- In transport context: reads from transport's environment (native + `env={}` overrides)
- Both `@env.X` and `$X` read from the same session environment (different timing)

#### Environment in Different Contexts

**Local execution:**
```opal
var user = @env.USER  # Reads from local os.Environ()
echo $USER            # Also reads from local environment
```

**Remote execution (idempotent transport):**
```opal
@ssh.connect(host="remote-server", env={"DEPLOY_ENV": "prod"}) {
    # @env reads from SSH session (remote environment + env={} overrides)
    var remote_user = @env.USER        # Remote USER (e.g., "deploy")
    var deploy_env = @env.DEPLOY_ENV   # "prod" (from env={} override)
    
    # Shell variables read from same SSH session
    echo $USER        # Remote USER (e.g., "deploy")
    echo $DEPLOY_ENV  # "prod"
}
```

**Passing local values to remote:**
```opal
var local_user = @env.USER  # "alice" from local session

@ssh.connect(host="remote-server", env={"DEPLOYER": @var.local_user}) {
    var deployer = @env.DEPLOYER      # "alice" (explicitly passed)
    var remote_user = @env.USER       # "deploy" (remote native)
    
    echo "Deployed by: @var.deployer"  # "alice"
    echo "Running as: $USER"           # "deploy"
}
```

#### Environment Isolation

**Critical:** Environments do NOT automatically pass between transports.

```opal
# Local: USER=alice
# Remote: USER=deploy

@ssh.connect(host="remote-server") {
    var user = @env.USER  # "deploy" (NOT "alice")
    # Remote session has no knowledge of local USER
}
```

**To pass values, use `env={}` explicitly:**
```opal
var local_user = @env.USER  # "alice"

@ssh.connect(host="remote-server", env={"DEPLOYER": @var.local_user}) {
    var deployer = @env.DEPLOYER  # "alice" (explicitly passed)
}
```

#### Idempotent vs Non-Idempotent Decorators

**Idempotent transports** (connect to existing resources):
- Connect at plan-time (safe, read-only)
- `@env` works in blocks (reads from connected session)
- Examples: `@ssh.connect`, `@docker.exec`

```opal
@ssh.connect(host="existing-server") {
    var home = @env.HOME  # ✅ OK: SSH connects at plan-time
}
```

**Non-idempotent decorators** (create new resources):
- Can't connect at plan-time (resource doesn't exist yet)
- `@env` FORBIDDEN in blocks
- Examples: `@aws.instance.deploy`, `@aws.rds.deploy`

```opal
@aws.instance.deploy(name="web") {
    var home = @env.HOME  # ❌ ERROR: Instance doesn't exist yet
    echo $HOME            # ✅ OK: Shell variable resolves at execution-time
}
```

**Error message:**
```
ERROR: @env cannot be used inside @aws.instance.deploy
Reason: Instance might not exist during planning (non-idempotent)
Solution: Use shell variables ($HOME, $USER) which resolve at execution-time,
          or hoist: var x = @env.X at root, then use @var.x
```

#### Summary

- **`@env.X`** = Reads from current session's environment (context-aware)
- **`$X`** = Shell variable (reads from same session, execution-time)
- **`@var.X`** = Plan-time variable (can be used anywhere)

**When to use each:**
- Local environment values? → `@env.X` at root
- Remote environment values? → `@env.X` in transport block OR `$X` in commands
- Pass local to remote? → `var x = @env.X` then `env={"KEY": @var.x}`
- Optional values? → `@env.X(default="fallback")`

## Plans: Three Execution Modes

Opal provides three distinct planning and execution modes, each serving different operational needs.

**Plan provenance**: Plans include source_commit, spec_version, and compiler_version in headers for audit trails.

### Quick Plans (`--dry-run`)

**Purpose**: Fast preview of likely execution paths without expensive operations.

```bash
opal deploy --dry-run
```

**What happens**:
- Resolves control flow (`if`/`when`/`for` conditions) using cheap value decorators only
- Shows single execution path after metaprogramming expansion
- Defers expensive value decorators (`@aws.secret`, `@http.get`, etc.)
- Displays which branches/iterations were taken for auditability

```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=¹@aws.secret("api-token")
└─ @if(ENV == "production")
   └─ kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app

Deferred Values:
1. @aws.secret("api-token") → <expensive: AWS lookup>
```

**Visual format note**: This tree structure is optimized for human readability. The internal contract format uses an optimized structure for efficient parsing and verification.

### Resolved Plans (`--dry-run --resolve`)

**Purpose**: Complete execution contract with all values resolved.

```bash
opal deploy --dry-run --resolve > prod.plan
```

**What happens**:
- Resolves ALL value decorators (including expensive ones)
- Expands all metaprogramming constructs into concrete execution steps
- Creates deterministic execution contract with hash placeholders
- Generates plan file for later contract-verified execution

```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=🔒 opal:s:3J98t56A
└─ @if(ENV == "production")
   └─ kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app

Contract Hash: sha256:abc123...
```

**Key principles**:
- All resolved values use `opal:s:ID` format (security by default)
- Metaprogramming constructs (`@if`, `@for`, `@when`) show which path was taken
- Original constructs are preserved for audit trails while showing expanded results

### Execution Plans (always happens)

**Purpose**: Runtime resolution and execution.

```bash
# Direct execution
opal deploy

# Contract-verified execution
opal run --plan prod.plan
```

**What happens**:
1. **Internal resolution**: Resolves all value decorators fresh at execution time
2. **Path determination**: Follows single execution path based on resolved values
3. **Contract verification** (if using plan file): Ensures resolved values match contract hashes
4. **Execution**: Runs commands with internally resolved values

**Security by default**: All values appear as `🔒 opal:s:3J98t56A` format (opaque context-aware ID, no length leak).

> **Placeholder Format**
> `opal:kind:ID` where kind is `s` (secret), `v` (value), etc., and ID is Base58-encoded.
> Format: `🔒 opal:s:3J98t56A` (with emoji for terminal display)
> Machine-readable: `opal:s:3J98t56A` (without emoji for JSON/files)
> 
> **DisplayID Determinism:**
> 
> Each plan has a random `PlanSalt` (32 bytes) that determines DisplayID generation:
> 
> - **Mode 1 (Direct execution)**: Random PlanSalt per run
>   - Different runs → different DisplayIDs (prevents correlation across runs)
>   - Security: Attackers can't track secrets across executions
> 
> - **Mode 3 (Resolved plans / Contract generation)**: Random PlanSalt per contract
>   - Different contracts → different PlanSalts → different DisplayIDs (prevents correlation)
>   - Within contract: Same PlanSalt → deterministic DisplayIDs (enables verification)
>   - Security: Each contract gets unique DisplayIDs, no cross-contract correlation
> 
> - **Mode 4 (Contract execution)**: Reuses PlanSalt from contract
>   - Fresh plan uses contract's PlanSalt → same DisplayIDs as contract
>   - Hash comparison succeeds if structure unchanged (DisplayIDs match)
>   - Hash comparison fails if structure changed (drift detected)
> 
> DisplayIDs use a keyed PRF: `BLAKE2s(PlanSalt, step_path || decorator || key_name || hash(value))`.
> This prevents oracle attacks while enabling deterministic contract verification.







**Plan hash scope**: Ordered steps + arguments (with `opal:s:ID` placeholders) + operator graph + resolution timing flags; excludes ephemeral run IDs/logs.

> **Security Invariant**
> Raw secrets never appear in plans or logs, only `🔒 opal:s:3J98t56A` placeholders.
> This applies to all value decorators: `@env.KEY`, `@var.NAME`, `@aws.secret.name`, etc.
> Compliance teams can review plans with confidence.
> 
> **Contract Verification with DisplayIDs:**
> Plan hashes include DisplayID placeholders in their canonical form. In resolved plans,
> DisplayIDs are deterministic (derived from plan digest + context), ensuring the same
> plan produces the same hash. This enables contract verification while maintaining
> security through opaque, context-aware identifiers.

## Contract Verification

When using plan files, opal ensures execution matches the reviewed contract exactly.

### Contract-Verified Execution

```bash
# Execute against resolved plan contract
opal run --plan prod.plan
```

**Critical: Plan files are NEVER executed directly.**

Contract execution always replans from current source and state. The plan file is not executed - it's the verification target. If the fresh plan doesn't match, execution aborts.

**Unlike Terraform** (which applies an old plan to new state), Opal verifies current reality would still produce the same plan. This prevents executing stale or tampered plans.

**Contract verification process**:
1. **Replan from source**: Parse current source code and query current infrastructure
2. **Fresh resolution**: Resolve all value decorators with current state
3. **Hash comparison**: Compare fresh plan hashes with contract hashes
4. **Path verification**: Ensure execution path matches contracted plan structure
5. **Execute or bail**: Run fresh plan if it matches contract, otherwise fail with diff

**Why this works**: The contract contains `<length:hash>` placeholders. Opal generates a fresh plan and compares hashes. If `@env.REPLICAS` was `3` during planning but is now `5`, the hashes won't match and execution stops.

### Verification Outcomes

**✅ Contract verified** - hashes match, execution proceeds:
```
✓ Contract verified: all value decorators resolve to expected hashes
→ Executing contracted plan...
```

**❌ Contract violated** - hashes differ, execution stops:
```
ERROR: Contract verification failed

Expected: kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app
Actual:   kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app

Variable REPLICAS changed: was 3, now 5
→ Source or environment changed since plan generation
→ Run 'opal deploy --dry-run --resolve' to generate new plan
```

**Contract violation causes**:
- `source_changed`: Source files modified since plan generation
- `env_changed`: Environment variables modified since plan generation
- `infra_drift`: Infrastructure state changed since plan generation

### Direct Execution (No Contract)

```bash
# Always resolves fresh, no contract verification
opal deploy
```

**Direct execution process**:
1. **Fresh resolution**: Resolve all value decorators with current state
2. **Path determination**: Follow execution path based on resolved values
3. **Execute**: Run commands with resolved values (no hash verification)

This mode is ideal for development and immediate execution where you want current values, not contracted values.

## Plan Visual Structure

Plans show the execution path after metaprogramming expansion using a consistent tree format. This visual structure is optimized for human readability and audit trails, while the internal contract format uses an optimized binary structure for efficient parsing and verification.

### Metaprogramming Expansion Patterns

**For loops** expand into sequential steps:
```opal
// Source: for service in ["api", "worker"] { kubectl apply -f k8s/@var.service/ }

// Plan shows:
deploy:
└─ @for(service in ["api", "worker"])
   ├─ kubectl apply -f k8s/api/
   └─ kubectl apply -f k8s/worker/
```

**If statements** show the taken branch:
```opal
// Source: if ENV == "production" { kubectl scale --replicas=3 }

// Plan shows:
deploy:
└─ @if(ENV == "production")
   └─ kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app
```

**When patterns** show the matched pattern:
```opal
// Source: when ENV { "production" -> kubectl scale --replicas=3; else -> kubectl scale --replicas=1 }

// Plan shows:
deploy:
└─ @when(ENV == "production")
   └─ kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app
```

**Try/catch blocks** show all possible paths:
```opal
// Source: try { kubectl apply } catch { kubectl rollout undo } finally { kubectl clean }

// Plan shows:
deploy:
└─ @try
   ├─ kubectl apply -f k8s/
   ├─ @catch
   │  └─ kubectl rollout undo deployment/app
   └─ @finally
      └─ kubectl delete pod -l job=temp
```

### Security and Hash Format

**Security by default**: Opal prevents secrets from leaking into plans, logs, and terminal output through automatic scrubbing. ALL value decorator results are treated as secrets - no exceptions.

**DisplayID format**: All resolved values appear as `🔒 opal:s:3J98t56A` (opaque context-aware ID):
- `🔒 opal:s:3J98t56A` - single character value (e.g., "3")
- `🔒 opal:s:3J98t56A` - 32 character value (e.g., secret token)
- `🔒 opal:s:3J98t56A` - 8 character value (e.g., hostname)

**Why all values are secrets**: Even seemingly innocuous values could leak sensitive information:
- `@env.HOME` - Could leak system paths
- `@var.username` - Could leak user information
- `@git.commit_hash` - Could leak repository state
- `@aws.secret.key` - Obviously sensitive

**Where scrubbing applies**:
- ✅ **Plans**: All values replaced with DisplayIDs
- ✅ **Terminal output**: Stdout/stderr scrubbed before display
- ✅ **Logs**: All logging output scrubbed
- ❌ **Pipes**: Raw values flow between operators (needed for work)
- ❌ **Redirects**: Raw values written to files (user controls destination)

**Explicit scrubbing**: Use the `scrub()` PipeOp to scrub output before redirects:
```opal
// Scrub secrets from kubectl output before writing to file
kubectl get secret db-password -o json |> scrub() > backup.json
```

**Security properties**:
- **No value leakage** in plans or logs
- **Contract verification** via hash comparison
- **Debugging support** via length hints (no correlation)
- **Algorithm agility** for future hash upgrades
- **Audit-friendly**: Compliance teams can review plans safely

## Planning Mode Summary

| Mode | Command | Value Resolution | Use Case |
|------|---------|------------------|----------|
| **Quick Plan** | `--dry-run` | Cheap values only | Fast preview, see possible paths |
| **Resolved Plan** | `--dry-run --resolve` | ALL values resolved | Complete contract, audit review |
| **Direct Execution** | `deploy` | Fresh resolution | Development, immediate execution |
| **Contract Execution** | `run --plan file` | Fresh + hash verification | Production, change detection |

### When to Use Each Mode

**Quick plans** for:
- Development workflow: "What will this probably do?"
- Fast feedback during script development
- Understanding possible execution branches

**Resolved plans** for:
- Change window planning: Generate contract hours before execution
- Audit review: Show exactly what will execute with real values
- Production contracts: Lock in execution plan with hash verification

**Direct execution** for:
- Development and testing: Run with current values immediately
- One-off operations: Execute without contract overhead
- Iterative script development

**Contract execution** for:
- Production deployments: Ensure reviewed plan matches execution
- Time-delayed execution: Execute plan generated hours earlier
- Change detection: Catch environment drift since plan generation

## Resolution Strategy

**Value timing rules**:
- **Quick plans**: Expensive value decorators deferred, show placeholders
- **Resolved plans**: ALL value decorators resolve, values frozen as execution contract
- **Direct execution**: Fresh resolution at execution time, no contracts
- **Contract execution**: Fresh resolution + hash verification against contract
- **Dead code elimination**: Value decorators in pruned branches never execute

**Non-determinism guardrail**: Value decorators must be referentially transparent during `--resolve`. Non-deterministic value decorators cause contract verification failures.

**Seeded determinism**: Operations requiring randomness or cryptography (like `@random.password()` or `@crypto.generate_key()`) use Plan Seed Envelopes (PSE) to be deterministic within resolved plans while maintaining security. **Note**: PSE is NOT used for `@var` or `@env` which have actual values - PSE is specifically for decorators that need deterministic randomness.

**Always planned**: Even direct script execution generates internal plans first, ensuring consistent behavior.

## Time-Delayed Execution

Real operations involve plan generation and execution at different times. Opal's verification model handles this cleanly.

### The Scenario

```bash
# 2:00 PM - Generate plan during change window planning
opal deploy --dry-run --resolve > evening-deploy.plan

# 5:00 PM - Execute plan during maintenance window
opal run --plan evening-deploy.plan
```

### Verification at Execution

When you execute a resolved plan, opal **verifies the contract**:

1. **Replan** from source files with current infrastructure state
2. **Compare** new plan structure with resolved plan
3. **Verify** all resolved values still match (structure and hashes)
4. **Execute** contracted plan if verification passes
5. **Fail** with diff if source changed or infrastructure drifted

### What Gets Verified

**Source changes detected**:
```
ERROR: Plan verification failed

Expected: for service in ["api", "worker"] {
Actual:   for service in ["api", "worker", "ui"] {

Source file modified since plan generation.
```

**Infrastructure drift detected**:
```
ERROR: Infrastructure state changed

Expected: kubectl scale --replicas=🔒 opal:s:3J98t56A deployment/app
Current:  No deployment/app found

Infrastructure changed since plan generation.
Consider regenerating plan or using --force.
```

**Non-deterministic value decorator detected**:
```
ERROR: Contract verification failed

@http.get("https://time-api.com/now") returned different value:
  Plan time: 🔒 opal:s:3J98t56A
  Execution:  🔒 opal:s:3J98t56A

Non-deterministic value decorators cannot be used in resolved plans.
Consider separating dynamic value acquisition from deterministic execution.
```

### Plan Seed Envelopes (PSE)

**Purpose**: PSE provides deterministic randomness for decorators like `@random.password()` and `@crypto.generate_key()`. It is NOT used for `@var` or `@env` which resolve to actual values.

For operations requiring randomness, resolved plans contain a Plan Seed Envelope - a minimal, immutable piece of state that enables deterministic random generation:

```opal
// Default: regenerates on every new plan
var TEMP_TOKEN = @random.password(length=16)

// Stable across plan changes until regeneration key changes
var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v1")
var API_KEY = @crypto.generate_key(type="ed25519", regen_key="api-key-v3")

deploy: {
    kubectl create secret generic db --from-literal=password=@var.DB_PASS
    kubectl create secret generic api --from-literal=key=@var.API_KEY
}

// Rotate secrets by changing regeneration key
rotate-secrets: {
    // Change v1 → v2 to get new password
    var NEW_DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v2")
    kubectl patch secret db --patch='{"data":{"password":"'$(echo -n @var.NEW_DB_PASS) | base64)'"}}'
}
```

**Plan shows placeholders** (maintaining security invariant):
```
kubectl create secret generic db --from-literal=password=¹🔒 opal:s:3J98t56A
kubectl create secret generic api --from-literal=key=¹🔒 opal:s:3J98t56A
```

**How PSE works**:
- High-entropy seed generated at `--resolve` time
- Seed encrypted and sealed to authorized runners (file-based key or KMS)
- Each value decorator derives unique deterministic values using plan context
- Same plan always generates identical random values
- Different plans generate different values (new seed per plan)

**PSE vs traditional state**:
- **Scoped**: Each plan contains its own PSE, no shared state
- **Immutable**: Never changes after plan creation, no updates or migrations
- **Self-contained**: Plan file includes everything needed, no external storage
- **Minimal**: Contains only cryptographic entropy, not infrastructure tracking
- **Contract-aligned**: Enables deterministic execution within resolved plans

This gives you secure, auditable randomness while maintaining the stateless execution model.

### Execution Options

**Strict verification** (default):
```bash
opal run --plan prod.plan
# Fails on any source or infrastructure changes
```

**Force execution**:
```bash
opal run --plan prod.plan --force
# Uses resolved plan values as targets, adapts to current infrastructure
```



### Why This Works

**Contract clarity**: Teams review resolved plans knowing exactly what will execute
**Change detection**: Any modifications between plan and execution are caught
**Audit trail**: Resolved plans become immutable execution records
**Flexibility**: Multiple execution modes for different operational needs

This verification model gives you the determinism of resolved plans with safety against unexpected changes.

## Parallel Resolution

Independent expensive operations resolve concurrently:

```opal
var CLUSTER = @env.KUBE_CLUSTER(default="minikube")
var DB_HOST = @aws.secret("@var.CLUSTER()-db-host")
var API_KEY = @http.get("https://keygen.com/api/new")

deploy: {
    when @var.CLUSTER {
        "production" -> {
            kubectl apply -f k8s/prod/ --server=@var.DB_HOST
            kubectl create secret --api-key=@var.API_KEY
        }
        else -> kubectl apply -f k8s/dev/
    }
}
```

**Resolution DAG**:
- `@env.KUBE_CLUSTER` resolves first (needed for conditional)
- `@aws.secret()` and `@http.get()` resolve in parallel
- Unused expensive operations in dev branch never execute

## Change Detection and Idempotency

Scripts are **idempotent by design** with smart change detection:

```bash
# First run
opal deploy
# Creates resources, shows what was done

# Second run (no changes)
opal deploy
# Shows "no-op" for existing resources

# Third run (environment changed)
REPLICAS=5 opal deploy
# Shows only the scale operation (replica count changed)
```

**How change detection works**:
- Value hashing: `<1:3>` → `<1:5>` indicates REPLICAS changed from 3 to 5
- Secret rotation: `<32:a1b>` → `<32:x7y>` indicates API token was rotated
- Infrastructure queries: Value decorators check current state vs desired state
- Character count in hash gives size hints for debugging

**Resolved plan verification adds another layer**:
- Source changes detected by comparing plan structures
- Hash changes show which values modified
- Infrastructure drift caught by re-querying current state

## Why This Design Works

**Contract-based**: Resolved plans are execution contracts with verification
**Auditable**: See exactly what will run, verify it matches before execution
**Secure**: No sensitive values in plans or logs, change detection without exposure
**Stateless**: No state files to corrupt - value decorators query reality fresh each time
**Readable**: More natural than YAML, more structured than shell scripts
**Extensible**: New value decorators and execution decorators integrate seamlessly

Opal transforms operations from "pray and deploy" to "plan, review, execute with verification" - bringing contract discipline to deployment workflows without traditional infrastructure tool complexity.

## Examples

### Web Application Deployment

```opal
var (
    ENV = @env.ENVIRONMENT(default="development")
    VERSION = @env.APP_VERSION(default="latest")
    REPLICAS = @env.REPLICAS(default=1)
    SERVICES = ["api", "worker", "ui"]
)

deploy: {
    echo "Deploying version @var.VERSION to @var.ENV"

    when @var.ENV {
        "production" -> {
            for service in @var.SERVICES {
                @retry(attempts=3, delay=10s) {
                    kubectl apply -f k8s/prod/@var.service/
                    kubectl set image deployment/@var.service app=@var.VERSION
                    kubectl scale --replicas=@var.REPLICAS deployment/@var.service
                    kubectl rollout status deployment/@var.service --timeout=300s
                }
            }
        }
        "staging" -> {
            @timeout(5m) {
                kubectl apply -f k8s/staging/
                kubectl set image deployment/app app=@var.VERSION
                kubectl rollout status deployment/app
            }
        }
        else -> {
            echo "Deploying to development"
            kubectl apply -f k8s/dev/
        }
    }

    echo "Deployment complete"
}
```

### Database Migration with Rollback

```opal
var DB_URL = @env.DATABASE_URL
var BACKUP_BUCKET = @env.BACKUP_BUCKET(default="db-backups")

migrate: {
    try {
        echo "Starting database migration"

        @timeout(30m) {
            # Backup first
            pg_dump @var.DB_URL | gzip > backup-$(date +%Y%m%d-%H%M%S).sql.gz
            aws s3 cp backup-*.sql.gz s3://@var.BACKUP_BUCKET/

            # Run migration
            @retry(attempts=2, delay=5s) {
                psql @var.DB_URL -f migrations/001-add-users.sql
                psql @var.DB_URL -f migrations/002-add-indexes.sql
            }

            # Verify
            psql @var.DB_URL -c "SELECT COUNT(*) FROM users;"
        }

        echo "Migration completed successfully"

    } catch {
        echo "Migration failed, restoring from backup"

        # Find latest backup
        LATEST_BACKUP=$(aws s3 ls s3://@var.BACKUP_BUCKET/ | tail -1 | awk '{print $4}')

        # Restore
        aws s3 cp s3://@var.BACKUP_BUCKET/@var.LATEST_BACKUP - | gunzip | psql @var.DB_URL

        echo "Database restored from backup"

    } finally {
        # Cleanup local backup files
        rm -f backup-*.sql.gz
        echo "Migration process completed"
    }
}
```

## Common Errors

### Defining `fun` Inside Loops

**❌ Error:**
```opal
for module in @var.MODULES {
    fun build_@var.module {  # ERROR: fun inside for loop
        npm run build
    }
}
```

**✅ Correct:**
```opal
# Define template function at top level
fun build_module(module) {
    @workdir(@var.module) {
        npm run build
    }
}

# Call it inside loop
for module in @var.MODULES {
    @cmd.build_module(module=@var.module)
}
```

**Why:** `fun` definitions must be at top level. They're plan-time templates, not runtime constructs.

### Using Shell Variables Instead of Opal Variables

**❌ Error:**
```opal
var SERVICE = "api"
kubectl scale deployment/${SERVICE} --replicas=3  # Wrong syntax
```

**✅ Correct:**
```opal
var SERVICE = "api"
kubectl scale deployment/@var.SERVICE --replicas=3  # Opal syntax
```

**Why:** Use `@var.NAME` for Opal variables, not `${NAME}` (shell syntax).

### Mixing Positional and Named Arguments Incorrectly

**❌ Error:**
```opal
@retry(3, delay=2s, 5)  # Positional after named
```

**✅ Correct:**
```opal
@retry(3, 5, delay=2s)           # Positional first
@retry(attempts=3, delay=2s)     # All named
@retry(3, delay=2s)              # Mixed (positional first)
```

**Why:** Positional arguments must come before named arguments.

### Forgetting Block Terminator for Variable Interpolation

**❌ Error:**
```opal
echo "@var.service_backup"  # Looks for variable "service_backup"
```

**✅ Correct:**
```opal
echo "@var.service()_backup"  # Terminates with (), then adds "_backup"
```

**Why:** Use `()` to terminate decorator when followed by ASCII characters without spaces.

### Non-Deterministic Value Decorators in Resolved Plans

**❌ Error:**
```opal
var timestamp = @time.now()  # Non-deterministic
opal deploy --dry-run --resolve > plan.json  # Will fail verification
```

**✅ Correct:**
```opal
# Use plan-time constants or seeded randomness
var deployTime = @env.DEPLOY_TIME(default="2024-01-01")
var randomPass = @random.password(length=16, regenKey="db-pass-v1")
```

**Why:** Resolved plans must be deterministic. Use Plan Seed Envelopes for randomness.

