package decorators

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ConfirmDecorator implements the @confirm decorator for user confirmation prompts
type ConfirmDecorator struct{}

// Name returns the decorator name
func (c *ConfirmDecorator) Name() string {
	return "confirm"
}

// Description returns a human-readable description
func (c *ConfirmDecorator) Description() string {
	return "Prompt user for confirmation before executing commands"
}

// ParameterSchema returns the expected parameters for this decorator
func (c *ConfirmDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "message",
			Type:        ast.StringType,
			Required:    false,
			Description: "Message to display to the user (default: 'Do you want to continue?')",
		},
		{
			Name:        "defaultYes",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Default to yes if user just presses enter (default: false)",
		},
		{
			Name:        "abortOnNo",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Abort execution if user says no (default: true)",
		},
		{
			Name:        "caseSensitive",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Make y/n matching case sensitive (default: false)",
		},
		{
			Name:        "ci",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Skip confirmation in CI environments (checks CI env var, default: true)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// ImportRequirements returns the dependencies needed for code generation
func (c *ConfirmDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.StandardImportRequirement(
		decorators.CoreImports,       // fmt
		decorators.FileSystemImports, // os
		decorators.StringImports,     // strings
		[]string{"bufio"},            // For user input reading
	)
}

// isCI checks if we're running in a CI environment using captured environment
func (c *ConfirmDecorator) isCI(ctx execution.BaseContext) bool {
	// Check common CI environment variables from captured environment
	ciVars := []string{
		"CI",                     // Most CI systems
		"CONTINUOUS_INTEGRATION", // Legacy/alternate
		"GITHUB_ACTIONS",         // GitHub Actions
		"TRAVIS",                 // Travis CI
		"CIRCLECI",               // Circle CI
		"JENKINS_URL",            // Jenkins
		"GITLAB_CI",              // GitLab CI
		"BUILDKITE",              // Buildkite
		"BUILD_NUMBER",           // Generic build systems
	}

	for _, envVar := range ciVars {
		if value, exists := ctx.GetEnv(envVar); exists && value != "" {
			return true
		}
	}
	return false
}

// ExecuteInterpreter executes confirmation prompt in interpreter mode
func (c *ConfirmDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	message, defaultYes, abortOnNo, caseSensitive, skipInCI, err := c.extractConfirmParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("confirm parameter error: %w", err),
		}
	}
	return c.executeInterpreterImpl(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
}

// GenerateTemplate generates template for confirmation logic
func (c *ConfirmDecorator) GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) (*execution.TemplateResult, error) {
	message, defaultYes, abortOnNo, caseSensitive, skipInCI, err := c.extractConfirmParams(params)
	if err != nil {
		return nil, fmt.Errorf("confirm parameter error: %w", err)
	}
	return c.generateTemplateImpl(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (c *ConfirmDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	message, defaultYes, abortOnNo, caseSensitive, skipInCI, err := c.extractConfirmParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("confirm parameter error: %w", err),
		}
	}
	return c.executePlanImpl(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
}

// extractConfirmParams extracts and validates confirmation parameters
func (c *ConfirmDecorator) extractConfirmParams(params []ast.NamedParameter) (string, bool, bool, bool, bool, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 0, 5, "confirm"); err != nil {
		return "", false, false, false, false, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, c.ParameterSchema(), "confirm"); err != nil {
		return "", false, false, false, false, err
	}

	// Validate string content for message parameter (no shell injection concerns here)
	if err := decorators.ValidateStringContent(params, "message", "confirm"); err != nil {
		return "", false, false, false, false, err
	}

	// Parse parameters (validation passed, so these should be safe)
	message := ast.GetStringParam(params, "message", "Do you want to continue?")
	defaultYes := ast.GetBoolParam(params, "defaultYes", false)
	abortOnNo := ast.GetBoolParam(params, "abortOnNo", true)
	caseSensitive := ast.GetBoolParam(params, "caseSensitive", false)
	skipInCI := ast.GetBoolParam(params, "ci", true)

	return message, defaultYes, abortOnNo, caseSensitive, skipInCI, nil
}

// executeInterpreterImpl executes confirmation prompt in interpreter mode using utilities
func (c *ConfirmDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Check if we should skip confirmation in CI environment
	if skipInCI && c.isCI(ctx) {
		// Auto-confirm in CI and execute commands in child context
		fmt.Printf("CI environment detected - auto-confirming: %s\n", message)

		// Use CommandExecutor utility to handle command execution
		commandExecutor := decorators.NewCommandExecutor()
		defer commandExecutor.Cleanup()

		err := commandExecutor.ExecuteCommandsWithInterpreter(ctx.Child(), content)
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	// Display the confirmation message
	fmt.Print(message)
	if defaultYes {
		fmt.Print(" [Y/n]: ")
	} else {
		fmt.Print(" [y/N]: ")
	}

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("failed to read user input: %w", err),
		}
	}

	response = strings.TrimSpace(response)

	// Determine if user confirmed
	confirmed := false
	if response == "" {
		confirmed = defaultYes
	} else {
		if caseSensitive {
			confirmed = response == "y" || response == "Y" || response == "yes" || response == "Yes"
		} else {
			lowerResponse := strings.ToLower(response)
			confirmed = lowerResponse == "y" || lowerResponse == "yes"
		}
	}

	if !confirmed {
		if abortOnNo {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("user cancelled execution"),
			}
		}
		// User said no but don't abort - just skip execution
		return &execution.ExecutionResult{
			Data:  nil,
			Error: nil,
		}
	}

	// User confirmed, execute the commands in child context using CommandExecutor utility
	commandExecutor := decorators.NewCommandExecutor()
	defer commandExecutor.Cleanup()

	err = commandExecutor.ExecuteCommandsWithInterpreter(ctx.Child(), content)
	return &execution.ExecutionResult{
		Data:  nil,
		Error: err,
	}
}

// generateTemplateImpl generates template for confirmation logic
func (c *ConfirmDecorator) generateTemplateImpl(ctx execution.GeneratorContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) (*execution.TemplateResult, error) {
	// Track CI environment variables for deterministic behavior
	if skipInCI {
		ctx.TrackEnvironmentVariableReference("CI", "")
		ctx.TrackEnvironmentVariableReference("GITHUB_ACTIONS", "")
		ctx.TrackEnvironmentVariableReference("TRAVIS", "")
		ctx.TrackEnvironmentVariableReference("CIRCLECI", "")
		ctx.TrackEnvironmentVariableReference("JENKINS_URL", "")
		ctx.TrackEnvironmentVariableReference("GITLAB_CI", "")
		ctx.TrackEnvironmentVariableReference("BUILDKITE", "")
		ctx.TrackEnvironmentVariableReference("BUILD_NUMBER", "")
		ctx.TrackEnvironmentVariableReference("CONTINUOUS_INTEGRATION", "")
	}

	// Create template for confirm logic
	tmplStr := `// Confirmation prompt: {{.Message}}
{{if .SkipInCI}}// Check for CI environment variables
if envContext["CI"] != "" || envContext["GITHUB_ACTIONS"] != "" || envContext["TRAVIS"] != "" || envContext["CIRCLECI"] != "" || envContext["JENKINS_URL"] != "" || envContext["GITLAB_CI"] != "" || envContext["BUILDKITE"] != "" || envContext["BUILD_NUMBER"] != "" || envContext["CONTINUOUS_INTEGRATION"] != "" {
	// CI environment detected - Skip confirmation in CI environment
{{range .Content}}	{{. | buildCommand}}
{{end}}	return nil
}
{{end}}fmt.Print({{printf "%q" .Message}} + " {{if .DefaultYes}}[Y/n]{{else}}[y/N]{{end}}: ")
reader := bufio.NewReader(os.Stdin)
response, err := reader.ReadString('\n')
if err != nil {
	return fmt.Errorf("failed to read user input: %w", err)
}
response = strings.TrimSpace(response)

confirmed := false
if response == "" {
	confirmed = {{.DefaultYes}}
} else {
{{if .CaseSensitive}}
	confirmed = response == "y" || response == "Y" || response == "yes" || response == "Yes"
{{else}}
	confirmed = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
{{end}}
}

{{if .AbortOnNo}}
if !confirmed {
	return fmt.Errorf("user cancelled execution")
}
{{else}}
if confirmed {
{{end}}
{{range .Content}}{{. | buildCommand}}
{{end}}
{{if not .AbortOnNo}}
}
{{end}}`

	// Parse template with helper functions
	tmpl, err := template.New("confirm").Funcs(ctx.GetTemplateFunctions()).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse confirm template: %w", err)
	}

	return &execution.TemplateResult{
		Template: tmpl,
		Data: struct {
			Message       string
			DefaultYes    bool
			AbortOnNo     bool
			CaseSensitive bool
			SkipInCI      bool
			Content       []ast.CommandContent
		}{
			Message:       message,
			DefaultYes:    defaultYes,
			AbortOnNo:     abortOnNo,
			CaseSensitive: caseSensitive,
			SkipInCI:      skipInCI,
			Content:       content,
		},
	}, nil
}

// executePlanImpl creates a plan element for dry-run mode
func (c *ConfirmDecorator) executePlanImpl(ctx execution.PlanContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Context-aware planning: check current environment
	var description string

	if skipInCI && c.isCI(ctx) {
		// We're in CI and should skip confirmation
		description = fmt.Sprintf("🤖 CI Environment Detected - Auto-confirming: %s", message)
	} else {
		// Interactive mode - show what user will see
		var prompt string
		if defaultYes {
			prompt = fmt.Sprintf("%s [Y/n]", message)
		} else {
			prompt = fmt.Sprintf("%s [y/N]", message)
		}

		var behavior string
		if abortOnNo {
			behavior = "execution will abort if user declines"
		} else {
			behavior = "execution will skip if user declines"
		}

		description = fmt.Sprintf("🤔 User Prompt: %s (%s)", prompt, behavior)
	}

	element := plan.Decorator("confirm").
		WithType("block").
		WithParameter("message", message).
		WithDescription(description)

	if defaultYes {
		element = element.WithParameter("defaultYes", "true")
	}
	if !abortOnNo {
		element = element.WithParameter("abortOnNo", "false")
	}
	if caseSensitive {
		element = element.WithParameter("caseSensitive", "true")
	}
	if !skipInCI {
		element = element.WithParameter("ci", "false")
	}

	return &execution.ExecutionResult{
		Data:  element,
		Error: nil,
	}
}

// init registers the confirm decorator
func init() {
	decorators.RegisterBlock(&ConfirmDecorator{})
}
