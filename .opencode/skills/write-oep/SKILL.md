---
name: write-oep
description: >
  Write high-quality Opal Enhancement Proposals (OEPs). Use when creating new
  proposals, editing existing proposals, or reviewing proposal drafts. Ensures
  proposals are problem-focused, evidence-based, and free from unsubstantiated
  claims or AI hype.
---

# Writing Opal Enhancement Proposals (OEPs)

This skill guides you through creating clear, honest, and technically sound OEPs.

## Core Principles

### 1. Problem-First Writing

Start with the problem. Not the solution. Not the hype.

```markdown
# GOOD
The current executor lacks cancellation semantics. Long-running operations
cannot be interrupted, causing hangs in CI pipelines when jobs time out.

# BAD
This proposal introduces revolutionary async capabilities that will transform
the executor into a world-class system.
```

### 2. Evidence-Based Claims

Every statistic or strong claim needs a citation.

```markdown
# GOOD
According to the OWASP Top 10 for LLM Applications (2024), prompt injection
is classified as the highest-risk vulnerability category.

# BAD
Studies show 99.9% of LLM applications are vulnerable to prompt injection.

# ACCEPTABLE
Prompt injection remains a significant security concern for LLM applications.
```

### 3. Concrete Over Abstract

Use specific examples, not vague descriptions.

```markdown
# GOOD
When a user runs `opal deploy production`, the plan shows:
1. kubectl set image deployment/app app=myapp:v1.2.3
2. kubectl rollout status deployment/app

# BAD
The system will intelligently orchestrate deployment workflows.
```

### 4. Honest About Trade-offs

Every solution has downsides. Acknowledge them.

```markdown
# GOOD
## Trade-offs

- **Pros**: Deterministic plans enable review before execution
- **Cons**: Requires learning Opal syntax; adds planning overhead
```

### 5. No AI Hype

Avoid buzzwords and superlatives.

**Forbidden words:** revolutionary, game-changing, cutting-edge, seamless,
extremely, incredibly, dramatically, fundamentally, transformative

**Instead use:** specific, measurable descriptions of what the proposal does.

## OEP Structure

### Required Sections

```markdown
## Summary
One paragraph. What problem and high-level solution.

## Motivation
- The problem (with concrete examples)
- Why it matters
- Current workarounds (if any)

## Proposal
- Core concept (in plain English)
- Specific mechanisms
- Syntax/configuration changes

## Examples
3+ complete, runnable examples showing the feature in use

## Rationale
- Why this approach
- Why not alternatives
- Trade-offs

## Alternatives Considered
At least 3 alternatives with honest reasons for rejection

## References
Citations for any claims made in the proposal
```

### What to Avoid

❌ **Implementation details in early sections** — Save for proposal section
❌ **Future work speculation** — Focus on what the proposal does
❌ **Marketing language** — "This will enable..." → "This enables..."
❌ **Over-promising** — Don't claim performance without benchmarks

## Writing the Motivation Section

The motivation should make readers feel the pain of the current state.

### Template

```markdown
## Motivation

### The Problem

Describe the current limitation. Include:
- Concrete scenario where user hits this
- What goes wrong
- Why existing workarounds fail

### Use Cases

List 3-5 specific use cases. For each:
1. Context (who, when)
2. Current painful approach
3. What the proposal enables
```

### Example

```markdown
## Motivation

### The Problem

Currently, Opal plans execute atomically. If step 15 of 20 fails, the
previous 14 steps are not rolled back, leaving infrastructure in a
partially-modified state that requires manual cleanup.

This occurred during last week's production deployment where:
1. Database migrations succeeded
2. Application deployments succeeded
3. Cache warming failed
4. The system remained in a degraded state for 45 minutes

### Use Cases

**1. Safe Database Migrations**

When running schema changes that affect multiple tables, partial failures
leave the database inconsistent. Automatic rollback ensures the schema
returns to the previous state.

[Additional use cases...]
```

## Writing the Proposal Section

### Core Concept First

Explain the idea in 2-3 sentences a non-expert could understand.

```markdown
## Proposal

Add rollback capabilities to Opal's plan execution. When a step fails,
the executor runs previously-defined rollback operations in reverse order,
returning the system to its initial state.
```

### Then Add Detail

After the concept is clear, add:
- Syntax changes
- Configuration options
- Integration points
- Error handling

## Writing Examples

Examples should be complete and realistic.

```markdown
## Example 1: Basic Rollback

```opal
migrate: {
    @rollback { kubectl delete job migrate-back }
    kubectl apply -f migrate-job.yaml
    
    @rollback { kubectl rollout undo deployment/app }
    kubectl set image deployment/app app=myapp:new
}
```

When `kubectl set image` fails, the migration job is automatically deleted
via `kubectl delete job migrate-back`.
```

## Citing Sources

### When to Cite

- Statistics or percentages
- Security vulnerability rankings
- Performance claims
- Academic concepts
- Industry standards

### Format

```markdown
Research indicates prompt injection is the #1 vulnerability for LLM
applications [1].

## References

[1] OWASP Foundation. "OWASP Top 10 for LLM Applications 2024."
    https://owasp.org/www-project-top-10-for-large-language-model-applications/
```

### When No Source Exists

Remove the claim or soften it:

```markdown
# Instead of "85% attack success rate"
Prompt injection attacks have demonstrated high success rates against
production LLM systems [2].

[2] Greshake et al. "Not What You've Signed Up For: Compromising Real-World
     LLM-Integrated Applications with Indirect Prompt Injection." arXiv:2302.12173
```

## Pre-Submit Checklist

Before finalizing an OEP:

- [ ] Summary is one paragraph, explains problem and solution
- [ ] Motivation includes concrete examples of the problem
- [ ] No unsubstantiated statistics or claims
- [ ] Examples are complete and runnable
- [ ] Alternatives section has at least 3 honest alternatives
- [ ] No forbidden hype words (revolutionary, game-changing, etc.)
- [ ] References section matches all citations in text
- [ ] Trade-offs are acknowledged
- [ ] Proposal explains WHY, not just WHAT

## Common Mistakes

### Mistake 1: Solution-First Writing

```markdown
# BAD - Starts with solution
## Summary

This proposal adds a new `@retry` decorator with exponential backoff,
circuit breaker patterns, and configurable timeouts.

## Motivation

Sometimes operations fail and need to be retried.
```

```markdown
# GOOD - Starts with problem
## Summary

Network operations in deployment scripts often fail transiently. This
proposal adds automatic retry capabilities to reduce manual intervention.

## Motivation

Last month, 23% of our CI deployments failed due to transient network
errors during Docker pulls. Each failure required manual re-triggering...
```

### Mistake 2: Vague Benefits

```markdown
# BAD
This will significantly improve performance and user experience.

# GOOD
This reduces plan generation time from ~500ms to ~50ms for scripts
with 100+ decorators, measured on a 2023 MacBook Pro.
```

### Mistake 3: Missing Context

```markdown
# BAD
Add a new function to handle errors.

# GOOD
Add error aggregation to the plan validation phase. Currently, validation
stops at the first error; users must fix and re-run repeatedly. This
change collects all validation errors before failing.
```

## Reviewing OEPs

When asked to review an OEP:

1. **Check for hype** — Scan for forbidden words
2. **Verify citations** — Every claim needs a source
3. **Test examples** — Are they complete and correct?
4. **Check alternatives** — Are rejected alternatives treated fairly?
5. **Read backwards** — Start with the problem, does the solution follow?

### Review Comment Template

```markdown
## Review

### Strengths
- Clear problem statement with concrete examples
- Good trade-off discussion

### Suggestions
- [Line 45] Remove "revolutionary" — use specific description instead
- [Line 67] Add citation for the 85% statistic
- [Section 4] Consider adding an example showing failure mode

### Questions
- How does this handle nested decorators?
- What's the migration path for existing scripts?
```

## References

- OEP-000: Template and structure guide
- Opal Documentation Guidelines
- Plain English Campaign: https://www.plainenglish.co.uk/
- Google's Technical Writing Guide
