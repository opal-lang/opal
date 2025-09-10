# Devcmd Architecture Overview - CommandSeq-Based Unified Execution

**Last Updated**: Phase 1 Implementation (IR-based unified execution)

## Core Principle

**A decorator is just another CommandSeq with nesting CommandSeq.**

Decorators receive raw command sequences (`CommandSeq`) and decide how to execute them. This enables maximum flexibility: sequential execution, parallel execution, retries, conditional branching, and environment modifications.

## Key Architecture Decisions

### 0. Nesting Rules (No Inline Chaining)

Decorators can be nested but **cannot be chained inline**:

```devcmd
// ✅ Correct - explicit nesting with braces
deploy: @timeout(5m) {
    @retry(3) {
        kubectl apply -f k8s/
    }
}

// ❌ Invalid - no inline chaining syntax
deploy: @timeout(5m) @retry(3) {
    kubectl apply -f k8s/
}
```

**Rule**: Each decorator requires its own explicit block structure for nesting.

### 1. CommandSeq-Based Decorator Interfaces

All decorators work with `CommandSeq` (command sequences) rather than closure functions:

```go
// BlockDecorator receives CommandSeq and decides execution strategy
type BlockDecorator interface {
    Name() string
    WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult
}

// PatternDecorator receives all branches and selects which to execute
type PatternDecorator interface {
    Name() string
    SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult
}
```

### 2. Standard Values for Pattern Matching

- **@when wildcard**: Use `default` (not `*`)
- **@try error handling**: Use `catch` (not `error`)

```devcmd
deploy: @when(ENV) {
    production: kubectl apply -f prod/
    staging: kubectl apply -f staging/
    default: echo "Unknown environment"  // wildcard
}

backup: @try {
    main: pg_dump mydb > backup.sql
    catch: echo "Backup failed"         // error handling
    finally: rm -f temp_files
}
```

### 3. IR Types Use CommandSeq Directly

The Intermediate Representation uses `CommandSeq` throughout:

```go
// Wrapper (block decorator) contains CommandSeq directly
type Wrapper struct {
    Kind   string
    Params map[string]interface{}
    Inner  CommandSeq  // No longer Node - direct CommandSeq
}

// Pattern (pattern decorator) contains map of CommandSeq
type Pattern struct {
    Kind     string
    Params   map[string]interface{}
    Branches map[string]CommandSeq  // Each branch is CommandSeq
}
```

### 4. Unified Execution Pipeline

```
commands.cli → AST → IR → [Interpreter|Generator|Quick Plan|Resolved Plan]
                      ↓
                   Same decorator contracts
                   Same CommandSeq execution
                   Context-aware plan representation
                   Hybrid value decorator caching
```

**Execution Modes**:
- **Interpreter**: Real-time execution with streaming output
- **Generator**: AOT compilation to standalone binaries (Phase 2)
- **Quick Plan**: Fast preview with cached values and runtime placeholders
- **Resolved Plan**: Complete resolution with frozen execution context

## Implementation Status

### ✅ Completed (Phase 1)
- IR types updated to use CommandSeq directly
- AST→IR transformation uses map[string]CommandSeq
- NodeEvaluator implements CommandSeq-based execution
- Wrapper and Pattern evaluation working
- Value decorator expansion with hybrid caching (`@var`, `@env`)
- Two-tier plan system (quick plans vs resolved plans)
- Plan generation with superscript marker visualization
- Comprehensive decorator architecture documentation

### 🚧 Next Steps
- Complete shell operator edge cases (`&&`, `||`, `|`, `>>`)
- Implement comprehensive nested decorator tests
- Complete interpreter mode functionality
- Begin generated mode implementation (Phase 2)

## Decorator Execution Examples

### Block Decorators Control Execution Strategy

```go
// @parallel executes commands concurrently
func (p *ParallelDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult {
    return ctx.ExecParallel(commands.Steps, p.getMode(args))
}

// @retry re-executes the entire sequence on failure
func (r *RetryDecorator) WrapCommands(ctx *Ctx, args []DecoratorParam, commands CommandSeq) CommandResult {
    for attempt := 1; attempt <= r.getAttempts(args); attempt++ {
        result := ctx.ExecSequential(commands)
        if result.Success() { return result }
        time.Sleep(r.getDelay(args))
    }
    return lastResult
}
```

### Pattern Decorators Select Branches

```go
// @when evaluates condition and executes matching branch
func (w *WhenDecorator) SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult {
    condition := w.evaluateCondition(ctx, args)
    if commands, exists := branches[condition]; exists {
        return ctx.ExecSequential(commands)
    }
    if fallback, exists := branches["default"]; exists {
        return ctx.ExecSequential(fallback)  // Default branch
    }
    return CommandResult{ExitCode: 0}  // No matching branch
}
```

## Benefits of CommandSeq Architecture

1. **Maximum Flexibility**: Decorators can implement any execution strategy
2. **Consistent Interfaces**: Same decorator works in interpreter and generated modes
3. **Testability**: Easy to test with synthetic CommandSeq inputs
4. **Composability**: Natural nesting of decorators
5. **Performance**: Direct execution without closure overhead

## Next Phase Preview

Phase 2 will implement generated mode while maintaining the same CommandSeq-based decorator interfaces, ensuring semantic equivalence between interpreter and generated execution.

---

**Key Takeaway**: The CommandSeq-based architecture provides the foundation for unified execution across all devcmd modes while giving decorators maximum control over how commands are executed.