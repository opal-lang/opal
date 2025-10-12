---
title: "Decorator Design Guide"
audience: "Decorator Authors & Plugin Developers"
summary: "Patterns, conventions, and best practices for building composable decorators"
---

# Decorator Design Guide

**Patterns, conventions, and best practices for building composable, deterministic decorators in Opal**

## Invariants

Every decorator must maintain these guarantees:

1. **Referential transparency** - Same inputs produce same plan
2. **No side effects during planning** - Planning only computes what will execute
3. **Deterministic resolution** - Ambiguity causes plan-time errors
4. **Observable execution** - All actions are traceable and auditable

**Philosophy:** Blocks tame clutter. When decorator composition gets complex, prefer block structure over long chains. Readability trumps brevity.

## Decorator Anatomy

Every decorator call has three optional components:

```opal
@path.primary(param1=value1, param2=value2) { block }
```

### 1. Primary Property (Optional)

The **primary property** is a shorthand for the most important parameter. Decorator authors decide if their decorator uses it.

```opal
# These are equivalent:
@env.HOME
@env("HOME")
@env(property="HOME")

# With additional parameters:
@env.HOME(default="/home/user")
@env("HOME", default="/home/user")
```

**When to use primary property:**
- **Value decorators** that access named resources: `@var.name`, `@env.PATH`, `@file.read.config`
- **Single required parameter** that's used 90% of the time
- **Readability** - makes common cases cleaner

**When NOT to use primary property:**
- **Execution decorators** with multiple equally-important parameters: `@retry(times=3, delay=2s)`
- **No obvious "main" parameter**
- **Block-based decorators** where the block is the primary input: `@parallel { ... }`

**Schema definition:**
```go
// Decorator with primary parameter
.PrimaryParam("property", "string", "Environment variable name")

// Decorator without primary parameter
// (just omit PrimaryParam() call)
```

### 2. Named Parameters (Optional)

Supplemental configuration beyond the primary property.

```opal
@file.read("config.yaml", encoding="utf-8", cache=true)
@retry(times=3, delay=2s, backoff="exponential")
@aws.secret.db_password(auth=prodAuth, version="latest")
```

**Parameter conventions:**
- Use `camelCase` for parameter names
- Support both positional and named forms
- Provide sensible defaults

### 3. Block (Optional)

A **lambda/unit of execution** passed to the decorator (Kotlin-style). The block is a parameter containing code to execute.

```opal
# Block as lambda for execution control:
@retry(times=3) {
    kubectl apply -f deployment.yaml
    kubectl rollout status deployment/app
}

# Block as scope for configuration:
@aws.auth(profile="prod") {
    var secret = @aws.secret.db_password
    var apiKey = @aws.secret.api_key
}

# Block for iteration:
@parallel {
    @task("build-frontend") { npm run build }
    @task("build-backend") { go build }
}
```

**Block semantics:**
- Blocks are **execution contexts**, not configuration objects
- The decorator controls when/how the block executes
- Blocks can be retried, parallelized, or conditionally executed
- Blocks have access to the parent scope's variables

## Decorator Kinds

Opal distinguishes between two kinds of decorators:

### Value Decorators

**Return data with no side effects.** Can be used in expressions and string interpolation.

```opal
# Value decorators return data:
var home = @env.HOME
var config = @file.read("config.yaml")
var secret = @aws.secret.db_password(auth=prodAuth)

# Can be interpolated in strings:
echo "Home directory: @env.HOME"
echo "Config: @file.read('settings.json')"
```

**Characteristics:**
- Pure functions (same inputs → same outputs)
- No side effects during plan or execution
- Can be used anywhere a value is expected
- Registered with `RegisterValue(path)`

### Execution Decorators

**Perform actions with side effects.** Cannot be interpolated in strings.

```opal
# Execution decorators perform actions:
@file.write("output.txt", content="Hello World")
@shell("kubectl apply -f deployment.yaml")
@aws.instance.deploy(config=myConfig)

# Cannot be interpolated (stays literal):
echo "Running @shell('ls')"  # Prints literally: "Running @shell('ls')"
```

**Characteristics:**
- Perform side effects (write files, make API calls, etc.)
- Cannot be used in string interpolation
- Can accept blocks for execution control
- Registered with `RegisterExecution(path)`

### Same Namespace, Different Kinds

A namespace can have both value and execution decorators:

```opal
# Value decorator (read):
var content = @file.read("config.yaml")

# Execution decorator (write):
@file.write("output.txt", content=@var.content)
```

## Defining Decorator Schemas

Decorators are defined using a **schema builder API** that provides type safety and self-documentation.

### Basic Value Decorator Schema

```go
// In runtime/decorators/env.go
func init() {
    schema := types.NewSchema("env", "value").
        Description("Access environment variables").
        PrimaryParam("property", "string", "Environment variable name").
        Param("default", "string").
            Description("Default value if env var not set").
            Optional().
            Examples("", "/home/user").
            Done().
        Returns("string", "Environment variable value").
        Build()
    
    types.Global().RegisterValueWithSchema(schema, envHandler)
}
```

**This schema enables:**
- `@env.HOME` - Primary parameter syntax sugar
- `@env.HOME(default="/home/user")` - Additional parameters
- Parser validates parameter names at parse-time
- IDE autocomplete for parameter names
- Auto-generated documentation

### Execution Decorator Schema

```go
schema := types.NewSchema("retry", "execution").
    Description("Retry failed operations with exponential backoff").
    Param("times", "int").
        Description("Number of retry attempts").
        Default(3).
        Done().
    Param("delay", "duration").
        Description("Initial delay between retries").
        Default("1s").
        Examples("1s", "5s", "30s").
        Done().
    Param("backoff", "string").
        Description("Backoff strategy").
        Default("exponential").
        Done().
    AcceptsBlock().
    Build()
```

**Usage:**
```opal
@retry(times=5, delay=2s) {
    kubectl apply -f deployment.yaml
}
```

### Schema Type System

**Supported parameter types:**
- `"string"` - UTF-8 text
- `"int"` - 64-bit signed integer
- `"float"` - 64-bit floating point
- `"bool"` - Boolean true/false
- `"duration"` - Time duration (1s, 5m, 2h)
- `"object"` - Key-value map
- `"array"` - Ordered list
- Custom types: `"AuthHandle"`, `"SecretHandle"`, etc.

**Parameter modifiers:**
- `.Required()` - Parameter must be provided
- `.Optional()` - Parameter is optional
- `.Default(value)` - Default value (implies optional)
- `.Examples(...)` - Example values for documentation
- `.Description(text)` - Human-readable description

### Primary Parameter Pattern

The **primary parameter** enables dot-notation syntax sugar:

```go
// Schema defines primary parameter
.PrimaryParam("secretName", "string", "Name or ARN of the secret")

// Enables these equivalent forms:
@aws.secret.db_password              // Dot syntax (sugar)
@aws.secret("db_password")           // Positional
@aws.secret(secretName="db_password") // Named
```

**Path matching:** Registry uses longest-match routing:
```opal
@aws.secret.db_password
    ^^^^^^^^^^           ← Registered path: "aws.secret"
              ^^^^^^^^^^^^ ← Primary parameter: "db_password"
```

### Schema Validation

Schemas are validated at registration time:
- Primary parameter must exist in parameters map
- Required parameters cannot have defaults
- Type names must be valid
- Path cannot be empty

**Example validation error:**
```
Error: invalid schema for "retry": primary parameter "attempts" not found in parameters
```

## Naming Conventions

**Verb-first naming** for clarity:
```opal
✅ Good: @retry, @timeout, @log, @aws.secret, @k8s.rollout
❌ Bad:  @retryPolicy, @timeoutHandler, @logger
```

**Avoid synonyms** - one concept, one name:
```opal
✅ Good: @retry (standard)
❌ Bad:  @repeat, @redo, @again (confusing alternatives)
```

## Parameter Flexibility

**Kotlin-style flexibility** - all forms supported:
```opal
@retry(3, 2s)                    # Positional
@retry(attempts=3, delay=2s)     # Named
@retry(3, delay=2s)              # Mixed
```

All three forms are valid. Use what's clearest for your use case.
- Consistent patterns: `maxAttempts` not `max_attempts` or `attemptsMax`

**Duration format**:
```opal
✅ Good: 500ms, 30s, 5m, 2h
❌ Bad:  300, 2000, "5 minutes"
```

**Enum values**:
```opal
✅ Good: level="info|warn|error|debug|trace"
❌ Bad:  level="INFO|Warning|err"  # Inconsistent casing
```

## Design Patterns

### Pattern: Opaque Capability Handles

**Use case**: Pass authentication, configuration, or connection context between decorators without embedding secrets.

**Value decorator returns a handle:**
```opal
var prodAuth = @aws.auth(profile="prod", roleArn="arn:aws:iam::123:role/ci")
var dbConn = @postgres.connection(host="db.prod", database="app")
```

The value is a **pure spec** (immutable parameters only, not live connections).

**Other decorators accept the handle:**
```opal
var db_password = @aws.secret.db_password(auth=prodAuth)
var users = @postgres.query(sql="SELECT * FROM users", conn=dbConn)
```

**Scoped vs Handle Style:**

Scoped (ergonomic for blocks):
```opal
@aws.auth(profile="prod") {
    var db_pass = @aws.secret.db_password
    var api_key = @aws.secret.api_key
}
```

Handle (composable, passable to functions):
```opal
var prodAuth = @aws.auth(profile="prod")

fun deploy(auth) {
    var secret = @aws.secret.db_password(@var.auth)
    kubectl apply -f k8s/
}

deploy(auth=prodAuth)
```

### Pattern: Resource Collections

**Use case**: Work with multiple cloud resources as a collection.

**Value decorator returns collection:**
```opal
var webServers = @aws.ec2.instances(
    tags={role: "web", env: "prod"},
    state="running"
)
```

**Execution decorator operates on collection:**
```opal
@aws.ec2.run(instances=webServers, transport="ssm") {
    sudo systemctl restart nginx
}
```

**Iteration:**
```opal
for instance in @var.webServers {
    echo "Checking @var.instance.id at @var.instance.privateIp"
}
```

### Pattern: Hierarchical Namespaces

**Use case**: Organize related decorators logically.

**Dot notation for hierarchy:**
```opal
# AWS services
@aws.secret.db_password
@aws.ec2.instances
@aws.s3.objects

# Kubernetes resources
@k8s.pods
@k8s.deployments

# Configuration sources
@config.app.databaseUrl
@env.HOME
```

### Pattern: Memoized Resolution

**Use case**: Avoid redundant API calls for the same value.

```opal
# First access: API call (~150ms)
var db_pass = @aws.secret.db_password(prodAuth)

# Second access: Cached (<1ms)
var db_pass_copy = @aws.secret.db_password(prodAuth)

# Different args: New API call
var api_key = @aws.secret.api_key(prodAuth)
```

### Pattern: Batch Resolution

**Use case**: Multiple decorators fetching from same provider batch requests.

```opal
var prodAuth = @aws.auth(profile="prod")

# All three collected during planning
var db_pass = @aws.secret.db_password(auth=prodAuth)
var api_key = @aws.secret.api_key(auth=prodAuth)
var cert = @aws.secret.tls_cert(auth=prodAuth)

# Executed as single batch API call
# Performance: 1 call (150ms) vs 3 calls (450ms)
```

### Pattern: Flexible Idempotence Matching

**Use case**: Let users decide which attributes matter for "does this already exist?"

**Recommended pattern** for decorators that ensure resources/state exists. Decorator authors can choose to implement this pattern for consistency across the ecosystem.

```opal
# Pragmatic: use existing if name matches
var web = @aws.instance.deploy(
    name="web-server",
    type="t3.medium",
    idempotenceKey=["name"],  # Array: only "name" field matters
    onMismatch="warn"          # Warn if type differs, but use it
)
# Found "web-server" with t3.large? Warn, then use it

# Strict: fail if mismatch
@k8s.deployment.ensure(
    name="api",
    replicas=3,
    image="api:v2",
    idempotenceKey=["name", "replicas", "image"],  # Array: all must match
    onMismatch="error"                              # Fail on any mismatch
)
# Found "api" with replicas=5? Error and abort

# Create new: make another if mismatch
var db = @aws.rds.deploy(
    name="app-db",
    engine="postgres",
    instanceClass="db.t3.micro",
    idempotenceKey=["name", "engine"],  # Array: name + engine must match
    onMismatch="create"                  # Create new if mismatch
)
# Found "app-db" with engine="mysql"? Create "app-db-2" with postgres

# Silent: use existing, no warnings
@file.ensure(
    path="/etc/app/config.json",
    content=@var.config,
    idempotenceKey=["path"],  # Array: only path matters
    onMismatch="ignore"        # Use existing, no warnings
)
# File exists with different content? Use it, no warnings
```

**Levels of pragmatism:**
- **Fully pragmatic**: `idempotenceKey=["name"]` - anything with this name is fine
- **Semantic**: `idempotenceKey=["name", "engine"]` - match what matters operationally
- **Strict**: `idempotenceKey=["name", "type", "version"]` - everything must match

**How it works:**
- `idempotenceKey`: Array of field names to check (e.g., `["name"]`, `["name", "engine"]`)
- `onMismatch`: User configuration for what happens when fields outside the key differ

**Mismatch handling (user decides):**
- `onMismatch="ignore"` - Use existing resource, no warnings (ephemeral environments)
- `onMismatch="warn"` - Use existing resource, warn about differences (default)
- `onMismatch="error"` - Fail and abort execution (production safety)
- `onMismatch="create"` - Create new resource with modified identifier (e.g., "app-db-2")

**How this works**: The runtime provides `idempotenceKey` and `onMismatch` as standard parameters. Decorator authors opt in by implementing the query logic to check existing state. This pattern aligns with Opal's outcome-focused philosophy - match what matters for your use case, not rigid state enforcement.

**Decorators that benefit from this pattern:**
- Resource creation: `@aws.instance.deploy`, `@k8s.deployment.ensure`, `@docker.container.ensure`
- File operations: `@file.ensure`, `@directory.create`
- Package management: `@apt.install`, `@npm.install`
- Any decorator that can query "does this already exist?"

**Decorators that don't need it:**
- Execution modifiers: `@retry`, `@timeout`, `@parallel` (no state to query)
- Value readers: `@env.VAR`, `@var.NAME` (read-only)
- Pure side effects: `@log`, `@shell` (no queryable state)

**Implementation is optional**: Decorator authors decide if their decorator benefits from this pattern. When implemented, users get consistent syntax across the ecosystem.

### Pattern: Path-Aware Resolution

**Use case**: Don't resolve values on code paths that won't execute.

```opal
when @var.ENV {
    "production" -> {
        var secret = @aws.secret.prod_db(prodAuth)  # Only if ENV=production
    }
    "staging" -> {
        var secret = @aws.secret.staging_db(stagingAuth)  # Only if ENV=staging
    }
}
```

If `ENV=production`, staging secret is never fetched.

### Pattern: Deterministic Fallbacks

**Use case**: Provide sensible defaults while maintaining determinism.

**Resolution order** (highest to lowest priority):
1. Explicit parameter
2. Scoped context
3. Project config
4. Environment variable
5. Default value

```opal
# Explicit wins
var auth = @aws.auth(profile="prod")
var secret = @aws.secret.db_password(auth)  # Uses "prod"

# Scoped context
@aws.auth(profile="staging") {
    var secret = @aws.secret.db_password  # Uses "staging"
}

# Ambiguous = plan-time error
```

## Composition Guidelines

### Understanding Block Decorator Semantics

**Block decorators only apply to their blocks:**

```opal
# Timeout applies to the entire block
@timeout(5m) {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
}

# Timeout has no effect - no block to apply to
@timeout(5m) && kubectl apply -f k8s/
```

**Why this matters:**
- `@timeout(5m) { ... }` - Timeout wraps the entire block
- `@timeout(5m) && command` - Timeout has nothing to wrap, does nothing
- Not a style preference - understanding what the decorator applies to

**Chaining blocks works as expected:**
```opal
# Both decorators apply to their respective blocks
@timeout(5m) { kubectl apply } && @retry(3) { kubectl rollout status }
```

**When to use blocks:**
- When you want the decorator to apply to multiple statements
- When nesting improves readability
- When you have ≥2 control decorators on the same logical operation

### Block Nesting Order

**Logical order** (outside to inside):
1. **Time constraints**: `@timeout`
2. **Error handling**: `@retry`
3. **Control flow**: `@parallel`, `@when`
4. **Logging/monitoring**: `@log`
5. **Execution**: shell commands, `@cmd`

Breaking this order requires a comment explaining why.

## Best Practices

### 1. Fail Fast at Plan-Time

```opal
# Good: Clear error at plan-time
var secret = @aws.secret.db_password  # ERROR: no auth specified

# Bad: Would fail at execution time
```

### 2. Make Ambiguity Explicit

```opal
# Bad: Implicit, ambiguous
var instances = @aws.ec2.instances  # Which region? Which account?

# Good: Explicit, deterministic
var prodAuth = @aws.auth(profile="prod", region="us-east-1")
var instances = @aws.ec2.instances(tags={env: "prod"}, auth=prodAuth)
```

### 3. Design for Observability

Every decorator should emit telemetry:
```
Decorator execution summary:
  aws.secret.db_password (auth=<3e8f...>): 145ms, success
  aws.secret.api_key (auth=<3e8f...>): <1ms, cached
  postgres.query (conn=<a7b2...>): 23ms, 150 rows
```

### 4. Redact Secrets

Never log, print, or store raw credentials:
```opal
var secret = @aws.secret.db_password(prodAuth)
echo "Secret: @var.secret"  # Output: "Secret: <secret:redacted>"
```

### 5. Support Composition

Decorators should compose naturally:
```opal
var prodAuth = @aws.auth(profile="prod")
var dbPass = @aws.secret.db_password(auth=prodAuth)
var dbConn = @postgres.connection(host="db.prod", password=dbPass)
var users = @postgres.query(sql="SELECT * FROM users", conn=dbConn)
```

## Decorator Categories

**Official taxonomy** (all decorators must declare):
- `control` - Flow control (@retry, @timeout, @parallel)
- `io` - Input/output (@log, @file, @http)
- `cloud` - Cloud providers (@aws.secret, @gcp.storage, @azure.vault)
- `k8s` - Kubernetes (@k8s.apply, @k8s.rollout)
- `git` - Version control (@git.branch, @git.commit)
- `proc` - Process management (@shell, @cmd)

## Quality Assurance

### Lint Rules (Enforced)

**D001: Chain complexity** (ERROR)
```opal
❌ @timeout(5m) && @retry(3) && @log("x") && command
✅ Fix: Use block structure
```

**D002: Unknown decorators** (ERROR)
```opal
❌ @retrry(3) { command }
✅ Fix: Did you mean @retry? (auto-fixable)
```

**D003: Deprecated decorator usage** (WARNING)
```opal
❌ @retryPolicy(3) { command }  # Old naming convention
✅ Fix: @retry(3) { command } (auto-fixable)
```

### CI Integration

```bash
opal lint --strict    # Fail on D001-D003
opal lint --fix       # Auto-fix D002 and D003
opal fmt --check      # Verify formatting
```

## Common Decorator Types

### Value Decorators

Return pure values (no side effects during planning):
- `@aws.auth()` - Auth handle
- `@aws.secret.NAME` - Secret value
- `@aws.ec2.instances()` - Instance collection
- `@env.VAR` - Environment variable
- `@var.NAME` - Script variable

### Execution Decorators

Perform actions (side effects during execution):
- `@aws.ec2.run()` - Execute on instances
- `@k8s.exec()` - Execute in pods
- `@retry()` - Retry with backoff
- `@parallel()` - Parallel execution
- `@shell()` - Shell command

### Scoped Decorators

Create context for nested blocks:
- `@aws.auth() { ... }` - Auth scope
- `@workdir() { ... }` - Working directory
- `@timeout() { ... }` - Timeout constraint

## Example: Well-Designed Composition

```opal
var ENV = "production"
var TIMEOUT = 10m

deploy: @timeout(TIMEOUT) {
    @log("Starting deployment to @var.ENV", level="info")
    
    @retry(attempts=3, delay=5s) {
        when @var.ENV {
            production: {
                kubectl apply -f k8s/prod/
                kubectl rollout status deployment/app --timeout=300s
            }
            staging: {
                kubectl apply -f k8s/staging/  
                kubectl rollout status deployment/app --timeout=60s
            }
        }
    }
    
    @parallel {
        kubectl get pods -l app=myapp
        @log("Deployment completed successfully", level="info")
    }
}
```

**Why this works**:
- Clear nesting hierarchy (timeout → retry → execution)
- Named parameters throughout
- Logical block structure
- Readable variable interpolation
- Mixed decorator types working together

## Decorator Author Checklist

Before implementing a new decorator, verify all requirements:

| Category | Requirement | Why It Matters |
|----------|-------------|----------------|
| **Transparency** | Same inputs → same output | Enables plan determinism |
| **Determinism** | Parameters resolve at plan-time | Supports contract verification |
| **Observability** | All actions logged with context | Enables debugging and auditing |
| **Error Handling** | Clear messages with suggestions | Improves developer experience |
| **Security** | Secrets never logged/exposed | Maintains security invariant |
| **Category** | Assigned to taxonomy | Enables discovery and organization |
| **Documentation** | Examples for common use cases | Helps users understand usage |
| **Testing** | Conformance tests verify invariants | Ensures reliability (see [TESTING_STRATEGY.md](TESTING_STRATEGY.md)) |
| **Telemetry** | Emits timing and status metrics | Supports observability |
| **Composability** | Can be passed to other decorators | Enables advanced patterns |
| **Memoization** | Identical calls return cached results | Improves performance |
| **Batching** | Multiple calls can be batched | Reduces API overhead |

**Quick validation:**
```bash
# Run decorator conformance suite
opal test decorators/@your.decorator

# Verify all checks pass:
# ✓ Referential transparency
# ✓ Deterministic parameters
# ✓ Observable actions
# ✓ Error handling
# ✓ Security (no secret leakage)
# ✓ Telemetry emitted
# ✓ Memoization works
```

## Summary

These patterns enable:
- **Composable handles** - Pass context between decorators
- **Deterministic planning** - Same inputs always produce same plan
- **Efficient execution** - Memoization and batching reduce API calls
- **Observable operations** - Full traceability without exposing secrets
- **Type safety** - Optional types catch errors at plan-time
- **Natural composition** - Decorators work together seamlessly

All while maintaining Opal's core guarantee: **resolved plans are execution contracts**.
