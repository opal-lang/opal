# Opal

**Plan-Verify-Execute Engine for Safe Automation**

Write automation that shows you exactly what it will do before it does it. Perfect for any domain where mistakes are expensive and auditability matters.

> **Domain-agnostic design**: Opal itself is domain-agnostic. It becomes useful in a given field through the decorator sets provided. The DevOps examples here use `@shell` and `@kubectl`, but the same rules apply equally to data, science, security, or any other domain.

## Quick Start

Define your operations:

```bash
# commands.opl
build: echo "Building project..."
test: echo "Running tests..."
deploy: kubectl apply -f k8s/
```

Run with planning:

```bash
# Install
go install github.com/aledsdavies/opal/cli@latest

# See what will execute
opal deploy --dry-run

# Run the operation
opal deploy
```

## Why Opal?

**Plan first, execute with confidence**: See exactly what will happen before it happens
**Contract-based execution**: Generate immutable execution contracts for perfect auditability
**Safe automation**: Deterministic execution with safety guarantees prevents runaway scripts
**Domain agnostic**: Extensible to any field where automation needs to be predictable and safe

## Core Features

### Planning Modes
```bash
# Quick plan - fast preview
opal deploy --dry-run

# Resolved plan - complete execution contract  
opal deploy --dry-run --resolve > prod.plan

# Contract execution - verify plan matches reality
opal run --plan prod.plan
```

### Operations Syntax
```opal
# Variables and environment
var ENV = @env("ENVIRONMENT", default="dev")
var REPLICAS = @env("REPLICAS", default=1)

# Conditional operations
deploy: {
    when @var(ENV) {
        "production" -> {
            kubectl apply -f k8s/prod/
            kubectl scale --replicas=@var(REPLICAS) deployment/app
        }
        else -> kubectl apply -f k8s/dev/
    }
}

# Retry and timeout decorators
migrate: @retry(attempts=3, delay=10s) {
    @timeout(5m) {
        psql @env("DATABASE_URL") -f migrations/
    }
}
```

### Value Decorators
Inject values inline:
- `@env("PORT", default=3000)` - Environment variables
- `@var(REPLICAS)` - Script variables  
- `@aws.secret("api-key")` - External value lookups

### Execution Decorators  
Enhance command execution:
- `@retry(attempts=3) { ... }` - Retry failed operations
- `@timeout(5m) { ... }` - Timeout protection
- `@parallel { ... }` - Concurrent execution

## Domain Examples

### DevOps & Infrastructure
```opal
deploy: {
    kubectl apply -f k8s/
    @retry(attempts=3) { kubectl rollout status deployment/app }
}
```

### Data Engineering  
```opal
etl_pipeline: {
    @snowflake.load(table="events", from_s3=@var(RAW_BUCKET))
    @dbt.run(model="daily_summary")
}
```

### Security Automation
```opal
incident_response: {
    @okta.suspend_user(email=@var(ALERT.user))
    @crowdstrike.isolate_host(hostname=@var(ALERT.host))
}
```

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
var ENV = @env("ENVIRONMENT", default="dev")
var VERSION = @env("APP_VERSION", default="latest")

deploy: {
    echo "Deploying @var(VERSION) to @var(ENV)"
    
    when @var(ENV) {
        "production" -> {
            @retry(attempts=3) {
                kubectl apply -f k8s/prod/
                kubectl set image deployment/app app=@var(VERSION)
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
        psql @env("DATABASE_URL") -f migrations/001-users.sql
        psql @env("DATABASE_URL") -f migrations/002-indexes.sql
        echo "Migration complete"
    } catch {
        echo "Migration failed, rolling back"
        psql @env("DATABASE_URL") -f rollback.sql
    }
}
```

## Development

This project uses Nix for development environments:

```bash
# Enter development environment
nix develop

# Available commands
opal build      # Build the project
opal test       # Run tests  
opal lint       # Run linting
opal format     # Format code
```

## Status

**Early Development**: Opal is in early development focusing on language design and architecture.

**Completed**:
- Language specification and syntax design
- V2 lexer with comprehensive tokenization
- Planning and contract execution model design
- Multi-module Go architecture

**In Progress**:
- Parser implementation for the new syntax
- Execution engine with decorator support
- Plan generation and contract verification

**Planned**:
- Complete execution decorators (`@retry`, `@timeout`, `@parallel`)
- Value decorators (`@env`, `@var`, `@aws.secret`)
- Plugin system for custom decorators
- Infrastructure-as-code decorators

## Philosophy

Opal treats operations as **plans that can be reviewed before execution**. Instead of "run and hope," you:

1. **Plan** your operation and see exactly what will execute
2. **Review** the plan with your team for safety and correctness
3. **Execute** with contract verification to catch environment changes

This gives operations teams the confidence to execute complex workflows safely.

## License

Apache License, Version 2.0