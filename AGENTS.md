# AGENTS.md

**devcmd** - CLI generation tool that creates standalone binaries from declarative command definitions.

## Build/Test Commands
```bash
# Single test: go test ./module/package -run TestName -v
# Module test: cd module && go test ./...
# All tests: go test ./...
# Build CLI: cd cli && go build -o devcmd . 
# Format: go fmt ./...
# Lint: golangci-lint run (if available)
```

## Code Style & Conventions
- **Go 1.25.0** with workspace modules (`core/`, `runtime/`, `cli/`)
- **Imports**: stdlib first, then third-party, then internal modules
- **Error handling**: Use typed error constants from `core/errors` (e.g., `ErrCommandNotFound`)
- **Testing**: Use `github.com/google/go-cmp/cmp` for diffs, `testify` for assertions
  - **CRITICAL**: Always test complete, exact output with `cmp.Diff` - no lazy partial tests with `assert.Contains`
  - Use table-driven tests for multiple scenarios
  - Test the entire output format, not just parts of it
  - **Bug reproduction**: Always reproduce parser/CLI bugs in existing test files before fixing
  - Add failing tests to `runtime/lexer/*_test.go` and `runtime/parser/*_test.go` for parsing issues
- **Types**: Define in `core/types/types.go`, use `TokenType` and `ExpressionType` enums
- **Registry pattern**: All decorators use global registry, no hardcoded implementations
- **Context passing**: Pass context explicitly, avoid global state
- **Naming**: CamelCase for public, camelCase for private, descriptive error names

## Architecture Rules
- **Module deps**: `core/` (foundation) → `runtime/` → `cli/` (top-level)
- **Go workspace**: Project uses `go.work` with modules in `core/`, `runtime/`, `cli/`
  - Always run `go work sync` after dependency changes
  - Use `go work use .` to add new modules to workspace
  - When testing modules individually: `cd module && go test ./...`
  - For cross-module imports: `github.com/aledsdavies/devcmd/core/...`
- **String system**: Use `StringLiteral.Parts[]` not deprecated `.Value`
- **Decorators**: Registry-driven via `decorators.RegisterAction/RegisterValue/etc.`
- **Fresh builds**: Rebuild CLI after any core/runtime changes
- **Phase 1**: Interpreter mode only, no code generation yet

## Commit & PR Writing Style

**Goal**: Write like a human developer, not an AI. Be clear and direct without corporate-speak or excessive enthusiasm.

### Commit Messages
- **Format**: `type: brief description` (conventional commits)
- **Body**: Explain what was done and why, not how amazing it is
- **Tone**: Straightforward, matter-of-fact
- **Avoid**: "This is the big one!", "Revolutionary", "Massive success!", platitudes
- **Structure**: Brief intro paragraph, then 3-5 key points (not extensive lists, not walls of text)

**Good example**:
```
refactor: complete core module separation and interface cleanup

Finished separating the core module so it only contains structural types and interfaces, with all execution logic moved to runtime. This gives us clean architectural boundaries between what things are (core) and how they execute (runtime).

Key changes:
- Moved AST, IR, and transformation logic from runtime to core
- Moved lexer/parser/validation from cli/internal to runtime (better separation)
- Replaced 4000-line test utilities file with focused helpers
- Added unified Context interface for decorators to access system info and variables

The CLI is now just a thin wrapper around runtime, and we have proper module boundaries.
```

### PR Descriptions
- **Summary**: What this accomplishes and why it was needed
- **Key Changes**: 3-5 bullet points of what actually changed
- **Impact**: What this enables, not how revolutionary it is
- **Avoid**: Emoji, excessive formatting, marketing language

## Project Specification Compliance

**CRITICAL**: Always follow the project specification and architecture documented in `docs/SPECIFICATION.md` and `docs/ARCHITECTURE.md`.

### Unified Architecture Principles
- **Everything is a Decorator**: No special cases - shell syntax becomes `@shell` decorators, all constructs use the unified decorator model
- **Two Interface System**: Only `ValueDecorator` and `ExecutionDecorator` - blocks and patterns are parameter types in `[]Param`
- **Decorator Completion Model**: Decorators execute their entire block before chain operators (`&&`, `||`) are evaluated
- **Plan-Then-Execute**: Support both quick plans (`--dry-run`) and resolved plans (`--dry-run --resolve`) with deterministic execution

### Implementation Requirements
- **Unified interfaces**: All decorators use `Plan(ctx Context, args []Param)` signature
- **Parameter system**: Blocks passed as `ArgTypeBlockFunction`/`ArgTypePatternBlockFunction` parameter types
- **Registry pattern**: Decorators register via `decorators.RegisterValue()` and `decorators.RegisterExecution()`
- **Shell conversion**: Parser automatically converts shell syntax to `@shell` decorators internally
- **Error propagation**: Follow decorator-first completion model for operator chains

### Code Changes Must
1. **Read specification first**: Check `docs/SPECIFICATION.md` for user-facing behavior
2. **Follow architecture**: Implement according to `docs/ARCHITECTURE.md` design patterns
3. **Maintain unified model**: No special cases that break "everything is a decorator"
4. **Preserve execution semantics**: Decorator blocks complete before chain evaluation
5. **Support dual mode**: Both command mode and script mode execution

### When Making Changes
- **AST/IR changes**: Must support single Decorator/DecoratorNode types
- **New decorators**: Must use unified interfaces and parameter system
- **Parser changes**: Must convert shell syntax to `@shell` decorators
- **Execution changes**: Must follow decorator completion model
- **Plan changes**: Must support both quick and resolved plan generation

**Never introduce special cases or break the unified decorator architecture.**

This is an early-stage project focused on generating truly standalone CLI tools from declarative definitions.