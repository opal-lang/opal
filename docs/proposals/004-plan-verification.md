---
oep: 004
title: Plan Verification
status: Draft
type: Feature
created: 2025-01-21
updated: 2025-01-21
---

# OEP-004: Plan Verification

## Summary

Add comprehensive plan verification capabilities for audit trails, compliance reporting, and CI/CD workflows. Enable generation, review, and execution of deterministic plans with hash-based contract verification.

## Motivation

### The Problem

Current Opal generates plans but doesn't provide strong verification guarantees:
- No audit trail of what was planned vs executed
- No way to review plans before execution in CI/CD
- No differential analysis between plan versions
- No compliance reporting for regulated environments

**Example of current limitations:**

```bash
# ❌ No way to verify plan before execution
opal deploy("prod")
# Executes immediately, no review step

# ❌ No differential analysis
opal plan deploy("prod") > plan-v1.txt
opal plan deploy("prod") > plan-v2.txt
# No built-in way to compare plans
```

### Use Cases

**1. CI/CD workflow with plan review:**
```bash
# Generate plan for review
opal plan deploy("prod") > plan.txt

# Human reviews plan.txt

# Execute exact plan
opal execute plan.txt
```

**2. Compliance reporting:**
```bash
# Generate plan with audit trail
opal plan deploy("prod") --audit > plan.json

# Extract compliance information
jq '.audit.resources_modified' plan.json
jq '.audit.network_changes' plan.json
```

**3. Differential analysis:**
```bash
# Compare two plans
opal diff plan-v1.txt plan-v2.txt
# Shows what changed between versions
```

**4. Contract verification:**
```bash
# Generate plan with hash
opal plan deploy("prod") --hash > plan.txt
# Plan Hash: sha256:5f6c...

# Later, verify execution matches plan
opal deploy("prod") --plan plan.txt
# Replans from current state, verifies hash matches, then executes
```

## Proposal

### Plan Generation

#### Basic Plan Generation

```bash
# Generate plan and display
opal plan deploy("prod")

# Generate plan and save
opal plan deploy("prod") > plan.txt

# Generate plan with hash
opal plan deploy("prod") --hash
# Plan Hash: sha256:5f6c...

# Generate resolved plan (with all values expanded)
opal plan deploy("prod") --resolve
```

#### Plan Output Format

**Quick plan (default):**
```
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/
  2. kubectl scale --replicas=3 deployment/app
  3. curl /health |> assert.re("Status 200")
```

**Resolved plan (--resolve):**
```
Plan: a3b2c1d4
Resolved: sha256:5f6c...
  1. kubectl apply -f k8s/prod/
     └─ Resolved: /home/user/k8s/prod/
  2. kubectl scale --replicas=3 deployment/app
     └─ Resolved: replicas=3
  3. curl /health |> assert.re("Status 200")
     └─ Resolved: http://prod.example.com/health
```

### Plan Verification

#### Hash-Based Contract Verification

Plans have deterministic hashes:

```bash
# Generate plan with hash
opal plan deploy("prod") --hash
# Plan Hash: sha256:5f6c...

# Later, verify execution matches plan
opal deploy("prod") --plan plan.txt
# Replans from current state, verifies hash matches, then executes
```

**Verification process:**
1. Read plan from file (includes hash)
2. Replan from current state
3. Compare new plan hash with stored hash
4. If hashes match: execute plan
5. If hashes don't match: error (plan changed)

#### Audit Trail

Track what was planned vs executed:

```bash
# Generate plan with audit trail
opal plan deploy("prod") --audit > plan.json

# Plan includes:
# - Timestamp
# - User who generated plan
# - Environment variables used
# - Resources that will be modified
# - Network changes
# - File modifications
```

**Audit trail format:**
```json
{
  "plan_id": "a3b2c1d4",
  "plan_hash": "sha256:5f6c...",
  "timestamp": "2025-01-21T10:30:00Z",
  "user": "alice@example.com",
  "environment": {
    "ENV": "prod",
    "REGION": "us-west-2"
  },
  "audit": {
    "resources_created": ["deployment/app", "service/app"],
    "resources_modified": ["configmap/app-config"],
    "resources_deleted": [],
    "network_changes": [
      {"type": "ingress", "port": 443, "protocol": "https"}
    ],
    "file_modifications": [
      {"path": "/etc/app/config.yaml", "action": "create"}
    ]
  }
}
```

### Plan Comparison

#### Differential Analysis

Compare two plans to see what changed:

```bash
# Generate two plans
opal plan deploy("prod") > plan-v1.txt
# ... make changes to script ...
opal plan deploy("prod") > plan-v2.txt

# Compare plans
opal diff plan-v1.txt plan-v2.txt
```

**Diff output:**
```
Plan Diff: v1 → v2

Added steps:
  + kubectl apply -f k8s/new-service.yaml

Removed steps:
  - kubectl delete deployment/old-app

Modified steps:
  ~ kubectl scale --replicas=3 deployment/app
    → kubectl scale --replicas=5 deployment/app

Unchanged steps:
  = kubectl apply -f k8s/prod/
  = curl /health |> assert.re("Status 200")
```

#### Plan Signature

Sign plans for trust verification:

```bash
# Generate and sign plan
opal plan deploy("prod") --sign > plan.txt
# Signature: sha256:a1b2c3... (signed with private key)

# Verify plan signature
opal verify plan.txt
# ✓ Signature valid (signed by alice@example.com)

# Execute verified plan
opal deploy("prod") --plan plan.txt
```

### CI/CD Integration

#### Workflow: Plan Review Before Execution

```bash
#!/bin/bash
# 1. Generate plan
opal plan deploy("prod") --hash > plan.txt

# 2. Commit plan for review
git add plan.txt
git commit -m "Deploy plan for review"

# 3. Wait for approval (via PR review)
# ... human reviews plan.txt ...

# 4. Execute approved plan
opal deploy("prod") --plan plan.txt
```

#### Workflow: Automated Compliance Check

```bash
#!/bin/bash
# 1. Generate plan with audit trail
opal plan deploy("prod") --audit > plan.json

# 2. Check compliance
if jq '.audit.resources_deleted | length > 0' plan.json | grep -q true; then
    echo "ERROR: Plan deletes resources"
    exit 1
fi

# 3. Execute plan
opal deploy("prod") --plan plan.json
```

#### Workflow: Canary Deployment with Plan Verification

```bash
#!/bin/bash
# 1. Generate canary plan
opal plan canary_deploy("prod") --hash > canary-plan.txt

# 2. Execute canary plan
opal canary_deploy("prod") --plan canary-plan.txt

# 3. Monitor canary
sleep 60

# 4. If canary succeeds, generate production plan
if [ $? -eq 0 ]; then
    opal plan prod_deploy("prod") --hash > prod-plan.txt
    
    # 5. Execute production plan
    opal prod_deploy("prod") --plan prod-plan.txt
fi
```

### Core Restrictions

#### Restriction 1: Plans are immutable

Once a plan is generated, it cannot be modified:

```bash
# ❌ FORBIDDEN: modifying plan file
echo "kubectl delete deployment/app" >> plan.txt

# ✅ CORRECT: generate new plan
opal plan deploy("prod") > plan-v2.txt
```

**Why?** Plan hash would be invalid if plan is modified.

#### Restriction 2: Plan execution must match plan

When executing with `--plan`, the execution must match the plan exactly:

```bash
# ❌ FORBIDDEN: executing different plan
opal deploy("prod") --plan plan-v1.txt
# But script has changed since plan-v1.txt was generated

# ✅ CORRECT: regenerate plan if script changes
opal plan deploy("prod") > plan-v2.txt
opal deploy("prod") --plan plan-v2.txt
```

**Why?** Contract verification requires exact match.

#### Restriction 3: Audit trails are read-only

Audit trails cannot be modified after generation:

```bash
# ❌ FORBIDDEN: modifying audit trail
jq '.audit.resources_created = []' plan.json > plan-modified.json

# ✅ CORRECT: generate new audit trail
opal plan deploy("prod") --audit > plan-v2.json
```

**Why?** Audit trails are for compliance and cannot be tampered with.

## Rationale

### Why hash-based verification?

**Determinism:** Plans are deterministic. Same script + same environment = same plan hash.

**Trust:** Community can vouch for plan hashes. "I reviewed this plan and it's safe."

**Auditability:** Plan hash is part of execution record. Can verify what was executed.

### Why differential analysis?

**Safety:** See exactly what changed between plan versions before executing.

**Compliance:** Track changes for audit trails.

**Debugging:** Understand why plan changed.

### Why audit trails?

**Compliance:** Regulated environments require audit trails.

**Accountability:** Track who generated plans and when.

**Debugging:** Understand what resources will be affected.

## Alternatives Considered

### Alternative 1: No plan verification, just execute

**Rejected:** No audit trail, no compliance support, no way to review plans before execution.

### Alternative 2: Cryptographic signatures only

**Rejected:** Hashes are simpler and sufficient for most use cases. Signatures can be added later.

### Alternative 3: Centralized plan registry

**Rejected:** Adds complexity. Plans should be stored in version control with code.

## Implementation

### Phase 1: Plan Generation
- Add `--hash` flag to generate plan hash
- Add `--resolve` flag to generate resolved plan
- Add `--audit` flag to generate audit trail
- Implement plan output formatting

### Phase 2: Plan Verification
- Implement hash verification
- Implement plan comparison (diff)
- Implement `--plan` flag for execution with plan verification

### Phase 3: Audit Trail
- Implement audit trail generation
- Track resources created/modified/deleted
- Track network changes
- Track file modifications

### Phase 4: CI/CD Integration
- Add plan signing support
- Add plan verification in CI/CD
- Documentation and examples

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Plan storage:** Should plans be stored in version control or a separate registry?
2. **Plan expiration:** Should plans expire after a certain time?
3. **Plan versioning:** How should we version plans?
4. **Signature algorithms:** Which cryptographic algorithms should we support?
5. **Audit retention:** How long should audit trails be retained?

## References

- **Terraform plan verification:** Similar concept for infrastructure code
- **Git commit hashing:** Inspiration for deterministic hashing
- **Compliance frameworks:** HIPAA, SOC 2, PCI-DSS audit requirements
- **Related OEPs:**
  - OEP-001: Runtime Variable Binding with `let` (affects plan hash)
  - OEP-002: Transform Pipeline Operator `|>` (affects plan hash)
  - OEP-003: Automatic Cleanup and Rollback (affects plan hash)
