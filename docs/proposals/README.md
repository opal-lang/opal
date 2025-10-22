---
title: "Opal Enhancement Proposals (OEPs)"
audience: "Project Leads & Contributors"
summary: "Formal design documents for future Opal features"
---

# Opal Enhancement Proposals (OEPs)

This directory contains formal design documents for proposed Opal features. Each OEP follows a standard format to ensure clarity, completeness, and community discussion.

## Status Levels

| Status | Meaning | Action |
|--------|---------|--------|
| **Draft** | Proposal under development, open for feedback | Read and comment |
| **Review** | Proposal ready for community review | Provide feedback via issues/PRs |
| **Accepted** | Design approved by maintainers | Implementation can begin |
| **Implementing** | Development in progress | Track progress in linked PR |
| **Implemented** | Merged and released | Feature is available |
| **Rejected** | Not proceeding (with rationale) | Learn from decision |
| **Withdrawn** | Author withdrew proposal | Revisit later if needed |

## OEP Categories

### Language Evolution (Core Features)

These OEPs propose new language constructs and semantics.

- **OEP-001: Runtime Variable Binding with `let`** (Draft)
  - Execution-time variable bindings from decorator effects
  - Separate `@let` namespace from plan-time `@var`
  - Enables capture of infrastructure handles, endpoints, certificates
  - **Status:** Design phase (core feature)
  - **Restrictions:** No `@let` in plan-time constructs, no reassignment, no shadowing

- **OEP-002: Transform Pipeline Operator `|>`** (Draft)
  - Deterministic transformations on command output
  - Pure, bounded, deterministic PipeOps
  - Inline assertions (`assert.re`, `assert.num`)
  - **Status:** Design phase (core feature)
  - **Restrictions:** PipeOps must be pure, bounded, deterministic

- **OEP-003: Automatic Cleanup and Rollback** (Draft)
   - `defer` for LIFO cleanup on failure
   - `ensure`/`rollback` operators for execution control
   - Comparison of both approaches and tradeoffs
   - **Status:** Design phase (core feature)
   - **Restrictions:** Defers run only on failure, LIFO ordering

- **OEP-004: Plan Verification** (Implemented)
   - Audit trail and contract verification
   - CI/CD workflow for plan review
   - Differential analysis between plans
   - **Status:** Core feature (see SPECIFICATION.md)

### Tooling Enhancements (Developer Experience)

These OEPs propose tools and integrations to improve the developer experience.

- **OEP-005: Interactive REPL** (Draft)
  - Command-line REPL for Opal
  - Function definitions, plan mode, decorator integration
  - **Status:** Design phase

- **OEP-006: LSP/IDE Integration** (Draft)
  - Language Server Protocol support
  - Syntax checking, autocomplete, jump to definition
  - Hover documentation, rename refactoring
  - **Status:** Design phase

- **OEP-007: Standalone Binary Generation** (Draft)
   - Compile Opal scripts into standalone CLI binaries
   - Zero dependencies, air-gapped deployment
   - Plan-first execution built-in
   - **Status:** Design phase

- **OEP-008: Plan-First Execution Model** (Implemented)
   - REPL modes for planning and execution
   - Safe remote code execution
   - Hash-based trust for plans
   - **Status:** Core feature (see SPECIFICATION.md)

- **OEP-014: Drift Review Command** (Draft)
  - `opal review` reports contract vs reality drift without mutations
  - Deterministic human-readable and JSON outputs for CI/governance
  - CI-friendly exit codes and telemetry integration
  - **Status:** Design phase

- **OEP-015: Bidirectional Drift Reconciliation** (Draft)
  - `opal apply` enforces code-defined desired state safely
  - `opal bless` updates source to reflect intentional real-world changes
  - Shared drift metadata, confirmations, and formatting guarantees
  - **Status:** Design phase

### Ecosystem Extensions (Reach & Integration)

These OEPs propose integrations with external systems and providers.

- **OEP-009: Terraform/Pulumi Provider Bridge** (Draft)
  - Schema import and codegen from Terraform/OpenTofu
  - Runtime adapter for provider execution
  - Plugin manifest for IDE experience
  - **Status:** Design phase

- **OEP-010: Infrastructure as Code (IaC)** (Draft)
  - Outcome-focused infrastructure provisioning
  - Deploy blocks (run on first creation) vs SSH blocks (run always)
  - Flexible idempotence matching
  - **Status:** Design phase

- **OEP-012: Module Composition and Plugin System** (Draft)
  - WebAssembly and native plugin support (language-agnostic)
  - Registry-based distribution (`opal add hashicorp/aws`)
  - Git repository sources
  - Host-driven execution (plugins provision, Opal executes)
  - **Status:** Design phase

- **OEP-013: Observability & Telemetry Hooks** (Draft)
  - Run tracking with plan/execution correlation
  - OpenTelemetry integration
  - Decorator usage tracking and security audit
  - Compliance reporting (SOC2, HIPAA)
  - **Status:** Design phase
  - **Note:** Advanced features may be part of Opal Cloud [PREMIUM]

### Long-Term Vision (Strategic Direction)

These OEPs propose strategic directions for the project.

- **OEP-011: System Shell** (Draft)
  - Could Opal be a daily-driver shell?
  - REPL infrastructure, built-in commands, job control
  - **Status:** Design phase (research phase)

## How to Read an OEP

Each OEP follows this structure:

1. **Summary** - One paragraph overview
2. **Motivation** - Why is this needed? What problems does it solve?
3. **Proposal** - Detailed design with examples
4. **Rationale** - Why this design? Key decisions explained
5. **Alternatives Considered** - What other approaches were evaluated and why rejected?
6. **Implementation** - Phases and technical approach
7. **Compatibility** - Breaking changes? Migration path?
8. **Open Questions** - Unresolved design issues
9. **References** - Related OEPs and external inspiration

## How to Contribute

### Providing Feedback

1. Read the OEP carefully
2. Look for:
   - **Clarity:** Is the proposal clear and unambiguous?
   - **Completeness:** Are all edge cases covered?
   - **Consistency:** Does it fit with existing Opal design?
   - **Feasibility:** Can this be implemented reasonably?
3. Open an issue or PR with your feedback
4. Reference the OEP number (e.g., "OEP-001: Runtime Variable Binding")

### Proposing New OEPs

1. Check if a similar OEP already exists
2. Copy the template from an existing OEP
3. Fill in all sections (don't skip "Alternatives Considered" or "Open Questions")
4. Submit as a PR to `docs/proposals/`
5. Engage with community feedback
6. Update based on feedback
7. Maintainers review and accept/reject

### OEP Template

```markdown
---
oep: NNN
title: Feature Name
status: Draft
type: Feature | Tooling | Integration
created: YYYY-MM-DD
updated: YYYY-MM-DD
---

# OEP-NNN: Feature Name

## Summary
One paragraph overview.

## Motivation
Why is this needed?

## Proposal
Detailed design with examples.

## Rationale
Why this design?

## Alternatives Considered
What else was evaluated?

## Implementation
Phases and technical approach.

## Compatibility
Breaking changes?

## Open Questions
Unresolved issues.

## References
Related OEPs and inspiration.
```

## Implementation Roadmap

### Phase 1: Language Evolution (Q1 2025)
- OEP-001: Runtime Variable Binding with `let`
- OEP-002: Transform Pipeline Operator `|>`
- OEP-003: Automatic Cleanup and Rollback

### Phase 2: Tooling Enhancements (Q2 2025)
- OEP-005: Interactive REPL
- OEP-006: LSP/IDE Integration
- OEP-007: Standalone Binary Generation

### Phase 3: Ecosystem Extensions (Q3 2025)
- OEP-009: Terraform/Pulumi Provider Bridge
- OEP-012: Module Composition and Plugin System

### Phase 4: Observability & Long-Term (Q4 2025+)
- OEP-013: Observability & Telemetry Hooks
- OEP-010: Infrastructure as Code (IaC)
- OEP-011: System Shell

## FAQ

**Q: What's the difference between an OEP and FUTURE_IDEAS.md?**

A: FUTURE_IDEAS.md is a brainstorm document with rough ideas. OEPs are formal design documents with detailed specifications, restrictions, and implementation plans. OEPs are what we commit to implementing.

**Q: Can I propose an OEP?**

A: Yes! Follow the template and submit a PR. Maintainers will review and provide feedback.

**Q: What if I disagree with an OEP?**

A: Open an issue or PR with your concerns. OEPs are living documents and can be updated based on feedback.

**Q: When will OEP-002 be available?**

A: Check the status in the OEP file. "Draft" means it's still being designed. "Accepted" means design is approved. "Implementing" means development is in progress.

**Q: Can I implement an OEP myself?**

A: Yes! Open an issue to discuss, then submit a PR. Maintainers will review and provide guidance.

## Related Documents

- **FUTURE_IDEAS.md** - Brainstorm document with rough ideas
- **SPECIFICATION.md** - Current language specification
- **ARCHITECTURE.md** - System architecture and design principles
- **TESTING_STRATEGY.md** - Testing approach and requirements
