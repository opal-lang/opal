# AGENTS.md

**opal** - CLI generation tool that creates standalone binaries from declarative command definitions.

## CRITICAL: Refactoring and Migration Rules (READ THIS FIRST)

**When refactoring or migrating code structure (e.g., replacing Commands with Tree):**

### MANDATORY PROCESS - NO EXCEPTIONS

1. **VERIFY EQUIVALENCE FIRST** - Before touching ANY tests:
   - Write NEW tests that prove the new structure produces IDENTICAL behavior to the old structure
   - Run BOTH old and new implementations side-by-side to verify exact equivalence
   - Document any behavioral differences explicitly
   - Get ALL new tests passing BEFORE removing old code

2. **FIX ROOT CAUSES, NOT SYMPTOMS**:
   - If tests fail, find and fix the BUG in the implementation
   - NEVER change test expectations to match broken behavior
   - NEVER comment out failing tests to "get the build working"
   - NEVER leave tests half-finished or broken

3. **COMPLETE EACH PIECE FULLY**:
   - Fix one module completely before moving to the next
   - Ensure ALL tests pass for that module
   - Verify functionality matches documentation
   - No "TODO" or "FIXME" comments without a tracking issue

4. **TEST SYSTEMATICALLY**:
   - After each change: run tests for that module
   - After each module: run full test suite
   - Before committing: verify ALL tests pass (core, runtime, cli)
   - No commits with failing tests unless explicitly discussed

5. **PRESERVE BEHAVIOR EXACTLY**:
   - If old code did X, new code must do X identically
   - If output format was Y, new output must be Y exactly
   - If error message was Z, new error must be Z exactly
   - Any deviation requires explicit approval

### Anti-Patterns That Cause Slop

❌ **Changing test expectations instead of fixing bugs**
- Example: Test expects "max=3", output is "max=", changing test to expect "max="
- Correct: Fix the formatter to output "max=3"

❌ **Leaving tests broken "to fix later"**
- Example: Commenting out 15 tests because they don't compile
- Correct: Rewrite each test properly until all pass

❌ **Assuming equivalence without verification**
- Example: "Tree should work like Commands" without testing
- Correct: Write tests proving Tree produces identical behavior

❌ **Partial migrations**
- Example: Updating 3 of 5 test files, leaving 2 broken
- Correct: Complete all test files before committing

### When You Catch Yourself Doing Slop

**STOP. Ask yourself:**
1. Am I changing a test expectation instead of fixing a bug?
2. Am I leaving something broken "to fix later"?
3. Have I verified the new code behaves identically to the old code?
4. Would this commit pass all tests?

**If any answer is YES to #1-2 or NO to #3-4: STOP and fix it properly.**

## Build/Test Commands (CRITICAL)

**ALWAYS use `nix develop` for all development commands. NEVER use `nix-shell` or install tools globally.**

```bash
# Test individual module
cd core && go test ./...
cd runtime && go test ./...
cd cli && go test ./...

# Format code (REQUIRED before PR)
nix develop --command gofumpt -w .

# Lint code (REQUIRED before PR)
cd core && nix develop ..#default --command golangci-lint run --timeout=5m
cd runtime && nix develop ..#default --command golangci-lint run --timeout=5m
cd cli && nix develop ..#default --command golangci-lint run --timeout=5m

# Build Nix package (REQUIRED before PR)
nix build

# If Nix build fails with hash mismatch:
# 1. Copy the "got:" hash from error message
# 2. Update vendorHash in .nix/package.nix with the new hash
# 3. Run `nix build` again to verify
```

## Performance & Telemetry

**Pipeline timing flag:**
```bash
# Show timing breakdown for all pipeline stages
opal -f commands.opl deploy --timing
```

**Output format:**
```
Pipeline Timing:
  Parse:   67.5µs
  Plan:    8.2µs
  Execute: 3.2ms (1 steps)
    Step 1: 3.1ms (exit 0)
  Total:   3.3ms
```

**Benchmarks:**
```bash
# Benchmark planner
cd runtime/planner && go test -bench=. -benchmem

# Benchmark executor
cd runtime/executor && go test -bench=. -benchmem
```

**Performance baselines (from benchmarks):**
- Planner (simple command): ~361ns/op, 392 B/op, 9 allocs/op
- Planner (complex script): ~4.7µs/op, 6480 B/op, 151 allocs/op
- Executor (single echo): ~2.46ms/op, 53476 B/op, 186 allocs/op

**Overhead breakdown:**
- Bash spawn: 81% (unavoidable OS overhead)
- SDK wrapper: 15% (safety + ergonomics)
- Executor: 4% (registry + telemetry)

**Conclusion:** Performance is excellent for MVP. No optimization needed.

## Pre-PR Checklist (MANDATORY)

**STOP. Before creating ANY PR, verify ALL of these pass IN THIS ORDER:**

1. ✅ **All tests pass**
   ```bash
   cd core && go test ./...
   cd runtime && go test ./...
   cd cli && go test ./...
   ```

2. ✅ **Nix build succeeds** (MUST be done before format/lint since nix develop depends on it)
   ```bash
   nix build
   # If hash mismatch:
   # 1. Copy the "got:" hash from error message
   # 2. Update vendorHash in .nix/package.nix (line 26)
   # 3. Run `nix build` again to verify
   ```

3. ✅ **Code is formatted**
   ```bash
   nix develop --command gofumpt -w .
   # Should show no changes
   ```

4. ✅ **All linters pass**
   ```bash
   cd core && nix develop ..#default --command golangci-lint run --timeout=5m
   cd runtime && nix develop ..#default --command golangci-lint run --timeout=5m
   cd cli && nix develop ..#default --command golangci-lint run --timeout=5m
   ```

5. ✅ **No uncommitted changes** (except intentional)
   ```bash
   jj status
   ```

**If ANY of these fail, FIX IT before creating the PR.**

## File Editing Guidelines (CRITICAL)

**ALWAYS use the Edit tool for modifying files. NEVER use `cat >` or `cat >>` to write files.**

- ✅ **Use Edit tool**: Precise, shows exactly what changed, preserves file structure
- ❌ **Never use `cat >`**: Overwrites entire file, error-prone, hard to review
- ❌ **Never use `cat >>`**: Appends blindly, can create duplicates, breaks syntax

**Why Edit tool is better:**
- Shows exact old/new strings in diff
- Catches errors (duplicate code, wrong location)
- Preserves formatting and structure
- Easy to review changes
- Fails safely if oldString not found

**Example:**
```bash
# ❌ BAD: Using cat to append
cat >> file.go << 'EOF'
func NewFunction() {}
EOF

# ✅ GOOD: Using Edit tool
# (Use Edit tool with oldString/newString parameters)
```

## Coding Guidelines (CRITICAL)

**ALWAYS follow the coding guidelines documented in this file and the global OpenCode configuration.**

### Core Safety Principles
- **Fail-Fast on Programming Errors**: Assert all invariants immediately - never silently continue
- **Use `core/invariant` package**: ALWAYS use invariant functions instead of raw `panic()`
  - `invariant.Precondition(cond, msg, args...)` - Check function inputs at entry
  - `invariant.Postcondition(cond, msg, args...)` - Check function outputs before return
  - `invariant.Invariant(cond, msg, args...)` - Check internal consistency (loops, state)
  - `invariant.NotNil(value, name)` - Check pointer/interface arguments
  - `invariant.InRange(value, min, max, name)` - Check numeric bounds
  - Provides clear violation messages with call stack context
- **Loop Invariants**: Every loop MUST track position and assert progress is made
- **Zero Performance Cost**: Assertions use simple checks (position comparison, nil checks)

### Error Taxonomy
- **Programming errors**: Panic immediately (invariant violations, stuck loops, nil dereferences)
- **User errors**: Return structured errors (invalid syntax, missing files)
- **System errors**: Log + alert + safe degradation (network failures, disk full)

## Code Style & Conventions
- **Go 1.25.0** with workspace modules (`core/`, `runtime/`, `cli/`)
- **Imports**: Standard library first, blank line, then third-party, blank line, then local packages
- **Error handling**: Use typed error constants from `core/errors` (e.g., `ErrCommandNotFound`)
- **Testing**: Use `github.com/google/go-cmp/cmp` for diffs, `testify` for assertions
  - **CRITICAL**: Always test complete, exact output with `cmp.Diff` - no lazy partial tests with `assert.Contains`
  - **Bug reproduction**: Always reproduce parser/CLI bugs in existing test files before fixing
  - Add failing tests to `runtime/lexer/*_test.go` and `runtime/parser/*_test.go` for parsing issues
  - **Golden tests**: Byte-exact plan output for contract verification (see `docs/TESTING_STRATEGY.md`)
- **Types**: Define in `core/types/types.go`, use `TokenType` and `ExpressionType` enums
- **Registry pattern**: All decorators use global registry, no hardcoded implementations
- **Naming**: Use `lower_snake_case` for decorator parameters, verb-first for decorator names (see `docs/CLEAN_CODE_GUIDELINES.md`)
- **Temporary debugging files**: ALWAYS prefix with `DELETE_ME_DEBUG_` (e.g., `DELETE_ME_DEBUG_main.go`, `DELETE_ME_DEBUG_memory_alloc_test.go`) so they're clearly identified as temporary and excluded from version control
- **Opal syntax in tests**:
  - Use `@var.NAME` for all Opal variables (strings, paths, args)
  - Example: `echo "Hello @var.name!"`
  - Example: `kubectl apply -f k8s/@var.service/`
  - Example: `kubectl scale --replicas=@var.COUNT deployment/@var.NAME`
  - Terminate with `()` if followed by ASCII: `@var.service()_backup`
  - **NEVER use `${var}`** - that's shell syntax, not Opal
  - Exception: Actual shell variables inside `@shell()` commands

## Observability & Debug Requirements (CRITICAL)

**REQUIRED**: All lexer and parser components MUST implement zero-overhead observability from day one.

### Debug Levels (see `docs/OBSERVABILITY.md`)
- **DebugOff**: Zero overhead (default, production)
- **DebugPaths**: Method entry/exit tracing (development)
- **DebugDetailed**: Token/event-level tracing (debugging)

### Telemetry Levels
- **TelemetryOff**: Zero overhead (default)
- **TelemetryBasic**: Counts only (production-safe)
- **TelemetryTiming**: Counts + timing (performance analysis)

### Implementation Pattern (Lexer & Parser)
```go
// Method-level tracing - DebugPaths
if p.config.debug >= DebugPaths {
    p.recordDebugEvent("enter_method", "context")
}

// Detailed tracing - DebugDetailed only
if p.config.debug >= DebugDetailed {
    p.recordDebugEvent("token_consumed", fmt.Sprintf("pos: %d, type: %v", p.pos, tok.Type))
    p.recordDebugEvent("loop_iteration", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current()))
}

// Zero overhead when disabled
if p.config.debug == DebugOff {
    // No checks, no allocations, no function calls
}
```

### Required Debug Events
**Parser must record (when DebugDetailed enabled)**:
- Token consumption: `consumed_token: pos X, type Y`
- Event emission: `emitted_event: EventOpen NodeKind`
- Loop iterations: `loop_iteration: pos X, token Y` (catches infinite loops)
- Error recovery: `recovery_start`, `recovery_sync_found`, `consumed_separator`
- Position tracking: `advanced_from: X to: Y`

**Why**: Detailed debug output would have caught the infinite loop bug immediately by showing repeated `loop_iteration: pos: 6` events.

### Performance Requirements
- **Zero overhead when disabled**: No checks, no allocations, no function calls
- **Minimal overhead when enabled**: Simple conditionals only
- **Benchmark verification**: Must verify zero overhead in benchmarks (see `runtime/lexer/benchmark_test.go`)

### Fail-Fast Invariants
- **Loop guards**: Assert progress in all loops (panic on stuck parser)
- **Position tracking**: `prevPos := p.pos` before loop, check after iteration
- **Clear panic messages**: Include position, token type, and context

**See**: `docs/OBSERVABILITY.md` for production observability model and `docs/TESTING_STRATEGY.md` for testing requirements.

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

## JJ to Git Workflow

**CRITICAL: Use fine-grained commits to prevent work loss**

```bash
# 1. Create new change for each logical unit of work
jj new -m "feat: add DOTDOTDOT token to lexer"
# make lexer changes
jj describe -m "feat: add DOTDOTDOT token to lexer

- Add DOTDOTDOT token type
- Implement lexing logic for ...
- Add 5 lexer tests"

# 2. Create NEW change for next unit (don't keep working in same change)
jj new -m "feat: add range pattern parsing"
# make parser changes
jj describe -m "feat: add range pattern parsing

- Add NodePatternRange
- Update pattern() parser
- Add parser tests"

# 3. Create NEW change for refactoring/cleanup
jj new -m "refactor: extract shared fuzz seed corpus"
# refactor fuzz tests
jj describe -m "refactor: extract shared fuzz seed corpus

- Add addSeedCorpus() function
- Update all 7 fuzz functions to use shared corpus
- Add FuzzParserSmokeTest as 8th function"

# 4. Fix empty descriptions before push
jj log --limit 5                    # Check for "(no description set)"
jj describe -r <change_id> -m "msg" # Fix any empty descriptions

# 5. Create bookmark and push
jj bookmark create -r @ feature-name
jj git push -b feature-name --allow-new

# 6. Update existing PR
jj bookmark set feature-name -r @
jj git push -b feature-name

# 7. Create GitHub PR (gh CLI with JJ)
# NOTE: gh CLI doesn't detect JJ branches automatically
# MUST pass branch name explicitly with --head flag
gh pr create --head feature-name --title "Title" --body "Description"
```

**Why fine-grained commits matter:**
- Each `jj new` creates a snapshot - if you lose work, you can recover from previous change
- Large changes in single commit = all-or-nothing (lose everything if something goes wrong)
- Small commits = easy to review, easy to revert, easy to recover
- Rule: If you're working on 2+ distinct things, use separate `jj new` calls

## Pre-PR Checklist
- **All commits have descriptions**: JJ won't push commits with "(no description set)" to Git
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
3. **Maintain unified model**: No special cases that break "everything is a decorator" (for work execution)
4. **Preserve execution semantics**: Decorator blocks complete before chain evaluation
5. **Support dual mode**: Both command mode and script mode execution
6. **Follow decorator guidelines**: Apply naming and composition rules from `docs/DECORATOR_GUIDE.md`

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
