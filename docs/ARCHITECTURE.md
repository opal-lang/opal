---
title: "Opal Architecture"
audience: "Core Developers & Contributors"
summary: "System design and implementation of the plan-verify-execute model"
---

# Opal Architecture

**Implementation requirements for the plan-verify-execute model**

**Audience**: Core developers, plugin authors, and contributors working on the Opal runtime, parser, or execution engine.

**See also**: [SPECIFICATION.md](SPECIFICATION.md) for user-facing language semantics and guarantees.

## Target Scope

Operations and developer task automation - the gap between "infrastructure is up" and "services are reliably operated."

**Why this scope?** Operations and task automation is the immediate need - reliable deployment, scaling, rollback, and operational workflows.

## Core Requirements

These principles implement the guarantees defined in [SPECIFICATION.md](SPECIFICATION.md):

- **Deterministic planning**: Same inputs → identical plan
- **Contract verification**: Detect environment changes between plan and execute
- **Fail-fast**: Errors at plan-time, not execution
- **Halting guarantee**: All plans terminate with predictable results

### Concept Mapping

| Concept | Purpose | Defined In | Tested In |
|---------|---------|------------|-----------|
| **Plans** | Execution contracts | [SPECIFICATION.md](SPECIFICATION.md#plans-three-execution-modes) | [TESTING_STRATEGY.md](TESTING_STRATEGY.md#golden-plan-tests) |
| **Decorators** | Value injection & execution control | [SPECIFICATION.md](SPECIFICATION.md#decorator-syntax) | [TESTING_STRATEGY.md](TESTING_STRATEGY.md#decorator-conformance-tests) |
| **Contract Verification** | Hash-based change detection | [SPECIFICATION.md](SPECIFICATION.md#contract-verification) | [TESTING_STRATEGY.md](TESTING_STRATEGY.md#contract-verification-tests) |
| **Event-Based Parsing** | Zero-copy plan generation | [AST_DESIGN.md](AST_DESIGN.md#event-based-plan-generation) | [TESTING_STRATEGY.md](TESTING_STRATEGY.md#parser-tests) |
| **Dual-Path Architecture** | Execution vs tooling | This document | [AST_DESIGN.md](AST_DESIGN.md#dual-path-pipeline) |
| **Observability** | Run tracking & debugging | [OBSERVABILITY.md](OBSERVABILITY.md) | [TESTING_STRATEGY.md](TESTING_STRATEGY.md#observability-tests) |

## Architectural Philosophy

**Stateless, reality-driven execution:**

> Opal's architecture treats *reality* as its database.

Traditional IaC tools maintain state files to track "what should exist." Opal takes a different approach:

1. **Query reality** - Decorators check actual current state (API calls, file checks, etc.)
2. **Generate plan** - Based on reality + user intent, create execution contract
3. **Freeze the contract** - Plan becomes immutable with hash-based verification
4. **Execute** - Perform work, verify contract still valid

**Why stateless works:**

- Reality is the source of truth, not a state file
- Re-query on every run - always current
- No state drift, no state locking, no state corruption
- Mix Opal with other tools freely - no coordination needed

**Plans as contracts:**

Plans aren't previews - they're immutable execution contracts. Hash-based verification detects if reality changed between plan and execute, failing fast instead of executing against stale assumptions.

## The Big Picture

```
User writes natural syntax  →  Parser converts to value decorators and execution decorators  →  Contract execution
```

Opal has two distinct layers that work together:

**Metaprogramming constructs** decide execution structure:

*Plan-time deterministic:*
- `for service in [...] { ... }` → unrolls loops into concrete steps
- `when ENV { ... }` → selects branches based on conditions  
- `if condition { ... } else { ... }` → evaluates conditionals at plan-time

*Execution-dependent path selection:*
- `try/catch/finally` → defines deterministic error handling paths, but which path executes depends on actual execution results (exceptions)

**Work execution** happens through decorators at runtime:
- `npm run build` → `@shell("npm run build")`
- `@retry(3) { ... }` → execution decorator with block
- `@var.NAME` → value decorator for interpolation

## Everything is a Decorator (For Work Execution)

The core architectural principle: **every operation that performs work** becomes one of two decorator types: value decorators or execution decorators.

This means metaprogramming constructs like `for`, `if`, `when` are **not** decorators - they're language constructs that decide what work gets done. The actual work is always performed by decorators.

**Value decorators** inject values inline:
- `@env.PORT` pulls environment variables
- `@var.REPLICAS` references script variables  
- `@aws.secret.api_key` fetches from AWS (expensive)

**Execution decorators** run commands:
- `@shell("npm run build")` executes shell commands
- `@retry(3) { ... }` adds retry logic around blocks
- `@parallel { ... }` runs commands concurrently

Even plain shell commands become `@shell` decorators internally:
```opal
// You write
npm run build

// Parser generates  
@shell("npm run build")
```

This separation means:
- **AST structure** represents both metaprogramming constructs and decorators appropriately
- **Execution model** is unified through decorators (no special cases for different work types)  
- **New features** integrate by adding decorators, not special execution paths

## The Execution Context Pattern

### How Decorators Execute Blocks

**Core insight:** Decorators take a declaration of what needs to run and wrap it in their execution context.

When you write:
```opal
@aws.instance.ssh(host="prod-server") {
    cat /var/log/app.log
}
```

The decorator does three things:
1. **Setup** - Establish SSH connection to prod-server
2. **Execute** - Run the block (cat command) within SSH context
3. **Teardown** - Close connection

### The Pattern: Execution Context with Callback

**Decorators receive:**
- **Parameters** - Configuration (`host="prod-server"`)
- **Block** - Child steps to execute
- **Execution Context** - Callback to executor

**Handler signature:**
```go
type ExecutionHandler func(
    ctx ExecutionContext,    // Execution context with args and I/O
    block []Step,            // Child steps to execute
) (exitCode int, err error)
```

**Execution context provides:**
```go
type ExecutionContext interface {
    // Execute block within this context
    ExecuteBlock(steps []Step) (exitCode int, err error)
    Context() context.Context  // For cancellation and deadlines
    
    // Decorator arguments (typed accessors)
    ArgString(key string) string
    ArgInt(key string) int64
    ArgBool(key string) bool
    ArgDuration(key string) time.Duration
    Args() map[string]interface{}  // Snapshot for logging
    
    // Environment and working directory (immutable snapshots)
    Environ() map[string]string
    Workdir() string
    
    // Context wrapping (returns new context with modifications)
    WithContext(ctx context.Context) ExecutionContext
    WithEnviron(env map[string]string) ExecutionContext
    WithWorkdir(dir string) ExecutionContext
    
    // I/O streams (for pipeline execution)
    Stdin() io.Reader      // Piped input (nil if not piped)
    StdoutPipe() io.Writer // Piped output (nil if not piped)
    
    // Context cloning (for child commands in pipelines)
    Clone(args map[string]interface{}, stdin io.Reader, stdout io.Writer) ExecutionContext
    
    // Transport (for remote execution - implementation detail)
    Transport() interface{}  // Returns executor.Transport implementation
}
```

**Key properties:**

- **Immutable:** All fields are immutable snapshots. Modifications create new contexts.
- **Copy-on-write:** `WithEnv()` and `WithWorkdir()` return new contexts without mutating original.
- **Pipeline support:** `Stdin()` and `StdoutPipe()` enable streaming between commands.
- **Context threading:** Environment and workdir propagate through decorator hierarchy.

**Security model:**

All I/O flows through the Session interface, not directly through ExecutionContext. The executor:
1. Maintains a registry of secret values from value decorator resolutions
2. Automatically replaces secret values with opaque DisplayIDs: `opal:s:3J98t56A`
3. Ensures audit trail shows which secrets were used without exposing values

Decorators execute commands via `Session.Run()`, which handles I/O routing and secret scrubbing.

### Recursive Execution: Nested Decorators

When decorators are nested, each wraps the next:

```opal
@retry(times=3) {
    @aws.instance.ssh(host="prod") {
        cat /var/log/app.log
    }
}
```

**Execution flow:**
1. Executor calls `retryHandler(ctx, block=[ssh step])`
2. Retry handler loops 3 times, calling `ctx.ExecuteBlock(block)` each time
3. Executor receives ssh step, calls `sshHandler(ctx, block=[cat command])`
4. SSH handler establishes connection, wraps context, calls `sshCtx.ExecuteBlock(block)`
5. Executor receives cat command, calls `shellHandler(ctx, block=[])`
6. Shell handler executes command and returns exit code
7. Exit code bubbles back up through SSH handler → Retry handler → Executor

Each decorator wraps the execution context, creating a chain of responsibility where:
- **Retry** controls whether to retry
- **SSH** controls how to execute (on remote host)
- **Shell** controls what to execute (the actual command)

### Context Wrapping: Modifying Execution Behavior

Decorators can wrap the execution context to modify behavior:

```go
type SSHExecutionContext struct {
    parent ExecutionContext  // Original context
    conn   *ssh.Client       // SSH connection
}

func (s *SSHExecutionContext) ExecuteBlock(steps []Step) (int, error) {
    // Redirect stdout/stderr through SSH
    // Run commands on remote host
    // Return exit code
}
```

This allows decorators to:
- **Redirect I/O** - Capture, filter, or transform output
- **Modify environment** - Set variables, change working directory
- **Add constraints** - Enforce timeouts, resource limits
- **Control flow** - Retry, parallelize, conditionally execute

### Why This Pattern Works

**1. Separation of Concerns**
- Executor doesn't know about SSH, retry logic, or timeouts
- Decorators don't know about plan structure or step ordering
- Each component has a single responsibility

**2. Extensibility**
- New decorators are just new handlers registered in the registry
- No changes to executor needed
- External packages can provide decorators

**3. Composability**
- Decorators can be nested arbitrarily deep
- Each decorator is independent and testable
- Composition is declarative (visible in source code)

**4. Observability**
- Each decorator can emit telemetry
- Execution context provides hooks for logging
- Full traceability without special cases

## Transport Abstraction: Remote Execution

The Transport abstraction enables decorators to execute commands in different environments (local machine, remote servers, containers, cloud instances) while preserving Opal's security model and execution semantics.

**Current Status:** LocalTransport is implemented. Remote transports (SSH, Docker, cloud) follow the same interface pattern and can be added without breaking changes.

### The Two-Environment Model

Opal distinguishes between **plan-time** and **execution-time** environments:

| Environment | When | Purpose | Syntax | Example |
|-------------|------|---------|--------|---------|
| **Plan-Time** | Planning | Resolve value decorators | `@env.HOME` | `var replicas = @env.REPLICAS` → `"3"` |
| **Execution-Time** | Execution | Command environment | `$HOME` | `echo $HOME` → local: `/home/user`, remote: `/home/deploy` |

**Key insight**: Value decorators (`@env.X`) resolve at plan-time using the **local** environment, even inside remote transport decorators. This is intentional - it enables plan-time contract verification.

### TransportScope: Declaring Decorator Scope

Value decorators declare where they can resolve using the `TransportScope` interface:

```go
// TransportScope defines where a value decorator can resolve
type TransportScope uint8

const (
    ScopeRootOnly     TransportScope = iota  // @env, @file.read - local only, plan-time
    ScopeAgnostic                            // @var, @random - anywhere, plan-seeded
    ScopeRemoteAware                         // @proc.env(transport=...) - explicit remote (future)
)

// ValueScopeProvider is an optional interface that value decorators implement
type ValueScopeProvider interface {
    TransportScope() TransportScope
}
```

**Example: @env decorator declares ScopeRootOnly**:

```go
type envDecorator struct{}

func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error) {
    // Read from local environment
    envVar := (*args.Primary).(string)
    value, exists := ctx.Env[envVar]
    // ...
    return value, nil
}

// Implements ValueScopeProvider
func (e *envDecorator) TransportScope() types.TransportScope {
    return types.ScopeRootOnly  // Can only resolve at root (plan-time, local)
}

// Register
decorator := &envDecorator{}
types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle)
```

**Validation**:
- Parser tracks `transportDepth` (increments when entering `@ssh.connect`, etc.)
- For each value decorator, checks `scope := registry.GetTransportScope(decoratorName)`
- If `scope == ScopeRootOnly && transportDepth > 0` → validation error
- Default for decorators without interface: `ScopeAgnostic` (safe, works anywhere)

**Benefits**:
- ✅ Plugin-friendly: decorators self-declare via interface
- ✅ No hard-coded decorator names in validator
- ✅ Type-safe: compiler enforces interface
- ✅ Optional: decorators without interface get sensible defaults

### Transport Interface

```go
// Transport abstracts command execution for different environments
type Transport interface {
    // Exec executes a command and returns exit code
    // All I/O flows through provided readers/writers (scrubber-compatible)
    Exec(ctx context.Context, argv []string, opts ExecOpts) (exitCode int, err error)
    
    // Put transfers a file to the destination
    Put(ctx context.Context, src io.Reader, dst string, mode fs.FileMode) error
    
    // Get retrieves a file from the source
    Get(ctx context.Context, src string, dst io.Writer) error
    
    // Close cleans up transport resources
    Close() error
}

type ExecOpts struct {
    Stdin  io.Reader         // Command stdin
    Stdout io.Writer         // Command stdout (scrubbed)
    Stderr io.Writer         // Command stderr (scrubbed)
    Env    map[string]string // Decorator-added environment variables
    Dir    string            // Working directory
}
```

### Transport Implementation Pattern

All transports follow the same interface pattern, enabling consistent behavior across execution environments:

**LocalTransport** (implemented):
- Executes commands using `os/exec`
- Base environment: `os.Environ()`
- File operations: Local filesystem
- Process groups: `Setpgid=true` for clean cancellation

**Remote Transports** (general pattern):
- Execute commands in remote/isolated environments
- Base environment: Target environment's native environment
- File operations: Transport-specific (SFTP, docker cp, S3, etc.)
- Cancellation: Transport-specific cleanup (close SSH, stop container, etc.)

**Key properties:**
- Same interface for all transports
- Decorators don't need transport-specific code
- I/O flows through same scrubbing pipeline
- Context cancellation propagates correctly

### How Decorators Use Transport

**Pattern**: Decorators wrap ExecutionContext and intercept command execution to redirect through custom transport.

**Example pattern (remote execution decorator):**

```go
// Remote execution decorator wraps execution context
func remoteConnectHandler(ctx sdk.ExecutionContext, block []sdk.Step) (int, error) {
    host := ctx.ArgString("host")
    
    // Establish connection to remote environment
    transport, err := executor.NewRemoteTransport(host)
    if err != nil {
        return 127, err
    }
    defer transport.Close()
    
    // Wrap context to use remote transport
    remoteCtx := &remoteExecutionContext{
        parent:    ctx,
        transport: transport,
    }
    
    // Execute block with remote context
    return remoteCtx.ExecuteBlock(block)
}

// remoteExecutionContext wraps ExecutionContext to use remote transport
type remoteExecutionContext struct {
    parent    sdk.ExecutionContext
    transport executor.Transport
}

// ExecuteBlock intercepts command execution and redirects through transport
func (r *remoteExecutionContext) ExecuteBlock(steps []sdk.Step) (int, error) {
    for _, step := range steps {
        if isShellCommand(step) {
            // Use remote transport instead of local
            exitCode, err := r.executeShellViaTransport(step)
            if exitCode != 0 || err != nil {
                return exitCode, err
            }
        } else {
            // Other decorators delegate to parent
            exitCode, err := r.parent.ExecuteBlock([]sdk.Step{step})
            if exitCode != 0 || err != nil {
                return exitCode, err
            }
        }
    }
    return 0, nil
}
```

### Security Guarantees

1. **I/O Scrubbing Preserved**: Transport receives `io.Writer` for stdout/stderr - scrubber sits above
2. **No Bypass**: Decorators can't bypass scrubber by using transport directly
3. **Connection Security**: Credentials and connection details managed by transport implementation
4. **File Transfer Safety**: Put/Get respect file permissions and ownership

### Environment Capture and Propagation

**Key principle:** Environments are captured once per session and modified through copy-on-write.

#### Session Environment Capture

**Local session:**
```go
session := decorator.NewLocalSession()
// Captures os.Environ() ONCE at creation time
// This snapshot is immutable - won't see future os.Setenv() calls
```

**Remote session (idempotent transports):**
```opal
@ssh.connect(host="remote", env={"VERSION": "3.0"}) {
    # 1. SSH connects to remote server
    # 2. Captures remote environment (one-time snapshot)
    # 3. Merges with env={} overrides: {"VERSION": "3.0"}
    # 4. Creates SSH session with merged environment
    # 5. Session environment is immutable from this point
}
```

**Environment precedence (highest to lowest):**
1. Decorator `env={}` overrides (e.g., `@ssh.connect(env={...})`)
2. Transport's native environment (remote server, container, etc.)
3. No defaults - if not set, variable doesn't exist

#### How `@env` Resolves

**`@env.X` reads from the current session's environment:**

```opal
# Root context (local session)
var user = @env.USER  # Reads from os.Environ() snapshot → "alice"

@ssh.connect(host="remote", env={"VERSION": "3.0"}) {
    # SSH session context (remote session)
    var remote_user = @env.USER     # Reads from remote environment → "deploy"
    var version = @env.VERSION      # Reads from env={} override → "3.0"
}
```

**Both `@env.X` and `$X` read from the same session environment:**

```opal
@ssh.connect(host="remote", env={"VERSION": "3.0"}) {
    var v1 = @env.VERSION  # Plan-time: reads from session → "3.0"
    echo $VERSION          # Execution-time: reads from session → "3.0"
}
```

#### Decorator Environment Modification (Copy-on-Write)

Decorators modify environment through context wrapping:

```opal
# Root context: USER=alice, DB=<not set>

@env(DB="prod") {
    var db1 = @env.DB      # "prod" (decorator override)
    var user1 = @env.USER  # "alice" (inherited from parent)
    
    @env(DB="staging") {
        var db2 = @env.DB  # "staging" (inner override)
    }
    
    var db3 = @env.DB  # "prod" (outer context unchanged)
}

var db4 = @env.DB  # <not set> (root context unchanged)
```

**How it works:**
1. `@env(DB="prod")` calls `ctx.WithEnviron({"DB": "prod"})`
2. Creates new context with `session.WithEnv({"DB": "prod"})`
3. New session = parent environment + {"DB": "prod"}
4. Original context/session unchanged (immutable)

#### Environment Isolation Between Transports

**Critical constraint:** Environments do NOT automatically pass between transport boundaries.

```opal
# Local environment: USER=alice, HOME=/home/alice
# Remote environment: USER=deploy, HOME=/home/deploy

var local_user = @env.USER  # "alice" from local session

@ssh.connect(host="remote") {
    var remote_user = @env.USER  # "deploy" from remote session (NOT "alice")
    
    # ❌ Local environment NOT inherited
    # Remote session has no knowledge of local USER=alice
}
```

**To pass values between transports, use explicit `env={}` argument:**

```opal
var local_user = @env.USER  # "alice" from local session

@ssh.connect(host="remote", env={"DEPLOYER": @var.local_user}) {
    var deployer = @env.DEPLOYER  # "alice" (explicitly passed)
    var remote_user = @env.USER   # "deploy" (remote native)
}
```

**Why this design:**
- **Security:** Prevents accidental leakage of local credentials to remote
- **Clarity:** Explicit about what crosses transport boundaries
- **Correctness:** Remote environment reflects actual remote state
- **Predictability:** No magic environment merging or inheritance

#### Idempotent vs Non-Idempotent Decorators

**Idempotent transports** (connect at plan-time):
- Connect to **existing** resources (servers, containers)
- Safe to connect during planning (read-only, no side effects)
- `@env` can be used in blocks (reads from connected session)
- Examples: `@ssh.connect`, `@docker.exec`

```opal
@ssh.connect(host="existing-server", env={"VERSION": "3.0"}) {
    var v = @env.VERSION  # ✅ OK: SSH connects at plan-time
                          # Reads from SSH session environment
}
```

**Non-idempotent decorators** (no plan-time connection):
- **Create** new resources (instances, databases)
- Unsafe to connect during planning (resource doesn't exist yet)
- `@env` FORBIDDEN in blocks (can't connect to non-existent resource)
- Examples: `@aws.instance.deploy`, `@aws.rds.deploy`

```opal
@aws.instance.deploy(name="web") {
    var home = @env.HOME  # ❌ ERROR: Instance doesn't exist yet
                          # Can't connect at plan-time
    
    echo $HOME  # ✅ OK: Shell variable resolves at execution-time
}
```

**Error message:**
```
ERROR: @env cannot be used inside @aws.instance.deploy
Reason: Instance might not exist during planning (non-idempotent)
Solution: Use shell variables ($HOME, $USER) which resolve at execution-time
```

**Why this distinction:**
- Plan-time must be idempotent (read-only, no side effects)
- Creating resources violates plan-verify-execute model
- Only execution-time shell variables are safe in provisioning blocks

#### Resolution Timing Summary

| Context | `@env.X` | `$X` (shell variable) |
|---------|----------|----------------------|
| **Local** | Reads from `os.Environ()` snapshot | Reads from local shell environment |
| **Idempotent transport** | Reads from transport session (plan-time) | Reads from transport session (execution-time) |
| **Non-idempotent decorator** | ❌ Forbidden (can't connect) | ✅ Reads from transport session (execution-time) |

**Key insight:** Both `@env.X` and `$X` read from the same session environment, just at different times (plan vs execution).

### Example Usage Patterns

**Local execution:**
```opal
var replicas = @env.REPLICAS  # Reads from os.Environ() snapshot
kubectl scale --replicas=@var.replicas deployment/app
```

**Remote execution with @env in transport block:**
```opal
var local_user = @env.USER  # "alice" from local session

@ssh.connect(host="prod-server", user="deploy", env={"DATABASE_URL": "postgres://prod"}) {
    # @env reads from SSH session environment
    var remote_user = @env.USER         # "deploy" (remote native)
    var db_url = @env.DATABASE_URL      # "postgres://prod" (env={} override)
    
    # @var uses plan-time resolved value
    echo "Deployed by: @var.local_user"  # "alice"
    
    # Shell variables read from same SSH session
    echo "Running as: $USER"             # "deploy"
    echo "Connecting to: $DATABASE_URL"  # "postgres://prod"
}
```

**Container execution with isolated environment:**
```opal
@docker.exec(container="app-prod", env={"NODE_ENV": "production"}) {
    # @env reads from container session
    var env = @env.NODE_ENV  # "production" (env={} override)
    var user = @env.USER     # Container's USER (not local)
    
    # Shell variables read from same container session
    echo "Environment: $NODE_ENV"  # "production"
    echo "Container user: $USER"   # Container's USER
}
```

**Parallel deployment to multiple targets:**
```opal
var version = @env.APP_VERSION  # "3.0" from local session

@parallel {
    @ssh.connect(host="web-1", env={"VERSION": @var.version}) {
        # @env.VERSION reads "3.0" from SSH session
        var v = @env.VERSION
        ./deploy.sh $VERSION  # Shell variable also "3.0"
    }
    @ssh.connect(host="web-2", env={"VERSION": @var.version}) {
        var v = @env.VERSION  # "3.0"
        ./deploy.sh $VERSION
    }
    @ssh.connect(host="web-3", env={"VERSION": @var.version}) {
        var v = @env.VERSION  # "3.0"
        ./deploy.sh $VERSION
    }
}
```

**Provisioning with non-idempotent decorator:**
```opal
var db_password = @env.DB_PASSWORD  # Resolve at root (local session)

@aws.rds.deploy(name="app-db", engine="postgres") {
    # ❌ @env.DB_PASSWORD forbidden here (RDS doesn't exist yet)
    
    # ✅ Use @var for plan-time values
    psql -c "ALTER USER postgres PASSWORD '@var.db_password';"
    
    # ✅ Shell variables OK (resolve at execution-time when RDS exists)
    echo "Database host: $RDS_ENDPOINT"
}
```

### Design Principles

**1. Transport as Implementation Detail**
- Transport is NOT a first-class ExecutionContext member
- Decorators wrap context and use transport internally
- Follows "decorators wrap context" pattern

**2. Security First**
- All I/O flows through scrubber
- Transport cannot bypass security model
- Connection credentials managed securely

**3. Scalable**
- No hard-coded decorator names
- Value decorators self-declare scope via `ValueScopeProvider` interface
- Execution decorators self-declare transport switching via schema flag (for now)
- Plugin-friendly: decorators just implement interfaces

**4. Clear Semantics**
- Plan-time vs execution-time distinction is explicit
- Validation prevents confusing usage patterns
- Error messages guide users to correct approach

## Steps, Decorators, and Operators

Understanding the distinction between steps, decorators, and operators is critical to Opal's execution model.

### What is a Step?

A **step** is a unit of work in Opal - one line of code that performs an action. Steps are the building blocks of execution plans.

```opal
// Three steps (three lines)
echo "First"
echo "Second"
echo "Third"
```

**Key insight**: Newlines separate steps. Each step is independently controlled by Opal's execution engine.

### Operators: Intra-Step Control Flow

**Operators** (`&&`, `||`, `|`, `;`) control flow **within a single step**. Opal implements bash-compatible operator semantics for cross-platform consistency.

```opal
// ONE step with operators (Opal controls flow within step)
echo "First" && echo "Second" || echo "Fallback"
```

When this executes:
- Opal sees **one step** with multiple commands
- Opal executes each command sequentially, applying operator logic
- Operator semantics match bash behavior exactly
- Result: Cross-platform consistency (same behavior on Windows, Linux, macOS)

**Operator semantics** (Opal-controlled, bash-compatible):
- `&&` - Execute next command only if previous succeeded (exit 0)
- `||` - Execute next command only if previous failed (exit non-zero)
- `|` - Pipe stdout of previous command to stdin of next (concurrent execution with streaming)
- `;` - Execute commands sequentially regardless of exit codes

**Operator precedence** (high to low): `|` > `>`, `>>` > `&&` > `||` > `;` > newlines

See [SPECIFICATION.md](SPECIFICATION.md#line-by-line-execution) for user-facing operator rules.

### Newlines: Inter-Step Boundaries

**Newlines** separate steps and give Opal control over execution order, error handling, and flow.

```opal
// TWO steps (Opal controls flow between steps)
echo "First"
echo "Second"
```

When this executes:
- Opal sees **two steps**
- Step 1: `@shell("echo \"First\"")`
- Step 2: `@shell("echo \"Second\"")`
- Opal controls: Should step 2 run? When? In parallel? With retry?
- Opal can log, time, and track each step independently

**Newline semantics** (Opal-controlled):
- Sequential execution by default
- Fail-fast: Stop on first error (unless wrapped in `@retry` or `try/catch`)
- Independent logging and telemetry per step
- Parallelization possible with `@parallel`

### Decorators: Work Execution

**Decorators** wrap steps and control how they execute. All work in Opal is performed by decorators.

```opal
// Explicit decorator
@retry(3) {
    curl https://api.example.com/deploy
}

// Implicit @shell decorator (parser converts)
echo "Hello, World!"  // Becomes: @shell("echo \"Hello, World!\"")
```

**Decorator responsibilities**:
- Execute the actual work (shell commands, API calls, etc.)
- Handle errors and retries
- Control parallelism and concurrency
- Inject values and interpolate variables

### Examples: Operators vs Newlines

**Example 1: Operators (Opal controls)**
```opal
// ONE step - Opal handles && logic
mkdir -p /tmp/build && cd /tmp/build && npm install
```

If `mkdir` fails, Opal stops and never runs `cd` or `npm install`. Opal sees one step with three commands and applies `&&` semantics.

**Example 2: Newlines (Opal controls)**
```opal
// THREE steps - Opal handles each independently
mkdir -p /tmp/build
cd /tmp/build
npm install
```

If `mkdir` fails, Opal stops execution and never runs `cd` or `npm install`. Each step is logged separately with timing and exit codes.

**Example 3: Mixed (both)**
```opal
// TWO steps - Opal controls within and between
mkdir -p /tmp/build && cd /tmp/build
npm install && npm run build
```

Step 1: `mkdir -p /tmp/build && cd /tmp/build` (Opal handles `&&`)
Step 2: `npm install && npm run build` (Opal handles `&&`)

Opal controls whether step 2 runs based on step 1's exit code (newline = fail-fast).

### Why This Matters

**For plan generation**:
- Operators create multiple commands within a single step
- Newlines create distinct steps in the plan
- Each step gets a unique ID and can be tracked independently

**For execution**:
- Operators are Opal's responsibility (cross-platform consistency)
- Newlines are Opal's responsibility (logging, telemetry, error handling)
- Decorators wrap steps and control execution behavior
- All decorators automatically support operators (no decorator-specific code needed)

**For contract verification**:
- Operators are part of the step structure (command count, operator types)
- Steps are the unit of comparison (step count, step order, step content)
- Changing operators changes the step hash, failing verification

### Design Principle: Cross-Platform Consistency

**Opal controls all operators** to ensure consistent behavior across platforms:
- Same `&&`, `||`, `;` semantics on Windows, Linux, macOS
- No dependency on shell-specific behavior
- Operators work with all decorators (`@retry`, `@timeout`, etc.)

By implementing bash-compatible operator semantics in Go, Opal provides the familiarity of bash with the reliability of cross-platform execution.

## TreeNode Execution Model

Opal represents operator precedence and execution flow using a tree structure. This separates parsing from execution and enables recursive evaluation of complex command chains.

### TreeNode Hierarchy

```go
type TreeNode interface { isTreeNode() }

// Leaf node - single decorator invocation
type CommandNode struct {
    Name  string                 // Decorator name: "shell", "retry", "parallel"
    Args  map[string]interface{} // Decorator arguments (typed values)
    Block []Step                 // Nested steps (for decorators with blocks)
}

// Operator nodes - combine multiple commands
type PipelineNode struct {
    Commands []TreeNode  // cmd1 | cmd2 | cmd3 (concurrent execution)
}

type AndNode struct {
    Left, Right TreeNode  // cmd1 && cmd2 (short-circuit on failure)
}

type OrNode struct {
    Left, Right TreeNode  // cmd1 || cmd2 (short-circuit on success)
}

type SequenceNode struct {
    Nodes []TreeNode  // cmd1; cmd2; cmd3 (execute all, return last exit)
}

type RedirectNode struct {
    Source TreeNode  // Command producing output
    Target string    // File path or sink decorator
    Append bool      // true for >>, false for >
}
```

### Operator Precedence

Operators are parsed into tree structure following precedence (high to low):

1. **Pipe (`|`)** - Highest precedence, creates PipelineNode
2. **Redirect (`>`, `>>`)** - Creates RedirectNode wrapping source
3. **And (`&&`)** - Creates AndNode
4. **Or (`||`)** - Creates OrNode
5. **Sequence (`;`)** - Lowest precedence, creates SequenceNode

**Note:** This matches the user-facing precedence rules in [SPECIFICATION.md](SPECIFICATION.md#line-by-line-execution).

**Example:**
```opal
echo "a" | grep "a" && echo "b" || echo "c" > out.txt
```

**Parsed as:**
```
RedirectNode {
  Source: OrNode {
    Left: AndNode {
      Left: PipelineNode {
        Commands: [
          CommandNode{Name: "shell", Args: {"command": "echo \"a\""}},
          CommandNode{Name: "shell", Args: {"command": "grep \"a\""}}
        ]
      },
      Right: CommandNode{Name: "shell", Args: {"command": "echo \"b\""}}
    },
    Right: CommandNode{Name: "shell", Args: {"command": "echo \"c\""}}
  },
  Target: "out.txt",
  Append: false
}
```

### Execution Semantics

**PipelineNode:**
- All commands run concurrently (bash behavior)
- Stdout→stdin streaming via `os.Pipe()` (kernel SIGPIPE semantics)
- Returns exit code of last command
- EPIPE normalized to success for intermediate writers

**AndNode:**
- Execute left first
- If left succeeds (exit 0), execute right
- If left fails, short-circuit (skip right)
- Returns exit code of last executed command

**OrNode:**
- Execute left first
- If left succeeds (exit 0), short-circuit (skip right)
- If left fails, execute right
- Returns exit code of last executed command

**SequenceNode:**
- Execute all nodes in order
- Never short-circuit (always run all)
- Returns exit code of last node

**RedirectNode:**
- Execute source command
- Redirect stdout to target (file or sink)
- Stderr always goes to terminal (POSIX compliance)
- Returns source's exit code

### Streaming I/O Implementation

**Design Choice:** `os.Pipe()` instead of `io.Pipe()`

Opal uses `os.Pipe()` for process-to-process pipes to enable proper SIGPIPE semantics:

**Why os.Pipe():**
- Kernel sends SIGPIPE to writers when readers close
- Enables proper `yes | head -n1` termination (writer gets EPIPE)
- Direct process-to-process pipes (no Go buffering layer)
- Concurrent execution (bash-compatible)
- Natural backpressure via kernel pipe buffers

**Why not io.Pipe():**
- Go-level pipes don't propagate SIGPIPE
- Writers block indefinitely when readers close early
- Requires manual pipe cleanup on cancellation
- Adds unnecessary copy goroutines

**Implementation:** See `runtime/executor/executor.go` lines 239-342

**Example:**
```opal
yes | head -n1  // Completes in ~5ms (not timeout)
```

Without `os.Pipe()`, `yes` would run forever because it never receives SIGPIPE when `head` exits.

### How Operators Work with Decorators

**Key insight:** Decorators wrap entire steps, including their operator trees. This means decorators automatically support all operators without special handling.

**Example:**
```opal
@retry(attempts=3) {
    npm run build && npm test
}
```

**Execution flow:**
1. Retry decorator receives step with AndNode tree
2. Retry executes the tree (build && test)
3. If tree fails (either command fails), retry executes again
4. Retry doesn't need to know about `&&` - it just executes the tree

**Another example:**
```opal
@timeout(5m) {
    kubectl logs api | grep ERROR > errors.txt
}
```

**Execution flow:**
1. Timeout decorator receives step with RedirectNode(PipelineNode(...))
2. Timeout starts timer and executes the tree
3. Pipeline runs concurrently, redirect captures output
4. If timeout expires, context cancellation kills all processes
5. Timeout doesn't need to know about `|` or `>` - it just manages context

**This design enables:**
- Composability: Any decorator works with any operator
- Simplicity: Decorators don't need operator-specific code
- Extensibility: New operators work with existing decorators
- Correctness: Operator semantics are consistent everywhere

## Session Abstraction and Execution Contexts

### Session: Ambient Execution Environment

The Session interface represents where and how commands execute (local machine, SSH remote, Docker container, etc.):

```go
type Session interface {
    // Execute command with arguments
    Run(ctx context.Context, argv []string, opts RunOpts) (Result, error)
    
    // File operations
    Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error
    Get(ctx context.Context, path string) ([]byte, error)
    
    // Environment management (copy-on-write)
    Env() map[string]string
    WithEnv(delta map[string]string) Session
    
    // Working directory management (copy-on-write)
    Cwd() string
    WithWorkdir(dir string) Session
    
    // Cleanup
    Close() error
}
```

**Key properties:**
- **Copy-on-write:** `WithEnv()` and `WithWorkdir()` return new sessions
- **Immutable:** Original session unchanged by modifications
- **Context-aware:** All operations accept `context.Context` for cancellation
- **Transport-agnostic:** Same interface for local, SSH, Docker, etc.

**Example - LocalSession:**
```go
session := decorator.NewLocalSession()

// Modify environment (returns new session)
prodSession := session.WithEnv(map[string]string{
    "ENV": "production",
    "REPLICAS": "3",
})

// Modify working directory (returns new session)
buildSession := prodSession.WithWorkdir("/tmp/build")

// Original session unchanged
fmt.Println(session.Env()["ENV"])  // "" (not set)
fmt.Println(prodSession.Env()["ENV"])  // "production"
```

### RunOpts: Per-Execution Configuration

```go
type RunOpts struct {
    Stdin  io.Reader // Streaming input (nil if not piped)
    Stdout io.Writer // Output destination (nil = capture in Result)
    Stderr io.Writer // Error output (nil = capture in Result)
    Dir    string    // Override session's working directory (optional)
}
```

**Stdin streaming:**
- Changed from `[]byte` to `io.Reader` for true streaming
- Enables pipelines: `cmd1 | cmd2` streams data without buffering
- Supports large inputs without memory exhaustion

**Stdout/Stderr routing:**
- `nil` = capture in `Result.Stdout`/`Result.Stderr` (default)
- Non-nil = write directly to provided writer (pipelines, redirects)

### Process Group Cancellation

LocalSession creates process groups to ensure clean cancellation:

```go
// Create new process group (Unix only)
if runtime.GOOS != "windows" {
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,  // Create new process group
    }
}

// On cancellation, kill entire process group
if runtime.GOOS != "windows" && cmd.Process != nil {
    // Negative PID kills entire process group
    syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
```

**Why process groups:**
- Kills entire process tree (parent + all children)
- Prevents zombie processes
- Critical for pipelines (kills all commands in pipeline)
- Example: `yes | head -n1` - both processes killed on cancel

## Variable Scoping via Scope Graph

### The Problem: Cross-Session Variable Leakage

Variables resolved in one session could leak to another session, exposing sensitive values:

```opal
var LOCAL_TOKEN = @env.GITHUB_TOKEN  # Resolved in local session

@ssh(host="untrusted-server") {
    curl -H "Auth: @var.LOCAL_TOKEN" ...  # ❌ Sends local token to remote!
}
```

**Security violation:** Local credentials sent to remote server.

### The Solution: Hierarchical Scope Graph

**Variables are lexically scoped to their session context.**

```
Scope Graph (Tree):

Root (local session)
├─ vars: { HOME: "/home/alice", TOKEN: "ghp_local123" }
├─ sessionID: "local"
│
├─ Child: @ssh(host="server1")
│  ├─ vars: { REMOTE_HOME: "/home/bob" }
│  ├─ sessionID: "ssh:server1"
│  ├─ parent: → Root
│  │
│  └─ Child: @docker(container="app")
│     ├─ vars: { CONTAINER_ID: "abc123" }
│     ├─ sessionID: "docker:abc123"
│     └─ parent: → ssh:server1
│
└─ Child: @ssh(host="server2")
   ├─ vars: { REMOTE_HOME: "/home/charlie" }
   ├─ sessionID: "ssh:server2"
   └─ parent: → Root
```

### Variable Resolution Algorithm

**Lookup via parent traversal:**

```
Resolve @var.HOME in docker scope:
1. Check current scope (docker) → not found
2. Check parent scope (ssh:server1) → not found  
3. Check parent scope (local) → found: "/home/alice"
4. Return value
```

**Automatic isolation between siblings:**

```
Resolve @var.REMOTE_HOME in ssh:server2:
1. Check current scope → found: "/home/charlie"

Resolve @var.REMOTE_HOME in ssh:server1:
1. Check current scope → found: "/home/bob"

# Different values! Each session has its own scope.
# No session checking needed - variables are simply not in scope.
```

### Implementation

```go
// ScopeGraph manages hierarchical variable scoping across sessions.
type ScopeGraph struct {
    root    *Scope
    current *Scope  // Current scope during planning/execution
}

// Scope represents a variable scope tied to a session context.
type Scope struct {
    id        string            // Unique scope ID
    sessionID string            // Session identifier from Session.ID()
    vars      map[string]VarEntry
    parent    *Scope            // Parent scope (nil for root)
    children  []*Scope          // Child scopes
    depth     int               // Distance from root
    path      []string          // Path from root (for debugging)
}

// Key operations:
func NewScopeGraph(rootSessionID string) *ScopeGraph
func (g *ScopeGraph) EnterScope(sessionID string)  // Create child scope
func (g *ScopeGraph) ExitScope() error              // Return to parent
func (g *ScopeGraph) Store(varName, origin string, value any)
func (g *ScopeGraph) Resolve(varName string) (any, *Scope, error)
```

### Transport Boundaries: Sealed Scopes

**Transport boundaries create sealed scopes** that block implicit variable access.

**What creates transport boundaries?**
- `@ssh(...)` - Remote execution over SSH
- `@docker(...)` - Container execution
- Any decorator that changes execution transport

**Control flow decorators do NOT create boundaries:**
- `@retry(...)` - Retry logic
- `@parallel(...)` - Parallel execution
- `@timeout(...)` - Timeout control

### Example: Sealed Boundaries Require Explicit Passing

```opal
var LOCAL_TOKEN = @env.GITHUB_TOKEN  # Stored in root scope

@ssh(host="server1") {
    # ❌ CANNOT access parent - this scope is SEALED
    # echo "Token: @var.LOCAL_TOKEN"  # Error: Transport boundary violation

    # ✅ Must pass explicitly via decorator parameters
    var REMOTE_TOKEN = @env.GITHUB_TOKEN  # Resolved in SSH session
    echo "Remote token: @var.REMOTE_TOKEN"  # ✅ Works (current scope)
}

# To pass variables across boundaries, use decorator parameters:
@ssh(host="server2", env={TOKEN: @var.LOCAL_TOKEN}) {
    # TOKEN is passed as environment variable
    echo "Token: $TOKEN"  # ✅ Works (passed via env parameter)

    # But still cannot access via @var
    # echo "@var.LOCAL_TOKEN"  # ❌ Error: Transport boundary violation
}

@ssh(host="server3") {
    # ❌ CANNOT access sibling scopes
    echo "@var.REMOTE_TOKEN"  # ❌ Error: variable not found
}
```

**Security properties:**
1. **Sibling isolation**: Variables from `server1` cannot leak to `server3`
2. **Boundary sealing**: Parent variables require explicit passing via parameters
3. **Explicit intent**: Each variable crossing must be declared in decorator parameters

### Benefits

**1. Automatic Isolation**
- No manual session checking needed
- Variables isolated by structure
- Secure by default

**2. Natural Semantics**
- Control flow maintains scope chain (like closures)
- Transport boundaries enforce explicit passing (like function parameters)
- Sibling scope isolation is automatic

**3. Efficient**
- O(depth) lookup, typical depth 2-3 levels
- No session compatibility checks
- Simple traversal algorithm

**4. Debuggable**
```go
func (g *ScopeGraph) DebugPrint() {
    // Visualize entire scope tree
}

// Output:
// root (session=local)
//   HOME = "/home/alice" (from @env.HOME)
//   ssh:server1 (session=ssh:server1)
//     REMOTE_HOME = "/home/bob" (from @env.HOME)
//   ssh:server2 (session=ssh:server2)
//     REMOTE_HOME = "/home/charlie" (from @env.HOME)
```

**5. Extensible**
- Variable shadowing
- Decorator parameters for explicit passing
- Scope introspection
- Immutable scopes

### Planner Integration

```go
// When entering @ssh block:
func (p *planner) planSSHBlock() error {
    sshSession, err := decorator.NewSSHSession(params)
    if err != nil {
        return err
    }
    
    // Enter new scope
    p.scopes.EnterScope(sshSession.ID())
    oldSession := p.session
    p.session = sshSession
    
    // Plan block contents (in new scope)
    err = p.planBlock()
    
    // Exit scope
    p.scopes.ExitScope()
    p.session = oldSession
    
    return err
}
```

## Two-Layer Architecture

```
Plan-time Layer (Metaprogramming):
├─ for loops unroll into concrete steps (deterministic)
├─ if/when conditionals select execution paths (deterministic)
├─ try/catch defines error handling structure (execution-dependent paths)
└─ AST represents all language constructs

Runtime Layer (Work Execution):
├─ @shell decorators execute commands
├─ @retry/@parallel decorators modify execution
├─ @var/@env decorators provide values
├─ try/catch path selection based on actual exceptions
└─ Unified decorator interfaces handle all work
```

**Key insight**: `try/catch` is a metaprogramming construct (not a decorator) that defines deterministic error handling paths. Unlike `for`/`if`/`when` which resolve to a single path at plan-time, `try/catch` creates multiple **known paths** where execution selects which one based on actual results (exceptions). The plan includes **all possible paths** through try/catch blocks.

## Dual-Path Architecture: Execution vs Tooling

Opal's parser produces a stream of events that can be consumed in two different ways:

### Path 1: Events → Plan (Execution)

For **runtime execution**, the interpreter consumes events directly to generate execution plans:

```
Source → Lexer → Parser → Events → Interpreter → Plan → Execute
                          ^^^^^^^^
                     No AST construction!
```

**Use cases:**
- CLI execution: `opal deploy production`
- Script execution: `opal run build.opl`
- CI/CD pipelines
- Automated workflows

**Benefits:**
- Fast plan generation
- Zero AST allocation overhead
- Natural branch pruning (skip unused code paths)
- Minimal memory footprint

### Path 2: Events → AST (Tooling)

For **development tooling**, events are materialized into a typed AST:

```
Source → Lexer → Parser → Events → AST Builder → Typed AST
                          ^^^^^^^^
                     Lazy construction
```

**Use cases:**
- LSP (Language Server Protocol): go-to-definition, find references, hover
- Code formatters: preserve comments and whitespace
- Linters: static analysis, style checking
- Documentation generators: extract function signatures
- Refactoring tools: rename, extract function

**Benefits:**
- Strongly typed node access
- Parent/child relationships
- Symbol table construction
- Semantic analysis
- Source location mapping

### When to Use Each Path

| Feature | Execution Path | Tooling Path |
|---------|---------------|--------------|
| **Memory** | Events only | Events + AST |
| **Use case** | Run commands | Analyze code |
| **Construction** | Never builds AST | Lazy AST from events |
| **Optimization** | Branch pruning | Full tree |

**Key insight**: The AST is **optional**. For execution, we never build it. For tooling, we build it lazily only when needed. This dual-path design gives us both speed (for execution) and rich analysis (for development).

**Implementation details**: See [AST_DESIGN.md](AST_DESIGN.md) for event-based parsing, zero-copy pipelines, and tooling integration.

## Plan Generation Process

Opal generates execution plans through a three-phase pipeline:

```
Source → Parse → Plan → Execute
         ↓       ↓       ↓
      Events  Contract  Work
```

**Phase 1: Parse** - Source code becomes parser events (no AST for execution path)
**Phase 2: Plan** - Events become deterministic execution contract with hash verification
**Phase 3: Execute** - Contract-verified execution performs the actual work

### Key Mechanisms

**Branch pruning**: Conditionals (`if`/`when`) evaluate at plan-time, only selected branch enters plan
```opal
when @var.ENV {
    "production" -> kubectl apply -f k8s/prod/  # Only this if ENV="production"
    "staging" -> kubectl apply -f k8s/staging/  # Pruned
}
```

**Loop unrolling**: `for` loops expand into concrete steps at plan-time
```opal
for service in ["api", "worker"] {
    kubectl scale deployment/@var.service --replicas=3
}
# Plan: Two concrete steps (api, worker)
```

**Parallel resolution**: Independent value decorators resolve concurrently
```opal
deploy: {
    @env.DATABASE_URL        # Resolve in parallel
    @aws.secret.api_key      # Resolve in parallel
    kubectl apply -f k8s/
}
```

**Performance**: Event-based pipeline avoids AST allocation for execution, achieving <10ms plan generation for typical scripts.

**See [AST_DESIGN.md](AST_DESIGN.md)** for implementation details: event streaming, zero-copy pipelines, and AST construction for tooling.

## Plan Format Implementation

Plans are **execution contracts** that capture resolved variables and determined execution paths. The planner consumes parser events to produce a plan, but the plan itself is a tree structure, not events.

### Planning Process (Event-Based Input)

```
Parser Events (syntax)
    ↓
[Planner consumes events]
    ↓
Plan (execution contract)
    - Variables resolved
    - Execution path determined
    - Hash placeholders generated
```

**Key distinction:** The planner is event-driven (consumes parser events), but the plan output is a tree structure (execution steps).

### Internal Representation (In-Memory)

Plans are execution trees with resolved values:

```go
type Plan struct {
    Header   PlanHeader              // Metadata (version, hashes, timestamp)
    Target   string                  // Function/command being executed
    Steps    []ExecutionStep         // Execution sequence (tree structure)
    values   map[string]ResolvedValue // Resolved decorators (never serialized)
    Telemetry   *PlanTelemetry       // Performance metrics
    DebugEvents []DebugEvent         // Debug trace
}

type ExecutionStep struct {
    // All steps are decorators (shell commands are @shell decorators)
    Decorator string                // "@shell", "@retry", "@parallel", etc.
    Args      map[string]interface{} // Decorator arguments
    Block     []ExecutionStep        // Nested steps for decorators with blocks
}

type ResolvedValue struct {
    Placeholder ValuePlaceholder    // opal:s:ID for display/hashing
    value       interface{}         // Actual value (memory only, never serialized)
}

type ValuePlaceholder struct {
    Length    int       // Character count
    Algorithm string    // "sha256" or "blake3"
    Hash      [32]byte  // Full 256-bit digest for verification
}
```

**Key design decisions:**
- **Tree structure**: Execution steps form a tree (not events)
- **Resolved ahead of time**: Variables interpolated, control flow determined during planning
- **Homogeneous values**: All decorators (@var, @env, @aws.secret) treated uniformly
- **Always resolve fresh**: Values never stored in plan files, always queried from reality
- **Placeholders only**: Serialized plans contain structure + hashes, never actual values

### Plan as Execution Contract

Plans serve two purposes:

**1. Resolve Variables Ahead of Time**

Before planning:
```opal
var replicas = @env.REPLICAS
kubectl scale --replicas=@var.replicas deployment/app
```

After planning (in Plan):
```go
Values: {
    "env.REPLICAS": ResolvedValue{Length: 1, Hash: [32]byte{...}},  // Hash of "3"
    "var.replicas": ResolvedValue{Length: 1, Hash: [32]byte{...}},
}
Steps: [
    ExecutionStep{
        Decorator: "@shell",
        Args: {"command": "kubectl scale --replicas=3 deployment/app"},  // Already interpolated!
    },
]
```

**2. Determine Execution Path Ahead of Time**

Before planning:
```opal
if @env.ENV == "production" {
    kubectl apply -f k8s/prod/
} else {
    kubectl apply -f k8s/dev/
}
```

After planning (if ENV="production"):
```go
Steps: [
    ExecutionStep{
        Decorator: "@shell",
        Args: {"command": "kubectl apply -f k8s/prod/"},
    },
]
// The else branch is PRUNED - not in the plan!
```

**Contract verification:** When executing with `--plan file.plan`, Opal replans fresh and compares hashes. If environment changed (REPLICAS went from "3" to "5"), hashes won't match and execution aborts.

### Serialization Format (.plan files)

Contract files use a binary format (encoding/gob for MVP, protobuf for production):

**MVP Format (encoding/gob):**
```go
// Simple Go serialization - handles tree structure automatically
func Encode(plan *Plan, w io.Writer) error {
    enc := gob.NewEncoder(w)
    return enc.Encode(plan)
}
```

**Production Format (protobuf - future):**
```
[Header: 32 bytes]
  Magic:      "OPAL" (4 bytes)
  Version:    uint16 (2 bytes) - major.minor
  Flags:      uint16 (2 bytes) - reserved
  Mode:       uint8 (1 byte)   - Quick/Resolved
  Reserved:   (7 bytes)
  StepCount:  uint32 (4 bytes)
  ValueCount: uint32 (4 bytes)
  Timestamp:  int64 (8 bytes)

[Hashes Section]
  SourceHash: [32 bytes] - SHA-256 of source code
  PlanHash:   [32 bytes] - SHA-256 of plan structure

[Target Section]
  TargetLen: uint16
  Target:    []byte  // "deploy", "hello", etc.

[Steps Section]
  Step[]:
    Kind:    uint8 (Shell=0, Decorator=1)
    DataLen: uint32
    Data:    []byte (command text or decorator info)
    // For decorators with blocks, nested steps follow

[Values Section]
  Value[]:
    KeyLen:    uint16
    Key:       []byte  // "var.REPLICAS", "env.HOME"
    ValueLen:  uint32  // Character count
    HashAlgo:  uint8   // SHA256=0, BLAKE3=1
    Hash:      [32]byte // Full 256-bit digest
```

**Why this approach:**
- **MVP (gob)**: Zero dependencies, handles Go types automatically, good enough for MVP
- **Production (protobuf)**: Better versioning, cross-language support, more compact
- **Tree structure**: Serializes execution steps directly (not events)
- **Full hashes**: 32-byte digests for security (not 6-char prefixes)

### Output Formats (Pluggable)

Plans can be formatted for different consumers via a pluggable interface:

```go
type PlanFormatter interface {
    Format(plan *Plan) ([]byte, error)
}
```

**Implemented formatters:**
- **TreeFormatter** - CLI human-readable tree view
- **JSONFormatter** - API/debugging structured output
- **BinaryFormatter** - Compact .plan contract files

**Future formatters** (designed, not yet implemented):
- **HTMLFormatter** - Web UI visualization
- **GraphQLFormatter** - Advanced query API
- **ProtobufFormatter** - gRPC API support

### Execution Modes

Plans support four execution modes:

**1. Direct Execution** (no plan file)
```bash
opal deploy
```
Flow: Source → Parse → Plan (resolve fresh) → Execute

**2. Quick Plan** (preview, defer expensive decorators)
```bash
opal deploy --dry-run
```
Flow: Source → Parse → Plan (cheap values only) → Display
- Resolves control flow and cheap decorators (@var, @env)
- Defers expensive decorators (@aws.secret, @http.get)
- Shows likely execution path

**3. Resolved Plan** (generate contract)
```bash
opal deploy --dry-run --resolve > prod.plan
```
Flow: Source → Parse → Plan (resolve ALL) → Serialize
- Resolves all value decorators (including expensive ones)
- Generates contract with hash placeholders
- Saves to .plan file for later verification

**4. Contract Execution** (verify + execute)
```bash
opal run --plan prod.plan
```
Flow: Load contract → Replan fresh → Compare hashes → Execute if match
- **Critical**: Plan files are NEVER executed directly
- Always replans from current source and reality
- Compares fresh plan hashes against contract
- Executes only if hashes match, aborts if different

**Why replan instead of execute?**
- Prevents executing stale plans against changed reality
- Detects drift (source changed, environment changed, infrastructure changed)
- Unlike Terraform (applies old plan to new state), Opal verifies current reality would produce same plan

### Hash Algorithm

**Default**: SHA-256 (widely supported, ~400 MB/s)
- Standard cryptographic hash
- Broad compatibility
- Sufficient security for contract verification

**Optional**: BLAKE3 via `--hash-algo=blake3` flag (~3 GB/s, 7x faster)
- Modern cryptographic hash
- Significantly faster for large values
- Requires explicit opt-in

### Value Placeholder Format

All resolved values use security placeholder format: `opal:s:ID`

Examples:
- `opal:s:3J98t56A` - single character (e.g., "3")
- `opal:s:7Kx9mN2p` - 32 characters (e.g., secret token)
- `opal:s:mQp4Tn8X` - 8 characters (e.g., hostname)

**Benefits:**
- **No value leakage** in plans or logs
- **Contract verification** via hash comparison
- **Debugging support** via length hints
- **Algorithm agility** for future hash upgrades

### Format Versioning

Plans include format version from day 1 for evolution:

**Version scheme**: `major.minor.patch`
- **Major**: Breaking changes to format structure
- **Minor**: Backward-compatible additions
- **Patch**: Bug fixes, no format changes

**Current version**: 1.0.0 (MVP)

**Future versions:**
- 1.1.0: Add compression (zstd), signature support
- 1.2.0: Extended metadata (git commit, author)
- 2.0.0: New event types, different hash defaults

### Observability

Plans include zero-overhead observability (like lexer/parser):

**Debug levels:**
- **DebugOff**: Zero overhead (default, production)
- **DebugPaths**: Method entry/exit tracing
- **DebugDetailed**: Event-level tracing

**Telemetry levels:**
- **TelemetryOff**: Zero overhead (default)
- **TelemetryBasic**: Counts only
- **TelemetryTiming**: Counts + timing

**Implementation**: Same pattern as lexer/parser - simple conditionals, no allocations when disabled.

## Plan Format Specification

This section defines the formal specification for plan serialization, versioning, and consumption by external tools.

### Plan Lifecycle and State Transitions

Plans evolve through distinct states during their lifecycle:

```
SOURCE CODE
    ↓
[Parse Events]
    ↓
QUICK PLAN (--dry-run)
    ├─ Cheap values resolved (@var, @env)
    ├─ Expensive values deferred (@aws.secret, @http.get)
    └─ Shows likely execution path
    ↓
RESOLVED PLAN (--dry-run --resolve)
    ├─ ALL values resolved
    ├─ Hash placeholders generated
    └─ Serialized to .plan file (CONTRACT)
    ↓
CONTRACT VERIFICATION (--plan file)
    ├─ Replan from current source + reality
    ├─ Compare fresh hashes vs contract
    ├─ MATCH → Execute
    └─ MISMATCH → Abort with diff
    ↓
EXECUTED
    ├─ Work performed
    └─ Execution log generated
```

**State transitions:**
- `Source → Quick Plan`: Parse + resolve cheap values
- `Quick Plan → Resolved Plan`: Resolve expensive values + serialize
- `Resolved Plan → Verified`: Replan + hash comparison
- `Verified → Executed`: Perform work
- `Verified → Drifted`: Hash mismatch, abort

**Terminal states:**
- `Executed`: Work completed successfully
- `Drifted`: Contract violated, execution aborted
- `Failed`: Execution error

### Serialization Layers

Plans have three distinct representations for different consumers:

| Layer | Purpose | Contains | Consumers | Format |
|-------|---------|----------|-----------|--------|
| **In-Memory Plan** | Runtime execution contract | `PlanHeader` + `ExecutionStep[]` + resolved values | Opal runtime | Go structs |
| **Contract Plan** | Persisted verification artifact | Header + Steps + Value placeholders + Provenance | `.plan` files, audit systems | Binary (gob/protobuf) |
| **View Plan** | Human/API consumption | Formatted representation | CLI, web UI, REST API | Tree/JSON/HTML |

**Key principle**: In-memory plans contain actual values (never serialized). Contract plans contain only structure + hash placeholders. View plans are derived from either.

### Binary Format Specification (.plan files)

**File extension**: `.plan`

**MIME type**: `application/x-opal-plan`

**Magic number**: `0x4F50414C` ("OPAL" in ASCII)

**Endianness**: Little-endian (all multi-byte integers)

**Alignment**: All sections 8-byte aligned with length prefixes

**Section ordering**: HEADER → HASH → TARGET → STEPS → VALUES → PROVENANCE → SIGNATURE (if flags set)

**Format version**: 1.0.0 (current)

**Hash digest policy**: All hash algorithms standardized to **256-bit (32-byte) output**
- SHA-256: Native 256-bit output
- BLAKE3: Configured for 256-bit output (extendable-output truncated)

#### Binary Layout

```
┌─────────────────────────────────────────────────────────────┐
│ HEADER SECTION (32 bytes, 8-byte aligned)                   │
├─────────────────────────────────────────────────────────────┤
│ Offset | Size | Type   | Field        | Description         │
│    0   |  4   | uint32 | Magic        | 0x4F50414C ("OPAL") │
│    4   |  2   | uint16 | VersionMajor | Format major version│
│    6   |  2   | uint16 | VersionMinor | Format minor version│
│    8   |  2   | uint16 | Flags        | See Flags section   │
│   10   |  1   | uint8  | Mode         | 0=Quick,1=Resolved  │
│   11   |  1   | uint8  | HashAlgo     | 0=SHA256,1=BLAKE3   │
│   12   |  4   | uint32 | StepCount    | Number of steps     │
│   16   |  4   | uint32 | ValueCount   | Number of values    │
│   20   |  4   | uint32 | ProvenanceLen| Provenance bytes    │
│   24   |  8   | int64  | Timestamp    | Unix epoch (UTC)    │
├─────────────────────────────────────────────────────────────┤
│ HASH SECTION (64 bytes, 8-byte aligned)                     │
├─────────────────────────────────────────────────────────────┤
│   32   | 32   | [32]u8 | SourceHash   | 256-bit digest      │
│   64   | 32   | [32]u8 | PlanHash     | 256-bit digest      │
├─────────────────────────────────────────────────────────────┤
│ TARGET SECTION (variable, 8-byte aligned)                   │
├─────────────────────────────────────────────────────────────┤
│   96   |  2   | uint16 | TargetLen    | UTF-8 length        │
│   98   |  T   | []u8   | Target       | "deploy", "hello"   │
│  98+T  |  P   | [P]u8  | Padding      | Align to 8 bytes    │
├─────────────────────────────────────────────────────────────┤
│ STEPS SECTION (variable, 8-byte aligned, zstd if COMPRESSED)│
├─────────────────────────────────────────────────────────────┤
│   N    |  4   | uint32 | DataLen      | JSON bytes          │
│  N+4   |  L   | []u8   | Data         | JSON decorator info │
│ N+4+L  |  P   | [P]u8  | Padding      | Align to 8 bytes    │
│   ...  | ...  | ...    | ...          | ...                 │
├─────────────────────────────────────────────────────────────┤
│ VALUES SECTION (variable, 8-byte aligned, zstd if COMPRESSED)│
├─────────────────────────────────────────────────────────────┤
│   N    |  2   | uint16 | KeyLength    | UTF-8 key length    │
│  N+2   |  K   | []u8   | Key          | e.g. "var.REPLICAS" │
│ N+2+K  |  4   | uint32 | ValueLength  | Character count     │
│ N+6+K  |  1   | uint8  | HashAlgo     | 0=SHA256,1=BLAKE3   │
│ N+7+K  | 32   | [32]u8 | HashDigest   | Full 256-bit hash   │
│ N+39+K |  1   | [1]u8  | Padding      | Align to 8 bytes    │
│   ...  | ...  | ...    | ...          | ...                 │
├─────────────────────────────────────────────────────────────┤
│ PROVENANCE SECTION (variable, 8-byte aligned)               │
├─────────────────────────────────────────────────────────────┤
│   P    |  4   | uint32 | Length       | Provenance bytes    │
│  P+4   |  L   | []u8   | Data         | JSON blob (UTF-8)   │
│ P+4+L  |  A   | [A]u8  | Padding      | Align to 8 bytes    │
├─────────────────────────────────────────────────────────────┤
│ SIGNATURE SECTION (variable, 8-byte aligned, if SIGNED)     │
├─────────────────────────────────────────────────────────────┤
│   S    |  1   | uint8  | SigAlgo      | 0=Ed25519           │
│  S+1   |  2   | uint16 | SigLength    | Signature bytes     │
│  S+3   |  L   | []u8   | Signature    | Detached signature  │
│ S+3+L  |  A   | [A]u8  | Padding      | Align to 8 bytes    │
└─────────────────────────────────────────────────────────────┘
```

**Step Data Format**:
- JSON-encoded decorator info (name, args, nested steps)
- Shell commands are `@shell` decorators with `command` arg
- All steps are decorators (unified model)

**Step ordering**: Pre-order traversal of execution tree (loops unrolled, conditionals pruned during planning).

**Note**: No separate "step types" - everything is a decorator. Shell commands use `@shell` decorator.

#### Header Flags

```go
const (
    FlagCompressed uint16 = 1 << 0  // Bit 0: EVENTS+VALUES are zstd-framed
    FlagSigned     uint16 = 1 << 1  // Bit 1: SIGNATURE section present
    // Bits 2-15: Reserved for future use
)
```

**Compression**: If `FlagCompressed` set, STEPS and VALUES sections are zstd-compressed independently. Each section prefixed with uncompressed length (uint32) before zstd frame.

**Signature**: If `FlagSigned` set, SIGNATURE section present at end. Signature covers HEADER+HASH+TARGET+STEPS+VALUES+PROVENANCE (everything except SIGNATURE itself).

#### Hash Algorithms

```go
const (
    HashSHA256  uint8 = 0  // SHA-256 (256-bit output)
    HashBLAKE3  uint8 = 1  // BLAKE3 (256-bit output, truncated)
    // 2-255: Reserved for future algorithms
)
```

**Note**: All hash algorithms produce exactly 32 bytes (256 bits) for consistency. BLAKE3's extendable output is truncated to 256 bits.

#### Plan Modes

```go
const (
    PlanModeQuick    uint8 = 0  // Quick plan (deferred expensive values)
    PlanModeResolved uint8 = 1  // Resolved plan (all values materialized)
    // 2-255: Reserved for future modes
)
```

**Note**: "Execution" is not a file mode - it's a runtime operation that uses Quick or Resolved plans.

#### Signature Algorithms

```go
const (
    SigEd25519 uint8 = 0  // Ed25519 (64-byte signature)
    // 1-255: Reserved for future algorithms
)
```

### JSON Format Specification (API)

**MIME type**: `application/json`

**Schema version**: 1.0.0

**Normalization**: Keys sorted alphabetically, no whitespace in compact mode

#### JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["header", "target", "steps", "values"],
  "properties": {
    "header": {
      "type": "object",
      "required": ["format_version", "source_hash", "plan_hash", "timestamp", "mode"],
      "properties": {
        "format_version": {
          "type": "string",
          "pattern": "^\\d+\\.\\d+\\.\\d+$",
          "description": "Semantic version (major.minor.patch)"
        },
        "source_hash": {
          "type": "string",
          "pattern": "^(sha256|blake3):[0-9a-f]{64}$",
          "description": "Hash of source code"
        },
        "plan_hash": {
          "type": "string",
          "pattern": "^(sha256|blake3):[0-9a-f]{64}$",
          "description": "Hash of plan structure"
        },
        "timestamp": {
          "type": "string",
          "format": "date-time",
          "description": "ISO 8601 timestamp (UTC)"
        },
        "mode": {
          "type": "string",
          "enum": ["quick", "resolved"],
          "description": "Plan generation mode"
        },
        "hash_algorithm": {
          "type": "string",
          "enum": ["sha256", "blake3"],
          "description": "Hash algorithm used for placeholders"
        }
      }
    },
    "target": {
      "type": "string",
      "description": "Function or command being executed (e.g., 'deploy', 'hello')"
    },
    "steps": {
      "type": "array",
      "description": "Execution steps (tree structure, all steps are decorators)",
      "items": {
        "type": "object",
        "required": ["decorator"],
        "properties": {
          "decorator": {
            "type": "string",
            "description": "Decorator name (e.g., '@shell', '@retry', '@parallel')"
          },
          "args": {
            "type": "object",
            "description": "Decorator arguments"
          },
          "block": {
            "type": "array",
            "description": "Nested steps (for decorators with blocks)",
            "items": { "$ref": "#/properties/steps/items" }
          }
        }
      }
    },
    "values": {
      "type": "object",
      "patternProperties": {
        "^(var|env|aws|http|k8s)\\..+$": {
          "type": "string",
          "pattern": "^<\\d+:(sha256|blake3):[0-9a-f]{6}>$",
          "description": "Value placeholder (display format, 6-char prefix)"
        }
      }
    },
    "provenance": {
      "type": "object",
      "description": "Plan generation metadata",
      "properties": {
        "compiler_version": {
          "type": "string",
          "description": "Opal compiler version (e.g., '1.0.0')"
        },
        "source_commit": {
          "type": "string",
          "description": "Git commit hash of source (if available)"
        },
        "generated_by": {
          "type": "string",
          "description": "User or system that generated plan"
        },
        "plugins": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": { "type": "string" },
              "version": { "type": "string" }
            }
          },
          "description": "Loaded plugins and versions"
        }
      }
    }
  }
}
```

**Note on placeholders**: JSON displays 6-char hash prefixes for readability. Binary format stores full 32-byte digests for security. Verification uses full digests.

#### Example JSON Plan

```json
{
  "header": {
    "format_version": "1.0.0",
    "source_hash": "sha256:a1b2c3d4e5f6...",
    "plan_hash": "sha256:x7y8z9a0b1c2...",
    "timestamp": "2025-10-12T20:00:00Z",
    "mode": "resolved",
    "hash_algorithm": "sha256"
  },
  "target": "deploy",
  "steps": [
    {
      "decorator": "@shell",
      "args": { "command": "kubectl apply -f k8s/prod/" }
    },
    {
      "decorator": "@shell",
      "args": { "command": "kubectl scale --replicas=3 deployment/app" }
    },
    {
      "decorator": "@retry",
      "args": { "times": 3, "delay": "2s" },
      "block": [
        {
          "decorator": "@shell",
          "args": { "command": "kubectl rollout status deployment/app" }
        }
      ]
    }
  ],
  "values": {
    "var.REPLICAS": "opal:s:3J98t56A",
    "env.HOME": "opal:s:nR5wKp2Y",
    "aws.secret.api_key": "opal:s:sQ9aTt6C"
  },
  "provenance": {
    "compiler_version": "1.0.0",
    "source_commit": "a1b2c3d4e5f6...",
    "generated_by": "user@hostname",
    "plugins": [
      { "name": "aws", "version": "1.0.0" },
      { "name": "k8s", "version": "1.2.0" }
    ]
  }
}
```

### Tree Format Specification (CLI)

**Purpose**: Human-readable plan visualization for CLI output

**Format**: UTF-8 text with box-drawing characters

**Structure**:
```
<command_name>:
├─ <step_1>
├─ <step_2>
│  ├─ <nested_step_2a>
│  └─ <nested_step_2b>
└─ <step_3>

Values:
  <key_1> = <placeholder_1>
  <key_2> = <placeholder_2>

Plan Hash: <algorithm>:<hash>
```

**Box-drawing characters**:
- `├─` Branch (not last child)
- `└─` Branch (last child)
- `│` Vertical continuation
- `   ` Indentation (3 spaces)

**Example**:
```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=opal:s:qN7yOr4A
└─ kubectl scale --replicas=opal:s:rP8zQs5B deployment/app

Values:
  var.REPLICAS = opal:s:rP8zQs5B
  env.HOME = opal:s:pL6xMq3Z

Plan Hash: sha256:xyz789...
```

### Format Versioning and Compatibility

**Versioning scheme**: Semantic versioning (major.minor.patch)

**Compatibility rules**:
- **Major version change**: Breaking changes, no backward compatibility
- **Minor version change**: Backward-compatible additions (new fields, new event types)
- **Patch version change**: Bug fixes, no format changes

**Version negotiation**:
1. Reader checks major version - must match exactly
2. Reader checks minor version - must be >= writer's minor version
3. Reader ignores unknown fields (forward compatibility)
4. Reader validates required fields (backward compatibility)

**Example evolution**:
- `1.0.0` → `1.1.0`: Add compression field (optional, readers can ignore)
- `1.1.0` → `1.2.0`: Add signature field (optional, readers can ignore)
- `1.2.0` → `2.0.0`: Change event encoding (breaking, requires major bump)

**Validation**:
- Plans with unsupported major version: **reject with error**
- Plans with newer minor version: **accept, ignore unknown fields**
- Plans with invalid structure: **reject with detailed error**

### Contract Verification Algorithm

When executing with a plan file (`opal run --plan prod.plan`):

```
1. Load contract plan from file
   - Deserialize binary/JSON
   - Validate format version
   - Extract placeholders

2. Replan from current source
   - Parse current source code
   - Resolve all value decorators fresh
   - Generate fresh plan with placeholders

3. Compare plan structures
   - Compare event sequences (must match exactly)
   - Compare value keys (must match exactly)
   - Compare placeholder hashes (must match exactly)

4. Verification outcomes
   - ALL match → Execute with fresh values
   - ANY mismatch → Abort with diff showing:
     * Which values changed
     * Which events differ
     * Suggested action (regenerate plan)

5. Execute (if verified)
   - Use fresh values (not contract values)
   - Log execution with contract reference
   - Generate execution report
```

**Hash comparison**:
- Contract stores full 32-byte digests in VALUES section
- Runtime recomputes full digests from fresh values
- Comparison uses full 256-bit hashes (timing-safe)
- Display uses 6-char hex prefix for readability
- Report first mismatch (fail fast)

**Why full digests in contract**: 6-char prefixes (~24 bits) insufficient for security. Full 256-bit digests prevent collisions and tampering. Display layer truncates for human readability.

**Drift error codes**:
```go
const (
    DriftSourceChanged   = "source_changed"    // Source code modified
    DriftEnvChanged      = "env_changed"       // Environment variables changed
    DriftInfraMissing    = "infra_missing"     // Infrastructure resource missing
    DriftInfraMutated    = "infra_mutated"     // Infrastructure state changed
    DriftValueChanged    = "value_changed"     // Generic value change
)
```

**Diff output** (on mismatch):
```
ERROR: Contract verification failed

Expected: kubectl scale --replicas=opal:s:3J98t56A deployment/app
Actual:   kubectl scale --replicas=opal:s:tR1bUv7D deployment/app

Value changed:
  var.REPLICAS
    Contract: opal:s:3J98t56A (was "3")
    Current:  opal:s:tR1bUv7D (now "5")

Drift Code: env_changed
Action: Run 'opal deploy --dry-run --resolve' to generate new plan
```

### External Tool Integration

**For Opal Cloud / Web UI**:
- Consume JSON format via REST API
- Display tree format for human review
- Store binary format for efficient storage
- Provide diff visualization for contract changes

**For CI/CD systems**:
- Generate resolved plans in CI pipeline
- Store as build artifacts
- Execute with contract verification in deployment
- Fail deployment if contract violated

**For audit systems**:
- Parse binary format for compliance review
- Extract value placeholders (no secrets exposed)
- Verify plan signatures (future)
- Generate audit trails

**For third-party tools**:
- Implement `PlanFormatter` interface
- Support custom output formats
- Consume JSON API for integration
- Respect format versioning rules

## Safety Guarantees

Opal guarantees that all operations halt with deterministic results.

### Plan-Time Safety

**Finite loops**: All loops must terminate during plan generation.
- `for item in collection` - collection size is known
- `while count > 0` - count value is resolved at plan-time
- Loop iteration happens during planning, not execution

**Command call DAG constraint**: Commands can call each other, but must form a directed acyclic graph.
- `fun` definitions called via `@cmd()` expand at plan-time with parameter binding
- Call graph analysis prevents cycles: `A → B → A` results in plan generation error  
- Parameters must be plan-time resolvable (value decorators, variables, literals)
- No dynamic dispatch - all calls resolved during planning

**Finite parallelism**: `@parallel` blocks have a known number of tasks after loop expansion.

### Runtime Safety

**User-controlled timeouts**: No automatic timeouts - users control when they want limits.
- Commands run until completion or manual termination (Ctrl+C)
- `@timeout(1h) { ... }` - explicit timeout when desired
- `--timeout 30m` flag - global safety net when needed
- Long-running processes (`dev servers`, `monitoring`) run naturally

**Resource limits**: Memory and process limits prevent system exhaustion.

### Determinism

**Reproducible plans**: Same source + environment = identical plan.
- Value decorators are referentially transparent
- Random values use cryptographic seeding (resolved plans only)
- Output ordering is deterministic

**Contract verification**: Resolved plans are execution contracts.
- Values re-resolved at runtime and hash-compared against plan
- Execution fails if any value changed since planning
- Exception: `try/catch` path selection based on actual runtime results

### Cancellation and Cleanup

**Graceful cancellation**: `finally` blocks run on interruption for safe cleanup.
- **First Ctrl+C**: Triggers cleanup sequence, shows "Cleaning up..."
- **Second Ctrl+C**: Force immediate termination, skips cleanup
- Allows resource cleanup (PIDs, temp files, containers) while providing escape hatch

## Decorator Design Requirements

When building decorators, follow these principles to maintain the contract model:

**Value decorators must be referentially transparent** during plan resolution. Non-deterministic value decorators (like `@http.get("time-api")`) will cause contract verification failures when plans are executed later.

**Execution decorators should be stateless**. Query current reality fresh each time rather than maintaining state between runs. This eliminates the complexity of state file management.

**Expose idempotency keys** so the same resolved plan can run multiple times safely. For example, `@aws.ec2.deploy` might use `region + name + instance_spec` as its key.

**Handle infrastructure drift gracefully**. When current infrastructure doesn't match plan expectations, provide clear error messages and suggested actions rather than cryptic failures.

## SDK for Decorator Authors

Opal provides a secure-by-default SDK in `core/sdk/` for building decorators:

### Secret Handling (`core/sdk/secret`)

**Core Principle:** Runtime controls all secret access through site-based authority. Secrets flow as opaque handles through planning, only unwrapped at authorized execution sites.

#### Security Model: Site-Based Authority

Secrets are accessible **only at their use-site**. No propagation to parent/child decorators.

```opal
var API_KEY = "sk-..."

@retry(apiKey=@var.API_KEY) {  # ✅ @retry can unwrap (authorized site)
    @timeout {                  # ❌ @timeout CANNOT unwrap (different site)
        @shell { ... }          # ❌ @shell CANNOT unwrap (different site)
    }
}
```

**Three-piece model:**
1. **Plan records use-sites** - `Plan.SecretUses[]` tracks DisplayID + AST path
2. **Runtime checks authority** - Executor verifies site before unwrap
3. **Unwrap fails if unauthorized** - Simple lookup, no complex leases

#### Decorator Parameter Classes

Decorators declare what parameters can accept via `ParamClass`:

```go
type ParamClass uint8

const (
    // Plain data/config. Must never receive a secret.
    ParamData ParamClass = iota
    
    // May receive a SecretRef (DisplayID/handle proxy) but cannot unwrap.
    ParamSecretRef
    
    // May receive a SecretRef and is allowed to unwrap it at the declared site.
    // Planner will emit a SecretUse(siteID, displayID) for this param.
    ParamSecretConsumer
)

type ParamSpec struct {
    Class       ParamClass
    Optional    bool
    Description string
}

type DecoratorSpec struct {
    Name   string
    Params map[string]ParamSpec
    MayConsumeSecrets bool  // If false, all consumer params downgraded to SecretRef
}
```

**Example specs:**
```go
var RetrySpec = DecoratorSpec{
    Name: "retry",
    Params: map[string]ParamSpec{
        "times":  {Class: ParamData},              // @retry(times=@var.retryCount) ✓
        "apiKey": {Class: ParamSecretConsumer, Optional: true}, // here-only unwrap
    },
    MayConsumeSecrets: true,
}

var ShellSpec = DecoratorSpec{
    Name: "shell",
    Params: map[string]ParamSpec{
        "cmd":         {Class: ParamData},
        "stdinSecret": {Class: ParamSecretConsumer, Optional: true}, // unwrap via FD only
    },
    MayConsumeSecrets: true,
}
```

**Plan-time validation:**
- Passing secret into `ParamData` → **plan-time error**
- Passing raw secret into `ParamSecretRef`/`ParamSecretConsumer` → **plan-time error** (must be SecretRef)
- `ParamSecretConsumer` records use-site for executor authorization

#### Secret Transport: FD/Stdin Only (Never Env/Argv)

**⚠️ CRITICAL SECURITY REQUIREMENT**

Secrets MUST be delivered via **file descriptors or stdin**, NEVER via environment variables or command-line arguments.

**Why env/argv are forbidden:**
- Visible in `/proc/PID/environ` and `/proc/PID/cmdline`
- Leaked in process tables (`ps aux`, `top`)
- Exposed in core dumps and crash logs
- Inherited by child processes
- Visible to monitoring tools

**Approved transport mechanisms:**

```go
// ✅ GOOD: Stdin delivery
cmd := exec.Command("sh", "-c", "read SECRET; echo $SECRET")
cmd.Stdin = strings.NewReader(secretValue)

// ✅ GOOD: Anonymous FD (Linux)
r, w := os.Pipe()
w.WriteString(secretValue)
w.Close()
cmd.ExtraFiles = []*os.File{r}  // FD 3
cmd := exec.Command("sh", "-c", "read SECRET <&3; echo $SECRET")

// ✅ GOOD: Memfd (Linux 3.17+)
fd, _ := unix.MemfdCreate("secret", 0)
unix.Write(fd, []byte(secretValue))
unix.Lseek(fd, 0, 0)
cmd.ExtraFiles = []*os.File{os.NewFile(uintptr(fd), "secret")}

// ❌ FORBIDDEN: Environment variables
cmd.Env = append(os.Environ(), "SECRET="+secretValue)  // NEVER DO THIS

// ❌ FORBIDDEN: Command-line arguments
cmd := exec.Command("app", "--secret", secretValue)  // NEVER DO THIS
```

**Decorator SDK enforcement:**

```go
// executor.Command() enforces FD/stdin delivery
func (e *Executor) DeliverSecret(cmd *exec.Cmd, secret string) error {
    // Only stdin or FD delivery allowed
    // Panics if attempted via env or argv
}
```

#### Plan-Time: Recording Use-Sites

```go
// Plan tracks where secrets are used with canonical site IDs
type SecretUse struct {
    DisplayID string  // "opal:v:3J98t56A"
    SiteID    string  // HMAC(planHash, canonicalPath) - unforgeable
    Site      string  // "root/retry[0]/params/apiKey" (human-readable diagnostic)
}

// Planner records each use with canonical site ID
func (p *Planner) recordSecretUse(displayID, stepID, paramName string) {
    canonicalPath := fmt.Sprintf("%s/params/%s", stepID, paramName)
    siteID := p.computeSiteID(canonicalPath)  // HMAC-based, unforgeable
    
    p.plan.SecretUses = append(p.plan.SecretUses, SecretUse{
        DisplayID: displayID,
        SiteID:    siteID,
        Site:      canonicalPath,  // For debugging only
    })
}

// Canonical site ID prevents forgery and refactor brittleness
func (p *Planner) computeSiteID(canonicalPath string) string {
    h := hmac.New(sha256.New, p.planKey)
    h.Write([]byte(canonicalPath))
    return base64.RawURLEncoding.EncodeToString(h.Sum(nil)[:16])
}
```

**Why canonical site IDs:**
- String comparison is spoofable and brittle across refactors
- HMAC-based IDs are unforgeable (require plan key)
- Executor compares `SiteID` only, `Site` is diagnostic
- Prevents decorator forgery attacks

#### Execution-Time: Authority Checks

```go
type ExecutionFrame struct {
    SiteID    string  // Canonical site ID (HMAC-based, immutable)
    Site      string  // Human-readable path (diagnostic only)
    StepID    string
    Decorator string
    ParamPath string
}

// Executor controls site (decorator cannot forge)
func (e *Executor) EnterFrame(stepID, decorator, paramPath string) *ExecutionFrame {
    canonicalPath := fmt.Sprintf("%s/%s/params/%s", stepID, decorator, paramPath)
    return &ExecutionFrame{
        SiteID:    e.computeSiteID(canonicalPath),  // Unforgeable
        Site:      canonicalPath,
        StepID:    stepID,
        Decorator: decorator,
        ParamPath: paramPath,
    }
}

// Build index for O(1) authority checks
type SecretAuthority struct {
    index map[string]map[string]bool  // DisplayID → set[SiteID]
}

func (e *Executor) buildAuthorityIndex() *SecretAuthority {
    index := make(map[string]map[string]bool)
    for _, use := range e.plan.SecretUses {
        if index[use.DisplayID] == nil {
            index[use.DisplayID] = make(map[string]bool)
        }
        index[use.DisplayID][use.SiteID] = true
    }
    return &SecretAuthority{index: index}
}

// Unwrap checks frame site (O(1) lookup)
func (e *Executor) unwrap(displayID string, frame *ExecutionFrame) (string, error) {
    // O(1) authority check via index
    if !e.authority.index[displayID][frame.SiteID] {
        // Audit failed attempt
        e.auditLog(AuditEvent{
            Timestamp: time.Now(),
            DisplayID: displayID,
            SiteID:    frame.SiteID,
            Site:      frame.Site,
            Decorator: frame.Decorator,
            Success:   false,
        })
        return "", fmt.Errorf("no authority to unwrap %s at %s", displayID, frame.Site)
    }
    
    handle := e.secretVault[displayID]
    value := handle.UnsafeUnwrap(e.capability)
    
    // Audit successful unwrap
    e.auditLog(AuditEvent{
        Timestamp: time.Now(),
        DisplayID: displayID,
        SiteID:    frame.SiteID,
        Site:      frame.Site,
        Decorator: frame.Decorator,
        ParamPath: frame.ParamPath,
        Success:   true,
    })
    
    return value, nil
}
```

**Audit logging (OpenTelemetry span events):**

Secret unwrap events are emitted as OpenTelemetry span events with standardized attributes:

```go
// Emit as span event (conforms to existing opal.* attribute convention)
span.AddEvent("secret.unwrap", trace.WithAttributes(
    attribute.String("opal.secret.display_id", displayID),
    attribute.String("opal.secret.site_id", frame.SiteID),
    attribute.String("opal.secret.site", frame.Site),
    attribute.String("opal.decorator", frame.Decorator),
    attribute.String("opal.param", frame.ParamPath),
    attribute.Bool("opal.secret.success", true),
))

// Failed unwrap attempts also logged
span.AddEvent("secret.unwrap.denied", trace.WithAttributes(
    attribute.String("opal.secret.display_id", displayID),
    attribute.String("opal.secret.site_id", frame.SiteID),
    attribute.String("opal.secret.site", frame.Site),
    attribute.String("opal.decorator", frame.Decorator),
    attribute.String("opal.param", frame.ParamPath),
    attribute.String("error", "no authority at site"),
))
```

**Attributes follow existing `opal.*` convention:**
- `opal.env`, `opal.target`, `opal.step_path` - Existing run context
- `opal.secret.display_id` - Which secret (DisplayID, never raw value)
- `opal.secret.site_id` - Canonical site ID (unforgeable)
- `opal.secret.site` - Human-readable path (diagnostic)
- `opal.decorator` - Which decorator requested unwrap
- `opal.param` - Which parameter
- `opal.secret.success` - Unwrap succeeded or denied

**Security properties:**
- Raw secret values NEVER logged
- DisplayIDs are opaque, safe to export
- Audit trail shows access patterns without exposing data
- Integrates with existing OpenTelemetry infrastructure

#### Variable Classification

Variables have different security levels:

```go
type VarClass int

const (
    VarClassData   VarClass = iota  // Public (no protection)
    VarClassConfig                  // Semi-public (no protection, but tracked)
    VarClassSecret                  // Protected (requires site authority)
)
```

**Examples:**
```opal
var NAME = "alice"          # VarClassData - no protection
var RETRIES = 3             # VarClassConfig - tracked but not protected
var API_KEY = "sk-..."      # VarClassSecret - requires authority to unwrap
```

#### Transport Boundary Enforcement

Secrets respect scope boundaries and cannot leak across transports:

```go
type VarTaint int

const (
    VarTaintAgnostic         VarTaint = iota  // Can cross any boundary
    VarTaintLocalOnly                         // Cannot leave local scope
    VarTaintBoundaryImported                  // Crossed via explicit import
)
```

**Boundary enforcement:**
```opal
var LOCAL_SECRET = "secret123"

# ❌ ERROR: Implicit cross-boundary use
@ssh(host="remote") {
    echo @var.LOCAL_SECRET
}

# ✅ OK: Explicit import via env
@ssh(host="remote", env={SECRET: @var.LOCAL_SECRET}) {
    # Inside SSH scope, capture as variable
    var SECRET = @env.SECRET
    
    @docker(container="app", env={SECRET: @var.SECRET}) {
        echo @var.SECRET  # ✅ OK - explicitly imported at each boundary
    }
}
```

**Plan-time validation:**
```go
func (p *Planner) checkTransportBoundary(varEntry VarEntry, originScope *Scope) error {
    currentScope := p.scopes.Current()
    
    // Check if crossing transport boundary
    if currentScope.transportDepth > originScope.transportDepth {
        if varEntry.Taint == VarTaintLocalOnly {
            return &TransportBoundaryError{
                VarName:      varEntry.Name,
                CurrentScope: currentScope.sessionID,
                OriginScope:  originScope.sessionID,
            }
        }
    }
    
    return nil
}
```

**Remote boundary enforcement:**

Capabilities and leases NEVER cross transport boundaries. Remote executors must re-resolve secrets:

```go
// Local executor has vault + capability
localExecutor := NewExecutor(plan, localVault, localCapability)

// Remote executor (SSH/Docker) CANNOT use local vault
// Must re-resolve via provider or planner blocks it
remoteExecutor := NewExecutor(plan, remoteVault, remoteCapability)

// If secret is LocalOnly and not explicitly imported → plan-time error
// If secret is imported → remote executor fetches from its own provider
```

**Why this matters:**
- Prevents secrets from leaking across trust boundaries
- Remote systems cannot access local secrets without explicit import
- Each transport has its own vault + capability (isolated)
- Plan-time validation prevents accidental cross-boundary use

#### Meta-Programming (Plan-Time Sites)

Plan-time control flow is just another site with planner authority:

```opal
var ENVIRONMENT = "production"

if @var.ENVIRONMENT == "production" {  # Site: "planner/if[0]/condition"
    deploy_to_prod()
}
```

**Same model applies:**
- Planner unwraps at site `"planner/if[0]/condition"`
- Recorded in `SecretUses[]` with planner site
- Planner has authority for plan-time sites
- Runtime has authority for execution-time sites

#### Security Hardening

**1. Executor Controls Site (Prevent Forgery)**

Decorators cannot determine their own site:

```go
// Decorator receives frame, cannot modify frame.Site
func (d *RetryDecorator) Execute(frame *ExecutionFrame) error {
    // frame.Site is immutable, set by executor
}
```

**2. No Handle Exposure (Prevent Reflection)**

Decorators never receive `*secret.Handle` objects:

```go
// ❌ BAD: Decorator can use reflection
func (d *Decorator) Execute(params map[string]any) {
    handle := params["apiKey"].(*secret.Handle)  // Reflection attack possible
}

// ✅ GOOD: Decorator gets DisplayIDs only
func (d *Decorator) Execute(params map[string]string) {
    displayID := params["apiKey"]  // Just a string "opal:v:3J98t56A"
}
```

**3. Capability Protection (Prevent Bypass)**

Capability is unforgeable:

```go
type Capability struct {
    token  uint64    // Random token
    issuer string    // "planner" or "executor"
    nonce  [16]byte  // Unforgeable nonce
}

// Package-private (not exported)
func newCapability(issuer string) *Capability {
    var nonce [16]byte
    rand.Read(nonce[:])
    return &Capability{
        token:  rand.Uint64(),
        issuer: issuer,
        nonce:  nonce,
    }
}
```

**4. Immutable Plan (Prevent TOCTOU)**

Plan is frozen after creation:

```go
type Plan struct {
    Secrets     []Secret
    SecretUses  []SecretUse
    hash        string  // Includes SecretUses
    frozen      bool
}

func (p *Plan) Freeze() {
    p.hash = p.computeHash()
    p.frozen = true
}

// Executor verifies hash
func (e *Executor) Execute(plan *Plan) error {
    if plan.hash != plan.computeHash() {
        return fmt.Errorf("plan tampered - hash mismatch")
    }
    // ...
}
```

**5. Planner Vault Separation**

Planner has its own vault + capability:

```go
type Planner struct {
    secretVault  map[string]*secret.Handle
    capability   *secret.Capability
    plannerSites map[string]bool  // Authorized plan-time sites
}
```

#### Secret Handle API

**Safe operations (always available):**
- `handle.ID()` - Opaque display ID: `opal:s:3J98t56A`
- `handle.Mask(3)` - Masked display: `abc***xyz`
- `handle.Len()` - Length without exposing value
- `handle.IsEmpty()` - Check if empty

**Unsafe operations (capability-gated):**
- `handle.UnsafeUnwrap()` - Raw value (requires capability)
- `handle.ForEnv("KEY")` - Environment variable (requires capability)
- `handle.Bytes()` - Raw bytes (requires capability)

**Capability gating:** Only executor/planner can issue capabilities. Decorators cannot forge or access raw values.

### Command Execution (`core/sdk/executor`)

Use `executor.Command()` instead of `os/exec` for automatic scrubbing:

```go
import "github.com/aledsdavies/opal/core/sdk/executor"

cmd := executor.Command("kubectl", "get", "pods")
cmd.AppendEnv(map[string]string{
    "KUBECONFIG": kubeconfigPath,
})
exitCode, err := cmd.Run()
```

**Why this is safe:**
- Output automatically routes through scrubber
- Secrets are redacted before display
- No way to bypass security

**API:**
- `Command(name, args...)` - Create command
- `Bash(script)` - Execute bash script
- `AppendEnv(map)` - Add environment (preserves PATH) - **⚠️ NEVER use for secrets**
- `SetStdin(reader)` - Feed input
- `Run()` - Execute and wait
- `Start()` / `Wait()` - Async execution
- `Output()` / `CombinedOutput()` - **⚠️ DANGEROUS: Capture output (NOT scrubbed)**

**⚠️ CRITICAL: Output() / CombinedOutput() Foot-Gun**

These methods capture output **without scrubbing**. Use only when:
1. You are certain no secrets are in the output
2. Output is immediately scrubbed before logging/display
3. You have a lint rule enforcing scrubbing

**Safer alternatives:**
```go
// ✅ GOOD: Use Run() with scrubbed stdout/stderr
cmd := executor.Command("kubectl", "get", "pods")
err := cmd.Run()  // Output automatically scrubbed

// ❌ DANGEROUS: Unscrubbed capture
output, _ := cmd.CombinedOutput()  // Secrets may leak!

// ✅ GOOD: Manual scrubbing if capture needed
output, _ := cmd.CombinedOutput()
scrubbed := scrubber.Scrub(output)  // Explicit scrubbing
```

### Security Model

- **Taint tracking**: Secrets panic on `String()` to catch accidental leaks
- **Per-run keyed fingerprints**: Prevent cross-run correlation
- **Locked-down I/O**: All subprocess output goes through scrubber
- **Capability gating**: Raw access requires executor-issued token

See `docs/SDK_GUIDE.md` for complete API reference and examples.

## Plugin System

Decorators work through a dual-path plugin system that balances safety with flexibility:

### Plugin Distribution Model

**Two distribution paths following Go modules and Nix flakes pattern:**

* **Registry path (curated, verified)** → strict conformance guarantees
* **Direct Git path (user-supplied)** → bypasses registry, user owns risk

```bash
# From registry (verified)
accord get accord.dev/aws.ec2@v1.4.2

# Direct Git (team-owned, unverified)  
accord get github.com/acme/accord-plugins/k8s@v0.1.0
```

### Registry vs Git-Sourced Plugins

**Registry plugins (accord.dev/...):**
- Come with signed manifests + verification reports
- Passed full conformance suite and security audits
- Deterministic, idempotent, secrets-safety verified
- SLSA Level 3 provenance + reproducible builds
- Automatic updates within semver constraints

**Git-sourced plugins (github.com/...):**
- Can pin by commit hash for reproducibility
- `accord verify-plugin ./...` runs locally but not centrally verified
- Warning displayed but not blocked
- Useful for private/experimental/internal plugins
- Enterprise can host private verified registries

### Plugin Verification

**Registry admission pipeline**: External value decorators and execution decorators must pass comprehensive verification before registry inclusion. No arbitrary code execution - plugins pass a compliance test suite that verifies they implement required interfaces correctly and respect security requirements.

**Local verification**: Git-sourced plugins run the same conformance suite locally, providing the same crash isolation and security sandboxing but without central verification guarantees.

**Plugin isolation**: All plugins (registry or Git) run in limited contexts and can't crash the main execution engine. Resource usage gets monitored and timeouts are enforced via cgroups/bwrap.

### Registry Pattern Implementation

**Startup registration**: Both built-in and plugin value decorators and execution decorators register themselves at startup. The runtime looks up decorators by name without hardcoded lists, making the system extensible.

**Capability verification**: Engine checks on load that manifest signature matches, spec_version overlaps with runtime, and capabilities match requested decorators (no "hidden" entrypoints).

This means organizations can build custom infrastructure value decorators and execution decorators (like `@company.k8s.deploy`) while maintaining the same security and verification guarantees as built-in decorators. Small teams can ship plugins immediately via Git without waiting on central registry approval, but audit trails clearly show verification status.

## Resolution Strategy

Two-phase resolution optimizes for both speed and determinism:

**Quick plans** defer expensive operations and show placeholders:
```
kubectl create secret --token=¹@aws.secret.api_token
Deferred: 1. @aws.secret.api_token → <expensive: AWS lookup>
```

**Resolved plans** materialize all values for deterministic execution:
```  
kubectl create secret --token=¹opal:s:qN7yOr4A
Resolved: 1. @aws.secret.api_token → opal:s:qN7yOr4A
```

Smart optimizations happen automatically:
- Expensive value decorators in unused conditional branches never execute
- Independent expensive operations resolve in parallel  
- Dead code elimination prevents unnecessary side effects

## Security Model

The placeholder system protects sensitive values while enabling change detection using a **two-track identity model**:

**Placeholder format (user-visible)**: `🔒 opal:s:3J98t56A` - opaque random ID, no length leak, no correlation across runs.

**Security invariant**: Raw secrets never appear in plans, logs, or error messages. This applies to all value decorators - `@env.NAME`, `@aws.secret.NAME`, whatever. Compliance teams can review plans confidently.

**Hash scope**: Plan hashes cover ordered steps, arguments, operator graphs, and timing flags. They exclude ephemeral data like run IDs or timestamps that shouldn't invalidate a plan.

### Two-Track Identity Model

Secrets need two representations for different purposes:

**Track 1: Display (User-Visible)**
- Format: `🔒 opal:s:3J98t56A` (Base58 encoded, context-aware ID)
- Used in: Terminal output, logs, CLI display, plan files
- Properties: No length leak, context-sensitive, deterministic in resolved plans
- Example: `API_KEY: 🔒 opal:s:3J98t56A`

**Track 2: Internal (Machine-Readable)**
- Format: BLAKE2b-256 keyed hash with per-run key
- Used in: Scrubber matching, secret detection, internal verification
- Properties: Keyed (per-run), deterministic within run, prevents oracle attacks
- Never displayed to users

**DisplayID Generation (Keyed PRF):**

DisplayIDs use a keyed BLAKE2s-128 PRF over `(plan_salt, step_path, decorator, key_name, hash(value))`:

- **Resolved plans** (`ModePlan`): Deterministic IDs with per-plan salt
  - Key: `plan_key = HKDF(plan_digest, "opal/displayid/plan/v1")`
  - Salt: `plan_salt = CSPRNG(32 bytes)` (generated once per plan, stored in plan header)
  - Same plan + context + value → same DisplayID (within that plan)
  - Different plans → different DisplayIDs (prevents cross-plan correlation)
  - Enables contract verification (plan hash includes DisplayIDs + salt)

- **Direct execution** (`ModeRun`): Random-looking IDs with fresh per-run key
  - Key: `run_key = CSPRNG(32 bytes)`
  - Different runs → different DisplayIDs
  - Prevents correlation and tracking

**DisplayID policy (structure-only, not value-linked):**
- DisplayIDs are derived from **structure** (step path, decorator, param name) + **per-plan salt**
- Value hash included in PRF input to prevent oracle attacks
- Same secret value in different plans → different DisplayIDs (unlinkability)
- Secret rotation does NOT change DisplayID (structure unchanged)
- Plan hash changes on rotation (new value → new plan)

**Why this works:**
- **Contract verification**: Deterministic DisplayIDs in resolved plans ensure same plan → same hash
- **Security**: Context-aware PRF prevents oracle attacks; per-plan salt prevents cross-plan correlation
- **Unlinkability**: Different plans produce different DisplayIDs even for same value
- **No length leak**: `hash(value)` used in PRF input, not raw value
- **Rotation-safe**: DisplayID stable across rotations, plan hash changes
- **UX**: Users see short, readable identifiers (11 chars typical)

**Secret rotation semantics:**
- DisplayID remains stable (structure unchanged)
- Plan hash changes (new value)
- Scrubber seeds updated (new value → new fingerprints)
- Audit trail shows rotation via plan hash change

**Implementation:**
- `secret.IDFactory` interface with `ModePlan` and `ModeRun` modes
- `planfmt.NewPlanIDFactory(plan)` creates deterministic factory for resolved plans
- `planfmt.NewRunIDFactory()` creates random factory for direct execution
- `secret.Handle.ID()` returns DisplayID from factory
- `secret.Handle.Fingerprint(key)` returns keyed hash for scrubber (separate from DisplayID)
- Scrubber uses fingerprints for matching, displays DisplayIDs in output

### Plan Provenance Headers

All resolved plans include provenance metadata for audit trails:

```json
{
  "header": {
    "spec_version": "1.1",
    "plan_version": "2024.1",
    "generated_at": "2024-09-20T10:22:30Z",
    "source_commit": "abc123def456",
    "compiler_version": "opal-1.4.2",
    "plugins": {
      "aws.ec2": {
        "version": "1.4.2",
        "source": "registry:accord.dev",
        "verification": "passed",
        "signed_by": "sigstore:accord.dev/publishers/aws-team"
      },
      "company.k8s": {
        "version": "0.2.1", 
        "source": "git:github.com/acme/accord-plugins@sha256:def789",
        "verification": "local-only",
        "signed_by": null
      }
    }
  },
  "plan_hash": "sha256:5f6c...",
  "plan_salt": "base64:Xj9K...",
  "steps": [...]
}
```

**Plan hash inputs (contract verification):**

The plan hash covers ALL security-relevant data to prevent tampering:

```go
func (p *Plan) computeHash() string {
    h := sha256.New()
    
    // Structure
    h.Write([]byte(p.Steps))           // Execution steps
    h.Write([]byte(p.Secrets))         // Secret inventory
    h.Write([]byte(p.SecretUses))      // Authorization list (critical!)
    
    // Provenance
    h.Write([]byte(p.Header.Plugins))  // Plugin versions + verification
    h.Write([]byte(p.PlanSalt))        // Per-plan salt for DisplayIDs
    
    // Metadata (optional, for drift detection)
    h.Write([]byte(p.Header.SourceCommit))
    
    return hex.EncodeToString(h.Sum(nil))
}
```

**Why SecretUses in hash is critical:**
- Prevents adding unauthorized use-sites after plan approval
- Tampering with SecretUses → hash mismatch → execution fails
- Ensures reviewed authorization list matches execution

**Provenance benefits:**
- **Audit compliance**: See exactly which plugins were used and their verification status
- **Risk assessment**: Distinguish registry-verified vs Git-sourced plugins
- **Reproducibility**: Pin exact plugin versions and sources
- **Security**: Track signing and verification chain

**Source classification:**
- `registry:accord.dev` - Centrally verified via registry admission pipeline  
- `registry:company.internal` - Private enterprise registry with internal verification
- `git:github.com/org/repo@sha` - Direct Git import with commit pinning
- `local:./plugins/custom` - Local development plugin

This ensures compliance teams can review plans knowing the verification status of every component, while developers retain flexibility to use unverified plugins when needed.

### Enterprise Plugin Strategies

**Private registry pattern:**
```bash
# Enterprise hosts internal registry with company plugins
accord config set registry https://plugins.company.internal

# Mix verified public and private plugins
accord get accord.dev/aws.ec2@v1.4.2        # Public verified
accord get company.internal/vault@v2.1.0     # Private verified  
accord get github.com/team/custom@v0.1.0     # Direct Git (unverified)
```

**Policy enforcement:**
- Production environments can require `verification: passed` in all plan headers
- Development environments allow unverified plugins with warnings
- CI/CD pipelines can gate on plugin verification status

**Air-gapped deployments:**
- Registry mirrors for offline environments
- Pre-verified plugin bundles with signatures
- Local verification without external registry access

This dual-path approach avoids "walled garden" criticism while maintaining security - developers can always opt out but know they're assuming risk, and audit trails preserve full accountability.

## Secret Scrubbing Architecture

Opal prevents secrets from leaking into **plans and terminal output** through automatic scrubbing. All value decorator results are treated as secrets - no exceptions.

### Design Philosophy

**Scrubbing scope (by design):**
- ✅ **Plans**: All value decorators replaced with DisplayIDs (`opal:s:3J98t56A`)
- ✅ **Terminal output**: Stdout/stderr scrubbed before display
- ✅ **Logs**: All logging output scrubbed
- ❌ **Pipes**: Raw values flow between operators (needed for work)
- ❌ **Redirects**: Raw values written to files (user controls destination)

**Why raw values in pipes/redirects:**
- Operators need actual values to function (grep, awk, jq, etc.)
- User explicitly controls where output goes
- Scrubbing would break legitimate workflows
- User can use `scrub()` PipeOp if needed (see OEP-002)

### Core Principle

**ALL value decorators are secrets** - even seemingly innocuous ones:
- `@env.HOME` - Could leak system paths
- `@git.commit_hash` - Could leak repository state  
- `@var.username` - Could leak user information
- `@aws.secret.key` - Obviously sensitive

**No exceptions.** Scrub everything by default, let user opt-in to exposure via pipes/redirects.

### How It Works

**Value decorator resolution:**
```go
// Value decorators return plain values (string, int, etc.)
func (d *EnvDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
    value := ctx.Session.Env()[*call.Primary]  // Raw value: "sk-abc123xyz"
    return value, nil  // Returns plain string, not secret.Handle
}
```

**Secret wrapping (planner/executor):**
```go
// Planner wraps ALL value decorator results in secret.Handle
resolvedValue, err := decorator.Resolve(ctx, call)
if err != nil {
    return err
}

// Wrap in secret.Handle with deterministic DisplayID
handle := secret.NewHandleWithFactory(
    fmt.Sprint(resolvedValue),  // Convert any to string
    ctx.IDFactory,
    secret.IDContext{
        PlanHash:  ctx.PlanHash,
        StepPath:  ctx.StepPath,
        Decorator: call.Path,
        KeyName:   *call.Primary,
        Kind:      "s",
    },
)
```

**Scrubber registration (executor):**
```go
// Executor maintains scrubber with all resolved secrets
scrubber := streamscrub.NewScrubber()
for _, handle := range plan.Secrets {
    // Register both raw value and common encodings
    scrubber.AddSecret(handle.Bytes())
    scrubber.AddSecret([]byte(url.QueryEscape(string(handle.Bytes()))))
    scrubber.AddSecret([]byte(base64.StdEncoding.EncodeToString(handle.Bytes())))
}
```

**I/O wrapping (executor):**
```go
// All subprocess I/O flows through scrubber
scrubbedStdout := scrubber.WrapWriter(os.Stdout, handle.ID())
scrubbedStderr := scrubber.WrapWriter(os.Stderr, handle.ID())

cmd.Stdout = scrubbedStdout
cmd.Stderr = scrubbedStderr
```

**Runtime replacement:**
```
Input:  "Connecting to API with key: sk-abc123xyz"
Output: "Connecting to API with key: 🔒 opal:s:3J98t56A"
```

### Scrubbing Layers

**Layer 1: Subprocess Output (Streaming)**
- Wraps `os.Stdout` and `os.Stderr` with scrubbing writers
- Streaming replacement using Aho-Corasick automaton
- Replaces secrets with DisplayIDs in real-time
- Zero buffering - output appears immediately

**Layer 2: Error Messages**
- All errors flow through scrubber before display
- Catches secrets in exception messages, stack traces
- Example: `panic("Failed to connect: sk-abc123xyz")` → `panic("Failed to connect: 🔒 opal:s:3J98t56A")`

**Layer 3: Plan Files**
- Resolved plans never contain raw secrets
- All values represented as DisplayIDs: `🔒 opal:s:3J98t56A`
- Plan files are safe to commit, share, review

**Layer 4: Logs and Telemetry**
- All logging output flows through scrubber
- Telemetry spans scrubbed before export
- Safe to send to external observability systems

### Encoding Detection

Secrets can leak in encoded forms. Opal scrubs common encodings:

**URL encoding:**
```
Raw:     "sk-abc123xyz"
Encoded: "sk-abc123xyz" (no change) or "sk%2Dabc123xyz" (percent-encoded)
Scrubbed: "🔒 opal:s:3J98t56A"
```

**Base64 encoding:**
```
Raw:     "sk-abc123xyz"
Encoded: "c2stYWJjMTIzeHl6"
Scrubbed: "🔒 opal:s:3J98t56A"
```

**JSON encoding:**
```
Raw:     {"key": "sk-abc123xyz"}
Scrubbed: {"key": "🔒 opal:s:3J98t56A"}
```

### Performance

**Aho-Corasick automaton:**
- Builds finite state machine from all secret patterns
- Single pass through output stream
- O(n) complexity where n = output length
- Constant overhead regardless of secret count

**Benchmarks:**
- 1 secret: ~5% overhead
- 10 secrets: ~8% overhead
- 100 secrets: ~12% overhead
- Streaming: No memory buffering

**Why fast:**
- Compiled automaton (not regex)
- Single pass (not multiple scans)
- Streaming (no buffering)
- Lazy initialization (only when secrets present)

### Security Properties

**No bypass paths:**
- All I/O routed through scrubber
- Decorators can't access raw stdout/stderr
- Transport implementations must use scrubbed writers
- No way to opt out

**Defense in depth:**
- Multiple encoding detection (raw, URL, base64)
- Error message scrubbing (catches panics)
- Plan file scrubbing (safe to share)
- Telemetry scrubbing (safe to export)

**Audit trail:**
- DisplayIDs show which secrets were used
- No correlation across runs (per-run keys)
- Compliance teams can review plans
- No raw values in any output

### Implementation Details

**Scrubber interface:**
```go
type Scrubber interface {
    // AddSecret registers a secret for scrubbing
    AddSecret(secret []byte)
    
    // WrapWriter wraps an io.Writer with scrubbing
    WrapWriter(w io.Writer, displayID string) io.Writer
    
    // ScrubString scrubs a string (for error messages)
    ScrubString(s string) string
}
```

**Usage in executor:**
```go
// Create scrubber with all resolved secrets
scrubber := streamscrub.NewScrubber()
for _, handle := range resolvedSecrets {
    scrubber.AddSecret(handle.Bytes())
}

// Wrap all I/O
cmd.Stdout = scrubber.WrapWriter(os.Stdout, handle.ID())
cmd.Stderr = scrubber.WrapWriter(os.Stderr, handle.ID())

// Scrub error messages
if err != nil {
    return fmt.Errorf("command failed: %w", scrubber.ScrubError(err))
}
```

**Key files:**
- `runtime/streamscrub/scrubber.go` - Main scrubber implementation
- `runtime/streamscrub/placeholder.go` - DisplayID generation
- `core/sdk/secret/handle.go` - Secret handle abstraction
- `runtime/executor/executor.go` - Scrubber integration

### Why This Design

**Automatic and transparent:**
- Decorators don't need to think about scrubbing
- Works for all transports (local, SSH, Docker)
- No way to accidentally leak secrets

**Performance:**
- Streaming (no buffering)
- Efficient automaton (not regex)
- Lazy (only when secrets present)

**Complete:**
- Covers all output paths
- Multiple encoding detection
- Error messages included
- Plan files safe

**Auditable:**
- DisplayIDs show secret usage
- No correlation across runs
- Compliance-friendly

This architecture ensures secrets never leak while maintaining performance and usability.

## Seeded Determinism

For operations requiring randomness or cryptography, opal will use seeded determinism to maintain contract verification while enabling secure random generation.

### Plan Seed Envelope (PSE)

**Purpose**: PSE provides deterministic randomness for value decorators that need random generation (e.g., `@random.password()`, `@crypto.generate_key()`). It is NOT used for `@var` or `@env` which resolve to actual user-provided or environment values.

**Seed generation**: High-entropy seed generated at `--resolve` time, never stored raw in plans.

**Sealed envelope**: Plans contain only encrypted seed envelopes with fields:
- `alg`: DRBG algorithm (e.g., "chacha20-drbg")  
- `kdf`: Key derivation function (e.g., "hkdf-sha256")
- `scope`: Derivation scope ("plan")
- `seed_hash`: Hash for tamper detection
- `enc_seed`: Seed sealed to runner key/KMS

**Security model**: Raw seeds never appear in plans, only sealed envelopes. Decryption requires proper runner authorization.

### Deterministic Derivation

**Scoped sub-seeds**: Each decorator gets unique deterministic sub-seed using:
```
HKDF(seed, info=plan_hash || step_path || decorator_name || counter)
```

**Stable generation**: Same plan produces same random values every time. Different plans (even with same source) produce different values due to new seed.

**Parallel safety**: Each step has unique `step_path`, ensuring no collisions in concurrent execution.

### Implementation Requirements

**API surface**:
```opal
var DB_PASS = @random.password(length=24, alphabet="A-Za-z0-9!@#")
var API_KEY = @crypto.generate_key(type="ed25519")

deploy: {
    kubectl create secret generic db --from-literal=password=@var.DB_PASS
}
```

**Plan display**: Shows placeholders maintaining security invariant:
```
kubectl create secret generic db --from-literal=password=¹opal:s:uS2cVw8E
```

**Execution flow**:
1. `--resolve`: Generate PSE, derive preview hashes, seal envelope
2. `run --plan`: Decrypt PSE, derive values on-demand during execution
3. Material values injected via secure channels, never stdout/logs

**Failure modes**:
- Missing decryption capability → `infra_missing:seed_keystore`
- Tampered envelope → verification failure  
- Structure changes → normal contract verification failure

### Security Guarantees

**No value exposure**: Generated secrets follow same placeholder rules as all other sensitive values.

**Audit trail**: Plan headers include seed algorithm metadata without exposing entropy.

**Deterministic contracts**: Same resolved plan produces identical random values across executions.

**Authorization boundaries**: PSE sealed to specific runner contexts, preventing unauthorized plan execution.

This enables secure, auditable randomness within the contract verification model while maintaining all existing security invariants.

### Seed Security and Scoping

**Cryptographic independence**: Seeds are generated using 256-bit CSPRNG entropy, never derived from plan content, hashes, or names. The plan provides scoping context via HKDF info parameter, not entropy.

**Safe derivation pattern**:
```
seed = CSPRNG(256_bits)  // Independent entropy 
subkey = HKDF(seed, info=plan_hash || step_path || decorator || counter)
output = DRBG(subkey, requested_length)
```

**Regeneration keys**: Decorators use explicit regeneration keys to control when values change:

```opal
// Default: regenerates on every plan (plan hash as key)
var TEMP_TOKEN = @random.password(length=16)

// Stable: same key = same password across plan changes  
var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v1")

// Rotate by changing the key
var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v2")
```

**Derivation with regeneration keys**:
```
effective_key = regen_key || decorator_name || step_path
subkey = HKDF(seed, info=effective_key)
output = DRBG(subkey, requested_length)
```

**Value stability rules**:
- Same `regen_key` = same values (regardless of plan changes)
- Change `regen_key` = new values  
- No `regen_key` = plan hash used as key (values change on plan regeneration)

**Security hardening options**:
- Keystore references instead of embedded encrypted seeds
- Require `--resolve` for any randomness operations  
- AEAD encryption with runner-specific keys or KMS
- Seed hash for tamper detection

**Threat model**:
- Plan-only attacker: Cannot decrypt seed, sees only length/hash placeholders
- Known outputs: Cannot recover seed due to HKDF+DRBG one-way properties  
- Stolen plans: Useless without runner authorization keys

This approach provides cryptographically sound randomness while maintaining deterministic contract execution.

## Plan-Time Determinism  

Control flow expands during plan generation, not execution:

```opal
// Source code
for service in ["api", "worker"] {
    kubectl apply -f k8s/@var.service/
}

// Plan shows expanded steps
kubectl apply -f k8s/api/      # Step: deploy.service[0]  
kubectl apply -f k8s/worker/   # Step: deploy.service[1]
```

This means execution decorators like `@parallel` receive predictable, static command lists rather than dynamic loops. Much easier to reason about.

**No chaining for control flow**: Constructs like `when`, `for`, `try/catch` are complete statements, not expressions. You can't write `when ENV { ... } && echo "done"` because it creates precedence confusion. Keep control flow self-contained.

## Contract Verification

The heart of the architecture: resolved plans become execution contracts.

**Verification process**: When executing a resolved plan, we replan from current source and infrastructure, then compare structures. If anything changed, we fail with a clear diff showing what's different.

**Drift classification**: We categorize verification failures to suggest appropriate actions:
- `source_changed`: Source files modified → regenerate plan
- `infra_missing`: Expected infrastructure not found → use `--force` or fix infrastructure  
- `infra_mutated`: Infrastructure present but different → use `--force` or regenerate plan

**Execution modes**: 
- Default: strict verification, fail on any changes
- `--force`: use plan values as targets, adapt to current infrastructure

This gives teams deployment confidence: the plan they reviewed is exactly what executes, with clear options when reality changes.

## Module Organization

Clean separation keeps the system maintainable:

**Core module**: Types, interfaces, and data structures only. No execution logic, no external dependencies. Defines the contracts that decorators must implement.

**Runtime module**: Lexer, parser, execution engine, and built-in decorators. Handles plugin loading and verification. Contains all the business logic.

**CLI module**: Thin wrapper around runtime. Handles command-line parsing and file I/O. No business logic.

Dependencies flow one direction: `cli/` → `runtime/` → `core/`. This prevents circular dependencies and keeps concerns separated.

## Module Structure

**Three clean modules:**

- **core/**: Types, interfaces, and plan structures
- **runtime/**: Lexer, parser, execution engine
- **cli/**: Command-line interface

Dependencies flow one way: `cli/` → `runtime/` → `core/`

## Error Handling

Try/catch is special - it's the only construct that creates non-deterministic execution paths:

```opal
deploy: {
    try {
        kubectl apply -f k8s/
        kubectl rollout status deployment/app  
    } catch {
        kubectl rollout undo deployment/app
    } finally {
        kubectl get pods
    }
}
```

Plans show all possible paths (try, catch, finally). Execution logs show which path was actually taken. This gives you predictable error handling without making plans completely deterministic.

Like other control flow, try/catch can't be chained with operators. Keep error handling self-contained to avoid precedence confusion.

## Implementation Pipeline

The compilation flow ensures contract verification works reliably:

1. **Lexer**: Fast tokenization with mode detection (command vs script mode)
2. **Parser**: Decorator AST generation  
3. **Transform**: Meta-programming expansion (loops, conditionals)
4. **Plan**: Deterministic execution sequence with stable step IDs
5. **Resolve**: Value materialization with security placeholders
6. **Verify**: Contract comparison and drift detection  
7. **Execute**: Actual command execution with idempotency

The key insight: meta-programming happens during transform, so all downstream stages work with predictable, static command sequences.

## Performance Design

**Lexer**: Zero allocations for hot paths. Use pre-compiled patterns and avoid regex where possible.

**Resolution optimization**: Expensive value decorators resolve in parallel using DAG analysis. Unused branches never execute, preventing unnecessary side effects.

**Plan caching**: Plans are cacheable and reusable between runs. Plan hashes enable this optimization.

**Partial execution**: Support resuming from specific steps with `--from step:path` for long pipelines.

## Testing Requirements

**Decorator compliance**: Every value decorator and execution decorator must pass a standard compliance test suite that verifies interface implementation, security placeholder handling, and contract verification behavior.

**Plugin verification**: External value decorators and execution decorators get the same compliance testing plus binary integrity verification through source hashing.

**Contract testing**: Comprehensive scenarios covering source changes, infrastructure drift, and all verification error types.

## IaC + Operations Together

A novel capability emerges from the decorator architecture: seamless mixing of infrastructure-as-code with operations scripts in a single language.

```opal
deploy: {
    // Infrastructure deployment
    @aws.ec2.deploy(name="web-prod", count=3)
    @aws.rds.deploy(name="db-prod", size="db.r5.large")
    
    // Operations on the deployed infrastructure  
    @aws.ec2.instances(tags={name:"web-prod"}, transport="ssm") {
        sudo systemctl start myapp
        @retry(attempts=3) { curl -f http://localhost:8080/health }
    }
    
    // Traditional ops commands
    kubectl apply -f k8s/monitoring/
    helm upgrade prometheus charts/prometheus
}
```

**The key insight**: Both infrastructure value decorators and execution decorators follow the same contract model - plan, verify, execute. This means you can mix provisioning with configuration management cleanly.

**Infrastructure value decorators** handle provisioning:
- Plan: Show what infrastructure will be created/modified
- Verify: Check current infrastructure state vs plan
- Execute: Create/modify infrastructure to match plan

**Execution decorators** handle operations:
- Plan: Show what commands will run where
- Verify: Check target systems are available and reachable
- Execute: Run commands with proper error handling and aggregation

Both types support the same features: contract verification, partial execution, idempotency, security placeholders, and plugin extensibility.

This eliminates the traditional boundary between "infrastructure tools" and "configuration management tools" - it's all just decorators with different responsibilities.

## Example: Advanced Infrastructure Execution

Here's how complex scenarios work within the decorator model:

```opal
maintenance: {
    // Select running instances
    @aws.ec2.instances(
        region="us-west-2",
        tags={env:"prod", role:"web"},
        transport="ssm",
        max_concurrency=3,
        tolerate=0
    ) {
        // Drain traffic
        sudo systemctl stop nginx
        
        // Update application  
        @retry(attempts=3, delay=10s) {
            sudo yum update -y myapp
            sudo systemctl start myapp
        }
        
        // Health check
        @timeout(30s) {
            curl -fsS http://127.0.0.1:8080/healthz
        }
        
        // Restore traffic
        sudo systemctl start nginx
    }
}
```

**Plan shows**:
- 5 instances selected by tags
- Commands that will run on each
- Concurrency and error tolerance policy
- Transport method (SSM vs SSH)

**Verification checks**:
- Selected instances still exist and match tags
- SSM transport is available on all instances  
- Classifies drift: `ok | infra_missing | infra_mutated`

**Execution provides**:
- Bounded concurrency across instances
- Per-instance stdout/stderr streaming
- Retry/timeout on individual commands
- Aggregated results with failure policy

This level of infrastructure operations was traditionally split across multiple tools. The decorator model handles it seamlessly.

## Why This Architecture Works

**Contract-first development**: Resolved plans are immutable execution contracts with verification, giving teams deployment confidence.

**IaC + ops together**: Mix infrastructure provisioning with operations scripts in one language, eliminating tool boundaries.

**Plugin extensibility**: Organizations can build custom decorators through verified, source-hashed plugins while maintaining security guarantees.

**Stateless simplicity**: No state files to corrupt or manage - decorators query reality fresh each time and use contract verification for consistency.

**Consistent execution model**: Everything becomes a decorator internally, making the system predictable and extensible without special cases.

**Performance optimization**: Plan-time expansion, parallel resolution, and dead code elimination ensure efficient execution at scale.

This delivers "Terraform for operations, but without state file complexity" through contract verification rather than state management.