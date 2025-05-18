# devcmd

A lightweight, extensible DSL for defining shell commands in Nix development environments.

## Overview

devcmd simplifies the creation of custom commands for Nix development shells. Define commands and aliases in a simple, maintainable syntax, and devcmd integrates them directly into your development environment.

```
# commands   <-- This is the default filename (no extension)
def go = ${pkgs.go}/bin/go
def npm = ${pkgs.nodejs}/bin/npm

build: go build -o ./bin/app ./cmd/main.go
run: go run ./cmd/main.go
test: go test ./...
dev: npm run dev
```

## Features

- Simple, declarative syntax for defining development commands
- Direct Nix store path variable substitution
- Easy integration with flake-based development environments
- No external dependencies required
- Extensible architecture for customization

## Installation

Add devcmd to your flake inputs:

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    devcmd.url = "github:yourusername/devcmd";
  };
  
  outputs = { self, nixpkgs, devcmd }: {
    # Your outputs...
  };
}
```

## Usage

1. Create a file named `commands` in your project root
2. Update your flake to use devcmd:

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    devcmd.url = "github:yourusername/devcmd";
  };
  
  outputs = { self, nixpkgs, devcmd }: {
    devShells.x86_64-linux.default = 
      let
        pkgs = nixpkgs.legacyPackages.x86_64-linux;
        commands = devcmd.lib.mkDevCommands {
          # By default, looks for ./commands
          # Optionally specify a different file:
          # commandsFile = ./path/to/commands;
          inherit pkgs;
        };
      in
      pkgs.mkShell {
        buildInputs = with pkgs; [ go nodejs ];
        inherit (commands) shellHook;
      };
  };
}
```

3. Enter your development environment:

```bash
$ nix develop
Dev shell initialized with your custom commands!
Available commands: build, run, test, dev

$ run
# Executes: go run ./cmd/main.go
```

## Command Files

devcmd will look for commands in the following order:

1. The path specified in `commandsFile` parameter
2. Inline commands provided in the `commands` parameter
3. A file named `commands` (no extension) in your project root (default)
4. A file named `commands.txt`
5. A file named `commands.devcmd`

The recommended approach is to use a file named `commands` in your project root.

## Command Syntax

The command file uses a simple syntax:

```
# Define variables (usually Nix store paths)
def <name> = <value>

# Define commands
<command-name>: <command-to-execute>

# Comments start with #
```

Variables are referenced using `${name}` syntax within commands.

Example:

```
# Define tools with full Nix store paths
def go = ${pkgs.go}/bin/go
def node = ${pkgs.nodejs}/bin/node
def python = ${pkgs.python3}/bin/python3

# Define project variables
def SRC_DIR = ./src
def OUT_DIR = ./dist

# Define commands
build: go build -o ${OUT_DIR}/app ${SRC_DIR}/main.go
run: ${OUT_DIR}/app
test: go test ./...
lint: go vet ./...

# Web development commands
dev: node ${SRC_DIR}/scripts/dev-server.js
serve: python -m http.server 8080 --directory ${OUT_DIR}

# Compound commands
all: build test lint
```

## Just Use Nix

While dev containers and Docker offer isolated environments, Nix provides several key advantages:

- **Truly reproducible environments**: Nix's content-addressed store ensures that every dependency is precisely tracked and can be perfectly reproduced.
- **Better resource efficiency**: No VM or container overhead - just the exact tools you need.
- **Cross-platform consistency**: The same Nix code works identically on Linux and macOS.
- **Incremental activation**: Nix environments can be entered instantly, no need to rebuild entire images.
- **Composable system**: Mix and match environments seamlessly, something containers can't easily do.

devcmd makes Nix development more approachable by simplifying the most common task: running commands in your project. No need to write complex shell hooks or remember esoteric Nix syntax - just define your commands once and they're available across all your development sessions.

## Extension Points

devcmd is designed to be extended. Key extension points include:

- Custom Go parser implementation in `pkg/parser/`
- Shell script generation templates
- Command metadata extraction

Example extension:

```go
// pkg/parser/types.go - Add a new command type
type AdvancedCommand struct {
    Name      string
    Command   string
    Description string
    Category  string
}

// pkg/parser/parser.go - Add pattern for advanced commands
var advancedCmdRegex = regexp.MustCompile(`^([A-Za-z0-9_-]+)\[([^]]+)\]: (.*)$`)

// In your parse function
if matches := advancedCmdRegex.FindStringSubmatch(line); matches != nil {
    result.AdvancedCommands = append(result.AdvancedCommands, AdvancedCommand{
        Name:        matches[1],
        Category:    matches[2],
        Command:     matches[3],
    })
    continue
}
```

See the source code for detailed extension documentation.

## Maintenance Status

devcmd was created to solve my own development workflow challenges. While released under Apache 2.0 for anyone to use, it comes with no maintenance guarantees.

The project's design prioritizes:
- A focused, minimal core that does one thing well
- Clear extension points for customization
- Well-documented, easily forkable code

I'll review issues and PRs as time permits, but response times will vary. Extensions and forks are encouraged over feature requests. If you need something different, the modular codebase should make it easy to adapt to your needs.

This tool exists because it makes my Nix workflow better - hopefully it helps yours too.

## License

This project is licensed under the Apache License, Version 2.0. See the LICENSE file for details.
