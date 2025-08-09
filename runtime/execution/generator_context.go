package execution

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// GeneratorExecutionContext implements GeneratorContext for Go code generation
type GeneratorExecutionContext struct {
	*BaseExecutionContext

	// Decorator lookup functions (set by engine during initialization)
	blockDecoratorLookup   func(name string) (interface{}, bool)
	patternDecoratorLookup func(name string) (interface{}, bool)
	valueDecoratorLookup   func(name string) (interface{}, bool)
	actionDecoratorLookup  func(name string) (interface{}, bool)

	// Environment variable tracking
	trackedEnvVars map[string]string
}

// ================================================================================================
// TEMPLATE-BASED CODE GENERATION
// ================================================================================================

// BuildCommandContent recursively generates template for command content
func (c *GeneratorExecutionContext) BuildCommandContent(commands []ast.CommandContent) (*TemplateResult, error) {
	// Create a composite template that includes all commands
	tmplStr := `{{range $i, $cmd := .Commands}}{{if $i}}
{{end}}{{$cmd | buildCommand}}{{end}}`

	tmpl, err := template.New("commands").Funcs(c.GetTemplateFunctions()).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commands template: %w", err)
	}

	// Prepare command data
	type CommandData struct {
		Type    string
		Content interface{}
	}

	var commandData []CommandData
	for _, cmd := range commands {
		switch content := cmd.(type) {
		case *ast.ShellContent:
			commandData = append(commandData, CommandData{
				Type:    "shell",
				Content: content,
			})
		case *ast.BlockDecorator:
			commandData = append(commandData, CommandData{
				Type:    "block",
				Content: content,
			})
		case *ast.PatternDecorator:
			commandData = append(commandData, CommandData{
				Type:    "pattern",
				Content: content,
			})
		default:
			return nil, fmt.Errorf("unsupported command content type: %T", cmd)
		}
	}

	return &TemplateResult{
		Template: tmpl,
		Data: struct {
			Commands []CommandData
		}{
			Commands: commandData,
		},
	}, nil
}

// ExecuteTemplate executes a template result and returns the generated code
func (c *GeneratorExecutionContext) ExecuteTemplate(result *TemplateResult) (string, error) {
	if result == nil || result.Template == nil {
		return "", fmt.Errorf("invalid template result")
	}

	var buf strings.Builder
	if err := result.Template.Execute(&buf, result.Data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// GetTemplateFunctions returns template functions for code generation
func (c *GeneratorExecutionContext) GetTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		// buildCommand processes individual commands - delegates to decorators or generates simple shell commands
		"buildCommand": func(cmd interface{}) string {
			// Handle CommandData wrapper using reflection
			var actualContent ast.CommandContent

			// Use reflection to access the CommandData fields
			cmdValue := reflect.ValueOf(cmd)
			if cmdValue.Kind() == reflect.Struct {
				contentField := cmdValue.FieldByName("Content")
				if contentField.IsValid() && !contentField.IsNil() {
					if astContent, ok := contentField.Interface().(ast.CommandContent); ok {
						actualContent = astContent
					} else {
						panic(fmt.Sprintf("Content field is not ast.CommandContent: %T", contentField.Interface()))
					}
				} else {
					panic("No Content field found in command data")
				}
			} else if astContent, ok := cmd.(ast.CommandContent); ok {
				actualContent = astContent
			} else {
				panic(fmt.Sprintf("Invalid command type: %T", cmd))
			}

			switch content := actualContent.(type) {
			case *ast.ShellContent:
				// Check if this shell content contains only a single ActionDecorator
				if len(content.Parts) == 1 {
					if actionDec, ok := content.Parts[0].(*ast.ActionDecorator); ok {
						// Handle standalone ActionDecorator - generate direct execution
						if c.actionDecoratorLookup != nil {
							if decoratorImpl, exists := c.actionDecoratorLookup(actionDec.Name); exists {
								if actionDecImpl, ok := decoratorImpl.(interface {
									GenerateTemplate(ctx GeneratorContext, params []ast.NamedParameter) (*TemplateResult, error)
								}); ok {
									if result, err := actionDecImpl.GenerateTemplate(c, actionDec.Args); err == nil {
										if code, err := c.ExecuteTemplate(result); err == nil {
											return `if err := ` + code + `; err != nil {
	return err
}`
										} else {
											panic("Error executing standalone action decorator template for @" + actionDec.Name + ": " + err.Error())
										}
									} else {
										panic("Error generating standalone action decorator template for @" + actionDec.Name + ": " + err.Error())
									}
								} else {
									panic("Standalone action decorator @" + actionDec.Name + " does not implement GenerateTemplate")
								}
							} else {
								panic("Unknown standalone action decorator: @" + actionDec.Name)
							}
						} else {
							panic("No action decorator lookup available for standalone @" + actionDec.Name)
						}
					}
				}

				// Process shell content with mixed text and value decorators
				var commandParts []string
				var sprintfArgs []string
				hasValueDecorators := false

				for _, part := range content.Parts {
					switch p := part.(type) {
					case *ast.TextPart:
						commandParts = append(commandParts, p.Text)
					case *ast.ValueDecorator:
						hasValueDecorators = true
						commandParts = append(commandParts, "%s") // Placeholder for value decorator

						// Delegate to value decorator's template generation
						if c.valueDecoratorLookup != nil {
							if decoratorImpl, exists := c.valueDecoratorLookup(p.Name); exists {
								if valueDec, ok := decoratorImpl.(interface {
									GenerateTemplate(ctx GeneratorContext, params []ast.NamedParameter) (*TemplateResult, error)
								}); ok {
									if result, err := valueDec.GenerateTemplate(c, p.Args); err == nil {
										if code, err := c.ExecuteTemplate(result); err == nil {
											sprintfArgs = append(sprintfArgs, code)
										} else {
											panic("Error executing value decorator template for @" + p.Name + ": " + err.Error())
										}
									} else {
										panic("Error generating value decorator template for @" + p.Name + ": " + err.Error())
									}
								} else {
									panic("Value decorator @" + p.Name + " does not implement GenerateTemplate")
								}
							} else {
								panic("Unknown value decorator: @" + p.Name)
							}
						} else {
							panic("No value decorator lookup available")
						}
					case *ast.ActionDecorator:
						// ActionDecorators in shell content should delegate to action decorators
						// This represents chained command execution like @cmd(build) && @cmd(test)
						hasValueDecorators = true
						commandParts = append(commandParts, "%s") // Placeholder for action decorator result

						// Look up the action decorator
						if c.actionDecoratorLookup != nil {
							if decoratorImpl, exists := c.actionDecoratorLookup(p.Name); exists {
								if actionDec, ok := decoratorImpl.(interface {
									GenerateTemplate(ctx GeneratorContext, params []ast.NamedParameter) (*TemplateResult, error)
								}); ok {
									if result, err := actionDec.GenerateTemplate(c, p.Args); err == nil {
										if code, err := c.ExecuteTemplate(result); err == nil {
											sprintfArgs = append(sprintfArgs, code)
										} else {
											panic("Error executing action decorator template for @" + p.Name + ": " + err.Error())
										}
									} else {
										panic("Error generating action decorator template for @" + p.Name + ": " + err.Error())
									}
								} else {
									panic("Action decorator @" + p.Name + " does not implement GenerateTemplate")
								}
							} else {
								panic("Unknown action decorator: @" + p.Name)
							}
						} else {
							panic("No action decorator lookup available for @" + p.Name)
						}
					default:
						panic(fmt.Sprintf("Unsupported shell part type: %T", p))
					}
				}

				if hasValueDecorators {
					// Build fmt.Sprintf call with format string and arguments
					formatString := strings.Join(commandParts, "")
					allArgs := []string{fmt.Sprintf("%q", formatString)}
					allArgs = append(allArgs, sprintfArgs...)
					commandExpr := fmt.Sprintf("fmt.Sprintf(%s)", strings.Join(allArgs, ", "))

					return `if err := exec(ctx, ` + commandExpr + `); err != nil {
	return err
}`
				} else {
					// Simple case: no value decorators, just text
					commandString := strings.Join(commandParts, "")
					return `if err := exec(ctx, ` + fmt.Sprintf("%q", commandString) + `); err != nil {
	return err
}`
				}
			case *ast.BlockDecorator:
				// For block decorators, we delegate to their GenerateTemplate method
				// The template execution happens at the parent level
				if c.blockDecoratorLookup != nil {
					if decoratorImpl, exists := c.blockDecoratorLookup(content.Name); exists {
						if blockDec, ok := decoratorImpl.(interface {
							GenerateTemplate(ctx GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) (*TemplateResult, error)
						}); ok {
							if result, err := blockDec.GenerateTemplate(c, content.Args, content.Content); err == nil {
								if code, err := c.ExecuteTemplate(result); err == nil {
									return code
								}
							}
						}
					}
				}
				panic("Unknown block decorator: @" + content.Name)
			case *ast.PatternDecorator:
				// For pattern decorators, we delegate to their GenerateTemplate method
				if c.patternDecoratorLookup != nil {
					if decoratorImpl, exists := c.patternDecoratorLookup(content.Name); exists {
						if patternDec, ok := decoratorImpl.(interface {
							GenerateTemplate(ctx GeneratorContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (*TemplateResult, error)
						}); ok {
							if result, err := patternDec.GenerateTemplate(c, content.Args, content.Patterns); err == nil {
								if code, err := c.ExecuteTemplate(result); err == nil {
									return code
								}
							}
						}
					}
				}
				panic("Unknown pattern decorator: @" + content.Name)
			default:
				panic(fmt.Sprintf("Unsupported command type: %T", actualContent))
			}
		},

		// Helper functions for template data processing
		"buildCommands": func(commands []ast.CommandContent) string {
			// Recursively build multiple commands using the same template system
			result, err := c.BuildCommandContent(commands)
			if err != nil {
				panic("Error building command content: " + err.Error())
			}
			code, err := c.ExecuteTemplate(result)
			if err != nil {
				panic("Template execution error: " + err.Error())
			}
			return code
		},

		// Duration formatting for clean Go code generation
		"formatDuration": func(d time.Duration) string {
			// Generate idiomatic Go duration expressions
			switch {
			case d == 0:
				return "0"
			case d%time.Hour == 0 && d >= time.Hour:
				return fmt.Sprintf("%d * time.Hour", d/time.Hour)
			case d%time.Minute == 0 && d >= time.Minute:
				return fmt.Sprintf("%d * time.Minute", d/time.Minute)
			case d%time.Second == 0 && d >= time.Second:
				return fmt.Sprintf("%d * time.Second", d/time.Second)
			case d%time.Millisecond == 0 && d >= time.Millisecond:
				return fmt.Sprintf("%d * time.Millisecond", d/time.Millisecond)
			case d%time.Microsecond == 0 && d >= time.Microsecond:
				return fmt.Sprintf("%d * time.Microsecond", d/time.Microsecond)
			default:
				return fmt.Sprintf("%d * time.Nanosecond", d.Nanoseconds())
			}
		},
	}
}

// ================================================================================================
// INTERNAL ACCESS METHODS FOR TEMPLATE GENERATION
// ================================================================================================

// GetShellCounter returns the current shell counter for unique variable naming
func (c *GeneratorExecutionContext) GetShellCounter() int {
	return c.shellCounter
}

// IncrementShellCounter increments the shell counter for unique variable naming
func (c *GeneratorExecutionContext) IncrementShellCounter() {
	c.shellCounter++
}

// GetCurrentCommand returns the current command name for variable generation
func (c *GeneratorExecutionContext) GetCurrentCommand() string {
	return c.currentCommand
}

// GetBlockDecoratorLookup returns the block decorator lookup function
func (c *GeneratorExecutionContext) GetBlockDecoratorLookup() func(name string) (interface{}, bool) {
	// Block decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.blockDecoratorLookup
}

// GetPatternDecoratorLookup returns the pattern decorator lookup function
func (c *GeneratorExecutionContext) GetPatternDecoratorLookup() func(name string) (interface{}, bool) {
	// Pattern decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.patternDecoratorLookup
}

// GetValueDecoratorLookup returns the value decorator lookup function
func (c *GeneratorExecutionContext) GetValueDecoratorLookup() func(name string) (interface{}, bool) {
	// Value decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.valueDecoratorLookup
}

// GetActionDecoratorLookup returns the action decorator lookup function
func (c *GeneratorExecutionContext) GetActionDecoratorLookup() func(name string) (interface{}, bool) {
	// Action decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.actionDecoratorLookup
}

// SetBlockDecoratorLookup sets the block decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetBlockDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.blockDecoratorLookup = lookup
}

// SetPatternDecoratorLookup sets the pattern decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetPatternDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.patternDecoratorLookup = lookup
}

// SetValueDecoratorLookup sets the value decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetValueDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.valueDecoratorLookup = lookup
}

// SetActionDecoratorLookup sets the action decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetActionDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.actionDecoratorLookup = lookup
}

// TrackEnvironmentVariableReference tracks which env vars are referenced for code generation
func (c *GeneratorExecutionContext) TrackEnvironmentVariableReference(key, defaultValue string) {
	// For now, generator context doesn't store these - they're handled by decorators
	// This method exists to satisfy calls from builtin decorators
}

// GetTrackedEnvironmentVariableReferences returns env var references for template generation
func (c *GeneratorExecutionContext) GetTrackedEnvironmentVariableReferences() map[string]string {
	// For now, return empty - the actual env var tracking happens in the engine
	// via decorator calls during code generation
	return make(map[string]string)
}

// ================================================================================================
// CONTEXT MANAGEMENT WITH TYPE SAFETY
// ================================================================================================

// Child creates a child generator context that inherits from the parent but can be modified independently
func (c *GeneratorExecutionContext) Child() GeneratorContext {
	// Increment child counter to ensure unique variable naming across parallel contexts
	c.childCounter++
	childID := c.childCounter

	childBase := &BaseExecutionContext{
		Context:   c.Context,
		Program:   c.Program,
		Variables: make(map[string]string),
		env:       c.env, // Share the same immutable environment reference

		// Copy execution state
		WorkingDir:     c.WorkingDir,
		Debug:          c.Debug,
		DryRun:         c.DryRun,
		currentCommand: c.currentCommand,

		// Initialize unique counter space for this child to avoid variable name conflicts
		// Each child gets a unique counter space based on parent's counter and child ID
		shellCounter: c.shellCounter + (childID * 1000), // Give each child 1000 numbers of space
		childCounter: 0,                                 // Reset child counter for this context's children
	}

	// Copy variables (child gets its own copy)
	for name, value := range c.Variables {
		childBase.Variables[name] = value
	}

	return &GeneratorExecutionContext{
		BaseExecutionContext: childBase,
		// Copy immutable configuration from parent to child
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
		actionDecoratorLookup:  c.actionDecoratorLookup,
		// Copy env var tracking from parent
		trackedEnvVars: c.trackedEnvVars,
	}
}

// WithTimeout creates a new generator context with timeout
func (c *GeneratorExecutionContext) WithTimeout(timeout time.Duration) (GeneratorContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
		actionDecoratorLookup:  c.actionDecoratorLookup,
	}, cancel
}

// WithCancel creates a new generator context with cancellation
func (c *GeneratorExecutionContext) WithCancel() (GeneratorContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
		actionDecoratorLookup:  c.actionDecoratorLookup,
	}, cancel
}

// WithWorkingDir creates a new generator context with the specified working directory
func (c *GeneratorExecutionContext) WithWorkingDir(workingDir string) GeneratorContext {
	newBase := *c.BaseExecutionContext
	newBase.WorkingDir = workingDir
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
		actionDecoratorLookup:  c.actionDecoratorLookup,
	}
}

// WithCurrentCommand creates a new generator context with the specified current command name
func (c *GeneratorExecutionContext) WithCurrentCommand(commandName string) GeneratorContext {
	newBase := *c.BaseExecutionContext
	newBase.currentCommand = commandName
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
		actionDecoratorLookup:  c.actionDecoratorLookup,
		// Copy env var tracking from parent
		trackedEnvVars: c.trackedEnvVars,
	}
}
