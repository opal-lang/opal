---
oep: 008
title: Plan-First Execution Model
status: Draft
type: Tooling
created: 2025-01-21
updated: 2025-01-21
---

# OEP-008: Plan-First Execution Model

## Summary

Formalize and document the plan-first execution model as a core feature. Plans are generated before execution, enabling review, verification, and safe remote code execution.

## Motivation

### The Problem

Current Opal supports plans but doesn't formalize the model:
- No clear semantics for plan generation
- No standard workflow for plan review
- No trust model for remote code

### Use Cases

**1. Safe remote code execution:**
```bash
# Download script from internet
curl https://example.com/deploy.opl > deploy.opl

# Generate plan for review
opal plan deploy("prod") > plan.txt

# Security team reviews plan.txt
# ✓ Approved

# Execute plan
opal execute plan.txt
```

**2. REPL plan mode:**
```bash
opal> plan deploy("prod")
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/
  2. kubectl scale --replicas=3 deployment/app

Execute? [y/N] y
```

**3. Hash-based trust:**
```bash
# Community vouches for plan hash
# "I reviewed this plan and it's safe"
# Plan Hash: sha256:5f6c...

# Verify you're running the same plan
opal plan deploy("prod") --hash
# Plan Hash: sha256:5f6c...
# ✓ Matches community review
```

## Proposal

### Plan Generation

Plans are generated before execution:

```bash
# Generate plan
opal plan deploy("prod")

# Plan shows what will happen
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/
  2. kubectl scale --replicas=3 deployment/app
  3. curl /health |> assert.re("Status 200")
```

### Plan Execution

Plans can be executed directly:

```bash
# Execute plan
opal execute plan.txt

# Or with verification
opal execute plan.txt --verify
```

### Plan Review Workflow

**Step 1: Generate plan**
```bash
opal plan deploy("prod") > plan.txt
```

**Step 2: Review plan**
```bash
# Human reviews plan.txt
# Checks:
# - What resources will be created/modified/deleted?
# - Are there any dangerous operations?
# - Does the plan match expectations?
```

**Step 3: Approve plan**
```bash
# If approved, commit to version control
git add plan.txt
git commit -m "Approved deployment plan"
```

**Step 4: Execute plan**
```bash
# Execute approved plan
opal execute plan.txt
```

### Trust Model

**Hash-based trust:**
- Plans have deterministic hashes
- Community can vouch for plan hashes
- Verify you're running the same plan others reviewed

**Signature-based trust:**
- Plans can be signed with private keys
- Verify plans are from trusted authors
- Prevent tampering

## Rationale

### Why plan-first?

**Safety:** Review before execution.

**Auditability:** Track what was planned vs executed.

**Reproducibility:** Same plan always produces same result.

**Trust:** Community can vouch for plans.

## Alternatives Considered

### Alternative 1: Execute immediately

**Rejected:** No review step, no safety.

### Alternative 2: No plan generation

**Rejected:** Can't review before execution.

## Implementation

### Phase 1: Formalize Model
- Document plan generation semantics
- Document plan execution semantics
- Document trust model

### Phase 2: Tooling
- `opal plan` command
- `opal execute` command
- Plan verification

### Phase 3: Workflow
- CI/CD integration
- Plan review workflow
- Community trust registry

## Compatibility

**Breaking changes:** None. This formalizes existing behavior.

**Migration path:** N/A (formalization, no code changes).

## Open Questions

1. **Plan format:** Should plans be JSON, YAML, or custom format?
2. **Plan versioning:** How should we version plans?
3. **Plan expiration:** Should plans expire after a certain time?
4. **Community registry:** Should we maintain a registry of trusted plan hashes?
5. **Offline verification:** Should plans be verifiable offline?

## References

- **Terraform plan/apply:** Inspiration for plan-first model
- **Git commit hashing:** Inspiration for deterministic hashing
- **Related OEPs:**
  - OEP-004: Plan Verification (detailed verification)
  - OEP-005: Interactive REPL (plan mode in REPL)
