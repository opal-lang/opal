# Shell Chaining Architecture

## Overview

Devcmd implements sophisticated shell chaining that works consistently across both interpreter and generator execution modes. The system allows ActionDecorators (like `@cmd()`) to be chained with shell operators (`&&`, `||`, `|`, `>>`) while preserving the exact semantics of shell execution.

## Core Architecture

### Parsing Layer
- **File**: `runtime/execution/templates.go`
- **Function**: `parseActionDecoratorChain()` and `parseShellOperators()`
- **Purpose**: Parses shell content into a sequence of commands and operators

### Template Generation
- **File**: `runtime/execution/templates.go` 
- **Function**: `GenerateDirectActionTemplate()`
- **Purpose**: Generates Go code that respects shell chaining semantics using `CommandResult`

### Data Structures
```go
type ChainElement struct {
    Type           string // "action", "operator", "text"
    ActionName     string // For ActionDecorator (@cmd, etc.)
    ActionArgs     []ast.NamedParameter
    Operator       string // "&&", "||", "|", ">>"
    Text           string // For shell commands
    VariableName   string // Generated Go variable name
    IsPipeTarget   bool   // Receives piped input
    IsFileTarget   bool   // Target for >> operation
}
```

## Shell Operator Semantics

### AND Operator (`&&`)
**Shell**: `cmd1 && cmd2` - cmd2 runs only if cmd1 succeeds
```go
// Generated Go code
result1 := executeCmd1()
if result1.Success() {
    result2 := executeCmd2()
    return result2
}
return result1 // Failed result
```

### OR Operator (`||`) 
**Shell**: `cmd1 || cmd2` - cmd2 runs only if cmd1 fails
```go
// Generated Go code
result1 := executeCmd1()
if result1.Failed() {
    result2 := executeCmd2()
    return result2
}
return result1 // Success result
```

### PIPE Operator (`|`)
**Shell**: `cmd1 | cmd2` - stdout of cmd1 feeds to stdin of cmd2
```go
// Generated Go code with helper function
result1 := executeCmd1()
result2 := executeShellCommandWithInput(ctx, "cmd2", result1.Stdout)
return result2
```

### APPEND Operator (`>>`)
**Shell**: `cmd1 >> file.txt` - stdout of cmd1 appends to file
```go
// Generated Go code with file operations
result1 := executeCmd1()
if err := appendToFile("file.txt", result1.Stdout); err != nil {
    return CommandResult{Stdout: "", Stderr: err.Error(), ExitCode: 1}
}
return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
```

## ActionDecorator Integration

### Command Reference (`@cmd`)
```devcmd
deploy: @cmd(build) && kubectl apply -f k8s/
```

Generates:
```go
buildResult := executeBuild()
if buildResult.Success() {
    kubernetesResult := executeShellCommand(ctx, "kubectl apply -f k8s/")
    return kubernetesResult
}
return buildResult
```

### Mixed Chaining
```devcmd
# ActionDecorators mixed with shell commands
verify: @cmd(build) && npm test && @cmd(deploy) || echo "Pipeline failed"
```

Generates:
```go
buildResult := executeBuild()
if buildResult.Success() {
    testResult := executeShellCommand(ctx, "npm test")
    if testResult.Success() {
        deployResult := executeDeploy()
        if deployResult.Success() {
            return deployResult
        }
    }
}
return executeShellCommand(ctx, "echo \"Pipeline failed\"")
```

## Execution Modes

### Interpreter Mode
- Uses shell directly for chaining: `exec.Command("sh", "-c", "cmd1 && cmd2")`
- ActionDecorators executed through devcmd runtime
- Native shell performance and behavior

### Generator Mode
- Generates Go code with explicit `CommandResult` logic
- Each chain element becomes Go function calls
- Preserves shell semantics through conditional execution
- No shell dependency in generated binary

## Template Functions

### Helper Functions Generated
```go
// For pipe operations
executeShellCommandWithInput := func(ctx context.Context, command, input string) CommandResult

// For regular commands  
executeShellCommand := func(ctx context.Context, command string) CommandResult

// For file operations
appendToFile := func(filename, content string) error
```

### Template Context
- Working directory propagation
- Environment variable capture
- Variable substitution
- Error handling consistency

## Key Implementation Files

1. **`runtime/execution/templates.go`**: Core chaining logic and template generation
2. **`runtime/execution/base_context.go`**: ChainElement data structure
3. **`cli/internal/engine/engine.go`**: CLI template integration
4. **`core/ast/ast.go`**: AST node definitions for ActionDecorators

## Error Handling

### Chain Termination
- `&&`: Chain terminates on first failure, returns failed CommandResult
- `||`: Chain terminates on first success, returns successful CommandResult  
- `|`: Pipe operations continue even with non-zero exit codes (shell behavior)
- `>>`: File operations return error CommandResult on write failure

### CommandResult Interface
```go
type CommandResult struct {
    Stdout   string
    Stderr   string  
    ExitCode int
}

func (r CommandResult) Success() bool { return r.ExitCode == 0 }
func (r CommandResult) Failed() bool { return r.ExitCode != 0 }
func (r CommandResult) ToError() error { /* converts to Go error */ }
```

## Performance Characteristics

### Interpreter Mode
- Direct shell execution: ~1-2ms overhead per command
- Native pipe buffering and streaming
- System shell optimizations

### Generator Mode  
- Go function calls: ~0.1ms overhead per command
- Explicit buffering with `io.MultiWriter` for streaming
- Compiled binary performance

## Testing Strategy

The chaining logic is tested through:
1. **Parser tests**: Verify correct AST generation for chains
2. **Engine integration tests**: End-to-end CLI generation with chaining
3. **Template tests**: Verify generated Go code correctness
4. **Execution tests**: Verify semantic equivalence between modes

This architecture ensures that developers can write natural shell-like command chains that work identically whether executed through the devcmd interpreter or compiled into standalone Go binaries.