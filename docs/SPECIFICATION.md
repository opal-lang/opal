# Devcmd Language Specification

**Write operations scripts that show you exactly what they'll do before they do it.**

## What is Devcmd?

Devcmd is an operations language for teams who want the reliability of infrastructure-as-code without the complexity of state files. Write scripts that feel like shell but generate auditable plans.

**Key principle**: Resolved plans are execution contracts that get verified before running.

## The Core Idea

Everything becomes a decorator internally. No special cases.

```devcmd
// You write natural syntax
deploy: {
    echo "Starting deployment"
    npm run build
    kubectl apply -f k8s/
}

// Parser converts to decorators
deploy: {
    @shell("echo \"Starting deployment\"")
    @shell("npm run build")
    @shell("kubectl apply -f k8s/")
}
```

## Two Ways to Run

**Command mode** - organized tasks:
```devcmd
// commands.cli
install: npm install
test: npm test
deploy: kubectl apply -f k8s/
```
```bash
devcmd deploy    # Run specific task
```

**Script mode** - direct execution:
```devcmd
#!/usr/bin/env devcmd
echo "Deploying version $VERSION"
kubectl apply -f k8s/
```
```bash
./deploy-script    # Execute directly
```

## Line-by-Line Execution

How commands connect matters:

```devcmd
// Newlines = fail-fast (stop on first failure)
deploy: {
    npm run build
    npm test
    kubectl apply -f k8s/
}

// Semicolons = keep going (shell behavior)
setup: npm install; npm run build; npm test

// Shell operators = standard behavior with pipefail
check: npm run build && npm test || echo "Build failed"
logs: kubectl logs app | grep ERROR
```

**Operator precedence**: `|` (pipe) > `&&`, `||` > `;` > newlines

## Variables

Pull values from real sources:

```devcmd
var ENV = @env("ENVIRONMENT", default="development")
var PORT = @env("PORT", default=3000)
var DEBUG = @env("DEBUG", default=false)
var TIMEOUT = @env("DEPLOY_TIMEOUT", default=30s)

// Arrays and maps
var SERVICES = ["api", "worker", "ui"]
var CONFIG = {
    "database": @env("DATABASE_URL"),
    "redis": @env("REDIS_URL", default="redis://localhost:6379")
}

// Go-style grouped declarations
var (
    API_URL = @env("API_URL", default="https://api.dev.com")
    REPLICAS = @env("REPLICAS", default=1)
    SERVICES = ["api", "worker"]
)
```

**Types**: `string | bool | int | duration | array | map`. Type errors caught at plan time.

### Accessing Data

Use dot notation for nested access:

```devcmd
// Array access
start-api: docker run -p @var(SERVICES.0):3000 app
start-worker: docker run -p @var(SERVICES.1):3001 app

// Map access  
connect: psql @var(CONFIG.database)
cache: redis-cli -u @var(CONFIG.redis)

// Wildcards expand to space-separated values
list-services: echo "Services: @var(SERVICES.*)"    # "api worker ui"

// All equivalent ways to access arrays
@var(SERVICES.0)    # Dot notation
@var(SERVICES[0])   # Bracket notation  
@var(SERVICES.[0])  # Mixed notation
```

## Control Flow

Control flow happens at **plan time**, creating deterministic execution sequences.

### Loops

```devcmd
deploy-all: {
    for service in @var(SERVICES) {
        echo "Deploying ${service}"
        kubectl apply -f k8s/${service}/
        kubectl rollout status deployment/${service}
    }
}

// Plan expands to independent steps:
// ├─ echo "Deploying api"
// ├─ kubectl apply -f k8s/api/
// ├─ kubectl rollout status deployment/api
// ├─ echo "Deploying worker"
// └─ ... (and so on)
```

**For loop guarantee**: Plan-time unrolling creates independent steps with stable IDs. Empty arrays produce no commands.

### Conditionals

```devcmd
deploy: {
    if @var(ENV) == "production" {
        kubectl apply -f k8s/prod/
        kubectl scale --replicas=3 deployment/app
    } else {
        kubectl apply -f k8s/dev/
        kubectl scale --replicas=1 deployment/app
    }
}
```

### Pattern Matching

```devcmd
deploy: {
    when @var(ENV) {
        "production" -> {
            kubectl apply -f k8s/prod/
            kubectl scale --replicas=3 deployment/app
        }
        "staging" -> kubectl apply -f k8s/staging/
        else -> echo "Unknown environment: @var(ENV)"
    }
}
```

**Pattern rules**:
- `"literal"` - exact match
- `"main" | "develop"` - any match (OR)
- `{"hotfix", "urgent"}` - any in set
- `r"^release/"` - regex (RE2 engine)
- `200..299` - inclusive range
- `else` - catch-all

**Evaluation**: First match wins, patterns evaluated in order.

### Error Handling

Try/catch is the **only** non-deterministic construct:

```devcmd
deploy: {
    try {
        kubectl apply -f k8s/
        kubectl rollout status deployment/app
        echo "Deployment successful"
    } catch {
        echo "Deployment failed, rolling back"
        kubectl rollout undo deployment/app
    } finally {
        echo "Cleaning up temporary resources"
        kubectl delete pod -l job=deployment-helper
    }
}
```

Plans show all possible paths. Execution logs show which path was taken.

**Control flow rules**:
- `for`, `if`, `when`, `try/catch` cannot be chained with operators
- They are complete statements, not expressions
- This eliminates complex precedence questions

## Decorators

### Value Decorators

Inject values inline:

```devcmd
// Environment variables
start: node app.js --port @env("PORT", default=3000)

// Devcmd variables  
scale: kubectl scale --replicas=@var(REPLICAS) deployment/app

// Expensive lookups (resolved lazily)
deploy: kubectl apply --token=@aws.secret("api-token")
config: curl -H "Authorization: @http.get('https://auth.com/token')"
```

### Execution Decorators

Enhance command execution:

```devcmd
// Retry with named parameters
deploy: @retry(attempts=3, delay=2s, backoff=1.5) {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
}

// Timeout protection
build: @timeout(5m) {
    npm run build
    npm test
}

// Parallel execution
services: @parallel {
    npm run api
    npm run worker
    npm run ui
}

// Command references
deploy: @cmd(build) && @cmd(test) && @cmd(apply)
```

## Plans: Execution Contracts

Devcmd turns operations into two-phase execution: plan generation, then verified execution.

**Plan provenance**: Plans include source_commit, spec_version, and compiler_version in headers for audit trails.

### Quick Plans

Fast preview without expensive operations:

```bash
devcmd deploy --dry-run
```
```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=¹@aws.secret("api-token")
└─ kubectl rollout status deployment/app

Deferred Values:
1. @aws.secret("api-token") → <expensive: AWS lookup>
```

### Resolved Plans

All values resolved, creating an execution contract:

```bash
devcmd deploy --dry-run --resolve > prod.plan
```
```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=¹<32:a1b2c3>
└─ kubectl rollout status deployment/app

Resolved Values:
1. @aws.secret("api-token") → <32:a1b2c3>
```

**Security by default**: All values appear as `<N:hashAlgo:hex>` format (e.g., `<32:sha256:a1b2c3d4>`).

> **Placeholder Format**  
> `<N:hashAlgo:hex>` where N=character count, hashAlgo=algorithm, hex=truncated hash.  
> Examples: `<32:sha256:a1b2c3>`, `<8:sha256:x7y8z9>`.  
> Future-proofs against algorithm changes and aids debugging.

**Plan hash scope**: Ordered steps + arguments (with `<len:hash>` placeholders) + operator graph + resolution timing flags; excludes ephemeral run IDs/logs.

> **Security Invariant**  
> Raw secrets never appear in plans or logs, only `<len:hash>` placeholders.  
> This applies to all value decorators: `@env()`, `@var()`, `@aws.secret()`, etc.  
> Compliance teams can review plans with confidence.

### Verified Execution

Resolved plans are **execution contracts** with verification:

```bash
# Execute against resolved plan
devcmd run --plan prod.plan
```

**What happens**:
1. **Replan** from source with current infrastructure state
2. **Verify** replanned structure matches resolved plan exactly
3. **Execute** the contracted plan if verification passes
4. **Fail** with diff if anything changed

**Verification outcomes**:
- `source_changed`: Source files modified → regenerate plan
- `infra_missing`: Expected infrastructure not found → use `--force` or fix infrastructure  
- `infra_mutated`: Infrastructure present but different → use `--force` or regenerate plan

**Example verification failure**:
```
ERROR: Plan verification failed

Expected: kubectl scale --replicas=<1:3> deployment/app
Actual:   kubectl scale --replicas=<1:5> deployment/app

Source file changed since plan generation.
Run 'devcmd plan --resolve deploy' to generate new plan.
```

**Partial execution**: Use `--from step:path` to resume from specific steps when verification passes (useful for long pipelines).

This ensures the resolved plan you reviewed is exactly what executes, even hours later.

## Resolution Strategy

**Value timing rules**:
- **Quick plans**: Expensive decorators deferred, show placeholders
- **Resolved plans**: ALL decorators execute, values frozen as execution contract
- **Verified execution**: Contract verification ensures resolved plan matches current source
- **Dead code elimination**: Decorators in pruned branches never execute

**Non-determinism guardrail**: Value decorators must be referentially transparent during `--resolve`. Non-deterministic decorators cause contract verification failures.

**Seeded determinism**: Operations requiring randomness or cryptography (like `@random.password()` or `@crypto.generate_key()`) use Plan Seed Envelopes (PSE) to be deterministic within resolved plans while maintaining security.

**Always planned**: Even direct script execution generates internal plans first, ensuring consistent behavior.

## Time-Delayed Execution

Real operations involve plan generation and execution at different times. Devcmd's verification model handles this cleanly.

### The Scenario

```bash
# 2:00 PM - Generate plan during change window planning
devcmd deploy --dry-run --resolve > evening-deploy.plan

# 5:00 PM - Execute plan during maintenance window  
devcmd run --plan evening-deploy.plan
```

### Verification at Execution

When you execute a resolved plan, devcmd **verifies the contract**:

1. **Replan** from source files with current infrastructure state
2. **Compare** new plan structure with resolved plan
3. **Verify** all resolved values still match (structure and hashes)
4. **Execute** contracted plan if verification passes
5. **Fail** with diff if source changed or infrastructure drifted

### What Gets Verified

**Source changes detected**:
```
ERROR: Plan verification failed

Expected: for service in ["api", "worker"] {
Actual:   for service in ["api", "worker", "ui"] {

Source file modified since plan generation.
```

**Infrastructure drift detected**:
```
ERROR: Infrastructure state changed

Expected: kubectl scale --replicas=<1:sha256:abc123> deployment/app  
Current:  No deployment/app found

Infrastructure changed since plan generation.
Consider regenerating plan or using --force.
```

**Non-deterministic decorator detected**:
```
ERROR: Contract verification failed

@http.get("https://time-api.com/now") returned different value:
  Plan time: <20:sha256:abc123>
  Execution:  <20:sha256:def456>

Non-deterministic decorators cannot be used in resolved plans.
Consider separating dynamic value acquisition from deterministic execution.
```

### Plan Seed Envelopes (PSE)

For operations requiring randomness, resolved plans contain a Plan Seed Envelope - a minimal, immutable piece of state that enables deterministic random generation:

```devcmd
// Default: regenerates on every new plan
var TEMP_TOKEN = @random.password(length=16)

// Stable across plan changes until regeneration key changes
var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v1")
var API_KEY = @crypto.generate_key(type="ed25519", regen_key="api-key-v3")

deploy: {
    kubectl create secret generic db --from-literal=password=@var(DB_PASS)
    kubectl create secret generic api --from-literal=key=@var(API_KEY)  
}

// Rotate secrets by changing regeneration key
rotate-secrets: {
    // Change v1 → v2 to get new password  
    var NEW_DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v2")
    kubectl patch secret db --patch='{"data":{"password":"'$(echo -n @var(NEW_DB_PASS) | base64)'"}}'
}
```

**Plan shows placeholders** (maintaining security invariant):
```
kubectl create secret generic db --from-literal=password=¹<24:sha256:abc123>
kubectl create secret generic api --from-literal=key=¹<64:sha256:def456>
```

**How PSE works**:
- High-entropy seed generated at `--resolve` time
- Seed encrypted and sealed to authorized runners (file-based key or KMS)
- Each decorator derives unique deterministic values using plan context
- Same plan always generates identical random values
- Different plans generate different values (new seed per plan)

**PSE vs traditional state**:
- **Scoped**: Each plan contains its own PSE, no shared state
- **Immutable**: Never changes after plan creation, no updates or migrations  
- **Self-contained**: Plan file includes everything needed, no external storage
- **Minimal**: Contains only cryptographic entropy, not infrastructure tracking
- **Contract-aligned**: Enables deterministic execution within resolved plans

This gives you secure, auditable randomness while maintaining the stateless execution model.

### Execution Options

**Strict verification** (default):
```bash
devcmd run --plan prod.plan
# Fails on any source or infrastructure changes
```

**Force execution**:
```bash
devcmd run --plan prod.plan --force
# Uses resolved plan values as targets, adapts to current infrastructure
```



### Why This Works

**Contract clarity**: Teams review resolved plans knowing exactly what will execute
**Change detection**: Any modifications between plan and execution are caught
**Audit trail**: Resolved plans become immutable execution records
**Flexibility**: Multiple execution modes for different operational needs

This verification model gives you the determinism of resolved plans with safety against unexpected changes.

## Parallel Resolution

Independent expensive operations resolve concurrently:

```devcmd
var CLUSTER = @env("KUBE_CLUSTER", default="minikube")
var DB_HOST = @aws.secret("${CLUSTER}-db-host")  
var API_KEY = @http.get("https://keygen.com/api/new")

deploy: {
    when @var(CLUSTER) {
        "production" -> {
            kubectl apply -f k8s/prod/ --server=@var(DB_HOST)
            kubectl create secret --api-key=@var(API_KEY)
        }
        else -> kubectl apply -f k8s/dev/
    }
}
```

**Resolution DAG**:
- `@env("KUBE_CLUSTER")` resolves first (needed for conditional)
- `@aws.secret()` and `@http.get()` resolve in parallel
- Unused expensive operations in dev branch never execute

## Change Detection and Idempotency

Scripts are **idempotent by design** with smart change detection:

```bash
# First run
devcmd deploy
# Creates resources, shows what was done

# Second run (no changes)
devcmd deploy  
# Shows "no-op" for existing resources

# Third run (environment changed)
REPLICAS=5 devcmd deploy
# Shows only the scale operation (replica count changed)
```

**How change detection works**:
- Value hashing: `<1:3>` → `<1:5>` indicates REPLICAS changed from 3 to 5
- Secret rotation: `<32:a1b>` → `<32:x7y>` indicates API token was rotated
- Infrastructure queries: Decorators check current state vs desired state
- Character count in hash gives size hints for debugging

**Resolved plan verification adds another layer**:
- Source changes detected by comparing plan structures
- Hash changes show which values modified
- Infrastructure drift caught by re-querying current state

## Future: Infrastructure Decorators

The decorator model extends naturally to infrastructure management:

```devcmd
// Future capabilities following same patterns
@aws.ec2.deploy(name="web-prod", count=3)
@k8s.apply(manifest="app.yaml")
@docker.build(tag="app:v1.2.3")
```

These will follow the same plan-first pattern without requiring state files - query current state at plan time, freeze inputs, execute deterministically. Execution decorators SHOULD expose idempotency keys so re-runs under the same contract are safe.

## Why This Design Works

**Contract-based**: Resolved plans are execution contracts with verification
**Auditable**: See exactly what will run, verify it matches before execution
**Secure**: No sensitive values in plans or logs, change detection without exposure
**Stateless**: No state files to corrupt - decorators query reality fresh each time
**Readable**: More natural than YAML, more structured than shell scripts
**Extensible**: New decorators integrate seamlessly

Devcmd transforms operations from "pray and deploy" to "plan, review, execute with verification" - bringing contract discipline to deployment workflows without traditional infrastructure tool complexity.

## Examples

### Web Application Deployment

```devcmd
var (
    ENV = @env("ENVIRONMENT", default="development")
    VERSION = @env("APP_VERSION", default="latest")
    REPLICAS = @env("REPLICAS", default=1)
    SERVICES = ["api", "worker", "ui"]
)

deploy: {
    echo "Deploying version @var(VERSION) to @var(ENV)"
    
    when @var(ENV) {
        "production" -> {
            for service in @var(SERVICES) {
                @retry(attempts=3, delay=10s) {
                    kubectl apply -f k8s/prod/${service}/
                    kubectl set image deployment/${service} app=@var(VERSION)
                    kubectl scale --replicas=@var(REPLICAS) deployment/${service}
                    kubectl rollout status deployment/${service} --timeout=300s
                }
            }
        }
        "staging" -> {
            @timeout(5m) {
                kubectl apply -f k8s/staging/
                kubectl set image deployment/app app=@var(VERSION)
                kubectl rollout status deployment/app
            }
        }
        else -> {
            echo "Deploying to development"
            kubectl apply -f k8s/dev/
        }
    }
    
    echo "Deployment complete"
}
```

### Database Migration with Rollback

```devcmd
var DB_URL = @env("DATABASE_URL")
var BACKUP_BUCKET = @env("BACKUP_BUCKET", default="db-backups")

migrate: {
    try {
        echo "Starting database migration"
        
        @timeout(30m) {
            # Backup first
            pg_dump @var(DB_URL) | gzip > backup-$(date +%Y%m%d-%H%M%S).sql.gz
            aws s3 cp backup-*.sql.gz s3://@var(BACKUP_BUCKET)/
            
            # Run migration
            @retry(attempts=2, delay=5s) {
                psql @var(DB_URL) -f migrations/001-add-users.sql
                psql @var(DB_URL) -f migrations/002-add-indexes.sql
            }
            
            # Verify
            psql @var(DB_URL) -c "SELECT COUNT(*) FROM users;"
        }
        
        echo "Migration completed successfully"
        
    } catch {
        echo "Migration failed, restoring from backup"
        
        # Find latest backup
        LATEST_BACKUP=$(aws s3 ls s3://@var(BACKUP_BUCKET)/ | tail -1 | awk '{print $4}')
        
        # Restore
        aws s3 cp s3://@var(BACKUP_BUCKET)/${LATEST_BACKUP} - | gunzip | psql @var(DB_URL)
        
        echo "Database restored from backup"
        
    } finally {
        # Cleanup local backup files
        rm -f backup-*.sql.gz
        echo "Migration process completed"
    }
}
```