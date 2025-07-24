package decorators

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// ConfirmDecorator implements the @confirm decorator for user confirmation prompts
type ConfirmDecorator struct{}

// Template for confirmation logic code generation
const confirmExecutionTemplate = `func() error {
	// Check if we should skip confirmation in CI environment
	if {{.SkipInCI}} && func() bool {
		// Check common CI environment variables
		ciVars := []string{
			"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "TRAVIS", 
			"CIRCLECI", "JENKINS_URL", "GITLAB_CI", "BUILDKITE", "BUILD_NUMBER",
		}
		for _, envVar := range ciVars {
			if os.Getenv(envVar) != "" {
				return true
			}
		}
		return false
	}() {
		// Auto-confirm in CI and execute commands
		fmt.Printf("CI environment detected - auto-confirming: {{.Message}}\n")
	} else {
		// Display the confirmation message
		fmt.Print({{.Message}})
		{{if .DefaultYes}}fmt.Print(" [Y/n]: "){{else}}fmt.Print(" [y/N]: "){{end}}
		
		// Read user input
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}
		
		response = strings.TrimSpace(response)
		
		// Determine if user confirmed
		confirmed := false
		if response == "" {
			confirmed = {{.DefaultYes}}
		} else {
			{{if .CaseSensitive}}confirmed = response == "y" || response == "Y" || response == "yes" || response == "Yes"{{else}}lowerResponse := strings.ToLower(response)
			confirmed = lowerResponse == "y" || lowerResponse == "yes"{{end}}
		}
		
		if !confirmed {
			{{if .AbortOnNo}}return fmt.Errorf("user cancelled execution"){{else}}return nil{{end}}
		}
	}
	
	// Execute the commands
	{{range $i, $cmd := .Commands}}{{$cmd}}
	{{end}}
	return nil
}()`

// Name returns the decorator name
func (c *ConfirmDecorator) Name() string {
	return "confirm"
}

// Description returns a human-readable description
func (c *ConfirmDecorator) Description() string {
	return "Prompt user for confirmation before executing commands"
}

// ParameterSchema returns the expected parameters for this decorator
func (c *ConfirmDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
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
func (c *ConfirmDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"bufio", "fmt", "os", "strings"},
	}
}

// isCI checks if we're running in a CI environment
func isCI() bool {
	// Check common CI environment variables
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
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	return false
}

// Execute provides unified execution for all modes using the execution package
func (c *ConfirmDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Validate parameters first

	// Parse parameters with defaults
	message := ast.GetStringParam(params, "message", "Do you want to continue?")
	defaultYes := ast.GetBoolParam(params, "defaultYes", false)
	abortOnNo := ast.GetBoolParam(params, "abortOnNo", true)
	caseSensitive := ast.GetBoolParam(params, "caseSensitive", false)
	skipInCI := ast.GetBoolParam(params, "ci", true)

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return c.executeInterpreter(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
	case execution.GeneratorMode:
		return c.executeGenerator(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
	case execution.PlanMode:
		return c.executePlan(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes confirmation prompt in interpreter mode
func (c *ConfirmDecorator) executeInterpreter(ctx *execution.ExecutionContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Check if we should skip confirmation in CI environment
	if skipInCI && isCI() {
		// Auto-confirm in CI and execute commands
		fmt.Printf("CI environment detected - auto-confirming: %s\n", message)
		for _, cmd := range content {
			if err := ctx.ExecuteCommandContent(cmd); err != nil {
				return &execution.ExecutionResult{
					Mode:  execution.InterpreterMode,
					Data:  nil,
					Error: fmt.Errorf("command execution failed: %w", err),
				}
			}
		}
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: nil,
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
			Mode:  execution.InterpreterMode,
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
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: fmt.Errorf("user cancelled execution"),
			}
		}
		// User said no but don't abort - just skip execution
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: nil,
		}
	}

	// User confirmed, execute the commands
	for _, cmd := range content {
		if err := ctx.ExecuteCommandContent(cmd); err != nil {
			return &execution.ExecutionResult{
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: fmt.Errorf("command execution failed: %w", err),
			}
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: nil,
	}
}

// executeGenerator generates Go code for confirmation logic
func (c *ConfirmDecorator) executeGenerator(ctx *execution.ExecutionContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Generate execution code for each command
	var commands []string
	for i, cmd := range content {
		if shellContent, ok := cmd.(*ast.ShellContent); ok {
			result := ctx.WithMode(execution.GeneratorMode).ExecuteShell(shellContent)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Mode:  execution.GeneratorMode,
					Data:  "",
					Error: fmt.Errorf("failed to generate shell command %d: %w", i, result.Error),
				}
			}
			if code, ok := result.Data.(string); ok {
				// Wrap the shell code in an error-returning function
				wrappedCode := fmt.Sprintf("if err := func() error {\n%s\n\t\treturn nil\n\t}(); err != nil {\n\t\treturn err\n\t}", code)
				commands = append(commands, wrappedCode)
			}
		} else {
			// TODO: Handle other command content types
			commands = append(commands, "// TODO: Generate execution for non-shell command content")
		}
	}

	// Use template to generate the full confirmation logic
	tmpl, err := template.New("confirmExecution").Parse(confirmExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse confirm template: %w", err),
		}
	}

	templateData := struct {
		Message       string
		DefaultYes    bool
		AbortOnNo     bool
		CaseSensitive bool
		SkipInCI      bool
		Commands      []string
	}{
		Message:       fmt.Sprintf("%q", message),
		DefaultYes:    defaultYes,
		AbortOnNo:     abortOnNo,
		CaseSensitive: caseSensitive,
		SkipInCI:      skipInCI,
		Commands:      commands,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute confirm template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (c *ConfirmDecorator) executePlan(ctx *execution.ExecutionContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Context-aware planning: check current environment
	var description string

	if skipInCI && isCI() {
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
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// init registers the confirm decorator
func init() {
	RegisterBlock(&ConfirmDecorator{})
}
