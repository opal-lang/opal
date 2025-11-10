# Changelog

All notable changes to Opal will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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
