# Nix Configuration for Sigil

Simple Nix flake configuration for building and developing Sigil.

## Basic Usage

```bash
# Build the sigil binary
nix build

# Enter development environment
nix develop

# Run directly without installing
nix run github:builtwithtofu/sigil -- deploy --dry-run
```

## Development Environment

The development shell provides:
- Go toolchain
- All required development dependencies
- `sigil` binary built from current source

```bash
nix develop
sigil --version
```

## Integration

Add Sigil to your project's development environment:

```nix
{
  inputs.sigil.url = "github:builtwithtofu/sigil";
  
  outputs = { nixpkgs, sigil, ... }: {
    devShells.default = nixpkgs.mkShell {
      buildInputs = [ sigil.packages.x86_64-linux.default ];
    };
  };
}
```

## Requirements

- Nix with flakes enabled
- No special experimental features required
