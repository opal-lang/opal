# CLI Module

The `cli` module provides the command-line interface and main executable for devcmd.

## Purpose

This module contains the CLI application that users interact with:

- **Main CLI Application**: The `devcmd` command-line tool
- **Built-in Decorators**: Core decorator implementations
- **Engine**: Execution engine for interpreter and generator modes
- **Language Processing**: Lexer and parser for the devcmd language

## Key Components

### Main Application (`cli/main.go`)
- CLI entry point with command-line argument parsing
- Integration point that wires together all components

### Built-in Decorators (`cli/internal/builtins/`)
- `var.go`, `env.go`, `cmd.go`: Function decorators
- `timeout.go`, `parallel.go`, `retry.go`, `workdir.go`: Block decorators  
- `when.go`, `try.go`: Pattern decorators
- `confirm.go`: Interactive decorators
- All decorators include comprehensive test suites

### Execution Engine (`cli/internal/engine/`)
- `engine.go`: Main execution engine supporting all modes
- Integration tests and performance benchmarks
- Code generation for standalone binaries

### Language Processing
- **Lexer** (`cli/internal/lexer/`): Tokenization with multi-mode parsing
- **Parser** (`cli/internal/parser/`): AST construction from tokens

## Module Dependencies

- **Depends on**: `core` and `runtime` modules
- **External**: `cobra` for CLI framework, `go-cmp` for testing
- **Imports builtins**: Automatically registers all built-in decorators

## CLI Commands

### Main Commands
- `devcmd <command>`: Execute a command from commands.cli
- `devcmd version`: Show version information

### Options  
- `--dry-run`: Show execution plan without running
- `--file/-f`: Specify custom commands file
- `--no-color`: Disable colored output

## Usage Examples

```bash
# Run a command
devcmd run build

# Dry-run to see execution plan
devcmd run deploy --dry-run

# Use custom commands file
devcmd run test -f my-commands.cli

# Show execution plan  
devcmd build --dry-run
```

## Architecture

### Execution Flow
1. **Parse**: Command-line arguments and commands.cli file
2. **Plan**: Generate execution plan (dry-run) or execute directly  
3. **Execute**: Run commands through engine with decorator support
4. **Output**: Results, errors, or generated code

### Decorator Integration
- All built-in decorators auto-register via `init()` functions
- Engine provides unified execution across interpreter/generator modes
- Plan generation supports nested decorator visualization

## Development

### Adding New Decorators
1. Implement decorator interface in `cli/internal/builtins/`
2. Add `init()` function to register decorator
3. Include comprehensive tests
4. Update documentation

### Testing Strategy
- Unit tests for individual decorators
- Integration tests for engine functionality  
- End-to-end tests for complete workflows
- Performance benchmarks for critical paths