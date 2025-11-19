# Opal Coding Style Guide

Opal's coding style is inspired by **Tiger Style** from TigerBeetle, adapted for Go. Our design goals are **safety**, **performance**, and **developer experience**, in that order.

## The Essence

> "The design is not just what it looks like and feels like. The design is how it works." — Steve Jobs

Style is not about aesthetics. Style is about how code works. Good style advances our design goals: making code safer, faster, and easier to understand.

## Safety First: Fail-Fast with Contracts

### Assertions Are a Force Multiplier

Assertions detect programmer errors early, before they cause silent corruption or catastrophic failures. We use assertions aggressively:

- **Minimum 2 assertions per function** (preconditions, postconditions, invariants)
- **Assert what you expect AND what you don't expect** (positive and negative space)
- **Pair assertions across boundaries** (validate before write, validate after read)
- **Split compound assertions** for clarity: `assert(a); assert(b);` not `assert(a && b);`

### Contract Assertions: Preconditions and Postconditions

Use the `invariant` package to express function contracts:

```go
// INPUT CONTRACT (preconditions)
invariant.NotNil(event, "event")
invariant.InRange(p.pos, 0, len(p.events)-1, "position")

// ... function work ...

// OUTPUT CONTRACT (postconditions)
invariant.Positive(step.ID, "step.ID")
invariant.Postcondition(step.Decorator != "", "step must have decorator")
```

**Why this matters:**
- Preconditions document what callers must provide
- Postconditions document what the function guarantees
- Violations are caught immediately, not silently
- Contracts are self-documenting code

### Invariants: Loop Progress and State Consistency

Every loop must make progress. Every state transition must be valid:

```go
// INVARIANT: position must advance in loop
prevPos := p.pos
for p.pos < len(p.events) {
    // ... process event ...
    invariant.Invariant(p.pos > prevPos, 
        "planner stuck at position %d (event: %v)", 
        prevPos, p.events[prevPos])
    prevPos = p.pos
}
```

**Why this matters:**
- Catches infinite loops immediately
- Prevents silent hangs in production
- Makes loop logic explicit and verifiable

### Paired Assertions Across Boundaries

When data crosses a boundary (write to disk, read from disk, encode, decode), assert validity on both sides:

```go
// Before write: validate data
invariant.Precondition(len(data) > 0, "data must not be empty")
invariant.Precondition(data.IsValid(), "data must be valid")
writeToFile(data)

// After read: validate data again
readData := readFromFile()
invariant.Postcondition(len(readData) > 0, "read data must not be empty")
invariant.Postcondition(readData.IsValid(), "read data must be valid")
```

**Why this matters:**
- Catches corruption at boundaries
- Detects bugs in serialization/deserialization
- Ensures data integrity across system boundaries

## Performance: Think First, Measure Later

### Back-of-the-Envelope Sketches

Before implementing, sketch performance characteristics:

- **Network**: How many round trips? Can we batch?
- **Disk**: How many I/O operations? Can we amortize?
- **Memory**: How much allocation? Can we pre-allocate?
- **CPU**: What's the algorithmic complexity? Can we optimize?

The best time to solve performance is in design, when you can't measure yet but can reason about it.

### Optimize for the Slowest Resource First

After compensating for frequency of use, optimize in this order:

1. **Network** (slowest, most expensive)
2. **Disk** (slower than memory, expensive)
3. **Memory** (faster than disk, but still expensive)
4. **CPU** (fastest, optimize last)

### Batching and Amortization

Batch operations to amortize costs:

```go
// ❌ BAD: One I/O per item
for _, item := range items {
    writeToFile(item)  // N disk operations
}

// ✅ GOOD: Batch writes
batch := make([]Item, 0, len(items))
for _, item := range items {
    batch = append(batch, item)
}
writeToFile(batch)  // 1 disk operation
```

## Developer Experience: Clarity and Precision

### Naming: Get the Nouns and Verbs Right

Names should capture what a thing is or does:

- Use `snake_case` for functions, variables, files
- Don't abbreviate unless it's a loop counter or matrix index
- Add units or qualifiers to variable names: `latency_ms_max` not `max_latency_ms`
- Use related names with same character count so they line up: `source` and `target` not `src` and `dest`

**Good names:**
```go
latency_ms_max := 100
latency_ms_min := 10
step_id := 42
command_text := "echo hello"
```

**Bad names:**
```go
max_lat := 100
min_lat := 10
id := 42
cmd := "echo hello"
```

### Comments: Explain Why, Not What

Code shows what. Comments explain why:

```go
// ❌ BAD: Explains what the code does
// Increment the counter
counter++

// ✅ GOOD: Explains why
// Increment counter to track number of steps executed
counter++

// ✅ GOOD: Explains non-obvious design decision
// Use arena allocation instead of individual allocations
// to reduce GC pressure and improve cache locality
arena := NewArena()
```

### Simplicity and Elegance

Simplicity is not the first attempt, it's the hardest revision:

> "Simplicity and elegance are unpopular because they require hard work and discipline to achieve" — Edsger Dijkstra

- Prefer explicit over implicit
- Prefer simple over clever
- Prefer readable over compact
- Prefer correct over fast (optimize later)

### Limits on Everything

Put explicit limits on everything:

```go
// ❌ BAD: Unbounded
for {
    items := queue.Get()  // Could grow forever
    process(items)
}

// ✅ GOOD: Bounded
const MaxBatchSize = 1000
for {
    items := queue.GetUpTo(MaxBatchSize)  // Bounded
    process(items)
}
```

**Why this matters:**
- Prevents tail latency spikes
- Makes performance predictable
- Catches bugs early (fail-fast)

## Go-Specific Conventions

### Error Handling: User Errors vs Programming Errors

**User errors** (return error):
- Invalid input: `command not found`
- Runtime failures: `disk full`
- Expected conditions: `file not found`

**Programming errors** (panic):
- Invariant violations: `position not advancing`
- Precondition failures: `nil pointer`
- Postcondition failures: `invalid output`

```go
// ❌ BAD: Returning error for programming bug
if step.ID == 0 {
    return nil, errors.New("step ID is zero")  // Should panic
}

// ✅ GOOD: Panic for programming bug
if step.ID == 0 {
    invariant.Postcondition(false, "step ID must not be zero")
}
```

### Use the Invariant Package

```go
import "github.com/opal-lang/opal/runtime/invariant"

func Process(data []byte) error {
    // Preconditions: what we require from caller
    invariant.NotNil(data, "data")
    invariant.Precondition(len(data) > 0, "data must not be empty")
    
    // ... work ...
    
    // Postconditions: what we guarantee to caller
    result := compute(data)
    invariant.Postcondition(result != nil, "result must not be nil")
    
    return nil
}
```

### Formatting and Style

- Run `gofmt` (Go's standard formatter)
- Use 4 spaces for indentation (more visible than 2)
- Hard limit: 100 columns per line
- Add braces to `if` statements unless single line
- Use `CamelCase` for exported names, `snake_case` for unexported

## Testing: Exhaustive and Exact

### Test Complete Output, Not Partial

```go
// ❌ BAD: Lazy partial test
assert.Contains(output, "success")

// ✅ GOOD: Complete exact test
expected := "Operation completed successfully with 3 items processed\n"
if diff := cmp.Diff(expected, actual); diff != "" {
    t.Errorf("Output mismatch (-want +got):\n%s", diff)
}
```

### Test Both Valid and Invalid Data

```go
// ✅ GOOD: Test positive and negative space
func TestValidInput(t *testing.T) {
    result, err := Process(validData)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result == nil {
        t.Fatal("result must not be nil")
    }
}

func TestInvalidInput(t *testing.T) {
    result, err := Process(nil)
    if err == nil {
        t.Fatal("expected error for nil input")
    }
    if result != nil {
        t.Fatal("result must be nil on error")
    }
}
```

### Test Invariant Violations

```go
func TestInvariantStuckLoop(t *testing.T) {
    defer func() {
        r := recover()
        if r == nil {
            t.Fatal("expected panic for stuck loop")
        }
        msg := fmt.Sprintf("%v", r)
        if !strings.Contains(msg, "INVARIANT VIOLATION") {
            t.Errorf("expected INVARIANT VIOLATION, got: %s", msg)
        }
    }()
    
    // Code that should trigger invariant violation
    // ...
}
```

## Summary: Three Design Goals

### 1. Safety
- Assert all contracts (preconditions, postconditions, invariants)
- Fail-fast on programming errors
- Pair assertions across boundaries
- Test both valid and invalid data

### 2. Performance
- Sketch performance before implementing
- Optimize slowest resources first
- Batch operations to amortize costs
- Put explicit limits on everything

### 3. Developer Experience
- Get names right (nouns and verbs)
- Explain why, not what
- Prefer simplicity and clarity
- Make contracts explicit

## References

- [TigerBeetle's Tiger Style](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/TIGER_STYLE.md)
- [NASA's Power of Ten](https://spinroot.com/gerard/pdf/P10.pdf)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
