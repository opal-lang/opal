# AGENTS.md

**opal** - Plan-first execution platform for deployments, infrastructure, and operations. Turn operational workflows into verifiable contracts.

## Build/Test Commands

**ALWAYS use `nix develop` for all development commands.**

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
```

## Nix Integration

**Update vendorHash when Go dependencies change:**

```bash
# Step 1: Set vendorHash to fake value in .nix/package.nix
vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

# Step 2: Run nix build to get the correct hash
nix build 2>&1 | grep -E "(got:|specified:)"

# Step 3: Update vendorHash with the "got:" hash
vendorHash = "sha256-MhpMwo7ayOwyyvjtu/BtPzpN/CZ6KZ/mKaQKeGPaPy0=";

# Step 4: Verify build succeeds
nix build
```

**When to update:**
- ✅ Changed function signatures in any Go module
- ✅ Added/removed Go dependencies
- ✅ Updated Go module versions
- ❌ Only changed implementation (no interface changes)

## Performance Baselines

**Current benchmarks:**
- Planner (simple): ~361ns/op, 392 B/op, 9 allocs/op
- Planner (complex): ~4.7µs/op, 6480 B/op, 151 allocs/op
- Executor (single echo): ~2.46ms/op, 53476 B/op, 186 allocs/op

**Overhead:** Bash spawn 81%, SDK wrapper 15%, Executor 4%

## Project-Specific Conventions

**Go workspace:**
- Module deps: `core/` → `runtime/` → `cli/`
- Cross-module imports: `github.com/aledsdavies/opal/core/...`
- Run `go work sync` after dependency changes

**Opal patterns:**
- **Invariants**: Use `core/invariant` package (not raw `panic()`)
- **Registry**: All decorators use global registry
- **Testing**: Golden tests for byte-exact plan output
- **Observability**: Zero-overhead debug levels (see `docs/OBSERVABILITY.md`)

**Opal syntax in tests:**
```opal
echo "Hello @var.name!"
kubectl scale --replicas=@var.COUNT deployment/@var.NAME
@var.service()_backup  # Terminate with () if followed by ASCII
```
**NEVER use `${var}`** - that's shell syntax, not Opal.

## Architecture Compliance

**CRITICAL**: Always follow `docs/SPECIFICATION.md` and `docs/ARCHITECTURE.md`.

**Core principles:**
- **Everything is a Decorator**: Shell syntax becomes `@shell` decorators internally
- **Two Interface System**: Only `ValueDecorator` and `ExecutionDecorator`
- **Plan-Then-Execute**: Support quick plans and resolved plans with contract verification

**When making changes:**
1. Read `docs/SPECIFICATION.md` for user-facing behavior
2. Follow `docs/ARCHITECTURE.md` design patterns
3. Maintain unified decorator model (no special cases)
4. Test complete exact output with `cmp.Diff` - no lazy partial tests

**Refactoring rules:**
- Write NEW tests proving identical behavior BEFORE removing old code
- Fix bugs in implementation, NEVER change test expectations to match broken behavior
- Complete one module fully before moving to next
- No "TODO" comments without tracking issue

This is an early-stage project focused on plan-first execution with contract verification for operational workflows.
