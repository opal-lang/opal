# AGENTS.md

**devcmd** - CLI generation tool that creates standalone binaries from declarative command definitions.

## Build/Test Commands
```bash
# Single test: go test ./module/package -run TestName -v
# Module test: devcmd run core-test (or runtime-test, cli-test, etc.)
# All tests: devcmd run test
# Full CI: devcmd run ci
# Build CLI: cd cli && go build -o devcmd . 
# Format: devcmd run format
# Lint: devcmd run lint
```

## Code Style & Conventions
- **Go 1.24.3** with workspace modules (`core/`, `runtime/`, `cli/`, `testing/`, `codegen/`)
- **Imports**: stdlib first, then third-party, then internal modules
- **Error handling**: Use typed error constants from `core/errors` (e.g., `ErrCommandNotFound`)
- **Testing**: Use `github.com/google/go-cmp/cmp` for diffs, `testify` for assertions
  - **CRITICAL**: Always test complete, exact output with `cmp.Diff` - no lazy partial tests with `assert.Contains`
  - Use table-driven tests for multiple scenarios
  - Test the entire output format, not just parts of it
  - **Bug reproduction**: Always reproduce parser/CLI bugs in existing test files before fixing
  - Add failing tests to `cli/internal/parser/*_test.go` for parser issues
- **Types**: Define in `core/types/types.go`, use `TokenType` and `ExpressionType` enums
- **Registry pattern**: All decorators use global registry, no hardcoded implementations
- **Context passing**: Pass context explicitly, avoid global state
- **Naming**: CamelCase for public, camelCase for private, descriptive error names

## Architecture Rules
- **Module deps**: `core/` (foundation) → `runtime/` → `cli/` (top-level)
- **String system**: Use `StringLiteral.Parts[]` not deprecated `.Value`
- **Decorators**: Registry-driven via `decorators.RegisterAction/RegisterValue/etc.`
- **Fresh builds**: Rebuild CLI after any core/runtime changes
- **Phase 1**: Interpreter mode only, no code generation yet

This is an early-stage project focused on generating truly standalone CLI tools from declarative definitions.