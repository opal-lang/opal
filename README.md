# Opal

**Deterministic task runner for operations and developer workflows**

## The Problem

After infrastructure is provisioned (Terraform, CloudFormation, etc.), teams fall back on shell scripts, Makefiles, or ad-hoc pipelines for day-2 operations and developer tasks. These are brittle, non-deterministic, and hard to audit.

Opal fills the gap between "infrastructure is up" and "services are reliably operated."

## Philosophy

**Outcome-focused execution:**

Opal doesn't maintain state files or enforce a rigid model. Instead:

1. **Reality is truth** - Query the world as it actually is
2. **Plan from reality** - Based on what exists now, here's what we'll do
3. **Execute the plan** - Accomplish the outcomes
4. **Verify the contract** - If reality changed between plan and execute, catch it

The **plan is your contract** - it shows what will happen before it happens. Not bureaucracy, just clarity.

No state files. No "desired state" to maintain. Just: see the world, make a plan, execute it.

## What Opal Does

- **Enforces determinism**: Same inputs always produce the same plan
- **Produces execution contracts**: Verifiable plans that can be reviewed before running
- **Keeps secrets safe**: Never logs or exposes credentials
- **Fails fast**: Catches errors during planning, not execution

## Quick Start

Define your tasks:

```bash
# commands.opl
build: npm run build
test: npm test
deploy: kubectl apply -f k8s/
```

Run with planning:

```bash
# See what will execute
opal deploy --dry-run

# Run the operation
opal deploy
```

## Current Scope

**Developer tasks**: Repeatable build/test/deploy workflows
**Operations tasks**: Day-2 activities like deployments, migrations, restarts, health checks

**Why this scope?** Fills the gap between "infrastructure is up" and "services are reliably operated" - the operational workflows that teams run daily.

## Planning Modes

```bash
# Quick plan - fast preview
opal deploy --dry-run

# Resolved plan - complete execution contract  
opal deploy --dry-run --resolve > prod.plan

# Contract execution - verify plan matches reality
opal run --plan prod.plan
```

## Basic Syntax

```opal
# Variables and environment
var ENV = @env.ENVIRONMENT
var REPLICAS = @env.REPLICAS

# Conditional operations
deploy: {
    when @var.ENV {
        "production" -> {
            kubectl apply -f k8s/prod/
            kubectl scale --replicas=@var.REPLICAS deployment/app
        }
        else -> kubectl apply -f k8s/dev/
    }
}

# Retry and timeout
migrate: @retry(attempts=3, delay=10s) {
    @timeout(duration=5m) {
        psql @env.DATABASE_URL -f migrations/
    }
}
```

## Value Decorators

Inject values inline:
- `@env.PORT` - Environment variables
- `@var.REPLICAS` - Script variables  
- `@aws.secret.api_key(auth=prodAuth)` - External value lookups

## Execution Decorators  

Enhance command execution:
- `@retry(attempts=3) { ... }` - Retry failed operations
- `@timeout(duration=5m) { ... }` - Timeout protection
- `@parallel { ... }` - Concurrent execution

## Installation

### With Go
```bash
go install github.com/aledsdavies/opal/cli@latest
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
var ENV = @env.ENVIRONMENT
var VERSION = @env.APP_VERSION

deploy: {
    echo "Deploying @var.VERSION to @var.ENV"
    
    when @var.ENV {
        "production" -> {
            @retry(attempts=3) {
                kubectl apply -f k8s/prod/
                kubectl set image deployment/app app=@var.VERSION
                kubectl rollout status deployment/app
            }
        }
        else -> kubectl apply -f k8s/dev/
    }
}
```

### Database Migration
```opal
migrate: {
    try {
        echo "Starting migration..."
        psql @env.DATABASE_URL -f migrations/001-users.sql
        psql @env.DATABASE_URL -f migrations/002-indexes.sql
        echo "Migration complete"
    } catch {
        echo "Migration failed, rolling back"
        psql @env.DATABASE_URL -f rollback.sql
    }
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
