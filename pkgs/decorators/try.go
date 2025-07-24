package decorators

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// TryDecorator implements the @try decorator for error handling with pattern matching
type TryDecorator struct{}

// Template for try execution code generation
const tryExecutionTemplate = `return func() error {
	var mainErr error

	// Execute main block
	mainErr = func() error {
		{{range $i, $cmd := .MainCommands}}
		if err := func() error {
			{{generateShellCode $cmd}}
		}(); err != nil {
			return err
		}
		{{end}}
		return nil
	}()

	{{if .HasErrorBranch}}
	// Execute error block if main failed
	if mainErr != nil {
		errorErr := func() error {
			{{range $i, $cmd := .ErrorCommands}}
			if err := func() error {
				{{generateShellCode $cmd}}
			}(); err != nil {
				return err
			}
			{{end}}
			return nil
		}()
		if errorErr != nil {
			fmt.Printf("Error handler also failed: %v\n", errorErr)
		}
	}
	{{end}}

	{{if .HasFinallyBranch}}
	// Always execute finally block
	finallyErr := func() error {
		{{range $i, $cmd := .FinallyCommands}}
		if err := func() error {
			{{generateShellCode $cmd}}
		}(); err != nil {
			return err
		}
		{{end}}
		return nil
	}()
	if finallyErr != nil {
		fmt.Printf("Finally block failed: %v\n", finallyErr)
	}
	{{end}}

	// Return the original main error
	return mainErr
}()`

// TryTemplateData holds data for template execution
type TryTemplateData struct {
	MainCommands     []ast.CommandContent
	ErrorCommands    []ast.CommandContent
	FinallyCommands  []ast.CommandContent
	HasErrorBranch   bool
	HasFinallyBranch bool
}

// Name returns the decorator name
func (t *TryDecorator) Name() string {
	return "try"
}

// Description returns a human-readable description
func (t *TryDecorator) Description() string {
	return "Execute commands with error handling via pattern matching (main required, error/finally optional but at least one required)"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TryDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{} // @try takes no parameters
}

// PatternSchema defines what patterns @try accepts
func (t *TryDecorator) PatternSchema() PatternSchema {
	return PatternSchema{
		AllowedPatterns:     []string{"main", "error", "finally"},
		RequiredPatterns:    []string{"main"},
		AllowsWildcard:      false, // No "default" wildcard for @try
		AllowsAnyIdentifier: false, // Only specific patterns allowed
		Description:         "Requires 'main', optionally accepts 'error' and 'finally'",
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (t *TryDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Validate parameters first
	if len(params) > 0 {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("try decorator takes no parameters, got %d", len(params)),
		}
	}

	// Find pattern branches
	var mainBranch, errorBranch, finallyBranch *ast.PatternBranch

	for i := range patterns {
		pattern := &patterns[i]
		patternStr := t.patternToString(pattern.Pattern)

		switch patternStr {
		case "main":
			mainBranch = pattern
		case "error":
			errorBranch = pattern
		case "finally":
			finallyBranch = pattern
		default:
			return &execution.ExecutionResult{
				Mode:  ctx.Mode(),
				Data:  nil,
				Error: fmt.Errorf("@try only supports 'main', 'error', and 'finally' patterns, got '%s'", patternStr),
			}
		}
	}

	// Validate required patterns
	if mainBranch == nil {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@try requires a 'main' pattern"),
		}
	}
	if errorBranch == nil && finallyBranch == nil {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@try requires at least one of 'error' or 'finally' patterns"),
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return t.executeInterpreter(ctx, mainBranch, errorBranch, finallyBranch)
	case execution.GeneratorMode:
		return t.executeGenerator(ctx, mainBranch, errorBranch, finallyBranch)
	case execution.PlanMode:
		return t.executePlan(ctx, mainBranch, errorBranch, finallyBranch)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes try-catch patterns in interpreter mode
func (t *TryDecorator) executeInterpreter(ctx *execution.ExecutionContext, mainBranch, errorBranch, finallyBranch *ast.PatternBranch) *execution.ExecutionResult {
	// Execute main block
	mainErr := t.executeCommands(ctx, mainBranch.Commands)

	// Execute error block if main failed and error pattern exists
	if mainErr != nil && errorBranch != nil {
		// If error handler also fails, we still want to run finally
		_ = t.executeCommands(ctx, errorBranch.Commands)
	}

	// Always execute finally block if it exists
	if finallyBranch != nil {
		// Finally block errors don't override main error
		_ = t.executeCommands(ctx, finallyBranch.Commands)
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: mainErr, // Return the original main error (if any)
	}
}

// executeGenerator generates Go code for try-catch logic
func (t *TryDecorator) executeGenerator(ctx *execution.ExecutionContext, mainBranch, errorBranch, finallyBranch *ast.PatternBranch) *execution.ExecutionResult {
	// Prepare template data
	templateData := TryTemplateData{
		MainCommands:     mainBranch.Commands,
		HasErrorBranch:   errorBranch != nil,
		HasFinallyBranch: finallyBranch != nil,
	}

	if errorBranch != nil {
		templateData.ErrorCommands = errorBranch.Commands
	}

	if finallyBranch != nil {
		templateData.FinallyCommands = finallyBranch.Commands
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("try").Funcs(ctx.GetTemplateFunctions()).Parse(tryExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse try template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute try template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (t *TryDecorator) executePlan(ctx *execution.ExecutionContext, mainBranch, errorBranch, finallyBranch *ast.PatternBranch) *execution.ExecutionResult {
	description := "Try-catch execution: "
	if mainBranch != nil {
		description += fmt.Sprintf("execute main (%d commands)", len(mainBranch.Commands))
	}
	if errorBranch != nil {
		description += fmt.Sprintf(", on error execute fallback (%d commands)", len(errorBranch.Commands))
	}
	if finallyBranch != nil {
		description += fmt.Sprintf(", always execute finally (%d commands)", len(finallyBranch.Commands))
	}

	element := plan.Decorator("try").
		WithType("pattern").
		WithDescription(description)

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// executeCommands executes commands using the unified execution engine
func (t *TryDecorator) executeCommands(ctx *execution.ExecutionContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		if err := ctx.ExecuteCommandContent(cmd); err != nil {
			return err
		}
	}
	return nil
}

// patternToString converts a pattern to its string representation
func (t *TryDecorator) patternToString(pattern ast.Pattern) string {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return p.Name
	default:
		return "unknown"
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (t *TryDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"fmt"}, // Try decorator needs fmt for error handling
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the try decorator
func init() {
	RegisterPattern(&TryDecorator{})
}
