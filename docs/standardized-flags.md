# Standardized Flags Documentation

**devcmd** provides a consistent set of global flags that work across all execution modes and affect decorator behavior uniformly.

## Overview

The standardized flags ensure consistent user experience whether you're using:
- **Interpreter mode**: `devcmd run <command>`
- **Generated mode**: `./mycli <command>` 
- **Plan mode**: `devcmd run <command> --dry-run` or `./mycli <command> --dry-run`

## Available Flags

### Output Control

#### `--color <mode>` 
Controls colored output and terminal UI features.

**Values**: `auto` (default), `always`, `never`
- `auto` - Use color/UI if stdout is a TTY
- `always` - Force color/UI even when piped
- `never` - Disable all color/UI features

**Shorthand**: `--no-color` (equivalent to `--color=never`)

**Examples**:
```bash
devcmd run deploy --color=always    # Force colors in CI
./mycli build --no-color            # Disable all colors
devcmd run test --dry-run --color=auto  # Auto-detect (default)
```

#### `--quiet`, `-q`
Minimal output mode - only show errors and final results.

**Effect on decorators**:
- `@parallel` - Disables terminal UI, shows simple output
- `@confirm` - Uses default answers (requires `--yes`)
- `@timeout` - Suppresses progress indicators
- Shell commands - Stderr only

**Examples**:
```bash
devcmd run build --quiet            # Minimal build output
./mycli deploy --quiet --yes        # Silent deployment
```

#### `--verbose`, `-v`
Extra debugging output and detailed progress information.

**Effect on decorators**:
- `@parallel` - Shows detailed task information
- `@timeout` - Shows countdown timers
- `@workdir` - Shows directory changes
- Shell commands - Full debug output

**Examples**:
```bash
devcmd run test --verbose           # Detailed test output
./mycli build --verbose             # Debug build process
```

**Note**: Cannot be used with `--quiet` (validation error).

### Interaction Control

#### `--interactive <mode>`
Controls interactive prompts and user input.

**Values**: `auto` (default), `always`, `never`
- `auto` - Interactive if stdin is a TTY
- `always` - Force interactive mode
- `never` - Disable all user interaction

**Effect on decorators**:
- `@confirm` - Controls prompt behavior
- `@parallel` - Affects error handling prompts
- Any decorator requiring user input

**Examples**:
```bash
devcmd run deploy --interactive=never    # No prompts in CI
./mycli setup --interactive=always      # Force prompts
```

#### `--yes`
Auto-confirm all prompts with "yes" response.

**Effect on decorators**:
- `@confirm` - Automatically confirms all prompts
- Pattern decorators - Uses default branches
- Error handling - Continues on recoverable errors

**Examples**:
```bash
devcmd run cleanup --yes            # Auto-confirm dangerous operations
./mycli deploy --yes --quiet        # Silent auto-confirmed deployment
```

### Environment Control

#### `--ci`
CI/Automation mode - optimized for non-interactive environments.

**Automatically sets**:
- `--interactive=never`
- `--color=never` 
- `--quiet=true`

**Effect on decorators**:
- `@parallel` - Uses simple output format
- `@confirm` - Requires `--yes` or fails
- `@timeout` - No progress indicators
- All decorators - Assume automation context

**Examples**:
```bash
devcmd run test --ci                # CI-optimized test run
./mycli deploy --ci --yes           # Automated deployment
```

#### `--resolve`
Forces complete resolution of all value decorators during plan generation.

**Only valid with `--dry-run`** - Creates resolved plans with frozen execution context.

**Effect on value decorators**:
- `@var` - Shows resolved variable values
- `@env` - Shows resolved environment values (including defaults)
- `@http` - Forces HTTP calls to resolve values
- All value decorators - Creates deterministic execution context

**Use cases**:
- Production deployment planning
- Creating executable plan files
- Debugging value resolution issues
- CI/CD pipelines requiring deterministic plans

**Examples**:
```bash
devcmd run deploy --dry-run --resolve        # Fully resolved plan
devcmd plan build --output build.plan        # Save resolved plan file
devcmd exec build.plan                       # Execute from resolved plan
```

## Execution Mode Behavior

### Interpreter Mode (`devcmd run`)

All flags work as documented. The flags are processed by the devcmd CLI and passed to the execution context.

```bash
devcmd run build --verbose --color=always
devcmd run deploy --ci --yes
devcmd run test --quiet
```

### Generated Mode (`./mycli`)

Generated CLIs inherit the same flag behavior. The flags are baked into the generated code with identical semantics.

```bash
./mycli build --verbose --color=always
./mycli deploy --ci --yes  
./mycli test --quiet
```

### Plan Mode (`--dry-run`)

Plan mode shows what *would* be executed without running commands. devcmd supports two plan generation modes:

#### Quick Plans (Default `--dry-run`)
Fast preview showing cached values and runtime placeholders for expensive operations.

#### Resolved Plans (`--dry-run --resolve`)
Complete resolution of all value decorators with frozen execution context.

**Flag behavior in plan mode**:
- `--color` - Controls plan output formatting and value decorator marker colors
- `--quiet` - Shows minimal plan structure
- `--verbose` - Shows detailed execution plan with resolution metadata
- `--resolve` - Forces resolution of all value decorators (expensive operations)
- `--interactive` - **Ignored** (no actual interaction)
- `--yes` - **Ignored** (no actual prompts)
- `--ci` - Affects plan formatting only

**Examples**:
```bash
devcmd run deploy --dry-run --verbose         # Detailed quick plan
./mycli build --dry-run --quiet              # Minimal quick plan
devcmd run deploy --dry-run --resolve        # Fully resolved plan
devcmd plan deploy --output deploy.plan      # Save resolved plan to file
```

## Decorator Implementation Guidelines

### Accessing Flag Values

Decorators access standardized flags through the typed UI configuration:

```go
func (d *MyDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
    // Check UI configuration (typed access)
    if ctx.UI == nil {
        // Use defaults if no UI config provided
        return d.executeWithDefaults(ctx, args, inner)
    }
    
    // Access typed UI configuration
    ui := ctx.UI
    
    // Adapt behavior based on flags
    if ui.Quiet {
        // Minimal output mode
    } else if ui.Verbose {
        // Detailed output mode  
    }
    
    if ui.CI || ui.Interactive == "never" {
        // Non-interactive mode
    }
    
    // Use color/UI only if appropriate
    useUI := (ui.ColorMode == "always") || 
             (ui.ColorMode == "auto" && isTerminal(ctx.Stdout)) &&
             !ui.Quiet && !ui.CI
}
```

### Flag Precedence Rules

1. **`--ci`** overrides all other UI settings
2. **`--quiet`** disables terminal UI features
3. **`--no-color`** sets `--color=never`
4. **`--yes`** implies non-interactive for prompts
5. **Conflicting flags** cause validation errors (`--quiet` + `--verbose`)

### Recommended Patterns

#### Terminal UI Decision
```go
func shouldUseTerminalUI(ctx *decorators.Ctx) bool {
    if ctx.UI == nil {
        return isTerminal(ctx.Stdout) // Default behavior
    }
    
    ui := ctx.UI
    if ui.CI || ui.Quiet {
        return false
    }
    
    switch ui.ColorMode {
    case "never":
        return false
    case "always":
        return true
    case "auto", "":
        return isTerminal(ctx.Stdout)
    default:
        return isTerminal(ctx.Stdout)
    }
}
```

#### Prompt Handling
```go
func shouldPrompt(ctx *decorators.Ctx) bool {
    if ctx.UI == nil {
        return isTerminal(ctx.Stdin) // Default behavior
    }
    
    ui := ctx.UI
    if ui.CI || ui.AutoConfirm {
        return false
    }
    
    switch ui.Interactive {
    case "never":
        return false
    case "always":
        return true
    case "auto", "":
        return isTerminal(ctx.Stdin)
    default:
        return isTerminal(ctx.Stdin)
    }
}
```

## Environment Variable Fallbacks

For generated CLIs, environment variables provide fallback configuration:

| Flag | Environment Variable | Example |
|------|---------------------|---------|
| `--color` | `DEVCMD_COLOR` | `DEVCMD_COLOR=never` |
| `--quiet` | `DEVCMD_QUIET` | `DEVCMD_QUIET=1` |
| `--verbose` | `DEVCMD_VERBOSE` | `DEVCMD_VERBOSE=1` |
| `--interactive` | `DEVCMD_INTERACTIVE` | `DEVCMD_INTERACTIVE=never` |
| `--yes` | `DEVCMD_YES` | `DEVCMD_YES=1` |
| `--ci` | `DEVCMD_CI` or `CI` | `CI=1` |

**Precedence**: CLI flags > Environment variables > Defaults

## Examples by Use Case

### Development Workflow
```bash
# Interactive development
devcmd run test --verbose               # Detailed output
devcmd run build --color=always         # Force colors

# Quick checks
devcmd run lint --quiet                 # Minimal output
devcmd run format --quiet --yes         # Silent formatting
```

### CI/CD Pipeline
```bash
# CI-optimized
./mycli test --ci                       # Minimal, no colors, no interaction
./mycli deploy --ci --yes               # Automated deployment

# Or with environment
export CI=1
export DEVCMD_YES=1
./mycli deploy                          # Uses environment defaults
```

### Debugging and Troubleshooting
```bash
# Maximum detail
devcmd run deploy --verbose --interactive=always

# Planned execution
devcmd run deploy --dry-run --verbose   # See what would happen
```

### Generated CLI Integration
```bash
# In docker containers
./mycli build --no-color --quiet        # Container-friendly

# In automation scripts  
./mycli deploy --ci --yes 2>/dev/null   # Fully silent automation
```

## Flag Behavior with Shell Operations

**Standardized flags affect both engine-level and shell-level operations**:
- `--verbose` shows both engine operations (`&&`, `||`, `>>`) and shell command details
- `--dry-run` displays complete execution plans including shell-specific syntax  
- `--ci` optimizes output for both engine coordination and shell execution
- `--quiet` suppresses output from both engine operations and shell commands

This standardized approach ensures predictable behavior across all execution modes while maintaining full control over user experience.