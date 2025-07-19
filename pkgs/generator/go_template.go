package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// TemplateData represents preprocessed data for template generation
type TemplateData struct {
	PackageName         string
	Imports             []string
	HasProcessMgmt      bool
	HasParallelCommands bool
	HasUserDefinedHelp  bool
	Commands            []TemplateCommand
	ProcessMgmtFuncs    []string
}

// CommandSegment represents a part of a mixed command (either parallel or sequential)
type CommandSegment struct {
	IsParallel bool
	Commands   []string // For parallel segments
	Command    string   // For sequential segments
}

// TemplateCommand represents a command ready for template generation
type TemplateCommand struct {
	Name             string           // Original command name
	FunctionName     string           // Sanitized Go function name
	GoCase           string           // Case statement value
	Type             string           // "regular", "watch-stop", "watch-only", "stop-only", "parallel", "mixed", "multi-regular"
	ShellCommand     string           // For single regular commands
	ShellCommands    []string         // For multiple regular commands (block with newlines)
	WatchCommand     string           // For watch part of watch-stop commands
	StopCommand      string           // For stop part of watch-stop commands
	ParallelCommands []string         // For pure parallel commands
	CommandSegments  []CommandSegment // For mixed commands
	IsBackground     bool             // For watch commands
	HelpDescription  string           // Description for help text
}

// validateHelpCommandRestrictions ensures help command isn't used with watch/stop
func validateHelpCommandRestrictions(program *ast.Program) error {
	for _, cmd := range program.Commands {
		if cmd.Name == "help" {
			if cmd.Type == ast.WatchCommand {
				return NewValidationError(
					"'help' command cannot be used with 'watch' modifier. Help is a special reserved command",
					cmd.Name, cmd.Position().Line, "")
			}
			if cmd.Type == ast.StopCommand {
				return NewValidationError(
					"'help' command cannot be used with 'stop' modifier. Help is a special reserved command",
					cmd.Name, cmd.Position().Line, "")
			}
		}
	}
	return nil
}

// PreprocessCommands converts ast.Program into template-ready data with standard library support
func PreprocessCommands(program *ast.Program) (*TemplateData, error) {
	if program == nil {
		return nil, fmt.Errorf("program cannot be nil")
	}

	data := &TemplateData{
		PackageName: "main",
		Imports:     []string{},
		Commands:    []TemplateCommand{},
	}

	// Create variable definitions map for expansion
	definitions := createDefinitionMapFromProgram(program)

	// Group commands by name to find watch/stop pairs
	commandGroups := make(map[string][]*ast.CommandDecl)
	for i := range program.Commands {
		cmd := &program.Commands[i]
		commandGroups[cmd.Name] = append(commandGroups[cmd.Name], cmd)
	}

	// Check if user defined a help command
	_, hasUserHelp := commandGroups["help"]
	data.HasUserDefinedHelp = hasUserHelp

	// Validate help command restrictions FIRST
	if err := validateHelpCommandRestrictions(program); err != nil {
		return nil, err
	}

	// Validate decorators before processing
	if err := validateProgramDecorators(program); err != nil {
		return nil, err
	}

	// Determine what features we need
	hasWatchCommands := false
	hasParallelCommands := false
	hasRegularCommands := len(program.Commands) > 0

	for _, cmd := range program.Commands {
		if cmd.Type == ast.WatchCommand {
			hasWatchCommands = true
		}
		if containsParallelDecorator(&cmd) {
			hasParallelCommands = true
		}
	}

	data.HasProcessMgmt = hasWatchCommands
	data.HasParallelCommands = hasParallelCommands

	// Set up minimal imports - only include what we actually need
	if hasRegularCommands {
		data.Imports = []string{
			"fmt",
			"os",
		}

		// Only add os/exec if we have actual commands
		if len(program.Commands) > 0 {
			data.Imports = append(data.Imports, "os/exec")
		}

		if hasWatchCommands {
			additionalImports := []string{
				"encoding/json",
				"io",
				"os/signal",
				"path/filepath",
				"strings",
				"syscall",
				"time",
			}
			data.Imports = append(data.Imports, additionalImports...)
		}

		if hasParallelCommands {
			data.Imports = append(data.Imports, "sync")
		}
	} else {
		// Minimal imports for empty command files
		data.Imports = []string{"fmt", "os"}
	}

	// Sort imports for consistent output
	sort.Strings(data.Imports)

	// Process command groups with variable expansion
	for name, commands := range commandGroups {
		templateCmd, err := processCommandGroup(name, commands, definitions)
		if err != nil {
			return nil, err
		}
		data.Commands = append(data.Commands, templateCmd)
	}

	// Sort commands for consistent output
	sort.Slice(data.Commands, func(i, j int) bool {
		return data.Commands[i].Name < data.Commands[j].Name
	})

	// Add process management functions if needed
	if hasWatchCommands {
		data.ProcessMgmtFuncs = []string{
			"showStatus",
			"showLogs",
			"stopCommand",
			"runInBackground",
		}
	}

	return data, nil
}

// containsParallelDecorator checks if a command contains @parallel decorator
func containsParallelDecorator(cmd *ast.CommandDecl) bool {
	// Check all content items in the command body
	for _, content := range cmd.Body.Content {
		if containsParallelInContent(content) {
			return true
		}
	}
	return false
}

// containsParallelInContent checks for @parallel in command content
func containsParallelInContent(content ast.CommandContent) bool {
	switch c := content.(type) {
	case *ast.BlockDecorator:
		if c.Name == "parallel" {
			return true
		}
		// Check content inside the block decorator
		for _, innerContent := range c.Content {
			if containsParallelInContent(innerContent) {
				return true
			}
		}
		return false
	case *ast.PatternContent:
		// Check if any commands in this pattern contain parallel
		for _, cmd := range c.Commands {
			if containsParallelInContent(cmd) {
				return true
			}
		}
		return false
	case *ast.PatternDecorator:
		// Check if any pattern branches contain parallel
		for _, pattern := range c.Patterns {
			for _, cmd := range pattern.Commands {
				if containsParallelInContent(cmd) {
					return true
				}
			}
		}
		return false
	case *ast.ShellContent:
		return false
	default:
		return false
	}
}

// processCommandGroup processes a group of commands
func processCommandGroup(name string, commands []*ast.CommandDecl, definitions map[string]string) (TemplateCommand, error) {
	templateCmd := TemplateCommand{
		Name:         name,
		FunctionName: sanitizeFunctionName(name),
		GoCase:       name,
	}

	var watchCmd, stopCmd, regularCmd *ast.CommandDecl

	// Categorize commands in the group
	for _, cmd := range commands {
		switch cmd.Type {
		case ast.WatchCommand:
			watchCmd = cmd
		case ast.StopCommand:
			stopCmd = cmd
		case ast.Command:
			regularCmd = cmd
		}
	}

	// Determine command type and structure
	if regularCmd != nil {
		// Check if it's a parallel or mixed command
		if containsParallelDecorator(regularCmd) {
			// Extract parallel commands directly from command body content array
			var parallelCommands []string
			for _, content := range regularCmd.Body.Content {
				if decoratedContent, ok := content.(*ast.BlockDecorator); ok {
					// For parallel block decorators, extract individual commands from the Content array
					if decoratedContent.Name == "parallel" {
						for _, innerContent := range decoratedContent.Content {
							shellCmd := buildContentString(innerContent, definitions)
							if strings.TrimSpace(shellCmd) != "" {
								parallelCommands = append(parallelCommands, shellCmd)
							}
						}
					} else {
						// Non-parallel decorated content
						shellCmd := buildContentString(decoratedContent, definitions)
						if strings.TrimSpace(shellCmd) != "" {
							parallelCommands = append(parallelCommands, shellCmd)
						}
					}
				} else {
					// Non-decorated content (shouldn't happen in parallel blocks, but handle it)
					shellCmd := buildContentString(content, definitions)
					if strings.TrimSpace(shellCmd) != "" {
						parallelCommands = append(parallelCommands, shellCmd)
					}
				}
			}

			templateCmd.Type = "parallel"
			templateCmd.ParallelCommands = parallelCommands
		} else {
			// Regular command (no parallel) - extract shell commands from content
			shellCommands, err := extractShellCommands(regularCmd.Body.Content, definitions)
			if err != nil {
				return templateCmd, fmt.Errorf("failed to extract shell commands for '%s': %w", name, err)
			}

			if len(shellCommands) > 1 {
				templateCmd.Type = "multi-regular"
				templateCmd.ShellCommands = shellCommands
			} else if len(shellCommands) == 1 {
				templateCmd.Type = "regular"
				templateCmd.ShellCommand = shellCommands[0]
			} else {
				templateCmd.Type = "regular"
				templateCmd.ShellCommand = ""
			}
		}
		templateCmd.HelpDescription = name
	} else if watchCmd != nil && stopCmd != nil {
		// Watch/stop pair
		templateCmd.Type = "watch-stop"
		watchShell, err := buildShellCommand(watchCmd, definitions)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build watch command for '%s': %w", name, err)
		}
		stopShell, err := buildShellCommand(stopCmd, definitions)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build stop command for '%s': %w", name, err)
		}
		templateCmd.WatchCommand = watchShell
		templateCmd.StopCommand = stopShell
		templateCmd.IsBackground = true
		templateCmd.HelpDescription = fmt.Sprintf("%s start|stop|logs", name)
	} else if watchCmd != nil {
		// Watch only
		templateCmd.Type = "watch-only"
		watchShell, err := buildShellCommand(watchCmd, definitions)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build watch command for '%s': %w", name, err)
		}
		templateCmd.WatchCommand = watchShell
		templateCmd.IsBackground = true
		templateCmd.HelpDescription = fmt.Sprintf("%s start|stop|logs", name)
	} else if stopCmd != nil {
		// Stop only (unusual, but handle it)
		templateCmd.Type = "stop-only"
		stopShell, err := buildShellCommand(stopCmd, definitions)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build stop command for '%s': %w", name, err)
		}
		templateCmd.StopCommand = stopShell
		templateCmd.HelpDescription = fmt.Sprintf("%s stop", name)
	} else {
		return templateCmd, fmt.Errorf("no valid commands found in group %s", name)
	}

	return templateCmd, nil
}

// extractShellCommands extracts individual shell commands from command content array
func extractShellCommands(contentArray []ast.CommandContent, definitions map[string]string) ([]string, error) {
	var allCommands []string

	for _, content := range contentArray {
		switch c := content.(type) {
		case *ast.ShellContent:
			commands := buildShellContentString(c, definitions)
			allCommands = append(allCommands, commands...)
		case *ast.BlockDecorator:
			// For decorated content, extract individual shell commands from the content array
			for _, innerContent := range c.Content {
				if shellContent, ok := innerContent.(*ast.ShellContent); ok {
					commands := buildShellContentString(shellContent, definitions)
					allCommands = append(allCommands, commands...)
				} else {
					// Non-shell content, treat as single command
					cmd := buildContentString(innerContent, definitions)
					allCommands = append(allCommands, cmd)
				}
			}
		case *ast.PatternContent:
			// Pattern content is complex - treat as single command
			cmd := buildContentString(c, definitions)
			allCommands = append(allCommands, cmd)
		}
	}

	return allCommands, nil
}

// buildShellCommand constructs the shell command string
func buildShellCommand(cmd *ast.CommandDecl, definitions map[string]string) (string, error) {
	// Build command from all content items
	var parts []string
	for _, content := range cmd.Body.Content {
		part := buildContentString(content, definitions)
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n"), nil
}

// buildContentString builds shell commands from command content
func buildContentString(content ast.CommandContent, definitions map[string]string) string {
	switch c := content.(type) {
	case *ast.ShellContent:
		// For shell content, join all commands with newlines
		commands := buildShellContentString(c, definitions)
		return strings.Join(commands, "\n")
	case *ast.BlockDecorator:
		return buildDecoratedContentString(c, definitions)
	case *ast.PatternContent:
		return buildPatternContentString(c, definitions)
	default:
		return ""
	}
}

// buildShellContentString returns a slice of commands, one for each line in block contexts
func buildShellContentString(content *ast.ShellContent, definitions map[string]string) []string {
	var builder strings.Builder

	// First, build the entire content string, processing variables and functions.
	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			builder.WriteString(p.Text)
		case *ast.FunctionDecorator:
			if p.Name == "var" && len(p.Args) > 0 {
				if id, ok := p.Args[0].(*ast.Identifier); ok {
					if value, exists := definitions[id.Name]; exists {
						builder.WriteString(value)
					} else {
						builder.WriteString(p.String())
					}
				} else {
					builder.WriteString(p.String())
				}
			} else {
				builder.WriteString(p.String())
			}
		default:
			builder.WriteString(part.String())
		}
	}

	// Now, split the fully constructed string by newlines.
	fullString := builder.String()
	lines := strings.Split(fullString, "\n")

	// Filter out empty lines that might result from splitting.
	var commands []string
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			commands = append(commands, trimmedLine)
		}
	}

	return commands
}

// buildDecoratedContentString builds decorated content into a string
func buildDecoratedContentString(content *ast.BlockDecorator, definitions map[string]string) string {
	// Build content from the content array
	var parts []string
	for _, innerContent := range content.Content {
		part := buildContentString(innerContent, definitions)
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n")
}

// buildPatternContentString builds pattern content into a string
func buildPatternContentString(content *ast.PatternContent, definitions map[string]string) string {
	// PatternContent has a single pattern with commands
	var parts []string
	for _, cmd := range content.Commands {
		cmdStr := buildContentString(cmd, definitions)
		if cmdStr != "" {
			parts = append(parts, cmdStr)
		}
	}
	return strings.Join(parts, "; ")
}

// createDefinitionMapFromProgram creates a map from variable definitions
func createDefinitionMapFromProgram(program *ast.Program) map[string]string {
	defMap := make(map[string]string)

	// Add individual variables
	for _, varDecl := range program.Variables {
		defMap[varDecl.Name] = getVariableValue(varDecl.Value)
	}

	// Add grouped variables
	for _, varGroup := range program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			defMap[varDecl.Name] = getVariableValue(varDecl.Value)
		}
	}

	return defMap
}

// getVariableValue extracts the string value from an expression
func getVariableValue(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.NumberLiteral:
		return e.Value
	case *ast.DurationLiteral:
		return e.Value
	case *ast.BooleanLiteral:
		return e.Raw
	case *ast.Identifier:
		return e.Name
	default:
		return expr.String()
	}
}

// validateProgramDecorators validates decorators in the program
func validateProgramDecorators(program *ast.Program) error {
	for _, cmd := range program.Commands {
		if err := validateCommandDecorators(&cmd); err != nil {
			return err
		}
	}
	return nil
}

// validateCommandDecorators validates decorators in a command
func validateCommandDecorators(cmd *ast.CommandDecl) error {
	for _, content := range cmd.Body.Content {
		if err := validateContentDecorators(content, cmd.Name, cmd.Position().Line); err != nil {
			return err
		}
	}
	return nil
}

// validateContentDecorators validates decorators in command content
func validateContentDecorators(content ast.CommandContent, cmdName string, cmdLine int) error {
	switch c := content.(type) {
	case *ast.BlockDecorator:
		if !stdlib.IsValidDecorator(c.Name) {
			return NewValidationError(
				fmt.Sprintf("unsupported decorator '@%s'. Use 'devcmd help decorators' to see supported decorators",
					c.Name),
				cmdName, cmdLine, "")
		}
		// Validate content inside the block decorator
		for _, innerContent := range c.Content {
			if err := validateContentDecorators(innerContent, cmdName, cmdLine); err != nil {
				return err
			}
		}
		return nil

	case *ast.PatternContent:
		// PatternContent doesn't have decorators directly, just validate commands
		for _, cmd := range c.Commands {
			if err := validateContentDecorators(cmd, cmdName, cmdLine); err != nil {
				return err
			}
		}
		return nil

	case *ast.PatternDecorator:
		if !stdlib.IsValidDecorator(c.Name) {
			return NewValidationError(
				fmt.Sprintf("unsupported decorator '@%s'. Use 'devcmd help decorators' to see supported decorators",
					c.Name),
				cmdName, cmdLine, "")
		}
		for _, pattern := range c.Patterns {
			// Validate decorators in all commands of this pattern branch
			for _, cmd := range pattern.Commands {
				if err := validateContentDecorators(cmd, cmdName, cmdLine); err != nil {
					return err
				}
			}
		}
		return nil

	case *ast.ShellContent:
		for _, part := range c.Parts {
			if funcDec, ok := part.(*ast.FunctionDecorator); ok {
				if !stdlib.IsValidDecorator(funcDec.Name) {
					return NewValidationError(
						fmt.Sprintf("unsupported decorator '@%s'. Use 'devcmd help decorators' to see supported decorators",
							funcDec.Name),
						cmdName, cmdLine, "")
				}
			}
		}
		return nil

	default:
		return nil
	}
}

// sanitizeFunctionName converts command names to valid Go function names
func sanitizeFunctionName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			runes := []rune(strings.ToLower(part))
			if len(runes) > 0 {
				runes[0] = unicode.ToUpper(runes[0])
			}
			result.WriteString(string(runes))
		}
	}

	funcName := result.String()
	if funcName == "" {
		funcName = "Command"
	}

	return "run" + funcName
}

// GenerateGo creates a Go CLI from a Program using the composable template system with standard library support
func GenerateGo(program *ast.Program) (string, error) {
	// Preprocess the program into template-ready data
	data, err := PreprocessCommands(program)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	// Create template registry and get all templates
	registry := NewTemplateRegistry()
	allTemplates := registry.GetAllTemplates()

	// Parse and execute template
	tmpl, err := template.New("go-cli").Parse(allTemplates)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "main", data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.String()
	if len(result) == 0 {
		return "", fmt.Errorf("generated empty Go code")
	}

	return result, nil
}

// TemplateRegistry holds all template components
type TemplateRegistry struct {
	templates map[string]string
}

// NewTemplateRegistry creates a new template registry with all components
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]string),
	}
	registry.registerComponents()
	return registry
}

// registerComponents registers all template components
func (tr *TemplateRegistry) registerComponents() {
	// Core templates from the updated templates
	tr.templates["package"] = packageTemplate
	tr.templates["imports"] = importsTemplate
	tr.templates["process-types"] = processTypesTemplate
	tr.templates["process-registry"] = processRegistryTemplate
	tr.templates["cli-struct"] = cliStructTemplate
	tr.templates["main-function"] = mainFunctionTemplate

	// Command templates
	tr.templates["command-switch"] = commandSwitchTemplate
	tr.templates["help-function"] = helpFunctionTemplate
	tr.templates["status-function"] = statusFunctionTemplate
	tr.templates["command-functions"] = commandFunctionsTemplate

	// Command type implementations
	tr.templates["regular-command"] = regularCommandTemplate
	tr.templates["multi-regular-command"] = multiRegularCommandTemplate
	tr.templates["watch-stop-command"] = watchStopCommandTemplate
	tr.templates["watch-only-command"] = watchOnlyCommandTemplate
	tr.templates["stop-only-command"] = stopOnlyCommandTemplate
	tr.templates["parallel-command"] = parallelCommandTemplate
	tr.templates["mixed-command"] = mixedCommandTemplate

	// Process management templates
	tr.templates["process-mgmt-functions"] = processMgmtFunctionsTemplate
}

// GetTemplate returns a specific template component
func (tr *TemplateRegistry) GetTemplate(name string) (string, bool) {
	tmpl, exists := tr.templates[name]
	return tmpl, exists
}

// GetMasterTemplate returns the master template that composes all components
func (tr *TemplateRegistry) GetMasterTemplate() string {
	return masterTemplate
}

// GetAllTemplates returns all template components as a single string
func (tr *TemplateRegistry) GetAllTemplates() string {
	var parts []string

	// Add all component templates
	for _, tmpl := range tr.templates {
		parts = append(parts, tmpl)
	}

	// Add master template
	parts = append(parts, tr.GetMasterTemplate())

	return strings.Join(parts, "\n")
}

// GenerateGoWithTemplate creates a Go CLI with a custom template (for testing)
func GenerateGoWithTemplate(program *ast.Program, templateStr string) (string, error) {
	if len(strings.TrimSpace(templateStr)) == 0 {
		return "", fmt.Errorf("template string cannot be empty")
	}

	// Preprocess the program
	data, err := PreprocessCommands(program)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	// Parse and execute custom template
	tmpl, err := template.New("custom").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GetTemplateComponent returns a specific template component by name
func GetTemplateComponent(name string) (string, error) {
	registry := NewTemplateRegistry()
	template, exists := registry.GetTemplate(name)
	if !exists {
		return "", fmt.Errorf("template component '%s' not found", name)
	}
	return template, nil
}

// GenerateComponentGo generates Go code using only specific template components
func GenerateComponentGo(program *ast.Program, componentNames []string) (string, error) {
	// Preprocess the program into template-ready data
	data, err := PreprocessCommands(program)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	registry := NewTemplateRegistry()
	var templateParts []string

	// Collect requested components
	for _, name := range componentNames {
		component, exists := registry.GetTemplate(name)
		if !exists {
			return "", fmt.Errorf("template component '%s' not found", name)
		}
		templateParts = append(templateParts, component)
	}

	// Add a simple execution template
	templateParts = append(templateParts, "{{template \"package\" .}}\n{{template \"imports\" .}}")

	allTemplates := strings.Join(templateParts, "\n")

	// Parse and execute template
	tmpl, err := template.New("component-cli").Parse(allTemplates)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Helper functions for handling arrays of CommandContent in the new AST structure
