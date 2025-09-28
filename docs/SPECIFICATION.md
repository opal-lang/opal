# Opal Language Specification

**Write operations scripts that show you exactly what they'll do before they do it.**

## What is Opal?

Opal is an operations language for teams who want the reliability of infrastructure-as-code without the complexity of state files. Write scripts that feel like shell but generate auditable plans.

**Key principle**: Resolved plans are execution contracts that get verified before running.

## The Core Idea

Everything becomes a value decorator or execution decorator internally. No special cases.

```opal
// You write natural syntax
deploy: {
    echo "Starting deployment"
    npm run build
    kubectl apply -f k8s/
}

// Parser converts to execution decorators
deploy: {
    @shell("echo \"Starting deployment\"")
    @shell("npm run build")
    @shell("kubectl apply -f k8s/")
}
```

## Two Ways to Run

**Command mode** - organized tasks:
```opal
// commands.opl
install: npm install
test: npm test
deploy: kubectl apply -f k8s/
```
```bash
opal deploy    # Run specific task
```

**Script mode** - direct execution:
```opal
#!/usr/bin/env opal
echo "Deploying version $VERSION"
kubectl apply -f k8s/
```
```bash
./deploy-script    # Execute directly
```

## Line-by-Line Execution

How commands connect matters:

```opal
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

```opal
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

**Types**: `string | bool | int | float | duration | array | map`. Type errors caught at plan time.

### Identifier Names

Variable names, command names, and value decorator and execution decorator names follow ASCII identifier rules for fast tokenization:

```opal
// Valid identifiers - start with letter, then letters/numbers/underscores
var apiUrl = "https://api.example.com"
var PORT = 3000
var service_name = "api-gateway"
var deployToProduction = true

// Command names follow same rules
deployToProduction: kubectl apply -f prod/
check_api_health: curl /health
buildAndTest: npm run build && npm test
```

**Identifier rules**:
- Must start with letter: `[a-zA-Z]`
- Can contain letters, numbers, underscores: `[a-zA-Z0-9_]*`
- Case-sensitive: `myVar` ≠ `MyVar` ≠ `MYVAR`
- No hyphens to avoid parsing ambiguity with minus operator
- ASCII-only for optimal performance

**Supported naming styles**:
- `camelCase` - common in JavaScript/Java
- `snake_case` - common in Python/Go
- `PascalCase` - common for types/commands  
- `SCREAMING_SNAKE` - common for constants

### Duration Format

Duration literals use human-readable time units common in operations:

```opal
// Simple durations
var TIMEOUT = 30s           // 30 seconds
var RETRY_DELAY = 5m        // 5 minutes  
var DEPLOY_WINDOW = 2h      // 2 hours
var RETENTION = 7d          // 7 days
var BACKUP_CYCLE = 1w       // 1 week
var LICENSE_EXPIRY = 2y     // 2 years

// Compound durations (high to low order)
var SESSION_TIMEOUT = 1h30m     // 1 hour 30 minutes
var MAINTENANCE_WINDOW = 2d12h  // 2 days 12 hours
var GRACE_PERIOD = 5m30s        // 5 minutes 30 seconds
var API_TIMEOUT = 1s500ms       // 1 second 500 milliseconds
```

**Supported units** (in descending order):
- `y` - years
- `w` - weeks
- `d` - days  
- `h` - hours
- `m` - minutes
- `s` - seconds
- `ms` - milliseconds
- `us` - microseconds  
- `ns` - nanoseconds

**Compound duration rules**:
- Must be in descending order: `1h30m` ✓, `30m1h` ✗
- No duplicate units: `1h30m` ✓, `1h2h` ✗
- No whitespace within: `1h30m` ✓, `1h 30m` = two separate durations
- Can skip units: `1d30m` ✓ (skipping hours is fine)
- Integer values only: `1h30m` ✓, `1.5h` ✗ (use compound format for precision)

**Duration arithmetic**:
```opal
// Duration arithmetic with other durations
var total = 1h30m + 45m        // total = 2h15m
var remaining = 5m - 2m30s     // remaining = 2m30s
var scaled = 30s * 3           // scaled = 1m30s

// Variables can hold negative durations
var grace = 1m - 5m            // grace = -4m (preserved for logic)
var timeout = 30s - 1h         // timeout = -29m30s (preserved for logic)

// Conditional logic with negative durations
if grace < 0s {
    echo "No grace period remaining"
    exit 1
} else {
    @timeout(grace) { deploy() }
}
```

**Duration execution rules**:
```opal
// Runtime functions clamp negative durations to zero
@timeout(-4m) { cmd }          // Executes with 0s timeout
@retry(attempts=3, delay=-30s) { cmd }  // Uses 0s delay
sleep(-1h)                     // Sleeps for 0s (no-op)

// Variables preserve negative values for arithmetic/logic
var remaining = deadline - current_time
echo "Time remaining: ${remaining}"     // Shows "-5m" if past deadline
@timeout(remaining) { task() }          // Uses max(remaining, 0s) = 0s
```

**Duration evaluation rules**:
- All duration arithmetic evaluated at plan time with resolved values
- Variables can store negative durations for conditional logic
- Runtime functions automatically clamp negative durations to zero
- Duration literals are always non-negative (`30s`, `1h30m`)
- Negative expressions use minus operator: `-30s` = `MINUS` + `DURATION`

### Accessing Data

Use dot notation for nested access:

```opal
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

## Arithmetic and Assignment

Opal supports arithmetic operations for deterministic calculations in operations scripts. All arithmetic is evaluated at plan time, ensuring predictable results.

### Basic Arithmetic

```opal
// Deterministic calculations for operations
var total_replicas = base_replicas * environments
var batch_size = total_items / worker_count
var shard_id = item_id % shard_count
var timeout = base_timeout + (retry_attempt * backoff_multiplier)

// Memory and resource calculations
var total_memory = service_memory * replica_count
var disk_space = data_size + (log_retention * daily_logs)
var cpu_limit = base_cpu + (load_factor * peak_multiplier)
```

**Supported operators**:
- `+` - addition
- `-` - subtraction  
- `*` - multiplication
- `/` - division
- `%` - modulo (remainder)

**Operator precedence** (highest to lowest):
1. `*`, `/`, `%` (multiplication, division, modulo)
2. `+`, `-` (addition, subtraction)
3. Use parentheses `()` for explicit ordering

### Assignment Operators

```opal
// Accumulation in deterministic loops
var total_cost = 0
for service in @var(SERVICES) {
    total_cost += @var(SERVICE_COSTS[service])
}

// Resource scaling
var replicas = 1
replicas *= @var(ENVIRONMENTS).length  // multiply by environment count
replicas += 1                          // add monitoring replica

// Batch processing
var remaining = total_items
for batch in batches {
    remaining -= batch.size
    echo "Processing batch, ${remaining} items left"
}
```

**Supported assignment operators**:
- `+=` - add and assign
- `-=` - subtract and assign
- `*=` - multiply and assign  
- `/=` - divide and assign
- `%=` - modulo and assign

### Increment and Decrement

```opal
// Counting in deterministic loops
var counter = 0
for service in @var(SERVICES) {
    counter++
    echo "Processing service ${counter}: ${service}"
}

// Countdown operations
var attempts = max_retries
while attempts > 0 {
    attempts--
    echo "Retry attempt ${max_retries - attempts}"
}
```

**Supported operators**:
- `++` - increment by 1
- `--` - decrement by 1

### Deterministic Evaluation

All arithmetic operations are evaluated at plan time with known values:

```opal
// Plan shows exact calculations
var replicas = 3 * 2 + 1    // Plan: replicas = 7
var timeout = 30 + (2 * 5)  // Plan: timeout = 40

// Variable resolution then calculation
var base = @env("BASE_REPLICAS", default=2)  // Resolves to 2
var total = base * 3                         // Plan: total = 6
```

**Type rules**:
- `int` + `int` = `int`
- `float` + `float` = `float`  
- `int` + `float` = `float` (automatic promotion)
- `duration` + `duration` = `duration`
- Division by zero detected at plan time

## Control Flow

Control flow happens at **plan time**, creating deterministic execution sequences.

### Block Phases and Deterministic Execution

Every `{ ... }` block in opal represents a **phase** - a unit of execution with strong deterministic guarantees. When you write a block, the planner expands it at plan time into a finite, ordered sequence of steps that will execute in a predictable way.

**Phase boundaries create execution order.** Each phase completes entirely before the next phase begins. This means all steps within a phase finish before any step in a subsequent phase can start. Within a phase, steps execute according to their canonical order - the order they appear after plan-time expansion.

**Variable mutations follow block semantics.** Most blocks (`for`, `if`, `when`, command blocks) allow variable mutations to affect the outer scope, since their execution is deterministic at plan time. However, `try/catch` blocks and execution decorator blocks use scope isolation to maintain predictable behavior (detailed below).

**Plans are verifiable contracts.** The resolved plan captures everything needed to verify execution: which steps run in what order, what commands they execute (with placeholders for resolved values), and how they handle retries or timeouts. If any of this changes between plan and execution, verification fails.

**Outputs are deterministically merged.** Each step produces its own stdout and stderr streams. When these need to be combined (for logging or display), they're merged in the canonical order, ensuring the same plan always produces the same combined output.

Block-specific constructs like `for`, `if`, `when`, `try/catch`, and `@parallel` work within this framework. They define how blocks expand (unrolling loops, selecting branches) or add constraints (parallel independence), but they all inherit the same phase execution guarantees.

### Loops

```opal
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

For loops unroll at plan time into a known number of steps. The collection (`@var(SERVICES)`) is resolved during planning, and each item creates a separate step in the canonical order. Empty collections produce zero steps.

### Conditionals

```opal
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

Conditionals are evaluated at plan time using resolved variable values. Only the taken branch expands into steps - the other branch becomes dead code that doesn't appear in the final plan.

### Pattern Matching

```opal
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

Pattern matching uses first-match-wins evaluation at plan time. Supported patterns include exact strings (`"production"`), OR expressions (`"main" | "develop"`), sets (`{"hotfix", "urgent"}`), regex patterns (`r"^release/"`), numeric ranges (`200..299`), and catch-all (`else`). Only the matching branch expands into the plan.

### Error Handling

Try/catch is the **only** non-deterministic construct:

```opal
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

The plan records all possible execution paths through try/catch blocks. At runtime, only one of `try` or `catch` executes, while `finally` always runs. Execution logs show which path was actually taken.

## Scope Isolation

The rule is simple: **values can flow in from the outer scope, but mutations never flow back out**.

```opal
var counter = 0
var status = "pending"

try {
    // Can READ outer scope values
    echo "Starting with counter=${counter}"  // counter = 0 ✓
    
    // Can MODIFY local copies
    counter++           // Local counter = 1
    status = "running"  // Local status = "running"
    
    kubectl apply -f k8s/
} catch {
    // Can READ outer scope values  
    echo "Failed with counter=${counter}"    // counter = 0 ✓
    
    // Can MODIFY local copies
    counter += 5        // Local counter = 5
    status = "failed"   // Local status = "failed"
}

// Outer scope unchanged after try/catch
echo "Final: counter=${counter}, status=${status}"  // counter=0, status="pending" ✓
```

**Decorator blocks work the same way:**

```opal
var attempts = 0
var result = ""

@retry(attempts=3) {
    // Can READ outer scope
    echo "Base attempts: ${attempts}"  // attempts = 0 ✓
    
    // Can MODIFY local copies
    attempts++         // Local attempts = 1, 2, 3...
    result = "done"    // Local result = "done"
    
    kubectl apply -f k8s/
}

// Outer scope unchanged after execution decorator
echo "Final: attempts=${attempts}, result=${result}"  // attempts=0, result="" ✓
```

This pattern ensures that both non-deterministic execution (try/catch) and execution decorator behaviors don't create unpredictable state mutations in the outer scope.

## Decorators

### Value Decorators

Inject values inline:

```opal
// Environment variables
start: node app.js --port @env("PORT", default=3000)

// Opal variables  
scale: kubectl scale --replicas=@var(REPLICAS) deployment/app

// Expensive lookups (resolved lazily)
deploy: kubectl apply --token=@aws.secret("api-token")
config: curl -H "Authorization: @http.get('https://auth.com/token')"
```

### Execution Decorators

Enhance command execution:

```opal
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

## Plans: Three Execution Modes

Opal provides three distinct planning and execution modes, each serving different operational needs.

**Plan provenance**: Plans include source_commit, spec_version, and compiler_version in headers for audit trails.

### Quick Plans (`--dry-run`)

**Purpose**: Fast preview of likely execution paths without expensive operations.

```bash
opal deploy --dry-run
```

**What happens**:
- Resolves control flow (if/when/for conditions) using cheap value decorators
- Shows all possible execution branches that code could take
- Defers expensive value decorators (`@aws.secret`, `@http.get`, etc.)
- Calculates execution paths based on current variable values

```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=¹@aws.secret("api-token")
└─ kubectl rollout status deployment/app

Possible Branches:
├─ if ENV == "production" → [kubectl scale --replicas=3]
└─ else → [kubectl scale --replicas=1]

Deferred Values:
1. @aws.secret("api-token") → <expensive: AWS lookup>
```

### Resolved Plans (`--dry-run --resolve`)

**Purpose**: Complete execution contract with all values resolved.

```bash
opal deploy --dry-run --resolve > prod.plan
```

**What happens**:
- Resolves ALL value decorators (including expensive ones)
- Determines exact execution path (no branches, single success path)
- Creates deterministic execution contract
- Generates plan file for later contract-verified execution

```
deploy:
├─ kubectl apply -f k8s/
├─ kubectl create secret --token=¹<32:a1b2c3>
├─ kubectl scale --replicas=<1:3> deployment/app
└─ kubectl rollout status deployment/app

Resolved Values:
1. @aws.secret("api-token") → <32:a1b2c3>

Contract Hash: sha256:def456...
```

### Execution Plans (always happens)

**Purpose**: Runtime resolution and execution.

```bash
# Direct execution
opal deploy

# Contract-verified execution  
opal run --plan prod.plan
```

**What happens**:
1. **Internal resolution**: Resolves all value decorators fresh at execution time
2. **Path determination**: Follows single execution path based on resolved values
3. **Contract verification** (if using plan file): Ensures resolved values match contract hashes
4. **Execution**: Runs commands with internally resolved values

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

## Contract Verification

When using plan files, opal ensures execution matches the reviewed contract exactly.

### Contract-Verified Execution

```bash
# Execute against resolved plan contract
opal run --plan prod.plan
```

**Contract verification process**:
1. **Fresh resolution**: Resolve all value decorators with current infrastructure state
2. **Hash comparison**: Compare newly resolved value hashes with plan contract hashes  
3. **Path verification**: Ensure execution path matches contracted plan structure
4. **Execute or bail**: Run contracted plan if hashes match, otherwise fail with diff

**Why this works**: The resolved plan contains `<length:hash>` placeholders. At execution time, opal resolves values fresh and compares their hashes. If `@env("REPLICAS")` was `3` during planning but is now `5`, the hashes won't match.

### Verification Outcomes

**✅ Contract verified** - hashes match, execution proceeds:
```
✓ Contract verified: all value decorators resolve to expected hashes
→ Executing contracted plan...
```

**❌ Contract violated** - hashes differ, execution stops:
```
ERROR: Contract verification failed

Expected: kubectl scale --replicas=<1:sha256:abc123> deployment/app  
Actual:   kubectl scale --replicas=<1:sha256:def456> deployment/app

Variable REPLICAS changed: was 3, now 5
→ Source or environment changed since plan generation
→ Run 'opal deploy --dry-run --resolve' to generate new plan
```

**Contract violation causes**:
- `source_changed`: Source files modified since plan generation
- `env_changed`: Environment variables modified since plan generation  
- `infra_drift`: Infrastructure state changed since plan generation

### Direct Execution (No Contract)

```bash
# Always resolves fresh, no contract verification
opal deploy
```

**Direct execution process**:
1. **Fresh resolution**: Resolve all value decorators with current state
2. **Path determination**: Follow execution path based on resolved values
3. **Execute**: Run commands with resolved values (no hash verification)

This mode is ideal for development and immediate execution where you want current values, not contracted values.

## Planning Mode Summary

| Mode | Command | Value Resolution | Use Case |
|------|---------|------------------|----------|
| **Quick Plan** | `--dry-run` | Cheap values only | Fast preview, see possible paths |
| **Resolved Plan** | `--dry-run --resolve` | ALL values resolved | Complete contract, audit review |
| **Direct Execution** | `deploy` | Fresh resolution | Development, immediate execution |
| **Contract Execution** | `run --plan file` | Fresh + hash verification | Production, change detection |

### When to Use Each Mode

**Quick plans** for:
- Development workflow: "What will this probably do?"
- Fast feedback during script development
- Understanding possible execution branches

**Resolved plans** for:
- Change window planning: Generate contract hours before execution
- Audit review: Show exactly what will execute with real values
- Production contracts: Lock in execution plan with hash verification

**Direct execution** for:
- Development and testing: Run with current values immediately
- One-off operations: Execute without contract overhead
- Iterative script development

**Contract execution** for:
- Production deployments: Ensure reviewed plan matches execution
- Time-delayed execution: Execute plan generated hours earlier
- Change detection: Catch environment drift since plan generation

## Resolution Strategy

**Value timing rules**:
- **Quick plans**: Expensive value decorators deferred, show placeholders
- **Resolved plans**: ALL value decorators resolve, values frozen as execution contract
- **Direct execution**: Fresh resolution at execution time, no contracts
- **Contract execution**: Fresh resolution + hash verification against contract
- **Dead code elimination**: Value decorators in pruned branches never execute

**Non-determinism guardrail**: Value decorators must be referentially transparent during `--resolve`. Non-deterministic value decorators cause contract verification failures.

**Seeded determinism**: Operations requiring randomness or cryptography (like `@random.password()` or `@crypto.generate_key()`) use Plan Seed Envelopes (PSE) to be deterministic within resolved plans while maintaining security.

**Always planned**: Even direct script execution generates internal plans first, ensuring consistent behavior.

## Time-Delayed Execution

Real operations involve plan generation and execution at different times. Opal's verification model handles this cleanly.

### The Scenario

```bash
# 2:00 PM - Generate plan during change window planning
opal deploy --dry-run --resolve > evening-deploy.plan

# 5:00 PM - Execute plan during maintenance window  
opal run --plan evening-deploy.plan
```

### Verification at Execution

When you execute a resolved plan, opal **verifies the contract**:

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

**Non-deterministic value decorator detected**:
```
ERROR: Contract verification failed

@http.get("https://time-api.com/now") returned different value:
  Plan time: <20:sha256:abc123>
  Execution:  <20:sha256:def456>

Non-deterministic value decorators cannot be used in resolved plans.
Consider separating dynamic value acquisition from deterministic execution.
```

### Plan Seed Envelopes (PSE)

For operations requiring randomness, resolved plans contain a Plan Seed Envelope - a minimal, immutable piece of state that enables deterministic random generation:

```opal
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
- Each value decorator derives unique deterministic values using plan context
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
opal run --plan prod.plan
# Fails on any source or infrastructure changes
```

**Force execution**:
```bash
opal run --plan prod.plan --force
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

```opal
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
opal deploy
# Creates resources, shows what was done

# Second run (no changes)
opal deploy  
# Shows "no-op" for existing resources

# Third run (environment changed)
REPLICAS=5 opal deploy
# Shows only the scale operation (replica count changed)
```

**How change detection works**:
- Value hashing: `<1:3>` → `<1:5>` indicates REPLICAS changed from 3 to 5
- Secret rotation: `<32:a1b>` → `<32:x7y>` indicates API token was rotated
- Infrastructure queries: Value decorators check current state vs desired state
- Character count in hash gives size hints for debugging

**Resolved plan verification adds another layer**:
- Source changes detected by comparing plan structures
- Hash changes show which values modified
- Infrastructure drift caught by re-querying current state

## Future: Infrastructure Decorators

The value decorator and execution decorator model extends naturally to infrastructure management:

```opal
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
**Stateless**: No state files to corrupt - value decorators query reality fresh each time
**Readable**: More natural than YAML, more structured than shell scripts
**Extensible**: New value decorators and execution decorators integrate seamlessly

Opal transforms operations from "pray and deploy" to "plan, review, execute with verification" - bringing contract discipline to deployment workflows without traditional infrastructure tool complexity.

## Examples

### Web Application Deployment

```opal
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

```opal
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