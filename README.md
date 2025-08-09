# devcmd

A simple tool for generating CLI commands from declarative definitions.

## Quick Start

Define your commands in a simple syntax:

```bash
# commands.cli
build: echo "Building project..."
test: echo "Running tests..."
clean: rm -rf build/
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

## Philosophy

**devcmd is a simple task runner with some useful bells and whistles.**

The core idea is declarative command definitions that work in multiple execution modes:
- **Interpreter mode**: `devcmd run build` for development
- **Generated mode**: `./mycli build` for distribution (no runtime dependencies)  
- **Plan mode**: `devcmd plan build` or `./mycli --dry-run build` shows execution plans

Being declarative enables powerful planning capabilities in **both modes** - you can see exactly what commands will run, which decorators will apply, and how execution will flow before actually running anything. This works whether you're using the devcmd interpreter or a generated standalone binary.

Additional conveniences like `@workdir()`, `@parallel{}`, `@timeout()` decorators and module-aware workflows make it practical for real projects without adding complexity.

The goal is to keep task definitions simple and readable while providing the tools needed for modern development workflows.

## Command Syntax

### Basic Commands

```bash
# Variables
var SRC = "./src"
var BUILD_DIR = "./build"

# Simple commands
build: cd @var(SRC) && make
test: go test ./...
clean: rm -rf @var(BUILD_DIR)

# Multi-step commands
setup: {
  echo "Installing dependencies..."
  npm install
  go mod download
  echo "Setup complete"
}
```

### Process Management

```bash
# Background processes
watch dev: npm start
stop dev: pkill -f "npm start"

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
  echo "Building and deploying..."
  @parallel {
    docker build -t frontend ./frontend
    docker build -t backend ./backend
  }
  kubectl apply -f k8s/
}

# Shell command substitution and variables work normally
timestamp: echo "Built at: $(date)"
user: echo "Current user: $USER"
backup: tar -czf backup-$(date +%Y%m%d).tar.gz ./data

# Variable expansion
var PORT = "8080"
serve: python -m http.server @var(PORT)
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

## Documentation

For complete syntax reference and language specification, see:

- **[Language Specification](docs/devcmd_specification.md)** - Complete syntax guide with examples
- **[Examples](https://github.com/aledsdavies/devcmd/tree/main/.nix/examples.nix)** - Real-world CLI configurations

## Status

⚠️ **Early Development** - This project is experimental. The syntax and API may change.

The goal is to get real usage and feedback to shape the direction. Your experience and suggestions are valuable!

## Development

### Setting Up the Development Environment

This project uses Nix for reproducible development environments. The Nix environment automatically builds the development CLI from `commands.cli`.

```bash
# Enter the Nix development environment
nix develop

# This will:
# - Install all required development tools (Go, golangci-lint, etc.)
# - Build the 'dev' CLI from commands.cli automatically
# - Make both 'devcmd' and 'dev' binaries available
```

**Requirements:**
- Nix with flakes enabled
- On first run, it will build the development environment (may take a few minutes)

**Available binaries after `nix develop`:**
- `devcmd` - The CLI generator (interpreter mode, requires `devcmd run <command>`)
- `dev` - Compiled CLI from commands.cli (direct usage: `dev <command>`)

### Project Architecture

The project uses a multi-module Go workspace with clear separation of concerns:

```
core/     - Foundation module (AST, types, errors)
├── runtime/  - Decorators and execution contexts (depends on core)
├── testing/  - Test utilities and frameworks (depends on core + runtime)
└── cli/      - Main CLI application (depends on all modules)
```

**Module Dependencies:**
- **`core/`** - No external dependencies, provides foundational types
- **`runtime/`** - Depends on `core/`, implements decorator system
- **`testing/`** - Depends on `core/` + `runtime/`, provides test utilities  
- **`cli/`** - Depends on all modules, contains lexer, parser, and engine

This hierarchy ensures clean separation and allows independent development of each module while maintaining proper dependency order.

### Development Commands

```bash
# Option 1: Use compiled dev CLI (recommended)
nix develop -c dev build
nix develop -c dev test  
nix develop -c dev test-quick
nix develop -c dev format
nix develop -c dev lint
nix develop -c dev ci

# Option 2: Use devcmd in interpreter mode
nix develop -c devcmd run build
nix develop -c devcmd run test

# Module-specific testing
nix develop -c dev test-core      # Test core module only
nix develop -c dev test-runtime   # Test runtime module only  
nix develop -c dev test-testing   # Test testing module only
nix develop -c dev test-cli       # Test CLI module only
```

### Maintaining Nix Packaging

The project uses fixed-output derivations for reliable builds. When updating dependencies or making changes that affect the build, you may need to update SHA hashes:

```bash
# 1. If you get hash mismatch errors, temporarily use fake hash
# Edit .nix/lib.nix and set:
# outputHash = lib.fakeHash;

# 2. Build to get the correct hash
nix build

# 3. Copy the correct hash from the error message
# Update the outputHash in .nix/lib.nix with the real hash

# 4. Build again to verify
nix build
```

**When to update hashes:**
- After updating Go dependencies (`go.mod` changes)
- When adding new dependencies to the project
- If you see "hash mismatch" errors during builds

**Files that may need hash updates:**
- `.nix/lib.nix` - Contains the `outputHash` for the fixed-output derivation

## Contributing

This is an early-stage project - experimentation encouraged!

- Try it with your projects and share your experience
- Open issues with feedback, questions, or interesting use cases
- Fork it and adapt it for your specific needs
- Share what works well and what doesn't

## License

Apache License, Version 2.0
