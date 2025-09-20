# Devcmd Observability

**Making "what ran in prod" trivial**

The dual-end story is the clincher: **safe to review before** and **observable after**.

## Design Goal

Answer "what ran, where it got to, and what changed" in seconds when prod is spicy.

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
devcmd runs list --env prod --target deploy --since 48h

# Filter by plan (approved artifact → all executions)
devcmd runs list --plan-hash 5f6c…

# Grep for recurring issues
devcmd runs grep --env prod --like "rollout status" --since 7d
```

### Inspect and Debug
```bash
# Show run details (status, critical path, failing step)
devcmd runs show run-2025-09-20T10:22:31Z-3f2a

# Open visual timeline
devcmd runs open run-…            # Launches report.html

# Compare runs (or last success vs failure)  
devcmd runs diff run-A run-B

# Stream failing step logs
devcmd runs tail run-… --step main/verify/healthz
```

## OpenTelemetry Integration

### Trace Mapping
- **trace_id**: `hash(plan-hash + env + target)`
- **span**: Each decorator/step (`kind=INTERNAL`)
- **attributes**: `devcmd.env`, `devcmd.target`, `devcmd.step_path`, `exit_code`, `runner.id`
- **events**: `retry {n, exit_code}`, `stderr_tail`
- **status**: OK/ERROR with message

### Benefits
- View live in Jaeger/Tempo **and** keep offline HTML per run
- Correlate deployment spans with application traces (same trace_id)
- Standard observability tooling integration

## Typical Production Incident Workflow

### 1. Triage
```bash
devcmd runs list --env prod --target deploy --since 24h
```

### 2. Investigate
```bash
devcmd runs show run-…     # See failing step + attempt history
devcmd runs open run-…     # Visual timeline
```

### 3. Correlate
- Click from failing span to app traces/logs (same trace_id)
- Cross-reference with monitoring alerts

### 4. Compare
```bash
devcmd runs diff run-fail run-ok    # Shows changed steps/args/env
```

### 5. Verify Safety
```bash
devcmd verify plan.json --sig ...   # Confirm approved plan executed
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
    "1": "@env(API_TOKEN) → <redacted:32chars>",
    "2": "@var(REPLICAS) → 3"
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
devcmd runs tail run-… --step main/deploy/api

# Find patterns across runs  
devcmd runs analyze --env prod --pattern "timeout" --since 30d

# Export for external analysis
devcmd runs export --env prod --format csv --since 7d
```

### Security and Compliance
```bash
# Audit trail
devcmd runs audit --env prod --user alice --since 24h

# Verify execution integrity
devcmd runs verify run-… --cert prod.pem

# Export compliance reports
devcmd runs report --env prod --format sox --month 2025-01
```

## Implementation Priority

### Phase 1: Core Tracking
- Run IDs and plan hashes
- Basic summary.json output
- Simple file storage

### Phase 2: Query Interface  
- SQLite indexing
- `devcmd runs` CLI commands
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
~/.devcmd/runs/
├── runs.db
└── dev/
    └── deploy/
        └── abc123/
            └── run-2025-01-15-143022-def456/
```

### Production
```
s3://company-devcmd-runs/
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

This observability design transforms devcmd from a deployment tool into a **deployment intelligence platform**:

- **Before**: Plan review and approval workflows
- **During**: Live execution monitoring and debugging
- **After**: Historical analysis and compliance reporting

The result: Operations teams can confidently deploy and quickly resolve issues when they occur, with complete audit trails for compliance and learning.