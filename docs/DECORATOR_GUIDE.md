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
    WithBlock(types.BlockRequired).
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

## SDK for Decorator Authors

Opal provides a secure-by-default SDK for building decorators safely.

### Secret Handling (`core/sdk/secret`)

All value decorators return `secret.Handle` for automatic scrubbing:

```go
import "github.com/aledsdavies/opal/core/sdk/secret"

func awsSecretHandler(ctx ExecutionContext, args []Param) (*secret.Handle, error) {
    secretName := ctx.ArgString("secretName")
    value := fetchFromAWS(secretName)
    return secret.NewHandle(value), nil
}
```

**Safe operations (always available):**
- `handle.ID()` - Opaque display ID: `opal:s:3J98t56A`
- `handle.IDWithEmoji()` - Display with emoji: `🔒 opal:s:3J98t56A`
- `handle.Mask(3)` - Masked display: `abc***xyz`
- `handle.Len()` - Length without exposing value
- `handle.Equal(other)` - Constant-time comparison

**Controlled access (requires capability from executor):**
- `handle.ForEnv("KEY")` - Environment variable: `KEY=value`
- `handle.Bytes()` - Raw bytes for subprocess
- `handle.UnsafeUnwrap()` - Raw value (explicit, panics in debug mode)

**Why capability gating?** Prevents accidental leaks in plugins while enabling legitimate subprocess/environment wiring. Only the executor can issue capabilities.

### Command Execution (`core/sdk/executor`)

Use `executor.Command()` instead of `os/exec` for automatic scrubbing:

```go
import "github.com/aledsdavies/opal/core/sdk/executor"

cmd := executor.Command("kubectl", "get", "pods")
cmd.AppendEnv(map[string]string{
    "KUBECONFIG": kubeconfigPath,
})
exitCode, err := cmd.Run()
```

**Why this is safe:**
- Output automatically routes through scrubber
- Secrets are redacted before display
- No way to bypass security

**API:**
- `Command(name, args...)` - Create command
- `Bash(script)` - Execute bash script
- `AppendEnv(map)` - Add environment (preserves PATH)
- `SetEnv([]string)` - Replace entire environment
- `SetStdin(reader)` - Feed input
- `SetDir(path)` - Set working directory
- `Run()` - Execute and wait (normalizes timeouts to exit code 124)
- `Start()` / `Wait()` - Async execution
- `Output()` / `CombinedOutput()` - Capture output (not scrubbed)

### Security Model

**Taint tracking**: Secrets panic on `String()` to catch accidental leaks during testing.

**Per-run keyed fingerprints**: Prevent cross-run correlation and oracle attacks.

**Locked-down I/O**: All subprocess output goes through scrubber automatically.

**Capability gating**: Raw access (ForEnv, Bytes, UnsafeUnwrap) requires executor-issued token.

**Debug mode**: Set `OPAL_SECRET_DEBUG=1` to catch leaks during testing.

See `docs/SDK_GUIDE.md` for complete API reference and examples.

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

**Use case:** Avoid redundant API calls for the same value.

```opal
# First access: API call (~150ms)
var db_pass = @aws.secret.db_password(prodAuth)

# Second access: Cached (<1ms)
var db_pass_copy = @aws.secret.db_password(prodAuth)

# Different args: New API call
var api_key = @aws.secret.api_key(prodAuth)
```

**Memoization semantics:**
- **Scope**: Per-plan execution only (not across runs)
- **Keying**: `(decorator path, canonicalized args)`
- **Invalidation**: Cleared after plan execution completes
- **Thread-safe**: Safe for concurrent access during parallel resolution

**Provenance captures effective context:**

When a value decorator is used inside a scoped execution decorator, provenance includes the context chain:

```opal
# Outside auth block - uses default auth context
var secret1 = @aws.secret.db_password
# Provenance: decorator="aws.secret.db_password", auth=<default>
# Hash: <len:algo:hash1>

# Inside auth block - uses prod auth context
@aws.auth(profile="prod") {
    var secret2 = @aws.secret.db_password
    # Provenance: decorator="aws.secret.db_password", auth=<prod>
    # Hash: <len:algo:hash2>  (different hash!)
}

# These are NOT memoized together (different effective contexts)
# Contract verification: same call in different contexts = different hashes
```

**Why this matters for verification:**
- Same decorator call in different contexts produces different hashes
- Contract verification detects context changes (e.g., auth profile changed)
- Ensures execution matches the exact context from planning

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

# ❌ Timeout has no effect - no block to apply to
@timeout(5m) && kubectl apply -f k8s/

# ✅ Use block form for decorator to apply
@timeout(5m) {
    kubectl apply -f k8s/
}
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

### @parallel Semantics

**Complete isolation per branch:**

Each branch in `@parallel` gets its own isolated execution context:

```opal
@parallel {
    cd /tmp && echo "Branch A: $(pwd)"     # Step 1
    cd /var && echo "Branch B: $(pwd)"     # Step 2
    echo "Branch C: $(pwd)"                # Step 3
}
```

**Output:**
```
Branch A: /tmp
Branch B: /var
Branch C: /home/user  # Original directory
```

**What's isolated:**
- **Working directory**: Each branch starts from parent's cwd, changes don't affect siblings
- **Environment variables**: Mutations in one branch don't leak to others
- **Stdout/stderr**: Buffered per branch, merged deterministically by step ID

**What's shared (read-only):**
- **Parent environment**: All branches see parent's environment variables
- **Parent working directory**: All branches start from same initial directory

**Deterministic final output:**

When parallel execution completes, final output is ordered by step ID (not completion time):

```opal
@parallel {
    sleep 2 && echo "Slow"   # Step 1: completes last
    echo "Fast"              # Step 2: completes first
}
```

**Final output (always):**
```
Slow
Fast
```

Even though "Fast" completes first, final output is reordered by step ID for determinism.

**During execution:** TUI may show live progress per branch (last N lines), but final output is always deterministic.

**Exit code policy:**
- Returns first non-zero exit code (by step ID order)
- Returns 0 if all branches succeed
- Fail-fast: Remaining branches cancelled on first failure

**Example with failure:**
```opal
@parallel {
    echo "Success"     # Step 1: exit 0
    exit 1             # Step 2: exit 1 (fails)
    sleep 10           # Step 3: cancelled
}
# Returns exit code 1, Step 3 never completes
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

## The Execution Context Pattern

### What Decorators Really Do

**Core insight:** Decorators take a declaration of what needs to run and wrap it in their execution context.

```opal
# User writes a declaration:
@aws.instance.ssh(host="prod-server") {
    cat /var/log/app.log
}

# Decorator does:
# 1. Setup: Establish SSH connection
# 2. Execute: Run the block in SSH context
# 3. Teardown: Close connection
```

### The Pattern: Execution Context with Callback

**Execution decorators receive three things:**

1. **Parameters** - Configuration for the decorator (`host="prod-server"`)
2. **Block** - The work to execute (the child steps)
3. **Execution Context** - A callback to execute the block

**Handler signature:**
```go
type ExecutionHandler func(
    ctx ExecutionContext,    // Execution context with args and I/O
    block []Step,            // Child steps to execute
) (exitCode int, err error)
```

**Execution context interface:**
```go
type ExecutionContext interface {
    // Execute block within this context
    ExecuteBlock(steps []Step) (exitCode int, err error)
    Context() context.Context  // For cancellation and deadlines
    
    // Decorator arguments (typed accessors)
    ArgString(key string) string
    ArgInt(key string) int64
    ArgBool(key string) bool
    ArgDuration(key string) time.Duration
    Args() map[string]Value  // Snapshot for logging
    
    // Controlled I/O (executor scrubs secrets automatically)
    // Decorators CANNOT write directly to streams
    LogOutput(message string)  // Scrubbed before output
    LogError(message string)   // Scrubbed before output
    
    // Environment and working directory
    Environ() map[string]string
    Workdir() string
    
    // Context wrapping (returns new context with modifications)
    WithContext(ctx context.Context) ExecutionContext
    WithEnviron(env map[string]string) ExecutionContext
    WithWorkdir(dir string) ExecutionContext
}
```

**Security model:**

Decorators have **no direct access to I/O streams**. All output flows through the executor which:
1. Maintains a registry of secret values from value decorator resolutions
2. Automatically replaces secret values with plan placeholders: `opal:s:ID`
3. Ensures audit trail shows which secrets were used without exposing values

This prevents decorators from accidentally (or maliciously) leaking secrets.

### Example: SSH Decorator

```go
func sshHandler(ctx ExecutionContext, block []Step) (int, error) {
    // 1. SETUP: Extract parameters and establish connection
    host := ctx.ArgString("host")
    conn, err := ssh.Dial("tcp", host+":22", sshConfig)
    if err != nil {
        return 1, err
    }
    defer conn.Close()
    
    // 2. EXECUTE: Create wrapped context and run block
    sshCtx := &SSHExecutionContext{
        parent: ctx,
        conn:   conn,
    }
    exitCode, err := sshCtx.ExecuteBlock(block)
    
    // 3. TEARDOWN: Connection closed via defer
    return exitCode, err
}
```

**Error Precedence (Normative):**

All handlers must follow this precedence:

1. **`err != nil`** → Failure (exit code informational)
2. **`err == nil` + `exitCode == 0`** → Success
3. **`err == nil` + `exitCode != 0`** → Failure

This ensures consistent behavior across all decorators and executors.

**Typed errors for policy decisions:**
```go
// Decorator can return typed errors for policy decisions
if err := validateConnection(conn); err != nil {
    return 1, &RetryableError{Cause: err}  // @retry can handle this
}
```

**Available typed errors:**
- `RetryableError` - Network failures, rate limits (decorator can retry)
- `TimeoutError` - Exceeded time limit (decorator can extend or fail)
- `CancelledError` - Context cancelled (user interrupt, deadline)

### How It Works: Recursive Execution

```
User writes:
  @retry(times=3) {
      @aws.instance.ssh(host="prod") {
          cat /var/log/app.log
      }
  }

Executor calls:
  retryHandler(ctx, block=[ssh step])
  # ctx.ArgInt("times") returns 3

Retry handler does:
  for attempt := 1; attempt <= ctx.ArgInt("times"); attempt++ {
      exitCode := ctx.ExecuteBlock(block)  ← Calls back to executor
      if exitCode == 0 { return 0 }
  }

Executor receives ssh step, calls:
  sshHandler(ctx, block=[cat command])
  # ctx.ArgString("host") returns "prod"

SSH handler does:
  conn := ssh.Dial(ctx.ArgString("host") + ":22", ...)
  sshCtx := wrapContext(ctx, conn)
  exitCode := sshCtx.ExecuteBlock(block)  ← Calls back to executor

Executor receives cat command, calls:
  shellHandler(ctx, block=[])

Shell handler does:
  cmd := exec.Command("bash", "-c", "cat /var/log/app.log")
  return cmd.Run()
```

### Key Properties

**1. Decorator Controls Execution**
- **When**: Immediately, after setup, conditionally, in parallel
- **How**: With modified stdout, within SSH session, with timeout
- **Whether**: Can skip block entirely, retry, or execute multiple times

**2. Context Wrapping**
Decorators can wrap the execution context to modify behavior:

```go
type SSHExecutionContext struct {
    parent ExecutionContext  // Original context
    conn   *ssh.Client       // SSH connection
}

func (s *SSHExecutionContext) ExecuteBlock(steps []Step) (int, error) {
    // Redirect stdout/stderr through SSH
    // Run commands on remote host
    // Return exit code
}
```

**3. Recursive Composition**
Decorators can be nested arbitrarily deep:

```opal
@retry(times=3) {
    @timeout(30s) {
        @aws.instance.ssh(host="prod") {
            @parallel {
                kubectl apply -f deployment.yaml
                kubectl rollout status deployment/app
            }
        }
    }
}
```

Each decorator wraps the next, creating a chain of execution contexts.

### Pattern Classification

**Name:** Execution Context Pattern

**Combines:**
- **Middleware's** wrap-around behavior (setup/execute/teardown)
- **Inversion of Control's** callback mechanism (handler calls executor)
- **Context pattern's** state management (ExecutionContext)
- **Registry pattern's** extensibility (decorator registration)

**Similar to:**
- Python's `with` statement (context managers)
- React's Context API (providers wrapping children)
- Go's http.Handler middleware (wrapping with `next.ServeHTTP()`)

### Implementing a Block-Based Decorator

**Step 1: Define schema with block**
```go
schema := types.NewSchema("timeout", "execution").
    Description("Execute block with timeout").
    Param("duration", "duration").
        Description("Maximum execution time").
        Required().
        Done().
    WithBlock(types.BlockRequired).  // ← Accepts block
    Build()
```

**Step 2: Implement handler**
```go
func timeoutHandler(ctx ExecutionContext, block []Step) (int, error) {
    duration := ctx.ArgDuration("duration")
    
    // Create context with timeout (composes with parent context)
    timeoutCtx, cancel := context.WithTimeout(ctx.Context(), duration)
    defer cancel()
    
    // Execute block with timeout context
    return ctx.WithContext(timeoutCtx).ExecuteBlock(block)
}
```

**Note:** This simplified version composes with parent context automatically. If parent has a deadline, the shorter one wins.

**Step 3: Register decorator**
```go
func init() {
    types.Global().RegisterExecutionWithSchema(schema, timeoutHandler)
}
```

## Summary

These patterns enable:
- **Composable handles** - Pass context between decorators
- **Deterministic planning** - Same inputs always produce same plan
- **Efficient execution** - Memoization and batching reduce API calls
- **Observable operations** - Full traceability without exposing secrets
- **Type safety** - Optional types catch errors at plan-time
- **Natural composition** - Decorators work together seamlessly
- **Context wrapping** - Decorators establish execution environments
- **Recursive execution** - Arbitrary nesting of decorators

All while maintaining Opal's core guarantee: **resolved plans are execution contracts**.
