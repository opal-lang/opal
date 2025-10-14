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

**Operators** (`&&`, `||`, `|`, `;`) control flow **within a single step**. They are part of the shell command string and handled by bash, not Opal.

```opal
// ONE step with operators (bash controls flow within step)
echo "First" && echo "Second" || echo "Fallback"
```

When this executes:
- Opal sees **one step** containing the entire command string
- Bash receives `echo "First" && echo "Second" || echo "Fallback"`
- Bash handles the `&&` and `||` logic internally
- Opal only sees the final exit code

**Operator semantics** (bash-controlled):
- `&&` - Execute next command only if previous succeeded (exit 0)
- `||` - Execute next command only if previous failed (exit non-zero)
- `|` - Pipe stdout of previous command to stdin of next
- `;` - Execute commands sequentially regardless of exit codes

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

**Example 1: Operators (bash controls)**
```opal
// ONE step - bash handles && logic
mkdir -p /tmp/build && cd /tmp/build && npm install
```

If `mkdir` fails, bash stops and never runs `cd` or `npm install`. Opal sees one step that either succeeded or failed.

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
// TWO steps - bash controls within, Opal controls between
mkdir -p /tmp/build && cd /tmp/build
npm install && npm run build
```

Step 1: `mkdir -p /tmp/build && cd /tmp/build` (bash handles `&&`)
Step 2: `npm install && npm run build` (bash handles `&&`)

Opal controls whether step 2 runs based on step 1's exit code.

### Why This Matters

**For plan generation**:
- Operators are part of the command string (opaque to planner)
- Newlines create distinct steps in the plan
- Each step gets a unique ID and can be tracked independently

**For execution**:
- Operators are bash's responsibility (fast, no Opal overhead)
- Newlines are Opal's responsibility (logging, telemetry, error handling)
- Decorators wrap steps and control execution behavior

**For contract verification**:
- Operators are part of the command string hash
- Steps are the unit of comparison (step count, step order, step content)
- Changing operators changes the command hash, failing verification

### Design Principle: Separation of Concerns

**Bash is good at**: Piping, chaining, conditional execution within a command
**Opal is good at**: Orchestration, error handling, parallelism, contract verification

By separating intra-step (operators) from inter-step (newlines), Opal leverages bash's strengths while adding orchestration capabilities bash lacks.

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
    Placeholder ValuePlaceholder    // <length:algo:hash> for display/hashing
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

All resolved values use security placeholder format: `<length:algorithm:hash>`

Examples:
- `<1:sha256:abc123>` - single character (e.g., "3")
- `<32:sha256:def456>` - 32 characters (e.g., secret token)
- `<8:sha256:xyz789>` - 8 characters (e.g., hostname)

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
    "var.REPLICAS": "<1:sha256:abc123>",
    "env.HOME": "<21:sha256:def456>",
    "aws.secret.api_key": "<32:sha256:xyz789>"
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
├─ kubectl create secret --token=<32:sha256:a1b2c3>
└─ kubectl scale --replicas=<1:sha256:def789> deployment/app

Values:
  var.REPLICAS = <1:sha256:def789>
  env.HOME = <21:sha256:abc123>

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

Expected: kubectl scale --replicas=<1:sha256:abc123> deployment/app
Actual:   kubectl scale --replicas=<1:sha256:def456> deployment/app

Value changed:
  var.REPLICAS
    Contract: <1:sha256:abc123...> (was "3")
    Current:  <1:sha256:def456...> (now "5")

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
kubectl create secret --token=¹<32:sha256:a1b2c3>
Resolved: 1. @aws.secret.api_token → <32:sha256:a1b2c3>
```

Smart optimizations happen automatically:
- Expensive value decorators in unused conditional branches never execute
- Independent expensive operations resolve in parallel  
- Dead code elimination prevents unnecessary side effects

## Security Model

The placeholder system protects sensitive values while enabling change detection:

**Placeholder format**: `<length:algorithm:hash>` like `<32:sha256:a1b2c3>`. The length gives size hints for debugging, the algorithm future-proofs against changes, and the hash detects value changes without exposing content.

**Security invariant**: Raw secrets never appear in plans, logs, or error messages. This applies to all value decorators - `@env.NAME`, `@aws.secret.NAME`, whatever. Compliance teams can review plans confidently.

**Hash scope**: Plan hashes cover ordered steps, arguments, operator graphs, and timing flags. They exclude ephemeral data like run IDs or timestamps that shouldn't invalidate a plan.

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
  "steps": [...]
}
```

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

## Seeded Determinism

For operations requiring randomness or cryptography, opal will use seeded determinism to maintain contract verification while enabling secure random generation.

### Plan Seed Envelope (PSE)

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
kubectl create secret generic db --from-literal=password=¹<24:sha256:abcd>
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