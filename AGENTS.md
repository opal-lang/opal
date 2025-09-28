# AGENTS.md

**opal** - CLI generation tool that creates standalone binaries from declarative command definitions.

## Build/Test Commands
```bash
# Single test: go test ./module/package -run TestName -v
# Module test: cd module && go test ./...
# All tests: go test ./...
# Build CLI: cd cli && go build -o opal .
# Format: go fmt ./...
# Lint: golangci-lint run (if available)
```

## Code Style & Conventions
- **Go 1.25.0** with workspace modules (`core/`, `runtime/`, `cli/`)
- **Error handling**: Use typed error constants from `core/errors` (e.g., `ErrCommandNotFound`)
- **Testing**: Use `github.com/google/go-cmp/cmp` for diffs, `testify` for assertions
  - **CRITICAL**: Always test complete, exact output with `cmp.Diff` - no lazy partial tests with `assert.Contains`
  - **Bug reproduction**: Always reproduce parser/CLI bugs in existing test files before fixing
  - Add failing tests to `runtime/lexer/*_test.go` and `runtime/parser/*_test.go` for parsing issues
- **Types**: Define in `core/types/types.go`, use `TokenType` and `ExpressionType` enums
- **Registry pattern**: All decorators use global registry, no hardcoded implementations
- **Temporary debugging files**: ALWAYS prefix with `DELETE_ME_DEBUG_` (e.g., `DELETE_ME_DEBUG_main.go`, `DELETE_ME_DEBUG_memory_alloc_test.go`) so they're clearly identified as temporary and excluded from version control

## Architecture Rules
- **Module deps**: `core/` (foundation) → `runtime/` → `cli/` (top-level)
- **Go workspace**: Project uses `go.work` with modules in `core/`, `runtime/`, `cli/`
  - Always run `go work sync` after dependency changes
  - Use `go work use .` to add new modules to workspace
  - When testing modules individually: `cd module && go test ./...`
  - For cross-module imports: `github.com/aledsdavies/opal/core/...`
- **String system**: Use `StringLiteral.Parts[]` not deprecated `.Value`
- **Decorators**: Registry-driven via `decorators.RegisterAction/RegisterValue/etc.`
- **Fresh builds**: Rebuild CLI after any core/runtime changes
- **Phase 1**: Interpreter mode only, no code generation yet

## Pre-PR Checklist
- **Nix package hash**: If Go dependencies changed, update `vendorHash` in `.nix/package.nix`
  - When Nix build fails with hash mismatch, copy the "got:" hash to replace the "expected:" hash
  - This ensures the package builds correctly in Nix environments

### Pre-PR Checklist
- **Nix package hash**: If Go dependencies changed, update `vendorHash` in `.nix/package.nix`
  - When Nix build fails with hash mismatch, copy the "got:" hash to replace the "expected:" hash
  - This ensures the package builds correctly in Nix environments

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

### TDD Development Rules
- **Test-Driven Development**: Always write failing tests before implementation
- **Test categories**: Golden tests (exact token output), performance tests (5000+ lines/ms), error tests (precise error messages)
- **Red-Green-Refactor**: See test fail → make it pass → run full suite → refactor safely
- **Logical test groups**: Group tests by language feature (control flow, decorators, interpolation)
- **No implementation without tests**: Every new feature starts with a failing test

### When Making Changes
- **Write test first**: Create failing test for new feature
- **Minimal implementation**: Write just enough code to pass the test
- **Run full suite**: Ensure no regressions in existing functionality
- **Follow coding guidelines**: Apply all code style and conventions
- **Follow docs specifications**: Implement according to SPECIFICATION.md and ARCHITECTURE.md
- **Update WORK.md**: Keep work tracking clean and terse throughout development
- **AST/IR changes**: Must support single Decorator/DecoratorNode types
- **New decorators**: Must use unified interfaces and parameter system
- **Parser changes**: Must convert shell syntax to `@shell` decorators
- **Execution changes**: Must follow decorator completion model
- **Plan changes**: Must support both quick and resolved plan generation

**Never introduce special cases or break the unified decorator architecture.**

This is an early-stage project focused on generating truly standalone CLI tools from declarative definitions.
