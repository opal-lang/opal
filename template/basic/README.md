# MyProject

A project using devcmd for development automation.

## Quick Start

```bash
# Enter development environment
nix develop

# Use the generated CLI
myproject --help
myproject build
myproject test
```

## Available Commands

The CLI provides these development commands:

- `myproject build` - Build the project
- `myproject test` - Run tests
- `myproject clean` - Clean build artifacts

## Customization

Edit `commands.cli` to add more commands for your project.

See the [devcmd documentation](https://github.com/aledsdavies/devcmd) for syntax details.

## Development

```bash
# Enter development shell
nix develop

# Build the CLI
nix build

# Run the CLI directly
nix run
```

## License

MIT
