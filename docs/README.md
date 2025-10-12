---
title: "Opal Documentation"
audience: "All"
summary: "Navigation guide for contributors, users, and operators"
---

# Opal Documentation

**Navigation guide for contributors, users, and operators**

## For End Users & Script Authors

**Start here** if you're writing Opal scripts for deployments, operations, or automation:

- **[SPECIFICATION.md](SPECIFICATION.md)** - Complete language guide with examples
  - Mental model: source → plan → contract → execute
  - Syntax, decorators, control flow, variables
  - Planning modes and contract verification
  - Real-world deployment examples

- **[GRAMMAR.md](GRAMMAR.md)** - Formal EBNF syntax reference
  - Quick lookup for syntax rules
  - Operator precedence
  - Lexical elements

## For Core Developers & Contributors

**Start here** if you're working on the Opal runtime, parser, or execution engine:

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System design and implementation model
  - Plan → contract → execute pipeline
  - Dual-path architecture (execution vs tooling)
  - Determinism and safety guarantees
  - Module organization

- **[AST_DESIGN.md](AST_DESIGN.md)** - Parser and AST implementation
  - Event-based parse tree
  - Zero-copy pipeline for execution
  - AST construction for tooling (LSP, linters)
  - Error recovery and resilient parsing

- **[TESTING_STRATEGY.md](TESTING_STRATEGY.md)** - Testing approach and invariants
  - Core tests (golden plans, conformance, performance)
  - Advanced tests (fuzzing, simulation, security)
  - Contract verification tests
  - Decorator conformance suite

## For Decorator Authors & Plugin Developers

**Start here** if you're building decorators or extending Opal:

- **[DECORATOR_GUIDE.md](DECORATOR_GUIDE.md)** - Design patterns and best practices
  - Invariants: referential transparency, determinism, observability
  - Patterns: opaque handles, resource collections, memoization
  - Composition guidelines and naming conventions
  - Checklist for new decorators

## For Operators & DevOps Engineers

**Start here** if you're running Opal in production:

- **[OBSERVABILITY.md](OBSERVABILITY.md)** - Tracing, artifacts, and debugging
  - Run identification and plan hashes
  - Artifact storage and retention
  - OpenTelemetry integration
  - Post-execution debugging

## Future Vision & Roadmap

**Experimental ideas and long-term direction:**

- **[FUTURE_IDEAS.md](FUTURE_IDEAS.md)** - Potential extensions and experiments
  - 🧪 Experimental: Plan-first execution, REPL
  - ⚙️ Feasible: LSP/IDE integration
  - 🧭 Long-term: IaC (ops-focused, ephemeral-friendly), system shell

## Quick Reference

| I want to... | Read this |
|--------------|-----------|
| Write Opal scripts | [SPECIFICATION.md](SPECIFICATION.md) |
| Look up syntax | [GRAMMAR.md](GRAMMAR.md) |
| Understand the architecture | [ARCHITECTURE.md](ARCHITECTURE.md) |
| Work on the parser | [AST_DESIGN.md](AST_DESIGN.md) |
| Build a decorator | [DECORATOR_GUIDE.md](DECORATOR_GUIDE.md) |
| Add tests | [TESTING_STRATEGY.md](TESTING_STRATEGY.md) |
| Debug production runs | [OBSERVABILITY.md](OBSERVABILITY.md) |
| Explore future ideas | [FUTURE_IDEAS.md](FUTURE_IDEAS.md) |

## Documentation Philosophy

Each document targets a specific audience:

- **User-facing** (SPECIFICATION, GRAMMAR): Focus on what Opal feels like to use
- **Developer-facing** (ARCHITECTURE, AST_DESIGN, TESTING): Focus on how it works
- **Contributor-facing** (DECORATOR_GUIDE): Focus on extending Opal
- **Operator-facing** (OBSERVABILITY): Focus on running Opal in production

Cross-references link related concepts across documents.

## Contributing to Documentation

**New to the project?** Start here:

1. **[ARCHITECTURE.md](ARCHITECTURE.md)** - Understand the system design
2. **[TESTING_STRATEGY.md](TESTING_STRATEGY.md)** - Learn how we verify correctness
3. **[AST_DESIGN.md](AST_DESIGN.md)** - Dive into parser implementation

**Improving documentation:**
- Keep audience separation clear (users vs developers vs operators)
- Add cross-references when introducing related concepts
- Include examples for abstract concepts
- Update WORK.md when making significant changes

**Documentation standards:**
- Use consistent terminology across documents
- Link to authoritative definitions (e.g., "See SPECIFICATION.md for...")
- Add "Semantic Notes" for validation rules vs syntax
- Include "Why this matters" context for design decisions
