package decorators

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// WorkdirDecorator implements the @workdir decorator for changing working directory
type WorkdirDecorator struct{}

// Name returns the decorator name
func (d *WorkdirDecorator) Name() string {
	return "workdir"
}

// Description returns a human-readable description
func (d *WorkdirDecorator) Description() string {
	return "Changes working directory for the duration of the block, then restores original directory"
}

// ParameterSchema returns the expected parameters
func (d *WorkdirDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "path",
			Type:        ast.StringType,
			Required:    true,
			Description: "Directory path to change to",
		},
		{
			Name:        "createIfNotExists",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Create directory if it doesn't exist (default: false)",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (d *WorkdirDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.RequiresFileSystem() // Uses ResourceCleanupPattern + os operations
}

// ExecuteInterpreter executes workdir in interpreter mode
func (d *WorkdirDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	pathParam, createIfNotExists, err := d.extractWorkdirParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("workdir parameter error: %w", err),
		}
	}

	return d.executeInterpreterImpl(ctx, pathParam, createIfNotExists, content)
}

// GenerateTemplate generates template for workdir logic
func (d *WorkdirDecorator) GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) (*execution.TemplateResult, error) {
	pathParam, createIfNotExists, err := d.extractWorkdirParams(params)
	if err != nil {
		return nil, fmt.Errorf("workdir parameter error: %w", err)
	}

	return d.generateTemplateImpl(ctx, pathParam, createIfNotExists, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (d *WorkdirDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	pathParam, createIfNotExists, err := d.extractWorkdirParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("workdir parameter error: %w", err),
		}
	}

	return d.executePlanImpl(pathParam, createIfNotExists, content)
}

// extractWorkdirParams extracts and validates workdir parameters
func (d *WorkdirDecorator) extractWorkdirParams(params []ast.NamedParameter) (string, bool, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 1, 2, "workdir"); err != nil {
		return "", false, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, d.ParameterSchema(), "workdir"); err != nil {
		return "", false, err
	}

	// Enhanced security validation for path safety (no directory traversal, etc.)
	if err := decorators.ValidatePathSafety(params, "path", "workdir"); err != nil {
		return "", false, err
	}

	// Perform comprehensive security validation for all parameters
	_, err := decorators.PerformComprehensiveSecurityValidation(params, d.ParameterSchema(), "workdir")
	if err != nil {
		return "", false, err
	}

	// Parse parameters (validation passed, so these should be safe)
	path := ast.GetStringParam(params, "path", "")
	createIfNotExists := ast.GetBoolParam(params, "createIfNotExists", false)

	return path, createIfNotExists, nil
}

// getPathParameter extracts and validates the path parameter (deprecated - use extractWorkdirParams)

// executePlanImpl creates a plan element for dry-run display
func (d *WorkdirDecorator) executePlanImpl(path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
	description := fmt.Sprintf("@workdir(\"%s\")", path)
	if createIfNotExists {
		description += " (create if needed)"
	}

	element := plan.Decorator("workdir").
		WithType("block").
		WithParameter("path", path).
		WithDescription(description)

	if createIfNotExists {
		element = element.WithParameter("createIfNotExists", "true")
	}

	// Add children for each content item to show nested structure
	for _, cmdContent := range content {
		switch c := cmdContent.(type) {
		case *ast.ShellContent:
			// Convert shell content to command element
			if len(c.Parts) > 0 {
				if text, ok := c.Parts[0].(*ast.TextPart); ok {
					cmd := strings.TrimSpace(text.Text)
					element.AddChild(plan.Command(cmd).WithDescription(cmd))
				}
			}
		case *ast.BlockDecorator:
			// For nested decorators, create a placeholder (the actual decorator will be processed separately)
			element.AddChild(plan.Command(fmt.Sprintf("@%s", c.Name)).WithDescription(fmt.Sprintf("@%s decorator", c.Name)))
		}
	}

	return &execution.ExecutionResult{
		Data:  element,
		Error: nil,
	}
}

// executeInterpreterImpl executes the workdir in interpreter mode using utilities
func (d *WorkdirDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Handle directory creation or verification
	if createIfNotExists {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(path, 0o755); err != nil {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("failed to create directory %s: %w", path, err),
			}
		}
	} else {
		// Verify the target directory exists before proceeding
		if _, err := os.Stat(path); err != nil {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("failed to access directory %s: %w", path, err),
			}
		}
	}

	// Create a new context with the updated working directory
	// This ensures isolated execution without affecting global process directory
	workdirCtx := ctx.WithWorkingDir(path)

	// Use CommandExecutor utility to handle command execution
	commandExecutor := decorators.NewCommandExecutor()
	defer commandExecutor.Cleanup()

	// Execute all commands in the workdir context
	err := commandExecutor.ExecuteCommandsWithInterpreter(workdirCtx, content)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("execution failed in directory %s: %w", path, err),
		}
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: nil,
	}
}

// generateTemplateImpl generates template for the workdir decorator
func (d *WorkdirDecorator) generateTemplateImpl(ctx execution.GeneratorContext, path string, createIfNotExists bool, content []ast.CommandContent) (*execution.TemplateResult, error) {
	// Create template string with workdir logic
	tmplStr := `{{if .CreateIfNotExists}}// Create directory if it doesn't exist
if err := os.MkdirAll({{printf "%q" .Path}}, 0755); err != nil {
	return fmt.Errorf("failed to create directory {{.Path}}: %w", err)
}
{{else}}// Verify target directory exists
if _, err := os.Stat({{printf "%q" .Path}}); err != nil {
	return fmt.Errorf("failed to access directory {{.Path}}: %w", err)
}
{{end}}// Execute in working directory: {{.Path}}
{
	// Create isolated context with updated working directory
	workdirCtx := ctx.Clone()
	workdirCtx.Dir = {{printf "%q" .Path}}
	ctx := workdirCtx  // Use workdir context for commands
	
{{range .Content}}	{{. | buildCommand}}
{{end}}
}`

	// Parse template with helper functions
	tmpl, err := template.New("workdir").Funcs(ctx.GetTemplateFunctions()).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workdir template: %w", err)
	}

	return &execution.TemplateResult{
		Template: tmpl,
		Data: struct {
			Path              string
			CreateIfNotExists bool
			Content           []ast.CommandContent
		}{
			Path:              path,
			CreateIfNotExists: createIfNotExists,
			Content:           content,
		},
	}, nil
}

// ShellTemplateData holds template data for workdir shell execution
type ShellTemplateData struct {
	Command string
}

// init registers the workdir decorator
func init() {
	decorators.RegisterBlock(&WorkdirDecorator{})
}
