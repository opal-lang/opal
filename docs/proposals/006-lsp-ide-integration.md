---
oep: 006
title: LSP/IDE Integration
status: Draft
type: Tooling
created: 2025-01-21
updated: 2025-01-21
---

# OEP-006: LSP/IDE Integration

## Summary

Add Language Server Protocol (LSP) support for Opal, enabling IDE integration with syntax checking, autocomplete, jump to definition, hover documentation, and rename refactoring.

## Motivation

### The Problem

Current Opal has no IDE support:
- No syntax checking as you type
- No autocomplete for decorators and functions
- No jump to definition
- No hover documentation
- No rename refactoring

**Example of current limitations:**

```bash
# ❌ No IDE support
# - Typos not caught until runtime
# - No autocomplete for decorators
# - No documentation on hover
```

### Use Cases

**1. Real-time syntax checking:**
- Catch typos and syntax errors as you type
- Underline errors in red
- Show error messages on hover

**2. Autocomplete:**
- `@` → show all decorators
- `@shell(` → show parameters
- `@retry(` → show parameters with types

**3. Jump to definition:**
- Click on function name → jump to definition
- Click on decorator → jump to documentation

**4. Hover documentation:**
- Hover over decorator → show documentation
- Hover over function → show signature

**5. Rename refactoring:**
- Rename function → rename all usages
- Rename variable → rename all usages

## Proposal

### LSP Server Implementation

Implement a Language Server that supports:

- **Diagnostics:** Syntax errors, type errors, unused variables
- **Completion:** Decorators, functions, variables, keywords
- **Hover:** Documentation, type information, signatures
- **Definition:** Jump to function/decorator definition
- **References:** Find all usages of a symbol
- **Rename:** Rename symbols across file
- **Formatting:** Format code (gofumpt-style)

### IDE Support

#### VS Code

```json
{
  "language": "opal",
  "scopeName": "source.opal",
  "extensions": [".opl"],
  "server": {
    "command": "opal",
    "args": ["lsp"]
  }
}
```

#### Vim/Neovim

```vim
let g:lsp_servers = [{
  'name': 'opal',
  'cmd': {server_info -> ['opal', 'lsp']},
  'whitelist': ['opal'],
}]
```

#### Emacs

```elisp
(lsp-register-client
  (make-lsp-client
    :new-connection (lsp-stdio-connection '("opal" "lsp"))
    :major-modes '(opal-mode)
    :server-id 'opal-lsp))
```

### Core Restrictions

#### Restriction 1: LSP is read-only

LSP server does not execute code:

```bash
# ❌ FORBIDDEN: LSP executing code
opal lsp --execute

# ✅ CORRECT: LSP only provides language services
opal lsp
```

**Why?** Safety. LSP is for analysis only, not execution.

#### Restriction 2: LSP respects file boundaries

LSP does not cross file boundaries for analysis:

```bash
# ❌ FORBIDDEN: analyzing imported files
# LSP analyzes deploy.opl which imports utils.opl
# LSP does not analyze utils.opl

# ✅ CORRECT: analyze each file independently
opal lsp < deploy.opl
opal lsp < utils.opl
```

**Why?** Simplicity. Each file is analyzed independently.

#### Restriction 3: LSP provides best-effort analysis

LSP may not catch all errors (some require execution):

```bash
# ❌ FORBIDDEN: expecting LSP to catch runtime errors
# LSP cannot catch errors that depend on runtime values

# ✅ CORRECT: LSP catches parse/type errors
# Runtime errors are caught during execution
```

**Why?** Some errors only manifest at runtime.

## Rationale

### Why LSP?

**Standard:** LSP is the standard for IDE integration.

**Ecosystem:** Works with all major IDEs (VS Code, Vim, Emacs, etc.).

**Maintenance:** Single implementation serves all IDEs.

### Why read-only?

**Safety:** LSP should not execute code.

**Simplicity:** Easier to implement and maintain.

**Performance:** Faster analysis without execution.

## Alternatives Considered

### Alternative 1: IDE-specific plugins

**Rejected:** Requires maintaining separate plugins for each IDE. LSP is more maintainable.

### Alternative 2: No IDE support

**Rejected:** IDE support is important for developer experience.

## Implementation

### Phase 1: Basic LSP
- Diagnostics (syntax errors)
- Completion (decorators, functions)
- Hover (documentation)

### Phase 2: Advanced Features
- Definition (jump to definition)
- References (find usages)
- Rename (rename symbols)

### Phase 3: IDE Integration
- VS Code extension
- Vim/Neovim plugin
- Emacs mode

### Phase 4: Polish
- Formatting support
- Semantic highlighting
- Code folding

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Incremental analysis:** Should LSP support incremental analysis for better performance?
2. **Caching:** Should LSP cache analysis results?
3. **Configuration:** Should LSP support configuration files (.opal-lsp.json)?
4. **Debugging:** Should LSP support debugging protocol (DAP)?
5. **Formatting:** Should LSP support code formatting?

## References

- **Language Server Protocol:** https://microsoft.github.io/language-server-protocol/
- **VS Code LSP:** https://code.visualstudio.com/api/language-extensions/language-server-extension-guide
- **Related OEPs:**
  - OEP-005: Interactive REPL (complementary tooling)
  - OEP-007: Standalone Binary Generation (distribution)
