---
oep: 009
title: Terraform/Pulumi Provider Bridge
status: Draft
type: Integration
created: 2025-01-21
updated: 2025-01-21
---

# OEP-009: Terraform/Pulumi Provider Bridge

## Summary

Add a bridge to Terraform and OpenTofu providers, enabling Opal to leverage the vast ecosystem of infrastructure providers. Schema import and codegen create Opal decorators from provider definitions.

## Motivation

### The Problem

Opal has limited provider ecosystem:
- Only built-in decorators available
- Can't use Terraform providers
- Can't use Pulumi providers
- Requires writing custom decorators

### Use Cases

**1. AWS provider:**
```bash
# Import AWS provider schema
opal provider add hashicorp/aws

# Use generated decorators
opal> var ami = @aws.ami.lookup(filters={...})
opal> @aws.s3.bucket.ensure(name="my-bucket", ...)
```

**2. Kubernetes provider:**
```bash
# Import Kubernetes provider schema
opal provider add kubernetes/kubernetes

# Use generated decorators
opal> @k8s.deployment.ensure(manifest="app.yaml")
opal> @k8s.service.get(name="app")
```

**3. Custom providers:**
```bash
# Import custom provider
opal provider add company/custom

# Use generated decorators
opal> @company.resource.create(...)
```

## Proposal

### Schema Import

#### Import Provider Schema

```bash
# Import from Terraform registry
opal provider add hashicorp/aws

# Import from local schema
opal provider add ./aws-schema.json

# Import from OpenTofu
opal provider add opentofu/aws
```

#### Schema Processing

1. Download provider schema
2. Parse schema (JSON)
3. Generate Opal decorators
4. Create plugin manifest
5. Register decorators

### Decorator Generation

#### Data Sources → Value Decorators

```opal
# From aws_ami data source
var ami = @aws.ami.lookup(
    filters={
        name="ubuntu/images/hvm-ssd/ubuntu-*",
        architecture="x86_64"
    },
    owners=["099720109477"]
)

echo "AMI ID: @var.ami.id"
```

#### Resources → Execution Decorators

```opal
# From aws_s3_bucket resource
@aws.s3.bucket.ensure(
    name="my-app-bucket",
    region="us-east-1",
    versioning=true,
    tags={env: "prod", team: "platform"}
)
```

### Type Mapping

| Terraform Type | Opal Type |
|---|---|
| string | String |
| number | Number |
| bool | Bool |
| list(T) | List<T> |
| map(T) | Map<String, T> |
| object({...}) | Struct |
| set(T) | Set<T> |

### Core Restrictions

#### Restriction 1: Providers are stateless

Opal queries reality each run, not state file:

```bash
# ❌ FORBIDDEN: relying on state file
opal deploy
# Error: state file not found

# ✅ CORRECT: query reality
opal deploy
# Queries current AWS state, applies changes
```

**Why?** Opal is stateless. Providers manage state internally.

#### Restriction 2: Generated decorators are read-only

Cannot modify generated decorators:

```bash
# ❌ FORBIDDEN: modifying generated decorator
# Edit ~/.opal/providers/aws/decorators.opl

# ✅ CORRECT: use generated decorators as-is
@aws.s3.bucket.ensure(...)
```

**Why?** Decorators are generated from schema. Modifications would be lost on regeneration.

#### Restriction 3: Provider versions are pinned

Provider versions must be explicitly pinned:

```bash
# ❌ FORBIDDEN: no version
opal provider add hashicorp/aws

# ✅ CORRECT: pin version
opal provider add hashicorp/aws@5.0.0
```

**Why?** Reproducibility. Different versions may have different schemas.

## Rationale

### Why bridge providers?

**Ecosystem:** Leverage existing provider ecosystem.

**Compatibility:** Use Terraform/Pulumi providers directly.

**Reach:** Enables Opal to work with any infrastructure.

### Why stateless?

**Simplicity:** No state file management.

**Determinism:** Query reality each run.

**Safety:** No state file corruption.

## Alternatives Considered

### Alternative 1: Write custom providers

**Rejected:** Requires writing providers for each service. Bridge is more efficient.

### Alternative 2: Use Terraform directly

**Rejected:** Opal has better ergonomics for procedural workflows.

### Alternative 3: No provider ecosystem

**Rejected:** Limits Opal's reach and usefulness.

## Implementation

### Phase 1: Proof of Concept
- Schema import for AWS provider
- Generate 2 value decorators
- Generate 2 execution decorators
- Basic gRPC adapter

### Phase 2: MVP
- Full AWS provider coverage
- Plugin manifest generation
- LSP integration
- Plan integration

### Phase 3: Production Ready
- Kubernetes provider
- Error handling & recovery
- Performance optimization
- Documentation

### Phase 4: Ecosystem
- Support for all major providers
- Provider registry & versioning
- Community contributions

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Provider registry:** Should we maintain a registry of available providers?
2. **Custom providers:** Should users be able to create custom providers?
3. **Provider versioning:** How should we handle provider version conflicts?
4. **Performance:** How should we optimize provider calls?
5. **Caching:** Should we cache provider schemas?

## References

- **Pulumi TF bridge:** Inspiration for provider bridging
- **Terraform provider protocol:** gRPC-based provider protocol
- **Related OEPs:**
  - OEP-006: LSP/IDE Integration (IDE support for generated decorators)
  - OEP-004: Plan Verification (plan integration with providers)
