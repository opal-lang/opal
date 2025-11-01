---
oep: 017
title: Shell Command Type Definitions
status: Draft
type: Enhancement
created: 2025-11-01
---

# OEP-017: Shell Command Type Definitions

## Summary

Make common shell commands (`echo`, `grep`, `sed`, `curl`, `jq`, etc.) **type-safe** inside Opal: validate flags/positionals at plan-time, model pipeline compatibility with stream types, enforce capability requirements, and track secret taint through operators.

## Motivation

### The Problem

Shell commands are the #1 source of "works on my machine" bugs:

1. **Flag incompatibility**: GNU vs BSD variants have different flags
2. **Pipeline type mismatches**: `jq` outputs JSON, `grep` expects lines
3. **Secret leakage**: Secrets in `echo` → file redirect persist to disk
4. **Runtime failures**: Unknown flags discovered during execution
5. **Platform assumptions**: Code works on Linux, fails on macOS

### Current State

```opal
# All of these fail at runtime (too late!)
grep --colour=auto "error" logs.txt     # BSD doesn't have --colour
jq '.items[]' data.json | grep "error"  # Type mismatch (JSON → Lines)
echo "@var.secret" > token.txt          # Secret persisted to disk
curl --max-time 30s api.example.com    # Invalid duration format
```

### Desired State

```opal
# Plan-time errors (before execution)
grep --colour=auto "error" logs.txt
# Error: Unknown flag '--colour' for grep (posix-core@1.0.0)
# Suggestion: Use '--color' (gnu-coreutils@3.2.0)

jq '.items[]' data.json | grep "error"
# Error: Pipeline type mismatch
#   jq outputs Stream<JSON>, grep requires Stream<Lines>
# Suggestion: Use 'jq -r' to output raw lines

echo "@var.secret" > token.txt
# Warning: Redirecting secret-tainted value to file 'token.txt'
# Note: File redirects persist output (potential secret exposure)

curl --max-time 30s api.example.com
# Error: Invalid duration format for --max-time
# Expected: integer (seconds), got: "30s"
# Suggestion: Use --max-time 30
```

## Proposal

### Core Concepts

**1. Stream Types** - Typed data flows through pipelines:

```
Stream<Text>    - Plain text (any encoding)
Stream<Lines>   - Newline-delimited text
Stream<JSON>    - JSON objects (one per line or single object)
Stream<NDJSON>  - Newline-delimited JSON (jq default)
Stream<Bytes>   - Raw bytes
Stream<CSV>     - CSV format
Stream<YAML>    - YAML documents
```

**2. Command Signatures** - Input/output types per command:

```
grep:  Stream<Lines> → Stream<Lines>
jq:    Stream<JSON>  → Stream<JSON> | Stream<Lines> (with -r)
sed:   Stream<Lines> → Stream<Lines>
curl:  Stream<Bytes> → Stream<Bytes>
```

**3. Capabilities** - Commands declare required capabilities (same as decorators):

```
curl:  [net]
grep:  [fs.read]  (if files provided)
echo:  []
```

**4. Taint Tracking** - Secrets tracked through pipelines, warnings on sinks:

```
Commands propagate taint (echo, grep, sed)
Operators create sinks (>, >>, 2>)
Opal scrubs stdout (safe by default)
```

### Type Pack Format

Type packs use JSON Schema (same as plugin manifests):

```yaml
# ~/.opal/types/posix-core@1.0.0/commands/grep.yaml
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://types.opal.dev/posix-core/v1.0.0/grep

command: grep
summary: Search for patterns in files or stdin

# Capabilities (same as decorators)
x-opal-capabilities: [fs.read]

# Signature
signature:
  stdin:
    type: Stream<Lines>
    required_if: { files.length: 0 }
  stdout:
    type: Stream<Lines>
    description: Matching lines from input
  stderr:
    type: Stream<Text>
  exit:
    "0": "one or more matches found"
    "1": "no matches found"
    "2": "error occurred"

# Taint propagation
x-opal-taint:
  propagates_stdin: true   # Tainted input → tainted output
  propagates_args: false   # Pattern doesn't taint output
  sinks: []                # stdout is not a sink (Opal controls it)
  filters: true            # grep filters, doesn't transform

# Arguments (JSON Schema)
args:
  type: object
  properties:
    extended_regexp:
      x-opal-flags: ["-E", "--extended-regexp"]
      type: boolean
      conflicts: ["fixed_strings"]
    fixed_strings:
      x-opal-flags: ["-F", "--fixed-strings"]
      type: boolean
      conflicts: ["extended_regexp"]
    ignore_case:
      x-opal-flags: ["-i", "--ignore-case"]
      type: boolean
    invert_match:
      x-opal-flags: ["-v", "--invert-match"]
      type: boolean
    pattern:
      x-opal-positional: 0
      type: string
      description: Pattern to search for
    files:
      x-opal-positional: 1
      x-opal-greedy: true
      type: array
      items:
        type: string
        x-opal-format: filepath-readable
  required: [pattern]
  additionalProperties: false
```

### Type Pack Distribution

Type packs are dependencies in `opal.mod`:

```toml
# opal.mod
[dependencies]
# Plugins
hashicorp/aws = { version = "5.0.0", type = "plugin" }

# Shell type packs
posix-core = { version = "1.0.0", type = "shell-types" }
gnu-coreutils = { version = "3.2.0", type = "shell-types" }
textproc = { version = "1.0.0", type = "shell-types" }  # grep, sed, awk, jq
```

**Installation:**

```bash
opal install  # Installs both plugins and type packs

# Type packs cached in
~/.opal/types/
  posix-core@1.0.0/
    manifest.yaml
    commands/
      echo.yaml
      grep.yaml
      sed.yaml
      ...
```

**Lock file:**

```toml
# opal.lock
[[dependency]]
name = "posix-core"
version = "1.0.0"
type = "shell-types"
source = "registry+https://registry.opal.dev/types/posix-core"
checksum = "sha256:a1b2c3d4..."

[[dependency]]
name = "gnu-coreutils"
version = "3.2.0"
type = "shell-types"
source = "registry+https://registry.opal.dev/types/gnu-coreutils"
checksum = "sha256:e5f6g7h8..."
```

### Operator Definitions (Built-in)

Operators are built into Opal and define sinks:

```yaml
# Built into Opal runtime (not in type packs)
operators:
  - operator: "|"
    type: pipeline
    connects: [stdout, stdin]
    taint: propagates
    typing:
      requires: stdout_type(left) <: stdin_type(right)
      coercions:
        - from: Stream<Text>
          to: Stream<Lines>
          rule: split_on_newlines
    
  - operator: ">"
    type: redirect
    connects: [stdout, file]
    taint: sink  # ⚠️ SINK
    warning: "Redirecting to file persists output"
    typing:
      requires: stdout_type(left) in [Stream<Text>, Stream<Bytes>]
    
  - operator: ">>"
    type: redirect_append
    connects: [stdout, file]
    taint: sink  # ⚠️ SINK
    warning: "Appending to file persists output"
    
  - operator: "2>"
    type: redirect_stderr
    connects: [stderr, file]
    taint: sink  # ⚠️ SINK
    
  - operator: "&&"
    type: control_flow
    taint: none
    
  - operator: "||"
    type: control_flow
    taint: none
```

### Plan-Time Validation

**1. Command Resolution:**

```
1. Parse shell command: grep --color=auto "error" logs.txt
2. Load type packs from opal.lock (deterministic)
3. Resolve 'grep' definition by pack precedence
4. Validate flags: --color=auto (unknown in posix-core@1.0.0)
5. Error: Unknown flag (suggest gnu-coreutils@3.2.0)
```

**2. Pipeline Type Checking:**

```
1. Parse pipeline: jq '.items[]' | grep "error"
2. Resolve signatures:
   - jq: Stream<JSON> → Stream<JSON>
   - grep: Stream<Lines> → Stream<Lines>
3. Check compatibility: Stream<JSON> <: Stream<Lines>? NO
4. Error: Type mismatch (suggest jq -r for Stream<Lines>)
```

**3. Capability Enforcement:**

```
1. Parse command: curl https://api.example.com
2. Resolve capabilities: curl requires [net]
3. Check context: @readonly allows [net]? YES
4. Allow execution
```

**4. Taint Tracking:**

```
1. Parse: echo "@var.secret" > token.txt
2. Resolve taint:
   - @var.secret is tainted (from @aws.secret.*)
   - echo propagates arg taint → stdout tainted
   - > is a sink (file redirect)
3. Warning: Secret-tainted value redirected to file
```

### Taint Policy Configuration

```toml
# opal.toml
[shell.taint]
# Policy levels: "strict", "warn", "permissive"
policy = "warn"

# Specific sinks
[shell.taint.sinks]
file_redirect = "warn"      # > and >>
stderr_redirect = "warn"    # 2>
url_params = "error"        # Secrets in URLs always error
environment = "warn"        # export SECRET=...

# Allow list (override policy)
[shell.taint.allow]
commands = ["vault", "pass"]  # Secret managers OK
```

### Error Messages

**Unknown flag:**

```
Error: Unknown flag '--colour' for command 'grep'
  --> deploy.opl:12:5
   |
12 |     grep --colour=auto "error" logs.txt
   |          ^^^^^^^^^^^^^ unknown flag
   |
   = Note: Available in type pack 'posix-core@1.0.0'
   = Suggestion: Did you mean '--color' (gnu-coreutils@3.2.0)?
   = Help: Add 'gnu-coreutils = "3.2.0"' to opal.mod
```

**Pipeline type mismatch:**

```
Error: Pipeline type mismatch
  --> deploy.opl:15:5
   |
15 |     jq '.items[]' data.json | grep "error"
   |     ----------------------- | ^^^^^^^^^^^^
   |     Stream<JSON>              requires Stream<Lines>
   |
   = Suggestion: Use 'jq -r' to output raw text (Stream<Lines>)
   = Example: jq -r '.items[]' data.json | grep "error"
```

**Secret taint warning:**

```
Warning: Redirecting secret-tainted value to file
  --> deploy.opl:20:5
   |
20 |     echo "Token: @var.apiToken" > /tmp/token.txt
   |                  ^^^^^^^^^^^^^ | ^^^^^^^^^^^^^^^
   |                  secret         file sink
   |
   = Note: File redirects persist output (potential secret exposure)
   = Suggestion: Secrets should not be persisted to disk
   = Help: Use secret-aware commands or Opal's secret management
```

**Capability violation:**

```
Error: Command requires capability not allowed in context
  --> deploy.opl:25:5
   |
25 |     @offline {
26 |         curl https://api.example.com
   |         ^^^^ requires capability 'net'
   |     }
   |
   = Note: @offline forbids network access
   = Suggestion: Remove @offline or use cached data
```

## Integration with Existing Type System

### Stream Types as Formats

Stream types integrate with existing format registry:

```go
// core/types/format.go

type StreamType string

const (
    StreamText   StreamType = "Stream<Text>"
    StreamLines  StreamType = "Stream<Lines>"
    StreamJSON   StreamType = "Stream<JSON>"
    StreamNDJSON StreamType = "Stream<NDJSON>"
    StreamBytes  StreamType = "Stream<Bytes>"
    StreamCSV    StreamType = "Stream<CSV>"
    StreamYAML   StreamType = "Stream<YAML>"
)

// Coercion rules (plan-time)
var StreamCoercions = map[StreamType][]StreamType{
    StreamText:  {StreamLines},           // Split on newlines
    StreamLines: {StreamText},            // Join with newlines
    StreamJSON:  {StreamText},            // Serialize
    StreamBytes: {StreamText},            // Decode (warn on encoding)
}
```

### Command Capabilities

Commands use same capability system as decorators:

```go
// core/types/capability.go

type Capability string

const (
    CapNet       Capability = "net"
    CapFSRead    Capability = "fs.read"
    CapFSWrite   Capability = "fs.write"
    CapClock     Capability = "clock"
    CapEnv       Capability = "env"
    CapExec      Capability = "exec"
    CapSecret    Capability = "secret"
    CapParallel  Capability = "parallel"
)

// Shell commands declare capabilities
type ShellCommandDef struct {
    Command      string
    Capabilities []Capability
    Signature    CommandSignature
    Taint        TaintPolicy
    Args         ParamSchema
}
```

## Type Pack Resolution

### Precedence Rules

Type packs are resolved in deterministic order:

**1. Project-local overrides** (highest precedence):
```
.opal/commands/my-tool.yaml
```
- Custom command definitions
- Namespaced to prevent conflicts
- `$id` + hash recorded in plan

**2. Explicit dependencies** (from `opal.mod`):
```toml
[dependencies]
gnu-coreutils = { version = "3.2.0", type = "shell-types" }
```
- Version-locked in `opal.lock`
- Deterministic resolution

**3. Platform-specific packs** (auto-selected):
```toml
[shell]
platform = "linux"  # or "bsd", "posix", "auto"
```
- `auto`: Detect from `uname`
- Explicit: Override for cross-platform scripts

**4. Base packs** (lowest precedence):
```
posix-core@1.0.0 (always loaded)
```

### Resolution Algorithm

```go
func ResolveCommand(cmd string, packs []TypePack) (*CommandDef, error) {
    // 1. Check project-local overrides
    if def := projectLocal.Lookup(cmd); def != nil {
        return def, nil
    }
    
    // 2. Check explicit dependencies (order in opal.mod)
    for _, pack := range packs {
        if def := pack.Lookup(cmd); def != nil {
            return def, nil
        }
    }
    
    // 3. Check platform-specific packs
    if def := platformPacks.Lookup(cmd); def != nil {
        return def, nil
    }
    
    // 4. Check base packs (posix-core)
    if def := basePacks.Lookup(cmd); def != nil {
        return def, nil
    }
    
    // 5. Unknown command fallback
    if config.StrictShellTypes {
        return nil, fmt.Errorf("unknown command: %s", cmd)
    }
    
    // Fallback: Stream<Bytes> → Stream<Bytes>
    log.Warn("Unknown command %s, using fallback signature", cmd)
    return &CommandDef{
        Command: cmd,
        Signature: CommandSignature{
            Stdin:  StreamBytes,
            Stdout: StreamBytes,
            Stderr: StreamText,
        },
    }, nil
}
```

### Platform Selection

**Auto-detection:**
```go
func DetectPlatform() string {
    switch runtime.GOOS {
    case "linux":
        // Check for GNU vs musl
        if hasGNUCoreutils() {
            return "gnu"
        }
        return "posix"
    case "darwin":
        return "bsd"
    case "freebsd", "openbsd", "netbsd":
        return "bsd"
    default:
        return "posix"
    }
}
```

**Explicit override:**
```toml
# opal.toml
[shell]
platform = "gnu"  # Force GNU semantics
```

## Stream Type Coercions

### Allowed Coercions (Exhaustive List)

**Implicit (no warning):**
```
Text  → Lines   (split on newlines)
Lines → Text    (join with newlines)
```

**Explicit (warn in strict mode):**
```
JSON  → Text    (serialize, warn: "Use jq -r for raw output")
Bytes → Text    (decode UTF-8, warn: "Assuming UTF-8 encoding")
```

**Forbidden (always error):**
```
Text  → JSON    (ambiguous, error: "Use jq to parse JSON")
Lines → JSON    (ambiguous, error: "Use jq to parse JSON")
JSON  → Lines   (error: "Use jq -r to output lines")
Bytes → JSON    (error: "Decode to Text first, then parse")
```

### Coercion Policy

```toml
# opal.toml
[shell.coercions]
# Policy: "strict", "warn", "permissive"
policy = "warn"

# Specific coercions
[shell.coercions.rules]
json_to_text = "warn"    # Warn on JSON → Text
bytes_to_text = "warn"   # Warn on Bytes → Text (encoding assumption)
```

**Strict mode:**
```bash
opal run --strict-shell-types deploy.opl
# All coercions become errors (except Text ↔ Lines)
```

## Exit Code Typing

### Exit Code Specifications

Commands declare exit code meanings:

```yaml
# grep.yaml
signature:
  exit:
    "0": "one or more matches found"
    "1": "no matches found"
    "2": "error occurred"
```

### Control Flow Operators

**`&&` (AND):**
- Executes right if left exits 0
- Short-circuits on non-zero
- Type: No stream transformation

**`||` (OR):**
- Executes right if left exits non-zero
- Short-circuits on zero
- Type: No stream transformation

**Extended exit codes:**
```yaml
# curl.yaml
signature:
  exit:
    "0": "success"
    "6": "couldn't resolve host"
    "7": "failed to connect"
    "28": "operation timeout"
    "35": "SSL connect error"
    # ... (curl has 90+ exit codes)
```

**Planner behavior:**
```opal
# Plan-time: no validation of exit codes in && / ||
curl https://api.example.com && echo "Success"

# Runtime: exit code determines flow
# curl exits 7 → echo not executed
```

**Future extension (OEP-018?):**
```opal
# Typed exit code matching
curl https://api.example.com
when exit {
    0 -> echo "Success"
    6 -> echo "DNS resolution failed"
    7 -> echo "Connection failed"
    _ -> echo "Unknown error"
}
```

## Secret Taint Sources

### Taint Origins

**1. Decorator returns with `secret` capability:**
```yaml
# DIM manifest
x-opal-decorators:
  - name: aws.secret.get
    returns:
      type: string
      x-opal-taint: secret  # ← Marks return value as tainted
```

**2. Environment variables (policy-based):**
```toml
# opal.toml
[shell.taint.sources]
# Regex patterns for env var names
env_patterns = [
    ".*TOKEN.*",
    ".*SECRET.*",
    ".*PASSWORD.*",
    ".*API_KEY.*",
]
```

**3. File reads (policy-based):**
```toml
[shell.taint.sources]
# Paths that contain secrets
file_patterns = [
    "/etc/secrets/**",
    "~/.ssh/id_*",
    "**/vault/**",
]
```

**4. Command outputs (explicit):**
```yaml
# vault.yaml (custom command def)
command: vault
signature:
  stdout:
    type: Stream<Text>
    x-opal-taint: secret  # ← Output is tainted
```

**5. Explicit taint annotation:**
```opal
# Future: explicit taint marking
var token = @taint.secret("my-secret-value")
```

### Taint Propagation

```go
type TaintPolicy struct {
    PropagatesStdin bool   // Taint flows stdin → stdout
    PropagatesArgs  bool   // Taint flows args → stdout
    Sinks           []Sink // Where taint is exposed
    SecretAware     bool   // Command handles secrets safely
    Filters         bool   // Command filters (grep, sed)
}

type Sink string

const (
    SinkStdout      Sink = "stdout"       // Visible output
    SinkFile        Sink = "file"         // Persistent storage
    SinkNetwork     Sink = "network"      // Network transmission
    SinkEnvironment Sink = "environment"  // Env vars
    SinkLog         Sink = "log"          // Logging systems
)
```

## Type Pack Versioning & Plan Contracts

### Semver Rules (Mirror OEP-012)

**Patch (1.0.x):**
- Documentation updates
- Error message improvements
- Examples
- **No schema changes**

**Minor (1.x.0):**
- Add new flags (optional)
- Add new commands
- Add new exit codes
- Add new coercions
- **Additive only**

**Major (x.0.0):**
- Remove flags
- Change flag semantics
- Remove commands
- Change exit code meanings
- Change stream types
- **Breaking changes**

### Plan Contract Enforcement

**Plan includes:**
```json
{
  "commands": [
    {
      "command": "grep",
      "pack": "gnu-coreutils",
      "version": "3.2.0",
      "$id": "https://types.opal.dev/gnu-coreutils/v3.2.0/grep",
      "hash": "sha256:abc123..."
    }
  ]
}
```

**Executor validation:**
```go
func ValidateCommandSchema(plan, actual CommandDef) error {
    planVer := semver.Parse(plan.Version)
    actualVer := semver.Parse(actual.Version)
    
    // Major version must match
    if planVer.Major != actualVer.Major {
        return fmt.Errorf(
            "command %s: major version mismatch (plan: v%d, actual: v%d)",
            plan.Command, planVer.Major, actualVer.Major,
        )
    }
    
    // Minor version can increase (additive)
    if actualVer.Minor < planVer.Minor {
        return fmt.Errorf(
            "command %s: minor version downgrade (plan: v%d.%d, actual: v%d.%d)",
            plan.Command, planVer.Major, planVer.Minor,
            actualVer.Major, actualVer.Minor,
        )
    }
    
    // Hash must match for same version
    if planVer.Equal(actualVer) && plan.Hash != actual.Hash {
        return fmt.Errorf(
            "command %s: schema hash mismatch for v%s (content changed without version bump)",
            plan.Command, plan.Version,
        )
    }
    
    return nil
}
```

## Local Command Overrides

### Project-Local Definitions

**Location:**
```
.opal/commands/
  my-tool.yaml
  internal-cli.yaml
```

**Format:**
```yaml
# .opal/commands/my-tool.yaml
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://company.internal/opal/my-tool/v1.0.0

command: my-tool
summary: Internal deployment tool

x-opal-capabilities: [net, fs.write]

signature:
  stdin: Stream<JSON>
  stdout: Stream<JSON>
  stderr: Stream<Text>
  exit:
    "0": "success"
    "1": "validation failed"
    "2": "deployment failed"

x-opal-taint:
  propagates_stdin: true
  propagates_args: true
  sinks: []
  secret_aware: true

args:
  type: object
  properties:
    environment:
      type: string
      enum: ["dev", "staging", "prod"]
    version:
      type: string
      x-opal-format: semver
  required: [environment, version]
```

**Precedence:**
- Project-local > Registry packs
- Namespaced to prevent conflicts
- `$id` + hash recorded in plan
- Versioned independently

**Plan contract:**
```json
{
  "commands": [
    {
      "command": "my-tool",
      "source": "project-local",
      "$id": "https://company.internal/opal/my-tool/v1.0.0",
      "hash": "sha256:def456..."
    }
  ]
}
```

## Initial Type Packs

**posix-core@1.0.0:**
- `echo`, `cat`, `head`, `tail`, `tr`, `sort`, `uniq`, `cut`, `xargs`, `tee`
- POSIX-compliant flags only
- Platform: all

**gnu-coreutils@3.2.0:**
- Extends posix-core with GNU-specific flags
- `--color`, `--help`, `--version`, etc.
- Platform: Linux

**bsd-utils@1.0.0:**
- BSD-specific variants
- Different flag syntax (e.g., `sed -i ''` vs `sed -i`)
- Platform: macOS, FreeBSD

**textproc@1.0.0:**
- `grep`, `sed`, `awk`, `jq`
- Stream type transformations
- Platform: all

**net@1.0.0:**
- `curl`, `wget`, `nc`, `telnet`
- Network capabilities
- Platform: all

## Evolution & Compatibility

**Pack Semver:**
- **Patch (1.0.x)**: Documentation, examples, error messages
- **Minor (1.x.0)**: Add flags, add commands, add coercions
- **Major (x.0.0)**: Remove flags, change signatures, change exit codes

**Per-command `$id` + hash:**
- Included in plan → contract verification detects definition drift
- Executor validates against embedded schemas

**Fallback:**
- Unknown commands default to `Stream<Bytes> → Stream<Bytes>` with warning
- Configurable to error in `--strict-shell-types`

## Implementation Plan

**Phase 1: Stream Types (Week 1)**
- Add stream type system
- Add coercion rules
- Add pipeline type checker

**Phase 2: Type Pack Loader (Week 2)**
- Add type pack format (JSON Schema)
- Add pack loader (from opal.lock)
- Add command resolver (precedence rules)

**Phase 3: Planner Integration (Week 3)**
- Integrate command validation into planner
- Add capability checking
- Add error formatting

**Phase 4: Taint Tracking (Week 4)**
- Add taint propagation
- Add operator sink definitions
- Add warning/error policies

**Phase 5: Starter Packs (Week 5)**
- Create posix-core@1.0.0
- Create gnu-coreutils@3.2.0
- Create textproc@1.0.0

## Benefits

1. **Plan-time validation** - Catch errors before execution
2. **Platform portability** - Explicit GNU vs BSD differences
3. **Type safety** - Pipeline type mismatches caught early
4. **Secret safety** - Taint tracking prevents leaks
5. **Capability enforcement** - Commands respect decorator policies
6. **Determinism** - Type packs locked in opal.lock
7. **Extensibility** - Third-party type packs for custom tools

## Shell Script Linting (Non-Opal Files)

Opal can validate **any shell script**, not just Opal files. This provides immediate value without requiring migration.

### Usage

**Basic linting:**
```bash
opal lint deploy.sh
opal lint scripts/*.sh
opal lint --recursive ./scripts/
```

**CI/CD integration:**
```bash
# GitHub Actions format
opal lint --format=github-actions deploy.sh

# GitLab format
opal lint --format=gitlab deploy.sh

# JSON output
opal lint --format=json deploy.sh
```

**Auto-fix mode:**
```bash
# Fix common issues
opal lint --fix deploy.sh

# Preview fixes without applying
opal lint --fix --dry-run deploy.sh
```

### What Gets Validated

**1. Command flags (platform-specific):**
```bash
# deploy.sh
grep --colour=auto "error" logs.txt

# Lint output
deploy.sh:5:6: Unknown flag '--colour' for grep (posix-core@1.0.0)
  Suggestion: Use '--color' (gnu-coreutils@3.2.0)
  Platform: This script assumes GNU coreutils
```

**2. Pipeline type mismatches:**
```bash
# deploy.sh
jq '.items[]' data.json | grep "error"

# Lint output
deploy.sh:10:1: Pipeline type mismatch
  jq outputs Stream<JSON>, grep requires Stream<Lines>
  Suggestion: Use 'jq -r .items[]' to output raw lines
```

**3. Secret exposure:**
```bash
# deploy.sh
TOKEN=$(vault read -field=value secret/api)
echo "Token: $TOKEN" > /tmp/token.txt

# Lint output
deploy.sh:12:1: Potential secret exposure
  Variable 'TOKEN' appears to contain secret (from vault)
  Redirecting to file persists output
  Suggestion: Avoid persisting secrets to disk
```

**4. Platform assumptions:**
```bash
# deploy.sh
sed -i 's/foo/bar/' config.txt

# Lint output
deploy.sh:15:1: Platform-specific flag
  'sed -i' requires extension argument on BSD/macOS
  GNU: sed -i 's/foo/bar/' config.txt
  BSD: sed -i '' 's/foo/bar/' config.txt
  Suggestion: Use 'sed -i.bak' for portability
```

### Configuration

**Per-script configuration (shebang comment):**
```bash
#!/usr/bin/env bash
# opal-lint: type-packs=gnu-coreutils@3.2.0,textproc@1.0.0
# opal-lint: platform=linux
# opal-lint: strict-types=true

grep --color=auto "error" logs.txt
```

**Project configuration (.opal-lint.toml):**
```toml
# .opal-lint.toml (in project root)
[lint]
type_packs = [
  "posix-core@1.0.0",
  "gnu-coreutils@3.2.0",
  "textproc@1.0.0",
]
platform = "linux"  # or "bsd", "posix"
strict_types = true

[lint.rules]
unknown_flags = "error"
pipeline_types = "error"
secret_exposure = "warn"
platform_specific = "warn"

[lint.ignore]
# Ignore specific rules
rules = ["secret_exposure"]
files = ["scripts/legacy/*.sh"]
```

### CI/CD Integration

**GitHub Actions:**
```yaml
name: Lint Shell Scripts
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: opal-lang/setup-opal@v1
      - run: opal lint --format=github-actions scripts/*.sh
```

**Pre-commit hook:**
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/opal-lang/opal
    rev: v1.0.0
    hooks:
      - id: opal-lint
        files: \.(sh|bash)$
```

### Comparison with Shellcheck

**Shellcheck (syntax + common mistakes):**
- Unquoted variables
- Unused variables
- Syntax errors
- Common pitfalls

**Opal Lint (types + portability):**
- Flag compatibility (GNU vs BSD)
- Pipeline type safety
- Secret exposure
- Platform assumptions

**Use both for comprehensive validation:**
```bash
shellcheck deploy.sh && opal lint deploy.sh
```

### Migration Path

**1. Start with linting (no migration):**
```bash
opal lint deploy.sh
```

**2. Fix issues:**
```bash
opal lint --fix deploy.sh
```

**3. Add to CI (prevent regressions):**
```yaml
- run: opal lint scripts/*.sh
```

**4. Gradual migration (optional):**
```bash
opal convert deploy.sh > deploy.opl
```

### Benefits

**For non-Opal users:**
- ✅ Validate shell scripts without migration
- ✅ Catch platform-specific issues
- ✅ Prevent secret leaks
- ✅ CI/CD integration

**For Opal adoption:**
- ✅ Gateway to Opal (discover through linting)
- ✅ See value before committing
- ✅ Natural upgrade path

## Future Extensions

**1. Custom command definitions:**

```yaml
# .opal/commands/my-tool.yaml
command: my-tool
signature:
  stdin: Stream<JSON>
  stdout: Stream<JSON>
args:
  format:
    x-opal-flags: ["--format"]
    type: string
    enum: ["json", "yaml"]
```

**2. Shell function types:**

```opal
fun deploy_service(name: String, version: Semver) {
    # Function signature inferred from usage
    kubectl set image deployment/@var.name app=@var.name:@var.version
}
```

**3. Macro expansion:**

```opal
# Define typed macro
macro retry_curl(url: Url, attempts: Int = 3) {
    @retry(attempts=@var.attempts) {
        curl @var.url
    }
}

# Usage (type-checked)
retry_curl("https://api.example.com", 5)
```

## Open Questions

1. **Platform detection**: Auto-detect vs explicit in opal.toml?
2. **Type pack precedence**: First-match or most-specific?
3. **Coercion warnings**: Always warn or only in strict mode?
4. **Custom commands**: Allow project-local definitions?

## Related Work

- **TypeScript**: Type definitions for JavaScript libraries
- **Shellcheck**: Linting for shell scripts (runtime-focused)
- **Nushell**: Structured data pipelines (different model)
- **PowerShell**: Typed cmdlets (similar concept)

---

**Status**: Draft  
**Next Steps**: Review design, create starter type pack, implement Phase 1
