# Plan System Documentation

## Overview

The devcmd plan system provides a structured way to visualize what commands will be executed without actually running them. This is crucial for understanding complex command workflows, debugging decorator behavior, and providing transparency in CI/CD environments.

## Core Concepts

### Execution Plans

An **ExecutionPlan** represents the complete execution tree that would be run for a given command. It contains:
- **Steps**: The hierarchical tree of execution steps
- **Context**: Environment and configuration information
- **Summary**: Aggregate statistics about the plan

### Execution Steps

Each **ExecutionStep** represents a node in the execution tree:
- **Type**: The kind of step (shell, timeout, parallel, etc.)
- **Description**: Human-readable description
- **Command**: The actual command (for shell steps)
- **Children**: Nested steps for decorators and blocks
- **Metadata**: Additional context-specific information

### Plan Generation Flow

```
User Command → Parser → AST → IR Transform → runtime/plan.GenerateFromIR() → ExecutionPlan → Visualization
                                         ↳ Decorator.Describe() for individual decorators
```

## Plan Generation Modes

devcmd supports two distinct plan generation modes to balance development speed with production reliability:

### Quick Plans (Default)

**Purpose**: Fast development feedback with immediate preview of command structure

**Command**: `devcmd run <command> --dry-run`

**Behavior**:
- Shows cached values for resolved decorators
- Displays placeholders for expensive operations (HTTP calls, API queries)
- Prioritizes speed over complete value resolution
- Ideal for development workflow and quick debugging

**Example**:
```bash
$ devcmd run deploy --dry-run
deploy:
├─ @timeout {5m timeout}
│  └─ @parallel {2 concurrent}
│     ├─ docker build -t myapp¹:@http(get-version)² ./build³
│     └─ kubectl apply -f ./k8s/staging.yaml
└─ echo "Deployment complete"

Legend: Variable mappings (¹ ² ³) shown below
{¹IMAGE_NAME=myapp, ²@http(get-version)=<will resolve at runtime>(~exec), ³BUILD_DIR=./build}
```

### Resolved Plans (Complete Resolution)

**Purpose**: Generate fully deterministic execution plans with all values resolved and frozen

**Commands**: 
- `devcmd run <command> --dry-run --resolve`
- `devcmd plan <command> --output <file>.plan`

**Behavior**:
- Forces resolution of ALL value decorators, including expensive operations
- Creates frozen execution context that can be reused
- Generates plan files for offline or deterministic execution
- Ideal for production deployment and CI/CD pipelines

**Example**:
```bash
$ devcmd run deploy --dry-run --resolve
deploy:
├─ @timeout {5m timeout}
│  └─ @parallel {2 concurrent}
│     ├─ docker build -t myapp¹:v2.1.4² ./build³
│     └─ kubectl apply -f ./k8s/staging.yaml
└─ echo "Deployment complete"

Legend: Variable mappings (¹ ² ³) shown below
{¹IMAGE_NAME=myapp, ²VERSION=v2.1.4(http), ³BUILD_DIR=./build}
```

### Plan Execution

**Purpose**: Execute commands using pre-resolved plan files with frozen values

**Command**: `devcmd exec <file>.plan`

**Behavior**:
- Loads frozen execution context from plan file
- **Never re-resolves values** - uses only pinned values from plan
- Deterministic execution matching the generated plan exactly
- Works offline when external dependencies are unavailable
- Validates environment fingerprint and warns on drift

**Lock File Semantics**: Plan files act as lock files - once values are resolved and pinned at plan time, execution uses exactly those frozen values, never re-resolving nondeterministic values like `@timestamp()` or `@http()`.

## Plan Visualization

### Tree Format (Default)

The default plan output uses a tree structure with aesthetic formatting:

```
build:
├─ @timeout {30s timeout}
│  ├─ @parallel {3 concurrent}
│  │  ├─ npm run lint
│  │  ├─ npm run typecheck
│  │  └─ npm run test
│  └─ npm run build
└─ echo "Build complete"
```

### Color Coding

When colors are enabled (`--color=auto` or `--color=always`):
- **Shell commands**: Default text color
- **Decorators**: Colored based on type
  - `@parallel`: Yellow
  - `@timeout`: Cyan
  - `@retry`: Yellow
  - `@when`: Cyan
- **Metadata**: Gray text in braces
- **Value decorator markers**: Gray/dim superscript numbers (¹ ² ³)
- **Value decorator metadata**: Gray text showing variable mappings

### Metadata Display

Important execution parameters are shown inline:
- Timeout durations: `{30s timeout}`
- Parallel concurrency: `{3 concurrent}`
- Retry attempts: `{3 attempts, 5s delay}`
- Conditional evaluation: `{OS = linux → linux-build}`
- Value decorator mappings: `{¹IMAGE_NAME=myapp, ²VERSION=v2.1.4(http)}`

### Value Decorator Markers

Commands containing value decorators (@var and @env) use superscript markers to show variable expansion:

**Marker System**:
- **Superscript numbers**: ¹ ² ³ ⁴ ⁵ ⁶ ⁷ ⁸ ⁹ (Unicode characters)
- **Fallback notation**: (1) (2) (3) for environments without Unicode support
- **Metadata format**: `{¹VARIABLE=value, ²ENV_KEY=value(source)}`

**Source Indicators**:
- `VERSION=v1.2.3` - Value from environment or variable
- `VERSION=latest(default)` - Default value used  
- `VERSION=v2.1.4(http)` - Value resolved from external source
- `@http(get-version)=<will resolve at runtime>(~exec)` - Runtime resolution placeholder

**Example**:
```
deploy:
└─ docker push myapp¹:v2.1.4² to registry.io³
   {¹IMAGE_NAME=myapp, ²VERSION=v2.1.4(http), ³REGISTRY=registry.io}
```

### Value Decorator Metadata Schema

Each value decorator in plan output includes standardized metadata fields for debugging and tooling:

**Standard Fields**:
- `nondeterministic: "true|false"` - Whether value could change between runs
- `resolve_at: "plan|exec"` - Where value is/will be resolved
- `pinned: "plan|""` - Set to "plan" only in resolved plans (frozen values)
- `cache_age: "12s"` - Age of cached value (for cached entries)
- `source: "environment|http|config|default"` - Value source type

**Quick Plan Example** (runtime resolution):
```json
{
  "decorator": "http",
  "nondeterministic": "true",
  "resolve_at": "exec",
  "pinned": ""
}
```

**Resolved Plan Example** (frozen value):
```json
{
  "decorator": "http", 
  "nondeterministic": "true",
  "resolve_at": "plan",
  "pinned": "plan",
  "resolved_at": "2024-01-15T10:30:00Z",
  "source": "http"
}
```

**Cached Value Example** (within TTL):
```json
{
  "decorator": "dns",
  "nondeterministic": "false", 
  "resolve_at": "plan",
  "pinned": "",
  "cache_age": "45s",
  "source": "dns"
}
```

## Decorator Implementation Guide

### The Describe() Method

Every decorator must implement the `Describe()` method to support plan generation:

```go
func (d *MyDecorator) Describe(
    ctx *decorators.Ctx,
    args []decorators.DecoratorParam,
    inner plan.ExecutionStep,
) plan.ExecutionStep {
    // Generate and return the execution step
}
```

### Basic Implementation Pattern

```go
func (d *TimeoutDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
    // 1. Extract and validate parameters
    duration, err := d.extractDuration(args)
    if err != nil {
        return plan.ExecutionStep{
            Type:        plan.StepTimeout,
            Description: fmt.Sprintf("@timeout(<error: %v>)", err),
        }
    }

    // 2. Build the execution step
    return plan.ExecutionStep{
        Type:        plan.StepTimeout,
        Description: fmt.Sprintf("@timeout(%v)", duration),
        Command:     fmt.Sprintf("timeout %v", duration), // Optional shell equivalent
        Children:    []plan.ExecutionStep{inner},         // Wrapped commands
        Metadata: map[string]string{
            "decorator": "timeout",
            "duration":  duration.String(),
        },
        Timing: &plan.TimingInfo{
            Timeout: &duration,
        },
    }
}
```

### Guidelines for Decorator Authors

#### 1. Choose the Appropriate StepType

Map your decorator to the correct `StepType`:
- `StepShell`: Direct shell commands
- `StepTimeout`: Commands with timeout
- `StepParallel`: Concurrent execution
- `StepRetry`: Retry logic
- `StepConditional`: Pattern-based conditions
- `StepTryCatch`: Error handling
- `StepSequence`: Sequential execution

#### 2. Provide Clear Descriptions

The description should be concise and informative:

```go
// Good - includes key parameters
Description: fmt.Sprintf("@retry(attempts=%d, delay=%v)", attempts, delay)

// Bad - too generic
Description: "retry decorator"
```

#### 3. Include Relevant Metadata

Add metadata that helps users understand the execution:

```go
Metadata: map[string]string{
    "decorator":     "parallel",
    "concurrency":   fmt.Sprintf("%d", concurrency),
    "failFast":      fmt.Sprintf("%v", failOnFirst),
    "estimatedTime": "30s",
}
```

#### 4. Handle Nested Execution

Always include child steps for block decorators:

```go
return plan.ExecutionStep{
    Type:        plan.StepParallel,
    Description: "@parallel",
    Children:    []plan.ExecutionStep{inner}, // Must include wrapped commands
}
```

#### 5. Show Environment Evaluation

For conditional decorators, show what will actually execute:

```go
func (d *WhenDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
    // Evaluate condition against current environment
    variable := args[0].Value.(string)
    currentValue := os.Getenv(variable)
    selectedBranch := d.evaluateBranch(currentValue, args[1:])
    
    return plan.ExecutionStep{
        Type:        plan.StepConditional,
        Description: fmt.Sprintf("@when(%s)", variable),
        Condition: &plan.ConditionInfo{
            Variable: variable,
            Evaluation: plan.ConditionResult{
                CurrentValue:   currentValue,
                SelectedBranch: selectedBranch,
                Reason:        fmt.Sprintf("%s matches pattern %s", currentValue, selectedBranch),
            },
            Branches: d.getBranchInfo(args[1:]),
        },
        Children: d.getSelectedBranchSteps(selectedBranch),
    }
}
```

#### 6. Respect UI Flags

Adapt plan detail based on UI configuration:

```go
func (d *MyDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
    step := plan.ExecutionStep{
        Type:     plan.StepSequence,
        Children: []plan.ExecutionStep{inner},
    }
    
    // Adjust detail level based on flags
    if ctx.UI != nil && ctx.UI.Verbose {
        // Include detailed metadata in verbose mode
        step.Description = fmt.Sprintf("@mydecorator(detailed info: %v)", args)
        step.Metadata = d.getDetailedMetadata(args)
    } else if ctx.UI != nil && ctx.UI.Quiet {
        // Minimal description in quiet mode
        step.Description = "@mydecorator"
    } else {
        // Standard description
        step.Description = fmt.Sprintf("@mydecorator(%v)", d.getKeyParam(args))
    }
    
    return step
}
```

## Best Practices for Plan Output

### 1. Consistency

Maintain consistent formatting across all decorators:
- Use `@decorator` prefix for decorator descriptions
- Show parameters in parentheses: `@timeout(30s)`
- Use braces for metadata: `{3 concurrent}`

### 2. Readability

Keep descriptions concise but informative:
- Truncate long commands at ~80 characters
- Use ellipses (...) for truncated content
- Group related information together

### 3. Actionability

Help users understand what will happen:
- Show actual values, not placeholders
- Include timing estimates when available
- Highlight conditional branches that will execute

### 4. Error Handling

Show errors clearly in plan output:

```go
if err != nil {
    return plan.ExecutionStep{
        Type:        plan.StepTimeout,
        Description: fmt.Sprintf("@timeout(<error: %v>)", err),
        Metadata: map[string]string{
            "error": err.Error(),
        },
    }
}
```

## Advanced Plan Features

### Timing Information

Include timing estimates to help users understand execution time:

```go
Timing: &plan.TimingInfo{
    Timeout:          &timeout,
    EstimatedTime:    &estimated,
    RetryAttempts:    3,
    RetryDelay:       &delay,
    ConcurrencyLimit: 5,
}
```

### Graph Edges (Future)

For complex workflows, edges can represent relationships:

```go
plan.AddEdge(plan.PlanEdge{
    FromID: step1.ID,
    ToID:   step2.ID,
    Kind:   plan.EdgeOnSuccess, // && operator
    Label:  "on success",
})
```

### Plan Hashing

Plans can be hashed for comparison and caching:

```go
hash := executionPlan.GraphHash()
// Use for cache keys or change detection
```

## Examples

### Simple Command Plan

```go
// Command: build: npm run build
plan.ExecutionStep{
    Type:        plan.StepShell,
    Description: "Execute: npm run build",
    Command:     "npm run build",
}
```

Output:
```
build:
└─ npm run build
```

### Nested Decorator Plan

```go
// Command: deploy: @timeout("5m") { @parallel { task1; task2 } }
plan.ExecutionStep{
    Type:        plan.StepTimeout,
    Description: "@timeout(5m)",
    Timing:      &plan.TimingInfo{Timeout: &fiveMinutes},
    Children: []plan.ExecutionStep{{
        Type:        plan.StepParallel,
        Description: "@parallel",
        Children: []plan.ExecutionStep{
            {Type: plan.StepShell, Command: "task1"},
            {Type: plan.StepShell, Command: "task2"},
        },
    }},
}
```

Output:
```
deploy:
└─ @timeout {5m timeout}
   └─ @parallel {2 concurrent}
      ├─ task1
      └─ task2
```

### Conditional Plan

```go
// Command: build: @when("OS", "linux": "make", "darwin": "xcodebuild")
plan.ExecutionStep{
    Type:        plan.StepConditional,
    Description: "@when(OS)",
    Condition: &plan.ConditionInfo{
        Variable: "OS",
        Evaluation: plan.ConditionResult{
            CurrentValue:   "linux",
            SelectedBranch: "linux",
            Reason:        "OS=linux matches pattern 'linux'",
        },
    },
    Children: []plan.ExecutionStep{
        {Type: plan.StepShell, Command: "make"},
    },
}
```

Output:
```
build:
└─ @when {OS = linux → linux}
   └─ make
```

## Testing Plan Output

### Unit Testing

Test your decorator's Describe() method:

```go
func TestDecoratorDescribe(t *testing.T) {
    decorator := NewMyDecorator()
    ctx := &decorators.Ctx{UI: &decorators.UIConfig{Verbose: true}}
    args := []decorators.DecoratorParam{{Name: "param", Value: "value"}}
    inner := plan.ExecutionStep{Type: plan.StepShell, Command: "echo test"}
    
    result := decorator.Describe(ctx, args, inner)
    
    assert.Equal(t, plan.StepSequence, result.Type)
    assert.Contains(t, result.Description, "@mydecorator")
    assert.Len(t, result.Children, 1)
    assert.Equal(t, "echo test", result.Children[0].Command)
}
```

### Integration Testing

Test complete plan generation:

```go
func TestPlanGeneration(t *testing.T) {
    // Parse command
    ast := parser.Parse("build: @timeout(\"30s\") { npm run build }")
    
    // Generate plan
    executionPlan := engine.GeneratePlan(ast, &Config{DryRun: true})
    
    // Verify structure
    assert.Len(t, executionPlan.Steps, 1)
    assert.Equal(t, plan.StepTimeout, executionPlan.Steps[0].Type)
    
    // Verify output
    output := executionPlan.String()
    assert.Contains(t, output, "@timeout")
    assert.Contains(t, output, "npm run build")
}
```

## Migration Guide

For existing decorators without Describe() implementation:

1. **Add the method signature**:
```go
func (d *MyDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
```

2. **Extract parameters** (reuse from WrapCommands):
```go
params := d.extractParams(args)
```

3. **Build the step**:
```go
return plan.ExecutionStep{
    Type:        d.getStepType(),
    Description: d.formatDescription(params),
    Children:    []plan.ExecutionStep{inner},
    Metadata:    d.getMetadata(params),
}
```

4. **Test the output**:
```bash
devcmd run mycommand --dry-run --verbose
```

## Value Decorator Display

Value decorators (@var and @env) require special handling in plan output since they are embedded within shell command text rather than being separate IR nodes. The plan system uses a **superscript marker approach** to clearly show variable expansion while maintaining readability.

### Display Format

For commands containing value decorators, the plan shows:
1. **Expanded command** with actual values and superscript markers
2. **Variable mapping** showing the correspondence between markers and variables

### Example Output

**Command definition:**
```
var BUILD_DIR = "./build"
var IMAGE_NAME = "myapp"

deploy: docker build -t @var(IMAGE_NAME):@env(VERSION,latest) @var(BUILD_DIR) && docker push @var(IMAGE_NAME):@env(VERSION,latest) --registry @env(DOCKER_REGISTRY)
```

**Plan output:**
```
deploy:
└─ docker build -t myapp¹:v1.2.3² ./build³ && docker push myapp¹:v1.2.3² --registry docker.io⁴
   {¹IMAGE_NAME=myapp, ²VERSION=v1.2.3(default), ³BUILD_DIR=./build, ⁴DOCKER_REGISTRY=docker.io}
```

### Marker System

- **Superscript numbers**: ¹ ² ³ ⁴ ⁵ ⁶ ⁷ ⁸ ⁹ (Unicode characters)
- **Fallback notation**: (1) (2) (3) for environments without Unicode support
- **Metadata format**: `{¹VARIABLE_NAME=value, ²ENV_KEY=value(source), ...}`

### Source Indicators

Environment variables show their source in the metadata:
- `VERSION=v1.2.3` - Value from environment
- `VERSION=latest(default)` - Default value used
- `VERSION=staging(captured)` - Value from frozen environment snapshot

### Benefits

- ✅ **Visual mapping** between expanded values and their variable sources
- ✅ **Readable commands** showing actual execution with clear markers
- ✅ **Complete context** including defaults and environment sources
- ✅ **Compact format** that scales with command complexity
- ✅ **Debugging support** for variable expansion issues

### Implementation Notes

Value decorator expansion in plans should:
1. Expand all @var and @env placeholders using current context
2. Track replacement positions and assign sequential markers
3. Generate metadata line with variable-to-marker mapping
4. Handle expansion errors gracefully with clear error indicators

## Plan Equivalence and Stability

### GraphHash for Structural Comparison

Plans include a `GraphHash()` method that generates deterministic hashes for structural comparison in tests and caching:

```go
hash := executionPlan.GraphHash()  // Returns 16-char hex string
```

**Hash Stability**:
- **Included**: Step types, commands, decorator names, static parameters, edges
- **Excluded**: Timestamps, process IDs, resolved values, cache ages
- **Deterministic**: Same IR structure always produces same hash

**Use Cases**:
- Test assertions for plan structure equivalence
- Caching plan generation results
- Change detection in CI pipelines

**Example**:
```go
// These plans have same GraphHash (different resolved values)
plan1 := GeneratePlan(cmd, quickCtx)     // @http() → <runtime>
plan2 := GeneratePlan(cmd, resolvedCtx)  // @http() → "v2.1.4"
assert.Equal(t, plan1.GraphHash(), plan2.GraphHash())
```

### Secret Redaction Policy

Value decorators can mark sensitive output for redaction in plan display:

**Default Redaction Patterns** (case-insensitive):
- Keys matching: `(?i)(token|secret|password|key|auth)`
- Environment variables: `*_TOKEN`, `*_SECRET`, `*_KEY`, `*_PASSWORD`

**Decorator Override**:
```go
func (d *SecretDecorator) Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep {
    return plan.ExecutionStep{
        Description: fmt.Sprintf("@secret(%s) → <redacted>", key),
        Command:     actualValue,  // Real value for execution
        Metadata: map[string]string{
            "decorator": "secret",
            "redact":    "true",     // Force redaction
            "source":    "vault",
        },
    }
}
```

**Plan Output**:
```
deploy:
└─ curl -H "Authorization: Bearer <redacted>¹" api.internal
   {¹AUTH_TOKEN=<redacted>}
```

## CLI Validation and Safety Rails

### Flag Validation

The `--resolve` flag is only valid with `--dry-run` and will be rejected otherwise:

```bash
$ devcmd run deploy --resolve
❌ Error: --resolve flag requires --dry-run
💡 Use: devcmd run deploy --dry-run --resolve

$ devcmd run deploy --dry-run --resolve
✅ Generating resolved plan with all values frozen...
```

### Environment Fingerprint Validation

Plan files include environment fingerprints for safety validation:

**Plan File Format**:
```json
{
  "command": "deploy",
  "env_fingerprint": "sha256:abc123...",
  "resolved_context": {...},
  "execution_plan": {...}
}
```

**Execution Validation**:
```bash
$ devcmd exec deploy.plan
⚠️  Warning: Environment fingerprint mismatch
   Plan: sha256:abc123...
   Current: sha256:def456...
   
   Environment variables changed since plan generation.
   Continue anyway? (y/N)
```

**Validation Policy** (configurable):
- **Default**: Warn on fingerprint mismatch, allow user choice
- **Strict mode**: `--strict` flag fails immediately on mismatch
- **CI mode**: `--ci` flag treats warnings as failures

### Execution Safety Rails

**Plan Execution Constraints**:
1. **Never re-resolve nondeterministic values** during `devcmd exec`
2. **Use only pinned values** from plan file for deterministic execution
3. **Validate environment fingerprint** before execution
4. **Fail fast** if plan file is corrupted or incompatible

**CI/CD Recommendations**:
- Use `--dry-run --resolve` to generate deterministic plans
- Store plan files as deployment artifacts
- Execute using `devcmd exec plan.json` for reproducible deployments

## Summary

The plan system provides essential visibility into command execution without side effects. By following these guidelines, decorator authors can create informative, consistent plan output that helps users understand and debug their command workflows.

Key takeaways:
- Every decorator must implement Describe() for plan support
- Use appropriate StepTypes and include all relevant metadata
- Show actual evaluated values, not just templates
- Value decorators use superscript markers for clear variable mapping
- Respect UI flags for output verbosity
- Test plan output as part of decorator development