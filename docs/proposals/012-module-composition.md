---
oep: 012
title: Module Composition and Plugin System
status: Draft
type: Integration
created: 2025-01-21
updated: 2025-01-27
---

# OEP-012: Module Composition and Plugin System

## Summary

Add a plugin system for sharing reusable decorators. Plugins are declared in `opal.mod`, locked in `opal.lock`, and available globally without imports. Plugins can come from a registry, git repositories, or local paths.

## Vision

Opal needs an ecosystem. Users should be able to:

```bash
# Add plugins from registry
opal add hashicorp/aws@5.0.0
opal add company/deploy-helpers@1.2.3

# Add from git
opal add github.com/team/opal-k8s@v1.0.0

# Use immediately - no imports needed
```

```opal
# Decorators from plugins are globally available
deploy: {
    @aws.instance.deploy(name="web", ami="ami-123", type="t3.medium") {
        sudo systemctl enable nginx
        curl http://127.0.0.1/health
    }
}
```

## Core Ideas

### The Mod File

`opal.mod` declares project dependencies:

```toml
# opal.mod
[plugins]
hashicorp/aws = "5.0.0"
company/custom = "1.2.3"
github.com/team/opal-k8s = "v1.0.0"

[shell-types]  # See OEP-017
posix-core = "1.0"
gnu-coreutils = "3.2"

[shell]
platform = "gnu"
```

### The Lock File

`opal.lock` pins exact versions for reproducibility:

```toml
# opal.lock - generated, committed to version control
[[plugin]]
name = "hashicorp/aws"
version = "5.0.0"
source = "registry+https://registry.opal.dev/hashicorp/aws"
checksum = "sha256:a1b2c3d4..."
```

### No Imports

Plugins registered in `opal.mod` are available globally. No import statements needed.

**Why?**
- Simpler mental model
- All dependencies visible in one place
- Matches how shell tools work (globally available)

### Host-Driven Execution

Plugins provision resources and return transport contexts. Opal executes blocks.

```
Plugin: "Here's an SSH connection to the EC2 instance I created"
Opal: "Thanks, I'll run the block commands over that connection"
```

**Why?**
- Opal stays in control of execution, secrets, determinism
- Decorators like `@retry`, `@timeout` work transparently
- Plugins can't leak secrets or break determinism

### Plugin Sources

**Registry:**
```bash
opal add hashicorp/aws@5.0.0
```

**Git repositories:**
```bash
opal add github.com/company/plugin@v1.2.3
opal add gitlab.com/team/utils@main
opal add git@github.com:company/private.git@v2.0.0
```

**Local path:**
```toml
[plugins]
./local/my-plugin = "1.0.0"
```

### Plugin Formats

Plugins can be:
- **WASM** - Portable, sandboxed, language-agnostic
- **Native** - For performance-critical plugins

Both can coexist; Opal chooses based on availability and performance needs.

### Terraform Provider Bridge

Terraform providers can be used as Opal plugins:

```bash
opal add terraform/hashicorp/aws@5.0.0
# Generates @aws.* decorators from Terraform provider schema
```

## CLI Commands

```bash
opal add <plugin>@<version>    # Add plugin
opal remove <plugin>           # Remove plugin
opal update <plugin>           # Update to latest compatible version
opal list                      # List installed plugins
opal search <query>            # Search registry
opal info <plugin>             # Show plugin details
```

## Open Questions

1. **Manifest format**: How do plugins declare their decorators? (Not YAML - should be Opal syntax)
2. **Registry**: Public registry? Self-hosted? Both?
3. **Plugin signing**: Cryptographic verification?
4. **Namespace management**: How to prevent squatting?
5. **Version conflicts**: How to resolve diamond dependencies?
6. **Capabilities**: How do plugins declare what they need (network, filesystem, etc.)?

## Related OEPs

- **OEP-009**: Terraform/Pulumi Provider Bridge
- **OEP-017**: Shell Command Type Definitions (uses `[shell-types]` in mod file)

## References

- **Go modules**: Inspiration for mod/lock file pattern
- **Cargo**: Rust's package manager (registry + git sources)
- **Nix flakes**: Reproducible builds with lock files
