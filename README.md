# devcmd

A **D**eclarative **E**xecution **V**ocabulary for generating standalone CLI tools from simple command definitions.

## Overview

devcmd transforms simple command definitions into fully-featured CLI binaries. Define your commands and workflows in a clean, maintainable syntax, and devcmd generates a standalone CLI tool that can be distributed and used anywhere.

```bash
# commands.cli
def VERSION = v1.0.0;
def BUILD_DIR = ./dist;

# Simple commands
build: go build -o $(BUILD_DIR)/myapp ./cmd;
test: go test -v ./...;
clean: rm -rf $(BUILD_DIR);

# Complex workflows with parallel execution
watch dev: {
  echo "Starting development environment...";
  @parallel: {
    go run ./cmd/server --dev;
    npm run watch-assets
  };
  echo "Development environment ready"
}

stop dev: {
  echo "Stopping services...";
  @parallel: {
    @sh(pkill -f "go run ./cmd/server" || true);
    @sh(pkill -f "npm run watch-assets" || true)
  }
}

# Multi-step deployments
deploy: {
  echo "Building for production...";
  make build;
  echo "Running tests...";
  make test;
  echo "Deploying to production...";
  kubectl apply -f k8s/
}
```

**Generated CLI:**
```bash
$ mycli --help
Available commands:
  status              - Show running background processes
  build               - build
  test                - test
  clean               - clean
  dev start|stop|logs - dev start|stop|logs
  deploy              - deploy

$ mycli build
$ mycli dev start
$ mycli deploy
```

## Features

- **Two Integration Modes**: Standalone CLI binaries OR embedded in development shells
- **Declarative syntax** for defining commands and workflows
- **Decorator system** for advanced shell operations and parallel execution
- **Variable substitution** with `$(name)` syntax
- **Process management** with watch/stop command pairing
- **Block commands** for multi-step workflows
- **Background processes** with automatic PID tracking
- **POSIX shell compatibility** - full support for pipes, redirections, subcommands
- **Command continuations** with backslash for readability
- **Standalone binaries** - no runtime dependencies
- **Cross-platform** - works on Linux, macOS, Windows

## Use Cases

**Development Tooling**
```bash
build: go build -o bin/app ./cmd;
test: go test ./...;
watch dev: air -c .air.toml;  # Live reload
```

**DevOps & Deployment**
```bash
def ENVIRONMENT = staging;
deploy: kubectl apply -f k8s/ -n $(ENVIRONMENT);
rollback: kubectl rollout undo deployment/app -n $(ENVIRONMENT);
status: kubectl get pods -n $(ENVIRONMENT);
```

**Data Pipeline Management**
```bash
def DATA_DIR = /var/data;
extract: python scripts/extract.py --output $(DATA_DIR);
transform: dbt run --target prod;
load: python scripts/load.py --source $(DATA_DIR);
```

**System Administration**
```bash
def LOG_DIR = /var/log/myapp;
backup: @sh(tar -czf backup-$(date +%Y%m%d).tar.gz $(LOG_DIR));
cleanup: @sh(find $(LOG_DIR) -name "*.log" -mtime +30 -delete);
monitor: tail -f $(LOG_DIR)/app.log;
```

## Installation & Usage

### Option 1: Development Shell Integration (Nix)

Embed the generated CLI directly in your development environment:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = builtins.currentSystem;
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs; [ go nodejs python3 ];

        # Embed generated CLI in development shell
        shellHook = (devcmd.lib.mkDevCommands {
          inherit pkgs system;
          commandsFile = ./commands.cli;  # Optional: auto-detects
        }).shellHook;
      };
    };
}
```

Usage:
```bash
$ nix develop
🚀 devcmd commands loaded from ./commands.cli
Generated CLI available as: devcmd-cli

$ devcmd-cli --help
$ devcmd-cli build
$ devcmd-cli dev start
```

### Option 2: Standalone CLI Binary

Generate a distributable CLI binary:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      # Generate CLI from commands.cli
      myCLI = devcmd.lib.mkDevCLI {
        name = "mycli";
        commandsFile = ./commands.cli;
        version = "1.0.0";
      };
    in {
      packages.${system}.default = myCLI;

      apps.${system}.default = {
        type = "app";
        program = "${myCLI}/bin/mycli";
      };
    };
}
```

Build and distribute:
```bash
$ nix build
$ ./result/bin/mycli --help
$ cp ./result/bin/mycli /usr/local/bin/  # Install system-wide
```

### Option 3: Without Nix

```bash
# Install devcmd
$ go install github.com/aledsdavies/devcmd/cmd/devcmd@latest

# Generate CLI from commands.cli
$ devcmd commands.cli > main.go
$ go build -o mycli main.go

# Use your CLI
$ ./mycli --help
```

## Command Syntax

### Basic Commands

```bash
# Variable definitions
def NAME = value;

# Simple commands
command-name: shell-command-here;

# Block commands
multi-step: {
  command1;
  command2;
  command3
}
```

### Process Management

```bash
# Watch commands start background processes
watch server: python -m http.server 8000;

# Stop commands clean up processes
stop server: @sh(pkill -f "python -m http.server" || true);

# Generated CLI automatically creates:
# mycli server start  (runs watch command)
# mycli server stop   (runs stop command)
```

### Decorator System

devcmd provides powerful decorators for advanced operations:

```bash
# Raw shell commands (for complex POSIX syntax)
cleanup: @sh(find . -name "*.tmp" -exec rm {} \;);

# Parallel execution
services: {
  @parallel: {
    api-server --port=8080;
    worker --queue=jobs;
    monitor --interval=5s
  };
  echo "All services started"
}

# Mixed decorators and regular commands
deploy: {
  echo "Starting deployment...";
  @parallel: {
    docker build -t frontend ./frontend;
    docker build -t backend ./backend
  };
  @sh(kubectl apply -f k8s/*.yaml);
  echo "Deployment complete"
}
```

### Advanced Features

```bash
# Variable substitution
def PORT = 8080;
def HOST = localhost;
serve: python -m http.server $(PORT) --bind $(HOST);

# Shell command substitution (escaped)
timestamp: echo "Current time: \$(date)";
git-info: echo "Commit: \$(git rev-parse HEAD)";

# Shell variables (escaped)
user-info: echo "Running as: \$USER";

# Command continuations
deploy: kubectl apply \
  -f deployment.yaml \
  -f service.yaml \
  --namespace production;

# Complex shell operations using @sh()
backup: @sh(tar -czf backup-$(date +%Y%m%d-%H%M%S).tar.gz ./data);
find-logs: @sh(find /var/log -name "*.log" -mtime -1 -exec grep "ERROR" {} \;);
```

### Syntax Rules

- All commands must end with semicolon (`;`)
- Use `$(VAR)` for devcmd variable references
- Use `\$(command)` for shell command substitution
- Use `\$VAR` for shell variable references
- Use `@sh(...)` for complex POSIX shell commands
- Use `@parallel: { ... }` for concurrent execution
- Comments start with `#`

## CLI Features

### Built-in Process Management

Generated CLIs automatically include:

```bash
$ mycli status
NAME            PID      STATUS     STARTED              COMMAND
server          12345    running    14:32:15             python -m http.server
worker          12346    running    14:32:20             python worker.py

$ mycli logs server
[14:32:15] Starting server on port 8000
[14:32:16] Server ready at http://localhost:8000

$ mycli server stop
Stopping process server (PID: 12345)...
Process stopped successfully
```

### Help & Discovery

```bash
$ mycli --help
Available commands:
  status              - Show running background processes
  build               - build
  test                - test
  server start|stop|logs - server start|stop|logs
  deploy              - deploy

$ mycli server
Usage: mycli server <start|stop|logs>
  start    Start the development server
  stop     Stop the development server
  logs     Show logs for the development server
```

## Command Files

devcmd looks for command definitions in:

1. File specified with `--file` flag
2. `./commands.cli` (preferred extension)
3. `./commands` (no extension)

## Library API (Nix)

### Embed CLI in Development Shell

```nix
devcmd.lib.mkDevCommands {
  inherit pkgs system;
  commandsFile = ./commands.cli;           # Command definitions
  name = "mycli";                          # CLI binary name
  debug = true;                            # Enable debug output
  extraShellHook = "echo Welcome!";        # Additional shell setup
}
```

### Generate Standalone CLI Package

```nix
devcmd.lib.mkDevCLI {
  name = "mycli";                          # CLI binary name
  commandsFile = ./commands.cli;           # Command definitions
  version = "1.0.0";                       # Version string
  meta = { description = "My CLI tool"; }; # Package metadata
}
```

### Development Shell with CLI

```nix
devcmd.lib.mkDevShell {
  name = "myproject-dev";               # Shell name
  cli = myGeneratedCLI;                 # Include generated CLI
  extraPackages = with pkgs; [ git ];  # Additional packages
  shellHook = "echo Welcome!";          # Custom shell setup
}
```

## Examples

### Web Application CLI

```bash
# commands.cli
def NODE_ENV = development;
def API_PORT = 3001;
def WEB_PORT = 3000;

install: {
  npm ci;
  cd api && go mod download
}

build: {
  echo "Building frontend...";
  npm run build;
  echo "Building API...";
  cd api && go build -o ../dist/api
}

watch dev: {
  echo "Starting development environment...";
  echo "Frontend: http://localhost:$(WEB_PORT)";
  echo "API: http://localhost:$(API_PORT)";
  @parallel: {
    npm start;
    @sh(cd api && go run . --port=$(API_PORT))
  }
}

stop dev: {
  echo "Stopping development environment...";
  @parallel: {
    @sh(pkill -f "npm start" || true);
    @sh(pkill -f "go run" || true)
  }
}

test: {
  @parallel: {
    npm test;
    @sh(cd api && go test ./...)
  }
}

deploy: {
  echo "Deploying to production...";
  @parallel: {
    docker build -t myapp-frontend .;
    @sh(cd api && docker build -t myapp-api .)
  };
  docker push myapp-frontend:latest;
  docker push myapp-api:latest;
  kubectl set image deployment/frontend frontend=myapp-frontend:latest;
  kubectl set image deployment/api api=myapp-api:latest
}
```

### DevOps CLI

```bash
# commands.cli
def CLUSTER = production;
def NAMESPACE = myapp;

status: kubectl get pods,svc,ing -n $(NAMESPACE);

logs: kubectl logs -f deployment/api -n $(NAMESPACE);

shell: kubectl exec -it deployment/api -n $(NAMESPACE) -- /bin/bash;

deploy: {
  echo "Deploying to $(CLUSTER)...";
  helm upgrade --install myapp ./chart \
    --namespace $(NAMESPACE) \
    --set image.tag=\$(git rev-parse HEAD)
}

rollback: {
  echo "Rolling back deployment...";
  helm rollback myapp -n $(NAMESPACE)
}

backup: {
  echo "Creating database backup...";
  @sh(DATE=$(date +%Y%m%d-%H%M%S));
  @sh(kubectl exec deployment/postgres -n $(NAMESPACE) -- \
    pg_dump myapp > backup-$DATE.sql);
  echo "Backup saved: backup-\$DATE.sql"
}

cleanup: {
  echo "Cleaning up old resources...";
  @parallel: {
    @sh(kubectl delete pods --field-selector=status.phase=Succeeded -n $(NAMESPACE));
    @sh(docker system prune -f);
    @sh(find ./logs -name "*.log" -mtime +7 -delete)
  };
  echo "Cleanup complete"
}

scale: {
  echo "Scaling services...";
  @parallel: {
    kubectl scale deployment/api --replicas=3 -n $(NAMESPACE);
    kubectl scale deployment/worker --replicas=5 -n $(NAMESPACE)
  }
}
```

## Architecture

- **ANTLR Grammar**: Robust parsing with decorator support and full POSIX shell compatibility
- **Go Code Generation**: Template-based CLI generation with decorator processing
- **Process Management**: Safe background process handling with PID tracking
- **Decorator System**: Extensible framework for advanced shell operations
- **Cross-platform**: Single binary works everywhere
- **No Runtime Dependencies**: Generated CLIs are self-contained

## Contributing

devcmd is in early development and experimentation is encouraged! Feel free to:

- **Fork it** and adapt it for your specific use cases
- **Iterate on the language design** - the DSL is still evolving
- **Share feedback** on what works well and what doesn't
- **Open issues** with suggestions, questions, or interesting use cases
- **Build cool stuff** with it and share your experience

The goal is for people to get real usage out of devcmd and help shape its direction through practical experience. Your fork might solve problems in ways that inspire the main project!

## License

Apache License, Version 2.0
