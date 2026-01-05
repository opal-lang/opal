---
oep: 002
title: Transform Pipeline Operator `|>`
status: Draft
type: Feature
created: 2025-01-21
updated: 2025-01-27
---

# OEP-002: Transform Pipeline Operator `|>`

## Summary

Add an Elm-style pipeline operator (`|>`) for deterministic data transformations. The operator passes command output through pure functions called PipeOps, enabling type-safe parsing, filtering, and validation within Opal's execution model.

## Motivation

Shell scripts frequently need to parse command output, but the standard approach has limitations:

```bash
# Typical shell pattern - works but fragile
count=$(kubectl get pods | grep Running | wc -l)
if [ "$count" -lt 3 ]; then
    echo "Not enough pods"
    exit 1
fi
```

**Problems:**
- `grep` and `wc` vary across systems (GNU vs BSD)
- No inline validation - requires separate `if` blocks
- Everything is text - no awareness of JSON, YAML, etc.
- Hard to compose multiple transformations cleanly

**Solution:** Introduce `|>` for pure, deterministic transformations:

```opal
kubectl get pods |> lines.grep("Running") |> lines.count() |> assert.num(">= 3")
```

## Proposal

### Syntax

```opal
<command> |> <pipeop>(<args>) |> <pipeop>(<args>) ...
```

The `|>` operator:
1. Captures stdout from the left-hand side
2. Passes it as input to the PipeOp on the right
3. Returns the transformed output (which can be piped further)

### What is a PipeOp?

A PipeOp is a **pure function** that transforms data. Pure means:
- **Deterministic**: Same input always produces same output
- **No side effects**: Doesn't read files, make requests, or modify state
- **Referentially transparent**: Can be cached, memoized, or run in parallel

This is the same concept as pure functions in Elm, Haskell, or functional JavaScript.

### Built-in PipeOps

**Text/Lines:**
- `lines.grep(pattern)` - Filter lines matching regex
- `lines.head(n)` - First N lines
- `lines.tail(n)` - Last N lines  
- `lines.count()` - Count lines

**JSON:**
- `json.pick(path)` - Extract using JSONPath
- `json.keys()` - Get object keys
- `json.values()` - Get object values

**Assertions:**
- `assert.re(pattern)` - Fail if doesn't match regex
- `assert.num(expr)` - Numeric comparison (`>= 3`, `== 0`)
- `assert.eq(value)` - Exact equality

**Security:**
- `scrub()` - Redact secrets from output

### Examples

**Health check with assertion:**
```opal
curl /health |> assert.re("Status 200")
```

**Count running pods:**
```opal
kubectl get pods |> lines.grep("Running") |> lines.count()
```

**Extract JSON field:**
```opal
kubectl get pod app -o json |> json.pick("$.status.phase")
```

**Chain with shell pipe (both work together):**
```opal
kubectl get pods |> lines.grep("Running") |> lines.count() | xargs -I {} echo "Running pods: {}"
```

**Validate before proceeding:**
```opal
deploy: {
    kubectl apply -f k8s/
    curl /health |> assert.re("Status 200") || kubectl rollout undo deployment/app
}
```

**Scrub secrets before writing:**
```opal
kubectl get secret db-creds -o yaml |> scrub() > backup.yaml
```

### Why `|>` Instead of `|`?

| Aspect | Shell Pipe `\|` | Transform Pipe `\|>` |
|--------|----------------|---------------------|
| Connects | Processes | Functions |
| Data flow | Streaming bytes | Buffered values |
| Determinism | Depends on tools | Guaranteed |
| Side effects | Allowed | None |

They serve different purposes and work together:
- Use `|` when you need to stream data between processes
- Use `|>` when you need deterministic transformations

### Composing with Shell Pipes

`|>` and `|` can be mixed freely. The `|>` sections are deterministic islands within a larger pipeline:

```opal
# Shell pipe -> Opal transform -> Shell pipe
cat logs.txt | head -1000 |> lines.grep("ERROR") |> lines.count() | xargs echo "Errors:"
```

### Handling Failures

When an assertion fails, the step fails. Use Opal's error handling:

```opal
# Retry pattern
@retry(attempts=3) {
    curl /health |> assert.re("Status 200")
}

# Fallback pattern  
curl /health |> assert.re("Status 200") || echo "Health check failed"

# Try/catch pattern
try {
    curl /health |> assert.re("Status 200")
} catch {
    kubectl rollout undo deployment/app
}
```

## Implementation

### Phase 1: Core Infrastructure
- Add `|>` token to lexer
- Parse PipeOp expressions
- Implement PipeOp interface

### Phase 2: Built-in PipeOps
- Implement `lines.*` family
- Implement `json.*` family
- Implement `assert.*` family

### Phase 3: Integration
- Plan representation for PipeOp chains
- Executor support for buffered transforms

## Open Questions

1. **Custom PipeOps**: How do users define their own? Opal functions? Plugins?
2. **Streaming**: Can some PipeOps (like `lines.grep`) stream instead of buffer?
3. **Error context**: Should assertion failures show the actual vs expected value?

## References

- **Elm**: Inspiration for `|>` syntax and pure function philosophy
- **Elixir**: Pipeline operator semantics
- **Nushell**: Structured data in shell pipelines
