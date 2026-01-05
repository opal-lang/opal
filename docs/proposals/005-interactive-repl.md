---
oep: 005
title: Interactive REPL
status: Draft
type: Tooling
created: 2025-01-21
updated: 2025-01-27
---

# OEP-005: Interactive REPL

## Summary

Add an interactive REPL (Read-Eval-Print Loop) for Opal. The REPL enables quick experimentation with commands, decorators, and functions without writing script files.

## Motivation

Currently, testing Opal functionality requires creating a script file:

```bash
# Current workflow - requires a file
echo 'echo "hello"' > test.opl
opal test.opl
rm test.opl
```

A REPL removes this friction:

```bash
$ opal
opal> echo "hello"
hello
```

### Use Cases

**Quick command testing:**
```bash
opal> kubectl get pods
NAME                    READY   STATUS    RESTARTS   AGE
app-1234567890-abcde    1/1     Running   0          2d
```

**Exploring decorators:**
```bash
opal> @env.USER
"alice"

opal> @env.HOME
"/home/alice"
```

**Defining and testing functions:**
```bash
opal> fun deploy(env: String) {
...     kubectl apply -f k8s/@var.env/
...   }
Function 'deploy' defined

opal> deploy("staging")
✓ Executed successfully
```

**Plan-first execution:**
```bash
opal> plan deploy("prod")
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/

Execute? [y/N] y
✓ Executed successfully
```

## Proposal

### Modes

**Execute Mode (default)** - Commands run immediately:
```bash
opal> echo hello
hello

opal> kubectl get pods
NAME         READY   STATUS
app-abc123   1/1     Running
```

**Plan Mode** - Review before executing:
```bash
opal> plan kubectl apply -f k8s/prod/
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/

Execute? [y/N] y
✓ Executed successfully
```

**Dry-Run Mode** - Generate plan without executing:
```bash
opal> dry-run kubectl apply -f k8s/prod/
Plan: a3b2c1d4
  1. kubectl apply -f k8s/prod/

(not executed)
```

### Features

**Command history** - Arrow keys navigate previous commands.

**Autocomplete** - Tab completion for commands and decorators:
```bash
opal> kube<TAB>
kubectl

opal> @re<TAB>
@retry
```

**Multi-line input** - Braces trigger multi-line mode:
```bash
opal> fun greet(name: String) {
...     echo "Hello, @var.name!"
...   }
Function 'greet' defined
```

**Variable binding:**
```bash
opal> var ENV = "prod"
opal> kubectl apply -f k8s/@var.ENV/
```

**Decorator integration:**
```bash
opal> @retry(attempts=3) {
...     curl /health
...   }
✓ Executed successfully

opal> @timeout(30s) {
...     long-running-command
...   }
✓ Executed successfully
```

### Built-in Commands

| Command | Description |
|---------|-------------|
| `help` | Show help |
| `exit` | Exit REPL |
| `clear` | Clear screen |
| `history` | Show command history |
| `plan <cmd>` | Generate plan, prompt to execute |
| `dry-run <cmd>` | Generate plan without executing |
| `vars` | Show defined variables |
| `funcs` | Show defined functions |

### Session Behavior

Each REPL session starts fresh. Functions and variables defined in a session are available until you exit:

```bash
$ opal
opal> var X = 1
opal> fun foo() { echo "hi" }
opal> exit

$ opal
opal> echo @var.X
# Error: X not defined
```

This keeps things simple - no hidden state files or persistence to manage.

## Implementation

### Phase 1: Core REPL
- Basic read-eval-print loop
- Command history (readline)
- Multi-line input detection

### Phase 2: Enhancements
- Tab completion
- Syntax highlighting
- Plan/dry-run modes

### Phase 3: Integration
- LSP support for REPL context
- Better error messages with suggestions

## Open Questions

1. **Persistence**: Should we offer optional session save/restore?
2. **Scripting mode**: Should `opal -` read commands from stdin?
3. **Customization**: Custom prompts, colors, key bindings?

## References

- **Python REPL**: Simple, effective interactive mode
- **Node.js REPL**: Good autocomplete and history
- **Elixir IEx**: Advanced introspection features
- **Related**: OEP-011 (System Shell) explores full shell replacement
