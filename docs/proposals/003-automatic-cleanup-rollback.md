---
oep: 003
title: Automatic Cleanup and Rollback
status: Draft
type: Feature
created: 2025-01-21
updated: 2025-01-21
---

# OEP-003: Automatic Cleanup and Rollback

## Summary

Add two complementary mechanisms for automatic cleanup and rollback on failure:

1. **`defer` blocks** - Register cleanup actions that execute in LIFO order when a block fails
2. **`ensure`/`rollback` operators** - Chain verification and compensation logic with `|>` operator

Both solve the same problem (automatic cleanup) but with different ergonomics. `defer` is simpler for straightforward cleanup; `ensure`/`rollback` is more powerful for complex verification chains.

## Motivation

### The Problem

Current Opal requires manual cleanup logic, which is error-prone:

```opal
// ❌ Manual cleanup - easy to forget or get ordering wrong
deploy: {
    try {
        kubectl apply -f k8s/
        kubectl create secret temp
        kubectl create configmap config
        curl /health |> assert.re("ready")
    } catch {
        // Must manually track and reverse order
        kubectl delete configmap config
        kubectl delete secret temp
        kubectl rollout undo deployment/app
    }
}
```

**Problems:**
- Easy to forget cleanup steps
- Easy to get cleanup order wrong (must be reverse of creation)
- Cleanup logic separated from creation logic
- Doesn't work well with retries or complex flows

### Use Cases

**1. Database migrations with rollback:**
```opal
migrate: {
    let.backup = @db.backup(database="prod")
    defer { @db.restore(file=@let.backup) }
    
    @db.migrate(file="migrations/001.sql")
    defer { @db.rollback(version="001") }
    
    @db.migrate(file="migrations/002.sql")
    defer { @db.rollback(version="002") }
    
    @db.query("SELECT COUNT(*) FROM users") |> assert.num("> 0")
}
```

**2. Multi-resource deployment:**
```opal
deploy: {
    kubectl create namespace @var.APP_NAME
    defer { kubectl delete namespace @var.APP_NAME }
    
    kubectl apply -f k8s/database.yaml -n @var.APP_NAME
    defer { kubectl delete -f k8s/database.yaml -n @var.APP_NAME }
    
    kubectl apply -f k8s/app.yaml -n @var.APP_NAME
    defer { kubectl delete -f k8s/app.yaml -n @var.APP_NAME }
    
    curl /health |> assert.re("Status 200")
}
```

**3. Canary deployment with progressive rollback:**
```opal
canary_deploy: {
    kubectl apply -f k8s/canary.yaml
    defer { kubectl delete -f k8s/canary.yaml }
    
    for percentage in [10, 25, 50, 100] {
        kubectl patch virtualservice @var.SERVICE \
            --patch='{"spec":{"http":[{"weight":@var.percentage}]}}'
        
        defer { kubectl patch virtualservice @var.SERVICE \
            --patch='{"spec":{"http":[{"weight":0}]}}' }
        
        @retry(attempts=5, delay=30s) {
            @http.post(url="http://prometheus:9090/api/v1/query", ...) 
                |> json.pick("$.data.result[0].value[1]") 
                |> assert.num("< 0.01")
        }
    }
}
```

## Proposal

### Approach 1: `defer` Blocks

Register cleanup actions that execute in LIFO order when a block fails.

#### Syntax

```opal
defer { <cleanup-action> }
```

#### Basic Usage

```opal
deploy: {
    kubectl apply -f k8s/
    defer { kubectl rollout undo deployment/app }
    
    kubectl create configmap temp-config
    defer { kubectl delete configmap temp-config }
    
    // If this fails, both defers run in reverse order:
    // 1. Delete configmap (most recent)
    // 2. Rollback deployment (first registered)
    curl /health |> assert.re("Status 200")
}
```

#### Execution Semantics

**On success:** Defers don't run, block completes normally

**On failure:** Defers execute in LIFO order, then error propagates

```opal
deploy: {
    kubectl apply -f k8s/              // ✓ succeeds
    defer { kubectl rollout undo }      // registered (position 1)
    
    kubectl create secret temp          // ✓ succeeds  
    defer { kubectl delete secret temp } // registered (position 2)
    
    curl /health |> assert.re("ready") // ✗ fails
    
    // Automatic execution:
    // 1. kubectl delete secret temp    (position 2 - LIFO)
    // 2. kubectl rollout undo           (position 1 - LIFO)
    // 3. Error propagates upward
}
```

#### Core Restrictions

**Restriction 1: Defers only run on failure**

Defers do not run if the block succeeds:

```opal
deploy: {
    kubectl apply -f k8s/
    defer { echo "This only runs if something fails" }
    
    curl /health |> assert.re("Status 200")  // ✓ succeeds
    // defer does NOT run
}
```

**Restriction 2: LIFO ordering (Last-In-First-Out)**

Defers execute in reverse registration order:

```opal
setup: {
    kubectl create namespace staging
    defer { kubectl delete namespace staging }  // Position 1
    
    kubectl create configmap app-config -n staging
    defer { kubectl delete configmap app-config -n staging }  // Position 2
    
    kubectl create secret db-creds -n staging
    defer { kubectl delete secret db-creds -n staging }  // Position 3
    
    // On failure, defers execute in reverse:
    // 1. kubectl delete secret db-creds -n staging
    // 2. kubectl delete configmap app-config -n staging
    // 3. kubectl delete namespace staging
}
```

**Restriction 3: Block-scoped**

Defers are visible only within their containing block:

```opal
deploy: {
    kubectl apply -f k8s/
    defer { echo "Outer cleanup" }
    
    {
        kubectl create configmap temp
        defer { echo "Inner cleanup" }
        
        curl /health |> assert.re("ready")  // Fails here
        // Runs: "Inner cleanup" only (inner block scope)
    }
    
    // Outer defer doesn't run - inner failure was contained
}
```

**Restriction 4: No conditional deferral**

Defers must be registered unconditionally:

```opal
// ❌ FORBIDDEN: defer inside if
if @var.ENV == "production" {
    defer { kubectl rollout undo }  // Parse error
}

// ✅ CORRECT: defer at block level
defer { kubectl rollout undo }
if @var.ENV == "production" {
    kubectl apply -f k8s/prod/
}
```

**Restriction 5: Parallel branch isolation**

Each `@parallel` branch has its own independent defer stack:

```opal
@parallel {
    {
        kubectl create configmap api-config
        defer { kubectl delete configmap api-config }
        kubectl apply -f k8s/api.yaml
        curl /api/health |> assert.re("Status 200")
    }
    
    {
        kubectl create configmap worker-config
        defer { kubectl delete configmap worker-config }
        kubectl apply -f k8s/worker.yaml
        kubectl logs -l app=worker |> assert.re("Started")
    }
}
```

If one branch fails, only its defers run. The other branch continues independently.

#### Scope Rules

**Block scoping:**
```opal
deploy: {
    let.instance_id = @aws.instance.deploy().id
    defer { @aws.instance.terminate(id=@let.instance_id) }
    
    // ✅ Visible in same block
    @aws.instance.tag(instanceId=@let.instance_id)
    
    // ✅ Visible in nested blocks
    {
        @aws.route53.update(instanceId=@let.instance_id)
    }
}

// ❌ Not visible here - out of scope
check: curl https://api.example.com/health
```

#### Interaction with Try/Catch

Defers run **before** the catch block executes:

```opal
deploy: {
    try {
        kubectl apply -f k8s/
        defer { kubectl rollout undo deployment/app }
        
        kubectl create configmap temp
        defer { kubectl delete configmap temp }
        
        curl /health |> assert.re("ready")  // Fails
        
        // Execution order:
        // 1. Defers run in LIFO: delete configmap, then rollback
        // 2. Error captured
        // 3. Catch block executes
        
    } catch {
        echo "Resources already cleaned up by defers"
        @slack.notify(message="Deployment failed and rolled back")
    }
}
```

#### Interaction with Retry Decorator

Defers in a retried block execute only on the final failed attempt (not between retries):

```opal
deploy: {
    kubectl apply -f k8s/
    defer { kubectl rollout undo deployment/app }
    
    @retry(attempts=3, delay=5s) {
        curl /health |> assert.re("ready")
        // Attempt 1 fails: retry (defer doesn't run)
        // Attempt 2 fails: retry (defer doesn't run)
        // Attempt 3 fails: defer runs, then error propagates
    }
}
```

This ensures cleanup only happens after all retry attempts are exhausted, not on transient failures.

#### Advanced Patterns

**Error-aware cleanup:**
```opal
deploy: {
    kubectl apply -f k8s/
    defer.with_error { err =>
        kubectl rollout undo deployment/app
        @slack.notify(
            channel="#alerts",
            message="Rollback due to: @var.err.message"
        )
    }
    
    curl /health |> assert.re("Status 200")
}
```

**Reliable cleanup with decorators:**
```opal
deploy: {
    kubectl create namespace @var.APP_NAME
    defer { 
        @timeout(30s) {
            kubectl delete namespace @var.APP_NAME 
        }
    }
    
    kubectl apply -f k8s/
    defer {
        @retry(attempts=3, delay=5s) {
            kubectl delete -f k8s/
        }
    }
}
```

### Approach 2: `ensure`/`rollback` Operators

Chain verification and compensation logic using `|>` operator.

#### Syntax

```opal
<work> |> ensure { <check> } |> rollback { <compensate> }
```

#### Basic Usage

```opal
kubectl apply -f k8s/app-v2.yaml
    |> ensure { curl -fsS http://app/health }
    |> rollback { kubectl rollout undo }
```

**Reads as:** "Do work, ensure this check passes, rollback if it fails"

#### Execution Semantics

Each operator in the chain responds to the result of what came before it:

1. **Work executes** (before any operators)
2. **`|> ensure { check }`** - runs if work succeeded
   - If check passes → continue chain
   - If check fails → continue to next operator (typically `rollback` or `catch`)
3. **`|> rollback { ... }`** - runs if anything before it failed
   - Catches work failures OR ensure failures
   - Executes compensation logic
   - **Error still propagates** (rollback doesn't recover)
4. **`|> catch { ... }`** - runs if anything before it failed
   - Handles the failure
   - **Recovers and continues** (error does not propagate)
5. **`|> finally { ... }`** - always runs
   - Executes regardless of success or failure
   - Used for cleanup (temp files, locks, notifications)

#### Standalone Operator Behavior

```opal
// Just work - errors if fails
kubectl apply -f k8s/app-v2.yaml

// Work + rollback - rollback runs if work fails, error propagates
kubectl apply -f k8s/app-v2.yaml
  |> rollback { kubectl rollout undo }

// Work + ensure - check runs if work succeeds, errors if check fails
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }

// Work + ensure + rollback - rollback runs if ensure fails, error propagates
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }

// Work + ensure + catch - catch runs if ensure fails, recovers and continues
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> catch { 
    kubectl rollout undo
    echo "Rolled back, continuing..."
  }

// Work + finally - cleanup always runs
kubectl apply -f k8s/app-v2.yaml
  |> finally { rm -f /tmp/deploy.lock }
```

#### Core Restrictions

**Restriction 1: Operators must be chained left-to-right**

```opal
// ❌ FORBIDDEN: out of order
kubectl apply -f k8s/
  |> catch { echo "error" }
  |> ensure { curl /health }  // ensure after catch doesn't make sense

// ✅ CORRECT: left-to-right
kubectl apply -f k8s/
  |> ensure { curl /health }
  |> catch { echo "error" }
```

**Restriction 2: Only one `catch` per chain**

```opal
// ❌ FORBIDDEN: multiple catch blocks
kubectl apply -f k8s/
  |> catch { echo "error 1" }
  |> catch { echo "error 2" }  // Error: multiple catch blocks

// ✅ CORRECT: single catch block
kubectl apply -f k8s/
  |> catch { 
    echo "error 1"
    echo "error 2"
  }
```

**Restriction 3: `finally` must be last**

```opal
// ❌ FORBIDDEN: finally not last
kubectl apply -f k8s/
  |> finally { rm -f /tmp/lock }
  |> catch { echo "error" }  // Error: finally must be last

// ✅ CORRECT: finally is last
kubectl apply -f k8s/
  |> catch { echo "error" }
  |> finally { rm -f /tmp/lock }
```

#### LIFO Unwind for Multi-Step Workflows

**Core mechanism:** `|> rollback` registers a compensator on a per-scope stack. Any unrecovered failure triggers a stack unroll (LIFO).

**Registration rules:**
1. Each scope maintains a `CompStack` (compensation stack)
2. After a frame **succeeds** (work ok + all `ensure` ok), every attached `rollback` is **pushed** to `CompStack`
3. On failure: run local handlers, then **unwind `CompStack` LIFO**
4. Recovery: any `catch` may absorb the error; if recovered, **do not unwind** outer stack

**Example:**

```opal
// Frame 1: Database migration
psql -f migrations/003-add-users.sql
  |> ensure { psql -c "SELECT COUNT(*) FROM users;" }
  |> rollback { psql -f migrations/003-rollback.sql }
// Success: rollback registered on CompStack

// Frame 2: Deploy app v2
kubectl apply -f k8s/app-v2.yaml
kubectl rollout status deployment/app
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }
// Success: rollback registered on CompStack

// Frame 3: Route traffic to v2
@lb.route_to("app-v2")
sleep 5
  |> ensure { curl -fsS http://lb/health }
  |> rollback { @lb.route_to("app-v1") }
// Failure: ensure fails

// Execution on Frame 3 failure:
// 1. Run Frame 3 rollback: @lb.route_to("app-v1")
// 2. UNWIND CompStack (LIFO):
//    - Pop and run Frame 2 rollback: kubectl rollout undo
//    - Pop and run Frame 1 rollback: psql -f migrations/003-rollback.sql
// 3. Propagate error (script exits with failure)
```

#### Using `catch` to recover and continue

```opal
// Try deploying v2, fall back to v1 if it fails, continue either way
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> catch { 
    kubectl rollout undo
    echo "Rolled back to v1"
  }

// This still runs because catch recovered
echo "Deployment complete"
```

#### Combining operators

```opal
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }
  |> catch { echo "Handled deployment failure" }
  |> finally { rm -f /tmp/deploy.lock }

// Execution on ensure failure:
// 1. ensure fails
// 2. rollback runs (compensate)
// 3. catch runs (recover)
// 4. finally runs (cleanup)
// 5. Continue (success, because catch recovered)
```

## Rationale

### `defer` vs `ensure`/`rollback`

| Aspect | `defer` | `ensure`/`rollback` |
|--------|---------|-------------------|
| **Syntax** | Block-based | Operator-based |
| **Coupling** | Setup + cleanup together | Verification + compensation chained |
| **Complexity** | Simple cleanup | Complex verification chains |
| **Readability** | Immediate cleanup visible | Left-to-right flow |
| **Use case** | Resource cleanup | Deployment verification |

**Use `defer` when:**
- Cleanup is straightforward (delete resource, rollback, etc.)
- Setup and cleanup are tightly coupled
- You want cleanup visible immediately after creation

**Use `ensure`/`rollback` when:**
- Verification is complex (multiple checks, retries)
- Compensation depends on verification result
- You want left-to-right flow (work → verify → compensate)

### Why LIFO ordering?

Resources must be cleaned up in reverse order of creation. LIFO ensures this automatically:

```opal
// Creation order: namespace → configmap → secret
// Cleanup order: secret → configmap → namespace (reverse)
```

### Why separate `defer` and `ensure`/`rollback`?

**`defer` is simpler** for straightforward cleanup. **`ensure`/`rollback` is more powerful** for complex verification chains. Both solve the same problem but with different ergonomics.

## Alternatives Considered

### Alternative 1: Only `defer`, no `ensure`/`rollback`

**Rejected:** `defer` doesn't handle complex verification chains well. You'd need nested try/catch blocks.

### Alternative 2: Only `ensure`/`rollback`, no `defer`

**Rejected:** `defer` is simpler for straightforward cleanup. Forcing operator syntax for simple cases is verbose.

### Alternative 3: Automatic cleanup without explicit registration

**Rejected:** Implicit cleanup is confusing. Explicit registration makes cleanup visible and testable.

## Implementation

### Phase 1: Parser & AST
- Add `DeferStatement` AST node
- Add `EnsureOperator`, `RollbackOperator`, `CatchOperator`, `FinallyOperator` nodes
- Parse `defer { ... }` syntax
- Parse `|> ensure { ... }`, `|> rollback { ... }`, `|> catch { ... }`, `|> finally { ... }` syntax
- Validate operator ordering (ensure before rollback, catch before finally, etc.)
- Validate no conditional deferral

### Phase 2: Planner
- Track defer registrations in plan
- Track operator chains in plan
- Represent LIFO unwind in plan
- Include in plan hash

### Phase 3: Executor
- Implement defer stack management
- Implement LIFO unwind on failure
- Implement operator chain execution
- Handle error propagation and recovery

### Phase 4: Integration
- Integrate with `@retry` decorator
- Integrate with `try/catch` blocks
- Integrate with `@parallel` branches
- LSP support for defer/operator syntax

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Error aggregation:** If multiple defers fail, how should errors be aggregated?
2. **Defer parameters:** Should defers have access to error information? (e.g., `defer.with_error { err => ... }`)
3. **Operator parameters:** Should `ensure` accept retry parameters? (e.g., `|> ensure(attempts=3, delay=5s) { ... }`)
4. **Nested operators:** Can operators be nested inside other operators?
5. **Performance:** Should we optimize LIFO unwind for large defer stacks?

## References

- **Go's defer statement:** Similar LIFO semantics for cleanup
- **Rust's Drop trait:** Automatic cleanup on scope exit
- **Elixir's pipe operator:** Inspiration for `|>` syntax
- **Related OEPs:**
  - OEP-001: Runtime Variable Binding with `let` (uses defer for error-aware cleanup)
  - OEP-002: Transform Pipeline Operator `|>` (uses assertions to trigger cleanup)
