---
oep: 011
title: System Shell
status: Draft
type: Long-term Vision
created: 2025-01-21
updated: 2025-01-21
---

# OEP-011: System Shell

## Summary

Explore the possibility of Opal becoming a daily-driver shell. This is a long-term vision requiring significant research and validation.

## Motivation

### The Question

Could Opal be a daily-driver shell?

**Current shells (bash, zsh):**
- Powerful but unsafe (no type system, error handling)
- Difficult to debug (implicit state, global variables)
- Hard to test (side effects everywhere)
- Poor IDE support (no autocomplete, no type checking)

**Opal advantages:**
- Type system (catch errors early)
- Error handling (explicit error propagation)
- Testability (pure functions, explicit dependencies)
- IDE support (LSP, autocomplete, type checking)
- Plan-first execution (review before running)

### Use Cases

**1. Interactive shell:**
```bash
$ opal
opal> ls
opal> cd /tmp
opal> pwd
/tmp
```

**2. Shell scripting:**
```bash
$ cat deploy.opl
deploy: {
    kubectl apply -f k8s/
    curl /health |> assert.re("Status 200")
}

$ opal deploy.opl
```

**3. Job control:**
```bash
opal> npm run dev &
opal> npm run test
opal> fg
```

## Proposal

### What's Needed

**Phase 1: REPL Infrastructure**
- Interactive command execution
- Command history
- Autocomplete
- Multi-line input

**Phase 2: Built-in Commands**
- `cd` - Change directory
- `pwd` - Print working directory
- `ls` - List files
- `cat` - Print file contents
- `echo` - Print text
- `exit` - Exit shell

**Phase 3: I/O Redirection**
- `>` - Redirect stdout
- `>>` - Append stdout
- `<` - Redirect stdin
- `2>` - Redirect stderr
- `|` - Pipe

**Phase 4: Job Control**
- `&` - Background job
- `fg` - Foreground job
- `bg` - Background job
- `jobs` - List jobs
- `kill` - Kill job

**Phase 5: Environment**
- Environment variables
- Shell variables
- Function definitions
- Aliases

### Comparison with Bash

| Feature | Bash | Opal |
|---------|------|------|
| Type system | No | Yes |
| Error handling | Implicit | Explicit |
| IDE support | No | Yes (LSP) |
| Plan-first | No | Yes |
| Testability | Hard | Easy |
| Learning curve | Steep | Moderate |
| Ecosystem | Huge | Growing |

## Rationale

### Why explore this?

**Potential:** Opal could be a better shell.

**Research:** Need to validate if it's feasible.

**Vision:** Long-term goal for the project.

### Why not do it now?

**Scope:** Too large for current phase.

**Validation:** Need to validate demand first.

**Priorities:** Focus on core language features first.

## Alternatives Considered

### Alternative 1: Extend bash

**Rejected:** Bash is too entrenched. Better to create new shell.

### Alternative 2: Ignore shell use case

**Rejected:** Shell is important use case.

### Alternative 3: Use existing shell (zsh, fish)

**Rejected:** Opal has unique advantages (type system, plan-first).

## Implementation

### Phase 1: Research (Q1 2025)
- Survey users about shell needs
- Prototype REPL
- Evaluate feasibility

### Phase 2: Prototype (Q2 2025)
- Implement basic REPL
- Implement built-in commands
- Test with early adopters

### Phase 3: MVP (Q3 2025)
- Full REPL functionality
- I/O redirection
- Job control

### Phase 4: Production (Q4 2025+)
- Performance optimization
- Compatibility with bash scripts
- Documentation

## Compatibility

**Breaking changes:** N/A (long-term vision, not implemented yet).

**Migration path:** N/A (long-term vision, not implemented yet).

## Open Questions

1. **Bash compatibility:** Should Opal shell be compatible with bash scripts?
2. **Performance:** Can Opal shell match bash performance?
3. **Ecosystem:** Can Opal shell leverage existing shell tools?
4. **Adoption:** Would users switch from bash to Opal?
5. **Maintenance:** Can we maintain both Opal and shell?

## References

- **Bash:** Traditional shell (contrast)
- **Zsh:** Modern shell (inspiration)
- **Fish:** User-friendly shell (inspiration)
- **Nushell:** Structured shell (inspiration)
- **Related OEPs:**
  - OEP-005: Interactive REPL (foundation for shell)
  - OEP-006: LSP/IDE Integration (IDE support for shell)
