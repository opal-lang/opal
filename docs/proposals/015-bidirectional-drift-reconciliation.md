---
oep: 015
title: Bidirectional Drift Reconciliation
status: Draft
type: Tooling
created: 2025-01-22
updated: 2025-01-22
---

# OEP-015: Bidirectional Drift Reconciliation

## Summary

Introduce two complementary commands—`opal apply` and `opal bless`—that resolve drift reported by `opal review` in either direction. `opal apply` enforces contracts by mutating reality to match code, while `opal bless` updates Opal source files to reflect intentional out-of-band changes.

## Motivation

Opal's contract model embraces real-world operations where both automated pipelines and human operators modify infrastructure. Today, reconciling drift requires manual editing, ad-hoc scripts, or risky imperative actions. Providing first-class commands for both directions of reconciliation delivers:

- Safe, auditable enforcement of desired state when code is authoritative.
- Pragmatic capture of emergency/manual fixes back into version control.
- A complete closed-loop workflow spanning review, enforcement, and codification.

## Proposal

### `opal apply` — Reality ← Code

- Consumes the drift report produced by `opal review` (or re-resolves the plan) and computes mutations necessary to align reality with contracts.
- Executes provider-specific actions with idempotence guarantees derived from contract metadata (idempotence keys, onMismatch policies).
- Requires explicit confirmation (`--yes` or interactive prompt) before destructive actions.
- Emits a structured summary (JSON + human-readable) detailing actions taken, failures, and residual drift.
- Exit codes: `0` when drift resolved successfully, `4` when residual drift remains, non-zero for execution errors.

### `opal bless` — Reality → Code

- Reads drift data and rewrites the corresponding Opal files so that contract values reflect reality.
- Applies updates deterministically:
  - Locates contract definitions via source spans recorded during planning.
  - Updates literals and decorator arguments while preserving formatting conventions.
  - Records annotations in git-style change summaries for operator review.
- Supports scope flags (`--resource`, `--file`, `--all`) to constrain which drifts are codified.
- Provides a dry-run mode that prints proposed edits without touching files.

### Shared Behaviors

- Both commands persist reconciliation metadata (timestamp, operator identity) for observability hooks introduced in OEP-013.
- Both integrate with the transport layer to ensure consistent execution semantics across local, SSH, and future providers.
- Commands are additive and discoverable via `opal help` grouping under a new "Drift" command family.

## Rationale

By treating reconciliation as two symmetric operations, Opal respects both developer-driven desired state and operator-driven actual state. This bifurcation avoids overloading a single command with conflicting semantics and aligns with the project's stateless philosophy: contracts remain the source of truth, but operators have a sanctioned path to accept reality as the new truth.

## Alternatives Considered

- **Single `opal amend` command:** rejected because conflating enforcement and codification increases risk and complicates UX. Splitting into `apply` and `bless` clarifies intent and reduces accidental destructive changes.
- **Relying on manual git edits:** rejected due to error-prone workflows, lack of deterministic formatting, and poor auditability.
- **Delegating to provider-native tooling:** rejected because Opal-specific decorators, contracts, and transports cannot be reconciled externally without losing semantic guarantees.

## Implementation

1. Extend the CLI router to register `apply` and `bless` under a shared drift reconciliation module.
2. For `apply`, enhance the execution runtime to interpret drift deltas as actions, reusing deployment code paths with additional safeguards (confirmation prompts, rollback on failure).
3. For `bless`, add a source-editing engine that maps drift fields to AST updates, leveraging formatter rules from OPAL_STYLE.md to preserve style.
4. Persist reconciliation results to telemetry streams defined in OEP-013 and log structured events for auditing.
5. Author integration tests that stub providers and file systems to validate both success and failure flows, including partial reconciliation scenarios.
6. Update documentation and tutorials to cover the review → apply/bless workflow.

## Compatibility

The commands are additive. `opal apply` may introduce breaking changes only when providers lack mutation support; such providers must advertise limited capability via feature flags. `opal bless` modifies source files but will respect `.opalignore` (if introduced) and should be gated behind explicit operator confirmation.

## Open Questions

- How should `opal bless` handle conflicts when concurrent edits alter the same file segments?
- Should `opal apply` support speculative execution (previewing mutations) beyond confirmation prompts?
- What safeguards are required to prevent accidental blessing of sensitive secrets into source control?

## References

- OEP-014: Drift Review Command
- OEP-010: Infrastructure as Code (IaC)
- OEP-013: Observability & Telemetry Hooks
- GitOps workflows (Argo CD, Flux) for reconciliation patterns
