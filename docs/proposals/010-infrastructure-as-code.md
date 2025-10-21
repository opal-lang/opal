---
oep: 010
title: Infrastructure as Code (IaC)
status: Draft
type: Integration
created: 2025-01-21
updated: 2025-01-21
---

# OEP-010: Infrastructure as Code (IaC)

## Summary

Add outcome-focused infrastructure provisioning with deploy blocks (run on first creation) and SSH blocks (run always). Flexible idempotence matching enables pragmatic infrastructure management without state files.

## Motivation

### The Problem

Traditional IaC (Terraform, Pulumi) requires managing state:
- State files can get corrupted
- State files must be backed up
- State files must be shared across team
- Drift detection is complex

### Use Cases

**1. Outcome-focused deployment:**
```opal
deploy: {
    var db = @aws.rds.deploy(
        name="app-db",
        engine="postgres"
    ) {
        # Runs ONLY on first creation
        psql -c "CREATE DATABASE app;"
    }
    
    var web = @aws.instance.deploy(
        name="web-server",
        type="t3.medium"
    ) {
        # Runs ONLY on first creation
        apt-get update
        apt-get install -y nginx
    }
    
    # Always runs (every execution)
    @aws.instance.ssh(instance=@var.web) {
        docker pull myapp:@var.VERSION
        docker run -d --name myapp myapp:@var.VERSION
    }
}
```

**2. Flexible idempotence matching:**
```opal
# Name-only matching (pragmatic)
var web = @aws.instance.deploy(
    name="web-server",
    type="t3.medium",
    idempotenceKey=["name"]
)
# Found instance with different type? Use it anyway

# Semantic matching
var db = @aws.rds.deploy(
    name="app-db",
    engine="postgres",
    version="14",
    idempotenceKey=["name", "engine", "version"]
)
# Found with different version? Different resource

# Strict matching
var web = @aws.instance.deploy(
    name="prod-web",
    type="t3.medium",
    idempotenceKey=["name", "type"],
    onMismatch="error"
)
# Found with different type? Error
```

## Proposal

### Deploy Blocks

Deploy blocks run only on first creation:

```opal
var instance = @aws.instance.deploy(
    name="web-server",
    type="t3.medium"
) {
    # Runs ONLY on first creation
    apt-get update
    apt-get install -y nginx
    systemctl enable nginx
}

# First run: Creates instance, runs block
# Second run: Instance exists, block skipped
```

### SSH Blocks

SSH blocks run every time:

```opal
@aws.instance.ssh(instance=@var.instance) {
    # Runs EVERY time
    systemctl restart nginx
    docker pull myapp:latest
    docker run -d -p 80:3000 myapp:latest
}

# First run: Runs after deploy block
# Second run: Runs immediately
# Every run: Same operational work
```

### Idempotence Matching

#### Name-Only Matching (Pragmatic)

```opal
var web = @aws.instance.deploy(
    name="web-server",
    type="t3.medium",
    idempotenceKey=["name"]
)

# Matching logic:
# - Found instance with name="web-server"? Use it
# - Type is t3.large instead of t3.medium? Don't care, use it
# - Someone manually changed it? Don't care, use it
```

#### Semantic Matching

```opal
var db = @aws.rds.deploy(
    name="app-db",
    engine="postgres",
    version="14",
    idempotenceKey=["name", "engine", "version"]
)

# Matching logic:
# - name="app-db", engine="postgres", version="14"? Use it
# - name="app-db", engine="postgres", version="15"? Different resource
# - name="app-db", engine="mysql"? Different resource
```

#### Strict Matching

```opal
var web = @aws.instance.deploy(
    name="prod-web",
    type="t3.medium",
    idempotenceKey=["name", "type"],
    onMismatch="error"
)

# Matching logic:
# - name="prod-web", type="t3.medium"? Use it
# - name="prod-web", type="t3.large"? Error: type mismatch
```

### Mismatch Handling

```opal
# Warn but use it anyway (default)
@aws.instance.deploy(
    name="web",
    type="t3.medium",
    onMismatch="warn"
)
# Found t3.large → WARNING: Expected t3.medium, found t3.large

# Fail on mismatch (strict)
@aws.instance.deploy(
    name="web",
    type="t3.medium",
    onMismatch="error"
)
# Found t3.large → ERROR: Instance type mismatch

# Ignore differences silently
@aws.instance.deploy(
    name="web",
    type="t3.medium",
    onMismatch="ignore"
)
# Found t3.large → Uses it, no warnings

# Create new if mismatch
@aws.instance.deploy(
    name="web",
    type="t3.medium",
    onMismatch="create"
)
# Found t3.large → Creates "web-2" with t3.medium
```

## Rationale

### Why outcome-focused?

**Pragmatism:** Focus on outcomes, not perfect state.

**Simplicity:** No state file management.

**Flexibility:** Different matching strategies for different needs.

### Why deploy blocks?

**Clarity:** Initialization logic is visible and explicit.

**Safety:** Initialization only runs once.

**Debugging:** Easy to see what happens on first creation.

### Why flexible idempotence?

**Pragmatism:** Different resources need different matching.

**Safety:** Can be strict when needed.

**Flexibility:** Can be pragmatic when appropriate.

## Alternatives Considered

### Alternative 1: Strict state management (like Terraform)

**Rejected:** Adds complexity. Opal is stateless.

### Alternative 2: No idempotence matching

**Rejected:** Would require manual state tracking.

### Alternative 3: Only name-based matching

**Rejected:** Too inflexible for complex scenarios.

## Implementation

### Phase 1: Deploy Blocks
- Implement deploy block syntax
- Implement first-creation detection
- Implement block execution

### Phase 2: Idempotence Matching
- Implement idempotenceKey parameter
- Implement matching logic
- Implement mismatch handling

### Phase 3: Integration
- Integrate with providers
- Documentation and examples

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **State detection:** How should we detect if a resource already exists?
2. **Drift detection:** Should we detect and report drift?
3. **Reconciliation:** Should we automatically reconcile drift?
4. **Versioning:** How should we handle resource version changes?
5. **Cleanup:** How should we handle resource cleanup?

## References

- **Terraform:** State-based IaC (contrast)
- **Pulumi:** Procedural IaC (inspiration)
- **Related OEPs:**
  - OEP-009: Terraform/Pulumi Provider Bridge (provider integration)
  - OEP-003: Automatic Cleanup and Rollback (cleanup on failure)
