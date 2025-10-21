---
oep: 001
title: Runtime Variable Binding with `let`
status: Draft
type: Feature
created: 2025-01-21
updated: 2025-01-21
---

# OEP-001: Runtime Variable Binding with `let`

## Summary

Add `let` for execution-time variable bindings from decorator effects, enabling capture of opaque handles (instance IDs, endpoints, certificates) during execution. Unlike `var` (plan-time), `let` bindings resolve during execution and use a separate `@let` namespace to prevent confusion between plan-time and execution-time values.

## Motivation

### The Problem

Current Opal only supports plan-time variables (`var`). All variable resolution happens before execution begins. This works for static configurations but breaks down when you need to capture values produced during execution:

1. **Capture infrastructure handles**: AWS instance IDs, K8s service URLs, database endpoints
2. **Thread opaque values**: Certificate fingerprints, session tokens, deployment IDs
3. **Chain dependent operations**: Deploy → capture endpoint → validate → configure DNS

**Example of what's missing:**

```opal
// ❌ Can't do this today - no way to capture runtime values
var instance_id = @aws.instance.deploy(type="t3.micro").id  // Error: can't resolve at plan time
@aws.instance.ssh(instanceId=@var.instance_id) { ... }
```

### Use Cases

**1. Infrastructure provisioning:**
```opal
let.instance_id = @aws.instance.deploy(type="t3.micro", region="us-west-2").id
@aws.instance.ssh(instanceId=@let.instance_id) {
    apt-get update
    systemctl restart app
}
@aws.route53.update(instanceId=@let.instance_id)
```

**2. Service endpoint validation:**
```opal
let.endpoint = @k8s.deploy(manifest="app.yaml").service_url
@retry(attempts=5, delay=2s) {
    @http.healthcheck(url=@let.endpoint)
}
```

**3. Database migrations:**
```opal
let.db_url = @aws.rds.create(name="prod-db").endpoint
@db.migrate(connection=@let.db_url)
@db.seed(connection=@let.db_url)
```

**4. Certificate management:**
```opal
let.cert_fingerprint = @tls.generate(cn="api.example.com").fingerprint
@aws.acm.import(cert=@let.cert_fingerprint)
```

## Proposal

### Syntax

**Write bindings:** `let.NAME = <expression>`  
**Read bindings:** `@let.NAME`

```opal
// Bind during execution
let.instance_id = @aws.instance.deploy().id

// Use in subsequent operations
@aws.instance.tag(instanceId=@let.instance_id, tags={"env": "prod"})
```

### Namespace Separation

| Aspect | `var` | `let` |
|--------|-------|-------|
| Resolution | Plan time (before execution) | Execution time (during run) |
| Source | Literals, value decorators | Decorator effects (opaque handles) |
| Determinism | Frozen in plan hash | Deterministic within a run |
| Control flow | Can drive `if`/`for`/`when` | **Cannot** drive plan-time constructs |
| Write syntax | `var NAME = ...` | `let.NAME = ...` |
| Read syntax | `@var.NAME` | `@let.NAME` |

### Core Restrictions

These restrictions are **non-negotiable** and enforced at parse time:

#### Restriction 1: No `@let` in Plan-Time Constructs

`@let` bindings **cannot** be used in any construct that affects the plan structure:

```opal
// ❌ FORBIDDEN: if condition
let.env = @aws.instance.deploy().tag("environment")
if @let.env == "production" {  // Parse error: @let not allowed in if condition
    kubectl apply -f k8s/prod/
}

// ❌ FORBIDDEN: for loop
let.count = @aws.describe_instances().count
for i in 1..@let.count {  // Parse error: @let not allowed in for loop
    echo "Instance @var.i"
}

// ❌ FORBIDDEN: when pattern match
let.status = @k8s.get_pod().status
when @let.status {  // Parse error: @let not allowed in when condition
    "Running" -> echo "Ready"
    else -> echo "Not ready"
}

// ❌ FORBIDDEN: function parameter defaults
fun deploy(replicas=@let.default_replicas) {  // Parse error
    kubectl scale --replicas=@var.replicas deployment/app
}
```

**Why?** Plan hash must be computable before execution. If `@let` could drive `if`/`for`/`when`, the plan structure would depend on runtime values, breaking contract verification.

#### Restriction 2: No `@let` in Value Decorators (Plan-Time)

`@let` cannot be used as arguments to value decorators (which resolve at plan time):

```opal
// ❌ FORBIDDEN: @let in value decorator
let.region = @aws.instance.deploy().region
var ami = @aws.ami.lookup(region=@let.region)  // Parse error: @let not allowed in value decorator

// ✅ CORRECT: Use @var for plan-time values
var REGION = @env.AWS_REGION
var ami = @aws.ami.lookup(region=@var.REGION)
```

**Why?** Value decorators execute at plan time. They can't access execution-time values.

#### Restriction 3: No `@let` in Decorator Parameters (Except Execution Decorators)

`@let` can only be used in parameters to **execution decorators** (decorators that run during execution):

```opal
// ❌ FORBIDDEN: @let in value decorator parameter
let.instance_id = @aws.instance.deploy().id
var instance_info = @aws.describe_instance(id=@let.instance_id)  // Parse error

// ✅ CORRECT: @let in execution decorator parameter
let.instance_id = @aws.instance.deploy().id
@aws.instance.tag(instanceId=@let.instance_id, tags={"env": "prod"})  // OK
```

**Why?** Value decorators resolve at plan time; execution decorators run at execution time. Only execution decorators can accept execution-time values.

#### Restriction 4: No Reassignment

Once a `let` binding is created, it cannot be reassigned:

```opal
// ❌ FORBIDDEN: reassignment
let.instance_id = @aws.instance.deploy().id
let.instance_id = @aws.instance.deploy().id  // Parse error: let.instance_id already bound

// ✅ CORRECT: Use different names
let.instance_id_1 = @aws.instance.deploy().id
let.instance_id_2 = @aws.instance.deploy().id
```

**Why?** Reassignment would make the plan ambiguous (which binding is active at any given point?). Single-assignment keeps semantics clear.

#### Restriction 5: No Shadowing Across Scopes

A `let` binding in an inner scope cannot shadow an outer scope binding:

```opal
deploy: {
    let.endpoint = @k8s.deploy(name="api").url
    
    {
        // ❌ FORBIDDEN: shadowing outer binding
        let.endpoint = @k8s.deploy(name="worker").url  // Parse error: endpoint already bound in outer scope
    }
}
```

**Why?** Shadowing makes code confusing. Explicit scoping prevents accidental overwrites.

#### Restriction 6: No Conditional Binding

`let` bindings must happen unconditionally (not inside `if`/`when`/`try`):

```opal
// ❌ FORBIDDEN: binding inside if
if @var.ENV == "production" {
    let.instance_id = @aws.instance.deploy().id  // Parse error: let binding in conditional block
}

// ❌ FORBIDDEN: binding inside try
try {
    let.instance_id = @aws.instance.deploy().id  // Parse error: let binding in try block
} catch {
    echo "Failed"
}

// ✅ CORRECT: Binding at block level, conditional usage
let.instance_id = @aws.instance.deploy().id
if @var.ENV == "production" {
    @aws.instance.tag(instanceId=@let.instance_id, tags={"env": "prod"})
}
```

**Why?** Conditional bindings would make the plan ambiguous (is the binding always present or sometimes?). Unconditional binding keeps the plan deterministic.

### Scope Rules

#### Block Scoping

Bindings are visible within the same block and inner blocks:

```opal
deploy: {
    let.instance_id = @aws.instance.deploy().id
    
    // ✅ Visible in same block
    @aws.instance.tag(instanceId=@let.instance_id, tags={"env": "prod"})
    
    // ✅ Visible in nested blocks
    {
        @aws.route53.update(instanceId=@let.instance_id)
    }
}

// ❌ Not visible here - out of scope
check: curl https://api.example.com/health
```

#### Parallel Scope Isolation

Each `@parallel` branch has an independent `@let` context. Bindings created in one branch are **not** visible in siblings:

```opal
@parallel {
    // Branch 1
    {
        let.endpoint_a = @k8s.deploy(name="api").url
        curl @let.endpoint_a  // ✅ Works
    }
    
    // Branch 2
    {
        let.endpoint_b = @k8s.deploy(name="worker").url
        curl @let.endpoint_b  // ✅ Works
        curl @let.endpoint_a  // ❌ Error: not visible across branches
    }
}
```

**Why?** Parallel branches execute independently. Sharing state between them would require synchronization and break the parallel execution model.

#### Read-After-Bind Enforcement

Reading `@let.NAME` before it is bound raises `UnboundLetError` at runtime:

```opal
// ❌ Error: used before binding
@aws.instance.ssh(instanceId=@let.instance_id) { ... }
let.instance_id = @aws.instance.deploy().id

// ✅ Correct: bind first, use after
let.instance_id = @aws.instance.deploy().id
@aws.instance.ssh(instanceId=@let.instance_id) { ... }
```

**Error message:**
```
UnboundLetError: @let.instance_id used before binding
  at line 5, column 42
  Binding occurs at line 6
```

### Structured Returns and Field Access

Decorators can return structured data (objects with multiple fields). Access fields using dot notation:

```opal
// Decorator returns object with multiple fields
let.instance = @aws.instance.deploy(type="t3.micro")
// instance = { id: "i-12345", public_ip: "1.2.3.4", private_ip: "10.0.0.1" }

// Access fields
@aws.instance.tag(instanceId=@let.instance.id)
curl http://@let.instance.public_ip/health
@aws.route53.update(ip=@let.instance.private_ip)
```

#### Nested Field Access

Support nested field access for complex structures:

```opal
let.deployment = @k8s.deploy(manifest="app.yaml")
// deployment = { 
//   service: { url: "http://app.default.svc.cluster.local", port: 8080 },
//   pods: [{ name: "app-1", status: "Running" }]
// }

curl http://@let.deployment.service.url:@let.deployment.service.port/health
```

#### Array Indexing

Support array indexing for list returns:

```opal
let.pods = @k8s.get_pods(label="app=web")
// pods = [{ name: "web-1", ip: "10.0.0.1" }, { name: "web-2", ip: "10.0.0.2" }]

curl http://@let.pods[0].ip:8080/health
```

#### Type Errors

Accessing non-existent fields or invalid indices raises `FieldAccessError` at runtime:

```opal
let.instance = @aws.instance.deploy()
curl http://@let.instance.nonexistent_field/health  // FieldAccessError: instance has no field 'nonexistent_field'

let.pods = @k8s.get_pods()
echo @let.pods[999].name  // IndexError: pods has 3 elements, index 999 out of bounds
```

### Security & Plan Representation

#### Scrubbing in Plans and Logs

Values render as `🔒 opal:h:ID` (handle placeholder) in plans and logs. This prevents sensitive data from leaking:

```
deploy:
├─ let.instance_id = @aws.instance.deploy().id
│  └─ Result: 🔒 opal:h:7Xm2Kp9
└─ @aws.instance.ssh(instanceId=🔒 opal:h:7Xm2Kp9) { ... }
```

#### Console Output

Console/TUI output is always scrubbed. Sensitive values never appear in terminal output.

#### File Output

File sinks via `>`/`>>` are raw (unscubbed) unless using `@to.display(...)`:

```opal
let.secret = @aws.secrets.get(name="db-password")

// ❌ Raw output - secret written to file
echo @let.secret > /tmp/secret.txt

// ✅ Scrubbed output - handle written to file
echo @let.secret | @to.display() > /tmp/secret.txt
```

#### Plan Hash

The plan hash includes `let` binding sites (as opaque handles), ensuring contract verification covers execution-time bindings:

```
Plan Hash: sha256:a1b2c3d4e5f6...
  Includes:
    - let.instance_id binding site
    - @aws.instance.deploy() call
    - @aws.instance.ssh() call with 🔒 opal:h:7Xm2Kp9
```

### Examples

#### Complete Deployment Workflow

```opal
deploy: {
    // Create infrastructure, capture handle
    let.instance = @aws.instance.deploy(
        type="t3.micro",
        region="us-west-2"
    )
    
    // Configure instance
    @aws.instance.ssh(instanceId=@let.instance.id) {
        apt-get update
        apt-get install -y nginx
        systemctl enable nginx
    }
    
    // Update DNS
    @aws.route53.update(
        instanceId=@let.instance.id,
        ip=@let.instance.public_ip
    )
    
    // Validate
    @retry(attempts=5, delay=10s) {
        curl http://@let.instance.public_ip/health
    }
}
```

#### Database Setup with Multiple Handles

```opal
setup: {
    // Create database
    let.db = @aws.rds.create(
        name="prod-db",
        engine="postgres",
        instanceClass="db.t3.micro"
    )
    
    // Create backup bucket
    let.bucket = @aws.s3.create(
        name="db-backups",
        region="us-west-2"
    )
    
    // Initialize database
    @aws.rds.psql(connection=@let.db.endpoint) {
        CREATE DATABASE app;
        CREATE USER app WITH PASSWORD 'secret';
    }
    
    // Configure backup
    @aws.rds.backup_config(
        database=@let.db.id,
        bucket=@let.bucket.name
    )
}
```

#### Conditional Usage (Not Binding)

```opal
deploy: {
    let.instance_id = @aws.instance.deploy().id
    
    // ✅ Conditional USAGE is OK
    if @var.ENV == "production" {
        @aws.instance.tag(
            instanceId=@let.instance_id,
            tags={"env": "prod", "critical": "true"}
        )
    }
    
    // ✅ Conditional USAGE in different branches
    when @var.REGION {
        "us-west-2" -> @aws.route53.update(instanceId=@let.instance_id, zone="us-west")
        "eu-west-1" -> @aws.route53.update(instanceId=@let.instance_id, zone="eu-west")
    }
}
```

## Rationale

### Why a separate namespace?

**Clarity:** `@var` vs `@let` makes plan-time vs execution-time explicit at every use site. Developers immediately know whether a value is available at plan time or execution time.

**Safety:** Prevents accidental use of execution-time values in plan-time constructs (which would break determinism).

**Tooling:** LSP can warn when `@let` is used in `if`/`for`/`when` conditions.

### Why not allow `let` in plan-time control flow?

**Determinism:** Plan hash must be computable before execution. If `let` could drive `if`/`for`/`when`, the plan structure would depend on runtime values, breaking contract verification.

**Example of what we prevent:**
```opal
// ❌ This would break determinism
let.env = @aws.instance.deploy().tag("environment")
if @let.env == "production" {  // Plan structure depends on runtime value!
    kubectl apply -f k8s/prod/
}
```

With this allowed, the plan would be different depending on what tag the instance has. This breaks the entire contract verification model.

### Why no reassignment?

**Clarity:** Single-assignment makes it obvious what value a binding holds at any point in the script.

**Plan determinism:** Reassignment would require tracking which binding is "active" at each point, complicating the plan representation.

**Debugging:** Single-assignment makes it easier to trace where a value came from.

### Why no conditional binding?

**Plan determinism:** If a binding might or might not exist depending on a condition, the plan structure becomes ambiguous.

**Example of what we prevent:**
```opal
// ❌ This would make the plan ambiguous
if @var.ENV == "production" {
    let.instance_id = @aws.instance.deploy().id
}
// Is instance_id always bound, or only sometimes?
```

**Workaround:** Bind unconditionally, use conditionally:
```opal
// ✅ Correct
let.instance_id = @aws.instance.deploy().id
if @var.ENV == "production" {
    @aws.instance.tag(instanceId=@let.instance_id, tags={"env": "prod"})
}
```

### Why no shadowing?

**Clarity:** Prevents accidental overwrites and makes scope relationships explicit.

**Debugging:** Easier to trace where a binding is defined.

## Alternatives Considered

### Alternative 1: Unified namespace (`@var` for both)

**Rejected:** Too confusing. Developers wouldn't know if a variable is plan-time or execution-time without checking the binding site. This leads to subtle bugs.

```opal
// ❌ Confusing - is this plan-time or execution-time?
var instance_id = @aws.instance.deploy().id
if @var.instance_id == "i-12345" { ... }  // Does this even make sense?
```

### Alternative 2: `@out` namespace (original design)

**Rejected:** `@out` implies "output from previous step" but doesn't convey execution-time semantics clearly. `@let` is more explicit about runtime binding.

### Alternative 3: No special syntax, just return values

**Rejected:** Opal is declarative, not imperative. Implicit return values would require statement-expression distinction and break the current execution model.

```opal
// ❌ This would require a completely different execution model
instance_id = @aws.instance.deploy().id  // Is this a variable or a step?
```

### Alternative 4: Allow `let` in plan-time control flow

**Rejected:** Breaks determinism and contract verification. Plan hash must be computable before execution.

### Alternative 5: Allow conditional binding

**Rejected:** Makes plan structure ambiguous. Binding must be unconditional; usage can be conditional.

### Alternative 6: Allow reassignment

**Rejected:** Makes plan ambiguous (which binding is active?). Single-assignment keeps semantics clear.

## Implementation

### Phase 1: Parser & AST
- Add `LetBinding` AST node
- Parse `let.NAME = <expr>` syntax
- Parse `@let.NAME` references
- Parse field access: `@let.NAME.field`, `@let.NAME[0]`
- Validate no `@let` in plan-time constructs (`if`/`for`/`when`)
- Validate no `@let` in value decorator parameters
- Validate no reassignment
- Validate no shadowing
- Validate no conditional binding

### Phase 2: Planner
- Track `let` binding sites in plan
- Represent bindings as opaque handles in plan hash
- Implement scrubbing for plan output (`🔒 opal:h:ID`)
- Validate field access types (catch type errors at plan time if possible)

### Phase 3: Executor
- Implement execution-time binding mechanism
- Scope management (block-scoped, parallel-isolated)
- Read-after-bind enforcement (runtime error if violated)
- Handle structured returns (`.field` accessor, `[index]` accessor)
- Implement scrubbing in logs and console output
- Implement raw output for file sinks

### Phase 4: Integration
- Update decorators to return structured data
- Add `@to.display()` for explicit unscrubbed output
- LSP warnings for `@let` in plan-time constructs
- Documentation and examples

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Type system:** Should `let` bindings have explicit type annotations? (e.g., `let.instance_id: String = ...`)
2. **Serialization:** How should `let` bindings be represented in resolved plans (`--dry-run --resolve`)?
3. **Error messages:** What level of detail should error messages include? (e.g., should we show the binding site when reporting UnboundLetError?)
4. **Performance:** Should we cache field access lookups for frequently accessed fields?
5. **Null handling:** What happens if a decorator returns `null` for a field? (e.g., `@let.instance.optional_field` when `optional_field` is null)

## References

- **Go's approach:** Similar to Go's runtime variable assignment, but with explicit namespace separation
- **Terraform's approach:** Terraform uses implicit dependencies; Opal makes them explicit with `let`
- **Related OEPs:**
  - OEP-002: Transform Pipeline Operator `|>` (uses `let` for chaining)
  - OEP-003: Automatic Cleanup with `defer` (uses `let` for error-aware cleanup)
