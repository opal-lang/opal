# Decorator Architecture Documentation

This document defines the complete decorator architecture for **devcmd**, showing the intended design after all surgical fixes are applied.

## Design Principles

### 1. Single Source of Truth for APIs
- **One decorator interface per behavior type** - clear, focused contracts
- **Command-based execution** - decorators receive raw commands for maximum control
- **Parser-agnostic parameters** - runtime types independent of syntax parsing

#### Focused Decorator Interfaces

Each decorator behavior has a single, purpose-built interface:

- **ValueDecorator** expands inline values via `Render()` 
- **ActionDecorator** executes commands via `Run()`
- **BlockDecorator** wraps command sequences via `WrapCommands()`
- **PatternDecorator** selects execution branches via `SelectBranch()`

#### Command-Based Execution Design

Decorators receive raw command structures to enable maximum execution flexibility:

```go
// @parallel receives the actual command sequence
func (d *ParallelDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, inner CommandSeq) CommandResult {
    // Decorator decides execution strategy:
    return ctx.ExecParallel(inner.Steps, d.getMode(args))
}

// @retry receives the same command sequence for each attempt
func (r *RetryDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, inner CommandSeq) CommandResult {
    for attempt := 1; attempt <= r.getAttempts(args); attempt++ {
        result := ctx.ExecSequential(inner)
        if result.Success() { return result }
        time.Sleep(r.getDelay(args))
    }
    return lastResult
}

// @when selects which branch commands to execute
func (w *WhenDecorator) SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult {
    condition := w.evaluateCondition(ctx, args)
    if commands, exists := branches[condition]; exists {
        return ctx.ExecSequential(commands)
    }
    return CommandResult{ExitCode: 0}
}
```

This design enables decorators to implement sophisticated execution strategies: parallel execution, retries, conditional branching, and environment modifications.

## Shell Operator Semantics

### Engine-Level vs Shell-Level Operations

**devcmd** distinguishes between two levels of command processing for cross-platform consistency:

#### Engine-Level Operators (Cross-Platform)
The devcmd engine handles these operators consistently across all platforms:

- `&&` - Execute next command only if previous succeeded
- `||` - Execute next command only if previous failed  
- `|` - Pipe stdout from previous command to stdin of next command
- `>>` - Append stdout from previous command to specified file

```devcmd
# Engine guarantees consistent behavior across Windows/Unix
deploy: docker build . && kubectl apply -f k8s.yaml
test: npm test || echo "Tests failed but continuing"
logs: kubectl logs pod >> /tmp/deployment.log
```

**Cross-platform guarantees**:
- `&&` and `||` work identically on Windows and Unix systems
- `>>` appends stdout to file with consistent encoding
- `|` streams data directly between processes without platform-specific shell syntax

#### Shell-Level Commands (Platform-Specific)
Complex redirection syntax is passed directly to the platform shell:

```devcmd
# Shell-specific syntax (platform-dependent)
build: make build 2>&1 >> build.log        # Unix shell handles 2>&1
backup: tar czf backup.tar.gz . 2>/dev/null # Unix stderr redirect
windows: echo "Hello" > NUL 2>&1            # Windows cmd syntax
```

**Shell responsibility**:
- `2>&1` syntax is handled by the underlying shell
- Platform-specific redirections remain platform-specific
- No cross-platform behavior guarantees for shell-level syntax

### Redirection Best Practices

**For cross-platform compatibility**, use engine-level operators:
```devcmd
# ✅ Cross-platform - engine handles >>
deploy: kubectl apply -f k8s/ >> deploy.log

# ✅ Cross-platform - engine handles && and ||
test: npm test && echo "Success" || echo "Failed"
```

**For platform-specific needs**, use shell-level syntax:
```devcmd
# Platform-specific when shell features are required
unix-build: make 2>&1 | tee build.log     # Unix-specific tee command
win-build: msbuild /v:q > build.log 2>&1   # Windows-specific MSBuild
```

### Engine `>>` Operator Details

The engine-level `>>` operator has specific semantics:

```devcmd
# Engine appends stdout only (stderr separate)
logs: kubectl logs deployment >> app.log

# Chain continues after successful append
deploy: {
    kubectl apply -f k8s/ >> deploy.log
    echo "Deployment complete" >> deploy.log
}
```

**Behavior**:
- Appends stdout from the previous command element to the specified file
- Stderr is not captured unless explicitly redirected by shell syntax
- Returns success (exit code 0) unless file operations fail
- File is created if it doesn't exist, appended to if it does

### 2. Cross-Mode Consistency
- **Semantic equivalence**: `devcmd run build` and `./mycli build` produce identical results
- **Same UI behavior**: standardized flags work identically in interpreter and generated modes
- **Unified plan generation**: `devcmd run build --dry-run` and `./mycli build --dry-run` show identical plans

### 3. Clean Separation of Concerns
- **Core interfaces** define contracts (no implementation)
- **Runtime execution** implements the IR evaluator
- **Codegen hints** optional for AOT optimization (Phase 2)

## Decorator Categories

### 1. ValueDecorator - Inline Value Substitution

**Purpose**: Provide values for inline substitution in shell commands

**Interface**:
```go
type ValueDecorator interface {
    Name() string
    Render(ctx *Ctx, args []DecoratorParam) (string, error)
    Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}
```

**Examples**:
- `@var(BUILD_DIR)` - Reference CLI variables
- `@env(HOME)` - Reference environment variables
- `@timestamp()` - Generate timestamp
- `@uuid()` - Generate unique identifier

**Usage**:
```devcmd
build: cd @var(SRC_DIR) && make install PREFIX=@var(INSTALL_DIR)
deploy: docker tag app:latest app:@timestamp()
```

### 2. ActionDecorator - Executable Commands

**Purpose**: Execute commands that return structured CommandResult

**Interface**:
```go
type ActionDecorator interface {
    Name() string
    Run(ctx *Ctx, args []DecoratorParam) CommandResult
    Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}
```

**Examples**:
- `@cmd(build)` - Execute another command
- `@log(message)` - Output logging message
- `@exec(binary, args...)` - Execute external binary

**Usage**:
```devcmd
test: @log("Starting tests") && go test ./... && @log("Tests complete")
deploy: @cmd(build) && @cmd(push) && kubectl apply -f k8s/
```

### 3. BlockDecorator - Execution Wrappers

**Purpose**: Wrap and modify execution behavior of inner commands. BlockDecorators receive **all commands** in the block and have full control over how to execute them.

**Interface**:
```go
type BlockDecorator interface {
    Name() string
    // Receive all commands in the block and decide how to execute them
    WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult
    Describe(ctx *Ctx, args []DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
}
```

**Key Principle**: Decorators get the **raw command sequence** and decide how to execute them. **A decorator is just another CommandSeq with nesting CommandSeq**:

```go
// Context provides execution helpers for decorators
func (ctx *Ctx) ExecSequential(commands CommandSeq) CommandResult  // Run commands in order
func (ctx *Ctx) ExecParallel(steps []CommandStep, mode ParallelMode) CommandResult  // Run steps concurrently  
func (ctx *Ctx) ExecStep(step CommandStep) CommandResult  // Execute a single step
```

**Examples**:
- `@timeout(duration)` - Time-limit execution of all commands
- `@parallel(concurrency)` - Run each command concurrently
- `@retry(count, delay)` - Retry the entire command sequence on failure
- `@workdir(path)` - Execute all commands in a different directory
- `@confirm(message)` - Prompt before executing any commands

**Usage**:
```devcmd
test: @timeout(5m) {
    go test -race ./...     # Command 1
    go test -bench ./...    # Command 2  
}

build: @parallel {
    go build -o bin/server ./cmd/server   # Runs concurrently
    go build -o bin/cli ./cmd/cli         # Runs concurrently
    go build -o bin/worker ./cmd/worker   # Runs concurrently
}
```

**Decorator Implementation Examples**:
```go
// @parallel decorator gets full control
func (p *ParallelDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult {
    return ctx.ExecParallel(commands.Steps, p.getMode(args))
}

// @retry decorator can retry the entire sequence
func (r *RetryDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult {
    for attempt := 1; attempt <= r.getAttempts(args); attempt++ {
        result := ctx.ExecSequential(commands)
        if result.Success() { return result }
        time.Sleep(r.getDelay(args))
    }
    return lastResult
}
```

### 4. PatternDecorator - Conditional Execution

**Purpose**: Select execution branch based on patterns or conditions. PatternDecorators receive **all branches** with their commands and decide which to execute.

**Interface**:
```go
type PatternDecorator interface {
    Name() string
    // Receive all branches with their command sequences and decide which to execute
    SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult
    Describe(ctx *Ctx, args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}
```

**Key Principle**: Pattern decorators get **all branch command sequences** and use evaluation logic to choose which branch(es) to execute:

```go
// @when decorator evaluates condition and executes matching branch
func (w *WhenDecorator) SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult {
    condition := w.evaluateCondition(ctx, args)
    if commands, exists := branches[condition]; exists {
        return ctx.ExecSequential(commands)
    }
    if fallback, exists := branches["default"]; exists {
        return ctx.ExecSequential(fallback)  // Default branch
    }
    return CommandResult{ExitCode: 0}  // No matching branch
}
```

**Examples**:
- `@when(expr)` - Pattern matching based on variable values
- `@try` - Error handling with catch/finally branches

**Usage**:
```devcmd
deploy: @when(@var(ENV)) {
    prod: kubectl apply -f prod.yaml       # Commands for prod branch
    staging: kubectl apply -f staging.yaml # Commands for staging branch  
    default: echo "Unknown environment"    # Default branch commands
}

risky: @try {
    main: ./dangerous-operation.sh         # Main execution branch
    catch: echo "Rolling back..."          # Error handling branch
    finally: ./cleanup.sh                  # Always executed branch
}
```

## Core Architectural Principle

**Decorators Control Execution, Not Just Wrapping**

The key insight is that decorators should receive the **raw commands/branches** and decide how to execute them, rather than receiving a pre-built execution function. This gives decorators full flexibility:

### What Decorators Receive:
- **BlockDecorator**: Gets `CommandSeq` (all commands in the block)
- **PatternDecorator**: Gets `map[string]CommandSeq` (all branches with their commands)

### What Decorators Can Do:
- Execute commands **sequentially** (current behavior): `ctx.ExecSequential(commands)`
- Execute commands **in parallel**: `ctx.ExecParallel(commands.Steps, mode)`
- Execute commands **with retries**: Loop and re-execute `ctx.ExecSequential(commands)`
- Execute commands **selectively**: Choose which commands/branches to run
- Execute commands **with modified context**: Create new context and execute

### Context Helper Methods:
```go
type Ctx struct {
    // ... existing fields ...
    
    // Execution helpers for decorators
    ExecSequential(commands CommandSeq) CommandResult
    ExecParallel(steps []CommandStep, mode ParallelMode) CommandResult  
    ExecStep(step CommandStep) CommandResult
    WithWorkDir(path string) *Ctx  // Create new context with different workdir
    WithTimeout(duration time.Duration) *Ctx  // Create new context with timeout
}
```

This design ensures that:
1. **@parallel** can actually run commands concurrently
2. **@retry** can retry the exact same command sequence
3. **@timeout** can apply timeout to all commands together
4. **@workdir** can execute all commands in the changed directory
5. **@when** can select which branch commands to execute

## Wrapper Stack Model (Nesting Semantics)

* **Blocks compose by nesting.** The engine builds a wrapper stack from outermost to innermost and executes the inner node through that stack.
* **Patterns don't wrap; they **select** a branch** and execute it under the current stack.
* **Failure/return propagation**: inner result bubbles up through wrappers; wrappers may transform/handle failures (e.g., `@retry`).
* **Parallelism**: wrappers that need step-level control (e.g., `@parallel`) require access to **individual steps** of the inner `Seq`.

**Evaluator sketch**

```go
func evalNode(ctx *Ctx, n Node) CommandResult {
  switch x := n.(type) {
  case CommandSeq:
    return evalCommandSeq(ctx, x)
  case Wrapper:
    decorator := reg.Blocks[x.Kind]
    // Decorator receives the inner CommandSeq directly
    return decorator.WrapCommands(ctx, x.Params, x.Inner)
  case Pattern:
    decorator := reg.Patterns[x.Kind]
    // Pattern decorator selects which CommandSeq branch to execute
    return decorator.SelectBranch(ctx, x.Params, x.Branches)
  }
}
```

## Unified Execution Model

### IR Node Types

```go
// Base node interface
type Node interface {
    NodeType() string
}

// Sequence of command steps
type CommandSeq struct {
    Steps []CommandStep
}

// Single command step with chain elements
type CommandStep struct {
    Chain []ChainElement
}

// ChainOp represents shell operators (strongly typed enum)
type ChainOp string

const (
    ChainOpNone   ChainOp = ""    // No operator (first/last element)
    ChainOpAnd    ChainOp = "&&"  // Execute next only if previous succeeded
    ChainOpOr     ChainOp = "||"  // Execute next only if previous failed
    ChainOpPipe   ChainOp = "|"   // Pipe stdout to next stdin
    ChainOpAppend ChainOp = ">>"  // Append stdout to file
)

// Chain element (shell command or action)
type ChainElement struct {
    Kind   string               // "shell" | "action"
    Name   string               // For actions
    Text   string               // For shell commands
    Args   []DecoratorParam // Decorator arguments
    OpNext ChainOp              // Operator to next element
    Target string               // Target file for >> operator
}

// Block wrapper
type Wrapper struct {
    Kind   string
    Params map[string]interface{}
    Inner  Node
}

// Pattern with branches
type Pattern struct {
    Kind     string
    Params   map[string]interface{}
    Branches []PatternBranch
}
```

### Execution Flow

1. **AST to IR Transformation**:
   - Parse commands.cli → AST
   - Transform AST → IR nodes
   - Decorators become Wrapper/Pattern nodes

2. **IR Evaluation**:
   - NodeEvaluator traverses IR tree
   - Decorators called via registry
   - Consistent operator semantics

3. **Plan Generation**:
   - Same IR tree traversal
   - Decorators provide descriptions
   - Structured plan output

## Enhanced Decorator Implementation

### Base Interface with Metadata

```go
// Base interface for all decorators
type DecoratorBase interface {
    Name() string
    Description() string
    ParameterSchema() []ParameterSchema
    Examples() []Example
}

// Parameter schema for validation
type ParameterSchema struct {
    Name        string
    Type        ast.ExpressionType
    Required    bool
    Description string
    Default     interface{}
}

// Example for documentation
type Example struct {
    Code        string
    Description string
}
```

### Lightweight Code Generation Patterns

Decorators can optionally use the lightweight `codegen` module for code generation hints. This module provides simple patterns without heavy coupling:

```go
// From github.com/aledsdavies/devcmd/codegen package
type GenOps interface {
    // Basic operations
    Shell(cmd string) TempResult
    CallAction(name string, args []DecoratorParam) TempResult
    
    // Chain operations (shell operators)
    And(left, right TempResult) TempResult        // && operator
    Or(left, right TempResult) TempResult         // || operator  
    Pipe(left, right TempResult) TempResult       // | operator
    Append(result TempResult, filename string) TempResult // >> operator
    
    // Value operations
    Var(name string) TempResult                   // @var(name)
    Env(name string) TempResult                   // @env(name)
    Literal(value string) TempResult              // String literal
    
    // Block operations
    Sequential(steps ...TempResult) TempResult    // Execute in sequence
    Parallel(steps ...TempResult) TempResult      // Execute in parallel
    
    // Context operations  
    WithWorkdir(dir string, body func(GenOps) TempResult) TempResult
    WithEnv(key, value string, body func(GenOps) TempResult) TempResult
    WithTimeout(seconds int, body func(GenOps) TempResult) TempResult
}

// Optional interfaces for decorators that want to provide code generation hints
type GenerateHint interface {
    GenerateHint(ops GenOps, args []DecoratorParam) TempResult
}

type BlockGenerateHint interface {
    GenerateBlockHint(ops GenOps, args []DecoratorParam, body func(GenOps) TempResult) TempResult
}

type PatternGenerateHint interface {
    GeneratePatternHint(ops GenOps, args []DecoratorParam, branches map[string]func(GenOps) TempResult) TempResult
}
```

**Key Benefits:**
- **Lightweight**: Decorators can import just the `codegen` module without coupling to the full project
- **Optional**: Decorators work perfectly without implementing generation hints
- **Self-contained**: Simple patterns focused on code generation utilities
- **Modular**: Separate workspace module that can evolve independently

### Complete Decorator Interfaces

```go
// ValueDecorator - Inline value substitution
type ValueDecorator interface {
    DecoratorBase
    // Runtime execution
    Render(ctx *Ctx, args []DecoratorParam) (string, error)
    // Plan generation
    Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}

// ActionDecorator - Executable commands that return CommandResult
type ActionDecorator interface {
    DecoratorBase
    // Runtime execution
    Run(ctx *Ctx, args []DecoratorParam) CommandResult
    // Plan generation
    Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}

// BlockDecorator - Execution wrappers that receive CommandSeq directly
type BlockDecorator interface {
    Name() string
    // Receive CommandSeq and decide how to execute it
    WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult
    // Plan generation
    Describe(ctx *Ctx, args []DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
}

// PatternDecorator - Conditional execution with branch selection
type PatternDecorator interface {
    Name() string
    // Receive all branches with their CommandSeq and decide which to execute
    SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult
    // Plan generation
    Describe(ctx *Ctx, args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}
```

**Simplified Design:**
- **Core Methods Only**: Each decorator implements only the essential `Run/Render/Wrap/Select` + `Describe` methods
- **No Complex Interfaces**: Removed graph description and code generation from core interfaces
- **Optional Enhancement**: Decorators can optionally implement `GenerateHint` interfaces from the `codegen` package
- **Self-Contained**: Each decorator is fully functional with just these core methods

## Determinism and Environment Handling

### Working Directory Context Pattern

**Never use `os.Chdir()` - always pass working directory through context**

All command execution must use the working directory from context, not change the global process directory:

```go
// ✅ Correct - pass working directory to command
func (d *MyDecorator) Run(ctx *Ctx, args []DecoratorParam) CommandResult {
    cmd := exec.Command("make", "build")
    cmd.Dir = ctx.WorkDir  // Use context working directory
    return executeCommand(cmd)
}

// ❌ Wrong - changes global process state
func (d *MyDecorator) Run(ctx *Ctx, args []DecoratorParam) CommandResult {
    os.Chdir(ctx.WorkDir)  // Breaks concurrency!
    return executeCommand(exec.Command("make", "build"))
}
```

**Context Working Directory Management:**

```go
// Create new context with different working directory
newCtx := ctx.WithWorkDir("/path/to/project")

// @workdir decorator implementation
func (w *WorkdirDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult {
    workDir := extractWorkDir(args)
    newCtx := ctx.WithWorkDir(workDir)
    return ctx.ExecSequential(commands)  // All inner commands use new working directory
}
```

**Benefits:**
- **Thread safety**: Multiple commands can run concurrently in different directories
- **Predictable execution**: No global state changes affecting other operations
- **Testability**: Easy to test with different working directories in parallel
- **Isolation**: Commands don't affect each other's execution environment

### Environment Freeze Pattern

All decorators must read environment variables via `ctx.Env` (frozen snapshot), never `os.Getenv()`:

```go
// ✅ Correct - deterministic
func (e *EnvDecorator) Render(ctx *Ctx, args []DecoratorParam) (string, error) {
    key := extractKey(args)
    value, exists := ctx.Env.Get(key)  // From frozen snapshot
    if !exists {
        return defaultValue, nil
    }
    return value, nil
}

// ❌ Wrong - non-deterministic 
func (e *EnvDecorator) Render(ctx *Ctx, args []DecoratorParam) (string, error) {
    key := extractKey(args)
    return os.Getenv(key), nil  // Breaks determinism!
}
```

**Benefits:**
- **Deterministic execution**: Same environment across interpreter and generated modes
- **Plan consistency**: Plans show actual values that will be used
- **Reproducible builds**: Environment captured once per process or from lock file

### Value Expansion and Quoting

`ValueDecorator.Render` runs in **shell text** and **action args** at execution time with no automatic quoting:

```devcmd
# Value decorators expand to raw strings - no auto-quoting
build: cd @var(BUILD_DIR) && make  # ✅ Safe if BUILD_DIR="./build"
deploy: kubectl --context @env(KUBE_CONTEXT) apply  # ❌ Unsafe if KUBE_CONTEXT contains spaces

# Use manual quoting or helper for safety
deploy: kubectl --context "@env(KUBE_CONTEXT)" apply  # ✅ Manual quoting
deploy: kubectl --context @quote(@env(KUBE_CONTEXT)) apply  # ✅ Decorator quoting
```

**Portability Guidelines:**
- Provide `shq(string) string` helper for POSIX shell quoting
- Consider `@quote(...)` value decorator for automatic quoting
- Document that decorators return raw strings without escaping

### Step Failure Rules and Newline Semantics

**Step Failure Rule**: A step "fails" if and only if its final chain result has exit code ≠ 0. The `&& || |` operators within the step determine that final result.

**Default Behavior**: Steps short-circuit on failure (like shell `set -e`). If a step fails, subsequent steps are **not executed**:

```devcmd
# Two separate steps - second won't run if first fails
deploy: {
    npm run build     # Step 1: fails with exit code 1
    kubectl apply     # Step 2: NOT executed (short-circuit)
}

# Single step - kubectl only runs if npm succeeds  
deploy: npm run build && kubectl apply

# Line continuation - still single step
deploy: npm run build && \
        kubectl apply
```

**Disable Short-Circuit**: To ignore failures and continue, use explicit error handling:

```devcmd
deploy: {
    npm run build || echo "Build failed, continuing anyway"
    kubectl apply     # This will run even if build failed
}
```

### @parallel Semantics

**Three execution modes** configurable via `mode` parameter:

1. **fail-fast** (default): Stop scheduling on first failure, wait for running tasks, return first error
2. **fail-immediate**: Stop scheduling and cancel running tasks immediately  
3. **all**: Run all tasks to completion, aggregate all errors

**stdout/stderr merging**: By start time, tagged with step ID

```devcmd
# Default: fail-fast mode
build: @parallel {
    go build ./cmd/server   # If this fails...
    go build ./cmd/cli      # ...stop scheduling remaining tasks
    go build ./cmd/worker   # ...but wait for running tasks to complete
}

# Immediate cancellation mode
build: @parallel(mode="immediate") {
    go build ./cmd/server   # If this fails...
    go build ./cmd/cli      # ...cancel all running tasks immediately
    go build ./cmd/worker   # ...return first error ASAP
}

# Run all mode
build: @parallel(mode="all") {
    go build ./cmd/server   # All three run to completion
    go build ./cmd/cli      # Even if some fail
    go build ./cmd/worker   # Errors are aggregated
}
```

### Pattern Determinism

**Branch order**: Must be deterministic in both execution and plan generation
**Selected branch**: When decidable from frozen env/const `@var`, include `selected` in `Describe`

```go
func (w *WhenDecorator) Describe(ctx *Ctx, args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep {
    // Evaluate condition deterministically
    value := evaluateCondition(ctx, args)
    selectedBranch := selectBranch(value, branches)
    
    return plan.ExecutionStep{
        Type: plan.StepConditional,
        Condition: &plan.ConditionInfo{
            Variable: extractVariable(args),
            Evaluation: plan.ConditionResult{
                CurrentValue:   value,
                SelectedBranch: selectedBranch,
                Reason:         "Evaluated from frozen environment",
            },
            Branches: sortedBranches(branches), // Deterministic order
        },
    }
}
```

### Pipe to Actions

**Rule**: If an action is on the right side of `|` and doesn't implement `StdinAware`, raise error:

```go
// In chain evaluation
if element.Kind == "action" && hasPipeInput {
    decorator := registry.Actions[element.Name]
    if stdinDecorator, ok := decorator.(StdinAware); ok {
        result = stdinDecorator.RunWithInput(ctx, element.Args, pipeInput)
    } else {
        return CommandResult{
            Stderr:   fmt.Sprintf("Action @%s is not pipe-capable", element.Name),
            ExitCode: 1,
        }
    }
}
```

### Execution Output Streaming

**Memory-Efficient Design**: Handle execution output without loading it all into memory:

**For Pipes (`|` operator)**:
```go
// Stream data directly between processes using io.Pipe
func executePipe(left, right Command) CommandResult {
    r, w := io.Pipe()
    defer r.Close()
    
    // Connect stdout -> stdin directly, no buffering
    left.Stdout = w
    right.Stdin = r
    
    // Start both processes, data flows through pipe
    go runLeft(left); w.Close() // Signal EOF when done
    return runRight(right) // Return exit code from right side
}
```

**For CommandResult capture**:
```go
// Only capture what's needed for error reporting/display
type CommandResult struct {
    Stdout   string `json:"stdout"`   // For small outputs, error context
    Stderr   string `json:"stderr"`   // Error messages only
    ExitCode int    `json:"exit_code"`
}

// For large outputs, stream to file or discard
func execLargeOutput(cmd Command) CommandResult {
    if cmd.expectsLargeOutput {
        // Stream directly to file, don't capture in memory
        cmd.Stdout = outputFile
        return CommandResult{
            Stdout: "[output streamed to file.log]",
            ExitCode: cmd.Run(),
        }
    } else {
        // Small outputs can be captured normally
        return execWithCapture(cmd)
    }
}
```

**Plan Generation**: Plans contain only structural descriptions (naturally small), not captured execution output.

### >> Append Operator Semantics

The `>>` operator appends the **stdout** of the previous element to a file:

```devcmd
# Append command output to file
build: go build -v . 2>&1 >> build.log   # Append stdout to build.log
test: go test -v ./... >> test-results.txt  # Append test output

# Chain continues after successful append
deploy: {
    kubectl apply -f k8s/ >> deploy.log     # Append deployment output
    echo "Deployment complete" >> deploy.log # Continue if append succeeded
}
```

**Result**: The `>>` operation itself returns success (exit code 0) unless the append operation fails (file permissions, disk space, etc.).

## Registry System

The registry uses the **database/sql driver pattern** for decorator registration, enabling both static compilation and dynamic plugin loading.

### Registration Pattern

Decorators register themselves via `init()` functions using the global registry:

```go
// Built-in decorator registration
package builtin

import "github.com/aledsdavies/devcmd/core/decorators"

func init() {
    decorators.Register(NewVarDecorator())     // @var
    decorators.RegisterAction(NewLogDecorator()) // @log
}
```

**Application Usage:**
```go
package main

import _ "github.com/aledsdavies/devcmd/runtime/decorators/builtin"
import _ "github.com/user/docker-plugin"

func main() {
    registry := decorators.GlobalRegistry()
    // All decorators auto-registered via init()
}
```

### Plugin Support

The init() pattern enables multiple plugin loading mechanisms:

**Static Plugins (Compile-time):**
```go
import _ "github.com/user/docker-plugin"
// Decorators registered automatically via init()
```

**Dynamic Plugins (.so files):**
```go
// docker-plugin.so source
package main

import "github.com/aledsdavies/devcmd/core/decorators"

func init() {
    decorators.RegisterAction(&DockerBuildDecorator{})
}

// Runtime loading
p, err := plugin.Open("./plugins/docker-plugin.so")
// init() runs automatically, decorators now available
```

**Benefits:**
- **Universal pattern**: Same registration for static and dynamic plugins
- **No circular dependencies**: Plugins only import core/decorators
- **Familiar Go idiom**: Like database/sql drivers
- **Plugin discovery**: Can scan directories and auto-load
- **Isolation**: Each plugin registers independently

The registry uses **fully-qualified decorator names** to avoid collisions and supports **injected dependencies** for testing.

### Registry Hygiene

**Collision Detection**: Registration fails on duplicate names **across all categories**, not just within one map:

```go
// ❌ This should fail - "log" exists as ActionDecorator
func (r *Registry) RegisterValue(decorator ValueDecorator) error {
    name := decorator.Name()
    
    // Check for collisions across ALL categories
    if _, exists := r.Actions[name]; exists {
        return fmt.Errorf("decorator %q already registered as Action", name)
    }
    if _, exists := r.Blocks[name]; exists {
        return fmt.Errorf("decorator %q already registered as Block", name)  
    }
    if _, exists := r.Patterns[name]; exists {
        return fmt.Errorf("decorator %q already registered as Pattern", name)
    }
    if _, exists := r.Values[name]; exists {
        return fmt.Errorf("decorator %q already registered as Value", name)
    }
    
    r.Values[name] = decorator
    return nil
}
```

**Fully-Qualified Names**: Use package prefixes for third-party decorators:

```go
type Registry struct {
    mu sync.RWMutex // Thread-safe access
    
    Actions  map[string]ActionDecorator
    Blocks   map[string]BlockDecorator
    Values   map[string]ValueDecorator
    Patterns map[string]PatternDecorator
}

func NewRegistry() *Registry {
    return &Registry{
        Actions:  make(map[string]ActionDecorator),
        Blocks:   make(map[string]BlockDecorator),
        Values:   make(map[string]ValueDecorator),
        Patterns: make(map[string]PatternDecorator),
    }
}

// RegisterValue with collision detection
func (r *Registry) RegisterValue(decorator ValueDecorator) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    name := decorator.Name()
    if _, exists := r.Values[name]; exists {
        return fmt.Errorf("value decorator %q already registered", name)
    }
    
    r.Values[name] = decorator
    return nil
}

// Similar for other decorator types...

func NewStandardRegistry() *Registry {
    r := NewRegistry()
    
    // Register builtin decorators with qualified names
    // Built-ins use simple names, third-party use "vendor.name"
    r.RegisterValue(builtin.NewVarDecorator())     // "var"
    r.RegisterValue(builtin.NewEnvDecorator())     // "env"
    r.RegisterAction(builtin.NewLogDecorator())    // "log"
    r.RegisterAction(builtin.NewCmdDecorator())    // "cmd"
    r.RegisterBlock(builtin.NewTimeoutDecorator()) // "timeout"
    r.RegisterBlock(builtin.NewParallelDecorator()) // "parallel"
    r.RegisterBlock(builtin.NewRetryDecorator())   // "retry"
    r.RegisterBlock(builtin.NewWorkdirDecorator()) // "workdir"
    r.RegisterPattern(builtin.NewWhenDecorator())  // "when"
    r.RegisterPattern(builtin.NewTryDecorator())   // "try"
    
    // Third-party example:
    // r.RegisterBlock(&acme.CustomTimeout{}) // "acme.timeout"
    
    return r
}
```

## Example Implementation: @var Decorator

```go
package builtin

import (
    "fmt"
    "github.com/aledsdavies/devcmd/core/ast"
    "github.com/aledsdavies/devcmd/runtime/execution"
    "github.com/aledsdavies/devcmd/core/plan"
    "github.com/aledsdavies/devcmd/codegen" // Optional import for code generation hints
)

type VarDecorator struct{}

func NewVarDecorator() *VarDecorator {
    return &VarDecorator{}
}

func (v *VarDecorator) Name() string {
    return "var"
}

func (v *VarDecorator) Description() string {
    return "Reference variables defined in the CLI file"
}

func (v *VarDecorator) ParameterSchema() []execution.ParameterSchema {
    return []execution.ParameterSchema{
        {
            Name:        "name",
            Type:        ast.IdentifierType,
            Required:    true,
            Description: "Variable name to reference",
        },
    }
}

func (v *VarDecorator) Examples() []execution.Example {
    return []execution.Example{
        {
            Code:        "@var(BUILD_DIR)",
            Description: "Reference the BUILD_DIR variable",
        },
        {
            Code:        "cd @var(SRC) && make",
            Description: "Use variable in shell command",
        },
    }
}

func (v *VarDecorator) ImportRequirements() execution.ImportRequirement {
    return execution.ImportRequirement{} // No special imports needed
}

// Core interface implementation
func (v *VarDecorator) Render(ctx *execution.Ctx, args []DecoratorParam) (string, error) {
    varName := v.extractVarName(args)
    if value, exists := ctx.Vars[varName]; exists {
        return value, nil
    }
    return "", fmt.Errorf("undefined variable: %s", varName)
}

func (v *VarDecorator) Describe(ctx *execution.Ctx, args []DecoratorParam) plan.ExecutionStep {
    varName := v.extractVarName(args)
    value := "<undefined>"
    if val, exists := ctx.Vars[varName]; exists {
        value = val
    }
    return plan.Text(fmt.Sprintf("@var(%s) = %s", varName, value)).Build()
}

// Optional code generation hint
func (v *VarDecorator) GenerateHint(ops codegen.GenOps, args []DecoratorParam) codegen.TempResult {
    varName := v.extractVarName(args)
    return ops.Var(varName) // Use GenOps pattern for variable reference
}

func (v *VarDecorator) extractVarName(args []DecoratorParam) string {
    if len(args) == 0 {
        return ""
    }
    // Extract from first positional argument
    if args[0].Name == "" {
        switch val := args[0].Value.(type) {
        case *ast.StringLiteral:
            return val.Value
        case *ast.Identifier:
            return val.Name
        }
    }
    return ""
}
```

**Key Features:**
- **Core Implementation**: `Render` and `Describe` methods provide full functionality
- **Optional Enhancement**: `GenerateHint` method uses the lightweight `codegen` module
- **Self-Contained**: Works perfectly even if `GenerateHint` is not implemented
- **Lightweight Coupling**: Only imports `codegen` package, not the entire project

## Code Generation Strategy

### IR-Driven Code Generation with Optional Decorator Hints

The code generator is IR-driven, walking the IR tree and generating code patterns. Decorators can optionally provide generation hints via the lightweight `codegen` module:

```go
// IR-driven code generator
func (g *CodeGenerator) GenerateCommand(cmd *IRCommand) string {
    return g.generateNode(cmd.Root)
}

func (g *CodeGenerator) generateNode(node IRNode) string {
    switch n := node.(type) {
    case *CommandSeq:
        return g.generateSequence(n)
    case *Wrapper:
        return g.generateWrapper(n)
    case *Pattern:
        return g.generatePattern(n)
    }
}

func (g *CodeGenerator) generateSequence(seq *CommandSeq) string {
    var steps []string
    for _, step := range seq.Steps {
        steps = append(steps, g.generateStep(step))
    }
    return g.joinSteps(steps)
}

func (g *CodeGenerator) generateStep(step CommandStep) string {
    var elements []string
    for i, element := range step.Chain {
        var elemCode string
        
        switch element.Kind {
        case "shell":
            elemCode = g.generateShell(element.Text)
        case "action":
            elemCode = g.generateAction(element.Name, element.Args)
        }
        
        if i == 0 {
            elements = append(elements, elemCode)
        } else {
            // Apply operator from previous element
            prevOp := step.Chain[i-1].OpNext
            elements = append(elements, g.generateOperator(prevOp, elements[len(elements)-1], elemCode))
        }
    }
    return strings.Join(elements, "")
}

func (g *CodeGenerator) generateAction(name string, args []DecoratorParam) string {
    decorator := g.registry.Actions[name]
    
    // Check if decorator provides a generation hint
    if hinter, ok := decorator.(codegen.GenerateHint); ok {
        ops := g.newGenOps()
        hint := hinter.GenerateHint(ops, args)
        return hint.String()
    }
    
    // Default: generate call to action decorator
    return fmt.Sprintf("callActionDecorator(%q, %s)", name, g.formatArgs(args))
}
```

### Lightweight GenOps Implementation

```go
// Simple Go backend for GenOps patterns
type GoGenOps struct {
    counter int
}

func (g *GoGenOps) Shell(cmd string) codegen.TempResult {
    return codegen.NewTempResult(fmt.Sprintf("ExecShell(ctx, %s)", codegen.QuoteString(cmd)))
}

func (g *GoGenOps) Var(name string) codegen.TempResult {
    // CONST-PROP LOGIC: Route through generator's const-prop table
    // @var(NAME) becomes Go const if determinable at build time
    // Otherwise becomes runtime helper function call
    if g.isConstVar(name) {
        return codegen.NewTempResult(fmt.Sprintf("CONST_%s", codegen.SanitizeIdentifier(name)))
    } else {
        return codegen.NewTempResult(fmt.Sprintf("VAR_%s(ctx)", codegen.SanitizeIdentifier(name)))
    }
}

func (g *GoGenOps) Env(name string) codegen.TempResult {
    // @env is ALWAYS runtime via ctx.Env - never becomes a Go const
    // This preserves portability and determinism via frozen environment
    return codegen.NewTempResult(fmt.Sprintf("ctx.Env.Get(%s)", codegen.QuoteString(name)))
}

func (g *GoGenOps) And(left, right codegen.TempResult) codegen.TempResult {
    return codegen.NewTempResult(fmt.Sprintf(`func() CommandResult {
        leftResult := %s
        if leftResult.Failed() {
            return leftResult
        }
        return %s
    }()`, left.String(), right.String()))
}

// Similar simple patterns for other operations...
```

### Benefits

- **IR-Driven**: Code generation logic centralized in the IR walker
- **Optional Hints**: Decorators work perfectly without providing generation hints  
- **Lightweight Coupling**: Decorators only import the small `codegen` module if needed
- **Simple Patterns**: GenOps provides helpful utilities without complexity
- **Optimization Control**: Generator handles const-prop and other optimizations
- **Self-Contained Decorators**: Core functionality independent of code generation

## Migration Strategy

1. **Phase 1**: Create new decorator interfaces alongside existing
2. **Phase 2**: Implement core decorators (@var, @env, @cmd, @log)
3. **Phase 3**: Update ChainEvaluator to use new registry
4. **Phase 4**: Test semantic equivalence between modes
5. **Phase 5**: Migrate remaining decorators incrementally
6. **Phase 6**: Update code generator to use new decorators
7. **Phase 7**: Remove old template-based system

## Benefits

- **Consistency**: Same behavior across all execution modes
- **Maintainability**: Clean separation of concerns
- **Debuggability**: Generated code is readable and debuggable
- **Performance**: Optimized execution paths
- **Extensibility**: Easy to add new decorators
- **Type Safety**: Strong typing with validation
- **Documentation**: Built-in examples and schemas

## Testing Strategy

### Decorator Testing Harness

Ship a helper that tests decorator behavior across all execution modes:

```go
// DecoratorTestHarness validates decorators across modes
type DecoratorTestHarness struct {
    registry *Registry
    noBuild  bool
}

func NewDecoratorTestHarness(registry *Registry) *DecoratorTestHarness {
    return &DecoratorTestHarness{registry: registry}
}

func (h *DecoratorTestHarness) WithNoBuild() *DecoratorTestHarness {
    h.noBuild = true
    return h
}

// TestDecorator validates a decorator across interpreter and generated modes
func (h *DecoratorTestHarness) TestDecorator(t *testing.T, testCase DecoratorTest) {
    t.Run(testCase.Name, func(t *testing.T) {
        // 1. Create synthetic command from IR
        cmd := &Command{
            Name: "test",
            Root: testCase.IR,
        }
        
        // 2. Test interpreter mode
        interpreterResult := h.runInterpreter(testCase.Ctx, cmd)
        
        // 3. Test generated mode (unless disabled)
        if !h.noBuild {
            generatedResult := h.runGenerated(testCase.Ctx, cmd)
            
            // 4. Assert semantic equivalence
            h.assertEquivalent(t, interpreterResult, generatedResult)
        }
        
        // 5. Test plan mode
        interpreterPlan := h.generatePlan(testCase.Ctx, cmd, "interpreter")
        generatedPlan := h.generatePlan(testCase.Ctx, cmd, "generated")
        
        // 6. Assert plan equivalence (JSON and graph hash)
        h.assertPlanEquivalent(t, interpreterPlan, generatedPlan)
        h.assertGraphHashEqual(t, interpreterPlan, generatedPlan)
        
        // 7. Test multiple output formats
        h.assertDOTEquivalent(t, interpreterPlan, generatedPlan)
        
        // 8. Optionally test that generated code builds
        if testCase.TestCodegen && !h.noBuild {
            h.assertCodeBuilds(t, cmd)
        }
    })
}

type DecoratorTest struct {
    Name        string
    IR          Node                 // IR to test
    Ctx         *Ctx                // Execution context
    Expected    CommandResult       // Expected result
    TestCodegen bool                // Whether to test code generation
}

func (h *DecoratorTestHarness) assertEquivalent(t *testing.T, interpreter, generated CommandResult) {
    assert.Equal(t, interpreter.ExitCode, generated.ExitCode, "exit codes must match")
    assert.Equal(t, interpreter.Stdout, generated.Stdout, "stdout must match")
    assert.Equal(t, interpreter.Stderr, generated.Stderr, "stderr must match")
}

func (h *DecoratorTestHarness) assertPlanEquivalent(t *testing.T, plan1, plan2 *plan.ExecutionPlan) {
    // Compare JSON representation (structural equivalence)
    json1, _ := json.Marshal(plan1)
    json2, _ := json.Marshal(plan2)
    assert.JSONEq(t, string(json1), string(json2), "plan JSON must be equivalent")
}

func (h *DecoratorTestHarness) assertGraphHashEqual(t *testing.T, plan1, plan2 *plan.ExecutionPlan) {
    // Compare plan graph hashes for structural equivalence
    hash1 := plan1.GraphHash()
    hash2 := plan2.GraphHash()
    assert.Equal(t, hash1, hash2, "plan graph hashes must be equal")
}

func (h *DecoratorTestHarness) assertDOTEquivalent(t *testing.T, plan1, plan2 *plan.ExecutionPlan) {
    // Compare DOT output (graph structure equivalence)
    dot1 := plan1.ToDOT()
    dot2 := plan2.ToDOT()
    assert.Equal(t, dot1, dot2, "DOT output must be equivalent")
}
```

### Example Decorator Test

```go
func TestVarDecorator(t *testing.T) {
    registry := NewStandardRegistry()
    harness := NewDecoratorTestHarness(registry)
    
    harness.TestDecorator(t, DecoratorTest{
        Name: "var substitution",
        IR: CommandSeq{
            Steps: []CommandStep{
                {
                    Chain: []ChainElement{
                        {Kind: "shell", Text: "echo @var(MESSAGE)"},
                    },
                },
            },
        },
        Ctx: &Ctx{
            Vars: map[string]string{"MESSAGE": "hello world"},
            Env:  &EnvSnapshot{Values: map[string]string{}},
        },
        Expected: CommandResult{
            Stdout:   "hello world\n",
            ExitCode: 0,
        },
        TestCodegen: true,
    })
}
```

## Data Resolution Strategies

Decorators that fetch external data must choose an appropriate **resolution timing strategy**. The decorator itself decides when and how to resolve values, with special considerations for plan generation modes.

### Resolution Strategy: Hybrid Caching (Recommended for Value Decorators)

Value decorators should implement **hybrid resolution with smart caching** to support devcmd's two-tier plan system:

```go
type ValueDecorator struct {
    cache map[string]cacheEntry
    mutex sync.RWMutex
}

type cacheEntry struct {
    value      string
    source     string    // "environment", "http", "config", etc.
    resolvedAt time.Time
    ttl        time.Duration
}

func (d *ValueDecorator) Render(ctx *Ctx, args []DecoratorParam) (string, error) {
    key := d.getCacheKey(args)
    
    // Check cache first
    d.mutex.RLock()
    if entry, exists := d.cache[key]; exists {
        if entry.ttl == 0 || time.Since(entry.resolvedAt) < entry.ttl {
            d.mutex.RUnlock()
            return entry.value, nil
        }
    }
    d.mutex.RUnlock()
    
    // Resolve and cache
    value, source, err := d.resolveValue(ctx, args)
    if err != nil {
        return "", err
    }
    
    d.mutex.Lock()
    d.cache[key] = cacheEntry{
        value:      value,
        source:     source,
        resolvedAt: time.Now(),
        ttl:        d.getTTL(source), // Different TTL per source type
    }
    d.mutex.Unlock()
    
    return value, nil
}
```

## Two-Tier Plan System

devcmd supports two distinct plan generation modes to balance performance with determinism:

### 1. Quick Plans (Default `--dry-run`)

**Purpose**: Fast development preview showing command structure and immediately available values

**Resolution behavior**: 
- Show cached values if already resolved
- Display placeholders for expensive operations (HTTP calls, API queries, etc.)
- Prioritize speed over complete resolution

**Example output**:
```
build:
└─ docker build -t myapp¹:@http(get-version)² ./build³ && push to docker.io⁴
   {¹IMAGE_NAME=myapp, ²@http(get-version)=<will resolve at runtime>, ³BUILD_DIR=./build, ⁴DOCKER_REGISTRY=docker.io}
```

**Usage**: `devcmd run build --dry-run`

### 2. Resolved Plans (`--dry-run --resolve` or plan files)

**Purpose**: Create fully deterministic execution plans with all values resolved and frozen

**Resolution behavior**:
- Force resolution of ALL value decorators, including expensive operations
- Create frozen execution context that can be reused
- Generate plan files that can be executed independently

**Example output**:
```
build:
└─ docker build -t myapp¹:v2.1.4² ./build³ && push to docker.io⁴
   {¹IMAGE_NAME=myapp, ²VERSION=v2.1.4(http), ³BUILD_DIR=./build, ⁴DOCKER_REGISTRY=docker.io}
```

**Usage**: 
- `devcmd run build --dry-run --resolve`
- `devcmd plan build --output build.plan`

### 3. Plan Execution

**Purpose**: Execute commands using pre-resolved plan files with frozen context

**Resolution behavior**:
- No resolution needed - all values loaded from plan file
- Deterministic execution matching the generated plan exactly
- Supports offline execution when external dependencies are unavailable

**Usage**: `devcmd exec build.plan`

## Decorator Implementation for Plan Modes

### Context-Aware Resolution

Decorators must adapt their behavior based on the plan generation context:

```go
func (d *ValueDecorator) Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep {
    key := d.getCacheKey(args)
    
    if ctx.PlanMode == "resolved" {
        // Resolved plans: Force resolution and cache all values
        value, source, err := d.resolveValue(ctx, args)
        if err != nil {
            return plan.ExecutionStep{
                Description: fmt.Sprintf("@%s(%s) → <error: %v>", d.Name(), key, err),
                Metadata: map[string]string{
                    "error": err.Error(),
                    "resolution": "failed",
                },
            }
        }
        
        // Cache for reuse during execution
        d.cacheValue(key, value, source)
        
        return plan.ExecutionStep{
            Description: fmt.Sprintf("@%s(%s) → %q", d.Name(), key, value),
            Command:     value,
            Metadata: map[string]string{
                "decorator":       d.Name(),
                "source":          source,
                "resolved_at":     time.Now().Format(time.RFC3339),
                "resolve_at":      "plan",           // NEW: resolved at plan time
                "pinned":          "plan",           // NEW: frozen in plan file  
                "nondeterministic": d.isNondeterministic(key), // NEW: @timestamp=true, @var=false
            },
        }
    } else {
        // Quick plans: Show cached values or placeholders
        if cached := d.getCachedValue(key); cached != nil {
            return plan.ExecutionStep{
                Description: fmt.Sprintf("@%s(%s) → %q", d.Name(), key, cached.value),
                Command:     cached.value,
                Metadata: map[string]string{
                    "decorator":        d.Name(),
                    "source":           cached.source,
                    "cached":           "true",
                    "cache_age":        time.Since(cached.resolvedAt).String(), // NEW: cache age
                    "resolve_at":       "plan",                                  // NEW: was resolved at plan
                    "pinned":           "",                                      // NEW: not pinned (cached)
                    "nondeterministic": d.isNondeterministic(key),
                },
            }
        }
        
        return plan.ExecutionStep{
            Description: fmt.Sprintf("@%s(%s) → <will resolve at runtime>", d.Name(), key),
            Command:     fmt.Sprintf("<%s:%s>", d.Name(), key),
            Metadata: map[string]string{
                "decorator":        d.Name(),
                "resolution":       "runtime",           // Legacy field
                "resolve_at":       "exec",              // NEW: will resolve at exec time
                "pinned":           "",                  // NEW: not pinned
                "nondeterministic": d.isNondeterministic(key), // NEW: varies by decorator
            },
        }
    }
}
```

### Plan File Format

Resolved plans generate JSON files containing frozen execution context:

```json
{
  "command": "build",
  "generated_at": "2024-01-15T10:30:00Z",
  "devcmd_version": "0.2.0",
  "resolved_context": {
    "vars": {
      "IMAGE_NAME": "myapp",
      "BUILD_DIR": "./build"
    },
    "env": {
      "VERSION": "v2.1.4", 
      "DOCKER_REGISTRY": "docker.io"
    },
    "resolution_metadata": {
      "VERSION": {
        "source": "http",
        "resolved_at": "2024-01-15T10:30:00Z",
        "url": "https://api.internal/version"
      },
      "DOCKER_REGISTRY": {
        "source": "environment",
        "resolved_at": "2024-01-15T10:30:00Z"
      }
    }
  },
  "execution_plan": {
    "steps": [...],
    "summary": {...}
  }
}
```

## Design Guidelines for Decorator Authors

### 1. Implement Hybrid Caching
- Cache resolved values for reuse within the same execution context
- Use appropriate TTL based on data source type
- Thread-safe caching for concurrent access

### 2. Support Both Plan Modes
- **Quick plans**: Show cached values or runtime placeholders
- **Resolved plans**: Force resolution of all values
- Provide clear metadata about resolution status

### 3. Choose Appropriate Resolution Strategy
- **Cheap operations** (variables, environment): Always resolve
- **Expensive operations** (HTTP, database): Cache with TTL
- **Dynamic operations** (timestamps, UUIDs): Resolve once per execution

### 4. Error Handling
- Graceful degradation in quick plans (show placeholders on errors)
- Clear error reporting in resolved plans (fail if resolution required)
- Use standardized error metadata format

**Standardized Error Metadata**:
```go
func (d *MyDecorator) Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep {
    value, err := d.resolveValue(ctx, args)
    if err != nil {
        return plan.ExecutionStep{
            Description: fmt.Sprintf("@%s(%s) → <error: %v>", d.Name(), key, err),
            Metadata: map[string]string{
                "decorator":        d.Name(),
                "resolution":       "failed",
                "error_class":      "network_timeout",    // Structured error type
                "error_msg":        err.Error(),          // Human readable message
                "resolve_at":       "plan",               // Where resolution failed
                "nondeterministic": "true",               // If applicable
            },
        }
    }
    // Success case...
}
```

### Resolution Strategy Examples

```go
// Example: @var - cheap, always resolve, static
func (v *VarDecorator) getCacheStrategy() CacheStrategy {
    return CacheStrategy{
        Scope:            "global",      // Cache across all executions
        Nondeterministic: false,         // Variables are static
    }
}

// Example: @env - cheap, cache for execution  
func (e *EnvDecorator) getCacheStrategy() CacheStrategy {
    return CacheStrategy{
        Scope:            "execution",   // Cache per execution
        Nondeterministic: false,         // Environment stable during run
    }
}

// Example: @http - expensive, cache with TTL
func (h *HTTPDecorator) getCacheStrategy() CacheStrategy {
    return CacheStrategy{
        Scope:            "execution",   // Fresh per execution
        TTL:              5 * time.Minute, // Cache within execution
        Nondeterministic: true,          // HTTP responses change
    }
}

// Example: @timestamp - nondeterministic, cache per execution
func (t *TimestampDecorator) getCacheStrategy() CacheStrategy {
    return CacheStrategy{
        Scope:            "execution",   // Same timestamp throughout run
        Nondeterministic: true,          // Changes between executions
    }
}

// Example: @uuid - nondeterministic, cache per execution
func (u *UUIDDecorator) getCacheStrategy() CacheStrategy {
    return CacheStrategy{
        Scope:            "execution",   // Same UUID throughout run
        Nondeterministic: true,          // Different UUID per execution
    }
}
```

### Per-Execution Caching

For nondeterministic values like `@timestamp()` and `@uuid()`, use **per-execution** caching instead of time-based TTL:

```go
type ExecutionContext struct {
    ExecutionID string    // Unique ID for this run
    Seed        int64     // Deterministic seed for @random()
    StartTime   time.Time // Execution start time
}

// Cache key combines decorator key with execution ID
func (d *TimestampDecorator) getCacheKey(ctx *Ctx, args []DecoratorParam) string {
    baseKey := d.getBaseKey(args)
    return fmt.Sprintf("%s:%s", baseKey, ctx.ExecutionID)
}
```

**Benefits**:
- **Consistent within execution**: `@timestamp()` returns same value throughout run
- **Fresh between executions**: Each run gets new timestamp/UUID
- **Deterministic randomness**: `@random()` can use execution seed
- **Plan reproducibility**: Resolved plans capture exact execution values

### Execution Safety Rails

**Plan Execution Constraints** (`devcmd exec plan.json`):
1. **Never call nondeterministic resolvers** - use only pinned values from plan
2. **Validate environment fingerprint** before execution  
3. **Fail fast** on missing or corrupted plan values
4. **No runtime resolution** - execution is purely deterministic

**CI/CD Best Practices**:
- Use `--dry-run --resolve` to generate lock files in CI
- Store plan files as deployment artifacts  
- Execute with `devcmd exec plan.json` for reproducible deployments
- Validate plan integrity before production deployment

**Example Safety Check**:
```go
func (d *TimestampDecorator) Render(ctx *Ctx, args []DecoratorParam) (string, error) {
    if ctx.IsExecutingPlan && d.isNondeterministic() {
        return "", fmt.Errorf("@timestamp cannot be resolved during plan execution - value should be pinned")
    }
    // Normal resolution logic...
}
```

## Benefits of This Architecture

- ✅ **Development efficiency**: Quick plans provide fast feedback
- ✅ **Production reliability**: Resolved plans ensure deterministic execution
- ✅ **Plan-execution consistency**: Frozen context guarantees matching behavior
- ✅ **Offline execution**: Plan files work without external dependencies
- ✅ **Decorator autonomy**: Each decorator chooses appropriate resolution strategy
- ✅ **Performance optimization**: Smart caching prevents redundant resolution

## String Processing and Interpolation

### String Processing Strategy

The lexer uses an optimized two-path approach for processing quoted strings:

**1. Simple Strings (No Decorators)**:
```devcmd
build: echo "Building project..."  # ← Uses fast STRING token
test: echo 'Running tests'         # ← Uses literal parsing
```

**2. Interpolated Strings (Contains Decorators)**:
```devcmd
deploy: echo "Deploying @var(APP) version @var(VERSION)"  # ← Uses STRING_START/STRING_TEXT/STRING_END sequence
status: echo "Service @env(NAME) on @env(HOST):@env(PORT)" # ← Complex parsing for decorator expansion
```

**Implementation**: The lexer pre-scans strings with `stringContainsDecorators()` to detect `@decorator` patterns and chooses the appropriate parsing strategy.

### String Interpolation Mode Transitions

**Critical Fix**: String interpolation with decorators requires careful lexer mode management to handle transitions properly.

**Working Pattern**:
```devcmd
setup: {
    @log("Setting up @var(PROJECT) development...")  # String interpolation
    @parallel {                                      # Block decorator works correctly
        echo "task1"
        echo "task2"
    }
}
```

**Mode Transition Flow**:
```
LanguageMode → STRING_START → LanguageMode (for @var) → STRING_END → LanguageMode (for @parallel)
```

The lexer maintains proper mode context through the `inStringDecorator` and `preInterpolatedMode` state tracking to ensure block decorators parse correctly after string interpolation.

### Test Coverage

String interpolation is validated with systematic tests covering:
- `@var()` and `@env()` in strings followed by block decorators
- Multiple decorators within single strings  
- Mixed decorator types after string interpolation
- Complete token sequence validation (not just parse success)

### Comprehensive Testing

See `testing-strategy.md` for the complete approach including:
- **Unit tests** for each decorator in isolation
- **Integration tests** for decorator composition
- **String interpolation tests** for lexer mode transitions
- **Semantic equivalence tests** between all modes
- **Performance benchmarks** for interpreter vs generated
- **Fuzz testing** for parameter validation
- **Cross-platform tests** for shell operator behavior
- **Graph validation** for plan generation