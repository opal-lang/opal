---
oep: 014
title: Drift Review Command
status: Draft
type: Tooling
created: 2025-01-22
updated: 2025-01-22
---

# OEP-014: Drift Review Command

## Summary

Introduce `opal review`, a read-only command that compares resolved Opal contracts against live infrastructure to produce a deterministic "drift report" for CI, governance, and operator workflows.

## Motivation

Teams operating Opal-managed infrastructure need clear visibility into configuration drift without risking unintended mutations. Today drift is detected opportunistically during deployments, forcing operators to choose between applying potentially destructive plans or running ad-hoc scripts. A dedicated review command allows:

- CI pipelines to surface drift in pull requests without mutating resources.
- Operators to audit production state before deciding whether to enforce or bless changes.
- Security and compliance teams to capture a traceable history of contract adherence.

## Proposal

- Add `opal review` to the CLI command set.
- The command executes a resolved plan in "observation" mode:
  - Fetch the declared resources for each contract.
  - Query the target provider(s) for the actual state.
  - Produce a structured drift report highlighting matches, mismatches, and missing resources.
- Output format requirements:
  - Deterministic ordering (sorted by contract identifier and resource name).
  - Rich annotations (contract source location, idempotence keys, mismatch fields).
  - Machine-parsable JSON option alongside human-readable text.
- Error handling:
  - Non-existent providers or authentication failures must fail the command with actionable messaging.
  - Partial failures should note which resources could not be inspected while continuing the report for others.
- Integrations:
  - Exit code `0` when no drift is detected, `3` when drift exists, and non-zero otherwise for execution errors to enable CI gating.
  - Emit OpenTelemetry spans so observability hooks can correlate review runs.

## Rationale

A dedicated review surface formalizes the "plan" phase for Opal's contract-first model while remaining non-destructive. Deterministic, read-only inspection aligns with Opal's stateless philosophy and enables governance workflows (policy checks, PR annotations) that would otherwise require bespoke tooling.

## Alternatives Considered

- **Overloading `opal plan --dry-run`:** rejected because deployments would still execute provider mutations for some decorators, and the semantics are harder to reason about than a purpose-built read-only command.
- **External drift detection scripts:** rejected because they splinter tooling, lack shared contracts, and cannot guarantee determinism across transports.
- **Provider-native drift tools:** rejected because they fail to account for Opal's contract semantics (idempotence keys, decorators) and cannot be integrated uniformly across providers.

## Implementation

1. Extend the CLI parser to register `review` with subcommand scaffolding consistent with existing commands.
2. Add a `ReviewEngine` to the runtime that resolves plans, invokes transport providers in observation mode, and aggregates comparison results.
3. Implement drift adapters for the first-party providers introduced in OEP-010, reusing their describe/read APIs.
4. Create a formatter that emits both human-readable tables and JSON payloads, with deterministic ordering and stable schemas.
5. Wire exit codes and telemetry exports, ensuring review runs integrate with existing logging and tracing infrastructure.
6. Add end-to-end tests that stub providers to simulate matched, mismatched, and missing resources, verifying both textual and JSON outputs.

## Compatibility

`opal review` is additive. It does not modify existing commands, but it depends on provider APIs exposing read-only describe capabilities. Providers lacking observation support should emit informative errors and be documented as unsupported until implemented.

## Open Questions

- How should review handle decorators that encapsulate imperative scripts (e.g., `@shell` blocks) with no concrete resource handles?
- What is the minimal JSON schema that balances human readability with machine parsing for policy engines?
- Should review cache provider responses to accelerate repeated runs within a single CI job?

## References

- OEP-010: Infrastructure as Code (IaC)
- OEP-008: Plan-First Execution Model
- Terraform `plan` drift detection semantics
