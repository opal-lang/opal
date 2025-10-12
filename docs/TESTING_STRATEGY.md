---
title: "Opal Testing Strategy"
audience: "Core Developers & Contributors"
summary: "Systematic testing approach for contract-based operations"
---

# Opal Testing Strategy

**Goal: Ridiculously stable contract-based operations through systematic testing**

**Audience**: Core developers and contributors implementing tests for the Opal runtime.

Build stability into the process, not just the code. Start with high-value tests that catch 80% of issues, then layer in comprehensive testing as the system matures.

## Testing Layer Model

```
SPECIFICATION.md → Defines guarantees
       ↓
ARCHITECTURE.md → Implements guarantees
       ↓
TESTING_STRATEGY.md → Verifies guarantees
       ↓
OBSERVABILITY.md → Monitors guarantees in production
```

**Each layer verifies the layer above it:**
- Tests verify architecture implements spec correctly
- Observability verifies tests caught real-world issues
- Feedback loop improves all layers

## Non-Negotiable Invariants

These protect the guarantees defined in [SPECIFICATION.md](SPECIFICATION.md) and must never break:

* **Resolved plans are execution contracts**: same input → identical plan bytes; execution refuses if structure/hash differs
* **Security invariant**: No raw secrets in plans/logs - only `<length:algorithm:hash>` placeholders 
* **Determinism**: Plan Seed Envelopes (PSE) for seeded randomness; forbid non-deterministic decorators in resolved plans
* **Crash isolation**: plugin failures can't crash the engine; bounded CPU/mem/time per step

## Testing Phases

### Core Tests (Implement Now)
Essential tests that catch 80% of issues with minimal effort.

#### Golden Plan Tests
**Canonical inputs → byte-exact plans (implements contract verification model)**
```bash
# Test structure
tests/golden/
├── simple-shell/
│   ├── input.cli           # Source: echo "hello"
│   ├── expected-quick.plan # Quick plan with deferred placeholders
│   └── expected-resolved.plan # Resolved plan (execution contract)
├── decorator-retry/
│   ├── input.cli           # @retry(attempts=3) { kubectl apply -f k8s/ }
│   ├── vars.env            # Environment for value decorators
│   ├── expected-quick.plan
│   └── expected-resolved.plan
├── control-flow/
│   ├── input.cli           # for/when/try patterns
│   ├── expected-quick.plan # Shows expanded plan-time control flow
│   └── expected-resolved.plan
```

**Implementation priority**: Start with 5-10 core scenarios, add more as features stabilize.

```go
func TestGoldenPlans(t *testing.T) {
    goldenTests := []struct {
        name string
        inputFile string
        envFile string // optional
    }{
        {"simple-shell", "simple-shell/input.cli", ""},
        {"value-decorators", "value-decorators/input.cli", "value-decorators/vars.env"},
        {"plan-time-expansion", "control-flow/input.cli", ""},
        // Add more as we go
    }
    
    for _, tt := range goldenTests {
        t.Run(tt.name, func(t *testing.T) {
            // Quick plan test (expensive decorators deferred)
            quickPlan := generateQuickPlan(tt.inputFile, tt.envFile)
            expectedQuick := readGoldenFile(tt.name + "/expected-quick.plan")
            assert.Equal(t, expectedQuick, quickPlan, "Quick plan mismatch")
            
            // Resolved plan test (execution contract with all values materialized)
            resolvedPlan := generateResolvedPlan(tt.inputFile, tt.envFile)
            expectedResolved := readGoldenFile(tt.name + "/expected-resolved.plan")
            assert.Equal(t, expectedResolved, resolvedPlan, "Resolved plan contract mismatch")
        })
    }
}
```

#### Parser Stability Tests
**Ensure parser doesn't crash on any input**
```go
func TestParserStability(t *testing.T) {
    // Test with known-good inputs
    validInputs := loadValidInputs()
    for _, input := range validInputs {
        t.Run("valid-"+input.name, func(t *testing.T) {
            _, err := parser.Parse(input.content)
            assert.NoError(t, err, "Parser should handle valid input")
        })
    }
    
    // Test with malformed inputs (should error gracefully, not crash)
    malformedInputs := loadMalformedInputs()
    for _, input := range malformedInputs {
        t.Run("malformed-"+input.name, func(t *testing.T) {
            defer func() {
                assert.Nil(t, recover(), "Parser should not panic on malformed input")
            }()
            
            _, err := parser.Parse(input.content)
            assert.Error(t, err, "Parser should return error for malformed input")
        })
    }
}
```

#### Basic Property Tests
**Plan diffs should be empty under cosmetic changes**
```go
func TestPlanStabilityUnderCosmetic(t *testing.T) {
    baseInput := `
deploy: {
    echo "hello"
    kubectl apply -f k8s/
}`

    variations := []string{
        // Extra whitespace
        `deploy: {
    echo "hello"
    kubectl apply -f k8s/
}`,
        // Different indentation
        `deploy: {
        echo "hello"
        kubectl apply -f k8s/
    }`,
        // Comments
        `deploy: {
    # Deploy the application
    echo "hello"
    kubectl apply -f k8s/
}`,
    }
    
    basePlan := generatePlan(baseInput)
    for i, variation := range variations {
        t.Run(fmt.Sprintf("variation-%d", i), func(t *testing.T) {
            varPlan := generatePlan(variation)
            assert.Equal(t, basePlan.Hash(), varPlan.Hash(), 
                "Plan hash should be identical for cosmetic changes")
        })
    }
}
```

### Advanced Tests (Build Up)
More sophisticated testing as the system matures.

#### Contract Verification Tests
**Verify resolved plans work as execution contracts per specification**
```go
func TestContractVerification(t *testing.T) {
    // Test the core specification principle: resolved plans are execution contracts
    testCases := []struct {
        name string
        sourceFile string
        environment map[string]string
        expectVerificationOutcome string
    }{
        {"unchanged-source", "deploy.cli", baseEnv, "ok"},
        {"source-changed", "deploy-modified.cli", baseEnv, "source_changed"},
        {"env-changed", "deploy.cli", modifiedEnv, "infra_mutated"},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Generate resolved plan (execution contract)
            resolvedPlan := generateResolvedPlan(tc.sourceFile, tc.environment)
            
            // Simulate time-delayed execution per spec section
            time.Sleep(100 * time.Millisecond) // Simulate plan→execute delay
            
            // Contract verification: replan and compare
            newPlan := generatePlanFromCurrentState(tc.sourceFile, tc.environment)
            outcome := verifyContract(resolvedPlan, newPlan)
            
            assert.Equal(t, tc.expectVerificationOutcome, outcome,
                "Contract verification must detect changes correctly")
                
            // Test verification failure messages are actionable per spec
            if outcome != "ok" {
                errorMsg := getVerificationError(resolvedPlan, newPlan)
                assert.Contains(t, errorMsg, "Expected:", "Error must show expected state")
                assert.Contains(t, errorMsg, "Actual:", "Error must show actual state") 
                assert.Contains(t, errorMsg, "Run 'opal plan", "Error must suggest remediation")
            }
        })
    }
}
```

#### Unified Decorator Conformance Suite
**Ensure all decorators follow the unified interface contracts from architecture**
```go
func TestUnifiedDecoratorConformance(t *testing.T) {
    // Test both value and execution decorators per unified architecture
    valueDecorators := registry.AllValueDecorators()
    execDecorators := registry.AllExecutionDecorators()
    
    for _, decorator := range valueDecorators {
        t.Run("value-"+decorator.Name(), func(t *testing.T) {
            testValueDecoratorConformance(t, decorator)
        })
    }
    
    for _, decorator := range execDecorators {
        t.Run("exec-"+decorator.Name(), func(t *testing.T) {
            testExecutionDecoratorConformance(t, decorator)
        })
    }
}

func testValueDecoratorConformance(t *testing.T, decorator ValueDecorator) {
    // Architecture requirement: ValueDecorators must be referentially transparent
    ctx := createTestContext()
    params := createValidParams(decorator)
    
    // Multiple resolutions should return identical values
    resolved1, err1 := decorator.Resolve(ctx, params)
    resolved2, err2 := decorator.Resolve(ctx, params)
    
    assert.NoError(t, err1)
    assert.NoError(t, err2)
    assert.Equal(t, resolved1.Hash(), resolved2.Hash(),
        "ValueDecorator must be referentially transparent (same hash)")
    
    // Test security invariant: placeholders only per specification
    testSecurityInvariants(t, decorator, resolved1)
    
    // Test expensive value decorator handling per architecture
    if decorator.IsExpensive() {
        // Expensive value decorators should be deferred in quick plans
        quickPlan := decorator.Plan(ctx, params)
        assert.Contains(t, quickPlan.String(), "¹@", 
            "Expensive value decorators should show deferred placeholders in quick plans")
    }
}

func testSecurityInvariants(t *testing.T, decorator interface{}, resolved Resolved) {
    // Specification security invariant: no raw secrets in plans/logs
    planString := resolved.String()
    
    // Test with known sensitive patterns
    sensitivePatterns := []string{
        "password123",
        "sk_live_abcd1234", 
        "-----BEGIN PRIVATE KEY-----",
        "AKIA[A-Z0-9]{16}", // AWS access key pattern
    }
    
    for _, pattern := range sensitivePatterns {
        assert.NotContains(t, planString, pattern,
            "Raw sensitive values must never appear in plans")
    }
    
    // Ensure placeholder format per specification: <length:algorithm:hash>
    if containsSensitiveValue(resolved) {
        assert.Regexp(t, `<\d+:[a-zA-Z0-9-]+:[a-f0-9]+>`, planString,
            "Sensitive values must use <length:algorithm:hash> placeholder format")
    }
}
```

#### Decorator Completion Model Testing
**Verify decorator completion before chain evaluation per architecture**
```go
func TestDecoratorCompletionModel(t *testing.T) {
    // Architecture principle: "Decorator Completion Model" - 
    // decorators execute their entire block before chain operators are evaluated
    
    testCases := []struct {
        name string
        input string
        retryExitCodes []int
        timeoutAfter time.Duration
        expectedBehavior string
    }{
        {
            name: "retry-completes-before-and",
            input: "@retry(attempts=3) { exit 1 } && echo 'success'",
            retryExitCodes: []int{1, 1, 1}, // all fail
            expectedBehavior: "retry_completes_then_and_skipped",
        },
        {
            name: "timeout-completes-before-or", 
            input: "@timeout(1s) { sleep 2 } || echo 'fallback'",
            timeoutAfter: 1 * time.Second,
            expectedBehavior: "timeout_completes_then_or_executes",
        },
        {
            name: "parallel-completes-before-pipe",
            input: "@parallel { echo 'a'; echo 'b' } | grep 'a'",
            expectedBehavior: "parallel_completes_then_pipe_processes_output",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            plan := generatePlan(tc.input)
            
            // Verify plan shows decorator completion before chain operators
            steps := plan.GetSteps()
            
            // Find decorator step and chain operator step
            var decoratorStep, chainStep *ExecutionStep
            for i, step := range steps {
                if strings.Contains(step.String(), "@") {
                    decoratorStep = &steps[i]
                } else if strings.Contains(step.String(), "&&") || 
                         strings.Contains(step.String(), "||") ||
                         strings.Contains(step.String(), "|") {
                    chainStep = &steps[i]
                }
            }
            
            assert.NotNil(t, decoratorStep, "Should find decorator step")
            assert.NotNil(t, chainStep, "Should find chain operator step")
            
            // Verify execution order: decorator must complete before chain
            assert.True(t, decoratorStep.Order < chainStep.Order,
                "Decorator must complete before chain operator per completion model")
        })
    }
}

func TestTryCatchNonDeterminism(t *testing.T) {
    // Specification: try/catch is the only non-deterministic construct
    input := `
deploy: {
    try {
        kubectl apply -f k8s/
        kubectl rollout status deployment/app
    } catch {
        kubectl rollout undo deployment/app  
    } finally {
        kubectl get pods
    }
}`
    
    plan := generatePlan(input)
    
    // Plan should show all possible paths (try, catch, finally)
    planStr := plan.String()
    assert.Contains(t, planStr, "try:", "Plan must show try path")
    assert.Contains(t, planStr, "catch:", "Plan must show catch path") 
    assert.Contains(t, planStr, "finally:", "Plan must show finally path")
    
    // Execution logs should show which path was taken
    result := executePlan(plan)
    executionLog := result.GetExecutionLog()
    
    // Should show which path was actually executed
    pathTaken := extractExecutionPath(executionLog)
    assert.Contains(t, []string{"try-success", "try-catch", "try-catch-finally"}, pathTaken,
        "Execution log must show which try/catch path was taken")
}
```

#### Plan-Time Expansion Testing  
**Verify control flow expands during plan generation per architecture**
```go
func TestPlanTimeExpansion(t *testing.T) {
    // Architecture principle: "Control flow expands during plan generation, not execution"
    // Specification: "for, if, when, try/catch cannot be chained with operators"
    
    testCases := []struct {
        name string
        input string
        variables map[string]interface{}
        expectedSteps []string
        expectedStepIDs []string
    }{
        {
            name: "for-loop-unrolling",
            input: `
deploy: {
    for service in @var.SERVICES {
        kubectl apply -f k8s/@var.service/
        kubectl rollout status deployment/@var.service
    }
}`,
            variables: map[string]interface{}{
                "SERVICES": []string{"api", "worker"},
            },
            expectedSteps: []string{
                "kubectl apply -f k8s/api/",
                "kubectl rollout status deployment/api", 
                "kubectl apply -f k8s/worker/",
                "kubectl rollout status deployment/worker",
            },
            expectedStepIDs: []string{
                "deploy.service[0].apply",
                "deploy.service[0].status",
                "deploy.service[1].apply", 
                "deploy.service[1].status",
            },
        },
        {
            name: "when-pattern-selection",
            input: `
deploy: {
    when @var.ENV {
        "production" -> {
            kubectl apply -f k8s/prod/
            kubectl scale --replicas=3 deployment/app
        }
        "staging" -> kubectl apply -f k8s/staging/
        else -> echo "Unknown environment: @var.ENV"
    }
}`,
            variables: map[string]interface{}{
                "ENV": "production",
            },
            expectedSteps: []string{
                "kubectl apply -f k8s/prod/",
                "kubectl scale --replicas=3 deployment/app",
            },
            expectedStepIDs: []string{
                "deploy.when[production].apply",
                "deploy.when[production].scale",
            },
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Generate plan with variables - control flow should expand at plan time
            plan := generatePlanWithVariables(tc.input, tc.variables)
            
            // Verify plan shows expanded steps with stable IDs per architecture
            steps := plan.GetSteps()
            
            assert.Equal(t, len(tc.expectedSteps), len(steps),
                "Plan should contain exact number of expanded steps")
                
            for i, step := range steps {
                assert.Equal(t, tc.expectedSteps[i], step.Command,
                    "Plan step command should match expected expansion")
                assert.Equal(t, tc.expectedStepIDs[i], step.ID,
                    "Step IDs must be stable and predictable per architecture")
            }
            
            // Architecture requirement: execution decorators receive static command lists
            planString := plan.String()
            assert.NotContains(t, planString, "for service in",
                "Plan should not contain unexpanded for loops")
            assert.NotContains(t, planString, "when @var.ENV",
                "Plan should not contain unexpanded when statements")
                
            // Verify no chaining allowed per specification
            invalidInputs := []string{
                "for service in @var.SERVICES { echo @var.service } && echo 'done'",
                "when @var.ENV { prod: kubectl apply } || echo 'failed'",
                "try { kubectl apply } catch { rollback } | tee log.txt",
            }
            
            for _, invalid := range invalidInputs {
                _, err := parseInput(invalid)
                assert.Error(t, err, "Control flow should not be chainable with operators")
                assert.Contains(t, err.Error(), "cannot be chained",
                    "Error should mention chaining restriction")
            }
        })
    }
}

#### Plan Seed Envelope (PSE) Testing
**Verify seeded determinism implementation per specification**
```go
func TestPlanSeedEnvelopeDeterminism(t *testing.T) {
    // Specification: PSE enables deterministic random generation within resolved plans
    
    testCases := []struct {
        name string
        input string
        regenKey string
        expectSameValue bool
    }{
        {
            name: "default-regeneration",
            input: `var TEMP_TOKEN = @random.password(length=16)`,
            regenKey: "", // Uses plan hash as key
            expectSameValue: false, // Different plans = different values
        },
        {
            name: "stable-regeneration-key",
            input: `var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v1")`,
            regenKey: "db-pass-prod-v1",
            expectSameValue: true, // Same regen_key = same value across plans
        },
        {
            name: "rotated-regeneration-key", 
            input: `var DB_PASS = @random.password(length=24, regen_key="db-pass-prod-v2")`,
            regenKey: "db-pass-prod-v2",
            expectSameValue: false, // Changed regen_key = new value
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Generate two resolved plans with PSE
            plan1 := generateResolvedPlanWithPSE(tc.input)
            plan2 := generateResolvedPlanWithPSE(tc.input)
            
            // Extract random values from plans (as placeholders)
            value1 := extractRandomValuePlaceholder(plan1)
            value2 := extractRandomValuePlaceholder(plan2)
            
            if tc.expectSameValue {
                assert.Equal(t, value1.Hash(), value2.Hash(),
                    "Same regen_key should produce same random value hash")
            } else {
                assert.NotEqual(t, value1.Hash(), value2.Hash(),
                    "Different plan/regen_key should produce different random value hash")
            }
            
            // Verify security invariant: no raw random values in plans
            planString1 := plan1.String()
            planString2 := plan2.String()
            
            assert.Regexp(t, `<\d+:sha256:[a-f0-9]+>`, planString1,
                "Random values must appear as placeholders in plans")
            assert.Regexp(t, `<\d+:sha256:[a-f0-9]+>`, planString2,
                "Random values must appear as placeholders in plans")
        })
    }
}

func TestPSEContractVerification(t *testing.T) {
    // Specification: PSE must work with contract verification
    
    input := `
deploy: {
    var API_KEY = @random.password(length=32, regen_key="api-key-v1")
    kubectl create secret generic api --from-literal=key=@var.API_KEY
}`
    
    // Generate resolved plan with PSE
    resolvedPlan := generateResolvedPlanWithPSE(input)
    
    // Simulate time-delayed execution
    time.Sleep(100 * time.Millisecond)
    
    // Contract verification: replan should match resolved plan structure
    newPlan := generateResolvedPlanWithPSE(input)
    
    // Structure should be identical
    assert.Equal(t, resolvedPlan.Structure(), newPlan.Structure(),
        "PSE plans should have identical structure")
        
    // Hash should be identical (same regen_key)
    assert.Equal(t, resolvedPlan.Hash(), newPlan.Hash(),
        "PSE with same regen_key should produce identical plan hash")
        
    // Verify execution using resolved plan
    result := executeResolvedPlan(resolvedPlan)
    assert.True(t, result.IsSuccess(), "PSE plan execution should succeed")
}

func TestPSENonDeterministicDetection(t *testing.T) {
    // Specification: forbid non-deterministic decorators in resolved plans
    
    invalidInputs := []string{
        `var CURRENT_TIME = @http.get("https://time-api.com/now")`,
        `var RANDOM_UUID = @system.uuid()`, // Non-seeded randomness
        `var TIMESTAMP = @time.now()`,
    }
    
    for _, input := range invalidInputs {
        t.Run("non-deterministic-"+input, func(t *testing.T) {
            // Attempt to generate resolved plan
            _, err := generateResolvedPlan(input)
            
            // Should fail with non-deterministic error
            assert.Error(t, err, "Non-deterministic decorators should be rejected")
            assert.Contains(t, err.Error(), "non-deterministic",
                "Error should mention non-deterministic nature")
            assert.Contains(t, err.Error(), "resolved plans",
                "Error should mention resolved plan restriction")
        })
    }
}

### Phase 3: Production Hardening (Advanced)
Comprehensive testing for production reliability.

#### Fuzzing
**Ensure parser/lexer never crash**
```go
func FuzzParser(f *testing.F) {
    // Seed with known inputs
    f.Add("deploy: echo hello")
    f.Add("var X = @env(\"TEST\")")
    f.Add("@retry(3) { kubectl apply -f k8s/ }")
    
    f.Fuzz(func(t *testing.T, input string) {
        defer func() {
            if r := recover(); r != nil {
                t.Errorf("Parser panicked on input %q: %v", input, r)
            }
        }()
        
        // Parser should never panic, even on garbage input
        _, err := parser.Parse(input)
        
        // Error is fine, panic is not
        if err != nil {
            // Ensure error is actionable
            assert.Contains(t, err.Error(), "line", 
                "Parse errors should include position information")
        }
    })
}
```

#### Soak Tests
**24-72h stability under load**
```go
func TestSoakStability(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping soak test in short mode")
    }
    
    duration := 24 * time.Hour
    if os.Getenv("SOAK_DURATION") != "" {
        duration, _ = time.ParseDuration(os.Getenv("SOAK_DURATION"))
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    var wg sync.WaitGroup
    errorChan := make(chan error, 1000)
    
    // Run multiple concurrent plan/resolve/execute cycles
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(worker int) {
            defer wg.Done()
            soakWorker(ctx, worker, errorChan)
        }(i)
    }
    
    wg.Wait()
    close(errorChan)
    
    // Collect and report any errors
    var errors []error
    for err := range errorChan {
        errors = append(errors, err)
    }
    
    if len(errors) > 0 {
        t.Errorf("Soak test found %d errors: %v", len(errors), errors[:min(5, len(errors))])
    }
}

func soakWorker(ctx context.Context, workerID int, errorChan chan<- error) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := runRandomPlanCycle(); err != nil {
                select {
                case errorChan <- err:
                case <-ctx.Done():
                    return
                }
            }
        }
    }
}
```

#### Resource Isolation Tests
**Verify CPU/memory/FD limits work**
```go
func TestResourceLimits(t *testing.T) {
    tests := []struct {
        name string
        decorator string
        input string
        expectTermination bool
        resourceType string
    }{
        {
            name: "cpu-bomb",
            decorator: "@shell",
            input: "while true; do :; done",
            expectTermination: true,
            resourceType: "cpu",
        },
        {
            name: "memory-bomb", 
            decorator: "@shell",
            input: "python -c 'x=[0]*10**9'",
            expectTermination: true,
            resourceType: "memory",
        },
        {
            name: "fd-bomb",
            decorator: "@shell", 
            input: "while true; do exec 3< /dev/null; done",
            expectTermination: true,
            resourceType: "fd",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            executor := NewResourceLimitedExecutor(ResourceLimits{
                MaxCPUPercent: 50,
                MaxMemoryMB: 100,
                MaxFileDescriptors: 100,
                TimeoutSeconds: 5,
            })
            
            result := executor.Execute(tt.input)
            
            if tt.expectTermination {
                assert.False(t, result.IsSuccess(), 
                    "Resource-intensive command should be terminated")
                assert.Contains(t, result.GetStderr(), tt.resourceType,
                    "Error should mention resource type")
            }
        })
    }
}
```

## CI Integration

### Stability Gate Checklist
Before merging any PR:

```yaml
# .github/workflows/stability.yml
name: Stability Gate
on: [pull_request]

jobs:
  golden-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Golden Plan Tests
        run: go test ./tests/golden/... -v
        
  parser-stability:
    runs-on: ubuntu-latest  
    steps:
      - name: Parser Stability
        run: go test ./runtime/parser/... -fuzz=FuzzParser -fuzztime=30s
        
  conformance:
    runs-on: ubuntu-latest
    steps:
      - name: Decorator Conformance
        run: go test ./runtime/decorators/... -run=TestDecoratorConformance
        
  cross-platform:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21', '1.22']
    runs-on: ${{ matrix.os }}
    steps:
      - name: Cross-platform Golden Tests
        run: go test ./tests/golden/... -v
```

### Nightly Stability
Comprehensive testing that runs overnight:

```yaml
# .github/workflows/nightly.yml  
name: Nightly Stability
on:
  schedule:
    - cron: '0 2 * * *' # 2 AM UTC
    
jobs:
  soak-test:
    runs-on: ubuntu-latest
    steps:
      - name: 24h Soak Test
        run: go test ./tests/soak/... -timeout=25h -run=TestSoakStability
        env:
          SOAK_DURATION: "24h"
          
  contract-replay:
    runs-on: ubuntu-latest
    steps:
      - name: Contract Replay
        run: go test ./tests/replay/... -v
        
  mutation-testing:
    runs-on: ubuntu-latest
    steps:
      - name: Mutation Tests
        run: go test ./tests/mutation/... -v
```

## Implementation Timeline

### Week 1-2: Foundation
- [ ] Set up golden test framework
- [ ] Add 5 core golden test cases
- [ ] Implement basic parser stability tests
- [ ] Add simple property tests for cosmetic changes

### Week 3-4: Decorator Testing
- [ ] Build conformance test framework
- [ ] Test all existing decorators for conformance
- [ ] Add security invariant tests (placeholder format)
- [ ] Implement basic mutation tests for @retry

### Month 2: Fuzzing & Reliability
- [ ] Set up fuzzing for parser/lexer
- [ ] Add contract replay framework
- [ ] Implement resource isolation tests
- [ ] Build soak test infrastructure

### Month 3+: Production Hardening
- [ ] Full 72h soak testing
- [ ] Comprehensive mutation testing
- [ ] Cross-platform CI matrix
- [ ] Performance regression detection

## Error Message Standards

All tests should verify error messages follow these standards:

```go
func TestErrorMessageQuality(t *testing.T) {
    malformedInput := "@retry(invalid)"
    
    _, err := parser.Parse(malformedInput)
    assert.Error(t, err)
    
    errMsg := err.Error()
    
    // Must include position
    assert.Regexp(t, `line \d+`, errMsg, "Error must include line number")
    
    // Must suggest fix
    assert.Contains(t, errMsg, "try:", "Error should suggest a fix")
    
    // Must not expose internal details
    assert.NotContains(t, errMsg, "panic", "Error should not expose panics")
    assert.NotContains(t, errMsg, "nil pointer", "Error should not expose internal errors")
}
```

## Stability Metrics

Track these metrics in CI to detect stability regressions:

- **Golden test pass rate**: Should be 100%
- **Parser crash rate**: Should be 0% (fuzzing)
- **Plan reproducibility**: Same input → same plan hash (100%)
- **Conformance pass rate**: All decorators pass conformance (100%)
- **Resource isolation success**: Resource bombs get terminated (100%)
- **Error message quality**: All errors include position + suggestion (100%)

## Plugin Registry Verification Strategy

**Pre-registry verification for external decorators following the architecture's plugin system design**

### Publisher Onboarding

* **Publisher key**: require verified org key (Sigstore/OIDC)
* **Policy doc**: plugin must declare supported `spec_version`s and semantic-versioning policy (N, N-1 minimum)
* **Contact + security.md**: CVE intake email, disclosure window, PGP key

### Plugin Manifest (Capability Contract)

Per-decorator YAML following the registry pattern from architecture:

```yaml
# decorators/aws.ec2.deploy.yaml
decorator: "@aws.ec2.deploy"
version: "1.4.2"
kind: exec                       # ExecutionDecorator
description: "Create/ensure EC2 instances"

args:
  - name: region
    type: string
    required: true
  - name: instance_spec
    type: object
    schema:
      type: object
      properties:
        type: { type: string }
        ami:  { type: string, pattern: "^ami-" }
      required: [type, ami]

# Architecture requirement: expose idempotency keys
idempotency:
  keys: ["region","name","instance_spec.type","instance_spec.ami"]

# Specification requirement: deterministic execution contracts
determinism:
  referentially_transparent: true
  randomness: forbidden   # or "seeded_only" for PSE

# Security invariant: placeholders only
security:
  secrets_policy: "placeholders_only"
  redaction_rules:
    - path: "$.userData"

# Architecture requirement: handle infrastructure drift gracefully  
drift:
  classify:
    - check: "missing_instance"
      map_to: "infra_missing"
    - check: "spec_changed" 
      map_to: "infra_mutated"
  remediation:
    spec_changed: "Apply with --force or update instance_spec to match."

tests:
  golden:
    - name: "minimal"
      plan_hash: "sha256:…"
```

### Automated Admission Pipeline

CLI entrypoint: `accord verify-plugin <artifact>`

**1. Integrity & Provenance**
- Verify signature (Sigstore/OIDC), SLSA provenance, SBOM
- Reproducible build verification
- Hash-pin toolchain

**2. Conformance Suite (aligns with architecture requirements)**
- **Contract determinism**: Value decorators return identical placeholders across resolves
- **PSE compliance**: `@random/*` decorators stable within resolved plans
- **Idempotency verification**: create→recreate→noop cycles work correctly
- **Drift classification**: Must classify `infra_missing` vs `infra_mutated` correctly
- **Security invariant**: No raw secrets in plans/logs/errors (placeholders only)
- **Crash isolation**: Plugin failures can't crash engine

**3. Security Sandboxing**
- Resource limits (CPU/mem) enforced per architecture
- Network allowlist from manifest
- File I/O limited to runtime scratch dir

**4. Golden Plan Tests**
- Each decorator ships byte-exact plan tests
- Re-generate across OS/arch matrix
- Verify contract verification works (plan→verify→execute)

### Runtime Enforcement

Following the architecture's plugin verification:

```go
func TestPluginConformance(t *testing.T) {
    plugin := loadPlugin("aws.ec2")
    
    // Architecture requirement: registry pattern lookup
    decorator := registry.LookupExecutionDecorator("@aws.ec2.deploy")
    assert.NotNil(t, decorator, "Plugin must register via unified registry")
    
    // Specification requirement: contract verification
    plan := decorator.Plan(validContext, validParams)
    resolvedPlan := resolvePlan(plan)
    
    // Verify contract verification catches changes
    mutatedParams := mutateParams(validParams)
    newPlan := decorator.Plan(validContext, mutatedParams)
    
    assert.NotEqual(t, resolvedPlan.Hash(), newPlan.Hash(), 
        "Contract verification must detect parameter changes")
        
    // Architecture requirement: security placeholders
    planString := resolvedPlan.String()
    assert.NotContains(t, planString, "sk_live_", 
        "Raw secrets must not appear in plans")
    assert.Regexp(t, `<\d+:\w+:[a-f0-9]+>`, planString,
        "Sensitive values must use placeholder format")
}
```

### Registry Lifecycle

* **Levels**: Provisional → Verified → Trusted (following architecture security model)
* **Auto-revocation**: CVE affecting deps, regression in re-tests, crash rate threshold
* **Transparency**: Publish verification results, SBOM hash, provenance

### Developer Tools

* `accord plugin init` - scaffolds manifest + tests
* `accord plugin test` - runs conformance suite locally  
* `accord verify-plugin` - local verification before publish

## Future Enhancements

As the system matures, add:

- **Contract verification fuzzing**: Fuzz the verify step to ensure it catches all changes
- **Plugin isolation testing**: Verify plugins can't crash the engine (per architecture)
- **Performance regression detection**: Benchmark key operations
- **PSE security testing**: Verify Plan Seed Envelope implementation
- **Chaos engineering**: Randomly inject failures during soak tests

This testing strategy ensures opal becomes ridiculously stable through systematic, incremental improvements while maintaining full alignment with the specification's contract verification model and architecture's plugin system design.