---
title: "Opal Observability"
audience: "Operators & DevOps Engineers"
summary: "Tracing, artifacts, and debugging for production deployments"
---

# Opal Observability

**Making "what ran in prod" trivial**

**Audience**: Operators, DevOps engineers, and SREs running Opal in production.

The dual-end story is the clincher: **safe to review before** and **observable after**.

## Design Goal

Answer "what ran, where it got to, and what changed" in seconds when prod is spicy.

## Integration Points

Opal's observability model connects directly to the architecture:

- **Run ID**: Unique identifier for each execution
- **Plan Hash**: From [ARCHITECTURE.md](ARCHITECTURE.md) contract verification - ensures reviewed plan matches execution
- **Trace IDs**: Map to plan steps for debugging
- **Artifacts**: Resolved plans, traces, and summaries stored per run

**When to use:**
- **Ops**: Post-execution debugging, audit trails, compliance
- **Dev**: Understanding execution flow, performance analysis

## Minimal Observability Design

### Run Identification
- **Run ID**: `run-<yyyyMMdd-HHmmss>-<shortsha>`
- **Plan Hash**: sha256 of resolved plan

### Artifacts Per Run
```
/runs/<env>/<target>/<plan-hash>/<run-id>/
├── plan.json            # Resolved plan (redacted)
├── otlp-traces.json     # OpenTelemetry trace, one span per step
├── summary.json         # Status, durations, env, runner, exit codes
└── report.html          # Optional: single-file timeline
```

### Storage Strategy
- **Object store**: S3/GCS/local for artifacts
- **Index**: Tiny SQLite (`runs.db`) for fast queries
- **Retention**: Configurable by environment (prod=90d, dev=7d)

## CLI You Can Ship

### List and Filter
```bash
# Recent prod runs
opal runs list --env prod --target deploy --since 48h

# Filter by plan (approved artifact → all executions)
opal runs list --plan-hash 5f6c…

# Grep for recurring issues
opal runs grep --env prod --like "rollout status" --since 7d
```

### Inspect and Debug
```bash
# Show run details (status, critical path, failing step)
opal runs show run-2025-09-20T10:22:31Z-3f2a

# Open visual timeline
opal runs open run-…            # Launches report.html

# Compare runs (or last success vs failure)  
opal runs diff run-A run-B

# Stream failing step logs
opal runs tail run-… --step main/verify/healthz
```

## OpenTelemetry Integration

### Trace Mapping

- **trace_id**: `hash(plan-hash + env + target)`
- **span**: Each decorator/step (`kind=INTERNAL`)
- **attributes**: `opal.env`, `opal.target`, `opal.step_path`, `exit_code`, `runner.id`
- **events**: `retry {n, exit_code}`, `stderr_tail`
- **status**: OK/ERROR with message

**Trace correlation**: The `plan-hash` in trace_id enables:
- Verify which reviewed plan actually executed
- Correlate runs with same plan across environments
- Detect when plan changed between runs

### Benefits
- View live in Jaeger/Tempo **and** keep offline HTML per run
- Correlate deployment spans with application traces (same trace_id)
- Standard observability tooling integration
- Link traces back to reviewed execution contracts

## Typical Production Incident Workflow

### 1. Triage
```bash
opal runs list --env prod --target deploy --since 24h
```

### 2. Investigate
```bash
opal runs show run-…     # See failing step + attempt history
opal runs open run-…     # Visual timeline
```

### 3. Correlate
- Click from failing span to app traces/logs (same trace_id)
- Cross-reference with monitoring alerts

### 4. Compare
```bash
opal runs diff run-fail run-ok    # Shows changed steps/args/env
```

### 5. Verify Safety
```bash
opal verify plan.json --sig ...   # Confirm approved plan executed
```

## Data Structures

### Summary JSON
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

### Plan JSON (Redacted)
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

## Small But Mighty Extras

### Automatic Context
- **Policy flags**: `deny_network`, `readonly_fs` recorded in summary
- **Change tracking**: Git SHA, branch, ticket numbers
- **Environment tags**: Deploy windows, maintenance modes

### Advanced Debugging
```bash
# Stream logs for specific step
opal runs tail run-… --step main/deploy/api

# Find patterns across runs  
opal runs analyze --env prod --pattern "timeout" --since 30d

# Export for external analysis
opal runs export --env prod --format csv --since 7d
```

### Security and Compliance
```bash
# Audit trail
opal runs audit --env prod --user alice --since 24h

# Verify execution integrity
opal runs verify run-… --cert prod.pem

# Export compliance reports
opal runs report --env prod --format sox --month 2025-01
```

### Decorator Usage Tracking & Security Audit
opal automatically tracks all value decorator and execution decorator usage for comprehensive security and performance audit:

```bash
# Complete decorator audit - see all value decorator and execution decorator usage
opal runs audit-decorators --env prod --since 24h
# Shows: @env.AWS_SECRET_KEY at deploy.opl:23, @shell("kubectl apply") at deploy.opl:45

# Security-focused audit - track sensitive data access  
opal runs audit-security --env prod --since 24h
# Shows: @env.SECRET_*, @var.PROD_*, @file.*.key, @shell("sudo *")

# Performance audit - find bottlenecks
opal runs audit-performance --env prod --since 24h --slowest 10
# Shows: @shell("kubectl rollout status") avg 5.4s, @http("healthcheck") avg 2.1s

# Compliance reporting - complete audit trail
opal runs export-audit --env prod --format soc2 --month 2025-01
# Generates: Complete decorator usage for SOC2/ISO27001 compliance
```

#### Built-in Security & Audit Features
- **Complete decorator tracking**: Every `@env.NAME`, `@var.NAME`, `@file.NAME`, `@shell()`, `@http()` value decorator and execution decorator call logged
- **Security pattern detection**: Automatic flagging of sensitive value decorator usage
- **Access pattern analysis**: Detect unusual value decorator and execution decorator combinations or frequencies  
- **Performance bottleneck identification**: Find slow value decorators and execution decorators across all runs
- **Zero-configuration audit**: Complete audit trail with no additional setup
- **Anomaly detection**: Flag scripts with unexpected value decorator and execution decorator usage patterns

#### Decorator Telemetry Data
The telemetry system captures comprehensive value decorator and execution decorator usage:
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

#### Security Use Cases
- **Secret sprawl detection**: "Which scripts access production secrets?"
- **Privilege escalation monitoring**: "Unusual @shell('sudo') usage detected"
- **Data access audit**: "Complete trail of @file() access to sensitive configs"
- **Network activity tracking**: "All @http() calls with URLs and response codes"
- **Compliance reporting**: "Generate quarterly audit of all decorator usage"

## Implementation Priority

### Phase 1: Core Tracking
- Run IDs and plan hashes
- Basic summary.json output
- Simple file storage

### Phase 2: Query Interface  
- SQLite indexing
- `opal runs` CLI commands
- HTML report generation

### Phase 3: Observability Integration
- OpenTelemetry span emission
- Live trace correlation
- External monitoring hooks

### Phase 4: Advanced Features
- Cross-run analysis
- Compliance reporting
- Predictive failure detection

## Storage Considerations

### Local Development
```
~/.opal/runs/
├── runs.db
└── dev/
    └── deploy/
        └── abc123/
            └── run-2025-01-15-143022-def456/
```

### Production
```
s3://company-opal-runs/
├── runs.db.gz          # Periodic snapshots
└── prod/
    └── deploy/
        └── 5f6c…/
            └── run-2025-01-15-143022-3f2a/
```

### Retention Policies
- **Development**: 7 days, local storage
- **Staging**: 30 days, cloud storage  
- **Production**: 90 days, cloud storage with backup

## Why This Matters

This observability design transforms opal from a deployment tool into a **deployment intelligence platform**:

- **Before**: Plan review and approval workflows
- **During**: Live execution monitoring and debugging
- **After**: Historical analysis and compliance reporting

The result: Operations teams can confidently deploy and quickly resolve issues when they occur, with complete audit trails for compliance and learning.