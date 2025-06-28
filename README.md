# devcmd

A simple tool for generating CLI commands from declarative definitions.

## Quick Start

Define your commands in a simple syntax:

```bash
# commands.cli
build: echo "Building project...";
test: echo "Running tests...";
clean: rm -rf build/;
```

Generate a CLI:

```bash
# With Nix
nix run github:aledsdavies/devcmd#basicDev -- --help

# Or install directly
go install github.com/aledsdavies/devcmd/cmd/devcmd@latest
devcmd commands.cli > main.go
go build -o mycli main.go
./mycli --help
```

## Features

- Simple declarative syntax for command definitions
- Generates standalone CLI binaries (no runtime dependencies)
- Variable substitution with `@var(NAME)` syntax
- Block commands for multi-step workflows
- Background process management with watch/stop commands
- Decorator support for shell commands and parallel execution
- Nix integration for development environments

## Command Syntax

### Basic Commands

```bash
# Variables
def SRC = ./src;
def BUILD_DIR = ./build;

# Simple commands
build: cd @var(SRC) && make;
test: go test ./...;
clean: rm -rf @var(BUILD_DIR);

# Multi-step commands
setup: {
  echo "Installing dependencies...";
  npm install;
  go mod download;
  echo "Setup complete"
}
```

### Process Management

```bash
# Background processes
watch dev: npm start;
stop dev: pkill -f "npm start";

# The CLI automatically creates:
# mycli dev start  (runs watch command)
# mycli dev stop   (runs stop command)
# mycli dev logs   (shows process logs)
# mycli status     (shows running processes)
```

### Advanced Features

```bash
# Parallel execution
deploy: {
  echo "Building and deploying...";
  @parallel {
    docker build -t frontend ./frontend;
    docker build -t backend ./backend
  };
  kubectl apply -f k8s/
}

# Shell commands with @sh() decorator
backup: @sh(tar -czf backup-$(date +%Y%m%d).tar.gz ./data);

# Variable expansion in decorators
serve: @sh(python -m http.server @var(PORT));

# Shell command substitution and variables work normally
timestamp: echo "Built at: $(date)";
user: echo "Current user: $USER";
```

## Installation & Usage

### Option 1: Nix Development Shell

Embed commands directly in your development environment:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { nixpkgs, devcmd, ... }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      devShells.x86_64-linux.default = pkgs.mkShell {
        buildInputs = with pkgs; [ go nodejs ];

        shellHook = (devcmd.lib.mkDevCommands {
          commandsFile = ./commands.cli;
        }).shellHook;
      };
    };
}
```

Usage:
```bash
$ nix develop
🚀 devcmd commands loaded from ./commands.cli

$ build    # Commands available directly
$ test
$ dev start
```

### Option 2: Standalone CLI Binary

Generate a distributable CLI:

```nix
{
  inputs.devcmd.url = "github:aledsdavies/devcmd";

  outputs = { devcmd, nixpkgs, ... }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      packages.x86_64-linux.default = devcmd.lib.mkDevCLI {
        name = "mycli";
        commandsFile = ./commands.cli;
        version = "1.0.0";
      };
    };
}
```

Usage:
```bash
$ nix build
$ ./result/bin/mycli --help
$ ./result/bin/mycli build
$ ./result/bin/mycli dev start
```

### Option 3: Manual Installation

```bash
# Install devcmd
go install github.com/aledsdavies/devcmd/cmd/devcmd@latest

# Generate CLI
devcmd commands.cli > main.go
go build -o mycli main.go

# Use the CLI
./mycli --help
./mycli build
```

## Library API (Nix)

### Generate Standalone CLI

```nix
devcmd.lib.mkDevCLI {
  name = "mycli";                          # CLI binary name
  commandsFile = ./commands.cli;           # Command definitions
  version = "1.0.0";                       # Version string
  meta = { description = "My CLI tool"; }; # Package metadata
}
```

### Embed in Development Shell

```nix
devcmd.lib.mkDevCommands {
  commandsFile = ./commands.cli;           # Command definitions
  debug = true;                            # Enable debug output
  extraShellHook = "echo Welcome!";        # Additional shell setup
}
```

### Convenience Functions

```nix
# Quick CLI generation
mydevCLI = devcmd.lib.quickCLI "mydev" ./commands.cli;

# Auto-detect commands.cli
mydevCLI = devcmd.lib.autoCLI "mydev";
```

## CLI Features

Generated CLIs include:

**Process Management:**
```bash
$ mycli status
NAME            PID      STATUS     STARTED
server          12345    running    14:32:15

$ mycli dev logs
[14:32:15] Starting server...

$ mycli dev stop
Stopping process server (PID: 12345)...
```

**Help & Discovery:**
```bash
$ mycli --help
Available commands:
  status              - Show running processes
  build               - Build the project
  dev start|stop|logs - Development server
```

## Examples

Try the included examples:

```bash
# Basic development CLI
nix run github:aledsdavies/devcmd#basicDev -- --help

# Web development CLI
nix run github:aledsdavies/devcmd#webDev -- install

# Go project CLI
nix run github:aledsdavies/devcmd#goProject -- build
```

## Get Started

Create a new project:

```bash
# Initialize from template
nix flake init -t github:aledsdavies/devcmd#basic

# Enter development shell
nix develop

# Try the CLI
myproject --help
myproject build
```

## Syntax Reference

```bash
# All commands end with semicolon
build: echo "Building...";

# Variables with @var(NAME) syntax
def PORT = 8080;
serve: python -m http.server @var(PORT);

# Variables in decorators
start: @sh(./server --port=@var(PORT));

# Shell command substitution and variables work normally
timestamp: echo "Time: $(date)";
user: echo "User: $USER";

# Block commands
setup: {
  echo "Step 1";
  echo "Step 2";
  echo "Done"
}

# Parallel execution
services: {
  @parallel {
    service1 --start;
    service2 --start
  }
}

# Watch/stop process pairs
watch server: @sh(./server --port=@var(PORT));
stop server: pkill -f "./server";

# Shell commands with @sh() decorator
backup: @sh(tar -czf backup-$(date +%Y%m%d).tar.gz .);
```

## Decorators

### @var(NAME)
Expands to the value of the defined variable:
```bash
def API_URL = https://api.example.com;
test: curl @var(API_URL)/health;
```

### @sh(command)
Wraps command for shell execution:
```bash
backup: @sh(find . -name "*.log" -exec rm {} \;);
```

### @parallel { ... }
Runs commands in parallel
```bash
build-all: {
  @parallel {
    go build ./cmd/server;
    go build ./cmd/client
  }
}
```

## Status

⚠️ **Early Development** - This project is experimental. The syntax and API may change.

The goal is to get real usage and feedback to shape the direction. Your experience and suggestions are valuable!

## Contributing

This is an early-stage project - experimentation encouraged!

- Try it with your projects and share your experience
- Open issues with feedback, questions, or interesting use cases
- Fork it and adapt it for your specific needs
- Share what works well and what doesn't

## License

Apache License, Version 2.0
