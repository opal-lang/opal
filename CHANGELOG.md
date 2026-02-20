# Changelog

All notable changes to Opal will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### 2026-02-20
- Changed executor cancellation semantics to stop scheduling new work after cancellation, preserve finalized command status during late cancellation, and document interrupt/cleanup invariants for infrastructure runs

### 2026-02-19
- Changed shell-worker output handling to stream stdout/stderr live with event-driven file watching, while preserving cancellation behavior and scrubber-compatible output routing

### 2026-02-17
- Changed runtime execution to use the plan-native `executor.ExecutePlan` entrypoint and removed the legacy SDK-step executor entrypoint
- Removed the planfmt-to-SDK execution conversion layer (`core/planfmt/sdk.go`) so execution uses planfmt trees directly
- Added executor-owned shell worker reuse keyed by transport with per-command subshell isolation to preserve non-leaking env/workdir semantics

### 2026-02-15
- Fixed planner command-argument emission to fail on unresolved `Type.Member` enum references instead of emitting raw enum keys
- Rejected unsupported struct inheritance and struct methods with explicit parser diagnostics
- Added deterministic validation for duplicate struct declarations/fields and required recursive struct type cycles, while allowing optional self-references (`Node?`)
- Added top-level `enum Name [String] { ... }` declarations, `Type.Member` enum constant references, and strict enum-typed function/struct validation with duplicate-name/value checks
- Fixed struct inheritance error recovery so malformed declarations do not consume subsequent top-level declarations
- Changed local-session cancellation semantics to use OS-specific termination paths (Unix process-group kill, Windows process-tree kill) with context-first early cancellation
- Added `@shell(..., shell="bash|pwsh|cmd")` with `OPAL_SHELL` fallback and deterministic shell resolution

### 2026-02-14
- Added deterministic function-call cycle detection with explicit call-path errors for recursive and mutual `fun` expansion
- Clarified canonical direct function call syntax (`name(...)`) in language docs and removed retired decorator-like call examples
- Added `none` literal support in expression evaluation and function defaults, with strict optional function parameter typing via `Type?`
- Added explicit expression casts (`expr as Type` and `expr as Type?`) with planner-time cast validation and runtime IR support
- Added top-level `struct Name { ... }` declarations and struct-typed function parameter validation
- Removed colon-style type annotation syntax (`name: Type`) in favor of canonical `name Type`
- Reserved `none` and `as` as language keywords (no identifier fallback)

### 2026-02-10
- Changed `fun` parameter contracts to require explicit types at plan-time validation
- Added grouped Go-style function parameter type syntax (`name, alias String`)
- Clarified language boundary docs: plan-time metaprogramming wraps and validates runtime shell execution

### 2026-02-09
- Changed executor internals to a single tree-runner path with transport-scoped session selection and command `TransportID` routing
- Added run-scoped transport session reuse with per-transport env/workdir freeze and guaranteed session cleanup on success, failure, and cancellation
- Added concrete execution semantics for `@retry`, `@timeout`, and `@parallel`, including timeout cancellation propagation and sandboxed parallel branch isolation
- Fixed nested block execution to inherit wrapper session transport identity so commands, secret resolution, and redirects stay on the intended transport boundary
- Changed decorator argument binding to deterministic required-first slot filling so mixed named+positional forms bind by next unfilled parameter (including named-before-positional forms)

### 2026-02-07
- Parser now accepts multi-line decorator parameter lists and multi-line array/object literals in expression contexts
- Fixed deprecated decorator-parameter remapping so parser tracks the replacement parameter key correctly
- Parser now allows newline after `=` before function default values and decorator parameter values

### 2026-02-05
- Changed planner APIs so `planner.Plan`/`planner.PlanWithObservability` are canonical and removed the legacy planner path
- Changed planner resolution hardening around wave execution, including deterministic decorator batch ordering and stricter decorator argument error propagation
- Changed metaprogramming blocks (`if`, `when`, `for`, `fun`) to lexical declaration scoping so block-local declarations do not leak outward
- Removed deprecated planner-era Vault text-resolution pathways and narrowed executor/decorator contracts around DisplayID resolution

### 2026-02-04
- Added extended plan value types (`Float`, `Duration`, `Array`, `Map`) with deterministic canonicalization for contract hashing
- Added `PlanSalt`-driven contract stability so plan hashing can remain stable across non-semantic code movement
- Added planner resolution telemetry for decorator batches and variable declaration/reference activity

### 2026-02-03
- Added transport table and per-command transport IDs to plan format for contract verification
- Enforced session-aware @env resolution for idempotent transports, with boundary checks for non-idempotent blocks

### 2026-02-02
- Added redirect operator support (`>` and `>>`) to the new planner pipeline and emitted execution trees
- Added planfmt binary/canonical round-trip support for `RedirectNode` and `LogicNode`
- Added inline array literal collections in `for` loop headers (for example `for x in ["a", "b"]`)

### 2026-01-31
- Added the three-phase planner pipeline (`BuildIR -> Resolve -> Emit`) with parity coverage against legacy planner behavior

### 2026-01-27
- Added `TryNode` plan/emission support for `try`/`catch`/`finally` execution trees

### 2026-01-13
- Fixed DisplayID scope binding for redeclared variables and clarified planner scope behavior docs

### 2026-01-05
- Changed resolver output from a flat work queue to a pruned execution tree for richer dry-run/control-flow output

### 2026-01-03
- Added wave-based resolver for expression/decorator resolution with blocker-aware branch pruning

### 2025-12-29
- Added IR builder for planner rewrite to construct execution graphs from parser events
- Fixed range-pattern evaluation to correctly handle ranges starting at `0`

### 2025-12-27
- Added planner expression IR and execution graph foundations for the planner rewrite
- Added stateless `Vault.Resolve(text, transport, site)` for explicit-context planner resolution

### 2025-12-01
- Added plan-time `if`/`else if`/`else` branch evaluation with untaken-branch pruning
- Added comparison operator support (`==`, `!=`) in planner conditions
- Parser now recognizes decorator expressions in condition contexts

### 2025-11-29
- Added transport boundary enforcement for decorator isolation (`@env` blocked at boundaries, `@var` can cross)
- Added `TransportSensitive` capability for decorators to opt-in to boundary enforcement
- Planner now calls `EnterTransport()`/`ExitTransport()` when entering transport decorator blocks

### 2025-11-19
- Changed Go module import paths from `github.com/aledsdavies/opal` to `github.com/opal-lang/opal`
- Added block scoping for execution decorators (`@retry`, `@timeout`, `@parallel`)
- Variables declared inside decorator blocks no longer leak to outer scope

### 2025-11-16
- Executor now resolves DisplayIDs to actual values during execution
- Added CommandIR to preserve temporal binding of variables (fixes shadowing bugs)
- Fixed empty planKey authorization bypass (now panics if planKey missing)

### 2025-11-15
- Commands now support multiple variable interpolations (e.g., `echo "@var.A and @var.B"`)
- Implemented three-pass interpolation: raw strings → record refs → replace with DisplayIDs

### 2025-11-13
- DisplayIDs now embedded directly in plan commands (not placeholder references)
- Removed plan.Secrets field; RuntimeValue never leaves Vault
- Added plan.SecretUses with authorization entries (DisplayID + SiteID + Site)
- Changed Vault API from domain-specific methods to generic Push/Pop/ResetCounts

### 2025-11-12
- Fixed variable scrubbing: CLI and planner now share same Vault instance

### 2025-11-11
- Fixed contract verification: verifier now reuses PlanSalt from original contract
- Wired Vault to scrubber in CLI for output scrubbing

### 2025-11-09
- Added scope-aware variable storage to Vault using pathStack as scope trie
- Variables now properly scoped with parent-to-child flow and shadowing support
- Added SecretProvider interface for pluggable secret scrubbing
- Scrubber now detects encoded secrets (hex, base64, percent-encoding, separators)
- Removed legacy RegisterSecret API in favor of provider-based scrubbing

### 2025-11-08
- Added site-based secret authority (SecretUse tracks authorized use-sites per secret)
- Plan.Freeze() prevents mutations after hash computation for tamper detection
- Plan hash changed from JSON encoding to deterministic binary serialization (BLAKE2b-256)
- Contract verification model documented (salt reuse enables drift detection)
- Shell commands now support decorator interpolation (e.g., `echo @var.HOME`)
- Parser now emits NodeDecorator events for variable references in command strings
- Planner now creates placeholders for decorator values in shell command arguments

### 2025-11-03
- Added object and array literal support with compile-time validation
- Added schema validation for decorator parameters (int range, enum, object, array)
- Added structured error codes for programmatic error handling (enables future tooling)

### 2025-11-01
- Added OEP-017: Shell Command Type Definitions (type-safe shell commands, type packs, stream types)
- Updated OEP-012: Module Composition with JSON Schema Draft 2020-12 and `x-opal-*` extensions

### 2025-10-31
- Aligned documentation after streaming I/O implementation

### 2025-10-30
- Implemented streaming I/O architecture with `io.Reader`/`io.Writer` interfaces

### 2025-10-29
- Implemented decorator architecture with Session abstraction (local and SSH execution)

### 2025-10-24
- Added context propagation for cancellation support
- Added output redirection operators (`>` and `>>`)

### 2025-10-22
- Fixed uint16 overflow in plan writer
- Fixed file transfers to honor context cancellation
- Fixed exec error classification (permission vs not found)
- Fixed infinite recursion in semicolon parsing
- Added OEP-014: Drift Review and OEP-015: Bidirectional Drift Reconciliation

### 2025-10-21
- Added Transport abstraction for remote execution
- Added `@env` decorator with validation
- Implemented pipe operator (`|`) with streaming I/O
- Added OEP proposals system (OEP-001 through OEP-013)

### 2025-10-20
- Added tree-based execution model
- Added script mode support (execute `.opl` files directly)

### 2025-10-19
- Added shell operators (`>`, `>>`, `|`, `&&`, `||`, `;`)
- Added pipeline telemetry and benchmarks
- Added streaming secret scrubber for output sanitization

### 2025-10-18
- Moved tree display to formatter package

### 2025-10-14
- Added complete MVP: planner, executor, and all four execution modes
- Clarified steps, decorators, and operators in architecture

### 2025-10-13
- Refactored plan format to support steps and operators
- Added step boundary events to parser

### 2025-10-12
- Added SDK execution model and `@shell` decorator registry
- Added contract diff display and error formatting

### 2025-10-11
- Added binary plan format

### 2025-10-10
- Added event-based parser

### 2025-09-30
- Changed syntax from function-call style `@var(NAME)` to dot notation `@var.NAME`
- Added optional parameters syntax: `@env.PORT(default=3000)`
- Added explicit rule: no `fun` inside `for` loops

### 2025-09-28
- Rebranded project from `devcmd` to `Opal`
- Changed file extension from `.cli` to `.opl`

### 2025-09-27
- Removed 27k+ lines of outdated Go code for fresh implementation
- Implemented high-performance V2 lexer with TDD approach

### 2025-09-20
- Streamlined specification and implemented generic decorator interfaces

### 2025-09-19
- Added comprehensive architecture documentation

### 2025-09-18
- Separated core module and cleaned up interfaces

### 2025-09-17
- Fixed CLI execution modes

### 2025-09-14
- Removed binary generation system, focused on interpreter mode
- Upgraded to Go 1.25.0

### 2025-09-10
- Added structured log format with improved plan display
- Refactored to IR architecture (AST→IR→Plan system)

---

## Format Guide

This changelog tracks **semantic changes only** - changes that affect language syntax, plan format, execution semantics, or decorator behavior.

**Include:**
- Language syntax changes (e.g., `@var(NAME)` → `@var.NAME`)
- New decorators or operators
- Plan format changes
- Execution model changes
- Breaking changes
- New OEPs or major documentation updates

**Exclude:**
- Internal refactors that don't change behavior
- Build system fixes (Nix hashes, CI tweaks)
- Test additions (unless they reveal new functionality)
- Documentation typos or clarifications
- Dependency updates

**Format:**
- Date headers: `### YYYY-MM-DD`
- One line per change, starting with verb (Added, Fixed, Changed, Removed)
- Be specific but brief
- Group related changes together when it makes sense
