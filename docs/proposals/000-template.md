---
oep: 000
title: OEP Template and Format Guide
status: Meta
type: Meta
created: 2025-01-21
updated: 2025-01-21
---

# OEP-000: OEP Template and Format Guide

**This is a meta-document explaining the OEP format. Use this as a template when creating new OEPs.**

## Summary

One paragraph overview of the feature. Should be clear enough that someone can understand the basic idea without reading the full proposal.

## Motivation

### The Problem

Describe the problem this OEP solves. What's missing or broken in current Opal?

**Format:**
- Start with current limitations
- Show concrete examples of what doesn't work
- Explain why this is a problem

### Use Cases

List 3-5 concrete use cases showing how this feature would be used in practice.

**Format:**
- Real-world scenarios
- Complete code examples
- Show the value/benefit

## Proposal

### Syntax

Show the syntax clearly with examples.

**Format:**
- Basic syntax first
- Advanced syntax second
- Edge cases third

**Example:**
```opal
// Basic syntax
let.NAME = <expression>
@let.NAME

// Advanced syntax
@let.instance.field
@let.pods[0].name
```

### Core Restrictions

**CRITICAL SECTION** - List all restrictions explicitly.

**Format:**
- Number each restriction (Restriction 1, 2, 3...)
- Show forbidden pattern with ❌
- Show correct pattern with ✅
- Explain WHY the restriction exists

**Example:**

#### Restriction 1: No `@let` in Plan-Time Constructs

`@let` bindings **cannot** be used in any construct that affects the plan structure:

```opal
// ❌ FORBIDDEN: if condition
let.env = @aws.instance.deploy().tag("environment")
if @let.env == "production" {  // Parse error
    kubectl apply -f k8s/prod/
}

// ✅ CORRECT: use @var for plan-time decisions
var ENV = @env.ENVIRONMENT
if @var.ENV == "production" {
    let.instance_id = @aws.instance.deploy().id
}
```

**Why?** Plan hash must be computable before execution. If `@let` could drive `if`/`for`/`when`, the plan structure would depend on runtime values, breaking contract verification.

### Semantics

Explain how the feature works in detail.

**Format:**
- Execution model
- Scope rules
- Interaction with other features
- Edge cases

### Examples

Show 3-5 complete, real-world examples.

**Format:**
- Complete workflows (not snippets)
- Real use cases (not toy examples)
- Show integration with other Opal features

## Rationale

### Why [design decision]?

For each major design decision, explain:
- What the decision was
- Why it was made
- What alternatives were considered
- What tradeoffs were accepted

**Format:**
- One subsection per major decision
- Clear reasoning
- Acknowledge tradeoffs

## Alternatives Considered

**CRITICAL SECTION** - Document what else was considered and why it was rejected.

**Format:**
- List 3-5 alternatives
- Explain each alternative
- Explain why it was rejected
- Be honest about tradeoffs

## Implementation

Break implementation into phases.

**Format:**
- 3-5 phases
- Clear deliverables per phase
- Dependencies between phases
- Estimated scope (if known)

## Compatibility
### Pre-Alpha Exception

During pre-alpha, Opal prefers clean breaks over migration scaffolding. Migration paths are **optional** unless:
 The proposal explicitly requests transition support
 A maintainer specifically requires compatibility handling

Default behavior for pre-alpha breaking changes:
 Remove old syntax/behavior directly
 Update tests and docs to canonical form
 No deprecation warnings or compatibility errors

### Breaking Changes

List any breaking changes. If none, say "None. This is a new feature."

**Format:**
 What breaks
 Why it breaks
 How to migrate (optional during pre-alpha)

### Migration Path

If a migration path is required (see Pre-Alpha Exception), provide clear steps:

**Format:**
 Step-by-step migration guide
 Automated migration tools (if applicable)
 Timeline for deprecation (post-alpha only)
## Open Questions

**CRITICAL SECTION** - List unresolved design issues.

**Format:**
- Number each question
- Be specific
- Indicate if community input is needed

## References

List related OEPs and external inspiration.

**Format:**
- Related OEPs (with links)
- External projects (with brief explanation)
- Academic papers (if applicable)

---

## OEP Writing Guidelines

### Tone and Style

- **Be clear:** Avoid ambiguity. Use concrete examples.
- **Be concise:** Get to the point. No fluff.
- **Be honest:** Acknowledge tradeoffs and limitations.
- **Be specific:** "No `@let` in if conditions" not "Some restrictions apply"

### Code Examples

- **Use ❌ for forbidden patterns**
- **Use ✅ for correct patterns**
- **Always explain WHY** after showing what's forbidden
- **Show complete examples**, not snippets
- **Use realistic scenarios**, not toy examples

### Restrictions

- **Number them** (Restriction 1, 2, 3...)
- **Be explicit** about what's forbidden
- **Show examples** of forbidden and correct patterns
- **Explain WHY** the restriction exists
- **Specify enforcement** (parse-time vs runtime)

### Alternatives

- **Be thorough** - list at least 3 alternatives
- **Be honest** - explain real reasons for rejection
- **Be fair** - acknowledge when alternatives have merit
- **Be specific** - show code examples of alternatives

### Open Questions

- **Be specific** - "Should X support Y?" not "What about X?"
- **Indicate priority** - which questions block implementation?
- **Ask for input** - which questions need community feedback?
- **Be realistic** - don't list questions you already know the answer to

## Checklist for New OEPs

Before submitting an OEP, verify:

- [ ] Summary is one paragraph and clear
- [ ] Motivation explains the problem with examples
- [ ] Use cases are realistic and complete
- [ ] Syntax is shown with examples
- [ ] All restrictions are listed and explained
- [ ] Semantics are explained in detail
- [ ] Examples are complete workflows
- [ ] Rationale explains major design decisions
- [ ] At least 3 alternatives are considered
- [ ] Implementation is broken into phases
- [ ] Compatibility section is complete
- [ ] Open questions are specific
- [ ] References are included
- [ ] All code examples use ❌/✅ markers
- [ ] All restrictions explain WHY

## Common Mistakes to Avoid

### ❌ Vague restrictions

```markdown
## Restrictions

Some restrictions apply to this feature.
```

### ✅ Specific restrictions

```markdown
## Core Restrictions

#### Restriction 1: No `@let` in Plan-Time Constructs

`@let` bindings **cannot** be used in if/for/when conditions.

[Show forbidden example]
[Show correct example]
[Explain WHY]
```

### ❌ Toy examples

Use complete, real-world examples instead of toy snippets.

### ✅ Real-world examples

Show complete workflows with realistic use cases.

### ❌ Missing rationale

Always explain why design decisions were made.

### ✅ Clear rationale

Explain clarity, safety, and tooling benefits. Acknowledge tradeoffs.

## Template Checklist

When using this template:

1. Copy this file to `NNN-feature-name.md`
2. Replace all placeholder text
3. Fill in all sections (don't skip any)
4. Add at least 3 alternatives considered
5. Add at least 3 open questions
6. Add at least 3 complete examples
7. Number all restrictions
8. Explain WHY for each restriction
9. Show ❌/✅ for all code examples
10. Review against checklist above

## Meta

This OEP (000) is a meta-document and does not propose a feature. It serves as:
- Template for new OEPs
- Format guide for OEP authors
- Quality checklist for OEP reviewers
- Reference for OEP structure

**Status:** This OEP is always "Meta" and never moves to other statuses.
