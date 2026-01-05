---
oep: 005
title: Interactive REPL
status: Draft
type: Tooling
created: 2025-01-21
updated: 2025-01-21
---

# OEP-005: Interactive REPL

## Summary

Add an interactive REPL (Read-Eval-Print Loop) for Opal, enabling command-line exploration, function definition, and plan-first execution. The REPL provides a familiar shell-like interface with Opal's safety and determinism guarantees.

## Motivation

### The Problem

Current Opal requires writing scripts to test functionality:

```bash
# ❌ Must write script file
cat > deploy.opl << 'EOF'
fun deploy(env: String) {
    kubectl apply -f k8s/@var.env/
}
deploy("staging")
EOF

opal deploy.opl
```

**Problems:**
- Friction for exploration and testing
- Can't easily try different commands
- No interactive feedback
- No history or completion

### Use Cases

**1. Interactive exploration:**
```bash
$ opal
opal> @env.USER
"alice"

opal> @env.HOME
"/home/alice"

opal> echo hello
hello
```

**2. Function definition and testing:**
```bash
opal> fun deploy(env: String) {
...     kubectl apply -f k8s/@var.env/
...   }
Function 'deploy' defined

opal> deploy("staging")
✓ Executed successfully

opal> deploy("prod")
✓ Executed successfully
```

**3. Plan-first execution:**
```bash
opal> plan deploy("prod")
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/
  2. kubectl scale --replicas=3 deployment/app

Execute? [y/N] y
✓ Executed successfully
```

**4. Decorator integration:**
```bash
opal> @retry(attempts=3) {
...     curl /health
...   }
✓ Executed successfully
```

## Proposal

### REPL Modes

#### Execute Mode (Default)

Execute commands immediately:

```bash
$ opal
opal> echo hello
hello

opal> @env.USER
"alice"

opal> kubectl get pods
NAME                    READY   STATUS    RESTARTS   AGE
app-1234567890-abcde    1/1     Running   0          2d
```

#### Plan Mode

Generate and review plans before execution:

```bash
opal> plan kubectl apply -f k8s/prod/
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/

Execute? [y/N] y
✓ Executed successfully
```

#### Dry-Run Mode

Generate plans without executing:

```bash
opal> dry-run kubectl apply -f k8s/prod/
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/

(not executed)
```

### Features

#### Command History

Access previous commands:

```bash
opal> echo hello
hello

opal> ↑  # Previous command
opal> echo hello
hello
```

#### Autocomplete

Tab completion for decorators, functions, and variables:

```bash
opal> @sh<TAB>
@shell

opal> kubectl <TAB>
apply    create   delete   describe   get   logs   scale
```

#### Function Definitions

Define and reuse functions:

```bash
opal> fun deploy(env: String) {
...     kubectl apply -f k8s/@var.env/
...   }
Function 'deploy' defined

opal> deploy("staging")
✓ Executed successfully

opal> deploy("prod")
✓ Executed successfully
```

#### Variable Binding

Bind variables for reuse:

```bash
opal> var ENV = "prod"
Variable 'ENV' bound

opal> kubectl apply -f k8s/@var.ENV/
✓ Executed successfully

opal> var ENV = "staging"
Variable 'ENV' rebound

opal> kubectl apply -f k8s/@var.ENV/
✓ Executed successfully
```

#### Decorator Integration

Use all Opal decorators:

```bash
opal> @retry(attempts=3) {
...     curl /health
...   }
✓ Executed successfully

opal> @timeout(30s) {
...     long-running-command
...   }
✓ Executed successfully

opal> @parallel {
...     curl /api/users
...     curl /api/posts
...   }
✓ Executed successfully
```

#### Multi-Line Input

Support multi-line commands:

```bash
opal> fun deploy(env: String) {
...     kubectl apply -f k8s/@var.env/
...     kubectl rollout status deployment/app
...   }
Function 'deploy' defined
```

### Core Restrictions

#### Restriction 1: REPL is stateless between sessions

State is not persisted between REPL sessions:

```bash
$ opal
opal> fun deploy(env: String) { ... }
opal> exit

$ opal
opal> deploy("prod")
# Error: deploy not defined
```

**Why?** Simplicity. Functions and variables are session-local.

#### Restriction 2: No script imports in REPL

Cannot import external scripts:

```bash
# ❌ FORBIDDEN: imports
opal> import "deploy.opl"
# Error: imports not allowed in REPL

# ✅ CORRECT: define functions in REPL
opal> fun deploy(env: String) { ... }
```

**Why?** REPL is for interactive exploration, not script composition.

#### Restriction 3: Plan mode requires explicit confirmation

Plan mode requires user confirmation before execution:

```bash
opal> plan kubectl apply -f k8s/prod/
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/

Execute? [y/N] y
✓ Executed successfully
```

**Why?** Safety. Prevents accidental execution of dangerous commands.

### REPL Commands

#### Built-in Commands

- `help` - Show help
- `exit` - Exit REPL
- `clear` - Clear screen
- `history` - Show command history
- `plan <command>` - Generate plan without executing
- `dry-run <command>` - Generate plan without executing (alias for plan)
- `set <var> <value>` - Set variable
- `unset <var>` - Unset variable
- `vars` - Show all variables
- `funcs` - Show all functions

#### Example Usage

```bash
opal> help
Opal REPL - Interactive command execution

Commands:
  help              Show this help
  exit              Exit REPL
  clear             Clear screen
  history           Show command history
  plan <cmd>        Generate plan without executing
  dry-run <cmd>    Generate plan without executing
  set <var> <val>   Set variable
  unset <var>       Unset variable
  vars              Show all variables
  funcs             Show all functions

opal> vars
ENV = "prod"
REGION = "us-west-2"

opal> funcs
deploy(env: String)
migrate(version: String)
```

## Rationale

### Why a REPL?

**Exploration:** Easy to try commands without writing scripts.

**Learning:** Familiar interface for new users.

**Debugging:** Quick way to test decorators and functions.

**Scripting:** Can copy-paste commands from REPL into scripts.

### Why stateless?

**Simplicity:** No need to persist state between sessions.

**Clarity:** Each session is independent.

**Safety:** No accidental state leakage between sessions.

### Why plan mode?

**Safety:** Review plans before execution.

**Debugging:** See what will happen before it happens.

**Learning:** Understand how Opal plans work.

## Alternatives Considered

### Alternative 1: Stateful REPL with persistence

**Rejected:** Adds complexity. Stateless is simpler and safer.

### Alternative 2: Full shell replacement

**Rejected:** Out of scope. REPL is for Opal exploration, not shell replacement. (See OEP-011 for System Shell.)

### Alternative 3: No plan mode

**Rejected:** Plan mode is important for safety and learning.

## Implementation

### Phase 1: Basic REPL
- Lexer and parser integration
- Execute mode
- Command history
- Basic autocomplete

### Phase 2: Advanced Features
- Plan mode
- Dry-run mode
- Function definitions
- Variable binding

### Phase 3: Polish
- Better error messages
- Syntax highlighting
- Multi-line input
- Built-in commands

### Phase 4: Integration
- LSP support for REPL
- Documentation and examples

## Compatibility

**Breaking changes:** None. This is a new feature.

**Migration path:** N/A (new feature, no existing code to migrate).

## Open Questions

1. **Persistence:** Should we add optional persistence for functions/variables?
2. **Scripting:** Should REPL support scripting mode (read commands from stdin)?
3. **Debugging:** Should we add debugging commands (breakpoints, step-through)?
4. **Performance:** Should we cache parsed commands for faster execution?
5. **Customization:** Should users be able to customize REPL prompts and colors?

## References

- **Python REPL:** Inspiration for interactive exploration
- **Node.js REPL:** Similar features and interface
- **Elixir IEx:** Advanced REPL with introspection
- **Related OEPs:**
  - OEP-011: System Shell (long-term vision for daily-driver shell)
