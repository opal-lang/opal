# Clean Code Guidelines for Devcmd

Guidelines for maintaining clean, discoverable, and scalable decorator composition as the opal ecosystem grows.

## Core Philosophy

**Blocks tame clutter.** When decorator composition gets complex, prefer block structure over long chains. Readability trumps brevity.

## Decorator Design Principles

### **Naming Conventions**

**Verb-first naming** for clarity and consistency:
```opal
✅ Good: @retry, @timeout, @log, @aws.secret, @k8s.rollout
❌ Bad:  @retryPolicy, @timeoutHandler, @logger
```

**Avoid synonyms** - one concept, one name:
```opal
✅ Good: @retry (standard)
❌ Bad:  @repeat, @redo, @again (confusing alternatives)
```

### **Parameter Design**

**Named parameters over positional soup**:
```opal
✅ Good: @retry(attempts=3, delay=2s)
❌ Bad:  @retry(3, 2000)
❌ Bad:  @retry(3, delay=2s)  # No mixing named and positional

✅ Good: @timeout(duration=5m, signal="TERM")
❌ Bad:  @timeout(300, 15)
```

**Parameter naming conventions**:
- Use `lower_snake_case` for all parameter names
- Use consistent verb/noun patterns: `max_attempts` not `maxAttempts` or `attempts_max`

**Duration format (strict)**:
```opal
✅ Good: 500ms, 30s, 5m, 2h
❌ Bad:  300, 2000, "5 minutes", "PT30S" (no ISO-8601)
```

**Enum values (standardized)**:
```opal
✅ Good: level="info|warn|error|debug|trace"
✅ Good: signal="TERM|KILL|INT|HUP"
❌ Bad:  level="INFO|Warning|err"  # Inconsistent casing
```

### **Decorator Cohesion**

**One concept → one decorator**:
- If two decorators overlap 80%+, merge or alias them
- Avoid feature creep in single decorators
- Prefer composition over complex single decorators

**Aliasing rule**: When introducing aliases:
- Old name becomes alias-only (no new features)
- Must be marked deprecated with specific removal version
- Example: `@repeat` → `@retry` (deprecated in v1.2, removed in v1.3)

## Composition Guidelines

### **Chain vs Block Decision Rule**

**Use blocks when**:
- Line has ≥2 control decorators
- Any pipe/chain operators present
- Nesting improves readability

```opal
❌ Bad: @timeout(5m) && @retry(3) && @log("starting") && kubectl apply

✅ Good:
@timeout(5m) {
    @retry(3) {
        @log("starting")
        kubectl apply -f k8s/
    }
}
```

### **Block Structure**

**Logical nesting order** (outside to inside):
1. **Time constraints**: `@timeout`
2. **Error handling**: `@retry`, `@try` 
3. **Control flow**: `@when`, `@parallel`
4. **Logging/monitoring**: `@log`
5. **Execution**: shell commands, `@cmd`

**Breaking nesting order requires a comment explaining why.**

```opal
✅ Good nesting order:
@timeout(10m) {
    @retry(attempts=3) {
        @when(ENV) {
            production: @log("prod deploy") && kubectl apply -f prod/
            staging: @log("staging deploy") && kubectl apply -f staging/
        }
    }
}
```

## Discoverability and Documentation

### **Built-in Help System**

**Every decorator must provide**:
```go
func (d *RetryDecorator) Description() string {
    return "Retry command execution with configurable attempts and delay"
}

func (d *RetryDecorator) Examples() []string {
    return []string{
        "@retry(3) { kubectl apply -f k8s/ }",
        "@retry(attempts=5, delay=2s) { npm test }",
    }
}
```

**Help should be concise**:
- One-line description 
- 2-3 practical examples
- No essays or excessive detail

### **CLI Discovery Commands**

```bash
opal decorators                           # List all available
opal decorators --category control        # Filter by category  
opal decorators --search rollout          # Search by keyword
opal help @retry                          # Specific decorator help
opal help @retry --examples               # Show usage examples
```

### **Decorator Categories**

**Official taxonomy** (registry contract - all decorators must declare):
- `control` - Flow control (@retry, @timeout, @when, @parallel)
- `io` - Input/output (@log(message, level="info"), @file(path), @http(url))
- `cloud` - Cloud providers (@aws.secret(name), @gcp.storage(bucket), @azure.vault(key))
- `k8s` - Kubernetes (@k8s.apply(manifest), @k8s.rollout(deployment))
- `git` - Version control (@git.branch(), @git.commit(message))
- `proc` - Process management (@shell(command), @cmd(name))

**CI requirement**: Decorators without category declarations fail CI build.

## Quality Assurance

### **Lint Rules (Enforced)**

**D001: Chain complexity** (ERROR)
```opal
❌ @timeout(5m) && @retry(3) && @log("x") && command
✅ Fix: Use block structure for ≥2 control decorators or any |/&&/|| operators
```

**D002: Unknown decorators** (ERROR)
```opal
❌ @retrry(3) { command }
✅ Fix: Did you mean @retry? (auto-fixable)
```

**D003: Mixed argument styles** (ERROR)
```opal
❌ @retry(3, delay=2s)  # No mixing positional and named
✅ Fix: @retry(attempts=3, delay=2s) (auto-fixable)
```

**CI Integration**:
```bash
opal lint --strict    # Fail on D001-D003, warn on deprecations
opal lint --fix       # Auto-fix D002 and D003 where possible
```

### **Collision Policy**

**Resolution order**:
1. **Built-ins** (always win)
2. **Project decorators** (local .opal/)
3. **Extensions** (installed packages)

**Edge case handling**:
- **Fully-qualified names always win**: `@aws.secret` bypasses local `@secret` alias
- **Shadowing warnings**: Warn when project decorators shadow extensions
- **Conflict resolution**: Use `--no-shadow` to fail on conflicts
- **Deterministic escape**: Always prefer explicit namespacing over implicit resolution

## Telemetry and Observability

### **Capability Tags**

Every decorator emits capability tags for filtering:
```json
{
  "decorator": "retry",
  "tags": ["control", "error-handling"],
  "duration_ms": 15420
}

{
  "decorator": "shell", 
  "tags": ["proc.spawn", "io.fs"],
  "command": "kubectl apply",
  "duration_ms": 2340
}
```

### **Stable IDs for Tracing**

**ID format**: `@decorator#<ordinal>@<line>:<col>`
```
deploy: 45.6s total
├─ @timeout#1@12:5(5m): 45.7s
│  ├─ @retry#1@13:9(3): 31.2s 
│  └─ @retry#2@18:9(2): 12.8s
```

This makes telemetry diffs stable across refactors while maintaining precise location tracking.

## Cross-Project Hygiene

### **Namespace Strategy**

**Local prefixes optional, not required**:
```opal
✅ Simple: @deploy, @scale (when unambiguous)
✅ Explicit: @myapp.deploy, @myapp.scale (when needed)
❌ Forced: Always requiring namespaces
```

**Bundle profiles** instead of decorator explosion:
- `core` - Essential decorators (@retry, @timeout, @log)
- `k8s` - Kubernetes workflow decorators
- `aws` - AWS-specific decorators
- `ci` - CI/CD-focused decorators

### **Deprecation Management**

**Clear migration paths**:
```bash
opal check-deprecated                     # Scan for deprecated usage
opal migrate @old-decorator @new-decorator # Automated migration
```

**Sunset versions**:
- **Warn in version N**: Deprecation warnings in current version
- **Remove in version N+1**: Removal in next major version unless `--allow-deprecated`
- **Clear migration**: Documentation with exact replacement syntax

## Example: Well-Designed Decorator Composition

```opal
var ENV = "production"
var TIMEOUT = 10m

deploy: @timeout(TIMEOUT) {
    @log("Starting deployment to @var(ENV)", level="info")
    
    @retry(attempts=3, delay=5s) {
        @when(ENV) {
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
- Clear nesting hierarchy (timeout → retry → conditional → execution)
- Named parameters throughout
- Logical block structure
- Readable variable interpolation
- Mixed decorator types working together

## Tooling Configuration

### **Starter Configuration**

`.opalrc` example for teams:
```yaml
lint:
  rules:
    D001: error   # Chain complexity → require blocks
    D002: error   # Unknown decorators
    D003: error   # Positional when named exists
  style:
    arg_case: lower_snake
    enum:
      log.level: [trace,debug,info,warn,error]
      timeout.signal: [TERM,KILL,INT,HUP]
fmt:
  max_line: 100
  wrap_blocks: true
  nesting_order: [timeout,retry,try,when,parallel,log,shell]
```

### **CI Integration**

**Pre-commit hook** (5-line setup):
```bash
#!/bin/bash
opal lint --strict || exit 1
opal fmt --check || exit 1
opal check-docs || exit 1  # Verify Description() + Examples()
```

**CI pipeline**:
```bash
opal lint --strict    # Fail on D001-D003, warn on deprecations
opal fmt --check      # Verify formatting consistency
opal check-docs       # Enforce Description() + Examples() on all decorators
```

### **Development Workflow**

```bash
opal lint --fix       # Auto-fix D002 and D003 issues
opal fmt              # Format decorator composition
opal check-deprecated # Scan for deprecated decorator usage
opal migrate @old @new # Automated migration for renamed decorators
```

This approach keeps decorator composition clean and discoverable while avoiding namespace ceremony until actually needed.