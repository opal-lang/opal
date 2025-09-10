# Devcmd Testing Strategy

## Overview

This document describes the testing approach for ensuring **semantic equivalence** between interpreter and generated modes, **deterministic execution**, and **decorator correctness**.

## Current Testing Architecture

**Phase 1 Focus**: Test-Driven Development of interpreter mode with clean separation of concerns.

### Module-Specific Testing Structure

```
testing/           - Generic test utilities (no runtime dependencies)
├── harness.go            - File utilities, environment helpers
├── decorator_harness.go  - CLI content builders, assertion helpers

cli/               - Interpreter mode tests
├── interpreter_test.go   - TDD tests for interpreter functionality

runtime/decorators/builtin/ - Individual decorator tests (future)
├── *_test.go             - Each decorator tested independently
```

**Key Testing Principles:**
- **TDD Approach**: Write failing tests first, implement to make them pass
- **Clean Dependencies**: `testing/` module has no runtime dependencies
- **Mock Shell**: Unit tests use mocked shell execution for fast, deterministic tests
- **Progressive Implementation**: Start with simple shell commands, add decorators incrementally

## Current Test Status

**✅ Working:**
- Basic shell command execution (`echo "hello"`)
- Test harness with file utilities and mock shell
- CLI content builders for generating test cases

**🚧 Failing (by design):**
- Shell operators (`&&`, `||`, `|`, `>>`) - transformation needs fixing
- Block decorators (`@workdir`, `@timeout`) - not implemented
- Action decorators (`@cmd`, `@log`) - not implemented
- Pattern decorators (`@when`) - not implemented
- Value decorators (`@var`, `@env`) - not implemented
- Plan generation - not implemented

## Test Goals

1. **Parity**: Interpreter vs Generated produce identical `{exit, stdout, stderr}` and **plan JSON**
2. **Compilation**: Generated code compiles, links, and runs correctly
3. **Determinism**: Frozen env → same fingerprint & outputs across runs
4. **Isolation**: No real side-effects in unit tests (mock shell); real shell covered in e2e

---

## Test Harness Architecture

### Core Runner Interface

```go
// test/harness/harness.go
type Mode int
const (Interp Mode = iota; Gen)

type RunResult struct {
    Exit           int
    Stdout, Stderr string
    PlanJSON       string
    EnvFingerprint string
}

type Runner interface {
    Exec(ctx *Ctx, cmd string, args ...string) RunResult
    Plan(ctx *Ctx, cmd string) RunResult
}
```

### Interpreter Runner

```go
type InterpRunner struct {
    Reg     *Registry
    CLIPath string  // path to commands.cli
}

func NewInterpRunner(reg *Registry, cliPath string) *InterpRunner

// Parse → IR → NodeEvaluator
func (r *InterpRunner) Exec(ctx *Ctx, name string, _ ...string) RunResult {
    ast := parser.ParseFile(r.CLIPath)
    command := TransformToIR(ast.Commands[name])
    evaluator := &NodeEvaluator{registry: r.Reg}
    result := evaluator.EvaluateNode(ctx, command.Root)
    return toRunResult(result)
}

func (r *InterpRunner) Plan(ctx *Ctx, name string) RunResult {
    ast := parser.ParseFile(r.CLIPath)
    command := TransformToIR(ast.Commands[name])
    plan := PlanNode(ctx, r.Reg, command.Root)
    return RunResult{
        PlanJSON:       plan.ToJSON(),
        EnvFingerprint: ctx.Env.Fingerprint,
    }
}
```

### Generated Runner

```go
type GenRunner struct {
    Reg     *Registry
    BinPath string  // compiled binary path
}

func NewGenRunner(t *testing.T, reg *Registry, ir *IR, outDir string) *GenRunner {
    // 1) Codegen to outDir/gen/...
    // 2) go build -trimpath -ldflags "-s -w" -o outDir/bin/mycli ./gen/cli
    // 3) Return &GenRunner{Reg: reg, BinPath: outDir+"/bin/mycli"}
}

// Run external binary
func (r *GenRunner) Exec(ctx *Ctx, name string, args ...string) RunResult {
    cmd := exec.Command(r.BinPath, name)
    cmd.Env = toKeyValList(ctx.Env.Values)  // Pass frozen env
    // Capture stdout, stderr, exit code
    return runAndCapture(cmd)
}

func (r *GenRunner) Plan(ctx *Ctx, name string) RunResult {
    cmd := exec.Command(r.BinPath, "--dry-run", name)
    cmd.Env = toKeyValList(ctx.Env.Values)
    // Capture plan JSON and fingerprint
    return runAndCapture(cmd)
}
```

---

## Core Parity Test Suite

### Table-Driven Parity Tests

```go
func Test_Parity_ExecAndPlan(t *testing.T) {
    cases := []struct{
        name string
        dsl  string
    }{
        {"simple_chain", `build: go build ./... && echo "ok"`},
        {"shell_operators", `test: @cmd(build) && npm test || echo "failed"`},
        {"timeout_retry", `run: @timeout(2s){ @retry(2){ echo hi && false || echo fallback }}`},
        {"pipe_append", `process: echo "x" | sed 's/x/y/' >> out.txt`},
        {"when_branches", `deploy: @when(ENV) { prod: kubectl apply -f prod.yaml }`},
        {"parallel_exec", `all: @parallel{ @cmd(build) \n @cmd(test) }`},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // Freeze env once
            env, _ := NewEnvSnapshot(EnvOptions{
                BlockList: []string{"PWD","OLDPWD","SHLVL","RANDOM","PS*","TERM"},
            })
            ctx := &Ctx{Env: env}

            reg := NewStandardRegistry()

            // Build IR from DSL
            cliPath, ir := MustParseToIR(t, tc.dsl)

            interp := NewInterpRunner(reg, cliPath)
            gen := NewGenRunner(t, reg, ir, t.TempDir())

            // Test execution parity
            gotI := interp.Exec(ctx, FirstCmdName(ir))
            gotG := gen.Exec(ctx, FirstCmdName(ir))
            
            assert.Equal(t, gotI.Exit, gotG.Exit, "exit code mismatch")
            assert.Equal(t, gotI.Stdout, gotG.Stdout, "stdout mismatch")
            assert.Equal(t, gotI.Stderr, gotG.Stderr, "stderr mismatch")

            // Test plan parity
            planI := interp.Plan(ctx, FirstCmdName(ir))
            planG := gen.Plan(ctx, FirstCmdName(ir))
            
            assert.JSONEq(t, planI.PlanJSON, planG.PlanJSON, "plan JSON mismatch")
            assert.Equal(t, planI.EnvFingerprint, planG.EnvFingerprint, "fingerprint mismatch")
        })
    }
}
```

---

## Decorator-Focused Test Harness

### Goals

* One call proves: **Run parity**, **Plan parity**, **Env freeze**, and (optionally) **builds generated CLI**.
* No custom test DSL required—just pass your decorator + a scenario.

### Package API

```go
package decorharness

// Core inputs/outputs
type CaseCtx struct {
  EnvLockPath string            // optional persistent env lock
  Env         map[string]string // additional env values
  WorkDir     string
  Timeout     time.Duration     // overall test timeout
}

type Expect struct {
  Exit    int
  Stdout  string
  Stderr  string
  PlanHas []string // substrings that must appear in plan JSON/text
}

// Minimal scenario model
type Scenario struct {
  // A small IR-ish description to wrap your decorator
  // e.g. for action: preceding/following chain ops, pipes, append
  BeforeShell string            // optional; executed before decorator with &&
  AfterShell  string            // optional; executed after decorator with &&
  PipeTo      string            // optional shell that receives stdout from decorator
  AppendTo    string            // optional file target for >>
  Inner       []string          // for Block/Pattern: newline-separated steps inside
  PatternVals map[string]any    // for Pattern: inputs used to select a branch
}

// Public entrypoints (run both modes by default)
func CheckAction(t TB, a ActionDecorator, args Args, scn Scenario, exp Expect, opts ...Option)
func CheckBlock(t TB, b BlockDecorator, args Args, scn Scenario, exp Expect, opts ...Option)
func CheckValue(t TB, v ValueDecorator, args Args, want string, planHas []string, opts ...Option)
func CheckPattern(t TB, p PatternDecorator, args Args, branches map[string][]string, choose map[string]any, exp Expect, opts ...Option)

// Options
type Option func(*config)
func WithInterpOnly() Option
func WithGenOnly() Option
func WithNoBuild() Option          // skip codegen build, run interpreter only
func WithClock(fake Clock) Option  // inject fake clock for timeout/retry/backoff
```

### How it works (behind the scenes)

* Builds a **tiny synthetic command** that uses your decorator per `Scenario`.
* **Interpreter path:** parse → IR → run with your decorator in the registry.
* **Generated path:** codegen the same command, `go build`, execute.
* Both use **frozen env** (`NewEnvSnapshot`) so results and plan are stable.

### Implementation notes

* **Env freeze:** harness creates `Ctx{Env: NewEnvSnapshot(...)}` and sets the same env when launching the generated binary. If `EnvLockPath` is set, it persists/loads the lock for reproducibility.
* **Pipes/append:** harness uses the same runtime helpers (`pipeExec`, `appendToFile`) ensuring decorator I/O matches chain semantics.
* **Clock:** wrappers read `ctx.Clock`; harness can inject a fake clock to make timeout/retry deterministic.
* **Builds:** by default `Check*` runs **both** modes; `WithNoBuild()` skips codegen build for speed.

### Quick Examples

#### 1) Action decorator

```go
CheckAction(t, &LogDecorator{}, Args{"msg": "built"},
  Scenario{
    BeforeShell: `echo start`,
    AfterShell:  `echo end`,
    PipeTo:      ``,
    AppendTo:    ``,
  },
  Expect{
    Exit:   0,
    Stdout: "start\nbuilt\nend\n",
    PlanHas: []string{`"kind":"action"`, `"name":"log"`, `"msg":"built"`},
  },
)
```

#### 2) Block decorator (timeout)

```go
CheckBlock(t, &TimeoutDecorator{}, Args{"d":"200ms"},
  Scenario{
    Inner: []string{`sleep 1`, `echo never`}, // newline-separated steps
  },
  Expect{
    Exit:    124,
    Stderr:  "timeout",
    PlanHas: []string{`"block":"timeout"`, `"d":"200ms"`},
  },
)
```

#### 3) Pattern decorator (@when)

```go
branches := map[string][]string{
  "prod": {`echo deploy-prod`},
  "dev":  {`echo deploy-dev`},
}
CheckPattern(t, &WhenDecorator{}, Args{"var":"ENV"},
  branches,
  map[string]any{"ENV":"prod"},
  Expect{Exit:0, Stdout:"deploy-prod\n", PlanHas:[]string{`"selected":"prod"`}},
)
```

#### 4) Value decorator

```go
CheckValue(t, &EnvValue{}, Args{"key":"SERVICE"}, "filesvc",
  []string{`"value":"filesvc"`, `"source":"env"`},
  WithInterpOnly(),
)
```

---

## Test Categories

### Unit Tests (Fast, Hermetic)

- **Mock ExecShell**: Replace with injected mock returning canned `CommandResult`s
- **Chain semantics**: Validate `&&`, `||`, `|`, `>>` operators
- **Wrapper behavior**: Test timeout, retry, parallel decorators
- **Pattern selection**: Verify @when branch selection logic
- **Golden files**: Compare plan JSON against `testdata/*.golden.json`

### Integration Tests (Real Execution)

- **Real shell**: Use actual `/bin/sh` execution
- **File operations**: Test `>>` append, file creation/modification
- **Pipe streaming**: Verify large output handling
- **Process management**: Test background processes, signals
- **Environment**: Test env freezing and child process inheritance

### Compilation Tests

```go
func Test_Generated_Builds(t *testing.T) {
    cases := []string{
        `hello: echo "hi"`,
        `complex: @timeout(5m){ @parallel{ build \n test }}`,
    }
    
    for _, dsl := range cases {
        _, ir := MustParseToIR(t, dsl)
        reg := NewStandardRegistry()
        out := t.TempDir()
        _ = NewGenRunner(t, reg, ir, out)  // Fails test if build fails
    }
}
```

### Determinism Tests

```go
func Test_EnvLock_Reproducible(t *testing.T) {
    lock := filepath.Join(t.TempDir(), "env.lock.json")
    
    // First run creates lock
    env1, _ := NewEnvSnapshot(EnvOptions{LockPath: lock})
    
    // Second run loads lock
    env2, _ := NewEnvSnapshot(EnvOptions{LockPath: lock})
    
    assert.Equal(t, env1.Fingerprint, env2.Fingerprint, "non-reproducible env")
}

func Test_Plan_Deterministic(t *testing.T) {
    dsl := `deploy: @when(ENV) { prod: echo prod \n dev: echo dev }`
    
    // Run plan generation multiple times
    plans := make([]string, 10)
    for i := range plans {
        ctx := &Ctx{Env: frozenEnv}
        plans[i] = generatePlanJSON(ctx, dsl)
    }
    
    // All plans must be identical
    for i := 1; i < len(plans); i++ {
        assert.JSONEq(t, plans[0], plans[i], "non-deterministic plan")
    }
}
```

---

## Golden Test Management

### Golden File Structure

```
testdata/
├── golden/
│   ├── simple_chain.golden.json
│   ├── timeout_decorator.golden.json
│   └── when_pattern.golden.json
└── fixtures/
    ├── complex.cli
    └── test_env.json
```

### Golden Test Helper

```go
func TestGoldenPlans(t *testing.T) {
    files, _ := filepath.Glob("testdata/fixtures/*.cli")
    
    for _, file := range files {
        name := strings.TrimSuffix(filepath.Base(file), ".cli")
        t.Run(name, func(t *testing.T) {
            golden := fmt.Sprintf("testdata/golden/%s.golden.json", name)
            
            // Generate plan
            plan := generatePlanFromFile(file)
            
            if *update {
                os.WriteFile(golden, []byte(plan), 0644)
            } else {
                expected, _ := os.ReadFile(golden)
                assert.JSONEq(t, string(expected), plan)
            }
        })
    }
}
```

---

## CI/CD Matrix

### Test Jobs

1. **Unit Tests**: `go test -short ./...` (mocked shell, fast)
2. **Integration Tests**: `go test -run Integration ./...` (real shell)
3. **Race Detection**: `go test -race ./runtime/...` (interpreter path)
4. **Compilation Tests**: `go test -run Compilation ./...` (codegen + build)
5. **Parity Suite**: `go test -run Parity ./...` (full interpreter vs generated)

### Platform Matrix

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest]
    go: [1.21, 1.22]
```

### Performance Benchmarks

```go
func BenchmarkInterpreter(b *testing.B) {
    ctx := newTestCtx()
    for i := 0; i < b.N; i++ {
        runInterpreter(ctx, "build")
    }
}

func BenchmarkGenerated(b *testing.B) {
    ctx := newTestCtx()
    for i := 0; i < b.N; i++ {
        runGenerated(ctx, "build")
    }
}
```

---

## Test Utilities

### Mock Shell Executor

```go
type MockExec struct {
    Commands map[string]CommandResult
}

func (m *MockExec) ExecShell(ctx *Ctx, cmd string) CommandResult {
    if result, ok := m.Commands[cmd]; ok {
        return result
    }
    return CommandResult{ExitCode: 127, Stderr: "command not found"}
}

// Inject mock for unit tests
func TestWithMockShell(t *testing.T) {
    mock := &MockExec{
        Commands: map[string]CommandResult{
            "go build ./...": {ExitCode: 0, Stdout: "ok"},
            "npm test":       {ExitCode: 1, Stderr: "failed"},
        },
    }
    
    // Replace global ExecShell
    oldExec := ExecShell
    ExecShell = mock.ExecShell
    defer func() { ExecShell = oldExec }()
    
    // Run tests with mock
}
```

### Assertion Helpers

```go
func AssertParity(t *testing.T, interp, gen RunResult) {
    t.Helper()
    assert.Equal(t, interp.Exit, gen.Exit, "exit code")
    assert.Equal(t, interp.Stdout, gen.Stdout, "stdout")
    assert.Equal(t, interp.Stderr, gen.Stderr, "stderr")
    assert.JSONEq(t, interp.PlanJSON, gen.PlanJSON, "plan")
    assert.Equal(t, interp.EnvFingerprint, gen.EnvFingerprint, "env")
}

func AssertPlanContains(t *testing.T, planJSON string, expected ...string) {
    t.Helper()
    for _, exp := range expected {
        assert.Contains(t, planJSON, exp, "plan missing: %s", exp)
    }
}
```

---

## Best Practices

1. **Keep tests fast**: Use mocks for unit tests, real shell only in integration
2. **Test both modes**: Every feature must work in interpreter AND generated
3. **Golden files**: Use for complex plan outputs, update with `--update` flag
4. **Determinism**: Always use frozen env, stable sorting, and fixed clocks
5. **Error cases**: Test malformed chains, missing decorators, invalid args
6. **Edge cases**: Large pipes, timeout during parallel, nested wrappers
7. **Benchmarks**: Track performance regression between modes

This comprehensive testing strategy ensures devcmd maintains semantic equivalence, deterministic behavior, and high reliability across all execution modes.