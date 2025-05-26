# MyProject

A project using devcmd for development automation.

## Quick Start

### Using Nix (Recommended)

```bash
# Enter development environment
nix develop

# Use the generated CLI
myproject --help
myproject build
myproject test
```

### Manual Setup

1. Install [devcmd](https://github.com/aledsdavies/devcmd)
2. Generate CLI from commands:
   ```bash
   devcmd generate commands.devcmd > myproject.go
   go build -o myproject myproject.go
   ```
3. Use the CLI:
   ```bash
   ./myproject --help
   ```

## Available Commands

The CLI provides these development commands:

- `myproject build` - Build the project
- `myproject test` - Run tests
- `myproject clean` - Clean build artifacts
- `myproject deps` - Install dependencies
- `myproject format` - Format code
- `myproject lint` - Run linters
- `myproject watch dev` - Start development mode
- `myproject stop dev` - Stop development processes

## Customization

Edit `commands.devcmd` to customize the available commands for your project.

The devcmd syntax supports:
- Variables: `def SRC = ./src;`
- References: `build: cd $(SRC) && make`
- POSIX shell: `check: (which go && echo "found") || exit 1`
- Background processes: `watch server: npm start &`
- Multi-step commands: `setup: { npm install; go mod tidy; echo done }`

## Development

This project uses Nix flakes for reproducible development environments.

```bash
# Enter development shell
nix develop

# Run checks
nix flake check

# Build the CLI
nix build

# Run the CLI directly
nix run
```

## License

MIT
