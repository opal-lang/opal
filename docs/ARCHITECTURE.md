# Devcmd Architecture: Unified Decorator System

**Version**: Unified Architecture Implementation

## Core Principle

**Everything is a decorator.**

There are no special cases, no different parsing modes, and no complex execution paths. All shell syntax becomes `@shell` decorators during parsing, creating a beautifully unified system where every construct uses the same execution model.

## Architecture Overview

```
User Input (Natural Syntax)
         ↓
    Lexer/Parser (Converts shell syntax to @shell decorators)
         ↓
    AST (Single Decorator type)
         ↓
    IR (Single DecoratorNode type)
         ↓
    Registry Lookup (ValueDecorator | ExecutionDecorator)
         ↓
    Unified Execution (Plan/Execute methods)
```

## The Two-Interface System

### 1. Value Decorators
**Purpose**: Inject values at specific locations within shell commands

```go
type ValueDecorator interface {
    DecoratorBase
    Plan(ctx Context, args []Param) plan.ExecutionStep
    Resolve(ctx Context, args []Param) (string, error)
    IsExpensive() bool  // For optimization strategies
}
```

**Parameter Handling**: All decorator parameters (including blocks) passed via `[]Param` with types:
- `ArgTypeString`, `ArgTypeNumber`, `ArgTypeDuration`, `ArgTypeBool` - Basic types
- `ArgTypeBlockFunction` - Shell command blocks `{ commands }`
- `ArgTypePatternBlockFunction` - Pattern blocks `{ branch: commands }`

**Examples**:
- `@var(NAME)` - Variable substitution (non-expensive)
- `@env(API_URL, default="localhost")` - Environment variable with default (non-expensive)  
- `@aws_secret(key)` - AWS lookup (expensive)
- `@http_get(url)` - HTTP request (expensive)

**Value Decorator Lifecycle** (Implementation in Progress):
- **Script startup**: Non-expensive value decorators resolved once and cached immutably
- **Plan mode**: Non-expensive values shown resolved, expensive values show placeholders  
- **Execution**: Expensive decorators resolved lazily only when used
- **Caching**: Values remain immutable for entire script execution

### 2. Execution Decorators
**Purpose**: Execute commands or wrap command sequences with enhanced behavior

```go
type ExecutionDecorator interface {
    DecoratorBase
    Plan(ctx Context, args []Param) plan.ExecutionStep
    Execute(ctx Context, args []Param) CommandResult
    RequiresBlock() BlockRequirement  // Describes structural needs
}
```

**Block Requirements** (Conceptual - determines parsing behavior):
- `BlockNone` - No block needed: `@cmd(build)`
- `BlockShell` - Shell command block: `@retry(3) { commands }`
- `BlockPattern` - Pattern matching block: `@when(ENV) { prod: ..., dev: ... }`

**Examples**:
```devcmd
// No block - simple execution
deploy: @cmd(build)

// Shell block - wraps multiple shell commands  
reliable: @retry(3) {
    kubectl apply -f k8s/
    kubectl rollout status
}

// Pattern block - conditional execution
build: @when(ENV) {
    production: docker build -t app:prod .
    development: docker build -t app:dev .
}
```

## Shell Decorator: First-Class Execution

The `@shell` decorator is a standard execution decorator that enables clean syntax:

```go
// @shell is registered as a standard ExecutionDecorator
func (s *ShellDecorator) RequiresBlock() BlockRequirement {
    return BlockRequirement{Type: BlockNone, Required: false}
}

func (s *ShellDecorator) Execute(ctx Context, args []Param) CommandResult {
    command := args[0].GetValue().(string)
    return ctx.ExecShell(command)
}
```

**Parser Conversion Examples**:
```devcmd
// Simple command
build: npm run build
// → @shell("npm run build")

// Command block  
deploy: {
    echo "Starting"    // → @shell("echo \"Starting\"")
    npm run build     // → @shell("npm run build")
    kubectl apply     // → @shell("kubectl apply")
}

// Shell operators split into separate decorators with chaining preserved
complex: echo "hello" && npm test && echo "done"
// → @shell("echo \"hello\"") && @shell("npm test") && @shell("echo \"done\"")
```

## Unified AST Structure

**Single decorator type handles everything**:

```go
type Decorator struct {
    Name   string             // "shell", "retry", "when", "var", etc.
    Args   []NamedParameter   // decorator arguments including blocks
    Pos    Position
    Tokens TokenRange
}
```

**Benefits**:
- Parser has single decorator parsing path
- LSP tools work uniformly across all decorators  
- AST transformations are consistent
- No special-case handling required

## Unified IR Structure

**Single decorator node type**:

```go
type DecoratorNode struct {
    Name   string            // decorator name
    Args   []decorators.Param // unified parameter system with block support
}
```

**Command sequences preserve shell operator chaining**:
```go
type CommandSeq struct {
    Steps []CommandStep
}

type CommandStep struct {
    Chain []ChainElement  // Decorators connected by shell operators
}

type ChainElement struct {
    Decorator DecoratorNode
    OpNext    ChainOp      // &&, ||, |, >> to next element
}
```

## Registry-Driven Execution

**Single lookup, unified behavior**:

```go
// All decorators register the same way
func init() {
    decorators.RegisterValue(NewVarDecorator())      // Value decorators
    decorators.RegisterExecution(NewRetryDecorator()) // Execution decorators  
    decorators.RegisterExecution(NewShellDecorator()) // Shell is just another execution decorator
}

// Runtime uses unified interface
decorator, exists := registry.GetExecution(name)
if exists {
    return decorator.Plan(ctx, args)
}

valueDecorator, exists := registry.GetValue(name)
if exists {
    return valueDecorator.Plan(ctx, args)
}
```

## Decorator Management and Extensibility

### **Decorator Namespacing Strategy**

To prevent decorator explosion and improve discoverability:

**Built-in decorators** (no namespace):
```devcmd
@retry(3) { ... }    // Core execution control
@var(NAME)           // Core value substitution  
@parallel { ... }    // Core concurrency
```

**Extension decorators** (namespaced):
```devcmd
@aws.secret(key)     // AWS-specific functionality
@k8s.rollout(app)    // Kubernetes-specific operations
@http.get(url)       // HTTP-specific value fetching
@git.branch()        // Git-specific value resolution
```

**Custom project decorators** (project namespace):
```devcmd
@myapp.deploy(env)   // Project-specific deployment logic
@myapp.scale(count)  // Project-specific scaling
```

### **Registry Organization**

**Hierarchical decorator discovery**:
1. **Built-in core** - Always available (retry, parallel, var, env)
2. **Standard extensions** - Opt-in packages (aws.*, k8s.*, http.*)  
3. **Project-specific** - Local .devcmd/decorators/ directory
4. **Custom plugins** - External Go modules

**Conflict resolution**:
- Namespaced decorators always win: `@aws.secret` vs `@secret`
- Built-ins cannot be overridden for safety
- Project decorators can shadow extensions (with warning)

### **Decorator Discovery and Documentation**

**Built-in help system**:
```bash
devcmd decorators                    # List all available decorators
devcmd decorators --namespace aws    # List AWS-specific decorators  
devcmd help @retry                   # Show specific decorator help
devcmd help @aws.secret             # Show namespaced decorator help
```

**Self-documenting decorators**:
```go
type RetryDecorator struct{}

func (r *RetryDecorator) Description() string {
    return "Retry command execution with configurable attempts and delay"
}

func (r *RetryDecorator) Examples() []string {
    return []string{
        "@retry(3) { kubectl apply -f k8s/ }",
        "@retry(5, delay=2s) { npm test }",
    }
}
```

## Shell Operator Semantics

Shell operators create decorator chains while preserving standard shell precedence:

```devcmd
// User input
deploy: npm run build && kubectl apply -f k8s/ || echo "Failed"

// Internal representation  
deploy: @shell("npm run build") && @shell("kubectl apply -f k8s/") || @shell("echo \"Failed\"")

// Execution behavior (left-to-right evaluation)
// 1. Execute @shell("npm run build")
// 2. If successful (&&), execute @shell("kubectl apply -f k8s/")  
// 3. If previous failed (||), execute @shell("echo \"Failed\"")
```

**Operator Types**:
- `&&` - Execute next only if previous succeeded
- `||` - Execute next only if previous failed
- `|` - Pipe stdout of previous to stdin of next
- `;` - Execute next unconditionally (shell behavior - continue on failure)
- `>>` - Append stdout of previous to file
- **Newline** - Execute next only if previous succeeded (fail-fast behavior)

## Error Handling and Operator Semantics

### **Error Propagation Rules**

#### **Semicolon vs Newline Execution Models**

**Semicolon (`;`) - Shell Behavior**:
```devcmd
@retry(3) { cmd1; cmd2; cmd3 }
// Internal: @retry(3) { @shell("cmd1; cmd2; cmd3") }
// Behavior: Traditional shell execution, all commands run
// Success: If overall shell sequence exits successfully
// Failure: If overall shell sequence fails
```

**Newline - Fail-Fast Behavior**:
```devcmd
@retry(3) {
    cmd1
    cmd2  
    cmd3
}
// Internal: @retry(3) { @shell("cmd1"); @shell("cmd2"); @shell("cmd3") }
// Behavior: Structured execution with immediate failure detection
// Success: Only if ALL commands succeed in sequence
// Failure: Immediately when ANY command fails
```

#### **Shell Operator Chains**

**Standard shell operators** follow traditional semantics:
```devcmd
// Success/failure propagation
cmd1 && cmd2 && cmd3    // Stop on first failure, success if all succeed
cmd1 || cmd2 || cmd3    // Stop on first success, fail if all fail

// Mixed operators (standard shell precedence)
cmd1 && cmd2 || cmd3    // ((cmd1 && cmd2) || cmd3)
```

**Decorator execution model** - Decorators complete entirely before chain evaluation:
```devcmd
// Decorator-first execution
@retry(3) {
    kubectl apply -f k8s/
    kubectl rollout status
} && echo "Deploy success" || echo "Deploy failed"

// Execution flow:
// 1. @retry executes its ENTIRE block (both kubectl commands, with retries)
// 2. Only after @retry fully completes does && evaluation happen  
// 3. echo executes based on @retry's final result (success/failure)

// Nested decorator completion
@timeout(5m) {
    @retry(3) {
        kubectl apply -f k8s/
        kubectl rollout status
    }
    kubectl get pods
} && echo "All operations completed"

// Execution flow:
// 1. @timeout starts its block
// 2. @retry completes entirely (all retries if needed)
// 3. kubectl get pods executes
// 4. @timeout completes (success if within 5m)
// 5. echo executes based on @timeout's final result
```

### **Debugging Decorator Chains**

**Plan mode shows execution flow**:
```bash
devcmd deploy --dry-run --verbose
```

**Error tracing strategies**:
1. **Plan first**: Always use `--dry-run` to understand execution flow
2. **Incremental testing**: Test decorator components individually  
3. **Verbose output**: Use `--verbose` for detailed execution logs
4. **Plan comparison**: Save working plans and diff against failing ones

**Common debugging patterns**:
```devcmd
// Debug individual decorators
test-retry: @retry(3) { echo "Testing retry logic" }
test-timeout: @timeout(5s) { sleep 2 && echo "Success" }

// Debug operator chains  
test-chain: echo "step1" && echo "step2" && echo "step3"

// Debug nested composition
test-nested: @timeout(10s) {
    @retry(3) {
        echo "Testing nested decorators"
    }
}
```

## Parallelism and Concurrency Model

### **@parallel Decorator Semantics**

The `@parallel` decorator executes all commands in its block concurrently:

```devcmd
// Parallel execution
services: @parallel {
    npm run api      // Starts immediately  
    npm run worker   // Starts immediately
    npm run ui       // Starts immediately
}
// @parallel completes only when ALL three services complete
```

**Execution guarantees**:
- All commands start simultaneously
- Decorator succeeds only if ALL commands succeed
- Decorator fails if ANY command fails
- Output is aggregated and presented coherently

### **Output Aggregation Strategy**

**Live output mode** (default):
```
[api]    Starting API server on port 3000
[worker] Starting background worker
[ui]     Starting UI development server
[api]    API server ready
[worker] Worker connected to queue
[ui]     UI ready at http://localhost:3001
```

**Quiet output mode** (`--quiet`):
- No live output during execution
- Final results only after all complete
- Aggregated success/failure summary

### **Failure Handling in Parallel Blocks**

**Fail-fast strategy**:
```devcmd
// If any service fails, immediately terminate others
deploy: @parallel {
    kubectl apply -f api/
    kubectl apply -f worker/  
    kubectl apply -f ui/
}
// Fails immediately if any kubectl command fails
```

**Mixed parallel/sequential patterns**:
```devcmd
// Sequential setup, then parallel execution
setup-and-run: {
    echo "Setting up infrastructure"
    kubectl create namespace myapp
    
    @parallel {
        kubectl apply -f api/
        kubectl apply -f worker/
        kubectl apply -f ui/
    }
    
    echo "All services deployed"
}
```

## Telemetry and Observability

### **Telemetry Flag (`--telemetry`)**

The `--telemetry` flag captures comprehensive execution data for analysis, optimization, and debugging:

```bash
# Enable telemetry for any execution
devcmd deploy --telemetry
devcmd script.cli --telemetry  
devcmd deploy --dry-run --resolve --telemetry  # Even for planning
```

### **Telemetry Data Captured**

**Execution flow tracking**:
- Decorator entry/exit times with microsecond precision
- Parameter resolution timing (expensive vs non-expensive decorators)
- Shell command execution duration and resource usage
- Operator chain evaluation paths and decision points

**Resource utilization**:
- CPU, memory, disk I/O per decorator
- Network calls and data transfer
- File system operations and access patterns
- Process spawning and lifecycle management

**Error and retry patterns**:
- Failure points and error propagation paths
- Retry attempt timing and success rates
- Timeout occurrences and performance bottlenecks
- Recovery strategies and their effectiveness

### **Telemetry Output Formats**

**JSON structured logs** (machine-readable):
```json
{
  "session_id": "deploy-2024-01-15-14:30:22",
  "command": "deploy",
  "total_duration_ms": 45620,
  "decorators": [
    {
      "name": "timeout",
      "args": {"duration": "5m"},
      "start_time": "2024-01-15T14:30:22.123Z",
      "end_time": "2024-01-15T14:31:07.845Z",
      "duration_ms": 45722,
      "status": "success",
      "children": [...]
    }
  ],
  "resource_usage": {
    "peak_memory_mb": 128,
    "cpu_time_ms": 2340,
    "network_bytes": 524288
  }
}
```

**Visual timeline** (human-readable):
```
deploy: 45.6s total
├─ @timeout(5m): 45.7s
│  ├─ @shell("npm run build"): 12.3s [CPU: 450ms, Mem: 85MB]
│  ├─ @retry(3): 31.2s (2 attempts)
│  │  ├─ @shell("kubectl apply"): 15.8s [Network: 2.1MB]
│  │  └─ @shell("kubectl rollout"): 15.4s [Network: 512KB]
│  └─ @shell("kubectl get pods"): 2.2s [Network: 64KB]
```

### **Telemetry Analysis and Optimization**

**Performance insights**:
- Identify slow decorators and bottlenecks
- Track expensive decorator resolution patterns
- Analyze parallel execution efficiency
- Monitor resource usage trends over time

**Reliability insights**:
- Retry success rates and optimal retry counts
- Timeout frequency and duration tuning
- Error pattern analysis and prevention
- Decorator composition effectiveness

**CI/CD optimization**:
```bash
# Collect telemetry across multiple runs
devcmd deploy --telemetry --output deploy-telemetry.json

# Analyze performance trends
devcmd analyze-telemetry deploy-telemetry-*.json --summary
devcmd analyze-telemetry deploy-telemetry-*.json --slowest-decorators
devcmd analyze-telemetry deploy-telemetry-*.json --retry-patterns
```

### **Privacy and Security**

**Data sanitization**:
- Automatic redaction of sensitive parameter values
- Optional command output filtering
- Configurable data retention policies
- Local-only storage by default

**Telemetry controls**:
```bash
# Fine-grained telemetry control
devcmd deploy --telemetry=timing,resources     # Exclude command output
devcmd deploy --telemetry=full --redact-secrets  # Full data, sanitized
devcmd deploy --telemetry=off                  # Explicit disable
```

This telemetry system provides much deeper insights than dependency graphs alone - it captures the actual runtime behavior of decorator compositions and enables data-driven optimization of devcmd workflows.

## Plan System Architecture

The plan system provides dry-run visualization using the same unified decorator interfaces.

### Core Plan Concepts

**ExecutionPlan**: Complete execution tree with steps, context, and summary
**ExecutionStep**: Individual nodes in execution tree with type, description, command, children, and metadata

### Unified Plan Generation

All decorators implement the same Plan interface:
```go
// Both ValueDecorator and ExecutionDecorator use same signature
Plan(ctx Context, args []Param) plan.ExecutionStep
```

**Registry-Driven Planning**:
```go
// All decorators use the same planning interface
decorator, exists := registry.GetExecution(name)
if exists {
    return decorator.Plan(ctx, args)
}

valueDecorator, exists := registry.GetValue(name)
if exists {
    return valueDecorator.Plan(ctx, args)
}
```

### Plan Display Examples

#### Value Decorators with Superscript Cross-References
```devcmd
# User input
server: node app.js --port @var(PORT) --env @env("NODE_ENV")

# Plan display
server:
└─ @shell("node app.js --port ¹3000 --env ²development")

   Resolved Values:
   1. @var(PORT) → 3000
   2. @env("NODE_ENV") → development
```

#### Block Decorators with Lambda-Style Composition
```devcmd
# User input  
deploy: @retry(3) {
    kubectl apply -f k8s/
    kubectl rollout status
}

# Plan output
deploy:
└─ @retry(attempts=3, delay=1s)
   ├─ @shell("kubectl apply -f k8s/")
   └─ @shell("kubectl rollout status")
```

#### Complex Nested Composition
```devcmd
# User input
deploy: @timeout(5m) {
    echo "Starting"
    @retry(3) {
        kubectl apply -f k8s/
        kubectl rollout status
    }
    echo "Complete"
}

# Plan output
deploy:
└─ @timeout(duration=5m)
   ├─ @shell("echo \"Starting\"")
   ├─ @retry(attempts=3)
   │  ├─ @shell("kubectl apply -f k8s/")
   │  └─ @shell("kubectl rollout status")
   └─ @shell("echo \"Complete\"")
```

### Plan Metadata System

All decorators provide consistent metadata:
```go
plan.ExecutionStep{
    Type:        plan.StepDecorator,
    Description: "@retry(attempts=3)",
    Command:     "retry with 3 attempts",
    Children:    [...],
    Metadata: map[string]string{
        "decorator":      "retry",
        "execution_mode": "error_handling",
        "color":          plan.ColorCyan,
        "info":           "{attempts: 3, delay: 1s}",
    },
}
```

### Value Resolution Strategy

#### **Quick Plans (Default `--dry-run`)**
Fast preview with cached values and deferred expensive operations:

**Non-expensive decorators** (like `@var`, `@env`):
- Resolved once at script startup (planned feature)
- Cached immutably for entire execution  
- Shown resolved in plan mode

**Expensive decorators** (like `@aws_secret`, `@http_get`):
- Resolved lazily only when used during execution (planned feature)
- Show placeholders in plan mode
- Marked clearly as deferred

```devcmd
# User input with expensive decorator
deploy: kubectl apply -f @aws_secret("k8s-config") --namespace @var(NAMESPACE)

# Quick plan display (expensive decorators deferred)
deploy:
└─ @shell("kubectl apply -f ¹@aws_secret(\"k8s-config\") --namespace ²production")

   Resolved Values:
   1. @aws_secret("k8s-config") → <deferred: AWS lookup>
   2. @var(NAMESPACE) → production
```

#### **Resolved Plans (`--dry-run --resolve`)**
Complete resolution of ALL decorators with frozen execution context:

```devcmd
# Same input with resolved plan
deploy: kubectl apply -f @aws_secret("k8s-config") --namespace @var(NAMESPACE)

# Resolved plan display (all values resolved)
deploy:
└─ @shell("kubectl apply -f ¹k8s-secret-content.yaml --namespace ²production")

   Resolved Values:
   1. @aws_secret("k8s-config") → k8s-secret-content.yaml
   2. @var(NAMESPACE) → production
```

**Resolved plans enable plan-then-execute workflows:**
- All expensive operations performed at plan time
- Execution becomes purely deterministic shell commands
- Plans can be saved, audited, and executed later
- Perfect for CI/CD pipelines and production deployments

## Execution Modes

### Dual Mode Support

**Command Mode**: Files with named command definitions
```bash
devcmd build                   # Execute 'build' command from commands.cli
devcmd deploy                  # Execute 'deploy' command
```

**Script Mode**: Files with commandless execution
```bash
#!/usr/bin/env devcmd
./deploy-script               # Direct execution via shebang
devcmd deploy-script          # Or via devcmd
```

**Advanced**: Scripts with local commands using `@cmd()` decorator for internal composition

### Interpreter Mode
All decorators execute through the same unified execution pipeline:

```go
// ValueDecorator execution
result := decorator.Resolve(ctx, args)
// Inject resolved value into shell command

// ExecutionDecorator execution  
result := decorator.Execute(ctx, args)
// Execute with enhancement (retry, timeout, etc.)

// Shell decorator execution
result := ctx.ExecShell(command)
// Direct shell execution
```

### Plan Mode (Dry Run)
All decorators generate plans through the same interface:

```go
// Unified plan generation
step := decorator.Plan(ctx, args)
// All decorators produce ExecutionStep for visualization
```

### Resolved Execution Plans
Plans can be **resolved** to create deterministic, executable artifacts with all values frozen:

```bash
# Generate resolved plan (all values resolved at plan time)
devcmd build --dry-run --resolve > build.plan

# Execute the resolved plan directly
devcmd --execute build.plan      # Execute from resolved plan file
```

**Resolved Plan Benefits:**
- **Deterministic execution**: All variable values locked at plan generation time
- **Auditable workflows**: Complete execution context captured
- **CI/CD pipelines**: Generate plans in one stage, execute in another
- **Debugging**: Inspect exact execution before running
- **Rollback capability**: Re-execute previous successful plans

**Resolved Plan Structure:**
```json
{
  "version": "1.0",
  "context": {
    "variables": {"ENV": "production", "TIMEOUT": "5m"},
    "resolved_values": {
      "1": "@var(PORT) → 3000",
      "2": "@env(API_URL) → https://api.prod.com"
    }
  },
  "execution": {
    "steps": [
      {
        "type": "shell",
        "command": "node app.js --port 3000 --url https://api.prod.com",
        "resolved": true
      }
    ]
  }
}
```

This enables **plan-then-execute** workflows for both scripts and commands:
```bash
# Commands
devcmd deploy --dry-run --resolve > deploy-prod.plan
devcmd --execute deploy-prod.plan

# Scripts  
devcmd deploy-script --dry-run --resolve > script.plan
devcmd --execute script.plan
```

## Key Architecture Benefits

### 1. Conceptual Simplicity
- **One mental model**: Everything is a decorator
- **No special cases**: Shell syntax, decorators, all use same execution model
- **Unified tooling**: LSP, formatting, analysis work consistently

### 2. Implementation Simplicity  
- **Single AST node type**: No more 4-way branching in parser
- **Single IR node type**: No more complex distinctions
- **Single execution path**: Registry lookup → interface method call

### 3. Extensibility
- **New decorators**: Register with same interface, no language changes needed
- **Custom behavior**: Implement ValueDecorator or ExecutionDecorator interface
- **Consistent integration**: New decorators work with existing composition patterns

### 4. Performance
- **Unified caching**: Value decorators can be cached consistently (planned)
- **Optimal execution**: Shell decorators enable direct shell execution where appropriate
- **Parallel opportunities**: Block decorators can implement parallel execution strategies

## Lambda-Style Composition

The unified architecture enables powerful functional composition:

```devcmd
// Nested execution decorators with lambda-style blocks
deploy: @timeout(5m) {
    echo "Starting deployment"
    @retry(3) {
        kubectl apply -f k8s/
        kubectl rollout status
    }
    @when(ENV) {
        production: kubectl scale deployment/app --replicas=3
        staging: kubectl scale deployment/app --replicas=1
    }
    echo "Deployment complete"
}

// Internal representation preserves nesting and converts shell syntax
deploy: @timeout(5m) {
    @shell("echo \"Starting deployment\"")
    @retry(3) {
        @shell("kubectl apply -f k8s/")
        @shell("kubectl rollout status")
    }
    @when(ENV) {
        production: @shell("kubectl scale deployment/app --replicas=3")
        staging: @shell("kubectl scale deployment/app --replicas=1")
    }
    @shell("echo \"Deployment complete\"")
}
```

## Migration Path

### Phase 1: Core Unification ✅
- [x] Simplified decorator interfaces (2 instead of 4)
- [x] Unified registry system
- [x] All builtin decorators migrated
- [x] Unified parameter system with block support

### Phase 2: AST/IR Simplification (In Progress)
- [ ] Single Decorator AST node type
- [ ] Single DecoratorNode IR type  
- [ ] Parser conversion of shell syntax to @shell decorators
- [ ] Unified runtime execution path

### Phase 3: Advanced Features
- [ ] Value decorator expense optimization
- [ ] Shell decorator optimization for direct execution
- [ ] Advanced composition patterns
- [ ] Generated mode implementation

### Phase 4: Production Enhancements
- [ ] **Incrementality via decorators**: `@produces`/`@consumes` decorators for dependency tracking
- [ ] **Cross-platform shell normalization**: Enhanced `@shell` with platform-agnostic quoting/paths
- [ ] **Nested complexity tooling**: LSP quick-fixes, plan simplification, composition guidelines
- [ ] **Performance telemetry**: Cache hit rates, dependency analysis, execution optimization

## Design Philosophy

### **Core Principles**

**Everything is a Decorator**: A single, unified abstraction eliminates special cases and creates consistent patterns across the entire system. Shell commands, control flow, value injection - all follow the same decorator model.

**Natural Syntax with Hidden Power**: Users write familiar shell-like syntax while gaining access to sophisticated orchestration capabilities. The complexity is hidden until needed, with decorators providing enhanced functionality on demand.

**Plan-Then-Execute Workflows**: The ability to visualize, audit, and freeze execution plans before running enables confident operations in production environments. Plans become first-class artifacts for review and compliance.

**Composable Enhancement**: New capabilities are added through decorators, not language changes. This keeps the core language stable while enabling unlimited extensibility through the decorator system.

**Dual Mode Flexibility**: Support both command-based task running and script-based execution provides natural growth paths from simple automation to complex orchestration.

**Observable Execution**: Built-in telemetry and plan visualization give deep insights into execution patterns, performance characteristics, and optimization opportunities.

**Incremental Evolution**: The decorator system naturally accommodates enhancements like dependency tracking (`@produces`/`@consumes`), cross-platform normalization (`@shell` improvements), and complexity management (tooling support) without architectural changes.

This architecture transforms devcmd from a complex system with multiple execution paths into an elegantly unified system where everything follows the same decorator-based execution model, while preserving the natural feel of shell syntax through automatic conversion to @shell decorators.