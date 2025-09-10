# Devcmd Engine Architecture - IR-Based Unified Execution

## Overview

The devcmd engine uses an Intermediate Representation (IR) approach to achieve semantic equivalence between **interpreter mode** and **generated mode**, while providing identical **plan mode** functionality in both.

**Core Principle**: Same decorator implementations, same IR evaluation logic, same plan output - whether running `devcmd run build` or `./mycli build` or `./mycli --dry-run build`.

## Current Implementation Status (Phase 1)

**✅ Completed:**
- AST package moved from `core/ast` to `runtime/ast` for better module organization
- DecoratorParam abstraction implemented - decorators no longer depend on AST types
- IR types and NodeEvaluator implemented in `runtime/execution/`
- AST to IR transformation pipeline (basic shell commands working)
- Basic interpreter mode functional for simple shell commands
- Clean testing utilities in `testing/` module (no runtime dependencies)
- TDD test harness in `cli/interpreter_test.go`

**🚧 In Progress:**
- Shell operator parsing (`&&`, `||`, `|`, `>>`) in AST to IR transformation
- Block decorator transformation (`@workdir`, `@timeout`, etc.)
- Action decorator transformation (`@cmd`, `@log`, etc.)
- Pattern decorator transformation (`@when`)

**✅ Recently Completed:**
- Value decorator expansion (`@var`, `@env`) with hybrid caching
- Two-tier plan system (quick plans vs resolved plans)
- Plan generation with value decorator marker system
- Decorator architecture documentation for data resolution strategies

**📋 Next Phase:**
- Complete shell operator edge cases
- Full interpreter mode parity testing
- Generated mode implementation (Phase 2)

## Task Runner UX Requirements

**CRITICAL**: devcmd must provide real-time feedback like modern task runners (make, npm, cargo, etc.).

### Streaming Output Requirements

1. **Real-time stdout streaming**: Users see output as commands execute, not buffered until completion
2. **Capture while streaming**: Stream to user's terminal AND capture for CommandResult
3. **Parallel task visibility**: In `@parallel` blocks, show output from multiple tasks as they complete
4. **Progress indication**: Users can see which tasks are running, completed, or failed

### Implementation Strategy

**Current Issue**: `ExecShell()` in `runtime/execution/ir.go:282` buffers all output until command completion.

**Solution**: 
- Use `io.TeeReader`/`io.MultiWriter` to split output streams
- Stream to user's terminal immediately 
- Capture to buffer for CommandResult
- For `@parallel`: prefix each task's output with task identifier

### Examples of Desired Behavior

```bash
# Sequential execution
$ devcmd run build
[build] Starting Go build...
[build] go build ./cmd/server
[build] ✓ Built server (2.3s)
[build] go build ./cmd/cli  
[build] ✓ Built cli (1.8s)

# Parallel execution with live TUI dashboard
$ devcmd run test-all
┌─ Parallel Execution (3 tasks) ─────────────────────┐
│ [⠋] task-1: go test ./pkg/frontend                 │
│ [⠋] task-2: npm test                               │  
│ [⠋] task-3: python -m pytest                      │
│                                                    │
│ task-1 output (last 5 lines):                     │
│   === RUN   TestUserAuth                          │
│   === RUN   TestValidation                        │
│                                                    │
│ task-2 output (last 5 lines):                     │
│   > jest --coverage                               │
│   PASS src/components/App.test.js                 │
│                                                    │
│ task-3 output (last 5 lines):                     │
│   collecting tests...                             │
│   test_auth.py::test_login PASSED                 │
└────────────────────────────────────────────────────┘

# As tasks complete, status updates:
┌─ Parallel Execution (1 running, 2 completed) ─────┐
│ [✓] task-1: Frontend tests passed (2.1s)          │
│ [⠋] task-2: npm test                               │
│ [✓] task-3: Python tests passed (1.8s)            │
│                                                    │
│ task-2 output (last 5 lines):                     │
│   PASS src/services/api.test.js                   │
│   Test Suites: 12 passed, 12 total                │
│   Tests:       89 passed, 89 total                │
└────────────────────────────────────────────────────┘
```

**Parallel UI Behavior with Standardized Flags:**

```bash
# Quiet mode - no TUI dashboard, minimal output
$ devcmd run test-all --quiet
✓ All parallel tasks completed (2.3s)

# Verbose mode - TUI dashboard + full command output
$ devcmd run test-all --verbose  
[Shows full TUI with complete output history, not just last 5 lines]

# CI mode - no interactive TUI, sequential-style output
$ devcmd run test-all --ci
[task-1] go test ./pkg/frontend
[task-1] === RUN   TestUserAuth
[task-2] npm test  
[task-1] ✓ Frontend tests passed (2.1s)
[task-3] python -m pytest
[task-2] ✓ npm tests passed (2.3s)
[task-3] ✓ Python tests passed (1.8s)

# Non-interactive mode - disables live TUI
$ devcmd run test-all --interactive=never
[Same as --ci mode for parallel execution]
```

This provides immediate feedback while maintaining compatibility with CommandResult for programmatic access.

## Module Structure Changes

The project now uses a cleaner module hierarchy:

```
core/     - Foundation (AST moved out, now just types, errors, decorators)
├── runtime/  - AST, decorators, execution engine (depends on core)
├── testing/  - Generic test utilities (depends only on core)
└── cli/      - Main CLI application + tests (depends on core + runtime)
```

**Key Architectural Decisions Made:**
- **AST in runtime/**: Moved from core to runtime since only parsing and execution need it
- **DecoratorParam abstraction**: Decorators use clean parameter interface, not AST types
- **Testing as utility package**: Pure test helpers, no runtime dependencies
- **TDD approach**: Failing tests drive implementation of each decorator type

---

## Environment Determinism

### Freeze-on-First-Init Pattern

Environment is captured once per process (or from a lock file) to ensure deterministic execution:

```go
// Env snapshot captured once per process (or from a lock file)
type EnvSnapshot struct {
    Values      map[string]string // immutable
    Fingerprint string            // sha256 of sorted KEY\x00VAL pairs
}

type EnvOptions struct {
    Manifest  []string          // keys referenced in IR (or nil=all)
    BlockList []string          // drop PWD, OLDPWD, SHLVL, RANDOM, PS*, TERM
    LockPath  string            // optional path to persist/reuse env
}

func NewEnvSnapshot(opts EnvOptions) (*EnvSnapshot, error)

// Context now uses frozen environment
type Ctx struct {
    Env *EnvSnapshot
    Vars map[string]string
    WorkDir string
    DryRun bool
    Debug bool
}

// Ensure child processes see the frozen env
func ExecShell(ctx *Ctx, cmd string) CommandResult {
    c := exec.Command("/bin/sh", "-lc", cmd)
    c.Env = toKeyValList(ctx.Env.Values)  // Apply frozen env
    // ...
}
```

### Determinism Guarantees

- **Process-level**: Environment frozen at startup
- **Run-to-run**: Optional `--env-lock` persists environment
- **Plan output**: Includes `EnvFingerprint` and redacts sensitive keys
- **Child processes**: All receive the same frozen environment

### Environment Lock Format

```json
{
  "version": 1,
  "values": {"PATH": "/usr/bin:/bin", "USER": "dev"},
  "manifest": ["PATH", "USER", "HOME"],
  "fingerprint": "sha256:abc123..."
}
```

On load: validate version + fingerprint. Use `devcmd env update` to refresh when needed.

---

## Execution Flow Architecture

### Unified Pipeline
```
commands.cli → AST → IR → [Interpreter|Generator|Plan]
                      ↓
                   Same decorator contracts
                   Same execution semantics  
                   Same plan representation
```

### Three Execution Paths

1. **Interpreter Mode**: `devcmd run build`
   - AST → IR → Runtime IR evaluation with shared decorators
   
2. **Generated Mode**: `./mycli build` 
   - AST → IR → Go code generation with embedded IR logic + static CLI harness
   
3. **Plan Mode**: `devcmd run build --dry-run` or `./mycli build --dry-run`
   - **Quick Plans**: AST → IR → Plan generation with cached values and runtime placeholders
   - **Resolved Plans**: `--dry-run --resolve` forces complete value resolution with frozen context
   - Uses same decorator `Describe()` methods with context-aware resolution

### Nesting Guarantees

* **Deterministic stack order**: source order = wrapper order (outer first).
* **Context layering**: Each decorator receives the parent's context and decides how to handle it - `@timeout` may set its own independent deadline, `@workdir` changes the working directory for inner decorators, etc. The layering behavior is decorator-specific.
* **Plan output** should show wrapper composition and edges; include golden tests for "wrapper stacking" and "nested wrapper combinations."

---

## Intermediate Representation (IR)

### Core IR Types

```go
// ChainOp represents shell operators
type ChainOp string // "&&" "||" "|" ">>" or ""

// ChainElement represents one element in a command chain
type ChainElement struct {
    Kind   string                  // "action" | "shell"
    Name   string                  // action name (for Kind=="action") 
    Text   string                  // raw shell command (for Kind=="shell")
    Args   []DecoratorParam    // action arguments
    OpNext ChainOp                 // operator to next element
    Target string                  // file target for ">>"
}

// Step represents one logical line (chain of elements)
type Step struct {
    Chain []ChainElement
}

// Seq represents newline-separated steps
type Seq struct {
    Steps []Step
}

// Node is a sum type for IR tree structure
type Node interface{}
// Implementations:
// - Seq                                           // sequence of steps
// - Block{Kind string, Args Args, Inner Node}    // @timeout{}, @parallel{}
// - Pattern{Kind string, Branches map[string]Seq} // @when branches

// Command represents a complete command definition
type Command struct {
    Name        string
    Description string
    Root        Node    // IR tree root
}

// CommandResult is the unified result type across all modes
type CommandResult struct {
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"` 
    ExitCode int    `json:"exit_code"`
}
```

### AST to IR Transformation

```go
// Pipeline: AST → IR
func TransformToIR(ast *ast.Command) *Command {
    // 1. Parse shell content into Steps (newline-separated)
    //    Note: backslash-newline joins into the same step; otherwise newline = new step
    // 2. Build ChainElements for each step (with operators)
    // 3. Identify ActionDecorators vs shell commands
    // 4. Build Block/Pattern wrappers for decorators
    // 5. Create IR tree with proper Node structure
    // Note: Variables (@var, @env) evaluated at render/exec time, not here
}
```

### Value Expansion Scope

`ValueDecorator.Render` runs in:
- **Shell text** before execution  
- **Action args** before `Run` invocation

**@var vs @env expansion**:
- `@var(NAME)` → Resolved from command variables (constant or computed)
- `@env(KEY)` → Always from `ctx.Env` frozen snapshot
- Both expand to strings; **no auto-quoting**
- Provide `shq(string) string` helper for POSIX quoting when needed

```go
// Example expansions:
// @var(BUILD_DIR) + "/app"     → "./build/app" (const folded)
// @env(USER) + "@" + @env(HOST) → "dev@prod-01" (runtime from frozen env)
```

---

## Unified Decorator Architecture

### Parallel Decorator Requirements

**CRITICAL**: `@parallel` requires access to individual steps for true parallel execution.

**Implementation**: `BlockDecorator.WrapCommands()` receives CommandSeq directly, enabling true parallel execution by accessing individual commands.

**Required Enhancement**: Access to `InnerSteps []CommandStep` from IR for step-level parallel execution:

```go
// Current (sequential through single function)
@parallel { 
    sleep 0.3 && echo "task1"  // All commands in one function
    sleep 0.1 && echo "task2"  // Cannot parallelize individual steps
}

// Required (true parallel execution)
@parallel {
    sleep 0.3 && echo "task1"  // ← Step 1: CommandStep  
    sleep 0.1 && echo "task2"  // ← Step 2: CommandStep
    sleep 0.2 && echo "task3"  // ← Step 3: CommandStep
}
// Should execute Steps 1, 2, 3 concurrently with streaming output
```

**Implementation Strategy**:
1. **Phase 1**: Add `BlockSeqDecorator` interface for step-level access
2. **Phase 2**: Update evaluator to detect and use `BlockSeqDecorator` for parallel-capable decorators
3. **Phase 3**: Implement streaming output with task prefixes

### Decorator Contracts

All decorators implement unified interfaces that work across interpreter and generated modes. **Value decorators are evaluated at render/exec time, not during AST→IR.**

```go
// ActionDecorator - Commands that return CommandResult
type ActionDecorator interface {
    Name() string
    Run(ctx *Ctx, args []DecoratorParam) CommandResult           // execution
    Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep // plan representation
}

// BlockDecorator - Wrappers that modify inner execution
type BlockDecorator interface {
    Name() string
    WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult
    Describe(ctx *Ctx, args []DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
}

// ValueDecorator - Inline values for shell substitution  
type ValueDecorator interface {
    Name() string
    Render(ctx *Ctx, args []DecoratorParam) (string, error)
    Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}

// PatternDecorator - Conditional execution
type PatternDecorator interface {
    Name() string
    SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult
    Describe(ctx *Ctx, args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}
```

### Registry System

Both interpreter and generated modes use the same decorator registry (injected for testability):

```go
// Registry passed into interpreter and generated harness
type Registry struct {
    Actions  map[string]ActionDecorator
    Blocks   map[string]BlockDecorator
    Values   map[string]ValueDecorator
    Patterns map[string]PatternDecorator
}

// Registry keying: use fully-qualified names to prevent collisions
// e.g., "acme.timeout" for third-party decorators
// Registration fails on duplicate keys

// Standard registry for production use
func NewStandardRegistry() *Registry {
    r := &Registry{
        Actions:  make(map[string]ActionDecorator),
        Blocks:   make(map[string]BlockDecorator),
        Values:   make(map[string]ValueDecorator),
        Patterns: make(map[string]PatternDecorator),
    }
    
    // Register builtin decorators
    r.RegisterAction(&LogDecorator{})
    r.RegisterBlock(&TimeoutDecorator{})
    r.RegisterValue(&VarDecorator{})
    r.RegisterPattern(&WhenDecorator{})
    
    return r
}
```

---

## Interpreter Mode Implementation

### IR Runtime Evaluation

Build wrapper stack from `Node` (outermost block first), then evaluate `Seq` step-by-step:

```go
// NodeEvaluator executes IR trees using live decorators
type NodeEvaluator struct {
    registry *Registry
}

func (e *NodeEvaluator) EvaluateNode(ctx *Ctx, node Node) CommandResult {
    switch n := node.(type) {
    case *Seq:
        return e.evaluateSeq(ctx, n)
    case *Block:
        // Build wrapper stack and execute inner
        decorator := e.registry.Blocks[n.Kind]
        return decorator.WrapCommands(ctx, n.Args, n.Inner)
    case *Pattern:
        // Select branch and execute
        decorator := e.registry.Patterns[n.Kind]
        return decorator.SelectBranch(ctx, n.Args, n.Branches)
    }
}

func (e *NodeEvaluator) evaluateSeq(ctx *Ctx, seq *Seq) CommandResult {
    var lastResult CommandResult
    for _, step := range seq.Steps {
        lastResult = e.evaluateStep(ctx, step)
        // Stop on failure (like shell set -e)
        if lastResult.Failed() {
            return lastResult
        }
    }
    return lastResult
}

func (e *NodeEvaluator) evaluateStep(ctx *Ctx, step Step) CommandResult {
    // Chain evaluation with proper operator semantics
    var prevResult *CommandResult
    
    for i, element := range step.Chain {
        shouldExecute := e.shouldExecuteElement(element, prevResult)
        if !shouldExecute {
            continue
        }
        
        var result CommandResult
        switch element.Kind {
        case "action":
            decorator := e.registry.Actions[element.Name]
            result = decorator.Run(ctx, element.Args)
        case "shell":
            result = ExecShell(ctx, element.Text)
        }
        
        // Handle >> operator: append stdout to file
        if element.OpNext == ">>" {
            if err := appendToFile(element.Target, result.Stdout); err != nil {
                return CommandResult{Stderr: err.Error(), ExitCode: 1}
            }
            result = CommandResult{ExitCode: 0}
        }
        
        prevResult = &result
        
        if e.shouldTerminateChain(element.OpNext, result) {
            break
        }
    }
    
    if prevResult != nil {
        return *prevResult
    }
    return CommandResult{ExitCode: 0} // empty chain
}
```

### Interpreter Entry Point

```go
// cli/main.go - interpreter mode
func runInterpreter(commandName string, registry *Registry) {
    // 1. Parse commands.cli → AST
    ast := parser.ParseFile("commands.cli")
    
    // 2. Transform AST → IR  
    command := TransformToIR(ast.Commands[commandName])
    
    // 3. Evaluate IR using runtime decorators
    ctx := cli.NewCtx()
    evaluator := &NodeEvaluator{registry: registry}
    result := evaluator.EvaluateNode(ctx, command.Root)
    
    // 4. Handle result
    handleCommandResult(result)
}
```

---

## Generated Mode Implementation  

### Minimal Code Generation

Generated CLIs use a **static harness + minimal codegen** approach:

#### Static CLI Harness (`runtime/cli/harness.go`)
```go
// Static Cobra CLI with injected registry
type CLIHarness struct {
    rootCmd   *cobra.Command
    globalCtx *Ctx
    registry  *Registry
}

func NewCLIHarness(name, version string, reg *Registry) *CLIHarness {
    return &CLIHarness{
        rootCmd:  newRootCmd(name, version),
        globalCtx: NewCtx(),
        registry: reg,
    }
}

func (h *CLIHarness) RegisterCommandsWithPlan(commands []GeneratedCommandWithPlan) {
    for _, cmd := range commands {
        h.addCommandWithPlan(cmd)  // Uses same plan/execution logic
    }
}
```

#### Generated Command Functions (codegen output)

**Phase 1 uses embedded evaluator; Phase 2 removes the evaluator from the binary - chains compile to straight-line `if/else` with wrapper calls; plan compiles to direct `Describe` calls. Both produce identical results and plan output.**

```go
// @var classification: const when provably constant
const (
    BuildDir = "./build"         // @var BUILD_DIR="./build" → Go const
    Port     = 8080             // @var PORT=8080 → Go const  
    Debug    = true             // @var DEBUG=true → Go const
    Timeout  = 5 * time.Minute  // @var TIMEOUT=5m → Go const
)

// Runtime variables for @env or computed @var
func VAR_IMAGE(ctx *Ctx) string {
    // @var IMAGE = "@env(REPO) + ':' + @env(TAG)"
    repo, _ := ctx.Env.Get("REPO")
    tag, _  := ctx.Env.Get("TAG")
    return repo + ":" + tag
}

// Phase 1: Generated command function using embedded IR evaluation
func cmdBuild(ctx *Ctx, reg *Registry) CommandResult {
    // Embedded IR tree (compiled from AST)
    seq := &Seq{
        Steps: []Step{
            {Chain: []ChainElement{
                {Kind: "shell", Text: "go build -o " + BuildDir + "/app ./..."},
            }},
        },
    }
    
    evaluator := &NodeEvaluator{registry: reg}
    return evaluator.EvaluateNode(ctx, seq)
}

// Phase 2 target: Direct Go compilation (future optimization)
// Compiles chains to straight-line Go (if/else + wrapper calls), calling the same decorator APIs
// No evaluator in the binary
func cmdBuildDirect(ctx *Ctx, reg *Registry) CommandResult {
    // Direct shell execution with same semantics
    return ExecShell(ctx, "go build -o " + BuildDir + "/app ./...")
}

// Generated plan function using same decorators
func planBuild(ctx *Ctx) *plan.ExecutionPlan {
    p := plan.NewExecutionPlan()
    p.AddStep(plan.Command("go build -o " + BuildDir + "/app ./...").Build())
    return p
}
```

### @var and @env Classification Rules

**@var → Go const** (compile-time constant):
- Pure literals: `"str"`, `42`, `true`, `5*time.Minute`
- Operations on constants (pre-folded string concat)
- Depends **only** on other constant `@var`s

**@var → Runtime var** (computed at runtime):
- References **any** `@env(...)`
- Calls non-const functions (`path.Join`, `fmt.Sprintf`, regex, etc.)
- Depends on any runtime `@var`

**@env(KEY) → Always runtime**:
- Read from `ctx.Env` (frozen snapshot)
- Never a Go `const` (maintains portability)
- Optional: "baked env-lock" mode inlines as consts (trades portability for speed)

```go
// Codegen examples:

// 1) Constant @var → Go const
// DSL: @var BUILD_DIR="./build", @var TIMEOUT=5m
const (
  BUILD_DIR = "./build"
  TIMEOUT   = 5 * time.Minute
)

// 2) Runtime @var → Helper function
func VAR_IMAGE(ctx *Ctx) string {
  repo, _ := ctx.Env.Get("REPO")
  tag, _  := ctx.Env.Get("TAG")
  return repo + ":" + tag
}

// 3) Usage in commands
cmd := fmt.Sprintf("kubectl apply -f %s/k8s.yaml", BUILD_DIR)
image := VAR_IMAGE(ctx)
```

### Dead Code Elimination

- **@when on constant @var** → Compile-time branch pruning
- **@when on @env or runtime @var** → Keep both branches (runtime decision)

---

## Plan Mode Implementation

### Unified Plan Generation

Both interpreter and generated modes use **identical Node-based plan generation**:

```go
// Node-based plan generation used by both modes
func PlanNode(ctx *Ctx, reg *Registry, n Node) plan.ExecutionStep {
    switch x := n.(type) {
    case *Seq:
        group := plan.Group("sequence")
        for _, s := range x.Steps { 
            group.Add(PlanStep(ctx, reg, s)) 
        }
        return group.Build()
    case *Block:
        inner := PlanNode(ctx, reg, x.Inner)
        dec := reg.Blocks[x.Kind]
        return dec.Describe(ctx, x.Args, inner)
    case *Pattern:
        // Deterministic order
        names := sortedKeys(x.Branches)
        branches := map[string]plan.ExecutionStep{}
        for _, name := range names {
            branches[name] = PlanNode(ctx, reg, x.Branches[name])
        }
        dec := reg.Patterns[x.Kind]
        return dec.Describe(ctx, x.Args, branches)
    default:
        return plan.Text("unknown node").Build()
    }
}

func PlanStep(ctx *Ctx, reg *Registry, step Step) plan.ExecutionStep {
    g := plan.Group("chain")
    for _, el := range step.Chain {
        switch el.Kind {
        case "action":
            g.Add(reg.Actions[el.Name].Describe(ctx, el.Args))
        case "shell":
            g.Add(plan.Command(el.Text).Build())
        }
        // Include operator info in plan
        if el.OpNext != "" {
            g.Add(plan.Text(string(el.OpNext)).Build())
        }
    }
    return g.Build()
}
```

### Plan Mode Entry Points

```bash
# Interpreter plan mode
devcmd run build --dry-run
# ↓ calls generatePlan() with runtime IR

# Generated CLI plan mode  
./mycli --dry-run build
# ↓ calls same generatePlan() with embedded IR
```

**Result**: Identical plan output regardless of execution mode.

---

## Performance Optimizations

### Generated Mode Optimizations

1. **@var Classification Performance**
   ```go
   // Constant @var → Go const (fastest access, zero runtime cost)
   const BuildDir = "./build"
   
   // Runtime @var → Ctx-bound function (evaluated on demand)
   func VAR_IMAGE(ctx *Ctx) string { return ctx.Env.Get("REPO") + ":" + ctx.Env.Get("TAG") }
   
   // @env → Always from frozen ctx.Env (deterministic)
   kubeCtx, _ := ctx.Env.Get("KUBE_CONTEXT")
   ```

2. **Pre-folded Constants**
   ```go
   // Pre-fold constant expressions so Go sees one literal
   const DockerCmd = "docker build -t myapp:v1 ."  // Pre-computed at codegen
   
   // Mix const + runtime efficiently
   cmd := fmt.Sprintf("kubectl --context=%s apply -f %s/k8s.yaml", 
       kubeCtx, BuildDir)  // BuildDir is const, kubeCtx from env
   ```

3. **Embedded IR Evaluation**
   - No runtime parsing
   - Direct function calls instead of AST traversal
   - Compiled operator logic

### Interpreter Mode Optimizations

1. **Cached AST Parsing**
2. **Reused Execution Context**  
3. **Direct Shell Delegation** for simple commands

---

## Shell Operator Semantics

Both modes implement **identical shell operator semantics**:

### AND Operator (`&&`)
```go
// cmd1 && cmd2 - cmd2 runs only if cmd1 succeeds
result1 := executeElement(element1)
if result1.Success() {
    result2 := executeElement(element2)
    return result2
}
return result1
```

### OR Operator (`||`)
```go  
// cmd1 || cmd2 - cmd2 runs only if cmd1 fails
result1 := executeElement(element1)
if result1.Failed() {
    result2 := executeElement(element2)  
    return result2
}
return result1
```

### PIPE Operator (`|`)
```go
// cmd1 | cmd2 - stdout of cmd1 feeds stdin of cmd2 (streaming by default)
func pipeExec(ctx *Ctx, left, right execCmd) CommandResult {
    r, w := io.Pipe()
    defer r.Close()
    left.Stdout = w
    right.Stdin = r
    // Start right first to ensure it's reading
    go run(right)
    resL := run(left); w.Close() // Signal EOF
    resR := wait(right)
    return resR // Exit code from right, stderr merged by harness
}
// Note: Streaming with io.Pipe to avoid buffer surprises
// Plan output captures up to 64KB per step
```

### APPEND Operator (`>>`)
```go
// cmd1 >> file.txt - append previous element's Stdout to Target file
result1 := executeElement(element1)
if err := appendToFile(element.Target, result1.Stdout); err != nil {
    return CommandResult{Stderr: err.Error(), ExitCode: 1}
}
return CommandResult{ExitCode: 0} // success unless append failed
```

### Empty Chain/Step Behavior
```go
// Empty chain or step returns success
if len(chain) == 0 {
    return CommandResult{ExitCode: 0}
}
```

### Step Semantics

- **Step failure rule**: A step "fails" if its final chain result is non-zero; `&&/||` operators inside the step determine that final result; only then does the next step get skipped
- **Steps execute sequentially** - like shell scripts, failure stops execution
- **Newline-separated steps** behave like `set -e` - exit on first error
- **Parallel blocks** execute all steps regardless of individual failures (unless fail-fast is set)
- To ignore failures, use explicit error handling (`|| true` or `|| echo "continuing"`)

### Stdin Handling for Actions

```go
// Optional interface for actions that accept stdin
type StdinAware interface {
    RunWithInput(ctx *Ctx, args []DecoratorParam, stdin io.Reader) CommandResult
}

// Rule: In a pipe, if the right element is an action AND implements StdinAware,
// call RunWithInput; else error "action is not pipe-capable"
```

---

## Wrapper Semantics

### Error Propagation and Composition Order
- **Composition**: Outer → Inner (timeout wraps parallel wraps sequence)
- **Error propagation**: Inner failures bubble through wrapper stack
- **Context cancellation**: Timeouts use context cancellation
- **Retries**: What resets between attempts (env, working dir, etc.)
- **Parallel**: Fail-fast vs fail-after, stdout/stderr merge order defined

### Decorator Behavior

**Key Principle**: Decorators receive raw commands and control execution completely.

#### Block Decorators (receive `CommandSeq`)
- **@timeout**: Applies timeout to `ctx.ExecSequential(commands)`, returns exit code 124 on timeout
- **@parallel**: Calls `ctx.ExecParallel(commands.Steps, mode)` - three modes:
  - `fail-fast` (default): Stop scheduling on first failure, wait running ones, return first error
  - `fail-immediate`: Cancel all running tasks on first failure  
  - `all`: Run all tasks to completion, aggregate all errors
  - Output merge by start time, tagged with step ID
- **@retry**: Loops `ctx.ExecSequential(commands)` with backoff strategies, max attempts
- **@workdir**: Creates `newCtx.WithWorkDir(path)` and calls `newCtx.ExecSequential(commands)`
- **@confirm**: Prompts user, then calls `ctx.ExecSequential(commands)` if confirmed

#### Pattern Decorators (receive `map[string]CommandSeq`)
- **@when**: Evaluates condition, executes matching branch with `ctx.ExecSequential(branches[match])`
- **@try**: Executes `main` branch, then `catch` on failure, always executes `finally`

---

## Plan/Describe Guarantees

### Stability and Determinism
- **Plan output must be stable/deterministic**: Sort maps, normalize durations/paths
- **Secret redaction**: Use `SecretRedactor` mix-in for sensitive values
- **Wrapper composition**: Plan mirrors wrapper composition and chosen branches
- **Condition evaluation**: `@when` shows current values and selected branch with reasons

### Plan Generation Requirements
```go
// Plan must be deterministic and stable
func generatePlan(ctx *Ctx, node Node) *plan.ExecutionPlan {
    // Sort maps for deterministic output (especially @when branch names)
    // Normalize paths (absolute → relative) and durations (ns → human readable)
    // Include top-level "plan_version" and "env_fingerprint" for stability
    // Apply secret redaction via SecretRedactor mix-in
    // Show wrapper composition clearly (outer → inner)
    // Pattern Describe contract: emit selected branch + sorted alternatives with reasons
    // Max 64KB capture per step output in plan
}
```

### Plan JSON Schema Stability

```json
{
  "plan_version": 1,
  "env_fingerprint": "sha256:abc123...",
  "steps": [...],
  "context": {...}
}
```

Guarantees: Sorted maps, stable field ordering, deterministic output for JSONEq tests.

---

## Testing Strategy

### Semantic Equivalence Tests

```go
func TestSemanticEquivalence(t *testing.T) {
    // Same command definition
    commandDef := `build: echo "Building" && go build ./...`
    
    // Execute in interpreter mode
    interpreterResult := executeInterpreter(commandDef)
    
    // Generate and execute CLI
    generatedResult := executeGenerated(commandDef)
    
    // Results must be identical
    assert.Equal(t, interpreterResult, generatedResult)
}

func TestPlanEquivalence(t *testing.T) {
    // Same command definition
    commandDef := `deploy: @timeout(5m) { @log("Deploying...") && kubectl apply }`
    
    // Plan in interpreter mode
    interpreterPlan := generateInterpreterPlan(commandDef)
    
    // Plan in generated mode
    generatedPlan := generateGeneratedPlan(commandDef)
    
    // Plans must be identical
    assert.Equal(t, interpreterPlan.String(), generatedPlan.String())
}
```

### Integration Test Coverage

1. **Variable resolution** - `@var`, `@env` work identically
2. **Decorator behavior** - `@log`, `@timeout`, `@parallel` have same semantics
3. **Shell chaining** - `&&`, `||`, `|` operators behave identically  
4. **Error handling** - Exit codes and error messages match
5. **Plan generation** - `--dry-run` produces identical output

### Golden Tests and Edge Cases

**Required golden tests:**
- **Plan JSON** output (deterministic, stable formatting)
- **Wrapper stacking** order and composition (`@timeout{@parallel{...}}`)
- **@when branch selection** with different variable values
- **>> append operator** behavior and error handling
- **Pipe streaming/buffering** with large outputs
- **Empty chains/steps** returning success (exit code 0)
- **Secret redaction** in plan output

**Edge case coverage:**
- Malformed chains, missing decorators, invalid arguments
- Timeout during parallel execution, context cancellation propagation  
- File append failures, permission errors
- Large pipe buffers, streaming vs buffered behavior
- Nested wrapper combinations, error propagation through stack

---

## Migration Path

### Legacy Template System → IR System

1. **Phase 1**: Implement IR types and transformation pipeline
2. **Phase 2**: Build new interpreter using IR evaluation  
3. **Phase 3**: Build new codegen using static harness + minimal generation
4. **Phase 4**: Migrate decorators to unified contracts
5. **Phase 5**: Remove legacy template system

### Backward Compatibility

- Same `.cli` file syntax
- Same command-line interfaces  
- Same behavior and semantics
- Improved performance and maintainability

---

## Plan Graph Support

### Graph-Based Plan Representation

While keeping the existing tree-based plan structure and pretty printers, the plan system supports first-class **graph representation** with edges and stable IDs for advanced tooling and visualization.

### Core Graph Types

```go
// Edge kinds for control/data flow
type EdgeKind string

const (
  EdgeThen      EdgeKind = "then"       // sequence between sibling steps
  EdgeOnSuccess EdgeKind = "on_success" // &&
  EdgeOnFailure EdgeKind = "on_failure" // ||
  EdgePipe      EdgeKind = "pipe"       // |
  EdgeAppend    EdgeKind = "append"     // >>
)

// Graph edge
type PlanEdge struct {
  FromID string   `json:"from_id"`
  ToID   string   `json:"to_id"`
  Kind   EdgeKind `json:"kind"`
  Label  string   `json:"label,omitempty"` // e.g. "&&", "|", ">>", reason text
}

// Extended ExecutionPlan
type ExecutionPlan struct {
  Steps   []ExecutionStep        `json:"steps"`
  Edges   []PlanEdge             `json:"edges"`   // NEW
  Context map[string]interface{} `json:"context"`
  Summary PlanSummary            `json:"summary"`
}
```

### Stable ID Assignment

```go
// Call once after building tree to assign stable IDs like "0", "0/2", "0/2/1"
func (ep *ExecutionPlan) AssignStableIDs() {
  var walk func([]int, *ExecutionStep)
  walk = func(path []int, s *ExecutionStep) {
    s.ID = pathToID(path) // e.g., join with "/" 
    for i := range s.Children { 
      walk(append(path, i), &s.Children[i]) 
    }
  }
  for i := range ep.Steps { 
    walk([]int{i}, &ep.Steps[i]) 
  }
}
```

### IR to Graph Translation

The IR→Plan translator adds edges based on chain semantics:

```go
plan := plan.NewPlan().
  Add(plan.Sequence().WithChildren(/* children built from a Step */)).
  Build()

plan.Context["command_name"] = cmdName
plan.AssignStableIDs()

// Example: edges for one Step with chain: a && b | c >> logs.txt
aID := findByPath(plan, "0/0") // first child in first top-level step
bID := findByPath(plan, "0/1")
cID := findByPath(plan, "0/2")

plan.AddEdge(PlanEdge{FromID: aID, ToID: bID, Kind: EdgeOnSuccess, Label: "&&"})
plan.AddEdge(PlanEdge{FromID: bID, ToID: cID, Kind: EdgePipe,      Label: "|"})
plan.AddEdge(PlanEdge{FromID: cID, ToID: fileNodeID("logs.txt"), Kind: EdgeAppend, Label: ">>"})

// Connect siblings in sequence:
connectSiblingsThen(plan, "0") // emits EdgeThen between 0/0 -> 0/1 -> 0/2
```

### File Nodes for Append Operations

Represent `>> file.txt` with dedicated file nodes:

```go
func ensureFileNode(ep *ExecutionPlan, name string) string {
  id := "file:" + name
  for i := range ep.Steps {
    if ep.Steps[i].ID == id { return id }
  }
  ep.Steps = append(ep.Steps, ExecutionStep{
    ID: id, Type: StepShell, Description: "file:" + name, 
    Command: "", Metadata: map[string]string{"file": name},
  })
  return id
}
```

### Multiple Output Views

- **Tree view**: Existing `String()` / `StringNoColor()` (default "pretty" plan)
- **Graph exporters** (read `Steps+Edges`, use `ID` as node key):
  - DOT/Graphviz: `func (ep *ExecutionPlan) ToDOT() string`
  - Mermaid: `ToMermaid() string` (flowchart or graph TD)
  - Cytoscape JSON: `ToCyto() []byte` (for web UI)

### Plan/Graph Variable Resolution

**Graph shows resolved values**:
- **Constant @var**: Literal value in node attributes
- **@env**: Value from `ctx.Env` (frozen) + include `env_fingerprint`  
- **Runtime @var**: Computed value from current context

```json
{
  "plan_version": 1,
  "env_fingerprint": "sha256:abc123...",
  "steps": [
    {
      "id": "0/0",
      "command": "kubectl apply -f ./build/k8s.yaml",
      "resolved_vars": {
        "BUILD_DIR": "./build",    // constant @var
        "KUBE_CONTEXT": "prod"     // @env value from frozen snapshot
      }
    }
  ]
}
```

### Determinism Requirements

- Call `AssignStableIDs()` before adding edges for content/location-stable IDs
- Sort map→slice conversions in exporters
- Include frozen env fingerprint in `Context["env_fingerprint"]`
- Variable resolution uses frozen environment for consistent plan output

---

## Key Benefits

### For Users
- **Identical behavior** between development (interpreter) and production (generated)
- **Consistent plan output** regardless of execution mode
- **Same decorator syntax** and semantics everywhere
- **Performance optimizations** without behavior changes

### For Developers  
- **Single decorator implementation** works in all modes
- **Unified testing** - test once, works everywhere
- **Simplified architecture** - no complex template system
- **Performance optimization** through const/var analysis and smart codegen

### For Maintainers
- **Semantic equivalence** is architecturally guaranteed
- **Plan consistency** through shared decorator contracts
- **Reduced complexity** compared to template-based approach
- **Clear separation** between IR logic and execution strategies

This IR-based architecture ensures that devcmd provides a unified, consistent experience whether used interactively during development or deployed as optimized standalone binaries in production.