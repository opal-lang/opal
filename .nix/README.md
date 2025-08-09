# Nix Configuration for devcmd

This project uses **dynamic derivations** to properly handle the two-stage build process:
1. Stage 1: Build devcmd binary 
2. Stage 2: Use devcmd to generate CLI with network access for Go modules

## Required Nix Configuration

To use this flake, you need to enable experimental features in your Nix configuration:

### Option 1: Global Configuration (recommended)

Add to your `~/.config/nix/nix.conf` or `/etc/nix/nix.conf`:

```
experimental-features = nix-command flakes dynamic-derivations ca-derivations recursive-nix
```

### Option 2: Per-command

```bash
nix --extra-experimental-features "dynamic-derivations ca-derivations recursive-nix" develop
```

### Option 3: Environment Variable

```bash
export NIX_CONFIG="experimental-features = nix-command flakes dynamic-derivations ca-derivations recursive-nix"
```

## Why Dynamic Derivations?

Dynamic derivations solve the fundamental issue with our build process:

- **Problem**: devcmd needs to generate CLIs that require network access for Go modules
- **Old approach**: Fixed-output derivations can't reference store paths (like devcmd binary)
- **New approach**: Stage 1 creates a derivation, Stage 2 builds it with network access

This provides clean separation while maintaining reproducibility and allowing network access where needed.

## Fallback

If dynamic derivations are not available, the system will fall back to an all-in-one fixed-output derivation approach that builds devcmd from source within the derivation.