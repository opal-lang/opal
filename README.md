# Opal

**Plan-first execution platform for deployments, infrastructure, and operations**

Turn operational workflows into verifiable contracts. See what will execute before it executes.

## How It Works

1. **Reality is truth** - Query the world as it actually is
2. **Plan from reality** - Generate execution plan based on current state
3. **Execute the plan** - Run the operations
4. **Verify the contract** - Catch changes between plan and execution

No state files. No "desired state" to maintain. The plan is your contract.

## Features

- **Contract verified**: Hash-based verification ensures reviewed plans match execution
- **Stateless**: No state files to corrupt—query reality fresh each run
- **Unified**: Deployments, infrastructure, and operations in one tool
- **Secure**: Secrets never logged or exposed

## Language Model

Opal is intentionally a two-lane system:

- **Shell lane (runtime work):** Commands, pipes, redirects, and operator chains execute with shell semantics.
- **Metaprogramming lane (plan-time structure):** `fun`, `if`, `for`, `when`, and decorators shape and validate what will execute.

This is a departure from plain shell scripting by design. Opal keeps shell as the execution substrate, then adds typed plan-time contracts and deterministic expansion on top.

## North Star

Opal is one operations language: start with commands, scale to full contract-driven workflows.

The thesis is a single language for infra changes, run scripts, and day-2 operations without split-brain tooling.

Design check for new features:

Does this make operations more describable in one language without losing practical escape hatches?

## Quick Start

Define your tasks:

```bash
# commands.opl
fun build = npm run build
fun test = npm test
fun deploy = kubectl apply -f k8s/
```

Run them:

```bash
opal deploy
```

## Execution Modes

1. **Direct execution**: `opal hello` - parse, plan, execute
2. **Quick plan**: `opal hello --dry-run` - show tree without executing
3. **Contract generation**: `opal hello --dry-run --resolve > hello.contract`
4. **Contract execution**: `opal --plan hello.contract` - verify and execute

## Current Scope

**Developer tasks**: Repeatable build/test/deploy workflows  
**Operations tasks**: Day-2 activities like deployments, migrations, restarts, health checks

## Basic Syntax

```opal
# Simple commands
fun build = npm run build
fun test = npm test

# Typed metaprogramming contracts for reusable tasks
fun deploy(env String, replicas Int = 3) {
    kubectl apply -f k8s/@var.env/
    kubectl scale deployment/app --replicas=@var.replicas
}

# Shell commands with operators
fun rollout = {
    kubectl apply -f k8s/ && kubectl rollout status deployment/app
}

# Multiple steps (newline-separated)
fun migrate = {
    psql $DATABASE_URL -f migrations/001-users.sql
    psql $DATABASE_URL -f migrations/002-indexes.sql
}
```

## Planned Features

**Value decorators** (inject values inline):
- `@env.PORT` - Environment variables
- `@var.REPLICAS` - Script variables  
- `@aws.secret.api_key` - External value lookups

**Execution decorators** (enhance command execution):
- `@retry(attempts=3) { ... }` - Retry failed operations
- `@timeout(duration=5m) { ... }` - Timeout protection
- `@parallel { ... }` - Concurrent execution

## Installation

### With Go
```bash
go install github.com/opal-lang/opal/cli@latest
```

### With Nix
```bash
# Direct run
nix run github:aledsdavies/opal -- deploy --dry-run

# Add to flake
{
  inputs.opal.url = "github:aledsdavies/opal";
  
  outputs = { nixpkgs, opal, ... }: {
    devShells.default = nixpkgs.mkShell {
      buildInputs = [ opal.packages.x86_64-linux.default ];
    };
  };
}
```

## Examples

### Web Application Deployment
```opal
fun deploy = {
    kubectl apply -f k8s/
    kubectl set image deployment/app app=$VERSION
    kubectl rollout status deployment/app
}
```

### Database Migration
```opal
fun migrate = {
    echo "Starting migration..."
    psql $DATABASE_URL -f migrations/001-users.sql
    psql $DATABASE_URL -f migrations/002-indexes.sql
    echo "Migration complete"
}
```

## Development

This project uses Nix for development environments:

```bash
# Enter development environment
nix develop

# Build and test
cd cli && go build -o opal .
cd runtime && go test ./...
```

## Status

**Early Development**: Focused on language design and parser implementation.

**Completed**:
- Language specification and syntax design
- High-performance lexer (>5000 lines/ms)
- Planning and contract execution model design
- Multi-module Go architecture

**In Progress**:
- Event-based parser implementation
- Execution engine with decorator support
- Plan generation and contract verification

**Planned**:
- Complete execution decorators (`@retry`, `@timeout`, `@parallel`)
- Value decorators (`@env`, `@var`, `@aws.secret`)
- Plugin system for custom decorators

## Pre-Alpha Compatibility Policy

Opal is pre-alpha. Breaking changes are expected. When syntax or behavior changes:

- Old patterns are removed directly, not deprecated
- No migration tools or compatibility layers unless explicitly requested
- Tests and docs update immediately to the current canonical form

This keeps the codebase clean during rapid iteration. Once Opal reaches alpha, a formal deprecation policy will apply.

## How It Works

Opal treats operations as plans that can be reviewed before execution:

1. **Plan** your operation and see exactly what will execute
2. **Review** the plan (or save it for later)
3. **Execute** with contract verification to catch environment changes

This gives you confidence to run complex workflows safely.

## Contributing

See documentation in `docs/`:
- [SPECIFICATION.md](docs/SPECIFICATION.md) - Language specification and user guide
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) - System design and implementation
- [TESTING_STRATEGY.md](docs/TESTING_STRATEGY.md) - Testing approach and standards

## Research & Roadmap

See [FUTURE_IDEAS.md](docs/FUTURE_IDEAS.md) for experimental features and potential extensions.

## License

Apache License, Version 2.0
