---
oep: 002
title: Transform Pipeline Operator `|>`
status: Draft
type: Feature
created: 2025-01-21
updated: 2025-01-21
---

# OEP-002: Transform Pipeline Operator `|>`

## Summary

Add `|>` operator for deterministic transformations and inline assertions on command output. Unlike `|` (which pipes raw stdout/stderr to another command), `|>` pipes through Opal-native, pure, bounded transforms called **PipeOps**. This enables safe, type-checked data transformations with memory guarantees.

## Motivation

### The Problem

Shell pipelines (`|`) are powerful but unsafe:
- **Unbounded memory:** `cat huge.log | grep pattern` can exhaust memory
- **Non-deterministic:** External commands may behave differently across systems
- **No type safety:** Piping JSON to a text tool silently fails
- **Hard to test:** External dependencies make tests brittle
- **No inline validation:** Must write separate steps to validate output

**Example of current limitations:**

```bash
# ❌ Shell pipeline - unbounded, non-deterministic, no validation
kubectl get pods -o json | jq '.items[].metadata.name' | head -5

# ❌ No inline assertions - must write separate validation steps
curl /health
if ! grep -q "Status 200" health.txt; then
    kubectl rollout undo deployment/app
fi
```

### Use Cases

**1. JSON extraction with validation:**
```opal
kubectl get pods -o json |> json.pick("$.items[].metadata.name") |> assert.num(">= 3")
```

**2. Health check validation:**
```opal
@http.healthcheck(url=@var.endpoint, retries=10) |> assert.re("Status 200")
```

**3. Log analysis:**
```opal
kubectl logs app |> lines.grep("ERROR") |> lines.count() |> assert.num("== 0")
```

**4. Deployment with rollback:**
```opal
deploy: {
    kubectl apply -f k8s/
    defer { kubectl rollout undo deployment/app }
    
    curl /health |> assert.re("Status 200")
}
```

**5. Secret scrubbing before file output:**
```opal
// Scrub secrets from kubectl output before writing to file
kubectl get secret db-password -o json |> scrub() > backup.json
```

## Proposal

### Syntax

**Basic transform:**
```opal
<command> |> <pipeop>(<args>)
```

**Chained transforms:**
```opal
<command> |> <pipeop1>(<args>) |> <pipeop2>(<args>) |> ...
```

### PipeOp Characteristics

PipeOps are Opal-native transforms with enforced traits:
- **Pure**: No filesystem, network, clock, or environment access
- **Bounded**: Declare `MaxExpansionFactor` for memory safety
- **Deterministic**: Same input always produces same output
- **Trait declarations**: Each PipeOp declares `Deterministic=true`, `Bounded=true`, `MaxExpansionFactor`, `ReadsStdin=true`, `WritesStdout=true`, `WritesStderr=false`

### Built-in PipeOps

**Transform PipeOps:**
- `json.pick(path)` - Extract JSON using JSONPath (MaxExpansionFactor=1.0)
- `lines.grep(pattern)` - Filter lines matching RE2 pattern (MaxExpansionFactor=1.0)
- `lines.head(n)` - Take first n lines (MaxExpansionFactor=0.0)
- `lines.tail(n)` - Take last n lines (MaxExpansionFactor=0.0)
- `lines.count()` - Count number of lines (MaxExpansionFactor=0.0)
- `columns.pick(n)` - Extract column n (space-delimited) (MaxExpansionFactor=1.0)
- `scrub()` - Replace secrets with DisplayIDs before output (MaxExpansionFactor=1.0)

**Assertion PipeOps:**
- `assert.re(pattern)` - Input must match RE2 pattern (MaxExpansionFactor=1.0)
- `assert.num(expr)` - Numeric predicate (`==`, `>=`, `<=`, `!=`, `>`, `<`) on parsed number (MaxExpansionFactor=0.0)

### Core Restrictions

#### Restriction 1: PipeOps Must Be Pure

PipeOps cannot access filesystem, network, clock, or environment:

```opal
// ❌ FORBIDDEN: filesystem access
@shell("cat /etc/passwd") |> my.custom_pipeop()  // Error if custom_pipeop reads files

// ❌ FORBIDDEN: network access
@shell("curl example.com") |> my.custom_pipeop()  // Error if custom_pipeop makes requests

// ✅ CORRECT: pure transformation
@shell("echo hello") |> lines.count()
```

**Why?** Pure transforms are deterministic and safe to cache/parallelize.

#### Restriction 2: PipeOps Must Be Bounded

Each PipeOp declares `MaxExpansionFactor`. Cumulative expansion cannot exceed 10.0:

```opal
// ✅ Safe: MaxExpansionFactor = 1.0 + 0.0 = 1.0
kubectl logs app |> lines.grep("ERROR") |> lines.head(10)

// ❌ Rejected: Cumulative MaxExpansionFactor > 10.0
huge.json |> json.pick("$..items") |> json.pick("$..items") |> ...  // 100 times
```

**Why?** Prevents unbounded memory growth.

#### Restriction 3: PipeOps Must Be Deterministic

Same input must always produce same output:

```opal
// ❌ FORBIDDEN: non-deterministic
@shell("date") |> my.custom_pipeop()  // Error if custom_pipeop depends on current time

// ✅ CORRECT: deterministic
@shell("echo hello") |> lines.count()
```

**Why?** Determinism enables plan verification and reproducibility.

#### Restriction 4: No Side Effects in Assertions

Assertion PipeOps must not modify state:

```opal
// ❌ FORBIDDEN: side effects
curl /health |> assert.re("Status 200") |> my.log_to_file()  // Error if assertion has side effects

// ✅ CORRECT: pure assertion
curl /health |> assert.re("Status 200")
```

**Why?** Assertions should only validate, not modify.

#### Restriction 5: No Arbitrary Commands in `|>` Chains

Cannot use `|` inside `|>` chains:

```opal
// ❌ FORBIDDEN: mixing | and |>
kubectl get pods -o json |> grep "Running" | jq '.metadata.name'  // Error: can't mix | and |>

// ✅ CORRECT: use PipeOps only
kubectl get pods -o json |> lines.grep("Running") |> json.pick("$.metadata.name")
```

**Why?** Defeats the purpose of bounded, pure, deterministic transforms.

### Execution Semantics

**Success:** Transform passes, output flows forward
```opal
curl /api/users |> json.pick("$.count") |> assert.num("> 0")
# If count > 0 → step succeeds, count value available downstream
```

**Failure:** Transform or assertion fails, step fails
```opal
curl /health |> assert.re("Status 200")
# If pattern doesn't match → step fails, raises AssertionFailed
# Execution stops (unless in @retry or try/catch)
```

### Handling Assertion Failures

Assertions are step failures - use Opal's execution-time control flow to handle them:

**Pattern 1: Rollback with `||` operator**
```opal
deploy: {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
    
    // Validate health, rollback on failure
    curl /health |> assert.re("Status 200") || kubectl rollout undo deployment/app
}
```

**Pattern 2: Try/catch for complex logic**
```opal
deploy: {
    try {
        kubectl apply -f k8s/
        curl /health |> assert.re("Status 200")
        kubectl get pods |> lines.grep("Running") |> lines.count() |> assert.num(">= 3")
    } catch {
        echo "Validation failed, rolling back"
        kubectl rollout undo deployment/app
    }
}
```

**Pattern 3: Retry before rollback**
```opal
deploy: {
    kubectl apply -f k8s/
    
    @retry(attempts=5, delay=10s) {
        curl /health |> assert.re("Status 200")
    } || {
        echo "Health check failed after retries, rolling back"
        kubectl rollout undo deployment/app
    }
}
```

### Plan Representation

The plan shows transform pipelines with their operator graph:

```
deploy:
├─ kubectl get pods -o json
│  └─ |> json.pick("$.items[].metadata.name")
│     └─ |> lines.count()
│        └─ |> assert.num(">= 3")
└─ echo "Pod validation passed"
```

### Operator Composition

`|>` composes with other Opal constructs:

```opal
// With execution decorators
@retry(attempts=3) {
    curl /api |> json.pick("$.status") |> assert.re("ready")
}

// With conditionals (plan-time)
if @var.ENV == "production" {
    kubectl get pods |> lines.count() |> assert.num(">= 3")
}

// With parallel execution
@parallel {
    curl /api/users |> assert.re("Status 200")
    curl /api/posts |> assert.re("Status 200")
}

// With runtime bindings (OEP-001)
let.endpoint = @k8s.deploy(manifest="app.yaml").service_url
@http.get(url=@let.endpoint) |> assert.re("healthy")
```

## Rationale

### Why `|>` instead of `|`?

**Distinction:** `|` is shell-native (pipes to external commands), `|>` is Opal-native (pipes through PipeOps).

**Safety:** `|>` enforces purity, boundedness, and determinism. `|` has no such guarantees.

**Tooling:** LSP can validate PipeOp chains at parse time, catching errors before execution.

### Why enforce purity?

**Determinism:** Pure transforms produce the same output for the same input, making plans reproducible.

**Security:** No filesystem/network access means PipeOps can't exfiltrate data or cause side effects.

**Performance:** Pure transforms can be cached, parallelized, and optimized safely.

### Why declare `MaxExpansionFactor`?

**Memory safety:** Prevents unbounded memory growth (e.g., `lines.head(5)` can't use more memory than input).

**Plan validation:** Planner can reject chains with excessive cumulative expansion.

**Predictability:** Operators have known memory bounds.

### Why not just use shell pipelines?

**Shell pipelines are fine for simple cases.** Use them when you don't need validation or safety:

```opal
// ✅ Shell pipeline - simple, no validation needed
kubectl logs app | grep ERROR | head -10
```

**Use `|>` when you need:**
- Inline assertions (`assert.re`, `assert.num`)
- Memory safety (bounded transforms)
- Deterministic behavior (pure transforms)
- Plan-time validation (type checking, expansion limits)

## Alternatives Considered

### Alternative 1: Extend `|` to support PipeOps

**Rejected:** Confusing. `|` already means "shell pipe". Overloading it would break existing scripts and make it unclear which semantics apply.

### Alternative 2: Use method chaining (`.pick().count()`)

**Rejected:** Doesn't fit Opal's syntax. Opal is command-oriented, not object-oriented.

```opal
// ❌ Doesn't fit Opal's style
kubectl.get("pods").json().pick("$.items[].metadata.name").count()
```

### Alternative 3: Use decorators (`@json.pick`, `@lines.count`)

**Rejected:** Decorators wrap blocks, not transform output. Using decorators for transforms would be confusing.

### Alternative 4: Allow arbitrary commands in `|>` chains

**Rejected:** Defeats the purpose. We want bounded, pure, deterministic transforms. Allowing arbitrary commands would bring back all the problems of shell pipelines.

### Alternative 5: No memory bounds

**Rejected:** Unbounded transforms can exhaust memory. Declaring `MaxExpansionFactor` prevents this.

## Implementation

### Phase 1: Parser & AST
- Add `PipeOp` AST node
- Parse `|>` operator and PipeOp calls
- Validate PipeOp syntax (name, args)
- Validate no `|` inside `|>` chains

### Phase 2: PipeOp Registry
- Define `PipeOp` trait (Pure, Bounded, Deterministic, MaxExpansionFactor)
- Implement built-in PipeOps (json.pick, lines.grep, assert.re, etc.)
- Registry for PipeOp lookup and validation

### Phase 3: Planner
- Represent PipeOp chains in plan
- Validate cumulative MaxExpansionFactor
- Include operator graph in plan hash
- Validate purity and determinism

### Phase 4: Executor
- Execute PipeOp chains (stdin → PipeOp1 → PipeOp2 → ... → stdout)
- Handle assertion failures (raise AssertionFailed)
- Integrate with error handling (`try/catch`, `||`, `defer`)
- Memory tracking for bounded transforms

### Phase 5: Extensibility
- Plugin API for custom PipeOps
- Validation of custom PipeOp traits
- Documentation for PipeOp authors

## Compatibility

**Breaking changes:** None. This is a new operator.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Parallel groups:** Should we support `|> [branch1, branch2]` for parallel transform branches?
2. **Custom PipeOps:** Should users be able to define custom PipeOps in Opal scripts, or only via plugins?
3. **Error context:** Should assertion failures include the input that failed? (e.g., "Expected 'Status 200', got 'Status 500'")
4. **Streaming:** Should PipeOps support streaming (process input line-by-line) or buffer entire input?
5. **Type system:** Should PipeOps have explicit input/output types for better error messages?

## References

- **Elixir's pipe operator:** `|>` for function chaining (inspiration for syntax)
- **F# pipe operator:** Similar concept, but for function composition
- **Unix philosophy:** Small, composable tools (inspiration for PipeOp design)
- **Related OEPs:**
  - OEP-001: Runtime Variable Binding with `let` (uses `|>` for validation)
  - OEP-003: Automatic Cleanup with `defer` (uses `|>` assertions to trigger cleanup)
