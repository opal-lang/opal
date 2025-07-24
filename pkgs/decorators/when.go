package decorators

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// WhenDecorator implements the @when decorator for conditional execution based on patterns
type WhenDecorator struct{}

// Template for when execution code generation
const whenExecutionTemplate = `func() error {
	// Pattern matching for variable: {{.VariableName}}
	value := os.Getenv({{printf "%q" .VariableName}})
	switch value {
	{{range $pattern := .Patterns}}
	{{if $pattern.IsDefault}}
	default:
	{{else}}
	case {{printf "%q" $pattern.Name}}:
	{{end}}
		// Execute commands for pattern: {{$pattern.Name}}
		if err := func() error {
			{{range $i, $cmd := $pattern.Commands}}
			if err := func() error {
				{{generateShellCode $cmd}}
			}(); err != nil {
				return err
			}
			{{end}}
			return nil
		}(); err != nil {
			return err
		}
	{{end}}
	}
	return nil
}()`

// WhenPatternData holds data for a single pattern branch
type WhenPatternData struct {
	Name      string
	IsDefault bool
	Commands  []ast.CommandContent
}

// WhenTemplateData holds data for template execution
type WhenTemplateData struct {
	VariableName string
	Patterns     []WhenPatternData
}

// Name returns the decorator name
func (w *WhenDecorator) Name() string {
	return "when"
}

// Description returns a human-readable description
func (w *WhenDecorator) Description() string {
	return "Conditionally execute commands based on pattern matching"
}

// ParameterSchema returns the expected parameters for this decorator
func (w *WhenDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "variable",
			Type:        ast.StringType,
			Required:    true,
			Description: "Variable name to match against",
		},
	}
}

// PatternSchema defines what patterns @when accepts
func (w *WhenDecorator) PatternSchema() PatternSchema {
	return PatternSchema{
		AllowedPatterns:     []string{}, // No specific patterns - any identifier is allowed
		RequiredPatterns:    []string{}, // No required patterns
		AllowsWildcard:      true,       // "default" wildcard is allowed
		AllowsAnyIdentifier: true,       // Any identifier is allowed (production, staging, etc.)
		Description:         "Accepts any identifier patterns and 'default' wildcard",
	}
}

// Validate checks if the decorator usage is correct during parsing

// Execute provides unified execution for all modes using the execution package
func (w *WhenDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Validate parameters first
	if len(params) == 0 {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("when decorator requires a 'variable' parameter"),
		}
	}
	if len(params) > 1 {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("when decorator accepts exactly 1 parameter (variable), got %d", len(params)),
		}
	}

	// Get the variable name to match against
	varName := ast.GetStringParam(params, "variable", "")
	if varName == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if varLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			varName = varLiteral.Value
		}
	}

	// Check that we got a valid variable name
	if varName == "" {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("when decorator requires a valid 'variable' parameter"),
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return w.executeInterpreter(ctx, varName, patterns)
	case execution.GeneratorMode:
		return w.executeGenerator(ctx, varName, patterns)
	case execution.PlanMode:
		return w.executePlan(ctx, varName, patterns)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes pattern matching in interpreter mode
func (w *WhenDecorator) executeInterpreter(ctx *execution.ExecutionContext, varName string, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Get the variable value (check context first, then environment variables)
	value := ""
	if ctxValue, exists := ctx.GetVariable(varName); exists {
		value = ctxValue
	} else {
		value = os.Getenv(varName)
	}

	// Find matching pattern branch
	for _, pattern := range patterns {
		if w.matchesPattern(value, pattern.Pattern) {
			// Execute the commands in the matching pattern
			if err := w.executeCommands(ctx, pattern.Commands); err != nil {
				return &execution.ExecutionResult{
					Mode:  execution.InterpreterMode,
					Data:  nil,
					Error: err,
				}
			}
			break
		}
	}

	// No pattern matched or execution succeeded
	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: nil,
	}
}

// executeGenerator generates Go code for pattern matching
func (w *WhenDecorator) executeGenerator(ctx *execution.ExecutionContext, varName string, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Convert patterns to template data
	var patternData []WhenPatternData
	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		isDefault := false
		if _, ok := pattern.Pattern.(*ast.WildcardPattern); ok {
			isDefault = true
		}

		patternData = append(patternData, WhenPatternData{
			Name:      patternStr,
			IsDefault: isDefault,
			Commands:  pattern.Commands,
		})
	}

	// Prepare template data
	templateData := WhenTemplateData{
		VariableName: varName,
		Patterns:     patternData,
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("when").Funcs(ctx.GetTemplateFunctions()).Parse(whenExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse when template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute when template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (w *WhenDecorator) executePlan(ctx *execution.ExecutionContext, varName string, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Get current value from context or environment
	currentValue := ""
	if value, exists := ctx.GetVariable(varName); exists {
		currentValue = value
	} else {
		currentValue = os.Getenv(varName)
	}

	// Find matching pattern
	selectedPattern := "default"
	var selectedCommands []ast.CommandContent

	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		if patternStr == currentValue {
			selectedPattern = patternStr
			selectedCommands = pattern.Commands
			break
		}
		if patternStr == "default" {
			selectedCommands = pattern.Commands
		}
	}

	description := fmt.Sprintf("Evaluate %s = %q → execute '%s' branch (%d commands)",
		varName, currentValue, selectedPattern, len(selectedCommands))

	element := plan.Decorator("when").
		WithType("pattern").
		WithParameter("variable", varName).
		WithDescription(description)

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// executeCommands executes commands using the unified execution engine
func (w *WhenDecorator) executeCommands(ctx *execution.ExecutionContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		if err := ctx.ExecuteCommandContent(cmd); err != nil {
			return err
		}
	}
	return nil
}

// matchesPattern checks if a value matches a pattern
func (w *WhenDecorator) matchesPattern(value string, pattern ast.Pattern) bool {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return value == p.Name
	case *ast.WildcardPattern:
		return true // Wildcard matches everything
	default:
		return false
	}
}

// patternToString converts a pattern to its string representation
func (w *WhenDecorator) patternToString(pattern ast.Pattern) string {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return p.Name
	case *ast.WildcardPattern:
		return "default"
	default:
		return "unknown"
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (w *WhenDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"os"}, // When decorator may need os for environment variables
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the when decorator
func init() {
	RegisterPattern(&WhenDecorator{})
}
