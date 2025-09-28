# Core Module

The `core` module provides fundamental types, data structures, and utilities used throughout the Opal project.

## Purpose

This module contains the foundational components that other modules depend on:

- **AST (Abstract Syntax Tree)**: Data structures representing parsed Opal language constructs
- **Types**: Token types, expression types, and core type definitions  
- **Plan**: Execution plan data structures and DSL for dry-run visualization
- **Errors**: Common error types and utilities

## Key Components

### AST Package (`core/ast/`)
- `ast.go`: Core AST node definitions for commands, decorators, variables, etc.
- `builder.go`: Utilities for constructing AST nodes programmatically

### Types Package (`core/types/`)
- `types.go`: Token types, expression types, string literal types, and semantic token types
- Position and span information for precise error reporting

### Plan Package (`core/plan/`)
- `types.go`: Execution plan types and step definitions
- `dsl.go`: Plan builder DSL for creating structured execution plans

### Errors Package (`core/errors/`)
- `errors.go`: Common error types and error handling utilities

## Module Dependencies

This is a foundational module with minimal external dependencies:
- No dependencies on other Opal modules
- Standard Go library only
- Used by: `runtime`, `cli`, `testing` modules

## Usage

```go
import "github.com/aledsdavies/opal/core/ast"
import "github.com/aledsdavies/opal/core/types"
import "github.com/aledsdavies/opal/core/plan"
```

## Design Principles

- **Stability**: Core types should remain stable across versions
- **Minimal dependencies**: Avoid external dependencies to reduce complexity
- **Clear interfaces**: Well-defined boundaries between core types and implementations
- **Extensibility**: Designed to support future language features and decorators