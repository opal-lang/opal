# Devcmd Execution Modes

Devcmd provides three distinct execution modes that handle the same command definitions in fundamentally different ways. Understanding these modes is crucial for effective use of devcmd in different scenarios.

---

## Overview

Devcmd operates in three execution modes:

1. **Interpreter Mode** - Just-in-time execution by the devcmd engine
2. **Generated Mode** - Ahead-of-time compilation to standalone CLI binaries  
3. **Plan Mode** - Dry-run visualization without execution

Each mode processes the same `.cli` command definitions but with different execution strategies optimized for different use cases.

---

## Interpreter Mode (JIT Execution)

Interpreter mode executes commands directly through the devcmd engine at runtime, similar to how shell scripts or Python interpreters work.

### How It Works

```
User runs: devcmd build
    ↓
1. Parse commands.cli file → AST
2. Initialize ExecutionContext with variables
3. Find "build" command in AST
4. Walk through command content immediately
5. Execute shell commands via exec.Command()
6. Return results directly
```

### Example Flow

**Command definition:**
```devcmd
var PROJECT = "myapp"
build: echo "Building @var(PROJECT)"
```

**Runtime execution in interpreter mode:**
```go
// At runtime:
PROJECT := "myapp"  // Variable resolved from AST
cmdStr := "echo \"Building " + PROJECT + "\""  // String interpolation
exec.CommandContext(ctx, "sh", "-c", cmdStr).Run()  // Direct execution
```

### Key Characteristics

- **Just-in-time execution** - Commands parsed and executed on-demand
- **Direct shell execution** - Uses `exec.CommandContext()` immediately
- **Dynamic variable resolution** - Variables resolved at runtime
- **No compilation step** - The devcmd binary acts as interpreter
- **Interactive debugging** - Easy to trace execution and modify commands

### Usage

```bash
# Direct command execution
devcmd build
devcmd test
devcmd deploy

# With environment variables
ENV=production devcmd deploy

# With debug output
devcmd --debug deploy
```

### Best For

- **Development and testing** - Quick iteration on command definitions
- **Interactive use** - Manual command execution during development
- **Dynamic environments** - Commands that change frequently
- **Debugging workflows** - Easy to trace and modify execution
- **Prototyping** - Testing new command combinations

---

## Generated Mode (AOT Compilation)

Generated mode produces standalone CLI binaries by generating complete Go source code from command definitions.

### How It Works

```
User runs: devcmd build -f commands.cli --binary mycli
    ↓
1. Parse commands.cli file → AST
2. Generate complete Go source code for CLI
3. Create main.go with Cobra command structure
4. Generate go.mod with dependencies
5. Compile to standalone binary
    ↓
User runs: ./mycli build
    ↓
6. Execute pre-compiled Go code (no parsing needed)
```

### Example Generated Code

**Command definition:**
```devcmd
var PROJECT = "myapp"
build: echo "Building @var(PROJECT)"
```

**Generated Go code:**
```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "github.com/spf13/cobra"
)

func main() {
    ctx := context.Background()
    
    // Variables embedded at compile time
    PROJECT := "myapp"
    
    // Generated command function
    executeBuild := func() error {
        BuildCmdStr := fmt.Sprintf("echo \"Building %s\"", PROJECT)
        BuildExecCmd := exec.CommandContext(ctx, "sh", "-c", BuildCmdStr)
        BuildExecCmd.Stdout = os.Stdout
        BuildExecCmd.Stderr = os.Stderr
        BuildExecCmd.Stdin = os.Stdin
        if err := BuildExecCmd.Run(); err != nil {
            fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
            os.Exit(1)
        }
        return nil
    }
    
    // Cobra CLI setup
    buildCmd := &cobra.Command{
        Use: "build",
        Run: func(cmd *cobra.Command, args []string) {
            if err := executeBuild(); err != nil {
                fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
                os.Exit(1)
            }
        },
    }
    
    rootCmd := &cobra.Command{Use: "mycli"}
    rootCmd.AddCommand(buildCmd)
    rootCmd.Execute()
}
```

### Key Characteristics

- **Ahead-of-time compilation** - Complete CLI generated and compiled once
- **Template-based code generation** - Uses Go templates for clean code
- **Static variable embedding** - Variables resolved at generation time
- **Standalone binaries** - No runtime dependency on devcmd
- **Cobra CLI framework** - Professional CLI with help, flags, and subcommands
- **Performance optimized** - No parsing overhead at runtime

### Usage

```bash
# Generate CLI binary
devcmd build -f commands.cli --binary mycli

# Use generated CLI
./mycli --help
./mycli build
./mycli test
./mycli deploy

# Generated CLI includes help and flags
./mycli build --help
./mycli --dry-run deploy
```

### Generated CLI Features

Generated CLIs include professional features:

- **Subcommand structure** - Each command becomes a subcommand
- **Help system** - Auto-generated help text and usage
- **Dry-run support** - Built-in `--dry-run` flag for plan mode
- **Error handling** - Proper exit codes and error messages
- **Shell completion** - Can be extended for bash/zsh completion

### Best For

- **Production deployments** - Reliable, self-contained binaries
- **CI/CD pipelines** - No external dependencies
- **Distribution** - Share CLIs with users who don't have devcmd
- **Performance critical** - No parsing overhead
- **Containerized environments** - Smaller final container images
- **Team collaboration** - Consistent CLI interface across environments

---

## Plan Mode (Dry-Run Visualization)

Plan mode provides execution visualization without actually running commands, similar to Terraform's plan functionality.

### How It Works

```
User runs: devcmd --dry-run deploy
    ↓
1. Parse commands.cli file → AST
2. Initialize ExecutionContext in PlanMode
3. Walk through command content
4. Generate execution plan tree
5. Display formatted plan without execution
```

### Example Plan Output

**Command definition:**
```devcmd
var PROJECT = "myapp"
var ENV = "production"

deploy: @when(ENV) {
    production: {
        echo "Deploying @var(PROJECT) to production"
        kubectl apply -f k8s/prod/
        kubectl rollout status deployment/api
    }
    staging: {
        echo "Deploying @var(PROJECT) to staging" 
        kubectl apply -f k8s/staging/
    }
}
```

**Plan output:**
```
🔍 Execution Plan for: deploy

deploy:
└─ @when(ENV) {production}
   ├─ echo "Deploying myapp to production"
   ├─ kubectl apply -f k8s/prod/
   └─ kubectl rollout status deployment/api
```

### Key Characteristics

- **No execution** - Commands are analyzed but not run
- **Tree visualization** - Hierarchical display of execution flow
- **Variable resolution** - Shows actual values that would be used
- **Decorator expansion** - Shows how decorators would behave
- **Conditional branch preview** - Shows which pattern branches would execute
- **Safe exploration** - Understand complex workflows without side effects

### Usage

```bash
# Plan mode in interpreter
devcmd --dry-run deploy
devcmd --dry-run build

# Plan mode in generated CLI
./mycli --dry-run deploy
./mycli --dry-run backup

# With environment variables
ENV=staging devcmd --dry-run deploy
```

### Plan Output Features

- **Color coding** - Commands, decorators, and variables highlighted
- **Indentation** - Clear hierarchy with tree-like structure
- **Variable substitution** - Shows resolved variable values
- **Conditional logic** - Indicates which branches would execute
- **Decorator behavior** - Shows how decorators modify execution

### Best For

- **Understanding workflows** - Visualize complex command sequences
- **Debugging logic** - See which conditional branches execute
- **Documentation** - Generate execution plans for review
- **Safety checks** - Verify commands before destructive operations
- **Team reviews** - Share execution plans for approval
- **Learning** - Understand how decorators and patterns work

---

## Decorator Behavior Across Modes

Decorators implement unified behavior across all execution modes through the same interface:

### ValueDecorators (e.g., `@var`, `@env`)

**Interpreter Mode:**
```go
// @var(PROJECT) resolves to:
value := ctx.GetVariable("PROJECT")  // "myapp"
// Directly substituted into shell command
```

**Generated Mode:**
```go
// @var(PROJECT) generates:
PROJECT := "myapp"  // Global variable in generated code
// Used in fmt.Sprintf() formatting
```

**Plan Mode:**
```go
// @var(PROJECT) displays as:
"@var(PROJECT) → myapp"  // Shows resolution in plan
```

### ActionDecorators (e.g., `@cmd`)

**Interpreter Mode:**
```go
// @cmd(build) executes:
err := ctx.ExecuteCommand("build")  // Direct recursive call
```

**Generated Mode:**
```go
// @cmd(build) generates:
buildResult := executeBuild()  // Call to generated function
if buildResult.Failed() { os.Exit(1) }
```

**Plan Mode:**
```go
// @cmd(build) displays as:
"@cmd(build) → Execute command: build"  // Shows what would happen
```

### BlockDecorators (e.g., `@parallel`, `@timeout`)

**All Modes:**
- Wrap contained commands with enhancement behavior
- Maintain consistent semantics across execution strategies
- Show enhancement details in plan mode

### PatternDecorators (e.g., `@when`, `@try`)

**All Modes:**
- Evaluate conditions using same logic
- Execute appropriate branches in interpreter/generated
- Show selected branches in plan mode

---

## Performance Comparison

| Aspect | Interpreter Mode | Generated Mode | Plan Mode |
|--------|------------------|----------------|-----------|
| **Startup Time** | Fast (no compilation) | Instant (pre-compiled) | Fast (no execution) |
| **Runtime Performance** | Moderate (parsing overhead) | Fast (no parsing) | Instant (no execution) |
| **Memory Usage** | Moderate (AST in memory) | Low (compiled code) | Low (plan only) |
| **File Size** | Small (source only) | Larger (binary) | Small (source only) |
| **Dependencies** | Requires devcmd | Self-contained | Requires devcmd |

---

## Mode Selection Guide

### Choose **Interpreter Mode** when:
- Developing and testing command definitions
- Working interactively with frequently changing commands
- Debugging workflow logic
- Prototyping new command combinations
- Need dynamic variable resolution

### Choose **Generated Mode** when:
- Deploying to production environments
- Distributing CLIs to users without devcmd
- Running in CI/CD pipelines
- Performance is critical
- Need self-contained binaries

### Choose **Plan Mode** when:
- Understanding complex workflows
- Debugging conditional logic
- Reviewing changes before execution
- Documenting execution plans
- Ensuring safe destructive operations

---

## Migration Between Modes

The same command definitions work across all modes, enabling smooth transitions:

### Development Workflow
```bash
# 1. Develop in interpreter mode
devcmd build
devcmd test

# 2. Verify with plan mode  
devcmd --dry-run deploy

# 3. Generate for production
devcmd build -f commands.cli --binary prodcli
```

### Debugging Workflow
```bash
# 1. Check execution plan
devcmd --dry-run problematic-command

# 2. Run in interpreter for debugging
devcmd --debug problematic-command

# 3. Generate final binary
devcmd build -f commands.cli --binary fixed-cli
```

This unified approach allows teams to use interpreter mode during development and switch to generated mode for production deployment, all from the same command definitions.