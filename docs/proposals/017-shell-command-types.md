---
oep: 017
title: Shell Command Type Definitions
status: Draft
type: Enhancement
created: 2025-11-01
updated: 2025-01-27
---

# OEP-017: Shell Command Type Definitions

## Summary

Create a **DefinitelyTyped for shell commands** - type definitions for common shell commands (`grep`, `curl`, `jq`, etc.) that enable plan-time validation, pipeline type checking, platform compatibility warnings, and secret taint tracking.

## Vision

Shell commands are the #1 source of "works on my machine" bugs:

```bash
# Works on Linux, fails on macOS
grep --color=auto "error" logs.txt   # BSD grep uses --colour

# Silent type mismatch  
jq '.items[]' data.json | grep "error"   # jq outputs JSON, grep expects lines

# Platform-specific behavior
sed -i 's/foo/bar/' config.txt   # GNU vs BSD have different -i syntax

# Version-specific flags
curl --json '{"key":"value"}' api.com   # --json added in curl 8.0

# Secret leaked to disk
echo "$API_TOKEN" > /tmp/debug.txt
```

**The solution:** Type definitions that make these problems visible at plan-time, before execution.

## Core Ideas

### Stream Types

Data flowing through pipelines has a type:

| Type | Description | Example |
|------|-------------|---------|
| `Stream<Lines>` | Newline-delimited text | `grep` output |
| `Stream<JSON>` | JSON data | `jq` output |
| `Stream<Text>` | Plain text | `echo` output |
| `Stream<Bytes>` | Raw bytes | `curl` output |

Commands declare input/output types. Pipeline mismatches become plan-time errors:

```opal
jq '.items[]' data.json | grep "error"
# Error: Pipeline type mismatch
#   jq outputs Stream<JSON>, grep expects Stream<Lines>
# Suggestion: Use 'jq -r' for raw line output
```

### Platform & Version Compatibility

Type definitions capture platform and version differences:

- **Platforms**: GNU (Linux) vs BSD (macOS) vs busybox (Alpine)
- **Versions**: `curl 7.x` vs `curl 8.x`, `jq 1.5` vs `jq 1.7`

```opal
grep --color=auto "error" logs.txt
# On macOS:
# Error: Unknown flag '--color' for grep
#   Platform: BSD (detected)
#   Suggestion: Use '--colour' or target GNU

sed -i 's/foo/bar/' config.txt
# Error: Platform-incompatible syntax
#   GNU: sed -i 's/foo/bar/' file
#   BSD: sed -i '' 's/foo/bar/' file

curl --json '{"key":"val"}' api.com
# Error: Flag '--json' requires curl 8.0+
#   Detected: curl 7.76.0
```

### Taint Tracking

Secrets are tracked through pipelines. Warnings when tainted values reach sinks:

```opal
var token = @aws.secret.get("api-token")
echo "Token: @var.token" > debug.txt
# Warning: Secret redirected to file (potential exposure)
```

### Type Packs

Type definitions distributed as versioned packs via `opal.mod` (see OEP-012):

```toml
# opal.mod
[plugins]
hashicorp/aws = "5.0.0"

[shell-types]
posix-core = "1.0"
gnu-coreutils = "3.2"
textproc = "1.0"

[shell]
platform = "gnu"  # or "bsd", "posix", "auto"
```

**Starter packs:**
- `posix-core` - POSIX-compliant commands (portable baseline)
- `gnu-coreutils` - GNU extensions
- `bsd-utils` - BSD/macOS variants  
- `textproc` - grep, sed, awk, jq
- `net` - curl, wget, nc

## The Gateway: `opal lint`

Lint any shell script without migration - the adoption gateway:

```bash
opal lint deploy.sh
opal lint --format=github-actions scripts/*.sh
```

```bash
# deploy.sh
grep --colour=auto "error" logs.txt
# ⚠ Unknown flag '--colour' (did you mean '--color'?)

sed -i 's/foo/bar/' config.txt  
# ⚠ Platform-specific: BSD requires 'sed -i ""'

TOKEN=$(vault read -field=value secret/api)
echo "Token: $TOKEN" > /tmp/token.txt
# ⚠ Potential secret exposure: $TOKEN redirected to file
```

**Adoption path:**
1. `opal lint deploy.sh` - Get value immediately
2. Fix issues, add to CI
3. Optionally migrate to `.opl` for full Opal features

## Open Questions

1. **Definition format**: Opal syntax? Something else? (Not YAML - we're building Opal!)
2. **Community model**: How do we scale maintenance? DefinitelyTyped-style contributions?
3. **Platform detection**: Auto-detect vs explicit configuration?
4. **Version detection**: Query installed tools at plan-time?
5. **Strictness**: Default to warn or error? Configurable per-rule?
6. **Scope**: Which commands in starter packs? Community-driven expansion?

## Related OEPs

- **OEP-012**: Module Composition - defines `opal.mod` and plugin system that type packs integrate with

## References

- **DefinitelyTyped**: TypeScript type definitions (inspiration for community model)
- **Shellcheck**: Shell linter (complementary - focuses on syntax/pitfalls, not types)
- **Nushell**: Structured pipelines (different model, similar goals)
