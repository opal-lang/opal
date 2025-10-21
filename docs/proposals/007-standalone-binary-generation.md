---
oep: 007
title: Standalone Binary Generation
status: Draft
type: Tooling
created: 2025-01-21
updated: 2025-01-21
---

# OEP-007: Standalone Binary Generation

## Summary

Add `opal compile` command to compile Opal scripts into standalone CLI binaries with built-in plan-first execution. Generated binaries have zero dependencies and work in air-gapped environments.

## Motivation

### The Problem

Current Opal requires installation:

```bash
# ❌ Requires Opal installation
opal deploy.opl

# ❌ Requires Go toolchain
go run deploy.opl
```

**Problems:**
- Can't distribute to users without Opal
- Can't use in air-gapped environments
- Can't use in locked-down systems
- Requires runtime dependency

### Use Cases

**1. Air-gapped deployment:**
```bash
# On internet-connected machine
opal compile deploy.opl -o deploy

# Transfer to air-gapped system
scp deploy air-gapped:/opt/ops/

# On air-gapped system (no Opal, no Go, nothing)
./deploy --dry-run
./deploy
```

**2. Customer distribution:**
```bash
# Compile ops tool
opal compile commands.opl -o myapp

# Distribute to customers
./myapp --help
./myapp deploy --dry-run
./myapp deploy
```

**3. Project CLI:**
```bash
# Compile project task runner
opal compile Makefile.opl -o dev

# Commit to repo
git add dev
git commit -m "Add compiled dev CLI"

# New developer clones repo
git clone repo
./dev setup
./dev test
./dev deploy
```

## Proposal

### Compilation

#### Basic Compilation

```bash
# Compile to binary
opal compile commands.opl -o myapp

# Use the binary
./myapp --help
./myapp deploy --dry-run
./myapp deploy
```

#### Advanced Compilation

```bash
# Custom metadata
opal compile commands.opl \
    --name "myapp" \
    --version "1.2.3" \
    --author "team@example.com" \
    -o dist/myapp

# Cross-compile
opal compile commands.opl \
    --targets linux-amd64,darwin-arm64,windows-amd64 \
    -o dist/

# Embed resources
opal compile commands.opl \
    --embed k8s/ \
    --embed configs/ \
    -o myapp
```

### Generated Binary Features

#### Plan-First Execution

All commands support `--dry-run`:

```bash
./myapp deploy --dry-run
# Plan: 5f6c...
#   1. kubectl apply -f k8s/prod/
#   2. kubectl scale --replicas=3 deployment/app

./myapp deploy
# ✓ Executed successfully
```

#### Contract Verification

Built-in plan verification:

```bash
# Generate plan
./myapp deploy --dry-run --resolve > plan.txt

# Execute with plan verification
./myapp deploy --plan plan.txt
```

#### Source Visibility

Extract source for security review:

```bash
./myapp --show-source > audit.opl

# Security team reviews audit.opl
```

### Core Restrictions

#### Restriction 1: Binaries are immutable

Once compiled, binaries cannot be modified:

```bash
# ❌ FORBIDDEN: modifying binary
echo "malicious code" >> myapp

# ✅ CORRECT: recompile
opal compile commands.opl -o myapp
```

**Why?** Integrity. Binaries must be trustworthy.

#### Restriction 2: Embedded source is read-only

Embedded source cannot be modified:

```bash
# ❌ FORBIDDEN: modifying embedded source
./myapp --show-source | sed 's/prod/staging/' > modified.opl

# ✅ CORRECT: recompile with new source
opal compile modified.opl -o myapp
```

**Why?** Integrity. Source must match binary behavior.

#### Restriction 3: Binary size is bounded

Compiled binaries must be reasonable size (< 50MB):

```bash
# ❌ FORBIDDEN: embedding huge files
opal compile commands.opl \
    --embed huge-database.db \
    -o myapp
# Error: embedded files too large

# ✅ CORRECT: embed reasonable files
opal compile commands.opl \
    --embed k8s/ \
    -o myapp
```

**Why?** Portability. Binaries must be distributable.

## Rationale

### Why standalone binaries?

**Portability:** Single binary, no dependencies.

**Security:** Can verify binary integrity.

**Simplicity:** No installation required.

**Air-gapped:** Works in isolated environments.

### Why embed source?

**Auditability:** Source visible for security review.

**Debugging:** Can inspect what binary does.

**Compliance:** Meets audit requirements.

## Alternatives Considered

### Alternative 1: Require Opal installation

**Rejected:** Limits distribution and use cases.

### Alternative 2: Compile to shell script

**Rejected:** Shell scripts are not portable and not secure.

### Alternative 3: No source embedding

**Rejected:** Limits auditability and compliance.

## Implementation

### Phase 1: Basic Compilation
- Embed Opal runtime in binary
- Pre-parse and validate at compile time
- Generate CLI parser from command definitions
- Support `--dry-run` flag

### Phase 2: Advanced Features
- Cross-compilation support
- Resource embedding
- Custom metadata
- Source visibility

### Phase 3: Distribution
- Binary signing
- Checksum generation
- Release automation

### Phase 4: Integration
- Documentation and examples
- Best practices guide

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Binary size:** What's the acceptable binary size limit?
2. **Compression:** Should we compress embedded source?
3. **Obfuscation:** Should we support code obfuscation?
4. **Licensing:** How should we handle licensing for compiled binaries?
5. **Updates:** Should compiled binaries support auto-updates?

## References

- **Go embedding:** Inspiration for embedding resources
- **Docker:** Inspiration for standalone distribution
- **Related OEPs:**
  - OEP-004: Plan Verification (contract verification in binaries)
  - OEP-008: Plan-First Execution Model (plan-first in binaries)
