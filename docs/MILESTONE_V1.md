---
title: "Milestone v1.0 - Powerful Ops-Focused Release"
audience: "Core Contributors & Project Leads"
summary: "Scope, acceptance criteria, and implementation roadmap for first production release"
---

# Milestone v1.0: Powerful Ops-Focused Release

## Executive Summary

**Goal**: Ship a production-ready operations tool that proves the plan-first, reality-driven model works in practice.

**Philosophy**: Keep it ops-focused. Prove the core model with real workflows before expanding scope.

### 10 Must-Haves for v1.0 Release

1. **Deterministic planning**: Same inputs → same plan hash (100%)
2. **Contract verification**: Replan + compare + verify on every execution
3. **Zero secret leakage**: No raw secrets in plans/logs (only placeholders)
4. **Performance SLOs**: <10ms plan generation (p50), <25ms (p95)
5. **8-12 decorators**: Value + execution with caching/batching
6. **Control flow**: `for`, `when`, `if`, `try/catch` with plan-time pruning
7. **Error taxonomy**: Plan vs execute vs provider errors with actionable messages
8. **Observability**: Run tracking, OTEL spans, artifacts (plan.json, traces, summary)
9. **Golden tests**: Byte-exact plan output verification
10. **External validation**: First user writes deploy playbook in <4 hours (survey ≥4/5)

### Guardrails (What's OUT)

- ❌ Infrastructure CRUD (ops-focused only)
- ❌ Plugin API (blessed decorators only)
- ❌ Advanced REPL/shell (minimal history/completion)
- ❌ AOT compilation (packaging only, same engine)

### Success Signals

- **Performance**: <10ms plan gen (p50), <25ms (p95), <50ms (p99)
- **Security**: Zero raw secrets in plans/logs (100% redaction)
- **Reliability**: Repeated runs against drifting reality are safe
- **Usability**: First external user completes playbook in <4 hours (survey ≥4/5)

### Timeline

**15-22 weeks** (4-5.5 months) across 5 milestones. Currently **60% through Milestone 1**.

## Make-or-Break Areas

These are the critical technical challenges that determine success or failure:

### 1. Determinism Everywhere
- **Requirement**: Identical inputs → byte-identical plan
- **PASS/FAIL KPI**: Same script + same environment = same plan hash 100% of the time (10,000 runs)
- **Implementation**:
  - Stable hashing algorithm (SHA-256)
  - Forbid non-deterministic decorators in resolved plans
  - Plan Seed Envelopes (PSE) for seeded randomness
  - Deterministic iteration order (sorted maps)
- **Test**: Golden plan tests verify byte-exact output

### 2. Reality-as-Database at Scale
- **Requirement**: Fast provider queries without blocking
- **PASS/FAIL KPI**: 50 `@aws.secret.*` calls complete in <500ms (batched), p95 <750ms
- **Implementation**:
  - Aggressive memoization (cache identical decorator calls)
  - Batch resolution (collect all AWS calls, execute as one batch)
  - Clear timeouts (5s default, configurable)
  - Exponential backoff on retries
- **Test**: Benchmark suite measures batch performance

### 3. Contract Verification Races
- **Requirement**: Crisp rules for "what counts as change"
- **PASS/FAIL KPI**: Zero false positives, zero false negatives in change detection (100 test scenarios)
- **Implementation**:
  - Replan from current reality
  - Compare plan structure hash
  - Friendly diff on mismatch (show what changed)
  - Clear error categories: `source_changed`, `infra_mutated`, `env_changed`
- **Test**: Change detection test suite covers all categories

### 4. Decorator Sandboxing & Safety
- **Requirement**: No side effects at plan-time
- **PASS/FAIL KPI**: Zero plan-time side effects (verified by monitoring test suite)
- **Implementation**:
  - Value decorators: pure resolution only
  - Execution decorators: plan-time = structure, execute-time = work
  - Shell/remote execution: strong boundaries, no plan-time execution
  - Secret redaction: automatic, no raw secrets in plans/logs
- **Test**: Conformance suite verifies no side effects during planning

### 5. Tooling Parity
- **Requirement**: Event-stream → plan (runtime) vs event-stream → AST (tooling) never diverge
- **PASS/FAIL KPI**: 100% event consistency between runtime and tooling paths (shared test suite)
- **Implementation**:
  - Single parser produces events
  - Runtime consumes events → plan
  - Tooling consumes events → AST (lazy)
  - Shared test suite for both paths
- **Test**: Same events produce consistent results in both paths

### 6. Performance SLOs
- **Requirement**: Sub-10ms plan generation for typical scripts
- **PASS/FAIL KPI**: Plan gen p50 <10ms, p95 <25ms, p99 <50ms (100-line script benchmark)
- **Implementation**:
  - Lexer: >5000 lines/ms (achieved)
  - Parser: >3000 lines/ms (target)
  - Plan generation: <5ms for 100-line script
  - Concurrent resolution without head-of-line blocking
  - Low memory footprint (<50MB for typical workloads)
- **Test**: Benchmark suite verifies SLOs on every commit

## Powerful v1 Checklist

### Core Language Features
- [ ] **Command mode**: Organized tasks in `commands.opl` **(RELEASE-BLOCKING)**
- [ ] **Script mode**: Direct execution scripts **(RELEASE-BLOCKING)**
- [ ] **Function declarations**: `fun deploy(env: String) { ... }` **(RELEASE-BLOCKING)**
- [ ] **Variable declarations**: `var ENV = @env.ENVIRONMENT` **(RELEASE-BLOCKING)**
- [ ] **Control flow**:
  - [ ] `when` pattern matching with plan-time branch selection **(RELEASE-BLOCKING)**
  - [ ] `if/else` conditionals **(RELEASE-BLOCKING)**
  - [ ] `for` loops with plan-time unrolling **(RELEASE-BLOCKING)**
  - [ ] `try/catch/finally` error handling **(RELEASE-BLOCKING)**
- [ ] **Plan-time pruning**: Only selected branches appear in plan **(RELEASE-BLOCKING)**

### Decorator System
- [ ] **Value decorators**: Pure resolution, no side effects **(RELEASE-BLOCKING)**
  - [ ] `@env.NAME` - Environment variables **(RELEASE-BLOCKING)**
  - [ ] `@var.NAME` - Script variables **(RELEASE-BLOCKING)**
  - [ ] `@file.read(path)` - File contents **(RELEASE-BLOCKING)**
  - [ ] `@aws.secret.NAME(auth)` - AWS Secrets Manager (can slip to v1.1)
  - [ ] `@aws.auth(profile, region)` - AWS auth handles (can slip to v1.1)
- [ ] **Execution decorators**: Work execution with observability **(RELEASE-BLOCKING)**
  - [ ] `@shell(command)` - Shell command execution **(RELEASE-BLOCKING)**
  - [ ] `@retry(attempts, delay)` - Retry with backoff **(RELEASE-BLOCKING)**
  - [ ] `@timeout(duration)` - Timeout protection **(RELEASE-BLOCKING)**
  - [ ] `@parallel { ... }` - Concurrent execution (can slip to v1.1)
  - [ ] `@workdir(path) { ... }` - Working directory scope (can slip to v1.1)
  - [ ] `@log(message, level)` - Structured logging **(RELEASE-BLOCKING)**
- [ ] **Cloud decorators** (8-12 first-party, at least 6 for v1.0):
  - [ ] `@k8s.exec(pod, command)` - Kubernetes pod execution **(RELEASE-BLOCKING)**
  - [ ] `@http.get(url)` / `@http.post(url, body)` - HTTP requests **(RELEASE-BLOCKING)**
  - [ ] `@postgres.query(sql, conn)` - Database queries (can slip to v1.1)
  - [ ] `@aws.ec2.instances(tags, auth)` - Resource queries (can slip to v1.1)
- [ ] **Caching & batching**: **(RELEASE-BLOCKING)**
  - [ ] Memoization: identical calls return cached results **(RELEASE-BLOCKING)**
  - [ ] Batch resolution: collect + execute as one API call **(RELEASE-BLOCKING)**

### Plan System
- [ ] **Plan generation**: Events → deterministic plan **(RELEASE-BLOCKING)**
- [ ] **Plan hash**: SHA-256 of resolved plan structure **(RELEASE-BLOCKING)**
  - **Hash domain**: Structure + step commands + decorator args (redacted form for secrets)
  - **Excludes**: Comments, whitespace, variable names (cosmetic changes don't affect hash)
  - **Example**: `sha256(sort(steps) + sort(decorators) + redacted(args))`
- [ ] **Contract verification**: Replan + compare + verify **(RELEASE-BLOCKING)**
- [ ] **Quick plans**: Fast preview with deferred expensive decorators **(RELEASE-BLOCKING)**
- [ ] **Resolved plans**: Complete execution contracts **(RELEASE-BLOCKING)**
- [ ] **Plan diff**: Friendly comparison on mismatch **(RELEASE-BLOCKING)**

### Execution Engine
- [ ] **Direct execution**: `opal deploy` **(RELEASE-BLOCKING)**
- [ ] **Contract execution**: `opal run --plan prod.plan` **(RELEASE-BLOCKING)**
- [ ] **Dry-run mode**: `opal deploy --dry-run` **(RELEASE-BLOCKING)**
- [ ] **Resolved plan mode**: `opal deploy --dry-run --resolve` **(RELEASE-BLOCKING)**
- [ ] **Error taxonomy**: Plan errors vs execution errors vs provider errors **(RELEASE-BLOCKING)**
- [ ] **Friendly error messages**: Context + suggestion + example **(RELEASE-BLOCKING)**

### Observability
- [ ] **Run identification**: `run-<timestamp>-<hash>` **(RELEASE-BLOCKING)**
- [ ] **Plan hash tracking**: Link runs to reviewed plans **(RELEASE-BLOCKING)**
- [ ] **OpenTelemetry spans**: One span per step **(RELEASE-BLOCKING)**
- [ ] **Run artifacts**: **(RELEASE-BLOCKING)**
  - [ ] `plan.json` - Resolved plan (redacted) **(RELEASE-BLOCKING)**
  - [ ] `otlp-traces.json` - Execution traces **(RELEASE-BLOCKING)**
  - [ ] `summary.json` - Status, durations, metadata
- [ ] **CLI commands**:
  - [ ] `opal runs list` - List recent runs
  - [ ] `opal runs show <run-id>` - Show run details
  - [ ] `opal runs open <run-id>` - Open HTML report

### Testing Infrastructure
- [ ] **Golden plan tests**: Byte-exact plan output **(RELEASE-BLOCKING)**
- [ ] **Plan stability tests**: Cosmetic changes don't affect hash **(RELEASE-BLOCKING)**
- [ ] **Decorator conformance suite**: All decorators pass invariant tests **(RELEASE-BLOCKING)**
- [ ] **Parser resilience tests**: Malformed input doesn't crash **(RELEASE-BLOCKING)**
- [ ] **Fuzzing**: Parser/lexer stability under random input **(RELEASE-BLOCKING)**
- [ ] **Benchmark suite**: Performance regression detection **(RELEASE-BLOCKING)**
- [ ] **Error message quality**: All errors include position + suggestion **(RELEASE-BLOCKING)**

### Release Gate Script
- [ ] **`opal verify-ci`**: Single command that runs all critical checks **(RELEASE-BLOCKING)**
  ```bash
  opal verify-ci
  # Runs in sequence:
  # 1. Golden plan tests (byte-exact output)
  # 2. Performance benchmarks (p50/p95/p99 SLOs)
  # 3. Fuzz seed corpus (parser stability)
  # 4. Decorator conformance (all decorators pass)
  # 5. Secret redaction audit (regex allowlist)
  # 6. Plan hash stability (cosmetic changes)
  # 
  # Exit 0: All checks pass, ready to release
  # Exit 1: At least one check failed, see report
  ```

### Security
- [ ] **Secret redaction**: No raw secrets in plans/logs **(RELEASE-BLOCKING)**
  - **Format**: `<length:algorithm:hash>` (e.g., `<32:sha256:a3b2c1d4...>`)
  - **Example**: `@aws.secret.db_password` → `<64:sha256:5f6c7e8d...>`
  - **Regex allowlist** (strict review):
    ```regex
    # Allowed in plans/logs:
    <\d+:(sha256|sha512|blake3):[a-f0-9]{8,}>
    
    # Forbidden patterns (auto-fail CI):
    (password|secret|key|token|credential)[\s:=]["'][^"']{8,}
    sk_live_[a-zA-Z0-9]+
    -----BEGIN (PRIVATE|RSA) KEY-----
    AKIA[A-Z0-9]{16}
    ```
- [ ] **Plan-time safety**: No side effects during planning **(RELEASE-BLOCKING)**
- [ ] **Execution boundaries**: Clear separation of plan vs execute **(RELEASE-BLOCKING)**
- [ ] **Audit trail**: Complete decorator usage tracking **(RELEASE-BLOCKING)**

## v1.1 Staging (Deferred from v1.0)

**These features are valuable but non-essential for proving the core model:**

- **Binary compilation**: Embed source + runtime for air-gapped deployment
- **REPL**: Interactive mode with history, completion, plan preview
- **Formatter**: Idempotent code formatting (preserve semantics, normalize style)
- **Advanced decorators**: `@parallel`, `@workdir`, `@postgres.query`, `@aws.*`
- **LSP server**: Full language server protocol implementation
- **Plugin API**: Third-party decorator registration

**Why defer:** Ship v1.0, get feedback, then expand based on real usage patterns.

## Implementation Milestones

**Derisk in this order to maximize learning and minimize wasted effort:**

### Milestone 1: Event Parser → Plan (4-6 weeks)
**Goal**: Prove the event-based architecture works

**Definition of Ready:**
- ✅ Lexer complete and tested (>5000 lines/ms)
- ✅ Event-based parser foundation (Iterations 1-2 complete)
- ✅ Grammar specification finalized (GRAMMAR.md)
- ✅ Testing strategy defined (TESTING_STRATEGY.md)

**Tasks:**
- [ ] Complete parser implementation (Iterations 3-5)
  - [ ] Statements & expressions
  - [ ] Control flow (`for`, `when`, `if`, `try/catch`)
  - [ ] Decorators (syntax only)
- [ ] Event → Plan converter (execution path)
- [ ] Golden plan tests (byte-exact output)
- [ ] Plan hash generation (SHA-256)
- [ ] Basic error reporting

**Success criteria:**
- Parser handles all grammar constructs
- Events → Plan pipeline works
- Golden tests verify determinism
- Performance: >3000 lines/ms parsing

### Milestone 2: Decorator Runtime (4-6 weeks)
**Goal**: Prove decorator model works in practice

**Definition of Ready:**
- ✅ Parser complete (Milestone 1 done)
- ✅ Plan generation working
- ✅ Decorator architecture designed (DECORATOR_GUIDE.md)
- ✅ Conformance test framework ready

**Tasks:**
- [ ] Decorator registry (value + execution)
- [ ] Built-in decorators (6-8 minimum for v1.0)
- [ ] Caching & batching infrastructure
- [ ] Value resolution (parallel where possible)
- [ ] Execution engine (sequential steps)
- [ ] Secret redaction system

**Success criteria:**
- All built-in decorators pass conformance tests
- Batching works (50 AWS calls in <500ms)
- No raw secrets in plans/logs
- Decorator composition works naturally

### Milestone 3: Contract Verification (2-3 weeks)
**Goal**: Prove contract model prevents stale execution

**Definition of Ready:**
- ✅ Decorator runtime working (Milestone 2 done)
- ✅ Plan hash generation stable
- ✅ Execution engine functional
- ✅ Test scenarios defined (100 change detection cases)

**Tasks:**
- [ ] Replan from current state
- [ ] Plan structure comparison
- [ ] Hash verification
- [ ] Friendly diff on mismatch
- [ ] Error categorization (source/infra/env changed)

**Success criteria:**
- Detects all categories of change
- Error messages are actionable
- False positives = 0%
- False negatives = 0%

### Milestone 4: Observability Minimal (2-3 weeks)
**Goal**: Make debugging and auditing trivial

**Definition of Ready:**
- ✅ Contract verification working (Milestone 3 done)
- ✅ Execution engine stable
- ✅ Plan hash tracking implemented
- ✅ Observability design finalized (OBSERVABILITY.md)

**Tasks:**
- [ ] Run ID generation
- [ ] Plan hash tracking
- [ ] OpenTelemetry span emission
- [ ] Artifact generation (plan.json, traces, summary)
- [ ] CLI commands (runs list/show/open)

**Success criteria:**
- Every run has unique ID + plan hash
- Traces correlate to plan steps
- Artifacts are human-readable
- CLI makes debugging fast

### Milestone 5: Tooling Path (3-4 weeks) - OPTIONAL for v1.0
**Goal**: Enable rich developer experience (can slip to v1.1)

**Definition of Ready:**
- ✅ Observability working (Milestone 4 done)
- ✅ Event-based parser stable
- ✅ AST design finalized (AST_DESIGN.md)
- ✅ Tooling requirements gathered

**Tasks:**
- [ ] Lazy AST construction from events (DEFERRED to v1.1)
- [ ] Formatter (preserve semantics, normalize style) (DEFERRED to v1.1)
- [ ] Basic REPL (history, completion, plan mode) (DEFERRED to v1.1)
- [ ] Binary compilation (embed source + runtime) (DEFERRED to v1.1)

**Success criteria (if implemented):**
- Formatter is idempotent
- REPL supports plan mode
- Compiled binaries work air-gapped
- LSP foundation ready (AST queries work)

**Note**: These features moved to v1.1 staging to reduce v1.0 scope and ship faster.

## Guardrails (Avoid Scope Creep)

**What's OUT of scope for v1:**

- ❌ **Infrastructure CRUD**: No `@aws.instance.create()` yet - focus on operations
- ❌ **Plugin API**: Only blessed decorators in v1, plugin system after core stabilizes
- ❌ **Advanced REPL**: Keep it minimal (history/complete/plan mode), shell ambitions later
- ❌ **AOT compilation**: Compilation = packaging (same engine), not a new optimizer
- ❌ **LSP server**: Foundation only (AST queries), full server after v1
- ❌ **Advanced IaC**: Prove ops model first, then extend to infrastructure

**Why these guardrails matter:**

Each excluded feature is valuable but non-essential for proving the core model. Ship v1, get feedback, then expand scope based on real usage.

## Success Signals (Objective Metrics)

### Performance
- [ ] Typical plan generation: ≤10ms (p50)
- [ ] Plan generation p95: ≤25ms
- [ ] Plan generation p99: ≤50ms
- [ ] Memory usage: <50MB for typical workloads
- [ ] Lexer throughput: >5000 lines/ms
- [ ] Parser throughput: >3000 lines/ms

### Security
- [ ] Zero raw secrets in plans/logs (100% redaction)
- [ ] No side effects during planning (verified by tests)
- [ ] All decorators pass security conformance suite
- [ ] Audit trail captures all decorator usage

### Reliability
- [ ] Repeated runs against drifting reality are safe
- [ ] Contract verification catches all change categories
- [ ] Error messages are actionable (user testing confirms)
- [ ] Parser never crashes on malformed input (fuzzing confirms)

### Usability
- [ ] First external user writes deploy playbook in <4 hours without help (survey ≥4/5)
  - **Test protocol**: Recruit external user, provide docs only, observe without helping
  - **Survey questions**: 
    - "How easy was it to get started?" (1-5)
    - "Were error messages helpful?" (1-5)
    - "Would you use this for real work?" (1-5)
  - **Pass criteria**: Average score ≥4/5, completes playbook in <4 hours
- [ ] Error messages include position + suggestion + example
- [ ] Plan output is human-readable
- [ ] Documentation covers all features with examples

### Determinism
- [ ] Same inputs → same plan hash (100% of the time)
- [ ] Golden tests verify byte-exact output
- [ ] Plan stability under cosmetic changes (whitespace, comments)
- [ ] No flaky tests in CI

## Acceptance Criteria for v1.0 Release

**Must have all of:**

1. ✅ **Core language complete**: All grammar constructs implemented and tested
2. ✅ **8-12 decorators working**: Value + execution decorators with caching/batching
3. ✅ **Plan system working**: Generation, hashing, verification, diffing
4. ✅ **Execution engine working**: Direct + contract execution modes
5. ✅ **Observability working**: Run tracking, artifacts, CLI commands
6. ✅ **Performance SLOs met**: <10ms plan generation, >3000 lines/ms parsing
7. ✅ **Security invariants met**: Zero raw secrets, no plan-time side effects
8. ✅ **Testing complete**: Golden tests, conformance, fuzzing, benchmarks
9. ✅ **Documentation complete**: User guide, decorator guide, examples
10. ✅ **External validation**: At least one user writes real workflow without help

**Nice to have (can defer to v1.1):**

- Formatter (idempotent, preserves semantics)
- REPL (basic history/completion)
- Binary compilation (embed source + runtime)
- LSP foundation (AST queries)

## Implementation Order

**5 milestones to derisk the core model:**

1. **Event Parser → Plan** - Prove event-based architecture works
2. **Decorator Runtime** - Prove decorator model works in practice
3. **Contract Verification** - Prove contract model prevents stale execution
4. **Observability Minimal** - Make debugging and auditing trivial
5. **Tooling Path** - Enable rich developer experience (OPTIONAL for v1.0)

## Current Progress

**Completed:**
- ✅ Event-based parser foundation (Iterations 1-2)
- ✅ Lexer (>5000 lines/ms achieved)
- ✅ Error reporting and recovery
- ✅ Telemetry and debug infrastructure
- ✅ Comprehensive documentation (9 production-ready docs)
- ✅ Testing strategy defined
- ✅ Architecture validated

**In Progress:**
- 🔄 Parser Iteration 3: Statements & Expressions

**Next Up:**
- Parser Iteration 4: Control Flow
- Parser Iteration 5: Decorators
- Event → Plan converter

**Estimated completion:** ~60% of Milestone 1 complete

## Risk Register

**Top 5 risks with mitigations:**

### 1. Parser/Tooling Divergence
**Risk**: Event-stream → plan (runtime) and event-stream → AST (tooling) produce inconsistent results

**Impact**: HIGH - Breaks LSP, formatter, and other tooling

**Mitigation**:
- Shared test suite for both paths (same events, verify consistency)
- Golden tests verify both paths produce expected output
- CI gate: `opal verify-ci` runs both paths on every commit
- Regular cross-validation during development

**Status**: Mitigated by architecture (single parser, dual consumers)

### 2. Provider Rate Limits
**Risk**: Batching/caching doesn't work well enough, hit AWS/cloud provider rate limits

**Impact**: MEDIUM - Slow plan generation, failed executions

**Mitigation**:
- Aggressive memoization (cache identical decorator calls)
- Batch resolution (collect all calls, execute as one batch)
- Exponential backoff with jitter on retries
- Clear timeout configuration (5s default, user-configurable)
- Performance benchmarks verify batching works (50 calls in <500ms)

**Status**: Mitigated by design (batching is core architecture)

### 3. Decorator Sandbox Leaks
**Risk**: Decorators execute side effects during planning (violates safety invariant)

**Impact**: HIGH - Security vulnerability, unpredictable behavior

**Mitigation**:
- Conformance test suite verifies no side effects during planning
- Monitoring test suite detects file/network/process activity during plan generation
- Code review checklist for all decorators
- CI gate: `opal verify-ci` runs sandbox leak detection
- Clear separation: value decorators = pure resolution, execution decorators = plan structure only

**Status**: Mitigated by testing (conformance suite catches violations)

### 4. Plan Hash Drift
**Risk**: Hash changes for semantically identical plans (cosmetic changes affect hash)

**Impact**: MEDIUM - False positive contract verification failures

**Mitigation**:
- Hash domain excludes cosmetic changes (comments, whitespace, variable names)
- Hash only: structure + step commands + decorator args (redacted form)
- Golden tests verify cosmetic changes don't affect hash
- Plan stability test suite (100+ scenarios)
- Clear hash specification in MILESTONE_V1.md

**Status**: Mitigated by design (hash domain explicitly defined)

### 5. Performance Regressions
**Risk**: Changes slow down lexer/parser/plan generation below SLOs

**Impact**: MEDIUM - Poor user experience, failed acceptance criteria

**Mitigation**:
- Benchmark suite runs on every commit
- CI gate: `opal verify-ci` enforces SLOs (p50 <10ms, p95 <25ms, p99 <50ms)
- Performance budgets tracked in CI
- Profiling infrastructure built-in (telemetry mode)
- Regular performance reviews during development

**Status**: Mitigated by testing (continuous benchmarking)

## How to Use This Document

**For contributors:**
- Check off items as you complete them
- Update progress regularly
- Raise blockers early
- Celebrate milestones!

**For project leads:**
- Track progress against timeline
- Identify risks and dependencies
- Make scope decisions based on guardrails
- Validate acceptance criteria

**For users:**
- Understand what's coming in v1
- See what's intentionally deferred
- Know when to expect features
- Provide feedback on priorities

---

**Remember:** This is a marathon, not a sprint. The goal is to ship a powerful, reliable v1 that proves the model works. Everything else can wait.
