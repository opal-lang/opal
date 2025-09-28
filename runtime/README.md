# Runtime Module

The `runtime` module provides the execution environment and runtime services for opal commands and decorators.

## Purpose

This module contains the runtime infrastructure that powers opal execution:

- **Execution Context**: Runtime environment for command execution
- **Decorator Registry**: Registration and lookup system for all decorators
- **Decorator Interfaces**: Abstract interfaces that all decorators must implement

## Key Components

### Execution Package (`runtime/execution/`)
- `context.go`: Execution context providing variables, shell execution, and decorator services
- `types.go`: Execution result types and execution mode definitions
- `shell_test.go`: Tests for shell execution functionality

### Decorators Package (`runtime/decorators/`)
- `interfaces.go`: Core decorator interfaces (FunctionDecorator, BlockDecorator, PatternDecorator)
- `registry.go`: Global decorator registration and lookup system

### Registry Package (`runtime/registry/`)
- Future home for additional registry services

## Module Dependencies

- **Depends on**: `core` module for AST types and plan structures
- **Used by**: `cli` module for decorator implementations and execution
- **External**: No external dependencies beyond Go standard library

## Architecture

### Execution Modes
1. **InterpreterMode**: Direct execution of commands
2. **GeneratorMode**: Code generation for standalone binaries  
3. **PlanMode**: Dry-run plan generation for visualization

### Decorator System
The runtime provides a unified decorator system where:
- **Function decorators** expand inline (e.g., `@var(name)`, `@cmd(target)`)
- **Block decorators** wrap command blocks (e.g., `@timeout{}`, `@workdir{}`)
- **Pattern decorators** provide conditional logic (e.g., `@when{}`, `@try{}`)

## Usage

```go
import "github.com/aledsdavies/opal/runtime/execution"
import "github.com/aledsdavies/opal/runtime/decorators"

// Register a new decorator
decorators.RegisterFunction(&MyDecorator{})

// Create execution context
ctx := execution.NewContext(program)
```

## Design Principles

- **Mode-agnostic**: Decorators work across interpreter, generator, and plan modes
- **Extensible**: Easy to add new decorators without modifying core runtime
- **Safe**: Proper isolation and error handling for decorator execution
- **Performance**: Efficient execution with minimal overhead