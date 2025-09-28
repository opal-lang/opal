# Opal Architecture

**Implementation requirements for building a Plan-Verify-Execute engine**

## Design Philosophy

Build a system where resolved plans are execution contracts that get verified before running. This creates a universal framework for safe automation across any domain where mistakes are expensive.

> **Domain-agnostic design**: Opal itself is domain-agnostic. It becomes useful in a given field through the decorator sets provided. The DevOps examples here use `@shell` and `@kubectl`, but the same rules apply equally to data, science, security, or any other domain.

The key insight: instead of managing state like traditional tools, we verify contracts. Plans aren't just previews, they're promises about what will execute. This pattern works for infrastructure, data pipelines, security automation, scientific computing, and any domain requiring predictable, auditable automation.

**Halting and determinism guarantees**: Opal is designed as a halting, deterministic automation engine. All plans are guaranteed to terminate with predictable, reproducible results across any domain.

## The Big Picture

```
User writes natural syntax  →  Parser converts to value decorators and execution decorators  →  Contract execution
```

Opal has two distinct layers that work together:

**Metaprogramming constructs** decide execution structure:

*Plan-time deterministic:*
- `for service in [...] { ... }` → unrolls loops into concrete steps
- `when ENV { ... }` → selects branches based on conditions  
- `if condition { ... } else { ... }` → evaluates conditionals at plan-time

*Execution-dependent path selection:*
- `try/catch/finally` → defines deterministic error handling paths, but which path executes depends on actual execution results (exceptions)

**Work execution** happens through decorators at runtime:
- `npm run build` → `@shell("npm run build")`
- `@retry(3) { ... }` → execution decorator with block
- `@var(NAME)` → value decorator for interpolation

## Everything is a Decorator (For Work Execution)

The core architectural principle: **every operation that performs work** becomes one of two decorator types: value decorators or execution decorators.

This means metaprogramming constructs like `for`, `if`, `when` are **not** decorators - they're language constructs that decide what work gets done. The actual work is always performed by decorators.

**Value decorators** inject values inline:
- `@env("PORT")` pulls environment variables
- `@var(REPLICAS)` references script variables  
- `@aws.secret("api-key")` fetches from AWS (expensive)

**Execution decorators** run commands:
- `@shell("npm run build")` executes shell commands
- `@retry(3) { ... }` adds retry logic around blocks
- `@parallel { ... }` runs commands concurrently

Even plain shell commands become `@shell` decorators internally:
```opal
// You write
npm run build

// Parser generates  
@shell("npm run build")
```

This separation means:
- **AST structure** represents both metaprogramming constructs and decorators appropriately
- **Execution model** is unified through decorators (no special cases for different work types)  
- **New features** integrate by adding decorators, not special execution paths

## Two-Layer Architecture

```
Plan-time Layer (Metaprogramming):
├─ for loops unroll into concrete steps (deterministic)
├─ if/when conditionals select execution paths (deterministic)
├─ try/catch defines error handling structure (execution-dependent paths)
└─ AST represents all language constructs

Runtime Layer (Work Execution):
├─ @shell decorators execute commands
├─ @retry/@parallel decorators modify execution
├─ @var/@env decorators provide values
├─ try/catch path selection based on actual exceptions
└─ Unified decorator interfaces handle all work
```

**Key insight**: `try/catch` is a metaprogramming construct (not a decorator) that defines deterministic error handling paths. Unlike `for`/`if`/`when` which resolve to a single path at plan-time, `try/catch` creates multiple **known paths** where execution selects which one based on actual results (exceptions). The plan includes **all possible paths** through try/catch blocks.

## Safety Guarantees

Opal guarantees that all operations halt with deterministic results.

### Plan-Time Safety

**Finite loops**: All loops must terminate during plan generation.
- `for item in collection` - collection size is known
- `while count > 0` - count value is resolved at plan-time
- Loop iteration happens during planning, not execution

**Command call DAG constraint**: Commands can call each other, but must form a directed acyclic graph.
- `fun` definitions called via `@cmd()` expand at plan-time with parameter binding
- Call graph analysis prevents cycles: `A → B → A` results in plan generation error  
- Parameters must be plan-time resolvable (value decorators, variables, literals)
- No dynamic dispatch - all calls resolved during planning

**Finite parallelism**: `@parallel` blocks have a known number of tasks after loop expansion.

### Runtime Safety

**User-controlled timeouts**: No automatic timeouts - users control when they want limits.
- Commands run until completion or manual termination (Ctrl+C)
- `@timeout(1h) { ... }` - explicit timeout when desired
- `--timeout 30m` flag - global safety net when needed
- Long-running processes (`dev servers`, `monitoring`) run naturally

**Resource limits**: Memory and process limits prevent system exhaustion.

### Determinism

**Reproducible plans**: Same source + environment = identical plan.
- Value decorators are referentially transparent
- Random values use cryptographic seeding (resolved plans only)
- Output ordering is deterministic

**Contract verification**: Resolved plans are execution contracts.
- Values re-resolved at runtime and hash-compared against plan
- Execution fails if any value changed since planning
- Exception: `try/catch` path selection based on actual runtime results

### Cancellation and Cleanup

**Graceful cancellation**: `finally` blocks run on interruption for safe cleanup.
- **First Ctrl+C**: Triggers cleanup sequence, shows "Cleaning up..."
- **Second Ctrl+C**: Force immediate termination, skips cleanup
- Allows resource cleanup (PIDs, temp files, containers) while providing escape hatch

## Decorator Design Requirements

When building decorators, follow these principles to maintain the contract model:

**Value decorators must be referentially transparent** during plan resolution. Non-deterministic value decorators (like `@http.get("time-api")`) will cause contract verification failures when plans are executed later.

**Execution decorators should be stateless**. Query current reality fresh each time rather than maintaining state between runs. This eliminates the complexity of state file management.

**Expose idempotency keys** so the same resolved plan can run multiple times safely. For example, `@aws.ec2.deploy` might use `region + name + instance_spec` as its key.

**Handle infrastructure drift gracefully**. When current infrastructure doesn't match plan expectations, provide clear error messages and suggested actions rather than cryptic failures.

## Plugin System

Decorators work through a dual-path plugin system that balances safety with flexibility:

### Plugin Distribution Model

**Two distribution paths following Go modules and Nix flakes pattern:**

* **Registry path (curated, verified)** → strict conformance guarantees
* **Direct Git path (user-supplied)** → bypasses registry, user owns risk

```bash
# From registry (verified)
accord get accord.dev/aws.ec2@v1.4.2

# Direct Git (team-owned, unverified)  
accord get github.com/acme/accord-plugins/k8s@v0.1.0
```

### Registry vs Git-Sourced Plugins

**Registry plugins (accord.dev/...):**
- Come with signed manifests + verification reports
- Passed full conformance suite and security audits
- Deterministic, idempotent, secrets-safety verified
- SLSA Level 3 provenance + reproducible builds
- Automatic updates within semver constraints

**Git-sourced plugins (github.com/...):**
- Can pin by commit hash for reproducibility
- `accord verify-plugin ./...` runs locally but not centrally verified
- Warning displayed but not blocked
- Useful for private/experimental/internal plugins
- Enterprise can host private verified registries

### Plugin Verification

**Registry admission pipeline**: External value decorators and execution decorators must pass comprehensive verification before registry inclusion. No arbitrary code execution - plugins pass a compliance test suite that verifies they implement required interfaces correctly and respect security requirements.

**Local verification**: Git-sourced plugins run the same conformance suite locally, providing the same crash isolation and security sandboxing but without central verification guarantees.

**Plugin isolation**: All plugins (registry or Git) run in limited contexts and can't crash the main execution engine. Resource usage gets monitored and timeouts are enforced via cgroups/bwrap.

### Registry Pattern Implementation

**Startup registration**: Both built-in and plugin value decorators and execution decorators register themselves at startup. The runtime looks up decorators by name without hardcoded lists, making the system extensible.

**Capability verification**: Engine checks on load that manifest signature matches, spec_version overlaps with runtime, and capabilities match requested decorators (no "hidden" entrypoints).

This means organizations can build custom infrastructure value decorators and execution decorators (like `@company.k8s.deploy`) while maintaining the same security and verification guarantees as built-in decorators. Small teams can ship plugins immediately via Git without waiting on central registry approval, but audit trails clearly show verification status.

## Resolution Strategy

Two-phase resolution optimizes for both speed and determinism:

**Quick plans** defer expensive operations and show placeholders:
```
kubectl create secret --token=¹@aws.secret("api-token")
Deferred: 1. @aws.secret("api-token") → <expensive: AWS lookup>
```

**Resolved plans** materialize all values for deterministic execution:
```  
kubectl create secret --token=¹<32:sha256:a1b2c3>
Resolved: 1. @aws.secret("api-token") → <32:sha256:a1b2c3>
```

Smart optimizations happen automatically:
- Expensive value decorators in unused conditional branches never execute
- Independent expensive operations resolve in parallel  
- Dead code elimination prevents unnecessary side effects

## Security Model

The placeholder system protects sensitive values while enabling change detection:

**Placeholder format**: `<length:algorithm:hash>` like `<32:sha256:a1b2c3>`. The length gives size hints for debugging, the algorithm future-proofs against changes, and the hash detects value changes without exposing content.

**Security invariant**: Raw secrets never appear in plans, logs, or error messages. This applies to all value decorators - `@env()`, `@aws.secret()`, whatever. Compliance teams can review plans confidently.

**Hash scope**: Plan hashes cover ordered steps, arguments, operator graphs, and timing flags. They exclude ephemeral data like run IDs or timestamps that shouldn't invalidate a plan.

### Plan Provenance Headers

All resolved plans include provenance metadata for audit trails:

```json
{
  "header": {
    "spec_version": "1.1",
    "plan_version": "2024.1",
    "generated_at": "2024-09-20T10:22:30Z",
    "source_commit": "abc123def456",
    "compiler_version": "opal-1.4.2",
    "plugins": {
      "aws.ec2": {
        "version": "1.4.2",
        "source": "registry:accord.dev",
        "verification": "passed",
        "signed_by": "sigstore:accord.dev/publishers/aws-team"
      },
      "company.k8s": {
        "version": "0.2.1", 
        "source": "git:github.com/acme/accord-plugins@sha256:def789",
        "verification": "local-only",
        "signed_by": null
      }
    }
  },
  "plan_hash": "sha256:5f6c...",
  "steps": [...]
}
```

**Provenance benefits:**
- **Audit compliance**: See exactly which plugins were used and their verification status
- **Risk assessment**: Distinguish registry-verified vs Git-sourced plugins
- **Reproducibility**: Pin exact plugin versions and sources
- **Security**: Track signing and verification chain

**Source classification:**
- `registry:accord.dev` - Centrally verified via registry admission pipeline  
- `registry:company.internal` - Private enterprise registry with internal verification
- `git:github.com/org/repo@sha` - Direct Git import with commit pinning
- `local:./plugins/custom` - Local development plugin

This ensures compliance teams can review plans knowing the verification status of every component, while developers retain flexibility to use unverified plugins when needed.

### Enterprise Plugin Strategies

**Private registry pattern:**
```bash
# Enterprise hosts internal registry with company plugins
accord config set registry https://plugins.company.internal

# Mix verified public and private plugins
accord get accord.dev/aws.ec2@v1.4.2        # Public verified
accord get company.internal/vault@v2.1.0     # Private verified  
accord get github.com/team/custom@v0.1.0     # Direct Git (unverified)
```

**Policy enforcement:**
- Production environments can require `verification: passed` in all plan headers
- Development environments allow unverified plugins with warnings
- CI/CD pipelines can gate on plugin verification status

**Air-gapped deployments:**
- Registry mirrors for offline environments
- Pre-verified plugin bundles with signatures
- Local verification without external registry access

This dual-path approach avoids "walled garden" criticism while maintaining security - developers can always opt out but know they're assuming risk, and audit trails preserve full accountability.

## Seeded Determinism

For operations requiring randomness or cryptography, opal will use seeded determinism to maintain contract verification while enabling secure random generation.

### Plan Seed Envelope (PSE)

**Seed generation**: High-entropy seed generated at `--resolve` time, never stored raw in plans.

**Sealed envelope**: Plans contain only encrypted seed envelopes with fields:
- `alg`: DRBG algorithm (e.g., "chacha20-drbg")  
- `kdf`: Key derivation function (e.g., "hkdf-sha256")
- `scope`: Derivation scope ("plan")
- `seed_hash`: Hash for tamper detection
- `enc_seed`: Seed sealed to runner key/KMS

**Security model**: Raw seeds never appear in plans, only sealed envelopes. Decryption requires proper runner authorization.

### Deterministic Derivation

**Scoped sub-seeds**: Each decorator gets unique deterministic sub-seed using:
```
HKDF(seed, info=plan_hash || step_path || decorator_name || counter)
```

**Stable generation**: Same plan produces same random values every time. Different plans (even with same source) produce different values due to new seed.

**Parallel safety**: Each step has unique `step_path`, ensuring no collisions in concurrent execution.

### Implementation Requirements

**API surface**:
```opal
var DB_PASS = @random.password(length=24, alphabet="A-Za-z0-9!@#")
var API_KEY = @crypto.generate_key(type="ed25519")

deploy: {
    kubectl create secret generic db --from-literal=password=@var(DB_PASS)
}
```

**Plan display**: Shows placeholders maintaining security invariant:
```
kubectl create secret generic db --from-literal=password=¹<24:sha256:abcd>
```

**Execution flow**:
1. `--resolve`: Generate PSE, derive preview hashes, seal envelope
2. `run --plan`: Decrypt PSE, derive values on-demand during execution
3. Material values injected via secure channels, never stdout/logs

**Failure modes**:
- Missing decryption capability → `infra_missing:seed_keystore`
- Tampered envelope → verification failure  
- Structure changes → normal contract verification failure

### Security Guarantees

**No value exposure**: Generated secrets follow same placeholder rules as all other sensitive values.

**Audit trail**: Plan headers include seed algorithm metadata without exposing entropy.

**Deterministic contracts**: Same resolved plan produces identical random values across executions.

**Authorization boundaries**: PSE sealed to specific runner contexts, preventing unauthorized plan execution.

This enables secure, auditable randomness within the contract verification model while maintaining all existing security invariants.

### Seed Security and Scoping

**Cryptographic independence**: Seeds are generated using 256-bit CSPRNG entropy, never derived from plan content, hashes, or names. The plan provides scoping context via HKDF info parameter, not entropy.

**Safe derivation pattern**:
```
seed = CSPRNG(256_bits)  // Independent entropy 
subkey = HKDF(seed, info=plan_hash || step_path || decorator || counter)
output = DRBG(subkey, requested_length)
```

**Regeneration keys**: Decorators use explicit regeneration keys to control when values change:

```opal
// Default: regenerates on every plan (plan hash as key)
var TEMP_TOKEN = @random.password(length=16)

// Stable: same key = same password across plan changes  
var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v1")

// Rotate by changing the key
var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v2")
```

**Derivation with regeneration keys**:
```
effective_key = regen_key || decorator_name || step_path
subkey = HKDF(seed, info=effective_key)
output = DRBG(subkey, requested_length)
```

**Value stability rules**:
- Same `regen_key` = same values (regardless of plan changes)
- Change `regen_key` = new values  
- No `regen_key` = plan hash used as key (values change on plan regeneration)

**Security hardening options**:
- Keystore references instead of embedded encrypted seeds
- Require `--resolve` for any randomness operations  
- AEAD encryption with runner-specific keys or KMS
- Seed hash for tamper detection

**Threat model**:
- Plan-only attacker: Cannot decrypt seed, sees only length/hash placeholders
- Known outputs: Cannot recover seed due to HKDF+DRBG one-way properties  
- Stolen plans: Useless without runner authorization keys

This approach provides cryptographically sound randomness while maintaining deterministic contract execution.

## Plan-Time Determinism  

Control flow expands during plan generation, not execution:

```opal
// Source code
for service in ["api", "worker"] {
    kubectl apply -f k8s/${service}/
}

// Plan shows expanded steps
kubectl apply -f k8s/api/      # Step: deploy.service[0]  
kubectl apply -f k8s/worker/   # Step: deploy.service[1]
```

This means execution decorators like `@parallel` receive predictable, static command lists rather than dynamic loops. Much easier to reason about.

**No chaining for control flow**: Constructs like `when`, `for`, `try/catch` are complete statements, not expressions. You can't write `when ENV { ... } && echo "done"` because it creates precedence confusion. Keep control flow self-contained.

## Contract Verification

The heart of the architecture: resolved plans become execution contracts.

**Verification process**: When executing a resolved plan, we replan from current source and infrastructure, then compare structures. If anything changed, we fail with a clear diff showing what's different.

**Drift classification**: We categorize verification failures to suggest appropriate actions:
- `source_changed`: Source files modified → regenerate plan
- `infra_missing`: Expected infrastructure not found → use `--force` or fix infrastructure  
- `infra_mutated`: Infrastructure present but different → use `--force` or regenerate plan

**Execution modes**: 
- Default: strict verification, fail on any changes
- `--force`: use plan values as targets, adapt to current infrastructure

This gives teams deployment confidence: the plan they reviewed is exactly what executes, with clear options when reality changes.

## Module Organization

Clean separation keeps the system maintainable:

**Core module**: Types, interfaces, and data structures only. No execution logic, no external dependencies. Defines the contracts that decorators must implement.

**Runtime module**: Lexer, parser, execution engine, and built-in decorators. Handles plugin loading and verification. Contains all the business logic.

**CLI module**: Thin wrapper around runtime. Handles command-line parsing and file I/O. No business logic.

Dependencies flow one direction: `cli/` → `runtime/` → `core/`. This prevents circular dependencies and keeps concerns separated.

## Module Structure

**Three clean modules:**

- **core/**: Types, interfaces, and plan structures
- **runtime/**: Lexer, parser, execution engine
- **cli/**: Command-line interface

Dependencies flow one way: `cli/` → `runtime/` → `core/`

## Error Handling

Try/catch is special - it's the only construct that creates non-deterministic execution paths:

```opal
deploy: {
    try {
        kubectl apply -f k8s/
        kubectl rollout status deployment/app  
    } catch {
        kubectl rollout undo deployment/app
    } finally {
        kubectl get pods
    }
}
```

Plans show all possible paths (try, catch, finally). Execution logs show which path was actually taken. This gives you predictable error handling without making plans completely deterministic.

Like other control flow, try/catch can't be chained with operators. Keep error handling self-contained to avoid precedence confusion.

## Implementation Pipeline

The compilation flow ensures contract verification works reliably:

1. **Lexer**: Fast tokenization with mode detection (command vs script mode)
2. **Parser**: Decorator AST generation  
3. **Transform**: Meta-programming expansion (loops, conditionals)
4. **Plan**: Deterministic execution sequence with stable step IDs
5. **Resolve**: Value materialization with security placeholders
6. **Verify**: Contract comparison and drift detection  
7. **Execute**: Actual command execution with idempotency

The key insight: meta-programming happens during transform, so all downstream stages work with predictable, static command sequences.

## Performance Requirements

**Lexer performance**: Target >5000 lines/ms with zero allocations for hot paths. Use pre-compiled patterns and avoid regex where possible.

**Resolution optimization**: Expensive value decorators resolve in parallel using DAG analysis. Unused branches never execute, preventing unnecessary side effects.

**Plan caching**: Plans should be cacheable and reusable between runs. Plan hashes enable this optimization.

**Partial execution**: Support resuming from specific steps with `--from step:path` for long pipelines.

## Testing Requirements

**Decorator compliance**: Every value decorator and execution decorator must pass a standard compliance test suite that verifies interface implementation, security placeholder handling, and contract verification behavior.

**Plugin verification**: External value decorators and execution decorators get the same compliance testing plus binary integrity verification through source hashing.

**Contract testing**: Comprehensive scenarios covering source changes, infrastructure drift, and all verification error types.

**Performance validation**: Lexer throughput, resolution DAG efficiency, and memory usage under load.

## IaC + Operations Together

A novel capability emerges from the decorator architecture: seamless mixing of infrastructure-as-code with operations scripts in a single language.

```opal
deploy: {
    // Infrastructure deployment
    @aws.ec2.deploy(name="web-prod", count=3)
    @aws.rds.deploy(name="db-prod", size="db.r5.large")
    
    // Operations on the deployed infrastructure  
    @aws.ec2.instances(tags={name:"web-prod"}, transport="ssm") {
        sudo systemctl start myapp
        @retry(attempts=3) { curl -f http://localhost:8080/health }
    }
    
    // Traditional ops commands
    kubectl apply -f k8s/monitoring/
    helm upgrade prometheus charts/prometheus
}
```

**The key insight**: Both infrastructure value decorators and execution decorators follow the same contract model - plan, verify, execute. This means you can mix provisioning with configuration management cleanly.

**Infrastructure value decorators** handle provisioning:
- Plan: Show what infrastructure will be created/modified
- Verify: Check current infrastructure state vs plan
- Execute: Create/modify infrastructure to match plan

**Execution decorators** handle operations:
- Plan: Show what commands will run where
- Verify: Check target systems are available and reachable
- Execute: Run commands with proper error handling and aggregation

Both types support the same features: contract verification, partial execution, idempotency, security placeholders, and plugin extensibility.

This eliminates the traditional boundary between "infrastructure tools" and "configuration management tools" - it's all just decorators with different responsibilities.

## Example: Advanced Infrastructure Execution

Here's how complex scenarios work within the decorator model:

```opal
maintenance: {
    // Select running instances
    @aws.ec2.instances(
        region="us-west-2",
        tags={env:"prod", role:"web"},
        transport="ssm",
        max_concurrency=3,
        tolerate=0
    ) {
        // Drain traffic
        sudo systemctl stop nginx
        
        // Update application  
        @retry(attempts=3, delay=10s) {
            sudo yum update -y myapp
            sudo systemctl start myapp
        }
        
        // Health check
        @timeout(30s) {
            curl -fsS http://127.0.0.1:8080/healthz
        }
        
        // Restore traffic
        sudo systemctl start nginx
    }
}
```

**Plan shows**:
- 5 instances selected by tags
- Commands that will run on each
- Concurrency and error tolerance policy
- Transport method (SSM vs SSH)

**Verification checks**:
- Selected instances still exist and match tags
- SSM transport is available on all instances  
- Classifies drift: `ok | infra_missing | infra_mutated`

**Execution provides**:
- Bounded concurrency across instances
- Per-instance stdout/stderr streaming
- Retry/timeout on individual commands
- Aggregated results with failure policy

This level of infrastructure operations was traditionally split across multiple tools. The decorator model handles it seamlessly.

## Why This Architecture Works

**Contract-first development**: Resolved plans are immutable execution contracts with verification, giving teams deployment confidence.

**IaC + ops together**: Mix infrastructure provisioning with operations scripts in one language, eliminating tool boundaries.

**Plugin extensibility**: Organizations can build custom decorators through verified, source-hashed plugins while maintaining security guarantees.

**Stateless simplicity**: No state files to corrupt or manage - decorators query reality fresh each time and use contract verification for consistency.

**Consistent execution model**: Everything becomes a decorator internally, making the system predictable and extensible without special cases.

**Performance optimization**: Plan-time expansion, parallel resolution, and dead code elimination ensure efficient execution at scale.

This delivers "Terraform for operations, but without state file complexity" through contract verification rather than state management.