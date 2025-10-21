---
oep: 013
title: Observability & Telemetry Hooks
status: Draft
type: Feature
created: 2025-01-21
updated: 2025-01-21
---

# OEP-013: Observability & Telemetry Hooks

## Summary

Comprehensive observability system with run tracking, OpenTelemetry integration, and plan/execution correlation. Provides run IDs, plan hashes, trace IDs, and artifacts for debugging, compliance, and audit trails. See [OBSERVABILITY.md](../OBSERVABILITY.md) for detailed design.

**Note:** Advanced features (cloud storage, compliance reporting, security audit, cross-run analysis) may be part of **Opal Cloud [PREMIUM]** offering. Core observability (local runs, basic tracking) will remain open-source.

## Motivation

### The Problem

Production deployments need:
- Post-execution debugging
- Audit trails for compliance
- Performance analysis
- Failure correlation
- Security monitoring

**Example of what's needed:**

```bash
# List recent prod runs
opal runs list --env prod --target deploy --since 48h

# Show run details
opal runs show run-2025-09-20T10:22:31Z-3f2a

# Compare failed vs successful run
opal runs diff run-fail run-ok

# Audit decorator usage
opal runs audit-decorators --env prod --since 24h
```

### Use Cases

**1. Post-execution debugging:**
```bash
# Show run details (status, critical path, failing step)
opal runs show run-2025-09-20T10:22:31Z-3f2a

# Open visual timeline
opal runs open run-…

# Stream failing step logs
opal runs tail run-… --step main/verify/healthz
```

**2. Plan/execution correlation:**
```bash
# Filter by plan (approved artifact → all executions)
opal runs list --plan-hash 5f6c…

# Verify execution integrity
opal runs verify run-… --cert prod.pem
```

**3. Security audit:**
```bash
# Track sensitive data access
opal runs audit-security --env prod --since 24h
# Shows: @env.SECRET_*, @var.PROD_*, @file.*.key, @shell("sudo *")

# Complete decorator audit
opal runs audit-decorators --env prod --since 24h
# Shows: @env.AWS_SECRET_KEY at deploy.opl:23, @shell("kubectl apply") at deploy.opl:45
```

**4. Compliance reporting:**
```bash
# Export compliance reports
opal runs report --env prod --format sox --month 2025-01

# Complete audit trail
opal runs export-audit --env prod --format soc2 --month 2025-01
```

## Proposal

### Run Identification

#### Run ID Format

```
run-<yyyyMMdd-HHmmss>-<shortsha>
```

**Example:** `run-2025-09-20T10:22:31Z-3f2a`

#### Plan Hash

```
sha256 of resolved plan
```

**Example:** `sha256:5f6c...`

### Artifacts Per Run

```
/runs/<env>/<target>/<plan-hash>/<run-id>/
├── plan.json            # Resolved plan (redacted)
├── otlp-traces.json     # OpenTelemetry trace, one span per step
├── summary.json         # Status, durations, env, runner, exit codes
└── report.html          # Optional: single-file timeline
```

### Storage Strategy

**Local (Open Source):** Local filesystem storage with SQLite index

**Cloud (Premium):** S3/GCS/Azure storage with managed index

**Retention:** Configurable by environment (prod=90d, dev=7d)

**Note:** Cloud storage and managed retention may be part of **Opal Cloud [PREMIUM]**.

### OpenTelemetry Integration

#### Trace Mapping

- **trace_id**: `hash(plan-hash + env + target)`
- **span**: Each decorator/step (`kind=INTERNAL`)
- **attributes**: `opal.env`, `opal.target`, `opal.step_path`, `exit_code`, `runner.id`
- **events**: `retry {n, exit_code}`, `stderr_tail`
- **status**: OK/ERROR with message

#### Trace Correlation

The `plan-hash` in trace_id enables:
- Verify which reviewed plan actually executed
- Correlate runs with same plan across environments
- Detect when plan changed between runs

### CLI Commands

#### List and Filter

```bash
# Recent prod runs
opal runs list --env prod --target deploy --since 48h

# Filter by plan (approved artifact → all executions)
opal runs list --plan-hash 5f6c…

# Grep for recurring issues
opal runs grep --env prod --like "rollout status" --since 7d
```

#### Inspect and Debug

```bash
# Show run details (status, critical path, failing step)
opal runs show run-2025-09-20T10:22:31Z-3f2a

# Open visual timeline
opal runs open run-…

# Compare runs (or last success vs failure)
opal runs diff run-A run-B

# Stream failing step logs
opal runs tail run-… --step main/verify/healthz
```

#### Security and Compliance

```bash
# Audit trail
opal runs audit --env prod --user alice --since 24h

# Verify execution integrity
opal runs verify run-… --cert prod.pem

# Export compliance reports
opal runs report --env prod --format sox --month 2025-01
```

### Decorator Usage Tracking

**Note:** Advanced security and compliance auditing may be part of **Opal Cloud [PREMIUM]**.

#### Complete Decorator Audit

```bash
# Complete decorator audit - see all decorator usage
opal runs audit-decorators --env prod --since 24h
# Shows: @env.AWS_SECRET_KEY at deploy.opl:23, @shell("kubectl apply") at deploy.opl:45

# Security-focused audit - track sensitive data access [PREMIUM]
opal runs audit-security --env prod --since 24h
# Shows: @env.SECRET_*, @var.PROD_*, @file.*.key, @shell("sudo *")

# Performance audit - find bottlenecks [PREMIUM]
opal runs audit-performance --env prod --since 24h --slowest 10
# Shows: @shell("kubectl rollout status") avg 5.4s, @http("healthcheck") avg 2.1s

# Compliance reporting - complete audit trail [PREMIUM]
opal runs export-audit --env prod --format soc2 --month 2025-01
# Generates: Complete decorator usage for SOC2/ISO27001 compliance
```

#### Built-in Security & Audit Features

- **Complete decorator tracking**: Every `@env.NAME`, `@var.NAME`, `@file.NAME`, `@shell()`, `@http()` call logged
- **Security pattern detection**: Automatic flagging of sensitive decorator usage
- **Access pattern analysis**: Detect unusual decorator combinations or frequencies
- **Performance bottleneck identification**: Find slow decorators across all runs
- **Zero-configuration audit**: Complete audit trail with no additional setup
- **Anomaly detection**: Flag scripts with unexpected decorator usage patterns

### Data Structures

#### Summary JSON

```json
{
  "run_id": "run-2025-09-20T10:22:31Z-3f2a",
  "plan_hash": "5f6c…",
  "target": "deploy",
  "env": "prod",
  "started_at": "2025-09-20T10:22:31Z",
  "ended_at": "2025-09-20T10:24:12Z",
  "status": "failed",
  "failed_step": "main/verify/healthz",
  "retries": 2,
  "runner": "runner-12",
  "git": {
    "sha": "abc123",
    "branch": "release/1.2.0"
  },
  "policy": {
    "deny_network": false,
    "readonly_fs": true
  }
}
```

#### Plan JSON (Redacted)

```json
{
  "version": "1.0",
  "plan_hash": "5f6c…",
  "resolved_at": "2025-09-20T10:22:30Z",
  "steps": [
    {
      "id": "main/deploy/api",
      "command": "kubectl apply -f k8s/api/",
      "decorator": "shell",
      "estimated_duration": "15s"
    }
  ],
  "resolved_values": {
    "1": "@env.API_TOKEN → <redacted:32chars>",
    "2": "@var.REPLICAS → 3"
  }
}
```

#### Decorator Telemetry Data

```json
{
  "decorator_usage": [
    {
      "decorator": "@env",
      "parameter": "AWS_SECRET_ACCESS_KEY",
      "script": "deploy.opl",
      "line": 23,
      "timestamp": "2025-09-20T14:32:15Z",
      "duration_ms": 0.1,
      "user": "deploy-bot",
      "runner": "runner-12"
    },
    {
      "decorator": "@shell",
      "parameter": "kubectl apply -f k8s/",
      "script": "deploy.opl",
      "line": 45,
      "timestamp": "2025-09-20T14:32:18Z",
      "duration_ms": 1250,
      "exit_code": 0,
      "stdout_lines": 15
    },
    {
      "decorator": "@http",
      "parameter": "https://api.internal/health",
      "script": "verify.opl",
      "line": 12,
      "timestamp": "2025-09-20T14:32:20Z",
      "duration_ms": 180,
      "status_code": 200
    }
  ]
}
```

### Core Restrictions

#### Restriction 1: Run IDs are immutable

Once created, run IDs cannot be changed:

```bash
# ❌ FORBIDDEN: changing run ID
# Run IDs are generated automatically

# ✅ CORRECT: use generated run ID
opal runs show run-2025-09-20T10:22:31Z-3f2a
```

**Why?** Immutability ensures audit trail integrity.

#### Restriction 2: Artifacts are append-only

Artifacts cannot be modified or deleted:

```bash
# ❌ FORBIDDEN: modifying artifacts
rm /runs/prod/deploy/5f6c…/run-…/summary.json

# ✅ CORRECT: artifacts are immutable
# Only create new runs
```

**Why?** Compliance. Audit trails must be tamper-proof.

#### Restriction 3: Secrets must be redacted

Secrets must never appear in artifacts:

```json
// ❌ FORBIDDEN: secret in artifact
"resolved_values": {
  "1": "@env.API_TOKEN → sk-abc123..."
}

// ✅ CORRECT: secret redacted
"resolved_values": {
  "1": "@env.API_TOKEN → <redacted:32chars>"
}
```

**Why?** Security. Artifacts may be stored or transmitted.

#### Restriction 4: Plan hash must match execution

Plan hash in artifacts must match actual execution:

```bash
# ❌ FORBIDDEN: plan hash mismatch
# Execution with different plan than recorded

# ✅ CORRECT: plan hash matches
opal runs verify run-… --cert prod.pem
# ✓ Plan hash matches execution
```

**Why?** Contract verification. Ensures reviewed plan executed.

### Storage Considerations

#### Local Development

```
~/.opal/runs/
├── runs.db
└── dev/
    └── deploy/
        └── abc123/
            └── run-2025-01-15-143022-def456/
```

#### Production

```
s3://company-opal-runs/
├── runs.db.gz          # Periodic snapshots
└── prod/
    └── deploy/
        └── 5f6c…/
            └── run-2025-01-15-143022-3f2a/
```

#### Retention Policies

- **Development**: 7 days, local storage
- **Staging**: 30 days, cloud storage
- **Production**: 90 days, cloud storage with backup

## Rationale

### Why run IDs?

**Uniqueness:** Every execution has a unique identifier.

**Traceability:** Link artifacts to specific executions.

**Debugging:** Easy to reference specific runs.

### Why plan hashes?

**Verification:** Proves reviewed plan matched execution.

**Correlation:** Group executions by plan.

**Compliance:** Audit trail shows what was planned vs executed.

### Why OpenTelemetry?

**Standard:** Industry standard for observability.

**Ecosystem:** Works with all major observability platforms.

**Flexibility:** Supports traces, metrics, and logs.

### Why decorator tracking?

**Security:** Track sensitive data access.

**Performance:** Find bottlenecks.

**Compliance:** Complete audit trail.

**Debugging:** Understand what decorators were used.

## Alternatives Considered

### Alternative 1: No observability

**Rejected:** Observability is critical for production systems.

### Alternative 2: Custom observability only

**Rejected:** OpenTelemetry is the standard. Should integrate with it.

### Alternative 3: No decorator tracking

**Rejected:** Decorator tracking is important for security and compliance.

### Alternative 4: No plan/execution correlation

**Rejected:** Correlation is important for compliance and debugging.

## Implementation

### Phase 1: Core Tracking (Open Source)
- Run IDs and plan hashes
- Basic summary.json output
- Local file storage
- SQLite indexing

### Phase 2: Query Interface (Open Source)
- `opal runs` CLI commands
- HTML report generation
- Basic filtering and search

### Phase 3: Observability Integration (Open Source)
- OpenTelemetry span emission
- Live trace correlation
- External monitoring hooks

### Phase 4: Advanced Features (Opal Cloud [PREMIUM])
- Cloud storage (S3/GCS/Azure)
- Cross-run analysis
- Compliance reporting (SOC2, ISO27001, HIPAA)
- Security audit and anomaly detection
- Predictive failure detection
- Managed retention policies
- Team collaboration features

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Retention:** Should retention policies be configurable per-target?
2. **Storage:** Should we support multiple storage backends (S3, GCS, Azure)?
3. **Encryption:** Should artifacts be encrypted at rest?
4. **Compression:** Should artifacts be compressed?
5. **Sampling:** Should we support sampling for high-volume environments?
6. **Alerting:** Should we support alerting on observability events?
7. **Export:** What export formats should we support (CSV, JSON, Parquet)?
8. **Privacy:** How should we handle PII in decorator tracking?

## References

- **OBSERVABILITY.md:** Detailed observability design
- **OpenTelemetry:** https://opentelemetry.io/
- **W3C Trace Context:** https://www.w3.org/TR/trace-context/
- **Compliance frameworks:** HIPAA, SOC 2, PCI-DSS, GDPR
- **Related OEPs:**
  - OEP-004: Plan Verification (plan hash correlation)
  - OEP-012: Module Composition (plugin observability hooks)
