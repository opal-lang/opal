# Nix Configuration for Opal

Simple Nix flake configuration for building and developing Opal.

## Basic Usage

```bash
# Build the opal binary
nix build

# Enter development environment
nix develop

# Run directly without installing
nix run github:aledsdavies/opal -- deploy --dry-run
```

## Development Environment

The development shell provides:
- Go toolchain
- All required development dependencies
- `opal` binary built from current source

```bash
nix develop
opal --version
```

## Integration

Add Opal to your project's development environment:

```nix
{
  inputs.opal.url = "github:aledsdavies/opal";
  
  outputs = { nixpkgs, opal, ... }: {
    devShells.default = nixpkgs.mkShell {
      buildInputs = [ opal.packages.x86_64-linux.default ];
    };
  };
}
```

## Requirements

- Nix with flakes enabled
- No special experimental features required