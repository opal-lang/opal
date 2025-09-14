# devcmd

A simple tool for running and generating CLI commands from declarative definitions.

## Quick Start

Define your commands in a simple syntax:

```bash
# commands.cli
build: echo "Building project..."
test: echo "Running tests..."
clean: rm -rf build/
```

Run commands directly (Interpreter Mode):

```bash
# With Nix
nix run github:aledsdavies/devcmd -- run build

# Or install directly
go install github.com/aledsdavies/devcmd/cli@latest
devcmd run build
devcmd run test --dry-run  # Show execution plan
```

Execute commands with plan visualization:

```bash
# See execution plan without running  
devcmd build --dry-run

# Run commands directly
devcmd build
```

## Current Status - IR Refactoring in Progress

**🔄 Active Development:** The project is currently undergoing a major IR (Intermediate Representation) architecture refactoring to improve reliability and enable advanced features.

**✅ What's Working Now:**
- Plan generation with `--dry-run` mode (fully functional)
- String parsing and variable substitution (`@var(NAME)`, `@env(VAR)`)
- Tree visualization with ANSI colors
- Complex string interpolation including nested quotes

**🚧 What's Being Rebuilt:**
- Interpreter execution mode (temporarily broken during refactor)
- Generated CLI mode (being rebuilt on new IR foundation)
- Some complex decorator combinations

**🎯 Focus:** The refactoring provides a solid foundation for both execution modes with better reliability, testing, and advanced features.

## Features

**✅ Available Now (Plan Generation):**
- Simple declarative syntax for command definitions
- Rich execution plans with `devcmd run <command> --dry-run`
- Variable substitution with `@var(NAME)` syntax
- Shell operators: `&&`, `||`, `|`, `>>`
- Standardized global flags (`--quiet`, `--verbose`, `--ci`, etc.)
- Nix integration for development environments

**🚧 Coming Soon (Post-Refactor):**
- Reliable interpreter mode execution
- Full decorator support (@workdir, @timeout, @parallel)
- Background process management with watch/stop commands
- Standalone CLI binary generation (Phase 2)

**🎯 Current Focus:** The project is in active development focusing on a robust IR (Intermediate Representation) architecture. Plan generation and dry-run functionality are fully working, while execution modes are being rebuilt on the new foundation.

## Philosophy

**devcmd is a simple task runner with some useful bells and whistles.**

The core idea is declarative command definitions that work in multiple execution modes:
- **✅ Interpreter mode**: `devcmd build` for running commands (available now)
- **✅ Plan mode**: `devcmd build --dry-run` shows execution plans (available now)  
- **📋 Generator mode**: `./mycli build` standalone binaries (planned for future)

Being declarative enables powerful planning capabilities - you can see exactly what commands will run and how execution will flow before actually running anything.

The interpreter mode provides immediate productivity with shell commands, operators, and variables, while plan mode gives you visibility into execution flow. Generator mode will eventually create standalone binaries with no runtime dependencies.

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

### Shell Operators & Variables

```bash
# Shell operators (available now)
build: npm run build && npm run test
deploy: npm run build || echo "Build failed"
process: echo "data" | grep "pattern" | sort

# Variable expansion (available now)
var PORT = "8080"
var ENV = "development"
serve: python -m http.server @var(PORT)
status: echo "Running in @var(ENV) mode"

# Advanced features (coming soon)
deploy: {
  echo "Building and deploying..."
  @parallel {
    docker build -t frontend ./frontend
    docker build -t backend ./backend
  }
  kubectl apply -f k8s/
}
```

## Installation & Usage

### Option 1: Direct Installation

```bash
# Install with Go
go install github.com/aledsdavies/devcmd/cli@latest

# Or run with Nix (no installation needed)
nix run github:aledsdavies/devcmd -- run build

# Use interpreter mode
devcmd run build
devcmd run test --dry-run
devcmd run deploy --verbose
```

### Option 2: Nix Flake Integration

Add devcmd to your project's development environment:

```nix
{
  inputs.devcmd.url = "github:aledsdavies/devcmd";
  
  outputs = { nixpkgs, devcmd, ... }: {
    devShells.x86_64-linux.default = nixpkgs.legacyPackages.x86_64-linux.mkShell {
      buildInputs = [ 
        devcmd.packages.x86_64-linux.devcmd 
      ];
    };
  };
}
```

Usage:
```bash
$ nix develop
$ devcmd run build
$ devcmd run test --dry-run
```

## Current Examples

Try interpreter mode with these examples:

```bash
# Basic commands
echo 'build: echo "Building project..."' > commands.cli
echo 'test: echo "Running tests..."' >> commands.cli
devcmd run build

# With variables
echo 'var NAME = "myproject"' > commands.cli
echo 'greet: echo "Hello from @var(NAME)!"' >> commands.cli
devcmd run greet

# With operators
echo 'ci: npm run build && npm test && npm run lint' > commands.cli
devcmd run ci --dry-run  # See execution plan
```

## Available Commands

**Interpreter Mode Commands:**

```bash
# Help & Discovery
devcmd --help                    # Show all available commands
devcmd run --help                # Show run command options

# Execution
devcmd run <command>             # Execute a command
devcmd run <command> --dry-run   # Show execution plan
devcmd run <command> --verbose   # Detailed output
devcmd run <command> --quiet     # Minimal output
```

**Standardized Global Flags:**

Interpreter mode supports consistent global flags:

```bash
# Output control
devcmd run build --color=never --quiet    # CI-friendly output
devcmd run test --verbose                 # Detailed debugging

# Interaction control  
devcmd run deploy --yes --ci              # Automated execution

# Plan mode (dry-run)
devcmd run build --dry-run --verbose      # See execution plan
```

**Available flags**: `--color`, `--quiet`, `--verbose`, `--interactive`, `--yes`, `--ci`, `--dry-run`

## Documentation

For complete syntax reference and language specification, see:

- **[Language Specification](docs/devcmd_specification.md)** - Complete syntax guide with examples
- **[Examples](https://github.com/aledsdavies/devcmd/tree/main/.nix/examples.nix)** - Real-world CLI configurations

## Status

**🔄 IR Architecture Refactoring (Active Development)**
- Major architectural improvements in progress
- Focus on reliability and advanced features
- Plan generation fully functional for testing new features

**✅ What's Working (Available Now)**
- Plan generation with `--dry-run` mode
- Variable substitution (`@var(NAME)`, `@env(VAR)`)
- String interpolation including nested quotes
- Tree visualization with colors
- Standardized global flags
- Nix integration

**🚧 What's Being Rebuilt**
- Interpreter execution mode 
- Standalone CLI binary generation 
- Advanced decorators (@workdir, @timeout, @parallel)
- Process management features

The refactoring provides a solid IR foundation that will enable both interpreter and generator modes with better reliability, testing, and advanced features once completed.

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

## Development Status

**⚠️ Current State: Active Development (Phase 1)**

This project is undergoing a major architectural overhaul to implement IR-based unified execution:

**✅ Working Now:**
- Basic interpreter mode: `./devcmd run simple-command`
- Simple shell commands without operators
- Clean TDD test harness

**🚧 In Progress (Phase 1):**
- Shell operators (`&&`, `||`, `|`, `>>`)
- Decorators (`@workdir`, `@timeout`, `@cmd`, etc.)
- Plan generation (`--dry-run`)

**📋 Future Roadmap (Phase 2):**
- Generated mode (standalone CLI binaries) 
- Full semantic equivalence between interpreter and generated modes

The fundamental architecture is solid, but expect some commands to not work during this transition period.

## Contributing

This is an early-stage project - experimentation encouraged!

- Try it with your projects and share your experience
- Open issues with feedback, questions, or interesting use cases
- Fork it and adapt it for your specific needs
- Share what works well and what doesn't

## License

Apache License, Version 2.0
