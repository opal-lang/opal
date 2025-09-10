# Decorator Plan Display Guidelines

This document outlines how decorators should display in execution plans and provide expansion hints for complex decorators.

## Overview

When devcmd runs in dry-run mode, it generates an execution plan that shows users exactly what would be executed. Decorators play a crucial role in making this plan readable and informative.

## Universal Expansion System

Complex decorators that reference external resources (commands, templates, etc.) use a hint-based expansion system:

### Expansion Hints

Decorators provide expansion hints via standard `ExecutionStep.Metadata` keys:

- `"expansion_type"`: Type of expansion needed (`"command_reference"`, `"template_include"`, etc.)
- `"command_name"`: For `command_reference` expansion
- `"template_path"`: For `template_include` expansion  
- `"module_name"`: For `module_import` expansion
- `"include_path"`: For `file_include` expansion

### How It Works

1. **Decorator provides hints**: Returns ExecutionStep with expansion metadata
2. **Generator reads hints**: Uses metadata to determine what expansion is needed
3. **Generator resolves resources**: Uses appropriate resolver to fetch referenced resources
4. **Generator builds structure**: Creates expanded ExecutionStep tree from resolved resources

### Example: @cmd Decorator

```go
return plan.ExecutionStep{
    Type:        plan.StepSequence,
    Description: fmt.Sprintf("From '%s' command:", cmdName),
    Children:    []plan.ExecutionStep{}, // Generator populates this
    Metadata: map[string]string{
        "expansion_type": "command_reference",
        "command_name":   cmdName,
    },
}
```

The generator sees this hint and expands the command's actual structure as children.

## Display Patterns

### 1. Basic Action Decorators

Simple action decorators should show what they do:

```
my-command:
└─ Execute 1 command steps
   └─ @log("Starting build")
```

**Implementation**: Return a `StepShell` with a clear description of the action.

### 2. Block Decorators

Block decorators wrap other commands and should show their configuration:

```
timeout-example:
└─ Execute 1 command steps
   └─ @timeout {30s timeout}
      └─ Inner commands
         └─ echo test
```

**Key points**:
- Show the decorator name with `@` prefix
- Include configuration in `{braces}` after the decorator name
- Nest inner commands under "Inner commands"

### 3. Command References (@cmd)

When referencing other commands, use the "From 'commandName' command:" pattern:

```
deploy:
└─ Execute 1 command steps
   └─ From 'build' command:
      ├─ npm run compile
      ├─ npm run test
      └─ npm run package
```

**Key points**:
- Use "From 'commandName' command:" as the description
- Show the actual steps from the referenced command as children
- Preserve nesting when commands reference other commands

### 4. Pattern Decorators (Conditional)

Pattern decorators should show evaluation context:

```
conditional-example:
└─ Execute 1 command steps
   └─ @when {BUILD_TYPE = release → production}
      └─ Inner commands
         └─ deploy to production
```

## Implementation Guidelines

### For Action Decorators

```go
func (d *MyActionDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam) plan.ExecutionStep {
    return plan.ExecutionStep{
        Type:        plan.StepShell,
        Description: "@myaction(param)", // Show what the decorator does
        Command:     "@myaction(param)", // Keep consistent
        Metadata: map[string]string{
            "decorator": "myaction",
            "type":      "action",
        },
    }
}
```

### For Block Decorators

```go
func (d *MyBlockDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
    // Extract configuration
    timeout := extractTimeout(args)
    
    return plan.ExecutionStep{
        Type:        plan.StepTimeout, // Use specific step type
        Description: fmt.Sprintf("@timeout {%s timeout}", timeout),
        Children:    []plan.ExecutionStep{inner}, // Include inner commands
        Timing: &plan.TimingInfo{
            Timeout: &timeout,
        },
        Metadata: map[string]string{
            "decorator": "timeout",
            "type":      "block",
        },
    }
}
```

### For Command References

Use the special "needs_expansion" pattern for commands that should be expanded:

```go
func (d *CmdDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam) plan.ExecutionStep {
    cmdName := extractCommandName(args)
    
    return plan.ExecutionStep{
        Type:        plan.StepSequence,
        Description: fmt.Sprintf("From '%s' command:", cmdName),
        Children:    []plan.ExecutionStep{}, // Will be populated by plan generator
        Metadata: map[string]string{
            "decorator":        "cmd",
            "command":          cmdName,
            "type":             "command_reference",
            "needs_expansion":  "true", // Signals plan generator to expand
        },
    }
}
```

## Tree Structure Rules

1. **Root Level**: Command name followed by colon
2. **Main Step**: "Execute N command steps" (groups all top-level steps)
3. **Decorators**: Use `@decoratorName` prefix with configuration in `{braces}`
4. **Nesting**: Use proper tree characters (`└─`, `├─`) for hierarchy
5. **Command Boundaries**: Use "From 'commandName' command:" for command references

## Best Practices

1. **Be Descriptive**: Show what will happen, not just the decorator name
2. **Include Configuration**: Show important parameters in `{braces}`
3. **Preserve Hierarchy**: Maintain clear nesting for complex scenarios
4. **Handle Errors Gracefully**: Show meaningful error messages for invalid configurations
5. **Use Appropriate Step Types**: Use `StepTimeout`, `StepParallel`, etc. for better formatting

## Examples

### Complex Nested Example

```
deploy-app:
└─ Execute 1 command steps
   └─ @timeout {60s timeout}
      └─ Inner commands
         └─ @parallel {2 concurrent}
            └─ Inner commands
               ├─ From 'build-frontend' command:
               │  ├─ npm run build
               │  └─ npm run optimize
               └─ From 'build-backend' command:
                  ├─ go build ./cmd/server
                  └─ docker build -t myapp .
```

This shows:
- Clear command boundaries with "From 'commandName' command:"
- Proper nesting with decorators maintaining hierarchy
- Configuration information for timeout and parallel decorators
- Expanded command content showing actual steps

## Migration Notes

If you have existing decorators that don't follow these patterns:

1. Update `Describe` methods to use the new display format
2. Add proper metadata for plan generator integration
3. Use appropriate `StepType` constants for better formatting
4. Test plan output to ensure readability

This standardization ensures users get consistent, clear visibility into what their commands will execute.